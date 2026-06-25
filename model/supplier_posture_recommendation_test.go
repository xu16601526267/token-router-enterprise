package model

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestGenerateSupplierPostureRecommendationsDisableAndPreserveApply(t *testing.T) {
	truncateTables(t)

	supplier, scorecard := seedSupplierEvaluationScorecard(t, 42.5, SupplierScorecardGradeD)
	seedSupplierPostureInsight(t, supplier.Id, OperatingInsightCategoryCapacityRisk, OperatingInsightSeverityAction, "capacity:supplier:%d|node:hot|model:gpt-test|reason:high_gpu")
	seedSupplierPostureInsight(t, supplier.Id, OperatingInsightCategoryQualityWatch, OperatingInsightSeverityAction, "sla_probe:run|supplier:%d|channel:1")
	seedSupplierPostureInsight(t, supplier.Id*10, OperatingInsightCategoryCapacityRisk, OperatingInsightSeverityAction, "capacity:supplier:%d|node:other|model:gpt-test|reason:high_gpu")

	recommendations, err := GenerateSupplierPostureRecommendations(SupplierPostureRecommendationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, recommendations, 1)
	require.Equal(t, supplier.Id, recommendations[0].SupplierId)
	require.Equal(t, scorecard.Id, recommendations[0].SupplierScorecardId)
	require.Equal(t, SupplierPostureRecommendationStatusDraft, recommendations[0].Status)
	require.Equal(t, SupplierPostureRecommendationActionDisable, recommendations[0].RecommendedAction)
	require.Equal(t, SupplierScorecardGradeD, recommendations[0].Grade)
	require.InDelta(t, 42.5, recommendations[0].Score, 0.000001)
	require.Equal(t, 1, recommendations[0].QualityInsightCount)
	require.Equal(t, 1, recommendations[0].CapacityInsightCount)
	require.Equal(t, 2, recommendations[0].ActionInsightCount)
	require.Equal(t, common.ChannelStatusEnabled, recommendations[0].SupplierStatusCurrent)
	require.Contains(t, recommendations[0].Reason, "disable review threshold")

	var evidence map[string]any
	require.NoError(t, json.Unmarshal([]byte(recommendations[0].EvidenceJSON), &evidence))
	require.Equal(t, float64(scorecard.Id), evidence["scorecard_id"])
	require.Equal(t, float64(1), evidence["quality_insight_count"])
	require.Equal(t, float64(1), evidence["capacity_insight_count"])

	_, err = ApplySupplierPostureRecommendation(recommendations[0].Id, 1, "too early")
	require.ErrorContains(t, err, "must be approved")

	approved, err := UpdateSupplierPostureRecommendationReview(recommendations[0].Id, SupplierPostureRecommendationStatusApproved, 1, "approved severe posture review")
	require.NoError(t, err)
	require.Equal(t, SupplierPostureRecommendationStatusApproved, approved.Status)
	require.Equal(t, 1, approved.ReviewedBy)
	require.Greater(t, approved.ReviewedAt, int64(0))

	regenerated, err := GenerateSupplierPostureRecommendations(SupplierPostureRecommendationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, regenerated, 1)
	require.Equal(t, approved.Id, regenerated[0].Id)
	require.Equal(t, SupplierPostureRecommendationStatusApproved, regenerated[0].Status)
	require.Equal(t, "approved severe posture review", regenerated[0].ReviewNote)

	applied, err := ApplySupplierPostureRecommendation(regenerated[0].Id, 2, "disable until next SLA pass")
	require.NoError(t, err)
	require.Equal(t, SupplierPostureRecommendationStatusApplied, applied.Status)
	require.Equal(t, 2, applied.AppliedBy)
	require.Greater(t, applied.AppliedAt, int64(0))
	require.Equal(t, common.ChannelStatusEnabled, applied.SupplierStatusBefore)
	require.Equal(t, common.ChannelStatusManuallyDisabled, applied.SupplierStatusAfter)
	require.Contains(t, applied.AppliedNote, "action=disable")
	require.Contains(t, applied.AppliedNote, "disable until next SLA pass")

	var savedSupplier Supplier
	require.NoError(t, DB.First(&savedSupplier, supplier.Id).Error)
	require.Equal(t, common.ChannelStatusManuallyDisabled, savedSupplier.Status)
	require.Contains(t, savedSupplier.Notes, "supplier_posture_recommendation #")
	require.Contains(t, savedSupplier.Notes, "disable until next SLA pass")
	_, err = GetActiveSupplierRoutePreferenceBySupplierID(supplier.Id)
	require.Error(t, err)

	_, err = ApplySupplierPostureRecommendation(applied.Id, 2, "duplicate")
	require.ErrorContains(t, err, "must be approved")

	_, err = UpdateSupplierPostureRecommendationReview(applied.Id, SupplierPostureRecommendationStatusRejected, 3, "late rejection")
	require.ErrorContains(t, err, "cannot be reviewed")

	reappliedGenerate, err := GenerateSupplierPostureRecommendations(SupplierPostureRecommendationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, reappliedGenerate, 1)
	require.Equal(t, SupplierPostureRecommendationStatusApplied, reappliedGenerate[0].Status)
	require.Equal(t, SupplierPostureRecommendationActionDisable, reappliedGenerate[0].RecommendedAction)
	require.Equal(t, applied.AppliedAt, reappliedGenerate[0].AppliedAt)
	require.Equal(t, applied.AppliedBy, reappliedGenerate[0].AppliedBy)
	require.Equal(t, applied.AppliedNote, reappliedGenerate[0].AppliedNote)
	require.Equal(t, applied.SupplierStatusAfter, reappliedGenerate[0].SupplierStatusAfter)
	require.Equal(t, common.ChannelStatusManuallyDisabled, reappliedGenerate[0].SupplierStatusCurrent)
}

func TestApplySupplierPostureRecommendationDowngradeKeepsSupplierEnabled(t *testing.T) {
	truncateTables(t)

	supplier, _ := seedSupplierEvaluationScorecard(t, 65, SupplierScorecardGradeC)
	seedSupplierPostureInsight(t, supplier.Id, OperatingInsightCategoryQualityWatch, OperatingInsightSeverityWatch, "sla_probe:run|supplier:%d|channel:1")

	recommendations, err := GenerateSupplierPostureRecommendations(SupplierPostureRecommendationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, recommendations, 1)
	require.Equal(t, SupplierPostureRecommendationActionDowngrade, recommendations[0].RecommendedAction)

	approved, err := UpdateSupplierPostureRecommendationReview(recommendations[0].Id, SupplierPostureRecommendationStatusApproved, 1, "review downgrade posture")
	require.NoError(t, err)
	applied, err := ApplySupplierPostureRecommendation(approved.Id, 2, "operator will reduce reliance manually")
	require.NoError(t, err)
	require.Equal(t, SupplierPostureRecommendationStatusApplied, applied.Status)
	require.Equal(t, common.ChannelStatusEnabled, applied.SupplierStatusBefore)
	require.Equal(t, common.ChannelStatusEnabled, applied.SupplierStatusAfter)

	var savedSupplier Supplier
	require.NoError(t, DB.First(&savedSupplier, supplier.Id).Error)
	require.Equal(t, common.ChannelStatusEnabled, savedSupplier.Status)
	require.Contains(t, savedSupplier.Notes, "action=downgrade")
	require.Contains(t, savedSupplier.Notes, "operator will reduce reliance manually")

	preference, err := GetActiveSupplierRoutePreferenceBySupplierID(supplier.Id)
	require.NoError(t, err)
	require.Equal(t, supplier.Id, preference.SupplierId)
	require.Equal(t, applied.Id, preference.SourcePostureRecommendationId)
	require.Equal(t, SupplierRoutePreferenceStatusActive, preference.Status)
	require.Equal(t, SupplierRoutePreferenceDowngradeWeightPercent, preference.WeightPercent)
	require.Contains(t, preference.Reason, "supplier_posture_recommendation #")
	require.Contains(t, preference.OperatorNote, "operator will reduce reliance manually")
}

func TestApplySupplierPostureRecommendationBoostKeepsSupplierEnabled(t *testing.T) {
	truncateTables(t)

	supplier, _ := seedSupplierEvaluationScorecard(t, 94, SupplierScorecardGradeA)

	recommendations, err := GenerateSupplierPostureRecommendations(SupplierPostureRecommendationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, recommendations, 1)
	require.Equal(t, SupplierPostureRecommendationActionBoost, recommendations[0].RecommendedAction)
	require.Contains(t, recommendations[0].Reason, "boost review threshold")

	approved, err := UpdateSupplierPostureRecommendationReview(recommendations[0].Id, SupplierPostureRecommendationStatusApproved, 1, "review boost posture")
	require.NoError(t, err)
	applied, err := ApplySupplierPostureRecommendation(approved.Id, 2, "operator will increase reliance")
	require.NoError(t, err)
	require.Equal(t, SupplierPostureRecommendationStatusApplied, applied.Status)
	require.Equal(t, common.ChannelStatusEnabled, applied.SupplierStatusBefore)
	require.Equal(t, common.ChannelStatusEnabled, applied.SupplierStatusAfter)

	var savedSupplier Supplier
	require.NoError(t, DB.First(&savedSupplier, supplier.Id).Error)
	require.Equal(t, common.ChannelStatusEnabled, savedSupplier.Status)
	require.Contains(t, savedSupplier.Notes, "action=boost")
	require.Contains(t, savedSupplier.Notes, "operator will increase reliance")

	preference, err := GetActiveSupplierRoutePreferenceBySupplierID(supplier.Id)
	require.NoError(t, err)
	require.Equal(t, supplier.Id, preference.SupplierId)
	require.Equal(t, applied.Id, preference.SourcePostureRecommendationId)
	require.Equal(t, SupplierRoutePreferenceStatusActive, preference.Status)
	require.Equal(t, SupplierRoutePreferenceBoostWeightPercent, preference.WeightPercent)
	require.Contains(t, preference.Reason, "supplier_posture_recommendation #")
	require.Contains(t, preference.Reason, "boost")
	require.Contains(t, preference.OperatorNote, "operator will increase reliance")
}

func TestApplySupplierPostureRecommendationObserveClearsRoutePreference(t *testing.T) {
	truncateTables(t)

	supplier, _ := seedSupplierEvaluationScorecard(t, 82, SupplierScorecardGradeB)
	existing := &SupplierRoutePreference{
		SupplierId:                    supplier.Id,
		SourcePostureRecommendationId: 999,
		Status:                        SupplierRoutePreferenceStatusActive,
		WeightPercent:                 SupplierRoutePreferenceDowngradeWeightPercent,
		Reason:                        "previous downgrade",
		EffectiveFrom:                 100,
		ActivatedAt:                   100,
		ActivatedBy:                   1,
		CreatedAt:                     100,
		UpdatedAt:                     100,
	}
	require.NoError(t, DB.Create(existing).Error)

	recommendations, err := GenerateSupplierPostureRecommendations(SupplierPostureRecommendationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Len(t, recommendations, 1)
	require.Equal(t, SupplierPostureRecommendationActionObserve, recommendations[0].RecommendedAction)

	approved, err := UpdateSupplierPostureRecommendationReview(recommendations[0].Id, SupplierPostureRecommendationStatusApproved, 1, "observe posture review")
	require.NoError(t, err)
	applied, err := ApplySupplierPostureRecommendation(approved.Id, 2, "restore normal routing")
	require.NoError(t, err)
	require.Equal(t, SupplierPostureRecommendationStatusApplied, applied.Status)

	_, err = GetActiveSupplierRoutePreferenceBySupplierID(supplier.Id)
	require.Error(t, err)

	var savedPreference SupplierRoutePreference
	require.NoError(t, DB.First(&savedPreference, "supplier_id = ?", supplier.Id).Error)
	require.Equal(t, SupplierRoutePreferenceStatusDisabled, savedPreference.Status)
	require.Equal(t, 100, savedPreference.WeightPercent)
	require.Equal(t, applied.Id, savedPreference.SourcePostureRecommendationId)
	require.Equal(t, 2, savedPreference.DisabledBy)
	require.Contains(t, savedPreference.OperatorNote, "restore normal routing")
}

func TestGenerateSupplierPostureRecommendationsWithoutScorecardsReturnsEmpty(t *testing.T) {
	truncateTables(t)

	recommendations, err := GenerateSupplierPostureRecommendations(SupplierPostureRecommendationGenerateInput{
		PeriodStart: 100,
		PeriodEnd:   200,
	})
	require.NoError(t, err)
	require.Empty(t, recommendations)
}

func seedSupplierPostureInsight(t *testing.T, supplierID int, category string, severity string, sliceKeyFormat string) *OperatingInsight {
	t.Helper()
	insight := &OperatingInsight{
		InsightKey:        fmt.Sprintf("posture-insight:%d:%s:%s:%s", supplierID, category, severity, sliceKeyFormat),
		SliceKey:          fmt.Sprintf(sliceKeyFormat, supplierID),
		ModelName:         "gpt-test",
		SlaTier:           "default",
		PeriodStart:       100,
		PeriodEnd:         200,
		Status:            OperatingInsightStatusDraft,
		Severity:          severity,
		Category:          category,
		Title:             "supplier posture test insight",
		Summary:           "supplier posture evidence",
		RecommendedAction: "review supplier posture",
		GeneratedAt:       123,
		CreatedAt:         123,
		UpdatedAt:         123,
	}
	require.NoError(t, DB.Create(insight).Error)
	return insight
}
