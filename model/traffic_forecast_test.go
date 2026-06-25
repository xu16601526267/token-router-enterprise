package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateTrafficForecastsFromProfiles(t *testing.T) {
	truncateTables(t)

	sliceKey := trafficProfileSliceKey("gpt-test", "default", 2)
	profiles := []*TrafficProfile{
		{
			SliceKey:             sliceKey,
			ModelName:            "gpt-test",
			SlaTier:              "default",
			UserId:               2,
			PeriodStart:          1000,
			PeriodEnd:            2000,
			RequestCount:         1,
			SuccessRequestCount:  1,
			DemandTokens:         100,
			PeakTokens:           150,
			CacheHitRate:         0.25,
			SlaMetRate:           1,
			GrossProfitQuota:     40,
			SupplyHeadroomTokens: 500,
			AvgUnitCostQuota:     0.4,
			GeneratedAt:          1500,
			CreatedAt:            1500,
			UpdatedAt:            1500,
		},
		{
			SliceKey:             sliceKey,
			ModelName:            "gpt-test",
			SlaTier:              "default",
			UserId:               2,
			PeriodStart:          2000,
			PeriodEnd:            3000,
			RequestCount:         3,
			SuccessRequestCount:  3,
			DemandTokens:         300,
			PeakTokens:           360,
			CacheHitRate:         0.75,
			SlaMetRate:           0.9,
			GrossProfitQuota:     120,
			SupplyHeadroomTokens: 250,
			AvgUnitCostQuota:     0.6,
			GeneratedAt:          2500,
			CreatedAt:            2500,
			UpdatedAt:            2500,
		},
	}
	require.NoError(t, DB.Create(&profiles).Error)

	forecasts, err := GenerateTrafficForecasts(TrafficForecastGenerateInput{
		PeriodStart:       1000,
		PeriodEnd:         3000,
		TargetPeriodStart: 3000,
		TargetPeriodEnd:   5000,
	})
	require.NoError(t, err)
	require.Len(t, forecasts, 1)
	require.Positive(t, forecasts[0].Id)
	require.Equal(t, sliceKey, forecasts[0].SliceKey)
	require.Equal(t, "gpt-test", forecasts[0].ModelName)
	require.Equal(t, "default", forecasts[0].SlaTier)
	require.Equal(t, 2, forecasts[0].UserId)
	require.Equal(t, int64(1000), forecasts[0].SourcePeriodStart)
	require.Equal(t, int64(3000), forecasts[0].SourcePeriodEnd)
	require.Equal(t, int64(3000), forecasts[0].TargetPeriodStart)
	require.Equal(t, int64(5000), forecasts[0].TargetPeriodEnd)
	require.Equal(t, int64(2), forecasts[0].SourceProfileCount)
	require.Equal(t, int64(4), forecasts[0].ObservedRequestCount)
	require.Equal(t, int64(400), forecasts[0].ObservedDemandTokens)
	require.Equal(t, int64(360), forecasts[0].ObservedPeakTokens)
	require.Equal(t, int64(234), forecasts[0].BaselineDemandTokens)
	require.Equal(t, int64(234), forecasts[0].ForecastDemandTokens)
	require.Equal(t, int64(360), forecasts[0].ForecastPeakTokens)
	require.Equal(t, int64(250), forecasts[0].ForecastHeadroomTokens)
	require.Equal(t, int64(110), forecasts[0].ForecastGapTokens)
	require.Equal(t, int64(200), forecasts[0].TrendDemandDeltaTokens)
	require.InDelta(t, 2.0, forecasts[0].TrendDemandDeltaRate, 0.000001)
	require.Equal(t, 0, forecasts[0].SeasonalPeriodCount)
	require.InDelta(t, 1.0, forecasts[0].SeasonalIndex, 0.000001)
	require.Equal(t, int64(234), forecasts[0].SeasonalDemandTokens)
	require.Equal(t, TrafficForecastAnomalyNotEvaluated, forecasts[0].AnomalyStatus)
	require.Zero(t, forecasts[0].AnomalyProfileId)
	require.Zero(t, forecasts[0].AnomalyDemandRatio)
	require.InDelta(t, 0.5833333333333334, forecasts[0].CacheHitRate, 0.000001)
	require.InDelta(t, 0.9333333333333333, forecasts[0].SlaMetRate, 0.000001)
	require.Equal(t, int64(94), forecasts[0].GrossProfitQuota)
	require.InDelta(t, 0.5333333333333333, forecasts[0].AvgUnitCostQuota, 0.000001)
	require.InDelta(t, 2.0/3.0, forecasts[0].Confidence, 0.000001)
	require.Equal(t, TrafficForecastMethodWeightedMovingAverage, forecasts[0].Method)
	require.Contains(t, forecasts[0].Reason, "recency-weighted")
	require.Contains(t, forecasts[0].Reason, "2 traffic profile")

	regenerated, err := GenerateTrafficForecasts(TrafficForecastGenerateInput{
		PeriodStart:       1000,
		PeriodEnd:         3000,
		TargetPeriodStart: 3000,
		TargetPeriodEnd:   5000,
	})
	require.NoError(t, err)
	require.Len(t, regenerated, 1)
	require.Equal(t, forecasts[0].Id, regenerated[0].Id)

	queried, total, err := SearchTrafficForecasts(TrafficForecastFilters{
		ModelName:         "gpt-test",
		SlaTier:           "default",
		UserId:            2,
		Method:            TrafficForecastMethodWeightedMovingAverage,
		SourcePeriodStart: 1000,
		SourcePeriodEnd:   3000,
		TargetPeriodStart: 3000,
		TargetPeriodEnd:   5000,
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, queried, 1)
	require.Equal(t, forecasts[0].Id, queried[0].Id)
}

func TestGenerateTrafficForecastsWithSeasonalAnomalyAdjustment(t *testing.T) {
	truncateTables(t)

	sliceKey := trafficProfileSliceKey("gpt-test", "gold", 9)
	profiles := []*TrafficProfile{
		{
			SliceKey:             sliceKey,
			ModelName:            "gpt-test",
			SlaTier:              "gold",
			UserId:               9,
			PeriodStart:          1000,
			PeriodEnd:            2000,
			RequestCount:         1,
			SuccessRequestCount:  1,
			DemandTokens:         100,
			PeakTokens:           130,
			CacheHitRate:         0.2,
			SlaMetRate:           1,
			GrossProfitQuota:     40,
			SupplyHeadroomTokens: 500,
			AvgUnitCostQuota:     0.4,
			GeneratedAt:          1500,
			CreatedAt:            1500,
			UpdatedAt:            1500,
		},
		{
			SliceKey:             sliceKey,
			ModelName:            "gpt-test",
			SlaTier:              "gold",
			UserId:               9,
			PeriodStart:          2000,
			PeriodEnd:            3000,
			RequestCount:         1,
			SuccessRequestCount:  1,
			DemandTokens:         300,
			PeakTokens:           330,
			CacheHitRate:         0.3,
			SlaMetRate:           1,
			GrossProfitQuota:     90,
			SupplyHeadroomTokens: 400,
			AvgUnitCostQuota:     0.5,
			GeneratedAt:          2500,
			CreatedAt:            2500,
			UpdatedAt:            2500,
		},
		{
			SliceKey:             sliceKey,
			ModelName:            "gpt-test",
			SlaTier:              "gold",
			UserId:               9,
			PeriodStart:          3000,
			PeriodEnd:            4000,
			RequestCount:         1,
			SuccessRequestCount:  1,
			DemandTokens:         120,
			PeakTokens:           160,
			CacheHitRate:         0.4,
			SlaMetRate:           0.9,
			GrossProfitQuota:     50,
			SupplyHeadroomTokens: 350,
			AvgUnitCostQuota:     0.45,
			GeneratedAt:          3500,
			CreatedAt:            3500,
			UpdatedAt:            3500,
		},
		{
			SliceKey:             sliceKey,
			ModelName:            "gpt-test",
			SlaTier:              "gold",
			UserId:               9,
			PeriodStart:          4000,
			PeriodEnd:            5000,
			RequestCount:         1,
			SuccessRequestCount:  1,
			DemandTokens:         360,
			PeakTokens:           390,
			CacheHitRate:         0.5,
			SlaMetRate:           0.95,
			GrossProfitQuota:     110,
			SupplyHeadroomTokens: 250,
			AvgUnitCostQuota:     0.55,
			GeneratedAt:          4500,
			CreatedAt:            4500,
			UpdatedAt:            4500,
		},
	}
	require.NoError(t, DB.Create(&profiles).Error)

	forecasts, err := GenerateTrafficForecasts(TrafficForecastGenerateInput{
		PeriodStart:          1000,
		PeriodEnd:            5000,
		TargetPeriodStart:    5000,
		TargetPeriodEnd:      6000,
		ModelName:            "gpt-test",
		SlaTier:              "gold",
		UserId:               9,
		SeasonalPeriodCount:  2,
		AnomalyGuard:         true,
		AnomalyThresholdRate: 1.8,
	})
	require.NoError(t, err)
	require.Len(t, forecasts, 1)
	forecast := forecasts[0]
	require.Equal(t, TrafficForecastMethodSeasonalAnomaly, forecast.Method)
	require.Equal(t, int64(4), forecast.SourceProfileCount)
	require.Equal(t, int64(880), forecast.ObservedDemandTokens)
	require.Equal(t, int64(390), forecast.ObservedPeakTokens)
	require.Equal(t, int64(250), forecast.BaselineDemandTokens)
	require.Equal(t, int64(150), forecast.ForecastDemandTokens)
	require.Equal(t, int64(390), forecast.ForecastPeakTokens)
	require.Equal(t, int64(250), forecast.ForecastHeadroomTokens)
	require.Equal(t, int64(140), forecast.ForecastGapTokens)
	require.Equal(t, int64(260), forecast.TrendDemandDeltaTokens)
	require.InDelta(t, 2.6, forecast.TrendDemandDeltaRate, 0.000001)
	require.Equal(t, 2, forecast.SeasonalPeriodCount)
	require.InDelta(t, 0.5, forecast.SeasonalIndex, 0.000001)
	require.Equal(t, int64(125), forecast.SeasonalDemandTokens)
	require.Equal(t, TrafficForecastAnomalySpike, forecast.AnomalyStatus)
	require.Equal(t, profiles[3].Id, forecast.AnomalyProfileId)
	require.InDelta(t, 360.0/(520.0/3.0), forecast.AnomalyDemandRatio, 0.000001)
	require.Contains(t, forecast.Reason, "seasonal/anomaly adjusted")
	require.Contains(t, forecast.Reason, "anomaly_status=spike")

	queried, total, err := SearchTrafficForecasts(TrafficForecastFilters{
		ModelName:         "gpt-test",
		SlaTier:           "gold",
		UserId:            9,
		Method:            TrafficForecastMethodSeasonalAnomaly,
		SourcePeriodStart: 1000,
		SourcePeriodEnd:   5000,
		TargetPeriodStart: 5000,
		TargetPeriodEnd:   6000,
	}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, queried, 1)
	require.Equal(t, forecast.Id, queried[0].Id)
	require.Equal(t, TrafficForecastAnomalySpike, queried[0].AnomalyStatus)
}

func TestGenerateTrafficForecastsWithoutProfilesReturnsEmpty(t *testing.T) {
	truncateTables(t)

	forecasts, err := GenerateTrafficForecasts(TrafficForecastGenerateInput{
		PeriodStart: 1000,
		PeriodEnd:   3000,
	})
	require.NoError(t, err)
	require.Empty(t, forecasts)
}
