package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const runnerVersion = "token-router-sla/0.1"

type slaProbePlan struct {
	Id                         int    `json:"id"`
	PlanKey                    string `json:"plan_key"`
	ContractId                 int    `json:"contract_id"`
	SupplierId                 int    `json:"supplier_id"`
	ChannelId                  int    `json:"channel_id"`
	ModelName                  string `json:"model_name"`
	SlaTier                    string `json:"sla_tier"`
	ProbeType                  string `json:"probe_type"`
	RouteMode                  string `json:"route_mode"`
	PromptSuiteKey             string `json:"prompt_suite_key"`
	TokenizerRef               string `json:"tokenizer_ref"`
	SampleSize                 int    `json:"sample_size"`
	RepeatCount                int    `json:"repeat_count"`
	CacheProfile               string `json:"cache_profile"`
	OutputProfileJSON          string `json:"output_profile_json"`
	StreamProfileJSON          string `json:"stream_profile_json"`
	MeasurementProfileSnapshot string `json:"measurement_profile_snapshot"`
}

type slaContract struct {
	Id                     int    `json:"id"`
	ContractKey            string `json:"contract_key"`
	ModelName              string `json:"model_name"`
	ProviderFamily         string `json:"provider_family"`
	SourceName             string `json:"source_name"`
	SourceRef              string `json:"source_ref"`
	SourceSHA256           string `json:"source_sha256"`
	Version                string `json:"version"`
	Status                 string `json:"status"`
	EffectiveFrom          int64  `json:"effective_from"`
	EffectiveTo            int64  `json:"effective_to"`
	MeasurementProfileJSON string `json:"measurement_profile_json"`
	HardGateJSON           string `json:"hard_gate_json"`
	SoftGateJSON           string `json:"soft_gate_json"`
}

type slaContractImportInput struct {
	ContractKey            string `json:"contract_key"`
	ModelName              string `json:"model_name"`
	ModelAliases           string `json:"model_aliases"`
	ProviderFamily         string `json:"provider_family"`
	SourceName             string `json:"source_name"`
	SourceRef              string `json:"source_ref"`
	SourceSHA256           string `json:"source_sha256"`
	Version                string `json:"version"`
	Status                 string `json:"status"`
	EffectiveFrom          int64  `json:"effective_from"`
	EffectiveTo            int64  `json:"effective_to"`
	MeasurementProfileJSON string `json:"measurement_profile_json"`
	HardGateJSON           string `json:"hard_gate_json"`
	SoftGateJSON           string `json:"soft_gate_json"`
}

type slaProbePlanGenerateInput struct {
	ContractId              int    `json:"contract_id"`
	ContractKey             string `json:"contract_key"`
	SupplierId              int    `json:"supplier_id"`
	ChannelId               int    `json:"channel_id"`
	ModelName               string `json:"model_name"`
	SlaTier                 string `json:"sla_tier"`
	ProbeType               string `json:"probe_type"`
	RouteMode               string `json:"route_mode"`
	PromptSuiteKey          string `json:"prompt_suite_key"`
	TokenizerRef            string `json:"tokenizer_ref"`
	SampleSize              int    `json:"sample_size"`
	RepeatCount             int    `json:"repeat_count"`
	InputProfileJSON        string `json:"input_profile_json"`
	OutputProfileJSON       string `json:"output_profile_json"`
	ConcurrencyProfileJSON  string `json:"concurrency_profile_json"`
	RateProfileJSON         string `json:"rate_profile_json"`
	StreamProfileJSON       string `json:"stream_profile_json"`
	ErrorProfileJSON        string `json:"error_profile_json"`
	AvailabilityProfileJSON string `json:"availability_profile_json"`
	CacheProfile            string `json:"cache_profile"`
	ScheduleIntervalSeconds int    `json:"schedule_interval_seconds"`
	JitterSeconds           int    `json:"jitter_seconds"`
	MaxProbeQuota           int64  `json:"max_probe_quota"`
}

type slaProbeRunRecordInput struct {
	RunKey           string `json:"run_key"`
	PlanId           int    `json:"plan_id"`
	Status           string `json:"status"`
	StartedAt        int64  `json:"started_at"`
	EndedAt          int64  `json:"ended_at"`
	RunnerVersion    string `json:"runner_version"`
	GitCommit        string `json:"git_commit"`
	RuntimeRef       string `json:"runtime_ref"`
	Endpoint         string `json:"endpoint"`
	SummaryJSON      string `json:"summary_json"`
	HardGatePassed   bool   `json:"hard_gate_passed"`
	SoftGateWarnings string `json:"soft_gate_warnings"`
	FailureReasons   string `json:"failure_reasons"`
	ArtifactURI      string `json:"artifact_uri"`
	ArtifactSHA256   string `json:"artifact_sha256"`
}

type probeConfig struct {
	APIBase     string
	AdminToken  string
	PlanID      int
	Endpoint    string
	DemandToken string
	OutPath     string
	ArtifactURI string
	Record      bool
	Stream      bool
	Prompt      string
	RuntimeRef  string
	GitCommit   string
	RunKey      string
	Timeout     time.Duration
}

type contractImportConfig struct {
	APIBase    string
	AdminToken string
	InputPath  string
	Timeout    time.Duration
}

type planGenerateConfig struct {
	APIBase                 string
	AdminToken              string
	ContractId              int
	ContractKey             string
	SupplierId              int
	ChannelId               int
	ModelName               string
	SlaTier                 string
	ProbeType               string
	RouteMode               string
	PromptSuiteKey          string
	TokenizerRef            string
	SampleSize              int
	RepeatCount             int
	InputProfileJSON        string
	OutputProfileJSON       string
	ConcurrencyProfileJSON  string
	RateProfileJSON         string
	StreamProfileJSON       string
	ErrorProfileJSON        string
	AvailabilityProfileJSON string
	CacheProfile            string
	ScheduleIntervalSeconds int
	JitterSeconds           int
	MaxProbeQuota           int64
	Timeout                 time.Duration
}

type probeArtifact struct {
	RunnerVersion string        `json:"runner_version"`
	RunKey        string        `json:"run_key"`
	Endpoint      string        `json:"endpoint"`
	StartedAt     int64         `json:"started_at"`
	EndedAt       int64         `json:"ended_at"`
	Stream        bool          `json:"stream"`
	Plan          slaProbePlan  `json:"plan"`
	Contract      *slaContract  `json:"contract,omitempty"`
	Samples       []probeSample `json:"samples"`
	Summary       probeSummary  `json:"summary"`
}

type probeSample struct {
	SampleId         string `json:"sample_id"`
	SessionId        string `json:"session_id"`
	OK               bool   `json:"ok"`
	HTTPStatus       int    `json:"http_status"`
	FailureClass     string `json:"failure_class,omitempty"`
	FailureMessage   string `json:"failure_message,omitempty"`
	FirstByteMs      int64  `json:"first_byte_ms"`
	FirstEventMs     int64  `json:"first_event_ms,omitempty"`
	FirstTokenMs     int64  `json:"first_token_ms,omitempty"`
	TotalLatencyMs   int64  `json:"total_latency_ms"`
	TTFTObserved     bool   `json:"ttft_observed"`
	PromptTokens     int    `json:"prompt_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	ResponseId       string `json:"response_id,omitempty"`
	Model            string `json:"model,omitempty"`
	RequestId        string `json:"request_id,omitempty"`
}

type probeSummary struct {
	SampleCount         int               `json:"sample_count"`
	SuccessCount        int               `json:"success_count"`
	FailureCount        int               `json:"failure_count"`
	SuccessRate         float64           `json:"success_rate"`
	Streaming           bool              `json:"streaming"`
	TTFTObserved        bool              `json:"ttft_observed"`
	SessionPolicy       string            `json:"session_policy"`
	CacheProfile        string            `json:"cache_profile"`
	TTFTMs              *metricSummary    `json:"ttft_ms,omitempty"`
	FirstByteMs         *metricSummary    `json:"first_byte_ms,omitempty"`
	TotalLatencyMs      *metricSummary    `json:"total_latency_ms,omitempty"`
	Usage               usageSummary      `json:"usage"`
	FailureClassCounts  map[string]int    `json:"failure_class_counts"`
	HardGateEvaluations []hardGateOutcome `json:"hard_gate_evaluations,omitempty"`
}

type metricSummary struct {
	P50 float64 `json:"p50"`
	P90 float64 `json:"p90"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
	Max float64 `json:"max"`
}

type usageSummary struct {
	PromptTokens     int `json:"prompt_tokens"`
	CachedTokens     int `json:"cached_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CacheHitCount    int `json:"cache_hit_count"`
}

type hardGateOutcome struct {
	Name      string  `json:"name"`
	ActualMs  float64 `json:"actual_ms"`
	LimitMs   float64 `json:"limit_ms"`
	Passed    bool    `json:"passed"`
	Supported bool    `json:"supported"`
}

type apiEnvelope[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: token-router-sla <contract import|plan generate|probe run> [options]")
	os.Exit(2)
}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 3 {
		usage()
	}
	var err error
	switch {
	case os.Args[1] == "contract" && os.Args[2] == "import":
		err = runContractImportCLI(os.Args[3:])
	case os.Args[1] == "plan" && os.Args[2] == "generate":
		err = runPlanGenerateCLI(os.Args[3:])
	case os.Args[1] == "probe" && os.Args[2] == "run":
		err = runProbeCLI(os.Args[3:])
	default:
		usage()
	}
	if err != nil {
		log.Fatal(err)
	}
}

func runContractImportCLI(args []string) error {
	cfg := contractImportConfig{}
	fs := flag.NewFlagSet("contract import", flag.ExitOnError)
	fs.StringVar(&cfg.APIBase, "api", "http://127.0.0.1:19090", "token-router admin API base URL")
	fs.StringVar(&cfg.AdminToken, "admin-token", "", "admin access token for contract import")
	fs.StringVar(&cfg.InputPath, "input", "", "contract import JSON payload path")
	fs.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "API request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.AdminToken) == "" {
		return errors.New("--admin-token is required")
	}
	if strings.TrimSpace(cfg.InputPath) == "" {
		return errors.New("--input is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	contract, err := importContract(ctx, cfg)
	if err != nil {
		return err
	}
	encoded, _ := json.MarshalIndent(contract, "", "  ")
	fmt.Println(string(encoded))
	return nil
}

func runPlanGenerateCLI(args []string) error {
	cfg := planGenerateConfig{}
	fs := flag.NewFlagSet("plan generate", flag.ExitOnError)
	fs.StringVar(&cfg.APIBase, "api", "http://127.0.0.1:19090", "token-router admin API base URL")
	fs.StringVar(&cfg.AdminToken, "admin-token", "", "admin access token for plan generation")
	fs.IntVar(&cfg.ContractId, "contract-id", 0, "SLA contract id")
	fs.StringVar(&cfg.ContractKey, "contract-key", "", "SLA contract key")
	fs.IntVar(&cfg.SupplierId, "supplier-id", 0, "supplier id to measure")
	fs.IntVar(&cfg.ChannelId, "channel-id", 0, "optional channel id to measure")
	fs.StringVar(&cfg.ModelName, "model", "", "model name override; defaults to contract model")
	fs.StringVar(&cfg.SlaTier, "sla-tier", "default", "SLA tier")
	fs.StringVar(&cfg.ProbeType, "type", "admission", "probe type")
	fs.StringVar(&cfg.RouteMode, "route-mode", "through_token_router", "probe route mode")
	fs.StringVar(&cfg.PromptSuiteKey, "prompt-suite", "default", "prompt suite key")
	fs.StringVar(&cfg.TokenizerRef, "tokenizer-ref", "contract", "tokenizer reference")
	fs.IntVar(&cfg.SampleSize, "sample-size", 1, "sample size")
	fs.IntVar(&cfg.RepeatCount, "repeat-count", 1, "repeat count")
	fs.StringVar(&cfg.InputProfileJSON, "input-profile-json", "", "input profile JSON override")
	fs.StringVar(&cfg.OutputProfileJSON, "output-profile-json", "", "output profile JSON override")
	fs.StringVar(&cfg.ConcurrencyProfileJSON, "concurrency-profile-json", "", "concurrency profile JSON override")
	fs.StringVar(&cfg.RateProfileJSON, "rate-profile-json", "", "rate profile JSON override")
	fs.StringVar(&cfg.StreamProfileJSON, "stream-profile-json", "", "stream profile JSON override")
	fs.StringVar(&cfg.ErrorProfileJSON, "error-profile-json", "", "error profile JSON override")
	fs.StringVar(&cfg.AvailabilityProfileJSON, "availability-profile-json", "", "availability profile JSON override")
	fs.StringVar(&cfg.CacheProfile, "cache-profile", "", "cache profile override")
	fs.IntVar(&cfg.ScheduleIntervalSeconds, "schedule-interval-seconds", 0, "runtime schedule interval seconds")
	fs.IntVar(&cfg.JitterSeconds, "jitter-seconds", 0, "runtime schedule jitter seconds")
	fs.Int64Var(&cfg.MaxProbeQuota, "max-probe-quota", 0, "maximum probe quota budget")
	fs.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "API request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.AdminToken) == "" {
		return errors.New("--admin-token is required")
	}
	if cfg.ContractId <= 0 && strings.TrimSpace(cfg.ContractKey) == "" {
		return errors.New("--contract-id or --contract-key is required")
	}
	if cfg.SupplierId <= 0 {
		return errors.New("--supplier-id is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	plan, err := generatePlan(ctx, cfg)
	if err != nil {
		return err
	}
	encoded, _ := json.MarshalIndent(plan, "", "  ")
	fmt.Println(string(encoded))
	return nil
}

func runProbeCLI(args []string) error {
	cfg := probeConfig{}
	fs := flag.NewFlagSet("probe run", flag.ExitOnError)
	fs.StringVar(&cfg.APIBase, "api", "http://127.0.0.1:19090", "token-router admin API base URL")
	fs.StringVar(&cfg.AdminToken, "admin-token", "", "admin access token for plan fetch and record")
	fs.IntVar(&cfg.PlanID, "plan-id", 0, "SLA probe plan id")
	fs.StringVar(&cfg.Endpoint, "endpoint", "", "OpenAI-compatible /v1/chat/completions endpoint to measure")
	fs.StringVar(&cfg.DemandToken, "demand-token", "", "Bearer token for measured endpoint; sk- prefix is optional")
	fs.StringVar(&cfg.OutPath, "out", "output/sla/run.json", "artifact JSON path or directory")
	fs.StringVar(&cfg.ArtifactURI, "artifact-uri", "", "artifact URI to record; defaults to file://<abs artifact path>")
	fs.BoolVar(&cfg.Record, "record", false, "record run through /api/sla_probe_runs/record")
	fs.BoolVar(&cfg.Stream, "stream", true, "request OpenAI streaming chat completions")
	fs.StringVar(&cfg.Prompt, "prompt", "token-router SLA probe smoke prompt", "user prompt for each sample")
	fs.StringVar(&cfg.RuntimeRef, "runtime-ref", "local", "runtime reference recorded with the run")
	fs.StringVar(&cfg.GitCommit, "git-commit", "", "git commit recorded with the run")
	fs.StringVar(&cfg.RunKey, "run-key", "", "run key; defaults to plan/time")
	fs.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "per-run HTTP timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if cfg.PlanID <= 0 {
		return errors.New("--plan-id is required")
	}
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return errors.New("--endpoint is required")
	}
	if strings.TrimSpace(cfg.AdminToken) == "" {
		return errors.New("--admin-token is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	result, err := executeProbeRun(ctx, cfg)
	if err != nil {
		return err
	}
	encoded, _ := json.MarshalIndent(result.RecordInput, "", "  ")
	fmt.Println(string(encoded))
	return nil
}

func importContract(ctx context.Context, cfg contractImportConfig) (slaContract, error) {
	var zero slaContract
	payload, err := os.ReadFile(cfg.InputPath)
	if err != nil {
		return zero, err
	}
	var input slaContractImportInput
	if err := json.Unmarshal(payload, &input); err != nil {
		return zero, err
	}
	client := &http.Client{Timeout: cfg.Timeout}
	return postAPI[slaContract](ctx, client, cfg.APIBase, cfg.AdminToken, "/api/sla_contracts/import", input)
}

func generatePlan(ctx context.Context, cfg planGenerateConfig) (slaProbePlan, error) {
	client := &http.Client{Timeout: cfg.Timeout}
	input := slaProbePlanGenerateInput{
		ContractId:              cfg.ContractId,
		ContractKey:             strings.TrimSpace(cfg.ContractKey),
		SupplierId:              cfg.SupplierId,
		ChannelId:               cfg.ChannelId,
		ModelName:               strings.TrimSpace(cfg.ModelName),
		SlaTier:                 strings.TrimSpace(cfg.SlaTier),
		ProbeType:               strings.TrimSpace(cfg.ProbeType),
		RouteMode:               strings.TrimSpace(cfg.RouteMode),
		PromptSuiteKey:          strings.TrimSpace(cfg.PromptSuiteKey),
		TokenizerRef:            strings.TrimSpace(cfg.TokenizerRef),
		SampleSize:              cfg.SampleSize,
		RepeatCount:             cfg.RepeatCount,
		InputProfileJSON:        strings.TrimSpace(cfg.InputProfileJSON),
		OutputProfileJSON:       strings.TrimSpace(cfg.OutputProfileJSON),
		ConcurrencyProfileJSON:  strings.TrimSpace(cfg.ConcurrencyProfileJSON),
		RateProfileJSON:         strings.TrimSpace(cfg.RateProfileJSON),
		StreamProfileJSON:       strings.TrimSpace(cfg.StreamProfileJSON),
		ErrorProfileJSON:        strings.TrimSpace(cfg.ErrorProfileJSON),
		AvailabilityProfileJSON: strings.TrimSpace(cfg.AvailabilityProfileJSON),
		CacheProfile:            strings.TrimSpace(cfg.CacheProfile),
		ScheduleIntervalSeconds: cfg.ScheduleIntervalSeconds,
		JitterSeconds:           cfg.JitterSeconds,
		MaxProbeQuota:           cfg.MaxProbeQuota,
	}
	return postAPI[slaProbePlan](ctx, client, cfg.APIBase, cfg.AdminToken, "/api/sla_probe_plans/generate", input)
}

type probeRunResult struct {
	ArtifactPath string
	ArtifactSHA  string
	RecordInput  slaProbeRunRecordInput
	Recorded     bool
}

func executeProbeRun(ctx context.Context, cfg probeConfig) (*probeRunResult, error) {
	client := &http.Client{Timeout: cfg.Timeout}
	plan, err := fetchAPI[slaProbePlan](ctx, client, cfg.APIBase, cfg.AdminToken, fmt.Sprintf("/api/sla_probe_plans/%d", cfg.PlanID))
	if err != nil {
		return nil, err
	}
	contract, err := fetchAPI[slaContract](ctx, client, cfg.APIBase, cfg.AdminToken, fmt.Sprintf("/api/sla_contracts/%d", plan.ContractId))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.RunKey) == "" {
		cfg.RunKey = fmt.Sprintf("sla-plan-%d-%d", plan.Id, time.Now().Unix())
	}
	started := time.Now()
	samples := make([]probeSample, 0, sampleCount(plan))
	for i := 0; i < sampleCount(plan); i++ {
		sample, err := executeSample(ctx, client, cfg, plan, i)
		if err != nil {
			sample = failedSample(plan, cfg.RunKey, i, "client_error", err.Error())
		}
		samples = append(samples, sample)
	}
	ended := time.Now()
	summary := summarizeSamples(plan, cfg.Stream, samples)
	hardGatePassed, failureReasons := evaluateHardGate(&summary, contract.HardGateJSON)
	status := "passed"
	if !hardGatePassed {
		status = "failed"
	}
	if summary.SuccessCount == 0 {
		status = "invalid"
	}
	artifact := probeArtifact{
		RunnerVersion: runnerVersion,
		RunKey:        cfg.RunKey,
		Endpoint:      cfg.Endpoint,
		StartedAt:     started.Unix(),
		EndedAt:       ended.Unix(),
		Stream:        cfg.Stream,
		Plan:          plan,
		Contract:      &contract,
		Samples:       samples,
		Summary:       summary,
	}
	artifactPath, artifactSHA, err := writeArtifact(cfg.OutPath, artifact)
	if err != nil {
		return nil, err
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return nil, err
	}
	artifactURI := strings.TrimSpace(cfg.ArtifactURI)
	if artifactURI == "" {
		abs, err := filepath.Abs(artifactPath)
		if err != nil {
			return nil, err
		}
		artifactURI = "file://" + abs
	}
	recordInput := slaProbeRunRecordInput{
		RunKey:           cfg.RunKey,
		PlanId:           plan.Id,
		Status:           status,
		StartedAt:        started.Unix(),
		EndedAt:          ended.Unix(),
		RunnerVersion:    runnerVersion,
		GitCommit:        cfg.GitCommit,
		RuntimeRef:       cfg.RuntimeRef,
		Endpoint:         cfg.Endpoint,
		SummaryJSON:      string(summaryJSON),
		HardGatePassed:   hardGatePassed,
		FailureReasons:   strings.Join(failureReasons, "; "),
		ArtifactURI:      artifactURI,
		ArtifactSHA256:   artifactSHA,
		SoftGateWarnings: "",
	}
	result := &probeRunResult{
		ArtifactPath: artifactPath,
		ArtifactSHA:  artifactSHA,
		RecordInput:  recordInput,
	}
	if cfg.Record {
		if _, err := postAPI[json.RawMessage](ctx, client, cfg.APIBase, cfg.AdminToken, "/api/sla_probe_runs/record", recordInput); err != nil {
			return nil, err
		}
		result.Recorded = true
	}
	return result, nil
}

func fetchAPI[T any](ctx context.Context, client *http.Client, baseURL string, adminToken string, path string) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+path, nil)
	if err != nil {
		return zero, err
	}
	setAdminHeaders(req, adminToken)
	resp, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("GET %s returned status %d", path, resp.StatusCode)
	}
	var envelope apiEnvelope[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return zero, err
	}
	if !envelope.Success {
		if envelope.Message == "" {
			envelope.Message = "api returned success=false"
		}
		return zero, errors.New(envelope.Message)
	}
	return envelope.Data, nil
}

func postAPI[T any](ctx context.Context, client *http.Client, baseURL string, adminToken string, path string, payload any) (T, error) {
	var zero T
	encoded, err := json.Marshal(payload)
	if err != nil {
		return zero, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+path, bytes.NewReader(encoded))
	if err != nil {
		return zero, err
	}
	setAdminHeaders(req, adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("POST %s returned status %d", path, resp.StatusCode)
	}
	var envelope apiEnvelope[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return zero, err
	}
	if !envelope.Success {
		if envelope.Message == "" {
			envelope.Message = "api returned success=false"
		}
		return zero, errors.New(envelope.Message)
	}
	return envelope.Data, nil
}

func setAdminHeaders(req *http.Request, adminToken string) {
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("New-Api-User", "1")
}

func sampleCount(plan slaProbePlan) int {
	sampleSize := plan.SampleSize
	if sampleSize <= 0 {
		sampleSize = 1
	}
	repeatCount := plan.RepeatCount
	if repeatCount <= 0 {
		repeatCount = 1
	}
	return sampleSize * repeatCount
}

func executeSample(ctx context.Context, client *http.Client, cfg probeConfig, plan slaProbePlan, index int) (probeSample, error) {
	sessionID := sampleSessionID(plan, cfg.RunKey, index)
	sample := probeSample{
		SampleId:  fmt.Sprintf("sample-%04d", index+1),
		SessionId: sessionID,
		RequestId: fmt.Sprintf("%s-%04d", cfg.RunKey, index+1),
	}
	body := map[string]any{
		"model": plan.ModelName,
		"messages": []map[string]string{{
			"role":    "user",
			"content": fmt.Sprintf("%s\nplan=%d sample=%d", cfg.Prompt, plan.Id, index+1),
		}},
		"user":   sessionID,
		"stream": cfg.Stream,
	}
	if maxTokens := outputTargetTokens(plan.OutputProfileJSON); maxTokens > 0 {
		body["max_tokens"] = maxTokens
	}
	if cfg.Stream {
		body["stream_options"] = map[string]bool{"include_usage": true}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return sample, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return sample, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-Id", sessionID)
	req.Header.Set("Session_id", sessionID)
	if strings.TrimSpace(cfg.DemandToken) != "" {
		req.Header.Set("Authorization", authHeader(cfg.DemandToken))
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		sample.TotalLatencyMs = millisSince(start)
		sample.FailureClass = failureClassForError(err)
		sample.FailureMessage = err.Error()
		return sample, nil
	}
	defer resp.Body.Close()
	sample.HTTPStatus = resp.StatusCode
	sample.FirstByteMs = millisSince(start)
	if requestID := strings.TrimSpace(resp.Header.Get("X-Request-Id")); requestID != "" {
		sample.RequestId = requestID
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		sample.TotalLatencyMs = millisSince(start)
		sample.FailureClass = failureClassForStatus(resp.StatusCode)
		sample.FailureMessage = strings.TrimSpace(string(body))
		return sample, nil
	}
	if cfg.Stream {
		return readStreamingSample(resp.Body, start, sample)
	}
	return readJSONSample(resp.Body, start, sample)
}

func failedSample(plan slaProbePlan, runKey string, index int, failureClass string, message string) probeSample {
	return probeSample{
		SampleId:       fmt.Sprintf("sample-%04d", index+1),
		SessionId:      sampleSessionID(plan, runKey, index),
		OK:             false,
		FailureClass:   failureClass,
		FailureMessage: message,
	}
}

func readStreamingSample(body io.Reader, start time.Time, sample probeSample) (probeSample, error) {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			continue
		}
		if sample.FirstEventMs == 0 {
			sample.FirstEventMs = millisSince(start)
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			sample.FailureClass = "invalid_json"
			sample.FailureMessage = err.Error()
			continue
		}
		if id, ok := event["id"].(string); ok && sample.ResponseId == "" {
			sample.ResponseId = id
		}
		if model, ok := event["model"].(string); ok && sample.Model == "" {
			sample.Model = model
		}
		if sample.FirstTokenMs == 0 && chunkHasContent(event) {
			sample.FirstTokenMs = millisSince(start)
		}
		if usage, ok := event["usage"].(map[string]any); ok {
			applyUsage(&sample, usage)
		}
	}
	sample.TotalLatencyMs = millisSince(start)
	if err := scanner.Err(); err != nil {
		sample.FailureClass = "stream_read"
		sample.FailureMessage = err.Error()
		return sample, nil
	}
	if sample.FailureClass != "" {
		return sample, nil
	}
	if sample.FirstTokenMs == 0 {
		sample.FailureClass = "missing_first_token"
		sample.FailureMessage = "stream ended without a content delta"
		return sample, nil
	}
	if sample.TotalTokens == 0 && sample.PromptTokens == 0 && sample.CompletionTokens == 0 {
		sample.FailureClass = "missing_usage"
		sample.FailureMessage = "stream ended without usage"
		return sample, nil
	}
	sample.TTFTObserved = true
	sample.OK = true
	return sample, nil
}

func readJSONSample(body io.Reader, start time.Time, sample probeSample) (probeSample, error) {
	var decoded map[string]any
	if err := json.NewDecoder(body).Decode(&decoded); err != nil {
		sample.TotalLatencyMs = millisSince(start)
		sample.FailureClass = "invalid_json"
		sample.FailureMessage = err.Error()
		return sample, nil
	}
	sample.TotalLatencyMs = millisSince(start)
	if id, ok := decoded["id"].(string); ok {
		sample.ResponseId = id
	}
	if model, ok := decoded["model"].(string); ok {
		sample.Model = model
	}
	usage, ok := decoded["usage"].(map[string]any)
	if !ok {
		sample.FailureClass = "missing_usage"
		sample.FailureMessage = "response missing usage"
		return sample, nil
	}
	applyUsage(&sample, usage)
	sample.OK = true
	return sample, nil
}

func sampleSessionID(plan slaProbePlan, runKey string, index int) string {
	base := sanitizeID(runKey)
	switch strings.TrimSpace(plan.CacheProfile) {
	case "warm_same_session":
		return "sla_" + base + "_warm"
	case "mixed_trace_replay":
		return fmt.Sprintf("sla_%s_trace_%02d", base, (index%4)+1)
	default:
		return fmt.Sprintf("sla_%s_cold_%04d", base, index+1)
	}
}

func sanitizeID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return strconv.FormatInt(time.Now().Unix(), 10)
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func outputTargetTokens(profileJSON string) int {
	if strings.TrimSpace(profileJSON) == "" {
		return 0
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(profileJSON), &decoded); err != nil {
		return 0
	}
	return intFromAny(decoded["target_tokens"])
}

func authHeader(token string) string {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return token
	}
	if strings.HasPrefix(token, "sk-") {
		return "Bearer " + token
	}
	return "Bearer sk-" + token
}

func chunkHasContent(event map[string]any) bool {
	choices, ok := event["choices"].([]any)
	if !ok {
		return false
	}
	for _, choice := range choices {
		choiceMap, ok := choice.(map[string]any)
		if !ok {
			continue
		}
		delta, ok := choiceMap["delta"].(map[string]any)
		if !ok {
			continue
		}
		if content, ok := delta["content"].(string); ok && content != "" {
			return true
		}
	}
	return false
}

func applyUsage(sample *probeSample, usage map[string]any) {
	sample.PromptTokens = intFromAny(usage["prompt_tokens"])
	sample.CompletionTokens = intFromAny(usage["completion_tokens"])
	sample.TotalTokens = intFromAny(usage["total_tokens"])
	if details, ok := usage["prompt_tokens_details"].(map[string]any); ok {
		sample.CachedTokens = intFromAny(details["cached_tokens"])
	}
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}

func summarizeSamples(plan slaProbePlan, stream bool, samples []probeSample) probeSummary {
	summary := probeSummary{
		SampleCount:        len(samples),
		Streaming:          stream,
		CacheProfile:       plan.CacheProfile,
		SessionPolicy:      sessionPolicy(plan.CacheProfile),
		FailureClassCounts: map[string]int{},
	}
	var firstByte, totalLatency, ttft []float64
	for _, sample := range samples {
		if sample.OK {
			summary.SuccessCount++
		} else {
			summary.FailureCount++
			if sample.FailureClass == "" {
				summary.FailureClassCounts["unknown"]++
			} else {
				summary.FailureClassCounts[sample.FailureClass]++
			}
		}
		if sample.FirstByteMs > 0 {
			firstByte = append(firstByte, float64(sample.FirstByteMs))
		}
		if sample.TotalLatencyMs > 0 {
			totalLatency = append(totalLatency, float64(sample.TotalLatencyMs))
		}
		if sample.TTFTObserved && sample.FirstTokenMs > 0 {
			summary.TTFTObserved = true
			ttft = append(ttft, float64(sample.FirstTokenMs))
		}
		summary.Usage.PromptTokens += sample.PromptTokens
		summary.Usage.CachedTokens += sample.CachedTokens
		summary.Usage.CompletionTokens += sample.CompletionTokens
		summary.Usage.TotalTokens += sample.TotalTokens
		if sample.CachedTokens > 0 {
			summary.Usage.CacheHitCount++
		}
	}
	if len(samples) > 0 {
		summary.SuccessRate = float64(summary.SuccessCount) / float64(len(samples))
	}
	summary.FirstByteMs = summarizeMetric(firstByte)
	summary.TotalLatencyMs = summarizeMetric(totalLatency)
	summary.TTFTMs = summarizeMetric(ttft)
	return summary
}

func sessionPolicy(cacheProfile string) string {
	switch strings.TrimSpace(cacheProfile) {
	case "warm_same_session":
		return "shared_session"
	case "mixed_trace_replay":
		return "trace_replay_sessions"
	default:
		return "unique_per_sample"
	}
}

func summarizeMetric(values []float64) *metricSummary {
	if len(values) == 0 {
		return nil
	}
	sort.Float64s(values)
	return &metricSummary{
		P50: percentile(values, 0.50),
		P90: percentile(values, 0.90),
		P95: percentile(values, 0.95),
		P99: percentile(values, 0.99),
		Max: values[len(values)-1],
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	pos := p * float64(len(sorted)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}
	weight := pos - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func evaluateHardGate(summary *probeSummary, hardGateJSON string) (bool, []string) {
	passed := summary.FailureCount == 0
	var reasons []string
	if summary.FailureCount > 0 {
		reasons = append(reasons, fmt.Sprintf("%d samples failed", summary.FailureCount))
	}
	gates := ttftGates(hardGateJSON)
	for _, gate := range gates {
		outcome := hardGateOutcome{Name: gate.name, LimitMs: gate.limitMs, Supported: true}
		if summary.TTFTMs == nil {
			outcome.Passed = false
			summary.HardGateEvaluations = append(summary.HardGateEvaluations, outcome)
			passed = false
			reasons = append(reasons, fmt.Sprintf("%s requires streaming TTFT samples", gate.name))
			continue
		}
		outcome.ActualMs = gate.value(summary.TTFTMs)
		outcome.Passed = outcome.ActualMs <= outcome.LimitMs
		summary.HardGateEvaluations = append(summary.HardGateEvaluations, outcome)
		if !outcome.Passed {
			passed = false
			reasons = append(reasons, fmt.Sprintf("%s %.2fms exceeds %.2fms", gate.name, outcome.ActualMs, outcome.LimitMs))
		}
	}
	return passed, reasons
}

type ttftGate struct {
	name    string
	limitMs float64
	value   func(*metricSummary) float64
}

func ttftGates(hardGateJSON string) []ttftGate {
	if strings.TrimSpace(hardGateJSON) == "" {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(hardGateJSON), &decoded); err != nil {
		return nil
	}
	var gates []ttftGate
	addGate := func(name string, limit any, value func(*metricSummary) float64) {
		limitMs := floatFromAny(limit)
		if limitMs > 0 {
			gates = append(gates, ttftGate{name: name, limitMs: limitMs, value: value})
		}
	}
	if ttft, ok := decoded["ttft_ms"].(map[string]any); ok {
		addGate("ttft_p90_lte", ttft["p90_lte"], func(m *metricSummary) float64 { return m.P90 })
		addGate("ttft_p95_lte", ttft["p95_lte"], func(m *metricSummary) float64 { return m.P95 })
		addGate("ttft_p99_lte", ttft["p99_lte"], func(m *metricSummary) float64 { return m.P99 })
	}
	addGate("ttft_p90_ms", decoded["ttft_p90_ms"], func(m *metricSummary) float64 { return m.P90 })
	addGate("ttft_p95_ms", decoded["ttft_p95_ms"], func(m *metricSummary) float64 { return m.P95 })
	addGate("ttft_p99_ms", decoded["ttft_p99_ms"], func(m *metricSummary) float64 { return m.P99 })
	return gates
}

func floatFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

func writeArtifact(path string, artifact probeArtifact) (string, string, error) {
	if strings.TrimSpace(path) == "" {
		return "", "", errors.New("artifact path is empty")
	}
	if filepath.Ext(path) == "" {
		path = filepath.Join(path, artifact.RunKey+".json")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", "", err
	}
	encoded, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(append(encoded, '\n'))
	return path, hex.EncodeToString(sum[:]), nil
}

func failureClassForStatus(status int) string {
	if status >= 500 {
		return "http_5xx"
	}
	if status >= 400 {
		return "http_4xx"
	}
	return "http_error"
}

func failureClassForError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "connect"
}

func millisSince(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
