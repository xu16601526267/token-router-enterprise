package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestRefreshSupplyActionExecutionUsageFromLedger(t *testing.T) {
	truncateTables(t)

	now := common.GetTimestamp()
	supplier := &Supplier{Name: "gb10-4t-execution-drawdown", Type: SupplierTypeSelfHosted, Status: common.ChannelStatusEnabled}
	require.NoError(t, supplier.Insert())
	capacity := &SupplyCapacity{
		SupplierId:     supplier.Id,
		SupplyNode:     "gb10-4t-execution-drawdown",
		ModelName:      "gpt-execution-drawdown",
		PeriodStart:    now - 3600,
		PeriodEnd:      now + 3600,
		CapacityTokens: 1000,
		UsedTokens:     100,
		QualityScore:   99,
		UnitCostQuota:  0.4,
		Status:         1,
	}
	require.NoError(t, capacity.Insert())

	execution := &SupplyActionExecution{
		SupplyActionPlanId:   1001,
		SupplyDecisionId:     1002,
		DecisionKey:          "execution-drawdown",
		TrafficProfileId:     1003,
		SliceKey:             "execution-drawdown-slice",
		ModelName:            "gpt-execution-drawdown",
		SlaTier:              "premium",
		UserId:               7,
		PeriodStart:          now - 3600,
		PeriodEnd:            now + 3600,
		DecisionType:         SupplyDecisionTypeSelfHostedEvaluate,
		Track:                SupplyDecisionTrackSelfHosted,
		ActionType:           SupplyActionTypeEvaluateSelfHostedCapacity,
		ExecutionStatus:      SupplyActionExecutionStatusRecorded,
		SupplierId:           supplier.Id,
		ChannelId:            31,
		SupplyCapacityId:     capacity.Id,
		RecommendedCapacity:  900,
		ActualCapacityTokens: 1000,
		DrawdownTokens:       999,
		RemainingTokens:      1,
		DrawdownRate:         0.999,
		EffectiveFrom:        now - 1800,
		EffectiveTo:          now + 1800,
		RecordedAt:           now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	require.NoError(t, DB.Create(execution).Error)

	require.NoError(t, (&UsageLedger{
		RequestId:        "execution-drawdown-1",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		UserId:           7,
		ModelName:        "gpt-execution-drawdown",
		PromptTokens:     100,
		CompletionTokens: 40,
		Status:           "success",
		SlaTier:          "premium",
		SupplyNode:       "gb10-4t-execution-drawdown",
		CreatedAt:        now,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "execution-drawdown-2",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		UserId:           0,
		ModelName:        "gpt-execution-drawdown",
		PromptTokens:     120,
		CompletionTokens: 60,
		Status:           "success",
		SlaTier:          "",
		SupplyNode:       "gb10-4t-execution-drawdown",
		CreatedAt:        now + 1,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "execution-drawdown-failed",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		UserId:           7,
		ModelName:        "gpt-execution-drawdown",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "failed",
		SlaTier:          "premium",
		SupplyNode:       "gb10-4t-execution-drawdown",
		CreatedAt:        now + 2,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "execution-drawdown-other-channel",
		SupplierId:       supplier.Id,
		ChannelId:        32,
		UserId:           7,
		ModelName:        "gpt-execution-drawdown",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "success",
		SlaTier:          "premium",
		SupplyNode:       "gb10-4t-execution-drawdown",
		CreatedAt:        now + 3,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "execution-drawdown-other-node",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		UserId:           7,
		ModelName:        "gpt-execution-drawdown",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "success",
		SlaTier:          "premium",
		SupplyNode:       "gb10-4t-other",
		CreatedAt:        now + 4,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "execution-drawdown-other-user",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		UserId:           8,
		ModelName:        "gpt-execution-drawdown",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "success",
		SlaTier:          "premium",
		SupplyNode:       "gb10-4t-execution-drawdown",
		CreatedAt:        now + 5,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "execution-drawdown-other-sla",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		UserId:           7,
		ModelName:        "gpt-execution-drawdown",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "success",
		SlaTier:          "default",
		SupplyNode:       "gb10-4t-execution-drawdown",
		CreatedAt:        now + 6,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "execution-drawdown-outside-window",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		UserId:           7,
		ModelName:        "gpt-execution-drawdown",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "success",
		SlaTier:          "premium",
		SupplyNode:       "gb10-4t-execution-drawdown",
		CreatedAt:        now - 1900,
	}).InsertIdempotent())

	refreshed, err := RefreshSupplyActionExecutionUsage(SupplyActionExecutionUsageRefreshInput{
		ExecutionId: execution.Id,
	})
	require.NoError(t, err)
	require.Len(t, refreshed, 1)
	require.Equal(t, int64(320), refreshed[0].DrawdownTokens)
	require.Equal(t, int64(2), refreshed[0].DrawdownRequestCount)
	require.Equal(t, int64(680), refreshed[0].RemainingTokens)
	require.InDelta(t, 0.32, refreshed[0].DrawdownRate, 0.000001)
	require.Equal(t, SupplyActionExecutionDrawdownSourceUsageLedger, refreshed[0].DrawdownSourceType)
	require.Contains(t, refreshed[0].DrawdownSourceRef, "usage_ledger:execution:")
	require.Greater(t, refreshed[0].DrawdownRefreshedAt, int64(0))

	saved, _, err := SearchSupplyActionExecutions(SupplyActionExecutionFilters{ExecutionId: execution.Id}, 0, 10)
	require.NoError(t, err)
	require.Len(t, saved, 1)
	require.Equal(t, int64(320), saved[0].DrawdownTokens)
	require.Equal(t, int64(680), saved[0].RemainingTokens)
	require.Equal(t, SupplyActionExecutionDrawdownSourceUsageLedger, saved[0].DrawdownSourceType)
}
