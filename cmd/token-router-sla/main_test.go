package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestExecuteProbeRunStreamingRecordsArtifact(t *testing.T) {
	var mu sync.Mutex
	var sessions []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-demandtoken" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		sessionID := r.Header.Get("X-Session-Id")
		mu.Lock()
		sessions = append(sessions, sessionID)
		callIndex := len(sessions)
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Request-Id", fmt.Sprintf("trace-%d", callIndex))
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not flush")
		}
		fmt.Fprintf(w, "data: {\"id\":\"chunk-%d\",\"model\":\"gpt-test\",\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n", callIndex)
		flusher.Flush()
		time.Sleep(5 * time.Millisecond)
		fmt.Fprintf(w, "data: {\"id\":\"chunk-%d\",\"model\":\"gpt-test\",\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n", callIndex)
		flusher.Flush()
		fmt.Fprintf(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":10,\"total_tokens\":110,\"prompt_tokens_details\":{\"cached_tokens\":0}}}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer upstream.Close()

	var recorded slaProbeRunRecordInput
	api := newProbeAPIServer(t, func(input slaProbeRunRecordInput) {
		recorded = input
	})
	defer api.Close()

	outPath := filepath.Join(t.TempDir(), "run.json")
	result, err := executeProbeRun(context.Background(), probeConfig{
		APIBase:     api.URL,
		AdminToken:  "admin-token",
		PlanID:      7,
		Endpoint:    upstream.URL + "/v1/chat/completions",
		DemandToken: "demandtoken",
		OutPath:     outPath,
		Record:      true,
		Stream:      true,
		RunKey:      "test-run",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("execute probe run: %v", err)
	}
	if !result.Recorded {
		t.Fatal("expected run to be recorded")
	}
	if result.ArtifactPath != outPath || result.ArtifactSHA == "" {
		t.Fatalf("unexpected artifact result: %+v", result)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("artifact not written: %v", err)
	}
	if recorded.Status != "passed" || !recorded.HardGatePassed {
		t.Fatalf("expected passed hard gate record, got %+v", recorded)
	}
	if recorded.ArtifactSHA256 != result.ArtifactSHA {
		t.Fatalf("recorded artifact hash mismatch: got %s want %s", recorded.ArtifactSHA256, result.ArtifactSHA)
	}
	var summary probeSummary
	if err := json.Unmarshal([]byte(recorded.SummaryJSON), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if !summary.TTFTObserved || summary.TTFTMs == nil || summary.Usage.PromptTokens != 200 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0] == "" || sessions[1] == "" || sessions[0] == sessions[1] {
		t.Fatalf("cold_no_cache sessions should be unique and non-empty: %#v", sessions)
	}
}

func TestExecuteProbeRunNonStreamingDoesNotSatisfyTTFTGate(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "chatcmpl-1",
			"model": "gpt-test",
			"choices": []map[string]any{{
				"message": map[string]string{"role": "assistant", "content": "ok"},
			}},
			"usage": map[string]any{
				"prompt_tokens":     100,
				"completion_tokens": 10,
				"total_tokens":      110,
			},
		})
	}))
	defer upstream.Close()

	api := newProbeAPIServer(t, nil)
	defer api.Close()

	result, err := executeProbeRun(context.Background(), probeConfig{
		APIBase:     api.URL,
		AdminToken:  "admin-token",
		PlanID:      7,
		Endpoint:    upstream.URL + "/v1/chat/completions",
		DemandToken: "sk-demandtoken",
		OutPath:     filepath.Join(t.TempDir(), "run.json"),
		Stream:      false,
		RunKey:      "non-streaming-run",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("execute probe run: %v", err)
	}
	if result.RecordInput.Status != "failed" || result.RecordInput.HardGatePassed {
		t.Fatalf("expected TTFT gate failure for non-streaming run, got %+v", result.RecordInput)
	}
	if result.RecordInput.FailureReasons == "" {
		t.Fatal("expected failure reasons for non-streaming TTFT gate")
	}
	var summary probeSummary
	if err := json.Unmarshal([]byte(result.RecordInput.SummaryJSON), &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.TTFTObserved || summary.TTFTMs != nil {
		t.Fatalf("non-streaming run must not claim TTFT: %+v", summary)
	}
}

func TestImportContractPostsFilePayload(t *testing.T) {
	var posted slaContractImportInput
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sla_contracts/import" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		requireAdminHeaders(t, r)
		if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
			t.Fatalf("decode import input: %v", err)
		}
		writeEnvelope(w, slaContract{
			Id:             11,
			ContractKey:    posted.ContractKey,
			ModelName:      posted.ModelName,
			ProviderFamily: posted.ProviderFamily,
			Version:        posted.Version,
		})
	}))
	defer api.Close()

	inputPath := filepath.Join(t.TempDir(), "contract.json")
	input := []byte(`{
		"contract_key":"kimi-k25-official",
		"model_name":"kimi-k2.5",
		"provider_family":"kimi",
		"source_name":"official",
		"source_ref":"https://example.test/kimi",
		"source_sha256":"abc123",
		"version":"2026-06",
		"status":"active",
		"measurement_profile_json":"{\"cache_profile\":\"cold_no_cache\"}",
		"hard_gate_json":"{\"ttft_ms\":{\"p90_lte\":1000}}",
		"soft_gate_json":"{}"
	}`)
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		t.Fatalf("write contract input: %v", err)
	}

	contract, err := importContract(context.Background(), contractImportConfig{
		APIBase:    api.URL,
		AdminToken: "admin-token",
		InputPath:  inputPath,
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("import contract: %v", err)
	}
	if contract.Id != 11 || contract.ContractKey != "kimi-k25-official" {
		t.Fatalf("unexpected contract response: %+v", contract)
	}
	if posted.SourceSHA256 != "abc123" || posted.Status != "active" {
		t.Fatalf("unexpected posted contract payload: %+v", posted)
	}
	if posted.HardGateJSON == "" || posted.MeasurementProfileJSON == "" {
		t.Fatalf("expected JSON profile payloads: %+v", posted)
	}
}

func TestGeneratePlanPostsFlagPayload(t *testing.T) {
	var posted slaProbePlanGenerateInput
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sla_probe_plans/generate" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		requireAdminHeaders(t, r)
		if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
			t.Fatalf("decode plan input: %v", err)
		}
		writeEnvelope(w, slaProbePlan{
			Id:             17,
			PlanKey:        "plan-17",
			ContractId:     11,
			SupplierId:     posted.SupplierId,
			ChannelId:      posted.ChannelId,
			ModelName:      posted.ModelName,
			SlaTier:        posted.SlaTier,
			ProbeType:      posted.ProbeType,
			RouteMode:      posted.RouteMode,
			PromptSuiteKey: posted.PromptSuiteKey,
			SampleSize:     posted.SampleSize,
			RepeatCount:    posted.RepeatCount,
			CacheProfile:   posted.CacheProfile,
		})
	}))
	defer api.Close()

	plan, err := generatePlan(context.Background(), planGenerateConfig{
		APIBase:           api.URL,
		AdminToken:        "admin-token",
		ContractKey:       "kimi-k25-official",
		SupplierId:        1,
		ChannelId:         2,
		ModelName:         "kimi-k2.5",
		SlaTier:           "gold",
		ProbeType:         "admission",
		RouteMode:         "through_token_router",
		PromptSuiteKey:    "official-admission",
		TokenizerRef:      "contract",
		SampleSize:        4,
		RepeatCount:       2,
		OutputProfileJSON: `{"target_tokens":128}`,
		CacheProfile:      "warm_same_session",
		MaxProbeQuota:     9000,
		Timeout:           5 * time.Second,
	})
	if err != nil {
		t.Fatalf("generate plan: %v", err)
	}
	if plan.Id != 17 || plan.SupplierId != 1 || plan.ChannelId != 2 {
		t.Fatalf("unexpected plan response: %+v", plan)
	}
	if posted.ContractKey != "kimi-k25-official" || posted.SupplierId != 1 || posted.ChannelId != 2 {
		t.Fatalf("unexpected posted plan identity: %+v", posted)
	}
	if posted.OutputProfileJSON != `{"target_tokens":128}` || posted.CacheProfile != "warm_same_session" || posted.MaxProbeQuota != 9000 {
		t.Fatalf("unexpected posted plan profile: %+v", posted)
	}
	if posted.SampleSize != 4 || posted.RepeatCount != 2 {
		t.Fatalf("unexpected posted plan sample shape: %+v", posted)
	}
}

func newProbeAPIServer(t *testing.T, onRecord func(slaProbeRunRecordInput)) *httptest.Server {
	t.Helper()
	plan := slaProbePlan{
		Id:                7,
		PlanKey:           "plan-7",
		ContractId:        3,
		SupplierId:        1,
		ChannelId:         2,
		ModelName:         "gpt-test",
		SlaTier:           "default",
		ProbeType:         "admission",
		RouteMode:         "through_token_router",
		PromptSuiteKey:    "unit-smoke",
		SampleSize:        2,
		RepeatCount:       1,
		CacheProfile:      "cold_no_cache",
		OutputProfileJSON: `{"target_tokens":16}`,
	}
	contract := slaContract{
		Id:           3,
		ContractKey:  "contract-3",
		ModelName:    "gpt-test",
		HardGateJSON: `{"ttft_ms":{"p90_lte":1000}}`,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sla_probe_plans/7", func(w http.ResponseWriter, r *http.Request) {
		requireAdminHeaders(t, r)
		writeEnvelope(w, plan)
	})
	mux.HandleFunc("/api/sla_contracts/3", func(w http.ResponseWriter, r *http.Request) {
		requireAdminHeaders(t, r)
		writeEnvelope(w, contract)
	})
	mux.HandleFunc("/api/sla_probe_runs/record", func(w http.ResponseWriter, r *http.Request) {
		requireAdminHeaders(t, r)
		var input slaProbeRunRecordInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("decode record input: %v", err)
		}
		if onRecord != nil {
			onRecord(input)
		}
		writeEnvelope(w, map[string]any{"id": 9})
	})
	return httptest.NewServer(mux)
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

func writeEnvelope(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "",
		"data":    data,
	})
}
