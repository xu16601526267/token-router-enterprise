package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratePricingRecommendationsFromTrafficProfileAndPreserveReview(t *testing.T) {
	truncateTables(t)

	profile := &TrafficProfile{
		SliceKey:              trafficProfileSliceKey("gpt-test", "default", 2),
		ModelName:             "gpt-test",
		SlaTier:               "default",
		UserId:                2,
		PeriodStart:           100,
		PeriodEnd:             200,
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
		AvgLatencyMs:          42,
		MaxLatencyMs:          55,
		TotalSellQuota:        228,
		TotalCostQuota:        112,
		GrossProfitQuota:      116,
		SupplyCapacityTokens:  1000,
		SupplyUsedTokens:      300,
		SupplyHeadroomTokens:  700,
		AvgSupplyQualityScore: 98.5,
		AvgUnitCostQuota:      0.5,
		GeneratedAt:           123,
		CreatedAt:             123,
		UpdatedAt:             123,
	}
	require.NoError(t, DB.Create(profile).Error)

	recommendations, err := GeneratePricingRecommendations(PricingRecommendationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, recommendations, 1)
	require.Equal(t, profile.Id, recommendations[0].TrafficProfileId)
	require.Equal(t, profile.SliceKey, recommendations[0].SliceKey)
	require.Equal(t, PricingRecommendationActionShareSavings, recommendations[0].Action)
	require.Equal(t, PricingRecommendationStatusDraft, recommendations[0].Status)
	require.Equal(t, int64(300), recommendations[0].DemandTokens)
	require.Equal(t, int64(700), recommendations[0].SupplyHeadroomTokens)
	require.InDelta(t, 0.76, recommendations[0].CurrentUnitPriceQuota, 0.000001)
	require.InDelta(t, 112.0/300.0, recommendations[0].CurrentUnitCostQuota, 0.000001)
	require.InDelta(t, 116.0/228.0, recommendations[0].CurrentMarginRate, 0.000001)
	require.InDelta(t, 0.684, recommendations[0].RecommendedUnitPriceQuota, 0.000001)
	require.Contains(t, recommendations[0].Reason, "share efficiency savings")

	approved, err := UpdatePricingRecommendationReview(recommendations[0].Id, PricingRecommendationStatusApproved, 1, "accepted pricing recommendation")
	require.NoError(t, err)
	require.Equal(t, PricingRecommendationStatusApproved, approved.Status)
	require.Equal(t, 1, approved.ReviewedBy)
	require.Greater(t, approved.ReviewedAt, int64(0))
	require.Equal(t, "accepted pricing recommendation", approved.ReviewNote)

	regenerated, err := GeneratePricingRecommendations(PricingRecommendationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, regenerated, 1)
	require.Equal(t, recommendations[0].Id, regenerated[0].Id)
	require.Equal(t, PricingRecommendationStatusApproved, regenerated[0].Status)
	require.Equal(t, 1, regenerated[0].ReviewedBy)
	require.Equal(t, "accepted pricing recommendation", regenerated[0].ReviewNote)
}

func TestGeneratePricingRecommendationsWithoutProfilesReturnsEmpty(t *testing.T) {
	truncateTables(t)

	recommendations, err := GeneratePricingRecommendations(PricingRecommendationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Empty(t, recommendations)
}
