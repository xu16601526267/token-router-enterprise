package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateOperatingInsightsLinksRecommendationsAndPreservesReview(t *testing.T) {
	truncateTables(t)
	profile := &TrafficProfile{
		SliceKey:              "model:gpt-test|sla:default|user:2",
		ModelName:             "gpt-test",
		SlaTier:               "default",
		UserId:                2,
		PeriodStart:           1000,
		PeriodEnd:             2000,
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
		GeneratedAt:           1500,
		CreatedAt:             1500,
		UpdatedAt:             1500,
	}
	require.NoError(t, DB.Create(profile).Error)

	decisions, err := GenerateSupplyDecisions(SupplyDecisionGenerateInput{
		PeriodStart: profile.PeriodStart,
		PeriodEnd:   profile.PeriodEnd,
	})
	require.NoError(t, err)
	require.Len(t, decisions, 1)
	require.Equal(t, SupplyDecisionTrackSelfHosted, decisions[0].Track)

	recommendations, err := GeneratePricingRecommendations(PricingRecommendationGenerateInput{
		PeriodStart: profile.PeriodStart,
		PeriodEnd:   profile.PeriodEnd,
	})
	require.NoError(t, err)
	require.Len(t, recommendations, 1)
	require.Equal(t, PricingRecommendationActionShareSavings, recommendations[0].Action)

	insights, err := GenerateOperatingInsights(OperatingInsightGenerateInput{
		PeriodStart: profile.PeriodStart,
		PeriodEnd:   profile.PeriodEnd,
	})
	require.NoError(t, err)
	require.Len(t, insights, 1)
	require.Equal(t, OperatingInsightCategoryCacheEfficiency, insights[0].Category)
	require.Equal(t, OperatingInsightSeverityAction, insights[0].Severity)
	require.Equal(t, OperatingInsightStatusDraft, insights[0].Status)
	require.Equal(t, profile.Id, insights[0].TrafficProfileId)
	require.Equal(t, decisions[0].Id, insights[0].SupplyDecisionId)
	require.Equal(t, recommendations[0].Id, insights[0].PricingRecommendationId)
	require.Equal(t, SupplyDecisionTrackSelfHosted, insights[0].SupplyDecisionTrack)
	require.Equal(t, PricingRecommendationActionShareSavings, insights[0].PricingRecommendationAction)
	require.Equal(t, int64(300), insights[0].DemandTokens)
	require.Equal(t, int64(700), insights[0].SupplyHeadroomTokens)
	require.InDelta(t, 0.5, insights[0].CacheHitRate, 0.000001)
	require.True(t, strings.Contains(insights[0].RecommendedAction, "self-hosted"))

	acknowledged, err := UpdateOperatingInsightReview(insights[0].Id, OperatingInsightStatusAcknowledged, 1, "operator reviewed positive-sum slice")
	require.NoError(t, err)
	require.Equal(t, OperatingInsightStatusAcknowledged, acknowledged.Status)
	require.Equal(t, 1, acknowledged.ReviewedBy)
	require.Equal(t, "operator reviewed positive-sum slice", acknowledged.ReviewNote)

	regenerated, err := GenerateOperatingInsights(OperatingInsightGenerateInput{
		PeriodStart: profile.PeriodStart,
		PeriodEnd:   profile.PeriodEnd,
	})
	require.NoError(t, err)
	require.Len(t, regenerated, 1)
	require.Equal(t, insights[0].Id, regenerated[0].Id)
	require.Equal(t, OperatingInsightStatusAcknowledged, regenerated[0].Status)
	require.Equal(t, "operator reviewed positive-sum slice", regenerated[0].ReviewNote)
	require.Equal(t, OperatingInsightCategoryCacheEfficiency, regenerated[0].Category)
}

func TestGenerateOperatingInsightsWithoutProfiles(t *testing.T) {
	truncateTables(t)
	insights, err := GenerateOperatingInsights(OperatingInsightGenerateInput{
		PeriodStart: 1000,
		PeriodEnd:   2000,
	})
	require.NoError(t, err)
	require.Empty(t, insights)
}

func TestGenerateOperatingInsightsFromSupplyCapacityTelemetryRisk(t *testing.T) {
	truncateTables(t)
	require.NoError(t, (&Supplier{
		Id:     1,
		Name:   "gb10-capacity-risk",
		Type:   SupplierTypeThirdParty,
		Status: 1,
	}).Insert())

	missing := &SupplyCapacity{
		SupplierId:     1,
		SupplyNode:     "gb10-missing",
		ModelName:      "gpt-test",
		PeriodStart:    3000,
		PeriodEnd:      8000,
		CapacityTokens: 1000,
		UsedTokens:     100,
		QualityScore:   98,
		UnitCostQuota:  0.5,
		Status:         1,
	}
	require.NoError(t, missing.Insert())
	_, err := RecordSupplyCapacityTelemetry(SupplyCapacityTelemetryRecordInput{
		SupplierId:         1,
		SupplyNode:         "gb10-stale",
		ModelName:          "gpt-test",
		PeriodStart:        3000,
		PeriodEnd:          8000,
		CapacityTokens:     1000,
		UsedTokens:         100,
		GpuUtilizationRate: 0.5,
		QualityScore:       98,
		UnitCostQuota:      0.5,
		SourceType:         SupplyCapacityTelemetrySourceNodeReport,
		SourceRef:          "stale-capacity-telemetry",
		ObservedAt:         3100,
	}, 1)
	require.NoError(t, err)
	_, err = RecordSupplyCapacityTelemetry(SupplyCapacityTelemetryRecordInput{
		SupplierId:         1,
		SupplyNode:         "gb10-hot",
		ModelName:          "gpt-test",
		PeriodStart:        3000,
		PeriodEnd:          8000,
		CapacityTokens:     1000,
		UsedTokens:         950,
		GpuUtilizationRate: 0.94,
		QualityScore:       98,
		UnitCostQuota:      0.5,
		SourceType:         SupplyCapacityTelemetrySourceNodeReport,
		SourceRef:          "hot-capacity-telemetry",
		ObservedAt:         7900,
	}, 1)
	require.NoError(t, err)
	_, err = RecordSupplyCapacityTelemetry(SupplyCapacityTelemetryRecordInput{
		SupplierId:         1,
		SupplyNode:         "gb10-low-headroom",
		ModelName:          "gpt-test",
		PeriodStart:        3000,
		PeriodEnd:          8000,
		CapacityTokens:     1000,
		UsedTokens:         920,
		GpuUtilizationRate: 0.55,
		QualityScore:       98,
		UnitCostQuota:      0.5,
		SourceType:         SupplyCapacityTelemetrySourceNodeReport,
		SourceRef:          "low-headroom-capacity-telemetry",
		ObservedAt:         7900,
	}, 1)
	require.NoError(t, err)

	insights, err := GenerateOperatingInsights(OperatingInsightGenerateInput{
		PeriodStart: 3000,
		PeriodEnd:   8000,
		ModelName:   "gpt-test",
	})
	require.NoError(t, err)
	require.Len(t, insights, 4)

	byNode := map[string]*OperatingInsight{}
	for _, insight := range insights {
		require.Equal(t, OperatingInsightCategoryCapacityRisk, insight.Category)
		require.Equal(t, OperatingInsightStatusDraft, insight.Status)
		require.Equal(t, "gpt-test", insight.ModelName)
		require.Equal(t, "default", insight.SlaTier)
		require.Zero(t, insight.UserId)
		byNode[insight.SliceKey] = insight
	}
	missingInsight := byNode["capacity:supplier:1|node:gb10-missing|model:gpt-test|reason:missing_telemetry"]
	require.NotNil(t, missingInsight)
	require.Equal(t, OperatingInsightSeverityWatch, missingInsight.Severity)
	require.Contains(t, missingInsight.Summary, "no linked telemetry evidence")

	staleInsight := byNode["capacity:supplier:1|node:gb10-stale|model:gpt-test|reason:stale_telemetry"]
	require.NotNil(t, staleInsight)
	require.Equal(t, OperatingInsightSeverityWatch, staleInsight.Severity)
	require.Contains(t, staleInsight.Summary, "last observed telemetry")

	hotInsight := byNode["capacity:supplier:1|node:gb10-hot|model:gpt-test|reason:high_gpu"]
	require.NotNil(t, hotInsight)
	require.Equal(t, OperatingInsightSeverityAction, hotInsight.Severity)
	require.Equal(t, int64(50), hotInsight.SupplyHeadroomTokens)
	require.Contains(t, hotInsight.Title, "GPU utilization")

	lowHeadroomInsight := byNode["capacity:supplier:1|node:gb10-low-headroom|model:gpt-test|reason:low_headroom"]
	require.NotNil(t, lowHeadroomInsight)
	require.Equal(t, OperatingInsightSeverityAction, lowHeadroomInsight.Severity)
	require.Equal(t, int64(80), lowHeadroomInsight.SupplyHeadroomTokens)
	require.Contains(t, lowHeadroomInsight.Title, "headroom")

	acknowledged, err := UpdateOperatingInsightReview(hotInsight.Id, OperatingInsightStatusAcknowledged, 1, "operator reviewed hot node")
	require.NoError(t, err)
	require.Equal(t, OperatingInsightStatusAcknowledged, acknowledged.Status)

	regenerated, err := GenerateOperatingInsights(OperatingInsightGenerateInput{
		PeriodStart: 3000,
		PeriodEnd:   8000,
		ModelName:   "gpt-test",
	})
	require.NoError(t, err)
	require.Len(t, regenerated, 4)
	for _, insight := range regenerated {
		if insight.InsightKey == hotInsight.InsightKey {
			require.Equal(t, hotInsight.Id, insight.Id)
			require.Equal(t, OperatingInsightStatusAcknowledged, insight.Status)
			require.Equal(t, "operator reviewed hot node", insight.ReviewNote)
			return
		}
	}
	t.Fatalf("hot capacity telemetry insight not found after regeneration")
}

func TestGenerateOperatingInsightsFromFailedSlaProbeRun(t *testing.T) {
	resetSlaMeasurementTables(t)
	supplier, channel := seedSlaMeasurementSupplierChannel(t)

	contract, err := ImportSlaContract(slaMeasurementContractInput(), 1)
	require.NoError(t, err)
	plan, err := GenerateSlaProbePlan(SlaProbePlanGenerateInput{
		ContractId:              contract.Id,
		SupplierId:              supplier.Id,
		ChannelId:               channel.Id,
		ProbeType:               SlaProbeTypeRuntimeLight,
		RouteMode:               SlaProbeRouteModeDirectUpstream,
		PromptSuiteKey:          "runtime-watch",
		TokenizerRef:            "test-tokenizer",
		SampleSize:              1,
		RepeatCount:             1,
		MaxProbeQuota:           100,
		InputProfileJSON:        "{}",
		OutputProfileJSON:       "{}",
		StreamProfileJSON:       "{}",
		ErrorProfileJSON:        "{}",
		RateProfileJSON:         "{}",
		CacheProfile:            "cold_no_cache",
		AvailabilityProfileJSON: "{}",
	}, 1)
	require.NoError(t, err)
	run, err := RecordSlaProbeRun(SlaProbeRunRecordInput{
		RunKey:         "runtime-sla-failed-run",
		PlanId:         plan.Id,
		Status:         SlaProbeRunStatusFailed,
		StartedAt:      1100,
		EndedAt:        1200,
		RunnerVersion:  "token-router-sla-test",
		RuntimeRef:     "aima2/runtime-watch",
		Endpoint:       "http://gb10-4t.test/v1/chat/completions",
		SummaryJSON:    `{"ttft_ms_p90": 9000}`,
		HardGatePassed: false,
		FailureReasons: "ttft p90 exceeded hard gate",
		ArtifactURI:    "file:///tmp/runtime-sla-failed-run.json",
		ArtifactSHA256: "failed-run-sha",
	}, 1)
	require.NoError(t, err)

	insights, err := GenerateOperatingInsights(OperatingInsightGenerateInput{
		PeriodStart: 1000,
		PeriodEnd:   2000,
	})
	require.NoError(t, err)
	require.Len(t, insights, 1)
	require.Equal(t, OperatingInsightCategoryQualityWatch, insights[0].Category)
	require.Equal(t, OperatingInsightSeverityAction, insights[0].Severity)
	require.Equal(t, OperatingInsightStatusDraft, insights[0].Status)
	require.Equal(t, contract.Id, insights[0].SlaContractId)
	require.Equal(t, run.Id, insights[0].SlaProbeRunId)
	require.Equal(t, run.RunKey, insights[0].SlaProbeRunKey)
	require.Equal(t, SlaProbeRunStatusFailed, insights[0].SlaProbeStatus)
	require.False(t, insights[0].SlaHardGatePassed)
	require.Equal(t, "ttft p90 exceeded hard gate", insights[0].SlaFailureReasons)
	require.Equal(t, "failed-run-sha", insights[0].SlaArtifactSHA256)
	require.Equal(t, "aima2/runtime-watch", insights[0].SlaRuntimeRef)
	require.Contains(t, insights[0].Summary, "runtime-sla-failed-run")

	acknowledged, err := UpdateOperatingInsightReview(insights[0].Id, OperatingInsightStatusAcknowledged, 1, "operator reviewed failed SLA run")
	require.NoError(t, err)
	require.Equal(t, OperatingInsightStatusAcknowledged, acknowledged.Status)

	regenerated, err := GenerateOperatingInsights(OperatingInsightGenerateInput{
		PeriodStart: 1000,
		PeriodEnd:   2000,
	})
	require.NoError(t, err)
	require.Len(t, regenerated, 1)
	require.Equal(t, insights[0].Id, regenerated[0].Id)
	require.Equal(t, OperatingInsightStatusAcknowledged, regenerated[0].Status)
	require.Equal(t, "operator reviewed failed SLA run", regenerated[0].ReviewNote)
}
