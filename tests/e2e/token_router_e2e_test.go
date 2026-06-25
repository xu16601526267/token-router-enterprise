package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/router"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const (
	e2eModelName   = "gpt-test"
	e2eSessionID   = "session-e2e-001"
	e2eDemandToken = "demandtoken"
	e2eAdminToken  = "admin-access-token-00000000000001"
)

type gb10MockSupply struct {
	server        *httptest.Server
	mu            sync.Mutex
	requestsBySID map[string]int
}

func newGB10MockSupply(t *testing.T) *gb10MockSupply {
	t.Helper()
	mock := &gb10MockSupply{requestsBySID: map[string]int{}}
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		sessionID := r.Header.Get("X-Session-Id")
		if sessionID == "" {
			sessionID = r.Header.Get("session_id")
		}
		require.NotEmpty(t, sessionID)
		if sessionID != e2eSessionID {
			require.True(t, strings.HasPrefix(sessionID, "trsess_"), "unexpected session id %q", sessionID)
		}
		mock.mu.Lock()
		mock.requestsBySID[sessionID]++
		callIndex := mock.requestsBySID[sessionID]
		mock.mu.Unlock()

		cachedTokens := 0
		promptTokens := 120
		if callIndex > 1 {
			cachedTokens = 80
			promptTokens = 140
		}
		completionTokens := 20
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      fmt.Sprintf("chatcmpl-gb10-%d", callIndex),
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   e2eModelName,
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
	mock.server = httptest.NewServer(handler)
	t.Cleanup(mock.server.Close)
	return mock
}

func setupTokenRouterE2E(t *testing.T, supply *gb10MockSupply) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "token-router-e2e.db")
	t.Setenv("SQL_DSN", "local")
	t.Setenv("SQLITE_PATH", dbPath+"?_busy_timeout=30000")
	t.Setenv("LOG_SQL_DSN", "")
	common.SQLitePath = dbPath + "?_busy_timeout=30000"
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false
	common.BatchUpdateEnabled = false
	common.DataExportEnabled = false
	common.LogConsumeEnabled = true
	common.IsMasterNode = true
	common.UsingSQLite = false
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	ratio_setting.InitRatioSettings()
	service.ClearChannelAffinityCacheAll()
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"`+e2eModelName+`":1}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"`+e2eModelName+`":1}`))
	require.NoError(t, ratio_setting.UpdateCacheRatioByJSONString(`{"`+e2eModelName+`":0.1}`))
	require.NoError(t, ratio_setting.UpdateCreateCacheRatioByJSONString(`{"`+e2eModelName+`":1.25}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1}`))
	service.InitHttpClient()
	service.InitTokenEncoders()

	require.NoError(t, model.InitDB())
	model.InitOptionMap()
	require.NoError(t, model.InitLogDB())
	t.Cleanup(func() {
		_ = model.CloseDB()
		_ = os.Remove(dbPath)
	})

	seedTokenRouterE2E(t, supply.server.URL)

	engine := gin.New()
	engine.Use(middleware.RequestId())
	engine.Use(middleware.I18n())
	engine.Use(sessions.Sessions("session", cookie.NewStore([]byte(common.SessionSecret))))
	router.SetApiRouter(engine)
	router.SetRelayRouter(engine)
	server := httptest.NewServer(engine)
	t.Cleanup(server.Close)
	return server
}

func seedTokenRouterE2E(t *testing.T, supplyURL string) {
	t.Helper()
	adminAccessToken := e2eAdminToken
	admin := &model.User{
		Id:          1,
		Username:    "root",
		Password:    "unused",
		Role:        common.RoleRootUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Quota:       1_000_000,
		AffCode:     "root-aff",
		AccessToken: &adminAccessToken,
	}
	require.NoError(t, model.DB.Create(admin).Error)
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
	require.NoError(t, model.DB.Create(user).Error)
	token := &model.Token{
		Id:             1,
		UserId:         user.Id,
		Key:            e2eDemandToken,
		Name:           "demand-simulator",
		Status:         common.TokenStatusEnabled,
		ExpiredTime:    -1,
		UnlimitedQuota: true,
		Group:          "default",
	}
	require.NoError(t, model.DB.Create(token).Error)

	supplier := &model.Supplier{Name: "gb10-4t", Type: model.SupplierTypeThirdParty, Status: 1}
	require.NoError(t, supplier.Insert())
	selfHostedSupplier := &model.Supplier{Name: "gb10-4t-self-hosted", Type: model.SupplierTypeSelfHosted, Status: 1}
	require.NoError(t, selfHostedSupplier.Insert())
	selfOperatedSupplier := &model.Supplier{Name: "gb10-4t-self-operated", Type: model.SupplierTypeSelfOperated, Status: 1}
	require.NoError(t, selfOperatedSupplier.Insert())
	for channelID := 1; channelID <= 2; channelID++ {
		channel := &model.Channel{
			Id:         channelID,
			Type:       constant.ChannelTypeOpenAI,
			Key:        "sk-gb10-4t-mock",
			Status:     common.ChannelStatusEnabled,
			Name:       "gb10-4t",
			BaseURL:    &supplyURL,
			Models:     e2eModelName,
			Group:      "default",
			SupplierId: supplier.Id,
		}
		require.NoError(t, model.DB.Create(channel).Error)
		require.NoError(t, model.DB.Create(&model.Ability{
			Group:     "default",
			Model:     e2eModelName,
			ChannelId: channel.Id,
			Enabled:   true,
			Weight:    100,
		}).Error)
	}
	selfHostedPriority := int64(-10)
	selfHostedChannel := &model.Channel{
		Id:         3,
		Type:       constant.ChannelTypeOpenAI,
		Key:        "sk-gb10-4t-self-hosted-mock",
		Status:     common.ChannelStatusEnabled,
		Name:       "gb10-4t-self-hosted",
		BaseURL:    &supplyURL,
		Models:     e2eModelName,
		Group:      "default",
		SupplierId: selfHostedSupplier.Id,
		Priority:   &selfHostedPriority,
	}
	require.NoError(t, model.DB.Create(selfHostedChannel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     "default",
		Model:     e2eModelName,
		ChannelId: selfHostedChannel.Id,
		Enabled:   true,
		Priority:  &selfHostedPriority,
		Weight:    100,
	}).Error)
	require.NoError(t, (&model.SupplierAgreement{
		SupplierId:             supplier.Id,
		ModelName:              e2eModelName,
		EffectiveFrom:          0,
		CostModelRatio:         0.5,
		CostCompletionRatio:    1,
		CostCacheRatio:         0.05,
		CostCacheCreationRatio: 0.5,
		Status:                 1,
	}).Insert())
	require.NoError(t, (&model.SupplierAgreement{
		SupplierId:             selfHostedSupplier.Id,
		ModelName:              e2eModelName,
		EffectiveFrom:          0,
		CostModelRatio:         0.35,
		CostCompletionRatio:    1,
		CostCacheRatio:         0.02,
		CostCacheCreationRatio: 0.35,
		Status:                 1,
	}).Insert())
	now := common.GetTimestamp()
	require.NoError(t, (&model.SupplyCapacity{
		SupplierId:     supplier.Id,
		SupplyNode:     "gb10-4t",
		ModelName:      e2eModelName,
		PeriodStart:    now - 3600,
		PeriodEnd:      now + 3600,
		CapacityTokens: 1000,
		UsedTokens:     0,
		QualityScore:   98.5,
		UnitCostQuota:  0.5,
		Status:         1,
		Notes:          "seeded gb10-4t capacity snapshot",
	}).Insert())
}

func demandSimulatorChat(t *testing.T, baseURL string, requestID string) string {
	t.Helper()
	body := map[string]any{
		"model": e2eModelName,
		"messages": []map[string]string{{
			"role":    "user",
			"content": "hello from demand simulator",
		}},
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-"+e2eDemandToken)
	req.Header.Set("X-Session-Id", e2eSessionID)
	req.Header.Set(common.UsageLedgerRequestIdHeader, requestID)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assignedSession := strings.TrimSpace(resp.Header.Get("X-Session-Id"))
	var decoded map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
	require.NotEmpty(t, decoded["choices"])
	return assignedSession
}

func demandSimulatorChatWithoutSession(t *testing.T, baseURL string, requestID string) string {
	t.Helper()
	body := map[string]any{
		"model": e2eModelName,
		"messages": []map[string]string{{
			"role":    "user",
			"content": "hello from demand simulator without session",
		}},
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-"+e2eDemandToken)
	req.Header.Set(common.UsageLedgerRequestIdHeader, requestID)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assignedSession := strings.TrimSpace(resp.Header.Get("X-Session-Id"))
	require.NotEmpty(t, assignedSession)
	var decoded map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
	require.NotEmpty(t, decoded["choices"])
	return assignedSession
}

func adminGetUsageLedgers(t *testing.T, baseURL string) []model.UsageLedger {
	t.Helper()
	return adminGetUsageLedgersBySession(t, baseURL, e2eSessionID)
}

func adminGetUsageLedgersBySession(t *testing.T, baseURL string, sessionID string) []model.UsageLedger {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/usage_ledgers?page_size=10&session_id="+sessionID, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Total int                 `json:"total"`
			Items []model.UsageLedger `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminGetMarginSummary(t *testing.T, baseURL string) []model.MarginSummaryRow {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/reports/margin_summary?group_by=supplier&supplier_id=1", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                     `json:"success"`
		Data    []model.MarginSummaryRow `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetQualitySummary(t *testing.T, baseURL string, groupBy string) []model.QualitySummaryRow {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/reports/quality_summary?group_by="+groupBy+"&supplier_id=1", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                      `json:"success"`
		Data    []model.QualitySummaryRow `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplyCapacities(t *testing.T, baseURL string) []model.SupplyCapacity {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/supply_capacities?supplier_id=1&supply_node=gb10-4t&model_name="+e2eModelName, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplyCapacity `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminRecordSupplyCapacityTelemetry(t *testing.T, baseURL string, capacity model.SupplyCapacity) model.SupplyCapacityTelemetry {
	t.Helper()
	return adminRecordSupplyCapacityTelemetryInput(t, baseURL, model.SupplyCapacityTelemetryRecordInput{
		SupplierId:         capacity.SupplierId,
		SupplyNode:         capacity.SupplyNode,
		ModelName:          capacity.ModelName,
		PeriodStart:        capacity.PeriodStart,
		PeriodEnd:          capacity.PeriodEnd,
		CapacityTokens:     capacity.CapacityTokens,
		UsedTokens:         capacity.UsedTokens,
		GpuUtilizationRate: 0.62,
		QualityScore:       capacity.QualityScore,
		UnitCostQuota:      capacity.UnitCostQuota,
		SourceType:         model.SupplyCapacityTelemetrySourceNodeReport,
		SourceRef:          "e2e-gb10-4t-capacity-telemetry",
		ObservedAt:         common.GetTimestamp(),
		Notes:              "e2e capacity telemetry",
	})
}

func adminRecordSupplyCapacityTelemetryInput(t *testing.T, baseURL string, input model.SupplyCapacityTelemetryRecordInput) model.SupplyCapacityTelemetry {
	t.Helper()
	payload, err := json.Marshal(input)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_capacity_telemetries/record", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                          `json:"success"`
		Data    model.SupplyCapacityTelemetry `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplyCapacityTelemetries(t *testing.T, baseURL string, supplierID int, modelName string) []model.SupplyCapacityTelemetry {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/supply_capacity_telemetries?supplier_id=%d&supply_node=gb10-4t&model_name=%s", baseURL, supplierID, modelName), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplyCapacityTelemetry `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminRecordSupplyCostProfile(t *testing.T, baseURL string, supplierID int, periodStart int64, periodEnd int64) model.SupplyCostProfile {
	t.Helper()
	payload, err := json.Marshal(model.SupplyCostProfileRecordInput{
		SupplierId:            supplierID,
		SupplyNode:            "gb10-4t-self-hosted",
		ModelName:             e2eModelName,
		PeriodStart:           periodStart,
		PeriodEnd:             periodEnd,
		CapacityTokens:        1000,
		FixedCostQuota:        100,
		VariableUnitCostQuota: 0.02,
		SourceType:            model.SupplyCostProfileSourceAccounting,
		SourceRef:             "e2e-gb10-4t-self-hosted-cost",
		ObservedAt:            common.GetTimestamp(),
		Notes:                 "e2e self-hosted amortized cost basis",
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_cost_profiles/record", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                    `json:"success"`
		Data    model.SupplyCostProfile `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplyCostProfiles(t *testing.T, baseURL string, supplierID int, modelName string) []model.SupplyCostProfile {
	t.Helper()
	values := url.Values{}
	values.Set("supplier_id", fmt.Sprintf("%d", supplierID))
	values.Set("supply_node", "gb10-4t-self-hosted")
	values.Set("model_name", modelName)
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/supply_cost_profiles?"+values.Encode(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplyCostProfile `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminRecordSupplyPrepaidLot(t *testing.T, baseURL string, supplierID int, modelName string, periodStart int64, periodEnd int64) model.SupplyPrepaidLot {
	t.Helper()
	payload, err := json.Marshal(model.SupplyPrepaidLotRecordInput{
		SupplierId:      supplierID,
		SupplyNode:      "gb10-4t-self-operated",
		ModelName:       modelName,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		PurchasedTokens: 1000,
		UnitCostQuota:   0.42,
		SourceType:      model.SupplyPrepaidLotSourceAccounting,
		SourceRef:       "e2e-gb10-4t-self-operated-prepaid",
		ObservedAt:      common.GetTimestamp(),
		ExternalRef:     "po://e2e-gb10-4t-self-operated",
		Notes:           "e2e self-operated prepaid lot",
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_prepaid_lots/record", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                   `json:"success"`
		Data    model.SupplyPrepaidLot `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminRejectSupplyPrepaidLot(t *testing.T, baseURL string, supplierID int, message string) {
	t.Helper()
	now := common.GetTimestamp()
	payload, err := json.Marshal(model.SupplyPrepaidLotRecordInput{
		SupplierId:      supplierID,
		SupplyNode:      "gb10-4t",
		ModelName:       "gpt-prepaid-e2e",
		PeriodStart:     now - 3600,
		PeriodEnd:       now + 3600,
		PurchasedTokens: 1000,
		UnitCostQuota:   0.42,
		SourceType:      model.SupplyPrepaidLotSourceAccounting,
		SourceRef:       "e2e-prepaid-reject",
		ObservedAt:      now,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_prepaid_lots/record", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.False(t, envelope.Success)
	require.Contains(t, envelope.Message, message)
}

func adminGetSupplyPrepaidLots(t *testing.T, baseURL string, supplierID int, modelName string) []model.SupplyPrepaidLot {
	t.Helper()
	values := url.Values{}
	values.Set("supplier_id", fmt.Sprintf("%d", supplierID))
	values.Set("supply_node", "gb10-4t-self-operated")
	values.Set("model_name", modelName)
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/supply_prepaid_lots?"+values.Encode(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplyPrepaidLot `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminRefreshSupplyPrepaidLotUsage(t *testing.T, baseURL string, lotID int) model.SupplyPrepaidLot {
	t.Helper()
	payload, err := json.Marshal(model.SupplyPrepaidLotUsageRefreshInput{PrepaidLotId: lotID})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_prepaid_lots/refresh_usage", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                     `json:"success"`
		Data    []model.SupplyPrepaidLot `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	require.Len(t, envelope.Data, 1)
	return envelope.Data[0]
}

func seedE2EPrepaidUsageLedgers(t *testing.T, supplierID int, modelName string, periodStart int64) {
	t.Helper()
	require.NoError(t, (&model.UsageLedger{
		RequestId:        "e2e-prepaid-drawdown-1",
		SessionId:        "e2e-prepaid-session",
		SupplierId:       supplierID,
		ModelName:        modelName,
		PromptTokens:     100,
		CompletionTokens: 40,
		Status:           "success",
		SupplyNode:       "gb10-4t-self-operated",
		CreatedAt:        periodStart + 10,
	}).InsertIdempotent())
	require.NoError(t, (&model.UsageLedger{
		RequestId:        "e2e-prepaid-drawdown-2",
		SessionId:        "e2e-prepaid-session",
		SupplierId:       supplierID,
		ModelName:        modelName,
		PromptTokens:     120,
		CompletionTokens: 60,
		Status:           "success",
		SupplyNode:       "gb10-4t-self-operated",
		CreatedAt:        periodStart + 20,
	}).InsertIdempotent())
	require.NoError(t, (&model.UsageLedger{
		RequestId:        "e2e-prepaid-drawdown-failed",
		SessionId:        "e2e-prepaid-session",
		SupplierId:       supplierID,
		ModelName:        modelName,
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "failed",
		SupplyNode:       "gb10-4t-self-operated",
		CreatedAt:        periodStart + 30,
	}).InsertIdempotent())
}

func adminRefreshSupplyCapacityUsage(t *testing.T, baseURL string) []model.SupplyCapacity {
	t.Helper()
	now := common.GetTimestamp()
	payload, err := json.Marshal(model.SupplyCapacityUsageRefreshInput{
		SupplierId:  1,
		SupplyNode:  "gb10-4t",
		ModelName:   e2eModelName,
		PeriodStart: now - 3600,
		PeriodEnd:   now + 3600,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_capacities/refresh_usage", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                   `json:"success"`
		Data    []model.SupplyCapacity `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGenerateTrafficProfiles(t *testing.T, baseURL string) []model.TrafficProfile {
	t.Helper()
	now := common.GetTimestamp()
	payload, err := json.Marshal(model.TrafficProfileGenerateInput{
		PeriodStart: now - 3600,
		PeriodEnd:   now + 3600,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/traffic_profiles/generate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                   `json:"success"`
		Data    []model.TrafficProfile `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetTrafficProfiles(t *testing.T, baseURL string) []model.TrafficProfile {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/traffic_profiles?page_size=10&model_name="+e2eModelName+"&sla_tier=default&user_id=2", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.TrafficProfile `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminGenerateTrafficForecasts(t *testing.T, baseURL string, periodStart int64, periodEnd int64) []model.TrafficForecast {
	t.Helper()
	return adminGenerateTrafficForecastsWithInput(t, baseURL, model.TrafficForecastGenerateInput{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
}

func adminGenerateTrafficForecastsWithInput(t *testing.T, baseURL string, input model.TrafficForecastGenerateInput) []model.TrafficForecast {
	t.Helper()
	payload, err := json.Marshal(input)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/traffic_forecasts/generate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                    `json:"success"`
		Data    []model.TrafficForecast `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetTrafficForecasts(t *testing.T, baseURL string, targetStart int64, targetEnd int64) []model.TrafficForecast {
	t.Helper()
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("model_name", e2eModelName)
	values.Set("sla_tier", "default")
	values.Set("user_id", "2")
	values.Set("target_start_timestamp", fmt.Sprintf("%d", targetStart))
	values.Set("target_end_timestamp", fmt.Sprintf("%d", targetEnd))
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/traffic_forecasts?"+values.Encode(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.TrafficForecast `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func seedE2ESeasonalAnomalyTrafficProfiles(t *testing.T, sourceStart int64) (int64, int64, int64, int64) {
	t.Helper()
	periodSeconds := int64(1_000)
	sliceKey := fmt.Sprintf("model:%s|sla:seasonal|user:99", e2eModelName)
	demands := []int64{100, 300, 120, 360}
	peaks := []int64{130, 330, 160, 390}
	headrooms := []int64{500, 400, 350, 250}
	profiles := make([]*model.TrafficProfile, 0, len(demands))
	for index, demand := range demands {
		periodStart := sourceStart + int64(index)*periodSeconds
		periodEnd := periodStart + periodSeconds
		profiles = append(profiles, &model.TrafficProfile{
			SliceKey:             sliceKey,
			ModelName:            e2eModelName,
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
	require.NoError(t, model.DB.Create(&profiles).Error)
	sourceEnd := sourceStart + int64(len(demands))*periodSeconds
	targetStart := sourceEnd
	targetEnd := targetStart + periodSeconds
	return sourceStart, sourceEnd, targetStart, targetEnd
}

func adminGenerateSupplyDecisions(t *testing.T, baseURL string, periodStart int64, periodEnd int64) []model.SupplyDecision {
	t.Helper()
	payload, err := json.Marshal(model.SupplyDecisionGenerateInput{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_decisions/generate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                   `json:"success"`
		Data    []model.SupplyDecision `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplyDecisions(t *testing.T, baseURL string, status string) []model.SupplyDecision {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/supply_decisions?page_size=10&model_name="+e2eModelName+"&sla_tier=default&user_id=2&status="+status, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplyDecision `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminApproveSupplyDecision(t *testing.T, baseURL string, decisionID int) model.SupplyDecision {
	t.Helper()
	payload, err := json.Marshal(model.SupplyDecisionReviewInput{ReviewNote: "accepted in e2e"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supply_decisions/%d/approve", baseURL, decisionID), bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                 `json:"success"`
		Data    model.SupplyDecision `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGenerateSupplyExpansionOpportunities(t *testing.T, baseURL string, periodStart int64, periodEnd int64) []model.SupplyExpansionOpportunity {
	t.Helper()
	payload, err := json.Marshal(model.SupplyExpansionOpportunityGenerateInput{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_expansion_opportunities/generate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                               `json:"success"`
		Data    []model.SupplyExpansionOpportunity `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplyExpansionOpportunities(t *testing.T, baseURL string, opportunityType string) []model.SupplyExpansionOpportunity {
	t.Helper()
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("model_name", e2eModelName)
	values.Set("sla_tier", "default")
	values.Set("user_id", "2")
	if opportunityType != "" {
		values.Set("opportunity_type", opportunityType)
	}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/supply_expansion_opportunities?"+values.Encode(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplyExpansionOpportunity `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminGeneratePricingRecommendations(t *testing.T, baseURL string, periodStart int64, periodEnd int64) []model.PricingRecommendation {
	t.Helper()
	payload, err := json.Marshal(model.PricingRecommendationGenerateInput{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/pricing_recommendations/generate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                          `json:"success"`
		Data    []model.PricingRecommendation `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetPricingRecommendations(t *testing.T, baseURL string, status string) []model.PricingRecommendation {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/pricing_recommendations?page_size=10&model_name="+e2eModelName+"&sla_tier=default&user_id=2&status="+status, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.PricingRecommendation `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminApprovePricingRecommendation(t *testing.T, baseURL string, recommendationID int) model.PricingRecommendation {
	t.Helper()
	payload, err := json.Marshal(model.PricingRecommendationReviewInput{ReviewNote: "accepted pricing recommendation in e2e"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/pricing_recommendations/%d/approve", baseURL, recommendationID), bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                        `json:"success"`
		Data    model.PricingRecommendation `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGenerateOperatingInsights(t *testing.T, baseURL string, periodStart int64, periodEnd int64) []model.OperatingInsight {
	t.Helper()
	payload, err := json.Marshal(model.OperatingInsightGenerateInput{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/operating_insights/generate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                     `json:"success"`
		Data    []model.OperatingInsight `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetOperatingInsights(t *testing.T, baseURL string, status string, category string) []model.OperatingInsight {
	t.Helper()
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("model_name", e2eModelName)
	values.Set("sla_tier", "default")
	values.Set("user_id", "2")
	values.Set("status", status)
	if category != "" {
		values.Set("category", category)
	}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/operating_insights?"+values.Encode(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.OperatingInsight `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminGetGlobalOperatingInsights(t *testing.T, baseURL string, status string, category string, periodStart int64, periodEnd int64) []model.OperatingInsight {
	t.Helper()
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("model_name", e2eModelName)
	values.Set("sla_tier", "default")
	values.Set("status", status)
	if category != "" {
		values.Set("category", category)
	}
	if periodStart > 0 {
		values.Set("start_timestamp", fmt.Sprintf("%d", periodStart))
	}
	if periodEnd > 0 {
		values.Set("end_timestamp", fmt.Sprintf("%d", periodEnd))
	}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/operating_insights?"+values.Encode(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.OperatingInsight `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminAcknowledgeOperatingInsight(t *testing.T, baseURL string, insightID int) model.OperatingInsight {
	t.Helper()
	payload, err := json.Marshal(model.OperatingInsightReviewInput{ReviewNote: "acknowledged operating insight in e2e"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/operating_insights/%d/acknowledge", baseURL, insightID), bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                   `json:"success"`
		Data    model.OperatingInsight `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGenerateSupplyActionPlans(t *testing.T, baseURL string, decisionID int) []model.SupplyActionPlan {
	t.Helper()
	payload, err := json.Marshal(model.SupplyActionPlanGenerateInput{
		DecisionId: decisionID,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_action_plans/generate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                     `json:"success"`
		Data    []model.SupplyActionPlan `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplyActionPlans(t *testing.T, baseURL string, decisionID int) []model.SupplyActionPlan {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/supply_action_plans?page_size=10&decision_id=%d&status=%s", baseURL, decisionID, model.SupplyActionPlanStatusPlanned), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplyActionPlan `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminUpdateSupplyActionPlanStatus(t *testing.T, baseURL string, planID int, status string, note string) model.SupplyActionPlan {
	t.Helper()
	payload, err := json.Marshal(model.SupplyActionPlanStatusInput{
		Status:       status,
		OperatorNote: note,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supply_action_plans/%d/status", baseURL, planID), bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                   `json:"success"`
		Data    model.SupplyActionPlan `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminRejectSupplyActionPlanStatus(t *testing.T, baseURL string, planID int, status string) {
	t.Helper()
	payload, err := json.Marshal(model.SupplyActionPlanStatusInput{
		Status:       status,
		OperatorNote: "should be rejected",
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supply_action_plans/%d/status", baseURL, planID), bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.False(t, envelope.Success)
	require.Contains(t, envelope.Message, "invalid supply action plan status transition")
}

func adminRecordSupplyActionExecution(t *testing.T, baseURL string, input model.SupplyActionExecutionRecordInput) model.SupplyActionExecution {
	t.Helper()
	payload, err := json.Marshal(input)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_action_executions/record", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                        `json:"success"`
		Data    model.SupplyActionExecution `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminRejectSupplyActionExecution(t *testing.T, baseURL string, input model.SupplyActionExecutionRecordInput, message string) {
	t.Helper()
	payload, err := json.Marshal(input)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_action_executions/record", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.False(t, envelope.Success)
	require.Contains(t, envelope.Message, message)
}

func adminGetSupplyActionExecutions(t *testing.T, baseURL string, planID int) []model.SupplyActionExecution {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/supply_action_executions?page_size=10&supply_action_plan_id=%d&execution_status=%s", baseURL, planID, model.SupplyActionExecutionStatusRecorded), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplyActionExecution `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminActivateSupplyRoutingPolicy(t *testing.T, baseURL string, executionID int) model.SupplyRoutingPolicy {
	t.Helper()
	payload, err := json.Marshal(model.SupplyRoutingPolicyActivateInput{
		SupplyActionExecutionId: executionID,
		Priority:                100,
		OperatorNote:            "activate self-hosted routing policy in e2e",
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_routing_policies/activate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                      `json:"success"`
		Data    model.SupplyRoutingPolicy `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminRejectSupplyRoutingPolicy(t *testing.T, baseURL string, executionID int, message string) {
	t.Helper()
	payload, err := json.Marshal(model.SupplyRoutingPolicyActivateInput{
		SupplyActionExecutionId: executionID,
		Priority:                100,
		OperatorNote:            "should be rejected",
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_routing_policies/activate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.False(t, envelope.Success)
	require.Contains(t, envelope.Message, message)
}

func adminDisableSupplyRoutingPolicy(t *testing.T, baseURL string, policyID int) model.SupplyRoutingPolicy {
	t.Helper()
	payload, err := json.Marshal(model.SupplyRoutingPolicyDisableInput{OperatorNote: "disable self-hosted routing policy in e2e"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supply_routing_policies/%d/disable", baseURL, policyID), bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                      `json:"success"`
		Data    model.SupplyRoutingPolicy `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplyRoutingPolicies(t *testing.T, baseURL string, executionID int) []model.SupplyRoutingPolicy {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/supply_routing_policies?page_size=10&supply_action_execution_id=%d", baseURL, executionID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplyRoutingPolicy `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminGenerateSupplierScorecards(t *testing.T, baseURL string) []model.SupplierScorecard {
	t.Helper()
	now := common.GetTimestamp()
	payload, err := json.Marshal(model.SupplierScorecardGenerateInput{
		PeriodStart: now - 3600,
		PeriodEnd:   now + 3600,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supplier_scorecards/generate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                      `json:"success"`
		Data    []model.SupplierScorecard `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplierScorecards(t *testing.T, baseURL string, grade string) []model.SupplierScorecard {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/supplier_scorecards?page_size=10&supplier_id=1&grade="+grade, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplierScorecard `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminGenerateSupplierEvaluations(t *testing.T, baseURL string, periodStart int64, periodEnd int64) []model.SupplierEvaluation {
	t.Helper()
	payload, err := json.Marshal(model.SupplierEvaluationGenerateInput{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supplier_evaluations/generate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                       `json:"success"`
		Data    []model.SupplierEvaluation `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplierEvaluations(t *testing.T, baseURL string, status string) []model.SupplierEvaluation {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/supplier_evaluations?page_size=10&supplier_id=1&status="+status, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplierEvaluation `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminApproveSupplierEvaluation(t *testing.T, baseURL string, evaluationID int) model.SupplierEvaluation {
	t.Helper()
	payload, err := json.Marshal(model.SupplierEvaluationReviewInput{ReviewNote: "accepted admission evaluation in e2e"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supplier_evaluations/%d/approve", baseURL, evaluationID), bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                     `json:"success"`
		Data    model.SupplierEvaluation `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminApplySupplierEvaluation(t *testing.T, baseURL string, evaluationID int) model.SupplierEvaluation {
	t.Helper()
	payload, err := json.Marshal(model.SupplierEvaluationApplyInput{OperatorNote: "applied admission evaluation in e2e"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supplier_evaluations/%d/apply", baseURL, evaluationID), bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                     `json:"success"`
		Data    model.SupplierEvaluation `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGenerateSupplierPostureRecommendations(t *testing.T, baseURL string, periodStart int64, periodEnd int64) []model.SupplierPostureRecommendation {
	t.Helper()
	payload, err := json.Marshal(model.SupplierPostureRecommendationGenerateInput{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supplier_posture_recommendations/generate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                                  `json:"success"`
		Data    []model.SupplierPostureRecommendation `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplierPostureRecommendations(t *testing.T, baseURL string, status string, action string) []model.SupplierPostureRecommendation {
	t.Helper()
	values := url.Values{}
	values.Set("page_size", "10")
	values.Set("supplier_id", "1")
	if status != "" {
		values.Set("status", status)
	}
	if action != "" {
		values.Set("recommended_action", action)
	}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/supplier_posture_recommendations?"+values.Encode(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplierPostureRecommendation `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminApproveSupplierPostureRecommendation(t *testing.T, baseURL string, recommendationID int) model.SupplierPostureRecommendation {
	t.Helper()
	payload, err := json.Marshal(model.SupplierPostureRecommendationReviewInput{ReviewNote: "approved posture recommendation in e2e"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supplier_posture_recommendations/%d/approve", baseURL, recommendationID), bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                                `json:"success"`
		Data    model.SupplierPostureRecommendation `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminApplySupplierPostureRecommendation(t *testing.T, baseURL string, recommendationID int) model.SupplierPostureRecommendation {
	t.Helper()
	payload, err := json.Marshal(model.SupplierPostureRecommendationApplyInput{OperatorNote: "applied posture recommendation in e2e"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supplier_posture_recommendations/%d/apply", baseURL, recommendationID), bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                                `json:"success"`
		Data    model.SupplierPostureRecommendation `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplierRoutePreferences(t *testing.T, baseURL string, supplierID int, status string) []model.SupplierRoutePreference {
	t.Helper()
	values := url.Values{}
	values.Set("page_size", "10")
	if supplierID > 0 {
		values.Set("supplier_id", fmt.Sprintf("%d", supplierID))
	}
	if status != "" {
		values.Set("status", status)
	}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/supplier_route_preferences?"+values.Encode(), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplierRoutePreference `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminActivateSupplierRoutePreference(t *testing.T, baseURL string, input model.SupplierRoutePreferenceActivateInput) model.SupplierRoutePreference {
	t.Helper()
	body, err := json.Marshal(input)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supplier_route_preferences/activate", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                          `json:"success"`
		Data    model.SupplierRoutePreference `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminDisableSupplierRoutePreference(t *testing.T, baseURL string, supplierID int, operatorNote string) model.SupplierRoutePreference {
	t.Helper()
	body, err := json.Marshal(model.SupplierRoutePreferenceDisableInput{OperatorNote: operatorNote})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/supplier_route_preferences/%d/disable", baseURL, supplierID), bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                          `json:"success"`
		Data    model.SupplierRoutePreference `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplier(t *testing.T, baseURL string, supplierID int) model.Supplier {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/suppliers/%d", baseURL, supplierID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool           `json:"success"`
		Data    model.Supplier `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSupplyCapacitiesByNode(t *testing.T, baseURL string, supplyNode string) []model.SupplyCapacity {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/supply_capacities?supplier_id=1&supply_node="+supplyNode+"&model_name="+e2eModelName, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SupplyCapacity `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminCreateSupplyCapacity(t *testing.T, baseURL string, capacity model.SupplyCapacity) model.SupplyCapacity {
	t.Helper()
	payload, err := json.Marshal(capacity)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/supply_capacities", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                 `json:"success"`
		Data    model.SupplyCapacity `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminUpdateSupplyCapacity(t *testing.T, baseURL string, capacity model.SupplyCapacity) model.SupplyCapacity {
	t.Helper()
	payload, err := json.Marshal(capacity)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPut, baseURL+"/api/supply_capacities", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                 `json:"success"`
		Data    model.SupplyCapacity `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminDeleteSupplyCapacity(t *testing.T, baseURL string, capacityID int) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/supply_capacities/%d", baseURL, capacityID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
}

func adminGenerateSupplierStatement(t *testing.T, baseURL string) model.SettlementStatement {
	t.Helper()
	now := common.GetTimestamp()
	payload, err := json.Marshal(model.SettlementStatementGenerateInput{
		SubjectType: model.SettlementSubjectSupplier,
		SupplierId:  1,
		PeriodStart: now - 3600,
		PeriodEnd:   now + 3600,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/settlement_statements/generate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool                      `json:"success"`
		Data    model.SettlementStatement `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSettlementItems(t *testing.T, baseURL string, statementID int) []model.UsageLedger {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/settlement_statements/%d/items?page_size=10", baseURL, statementID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Total int                 `json:"total"`
			Items []model.UsageLedger `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	require.Equal(t, len(envelope.Data.Items), envelope.Data.Total)
	return envelope.Data.Items
}

func adminGetSettlementCSV(t *testing.T, baseURL string, statementID int) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/settlement_statements/%d/items.csv", baseURL, statementID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body bytes.Buffer
	_, err = body.ReadFrom(resp.Body)
	require.NoError(t, err)
	return body.String()
}

func adminImportSlaContract(t *testing.T, baseURL string, input model.SlaContractImportInput) model.SlaContract {
	t.Helper()
	payload, err := json.Marshal(input)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/sla_contracts/import", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool              `json:"success"`
		Data    model.SlaContract `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSlaContract(t *testing.T, baseURL string, id int) model.SlaContract {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/sla_contracts/%d", baseURL, id), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool              `json:"success"`
		Data    model.SlaContract `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSlaContracts(t *testing.T, baseURL string, status string) []model.SlaContract {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/sla_contracts?page_size=10&status=%s", baseURL, url.QueryEscape(status)), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SlaContract `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminGenerateSlaProbePlan(t *testing.T, baseURL string, input model.SlaProbePlanGenerateInput) model.SlaProbePlan {
	t.Helper()
	payload, err := json.Marshal(input)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/sla_probe_plans/generate", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool               `json:"success"`
		Data    model.SlaProbePlan `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSlaProbePlan(t *testing.T, baseURL string, id int) model.SlaProbePlan {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/sla_probe_plans/%d", baseURL, id), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool               `json:"success"`
		Data    model.SlaProbePlan `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSlaProbePlans(t *testing.T, baseURL string, contractID int) []model.SlaProbePlan {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/sla_probe_plans?page_size=10&contract_id=%d", baseURL, contractID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SlaProbePlan `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func adminRecordSlaProbeRun(t *testing.T, baseURL string, input model.SlaProbeRunRecordInput) model.SlaProbeRun {
	t.Helper()
	payload, err := json.Marshal(input)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/sla_probe_runs/record", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool              `json:"success"`
		Data    model.SlaProbeRun `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSlaProbeRun(t *testing.T, baseURL string, id int) model.SlaProbeRun {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/sla_probe_runs/%d", baseURL, id), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool              `json:"success"`
		Data    model.SlaProbeRun `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data
}

func adminGetSlaProbeRuns(t *testing.T, baseURL string, planID int) []model.SlaProbeRun {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/sla_probe_runs?page_size=10&plan_id=%d", baseURL, planID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+e2eAdminToken)
	req.Header.Set("New-Api-User", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.SlaProbeRun `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success)
	return envelope.Data.Items
}

func TestTokenRouterSlaMeasurementEvidenceAPI(t *testing.T) {
	supply := newGB10MockSupply(t)
	tokenRouter := setupTokenRouterE2E(t, supply)

	contract := adminImportSlaContract(t, tokenRouter.URL, model.SlaContractImportInput{
		ContractKey:    "kimi-k25-official-e2e",
		ModelName:      e2eModelName,
		ModelAliases:   "kimi-k2.5,kimi-k25",
		ProviderFamily: "kimi",
		SourceName:     "Kimi K2.5 serving requirements",
		SourceRef:      "contracts/kimi-k25-official.json",
		SourceSHA256:   "e2e-sha256",
		Version:        "2026-06-23",
		Status:         model.SlaContractStatusActive,
		MeasurementProfileJSON: `{
			"input_profile":{"buckets":[{"name":"smoke","max_tokens":4096}]},
			"output_profile":{"target_tokens":128},
			"concurrency_profile":{"concurrency":1},
			"rate_profile":{"rpm":10},
			"stream_profile":{"include_usage_required":true},
			"error_profile":{"max_error_rate":0.01},
			"availability_profile":{"window_seconds":600},
			"cache_profile":"cold_no_cache"
		}`,
		HardGateJSON: `{"ttft_ms":{"p90_lte":8000}}`,
		SoftGateJSON: `{"warning_error_rate":0.005}`,
	})
	require.Positive(t, contract.Id)
	require.Equal(t, model.SlaContractStatusActive, contract.Status)
	require.Equal(t, 1, contract.ImportedBy)
	require.Contains(t, adminGetSlaContract(t, tokenRouter.URL, contract.Id).MeasurementProfileJSON, "input_profile")
	require.Len(t, adminGetSlaContracts(t, tokenRouter.URL, model.SlaContractStatusActive), 1)

	plan := adminGenerateSlaProbePlan(t, tokenRouter.URL, model.SlaProbePlanGenerateInput{
		ContractId:     contract.Id,
		SupplierId:     1,
		ChannelId:      1,
		SlaTier:        "default",
		ProbeType:      model.SlaProbeTypeAdmission,
		RouteMode:      model.SlaProbeRouteModeThroughTokenRouter,
		PromptSuiteKey: "e2e-smoke",
		SampleSize:     2,
		RepeatCount:    1,
		MaxProbeQuota:  1000,
	})
	require.Positive(t, plan.Id)
	require.Equal(t, contract.Id, plan.ContractId)
	require.Equal(t, 1, plan.SupplierId)
	require.Equal(t, 1, plan.ChannelId)
	require.Equal(t, model.SlaProbeRouteModeThroughTokenRouter, plan.RouteMode)
	require.Contains(t, plan.InputProfileJSON, "smoke")
	require.Contains(t, plan.StreamProfileJSON, "include_usage_required")
	require.Contains(t, plan.ErrorProfileJSON, "max_error_rate")
	require.Equal(t, "cold_no_cache", plan.CacheProfile)
	require.Equal(t, plan.Id, adminGetSlaProbePlan(t, tokenRouter.URL, plan.Id).Id)
	require.Len(t, adminGetSlaProbePlans(t, tokenRouter.URL, contract.Id), 1)

	run := adminRecordSlaProbeRun(t, tokenRouter.URL, model.SlaProbeRunRecordInput{
		RunKey:         "e2e-sla-run-1",
		PlanId:         plan.Id,
		Status:         model.SlaProbeRunStatusPassed,
		StartedAt:      1000,
		EndedAt:        1200,
		RunnerVersion:  "token-router-sla-e2e",
		GitCommit:      "e2e",
		RuntimeRef:     "httptest",
		Endpoint:       tokenRouter.URL + "/v1/chat/completions",
		SummaryJSON:    `{"ttft_ms":{"p90":500},"usage":{"streaming":true}}`,
		HardGatePassed: true,
		ArtifactURI:    "output/sla/e2e-sla-run-1",
		ArtifactSHA256: "e2e-artifact-sha",
	})
	require.Positive(t, run.Id)
	require.Equal(t, plan.Id, run.PlanId)
	require.Equal(t, contract.Id, run.ContractId)
	require.Equal(t, model.SlaProbeRunStatusPassed, run.Status)
	require.True(t, run.HardGatePassed)
	require.Equal(t, "e2e-artifact-sha", run.ArtifactSHA256)
	require.Equal(t, 1, run.RecordedBy)
	require.Contains(t, adminGetSlaProbeRun(t, tokenRouter.URL, run.Id).SummaryJSON, "ttft_ms")
	require.Len(t, adminGetSlaProbeRuns(t, tokenRouter.URL, plan.Id), 1)
}

func TestTokenRouterOperatingInsightSlaProbeRunEvidenceAPI(t *testing.T) {
	supply := newGB10MockSupply(t)
	tokenRouter := setupTokenRouterE2E(t, supply)
	now := common.GetTimestamp()
	periodStart := now - 60
	startedAt := now - 20
	endedAt := now - 10
	periodEnd := now + 60

	contract := adminImportSlaContract(t, tokenRouter.URL, model.SlaContractImportInput{
		ContractKey:    "operating-insight-sla-e2e",
		ModelName:      e2eModelName,
		ModelAliases:   "kimi-k2.5,kimi-k25",
		ProviderFamily: "kimi",
		SourceName:     "Kimi K2.5 serving requirements",
		SourceRef:      "contracts/kimi-k25-official.json",
		SourceSHA256:   "operating-insight-e2e-sha256",
		Version:        "2026-06-23",
		Status:         model.SlaContractStatusActive,
		MeasurementProfileJSON: `{
			"input_profile":{"buckets":[{"name":"smoke","max_tokens":4096}]},
			"output_profile":{"target_tokens":128},
			"concurrency_profile":{"concurrency":1},
			"rate_profile":{"rpm":10},
			"stream_profile":{"include_usage_required":true},
			"error_profile":{"max_error_rate":0.01},
			"availability_profile":{"window_seconds":600},
			"cache_profile":"cold_no_cache"
		}`,
		HardGateJSON: `{"ttft_ms":{"p90_lte":8000}}`,
		SoftGateJSON: `{"warning_error_rate":0.005}`,
	})
	plan := adminGenerateSlaProbePlan(t, tokenRouter.URL, model.SlaProbePlanGenerateInput{
		ContractId:     contract.Id,
		SupplierId:     1,
		ChannelId:      1,
		SlaTier:        "default",
		ProbeType:      model.SlaProbeTypeRuntimeLight,
		RouteMode:      model.SlaProbeRouteModeDirectUpstream,
		PromptSuiteKey: "runtime-watch",
		SampleSize:     1,
		RepeatCount:    1,
		MaxProbeQuota:  1000,
	})
	run := adminRecordSlaProbeRun(t, tokenRouter.URL, model.SlaProbeRunRecordInput{
		RunKey:         "e2e-operating-insight-sla-run",
		PlanId:         plan.Id,
		Status:         model.SlaProbeRunStatusFailed,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		RunnerVersion:  "token-router-sla-e2e",
		GitCommit:      "e2e",
		RuntimeRef:     "aima2/runtime-watch",
		Endpoint:       tokenRouter.URL + "/v1/chat/completions",
		SummaryJSON:    `{"ttft_ms":{"p90":9000},"usage":{"streaming":true}}`,
		HardGatePassed: false,
		FailureReasons: "ttft p90 exceeded hard gate",
		ArtifactURI:    "output/sla/e2e-operating-insight-sla-run",
		ArtifactSHA256: "operating-insight-failed-run-sha",
	})

	insights := adminGenerateOperatingInsights(t, tokenRouter.URL, periodStart, periodEnd)
	var slaInsight *model.OperatingInsight
	for i := range insights {
		if insights[i].Category == model.OperatingInsightCategoryQualityWatch && insights[i].SlaProbeRunId == run.Id {
			slaInsight = &insights[i]
			break
		}
	}
	require.NotNil(t, slaInsight)
	require.Equal(t, model.OperatingInsightSeverityAction, slaInsight.Severity)
	require.Equal(t, model.OperatingInsightStatusDraft, slaInsight.Status)
	require.Zero(t, slaInsight.TrafficProfileId)
	require.Zero(t, slaInsight.UserId)
	require.Equal(t, e2eModelName, slaInsight.ModelName)
	require.Equal(t, "default", slaInsight.SlaTier)
	require.Equal(t, startedAt, slaInsight.PeriodStart)
	require.Equal(t, endedAt, slaInsight.PeriodEnd)
	require.Equal(t, contract.Id, slaInsight.SlaContractId)
	require.Equal(t, run.Id, slaInsight.SlaProbeRunId)
	require.Equal(t, run.RunKey, slaInsight.SlaProbeRunKey)
	require.Equal(t, model.SlaProbeRunStatusFailed, slaInsight.SlaProbeStatus)
	require.False(t, slaInsight.SlaHardGatePassed)
	require.Equal(t, "ttft p90 exceeded hard gate", slaInsight.SlaFailureReasons)
	require.Equal(t, "operating-insight-failed-run-sha", slaInsight.SlaArtifactSHA256)
	require.Equal(t, "aima2/runtime-watch", slaInsight.SlaRuntimeRef)
	require.Contains(t, slaInsight.Summary, "e2e-operating-insight-sla-run")
	require.Contains(t, slaInsight.RecommendedAction, "probe artifact")

	queriedInsights := adminGetGlobalOperatingInsights(t, tokenRouter.URL, model.OperatingInsightStatusDraft, model.OperatingInsightCategoryQualityWatch, periodStart, periodEnd)
	require.Len(t, queriedInsights, 1)
	require.Equal(t, slaInsight.InsightKey, queriedInsights[0].InsightKey)
	require.Equal(t, run.Id, queriedInsights[0].SlaProbeRunId)

	acknowledgedInsight := adminAcknowledgeOperatingInsight(t, tokenRouter.URL, slaInsight.Id)
	require.Equal(t, model.OperatingInsightStatusAcknowledged, acknowledgedInsight.Status)
	require.Equal(t, 1, acknowledgedInsight.ReviewedBy)
	require.Greater(t, acknowledgedInsight.ReviewedAt, int64(0))
	require.Equal(t, "acknowledged operating insight in e2e", acknowledgedInsight.ReviewNote)
	require.Len(t, adminGetGlobalOperatingInsights(t, tokenRouter.URL, model.OperatingInsightStatusAcknowledged, model.OperatingInsightCategoryQualityWatch, periodStart, periodEnd), 1)

	regeneratedInsights := adminGenerateOperatingInsights(t, tokenRouter.URL, periodStart, periodEnd)
	var regeneratedSlaInsight *model.OperatingInsight
	for i := range regeneratedInsights {
		if regeneratedInsights[i].Category == model.OperatingInsightCategoryQualityWatch && regeneratedInsights[i].SlaProbeRunId == run.Id {
			regeneratedSlaInsight = &regeneratedInsights[i]
			break
		}
	}
	require.NotNil(t, regeneratedSlaInsight)
	require.Equal(t, slaInsight.Id, regeneratedSlaInsight.Id)
	require.Equal(t, model.OperatingInsightStatusAcknowledged, regeneratedSlaInsight.Status)
	require.Equal(t, acknowledgedInsight.ReviewNote, regeneratedSlaInsight.ReviewNote)
	require.Equal(t, run.Id, regeneratedSlaInsight.SlaProbeRunId)
}

func TestTokenRouterSupplierPostureRecommendationAPI(t *testing.T) {
	supply := newGB10MockSupply(t)
	tokenRouter := setupTokenRouterE2E(t, supply)
	now := common.GetTimestamp()
	periodStart := now - 600
	periodEnd := now + 600

	scorecard := &model.SupplierScorecard{
		SupplierId:            1,
		PeriodStart:           periodStart,
		PeriodEnd:             periodEnd,
		TotalRequests:         10,
		SuccessRequests:       3,
		ErrorRequests:         7,
		SuccessRate:           0.3,
		AvgLatencyMs:          2500,
		MaxLatencyMs:          4000,
		SupplyHeadroomTokens:  10,
		AvgSupplyQualityScore: 45,
		Score:                 42.5,
		Grade:                 model.SupplierScorecardGradeD,
		GeneratedAt:           now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	require.NoError(t, model.DB.Create(scorecard).Error)
	insight := &model.OperatingInsight{
		InsightKey:           "e2e-posture-capacity-risk",
		SliceKey:             "capacity:supplier:1|node:gb10-4t|model:gpt-test|reason:low_headroom",
		ModelName:            e2eModelName,
		SlaTier:              "default",
		PeriodStart:          periodStart,
		PeriodEnd:            periodEnd,
		Status:               model.OperatingInsightStatusDraft,
		Severity:             model.OperatingInsightSeverityAction,
		Category:             model.OperatingInsightCategoryCapacityRisk,
		Title:                "Supply node token headroom is low",
		Summary:              "e2e posture capacity risk",
		RecommendedAction:    "review supplier posture",
		SupplyHeadroomTokens: 10,
		GeneratedAt:          now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	require.NoError(t, model.DB.Create(insight).Error)

	recommendations := adminGenerateSupplierPostureRecommendations(t, tokenRouter.URL, periodStart, periodEnd)
	require.Len(t, recommendations, 1)
	require.Equal(t, model.SupplierPostureRecommendationStatusDraft, recommendations[0].Status)
	require.Equal(t, model.SupplierPostureRecommendationActionDisable, recommendations[0].RecommendedAction)
	require.Equal(t, scorecard.Id, recommendations[0].SupplierScorecardId)
	require.Equal(t, 1, recommendations[0].SupplierId)
	require.Equal(t, 1, recommendations[0].CapacityInsightCount)
	require.Equal(t, 1, recommendations[0].ActionInsightCount)

	queried := adminGetSupplierPostureRecommendations(t, tokenRouter.URL, model.SupplierPostureRecommendationStatusDraft, model.SupplierPostureRecommendationActionDisable)
	require.Len(t, queried, 1)
	require.Equal(t, recommendations[0].Id, queried[0].Id)

	approved := adminApproveSupplierPostureRecommendation(t, tokenRouter.URL, recommendations[0].Id)
	require.Equal(t, model.SupplierPostureRecommendationStatusApproved, approved.Status)
	require.Equal(t, 1, approved.ReviewedBy)
	require.Greater(t, approved.ReviewedAt, int64(0))
	require.Equal(t, "approved posture recommendation in e2e", approved.ReviewNote)
	require.Equal(t, common.ChannelStatusEnabled, adminGetSupplier(t, tokenRouter.URL, 1).Status)

	applied := adminApplySupplierPostureRecommendation(t, tokenRouter.URL, recommendations[0].Id)
	require.Equal(t, model.SupplierPostureRecommendationStatusApplied, applied.Status)
	require.Greater(t, applied.AppliedAt, int64(0))
	require.Equal(t, 1, applied.AppliedBy)
	require.Equal(t, common.ChannelStatusEnabled, applied.SupplierStatusBefore)
	require.Equal(t, common.ChannelStatusManuallyDisabled, applied.SupplierStatusAfter)
	require.Contains(t, applied.AppliedNote, "applied posture recommendation in e2e")
	appliedSupplier := adminGetSupplier(t, tokenRouter.URL, 1)
	require.Equal(t, common.ChannelStatusManuallyDisabled, appliedSupplier.Status)
	require.Contains(t, appliedSupplier.Notes, "supplier_posture_recommendation #")
	require.Empty(t, adminGetSupplierRoutePreferences(t, tokenRouter.URL, 1, model.SupplierRoutePreferenceStatusActive))

	downgradeSupplier := &model.Supplier{
		Name:   "e2e-posture-downgrade-supplier",
		Type:   model.SupplierTypeThirdParty,
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, downgradeSupplier.Insert())
	downgradeScorecard := &model.SupplierScorecard{
		SupplierId:            downgradeSupplier.Id,
		PeriodStart:           periodStart,
		PeriodEnd:             periodEnd,
		TotalRequests:         10,
		SuccessRequests:       8,
		ErrorRequests:         2,
		SuccessRate:           0.8,
		AvgLatencyMs:          1200,
		MaxLatencyMs:          1800,
		SupplyHeadroomTokens:  100,
		AvgSupplyQualityScore: 70,
		Score:                 65,
		Grade:                 model.SupplierScorecardGradeC,
		GeneratedAt:           now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	require.NoError(t, model.DB.Create(downgradeScorecard).Error)

	downgradeRecommendations := adminGenerateSupplierPostureRecommendations(t, tokenRouter.URL, periodStart, periodEnd)
	var downgradeRecommendation *model.SupplierPostureRecommendation
	for i := range downgradeRecommendations {
		if downgradeRecommendations[i].SupplierId == downgradeSupplier.Id {
			downgradeRecommendation = &downgradeRecommendations[i]
			break
		}
	}
	require.NotNil(t, downgradeRecommendation)
	require.Equal(t, model.SupplierPostureRecommendationStatusDraft, downgradeRecommendation.Status)
	require.Equal(t, model.SupplierPostureRecommendationActionDowngrade, downgradeRecommendation.RecommendedAction)

	approvedDowngrade := adminApproveSupplierPostureRecommendation(t, tokenRouter.URL, downgradeRecommendation.Id)
	require.Equal(t, model.SupplierPostureRecommendationStatusApproved, approvedDowngrade.Status)
	appliedDowngrade := adminApplySupplierPostureRecommendation(t, tokenRouter.URL, downgradeRecommendation.Id)
	require.Equal(t, model.SupplierPostureRecommendationStatusApplied, appliedDowngrade.Status)
	require.Equal(t, common.ChannelStatusEnabled, appliedDowngrade.SupplierStatusBefore)
	require.Equal(t, common.ChannelStatusEnabled, appliedDowngrade.SupplierStatusAfter)
	require.Equal(t, common.ChannelStatusEnabled, adminGetSupplier(t, tokenRouter.URL, downgradeSupplier.Id).Status)

	preferences := adminGetSupplierRoutePreferences(t, tokenRouter.URL, downgradeSupplier.Id, model.SupplierRoutePreferenceStatusActive)
	require.Len(t, preferences, 1)
	require.Equal(t, downgradeSupplier.Id, preferences[0].SupplierId)
	require.Equal(t, appliedDowngrade.Id, preferences[0].SourcePostureRecommendationId)
	require.Equal(t, model.SupplierRoutePreferenceDowngradeWeightPercent, preferences[0].WeightPercent)
	require.Contains(t, preferences[0].Reason, "supplier_posture_recommendation #")

	manualPreference := adminActivateSupplierRoutePreference(t, tokenRouter.URL, model.SupplierRoutePreferenceActivateInput{
		SupplierId:    downgradeSupplier.Id,
		WeightPercent: 150,
		Reason:        "operator manual route preference boost in e2e",
		EffectiveFrom: periodStart + 10,
		EffectiveTo:   periodEnd,
		OperatorNote:  "e2e manual route preference boost",
	})
	require.Equal(t, downgradeSupplier.Id, manualPreference.SupplierId)
	require.Equal(t, 0, manualPreference.SourcePostureRecommendationId)
	require.Equal(t, model.SupplierRoutePreferenceStatusActive, manualPreference.Status)
	require.Equal(t, 150, manualPreference.WeightPercent)
	require.Equal(t, "operator manual route preference boost in e2e", manualPreference.Reason)
	require.Equal(t, "e2e manual route preference boost", manualPreference.OperatorNote)

	preferences = adminGetSupplierRoutePreferences(t, tokenRouter.URL, downgradeSupplier.Id, model.SupplierRoutePreferenceStatusActive)
	require.Len(t, preferences, 1)
	require.Equal(t, manualPreference.Id, preferences[0].Id)
	require.Equal(t, 0, preferences[0].SourcePostureRecommendationId)
	require.Equal(t, 150, preferences[0].WeightPercent)

	disabledPreference := adminDisableSupplierRoutePreference(t, tokenRouter.URL, downgradeSupplier.Id, "restore baseline after e2e")
	require.Equal(t, manualPreference.Id, disabledPreference.Id)
	require.Equal(t, model.SupplierRoutePreferenceStatusDisabled, disabledPreference.Status)
	require.Equal(t, model.SupplierRoutePreferenceBaselineWeightPercent, disabledPreference.WeightPercent)
	require.Equal(t, 1, disabledPreference.DisabledBy)
	require.Equal(t, "restore baseline after e2e", disabledPreference.OperatorNote)
	require.Empty(t, adminGetSupplierRoutePreferences(t, tokenRouter.URL, downgradeSupplier.Id, model.SupplierRoutePreferenceStatusActive))

	boostSupplier := &model.Supplier{
		Name:   "e2e-posture-boost-supplier",
		Type:   model.SupplierTypeThirdParty,
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, boostSupplier.Insert())
	boostScorecard := &model.SupplierScorecard{
		SupplierId:            boostSupplier.Id,
		PeriodStart:           periodStart,
		PeriodEnd:             periodEnd,
		TotalRequests:         12,
		SuccessRequests:       12,
		SuccessRate:           1,
		AvgLatencyMs:          120,
		MaxLatencyMs:          300,
		SupplyHeadroomTokens:  900,
		AvgSupplyQualityScore: 98,
		Score:                 94,
		Grade:                 model.SupplierScorecardGradeA,
		GeneratedAt:           now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	require.NoError(t, model.DB.Create(boostScorecard).Error)

	boostRecommendations := adminGenerateSupplierPostureRecommendations(t, tokenRouter.URL, periodStart, periodEnd)
	var boostRecommendation *model.SupplierPostureRecommendation
	for i := range boostRecommendations {
		if boostRecommendations[i].SupplierId == boostSupplier.Id {
			boostRecommendation = &boostRecommendations[i]
			break
		}
	}
	require.NotNil(t, boostRecommendation)
	require.Equal(t, model.SupplierPostureRecommendationStatusDraft, boostRecommendation.Status)
	require.Equal(t, model.SupplierPostureRecommendationActionBoost, boostRecommendation.RecommendedAction)
	require.Equal(t, boostScorecard.Id, boostRecommendation.SupplierScorecardId)
	require.Zero(t, boostRecommendation.QualityInsightCount)
	require.Zero(t, boostRecommendation.CapacityInsightCount)
	require.Zero(t, boostRecommendation.ActionInsightCount)

	approvedBoost := adminApproveSupplierPostureRecommendation(t, tokenRouter.URL, boostRecommendation.Id)
	require.Equal(t, model.SupplierPostureRecommendationStatusApproved, approvedBoost.Status)
	appliedBoost := adminApplySupplierPostureRecommendation(t, tokenRouter.URL, boostRecommendation.Id)
	require.Equal(t, model.SupplierPostureRecommendationStatusApplied, appliedBoost.Status)
	require.Equal(t, common.ChannelStatusEnabled, appliedBoost.SupplierStatusBefore)
	require.Equal(t, common.ChannelStatusEnabled, appliedBoost.SupplierStatusAfter)
	require.Equal(t, common.ChannelStatusEnabled, adminGetSupplier(t, tokenRouter.URL, boostSupplier.Id).Status)

	boostPreferences := adminGetSupplierRoutePreferences(t, tokenRouter.URL, boostSupplier.Id, model.SupplierRoutePreferenceStatusActive)
	require.Len(t, boostPreferences, 1)
	require.Equal(t, boostSupplier.Id, boostPreferences[0].SupplierId)
	require.Equal(t, appliedBoost.Id, boostPreferences[0].SourcePostureRecommendationId)
	require.Equal(t, model.SupplierRoutePreferenceBoostWeightPercent, boostPreferences[0].WeightPercent)
	require.Contains(t, boostPreferences[0].Reason, "supplier_posture_recommendation #")
	require.Contains(t, boostPreferences[0].Reason, "boost")
}

func TestTokenRouterE2EGB10SupplyDemandLedger(t *testing.T) {
	supply := newGB10MockSupply(t)
	tokenRouter := setupTokenRouterE2E(t, supply)

	demandSimulatorChat(t, tokenRouter.URL, "e2e-request-1")
	demandSimulatorChat(t, tokenRouter.URL, "e2e-request-2")
	require.Eventually(t, func() bool {
		return len(adminGetUsageLedgers(t, tokenRouter.URL)) == 2
	}, 3*time.Second, 50*time.Millisecond)

	ledgers := adminGetUsageLedgers(t, tokenRouter.URL)
	require.Len(t, ledgers, 2)
	require.Equal(t, e2eSessionID, ledgers[0].SessionId)
	require.Equal(t, e2eSessionID, ledgers[1].SessionId)
	require.Equal(t, ledgers[0].ChannelId, ledgers[1].ChannelId)
	require.Equal(t, "gb10-4t", ledgers[0].SupplyNode)
	require.Greater(t, ledgers[0].SellQuota, ledgers[0].CostQuota)

	var cachedLedger *model.UsageLedger
	for i := range ledgers {
		if ledgers[i].CachedTokens > 0 {
			cachedLedger = &ledgers[i]
			break
		}
	}
	require.NotNil(t, cachedLedger)
	require.True(t, cachedLedger.CacheHit)
	require.Greater(t, cachedLedger.SellQuota, cachedLedger.CostQuota)

	summary := adminGetMarginSummary(t, tokenRouter.URL)
	require.Len(t, summary, 1)
	require.Equal(t, 1, summary[0].SupplierId)
	require.Equal(t, int64(2), summary[0].TotalRequests)
	require.Equal(t, int64(228), summary[0].TotalSellQuota)
	require.Equal(t, int64(112), summary[0].TotalCostQuota)
	require.Equal(t, int64(116), summary[0].GrossProfitQuota)
	require.Equal(t, int64(1), summary[0].CacheHitCount)
	require.Equal(t, 0.5, summary[0].CacheHitRate)

	quality := adminGetQualitySummary(t, tokenRouter.URL, "supplier")
	require.Len(t, quality, 1)
	require.Equal(t, 1, quality[0].SupplierId)
	require.Equal(t, int64(2), quality[0].TotalRequests)
	require.Equal(t, int64(2), quality[0].SuccessRequests)
	require.Equal(t, int64(0), quality[0].ErrorRequests)
	require.Equal(t, 1.0, quality[0].SuccessRate)
	require.Equal(t, int64(1), quality[0].CacheHitCount)
	require.Equal(t, 0.5, quality[0].CacheHitRate)
	require.GreaterOrEqual(t, quality[0].MaxLatencyMs, 0)

	nodeQuality := adminGetQualitySummary(t, tokenRouter.URL, "supply_node")
	require.Len(t, nodeQuality, 1)
	require.Equal(t, "gb10-4t", nodeQuality[0].SupplyNode)
	require.Equal(t, int64(2), nodeQuality[0].TotalRequests)

	refreshedCapacities := adminRefreshSupplyCapacityUsage(t, tokenRouter.URL)
	require.Len(t, refreshedCapacities, 1)
	require.Equal(t, int64(300), refreshedCapacities[0].UsedTokens)
	require.Equal(t, int64(700), refreshedCapacities[0].HeadroomTokens)
	require.InDelta(t, 0.3, refreshedCapacities[0].UtilizationRate, 0.000001)

	capacities := adminGetSupplyCapacities(t, tokenRouter.URL)
	require.Len(t, capacities, 1)
	require.Equal(t, 1, capacities[0].SupplierId)
	require.Equal(t, "gb10-4t", capacities[0].SupplyNode)
	require.Equal(t, e2eModelName, capacities[0].ModelName)
	require.Equal(t, int64(1000), capacities[0].CapacityTokens)
	require.Equal(t, int64(300), capacities[0].UsedTokens)
	require.Equal(t, int64(700), capacities[0].HeadroomTokens)
	require.InDelta(t, 0.3, capacities[0].UtilizationRate, 0.000001)
	require.InDelta(t, 98.5, capacities[0].QualityScore, 0.000001)
	require.InDelta(t, 0.5, capacities[0].UnitCostQuota, 0.000001)

	telemetry := adminRecordSupplyCapacityTelemetry(t, tokenRouter.URL, capacities[0])
	require.Equal(t, capacities[0].Id, telemetry.AppliedCapacityId)
	require.InDelta(t, 0.62, telemetry.GpuUtilizationRate, 0.000001)
	queriedTelemetry := adminGetSupplyCapacityTelemetries(t, tokenRouter.URL, 1, e2eModelName)
	require.Len(t, queriedTelemetry, 1)
	require.Equal(t, telemetry.Id, queriedTelemetry[0].Id)

	capacities = adminGetSupplyCapacities(t, tokenRouter.URL)
	require.Len(t, capacities, 1)
	require.Equal(t, telemetry.Id, capacities[0].LastTelemetryId)
	require.Equal(t, model.SupplyCapacityTelemetrySourceNodeReport, capacities[0].TelemetrySourceType)
	require.Equal(t, "e2e-gb10-4t-capacity-telemetry", capacities[0].TelemetrySourceRef)
	require.Equal(t, telemetry.ObservedAt, capacities[0].TelemetryObservedAt)
	require.InDelta(t, 0.62, capacities[0].GpuUtilizationRate, 0.000001)

	scorecards := adminGenerateSupplierScorecards(t, tokenRouter.URL)
	require.Len(t, scorecards, 1)
	require.Positive(t, scorecards[0].Id)
	require.Equal(t, 1, scorecards[0].SupplierId)
	require.Equal(t, int64(2), scorecards[0].TotalRequests)
	require.Equal(t, int64(2), scorecards[0].SuccessRequests)
	require.Equal(t, int64(0), scorecards[0].ErrorRequests)
	require.Equal(t, int64(1), scorecards[0].CacheHitCount)
	require.Equal(t, int64(228), scorecards[0].TotalSellQuota)
	require.Equal(t, int64(112), scorecards[0].TotalCostQuota)
	require.Equal(t, int64(116), scorecards[0].GrossProfitQuota)
	require.Equal(t, int64(1000), scorecards[0].SupplyCapacityTokens)
	require.Equal(t, int64(300), scorecards[0].SupplyUsedTokens)
	require.Equal(t, int64(700), scorecards[0].SupplyHeadroomTokens)
	require.InDelta(t, 1.0, scorecards[0].SuccessRate, 0.000001)
	require.InDelta(t, 0.5, scorecards[0].CacheHitRate, 0.000001)
	require.InDelta(t, 98.5, scorecards[0].AvgSupplyQualityScore, 0.000001)
	require.InDelta(t, 0.5, scorecards[0].AvgUnitCostQuota, 0.000001)
	require.GreaterOrEqual(t, scorecards[0].Score, 85.0)
	require.LessOrEqual(t, scorecards[0].Score, 100.0)
	require.Equal(t, model.SupplierScorecardGradeA, scorecards[0].Grade)
	queriedScorecards := adminGetSupplierScorecards(t, tokenRouter.URL, model.SupplierScorecardGradeA)
	require.Len(t, queriedScorecards, 1)
	require.Equal(t, scorecards[0].SupplierId, queriedScorecards[0].SupplierId)
	require.Equal(t, scorecards[0].TotalRequests, queriedScorecards[0].TotalRequests)
	require.Equal(t, scorecards[0].Grade, queriedScorecards[0].Grade)

	evaluationContract := adminImportSlaContract(t, tokenRouter.URL, model.SlaContractImportInput{
		ContractKey:            "supplier-evaluation-sla-e2e",
		ModelName:              e2eModelName,
		ProviderFamily:         "kimi",
		SourceName:             "supplier evaluation SLA e2e",
		SourceRef:              "e2e://supplier-evaluation-sla",
		SourceSHA256:           "e2e-supplier-evaluation-sla-sha",
		Version:                "2026-06-23-eval",
		Status:                 model.SlaContractStatusActive,
		MeasurementProfileJSON: `{"input_profile":{"tokens":128},"output_profile":{"target_tokens":16},"cache_profile":"cold_no_cache"}`,
		HardGateJSON:           `{"ttft_ms":{"p90_lte":1000}}`,
		SoftGateJSON:           `{}`,
	})
	evaluationPlan := adminGenerateSlaProbePlan(t, tokenRouter.URL, model.SlaProbePlanGenerateInput{
		ContractId:     evaluationContract.Id,
		SupplierId:     scorecards[0].SupplierId,
		ChannelId:      ledgers[0].ChannelId,
		SlaTier:        "default",
		ProbeType:      model.SlaProbeTypeAdmission,
		RouteMode:      model.SlaProbeRouteModeThroughTokenRouter,
		PromptSuiteKey: "supplier-evaluation-e2e",
		SampleSize:     2,
		RepeatCount:    1,
	})
	evaluationRun := adminRecordSlaProbeRun(t, tokenRouter.URL, model.SlaProbeRunRecordInput{
		RunKey:         "supplier-evaluation-sla-run-e2e",
		PlanId:         evaluationPlan.Id,
		Status:         model.SlaProbeRunStatusPassed,
		StartedAt:      common.GetTimestamp() - 10,
		EndedAt:        common.GetTimestamp(),
		RunnerVersion:  "token-router-sla/e2e",
		RuntimeRef:     "httptest",
		Endpoint:       tokenRouter.URL + "/v1/chat/completions",
		SummaryJSON:    `{"ttft_ms":{"p90":500},"usage":{"prompt_tokens":200}}`,
		HardGatePassed: true,
		ArtifactURI:    "output/sla/supplier-evaluation-sla-run-e2e",
		ArtifactSHA256: "supplier-evaluation-sla-artifact-sha",
	})

	evaluations := adminGenerateSupplierEvaluations(t, tokenRouter.URL, scorecards[0].PeriodStart, scorecards[0].PeriodEnd)
	require.Len(t, evaluations, 1)
	require.Positive(t, evaluations[0].Id)
	require.Equal(t, scorecards[0].Id, evaluations[0].SupplierScorecardId)
	require.Equal(t, scorecards[0].SupplierId, evaluations[0].SupplierId)
	require.Equal(t, evaluationContract.Id, evaluations[0].SlaContractId)
	require.Equal(t, evaluationRun.Id, evaluations[0].SlaProbeRunId)
	require.Contains(t, evaluations[0].SlaGateSummaryJSON, "supplier-evaluation-sla-run-e2e")
	require.Equal(t, model.SupplierEvaluationTypeAdmission, evaluations[0].EvaluationType)
	require.Equal(t, model.SupplierEvaluationStatusDraft, evaluations[0].Status)
	require.Equal(t, model.SupplierEvaluationRecommendationAdmit, evaluations[0].Recommendation)
	require.Equal(t, model.SupplierScorecardGradeA, evaluations[0].Grade)
	require.InDelta(t, scorecards[0].Score, evaluations[0].Score, 0.000001)
	require.Equal(t, scorecards[0].SupplyHeadroomTokens, evaluations[0].SupplyHeadroomTokens)
	queriedEvaluations := adminGetSupplierEvaluations(t, tokenRouter.URL, model.SupplierEvaluationStatusDraft)
	require.Len(t, queriedEvaluations, 1)
	require.Equal(t, evaluations[0].Id, queriedEvaluations[0].Id)
	approvedEvaluation := adminApproveSupplierEvaluation(t, tokenRouter.URL, evaluations[0].Id)
	require.Equal(t, model.SupplierEvaluationStatusApproved, approvedEvaluation.Status)
	require.Equal(t, 1, approvedEvaluation.ReviewedBy)
	require.Greater(t, approvedEvaluation.ReviewedAt, int64(0))
	require.Equal(t, "accepted admission evaluation in e2e", approvedEvaluation.ReviewNote)
	require.Equal(t, common.ChannelStatusEnabled, adminGetSupplier(t, tokenRouter.URL, evaluations[0].SupplierId).Status)
	appliedEvaluation := adminApplySupplierEvaluation(t, tokenRouter.URL, evaluations[0].Id)
	require.Equal(t, model.SupplierEvaluationStatusApproved, appliedEvaluation.Status)
	require.Greater(t, appliedEvaluation.AppliedAt, int64(0))
	require.Equal(t, 1, appliedEvaluation.AppliedBy)
	require.Equal(t, common.ChannelStatusEnabled, appliedEvaluation.SupplierStatusBefore)
	require.Equal(t, common.ChannelStatusEnabled, appliedEvaluation.SupplierStatusAfter)
	require.Contains(t, appliedEvaluation.AppliedNote, "applied admission evaluation in e2e")
	appliedSupplier := adminGetSupplier(t, tokenRouter.URL, evaluations[0].SupplierId)
	require.Equal(t, common.ChannelStatusEnabled, appliedSupplier.Status)
	require.Contains(t, appliedSupplier.Notes, "supplier_evaluation #")

	profiles := adminGenerateTrafficProfiles(t, tokenRouter.URL)
	require.Len(t, profiles, 1)
	require.Equal(t, "gpt-test", profiles[0].ModelName)
	require.Equal(t, "default", profiles[0].SlaTier)
	require.Equal(t, 2, profiles[0].UserId)
	require.Equal(t, int64(2), profiles[0].RequestCount)
	require.Equal(t, int64(2), profiles[0].SuccessRequestCount)
	require.Equal(t, int64(300), profiles[0].DemandTokens)
	require.Equal(t, int64(300), profiles[0].PeakTokens)
	require.Equal(t, int64(1), profiles[0].UniqueSessions)
	require.Equal(t, int64(1), profiles[0].CacheHitCount)
	require.Equal(t, int64(80), profiles[0].TotalCachedTokens)
	require.Equal(t, int64(228), profiles[0].TotalSellQuota)
	require.Equal(t, int64(112), profiles[0].TotalCostQuota)
	require.Equal(t, int64(116), profiles[0].GrossProfitQuota)
	require.Equal(t, int64(1000), profiles[0].SupplyCapacityTokens)
	require.Equal(t, int64(300), profiles[0].SupplyUsedTokens)
	require.Equal(t, int64(700), profiles[0].SupplyHeadroomTokens)
	require.InDelta(t, 1.0, profiles[0].PeakRatio, 0.000001)
	require.InDelta(t, 0.5, profiles[0].CacheHitRate, 0.000001)
	require.InDelta(t, 1.0, profiles[0].SlaMetRate, 0.000001)
	require.InDelta(t, 98.5, profiles[0].AvgSupplyQualityScore, 0.000001)
	require.InDelta(t, 0.5, profiles[0].AvgUnitCostQuota, 0.000001)
	queriedProfiles := adminGetTrafficProfiles(t, tokenRouter.URL)
	require.Len(t, queriedProfiles, 1)
	require.Equal(t, profiles[0].SliceKey, queriedProfiles[0].SliceKey)
	require.Equal(t, profiles[0].DemandTokens, queriedProfiles[0].DemandTokens)
	require.Equal(t, profiles[0].SupplyHeadroomTokens, queriedProfiles[0].SupplyHeadroomTokens)

	forecasts := adminGenerateTrafficForecasts(t, tokenRouter.URL, profiles[0].PeriodStart, profiles[0].PeriodEnd)
	require.Len(t, forecasts, 1)
	require.Positive(t, forecasts[0].Id)
	require.Equal(t, profiles[0].SliceKey, forecasts[0].SliceKey)
	require.Equal(t, "gpt-test", forecasts[0].ModelName)
	require.Equal(t, "default", forecasts[0].SlaTier)
	require.Equal(t, 2, forecasts[0].UserId)
	require.Equal(t, profiles[0].PeriodStart, forecasts[0].SourcePeriodStart)
	require.Equal(t, profiles[0].PeriodEnd, forecasts[0].SourcePeriodEnd)
	require.Equal(t, profiles[0].PeriodEnd, forecasts[0].TargetPeriodStart)
	require.Equal(t, profiles[0].PeriodEnd+(profiles[0].PeriodEnd-profiles[0].PeriodStart), forecasts[0].TargetPeriodEnd)
	require.Equal(t, int64(1), forecasts[0].SourceProfileCount)
	require.Equal(t, int64(2), forecasts[0].ObservedRequestCount)
	require.Equal(t, int64(300), forecasts[0].ObservedDemandTokens)
	require.Equal(t, int64(300), forecasts[0].ObservedPeakTokens)
	require.Equal(t, int64(300), forecasts[0].BaselineDemandTokens)
	require.Equal(t, int64(300), forecasts[0].ForecastDemandTokens)
	require.Equal(t, int64(300), forecasts[0].ForecastPeakTokens)
	require.Equal(t, int64(700), forecasts[0].ForecastHeadroomTokens)
	require.Equal(t, int64(0), forecasts[0].ForecastGapTokens)
	require.Equal(t, int64(0), forecasts[0].TrendDemandDeltaTokens)
	require.InDelta(t, 0, forecasts[0].TrendDemandDeltaRate, 0.000001)
	require.Equal(t, 0, forecasts[0].SeasonalPeriodCount)
	require.InDelta(t, 1, forecasts[0].SeasonalIndex, 0.000001)
	require.Equal(t, int64(300), forecasts[0].SeasonalDemandTokens)
	require.Equal(t, model.TrafficForecastAnomalyNotEvaluated, forecasts[0].AnomalyStatus)
	require.InDelta(t, 0.5, forecasts[0].CacheHitRate, 0.000001)
	require.InDelta(t, 1.0, forecasts[0].SlaMetRate, 0.000001)
	require.Equal(t, int64(116), forecasts[0].GrossProfitQuota)
	require.InDelta(t, 0.5, forecasts[0].AvgUnitCostQuota, 0.000001)
	require.InDelta(t, 1.0/3.0, forecasts[0].Confidence, 0.000001)
	require.Equal(t, model.TrafficForecastMethodWeightedMovingAverage, forecasts[0].Method)
	queriedForecasts := adminGetTrafficForecasts(t, tokenRouter.URL, forecasts[0].TargetPeriodStart, forecasts[0].TargetPeriodEnd)
	require.Len(t, queriedForecasts, 1)
	require.Equal(t, forecasts[0].ForecastKey, queriedForecasts[0].ForecastKey)

	seasonalSourceStart, seasonalSourceEnd, seasonalTargetStart, seasonalTargetEnd := seedE2ESeasonalAnomalyTrafficProfiles(t, profiles[0].PeriodEnd+10_000)
	seasonalForecasts := adminGenerateTrafficForecastsWithInput(t, tokenRouter.URL, model.TrafficForecastGenerateInput{
		PeriodStart:          seasonalSourceStart,
		PeriodEnd:            seasonalSourceEnd,
		TargetPeriodStart:    seasonalTargetStart,
		TargetPeriodEnd:      seasonalTargetEnd,
		ModelName:            e2eModelName,
		SlaTier:              "seasonal",
		UserId:               99,
		SeasonalPeriodCount:  2,
		AnomalyGuard:         true,
		AnomalyThresholdRate: 1.8,
	})
	require.Len(t, seasonalForecasts, 1)
	require.Equal(t, model.TrafficForecastMethodSeasonalAnomaly, seasonalForecasts[0].Method)
	require.Equal(t, int64(250), seasonalForecasts[0].BaselineDemandTokens)
	require.Equal(t, int64(150), seasonalForecasts[0].ForecastDemandTokens)
	require.Equal(t, int64(390), seasonalForecasts[0].ForecastPeakTokens)
	require.Equal(t, int64(250), seasonalForecasts[0].ForecastHeadroomTokens)
	require.Equal(t, int64(140), seasonalForecasts[0].ForecastGapTokens)
	require.Equal(t, 2, seasonalForecasts[0].SeasonalPeriodCount)
	require.InDelta(t, 0.5, seasonalForecasts[0].SeasonalIndex, 0.000001)
	require.Equal(t, int64(125), seasonalForecasts[0].SeasonalDemandTokens)
	require.Equal(t, model.TrafficForecastAnomalySpike, seasonalForecasts[0].AnomalyStatus)
	require.Greater(t, seasonalForecasts[0].AnomalyProfileId, 0)
	require.InDelta(t, 360.0/(520.0/3.0), seasonalForecasts[0].AnomalyDemandRatio, 0.000001)
	require.Contains(t, seasonalForecasts[0].Reason, "seasonal/anomaly adjusted")

	pricingRecommendations := adminGeneratePricingRecommendations(t, tokenRouter.URL, profiles[0].PeriodStart, profiles[0].PeriodEnd)
	require.Len(t, pricingRecommendations, 1)
	require.Positive(t, pricingRecommendations[0].Id)
	require.Equal(t, profiles[0].Id, pricingRecommendations[0].TrafficProfileId)
	require.Equal(t, profiles[0].SliceKey, pricingRecommendations[0].SliceKey)
	require.Equal(t, "gpt-test", pricingRecommendations[0].ModelName)
	require.Equal(t, "default", pricingRecommendations[0].SlaTier)
	require.Equal(t, 2, pricingRecommendations[0].UserId)
	require.Equal(t, model.PricingRecommendationActionShareSavings, pricingRecommendations[0].Action)
	require.Equal(t, model.PricingRecommendationStatusDraft, pricingRecommendations[0].Status)
	require.Equal(t, int64(300), pricingRecommendations[0].DemandTokens)
	require.Equal(t, int64(700), pricingRecommendations[0].SupplyHeadroomTokens)
	require.Equal(t, int64(116), pricingRecommendations[0].GrossProfitQuota)
	require.InDelta(t, 0.76, pricingRecommendations[0].CurrentUnitPriceQuota, 0.000001)
	require.InDelta(t, 112.0/300.0, pricingRecommendations[0].CurrentUnitCostQuota, 0.000001)
	require.InDelta(t, 116.0/228.0, pricingRecommendations[0].CurrentMarginRate, 0.000001)
	require.InDelta(t, 0.684, pricingRecommendations[0].RecommendedUnitPriceQuota, 0.000001)
	require.Contains(t, pricingRecommendations[0].Reason, "share efficiency savings")
	queriedPricingRecommendations := adminGetPricingRecommendations(t, tokenRouter.URL, model.PricingRecommendationStatusDraft)
	require.Len(t, queriedPricingRecommendations, 1)
	require.Equal(t, pricingRecommendations[0].RecommendationKey, queriedPricingRecommendations[0].RecommendationKey)
	approvedPricingRecommendation := adminApprovePricingRecommendation(t, tokenRouter.URL, pricingRecommendations[0].Id)
	require.Equal(t, model.PricingRecommendationStatusApproved, approvedPricingRecommendation.Status)
	require.Equal(t, 1, approvedPricingRecommendation.ReviewedBy)
	require.Greater(t, approvedPricingRecommendation.ReviewedAt, int64(0))
	require.Equal(t, "accepted pricing recommendation in e2e", approvedPricingRecommendation.ReviewNote)
	require.Len(t, adminGetPricingRecommendations(t, tokenRouter.URL, model.PricingRecommendationStatusApproved), 1)
	regeneratedPricingRecommendations := adminGeneratePricingRecommendations(t, tokenRouter.URL, profiles[0].PeriodStart, profiles[0].PeriodEnd)
	require.Len(t, regeneratedPricingRecommendations, 1)
	require.Equal(t, model.PricingRecommendationStatusApproved, regeneratedPricingRecommendations[0].Status)
	require.Equal(t, approvedPricingRecommendation.ReviewNote, regeneratedPricingRecommendations[0].ReviewNote)

	decisions := adminGenerateSupplyDecisions(t, tokenRouter.URL, profiles[0].PeriodStart, profiles[0].PeriodEnd)
	require.Len(t, decisions, 1)
	require.Positive(t, decisions[0].Id)
	require.Equal(t, profiles[0].Id, decisions[0].TrafficProfileId)
	require.Equal(t, forecasts[0].Id, decisions[0].TrafficForecastId)
	require.Equal(t, model.SupplyDecisionSourceForecast, decisions[0].DecisionSource)
	require.Equal(t, profiles[0].SliceKey, decisions[0].SliceKey)
	require.Equal(t, "gpt-test", decisions[0].ModelName)
	require.Equal(t, "default", decisions[0].SlaTier)
	require.Equal(t, 2, decisions[0].UserId)
	require.Equal(t, forecasts[0].TargetPeriodStart, decisions[0].ForecastTargetStart)
	require.Equal(t, forecasts[0].TargetPeriodEnd, decisions[0].ForecastTargetEnd)
	require.InDelta(t, forecasts[0].Confidence, decisions[0].ForecastConfidence, 0.000001)
	require.Equal(t, forecasts[0].Method, decisions[0].ForecastMethod)
	require.Equal(t, model.SupplyDecisionTypeSelfHostedEvaluate, decisions[0].DecisionType)
	require.Equal(t, model.SupplyDecisionTrackSelfHosted, decisions[0].Track)
	require.Equal(t, model.SupplyDecisionStatusDraft, decisions[0].Status)
	require.Equal(t, forecasts[0].ForecastDemandTokens, decisions[0].DemandTokens)
	require.Equal(t, forecasts[0].ForecastPeakTokens, decisions[0].PeakTokens)
	require.Equal(t, forecasts[0].ForecastHeadroomTokens, decisions[0].SupplyHeadroomTokens)
	require.Equal(t, forecasts[0].ForecastGapTokens, decisions[0].GapTokens)
	require.Equal(t, forecasts[0].ForecastDemandTokens, decisions[0].RecommendedCapacity)
	require.Equal(t, int64(116), decisions[0].GrossProfitQuota)
	require.InDelta(t, 0.5, decisions[0].CacheHitRate, 0.000001)
	require.InDelta(t, 1.0, decisions[0].SlaMetRate, 0.000001)
	require.InDelta(t, 98.5, decisions[0].AvgSupplyQualityScore, 0.000001)
	require.InDelta(t, 0.5, decisions[0].AvgUnitCostQuota, 0.000001)
	require.InDelta(t, 191.0, decisions[0].RoiScore, 0.000001)
	require.Contains(t, decisions[0].Reason, "forecast-informed")
	queriedDecisions := adminGetSupplyDecisions(t, tokenRouter.URL, model.SupplyDecisionStatusDraft)
	require.Len(t, queriedDecisions, 1)
	require.Equal(t, decisions[0].DecisionKey, queriedDecisions[0].DecisionKey)
	approvedDecision := adminApproveSupplyDecision(t, tokenRouter.URL, decisions[0].Id)
	require.Equal(t, model.SupplyDecisionStatusApproved, approvedDecision.Status)
	require.Equal(t, 1, approvedDecision.ReviewedBy)
	require.Greater(t, approvedDecision.ReviewedAt, int64(0))
	require.Equal(t, "accepted in e2e", approvedDecision.ReviewNote)
	require.Len(t, adminGetSupplyDecisions(t, tokenRouter.URL, model.SupplyDecisionStatusApproved), 1)

	costProfile := adminRecordSupplyCostProfile(t, tokenRouter.URL, 2, profiles[0].PeriodStart, profiles[0].PeriodEnd)
	require.Positive(t, costProfile.Id)
	require.InDelta(t, 0.12, costProfile.AmortizedUnitCostQuota, 0.000001)
	queriedCostProfiles := adminGetSupplyCostProfiles(t, tokenRouter.URL, 2, e2eModelName)
	require.Len(t, queriedCostProfiles, 1)
	require.Equal(t, costProfile.Id, queriedCostProfiles[0].Id)
	require.Equal(t, "e2e-gb10-4t-self-hosted-cost", queriedCostProfiles[0].SourceRef)

	prepaidModelName := "gpt-prepaid-e2e"
	adminRejectSupplyPrepaidLot(t, tokenRouter.URL, 1, "supplier must be self_operated")
	prepaidLot := adminRecordSupplyPrepaidLot(t, tokenRouter.URL, 3, prepaidModelName, profiles[0].PeriodStart, profiles[0].PeriodEnd)
	require.Positive(t, prepaidLot.Id)
	require.Equal(t, 3, prepaidLot.SupplierId)
	require.Equal(t, "gb10-4t-self-operated", prepaidLot.SupplyNode)
	require.Equal(t, prepaidModelName, prepaidLot.ModelName)
	require.Equal(t, int64(1000), prepaidLot.PurchasedTokens)
	require.InDelta(t, 0.42, prepaidLot.UnitCostQuota, 0.000001)
	require.InDelta(t, 420.0, prepaidLot.TotalCostQuota, 0.000001)
	require.Equal(t, int64(0), prepaidLot.DrawdownTokens)
	require.Equal(t, int64(1000), prepaidLot.RemainingTokens)
	require.Equal(t, model.SupplyPrepaidLotSourceAccounting, prepaidLot.SourceType)
	require.Equal(t, "e2e-gb10-4t-self-operated-prepaid", prepaidLot.SourceRef)
	require.Equal(t, "po://e2e-gb10-4t-self-operated", prepaidLot.ExternalRef)
	queriedPrepaidLots := adminGetSupplyPrepaidLots(t, tokenRouter.URL, 3, prepaidModelName)
	require.Len(t, queriedPrepaidLots, 1)
	require.Equal(t, prepaidLot.Id, queriedPrepaidLots[0].Id)
	seedE2EPrepaidUsageLedgers(t, 3, prepaidModelName, profiles[0].PeriodStart)
	refreshedPrepaidLot := adminRefreshSupplyPrepaidLotUsage(t, tokenRouter.URL, prepaidLot.Id)
	require.Equal(t, prepaidLot.Id, refreshedPrepaidLot.Id)
	require.Equal(t, int64(320), refreshedPrepaidLot.DrawdownTokens)
	require.Equal(t, int64(2), refreshedPrepaidLot.DrawdownRequestCount)
	require.Equal(t, int64(680), refreshedPrepaidLot.RemainingTokens)
	require.InDelta(t, 0.32, refreshedPrepaidLot.DrawdownRate, 0.000001)
	require.Equal(t, model.SupplyPrepaidLotDrawdownSourceUsageLedger, refreshedPrepaidLot.DrawdownSourceType)
	require.Contains(t, refreshedPrepaidLot.DrawdownSourceRef, "usage_ledger:prepaid_lot:")
	require.Greater(t, refreshedPrepaidLot.DrawdownRefreshedAt, int64(0))

	opportunities := adminGenerateSupplyExpansionOpportunities(t, tokenRouter.URL, profiles[0].PeriodStart, profiles[0].PeriodEnd)
	require.Len(t, opportunities, 1)
	require.Positive(t, opportunities[0].Id)
	require.Equal(t, approvedDecision.Id, opportunities[0].SupplyDecisionId)
	require.Equal(t, profiles[0].Id, opportunities[0].TrafficProfileId)
	require.Equal(t, forecasts[0].Id, opportunities[0].TrafficForecastId)
	require.Equal(t, model.SupplyDecisionSourceForecast, opportunities[0].DecisionSource)
	require.Equal(t, model.SupplyDecisionStatusApproved, opportunities[0].DecisionStatus)
	require.Equal(t, model.SupplyExpansionOpportunityTypeSelfHosted, opportunities[0].OpportunityType)
	require.Equal(t, model.SupplyExpansionOpportunityPriorityAction, opportunities[0].Priority)
	require.Equal(t, model.SupplyExpansionOpportunityClusterHighCacheStable, opportunities[0].ClusterKey)
	require.Equal(t, model.SupplyDecisionTrackSelfHosted, opportunities[0].Track)
	require.Equal(t, model.SupplyDecisionTypeSelfHostedEvaluate, opportunities[0].DecisionType)
	require.Equal(t, forecasts[0].TargetPeriodStart, opportunities[0].ForecastTargetStart)
	require.Equal(t, forecasts[0].TargetPeriodEnd, opportunities[0].ForecastTargetEnd)
	require.InDelta(t, forecasts[0].Confidence, opportunities[0].ForecastConfidence, 0.000001)
	require.Equal(t, forecasts[0].Method, opportunities[0].ForecastMethod)
	require.Equal(t, forecasts[0].ForecastDemandTokens, opportunities[0].DemandTokens)
	require.Equal(t, forecasts[0].ForecastPeakTokens, opportunities[0].PeakTokens)
	require.Equal(t, forecasts[0].ForecastHeadroomTokens, opportunities[0].SupplyHeadroomTokens)
	require.Equal(t, forecasts[0].ForecastGapTokens, opportunities[0].GapTokens)
	require.Equal(t, forecasts[0].ForecastDemandTokens, opportunities[0].RecommendedCapacity)
	require.InDelta(t, 0.5, opportunities[0].LocalityScore, 0.000001)
	require.InDelta(t, 1.0, opportunities[0].StabilityScore, 0.000001)
	require.InDelta(t, 0.0, opportunities[0].HeadroomRiskScore, 0.000001)
	require.Equal(t, costProfile.Id, opportunities[0].SelfHostedCostProfileId)
	require.InDelta(t, 0.12, opportunities[0].SelfHostedUnitCostQuota, 0.000001)
	require.InDelta(t, 0.38, opportunities[0].SelfHostedSavingsUnitQuota, 0.000001)
	require.InDelta(t, 114.0, opportunities[0].SelfHostedSavingsQuota, 0.000001)
	require.InDelta(t, 405.0, opportunities[0].RankScore, 0.000001)
	require.Contains(t, opportunities[0].Reason, "self-hosted expansion candidate")
	require.Contains(t, opportunities[0].Reason, "e2e-gb10-4t-self-hosted-cost")
	queriedOpportunities := adminGetSupplyExpansionOpportunities(t, tokenRouter.URL, model.SupplyExpansionOpportunityTypeSelfHosted)
	require.Len(t, queriedOpportunities, 1)
	require.Equal(t, opportunities[0].OpportunityKey, queriedOpportunities[0].OpportunityKey)
	require.Equal(t, costProfile.Id, queriedOpportunities[0].SelfHostedCostProfileId)

	insights := adminGenerateOperatingInsights(t, tokenRouter.URL, profiles[0].PeriodStart, profiles[0].PeriodEnd)
	require.Len(t, insights, 1)
	require.Positive(t, insights[0].Id)
	require.Equal(t, profiles[0].Id, insights[0].TrafficProfileId)
	require.Equal(t, approvedDecision.Id, insights[0].SupplyDecisionId)
	require.Equal(t, regeneratedPricingRecommendations[0].Id, insights[0].PricingRecommendationId)
	require.Equal(t, profiles[0].SliceKey, insights[0].SliceKey)
	require.Equal(t, model.OperatingInsightCategoryCacheEfficiency, insights[0].Category)
	require.Equal(t, model.OperatingInsightSeverityAction, insights[0].Severity)
	require.Equal(t, model.OperatingInsightStatusDraft, insights[0].Status)
	require.Equal(t, model.SupplyDecisionTrackSelfHosted, insights[0].SupplyDecisionTrack)
	require.Equal(t, model.SupplyDecisionStatusApproved, insights[0].SupplyDecisionStatus)
	require.Equal(t, model.PricingRecommendationActionShareSavings, insights[0].PricingRecommendationAction)
	require.Equal(t, model.PricingRecommendationStatusApproved, insights[0].PricingRecommendationStatus)
	require.Equal(t, int64(300), insights[0].DemandTokens)
	require.Equal(t, int64(700), insights[0].SupplyHeadroomTokens)
	require.InDelta(t, 0.5, insights[0].CacheHitRate, 0.000001)
	require.InDelta(t, 191.0, insights[0].SupplyDecisionRoiScore, 0.000001)
	require.Contains(t, insights[0].RecommendedAction, "self-hosted")
	queriedInsights := adminGetOperatingInsights(t, tokenRouter.URL, model.OperatingInsightStatusDraft, model.OperatingInsightCategoryCacheEfficiency)
	require.Len(t, queriedInsights, 1)
	require.Equal(t, insights[0].InsightKey, queriedInsights[0].InsightKey)
	acknowledgedInsight := adminAcknowledgeOperatingInsight(t, tokenRouter.URL, insights[0].Id)
	require.Equal(t, model.OperatingInsightStatusAcknowledged, acknowledgedInsight.Status)
	require.Equal(t, 1, acknowledgedInsight.ReviewedBy)
	require.Greater(t, acknowledgedInsight.ReviewedAt, int64(0))
	require.Equal(t, "acknowledged operating insight in e2e", acknowledgedInsight.ReviewNote)
	require.Len(t, adminGetOperatingInsights(t, tokenRouter.URL, model.OperatingInsightStatusAcknowledged, model.OperatingInsightCategoryCacheEfficiency), 1)
	regeneratedInsights := adminGenerateOperatingInsights(t, tokenRouter.URL, profiles[0].PeriodStart, profiles[0].PeriodEnd)
	require.Len(t, regeneratedInsights, 1)
	require.Equal(t, model.OperatingInsightStatusAcknowledged, regeneratedInsights[0].Status)
	require.Equal(t, acknowledgedInsight.ReviewNote, regeneratedInsights[0].ReviewNote)

	hotTelemetry := adminRecordSupplyCapacityTelemetryInput(t, tokenRouter.URL, model.SupplyCapacityTelemetryRecordInput{
		SupplierId:         1,
		SupplyNode:         "gb10-hot",
		ModelName:          e2eModelName,
		PeriodStart:        profiles[0].PeriodStart,
		PeriodEnd:          profiles[0].PeriodEnd,
		CapacityTokens:     1000,
		UsedTokens:         950,
		GpuUtilizationRate: 0.94,
		QualityScore:       97.5,
		UnitCostQuota:      0.52,
		SourceType:         model.SupplyCapacityTelemetrySourceNodeReport,
		SourceRef:          "e2e-gb10-hot-capacity-telemetry",
		ObservedAt:         common.GetTimestamp(),
		Notes:              "e2e hot node telemetry for operating insight",
	})
	require.Positive(t, hotTelemetry.AppliedCapacityId)
	generatedWithCapacityRisk := adminGenerateOperatingInsights(t, tokenRouter.URL, profiles[0].PeriodStart, profiles[0].PeriodEnd)
	require.GreaterOrEqual(t, len(generatedWithCapacityRisk), 2)
	capacityRiskInsights := adminGetGlobalOperatingInsights(t, tokenRouter.URL, model.OperatingInsightStatusDraft, model.OperatingInsightCategoryCapacityRisk, profiles[0].PeriodStart, profiles[0].PeriodEnd)
	require.Len(t, capacityRiskInsights, 1)
	require.Equal(t, model.OperatingInsightSeverityAction, capacityRiskInsights[0].Severity)
	require.Equal(t, "gpt-test", capacityRiskInsights[0].ModelName)
	require.Equal(t, "default", capacityRiskInsights[0].SlaTier)
	require.Zero(t, capacityRiskInsights[0].UserId)
	require.Contains(t, capacityRiskInsights[0].SliceKey, "gb10-hot")
	require.Contains(t, capacityRiskInsights[0].InsightKey, "reason:high_gpu")
	require.Equal(t, int64(50), capacityRiskInsights[0].SupplyHeadroomTokens)
	require.Contains(t, capacityRiskInsights[0].Summary, "GPU utilization")

	actionPlans := adminGenerateSupplyActionPlans(t, tokenRouter.URL, approvedDecision.Id)
	require.Len(t, actionPlans, 1)
	require.Positive(t, actionPlans[0].Id)
	require.Equal(t, approvedDecision.Id, actionPlans[0].SupplyDecisionId)
	require.Equal(t, approvedDecision.DecisionKey, actionPlans[0].DecisionKey)
	require.Equal(t, opportunities[0].Id, actionPlans[0].SupplyExpansionOpportunityId)
	require.Equal(t, opportunities[0].OpportunityKey, actionPlans[0].OpportunityKey)
	require.Equal(t, opportunities[0].OpportunityType, actionPlans[0].OpportunityType)
	require.Equal(t, opportunities[0].Priority, actionPlans[0].OpportunityPriority)
	require.Equal(t, opportunities[0].ClusterKey, actionPlans[0].OpportunityClusterKey)
	require.InDelta(t, opportunities[0].RankScore, actionPlans[0].OpportunityRankScore, 0.000001)
	require.Equal(t, model.SupplyActionTypeEvaluateSelfHostedCapacity, actionPlans[0].ActionType)
	require.Equal(t, model.SupplyActionPlanStatusPlanned, actionPlans[0].Status)
	require.Equal(t, model.SupplyDecisionTrackSelfHosted, actionPlans[0].Track)
	require.Equal(t, int64(300), actionPlans[0].RecommendedCapacity)
	require.Equal(t, int64(0), actionPlans[0].GapTokens)
	require.InDelta(t, 191.0, actionPlans[0].RoiScore, 0.000001)
	require.Equal(t, approvedDecision.ReviewedAt, actionPlans[0].SourceReviewedAt)
	require.Equal(t, approvedDecision.ReviewedBy, actionPlans[0].SourceReviewedBy)
	queriedActionPlans := adminGetSupplyActionPlans(t, tokenRouter.URL, approvedDecision.Id)
	require.Len(t, queriedActionPlans, 1)
	require.Equal(t, actionPlans[0].SupplyDecisionId, queriedActionPlans[0].SupplyDecisionId)
	require.Equal(t, actionPlans[0].ActionType, queriedActionPlans[0].ActionType)
	require.Equal(t, actionPlans[0].SupplyExpansionOpportunityId, queriedActionPlans[0].SupplyExpansionOpportunityId)
	require.Equal(t, actionPlans[0].OpportunityKey, queriedActionPlans[0].OpportunityKey)
	require.Equal(t, actionPlans[0].Status, queriedActionPlans[0].Status)
	adminRejectSupplyActionExecution(t, tokenRouter.URL, model.SupplyActionExecutionRecordInput{
		SupplyActionPlanId:   actionPlans[0].Id,
		ExecutionStatus:      model.SupplyActionExecutionStatusRecorded,
		SupplierId:           1,
		ActualCapacityTokens: 300,
		UnitCostQuota:        0.5,
		ExternalRef:          "e2e-before-complete",
		OperatorNote:         "should be rejected before completion",
	}, "must be completed")
	inProgressPlan := adminUpdateSupplyActionPlanStatus(t, tokenRouter.URL, actionPlans[0].Id, model.SupplyActionPlanStatusInProgress, "operator started capacity evaluation")
	require.Equal(t, model.SupplyActionPlanStatusInProgress, inProgressPlan.Status)
	require.Equal(t, "operator started capacity evaluation", inProgressPlan.OperatorNote)
	require.Equal(t, 1, inProgressPlan.StatusUpdatedBy)
	require.Greater(t, inProgressPlan.StatusUpdatedAt, int64(0))
	require.Greater(t, inProgressPlan.StartedAt, int64(0))
	require.Equal(t, int64(0), inProgressPlan.CompletedAt)
	completedPlan := adminUpdateSupplyActionPlanStatus(t, tokenRouter.URL, actionPlans[0].Id, model.SupplyActionPlanStatusCompleted, "capacity evaluation completed offline")
	require.Equal(t, model.SupplyActionPlanStatusCompleted, completedPlan.Status)
	require.Equal(t, "capacity evaluation completed offline", completedPlan.OperatorNote)
	require.Equal(t, 1, completedPlan.StatusUpdatedBy)
	require.GreaterOrEqual(t, completedPlan.StatusUpdatedAt, inProgressPlan.StatusUpdatedAt)
	require.Equal(t, inProgressPlan.StartedAt, completedPlan.StartedAt)
	require.Greater(t, completedPlan.CompletedAt, int64(0))
	regeneratedPlans := adminGenerateSupplyActionPlans(t, tokenRouter.URL, approvedDecision.Id)
	require.Len(t, regeneratedPlans, 1)
	require.Equal(t, model.SupplyActionPlanStatusCompleted, regeneratedPlans[0].Status)
	require.Equal(t, completedPlan.CompletedAt, regeneratedPlans[0].CompletedAt)
	adminRejectSupplyActionPlanStatus(t, tokenRouter.URL, actionPlans[0].Id, model.SupplyActionPlanStatusInProgress)

	now := common.GetTimestamp()
	createdCapacity := adminCreateSupplyCapacity(t, tokenRouter.URL, model.SupplyCapacity{
		SupplierId:     1,
		SupplyNode:     "gb10-4t-burst",
		ModelName:      e2eModelName,
		PeriodStart:    now - 3600,
		PeriodEnd:      now + 3600,
		CapacityTokens: 2000,
		UsedTokens:     500,
		QualityScore:   95,
		UnitCostQuota:  0.4,
		Status:         1,
	})
	require.Positive(t, createdCapacity.Id)
	require.Equal(t, int64(1500), createdCapacity.HeadroomTokens)
	require.InDelta(t, 0.25, createdCapacity.UtilizationRate, 0.000001)

	createdCapacity.UsedTokens = 1000
	updatedCapacity := adminUpdateSupplyCapacity(t, tokenRouter.URL, createdCapacity)
	require.Equal(t, int64(1000), updatedCapacity.HeadroomTokens)
	require.InDelta(t, 0.5, updatedCapacity.UtilizationRate, 0.000001)

	execution := adminRecordSupplyActionExecution(t, tokenRouter.URL, model.SupplyActionExecutionRecordInput{
		SupplyActionPlanId:   completedPlan.Id,
		ExecutionStatus:      model.SupplyActionExecutionStatusRecorded,
		SupplierId:           updatedCapacity.SupplierId,
		SupplyCapacityId:     updatedCapacity.Id,
		ActualCapacityTokens: updatedCapacity.CapacityTokens,
		UnitCostQuota:        updatedCapacity.UnitCostQuota,
		EffectiveFrom:        updatedCapacity.PeriodStart,
		EffectiveTo:          updatedCapacity.PeriodEnd,
		ExternalRef:          "e2e-self-hosted-evaluation",
		OperatorNote:         "self-hosted evaluation result recorded",
	})
	require.Positive(t, execution.Id)
	require.Equal(t, completedPlan.Id, execution.SupplyActionPlanId)
	require.Equal(t, completedPlan.SupplyDecisionId, execution.SupplyDecisionId)
	require.Equal(t, completedPlan.DecisionKey, execution.DecisionKey)
	require.Equal(t, completedPlan.ActionType, execution.ActionType)
	require.Equal(t, completedPlan.Track, execution.Track)
	require.Equal(t, model.SupplyActionExecutionStatusRecorded, execution.ExecutionStatus)
	require.Equal(t, updatedCapacity.SupplierId, execution.SupplierId)
	require.Equal(t, updatedCapacity.Id, execution.SupplyCapacityId)
	require.Equal(t, updatedCapacity.CapacityTokens, execution.ActualCapacityTokens)
	require.Equal(t, updatedCapacity.UnitCostQuota, execution.UnitCostQuota)
	require.Equal(t, completedPlan.CompletedAt, execution.ActionPlanCompletedAt)
	require.Equal(t, completedPlan.StatusUpdatedBy, execution.ActionPlanCompletedBy)
	require.Equal(t, 1, execution.RecordedBy)
	require.Greater(t, execution.RecordedAt, int64(0))
	require.Equal(t, "e2e-self-hosted-evaluation", execution.ExternalRef)
	require.Equal(t, "self-hosted evaluation result recorded", execution.OperatorNote)
	updatedExecution := adminRecordSupplyActionExecution(t, tokenRouter.URL, model.SupplyActionExecutionRecordInput{
		SupplyActionPlanId:   completedPlan.Id,
		ExecutionStatus:      model.SupplyActionExecutionStatusRecorded,
		SupplierId:           updatedCapacity.SupplierId,
		SupplyCapacityId:     updatedCapacity.Id,
		ActualCapacityTokens: 2500,
		UnitCostQuota:        0.35,
		EffectiveFrom:        updatedCapacity.PeriodStart,
		EffectiveTo:          updatedCapacity.PeriodEnd,
		ExternalRef:          "e2e-self-hosted-evaluation-updated",
		OperatorNote:         "updated execution record without duplicate",
	})
	require.Equal(t, execution.Id, updatedExecution.Id)
	require.Equal(t, int64(2500), updatedExecution.ActualCapacityTokens)
	require.InDelta(t, 0.35, updatedExecution.UnitCostQuota, 0.000001)
	require.Equal(t, "updated execution record without duplicate", updatedExecution.OperatorNote)
	queriedExecutions := adminGetSupplyActionExecutions(t, tokenRouter.URL, completedPlan.Id)
	require.Len(t, queriedExecutions, 1)
	require.Equal(t, updatedExecution.Id, queriedExecutions[0].Id)
	require.Equal(t, "e2e-self-hosted-evaluation-updated", queriedExecutions[0].ExternalRef)

	adminDeleteSupplyCapacity(t, tokenRouter.URL, updatedCapacity.Id)
	require.Len(t, adminGetSupplyCapacitiesByNode(t, tokenRouter.URL, "gb10-4t-burst"), 0)

	statement := adminGenerateSupplierStatement(t, tokenRouter.URL)
	require.Positive(t, statement.Id)
	require.Equal(t, model.SettlementSubjectSupplier, statement.SubjectType)
	require.Equal(t, int64(2), statement.TotalRequests)
	require.Equal(t, int64(228), statement.TotalSellQuota)
	require.Equal(t, int64(112), statement.TotalCostQuota)
	require.Equal(t, int64(116), statement.GrossProfitQuota)
	require.Equal(t, 0.5, statement.CacheHitRate)

	items := adminGetSettlementItems(t, tokenRouter.URL, statement.Id)
	require.Len(t, items, 2)
	csvBody := adminGetSettlementCSV(t, tokenRouter.URL, statement.Id)
	require.True(t, strings.Contains(csvBody, "request_id,session_id,supplier_id"))
	require.True(t, strings.Contains(csvBody, "e2e-request-1"))
	require.True(t, strings.Contains(csvBody, "e2e-request-2"))

	demandSimulatorChat(t, tokenRouter.URL, "e2e-request-2")
	requireLedgerCountStable(t, tokenRouter.URL, 2, 750*time.Millisecond)

	adminRejectSupplyRoutingPolicy(t, tokenRouter.URL, updatedExecution.Id, "channel_id is required")
	selfHostedCapacity := adminCreateSupplyCapacity(t, tokenRouter.URL, model.SupplyCapacity{
		SupplierId:     2,
		SupplyNode:     "gb10-4t-self-hosted",
		ModelName:      e2eModelName,
		PeriodStart:    now - 3600,
		PeriodEnd:      now + 3600,
		CapacityTokens: 3000,
		UsedTokens:     200,
		QualityScore:   99,
		UnitCostQuota:  0.35,
		Status:         1,
		Notes:          "self-hosted routing capacity",
	})
	selfHostedExecution := adminRecordSupplyActionExecution(t, tokenRouter.URL, model.SupplyActionExecutionRecordInput{
		SupplyActionPlanId:   completedPlan.Id,
		ExecutionStatus:      model.SupplyActionExecutionStatusRecorded,
		SupplierId:           2,
		ChannelId:            3,
		SupplyCapacityId:     selfHostedCapacity.Id,
		ActualCapacityTokens: selfHostedCapacity.CapacityTokens,
		UnitCostQuota:        selfHostedCapacity.UnitCostQuota,
		EffectiveFrom:        selfHostedCapacity.PeriodStart,
		EffectiveTo:          selfHostedCapacity.PeriodEnd,
		ExternalRef:          "e2e-self-hosted-routing-ready",
		OperatorNote:         "self-hosted routing execution ready",
	})
	require.Equal(t, updatedExecution.Id, selfHostedExecution.Id)
	require.Equal(t, model.SupplyDecisionTrackSelfHosted, selfHostedExecution.Track)
	require.Equal(t, 2, selfHostedExecution.SupplierId)
	require.Equal(t, 3, selfHostedExecution.ChannelId)
	require.Equal(t, selfHostedCapacity.Id, selfHostedExecution.SupplyCapacityId)

	adminRejectSupplyRoutingPolicy(t, tokenRouter.URL, selfHostedExecution.Id, "passed runtime SLA probe run is required")
	routingSlaPlan := adminGenerateSlaProbePlan(t, tokenRouter.URL, model.SlaProbePlanGenerateInput{
		ContractId:     evaluationContract.Id,
		SupplierId:     selfHostedExecution.SupplierId,
		ChannelId:      selfHostedExecution.ChannelId,
		SlaTier:        selfHostedExecution.SlaTier,
		ProbeType:      model.SlaProbeTypeRuntimeLight,
		RouteMode:      model.SlaProbeRouteModeDirectUpstream,
		PromptSuiteKey: "self-hosted-routing-e2e",
		SampleSize:     1,
		RepeatCount:    1,
		MaxProbeQuota:  1000,
	})
	routingSlaRun := adminRecordSlaProbeRun(t, tokenRouter.URL, model.SlaProbeRunRecordInput{
		RunKey:         "self-hosted-routing-sla-run-e2e",
		PlanId:         routingSlaPlan.Id,
		Status:         model.SlaProbeRunStatusPassed,
		StartedAt:      common.GetTimestamp() - 10,
		EndedAt:        common.GetTimestamp(),
		RunnerVersion:  "token-router-sla/e2e",
		RuntimeRef:     "aima2/self-hosted-routing",
		Endpoint:       tokenRouter.URL + "/v1/chat/completions",
		SummaryJSON:    `{"ttft_ms":{"p90":500},"usage":{"streaming":true}}`,
		HardGatePassed: true,
		ArtifactURI:    "output/sla/self-hosted-routing-sla-run-e2e",
		ArtifactSHA256: "self-hosted-routing-sla-artifact-sha",
	})

	policy := adminActivateSupplyRoutingPolicy(t, tokenRouter.URL, selfHostedExecution.Id)
	require.Positive(t, policy.Id)
	require.Equal(t, selfHostedExecution.Id, policy.SupplyActionExecutionId)
	require.Equal(t, model.SupplyRoutingPolicyStatusActive, policy.Status)
	require.Equal(t, 2, policy.SupplierId)
	require.Equal(t, 3, policy.ChannelId)
	require.Equal(t, selfHostedCapacity.Id, policy.SupplyCapacityId)
	require.Equal(t, evaluationContract.Id, policy.SlaContractId)
	require.Equal(t, routingSlaRun.Id, policy.SlaProbeRunId)
	require.Equal(t, routingSlaRun.RunKey, policy.SlaProbeRunKey)
	require.Equal(t, "self-hosted-routing-sla-artifact-sha", policy.SlaArtifactSHA256)
	require.Equal(t, "aima2/self-hosted-routing", policy.SlaRuntimeRef)
	require.Equal(t, e2eModelName, policy.ModelName)
	require.Equal(t, "default", policy.SlaTier)
	require.Equal(t, 2, policy.UserId)
	require.Equal(t, 1, policy.ActivatedBy)
	require.Greater(t, policy.ActivatedAt, int64(0))
	updatedPolicy := adminActivateSupplyRoutingPolicy(t, tokenRouter.URL, selfHostedExecution.Id)
	require.Equal(t, policy.Id, updatedPolicy.Id)
	require.Equal(t, model.SupplyRoutingPolicyStatusActive, updatedPolicy.Status)
	require.Equal(t, routingSlaRun.Id, updatedPolicy.SlaProbeRunId)
	queriedPolicies := adminGetSupplyRoutingPolicies(t, tokenRouter.URL, selfHostedExecution.Id)
	require.Len(t, queriedPolicies, 1)
	require.Equal(t, policy.Id, queriedPolicies[0].Id)
	require.Equal(t, routingSlaRun.Id, queriedPolicies[0].SlaProbeRunId)

	demandSimulatorChat(t, tokenRouter.URL, "e2e-request-3")
	require.Eventually(t, func() bool {
		ledgers := adminGetUsageLedgers(t, tokenRouter.URL)
		return len(ledgers) == 3 && ledgers[0].RequestId == "e2e-request-3"
	}, 3*time.Second, 50*time.Millisecond)
	policyLedgers := adminGetUsageLedgers(t, tokenRouter.URL)
	require.Len(t, policyLedgers, 3)
	require.Equal(t, "e2e-request-3", policyLedgers[0].RequestId)
	require.Equal(t, e2eSessionID, policyLedgers[0].SessionId)
	require.Equal(t, 3, policyLedgers[0].ChannelId)
	require.Equal(t, 2, policyLedgers[0].SupplierId)
	require.Equal(t, "gb10-4t-self-hosted", policyLedgers[0].SupplyNode)
	require.True(t, policyLedgers[0].CacheHit)
	require.Greater(t, policyLedgers[0].SellQuota, policyLedgers[0].CostQuota)

	disabledPolicy := adminDisableSupplyRoutingPolicy(t, tokenRouter.URL, policy.Id)
	require.Equal(t, model.SupplyRoutingPolicyStatusDisabled, disabledPolicy.Status)
	require.Equal(t, 1, disabledPolicy.DisabledBy)
	require.Greater(t, disabledPolicy.DisabledAt, int64(0))
	reactivatedPolicy := adminActivateSupplyRoutingPolicy(t, tokenRouter.URL, selfHostedExecution.Id)
	require.Equal(t, policy.Id, reactivatedPolicy.Id)
	require.Equal(t, model.SupplyRoutingPolicyStatusActive, reactivatedPolicy.Status)
	require.Zero(t, reactivatedPolicy.DisabledBy)
	require.Zero(t, reactivatedPolicy.DisabledAt)

	require.True(t, model.UpdateChannelStatus(3, "", common.ChannelStatusManuallyDisabled, "policy miss e2e"))
	model.InitChannelCache()
	demandSimulatorChat(t, tokenRouter.URL, "e2e-request-policy-miss")
	require.Eventually(t, func() bool {
		ledgers := adminGetUsageLedgers(t, tokenRouter.URL)
		return len(ledgers) == 4 && ledgers[0].RequestId == "e2e-request-policy-miss"
	}, 3*time.Second, 50*time.Millisecond)
	policyMissLedgers := adminGetUsageLedgers(t, tokenRouter.URL)
	require.Len(t, policyMissLedgers, 4)
	require.Equal(t, "e2e-request-policy-miss", policyMissLedgers[0].RequestId)
	require.NotEqual(t, 3, policyMissLedgers[0].ChannelId)
	require.Equal(t, 1, policyMissLedgers[0].SupplierId)
	policyMissInsights := adminGetOperatingInsights(t, tokenRouter.URL, model.OperatingInsightStatusDraft, model.OperatingInsightCategoryQualityWatch)
	require.NotEmpty(t, policyMissInsights)
	var policyMissInsight *model.OperatingInsight
	for i := range policyMissInsights {
		if strings.Contains(policyMissInsights[i].InsightKey, "supply_routing_policy_miss") {
			policyMissInsight = &policyMissInsights[i]
			break
		}
	}
	require.NotNil(t, policyMissInsight)
	require.Equal(t, model.OperatingInsightSeverityWatch, policyMissInsight.Severity)
	require.Equal(t, policy.SupplyDecisionId, policyMissInsight.SupplyDecisionId)
	require.Contains(t, policyMissInsight.InsightKey, fmt.Sprintf("policy:%d", policy.Id))
	require.Equal(t, routingSlaRun.Id, policyMissInsight.SlaProbeRunId)
	require.Contains(t, policyMissInsight.Summary, "policy channel is disabled")

	disabledPolicy = adminDisableSupplyRoutingPolicy(t, tokenRouter.URL, policy.Id)
	require.Equal(t, model.SupplyRoutingPolicyStatusDisabled, disabledPolicy.Status)
	require.Equal(t, 1, disabledPolicy.DisabledBy)
	require.Greater(t, disabledPolicy.DisabledAt, int64(0))

	assignedSession := demandSimulatorChatWithoutSession(t, tokenRouter.URL, "e2e-request-assigned-session")
	require.True(t, strings.HasPrefix(assignedSession, "trsess_"))
	require.Eventually(t, func() bool {
		return len(adminGetUsageLedgersBySession(t, tokenRouter.URL, assignedSession)) == 1
	}, 3*time.Second, 50*time.Millisecond)
	assignedLedgers := adminGetUsageLedgersBySession(t, tokenRouter.URL, assignedSession)
	require.Len(t, assignedLedgers, 1)
	require.Equal(t, "e2e-request-assigned-session", assignedLedgers[0].RequestId)
	require.Equal(t, assignedSession, assignedLedgers[0].SessionId)
	require.Positive(t, assignedLedgers[0].ChannelId)
	require.Positive(t, assignedLedgers[0].SupplierId)
	require.NotEmpty(t, assignedLedgers[0].SupplyNode)
	require.Greater(t, assignedLedgers[0].SellQuota, assignedLedgers[0].CostQuota)
}

func requireLedgerCountStable(t *testing.T, baseURL string, expected int, duration time.Duration) {
	t.Helper()
	deadline := time.Now().Add(duration)
	for {
		require.Len(t, adminGetUsageLedgers(t, baseURL), expected)
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
