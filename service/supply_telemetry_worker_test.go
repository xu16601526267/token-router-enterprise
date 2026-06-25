package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestSupplyTelemetryWorkerConfigDisabledByDefault(t *testing.T) {
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_ENABLED", "")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_INTERVAL", "")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_AGENT_KEY", "")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_SUPPLIER_ID", "")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_CHANNEL_ID", "")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_SUPPLY_NODE", "")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_MODEL", "")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_PERIOD_START", "")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_PERIOD_END", "")

	cfg, err := loadSupplyTelemetryWorkerConfigFromEnv()
	require.NoError(t, err)
	require.False(t, cfg.Enabled)
	require.Equal(t, supplyTelemetryWorkerDefaultInterval, cfg.Interval)
	require.Contains(t, cfg.AgentKey, ":supply-telemetry-worker")
}

func TestSupplyTelemetryWorkerConfigParsesFilters(t *testing.T) {
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_ENABLED", "true")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_INTERVAL", "2m")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_AGENT_KEY", "api-server:test")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_SUPPLIER_ID", "11")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_CHANNEL_ID", "12")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_SUPPLY_NODE", " gb10-4t ")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_MODEL", " gpt-test ")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_PERIOD_START", "100")
	t.Setenv("TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_PERIOD_END", "200")

	cfg, err := loadSupplyTelemetryWorkerConfigFromEnv()
	require.NoError(t, err)
	require.True(t, cfg.Enabled)
	require.Equal(t, 2*time.Minute, cfg.Interval)
	require.Equal(t, "api-server:test", cfg.AgentKey)
	require.Equal(t, 11, cfg.SweepInput.SupplierId)
	require.Equal(t, 12, cfg.SweepInput.ChannelId)
	require.Equal(t, "gb10-4t", cfg.SweepInput.SupplyNode)
	require.Equal(t, "gpt-test", cfg.SweepInput.ModelName)
	require.Equal(t, int64(100), cfg.SweepInput.PeriodStart)
	require.Equal(t, int64(200), cfg.SweepInput.PeriodEnd)
}

func TestSupplyTelemetryWorkerCycleRecordsAgentAndTelemetry(t *testing.T) {
	truncate(t)

	const periodStart = int64(100)
	const periodEnd = int64(200)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != model.SupplyCapacityTelemetryCollectPath {
			http.NotFound(w, r)
			return
		}
		require.Equal(t, "gb10-4t", r.URL.Query().Get("supply_node"))
		require.Equal(t, "gpt-test", r.URL.Query().Get("model"))
		require.Equal(t, "Bearer sk-gb10-4t-mock", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"supply_node":          "gb10-4t",
			"model_name":           "gpt-test",
			"capacity_tokens":      1000,
			"used_tokens":          320,
			"gpu_utilization_rate": 0.62,
			"quality_score":        98.5,
			"unit_cost_quota":      0.5,
			"observed_at":          150,
			"source_ref":           "worker-gb10-4t-capacity",
			"notes":                "worker telemetry test",
		}))
	}))
	defer upstream.Close()

	require.NoError(t, model.DB.Create(&model.Supplier{
		Id:     1,
		Name:   "gb10-4t",
		Type:   model.SupplierTypeThirdParty,
		Status: common.ChannelStatusEnabled,
	}).Error)
	baseURL := upstream.URL
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:         1,
		Type:       constant.ChannelTypeOpenAI,
		Key:        "sk-gb10-4t-mock",
		Status:     common.ChannelStatusEnabled,
		Name:       "gb10-4t",
		BaseURL:    &baseURL,
		Models:     "gpt-test",
		Group:      "default",
		SupplierId: 1,
	}).Error)
	require.NoError(t, model.DB.Create(&model.SupplyCapacity{
		Id:             1,
		SupplierId:     1,
		SupplyNode:     "gb10-4t",
		ModelName:      "gpt-test",
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		CapacityTokens: 1000,
		Status:         common.ChannelStatusEnabled,
	}).Error)

	agent, err := runSupplyTelemetryWorkerCycle(supplyTelemetryWorkerConfig{
		AgentKey:   "api-server:test-worker",
		Hostname:   "api-host",
		RuntimeRef: "api-server-test",
		Version:    "test",
		SweepInput: model.SupplyCapacityTelemetrySweepInput{
			SupplierId: 1,
			SupplyNode: "gb10-4t",
			ModelName:  "gpt-test",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "api-server:test-worker", agent.AgentKey)
	require.Equal(t, model.SupplyTelemetryAgentStatusActive, agent.Status)
	require.Equal(t, model.SupplyTelemetryAgentSweepStatusOK, agent.LastSweepStatus)
	require.Equal(t, 1, agent.LastSweepAttemptedCount)
	require.Equal(t, 1, agent.LastSweepCollectedCount)
	require.Equal(t, 0, agent.LastSweepSkippedCount)
	require.Equal(t, "gb10-4t", agent.LastSweepSupplyNode)
	require.Equal(t, "gpt-test", agent.LastSweepModelName)

	var telemetry model.SupplyCapacityTelemetry
	require.NoError(t, model.DB.First(&telemetry).Error)
	require.Equal(t, "worker-gb10-4t-capacity", telemetry.SourceRef)
	require.Equal(t, int64(320), telemetry.UsedTokens)
	require.Equal(t, 0.62, telemetry.GpuUtilizationRate)
	require.Equal(t, 1, telemetry.AppliedCapacityId)

	var capacity model.SupplyCapacity
	require.NoError(t, model.DB.First(&capacity, 1).Error)
	require.Equal(t, "worker-gb10-4t-capacity", capacity.TelemetrySourceRef)
	require.Equal(t, telemetry.Id, capacity.LastTelemetryId)
	require.Equal(t, int64(680), capacity.HeadroomTokens)
}

func TestSupplyTelemetryWorkerCycleRecordsFailedSweep(t *testing.T) {
	truncate(t)

	agent, err := runSupplyTelemetryWorkerCycle(supplyTelemetryWorkerConfig{
		AgentKey:   "api-server:test-worker-failed",
		Hostname:   "api-host",
		RuntimeRef: "api-server-test",
		Version:    "test",
		SweepInput: model.SupplyCapacityTelemetrySweepInput{
			PeriodEnd: 200,
		},
	})
	require.Error(t, err)
	require.Equal(t, model.SupplyTelemetryAgentStatusError, agent.Status)
	require.Equal(t, model.SupplyTelemetryAgentSweepStatusFailed, agent.LastSweepStatus)
	require.Contains(t, agent.LastSweepError, "period_start is required")
}
