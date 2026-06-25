package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateSupplierEvaluationsFromScorecardAndPreserveReview(t *testing.T) {
	truncateTables(t)

	supplier, scorecard := seedSupplierEvaluationScorecard(t, 89.665, SupplierScorecardGradeA)

	evaluations, err := GenerateSupplierEvaluations(SupplierEvaluationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, evaluations, 1)
	require.Equal(t, supplier.Id, evaluations[0].SupplierId)
	require.Equal(t, scorecard.Id, evaluations[0].SupplierScorecardId)
	require.Equal(t, SupplierEvaluationRecommendationAdmit, evaluations[0].Recommendation)
	require.Equal(t, SupplierEvaluationStatusDraft, evaluations[0].Status)
	require.Equal(t, SupplierScorecardGradeA, evaluations[0].Grade)
	require.InDelta(t, 89.665, evaluations[0].Score, 0.000001)
	require.Equal(t, int64(700), evaluations[0].SupplyHeadroomTokens)
	require.Zero(t, evaluations[0].SlaContractId)
	require.Zero(t, evaluations[0].SlaProbeRunId)
	require.Empty(t, evaluations[0].SlaGateSummaryJSON)

	_, err = ApplySupplierEvaluation(evaluations[0].Id, 1, "too early")
	require.ErrorContains(t, err, "must be approved")

	approved, err := UpdateSupplierEvaluationReview(evaluations[0].Id, SupplierEvaluationStatusApproved, 1, "accepted for controlled supply")
	require.NoError(t, err)
	require.Equal(t, SupplierEvaluationStatusApproved, approved.Status)
	require.Equal(t, 1, approved.ReviewedBy)
	require.Greater(t, approved.ReviewedAt, int64(0))
	require.Equal(t, "accepted for controlled supply", approved.ReviewNote)

	regenerated, err := GenerateSupplierEvaluations(SupplierEvaluationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, regenerated, 1)
	require.Equal(t, evaluations[0].Id, regenerated[0].Id)
	require.Equal(t, SupplierEvaluationStatusApproved, regenerated[0].Status)
	require.Equal(t, 1, regenerated[0].ReviewedBy)
	require.Equal(t, "accepted for controlled supply", regenerated[0].ReviewNote)

	var savedSupplier Supplier
	require.NoError(t, DB.First(&savedSupplier, supplier.Id).Error)
	require.Equal(t, 1, savedSupplier.Status)

	applied, err := ApplySupplierEvaluation(regenerated[0].Id, 2, "operator accepted gb10-4t")
	require.NoError(t, err)
	require.Greater(t, applied.AppliedAt, int64(0))
	require.Equal(t, 2, applied.AppliedBy)
	require.Equal(t, 1, applied.SupplierStatusBefore)
	require.Equal(t, 1, applied.SupplierStatusAfter)
	require.Contains(t, applied.AppliedNote, "recommendation=admit")
	require.Contains(t, applied.AppliedNote, "operator accepted gb10-4t")

	require.NoError(t, DB.First(&savedSupplier, supplier.Id).Error)
	require.Equal(t, 1, savedSupplier.Status)
	require.Contains(t, savedSupplier.Notes, "supplier_evaluation #")
	require.Contains(t, savedSupplier.Notes, "operator accepted gb10-4t")

	_, err = ApplySupplierEvaluation(regenerated[0].Id, 2, "duplicate")
	require.ErrorContains(t, err, "already applied")

	_, err = UpdateSupplierEvaluationReview(regenerated[0].Id, SupplierEvaluationStatusRejected, 3, "late rejection")
	require.ErrorContains(t, err, "cannot be reviewed")

	reappliedGenerate, err := GenerateSupplierEvaluations(SupplierEvaluationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, reappliedGenerate, 1)
	require.Equal(t, applied.AppliedAt, reappliedGenerate[0].AppliedAt)
	require.Equal(t, applied.AppliedBy, reappliedGenerate[0].AppliedBy)
	require.Equal(t, applied.AppliedNote, reappliedGenerate[0].AppliedNote)
	require.Equal(t, applied.SupplierStatusAfter, reappliedGenerate[0].SupplierStatusAfter)
}

func TestGenerateSupplierEvaluationsCapsAdmitWithoutPassedSlaRun(t *testing.T) {
	truncateTables(t)

	_, _ = seedSupplierEvaluationScorecard(t, 91.5, SupplierScorecardGradeA)
	seedSupplierEvaluationContract(t)

	evaluations, err := GenerateSupplierEvaluations(SupplierEvaluationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, evaluations, 1)
	require.Equal(t, SupplierEvaluationRecommendationObserve, evaluations[0].Recommendation)
	require.Zero(t, evaluations[0].SlaContractId)
	require.Zero(t, evaluations[0].SlaProbeRunId)
	require.Contains(t, evaluations[0].Reason, "no passed admission probe run")
}

func TestGenerateSupplierEvaluationsLinksPassedSlaAdmissionRun(t *testing.T) {
	truncateTables(t)

	supplier, _ := seedSupplierEvaluationScorecard(t, 91.5, SupplierScorecardGradeA)
	contract := seedSupplierEvaluationContract(t)
	plan, err := GenerateSlaProbePlan(SlaProbePlanGenerateInput{
		ContractId:     contract.Id,
		SupplierId:     supplier.Id,
		SlaTier:        "default",
		ProbeType:      SlaProbeTypeAdmission,
		RouteMode:      SlaProbeRouteModeThroughTokenRouter,
		PromptSuiteKey: "unit-admission",
		SampleSize:     2,
		RepeatCount:    1,
	}, 1)
	require.NoError(t, err)
	run, err := RecordSlaProbeRun(SlaProbeRunRecordInput{
		RunKey:         "unit-sla-run",
		PlanId:         plan.Id,
		Status:         SlaProbeRunStatusPassed,
		RunnerVersion:  "token-router-sla/unit",
		RuntimeRef:     "unit",
		Endpoint:       "mock://unit",
		SummaryJSON:    `{"ttft_ms":{"p90":500},"usage":{"prompt_tokens":200}}`,
		HardGatePassed: true,
		ArtifactURI:    "file:///tmp/unit-sla-run.json",
		ArtifactSHA256: "unit-sha",
	}, 1)
	require.NoError(t, err)

	evaluations, err := GenerateSupplierEvaluations(SupplierEvaluationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, evaluations, 1)
	require.Equal(t, SupplierEvaluationRecommendationAdmit, evaluations[0].Recommendation)
	require.Equal(t, contract.Id, evaluations[0].SlaContractId)
	require.Equal(t, run.Id, evaluations[0].SlaProbeRunId)
	require.Contains(t, evaluations[0].Reason, "SLA admission evidence run")

	var gateSummary map[string]any
	require.NoError(t, json.Unmarshal([]byte(evaluations[0].SlaGateSummaryJSON), &gateSummary))
	require.Equal(t, float64(contract.Id), gateSummary["contract_id"])
	require.Equal(t, float64(run.Id), gateSummary["probe_run_id"])
	require.Equal(t, "unit-sha", gateSummary["artifact_sha256"])
	require.Contains(t, gateSummary["summary_json"], "ttft_ms")
}

func TestGenerateSupplierEvaluationsWithoutScorecardsReturnsEmpty(t *testing.T) {
	truncateTables(t)

	evaluations, err := GenerateSupplierEvaluations(SupplierEvaluationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Empty(t, evaluations)
}

func seedSupplierEvaluationScorecard(t *testing.T, score float64, grade string) (*Supplier, *SupplierScorecard) {
	t.Helper()
	supplier := &Supplier{Name: "gb10-4t", Type: SupplierTypeThirdParty}
	require.NoError(t, supplier.Insert())
	scorecard := &SupplierScorecard{
		SupplierId:            supplier.Id,
		PeriodStart:           100,
		PeriodEnd:             200,
		TotalRequests:         2,
		SuccessRequests:       2,
		SuccessRate:           1,
		AvgLatencyMs:          35,
		CacheHitCount:         1,
		CacheHitRate:          0.5,
		TotalSellQuota:        228,
		TotalCostQuota:        112,
		GrossProfitQuota:      116,
		SupplyCapacityTokens:  1000,
		SupplyUsedTokens:      300,
		SupplyHeadroomTokens:  700,
		AvgSupplyQualityScore: 98.5,
		AvgUnitCostQuota:      0.5,
		Score:                 score,
		Grade:                 grade,
		GeneratedAt:           123,
		CreatedAt:             123,
		UpdatedAt:             123,
	}
	require.NoError(t, DB.Create(scorecard).Error)
	return supplier, scorecard
}

func seedSupplierEvaluationContract(t *testing.T) *SlaContract {
	t.Helper()
	contract, err := ImportSlaContract(SlaContractImportInput{
		ContractKey:            "unit-sla-contract",
		ModelName:              "gpt-test",
		ProviderFamily:         "unit",
		SourceName:             "unit contract",
		SourceRef:              "unit://contract",
		SourceSHA256:           "unit-sha",
		Version:                "2026-06",
		Status:                 SlaContractStatusActive,
		MeasurementProfileJSON: `{"input_profile":{"tokens":128},"output_profile":{"target_tokens":16},"cache_profile":"cold_no_cache"}`,
		HardGateJSON:           `{"ttft_ms":{"p90_lte":1000}}`,
		SoftGateJSON:           `{}`,
	}, 1)
	require.NoError(t, err)
	return contract
}
