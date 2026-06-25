package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type telemetrySweepConfig struct {
	APIBase      string
	AdminToken   string
	SupplierId   int
	ChannelId    int
	SupplyNode   string
	ModelName    string
	PeriodStart  int64
	PeriodEnd    int64
	FailOnSkip   bool
	MinCollected int
	Timeout      time.Duration
}

type telemetryAgentConfig struct {
	telemetrySweepConfig
	AgentKey   string
	AgentType  string
	Hostname   string
	RuntimeRef string
	Version    string
	Interval   time.Duration
	Once       bool
}

type reviewOnceConfig struct {
	APIBase           string
	AdminToken        string
	SupplierId        int
	ModelName         string
	SlaTier           string
	UserId            int
	PeriodStart       int64
	PeriodEnd         int64
	TargetPeriodStart int64
	TargetPeriodEnd   int64
	MinGenerated      int
	Timeout           time.Duration
}

type reviewAgentConfig struct {
	reviewOnceConfig
	AgentKey      string
	Hostname      string
	RuntimeRef    string
	Version       string
	Interval      time.Duration
	Once          bool
	DynamicPeriod bool
}

type telemetrySweepInput struct {
	ChannelId   int    `json:"channel_id"`
	SupplierId  int    `json:"supplier_id"`
	SupplyNode  string `json:"supply_node"`
	ModelName   string `json:"model_name"`
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
}

type telemetrySweepSkip struct {
	CapacityId  int    `json:"capacity_id"`
	SupplierId  int    `json:"supplier_id"`
	SupplyNode  string `json:"supply_node"`
	ModelName   string `json:"model_name"`
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
	Reason      string `json:"reason"`
}

type telemetrySweepResult struct {
	AttemptedCount int                  `json:"attempted_count"`
	CollectedCount int                  `json:"collected_count"`
	SkippedCount   int                  `json:"skipped_count"`
	Collected      []json.RawMessage    `json:"collected"`
	Skipped        []telemetrySweepSkip `json:"skipped"`
}

type telemetryAgentHeartbeatInput struct {
	AgentKey    string `json:"agent_key"`
	AgentType   string `json:"agent_type"`
	Hostname    string `json:"hostname"`
	RuntimeRef  string `json:"runtime_ref"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	HeartbeatAt int64  `json:"heartbeat_at"`
}

type telemetryAgentSweepResultInput struct {
	AgentKey       string `json:"agent_key"`
	AgentType      string `json:"agent_type"`
	Hostname       string `json:"hostname"`
	RuntimeRef     string `json:"runtime_ref"`
	Version        string `json:"version"`
	StartedAt      int64  `json:"started_at"`
	FinishedAt     int64  `json:"finished_at"`
	Status         string `json:"status"`
	Error          string `json:"error"`
	AttemptedCount int    `json:"attempted_count"`
	CollectedCount int    `json:"collected_count"`
	SkippedCount   int    `json:"skipped_count"`
	SupplierId     int    `json:"supplier_id"`
	SupplyNode     string `json:"supply_node"`
	ModelName      string `json:"model_name"`
	PeriodStart    int64  `json:"period_start"`
	PeriodEnd      int64  `json:"period_end"`
}

type telemetryAgentCycleSummary struct {
	AgentKey   string               `json:"agent_key"`
	Status     string               `json:"status"`
	StartedAt  int64                `json:"started_at"`
	FinishedAt int64                `json:"finished_at"`
	Sweep      telemetrySweepResult `json:"sweep"`
	Error      string               `json:"error,omitempty"`
}

type reviewStepSummary struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Count int    `json:"count"`
}

type reviewOnceSummary struct {
	Status            string              `json:"status"`
	PeriodStart       int64               `json:"period_start"`
	PeriodEnd         int64               `json:"period_end"`
	TargetPeriodStart int64               `json:"target_period_start"`
	TargetPeriodEnd   int64               `json:"target_period_end"`
	SupplierId        int                 `json:"supplier_id,omitempty"`
	ModelName         string              `json:"model_name,omitempty"`
	SlaTier           string              `json:"sla_tier,omitempty"`
	UserId            int                 `json:"user_id,omitempty"`
	TotalGenerated    int                 `json:"total_generated"`
	Steps             []reviewStepSummary `json:"steps"`
}

type reviewAgentCycleSummary struct {
	AgentKey   string            `json:"agent_key"`
	Hostname   string            `json:"hostname"`
	RuntimeRef string            `json:"runtime_ref"`
	Version    string            `json:"version"`
	Status     string            `json:"status"`
	StartedAt  int64             `json:"started_at"`
	FinishedAt int64             `json:"finished_at"`
	Review     reviewOnceSummary `json:"review"`
	Error      string            `json:"error,omitempty"`
}

type apiEnvelope[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: token-router-supply <telemetry sweep|telemetry agent|review once|review agent> [options]")
	os.Exit(2)
}

func main() {
	if len(os.Args) < 3 {
		usage()
	}
	var err error
	switch {
	case os.Args[1] == "telemetry" && os.Args[2] == "sweep":
		err = runTelemetrySweepCLI(os.Args[3:])
	case os.Args[1] == "telemetry" && os.Args[2] == "agent":
		err = runTelemetryAgentCLI(os.Args[3:])
	case os.Args[1] == "review" && os.Args[2] == "once":
		err = runReviewOnceCLI(os.Args[3:])
	case os.Args[1] == "review" && os.Args[2] == "agent":
		err = runReviewAgentCLI(os.Args[3:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runReviewOnceCLI(args []string) error {
	cfg := reviewOnceConfig{}
	fs := flag.NewFlagSet("review once", flag.ExitOnError)
	fs.StringVar(&cfg.APIBase, "api", "http://127.0.0.1:19090", "token-router admin API base URL")
	fs.StringVar(&cfg.AdminToken, "admin-token", "", "admin access token for review generation")
	fs.IntVar(&cfg.SupplierId, "supplier-id", 0, "optional supplier id filter for scorecard/posture generation")
	fs.StringVar(&cfg.ModelName, "model", "", "optional model name filter")
	fs.StringVar(&cfg.SlaTier, "sla-tier", "", "optional SLA tier filter")
	fs.IntVar(&cfg.UserId, "user-id", 0, "optional user id filter")
	fs.Int64Var(&cfg.PeriodStart, "period-start", 0, "review source period start timestamp; defaults to now-1h")
	fs.Int64Var(&cfg.PeriodEnd, "period-end", 0, "review source period end timestamp; defaults to now")
	fs.Int64Var(&cfg.TargetPeriodStart, "target-period-start", 0, "optional forecast target period start")
	fs.Int64Var(&cfg.TargetPeriodEnd, "target-period-end", 0, "optional forecast target period end")
	fs.IntVar(&cfg.MinGenerated, "min-generated", 0, "return non-zero unless total generated rows is at least this value")
	fs.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "API request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := completeReviewOnceConfig(&cfg); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	summary, err := runReviewOnce(ctx, cfg)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return checkReviewOnceResult(summary, cfg)
}

func runReviewAgentCLI(args []string) error {
	cfg := reviewAgentConfig{}
	fs := flag.NewFlagSet("review agent", flag.ExitOnError)
	fs.StringVar(&cfg.APIBase, "api", "http://127.0.0.1:19090", "token-router admin API base URL")
	fs.StringVar(&cfg.AdminToken, "admin-token", "", "admin access token for review generation")
	fs.IntVar(&cfg.SupplierId, "supplier-id", 0, "optional supplier id filter for scorecard/posture generation")
	fs.StringVar(&cfg.ModelName, "model", "", "optional model name filter")
	fs.StringVar(&cfg.SlaTier, "sla-tier", "", "optional SLA tier filter")
	fs.IntVar(&cfg.UserId, "user-id", 0, "optional user id filter")
	fs.Int64Var(&cfg.PeriodStart, "period-start", 0, "review source period start timestamp; defaults to now-1h per cycle")
	fs.Int64Var(&cfg.PeriodEnd, "period-end", 0, "review source period end timestamp; defaults to now per cycle")
	fs.Int64Var(&cfg.TargetPeriodStart, "target-period-start", 0, "optional forecast target period start")
	fs.Int64Var(&cfg.TargetPeriodEnd, "target-period-end", 0, "optional forecast target period end")
	fs.IntVar(&cfg.MinGenerated, "min-generated", 0, "return non-zero in --once mode unless total generated rows is at least this value")
	fs.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "API request timeout")
	fs.StringVar(&cfg.AgentKey, "agent-key", "", "stable review agent identity; defaults to <hostname>:review")
	fs.StringVar(&cfg.Hostname, "hostname", "", "agent hostname; defaults to OS hostname")
	fs.StringVar(&cfg.RuntimeRef, "runtime-ref", "", "deployment runtime reference, e.g. systemd unit or pid")
	fs.StringVar(&cfg.Version, "version", "dev", "agent binary or deployment version")
	fs.DurationVar(&cfg.Interval, "interval", 15*time.Minute, "resident review interval")
	fs.BoolVar(&cfg.Once, "once", false, "run a single review cycle and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := completeReviewAgentConfig(&cfg); err != nil {
		return err
	}
	for {
		summary, err := runReviewAgentCycle(context.Background(), cfg)
		encoded, encodeErr := json.MarshalIndent(summary, "", "  ")
		if encodeErr != nil {
			return encodeErr
		}
		fmt.Println(string(encoded))
		if cfg.Once {
			return err
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		time.Sleep(cfg.Interval)
	}
}

func runTelemetrySweepCLI(args []string) error {
	cfg := telemetrySweepConfig{}
	fs := flag.NewFlagSet("telemetry sweep", flag.ExitOnError)
	fs.StringVar(&cfg.APIBase, "api", "http://127.0.0.1:19090", "token-router admin API base URL")
	fs.StringVar(&cfg.AdminToken, "admin-token", "", "admin access token for telemetry sweep")
	fs.IntVar(&cfg.SupplierId, "supplier-id", 0, "optional supplier id filter")
	fs.IntVar(&cfg.ChannelId, "channel-id", 0, "optional channel id override")
	fs.StringVar(&cfg.SupplyNode, "supply-node", "", "optional supply node filter")
	fs.StringVar(&cfg.ModelName, "model", "", "optional model name filter")
	fs.Int64Var(&cfg.PeriodStart, "period-start", 0, "optional period start timestamp filter")
	fs.Int64Var(&cfg.PeriodEnd, "period-end", 0, "optional period end timestamp filter")
	fs.BoolVar(&cfg.FailOnSkip, "fail-on-skip", false, "return non-zero when any capacity is skipped")
	fs.IntVar(&cfg.MinCollected, "min-collected", 0, "return non-zero unless at least this many telemetries are collected")
	fs.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "API request timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.AdminToken) == "" {
		return errors.New("--admin-token is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	result, err := sweepTelemetry(ctx, cfg)
	if err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return checkTelemetrySweepResult(result, cfg)
}

func completeReviewOnceConfig(cfg *reviewOnceConfig) error {
	if strings.TrimSpace(cfg.AdminToken) == "" {
		return errors.New("--admin-token is required")
	}
	if cfg.Timeout <= 0 {
		return errors.New("--timeout must be positive")
	}
	now := time.Now().Unix()
	if cfg.PeriodStart == 0 && cfg.PeriodEnd == 0 {
		cfg.PeriodEnd = now
		cfg.PeriodStart = now - int64(time.Hour/time.Second)
	}
	if cfg.PeriodStart <= 0 {
		return errors.New("--period-start is required when --period-end is set")
	}
	if cfg.PeriodEnd <= cfg.PeriodStart {
		return errors.New("--period-end must be greater than --period-start")
	}
	if cfg.TargetPeriodEnd > 0 && cfg.TargetPeriodStart <= 0 {
		return errors.New("--target-period-start is required when --target-period-end is set")
	}
	if cfg.TargetPeriodStart > 0 && cfg.TargetPeriodEnd <= cfg.TargetPeriodStart {
		return errors.New("--target-period-end must be greater than --target-period-start")
	}
	if cfg.MinGenerated < 0 {
		return errors.New("--min-generated cannot be negative")
	}
	cfg.APIBase = strings.TrimSpace(cfg.APIBase)
	cfg.ModelName = strings.TrimSpace(cfg.ModelName)
	cfg.SlaTier = strings.TrimSpace(cfg.SlaTier)
	return nil
}

func completeReviewAgentConfig(cfg *reviewAgentConfig) error {
	if strings.TrimSpace(cfg.AdminToken) == "" {
		return errors.New("--admin-token is required")
	}
	if cfg.Timeout <= 0 {
		return errors.New("--timeout must be positive")
	}
	if cfg.Interval <= 0 {
		return errors.New("--interval must be positive")
	}
	cfg.DynamicPeriod = cfg.PeriodStart == 0 && cfg.PeriodEnd == 0
	if !cfg.DynamicPeriod {
		if cfg.PeriodStart <= 0 {
			return errors.New("--period-start is required when --period-end is set")
		}
		if cfg.PeriodEnd <= cfg.PeriodStart {
			return errors.New("--period-end must be greater than --period-start")
		}
	}
	if cfg.TargetPeriodEnd > 0 && cfg.TargetPeriodStart <= 0 {
		return errors.New("--target-period-start is required when --target-period-end is set")
	}
	if cfg.TargetPeriodStart > 0 && cfg.TargetPeriodEnd <= cfg.TargetPeriodStart {
		return errors.New("--target-period-end must be greater than --target-period-start")
	}
	if cfg.MinGenerated < 0 {
		return errors.New("--min-generated cannot be negative")
	}
	cfg.APIBase = strings.TrimSpace(cfg.APIBase)
	cfg.ModelName = strings.TrimSpace(cfg.ModelName)
	cfg.SlaTier = strings.TrimSpace(cfg.SlaTier)
	if strings.TrimSpace(cfg.Hostname) == "" {
		hostname, err := os.Hostname()
		if err != nil || strings.TrimSpace(hostname) == "" {
			hostname = "localhost"
		}
		cfg.Hostname = hostname
	}
	cfg.Hostname = strings.TrimSpace(cfg.Hostname)
	cfg.AgentKey = strings.TrimSpace(cfg.AgentKey)
	if cfg.AgentKey == "" {
		cfg.AgentKey = cfg.Hostname + ":review"
	}
	cfg.RuntimeRef = strings.TrimSpace(cfg.RuntimeRef)
	if cfg.RuntimeRef == "" {
		cfg.RuntimeRef = fmt.Sprintf("pid:%d", os.Getpid())
	}
	cfg.Version = strings.TrimSpace(cfg.Version)
	return nil
}

func runTelemetryAgentCLI(args []string) error {
	cfg := telemetryAgentConfig{}
	fs := flag.NewFlagSet("telemetry agent", flag.ExitOnError)
	fs.StringVar(&cfg.APIBase, "api", "http://127.0.0.1:19090", "token-router admin API base URL")
	fs.StringVar(&cfg.AdminToken, "admin-token", "", "admin access token for telemetry agent")
	fs.IntVar(&cfg.SupplierId, "supplier-id", 0, "optional supplier id filter")
	fs.IntVar(&cfg.ChannelId, "channel-id", 0, "optional channel id override")
	fs.StringVar(&cfg.SupplyNode, "supply-node", "", "optional supply node filter")
	fs.StringVar(&cfg.ModelName, "model", "", "optional model name filter")
	fs.Int64Var(&cfg.PeriodStart, "period-start", 0, "optional period start timestamp filter")
	fs.Int64Var(&cfg.PeriodEnd, "period-end", 0, "optional period end timestamp filter")
	fs.BoolVar(&cfg.FailOnSkip, "fail-on-skip", false, "return non-zero in --once mode when any capacity is skipped")
	fs.IntVar(&cfg.MinCollected, "min-collected", 0, "return non-zero in --once mode unless at least this many telemetries are collected")
	fs.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "API request timeout")
	fs.StringVar(&cfg.AgentKey, "agent-key", "", "stable fleet agent identity; defaults to <hostname>:telemetry")
	fs.StringVar(&cfg.AgentType, "agent-type", "telemetry", "fleet agent type")
	fs.StringVar(&cfg.Hostname, "hostname", "", "agent hostname; defaults to OS hostname")
	fs.StringVar(&cfg.RuntimeRef, "runtime-ref", "", "deployment runtime reference, e.g. systemd unit or pid")
	fs.StringVar(&cfg.Version, "version", "dev", "agent binary or deployment version")
	fs.DurationVar(&cfg.Interval, "interval", 5*time.Minute, "resident sweep interval")
	fs.BoolVar(&cfg.Once, "once", false, "run a single heartbeat+sweep+sweep_result cycle and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := completeTelemetryAgentConfig(&cfg); err != nil {
		return err
	}
	for {
		summary, err := runTelemetryAgentCycle(context.Background(), cfg)
		if summary.AgentKey != "" {
			encoded, encodeErr := json.MarshalIndent(summary, "", "  ")
			if encodeErr != nil {
				return encodeErr
			}
			fmt.Println(string(encoded))
		}
		if cfg.Once {
			return err
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		time.Sleep(cfg.Interval)
	}
}

func runReviewOnce(ctx context.Context, cfg reviewOnceConfig) (reviewOnceSummary, error) {
	client := &http.Client{Timeout: cfg.Timeout}
	summary := reviewOnceSummary{
		Status:            "ok",
		PeriodStart:       cfg.PeriodStart,
		PeriodEnd:         cfg.PeriodEnd,
		TargetPeriodStart: cfg.TargetPeriodStart,
		TargetPeriodEnd:   cfg.TargetPeriodEnd,
		SupplierId:        cfg.SupplierId,
		ModelName:         strings.TrimSpace(cfg.ModelName),
		SlaTier:           strings.TrimSpace(cfg.SlaTier),
		UserId:            cfg.UserId,
	}
	steps := []struct {
		name    string
		path    string
		payload map[string]any
	}{
		{"supplier_scorecards", "/api/supplier_scorecards/generate", supplierReviewPayload(cfg)},
		{"supplier_posture_recommendations", "/api/supplier_posture_recommendations/generate", supplierReviewPayload(cfg)},
		{"traffic_profiles", "/api/traffic_profiles/generate", trafficReviewPayload(cfg)},
		{"traffic_forecasts", "/api/traffic_forecasts/generate", forecastReviewPayload(cfg)},
		{"pricing_recommendations", "/api/pricing_recommendations/generate", trafficReviewPayload(cfg)},
		{"supply_decisions", "/api/supply_decisions/generate", trafficReviewPayload(cfg)},
		{"supply_expansion_opportunities", "/api/supply_expansion_opportunities/generate", trafficReviewPayload(cfg)},
		{"operating_insights", "/api/operating_insights/generate", trafficReviewPayload(cfg)},
	}
	for _, step := range steps {
		items, err := postAPI[[]json.RawMessage](ctx, client, cfg.APIBase, cfg.AdminToken, step.path, step.payload)
		if err != nil {
			return summary, fmt.Errorf("%s: %w", step.name, err)
		}
		count := len(items)
		summary.TotalGenerated += count
		summary.Steps = append(summary.Steps, reviewStepSummary{
			Name:  step.name,
			Path:  step.path,
			Count: count,
		})
	}
	return summary, nil
}

func runReviewAgentCycle(ctx context.Context, cfg reviewAgentConfig) (reviewAgentCycleSummary, error) {
	reviewCfg := cfg.reviewOnceConfig
	if cfg.DynamicPeriod || (reviewCfg.PeriodStart == 0 && reviewCfg.PeriodEnd == 0) {
		now := time.Now().Unix()
		reviewCfg.PeriodEnd = now
		reviewCfg.PeriodStart = now - int64(time.Hour/time.Second)
	}
	startedAt := time.Now().Unix()
	cycleCtx, cancel := context.WithTimeout(ctx, reviewCfg.Timeout)
	defer cancel()
	review, err := runReviewOnce(cycleCtx, reviewCfg)
	finishedAt := time.Now().Unix()
	status := "ok"
	errorText := ""
	cycleErr := err
	if err != nil {
		status = "failed"
		errorText = err.Error()
		review.Status = "failed"
	} else if gateErr := checkReviewOnceResult(review, reviewCfg); gateErr != nil {
		status = "failed"
		errorText = gateErr.Error()
		cycleErr = gateErr
		review.Status = "failed"
	}
	return reviewAgentCycleSummary{
		AgentKey:   cfg.AgentKey,
		Hostname:   cfg.Hostname,
		RuntimeRef: cfg.RuntimeRef,
		Version:    cfg.Version,
		Status:     status,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Review:     review,
		Error:      errorText,
	}, cycleErr
}

func supplierReviewPayload(cfg reviewOnceConfig) map[string]any {
	payload := map[string]any{
		"period_start": cfg.PeriodStart,
		"period_end":   cfg.PeriodEnd,
	}
	if cfg.SupplierId > 0 {
		payload["supplier_id"] = cfg.SupplierId
	}
	return payload
}

func trafficReviewPayload(cfg reviewOnceConfig) map[string]any {
	payload := map[string]any{
		"period_start": cfg.PeriodStart,
		"period_end":   cfg.PeriodEnd,
	}
	if strings.TrimSpace(cfg.ModelName) != "" {
		payload["model_name"] = strings.TrimSpace(cfg.ModelName)
	}
	if strings.TrimSpace(cfg.SlaTier) != "" {
		payload["sla_tier"] = strings.TrimSpace(cfg.SlaTier)
	}
	if cfg.UserId > 0 {
		payload["user_id"] = cfg.UserId
	}
	return payload
}

func forecastReviewPayload(cfg reviewOnceConfig) map[string]any {
	payload := trafficReviewPayload(cfg)
	if cfg.TargetPeriodStart > 0 {
		payload["target_period_start"] = cfg.TargetPeriodStart
	}
	if cfg.TargetPeriodEnd > 0 {
		payload["target_period_end"] = cfg.TargetPeriodEnd
	}
	return payload
}

func sweepTelemetry(ctx context.Context, cfg telemetrySweepConfig) (telemetrySweepResult, error) {
	input := telemetrySweepInput{
		ChannelId:   cfg.ChannelId,
		SupplierId:  cfg.SupplierId,
		SupplyNode:  strings.TrimSpace(cfg.SupplyNode),
		ModelName:   strings.TrimSpace(cfg.ModelName),
		PeriodStart: cfg.PeriodStart,
		PeriodEnd:   cfg.PeriodEnd,
	}
	client := &http.Client{Timeout: cfg.Timeout}
	return postAPI[telemetrySweepResult](ctx, client, cfg.APIBase, cfg.AdminToken, "/api/supply_capacity_telemetries/sweep", input)
}

func completeTelemetryAgentConfig(cfg *telemetryAgentConfig) error {
	if strings.TrimSpace(cfg.AdminToken) == "" {
		return errors.New("--admin-token is required")
	}
	if cfg.Timeout <= 0 {
		return errors.New("--timeout must be positive")
	}
	if cfg.Interval <= 0 {
		return errors.New("--interval must be positive")
	}
	if strings.TrimSpace(cfg.Hostname) == "" {
		hostname, err := os.Hostname()
		if err != nil || strings.TrimSpace(hostname) == "" {
			hostname = "localhost"
		}
		cfg.Hostname = hostname
	}
	cfg.Hostname = strings.TrimSpace(cfg.Hostname)
	cfg.AgentType = strings.TrimSpace(cfg.AgentType)
	if cfg.AgentType == "" {
		cfg.AgentType = "telemetry"
	}
	cfg.AgentKey = strings.TrimSpace(cfg.AgentKey)
	if cfg.AgentKey == "" {
		cfg.AgentKey = cfg.Hostname + ":telemetry"
	}
	cfg.RuntimeRef = strings.TrimSpace(cfg.RuntimeRef)
	if cfg.RuntimeRef == "" {
		cfg.RuntimeRef = fmt.Sprintf("pid:%d", os.Getpid())
	}
	cfg.Version = strings.TrimSpace(cfg.Version)
	return nil
}

func runTelemetryAgentCycle(ctx context.Context, cfg telemetryAgentConfig) (telemetryAgentCycleSummary, error) {
	client := &http.Client{Timeout: cfg.Timeout}
	now := time.Now().Unix()
	if _, err := postAPI[json.RawMessage](ctx, client, cfg.APIBase, cfg.AdminToken, "/api/supply_telemetry_agents/heartbeat", telemetryAgentHeartbeatInput{
		AgentKey:    cfg.AgentKey,
		AgentType:   cfg.AgentType,
		Hostname:    cfg.Hostname,
		RuntimeRef:  cfg.RuntimeRef,
		Version:     cfg.Version,
		Status:      "active",
		HeartbeatAt: now,
	}); err != nil {
		return telemetryAgentCycleSummary{}, err
	}

	startedAt := time.Now().Unix()
	result, sweepErr := sweepTelemetry(ctx, cfg.telemetrySweepConfig)
	finishedAt := time.Now().Unix()
	status := "ok"
	errorText := ""
	cycleErr := sweepErr
	if sweepErr != nil {
		status = "failed"
		errorText = sweepErr.Error()
	} else {
		if result.SkippedCount > 0 {
			status = "skipped"
		}
		if gateErr := checkTelemetrySweepResult(result, cfg.telemetrySweepConfig); gateErr != nil {
			cycleErr = gateErr
			errorText = gateErr.Error()
			if result.SkippedCount == 0 {
				status = "failed"
			}
		}
	}

	summary := telemetryAgentCycleSummary{
		AgentKey:   cfg.AgentKey,
		Status:     status,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Sweep:      result,
		Error:      errorText,
	}
	if _, recordErr := postAPI[json.RawMessage](ctx, client, cfg.APIBase, cfg.AdminToken, "/api/supply_telemetry_agents/sweep_result", telemetryAgentSweepResultInput{
		AgentKey:       cfg.AgentKey,
		AgentType:      cfg.AgentType,
		Hostname:       cfg.Hostname,
		RuntimeRef:     cfg.RuntimeRef,
		Version:        cfg.Version,
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
		Status:         status,
		Error:          errorText,
		AttemptedCount: result.AttemptedCount,
		CollectedCount: result.CollectedCount,
		SkippedCount:   result.SkippedCount,
		SupplierId:     cfg.SupplierId,
		SupplyNode:     strings.TrimSpace(cfg.SupplyNode),
		ModelName:      strings.TrimSpace(cfg.ModelName),
		PeriodStart:    cfg.PeriodStart,
		PeriodEnd:      cfg.PeriodEnd,
	}); recordErr != nil {
		if cycleErr != nil {
			return summary, fmt.Errorf("%v; record sweep result: %w", cycleErr, recordErr)
		}
		return summary, recordErr
	}
	return summary, cycleErr
}

func checkTelemetrySweepResult(result telemetrySweepResult, cfg telemetrySweepConfig) error {
	if cfg.MinCollected > 0 && result.CollectedCount < cfg.MinCollected {
		return fmt.Errorf("telemetry sweep collected %d rows, below --min-collected=%d", result.CollectedCount, cfg.MinCollected)
	}
	if cfg.FailOnSkip && result.SkippedCount > 0 {
		return fmt.Errorf("telemetry sweep skipped %d capacities", result.SkippedCount)
	}
	return nil
}

func checkReviewOnceResult(summary reviewOnceSummary, cfg reviewOnceConfig) error {
	if cfg.MinGenerated > 0 && summary.TotalGenerated < cfg.MinGenerated {
		return fmt.Errorf("review generated %d rows, below --min-generated=%d", summary.TotalGenerated, cfg.MinGenerated)
	}
	return nil
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
