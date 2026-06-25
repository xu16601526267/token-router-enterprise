package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateSupplyExpansionOpportunitiesFromForecastSelfHostedDecision(t *testing.T) {
	truncateTables(t)
	profile := seedSupplyDecisionProfile(t, 5000, 6000)
	forecast := seedSupplyDecisionForecast(t, profile, 6000, 7000, 300, 300, 700, 0)

	decisions, err := GenerateSupplyDecisions(SupplyDecisionGenerateInput{
		PeriodStart: profile.PeriodStart,
		PeriodEnd:   profile.PeriodEnd,
	})
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	approved, err := UpdateSupplyDecisionReview(decisions[0].Id, SupplyDecisionStatusApproved, 1, "approved before opportunity generation")
	require.NoError(t, err)

	opportunities, err := GenerateSupplyExpansionOpportunities(SupplyExpansionOpportunityGenerateInput{
		PeriodStart:    profile.PeriodStart,
		PeriodEnd:      profile.PeriodEnd,
		DecisionStatus: SupplyDecisionStatusApproved,
	})
	require.NoError(t, err)
	require.Len(t, opportunities, 1)

	opportunity := opportunities[0]
	require.Equal(t, approved.Id, opportunity.SupplyDecisionId)
	require.Equal(t, profile.Id, opportunity.TrafficProfileId)
	require.Equal(t, forecast.Id, opportunity.TrafficForecastId)
	require.Equal(t, SupplyDecisionSourceForecast, opportunity.DecisionSource)
	require.Equal(t, SupplyDecisionStatusApproved, opportunity.DecisionStatus)
	require.Equal(t, SupplyExpansionOpportunityTypeSelfHosted, opportunity.OpportunityType)
	require.Equal(t, SupplyExpansionOpportunityPriorityAction, opportunity.Priority)
	require.Equal(t, SupplyExpansionOpportunityClusterHighCacheStable, opportunity.ClusterKey)
	require.Equal(t, SupplyDecisionTrackSelfHosted, opportunity.Track)
	require.Equal(t, SupplyDecisionTypeSelfHostedEvaluate, opportunity.DecisionType)
	require.Equal(t, forecast.TargetPeriodStart, opportunity.ForecastTargetStart)
	require.Equal(t, forecast.TargetPeriodEnd, opportunity.ForecastTargetEnd)
	require.InDelta(t, forecast.Confidence, opportunity.ForecastConfidence, 0.000001)
	require.Equal(t, forecast.Method, opportunity.ForecastMethod)
	require.Equal(t, forecast.ForecastDemandTokens, opportunity.DemandTokens)
	require.Equal(t, forecast.ForecastPeakTokens, opportunity.PeakTokens)
	require.Equal(t, forecast.ForecastHeadroomTokens, opportunity.SupplyHeadroomTokens)
	require.Equal(t, forecast.ForecastGapTokens, opportunity.GapTokens)
	require.Equal(t, forecast.ForecastDemandTokens, opportunity.RecommendedCapacity)
	require.InDelta(t, 0.5, opportunity.LocalityScore, 0.000001)
	require.InDelta(t, 1.0, opportunity.StabilityScore, 0.000001)
	require.InDelta(t, 0.0, opportunity.HeadroomRiskScore, 0.000001)
	require.InDelta(t, approved.RoiScore, opportunity.RoiScore, 0.000001)
	require.Zero(t, opportunity.SelfHostedCostProfileId)
	require.Zero(t, opportunity.SelfHostedUnitCostQuota)
	require.Zero(t, opportunity.SelfHostedSavingsUnitQuota)
	require.Zero(t, opportunity.SelfHostedSavingsQuota)
	require.InDelta(t, 291.0, opportunity.RankScore, 0.000001)
	require.Contains(t, opportunity.Reason, "self-hosted expansion candidate")

	queried, total, err := SearchSupplyExpansionOpportunities(SupplyExpansionOpportunityFilters{
		Priority:   SupplyExpansionOpportunityPriorityAction,
		ClusterKey: SupplyExpansionOpportunityClusterHighCacheStable,
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, queried, 1)
	require.Equal(t, opportunity.OpportunityKey, queried[0].OpportunityKey)
}

func TestGenerateSupplyExpansionOpportunitiesUsesSelfHostedCostProfileEvidence(t *testing.T) {
	truncateTables(t)
	profile := seedSupplyDecisionProfile(t, 9000, 10000)
	forecast := seedSupplyDecisionForecast(t, profile, 10000, 11000, 300, 300, 700, 0)
	supplier := &Supplier{Name: "gb10-4t-opportunity-cost", Type: SupplierTypeSelfHosted, Status: 1}
	require.NoError(t, supplier.Insert())

	decisions, err := GenerateSupplyDecisions(SupplyDecisionGenerateInput{
		PeriodStart: profile.PeriodStart,
		PeriodEnd:   profile.PeriodEnd,
	})
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	approved, err := UpdateSupplyDecisionReview(decisions[0].Id, SupplyDecisionStatusApproved, 1, "approved before cost-based opportunity generation")
	require.NoError(t, err)

	costProfile, err := RecordSupplyCostProfile(SupplyCostProfileRecordInput{
		SupplierId:            supplier.Id,
		SupplyNode:            "gb10-4t-opportunity-cost",
		ModelName:             profile.ModelName,
		PeriodStart:           profile.PeriodStart,
		PeriodEnd:             profile.PeriodEnd,
		CapacityTokens:        1000,
		FixedCostQuota:        100,
		VariableUnitCostQuota: 0.02,
		SourceType:            SupplyCostProfileSourceAccounting,
		SourceRef:             "cost-ledger/opportunity",
		ObservedAt:            profile.PeriodEnd,
		Notes:                 "self-hosted amortized cost proof",
	}, 1)
	require.NoError(t, err)
	require.InDelta(t, 0.12, costProfile.AmortizedUnitCostQuota, 0.000001)

	opportunities, err := GenerateSupplyExpansionOpportunities(SupplyExpansionOpportunityGenerateInput{
		PeriodStart:    profile.PeriodStart,
		PeriodEnd:      profile.PeriodEnd,
		DecisionStatus: SupplyDecisionStatusApproved,
	})
	require.NoError(t, err)
	require.Len(t, opportunities, 1)

	opportunity := opportunities[0]
	require.Equal(t, approved.Id, opportunity.SupplyDecisionId)
	require.Equal(t, forecast.Id, opportunity.TrafficForecastId)
	require.Equal(t, costProfile.Id, opportunity.SelfHostedCostProfileId)
	require.InDelta(t, 0.12, opportunity.SelfHostedUnitCostQuota, 0.000001)
	require.InDelta(t, 0.38, opportunity.SelfHostedSavingsUnitQuota, 0.000001)
	require.InDelta(t, 114.0, opportunity.SelfHostedSavingsQuota, 0.000001)
	require.InDelta(t, 405.0, opportunity.RankScore, 0.000001)
	require.Contains(t, opportunity.Reason, "cost profile cost-ledger/opportunity")

	queried, total, err := SearchSupplyExpansionOpportunities(SupplyExpansionOpportunityFilters{
		OpportunityType: SupplyExpansionOpportunityTypeSelfHosted,
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, queried, 1)
	require.Equal(t, costProfile.Id, queried[0].SelfHostedCostProfileId)
	require.InDelta(t, 405.0, queried[0].RankScore, 0.000001)
}

func TestGenerateSupplyExpansionOpportunitiesFromForecastGapDecision(t *testing.T) {
	truncateTables(t)
	profile := seedSupplyDecisionProfile(t, 7000, 8000)
	forecast := seedSupplyDecisionForecast(t, profile, 8000, 9000, 900, 700, 100, 600)

	decisions, err := GenerateSupplyDecisions(SupplyDecisionGenerateInput{
		PeriodStart: profile.PeriodStart,
		PeriodEnd:   profile.PeriodEnd,
	})
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	require.Equal(t, SupplyDecisionTypeThirdPartyRecruit, decisions[0].DecisionType)

	opportunities, err := GenerateSupplyExpansionOpportunities(SupplyExpansionOpportunityGenerateInput{
		PeriodStart: profile.PeriodStart,
		PeriodEnd:   profile.PeriodEnd,
	})
	require.NoError(t, err)
	require.Len(t, opportunities, 1)

	opportunity := opportunities[0]
	require.Equal(t, decisions[0].Id, opportunity.SupplyDecisionId)
	require.Equal(t, forecast.Id, opportunity.TrafficForecastId)
	require.Equal(t, SupplyExpansionOpportunityTypeThirdPartyGap, opportunity.OpportunityType)
	require.Equal(t, SupplyExpansionOpportunityPriorityAction, opportunity.Priority)
	require.Equal(t, SupplyExpansionOpportunityClusterCapacityGap, opportunity.ClusterKey)
	require.Equal(t, SupplyDecisionTrackThirdParty, opportunity.Track)
	require.Equal(t, forecast.ForecastGapTokens, opportunity.GapTokens)
	require.Equal(t, forecast.ForecastGapTokens, opportunity.RecommendedCapacity)
	require.InDelta(t, float64(600)/float64(700), opportunity.HeadroomRiskScore, 0.000001)
	require.Greater(t, opportunity.RankScore, float64(0))
	require.Contains(t, opportunity.Reason, "recruit or expand third-party supply")
}
