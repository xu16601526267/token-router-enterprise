package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecordSupplyTelemetryAgentHeartbeatUpsertsIdentity(t *testing.T) {
	truncateTables(t)

	first, err := RecordSupplyTelemetryAgentHeartbeat(SupplyTelemetryAgentHeartbeatInput{
		AgentKey:    " aima2:telemetry ",
		Hostname:    " aima2 ",
		RuntimeRef:  " systemd:token-router-supply ",
		Version:     " v1 ",
		HeartbeatAt: 100,
	}, 7)
	require.NoError(t, err)
	require.Equal(t, "aima2:telemetry", first.AgentKey)
	require.Equal(t, SupplyTelemetryAgentTypeTelemetry, first.AgentType)
	require.Equal(t, "aima2", first.Hostname)
	require.Equal(t, "systemd:token-router-supply", first.RuntimeRef)
	require.Equal(t, "v1", first.Version)
	require.Equal(t, SupplyTelemetryAgentStatusActive, first.Status)
	require.Equal(t, int64(100), first.LastHeartbeatAt)
	require.Equal(t, 7, first.RecordedBy)

	second, err := RecordSupplyTelemetryAgentHeartbeat(SupplyTelemetryAgentHeartbeatInput{
		AgentKey:    "aima2:telemetry",
		AgentType:   "telemetry",
		Hostname:    "aima2-new",
		RuntimeRef:  "pid:42",
		Version:     "v2",
		HeartbeatAt: 200,
	}, 8)
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)
	require.Equal(t, "aima2-new", second.Hostname)
	require.Equal(t, "pid:42", second.RuntimeRef)
	require.Equal(t, "v2", second.Version)
	require.Equal(t, int64(200), second.LastHeartbeatAt)
	require.Equal(t, 8, second.RecordedBy)

	var count int64
	require.NoError(t, DB.Model(&SupplyTelemetryAgent{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestRecordSupplyTelemetryAgentSweepResultStoresLastRunSummary(t *testing.T) {
	truncateTables(t)

	agent, err := RecordSupplyTelemetryAgentSweepResult(SupplyTelemetryAgentSweepResultInput{
		AgentKey:       "aima2:telemetry",
		Hostname:       "aima2",
		RuntimeRef:     "pid:42",
		Version:        "v2",
		StartedAt:      100,
		FinishedAt:     110,
		Status:         SupplyTelemetryAgentSweepStatusSkipped,
		AttemptedCount: 3,
		CollectedCount: 2,
		SkippedCount:   1,
		SupplierId:     11,
		SupplyNode:     " gb10-4t ",
		ModelName:      " gpt-test ",
		PeriodStart:    10,
		PeriodEnd:      20,
	}, 7)
	require.NoError(t, err)
	require.Equal(t, SupplyTelemetryAgentStatusActive, agent.Status)
	require.Equal(t, SupplyTelemetryAgentSweepStatusSkipped, agent.LastSweepStatus)
	require.Equal(t, 3, agent.LastSweepAttemptedCount)
	require.Equal(t, 2, agent.LastSweepCollectedCount)
	require.Equal(t, 1, agent.LastSweepSkippedCount)
	require.Equal(t, 11, agent.LastSweepSupplierId)
	require.Equal(t, "gb10-4t", agent.LastSweepSupplyNode)
	require.Equal(t, "gpt-test", agent.LastSweepModelName)
	require.Equal(t, int64(10), agent.LastSweepPeriodStart)
	require.Equal(t, int64(20), agent.LastSweepPeriodEnd)
	require.Equal(t, int64(110), agent.LastHeartbeatAt)

	failed, err := RecordSupplyTelemetryAgentSweepResult(SupplyTelemetryAgentSweepResultInput{
		AgentKey:   "aima2:telemetry",
		StartedAt:  200,
		FinishedAt: 205,
		Status:     SupplyTelemetryAgentSweepStatusFailed,
		Error:      "POST /api/supply_capacity_telemetries/sweep returned status 500",
	}, 9)
	require.NoError(t, err)
	require.Equal(t, agent.Id, failed.Id)
	require.Equal(t, SupplyTelemetryAgentStatusError, failed.Status)
	require.Equal(t, SupplyTelemetryAgentSweepStatusFailed, failed.LastSweepStatus)
	require.Contains(t, failed.LastSweepError, "status 500")
	require.Equal(t, int64(205), failed.LastHeartbeatAt)
	require.Equal(t, 9, failed.RecordedBy)
}

func TestSearchSupplyTelemetryAgentsFiltersStaleRows(t *testing.T) {
	truncateTables(t)

	_, err := RecordSupplyTelemetryAgentHeartbeat(SupplyTelemetryAgentHeartbeatInput{
		AgentKey:    "fresh",
		HeartbeatAt: 300,
	}, 1)
	require.NoError(t, err)
	_, err = RecordSupplyTelemetryAgentHeartbeat(SupplyTelemetryAgentHeartbeatInput{
		AgentKey:    "stale",
		HeartbeatAt: 100,
	}, 1)
	require.NoError(t, err)

	agents, total, err := SearchSupplyTelemetryAgents(SupplyTelemetryAgentFilters{StaleBefore: 200}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, agents, 1)
	require.Equal(t, "stale", agents[0].AgentKey)
}
