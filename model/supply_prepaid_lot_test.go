package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestRecordSupplyPrepaidLotRequiresSelfOperatedSupplier(t *testing.T) {
	truncateTables(t)

	now := common.GetTimestamp()
	thirdParty := &Supplier{Name: "gb10-4t-prepaid-third-party", Type: SupplierTypeThirdParty, Status: common.ChannelStatusEnabled}
	require.NoError(t, thirdParty.Insert())

	_, err := RecordSupplyPrepaidLot(SupplyPrepaidLotRecordInput{
		SupplierId:      thirdParty.Id,
		PeriodStart:     now - 3600,
		PeriodEnd:       now + 3600,
		PurchasedTokens: 1000,
		UnitCostQuota:   0.35,
		SourceType:      SupplyPrepaidLotSourceAccounting,
		SourceRef:       "accounting-prepaid-third-party",
		ObservedAt:      now,
	}, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "supplier must be self_operated")
}

func TestRecordAndRefreshSupplyPrepaidLotUsage(t *testing.T) {
	truncateTables(t)

	now := common.GetTimestamp()
	supplier := &Supplier{Name: "gb10-4t-prepaid-self-operated", Type: SupplierTypeSelfOperated, Status: common.ChannelStatusEnabled}
	require.NoError(t, supplier.Insert())

	lot, err := RecordSupplyPrepaidLot(SupplyPrepaidLotRecordInput{
		SupplierId:      supplier.Id,
		ChannelId:       31,
		SupplyNode:      "gb10-4t-prepaid",
		ModelName:       "gpt-prepaid",
		PeriodStart:     now - 3600,
		PeriodEnd:       now + 3600,
		PurchasedTokens: 1000,
		UnitCostQuota:   0.35,
		SourceType:      SupplyPrepaidLotSourceAccounting,
		SourceRef:       "offline-po-20260623",
		ObservedAt:      now,
		ExternalRef:     "po://20260623-gb10",
		Notes:           "offline self-operated prepaid batch",
	}, 1)
	require.NoError(t, err)
	require.Greater(t, lot.Id, 0)
	require.Contains(t, lot.PrepaidLotKey, "offline-po-20260623")
	require.Equal(t, supplier.Id, lot.SupplierId)
	require.Equal(t, int64(1000), lot.PurchasedTokens)
	require.InDelta(t, 350.0, lot.TotalCostQuota, 0.000001)
	require.Equal(t, int64(0), lot.DrawdownTokens)
	require.Equal(t, int64(1000), lot.RemainingTokens)
	require.Equal(t, SupplyPrepaidLotSourceAccounting, lot.SourceType)
	require.Equal(t, "po://20260623-gb10", lot.ExternalRef)
	require.Equal(t, 1, lot.RecordedBy)

	require.NoError(t, (&UsageLedger{
		RequestId:        "prepaid-drawdown-1",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		ModelName:        "gpt-prepaid",
		PromptTokens:     100,
		CompletionTokens: 40,
		Status:           "success",
		SupplyNode:       "gb10-4t-prepaid",
		CreatedAt:        now,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "prepaid-drawdown-2",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		ModelName:        "gpt-prepaid",
		PromptTokens:     120,
		CompletionTokens: 60,
		Status:           "success",
		SupplyNode:       "gb10-4t-prepaid",
		CreatedAt:        now + 1,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "prepaid-drawdown-failed",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		ModelName:        "gpt-prepaid",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "failed",
		SupplyNode:       "gb10-4t-prepaid",
		CreatedAt:        now + 2,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "prepaid-drawdown-other-channel",
		SupplierId:       supplier.Id,
		ChannelId:        32,
		ModelName:        "gpt-prepaid",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "success",
		SupplyNode:       "gb10-4t-prepaid",
		CreatedAt:        now + 3,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "prepaid-drawdown-other-model",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		ModelName:        "gpt-other",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "success",
		SupplyNode:       "gb10-4t-prepaid",
		CreatedAt:        now + 4,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "prepaid-drawdown-other-node",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		ModelName:        "gpt-prepaid",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "success",
		SupplyNode:       "gb10-4t-other",
		CreatedAt:        now + 5,
	}).InsertIdempotent())
	require.NoError(t, (&UsageLedger{
		RequestId:        "prepaid-drawdown-outside-window",
		SupplierId:       supplier.Id,
		ChannelId:        31,
		ModelName:        "gpt-prepaid",
		PromptTokens:     1000,
		CompletionTokens: 1000,
		Status:           "success",
		SupplyNode:       "gb10-4t-prepaid",
		CreatedAt:        now - 7200,
	}).InsertIdempotent())

	refreshed, err := RefreshSupplyPrepaidLotUsage(SupplyPrepaidLotUsageRefreshInput{PrepaidLotId: lot.Id})
	require.NoError(t, err)
	require.Len(t, refreshed, 1)
	require.Equal(t, int64(320), refreshed[0].DrawdownTokens)
	require.Equal(t, int64(2), refreshed[0].DrawdownRequestCount)
	require.Equal(t, int64(680), refreshed[0].RemainingTokens)
	require.InDelta(t, 0.32, refreshed[0].DrawdownRate, 0.000001)
	require.Equal(t, SupplyPrepaidLotDrawdownSourceUsageLedger, refreshed[0].DrawdownSourceType)
	require.Contains(t, refreshed[0].DrawdownSourceRef, "usage_ledger:prepaid_lot:")
	require.Greater(t, refreshed[0].DrawdownRefreshedAt, int64(0))

	queried, total, err := SearchSupplyPrepaidLots(SupplyPrepaidLotFilters{
		SupplierId: supplier.Id,
		SupplyNode: "gb10-4t-prepaid",
		ModelName:  "gpt-prepaid",
		SourceType: SupplyPrepaidLotSourceAccounting,
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, queried, 1)
	require.Equal(t, refreshed[0].Id, queried[0].Id)

	updated, err := RecordSupplyPrepaidLot(SupplyPrepaidLotRecordInput{
		PrepaidLotKey:   lot.PrepaidLotKey,
		SupplierId:      supplier.Id,
		ChannelId:       31,
		SupplyNode:      "gb10-4t-prepaid",
		ModelName:       "gpt-prepaid",
		PeriodStart:     now - 3600,
		PeriodEnd:       now + 3600,
		PurchasedTokens: 1200,
		UnitCostQuota:   0.34,
		SourceType:      SupplyPrepaidLotSourceAccounting,
		SourceRef:       "offline-po-20260623-updated",
		ObservedAt:      now + 10,
		ExternalRef:     "po://20260623-gb10-updated",
		Notes:           "updated offline prepaid batch",
	}, 2)
	require.NoError(t, err)
	require.Equal(t, lot.Id, updated.Id)
	require.Equal(t, int64(1200), updated.PurchasedTokens)
	require.InDelta(t, 408.0, updated.TotalCostQuota, 0.000001)
	require.Equal(t, int64(320), updated.DrawdownTokens)
	require.Equal(t, int64(880), updated.RemainingTokens)
	require.InDelta(t, float64(320)/float64(1200), updated.DrawdownRate, 0.000001)
	require.Equal(t, SupplyPrepaidLotDrawdownSourceUsageLedger, updated.DrawdownSourceType)
	require.Equal(t, "offline-po-20260623-updated", updated.SourceRef)
	require.Equal(t, "po://20260623-gb10-updated", updated.ExternalRef)
	require.Equal(t, 2, updated.RecordedBy)
}
