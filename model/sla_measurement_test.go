package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func resetSlaMeasurementTables(t *testing.T) {
	t.Helper()
	truncateTables(t)
	for _, table := range []string{
		"sla_probe_runs",
		"sla_probe_plans",
		"sla_contracts",
		"abilities",
		"channels",
		"suppliers",
	} {
		require.NoError(t, DB.Exec("DELETE FROM "+table).Error)
	}
}

func seedSlaMeasurementSupplierChannel(t *testing.T) (*Supplier, *Channel) {
	t.Helper()
	supplier := &Supplier{Name: "gb10-4t-sla", Type: SupplierTypeThirdParty, Status: common.ChannelStatusEnabled}
	require.NoError(t, supplier.Insert())
	priority := int64(10)
	channel := &Channel{
		Id:         901,
		Type:       constant.ChannelTypeOpenAI,
		Key:        "sk-sla",
		Status:     common.ChannelStatusEnabled,
		Name:       "gb10-4t-sla",
		SupplierId: supplier.Id,
		Models:     "kimi-k2.5-test",
		Group:      "default",
		Priority:   &priority,
	}
	require.NoError(t, DB.Create(channel).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     "kimi-k2.5-test",
		ChannelId: channel.Id,
		Enabled:   true,
		Priority:  &priority,
		Weight:    100,
	}).Error)
	return supplier, channel
}

func slaMeasurementContractInput() SlaContractImportInput {
	return SlaContractImportInput{
		ContractKey:    "kimi-k25-official-v2026-test",
		ModelName:      "kimi-k2.5-test",
		ModelAliases:   "kimi-k2.5,kimi-k25",
		ProviderFamily: "kimi",
		SourceName:     "Kimi K2.5 serving requirements",
		SourceRef:      "contracts/kimi-k25-official.json",
		SourceSHA256:   "e3b0c44298fc1c149afbf4c8996fb924",
		Version:        "2026-06-23",
		Status:         SlaContractStatusActive,
		MeasurementProfileJSON: `{
			"input_profile":{"buckets":[{"name":"lt32k","max_tokens":32768}]},
			"output_profile":{"target_tokens":512},
			"concurrency_profile":{"concurrency":1},
			"rate_profile":{"rpm":30},
			"stream_profile":{"include_usage_required":true,"inter_packet_max_ms":500},
			"error_profile":{"max_error_rate":0.01},
			"availability_profile":{"window_seconds":600,"max_failure_rate":0.01},
			"cache_profile":"cold_no_cache"
		}`,
		HardGateJSON: `{"ttft_ms":{"p90_lte":8000},"otps":{"min":10}}`,
		SoftGateJSON: `{"warning_error_rate":0.005}`,
	}
}

func TestSlaMeasurementContractPlanAndRunLifecycle(t *testing.T) {
	resetSlaMeasurementTables(t)
	supplier, channel := seedSlaMeasurementSupplierChannel(t)

	_, err := ImportSlaContract(SlaContractImportInput{
		ContractKey:            "bad-json",
		ModelName:              "kimi-k2.5-test",
		ProviderFamily:         "kimi",
		SourceName:             "bad",
		SourceRef:              "bad",
		Version:                "bad",
		MeasurementProfileJSON: `{bad-json`,
	}, 1)
	require.ErrorContains(t, err, "measurement_profile_json must be valid JSON")

	contract, err := ImportSlaContract(slaMeasurementContractInput(), 1)
	require.NoError(t, err)
	require.Positive(t, contract.Id)
	require.Equal(t, SlaContractStatusActive, contract.Status)
	require.Equal(t, 1, contract.ImportedBy)
	require.Contains(t, contract.MeasurementProfileJSON, "input_profile")

	contractInput := slaMeasurementContractInput()
	contractInput.SourceSHA256 = "updated-sha"
	updatedContract, err := ImportSlaContract(contractInput, 2)
	require.NoError(t, err)
	require.Equal(t, contract.Id, updatedContract.Id)
	require.Equal(t, "updated-sha", updatedContract.SourceSHA256)
	require.Equal(t, 2, updatedContract.ImportedBy)

	contracts, total, err := SearchSlaContracts(SlaContractFilters{Status: SlaContractStatusActive, ProviderFamily: "kimi"}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, contracts, 1)
	require.Equal(t, updatedContract.Id, contracts[0].Id)

	plan, err := GenerateSlaProbePlan(SlaProbePlanGenerateInput{
		ContractId:     updatedContract.Id,
		SupplierId:     supplier.Id,
		ChannelId:      channel.Id,
		SlaTier:        "gold",
		ProbeType:      SlaProbeTypeAdmission,
		RouteMode:      SlaProbeRouteModeThroughTokenRouter,
		PromptSuiteKey: "kimi-admission-smoke",
		SampleSize:     3,
		RepeatCount:    2,
		MaxProbeQuota:  5000,
	}, 3)
	require.NoError(t, err)
	require.Positive(t, plan.Id)
	require.Equal(t, updatedContract.Id, plan.ContractId)
	require.Equal(t, supplier.Id, plan.SupplierId)
	require.Equal(t, channel.Id, plan.ChannelId)
	require.Equal(t, "gold", plan.SlaTier)
	require.Equal(t, SlaProbeRouteModeThroughTokenRouter, plan.RouteMode)
	require.Equal(t, "cold_no_cache", plan.CacheProfile)
	require.Contains(t, plan.InputProfileJSON, "lt32k")
	require.Contains(t, plan.OutputProfileJSON, "target_tokens")
	require.Contains(t, plan.StreamProfileJSON, "include_usage_required")
	require.Contains(t, plan.ErrorProfileJSON, "max_error_rate")
	require.Contains(t, plan.AvailabilityProfileJSON, "window_seconds")
	require.Equal(t, 3, plan.GeneratedBy)

	duplicatePlan, err := GenerateSlaProbePlan(SlaProbePlanGenerateInput{
		ContractId: updatedContract.Id,
		SupplierId: supplier.Id,
		ChannelId:  channel.Id,
		SlaTier:    "gold",
		ProbeType:  SlaProbeTypeAdmission,
		RouteMode:  SlaProbeRouteModeThroughTokenRouter,
		SampleSize: 4,
	}, 4)
	require.NoError(t, err)
	require.Equal(t, plan.Id, duplicatePlan.Id)
	require.Equal(t, 4, duplicatePlan.SampleSize)
	require.Equal(t, 4, duplicatePlan.GeneratedBy)

	plans, total, err := SearchSlaProbePlans(SlaProbePlanFilters{
		ContractId: updatedContract.Id,
		SupplierId: supplier.Id,
		ProbeType:  SlaProbeTypeAdmission,
		RouteMode:  SlaProbeRouteModeThroughTokenRouter,
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, plans, 1)
	require.Equal(t, duplicatePlan.Id, plans[0].Id)

	runKey := fmt.Sprintf("sla-run-%d", duplicatePlan.Id)
	run, err := RecordSlaProbeRun(SlaProbeRunRecordInput{
		RunKey:           runKey,
		PlanId:           duplicatePlan.Id,
		Status:           SlaProbeRunStatusPassed,
		StartedAt:        1000,
		EndedAt:          1300,
		RunnerVersion:    "token-router-sla-dev",
		GitCommit:        "abc123",
		RuntimeRef:       "aima2/mock",
		Endpoint:         "http://127.0.0.1:3000/v1/chat/completions",
		SummaryJSON:      `{"ttft_ms":{"p90":6200},"otps":{"p50":42},"cache":{"cold_samples":3}}`,
		HardGatePassed:   true,
		SoftGateWarnings: "[]",
		ArtifactURI:      "output/sla/run-1",
		ArtifactSHA256:   "4bf5122f344554c53bde2ebb8cd2b7e3",
	}, 5)
	require.NoError(t, err)
	require.Positive(t, run.Id)
	require.Equal(t, duplicatePlan.Id, run.PlanId)
	require.Equal(t, updatedContract.Id, run.ContractId)
	require.Equal(t, supplier.Id, run.SupplierId)
	require.Equal(t, channel.Id, run.ChannelId)
	require.Equal(t, SlaProbeRunStatusPassed, run.Status)
	require.True(t, run.HardGatePassed)
	require.Equal(t, "4bf5122f344554c53bde2ebb8cd2b7e3", run.ArtifactSHA256)
	require.Equal(t, 5, run.RecordedBy)

	updatedRun, err := RecordSlaProbeRun(SlaProbeRunRecordInput{
		RunKey:         runKey,
		PlanId:         duplicatePlan.Id,
		Status:         SlaProbeRunStatusFailed,
		StartedAt:      1000,
		EndedAt:        1300,
		SummaryJSON:    `{"failure_rate":0.2}`,
		FailureReasons: "timeout",
	}, 6)
	require.NoError(t, err)
	require.Equal(t, run.Id, updatedRun.Id)
	require.Equal(t, SlaProbeRunStatusFailed, updatedRun.Status)
	require.False(t, updatedRun.HardGatePassed)
	require.Equal(t, "timeout", updatedRun.FailureReasons)
	require.Equal(t, 6, updatedRun.RecordedBy)

	runs, total, err := SearchSlaProbeRuns(SlaProbeRunFilters{
		PlanId:     duplicatePlan.Id,
		ContractId: updatedContract.Id,
		SupplierId: supplier.Id,
		Status:     SlaProbeRunStatusFailed,
		RouteMode:  SlaProbeRouteModeThroughTokenRouter,
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, runs, 1)
	require.Equal(t, updatedRun.Id, runs[0].Id)

	_, err = RecordSlaProbeRun(SlaProbeRunRecordInput{
		PlanId:      duplicatePlan.Id,
		Status:      SlaProbeRunStatusPassed,
		StartedAt:   2000,
		EndedAt:     1000,
		SummaryJSON: `{}`,
	}, 1)
	require.ErrorContains(t, err, "ended_at must be greater")
}
