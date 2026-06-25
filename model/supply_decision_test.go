package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateSupplyDecisionsFallsBackToProfileEvidence(t *testing.T) {
	truncateTables(t)
	profile := seedSupplyDecisionProfile(t, 1000, 2000)

	decisions, err := GenerateSupplyDecisions(SupplyDecisionGenerateInput{
		PeriodStart: profile.PeriodStart,
		PeriodEnd:   profile.PeriodEnd,
	})
	require.NoError(t, err)
	require.Len(t, decisions, 1)

	decision := decisions[0]
	require.Equal(t, profile.Id, decision.TrafficProfileId)
	require.Equal(t, 0, decision.TrafficForecastId)
	require.Equal(t, SupplyDecisionSourceProfile, decision.DecisionSource)
	require.Equal(t, int64(0), decision.ForecastTargetStart)
	require.Equal(t, int64(0), decision.ForecastTargetEnd)
	require.Equal(t, float64(0), decision.ForecastConfidence)
	require.Equal(t, "", decision.ForecastMethod)
	require.Equal(t, profile.DemandTokens, decision.DemandTokens)
	require.Equal(t, profile.PeakTokens, decision.PeakTokens)
	require.Equal(t, profile.SupplyHeadroomTokens, decision.SupplyHeadroomTokens)
	require.Equal(t, int64(0), decision.GapTokens)
	require.Equal(t, SupplyDecisionTrackSelfHosted, decision.Track)
	require.Equal(t, SupplyDecisionTypeSelfHostedEvaluate, decision.DecisionType)
}

func TestGenerateSupplyDecisionsUsesForecastEvidenceAndPreservesReview(t *testing.T) {
	truncateTables(t)
	profile := seedSupplyDecisionProfile(t, 3000, 4000)
	forecast := seedSupplyDecisionForecast(t, profile, 4000, 5000, 900, 700, 100, 600)

	decisions, err := GenerateSupplyDecisions(SupplyDecisionGenerateInput{
		PeriodStart: profile.PeriodStart,
		PeriodEnd:   profile.PeriodEnd,
	})
	require.NoError(t, err)
	require.Len(t, decisions, 1)

	decision := decisions[0]
	require.Equal(t, profile.Id, decision.TrafficProfileId)
	require.Equal(t, forecast.Id, decision.TrafficForecastId)
	require.Equal(t, SupplyDecisionSourceForecast, decision.DecisionSource)
	require.Equal(t, forecast.TargetPeriodStart, decision.ForecastTargetStart)
	require.Equal(t, forecast.TargetPeriodEnd, decision.ForecastTargetEnd)
	require.InDelta(t, forecast.Confidence, decision.ForecastConfidence, 0.000001)
	require.Equal(t, forecast.Method, decision.ForecastMethod)
	require.Equal(t, forecast.ForecastDemandTokens, decision.DemandTokens)
	require.Equal(t, forecast.ForecastPeakTokens, decision.PeakTokens)
	require.Equal(t, forecast.ForecastHeadroomTokens, decision.SupplyHeadroomTokens)
	require.Equal(t, forecast.ForecastGapTokens, decision.GapTokens)
	require.Equal(t, forecast.ForecastGapTokens, decision.RecommendedCapacity)
	require.Equal(t, SupplyDecisionTrackThirdParty, decision.Track)
	require.Equal(t, SupplyDecisionTypeThirdPartyRecruit, decision.DecisionType)
	require.Contains(t, decision.Reason, "forecast-informed")
	require.Contains(t, decision.Reason, TrafficForecastMethodMovingAverage)

	approved, err := UpdateSupplyDecisionReview(decision.Id, SupplyDecisionStatusApproved, 1, "operator accepted forecast evidence")
	require.NoError(t, err)
	require.Equal(t, SupplyDecisionStatusApproved, approved.Status)

	require.NoError(t, DB.Model(&TrafficForecast{}).Where("id = ?", forecast.Id).Updates(map[string]any{
		"forecast_demand_tokens":   int64(1000),
		"forecast_peak_tokens":     int64(800),
		"forecast_headroom_tokens": int64(300),
		"forecast_gap_tokens":      int64(500),
		"confidence":               0.91,
	}).Error)

	regenerated, err := GenerateSupplyDecisions(SupplyDecisionGenerateInput{
		PeriodStart: profile.PeriodStart,
		PeriodEnd:   profile.PeriodEnd,
	})
	require.NoError(t, err)
	require.Len(t, regenerated, 1)
	require.Equal(t, decision.Id, regenerated[0].Id)
	require.Equal(t, SupplyDecisionStatusApproved, regenerated[0].Status)
	require.Equal(t, "operator accepted forecast evidence", regenerated[0].ReviewNote)
	require.Equal(t, int64(1000), regenerated[0].DemandTokens)
	require.Equal(t, int64(500), regenerated[0].GapTokens)
	require.InDelta(t, 0.91, regenerated[0].ForecastConfidence, 0.000001)
}

func seedSupplyDecisionProfile(t *testing.T, periodStart int64, periodEnd int64) *TrafficProfile {
	t.Helper()
	profile := &TrafficProfile{
		SliceKey:              trafficProfileSliceKey("gpt-test", "default", 2),
		ModelName:             "gpt-test",
		SlaTier:               "default",
		UserId:                2,
		PeriodStart:           periodStart,
		PeriodEnd:             periodEnd,
		RequestCount:          2,
		SuccessRequestCount:   2,
		DemandTokens:          300,
		PeakTokens:            300,
		PeakRatio:             1,
		UniqueSessions:        1,
		CacheHitCount:         1,
		CacheHitRate:          0.5,
		TotalCachedTokens:     80,
		SlaMetRate:            1,
		AvgLatencyMs:          120,
		MaxLatencyMs:          140,
		TotalSellQuota:        228,
		TotalCostQuota:        112,
		GrossProfitQuota:      116,
		SupplyCapacityTokens:  1000,
		SupplyUsedTokens:      300,
		SupplyHeadroomTokens:  700,
		AvgSupplyQualityScore: 98.5,
		AvgUnitCostQuota:      0.5,
		GeneratedAt:           periodEnd,
		CreatedAt:             periodEnd,
		UpdatedAt:             periodEnd,
	}
	require.NoError(t, DB.Create(profile).Error)
	return profile
}

func seedSupplyDecisionForecast(
	t *testing.T,
	profile *TrafficProfile,
	targetStart int64,
	targetEnd int64,
	demandTokens int64,
	peakTokens int64,
	headroomTokens int64,
	gapTokens int64,
) *TrafficForecast {
	t.Helper()
	forecast := &TrafficForecast{
		ForecastKey:            strings.Join([]string{"forecast", profile.SliceKey, "target", "test"}, ":"),
		SliceKey:               profile.SliceKey,
		ModelName:              profile.ModelName,
		SlaTier:                profile.SlaTier,
		UserId:                 profile.UserId,
		SourcePeriodStart:      profile.PeriodStart,
		SourcePeriodEnd:        profile.PeriodEnd,
		TargetPeriodStart:      targetStart,
		TargetPeriodEnd:        targetEnd,
		SourceProfileCount:     1,
		ObservedRequestCount:   profile.RequestCount,
		ObservedDemandTokens:   profile.DemandTokens,
		ObservedPeakTokens:     profile.PeakTokens,
		ForecastDemandTokens:   demandTokens,
		ForecastPeakTokens:     peakTokens,
		ForecastHeadroomTokens: headroomTokens,
		ForecastGapTokens:      gapTokens,
		CacheHitRate:           profile.CacheHitRate,
		SlaMetRate:             profile.SlaMetRate,
		GrossProfitQuota:       profile.GrossProfitQuota,
		AvgUnitCostQuota:       profile.AvgUnitCostQuota,
		Confidence:             0.82,
		Method:                 TrafficForecastMethodMovingAverage,
		Reason:                 "forecast test fixture",
		GeneratedAt:            targetStart,
		CreatedAt:              targetStart,
		UpdatedAt:              targetStart,
	}
	require.NoError(t, DB.Create(forecast).Error)
	return forecast
}
