package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

const (
	defaultModelName   = "gpt-test"
	defaultSessionID   = "session-process-e2e"
	defaultDemandToken = "demandtoken"
	defaultAdminToken  = "adminaccesstoken000000000001"

	selfHostedRoutingSlaContractKey     = "process-self-hosted-routing-sla"
	selfHostedRoutingSlaRunKey          = "process-self-hosted-routing-sla-run"
	selfHostedRoutingSlaArtifactSHA256  = "f4a7538d2b9e2df34284cc85e35d31ac6f2c39b1a2a6d8ab2b710f0a14f4c329"
	selfHostedRoutingSlaMeasurementJSON = `{"input_profile":{"buckets":[{"name":"process-routing","max_tokens":4096}]},"output_profile":{"target_tokens":128},"concurrency_profile":{"concurrency":1},"rate_profile":{"rpm":10},"stream_profile":{"include_usage_required":true},"error_profile":{"max_error_rate":0.01},"availability_profile":{"window_seconds":600},"cache_profile":"cold_no_cache"}`
	selfHostedRoutingSlaHardGateJSON    = `{"ttft_ms":{"p90_lte":8000},"error_rate":{"lte":0.01}}`
	selfHostedRoutingSlaSoftGateJSON    = `{"warning_ttft_ms":{"p90_lte":6000}}`
	selfHostedRoutingSlaSummaryJSON     = `{"ttft_ms":{"p90":500},"error_rate":0,"usage":{"streaming":true},"route_mode":"direct_upstream"}`
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: token-router-sim <mock-supply|seed|run> [options]\n")
	os.Exit(2)
}

func initDBForSeed() error {
	if sqlitePath := os.Getenv("SQLITE_PATH"); sqlitePath != "" {
		common.SQLitePath = sqlitePath
	}
	common.DebugEnabled = os.Getenv("DEBUG") == "true"
	common.MemoryCacheEnabled = os.Getenv("MEMORY_CACHE_ENABLED") == "true"
	common.IsMasterNode = true
	common.NodeName = os.Getenv("NODE_NAME")
	logger.SetupLogger()
	ratio_setting.InitRatioSettings()
	service.InitHttpClient()
	service.InitTokenEncoders()
	if err := model.InitDB(); err != nil {
		return err
	}
	model.InitOptionMap()
	return model.InitLogDB()
}

func upsert[T any](value *T, where string, args ...any) error {
	tx := model.DB.Where(where, args...).Assign(value).FirstOrCreate(value)
	return tx.Error
}

func marshalJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		log.Fatalf("marshal seed option: %v", err)
	}
	return string(encoded)
}

func seedPricingOptions(modelName string) error {
	modelRatios := ratio_setting.GetModelRatioCopy()
	modelRatios[modelName] = 1
	completionRatios := ratio_setting.GetCompletionRatioCopy()
	completionRatios[modelName] = 1
	cacheRatios := ratio_setting.GetCacheRatioCopy()
	cacheRatios[modelName] = 0.1
	createCacheRatios := ratio_setting.GetCreateCacheRatioCopy()
	createCacheRatios[modelName] = 1.25
	groupRatios := ratio_setting.GetGroupRatioCopy()
	groupRatios["default"] = 1

	return model.UpdateOptionsBulk(map[string]string{
		"ModelRatio":       marshalJSON(modelRatios),
		"CompletionRatio":  marshalJSON(completionRatios),
		"CacheRatio":       marshalJSON(cacheRatios),
		"CreateCacheRatio": marshalJSON(createCacheRatios),
		"GroupRatio":       marshalJSON(groupRatios),
	})
}

func runSeed(args []string) {
	fs := flag.NewFlagSet("seed", flag.ExitOnError)
	supplyURL := fs.String("supply-url", "http://127.0.0.1:19091", "gb10-4t mock supply base URL")
	modelName := fs.String("model", defaultModelName, "model name")
	adminToken := fs.String("admin-token", defaultAdminToken, "root admin access token")
	demandToken := fs.String("demand-token", defaultDemandToken, "demand-side API token without sk- prefix")
	_ = fs.Parse(args)

	common.IsMasterNode = true
	if err := initDBForSeed(); err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer func() { _ = model.CloseDB() }()
	if err := seedPricingOptions(*modelName); err != nil {
		log.Fatalf("seed pricing options: %v", err)
	}

	requireNoDash(*demandToken, "demand-token")
	requireNoDash(*adminToken, "admin-token")

	rootAccessToken := *adminToken
	root := &model.User{
		Id:          1,
		Username:    "root",
		Password:    "unused",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Quota:       1_000_000,
		AffCode:     "root-aff",
		AccessToken: &rootAccessToken,
	}
	if err := upsert(root, "id = ?", root.Id); err != nil {
		log.Fatalf("upsert root user: %v", err)
	}
	user := &model.User{
		Id:       2,
		Username: "demand-simulator",
		Password: "unused",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Quota:    1_000_000,
		AffCode:  "demand-aff",
	}
	if err := upsert(user, "id = ?", user.Id); err != nil {
		log.Fatalf("upsert demand user: %v", err)
	}
	token := &model.Token{
		Id:             1,
		UserId:         user.Id,
		Key:            *demandToken,
		Name:           "demand-simulator",
		Status:         common.TokenStatusEnabled,
		ExpiredTime:    -1,
		UnlimitedQuota: true,
		Group:          "default",
	}
	if err := upsert(token, "id = ?", token.Id); err != nil {
		log.Fatalf("upsert demand token: %v", err)
	}

	supplier := &model.Supplier{
		Id:     1,
		Name:   "gb10-4t",
		Type:   model.SupplierTypeThirdParty,
		Status: 1,
	}
	if err := upsert(supplier, "id = ?", supplier.Id); err != nil {
		log.Fatalf("upsert supplier: %v", err)
	}
	selfHostedSupplier := &model.Supplier{
		Id:     2,
		Name:   "gb10-4t-self-hosted",
		Type:   model.SupplierTypeSelfHosted,
		Status: 1,
	}
	if err := upsert(selfHostedSupplier, "id = ?", selfHostedSupplier.Id); err != nil {
		log.Fatalf("upsert self-hosted supplier: %v", err)
	}
	selfOperatedSupplier := &model.Supplier{
		Id:     3,
		Name:   "gb10-4t-self-operated",
		Type:   model.SupplierTypeSelfOperated,
		Status: 1,
	}
	if err := upsert(selfOperatedSupplier, "id = ?", selfOperatedSupplier.Id); err != nil {
		log.Fatalf("upsert self-operated supplier: %v", err)
	}
	for channelID := 1; channelID <= 2; channelID++ {
		channel := &model.Channel{
			Id:         channelID,
			Type:       constant.ChannelTypeOpenAI,
			Key:        "sk-gb10-4t-mock",
			Status:     common.ChannelStatusEnabled,
			Name:       "gb10-4t",
			BaseURL:    supplyURL,
			Models:     *modelName,
			Group:      "default",
			SupplierId: supplier.Id,
		}
		if err := upsert(channel, "id = ?", channel.Id); err != nil {
			log.Fatalf("upsert channel: %v", err)
		}
		ability := &model.Ability{
			Group:     "default",
			Model:     *modelName,
			ChannelId: channel.Id,
			Enabled:   true,
			Weight:    100,
		}
		if err := upsert(ability, "`group` = ? AND model = ? AND channel_id = ?", ability.Group, ability.Model, ability.ChannelId); err != nil {
			log.Fatalf("upsert ability: %v", err)
		}
	}
	selfHostedPriority := int64(-10)
	selfHostedChannel := &model.Channel{
		Id:         3,
		Type:       constant.ChannelTypeOpenAI,
		Key:        "sk-gb10-4t-self-hosted-mock",
		Status:     common.ChannelStatusEnabled,
		Name:       "gb10-4t-self-hosted",
		BaseURL:    supplyURL,
		Models:     *modelName,
		Group:      "default",
		SupplierId: selfHostedSupplier.Id,
		Priority:   &selfHostedPriority,
	}
	if err := upsert(selfHostedChannel, "id = ?", selfHostedChannel.Id); err != nil {
		log.Fatalf("upsert self-hosted channel: %v", err)
	}
	selfHostedAbility := &model.Ability{
		Group:     "default",
		Model:     *modelName,
		ChannelId: selfHostedChannel.Id,
		Enabled:   true,
		Priority:  &selfHostedPriority,
		Weight:    100,
	}
	if err := upsert(selfHostedAbility, "`group` = ? AND model = ? AND channel_id = ?", selfHostedAbility.Group, selfHostedAbility.Model, selfHostedAbility.ChannelId); err != nil {
		log.Fatalf("upsert self-hosted ability: %v", err)
	}
	agreement := &model.SupplierAgreement{
		Id:                     1,
		SupplierId:             supplier.Id,
		ModelName:              *modelName,
		EffectiveFrom:          0,
		CostModelRatio:         0.5,
		CostCompletionRatio:    1,
		CostCacheRatio:         0.05,
		CostCacheCreationRatio: 0.5,
		Status:                 1,
	}
	if err := upsert(agreement, "id = ?", agreement.Id); err != nil {
		log.Fatalf("upsert supplier agreement: %v", err)
	}
	selfHostedAgreement := &model.SupplierAgreement{
		Id:                     2,
		SupplierId:             selfHostedSupplier.Id,
		ModelName:              *modelName,
		EffectiveFrom:          0,
		CostModelRatio:         0.35,
		CostCompletionRatio:    1,
		CostCacheRatio:         0.02,
		CostCacheCreationRatio: 0.35,
		Status:                 1,
	}
	if err := upsert(selfHostedAgreement, "id = ?", selfHostedAgreement.Id); err != nil {
		log.Fatalf("upsert self-hosted supplier agreement: %v", err)
	}
	now := common.GetTimestamp()
	capacity := &model.SupplyCapacity{
		Id:             1,
		SupplierId:     supplier.Id,
		SupplyNode:     "gb10-4t",
		ModelName:      *modelName,
		PeriodStart:    now - 3600,
		PeriodEnd:      now + 3600,
		CapacityTokens: 1000,
		UsedTokens:     0,
		QualityScore:   98.5,
		UnitCostQuota:  0.5,
		Status:         1,
		Notes:          "seeded gb10-4t capacity snapshot",
	}
	if err := upsert(capacity, "id = ?", capacity.Id); err != nil {
		log.Fatalf("upsert supply capacity: %v", err)
	}
	fmt.Printf("seeded gb10-4t supply with 2 third-party channels, 1 self-hosted channel, and 1 self-operated supplier, demand token sk-%s, admin token %s\n", *demandToken, *adminToken)
}

func requireNoDash(value string, name string) {
	if strings.Contains(value, "-") {
		log.Fatalf("%s must not contain '-' because new-api token auth splits sk tokens on '-'", name)
	}
}

type mockSupply struct {
	mu           sync.Mutex
	bySessID     map[string]int
	requestCount int64
	usedTokens   int64
}

func runMockSupply(args []string) {
	fs := flag.NewFlagSet("mock-supply", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:19091", "listen address")
	modelName := fs.String("model", defaultModelName, "model name")
	requiredSessionID := fs.String("require-session", "", "reject requests that do not carry this upstream session id")
	_ = fs.Parse(args)

	mock := &mockSupply{bySessID: map[string]int{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/token-router/telemetry/capacity", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Query().Get("model") != *modelName {
			http.Error(w, "unexpected model", http.StatusBadRequest)
			return
		}
		mock.mu.Lock()
		requestCount := mock.requestCount
		usedTokens := mock.usedTokens
		mock.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"supply_node":          "gb10-4t",
			"model_name":           *modelName,
			"capacity_tokens":      1000,
			"used_tokens":          usedTokens,
			"gpu_utilization_rate": 0.62,
			"quality_score":        98.5,
			"unit_cost_quota":      0.5,
			"observed_at":          time.Now().Unix(),
			"source_ref":           "gb10-4t-mock-capacity",
			"notes":                fmt.Sprintf("gb10-4t mock telemetry after %d request(s)", requestCount),
		})
	})
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var requestBody struct {
			Stream bool `json:"stream"`
		}
		_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&requestBody)
		sessionID := r.Header.Get("X-Session-Id")
		if sessionID == "" {
			sessionID = r.Header.Get("session_id")
		}
		if *requiredSessionID == "*" && sessionID == "" {
			http.Error(w, "expected non-empty session", http.StatusBadRequest)
			return
		}
		if *requiredSessionID != "" && *requiredSessionID != "*" && sessionID != *requiredSessionID {
			http.Error(w, fmt.Sprintf("expected session %q, got %q", *requiredSessionID, sessionID), http.StatusBadRequest)
			return
		}
		mock.mu.Lock()
		mock.bySessID[sessionID]++
		callIndex := mock.bySessID[sessionID]
		mock.mu.Unlock()

		promptTokens := 120
		cachedTokens := 0
		if callIndex > 1 {
			promptTokens = 140
			cachedTokens = 80
		}
		completionTokens := 20
		mock.mu.Lock()
		mock.requestCount++
		mock.usedTokens += int64(promptTokens + completionTokens)
		mock.mu.Unlock()
		if requestBody.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("X-Request-Id", fmt.Sprintf("gb10-stream-%d", callIndex))
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming unsupported", http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-gb10-%d\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"}}]}\n\n", callIndex, time.Now().Unix(), *modelName)
			flusher.Flush()
			time.Sleep(5 * time.Millisecond)
			fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-gb10-%d\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"}}]}\n\n", callIndex, time.Now().Unix(), *modelName)
			flusher.Flush()
			fmt.Fprintf(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":%d,\"completion_tokens\":%d,\"total_tokens\":%d,\"prompt_tokens_details\":{\"cached_tokens\":%d}}}\n\n", promptTokens, completionTokens, promptTokens+completionTokens, cachedTokens)
			fmt.Fprint(w, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      fmt.Sprintf("chatcmpl-gb10-%d", callIndex),
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   *modelName,
			"choices": []map[string]any{{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": fmt.Sprintf("gb10-4t mock response %d", callIndex),
				},
				"finish_reason": "stop",
			}},
			"usage": map[string]any{
				"prompt_tokens":     promptTokens,
				"completion_tokens": completionTokens,
				"total_tokens":      promptTokens + completionTokens,
				"prompt_tokens_details": map[string]any{
					"cached_tokens": cachedTokens,
				},
			},
		})
	})
	log.Printf("gb10-4t mock supply listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func runDemand(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	baseURL := fs.String("base-url", "http://127.0.0.1:19090", "token-router base URL")
	modelName := fs.String("model", defaultModelName, "model name")
	sessionID := fs.String("session-id", defaultSessionID, "session id")
	adminToken := fs.String("admin-token", defaultAdminToken, "root admin access token")
	demandToken := fs.String("demand-token", defaultDemandToken, "demand-side API token without sk- prefix")
	expectSlaEvidence := fs.Bool("expect-sla-evidence", false, "require supplier evaluation to reference passed SLA admission evidence")
	expectedSlaRunKey := fs.String("expected-sla-run-key", "", "expected SLA probe run key substring in supplier evaluation gate summary")
	_ = fs.Parse(args)

	client := &http.Client{Timeout: 10 * time.Second}
	mustChat(client, *baseURL, *modelName, *sessionID, *demandToken, "process-e2e-request-1")
	mustChat(client, *baseURL, *modelName, *sessionID, *demandToken, "process-e2e-request-2")
	ledgers := waitLedgers(client, *baseURL, *sessionID, *adminToken, 2)
	verifyLedgers(ledgers, *sessionID)
	verifySettlement(client, *baseURL, *adminToken, ledgers)
	verifySupplyCapacity(client, *baseURL, *adminToken, ledgers, *modelName)
	mustChat(client, *baseURL, *modelName, *sessionID, *demandToken, "process-e2e-request-2")
	ledgers = requireLedgerCountStable(client, *baseURL, *sessionID, *adminToken, 2, 2*time.Second)
	verifyLedgers(ledgers, *sessionID)
	mustChat(client, *baseURL, *modelName, *sessionID, *demandToken, "process-e2e-request-scorecard-boost")
	ledgers = waitLedgers(client, *baseURL, *sessionID, *adminToken, 3)
	verifyLedgers(ledgers, *sessionID)
	verifySupplierScorecard(client, *baseURL, *adminToken, ledgers, *expectSlaEvidence, *expectedSlaRunKey)
	profile := verifyTrafficProfile(client, *baseURL, *adminToken, ledgers, *modelName)
	forecast := verifyTrafficForecast(client, *baseURL, *adminToken, profile, *modelName)
	verifyPricingRecommendation(client, *baseURL, *adminToken, profile, ledgers, *modelName)
	ledgers = verifySupplyDecision(client, *baseURL, *adminToken, *demandToken, *sessionID, profile, forecast, ledgers, *modelName)
	verifySeasonalAnomalyTrafficForecast(client, *baseURL, *adminToken, *modelName)
	assignedSession := verifyAssignedSession(client, *baseURL, *modelName, *demandToken, *adminToken)
	fmt.Printf("process e2e ok: ledgers=%d session=%s assigned_session=%s cached_tokens_verified=true margin_verified=true settlement_verified=true capacity_verified=true capacity_usage_refresh_verified=true capacity_telemetry_verified=true capacity_telemetry_collect_verified=true capacity_telemetry_sweep_verified=true capacity_telemetry_insight_verified=true supplier_scorecard_verified=true supplier_evaluation_verified=true supplier_posture_verified=true supplier_route_preference_verified=true sla_evidence_verified=%t traffic_profile_verified=true traffic_forecast_verified=true traffic_forecast_seasonal_anomaly_verified=true pricing_recommendation_verified=true supply_decision_verified=true self_hosted_cost_profile_verified=true supply_prepaid_lot_verified=true supply_expansion_opportunity_verified=true operating_insight_verified=true supply_action_plan_verified=true supply_action_execution_verified=true supply_action_execution_drawdown_verified=true supply_routing_policy_verified=true supply_routing_policy_canary_verified=true routing_sla_evidence_verified=true policy_miss_insight_verified=true assigned_session_verified=true\n", len(ledgers), *sessionID, assignedSession, *expectSlaEvidence)
}

func mustChat(client *http.Client, baseURL string, modelName string, sessionID string, demandToken string, requestID string) string {
	body := map[string]any{
		"model": modelName,
		"messages": []map[string]string{{
			"role":    "user",
			"content": "hello from process demand simulator",
		}},
	}
	payload, err := json.Marshal(body)
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/chat/completions", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-"+demandToken)
	req.Header.Set("X-Session-Id", sessionID)
	req.Header.Set(common.UsageLedgerRequestIdHeader, requestID)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("chat request %s failed with status %d", requestID, resp.StatusCode)
	}
	assignedSession := strings.TrimSpace(resp.Header.Get("X-Session-Id"))
	var decoded map[string]any
	must(json.NewDecoder(resp.Body).Decode(&decoded))
	if _, ok := decoded["choices"]; !ok {
		log.Fatalf("chat request %s response missing choices", requestID)
	}
	return assignedSession
}

func mustChatWithoutSession(client *http.Client, baseURL string, modelName string, demandToken string, requestID string) string {
	body := map[string]any{
		"model": modelName,
		"messages": []map[string]string{{
			"role":    "user",
			"content": "hello without a caller session",
		}},
	}
	payload, err := json.Marshal(body)
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/chat/completions", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-"+demandToken)
	req.Header.Set(common.UsageLedgerRequestIdHeader, requestID)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("chat request %s without session failed with status %d", requestID, resp.StatusCode)
	}
	assignedSession := strings.TrimSpace(resp.Header.Get("X-Session-Id"))
	if assignedSession == "" {
		log.Fatalf("chat request %s response missing X-Session-Id", requestID)
	}
	var decoded map[string]any
	must(json.NewDecoder(resp.Body).Decode(&decoded))
	if _, ok := decoded["choices"]; !ok {
		log.Fatalf("chat request %s response missing choices", requestID)
	}
	return assignedSession
}

func verifyAssignedSession(client *http.Client, baseURL string, modelName string, demandToken string, adminToken string) string {
	const requestID = "process-e2e-request-assigned-session"
	assignedSession := mustChatWithoutSession(client, baseURL, modelName, demandToken, requestID)
	if !strings.HasPrefix(assignedSession, "trsess_") {
		log.Fatalf("expected generated session to start with trsess_, got %q", assignedSession)
	}
	ledgers := waitLedgers(client, baseURL, assignedSession, adminToken, 1)
	ledger := ledgers[0]
	if ledger.RequestId != requestID ||
		ledger.SessionId != assignedSession ||
		ledger.ChannelId <= 0 ||
		ledger.SupplierId <= 0 ||
		ledger.SupplyNode == "" ||
		ledger.SellQuota <= ledger.CostQuota {
		log.Fatalf("unexpected assigned-session ledger: got=%+v assigned=%s", ledger, assignedSession)
	}
	return assignedSession
}

func waitLedgers(client *http.Client, baseURL string, sessionID string, adminToken string, expected int) []ledgerEnvelopeItem {
	deadline := time.Now().Add(5 * time.Second)
	var last []ledgerEnvelopeItem
	for {
		last = getLedgers(client, baseURL, sessionID, adminToken)
		if len(last) == expected {
			return last
		}
		if time.Now().After(deadline) {
			log.Fatalf("expected %d ledgers, got %d", expected, len(last))
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func requireLedgerCountStable(client *http.Client, baseURL string, sessionID string, adminToken string, expected int, duration time.Duration) []ledgerEnvelopeItem {
	deadline := time.Now().Add(duration)
	var last []ledgerEnvelopeItem
	for {
		last = getLedgers(client, baseURL, sessionID, adminToken)
		if len(last) != expected {
			log.Fatalf("expected %d ledgers to remain stable, got %d", expected, len(last))
		}
		if time.Now().After(deadline) {
			return last
		}
		time.Sleep(100 * time.Millisecond)
	}
}

type ledgerEnvelopeItem struct {
	RequestId        string `json:"request_id"`
	SessionId        string `json:"session_id"`
	ChannelId        int    `json:"channel_id"`
	SupplierId       int    `json:"supplier_id"`
	SupplyNode       string `json:"supply_node"`
	PromptTokens     int    `json:"prompt_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	SellQuota        int    `json:"sell_quota"`
	CostQuota        int    `json:"cost_quota"`
	CacheHit         bool   `json:"cache_hit"`
}

func getLedgers(client *http.Client, baseURL string, sessionID string, adminToken string) []ledgerEnvelopeItem {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/usage_ledgers?page_size=10&session_id="+sessionID, nil)
	must(err)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("usage ledger query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []ledgerEnvelopeItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("usage ledger query returned success=false")
	}
	return envelope.Data.Items
}

func verifyLedgers(ledgers []ledgerEnvelopeItem, sessionID string) {
	if len(ledgers) < 2 {
		log.Fatalf("expected at least 2 ledgers, got %d", len(ledgers))
	}
	cached := false
	channelID := ledgers[0].ChannelId
	if channelID <= 0 {
		log.Fatalf("expected positive channel id, got %d", channelID)
	}
	for _, ledger := range ledgers {
		if ledger.SessionId != sessionID {
			log.Fatalf("unexpected session id %q", ledger.SessionId)
		}
		if ledger.ChannelId != channelID {
			log.Fatalf("expected session affinity to keep channel %d, got %d", channelID, ledger.ChannelId)
		}
		if ledger.SupplyNode != "gb10-4t" {
			log.Fatalf("unexpected supply node %q", ledger.SupplyNode)
		}
		if ledger.SellQuota <= ledger.CostQuota {
			log.Fatalf("expected sell quota > cost quota, got sell=%d cost=%d", ledger.SellQuota, ledger.CostQuota)
		}
		if ledger.CachedTokens > 0 && ledger.CacheHit {
			cached = true
		}
	}
	if !cached {
		log.Fatalf("expected at least one cache-hit ledger")
	}
}

type expectedSettlementTotals struct {
	SupplierId            int
	TotalRequests         int64
	TotalSellQuota        int64
	TotalCostQuota        int64
	GrossProfitQuota      int64
	TotalPromptTokens     int64
	TotalCachedTokens     int64
	TotalCompletionTokens int64
	CacheHitCount         int64
	CacheHitRate          float64
}

func expectedTotalsFromLedgers(ledgers []ledgerEnvelopeItem) expectedSettlementTotals {
	totals := expectedSettlementTotals{SupplierId: ledgers[0].SupplierId}
	for _, ledger := range ledgers {
		totals.TotalRequests++
		totals.TotalSellQuota += int64(ledger.SellQuota)
		totals.TotalCostQuota += int64(ledger.CostQuota)
		totals.TotalPromptTokens += int64(ledger.PromptTokens)
		totals.TotalCachedTokens += int64(ledger.CachedTokens)
		totals.TotalCompletionTokens += int64(ledger.CompletionTokens)
		if ledger.CacheHit {
			totals.CacheHitCount++
		}
	}
	totals.GrossProfitQuota = totals.TotalSellQuota - totals.TotalCostQuota
	if totals.TotalRequests > 0 {
		totals.CacheHitRate = float64(totals.CacheHitCount) / float64(totals.TotalRequests)
	}
	return totals
}

type marginSummaryItem struct {
	SupplierId            int     `json:"supplier_id"`
	TotalRequests         int64   `json:"total_requests"`
	TotalSellQuota        int64   `json:"total_sell_quota"`
	TotalCostQuota        int64   `json:"total_cost_quota"`
	GrossProfitQuota      int64   `json:"gross_profit_quota"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCachedTokens     int64   `json:"total_cached_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	CacheHitCount         int64   `json:"cache_hit_count"`
	CacheHitRate          float64 `json:"cache_hit_rate"`
}

type settlementStatementItem struct {
	Id                    int     `json:"id"`
	SubjectType           string  `json:"subject_type"`
	SupplierId            int     `json:"supplier_id"`
	TotalRequests         int64   `json:"total_requests"`
	TotalSellQuota        int64   `json:"total_sell_quota"`
	TotalCostQuota        int64   `json:"total_cost_quota"`
	GrossProfitQuota      int64   `json:"gross_profit_quota"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCachedTokens     int64   `json:"total_cached_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	CacheHitRate          float64 `json:"cache_hit_rate"`
}

type supplyCapacityItem struct {
	Id                  int     `json:"id"`
	SupplierId          int     `json:"supplier_id"`
	SupplyNode          string  `json:"supply_node"`
	ModelName           string  `json:"model_name"`
	PeriodStart         int64   `json:"period_start"`
	PeriodEnd           int64   `json:"period_end"`
	CapacityTokens      int64   `json:"capacity_tokens"`
	UsedTokens          int64   `json:"used_tokens"`
	HeadroomTokens      int64   `json:"headroom_tokens"`
	UtilizationRate     float64 `json:"utilization_rate"`
	GpuUtilizationRate  float64 `json:"gpu_utilization_rate"`
	QualityScore        float64 `json:"quality_score"`
	UnitCostQuota       float64 `json:"unit_cost_quota"`
	TelemetrySourceType string  `json:"telemetry_source_type"`
	TelemetrySourceRef  string  `json:"telemetry_source_ref"`
	TelemetryObservedAt int64   `json:"telemetry_observed_at"`
	LastTelemetryId     int     `json:"last_telemetry_id"`
}

type supplyCapacityTelemetryItem struct {
	Id                 int     `json:"id"`
	TelemetryKey       string  `json:"telemetry_key"`
	SupplierId         int     `json:"supplier_id"`
	SupplyNode         string  `json:"supply_node"`
	ModelName          string  `json:"model_name"`
	PeriodStart        int64   `json:"period_start"`
	PeriodEnd          int64   `json:"period_end"`
	CapacityTokens     int64   `json:"capacity_tokens"`
	UsedTokens         int64   `json:"used_tokens"`
	HeadroomTokens     int64   `json:"headroom_tokens"`
	UtilizationRate    float64 `json:"utilization_rate"`
	GpuUtilizationRate float64 `json:"gpu_utilization_rate"`
	QualityScore       float64 `json:"quality_score"`
	UnitCostQuota      float64 `json:"unit_cost_quota"`
	SourceType         string  `json:"source_type"`
	SourceRef          string  `json:"source_ref"`
	ObservedAt         int64   `json:"observed_at"`
	AppliedCapacityId  int     `json:"applied_capacity_id"`
	RecordedBy         int     `json:"recorded_by"`
}

type supplyCapacityTelemetrySweepResult struct {
	AttemptedCount int                           `json:"attempted_count"`
	CollectedCount int                           `json:"collected_count"`
	SkippedCount   int                           `json:"skipped_count"`
	Collected      []supplyCapacityTelemetryItem `json:"collected"`
}

type supplyCostProfileItem struct {
	Id                     int     `json:"id"`
	CostProfileKey         string  `json:"cost_profile_key"`
	SupplierId             int     `json:"supplier_id"`
	SupplyNode             string  `json:"supply_node"`
	ModelName              string  `json:"model_name"`
	PeriodStart            int64   `json:"period_start"`
	PeriodEnd              int64   `json:"period_end"`
	CapacityTokens         int64   `json:"capacity_tokens"`
	FixedCostQuota         float64 `json:"fixed_cost_quota"`
	VariableUnitCostQuota  float64 `json:"variable_unit_cost_quota"`
	AmortizedUnitCostQuota float64 `json:"amortized_unit_cost_quota"`
	SourceType             string  `json:"source_type"`
	SourceRef              string  `json:"source_ref"`
	ObservedAt             int64   `json:"observed_at"`
	RecordedBy             int     `json:"recorded_by"`
}

type supplyPrepaidLotItem struct {
	Id                   int     `json:"id"`
	PrepaidLotKey        string  `json:"prepaid_lot_key"`
	SupplierId           int     `json:"supplier_id"`
	ChannelId            int     `json:"channel_id"`
	SupplyNode           string  `json:"supply_node"`
	ModelName            string  `json:"model_name"`
	PeriodStart          int64   `json:"period_start"`
	PeriodEnd            int64   `json:"period_end"`
	PurchasedTokens      int64   `json:"purchased_tokens"`
	UnitCostQuota        float64 `json:"unit_cost_quota"`
	TotalCostQuota       float64 `json:"total_cost_quota"`
	DrawdownTokens       int64   `json:"drawdown_tokens"`
	DrawdownRequestCount int64   `json:"drawdown_request_count"`
	RemainingTokens      int64   `json:"remaining_tokens"`
	DrawdownRate         float64 `json:"drawdown_rate"`
	DrawdownSourceType   string  `json:"drawdown_source_type"`
	DrawdownSourceRef    string  `json:"drawdown_source_ref"`
	DrawdownRefreshedAt  int64   `json:"drawdown_refreshed_at"`
	SourceType           string  `json:"source_type"`
	SourceRef            string  `json:"source_ref"`
	ObservedAt           int64   `json:"observed_at"`
	ExternalRef          string  `json:"external_ref"`
	RecordedBy           int     `json:"recorded_by"`
}

type supplierScorecardItem struct {
	Id                    int     `json:"id"`
	SupplierId            int     `json:"supplier_id"`
	PeriodStart           int64   `json:"period_start"`
	PeriodEnd             int64   `json:"period_end"`
	TotalRequests         int64   `json:"total_requests"`
	SuccessRequests       int64   `json:"success_requests"`
	ErrorRequests         int64   `json:"error_requests"`
	SuccessRate           float64 `json:"success_rate"`
	AvgLatencyMs          float64 `json:"avg_latency_ms"`
	MaxLatencyMs          int     `json:"max_latency_ms"`
	CacheHitCount         int64   `json:"cache_hit_count"`
	CacheHitRate          float64 `json:"cache_hit_rate"`
	TotalSellQuota        int64   `json:"total_sell_quota"`
	TotalCostQuota        int64   `json:"total_cost_quota"`
	GrossProfitQuota      int64   `json:"gross_profit_quota"`
	SupplyCapacityTokens  int64   `json:"supply_capacity_tokens"`
	SupplyUsedTokens      int64   `json:"supply_used_tokens"`
	SupplyHeadroomTokens  int64   `json:"supply_headroom_tokens"`
	AvgSupplyQualityScore float64 `json:"avg_supply_quality_score"`
	AvgUnitCostQuota      float64 `json:"avg_unit_cost_quota"`
	Score                 float64 `json:"score"`
	Grade                 string  `json:"grade"`
}

type supplierEvaluationItem struct {
	Id                    int     `json:"id"`
	EvaluationType        string  `json:"evaluation_type"`
	SupplierId            int     `json:"supplier_id"`
	SupplierScorecardId   int     `json:"supplier_scorecard_id"`
	SlaContractId         int     `json:"sla_contract_id"`
	SlaProbeRunId         int     `json:"sla_probe_run_id"`
	SlaGateSummaryJSON    string  `json:"sla_gate_summary_json"`
	PeriodStart           int64   `json:"period_start"`
	PeriodEnd             int64   `json:"period_end"`
	Status                string  `json:"status"`
	Recommendation        string  `json:"recommendation"`
	Score                 float64 `json:"score"`
	Grade                 string  `json:"grade"`
	TotalRequests         int64   `json:"total_requests"`
	SuccessRate           float64 `json:"success_rate"`
	AvgLatencyMs          float64 `json:"avg_latency_ms"`
	CacheHitRate          float64 `json:"cache_hit_rate"`
	GrossProfitQuota      int64   `json:"gross_profit_quota"`
	SupplyHeadroomTokens  int64   `json:"supply_headroom_tokens"`
	AvgSupplyQualityScore float64 `json:"avg_supply_quality_score"`
	AvgUnitCostQuota      float64 `json:"avg_unit_cost_quota"`
	Reason                string  `json:"reason"`
	ReviewedAt            int64   `json:"reviewed_at"`
	ReviewedBy            int     `json:"reviewed_by"`
	ReviewNote            string  `json:"review_note"`
	AppliedAt             int64   `json:"applied_at"`
	AppliedBy             int     `json:"applied_by"`
	AppliedNote           string  `json:"applied_note"`
	SupplierStatusBefore  int     `json:"supplier_status_before"`
	SupplierStatusAfter   int     `json:"supplier_status_after"`
}

type supplierPostureRecommendationItem struct {
	Id                    int     `json:"id"`
	RecommendationKey     string  `json:"recommendation_key"`
	SupplierId            int     `json:"supplier_id"`
	SupplierScorecardId   int     `json:"supplier_scorecard_id"`
	PeriodStart           int64   `json:"period_start"`
	PeriodEnd             int64   `json:"period_end"`
	Status                string  `json:"status"`
	RecommendedAction     string  `json:"recommended_action"`
	Score                 float64 `json:"score"`
	Grade                 string  `json:"grade"`
	TotalRequests         int64   `json:"total_requests"`
	QualityInsightCount   int     `json:"quality_insight_count"`
	CapacityInsightCount  int     `json:"capacity_insight_count"`
	ActionInsightCount    int     `json:"action_insight_count"`
	SupplierStatusCurrent int     `json:"supplier_status_current"`
	Reason                string  `json:"reason"`
	ReviewedAt            int64   `json:"reviewed_at"`
	ReviewedBy            int     `json:"reviewed_by"`
	ReviewNote            string  `json:"review_note"`
	AppliedAt             int64   `json:"applied_at"`
	AppliedBy             int     `json:"applied_by"`
	AppliedNote           string  `json:"applied_note"`
	SupplierStatusBefore  int     `json:"supplier_status_before"`
	SupplierStatusAfter   int     `json:"supplier_status_after"`
}

type supplierRoutePreferenceItem struct {
	Id                            int    `json:"id"`
	SupplierId                    int    `json:"supplier_id"`
	SourcePostureRecommendationId int    `json:"source_posture_recommendation_id"`
	Status                        string `json:"status"`
	WeightPercent                 int    `json:"weight_percent"`
	Reason                        string `json:"reason"`
	EffectiveFrom                 int64  `json:"effective_from"`
	EffectiveTo                   int64  `json:"effective_to"`
	ActivatedAt                   int64  `json:"activated_at"`
	ActivatedBy                   int    `json:"activated_by"`
	OperatorNote                  string `json:"operator_note"`
}

type supplierItem struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status int    `json:"status"`
	Notes  string `json:"notes"`
}

type trafficProfileItem struct {
	Id                    int     `json:"id"`
	SliceKey              string  `json:"slice_key"`
	ModelName             string  `json:"model_name"`
	SlaTier               string  `json:"sla_tier"`
	UserId                int     `json:"user_id"`
	PeriodStart           int64   `json:"period_start"`
	PeriodEnd             int64   `json:"period_end"`
	RequestCount          int64   `json:"request_count"`
	SuccessRequestCount   int64   `json:"success_request_count"`
	DemandTokens          int64   `json:"demand_tokens"`
	PeakTokens            int64   `json:"peak_tokens"`
	PeakRatio             float64 `json:"peak_ratio"`
	UniqueSessions        int64   `json:"unique_sessions"`
	CacheHitCount         int64   `json:"cache_hit_count"`
	CacheHitRate          float64 `json:"cache_hit_rate"`
	TotalCachedTokens     int64   `json:"total_cached_tokens"`
	SlaMetRate            float64 `json:"sla_met_rate"`
	TotalSellQuota        int64   `json:"total_sell_quota"`
	TotalCostQuota        int64   `json:"total_cost_quota"`
	GrossProfitQuota      int64   `json:"gross_profit_quota"`
	SupplyCapacityTokens  int64   `json:"supply_capacity_tokens"`
	SupplyUsedTokens      int64   `json:"supply_used_tokens"`
	SupplyHeadroomTokens  int64   `json:"supply_headroom_tokens"`
	AvgSupplyQualityScore float64 `json:"avg_supply_quality_score"`
	AvgUnitCostQuota      float64 `json:"avg_unit_cost_quota"`
}

type trafficForecastItem struct {
	Id                     int     `json:"id"`
	ForecastKey            string  `json:"forecast_key"`
	SliceKey               string  `json:"slice_key"`
	ModelName              string  `json:"model_name"`
	SlaTier                string  `json:"sla_tier"`
	UserId                 int     `json:"user_id"`
	SourcePeriodStart      int64   `json:"source_period_start"`
	SourcePeriodEnd        int64   `json:"source_period_end"`
	TargetPeriodStart      int64   `json:"target_period_start"`
	TargetPeriodEnd        int64   `json:"target_period_end"`
	SourceProfileCount     int64   `json:"source_profile_count"`
	ObservedRequestCount   int64   `json:"observed_request_count"`
	ObservedDemandTokens   int64   `json:"observed_demand_tokens"`
	ObservedPeakTokens     int64   `json:"observed_peak_tokens"`
	BaselineDemandTokens   int64   `json:"baseline_demand_tokens"`
	ForecastDemandTokens   int64   `json:"forecast_demand_tokens"`
	ForecastPeakTokens     int64   `json:"forecast_peak_tokens"`
	ForecastHeadroomTokens int64   `json:"forecast_headroom_tokens"`
	ForecastGapTokens      int64   `json:"forecast_gap_tokens"`
	TrendDemandDeltaTokens int64   `json:"trend_demand_delta_tokens"`
	TrendDemandDeltaRate   float64 `json:"trend_demand_delta_rate"`
	SeasonalPeriodCount    int     `json:"seasonal_period_count"`
	SeasonalIndex          float64 `json:"seasonal_index"`
	SeasonalDemandTokens   int64   `json:"seasonal_demand_tokens"`
	AnomalyStatus          string  `json:"anomaly_status"`
	AnomalyProfileId       int     `json:"anomaly_profile_id"`
	AnomalyDemandRatio     float64 `json:"anomaly_demand_ratio"`
	CacheHitRate           float64 `json:"cache_hit_rate"`
	SlaMetRate             float64 `json:"sla_met_rate"`
	GrossProfitQuota       int64   `json:"gross_profit_quota"`
	AvgUnitCostQuota       float64 `json:"avg_unit_cost_quota"`
	Confidence             float64 `json:"confidence"`
	Method                 string  `json:"method"`
}

type pricingRecommendationItem struct {
	Id                        int     `json:"id"`
	RecommendationKey         string  `json:"recommendation_key"`
	TrafficProfileId          int     `json:"traffic_profile_id"`
	SliceKey                  string  `json:"slice_key"`
	ModelName                 string  `json:"model_name"`
	SlaTier                   string  `json:"sla_tier"`
	UserId                    int     `json:"user_id"`
	Status                    string  `json:"status"`
	Action                    string  `json:"action"`
	RequestCount              int64   `json:"request_count"`
	DemandTokens              int64   `json:"demand_tokens"`
	PeakTokens                int64   `json:"peak_tokens"`
	SupplyHeadroomTokens      int64   `json:"supply_headroom_tokens"`
	CacheHitRate              float64 `json:"cache_hit_rate"`
	SlaMetRate                float64 `json:"sla_met_rate"`
	TotalSellQuota            int64   `json:"total_sell_quota"`
	TotalCostQuota            int64   `json:"total_cost_quota"`
	GrossProfitQuota          int64   `json:"gross_profit_quota"`
	CurrentUnitPriceQuota     float64 `json:"current_unit_price_quota"`
	CurrentUnitCostQuota      float64 `json:"current_unit_cost_quota"`
	CurrentMarginRate         float64 `json:"current_margin_rate"`
	RecommendedUnitPriceQuota float64 `json:"recommended_unit_price_quota"`
	RecommendedMarginRate     float64 `json:"recommended_margin_rate"`
	AvgSupplyQualityScore     float64 `json:"avg_supply_quality_score"`
	AvgUnitCostQuota          float64 `json:"avg_unit_cost_quota"`
	Reason                    string  `json:"reason"`
	ReviewedAt                int64   `json:"reviewed_at"`
	ReviewedBy                int     `json:"reviewed_by"`
	ReviewNote                string  `json:"review_note"`
}

type supplyDecisionItem struct {
	Id                    int     `json:"id"`
	DecisionKey           string  `json:"decision_key"`
	TrafficProfileId      int     `json:"traffic_profile_id"`
	TrafficForecastId     int     `json:"traffic_forecast_id"`
	DecisionSource        string  `json:"decision_source"`
	SliceKey              string  `json:"slice_key"`
	ModelName             string  `json:"model_name"`
	SlaTier               string  `json:"sla_tier"`
	UserId                int     `json:"user_id"`
	ForecastTargetStart   int64   `json:"forecast_target_period_start"`
	ForecastTargetEnd     int64   `json:"forecast_target_period_end"`
	ForecastConfidence    float64 `json:"forecast_confidence"`
	ForecastMethod        string  `json:"forecast_method"`
	DecisionType          string  `json:"decision_type"`
	Track                 string  `json:"track"`
	Status                string  `json:"status"`
	DemandTokens          int64   `json:"demand_tokens"`
	PeakTokens            int64   `json:"peak_tokens"`
	SupplyHeadroomTokens  int64   `json:"supply_headroom_tokens"`
	GapTokens             int64   `json:"gap_tokens"`
	RecommendedCapacity   int64   `json:"recommended_capacity"`
	CacheHitRate          float64 `json:"cache_hit_rate"`
	SlaMetRate            float64 `json:"sla_met_rate"`
	GrossProfitQuota      int64   `json:"gross_profit_quota"`
	AvgSupplyQualityScore float64 `json:"avg_supply_quality_score"`
	AvgUnitCostQuota      float64 `json:"avg_unit_cost_quota"`
	RoiScore              float64 `json:"roi_score"`
	ReviewedAt            int64   `json:"reviewed_at"`
	ReviewedBy            int     `json:"reviewed_by"`
	ReviewNote            string  `json:"review_note"`
}

type supplyExpansionOpportunityItem struct {
	Id                         int     `json:"id"`
	OpportunityKey             string  `json:"opportunity_key"`
	SupplyDecisionId           int     `json:"supply_decision_id"`
	TrafficProfileId           int     `json:"traffic_profile_id"`
	TrafficForecastId          int     `json:"traffic_forecast_id"`
	DecisionSource             string  `json:"decision_source"`
	DecisionStatus             string  `json:"decision_status"`
	SliceKey                   string  `json:"slice_key"`
	ModelName                  string  `json:"model_name"`
	SlaTier                    string  `json:"sla_tier"`
	UserId                     int     `json:"user_id"`
	ForecastTargetStart        int64   `json:"forecast_target_period_start"`
	ForecastTargetEnd          int64   `json:"forecast_target_period_end"`
	ForecastConfidence         float64 `json:"forecast_confidence"`
	ForecastMethod             string  `json:"forecast_method"`
	OpportunityType            string  `json:"opportunity_type"`
	Track                      string  `json:"track"`
	DecisionType               string  `json:"decision_type"`
	Priority                   string  `json:"priority"`
	ClusterKey                 string  `json:"cluster_key"`
	DemandTokens               int64   `json:"demand_tokens"`
	PeakTokens                 int64   `json:"peak_tokens"`
	SupplyHeadroomTokens       int64   `json:"supply_headroom_tokens"`
	GapTokens                  int64   `json:"gap_tokens"`
	RecommendedCapacity        int64   `json:"recommended_capacity"`
	CacheHitRate               float64 `json:"cache_hit_rate"`
	SlaMetRate                 float64 `json:"sla_met_rate"`
	GrossProfitQuota           int64   `json:"gross_profit_quota"`
	AvgSupplyQualityScore      float64 `json:"avg_supply_quality_score"`
	AvgUnitCostQuota           float64 `json:"avg_unit_cost_quota"`
	RoiScore                   float64 `json:"roi_score"`
	SelfHostedCostProfileId    int     `json:"self_hosted_cost_profile_id"`
	SelfHostedUnitCostQuota    float64 `json:"self_hosted_unit_cost_quota"`
	SelfHostedSavingsUnitQuota float64 `json:"self_hosted_savings_unit_quota"`
	SelfHostedSavingsQuota     float64 `json:"self_hosted_savings_quota"`
	PeakRatio                  float64 `json:"peak_ratio"`
	UniqueSessions             int64   `json:"unique_sessions"`
	LocalityScore              float64 `json:"locality_score"`
	StabilityScore             float64 `json:"stability_score"`
	HeadroomRiskScore          float64 `json:"headroom_risk_score"`
	RankScore                  float64 `json:"rank_score"`
	Reason                     string  `json:"reason"`
}

type operatingInsightItem struct {
	Id                          int     `json:"id"`
	InsightKey                  string  `json:"insight_key"`
	TrafficProfileId            int     `json:"traffic_profile_id"`
	SupplyDecisionId            int     `json:"supply_decision_id"`
	PricingRecommendationId     int     `json:"pricing_recommendation_id"`
	SliceKey                    string  `json:"slice_key"`
	ModelName                   string  `json:"model_name"`
	SlaTier                     string  `json:"sla_tier"`
	UserId                      int     `json:"user_id"`
	Status                      string  `json:"status"`
	Severity                    string  `json:"severity"`
	Category                    string  `json:"category"`
	Title                       string  `json:"title"`
	Summary                     string  `json:"summary"`
	RecommendedAction           string  `json:"recommended_action"`
	DemandTokens                int64   `json:"demand_tokens"`
	PeakTokens                  int64   `json:"peak_tokens"`
	SupplyHeadroomTokens        int64   `json:"supply_headroom_tokens"`
	CacheHitRate                float64 `json:"cache_hit_rate"`
	SlaMetRate                  float64 `json:"sla_met_rate"`
	GrossProfitQuota            int64   `json:"gross_profit_quota"`
	AvgUnitCostQuota            float64 `json:"avg_unit_cost_quota"`
	SupplyDecisionTrack         string  `json:"supply_decision_track"`
	SupplyDecisionType          string  `json:"supply_decision_type"`
	SupplyDecisionStatus        string  `json:"supply_decision_status"`
	SupplyDecisionRoiScore      float64 `json:"supply_decision_roi_score"`
	PricingRecommendationAction string  `json:"pricing_recommendation_action"`
	PricingRecommendationStatus string  `json:"pricing_recommendation_status"`
	RecommendedUnitPriceQuota   float64 `json:"recommended_unit_price_quota"`
	RecommendedMarginRate       float64 `json:"recommended_margin_rate"`
	SlaProbeRunId               int     `json:"sla_probe_run_id"`
	ReviewedAt                  int64   `json:"reviewed_at"`
	ReviewedBy                  int     `json:"reviewed_by"`
	ReviewNote                  string  `json:"review_note"`
}

type supplyActionPlanItem struct {
	Id                           int     `json:"id"`
	SupplyDecisionId             int     `json:"supply_decision_id"`
	DecisionKey                  string  `json:"decision_key"`
	SupplyExpansionOpportunityId int     `json:"supply_expansion_opportunity_id"`
	OpportunityKey               string  `json:"opportunity_key"`
	OpportunityType              string  `json:"opportunity_type"`
	OpportunityPriority          string  `json:"opportunity_priority"`
	OpportunityClusterKey        string  `json:"opportunity_cluster_key"`
	OpportunityRankScore         float64 `json:"opportunity_rank_score"`
	TrafficProfileId             int     `json:"traffic_profile_id"`
	SliceKey                     string  `json:"slice_key"`
	ModelName                    string  `json:"model_name"`
	SlaTier                      string  `json:"sla_tier"`
	UserId                       int     `json:"user_id"`
	DecisionType                 string  `json:"decision_type"`
	Track                        string  `json:"track"`
	ActionType                   string  `json:"action_type"`
	Status                       string  `json:"status"`
	RecommendedCapacity          int64   `json:"recommended_capacity"`
	GapTokens                    int64   `json:"gap_tokens"`
	RoiScore                     float64 `json:"roi_score"`
	SourceReviewedAt             int64   `json:"source_reviewed_at"`
	SourceReviewedBy             int     `json:"source_reviewed_by"`
	StartedAt                    int64   `json:"started_at"`
	CompletedAt                  int64   `json:"completed_at"`
	CancelledAt                  int64   `json:"cancelled_at"`
	StatusUpdatedAt              int64   `json:"status_updated_at"`
	StatusUpdatedBy              int     `json:"status_updated_by"`
	OperatorNote                 string  `json:"operator_note"`
}

type supplyActionExecutionItem struct {
	Id                    int     `json:"id"`
	SupplyActionPlanId    int     `json:"supply_action_plan_id"`
	SupplyDecisionId      int     `json:"supply_decision_id"`
	DecisionKey           string  `json:"decision_key"`
	TrafficProfileId      int     `json:"traffic_profile_id"`
	SliceKey              string  `json:"slice_key"`
	ModelName             string  `json:"model_name"`
	SlaTier               string  `json:"sla_tier"`
	UserId                int     `json:"user_id"`
	DecisionType          string  `json:"decision_type"`
	Track                 string  `json:"track"`
	ActionType            string  `json:"action_type"`
	ExecutionStatus       string  `json:"execution_status"`
	SupplierId            int     `json:"supplier_id"`
	ChannelId             int     `json:"channel_id"`
	SupplyCapacityId      int     `json:"supply_capacity_id"`
	RecommendedCapacity   int64   `json:"recommended_capacity"`
	ActualCapacityTokens  int64   `json:"actual_capacity_tokens"`
	GapTokens             int64   `json:"gap_tokens"`
	RoiScore              float64 `json:"roi_score"`
	UnitCostQuota         float64 `json:"unit_cost_quota"`
	DrawdownTokens        int64   `json:"drawdown_tokens"`
	DrawdownRequestCount  int64   `json:"drawdown_request_count"`
	RemainingTokens       int64   `json:"remaining_tokens"`
	DrawdownRate          float64 `json:"drawdown_rate"`
	DrawdownSourceType    string  `json:"drawdown_source_type"`
	DrawdownSourceRef     string  `json:"drawdown_source_ref"`
	DrawdownRefreshedAt   int64   `json:"drawdown_refreshed_at"`
	EffectiveFrom         int64   `json:"effective_from"`
	EffectiveTo           int64   `json:"effective_to"`
	ExternalRef           string  `json:"external_ref"`
	OperatorNote          string  `json:"operator_note"`
	ActionPlanCompletedAt int64   `json:"action_plan_completed_at"`
	ActionPlanCompletedBy int     `json:"action_plan_completed_by"`
	RecordedAt            int64   `json:"recorded_at"`
	RecordedBy            int     `json:"recorded_by"`
}

type supplyRoutingPolicyItem struct {
	Id                      int    `json:"id"`
	SupplyActionExecutionId int    `json:"supply_action_execution_id"`
	SupplyActionPlanId      int    `json:"supply_action_plan_id"`
	SupplyDecisionId        int    `json:"supply_decision_id"`
	ModelName               string `json:"model_name"`
	SlaTier                 string `json:"sla_tier"`
	UserId                  int    `json:"user_id"`
	Track                   string `json:"track"`
	ActionType              string `json:"action_type"`
	Status                  string `json:"status"`
	SupplierId              int    `json:"supplier_id"`
	ChannelId               int    `json:"channel_id"`
	SupplyCapacityId        int    `json:"supply_capacity_id"`
	SlaContractId           int    `json:"sla_contract_id"`
	SlaProbeRunId           int    `json:"sla_probe_run_id"`
	SlaProbeRunKey          string `json:"sla_probe_run_key"`
	SlaArtifactSHA256       string `json:"sla_artifact_sha256"`
	SlaRuntimeRef           string `json:"sla_runtime_ref"`
	EffectiveFrom           int64  `json:"effective_from"`
	EffectiveTo             int64  `json:"effective_to"`
	Priority                int    `json:"priority"`
	TrafficPercent          int    `json:"traffic_percent"`
	ActivatedAt             int64  `json:"activated_at"`
	ActivatedBy             int    `json:"activated_by"`
	DisabledAt              int64  `json:"disabled_at"`
	DisabledBy              int    `json:"disabled_by"`
	OperatorNote            string `json:"operator_note"`
}

func verifySettlement(client *http.Client, baseURL string, adminToken string, ledgers []ledgerEnvelopeItem) {
	totals := expectedTotalsFromLedgers(ledgers)
	summary := getMarginSummary(client, baseURL, adminToken, totals.SupplierId)
	if len(summary) != 1 {
		log.Fatalf("expected one margin summary row, got %d", len(summary))
	}
	assertMarginSummary(summary[0], totals)
	statement := generateSupplierStatement(client, baseURL, adminToken, totals.SupplierId)
	assertStatement(statement, totals)
	items := getSettlementItems(client, baseURL, adminToken, statement.Id)
	if len(items) != len(ledgers) {
		log.Fatalf("expected %d settlement items, got %d", len(ledgers), len(items))
	}
	csvBody := getSettlementCSV(client, baseURL, adminToken, statement.Id)
	if !strings.Contains(csvBody, "request_id,session_id,supplier_id") ||
		!strings.Contains(csvBody, "process-e2e-request-1") ||
		!strings.Contains(csvBody, "process-e2e-request-2") {
		log.Fatalf("settlement CSV missing expected rows")
	}
}

func verifySupplyCapacity(client *http.Client, baseURL string, adminToken string, ledgers []ledgerEnvelopeItem, modelName string) {
	totals := expectedTotalsFromLedgers(ledgers)
	expectedUsedTokens := totals.TotalPromptTokens + totals.TotalCompletionTokens
	refreshed := refreshSupplyCapacityUsage(client, baseURL, adminToken, totals.SupplierId, modelName)
	if len(refreshed) != 1 ||
		refreshed[0].UsedTokens != expectedUsedTokens ||
		refreshed[0].HeadroomTokens != 1000-expectedUsedTokens ||
		refreshed[0].UtilizationRate < 0.299999 ||
		refreshed[0].UtilizationRate > 0.300001 {
		log.Fatalf("unexpected refreshed supply capacity usage: got=%+v expected_used=%d", refreshed, expectedUsedTokens)
	}
	supplierID := totals.SupplierId
	capacities := getSupplyCapacities(client, baseURL, adminToken, supplierID, modelName)
	if len(capacities) != 1 {
		log.Fatalf("expected one supply capacity row, got %d", len(capacities))
	}
	capacity := capacities[0]
	if capacity.SupplierId != supplierID ||
		capacity.SupplyNode != "gb10-4t" ||
		capacity.ModelName != modelName ||
		capacity.CapacityTokens != 1000 ||
		capacity.UsedTokens != expectedUsedTokens ||
		capacity.HeadroomTokens != 1000-expectedUsedTokens ||
		capacity.UtilizationRate < 0.299999 ||
		capacity.UtilizationRate > 0.300001 ||
		capacity.QualityScore < 98.499999 ||
		capacity.QualityScore > 98.500001 ||
		capacity.UnitCostQuota < 0.499999 ||
		capacity.UnitCostQuota > 0.500001 {
		log.Fatalf("unexpected supply capacity: got=%+v", capacity)
	}
	telemetry := sweepSupplyCapacityTelemetry(client, baseURL, adminToken, supplierID, modelName, capacity)
	telemetries := getSupplyCapacityTelemetries(client, baseURL, adminToken, supplierID, modelName)
	if len(telemetries) != 1 ||
		telemetries[0].Id != telemetry.Id ||
		telemetries[0].AppliedCapacityId != capacity.Id ||
		telemetries[0].UsedTokens != expectedUsedTokens ||
		telemetries[0].SourceRef != "gb10-4t-mock-capacity" ||
		telemetries[0].GpuUtilizationRate < 0.619999 ||
		telemetries[0].GpuUtilizationRate > 0.620001 {
		log.Fatalf("unexpected supply capacity telemetries: got=%+v recorded=%+v capacity=%+v", telemetries, telemetry, capacity)
	}
	capacities = getSupplyCapacities(client, baseURL, adminToken, supplierID, modelName)
	if len(capacities) != 1 ||
		capacities[0].LastTelemetryId != telemetry.Id ||
		capacities[0].TelemetrySourceType != model.SupplyCapacityTelemetrySourceNodeReport ||
		capacities[0].TelemetrySourceRef != "gb10-4t-mock-capacity" ||
		capacities[0].TelemetryObservedAt != telemetry.ObservedAt ||
		capacities[0].GpuUtilizationRate < 0.619999 ||
		capacities[0].GpuUtilizationRate > 0.620001 {
		log.Fatalf("unexpected telemetry-backed supply capacity: got=%+v telemetry=%+v", capacities, telemetry)
	}
}

func refreshSupplyCapacityUsage(client *http.Client, baseURL string, adminToken string, supplierID int, modelName string) []supplyCapacityItem {
	return postAdminJSON[[]supplyCapacityItem](client, baseURL, adminToken, "/api/supply_capacities/refresh_usage", model.SupplyCapacityUsageRefreshInput{
		SupplierId: supplierID,
		SupplyNode: "gb10-4t",
		ModelName:  modelName,
	}, "supply capacity usage refresh")
}

func getSupplyCapacities(client *http.Client, baseURL string, adminToken string, supplierID int, modelName string) []supplyCapacityItem {
	values := url.Values{}
	values.Set("supplier_id", fmt.Sprintf("%d", supplierID))
	values.Set("supply_node", "gb10-4t")
	values.Set("model_name", modelName)
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supply_capacities?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply capacity query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplyCapacityItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply capacity query returned success=false")
	}
	return envelope.Data.Items
}

func sweepSupplyCapacityTelemetry(client *http.Client, baseURL string, adminToken string, supplierID int, modelName string, capacity supplyCapacityItem) supplyCapacityTelemetryItem {
	result := postAdminJSON[supplyCapacityTelemetrySweepResult](client, baseURL, adminToken, "/api/supply_capacity_telemetries/sweep", model.SupplyCapacityTelemetrySweepInput{
		SupplierId:  supplierID,
		SupplyNode:  "gb10-4t",
		ModelName:   modelName,
		PeriodStart: capacity.PeriodStart,
		PeriodEnd:   capacity.PeriodEnd,
	}, "supply capacity telemetry sweep")
	if result.AttemptedCount != 1 || result.CollectedCount != 1 || result.SkippedCount != 0 || len(result.Collected) != 1 {
		log.Fatalf("unexpected supply capacity telemetry sweep result: got=%+v capacity=%+v", result, capacity)
	}
	return result.Collected[0]
}

func recordSupplyCapacityTelemetryInput(client *http.Client, baseURL string, adminToken string, input model.SupplyCapacityTelemetryRecordInput) supplyCapacityTelemetryItem {
	return postAdminJSON[supplyCapacityTelemetryItem](client, baseURL, adminToken, "/api/supply_capacity_telemetries/record", input, "supply capacity telemetry record")
}

func getSupplyCapacityTelemetries(client *http.Client, baseURL string, adminToken string, supplierID int, modelName string) []supplyCapacityTelemetryItem {
	values := url.Values{}
	values.Set("supplier_id", fmt.Sprintf("%d", supplierID))
	values.Set("supply_node", "gb10-4t")
	values.Set("model_name", modelName)
	values.Set("source_type", model.SupplyCapacityTelemetrySourceNodeReport)
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supply_capacity_telemetries?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply capacity telemetry query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplyCapacityTelemetryItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply capacity telemetry query returned success=false")
	}
	return envelope.Data.Items
}

func createSupplyCapacity(client *http.Client, baseURL string, adminToken string, supplierID int, supplyNode string, modelName string, periodStart int64, periodEnd int64, capacityTokens int64, usedTokens int64, qualityScore float64, unitCostQuota float64) supplyCapacityItem {
	payload, err := json.Marshal(map[string]any{
		"supplier_id":     supplierID,
		"supply_node":     supplyNode,
		"model_name":      modelName,
		"period_start":    periodStart,
		"period_end":      periodEnd,
		"capacity_tokens": capacityTokens,
		"used_tokens":     usedTokens,
		"quality_score":   qualityScore,
		"unit_cost_quota": unitCostQuota,
		"status":          1,
		"notes":           "process e2e self-hosted routing capacity",
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/supply_capacities", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply capacity create failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool               `json:"success"`
		Data    supplyCapacityItem `json:"data"`
		Message string             `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply capacity create returned success=false message=%s", envelope.Message)
	}
	return envelope.Data
}

func verifySupplierScorecard(client *http.Client, baseURL string, adminToken string, ledgers []ledgerEnvelopeItem, expectSlaEvidence bool, expectedSlaRunKey string) {
	expected := expectedTotalsFromLedgers(ledgers)
	generated := generateSupplierScorecards(client, baseURL, adminToken)
	if len(generated) != 1 {
		log.Fatalf("expected one generated supplier scorecard, got %d", len(generated))
	}
	assertSupplierScorecard(generated[0], expected)

	queried := getSupplierScorecards(client, baseURL, adminToken, expected.SupplierId, model.SupplierScorecardGradeA)
	if len(queried) != 1 {
		log.Fatalf("expected one queried supplier scorecard, got %d", len(queried))
	}
	if queried[0].SupplierId != generated[0].SupplierId ||
		queried[0].TotalRequests != generated[0].TotalRequests ||
		queried[0].Grade != generated[0].Grade {
		log.Fatalf("queried supplier scorecard mismatch: got=%+v generated=%+v", queried[0], generated[0])
	}
	verifySupplierEvaluation(client, baseURL, adminToken, generated[0], expectSlaEvidence, expectedSlaRunKey)
	verifySupplierPostureRoutePreference(client, baseURL, adminToken, generated[0])
}

func generateSupplierScorecards(client *http.Client, baseURL string, adminToken string) []supplierScorecardItem {
	now := time.Now().Unix()
	payload, err := json.Marshal(map[string]any{
		"period_start": now - 3600,
		"period_end":   now + 3600,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/supplier_scorecards/generate", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supplier scorecard generate failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                    `json:"success"`
		Data    []supplierScorecardItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supplier scorecard generate returned success=false")
	}
	return envelope.Data
}

func getSupplierScorecards(client *http.Client, baseURL string, adminToken string, supplierID int, grade string) []supplierScorecardItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("supplier_id", fmt.Sprintf("%d", supplierID))
	values.Set("grade", grade)
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supplier_scorecards?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supplier scorecard query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplierScorecardItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supplier scorecard query returned success=false")
	}
	return envelope.Data.Items
}

func verifySupplierEvaluation(client *http.Client, baseURL string, adminToken string, scorecard supplierScorecardItem, expectSlaEvidence bool, expectedSlaRunKey string) {
	generated := generateSupplierEvaluations(client, baseURL, adminToken, scorecard.PeriodStart, scorecard.PeriodEnd)
	if len(generated) != 1 {
		log.Fatalf("expected one generated supplier evaluation, got %d", len(generated))
	}
	assertSupplierEvaluation(generated[0], scorecard, model.SupplierEvaluationStatusDraft, expectSlaEvidence, expectedSlaRunKey)

	queried := getSupplierEvaluations(client, baseURL, adminToken, scorecard.SupplierId, model.SupplierEvaluationStatusDraft)
	if len(queried) != 1 || queried[0].Id != generated[0].Id {
		log.Fatalf("queried supplier evaluation mismatch: got=%+v generated=%+v", queried, generated[0])
	}
	approved := approveSupplierEvaluation(client, baseURL, adminToken, generated[0].Id)
	assertSupplierEvaluation(approved, scorecard, model.SupplierEvaluationStatusApproved, expectSlaEvidence, expectedSlaRunKey)
	if approved.ReviewedBy != 1 ||
		approved.ReviewedAt <= 0 ||
		approved.ReviewNote != "accepted admission evaluation in process e2e" {
		log.Fatalf("unexpected approved supplier evaluation review fields: got=%+v", approved)
	}
	supplier := getSupplier(client, baseURL, adminToken, scorecard.SupplierId)
	if supplier.Status != common.ChannelStatusEnabled ||
		supplier.Name != "gb10-4t" ||
		supplier.Type != model.SupplierTypeThirdParty {
		log.Fatalf("supplier evaluation should not mutate supplier: got=%+v", supplier)
	}
	applied := applySupplierEvaluation(client, baseURL, adminToken, approved.Id)
	if applied.AppliedBy != 1 ||
		applied.AppliedAt <= 0 ||
		applied.SupplierStatusBefore != common.ChannelStatusEnabled ||
		applied.SupplierStatusAfter != common.ChannelStatusEnabled ||
		!strings.Contains(applied.AppliedNote, "applied admission evaluation in process e2e") {
		log.Fatalf("unexpected applied supplier evaluation fields: got=%+v", applied)
	}
	appliedSupplier := getSupplier(client, baseURL, adminToken, scorecard.SupplierId)
	if appliedSupplier.Status != common.ChannelStatusEnabled ||
		!strings.Contains(appliedSupplier.Notes, "supplier_evaluation #") {
		log.Fatalf("supplier evaluation apply should audit supplier posture: got=%+v", appliedSupplier)
	}
	regenerated := generateSupplierEvaluations(client, baseURL, adminToken, scorecard.PeriodStart, scorecard.PeriodEnd)
	if len(regenerated) != 1 ||
		regenerated[0].Id != approved.Id ||
		regenerated[0].Status != model.SupplierEvaluationStatusApproved ||
		regenerated[0].ReviewedBy != 1 ||
		regenerated[0].AppliedAt != applied.AppliedAt ||
		regenerated[0].AppliedBy != applied.AppliedBy {
		log.Fatalf("supplier evaluation regenerate should preserve review/apply: got=%+v approved=%+v applied=%+v", regenerated, approved, applied)
	}
}

func generateSupplierEvaluations(client *http.Client, baseURL string, adminToken string, periodStart int64, periodEnd int64) []supplierEvaluationItem {
	payload, err := json.Marshal(map[string]any{
		"period_start": periodStart,
		"period_end":   periodEnd,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/supplier_evaluations/generate", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supplier evaluation generate failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                     `json:"success"`
		Data    []supplierEvaluationItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supplier evaluation generate returned success=false")
	}
	return envelope.Data
}

func getSupplierEvaluations(client *http.Client, baseURL string, adminToken string, supplierID int, status string) []supplierEvaluationItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("supplier_id", fmt.Sprintf("%d", supplierID))
	values.Set("status", status)
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supplier_evaluations?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supplier evaluation query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplierEvaluationItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supplier evaluation query returned success=false")
	}
	return envelope.Data.Items
}

func verifySupplierPostureRoutePreference(client *http.Client, baseURL string, adminToken string, scorecard supplierScorecardItem) {
	generated := generateSupplierPostureRecommendations(client, baseURL, adminToken, scorecard.PeriodStart, scorecard.PeriodEnd)
	var recommendation *supplierPostureRecommendationItem
	for index := range generated {
		if generated[index].SupplierId == scorecard.SupplierId {
			recommendation = &generated[index]
			break
		}
	}
	if recommendation == nil {
		log.Fatalf("expected supplier posture recommendation for scorecard=%+v got=%+v", scorecard, generated)
	}
	assertSupplierPostureRecommendation(*recommendation, scorecard, model.SupplierPostureRecommendationStatusDraft)

	queried := getSupplierPostureRecommendations(client, baseURL, adminToken, scorecard.SupplierId, model.SupplierPostureRecommendationStatusDraft, model.SupplierPostureRecommendationActionBoost)
	if len(queried) != 1 || queried[0].Id != recommendation.Id {
		log.Fatalf("queried boost posture recommendation mismatch: got=%+v generated=%+v", queried, recommendation)
	}

	approved := approveSupplierPostureRecommendation(client, baseURL, adminToken, recommendation.Id)
	assertSupplierPostureRecommendation(approved, scorecard, model.SupplierPostureRecommendationStatusApproved)
	if approved.ReviewedBy != 1 ||
		approved.ReviewedAt <= 0 ||
		approved.ReviewNote != "approved posture boost in process e2e" {
		log.Fatalf("unexpected approved supplier posture recommendation: got=%+v", approved)
	}

	applied := applySupplierPostureRecommendation(client, baseURL, adminToken, recommendation.Id)
	assertSupplierPostureRecommendation(applied, scorecard, model.SupplierPostureRecommendationStatusApplied)
	if applied.AppliedBy != 1 ||
		applied.AppliedAt <= 0 ||
		applied.SupplierStatusBefore != common.ChannelStatusEnabled ||
		applied.SupplierStatusAfter != common.ChannelStatusEnabled ||
		!strings.Contains(applied.AppliedNote, "applied posture boost in process e2e") {
		log.Fatalf("unexpected applied supplier posture recommendation: got=%+v", applied)
	}

	preferences := getSupplierRoutePreferences(client, baseURL, adminToken, scorecard.SupplierId, model.SupplierRoutePreferenceStatusActive)
	if len(preferences) != 1 {
		log.Fatalf("expected one active supplier route preference, got=%+v", preferences)
	}
	preference := preferences[0]
	if preference.SupplierId != scorecard.SupplierId ||
		preference.SourcePostureRecommendationId != applied.Id ||
		preference.Status != model.SupplierRoutePreferenceStatusActive ||
		preference.WeightPercent != model.SupplierRoutePreferenceBoostWeightPercent ||
		preference.ActivatedBy != 1 ||
		preference.ActivatedAt <= 0 ||
		!strings.Contains(preference.Reason, "boost") ||
		!strings.Contains(preference.OperatorNote, "applied posture boost in process e2e") {
		log.Fatalf("unexpected active supplier route preference: got=%+v applied=%+v", preference, applied)
	}
}

func generateSupplierPostureRecommendations(client *http.Client, baseURL string, adminToken string, periodStart int64, periodEnd int64) []supplierPostureRecommendationItem {
	return postAdminJSON[[]supplierPostureRecommendationItem](client, baseURL, adminToken, "/api/supplier_posture_recommendations/generate", map[string]any{
		"period_start": periodStart,
		"period_end":   periodEnd,
	}, "supplier posture recommendation generate")
}

func getSupplierPostureRecommendations(client *http.Client, baseURL string, adminToken string, supplierID int, status string, action string) []supplierPostureRecommendationItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("supplier_id", fmt.Sprintf("%d", supplierID))
	if status != "" {
		values.Set("status", status)
	}
	if action != "" {
		values.Set("recommended_action", action)
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supplier_posture_recommendations?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supplier posture recommendation query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplierPostureRecommendationItem `json:"items"`
		} `json:"data"`
		Message string `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supplier posture recommendation query returned success=false message=%s", envelope.Message)
	}
	return envelope.Data.Items
}

func approveSupplierPostureRecommendation(client *http.Client, baseURL string, adminToken string, recommendationID int) supplierPostureRecommendationItem {
	return postAdminJSON[supplierPostureRecommendationItem](client, baseURL, adminToken, fmt.Sprintf("/api/supplier_posture_recommendations/%d/approve", recommendationID), map[string]any{
		"review_note": "approved posture boost in process e2e",
	}, "supplier posture recommendation approve")
}

func applySupplierPostureRecommendation(client *http.Client, baseURL string, adminToken string, recommendationID int) supplierPostureRecommendationItem {
	path := fmt.Sprintf("/api/supplier_posture_recommendations/%d/apply", recommendationID)
	input := map[string]any{
		"operator_note": "applied posture boost in process e2e",
	}
	var lastMessage string
	for attempt := 1; attempt <= 5; attempt++ {
		applied, message, ok := postAdminJSONAttempt[supplierPostureRecommendationItem](client, baseURL, adminToken, path, input, "supplier posture recommendation apply")
		if ok {
			return applied
		}
		lastMessage = message
		if !isSQLiteBusyMessage(message) {
			log.Fatalf("supplier posture recommendation apply returned success=false message=%s", message)
		}
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}
	log.Fatalf("supplier posture recommendation apply still busy after retries: %s", lastMessage)
	return supplierPostureRecommendationItem{}
}

func getSupplierRoutePreferences(client *http.Client, baseURL string, adminToken string, supplierID int, status string) []supplierRoutePreferenceItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("supplier_id", fmt.Sprintf("%d", supplierID))
	if status != "" {
		values.Set("status", status)
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supplier_route_preferences?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supplier route preference query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplierRoutePreferenceItem `json:"items"`
		} `json:"data"`
		Message string `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supplier route preference query returned success=false message=%s", envelope.Message)
	}
	return envelope.Data.Items
}

func approveSupplierEvaluation(client *http.Client, baseURL string, adminToken string, evaluationID int) supplierEvaluationItem {
	payload, err := json.Marshal(map[string]any{
		"review_note": "accepted admission evaluation in process e2e",
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supplier_evaluations/%d/approve", strings.TrimRight(baseURL, "/"), evaluationID), bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supplier evaluation approve failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                   `json:"success"`
		Data    supplierEvaluationItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supplier evaluation approve returned success=false")
	}
	return envelope.Data
}

func applySupplierEvaluation(client *http.Client, baseURL string, adminToken string, evaluationID int) supplierEvaluationItem {
	payload, err := json.Marshal(map[string]any{
		"operator_note": "applied admission evaluation in process e2e",
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supplier_evaluations/%d/apply", strings.TrimRight(baseURL, "/"), evaluationID), bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supplier evaluation apply failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                   `json:"success"`
		Data    supplierEvaluationItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supplier evaluation apply returned success=false")
	}
	return envelope.Data
}

func getSupplier(client *http.Client, baseURL string, adminToken string, supplierID int) supplierItem {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/suppliers/%d", strings.TrimRight(baseURL, "/"), supplierID), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supplier query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool         `json:"success"`
		Data    supplierItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supplier query returned success=false")
	}
	return envelope.Data
}

func verifyTrafficProfile(client *http.Client, baseURL string, adminToken string, ledgers []ledgerEnvelopeItem, modelName string) trafficProfileItem {
	generated := generateTrafficProfiles(client, baseURL, adminToken)
	if len(generated) != 1 {
		log.Fatalf("expected one generated traffic profile, got %d", len(generated))
	}
	assertTrafficProfile(generated[0], expectedTotalsFromLedgers(ledgers), modelName)

	queried := getTrafficProfiles(client, baseURL, adminToken, modelName)
	if len(queried) != 1 {
		log.Fatalf("expected one queried traffic profile, got %d", len(queried))
	}
	if queried[0].SliceKey != generated[0].SliceKey ||
		queried[0].DemandTokens != generated[0].DemandTokens ||
		queried[0].SupplyHeadroomTokens != generated[0].SupplyHeadroomTokens {
		log.Fatalf("queried traffic profile mismatch: got=%+v generated=%+v", queried[0], generated[0])
	}
	return generated[0]
}

func generateTrafficProfiles(client *http.Client, baseURL string, adminToken string) []trafficProfileItem {
	now := time.Now().Unix()
	payload, err := json.Marshal(map[string]any{
		"period_start": now - 3600,
		"period_end":   now + 3600,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/traffic_profiles/generate", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("traffic profile generate failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                 `json:"success"`
		Data    []trafficProfileItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("traffic profile generate returned success=false")
	}
	return envelope.Data
}

func getTrafficProfiles(client *http.Client, baseURL string, adminToken string, modelName string) []trafficProfileItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("model_name", modelName)
	values.Set("sla_tier", "default")
	values.Set("user_id", "2")
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/traffic_profiles?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("traffic profile query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []trafficProfileItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("traffic profile query returned success=false")
	}
	return envelope.Data.Items
}

func verifyTrafficForecast(client *http.Client, baseURL string, adminToken string, profile trafficProfileItem, modelName string) trafficForecastItem {
	generated := generateTrafficForecasts(client, baseURL, adminToken, profile)
	if len(generated) != 1 {
		log.Fatalf("expected one generated traffic forecast, got %d", len(generated))
	}
	assertTrafficForecast(generated[0], profile, modelName)

	queried := getTrafficForecasts(client, baseURL, adminToken, modelName, generated[0].TargetPeriodStart, generated[0].TargetPeriodEnd)
	if len(queried) != 1 {
		log.Fatalf("expected one queried traffic forecast, got %d", len(queried))
	}
	if queried[0].ForecastKey != generated[0].ForecastKey ||
		queried[0].ForecastDemandTokens != generated[0].ForecastDemandTokens ||
		queried[0].ForecastGapTokens != generated[0].ForecastGapTokens {
		log.Fatalf("queried traffic forecast mismatch: got=%+v generated=%+v", queried[0], generated[0])
	}
	return generated[0]
}

func verifySeasonalAnomalyTrafficForecast(client *http.Client, baseURL string, adminToken string, modelName string) {
	sourceStart, sourceEnd, targetStart, targetEnd := seedSeasonalAnomalyTrafficProfiles(modelName)
	generated := postAdminJSON[[]trafficForecastItem](client, baseURL, adminToken, "/api/traffic_forecasts/generate", map[string]any{
		"period_start":           sourceStart,
		"period_end":             sourceEnd,
		"target_period_start":    targetStart,
		"target_period_end":      targetEnd,
		"model_name":             modelName,
		"sla_tier":               "seasonal",
		"user_id":                99,
		"seasonal_period_count":  2,
		"anomaly_guard":          true,
		"anomaly_threshold_rate": 1.8,
	}, "seasonal anomaly traffic forecast generate")
	if len(generated) != 1 {
		log.Fatalf("expected one seasonal anomaly traffic forecast, got %d", len(generated))
	}
	assertSeasonalAnomalyTrafficForecast(generated[0], modelName, sourceStart, sourceEnd, targetStart, targetEnd)

	queried := getTrafficForecastsForSlice(client, baseURL, adminToken, modelName, "seasonal", 99, targetStart, targetEnd)
	if len(queried) != 1 ||
		queried[0].ForecastKey != generated[0].ForecastKey ||
		queried[0].Method != model.TrafficForecastMethodSeasonalAnomaly ||
		queried[0].AnomalyStatus != model.TrafficForecastAnomalySpike {
		log.Fatalf("queried seasonal anomaly traffic forecast mismatch: got=%+v generated=%+v", queried, generated[0])
	}
}

func seedSeasonalAnomalyTrafficProfiles(modelName string) (int64, int64, int64, int64) {
	if err := initDBForSeed(); err != nil {
		log.Fatalf("init db for seasonal anomaly forecast profiles: %v", err)
	}
	defer func() { _ = model.CloseDB() }()

	sourceStart := common.GetTimestamp() - 20_000
	periodSeconds := int64(1_000)
	sliceKey := fmt.Sprintf("model:%s|sla:seasonal|user:99", strings.TrimSpace(modelName))
	demands := []int64{100, 300, 120, 360}
	peaks := []int64{130, 330, 160, 390}
	headrooms := []int64{500, 400, 350, 250}
	profiles := make([]*model.TrafficProfile, 0, len(demands))
	for index, demand := range demands {
		periodStart := sourceStart + int64(index)*periodSeconds
		periodEnd := periodStart + periodSeconds
		profiles = append(profiles, &model.TrafficProfile{
			SliceKey:             sliceKey,
			ModelName:            modelName,
			SlaTier:              "seasonal",
			UserId:               99,
			PeriodStart:          periodStart,
			PeriodEnd:            periodEnd,
			RequestCount:         1,
			SuccessRequestCount:  1,
			DemandTokens:         demand,
			PeakTokens:           peaks[index],
			CacheHitRate:         0.5,
			SlaMetRate:           0.95,
			GrossProfitQuota:     demand / 3,
			SupplyHeadroomTokens: headrooms[index],
			AvgUnitCostQuota:     0.5,
			GeneratedAt:          periodEnd,
			CreatedAt:            periodEnd,
			UpdatedAt:            periodEnd,
		})
	}
	if err := model.DB.Create(&profiles).Error; err != nil {
		log.Fatalf("seed seasonal anomaly traffic profiles: %v", err)
	}
	sourceEnd := sourceStart + int64(len(demands))*periodSeconds
	targetStart := sourceEnd
	targetEnd := targetStart + periodSeconds
	return sourceStart, sourceEnd, targetStart, targetEnd
}

func generateTrafficForecasts(client *http.Client, baseURL string, adminToken string, profile trafficProfileItem) []trafficForecastItem {
	payload, err := json.Marshal(map[string]any{
		"period_start": profile.PeriodStart,
		"period_end":   profile.PeriodEnd,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/traffic_forecasts/generate", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("traffic forecast generate failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                  `json:"success"`
		Data    []trafficForecastItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("traffic forecast generate returned success=false")
	}
	return envelope.Data
}

func getTrafficForecastsForSlice(client *http.Client, baseURL string, adminToken string, modelName string, slaTier string, userID int, targetStart int64, targetEnd int64) []trafficForecastItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("model_name", modelName)
	values.Set("sla_tier", slaTier)
	values.Set("user_id", fmt.Sprintf("%d", userID))
	values.Set("target_start_timestamp", fmt.Sprintf("%d", targetStart))
	values.Set("target_end_timestamp", fmt.Sprintf("%d", targetEnd))
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/traffic_forecasts?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("traffic forecast slice query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []trafficForecastItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("traffic forecast slice query returned success=false")
	}
	return envelope.Data.Items
}

func getTrafficForecasts(client *http.Client, baseURL string, adminToken string, modelName string, targetStart int64, targetEnd int64) []trafficForecastItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("model_name", modelName)
	values.Set("sla_tier", "default")
	values.Set("user_id", "2")
	values.Set("target_start_timestamp", fmt.Sprintf("%d", targetStart))
	values.Set("target_end_timestamp", fmt.Sprintf("%d", targetEnd))
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/traffic_forecasts?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("traffic forecast query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []trafficForecastItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("traffic forecast query returned success=false")
	}
	return envelope.Data.Items
}

func verifyPricingRecommendation(client *http.Client, baseURL string, adminToken string, profile trafficProfileItem, ledgers []ledgerEnvelopeItem, modelName string) {
	generated := generatePricingRecommendations(client, baseURL, adminToken, profile)
	if len(generated) != 1 {
		log.Fatalf("expected one generated pricing recommendation, got %d", len(generated))
	}
	assertPricingRecommendation(generated[0], profile, expectedTotalsFromLedgers(ledgers), modelName, model.PricingRecommendationStatusDraft)

	queried := getPricingRecommendations(client, baseURL, adminToken, modelName, model.PricingRecommendationStatusDraft)
	if len(queried) != 1 {
		log.Fatalf("expected one queried draft pricing recommendation, got %d", len(queried))
	}
	if queried[0].RecommendationKey != generated[0].RecommendationKey ||
		queried[0].Action != generated[0].Action ||
		queried[0].RecommendedUnitPriceQuota != generated[0].RecommendedUnitPriceQuota {
		log.Fatalf("queried pricing recommendation mismatch: got=%+v generated=%+v", queried[0], generated[0])
	}

	approved := approvePricingRecommendation(client, baseURL, adminToken, generated[0].Id)
	assertPricingRecommendation(approved, profile, expectedTotalsFromLedgers(ledgers), modelName, model.PricingRecommendationStatusApproved)
	if approved.ReviewedBy != 1 ||
		approved.ReviewedAt <= 0 ||
		approved.ReviewNote != "accepted pricing recommendation in process e2e" {
		log.Fatalf("unexpected approved pricing recommendation review fields: got=%+v", approved)
	}
	if len(getPricingRecommendations(client, baseURL, adminToken, modelName, model.PricingRecommendationStatusApproved)) != 1 {
		log.Fatalf("expected one approved pricing recommendation after review")
	}
	regenerated := generatePricingRecommendations(client, baseURL, adminToken, profile)
	if len(regenerated) != 1 ||
		regenerated[0].Status != model.PricingRecommendationStatusApproved ||
		regenerated[0].ReviewNote != approved.ReviewNote {
		log.Fatalf("pricing recommendation regenerate should preserve review: got=%+v approved=%+v", regenerated, approved)
	}
}

func generatePricingRecommendations(client *http.Client, baseURL string, adminToken string, profile trafficProfileItem) []pricingRecommendationItem {
	payload, err := json.Marshal(map[string]any{
		"period_start": profile.PeriodStart,
		"period_end":   profile.PeriodEnd,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/pricing_recommendations/generate", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("pricing recommendation generate failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                        `json:"success"`
		Data    []pricingRecommendationItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("pricing recommendation generate returned success=false")
	}
	return envelope.Data
}

func getPricingRecommendations(client *http.Client, baseURL string, adminToken string, modelName string, status string) []pricingRecommendationItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("model_name", modelName)
	values.Set("sla_tier", "default")
	values.Set("user_id", "2")
	values.Set("status", status)
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/pricing_recommendations?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("pricing recommendation query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []pricingRecommendationItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("pricing recommendation query returned success=false")
	}
	return envelope.Data.Items
}

func approvePricingRecommendation(client *http.Client, baseURL string, adminToken string, recommendationID int) pricingRecommendationItem {
	payload, err := json.Marshal(map[string]any{
		"review_note": "accepted pricing recommendation in process e2e",
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/pricing_recommendations/%d/approve", strings.TrimRight(baseURL, "/"), recommendationID), bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("pricing recommendation approve failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                      `json:"success"`
		Data    pricingRecommendationItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("pricing recommendation approve returned success=false")
	}
	return envelope.Data
}

func verifySupplyDecision(client *http.Client, baseURL string, adminToken string, demandToken string, sessionID string, profile trafficProfileItem, forecast trafficForecastItem, ledgers []ledgerEnvelopeItem, modelName string) []ledgerEnvelopeItem {
	generated := generateSupplyDecisions(client, baseURL, adminToken, profile)
	if len(generated) != 1 {
		log.Fatalf("expected one generated supply decision, got %d", len(generated))
	}
	assertSupplyDecision(generated[0], profile, forecast, expectedTotalsFromLedgers(ledgers), modelName, model.SupplyDecisionStatusDraft)

	queried := getSupplyDecisions(client, baseURL, adminToken, modelName, model.SupplyDecisionStatusDraft)
	if len(queried) != 1 {
		log.Fatalf("expected one queried draft supply decision, got %d", len(queried))
	}
	if queried[0].DecisionKey != generated[0].DecisionKey ||
		queried[0].RoiScore != generated[0].RoiScore ||
		queried[0].RecommendedCapacity != generated[0].RecommendedCapacity {
		log.Fatalf("queried supply decision mismatch: got=%+v generated=%+v", queried[0], generated[0])
	}

	approved := approveSupplyDecision(client, baseURL, adminToken, generated[0].Id)
	assertSupplyDecision(approved, profile, forecast, expectedTotalsFromLedgers(ledgers), modelName, model.SupplyDecisionStatusApproved)
	if approved.ReviewedBy != 1 ||
		approved.ReviewedAt <= 0 ||
		approved.ReviewNote != "accepted in process e2e" {
		log.Fatalf("unexpected approved supply decision review fields: got=%+v", approved)
	}
	if len(getSupplyDecisions(client, baseURL, adminToken, modelName, model.SupplyDecisionStatusApproved)) != 1 {
		log.Fatalf("expected one approved supply decision after review")
	}
	costProfile := recordSupplyCostProfile(client, baseURL, adminToken, 2, modelName, profile.PeriodStart, profile.PeriodEnd)
	queriedCostProfiles := getSupplyCostProfiles(client, baseURL, adminToken, 2, modelName)
	if len(queriedCostProfiles) != 1 ||
		queriedCostProfiles[0].Id != costProfile.Id ||
		queriedCostProfiles[0].SourceRef != "process-gb10-4t-self-hosted-cost" {
		log.Fatalf("unexpected queried supply cost profiles: got=%+v recorded=%+v", queriedCostProfiles, costProfile)
	}
	verifySupplyPrepaidLot(client, baseURL, adminToken, profile)
	opportunity := verifySupplyExpansionOpportunity(client, baseURL, adminToken, profile, forecast, approved, costProfile, modelName)
	verifyOperatingInsight(client, baseURL, adminToken, profile, approved, modelName)
	plans := generateSupplyActionPlans(client, baseURL, adminToken, approved.Id)
	if len(plans) != 1 {
		log.Fatalf("expected one generated supply action plan, got %d", len(plans))
	}
	assertSupplyActionPlan(plans[0], approved, opportunity)

	queriedPlans := getSupplyActionPlans(client, baseURL, adminToken, approved.Id)
	if len(queriedPlans) != 1 {
		log.Fatalf("expected one queried supply action plan, got %d", len(queriedPlans))
	}
	if queriedPlans[0].SupplyDecisionId != plans[0].SupplyDecisionId ||
		queriedPlans[0].ActionType != plans[0].ActionType ||
		queriedPlans[0].Status != plans[0].Status {
		log.Fatalf("queried supply action plan mismatch: got=%+v generated=%+v", queriedPlans[0], plans[0])
	}
	rejectSupplyActionExecution(client, baseURL, adminToken, plans[0].Id)
	inProgressPlan := updateSupplyActionPlanStatus(client, baseURL, adminToken, plans[0].Id, model.SupplyActionPlanStatusInProgress, "operator started process e2e work")
	if inProgressPlan.Status != model.SupplyActionPlanStatusInProgress ||
		inProgressPlan.OperatorNote != "operator started process e2e work" ||
		inProgressPlan.StatusUpdatedBy != 1 ||
		inProgressPlan.StatusUpdatedAt <= 0 ||
		inProgressPlan.StartedAt <= 0 ||
		inProgressPlan.CompletedAt != 0 {
		log.Fatalf("unexpected in-progress supply action plan: got=%+v", inProgressPlan)
	}
	completedPlan := updateSupplyActionPlanStatus(client, baseURL, adminToken, plans[0].Id, model.SupplyActionPlanStatusCompleted, "operator completed process e2e work")
	if completedPlan.Status != model.SupplyActionPlanStatusCompleted ||
		completedPlan.OperatorNote != "operator completed process e2e work" ||
		completedPlan.StatusUpdatedBy != 1 ||
		completedPlan.StartedAt != inProgressPlan.StartedAt ||
		completedPlan.CompletedAt <= 0 {
		log.Fatalf("unexpected completed supply action plan: got=%+v in_progress=%+v", completedPlan, inProgressPlan)
	}
	regeneratedPlans := generateSupplyActionPlans(client, baseURL, adminToken, approved.Id)
	if len(regeneratedPlans) != 1 ||
		regeneratedPlans[0].Status != model.SupplyActionPlanStatusCompleted ||
		regeneratedPlans[0].CompletedAt != completedPlan.CompletedAt {
		log.Fatalf("supply action plan regenerate should preserve lifecycle: got=%+v completed=%+v", regeneratedPlans, completedPlan)
	}
	rejectSupplyActionPlanStatus(client, baseURL, adminToken, plans[0].Id, model.SupplyActionPlanStatusInProgress)
	execution := recordSupplyActionExecution(client, baseURL, adminToken, completedPlan.Id, 1, 0, 1, 1000, 0.5, 0, 0, "process-e2e-self-hosted-evaluation", "process e2e execution recorded")
	assertSupplyActionExecution(execution, completedPlan, 1, 1, 1000, 0.5, "process-e2e-self-hosted-evaluation", "process e2e execution recorded")
	updatedExecution := recordSupplyActionExecution(client, baseURL, adminToken, completedPlan.Id, 1, 0, 1, 1200, 0.45, 0, 0, "process-e2e-self-hosted-evaluation-updated", "process e2e execution updated")
	if updatedExecution.Id != execution.Id ||
		updatedExecution.ActualCapacityTokens != 1200 ||
		updatedExecution.UnitCostQuota < 0.449999 ||
		updatedExecution.UnitCostQuota > 0.450001 ||
		updatedExecution.OperatorNote != "process e2e execution updated" {
		log.Fatalf("unexpected updated supply action execution: got=%+v original=%+v", updatedExecution, execution)
	}
	queriedExecutions := getSupplyActionExecutions(client, baseURL, adminToken, completedPlan.Id)
	if len(queriedExecutions) != 1 ||
		queriedExecutions[0].Id != updatedExecution.Id ||
		queriedExecutions[0].ExternalRef != "process-e2e-self-hosted-evaluation-updated" {
		log.Fatalf("unexpected queried supply action executions: got=%+v updated=%+v", queriedExecutions, updatedExecution)
	}
	rejectSupplyRoutingPolicy(client, baseURL, adminToken, updatedExecution.Id, "channel_id is required")
	now := time.Now().Unix()
	selfHostedCapacity := createSupplyCapacity(client, baseURL, adminToken, 2, "gb10-4t-self-hosted", modelName, now-3600, now+3600, 3000, 200, 99, 0.35)
	selfHostedExecution := recordSupplyActionExecution(client, baseURL, adminToken, completedPlan.Id, 2, 3, selfHostedCapacity.Id, selfHostedCapacity.CapacityTokens, selfHostedCapacity.UnitCostQuota, selfHostedCapacity.PeriodStart, selfHostedCapacity.PeriodEnd, "process-e2e-self-hosted-routing-ready", "process e2e self-hosted routing ready")
	if selfHostedExecution.Id != updatedExecution.Id ||
		selfHostedExecution.SupplierId != 2 ||
		selfHostedExecution.ChannelId != 3 ||
		selfHostedExecution.SupplyCapacityId != selfHostedCapacity.Id ||
		selfHostedExecution.Track != model.SupplyDecisionTrackSelfHosted {
		log.Fatalf("unexpected self-hosted execution: got=%+v capacity=%+v", selfHostedExecution, selfHostedCapacity)
	}
	rejectSupplyRoutingPolicy(client, baseURL, adminToken, selfHostedExecution.Id, "passed runtime SLA probe run is required")
	routingSlaRun := recordSelfHostedRoutingSlaEvidence(client, baseURL, adminToken, modelName, selfHostedExecution)
	policy := activateSupplyRoutingPolicyWithTrafficPercent(client, baseURL, adminToken, selfHostedExecution.Id, 50)
	assertSupplyRoutingPolicy(policy, selfHostedExecution, selfHostedCapacity.Id, routingSlaRun, 50)
	updatedPolicy := activateSupplyRoutingPolicyWithTrafficPercent(client, baseURL, adminToken, selfHostedExecution.Id, 50)
	if updatedPolicy.Id != policy.Id || updatedPolicy.Status != model.SupplyRoutingPolicyStatusActive {
		log.Fatalf("unexpected updated supply routing policy: got=%+v original=%+v", updatedPolicy, policy)
	}
	assertSupplyRoutingPolicy(updatedPolicy, selfHostedExecution, selfHostedCapacity.Id, routingSlaRun, 50)
	queriedPolicies := getSupplyRoutingPolicies(client, baseURL, adminToken, selfHostedExecution.Id)
	if len(queriedPolicies) != 1 || queriedPolicies[0].Id != policy.Id {
		log.Fatalf("unexpected queried routing policies: got=%+v policy=%+v", queriedPolicies, policy)
	}
	assertSupplyRoutingPolicy(queriedPolicies[0], selfHostedExecution, selfHostedCapacity.Id, routingSlaRun, 50)
	canaryInSession := findSupplyRoutingPolicyCanarySession(policy, true)
	canaryOutSession := findSupplyRoutingPolicyCanarySession(policy, false)
	mustChat(client, baseURL, modelName, canaryInSession, demandToken, "process-e2e-request-policy-canary-in")
	canaryInLedgers := waitLedgers(client, baseURL, canaryInSession, adminToken, 1)
	assertSupplyRoutingPolicyLedger(canaryInLedgers[0], canaryInSession, "process-e2e-request-policy-canary-in")
	mustChat(client, baseURL, modelName, canaryOutSession, demandToken, "process-e2e-request-policy-canary-out")
	canaryOutLedgers := waitLedgers(client, baseURL, canaryOutSession, adminToken, 1)
	assertSupplyRoutingPolicyFallbackLedger(canaryOutLedgers[0], canaryOutSession, "process-e2e-request-policy-canary-out")
	refreshedExecution := refreshSupplyActionExecutionUsage(client, baseURL, adminToken, selfHostedExecution.Id)
	assertSupplyActionExecutionDrawdown(refreshedExecution, selfHostedExecution, canaryInLedgers[0])
	disabledPolicy := disableSupplyRoutingPolicy(client, baseURL, adminToken, policy.Id)
	if disabledPolicy.Status != model.SupplyRoutingPolicyStatusDisabled ||
		disabledPolicy.DisabledBy != 1 ||
		disabledPolicy.DisabledAt <= 0 {
		log.Fatalf("unexpected disabled routing policy: got=%+v", disabledPolicy)
	}
	reactivatedPolicy := activateSupplyRoutingPolicyWithTrafficPercent(client, baseURL, adminToken, selfHostedExecution.Id, 50)
	if reactivatedPolicy.Id != policy.Id ||
		reactivatedPolicy.Status != model.SupplyRoutingPolicyStatusActive ||
		reactivatedPolicy.DisabledBy != 0 ||
		reactivatedPolicy.DisabledAt != 0 {
		log.Fatalf("unexpected reactivated routing policy: got=%+v original=%+v", reactivatedPolicy, policy)
	}
	assertSupplyRoutingPolicy(reactivatedPolicy, selfHostedExecution, selfHostedCapacity.Id, routingSlaRun, 50)
	routedLedgers := verifySupplyRoutingPolicyMissInsight(client, baseURL, adminToken, demandToken, sessionID, modelName, policy, routingSlaRun)
	disabledPolicy = disableSupplyRoutingPolicy(client, baseURL, adminToken, policy.Id)
	if disabledPolicy.Status != model.SupplyRoutingPolicyStatusDisabled ||
		disabledPolicy.DisabledBy != 1 ||
		disabledPolicy.DisabledAt <= 0 {
		log.Fatalf("unexpected final disabled routing policy: got=%+v", disabledPolicy)
	}
	return routedLedgers
}

func generateSupplyDecisions(client *http.Client, baseURL string, adminToken string, profile trafficProfileItem) []supplyDecisionItem {
	payload, err := json.Marshal(map[string]any{
		"period_start": profile.PeriodStart,
		"period_end":   profile.PeriodEnd,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/supply_decisions/generate", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply decision generate failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                 `json:"success"`
		Data    []supplyDecisionItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply decision generate returned success=false")
	}
	return envelope.Data
}

func getSupplyDecisions(client *http.Client, baseURL string, adminToken string, modelName string, status string) []supplyDecisionItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("model_name", modelName)
	values.Set("sla_tier", "default")
	values.Set("user_id", "2")
	values.Set("status", status)
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supply_decisions?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply decision query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplyDecisionItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply decision query returned success=false")
	}
	return envelope.Data.Items
}

func approveSupplyDecision(client *http.Client, baseURL string, adminToken string, decisionID int) supplyDecisionItem {
	payload, err := json.Marshal(map[string]any{
		"review_note": "accepted in process e2e",
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supply_decisions/%d/approve", strings.TrimRight(baseURL, "/"), decisionID), bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply decision approve failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool               `json:"success"`
		Data    supplyDecisionItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply decision approve returned success=false")
	}
	return envelope.Data
}

func recordSupplyCostProfile(client *http.Client, baseURL string, adminToken string, supplierID int, modelName string, periodStart int64, periodEnd int64) supplyCostProfileItem {
	payload, err := json.Marshal(map[string]any{
		"supplier_id":              supplierID,
		"supply_node":              "gb10-4t-self-hosted",
		"model_name":               modelName,
		"period_start":             periodStart,
		"period_end":               periodEnd,
		"capacity_tokens":          1000,
		"fixed_cost_quota":         100,
		"variable_unit_cost_quota": 0.02,
		"source_type":              model.SupplyCostProfileSourceAccounting,
		"source_ref":               "process-gb10-4t-self-hosted-cost",
		"observed_at":              time.Now().Unix(),
		"notes":                    "process e2e self-hosted amortized cost basis",
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/supply_cost_profiles/record", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply cost profile record failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                  `json:"success"`
		Data    supplyCostProfileItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply cost profile record returned success=false")
	}
	if envelope.Data.SupplierId != supplierID ||
		envelope.Data.SupplyNode != "gb10-4t-self-hosted" ||
		envelope.Data.ModelName != modelName ||
		envelope.Data.PeriodStart != periodStart ||
		envelope.Data.PeriodEnd != periodEnd ||
		envelope.Data.CapacityTokens != 1000 ||
		envelope.Data.SourceType != model.SupplyCostProfileSourceAccounting ||
		envelope.Data.SourceRef != "process-gb10-4t-self-hosted-cost" ||
		envelope.Data.RecordedBy != 1 ||
		envelope.Data.AmortizedUnitCostQuota < 0.119999 ||
		envelope.Data.AmortizedUnitCostQuota > 0.120001 {
		log.Fatalf("unexpected supply cost profile: got=%+v", envelope.Data)
	}
	return envelope.Data
}

func getSupplyCostProfiles(client *http.Client, baseURL string, adminToken string, supplierID int, modelName string) []supplyCostProfileItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("supplier_id", fmt.Sprintf("%d", supplierID))
	values.Set("supply_node", "gb10-4t-self-hosted")
	values.Set("model_name", modelName)
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supply_cost_profiles?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply cost profile query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplyCostProfileItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply cost profile query returned success=false")
	}
	return envelope.Data.Items
}

func verifySupplyPrepaidLot(client *http.Client, baseURL string, adminToken string, profile trafficProfileItem) {
	const prepaidModelName = "gpt-prepaid-process"
	rejectSupplyPrepaidLot(client, baseURL, adminToken, 1, "supplier must be self_operated")
	lot := recordSupplyPrepaidLot(client, baseURL, adminToken, 3, prepaidModelName, profile.PeriodStart, profile.PeriodEnd)
	assertSupplyPrepaidLotRecorded(lot, prepaidModelName, profile.PeriodStart, profile.PeriodEnd)
	queried := getSupplyPrepaidLots(client, baseURL, adminToken, 3, prepaidModelName)
	if len(queried) != 1 || queried[0].Id != lot.Id || queried[0].SourceRef != "process-gb10-4t-self-operated-prepaid" {
		log.Fatalf("unexpected queried supply prepaid lots: got=%+v recorded=%+v", queried, lot)
	}
	seedProcessPrepaidUsageLedgers(prepaidModelName, profile.PeriodStart)
	refreshed := refreshSupplyPrepaidLotUsage(client, baseURL, adminToken, lot.Id)
	assertSupplyPrepaidLotDrawdown(refreshed, lot)
}

func recordSupplyPrepaidLot(client *http.Client, baseURL string, adminToken string, supplierID int, prepaidModelName string, periodStart int64, periodEnd int64) supplyPrepaidLotItem {
	return postAdminJSON[supplyPrepaidLotItem](client, baseURL, adminToken, "/api/supply_prepaid_lots/record", model.SupplyPrepaidLotRecordInput{
		SupplierId:      supplierID,
		SupplyNode:      "gb10-4t-self-operated",
		ModelName:       prepaidModelName,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		PurchasedTokens: 1000,
		UnitCostQuota:   0.42,
		SourceType:      model.SupplyPrepaidLotSourceAccounting,
		SourceRef:       "process-gb10-4t-self-operated-prepaid",
		ObservedAt:      time.Now().Unix(),
		ExternalRef:     "po://process-gb10-4t-self-operated",
		Notes:           "process e2e self-operated prepaid lot",
	}, "supply prepaid lot record")
}

func rejectSupplyPrepaidLot(client *http.Client, baseURL string, adminToken string, supplierID int, expectedMessage string) {
	now := time.Now().Unix()
	payload, err := json.Marshal(model.SupplyPrepaidLotRecordInput{
		SupplierId:      supplierID,
		SupplyNode:      "gb10-4t",
		ModelName:       "gpt-prepaid-process",
		PeriodStart:     now - 3600,
		PeriodEnd:       now + 3600,
		PurchasedTokens: 1000,
		UnitCostQuota:   0.42,
		SourceType:      model.SupplyPrepaidLotSourceAccounting,
		SourceRef:       "process-prepaid-reject",
		ObservedAt:      now,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/supply_prepaid_lots/record", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply prepaid lot reject request failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if envelope.Success || !strings.Contains(envelope.Message, expectedMessage) {
		log.Fatalf("expected supply prepaid lot reject containing %q, got success=%t message=%q", expectedMessage, envelope.Success, envelope.Message)
	}
}

func getSupplyPrepaidLots(client *http.Client, baseURL string, adminToken string, supplierID int, prepaidModelName string) []supplyPrepaidLotItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("supplier_id", fmt.Sprintf("%d", supplierID))
	values.Set("supply_node", "gb10-4t-self-operated")
	values.Set("model_name", prepaidModelName)
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supply_prepaid_lots?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply prepaid lot query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplyPrepaidLotItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply prepaid lot query returned success=false")
	}
	return envelope.Data.Items
}

func refreshSupplyPrepaidLotUsage(client *http.Client, baseURL string, adminToken string, lotID int) supplyPrepaidLotItem {
	refreshed := postAdminJSON[[]supplyPrepaidLotItem](client, baseURL, adminToken, "/api/supply_prepaid_lots/refresh_usage", model.SupplyPrepaidLotUsageRefreshInput{
		PrepaidLotId: lotID,
	}, "supply prepaid lot usage refresh")
	if len(refreshed) != 1 {
		log.Fatalf("expected one refreshed supply prepaid lot, got %d", len(refreshed))
	}
	return refreshed[0]
}

func seedProcessPrepaidUsageLedgers(prepaidModelName string, periodStart int64) {
	if err := initDBForSeed(); err != nil {
		log.Fatalf("init db for prepaid lot seed: %v", err)
	}
	defer func() { _ = model.CloseDB() }()
	ledgers := []*model.UsageLedger{
		{
			RequestId:        "process-prepaid-drawdown-1",
			SessionId:        "process-prepaid-session",
			SupplierId:       3,
			ModelName:        prepaidModelName,
			PromptTokens:     100,
			CompletionTokens: 40,
			Status:           "success",
			SupplyNode:       "gb10-4t-self-operated",
			CreatedAt:        periodStart + 10,
		},
		{
			RequestId:        "process-prepaid-drawdown-2",
			SessionId:        "process-prepaid-session",
			SupplierId:       3,
			ModelName:        prepaidModelName,
			PromptTokens:     120,
			CompletionTokens: 60,
			Status:           "success",
			SupplyNode:       "gb10-4t-self-operated",
			CreatedAt:        periodStart + 20,
		},
		{
			RequestId:        "process-prepaid-drawdown-failed",
			SessionId:        "process-prepaid-session",
			SupplierId:       3,
			ModelName:        prepaidModelName,
			PromptTokens:     1000,
			CompletionTokens: 1000,
			Status:           "failed",
			SupplyNode:       "gb10-4t-self-operated",
			CreatedAt:        periodStart + 30,
		},
	}
	for _, ledger := range ledgers {
		if err := ledger.InsertIdempotent(); err != nil {
			log.Fatalf("insert prepaid usage ledger %s: %v", ledger.RequestId, err)
		}
	}
}

func verifySupplyExpansionOpportunity(client *http.Client, baseURL string, adminToken string, profile trafficProfileItem, forecast trafficForecastItem, decision supplyDecisionItem, costProfile supplyCostProfileItem, modelName string) supplyExpansionOpportunityItem {
	generated := generateSupplyExpansionOpportunities(client, baseURL, adminToken, profile)
	if len(generated) != 1 {
		log.Fatalf("expected one generated supply expansion opportunity, got %d", len(generated))
	}
	assertSupplyExpansionOpportunity(generated[0], profile, forecast, decision, costProfile, modelName)

	queried := getSupplyExpansionOpportunities(client, baseURL, adminToken, modelName, model.SupplyExpansionOpportunityTypeSelfHosted)
	if len(queried) != 1 {
		log.Fatalf("expected one queried supply expansion opportunity, got %d", len(queried))
	}
	if queried[0].OpportunityKey != generated[0].OpportunityKey ||
		queried[0].SupplyDecisionId != generated[0].SupplyDecisionId ||
		queried[0].SelfHostedCostProfileId != costProfile.Id ||
		queried[0].RankScore != generated[0].RankScore {
		log.Fatalf("queried supply expansion opportunity mismatch: got=%+v generated=%+v", queried[0], generated[0])
	}
	return generated[0]
}

func generateSupplyExpansionOpportunities(client *http.Client, baseURL string, adminToken string, profile trafficProfileItem) []supplyExpansionOpportunityItem {
	payload, err := json.Marshal(map[string]any{
		"period_start": profile.PeriodStart,
		"period_end":   profile.PeriodEnd,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/supply_expansion_opportunities/generate", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply expansion opportunity generate failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                             `json:"success"`
		Data    []supplyExpansionOpportunityItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply expansion opportunity generate returned success=false")
	}
	return envelope.Data
}

func getSupplyExpansionOpportunities(client *http.Client, baseURL string, adminToken string, modelName string, opportunityType string) []supplyExpansionOpportunityItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("model_name", modelName)
	values.Set("sla_tier", "default")
	values.Set("user_id", "2")
	if opportunityType != "" {
		values.Set("opportunity_type", opportunityType)
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supply_expansion_opportunities?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply expansion opportunity query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplyExpansionOpportunityItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply expansion opportunity query returned success=false")
	}
	return envelope.Data.Items
}

func verifyOperatingInsight(client *http.Client, baseURL string, adminToken string, profile trafficProfileItem, decision supplyDecisionItem, modelName string) {
	pricing := getPricingRecommendations(client, baseURL, adminToken, modelName, model.PricingRecommendationStatusApproved)
	if len(pricing) != 1 {
		log.Fatalf("expected one approved pricing recommendation before operating insight, got %d", len(pricing))
	}
	generated := generateOperatingInsights(client, baseURL, adminToken, profile)
	if len(generated) != 1 {
		log.Fatalf("expected one generated operating insight, got %d", len(generated))
	}
	assertOperatingInsight(generated[0], profile, decision, pricing[0], model.OperatingInsightStatusDraft)

	queried := getOperatingInsights(client, baseURL, adminToken, modelName, model.OperatingInsightStatusDraft, model.OperatingInsightCategoryCacheEfficiency)
	if len(queried) != 1 {
		log.Fatalf("expected one queried draft operating insight, got %d", len(queried))
	}
	if queried[0].InsightKey != generated[0].InsightKey ||
		queried[0].SupplyDecisionId != generated[0].SupplyDecisionId ||
		queried[0].PricingRecommendationId != generated[0].PricingRecommendationId {
		log.Fatalf("queried operating insight mismatch: got=%+v generated=%+v", queried[0], generated[0])
	}

	acknowledged := acknowledgeOperatingInsight(client, baseURL, adminToken, generated[0].Id)
	assertOperatingInsight(acknowledged, profile, decision, pricing[0], model.OperatingInsightStatusAcknowledged)
	if acknowledged.ReviewedBy != 1 ||
		acknowledged.ReviewedAt <= 0 ||
		acknowledged.ReviewNote != "acknowledged operating insight in process e2e" {
		log.Fatalf("unexpected acknowledged operating insight review fields: got=%+v", acknowledged)
	}
	if len(getOperatingInsights(client, baseURL, adminToken, modelName, model.OperatingInsightStatusAcknowledged, model.OperatingInsightCategoryCacheEfficiency)) != 1 {
		log.Fatalf("expected one acknowledged operating insight after review")
	}
	regenerated := generateOperatingInsights(client, baseURL, adminToken, profile)
	if len(regenerated) != 1 ||
		regenerated[0].Status != model.OperatingInsightStatusAcknowledged ||
		regenerated[0].ReviewNote != acknowledged.ReviewNote {
		log.Fatalf("operating insight regenerate should preserve review: got=%+v acknowledged=%+v", regenerated, acknowledged)
	}

	hotTelemetry := recordSupplyCapacityTelemetryInput(client, baseURL, adminToken, model.SupplyCapacityTelemetryRecordInput{
		SupplierId:         1,
		SupplyNode:         "gb10-hot",
		ModelName:          modelName,
		PeriodStart:        profile.PeriodStart,
		PeriodEnd:          profile.PeriodEnd,
		CapacityTokens:     1000,
		UsedTokens:         950,
		GpuUtilizationRate: 0.94,
		QualityScore:       97.5,
		UnitCostQuota:      0.52,
		SourceType:         model.SupplyCapacityTelemetrySourceNodeReport,
		SourceRef:          "process-gb10-hot-capacity-telemetry",
		ObservedAt:         time.Now().Unix(),
		Notes:              "process e2e hot node telemetry for operating insight",
	})
	if hotTelemetry.AppliedCapacityId <= 0 {
		log.Fatalf("expected hot telemetry to apply capacity, got=%+v", hotTelemetry)
	}
	generatedWithCapacityRisk := generateOperatingInsights(client, baseURL, adminToken, profile)
	if len(generatedWithCapacityRisk) < 2 {
		log.Fatalf("expected capacity telemetry risk to be generated alongside profile insight, got=%+v", generatedWithCapacityRisk)
	}
	capacityRiskInsights := getGlobalOperatingInsights(client, baseURL, adminToken, modelName, model.OperatingInsightStatusDraft, model.OperatingInsightCategoryCapacityRisk, profile)
	if len(capacityRiskInsights) != 1 {
		log.Fatalf("expected one draft capacity telemetry risk insight, got %d: %+v", len(capacityRiskInsights), capacityRiskInsights)
	}
	capacityRisk := capacityRiskInsights[0]
	if capacityRisk.Category != model.OperatingInsightCategoryCapacityRisk ||
		capacityRisk.Severity != model.OperatingInsightSeverityAction ||
		capacityRisk.UserId != 0 ||
		capacityRisk.SupplyHeadroomTokens != 50 ||
		!strings.Contains(capacityRisk.SliceKey, "gb10-hot") ||
		!strings.Contains(capacityRisk.InsightKey, "reason:high_gpu") ||
		!strings.Contains(capacityRisk.Summary, "GPU utilization") {
		log.Fatalf("unexpected capacity telemetry risk insight: got=%+v telemetry=%+v", capacityRisk, hotTelemetry)
	}
}

func generateOperatingInsights(client *http.Client, baseURL string, adminToken string, profile trafficProfileItem) []operatingInsightItem {
	payload, err := json.Marshal(map[string]any{
		"period_start": profile.PeriodStart,
		"period_end":   profile.PeriodEnd,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/operating_insights/generate", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("operating insight generate failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                   `json:"success"`
		Data    []operatingInsightItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("operating insight generate returned success=false")
	}
	return envelope.Data
}

func getOperatingInsights(client *http.Client, baseURL string, adminToken string, modelName string, status string, category string) []operatingInsightItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("model_name", modelName)
	values.Set("sla_tier", "default")
	values.Set("user_id", "2")
	values.Set("status", status)
	if category != "" {
		values.Set("category", category)
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/operating_insights?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("operating insight query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []operatingInsightItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("operating insight query returned success=false")
	}
	return envelope.Data.Items
}

func getGlobalOperatingInsights(client *http.Client, baseURL string, adminToken string, modelName string, status string, category string, profile trafficProfileItem) []operatingInsightItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("model_name", modelName)
	values.Set("sla_tier", "default")
	values.Set("status", status)
	if category != "" {
		values.Set("category", category)
	}
	values.Set("start_timestamp", fmt.Sprintf("%d", profile.PeriodStart))
	values.Set("end_timestamp", fmt.Sprintf("%d", profile.PeriodEnd))
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/operating_insights?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("global operating insight query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []operatingInsightItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("global operating insight query returned success=false")
	}
	return envelope.Data.Items
}

func acknowledgeOperatingInsight(client *http.Client, baseURL string, adminToken string, insightID int) operatingInsightItem {
	payload, err := json.Marshal(map[string]any{
		"review_note": "acknowledged operating insight in process e2e",
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/operating_insights/%d/acknowledge", strings.TrimRight(baseURL, "/"), insightID), bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("operating insight acknowledge failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                 `json:"success"`
		Data    operatingInsightItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("operating insight acknowledge returned success=false")
	}
	return envelope.Data
}

func generateSupplyActionPlans(client *http.Client, baseURL string, adminToken string, decisionID int) []supplyActionPlanItem {
	payload, err := json.Marshal(map[string]any{
		"decision_id": decisionID,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/supply_action_plans/generate", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply action plan generate failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                   `json:"success"`
		Data    []supplyActionPlanItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply action plan generate returned success=false")
	}
	return envelope.Data
}

func getSupplyActionPlans(client *http.Client, baseURL string, adminToken string, decisionID int) []supplyActionPlanItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("decision_id", fmt.Sprintf("%d", decisionID))
	values.Set("status", model.SupplyActionPlanStatusPlanned)
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supply_action_plans?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply action plan query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplyActionPlanItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply action plan query returned success=false")
	}
	return envelope.Data.Items
}

func updateSupplyActionPlanStatus(client *http.Client, baseURL string, adminToken string, planID int, status string, note string) supplyActionPlanItem {
	payload, err := json.Marshal(map[string]any{
		"status":        status,
		"operator_note": note,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supply_action_plans/%d/status", strings.TrimRight(baseURL, "/"), planID), bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply action plan status update failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                 `json:"success"`
		Data    supplyActionPlanItem `json:"data"`
		Message string               `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply action plan status update returned success=false message=%s", envelope.Message)
	}
	return envelope.Data
}

func rejectSupplyActionPlanStatus(client *http.Client, baseURL string, adminToken string, planID int, status string) {
	payload, err := json.Marshal(map[string]any{
		"status":        status,
		"operator_note": "should be rejected",
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supply_action_plans/%d/status", strings.TrimRight(baseURL, "/"), planID), bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply action plan rejected status request failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if envelope.Success || !strings.Contains(envelope.Message, "invalid supply action plan status transition") {
		log.Fatalf("expected rejected supply action plan status transition, got success=%v message=%q", envelope.Success, envelope.Message)
	}
}

func recordSupplyActionExecution(client *http.Client, baseURL string, adminToken string, planID int, supplierID int, channelID int, capacityID int, actualCapacity int64, unitCost float64, effectiveFrom int64, effectiveTo int64, externalRef string, note string) supplyActionExecutionItem {
	payload, err := json.Marshal(map[string]any{
		"supply_action_plan_id":  planID,
		"execution_status":       model.SupplyActionExecutionStatusRecorded,
		"supplier_id":            supplierID,
		"channel_id":             channelID,
		"supply_capacity_id":     capacityID,
		"actual_capacity_tokens": actualCapacity,
		"unit_cost_quota":        unitCost,
		"effective_from":         effectiveFrom,
		"effective_to":           effectiveTo,
		"external_ref":           externalRef,
		"operator_note":          note,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/supply_action_executions/record", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply action execution record failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                      `json:"success"`
		Data    supplyActionExecutionItem `json:"data"`
		Message string                    `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply action execution record returned success=false message=%s", envelope.Message)
	}
	return envelope.Data
}

func rejectSupplyActionExecution(client *http.Client, baseURL string, adminToken string, planID int) {
	payload, err := json.Marshal(map[string]any{
		"supply_action_plan_id":  planID,
		"execution_status":       model.SupplyActionExecutionStatusRecorded,
		"supplier_id":            1,
		"actual_capacity_tokens": 300,
		"unit_cost_quota":        0.5,
		"external_ref":           "process-e2e-before-complete",
		"operator_note":          "should be rejected before completion",
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/supply_action_executions/record", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply action execution rejected request failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if envelope.Success || !strings.Contains(envelope.Message, "must be completed") {
		log.Fatalf("expected rejected supply action execution, got success=%v message=%q", envelope.Success, envelope.Message)
	}
}

func getSupplyActionExecutions(client *http.Client, baseURL string, adminToken string, planID int) []supplyActionExecutionItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("supply_action_plan_id", fmt.Sprintf("%d", planID))
	values.Set("execution_status", model.SupplyActionExecutionStatusRecorded)
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supply_action_executions?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply action execution query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplyActionExecutionItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply action execution query returned success=false")
	}
	return envelope.Data.Items
}

func refreshSupplyActionExecutionUsage(client *http.Client, baseURL string, adminToken string, executionID int) supplyActionExecutionItem {
	refreshed := postAdminJSON[[]supplyActionExecutionItem](client, baseURL, adminToken, "/api/supply_action_executions/refresh_usage", model.SupplyActionExecutionUsageRefreshInput{
		ExecutionId: executionID,
	}, "supply action execution usage refresh")
	if len(refreshed) != 1 {
		log.Fatalf("expected one refreshed supply action execution, got %d", len(refreshed))
	}
	return refreshed[0]
}

func recordSelfHostedRoutingSlaEvidence(client *http.Client, baseURL string, adminToken string, modelName string, execution supplyActionExecutionItem) model.SlaProbeRun {
	now := time.Now().Unix()
	contract := postAdminJSON[model.SlaContract](client, baseURL, adminToken, "/api/sla_contracts/import", model.SlaContractImportInput{
		ContractKey:            selfHostedRoutingSlaContractKey,
		ModelName:              modelName,
		ModelAliases:           "gb10-4t-self-hosted",
		ProviderFamily:         "kimi",
		SourceName:             "token-router process self-hosted routing SLA",
		SourceRef:              "process://token-router-sim/self-hosted-routing",
		SourceSHA256:           "self-hosted-routing-process-sla",
		Version:                "2026-06-23",
		Status:                 model.SlaContractStatusActive,
		EffectiveFrom:          now - 3600,
		EffectiveTo:            now + 86400,
		MeasurementProfileJSON: selfHostedRoutingSlaMeasurementJSON,
		HardGateJSON:           selfHostedRoutingSlaHardGateJSON,
		SoftGateJSON:           selfHostedRoutingSlaSoftGateJSON,
	}, "SLA contract import")
	if contract.Id <= 0 ||
		contract.ContractKey != selfHostedRoutingSlaContractKey ||
		contract.Status != model.SlaContractStatusActive ||
		contract.ModelName != modelName {
		log.Fatalf("unexpected self-hosted routing SLA contract: got=%+v model=%s", contract, modelName)
	}
	plan := postAdminJSON[model.SlaProbePlan](client, baseURL, adminToken, "/api/sla_probe_plans/generate", model.SlaProbePlanGenerateInput{
		ContractId:     contract.Id,
		SupplierId:     execution.SupplierId,
		ChannelId:      execution.ChannelId,
		ModelName:      execution.ModelName,
		SlaTier:        execution.SlaTier,
		ProbeType:      model.SlaProbeTypeRuntimeLight,
		RouteMode:      model.SlaProbeRouteModeDirectUpstream,
		PromptSuiteKey: "process-self-hosted-routing",
		SampleSize:     1,
		RepeatCount:    1,
		MaxProbeQuota:  1000,
	}, "SLA probe plan generate")
	if plan.Id <= 0 ||
		plan.ContractId != contract.Id ||
		plan.SupplierId != execution.SupplierId ||
		plan.ChannelId != execution.ChannelId ||
		plan.ModelName != execution.ModelName ||
		plan.SlaTier != execution.SlaTier ||
		plan.ProbeType != model.SlaProbeTypeRuntimeLight ||
		plan.RouteMode != model.SlaProbeRouteModeDirectUpstream {
		log.Fatalf("unexpected self-hosted routing SLA plan: got=%+v execution=%+v contract=%+v", plan, execution, contract)
	}
	run := postAdminJSON[model.SlaProbeRun](client, baseURL, adminToken, "/api/sla_probe_runs/record", model.SlaProbeRunRecordInput{
		RunKey:         selfHostedRoutingSlaRunKey,
		PlanId:         plan.Id,
		Status:         model.SlaProbeRunStatusPassed,
		StartedAt:      now - 30,
		EndedAt:        now,
		RunnerVersion:  "token-router-sim/process",
		RuntimeRef:     processRuntimeRef(),
		Endpoint:       strings.TrimRight(baseURL, "/") + "/v1/chat/completions",
		SummaryJSON:    selfHostedRoutingSlaSummaryJSON,
		HardGatePassed: true,
		ArtifactURI:    "output/sla/" + selfHostedRoutingSlaRunKey,
		ArtifactSHA256: selfHostedRoutingSlaArtifactSHA256,
	}, "SLA probe run record")
	if run.Id <= 0 ||
		run.RunKey != selfHostedRoutingSlaRunKey ||
		run.PlanId != plan.Id ||
		run.ContractId != contract.Id ||
		run.SupplierId != execution.SupplierId ||
		run.ChannelId != execution.ChannelId ||
		run.ModelName != execution.ModelName ||
		run.SlaTier != execution.SlaTier ||
		run.Status != model.SlaProbeRunStatusPassed ||
		!run.HardGatePassed ||
		run.ArtifactSHA256 != selfHostedRoutingSlaArtifactSHA256 ||
		strings.TrimSpace(run.RuntimeRef) == "" {
		log.Fatalf("unexpected self-hosted routing SLA run: got=%+v plan=%+v execution=%+v", run, plan, execution)
	}
	return run
}

func postAdminJSON[T any](client *http.Client, baseURL string, adminToken string, path string, input any, operation string) T {
	data, message, ok := postAdminJSONAttempt[T](client, baseURL, adminToken, path, input, operation)
	if !ok {
		log.Fatalf("%s returned success=false message=%s", operation, message)
	}
	return data
}

func postAdminJSONAttempt[T any](client *http.Client, baseURL string, adminToken string, path string, input any, operation string) (T, string, bool) {
	payload, err := json.Marshal(input)
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+path, bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("%s failed with status %d", operation, resp.StatusCode)
	}
	var envelope struct {
		Success bool   `json:"success"`
		Data    T      `json:"data"`
		Message string `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	return envelope.Data, envelope.Message, envelope.Success
}

func isSQLiteBusyMessage(message string) bool {
	normalized := strings.ToLower(message)
	return strings.Contains(normalized, "database is locked") || strings.Contains(normalized, "sqlite_busy")
}

func processRuntimeRef() string {
	if value := strings.TrimSpace(os.Getenv("TOKEN_ROUTER_RUNTIME_REF")); value != "" {
		return value
	}
	hostname, err := os.Hostname()
	if err == nil && strings.TrimSpace(hostname) != "" {
		return strings.TrimSpace(hostname) + "/self-hosted-routing"
	}
	return "process/self-hosted-routing"
}

func activateSupplyRoutingPolicy(client *http.Client, baseURL string, adminToken string, executionID int) supplyRoutingPolicyItem {
	return activateSupplyRoutingPolicyWithTrafficPercent(client, baseURL, adminToken, executionID, 100)
}

func activateSupplyRoutingPolicyWithTrafficPercent(client *http.Client, baseURL string, adminToken string, executionID int, trafficPercent int) supplyRoutingPolicyItem {
	payload, err := json.Marshal(map[string]any{
		"supply_action_execution_id": executionID,
		"priority":                   100,
		"traffic_percent":            trafficPercent,
		"operator_note":              "process e2e self-hosted routing policy active",
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/supply_routing_policies/activate", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply routing policy activate failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                    `json:"success"`
		Data    supplyRoutingPolicyItem `json:"data"`
		Message string                  `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply routing policy activate returned success=false message=%s", envelope.Message)
	}
	return envelope.Data
}

func findSupplyRoutingPolicyCanarySession(policy supplyRoutingPolicyItem, included bool) string {
	for i := 0; i < 1000; i++ {
		sessionID := fmt.Sprintf("session-process-e2e-canary-%t-%03d", included, i)
		bucket := processSupplyRoutingPolicyTrafficBucket(policy.Id, "session:"+sessionID)
		if (bucket <= policy.TrafficPercent) == included {
			return sessionID
		}
	}
	log.Fatalf("could not find canary session for included=%t policy=%+v", included, policy)
	return ""
}

func processSupplyRoutingPolicyTrafficBucket(policyID int, routeKey string) int {
	hasher := fnv.New32a()
	_, _ = fmt.Fprintf(hasher, "%d|%s", policyID, strings.TrimSpace(routeKey))
	return int(hasher.Sum32()%100) + 1
}

func rejectSupplyRoutingPolicy(client *http.Client, baseURL string, adminToken string, executionID int, expectedMessage string) {
	payload, err := json.Marshal(map[string]any{
		"supply_action_execution_id": executionID,
		"priority":                   100,
		"operator_note":              "should be rejected",
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/supply_routing_policies/activate", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply routing policy rejected request failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if envelope.Success || !strings.Contains(envelope.Message, expectedMessage) {
		log.Fatalf("expected rejected supply routing policy, got success=%v message=%q", envelope.Success, envelope.Message)
	}
}

func disableSupplyRoutingPolicy(client *http.Client, baseURL string, adminToken string, policyID int) supplyRoutingPolicyItem {
	payload, err := json.Marshal(map[string]any{
		"operator_note": "process e2e self-hosted routing policy disabled",
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supply_routing_policies/%d/disable", strings.TrimRight(baseURL, "/"), policyID), bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply routing policy disable failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                    `json:"success"`
		Data    supplyRoutingPolicyItem `json:"data"`
		Message string                  `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply routing policy disable returned success=false message=%s", envelope.Message)
	}
	return envelope.Data
}

func getSupplyRoutingPolicies(client *http.Client, baseURL string, adminToken string, executionID int) []supplyRoutingPolicyItem {
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("supply_action_execution_id", fmt.Sprintf("%d", executionID))
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/supply_routing_policies?"+values.Encode(), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("supply routing policy query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []supplyRoutingPolicyItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("supply routing policy query returned success=false")
	}
	return envelope.Data.Items
}

func getChannel(client *http.Client, baseURL string, adminToken string, channelID int) model.Channel {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/channel/%d", strings.TrimRight(baseURL, "/"), channelID), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("channel query failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool          `json:"success"`
		Data    model.Channel `json:"data"`
		Message string        `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("channel query returned success=false message=%s", envelope.Message)
	}
	return envelope.Data
}

func updateChannelStatus(client *http.Client, baseURL string, adminToken string, channelID int, status int) model.Channel {
	channel := getChannel(client, baseURL, adminToken, channelID)
	channel.Status = status
	payload, err := json.Marshal(channel)
	must(err)
	req, err := http.NewRequest(http.MethodPut, strings.TrimRight(baseURL, "/")+"/api/channel", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("channel update failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool          `json:"success"`
		Data    model.Channel `json:"data"`
		Message string        `json:"message"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("channel update returned success=false message=%s", envelope.Message)
	}
	if envelope.Data.Status != status {
		log.Fatalf("channel update returned unexpected status: got=%d want=%d channel=%+v", envelope.Data.Status, status, envelope.Data)
	}
	return envelope.Data
}

func verifySupplyRoutingPolicyMissInsight(client *http.Client, baseURL string, adminToken string, demandToken string, sessionID string, modelName string, policy supplyRoutingPolicyItem, slaRun model.SlaProbeRun) []ledgerEnvelopeItem {
	updateChannelStatus(client, baseURL, adminToken, policy.ChannelId, common.ChannelStatusManuallyDisabled)
	existingLedgers := getLedgers(client, baseURL, sessionID, adminToken)
	mustChat(client, baseURL, modelName, sessionID, demandToken, "process-e2e-request-policy-miss")
	ledgers := waitLedgers(client, baseURL, sessionID, adminToken, len(existingLedgers)+1)
	if len(ledgers) != len(existingLedgers)+1 ||
		ledgers[0].RequestId != "process-e2e-request-policy-miss" ||
		ledgers[0].ChannelId == policy.ChannelId ||
		ledgers[0].SupplierId != 1 {
		log.Fatalf("unexpected policy miss fallback ledger: got=%+v policy=%+v", ledgers, policy)
	}
	insights := getOperatingInsights(client, baseURL, adminToken, modelName, model.OperatingInsightStatusDraft, model.OperatingInsightCategoryQualityWatch)
	for _, insight := range insights {
		if !strings.Contains(insight.InsightKey, "supply_routing_policy_miss") {
			continue
		}
		if !strings.Contains(insight.InsightKey, fmt.Sprintf("policy:%d", policy.Id)) ||
			insight.Severity != model.OperatingInsightSeverityWatch ||
			insight.SupplyDecisionId != policy.SupplyDecisionId ||
			insight.SlaProbeRunId != slaRun.Id ||
			!strings.Contains(insight.Summary, "policy channel is disabled") {
			log.Fatalf("unexpected policy miss insight: got=%+v policy=%+v sla_run=%+v", insight, policy, slaRun)
		}
		return ledgers
	}
	log.Fatalf("expected policy miss insight for policy=%+v got=%+v", policy, insights)
	return ledgers
}

func getMarginSummary(client *http.Client, baseURL string, adminToken string, supplierID int) []marginSummaryItem {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/reports/margin_summary?group_by=supplier&supplier_id=%d", strings.TrimRight(baseURL, "/"), supplierID), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("margin summary failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                `json:"success"`
		Data    []marginSummaryItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("margin summary returned success=false")
	}
	return envelope.Data
}

func generateSupplierStatement(client *http.Client, baseURL string, adminToken string, supplierID int) settlementStatementItem {
	now := time.Now().Unix()
	payload, err := json.Marshal(map[string]any{
		"subject_type": "supplier",
		"supplier_id":  supplierID,
		"period_start": now - 3600,
		"period_end":   now + 3600,
	})
	must(err)
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/settlement_statements/generate", bytes.NewReader(payload))
	must(err)
	req.Header.Set("Content-Type", "application/json")
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("settlement generate failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool                    `json:"success"`
		Data    settlementStatementItem `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("settlement generate returned success=false")
	}
	return envelope.Data
}

func getSettlementItems(client *http.Client, baseURL string, adminToken string, statementID int) []ledgerEnvelopeItem {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/settlement_statements/%d/items?page_size=10", strings.TrimRight(baseURL, "/"), statementID), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("settlement items failed with status %d", resp.StatusCode)
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []ledgerEnvelopeItem `json:"items"`
		} `json:"data"`
	}
	must(json.NewDecoder(resp.Body).Decode(&envelope))
	if !envelope.Success {
		log.Fatalf("settlement items returned success=false")
	}
	return envelope.Data.Items
}

func getSettlementCSV(client *http.Client, baseURL string, adminToken string, statementID int) string {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/settlement_statements/%d/items.csv", strings.TrimRight(baseURL, "/"), statementID), nil)
	must(err)
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("settlement csv failed with status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	must(err)
	return string(body)
}

func setAdminHeaders(req *http.Request, adminToken string) {
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("New-Api-User", "1")
}

func assertMarginSummary(row marginSummaryItem, expected expectedSettlementTotals) {
	if row.SupplierId != expected.SupplierId ||
		row.TotalRequests != expected.TotalRequests ||
		row.TotalSellQuota != expected.TotalSellQuota ||
		row.TotalCostQuota != expected.TotalCostQuota ||
		row.GrossProfitQuota != expected.GrossProfitQuota ||
		row.TotalPromptTokens != expected.TotalPromptTokens ||
		row.TotalCachedTokens != expected.TotalCachedTokens ||
		row.TotalCompletionTokens != expected.TotalCompletionTokens ||
		row.CacheHitCount != expected.CacheHitCount ||
		row.CacheHitRate != expected.CacheHitRate {
		log.Fatalf("unexpected margin summary: got=%+v expected=%+v", row, expected)
	}
}

func assertStatement(statement settlementStatementItem, expected expectedSettlementTotals) {
	if statement.Id <= 0 ||
		statement.SubjectType != "supplier" ||
		statement.SupplierId != expected.SupplierId ||
		statement.TotalRequests != expected.TotalRequests ||
		statement.TotalSellQuota != expected.TotalSellQuota ||
		statement.TotalCostQuota != expected.TotalCostQuota ||
		statement.GrossProfitQuota != expected.GrossProfitQuota ||
		statement.TotalPromptTokens != expected.TotalPromptTokens ||
		statement.TotalCachedTokens != expected.TotalCachedTokens ||
		statement.TotalCompletionTokens != expected.TotalCompletionTokens ||
		statement.CacheHitRate != expected.CacheHitRate {
		log.Fatalf("unexpected settlement statement: got=%+v expected=%+v", statement, expected)
	}
}

func assertTrafficProfile(profile trafficProfileItem, expected expectedSettlementTotals, modelName string) {
	if profile.ModelName != modelName ||
		profile.SlaTier != "default" ||
		profile.UserId != 2 ||
		profile.RequestCount != expected.TotalRequests ||
		profile.SuccessRequestCount != expected.TotalRequests ||
		profile.DemandTokens != expected.TotalPromptTokens+expected.TotalCompletionTokens ||
		profile.PeakTokens != expected.TotalPromptTokens+expected.TotalCompletionTokens ||
		profile.UniqueSessions != 1 ||
		profile.CacheHitCount != expected.CacheHitCount ||
		profile.TotalCachedTokens != expected.TotalCachedTokens ||
		profile.TotalSellQuota != expected.TotalSellQuota ||
		profile.TotalCostQuota != expected.TotalCostQuota ||
		profile.GrossProfitQuota != expected.GrossProfitQuota ||
		profile.SupplyCapacityTokens != 1000 ||
		profile.SupplyUsedTokens != 300 ||
		profile.SupplyHeadroomTokens != 700 ||
		profile.PeakRatio < 0.999999 ||
		profile.PeakRatio > 1.000001 ||
		profile.CacheHitRate != expected.CacheHitRate ||
		profile.SlaMetRate < 0.999999 ||
		profile.SlaMetRate > 1.000001 ||
		profile.AvgSupplyQualityScore < 98.499999 ||
		profile.AvgSupplyQualityScore > 98.500001 ||
		profile.AvgUnitCostQuota < 0.499999 ||
		profile.AvgUnitCostQuota > 0.500001 {
		log.Fatalf("unexpected traffic profile: got=%+v expected=%+v", profile, expected)
	}
}

func assertTrafficForecast(forecast trafficForecastItem, profile trafficProfileItem, modelName string) {
	expectedTargetEnd := profile.PeriodEnd + (profile.PeriodEnd - profile.PeriodStart)
	if forecast.Id <= 0 ||
		forecast.SliceKey != profile.SliceKey ||
		forecast.ModelName != modelName ||
		forecast.SlaTier != "default" ||
		forecast.UserId != 2 ||
		forecast.SourcePeriodStart != profile.PeriodStart ||
		forecast.SourcePeriodEnd != profile.PeriodEnd ||
		forecast.TargetPeriodStart != profile.PeriodEnd ||
		forecast.TargetPeriodEnd != expectedTargetEnd ||
		forecast.SourceProfileCount != 1 ||
		forecast.ObservedRequestCount != profile.RequestCount ||
		forecast.ObservedDemandTokens != profile.DemandTokens ||
		forecast.ObservedPeakTokens != profile.PeakTokens ||
		forecast.ForecastDemandTokens != profile.DemandTokens ||
		forecast.ForecastPeakTokens != profile.PeakTokens ||
		forecast.ForecastHeadroomTokens != profile.SupplyHeadroomTokens ||
		forecast.ForecastGapTokens != 0 ||
		forecast.CacheHitRate != profile.CacheHitRate ||
		forecast.SlaMetRate < 0.999999 ||
		forecast.SlaMetRate > 1.000001 ||
		forecast.GrossProfitQuota != profile.GrossProfitQuota ||
		forecast.AvgUnitCostQuota < 0.499999 ||
		forecast.AvgUnitCostQuota > 0.500001 ||
		forecast.Confidence < 0.333332 ||
		forecast.Confidence > 0.333334 ||
		forecast.Method != model.TrafficForecastMethodWeightedMovingAverage {
		log.Fatalf("unexpected traffic forecast: got=%+v profile=%+v", forecast, profile)
	}
}

func assertSeasonalAnomalyTrafficForecast(forecast trafficForecastItem, modelName string, sourceStart int64, sourceEnd int64, targetStart int64, targetEnd int64) {
	if forecast.Id <= 0 ||
		forecast.ModelName != modelName ||
		forecast.SlaTier != "seasonal" ||
		forecast.UserId != 99 ||
		forecast.SourcePeriodStart != sourceStart ||
		forecast.SourcePeriodEnd != sourceEnd ||
		forecast.TargetPeriodStart != targetStart ||
		forecast.TargetPeriodEnd != targetEnd ||
		forecast.SourceProfileCount != 4 ||
		forecast.ObservedDemandTokens != 880 ||
		forecast.ObservedPeakTokens != 390 ||
		forecast.BaselineDemandTokens != 250 ||
		forecast.ForecastDemandTokens != 150 ||
		forecast.ForecastPeakTokens != 390 ||
		forecast.ForecastHeadroomTokens != 250 ||
		forecast.ForecastGapTokens != 140 ||
		forecast.TrendDemandDeltaTokens != 260 ||
		forecast.TrendDemandDeltaRate < 2.599999 ||
		forecast.TrendDemandDeltaRate > 2.600001 ||
		forecast.SeasonalPeriodCount != 2 ||
		forecast.SeasonalIndex < 0.499999 ||
		forecast.SeasonalIndex > 0.500001 ||
		forecast.SeasonalDemandTokens != 125 ||
		forecast.AnomalyStatus != model.TrafficForecastAnomalySpike ||
		forecast.AnomalyProfileId <= 0 ||
		forecast.AnomalyDemandRatio < 2.076922 ||
		forecast.AnomalyDemandRatio > 2.076924 ||
		forecast.Method != model.TrafficForecastMethodSeasonalAnomaly {
		log.Fatalf("unexpected seasonal anomaly traffic forecast: got=%+v", forecast)
	}
}

func assertSupplierScorecard(scorecard supplierScorecardItem, expected expectedSettlementTotals) {
	if scorecard.Id <= 0 ||
		scorecard.SupplierId != expected.SupplierId ||
		scorecard.TotalRequests != expected.TotalRequests ||
		scorecard.SuccessRequests != expected.TotalRequests ||
		scorecard.ErrorRequests != 0 ||
		scorecard.CacheHitCount != expected.CacheHitCount ||
		scorecard.TotalSellQuota != expected.TotalSellQuota ||
		scorecard.TotalCostQuota != expected.TotalCostQuota ||
		scorecard.GrossProfitQuota != expected.GrossProfitQuota ||
		scorecard.SupplyCapacityTokens != 1000 ||
		scorecard.SupplyUsedTokens != 300 ||
		scorecard.SupplyHeadroomTokens != 700 ||
		scorecard.SuccessRate < 0.999999 ||
		scorecard.SuccessRate > 1.000001 ||
		scorecard.CacheHitRate != expected.CacheHitRate ||
		scorecard.AvgSupplyQualityScore < 98.499999 ||
		scorecard.AvgSupplyQualityScore > 98.500001 ||
		scorecard.AvgUnitCostQuota < 0.499999 ||
		scorecard.AvgUnitCostQuota > 0.500001 ||
		scorecard.Score < 85 ||
		scorecard.Score > 100 ||
		scorecard.Grade != model.SupplierScorecardGradeA {
		log.Fatalf("unexpected supplier scorecard: got=%+v expected=%+v", scorecard, expected)
	}
}

func assertSupplierEvaluation(evaluation supplierEvaluationItem, scorecard supplierScorecardItem, status string, expectSlaEvidence bool, expectedSlaRunKey string) {
	if evaluation.Id <= 0 ||
		evaluation.EvaluationType != model.SupplierEvaluationTypeAdmission ||
		evaluation.SupplierId != scorecard.SupplierId ||
		evaluation.SupplierScorecardId != scorecard.Id ||
		evaluation.PeriodStart != scorecard.PeriodStart ||
		evaluation.PeriodEnd != scorecard.PeriodEnd ||
		evaluation.Status != status ||
		evaluation.Recommendation != model.SupplierEvaluationRecommendationAdmit ||
		evaluation.Grade != model.SupplierScorecardGradeA ||
		evaluation.TotalRequests != scorecard.TotalRequests ||
		evaluation.GrossProfitQuota != scorecard.GrossProfitQuota ||
		evaluation.SupplyHeadroomTokens != scorecard.SupplyHeadroomTokens ||
		evaluation.SuccessRate < 0.999999 ||
		evaluation.SuccessRate > 1.000001 ||
		evaluation.CacheHitRate != scorecard.CacheHitRate ||
		evaluation.AvgSupplyQualityScore < 98.499999 ||
		evaluation.AvgSupplyQualityScore > 98.500001 ||
		evaluation.AvgUnitCostQuota < 0.499999 ||
		evaluation.AvgUnitCostQuota > 0.500001 ||
		evaluation.Score < 85 ||
		evaluation.Score > 100 ||
		!strings.Contains(evaluation.Reason, "admission threshold") {
		log.Fatalf("unexpected supplier evaluation: got=%+v scorecard=%+v status=%s", evaluation, scorecard, status)
	}
	if !expectSlaEvidence {
		return
	}
	if evaluation.SlaContractId <= 0 ||
		evaluation.SlaProbeRunId <= 0 ||
		strings.TrimSpace(evaluation.SlaGateSummaryJSON) == "" ||
		!strings.Contains(evaluation.Reason, "SLA admission evidence") {
		log.Fatalf("expected supplier evaluation to reference SLA evidence: got=%+v scorecard=%+v status=%s", evaluation, scorecard, status)
	}
	if strings.TrimSpace(expectedSlaRunKey) != "" && !strings.Contains(evaluation.SlaGateSummaryJSON, expectedSlaRunKey) {
		log.Fatalf("expected supplier evaluation SLA gate summary to contain run key %q, got %s", expectedSlaRunKey, evaluation.SlaGateSummaryJSON)
	}
}

func assertSupplierPostureRecommendation(recommendation supplierPostureRecommendationItem, scorecard supplierScorecardItem, status string) {
	if recommendation.Id <= 0 ||
		recommendation.SupplierId != scorecard.SupplierId ||
		recommendation.SupplierScorecardId != scorecard.Id ||
		recommendation.PeriodStart != scorecard.PeriodStart ||
		recommendation.PeriodEnd != scorecard.PeriodEnd ||
		recommendation.Status != status ||
		recommendation.RecommendedAction != model.SupplierPostureRecommendationActionBoost ||
		recommendation.Grade != model.SupplierScorecardGradeA ||
		recommendation.TotalRequests != scorecard.TotalRequests ||
		recommendation.QualityInsightCount != 0 ||
		recommendation.CapacityInsightCount != 0 ||
		recommendation.ActionInsightCount != 0 ||
		recommendation.SupplierStatusCurrent != common.ChannelStatusEnabled ||
		recommendation.Score < model.SupplierPostureRecommendationBoostMinScore ||
		!strings.Contains(recommendation.Reason, "boost review threshold") {
		log.Fatalf("unexpected supplier posture recommendation: got=%+v scorecard=%+v status=%s", recommendation, scorecard, status)
	}
}

func assertPricingRecommendation(recommendation pricingRecommendationItem, profile trafficProfileItem, expected expectedSettlementTotals, modelName string, status string) {
	expectedDemandTokens := expected.TotalPromptTokens + expected.TotalCompletionTokens
	if expectedDemandTokens <= 0 || expected.TotalSellQuota <= 0 {
		log.Fatalf("invalid expected pricing totals: expected=%+v", expected)
	}
	expectedCurrentUnitPrice := float64(expected.TotalSellQuota) / float64(expectedDemandTokens)
	expectedCurrentUnitCost := float64(expected.TotalCostQuota) / float64(expectedDemandTokens)
	expectedCurrentMarginRate := float64(expected.GrossProfitQuota) / float64(expected.TotalSellQuota)
	expectedRecommendedUnitPrice := expectedCurrentUnitPrice * 0.9
	expectedRecommendedMarginRate := (expectedRecommendedUnitPrice - expectedCurrentUnitCost) / expectedRecommendedUnitPrice
	if recommendation.Id <= 0 ||
		recommendation.TrafficProfileId != profile.Id ||
		recommendation.SliceKey != profile.SliceKey ||
		recommendation.ModelName != modelName ||
		recommendation.SlaTier != "default" ||
		recommendation.UserId != 2 ||
		recommendation.Action != model.PricingRecommendationActionShareSavings ||
		recommendation.Status != status ||
		recommendation.RequestCount != expected.TotalRequests ||
		recommendation.DemandTokens != expectedDemandTokens ||
		recommendation.PeakTokens != expectedDemandTokens ||
		recommendation.SupplyHeadroomTokens != 700 ||
		recommendation.TotalSellQuota != expected.TotalSellQuota ||
		recommendation.TotalCostQuota != expected.TotalCostQuota ||
		recommendation.GrossProfitQuota != expected.GrossProfitQuota ||
		!floatWithin(recommendation.CacheHitRate, expected.CacheHitRate, 0.000001) ||
		recommendation.SlaMetRate < 0.999999 ||
		recommendation.SlaMetRate > 1.000001 ||
		!floatWithin(recommendation.CurrentUnitPriceQuota, expectedCurrentUnitPrice, 0.000001) ||
		!floatWithin(recommendation.CurrentUnitCostQuota, expectedCurrentUnitCost, 0.000001) ||
		!floatWithin(recommendation.CurrentMarginRate, expectedCurrentMarginRate, 0.000001) ||
		!floatWithin(recommendation.RecommendedUnitPriceQuota, expectedRecommendedUnitPrice, 0.000001) ||
		!floatWithin(recommendation.RecommendedMarginRate, expectedRecommendedMarginRate, 0.000001) ||
		recommendation.AvgSupplyQualityScore < 98.499999 ||
		recommendation.AvgSupplyQualityScore > 98.500001 ||
		recommendation.AvgUnitCostQuota < 0.499999 ||
		recommendation.AvgUnitCostQuota > 0.500001 ||
		!strings.Contains(recommendation.Reason, "share efficiency savings") {
		log.Fatalf("unexpected pricing recommendation: got=%+v expected=%+v profile=%+v status=%s", recommendation, expected, profile, status)
	}
}

func floatWithin(got float64, want float64, tolerance float64) bool {
	return got >= want-tolerance && got <= want+tolerance
}

func expectedSupplyDecisionRoi(demandTokens int64, gapTokens int64, grossProfitQuota int64, cacheHitRate float64, avgUnitCostQuota float64) float64 {
	return float64(grossProfitQuota) +
		cacheHitRate*float64(demandTokens)*avgUnitCostQuota -
		float64(gapTokens)*avgUnitCostQuota
}

func expectedSupplyExpansionStabilityScore(peakRatio float64) float64 {
	if peakRatio <= 0 {
		return 0
	}
	if peakRatio <= 1 {
		return 1
	}
	score := 1 / peakRatio
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func expectedSupplyExpansionHeadroomRiskScore(gapTokens int64, peakTokens int64) float64 {
	if gapTokens <= 0 || peakTokens <= 0 {
		return 0
	}
	score := float64(gapTokens) / float64(peakTokens)
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

func assertSupplyDecision(decision supplyDecisionItem, profile trafficProfileItem, forecast trafficForecastItem, expected expectedSettlementTotals, modelName string, status string) {
	expectedRoiScore := expectedSupplyDecisionRoi(
		forecast.ForecastDemandTokens,
		forecast.ForecastGapTokens,
		expected.GrossProfitQuota,
		expected.CacheHitRate,
		forecast.AvgUnitCostQuota,
	)
	if decision.Id <= 0 ||
		decision.TrafficProfileId != profile.Id ||
		decision.TrafficForecastId != forecast.Id ||
		decision.DecisionSource != model.SupplyDecisionSourceForecast ||
		decision.SliceKey != profile.SliceKey ||
		decision.ModelName != modelName ||
		decision.SlaTier != "default" ||
		decision.UserId != 2 ||
		decision.ForecastTargetStart != forecast.TargetPeriodStart ||
		decision.ForecastTargetEnd != forecast.TargetPeriodEnd ||
		decision.ForecastConfidence < 0.333332 ||
		decision.ForecastConfidence > 0.333334 ||
		decision.ForecastMethod != model.TrafficForecastMethodWeightedMovingAverage ||
		decision.DecisionType != model.SupplyDecisionTypeSelfHostedEvaluate ||
		decision.Track != model.SupplyDecisionTrackSelfHosted ||
		decision.Status != status ||
		decision.DemandTokens != forecast.ForecastDemandTokens ||
		decision.PeakTokens != forecast.ForecastPeakTokens ||
		decision.SupplyHeadroomTokens != forecast.ForecastHeadroomTokens ||
		decision.GapTokens != forecast.ForecastGapTokens ||
		decision.RecommendedCapacity != forecast.ForecastDemandTokens ||
		decision.GrossProfitQuota != expected.GrossProfitQuota ||
		!floatWithin(decision.CacheHitRate, expected.CacheHitRate, 0.000001) ||
		decision.SlaMetRate < 0.999999 ||
		decision.SlaMetRate > 1.000001 ||
		decision.AvgSupplyQualityScore < 98.499999 ||
		decision.AvgSupplyQualityScore > 98.500001 ||
		decision.AvgUnitCostQuota < 0.499999 ||
		decision.AvgUnitCostQuota > 0.500001 ||
		!floatWithin(decision.RoiScore, expectedRoiScore, 0.000001) {
		log.Fatalf("unexpected supply decision: got=%+v expected=%+v profile=%+v status=%s", decision, expected, profile, status)
	}
}

func assertSupplyExpansionOpportunity(opportunity supplyExpansionOpportunityItem, profile trafficProfileItem, forecast trafficForecastItem, decision supplyDecisionItem, costProfile supplyCostProfileItem, modelName string) {
	expectedSelfHostedSavingsUnitQuota := decision.AvgUnitCostQuota - costProfile.AmortizedUnitCostQuota
	expectedSelfHostedSavingsQuota := expectedSelfHostedSavingsUnitQuota * float64(decision.DemandTokens)
	expectedLocalityScore := decision.CacheHitRate
	expectedStabilityScore := expectedSupplyExpansionStabilityScore(profile.PeakRatio)
	expectedHeadroomRiskScore := expectedSupplyExpansionHeadroomRiskScore(decision.GapTokens, decision.PeakTokens)
	expectedRankScore := decision.RoiScore +
		expectedLocalityScore*100 +
		expectedStabilityScore*50 +
		expectedHeadroomRiskScore*150 +
		expectedSelfHostedSavingsQuota
	if opportunity.Id <= 0 ||
		opportunity.SupplyDecisionId != decision.Id ||
		opportunity.TrafficProfileId != profile.Id ||
		opportunity.TrafficForecastId != forecast.Id ||
		opportunity.DecisionSource != model.SupplyDecisionSourceForecast ||
		opportunity.DecisionStatus != model.SupplyDecisionStatusApproved ||
		opportunity.SliceKey != profile.SliceKey ||
		opportunity.ModelName != modelName ||
		opportunity.SlaTier != "default" ||
		opportunity.UserId != 2 ||
		opportunity.ForecastTargetStart != forecast.TargetPeriodStart ||
		opportunity.ForecastTargetEnd != forecast.TargetPeriodEnd ||
		opportunity.ForecastConfidence < 0.333332 ||
		opportunity.ForecastConfidence > 0.333334 ||
		opportunity.ForecastMethod != model.TrafficForecastMethodWeightedMovingAverage ||
		opportunity.OpportunityType != model.SupplyExpansionOpportunityTypeSelfHosted ||
		opportunity.Priority != model.SupplyExpansionOpportunityPriorityAction ||
		opportunity.ClusterKey != model.SupplyExpansionOpportunityClusterHighCacheStable ||
		opportunity.Track != model.SupplyDecisionTrackSelfHosted ||
		opportunity.DecisionType != model.SupplyDecisionTypeSelfHostedEvaluate ||
		opportunity.DemandTokens != forecast.ForecastDemandTokens ||
		opportunity.PeakTokens != forecast.ForecastPeakTokens ||
		opportunity.SupplyHeadroomTokens != forecast.ForecastHeadroomTokens ||
		opportunity.GapTokens != forecast.ForecastGapTokens ||
		opportunity.RecommendedCapacity != forecast.ForecastDemandTokens ||
		opportunity.GrossProfitQuota != decision.GrossProfitQuota ||
		!floatWithin(opportunity.CacheHitRate, decision.CacheHitRate, 0.000001) ||
		opportunity.SlaMetRate < 0.999999 ||
		opportunity.SlaMetRate > 1.000001 ||
		opportunity.AvgSupplyQualityScore < 98.499999 ||
		opportunity.AvgSupplyQualityScore > 98.500001 ||
		opportunity.AvgUnitCostQuota < 0.499999 ||
		opportunity.AvgUnitCostQuota > 0.500001 ||
		!floatWithin(opportunity.RoiScore, decision.RoiScore, 0.000001) ||
		opportunity.SelfHostedCostProfileId != costProfile.Id ||
		!floatWithin(opportunity.SelfHostedUnitCostQuota, costProfile.AmortizedUnitCostQuota, 0.000001) ||
		!floatWithin(opportunity.SelfHostedSavingsUnitQuota, expectedSelfHostedSavingsUnitQuota, 0.000001) ||
		!floatWithin(opportunity.SelfHostedSavingsQuota, expectedSelfHostedSavingsQuota, 0.000001) ||
		!floatWithin(opportunity.PeakRatio, profile.PeakRatio, 0.000001) ||
		opportunity.UniqueSessions != profile.UniqueSessions ||
		!floatWithin(opportunity.LocalityScore, expectedLocalityScore, 0.000001) ||
		!floatWithin(opportunity.StabilityScore, expectedStabilityScore, 0.000001) ||
		!floatWithin(opportunity.HeadroomRiskScore, expectedHeadroomRiskScore, 0.000001) ||
		!floatWithin(opportunity.RankScore, expectedRankScore, 0.000001) ||
		!strings.Contains(opportunity.Reason, "self-hosted expansion candidate") ||
		!strings.Contains(opportunity.Reason, "process-gb10-4t-self-hosted-cost") {
		log.Fatalf("unexpected supply expansion opportunity: got=%+v profile=%+v forecast=%+v decision=%+v", opportunity, profile, forecast, decision)
	}
}

func assertOperatingInsight(insight operatingInsightItem, profile trafficProfileItem, decision supplyDecisionItem, recommendation pricingRecommendationItem, status string) {
	if insight.Id <= 0 ||
		insight.TrafficProfileId != profile.Id ||
		insight.SupplyDecisionId != decision.Id ||
		insight.PricingRecommendationId != recommendation.Id ||
		insight.SliceKey != profile.SliceKey ||
		insight.ModelName != profile.ModelName ||
		insight.SlaTier != "default" ||
		insight.UserId != profile.UserId ||
		insight.Status != status ||
		insight.Category != model.OperatingInsightCategoryCacheEfficiency ||
		insight.Severity != model.OperatingInsightSeverityAction ||
		insight.DemandTokens != profile.DemandTokens ||
		insight.PeakTokens != profile.PeakTokens ||
		insight.SupplyHeadroomTokens != profile.SupplyHeadroomTokens ||
		insight.GrossProfitQuota != profile.GrossProfitQuota ||
		insight.SupplyDecisionTrack != decision.Track ||
		insight.SupplyDecisionType != decision.DecisionType ||
		insight.SupplyDecisionStatus != decision.Status ||
		insight.PricingRecommendationAction != recommendation.Action ||
		insight.PricingRecommendationStatus != recommendation.Status ||
		insight.CacheHitRate != profile.CacheHitRate ||
		insight.SlaMetRate < 0.999999 ||
		insight.SlaMetRate > 1.000001 ||
		insight.AvgUnitCostQuota < 0.499999 ||
		insight.AvgUnitCostQuota > 0.500001 ||
		!floatWithin(insight.SupplyDecisionRoiScore, decision.RoiScore, 0.000001) ||
		!floatWithin(insight.RecommendedUnitPriceQuota, recommendation.RecommendedUnitPriceQuota, 0.000001) ||
		!strings.Contains(insight.RecommendedAction, "self-hosted") {
		log.Fatalf("unexpected operating insight: got=%+v profile=%+v decision=%+v recommendation=%+v status=%s", insight, profile, decision, recommendation, status)
	}
}

func assertSupplyActionPlan(plan supplyActionPlanItem, decision supplyDecisionItem, opportunity supplyExpansionOpportunityItem) {
	if plan.Id <= 0 ||
		plan.SupplyDecisionId != decision.Id ||
		plan.DecisionKey != decision.DecisionKey ||
		plan.SupplyExpansionOpportunityId != opportunity.Id ||
		plan.OpportunityKey != opportunity.OpportunityKey ||
		plan.OpportunityType != opportunity.OpportunityType ||
		plan.OpportunityPriority != opportunity.Priority ||
		plan.OpportunityClusterKey != opportunity.ClusterKey ||
		plan.OpportunityRankScore < opportunity.RankScore-0.000001 ||
		plan.OpportunityRankScore > opportunity.RankScore+0.000001 ||
		plan.TrafficProfileId != decision.TrafficProfileId ||
		plan.SliceKey != decision.SliceKey ||
		plan.ModelName != decision.ModelName ||
		plan.SlaTier != decision.SlaTier ||
		plan.UserId != decision.UserId ||
		plan.DecisionType != decision.DecisionType ||
		plan.Track != decision.Track ||
		plan.ActionType != model.SupplyActionTypeEvaluateSelfHostedCapacity ||
		plan.Status != model.SupplyActionPlanStatusPlanned ||
		plan.RecommendedCapacity != decision.RecommendedCapacity ||
		plan.GapTokens != decision.GapTokens ||
		plan.RoiScore != decision.RoiScore ||
		plan.SourceReviewedAt != decision.ReviewedAt ||
		plan.SourceReviewedBy != decision.ReviewedBy {
		log.Fatalf("unexpected supply action plan: got=%+v decision=%+v", plan, decision)
	}
}

func assertSupplyActionExecution(execution supplyActionExecutionItem, plan supplyActionPlanItem, supplierID int, capacityID int, actualCapacity int64, unitCost float64, externalRef string, note string) {
	if execution.Id <= 0 ||
		execution.SupplyActionPlanId != plan.Id ||
		execution.SupplyDecisionId != plan.SupplyDecisionId ||
		execution.DecisionKey != plan.DecisionKey ||
		execution.TrafficProfileId != plan.TrafficProfileId ||
		execution.SliceKey != plan.SliceKey ||
		execution.ModelName != plan.ModelName ||
		execution.SlaTier != plan.SlaTier ||
		execution.UserId != plan.UserId ||
		execution.DecisionType != plan.DecisionType ||
		execution.Track != plan.Track ||
		execution.ActionType != plan.ActionType ||
		execution.ExecutionStatus != model.SupplyActionExecutionStatusRecorded ||
		execution.SupplierId != supplierID ||
		execution.SupplyCapacityId != capacityID ||
		execution.RecommendedCapacity != plan.RecommendedCapacity ||
		execution.ActualCapacityTokens != actualCapacity ||
		execution.GapTokens != plan.GapTokens ||
		execution.RoiScore != plan.RoiScore ||
		execution.UnitCostQuota < unitCost-0.000001 ||
		execution.UnitCostQuota > unitCost+0.000001 ||
		execution.ExternalRef != externalRef ||
		execution.OperatorNote != note ||
		execution.ActionPlanCompletedAt != plan.CompletedAt ||
		execution.ActionPlanCompletedBy != plan.StatusUpdatedBy ||
		execution.RecordedBy != 1 ||
		execution.RecordedAt <= 0 {
		log.Fatalf("unexpected supply action execution: got=%+v plan=%+v", execution, plan)
	}
}

func assertSupplyActionExecutionDrawdown(execution supplyActionExecutionItem, original supplyActionExecutionItem, ledger ledgerEnvelopeItem) {
	expectedUsed := int64(ledger.PromptTokens + ledger.CompletionTokens)
	expectedRate := 0.0
	if original.ActualCapacityTokens > 0 {
		expectedRate = float64(expectedUsed) / float64(original.ActualCapacityTokens)
	}
	if execution.Id != original.Id ||
		execution.DrawdownTokens != expectedUsed ||
		execution.DrawdownRequestCount != 1 ||
		execution.RemainingTokens != original.ActualCapacityTokens-expectedUsed ||
		execution.DrawdownRate < expectedRate-0.000001 ||
		execution.DrawdownRate > expectedRate+0.000001 ||
		execution.DrawdownSourceType != model.SupplyActionExecutionDrawdownSourceUsageLedger ||
		!strings.Contains(execution.DrawdownSourceRef, "usage_ledger:execution:") ||
		execution.DrawdownRefreshedAt <= 0 {
		log.Fatalf("unexpected supply action execution drawdown: got=%+v original=%+v ledger=%+v", execution, original, ledger)
	}
}

func assertSupplyPrepaidLotRecorded(lot supplyPrepaidLotItem, prepaidModelName string, periodStart int64, periodEnd int64) {
	if lot.Id <= 0 ||
		lot.SupplierId != 3 ||
		lot.SupplyNode != "gb10-4t-self-operated" ||
		lot.ModelName != prepaidModelName ||
		lot.PeriodStart != periodStart ||
		lot.PeriodEnd != periodEnd ||
		lot.PurchasedTokens != 1000 ||
		lot.UnitCostQuota < 0.419999 ||
		lot.UnitCostQuota > 0.420001 ||
		lot.TotalCostQuota < 419.999999 ||
		lot.TotalCostQuota > 420.000001 ||
		lot.DrawdownTokens != 0 ||
		lot.DrawdownRequestCount != 0 ||
		lot.RemainingTokens != 1000 ||
		lot.DrawdownRate != 0 ||
		lot.SourceType != model.SupplyPrepaidLotSourceAccounting ||
		lot.SourceRef != "process-gb10-4t-self-operated-prepaid" ||
		lot.ExternalRef != "po://process-gb10-4t-self-operated" ||
		lot.RecordedBy != 1 {
		log.Fatalf("unexpected recorded supply prepaid lot: got=%+v model=%s period=%d-%d", lot, prepaidModelName, periodStart, periodEnd)
	}
}

func assertSupplyPrepaidLotDrawdown(refreshed supplyPrepaidLotItem, original supplyPrepaidLotItem) {
	if refreshed.Id != original.Id ||
		refreshed.PurchasedTokens != original.PurchasedTokens ||
		refreshed.DrawdownTokens != 320 ||
		refreshed.DrawdownRequestCount != 2 ||
		refreshed.RemainingTokens != 680 ||
		refreshed.DrawdownRate < 0.319999 ||
		refreshed.DrawdownRate > 0.320001 ||
		refreshed.DrawdownSourceType != model.SupplyPrepaidLotDrawdownSourceUsageLedger ||
		!strings.Contains(refreshed.DrawdownSourceRef, "usage_ledger:prepaid_lot:") ||
		refreshed.DrawdownRefreshedAt <= 0 {
		log.Fatalf("unexpected supply prepaid lot drawdown: got=%+v original=%+v", refreshed, original)
	}
}

func assertSupplyRoutingPolicy(policy supplyRoutingPolicyItem, execution supplyActionExecutionItem, capacityID int, slaRun model.SlaProbeRun, trafficPercent int) {
	if policy.Id <= 0 ||
		policy.SupplyActionExecutionId != execution.Id ||
		policy.SupplyActionPlanId != execution.SupplyActionPlanId ||
		policy.SupplyDecisionId != execution.SupplyDecisionId ||
		policy.ModelName != execution.ModelName ||
		policy.SlaTier != execution.SlaTier ||
		policy.UserId != execution.UserId ||
		policy.Track != model.SupplyDecisionTrackSelfHosted ||
		policy.ActionType != execution.ActionType ||
		policy.Status != model.SupplyRoutingPolicyStatusActive ||
		policy.SupplierId != 2 ||
		policy.ChannelId != 3 ||
		policy.SupplyCapacityId != capacityID ||
		policy.SlaContractId != slaRun.ContractId ||
		policy.SlaProbeRunId != slaRun.Id ||
		policy.SlaProbeRunKey != slaRun.RunKey ||
		policy.SlaArtifactSHA256 != slaRun.ArtifactSHA256 ||
		policy.SlaRuntimeRef != slaRun.RuntimeRef ||
		policy.Priority != 100 ||
		policy.TrafficPercent != trafficPercent ||
		policy.ActivatedBy != 1 ||
		policy.ActivatedAt <= 0 {
		log.Fatalf("unexpected supply routing policy: got=%+v execution=%+v sla_run=%+v", policy, execution, slaRun)
	}
}

func assertSupplyRoutingPolicyLedger(ledger ledgerEnvelopeItem, sessionID string, requestID string) {
	if ledger.RequestId != requestID ||
		ledger.SessionId != sessionID ||
		ledger.ChannelId != 3 ||
		ledger.SupplierId != 2 ||
		ledger.SupplyNode != "gb10-4t-self-hosted" ||
		ledger.SellQuota <= ledger.CostQuota {
		log.Fatalf("unexpected policy-routed ledger: got=%+v session=%s", ledger, sessionID)
	}
}

func assertSupplyRoutingPolicyFallbackLedger(ledger ledgerEnvelopeItem, sessionID string, requestID string) {
	if ledger.RequestId != requestID ||
		ledger.SessionId != sessionID ||
		ledger.ChannelId == 3 ||
		ledger.SupplierId != 1 ||
		ledger.SupplyNode == "gb10-4t-self-hosted" ||
		ledger.SellQuota <= ledger.CostQuota {
		log.Fatalf("unexpected policy-canary fallback ledger: got=%+v session=%s", ledger, sessionID)
	}
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "mock-supply":
		runMockSupply(os.Args[2:])
	case "seed":
		runSeed(os.Args[2:])
	case "run":
		runDemand(os.Args[2:])
	default:
		must(errors.New("unknown subcommand: " + os.Args[1]))
	}
}
