package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunReviewOncePostsGenerationChain(t *testing.T) {
	expectedPaths := []string{
		"/api/supplier_scorecards/generate",
		"/api/supplier_posture_recommendations/generate",
		"/api/traffic_profiles/generate",
		"/api/traffic_forecasts/generate",
		"/api/pricing_recommendations/generate",
		"/api/supply_decisions/generate",
		"/api/supply_expansion_opportunities/generate",
		"/api/operating_insights/generate",
	}
	var paths []string
	payloads := map[string]map[string]any{}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAdminHeaders(t, r)
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		paths = append(paths, r.URL.Path)
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode review payload: %v", err)
		}
		payloads[r.URL.Path] = payload
		writeEnvelope(w, []json.RawMessage{json.RawMessage(`{"id":1}`)})
	}))
	defer api.Close()

	summary, err := runReviewOnce(context.Background(), reviewOnceConfig{
		APIBase:           api.URL,
		AdminToken:        "admin-token",
		SupplierId:        7,
		ModelName:         " gpt-test ",
		SlaTier:           " gold ",
		UserId:            3,
		PeriodStart:       100,
		PeriodEnd:         200,
		TargetPeriodStart: 200,
		TargetPeriodEnd:   300,
		Timeout:           5 * time.Second,
	})
	if err != nil {
		t.Fatalf("run review once: %v", err)
	}
	if summary.TotalGenerated != len(expectedPaths) || len(summary.Steps) != len(expectedPaths) {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	for i, expected := range expectedPaths {
		if paths[i] != expected {
			t.Fatalf("unexpected path at %d: got %s want %s all=%v", i, paths[i], expected, paths)
		}
	}
	supplierPayload := payloads["/api/supplier_scorecards/generate"]
	if supplierPayload["period_start"] != float64(100) ||
		supplierPayload["period_end"] != float64(200) ||
		supplierPayload["supplier_id"] != float64(7) {
		t.Fatalf("unexpected supplier payload: %+v", supplierPayload)
	}
	trafficPayload := payloads["/api/traffic_profiles/generate"]
	if trafficPayload["model_name"] != "gpt-test" ||
		trafficPayload["sla_tier"] != "gold" ||
		trafficPayload["user_id"] != float64(3) {
		t.Fatalf("unexpected traffic payload: %+v", trafficPayload)
	}
	forecastPayload := payloads["/api/traffic_forecasts/generate"]
	if forecastPayload["target_period_start"] != float64(200) ||
		forecastPayload["target_period_end"] != float64(300) {
		t.Fatalf("unexpected forecast payload: %+v", forecastPayload)
	}
}

func TestReviewOnceResultGates(t *testing.T) {
	summary := reviewOnceSummary{TotalGenerated: 1}
	if err := checkReviewOnceResult(summary, reviewOnceConfig{MinGenerated: 1}); err != nil {
		t.Fatalf("expected min-generated=1 to pass: %v", err)
	}
	if err := checkReviewOnceResult(summary, reviewOnceConfig{MinGenerated: 2}); err == nil {
		t.Fatal("expected min-generated gate to fail")
	}
}

func TestRunReviewOnceCLIRequiresAdminToken(t *testing.T) {
	if err := runReviewOnceCLI([]string{"--period-start", "100", "--period-end", "200"}); err == nil {
		t.Fatal("expected missing admin token error")
	}
}

func TestRunReviewAgentCyclePostsGenerationChain(t *testing.T) {
	expectedPaths := []string{
		"/api/supplier_scorecards/generate",
		"/api/supplier_posture_recommendations/generate",
		"/api/traffic_profiles/generate",
		"/api/traffic_forecasts/generate",
		"/api/pricing_recommendations/generate",
		"/api/supply_decisions/generate",
		"/api/supply_expansion_opportunities/generate",
		"/api/operating_insights/generate",
	}
	var paths []string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAdminHeaders(t, r)
		paths = append(paths, r.URL.Path)
		writeEnvelope(w, []json.RawMessage{json.RawMessage(`{"id":1}`)})
	}))
	defer api.Close()

	summary, err := runReviewAgentCycle(context.Background(), reviewAgentConfig{
		reviewOnceConfig: reviewOnceConfig{
			APIBase:     api.URL,
			AdminToken:  "admin-token",
			PeriodStart: 100,
			PeriodEnd:   200,
			Timeout:     5 * time.Second,
		},
		AgentKey:   "aima2:review",
		Hostname:   "aima2",
		RuntimeRef: "pid:42",
		Version:    "v1",
		Interval:   time.Minute,
	})
	if err != nil {
		t.Fatalf("run review agent cycle: %v", err)
	}
	if summary.AgentKey != "aima2:review" || summary.Status != "ok" {
		t.Fatalf("unexpected summary identity/status: %+v", summary)
	}
	if summary.Review.TotalGenerated != len(expectedPaths) || len(summary.Review.Steps) != len(expectedPaths) {
		t.Fatalf("unexpected review summary: %+v", summary.Review)
	}
	for i, expected := range expectedPaths {
		if paths[i] != expected {
			t.Fatalf("unexpected path at %d: got %s want %s all=%v", i, paths[i], expected, paths)
		}
	}
}

func TestRunReviewAgentCycleUsesRollingDefaultPeriod(t *testing.T) {
	var payloads []map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAdminHeaders(t, r)
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode review payload: %v", err)
		}
		payloads = append(payloads, payload)
		writeEnvelope(w, []json.RawMessage{})
	}))
	defer api.Close()

	before := time.Now().Unix() - 2
	summary, err := runReviewAgentCycle(context.Background(), reviewAgentConfig{
		reviewOnceConfig: reviewOnceConfig{
			APIBase:    api.URL,
			AdminToken: "admin-token",
			Timeout:    5 * time.Second,
		},
		AgentKey:   "aima2:review",
		Hostname:   "aima2",
		RuntimeRef: "pid:42",
		Version:    "v1",
		Interval:   time.Minute,
	})
	after := time.Now().Unix() + 2
	if err != nil {
		t.Fatalf("run review agent cycle: %v", err)
	}
	if summary.Review.PeriodStart <= 0 || summary.Review.PeriodEnd <= summary.Review.PeriodStart {
		t.Fatalf("unexpected rolling period in summary: %+v", summary.Review)
	}
	if len(payloads) == 0 {
		t.Fatal("expected review payloads")
	}
	periodStart := int64(payloads[0]["period_start"].(float64))
	periodEnd := int64(payloads[0]["period_end"].(float64))
	if periodEnd < before || periodEnd > after {
		t.Fatalf("period end was not current: got %d want between %d and %d", periodEnd, before, after)
	}
	if periodEnd-periodStart != int64(time.Hour/time.Second) {
		t.Fatalf("unexpected rolling period width: start=%d end=%d", periodStart, periodEnd)
	}
}

func TestRunReviewAgentCycleRecordsMinGeneratedFailure(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAdminHeaders(t, r)
		writeEnvelope(w, []json.RawMessage{})
	}))
	defer api.Close()

	summary, err := runReviewAgentCycle(context.Background(), reviewAgentConfig{
		reviewOnceConfig: reviewOnceConfig{
			APIBase:      api.URL,
			AdminToken:   "admin-token",
			PeriodStart:  100,
			PeriodEnd:    200,
			MinGenerated: 1,
			Timeout:      5 * time.Second,
		},
		AgentKey:   "aima2:review",
		Hostname:   "aima2",
		RuntimeRef: "pid:42",
		Version:    "v1",
		Interval:   time.Minute,
	})
	if err == nil {
		t.Fatal("expected min-generated failure")
	}
	if summary.Status != "failed" || summary.Review.Status != "failed" || summary.Error == "" {
		t.Fatalf("unexpected failed summary: %+v", summary)
	}
}

func TestRunReviewAgentCLIRequiresAdminToken(t *testing.T) {
	if err := runReviewAgentCLI([]string{"--once"}); err == nil {
		t.Fatal("expected missing admin token error")
	}
}

func TestSweepTelemetryPostsFlagPayload(t *testing.T) {
	var posted telemetrySweepInput
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/supply_capacity_telemetries/sweep" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		requireAdminHeaders(t, r)
		if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
			t.Fatalf("decode sweep input: %v", err)
		}
		writeEnvelope(w, telemetrySweepResult{
			AttemptedCount: 1,
			CollectedCount: 1,
			SkippedCount:   0,
			Collected:      []json.RawMessage{json.RawMessage(`{"id":12,"source_ref":"gb10-4t-mock-capacity"}`)},
			Skipped:        []telemetrySweepSkip{},
		})
	}))
	defer api.Close()

	result, err := sweepTelemetry(context.Background(), telemetrySweepConfig{
		APIBase:     api.URL,
		AdminToken:  "admin-token",
		SupplierId:  1,
		ChannelId:   2,
		SupplyNode:  " gb10-4t ",
		ModelName:   " gpt-test ",
		PeriodStart: 100,
		PeriodEnd:   200,
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("sweep telemetry: %v", err)
	}
	if result.AttemptedCount != 1 || result.CollectedCount != 1 || result.SkippedCount != 0 {
		t.Fatalf("unexpected sweep result: %+v", result)
	}
	if posted.SupplierId != 1 || posted.ChannelId != 2 || posted.SupplyNode != "gb10-4t" || posted.ModelName != "gpt-test" {
		t.Fatalf("unexpected posted identity: %+v", posted)
	}
	if posted.PeriodStart != 100 || posted.PeriodEnd != 200 {
		t.Fatalf("unexpected posted period: %+v", posted)
	}
}

func TestTelemetrySweepResultGates(t *testing.T) {
	result := telemetrySweepResult{
		CollectedCount: 1,
		SkippedCount:   1,
	}
	if err := checkTelemetrySweepResult(result, telemetrySweepConfig{}); err != nil {
		t.Fatalf("default sweep gate should pass: %v", err)
	}
	if err := checkTelemetrySweepResult(result, telemetrySweepConfig{FailOnSkip: true}); err == nil {
		t.Fatal("expected fail-on-skip gate to fail")
	}
	if err := checkTelemetrySweepResult(result, telemetrySweepConfig{MinCollected: 2}); err == nil {
		t.Fatal("expected min-collected gate to fail")
	}
	if err := checkTelemetrySweepResult(result, telemetrySweepConfig{MinCollected: 1}); err != nil {
		t.Fatalf("expected min-collected=1 to pass: %v", err)
	}
}

func TestRunTelemetrySweepCLIRequiresAdminToken(t *testing.T) {
	if err := runTelemetrySweepCLI([]string{}); err == nil {
		t.Fatal("expected missing admin token error")
	}
}

func TestRunTelemetryAgentCyclePostsHeartbeatSweepResult(t *testing.T) {
	var heartbeat telemetryAgentHeartbeatInput
	var sweep telemetrySweepInput
	var sweepResult telemetryAgentSweepResultInput
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAdminHeaders(t, r)
		switch r.URL.Path {
		case "/api/supply_telemetry_agents/heartbeat":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected heartbeat method: %s", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&heartbeat); err != nil {
				t.Fatalf("decode heartbeat input: %v", err)
			}
			writeEnvelope(w, json.RawMessage(`{"id":1,"agent_key":"aima2:telemetry"}`))
		case "/api/supply_capacity_telemetries/sweep":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected sweep method: %s", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&sweep); err != nil {
				t.Fatalf("decode sweep input: %v", err)
			}
			writeEnvelope(w, telemetrySweepResult{
				AttemptedCount: 1,
				CollectedCount: 1,
				SkippedCount:   0,
				Collected:      []json.RawMessage{json.RawMessage(`{"id":12}`)},
				Skipped:        []telemetrySweepSkip{},
			})
		case "/api/supply_telemetry_agents/sweep_result":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected sweep result method: %s", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&sweepResult); err != nil {
				t.Fatalf("decode sweep result input: %v", err)
			}
			writeEnvelope(w, json.RawMessage(`{"id":1,"agent_key":"aima2:telemetry"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	cfg := telemetryAgentConfig{
		telemetrySweepConfig: telemetrySweepConfig{
			APIBase:     api.URL,
			AdminToken:  "admin-token",
			SupplierId:  1,
			ChannelId:   2,
			SupplyNode:  " gb10-4t ",
			ModelName:   " gpt-test ",
			PeriodStart: 100,
			PeriodEnd:   200,
			Timeout:     5 * time.Second,
		},
		AgentKey:   "aima2:telemetry",
		AgentType:  "telemetry",
		Hostname:   "aima2",
		RuntimeRef: "pid:42",
		Version:    "v1",
		Interval:   time.Minute,
	}

	summary, err := runTelemetryAgentCycle(context.Background(), cfg)
	if err != nil {
		t.Fatalf("run telemetry agent cycle: %v", err)
	}
	if summary.AgentKey != "aima2:telemetry" || summary.Status != "ok" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if heartbeat.AgentKey != "aima2:telemetry" || heartbeat.Hostname != "aima2" || heartbeat.RuntimeRef != "pid:42" {
		t.Fatalf("unexpected heartbeat: %+v", heartbeat)
	}
	if sweep.SupplierId != 1 || sweep.ChannelId != 2 || sweep.SupplyNode != "gb10-4t" || sweep.ModelName != "gpt-test" {
		t.Fatalf("unexpected sweep input: %+v", sweep)
	}
	if sweepResult.Status != "ok" || sweepResult.AttemptedCount != 1 || sweepResult.CollectedCount != 1 || sweepResult.SkippedCount != 0 {
		t.Fatalf("unexpected sweep result input: %+v", sweepResult)
	}
	if sweepResult.SupplierId != 1 || sweepResult.SupplyNode != "gb10-4t" || sweepResult.ModelName != "gpt-test" {
		t.Fatalf("unexpected sweep result filters: %+v", sweepResult)
	}
}

func TestRunTelemetryAgentCycleRecordsSkippedGateFailure(t *testing.T) {
	var sweepResult telemetryAgentSweepResultInput
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireAdminHeaders(t, r)
		switch r.URL.Path {
		case "/api/supply_telemetry_agents/heartbeat":
			writeEnvelope(w, json.RawMessage(`{"id":1}`))
		case "/api/supply_capacity_telemetries/sweep":
			writeEnvelope(w, telemetrySweepResult{
				AttemptedCount: 1,
				CollectedCount: 0,
				SkippedCount:   1,
				Collected:      []json.RawMessage{},
				Skipped: []telemetrySweepSkip{{
					CapacityId: 1,
					Reason:     "no enabled channel with base_url supports capacity model",
				}},
			})
		case "/api/supply_telemetry_agents/sweep_result":
			if err := json.NewDecoder(r.Body).Decode(&sweepResult); err != nil {
				t.Fatalf("decode sweep result input: %v", err)
			}
			writeEnvelope(w, json.RawMessage(`{"id":1}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	summary, err := runTelemetryAgentCycle(context.Background(), telemetryAgentConfig{
		telemetrySweepConfig: telemetrySweepConfig{
			APIBase:    api.URL,
			AdminToken: "admin-token",
			FailOnSkip: true,
			Timeout:    5 * time.Second,
		},
		AgentKey:   "aima2:telemetry",
		AgentType:  "telemetry",
		Hostname:   "aima2",
		RuntimeRef: "pid:42",
		Version:    "v1",
		Interval:   time.Minute,
	})
	if err == nil {
		t.Fatal("expected fail-on-skip error")
	}
	if summary.Status != "skipped" {
		t.Fatalf("unexpected summary status: %+v", summary)
	}
	if sweepResult.Status != "skipped" || sweepResult.SkippedCount != 1 {
		t.Fatalf("unexpected recorded sweep result: %+v", sweepResult)
	}
	if sweepResult.Error == "" {
		t.Fatalf("expected recorded gate error: %+v", sweepResult)
	}
}

func TestRunTelemetryAgentCLIRequiresAdminToken(t *testing.T) {
	if err := runTelemetryAgentCLI([]string{"--once"}); err == nil {
		t.Fatal("expected missing admin token error")
	}
}

func requireAdminHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("Authorization"); got != "Bearer admin-token" {
		t.Fatalf("unexpected admin authorization: %q", got)
	}
	if got := r.Header.Get("New-Api-User"); got != "1" {
		t.Fatalf("unexpected New-Api-User: %q", got)
	}
}

func writeEnvelope(tw http.ResponseWriter, data any) {
	tw.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(tw).Encode(apiEnvelope[any]{
		Success: true,
		Data:    data,
	})
}
