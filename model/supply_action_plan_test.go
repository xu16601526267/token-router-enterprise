package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateSupplyActionPlansCopiesOpportunityEvidence(t *testing.T) {
	truncateTables(t)
	profile := seedSupplyDecisionProfile(t, 9000, 10000)
	forecast := seedSupplyDecisionForecast(t, profile, 10000, 11000, 300, 300, 700, 0)

	decisions, err := GenerateSupplyDecisions(SupplyDecisionGenerateInput{
		PeriodStart: profile.PeriodStart,
		PeriodEnd:   profile.PeriodEnd,
	})
	require.NoError(t, err)
	require.Len(t, decisions, 1)

	approved, err := UpdateSupplyDecisionReview(decisions[0].Id, SupplyDecisionStatusApproved, 1, "approved before action plan generation")
	require.NoError(t, err)

	opportunities, err := GenerateSupplyExpansionOpportunities(SupplyExpansionOpportunityGenerateInput{
		PeriodStart:    profile.PeriodStart,
		PeriodEnd:      profile.PeriodEnd,
		DecisionStatus: SupplyDecisionStatusApproved,
	})
	require.NoError(t, err)
	require.Len(t, opportunities, 1)
	require.Equal(t, forecast.Id, opportunities[0].TrafficForecastId)

	plans, err := GenerateSupplyActionPlans(SupplyActionPlanGenerateInput{
		DecisionId: approved.Id,
	})
	require.NoError(t, err)
	require.Len(t, plans, 1)

	plan := plans[0]
	require.Equal(t, approved.Id, plan.SupplyDecisionId)
	require.Equal(t, opportunities[0].Id, plan.SupplyExpansionOpportunityId)
	require.Equal(t, opportunities[0].OpportunityKey, plan.OpportunityKey)
	require.Equal(t, opportunities[0].OpportunityType, plan.OpportunityType)
	require.Equal(t, opportunities[0].Priority, plan.OpportunityPriority)
	require.Equal(t, opportunities[0].ClusterKey, plan.OpportunityClusterKey)
	require.InDelta(t, opportunities[0].RankScore, plan.OpportunityRankScore, 0.000001)
	require.Equal(t, SupplyActionPlanStatusPlanned, plan.Status)

	completed, err := UpdateSupplyActionPlanStatus(plan.Id, SupplyActionPlanStatusInput{
		Status:       SupplyActionPlanStatusCompleted,
		OperatorNote: "completed with copied opportunity evidence",
	}, 1)
	require.NoError(t, err)
	require.Equal(t, SupplyActionPlanStatusCompleted, completed.Status)

	regenerated, err := GenerateSupplyActionPlans(SupplyActionPlanGenerateInput{
		DecisionId: approved.Id,
	})
	require.NoError(t, err)
	require.Len(t, regenerated, 1)
	require.Equal(t, SupplyActionPlanStatusCompleted, regenerated[0].Status)
	require.Equal(t, opportunities[0].Id, regenerated[0].SupplyExpansionOpportunityId)
	require.Equal(t, opportunities[0].OpportunityKey, regenerated[0].OpportunityKey)
	require.Equal(t, opportunities[0].Priority, regenerated[0].OpportunityPriority)
	require.InDelta(t, opportunities[0].RankScore, regenerated[0].OpportunityRankScore, 0.000001)
}
