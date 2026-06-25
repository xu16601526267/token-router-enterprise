package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestRecordSupplyCostProfileComputesAmortizedUnitCostAndUpserts(t *testing.T) {
	truncateTables(t)

	supplier := &Supplier{Name: "gb10-4t-self-hosted-cost", Type: SupplierTypeSelfHosted, Status: common.ChannelStatusEnabled}
	require.NoError(t, supplier.Insert())

	first, err := RecordSupplyCostProfile(SupplyCostProfileRecordInput{
		SupplierId:            supplier.Id,
		SupplyNode:            "gb10-4t-self-hosted",
		ModelName:             "gpt-cost-profile",
		PeriodStart:           1000,
		PeriodEnd:             2000,
		CapacityTokens:        1000,
		FixedCostQuota:        100,
		VariableUnitCostQuota: 0.02,
		SourceType:            SupplyCostProfileSourceAccounting,
		SourceRef:             "cost-ledger/gb10-4t/2026-06",
		ObservedAt:            1500,
		Notes:                 "gb10 amortized capex and power",
	}, 7)
	require.NoError(t, err)
	require.Positive(t, first.Id)
	require.Equal(t, 7, first.RecordedBy)
	require.InDelta(t, 0.12, first.AmortizedUnitCostQuota, 0.000001)
	require.Equal(t, SupplyCostProfileSourceAccounting, first.SourceType)

	second, err := RecordSupplyCostProfile(SupplyCostProfileRecordInput{
		SupplierId:            supplier.Id,
		SupplyNode:            "gb10-4t-self-hosted",
		ModelName:             "gpt-cost-profile",
		PeriodStart:           1000,
		PeriodEnd:             2000,
		CapacityTokens:        2000,
		FixedCostQuota:        100,
		VariableUnitCostQuota: 0.01,
		SourceType:            SupplyCostProfileSourceAccounting,
		SourceRef:             "cost-ledger/gb10-4t/2026-06",
		ObservedAt:            1600,
		Notes:                 "updated cost basis",
	}, 9)
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)
	require.Equal(t, int64(2000), second.CapacityTokens)
	require.InDelta(t, 0.06, second.AmortizedUnitCostQuota, 0.000001)
	require.Equal(t, 9, second.RecordedBy)

	profiles, total, err := SearchSupplyCostProfiles(SupplyCostProfileFilters{
		SupplierId: supplier.Id,
		ModelName:  "gpt-cost-profile",
		SourceType: SupplyCostProfileSourceAccounting,
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, profiles, 1)
	require.Equal(t, second.Id, profiles[0].Id)
}

func TestRecordSupplyCostProfileRequiresSelfHostedSupplier(t *testing.T) {
	truncateTables(t)

	supplier := &Supplier{Name: "gb10-4t-third-party-cost", Type: SupplierTypeThirdParty, Status: common.ChannelStatusEnabled}
	require.NoError(t, supplier.Insert())

	_, err := RecordSupplyCostProfile(SupplyCostProfileRecordInput{
		SupplierId:            supplier.Id,
		SupplyNode:            "gb10-4t-third-party",
		ModelName:             "gpt-cost-profile",
		PeriodStart:           1000,
		PeriodEnd:             2000,
		CapacityTokens:        1000,
		FixedCostQuota:        100,
		VariableUnitCostQuota: 0.02,
		SourceType:            SupplyCostProfileSourceAccounting,
		SourceRef:             "cost-ledger/third-party",
		ObservedAt:            1500,
	}, 7)
	require.Error(t, err)
	require.Contains(t, err.Error(), "self_hosted")
}
