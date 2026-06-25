package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	supplyTelemetryWorkerDefaultInterval = 5 * time.Minute
	supplyTelemetryWorkerVersion         = "api-server"
)

type supplyTelemetryWorkerConfig struct {
	Enabled    bool
	Interval   time.Duration
	AgentKey   string
	Hostname   string
	RuntimeRef string
	Version    string
	SweepInput model.SupplyCapacityTelemetrySweepInput
}

var (
	supplyTelemetryWorkerOnce    sync.Once
	supplyTelemetryWorkerRunning atomic.Bool
)

func StartSupplyTelemetryWorker() {
	cfg, err := loadSupplyTelemetryWorkerConfigFromEnv()
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("supply telemetry worker config invalid: %v", err))
		return
	}
	startSupplyTelemetryWorkerWithConfig(cfg)
}

func startSupplyTelemetryWorkerWithConfig(cfg supplyTelemetryWorkerConfig) {
	if !cfg.Enabled || !common.IsMasterNode {
		return
	}
	supplyTelemetryWorkerOnce.Do(func() {
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("supply telemetry worker started: interval=%s agent_key=%s", cfg.Interval, cfg.AgentKey))
			ticker := time.NewTicker(cfg.Interval)
			defer ticker.Stop()

			runSupplyTelemetryWorkerOnce(cfg)
			for range ticker.C {
				runSupplyTelemetryWorkerOnce(cfg)
			}
		})
	})
}

func loadSupplyTelemetryWorkerConfigFromEnv() (supplyTelemetryWorkerConfig, error) {
	enabled := parseTruthyEnv(os.Getenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_ENABLED"))
	interval := supplyTelemetryWorkerDefaultInterval
	if rawInterval := strings.TrimSpace(os.Getenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_INTERVAL")); rawInterval != "" {
		parsed, err := time.ParseDuration(rawInterval)
		if err != nil {
			return supplyTelemetryWorkerConfig{}, fmt.Errorf("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_INTERVAL: %w", err)
		}
		if parsed <= 0 {
			return supplyTelemetryWorkerConfig{}, fmt.Errorf("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_INTERVAL must be positive")
		}
		interval = parsed
	}

	hostname := strings.TrimSpace(os.Getenv("HOSTNAME"))
	if hostname == "" {
		if osHostname, err := os.Hostname(); err == nil {
			hostname = strings.TrimSpace(osHostname)
		}
	}
	if hostname == "" {
		hostname = "localhost"
	}
	nodeName := strings.TrimSpace(common.NodeName)
	if nodeName == "" {
		nodeName = hostname
	}
	agentKey := strings.TrimSpace(os.Getenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_AGENT_KEY"))
	if agentKey == "" {
		agentKey = "api-server:" + nodeName + ":supply-telemetry-worker"
	}
	runtimeRef := strings.TrimSpace(os.Getenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_RUNTIME_REF"))
	if runtimeRef == "" {
		runtimeRef = "api-server"
	}
	cfg := supplyTelemetryWorkerConfig{
		Enabled:    enabled,
		Interval:   interval,
		AgentKey:   agentKey,
		Hostname:   hostname,
		RuntimeRef: runtimeRef,
		Version:    supplyTelemetryWorkerVersion,
	}

	var err error
	cfg.SweepInput.SupplierId, err = parseOptionalIntEnv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_SUPPLIER_ID")
	if err != nil {
		return supplyTelemetryWorkerConfig{}, err
	}
	cfg.SweepInput.ChannelId, err = parseOptionalIntEnv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_CHANNEL_ID")
	if err != nil {
		return supplyTelemetryWorkerConfig{}, err
	}
	cfg.SweepInput.SupplyNode = strings.TrimSpace(os.Getenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_SUPPLY_NODE"))
	cfg.SweepInput.ModelName = strings.TrimSpace(os.Getenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_MODEL"))
	cfg.SweepInput.PeriodStart, err = parseOptionalInt64Env("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_PERIOD_START")
	if err != nil {
		return supplyTelemetryWorkerConfig{}, err
	}
	cfg.SweepInput.PeriodEnd, err = parseOptionalInt64Env("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_PERIOD_END")
	if err != nil {
		return supplyTelemetryWorkerConfig{}, err
	}
	return cfg, nil
}

func runSupplyTelemetryWorkerOnce(cfg supplyTelemetryWorkerConfig) {
	if !supplyTelemetryWorkerRunning.CompareAndSwap(false, true) {
		return
	}
	defer supplyTelemetryWorkerRunning.Store(false)

	ctx := context.Background()
	agent, err := runSupplyTelemetryWorkerCycle(cfg)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("supply telemetry worker cycle failed: %v", err))
		return
	}
	if common.DebugEnabled && agent != nil {
		logger.LogDebug(ctx, "supply telemetry worker cycle recorded agent_id=%d status=%s sweep_status=%s collected=%d skipped=%d",
			agent.Id,
			agent.Status,
			agent.LastSweepStatus,
			agent.LastSweepCollectedCount,
			agent.LastSweepSkippedCount,
		)
	}
}

func runSupplyTelemetryWorkerCycle(cfg supplyTelemetryWorkerConfig) (*model.SupplyTelemetryAgent, error) {
	now := time.Now().Unix()
	if _, err := model.RecordSupplyTelemetryAgentHeartbeat(model.SupplyTelemetryAgentHeartbeatInput{
		AgentKey:    cfg.AgentKey,
		AgentType:   model.SupplyTelemetryAgentTypeTelemetry,
		Hostname:    cfg.Hostname,
		RuntimeRef:  cfg.RuntimeRef,
		Version:     cfg.Version,
		Status:      model.SupplyTelemetryAgentStatusActive,
		HeartbeatAt: now,
	}, 0); err != nil {
		return nil, err
	}

	startedAt := time.Now().Unix()
	result, sweepErr := model.SweepSupplyCapacityTelemetry(cfg.SweepInput, 0)
	finishedAt := time.Now().Unix()
	status := model.SupplyTelemetryAgentSweepStatusOK
	errorText := ""
	attemptedCount := 0
	collectedCount := 0
	skippedCount := 0
	if sweepErr != nil {
		status = model.SupplyTelemetryAgentSweepStatusFailed
		errorText = sweepErr.Error()
	} else if result != nil {
		attemptedCount = result.AttemptedCount
		collectedCount = result.CollectedCount
		skippedCount = result.SkippedCount
		if result.SkippedCount > 0 {
			status = model.SupplyTelemetryAgentSweepStatusSkipped
		}
	}
	agent, recordErr := model.RecordSupplyTelemetryAgentSweepResult(model.SupplyTelemetryAgentSweepResultInput{
		AgentKey:       cfg.AgentKey,
		AgentType:      model.SupplyTelemetryAgentTypeTelemetry,
		Hostname:       cfg.Hostname,
		RuntimeRef:     cfg.RuntimeRef,
		Version:        cfg.Version,
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
		Status:         status,
		Error:          errorText,
		AttemptedCount: attemptedCount,
		CollectedCount: collectedCount,
		SkippedCount:   skippedCount,
		SupplierId:     cfg.SweepInput.SupplierId,
		SupplyNode:     cfg.SweepInput.SupplyNode,
		ModelName:      cfg.SweepInput.ModelName,
		PeriodStart:    cfg.SweepInput.PeriodStart,
		PeriodEnd:      cfg.SweepInput.PeriodEnd,
	}, 0)
	if recordErr != nil {
		if sweepErr != nil {
			return nil, fmt.Errorf("%v; record sweep result: %w", sweepErr, recordErr)
		}
		return nil, recordErr
	}
	return agent, sweepErr
}

func parseTruthyEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func parseOptionalIntEnv(key string) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}

func parseOptionalInt64Env(key string) (int64, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}
