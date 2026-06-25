package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SupplierEvaluationTypeAdmission = "admission"

	SupplierEvaluationStatusDraft    = "draft"
	SupplierEvaluationStatusApproved = "approved"
	SupplierEvaluationStatusRejected = "rejected"

	SupplierEvaluationRecommendationAdmit   = "admit"
	SupplierEvaluationRecommendationObserve = "observe"
	SupplierEvaluationRecommendationReject  = "reject"
)

type SupplierEvaluation struct {
	Id                    int     `json:"id"`
	EvaluationKey         string  `json:"evaluation_key" gorm:"size:512;not null;uniqueIndex:uk_supplier_evaluation_key"`
	EvaluationType        string  `json:"evaluation_type" gorm:"size:32;not null;default:'admission';index"`
	SupplierId            int     `json:"supplier_id" gorm:"not null;index"`
	SupplierScorecardId   int     `json:"supplier_scorecard_id" gorm:"not null;index"`
	SlaContractId         int     `json:"sla_contract_id" gorm:"default:0;index"`
	SlaProbeRunId         int     `json:"sla_probe_run_id" gorm:"default:0;index"`
	SlaGateSummaryJSON    string  `json:"sla_gate_summary_json" gorm:"type:text"`
	PeriodStart           int64   `json:"period_start" gorm:"bigint;not null;index"`
	PeriodEnd             int64   `json:"period_end" gorm:"bigint;not null;index"`
	Status                string  `json:"status" gorm:"size:32;not null;default:'draft';index"`
	Recommendation        string  `json:"recommendation" gorm:"size:32;not null;index"`
	Score                 float64 `json:"score" gorm:"default:0;index"`
	Grade                 string  `json:"grade" gorm:"size:8;not null;default:'D';index"`
	TotalRequests         int64   `json:"total_requests" gorm:"default:0"`
	SuccessRate           float64 `json:"success_rate" gorm:"default:0"`
	AvgLatencyMs          float64 `json:"avg_latency_ms" gorm:"default:0"`
	CacheHitRate          float64 `json:"cache_hit_rate" gorm:"default:0"`
	GrossProfitQuota      int64   `json:"gross_profit_quota" gorm:"default:0"`
	SupplyHeadroomTokens  int64   `json:"supply_headroom_tokens" gorm:"default:0"`
	AvgSupplyQualityScore float64 `json:"avg_supply_quality_score" gorm:"default:0"`
	AvgUnitCostQuota      float64 `json:"avg_unit_cost_quota" gorm:"default:0"`
	Reason                string  `json:"reason" gorm:"type:text"`
	GeneratedAt           int64   `json:"generated_at" gorm:"bigint;index"`
	ReviewedAt            int64   `json:"reviewed_at" gorm:"bigint;default:0"`
	ReviewedBy            int     `json:"reviewed_by" gorm:"default:0;index"`
	ReviewNote            string  `json:"review_note,omitempty" gorm:"type:text"`
	AppliedAt             int64   `json:"applied_at" gorm:"bigint;default:0;index"`
	AppliedBy             int     `json:"applied_by" gorm:"default:0;index"`
	AppliedNote           string  `json:"applied_note,omitempty" gorm:"type:text"`
	SupplierStatusBefore  int     `json:"supplier_status_before" gorm:"default:0"`
	SupplierStatusAfter   int     `json:"supplier_status_after" gorm:"default:0"`
	CreatedAt             int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt             int64   `json:"updated_at" gorm:"bigint"`
}

type SupplierEvaluationGenerateInput struct {
	PeriodStart int64 `json:"period_start"`
	PeriodEnd   int64 `json:"period_end"`
	SupplierId  int   `json:"supplier_id"`
}

type SupplierEvaluationFilters struct {
	SupplierId     int
	EvaluationType string
	Status         string
	Recommendation string
	Grade          string
	StartTime      int64
	EndTime        int64
}

type SupplierEvaluationReviewInput struct {
	ReviewNote string `json:"review_note"`
}

type SupplierEvaluationApplyInput struct {
	OperatorNote string `json:"operator_note"`
}

type supplierEvaluationSlaEvidence struct {
	Required bool
	Run      *SlaProbeRun
}

func validateSupplierEvaluationGenerateInput(input SupplierEvaluationGenerateInput) error {
	if input.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if input.PeriodEnd <= input.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	return nil
}

func normalizeSupplierEvaluationType(value string) string {
	switch strings.TrimSpace(value) {
	case "", SupplierEvaluationTypeAdmission:
		return SupplierEvaluationTypeAdmission
	default:
		return ""
	}
}

func normalizeSupplierEvaluationStatus(value string) string {
	switch strings.TrimSpace(value) {
	case SupplierEvaluationStatusDraft:
		return SupplierEvaluationStatusDraft
	case SupplierEvaluationStatusApproved:
		return SupplierEvaluationStatusApproved
	case SupplierEvaluationStatusRejected:
		return SupplierEvaluationStatusRejected
	default:
		return ""
	}
}

func normalizeSupplierEvaluationRecommendation(value string) string {
	switch strings.TrimSpace(value) {
	case SupplierEvaluationRecommendationAdmit:
		return SupplierEvaluationRecommendationAdmit
	case SupplierEvaluationRecommendationObserve:
		return SupplierEvaluationRecommendationObserve
	case SupplierEvaluationRecommendationReject:
		return SupplierEvaluationRecommendationReject
	default:
		return ""
	}
}

func supplierEvaluationKey(scorecard SupplierScorecard) string {
	return fmt.Sprintf("%s:scorecard:%d", SupplierEvaluationTypeAdmission, scorecard.Id)
}

func SearchSupplierEvaluations(filters SupplierEvaluationFilters, offset int, limit int) ([]*SupplierEvaluation, int64, error) {
	db := DB.Model(&SupplierEvaluation{})
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if evaluationType := normalizeSupplierEvaluationType(filters.EvaluationType); evaluationType != "" {
		db = db.Where("evaluation_type = ?", evaluationType)
	}
	if status := normalizeSupplierEvaluationStatus(filters.Status); status != "" {
		db = db.Where("status = ?", status)
	}
	if recommendation := normalizeSupplierEvaluationRecommendation(filters.Recommendation); recommendation != "" {
		db = db.Where("recommendation = ?", recommendation)
	}
	if grade := normalizeSupplierScorecardGrade(filters.Grade); grade != "" {
		db = db.Where("grade = ?", grade)
	}
	if filters.StartTime > 0 {
		db = db.Where("period_end >= ?", filters.StartTime)
	}
	if filters.EndTime > 0 {
		db = db.Where("period_start <= ?", filters.EndTime)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var evaluations []*SupplierEvaluation
	err := db.Offset(offset).Limit(limit).Order("period_start DESC, score DESC, id DESC").Find(&evaluations).Error
	return evaluations, total, err
}

func GenerateSupplierEvaluations(input SupplierEvaluationGenerateInput) ([]*SupplierEvaluation, error) {
	if err := validateSupplierEvaluationGenerateInput(input); err != nil {
		return nil, err
	}

	scorecardDB := DB.Model(&SupplierScorecard{}).
		Where("period_start = ? AND period_end = ?", input.PeriodStart, input.PeriodEnd)
	if input.SupplierId > 0 {
		scorecardDB = scorecardDB.Where("supplier_id = ?", input.SupplierId)
	}

	var scorecards []SupplierScorecard
	if err := scorecardDB.Order("score DESC, id ASC").Find(&scorecards).Error; err != nil {
		return nil, err
	}
	if len(scorecards) == 0 {
		return []*SupplierEvaluation{}, nil
	}

	now := common.GetTimestamp()
	evaluations := make([]SupplierEvaluation, 0, len(scorecards))
	for _, scorecard := range scorecards {
		slaEvidence, err := findSupplierEvaluationSlaEvidence(scorecard)
		if err != nil {
			return nil, err
		}
		evaluations = append(evaluations, buildSupplierEvaluationFromScorecard(scorecard, slaEvidence, now))
	}

	err := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "evaluation_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"evaluation_type",
			"supplier_id",
			"supplier_scorecard_id",
			"sla_contract_id",
			"sla_probe_run_id",
			"sla_gate_summary_json",
			"period_start",
			"period_end",
			"recommendation",
			"score",
			"grade",
			"total_requests",
			"success_rate",
			"avg_latency_ms",
			"cache_hit_rate",
			"gross_profit_quota",
			"supply_headroom_tokens",
			"avg_supply_quality_score",
			"avg_unit_cost_quota",
			"reason",
			"generated_at",
			"updated_at",
		}),
	}).Create(&evaluations).Error
	if err != nil {
		return nil, err
	}

	resultDB := DB.Model(&SupplierEvaluation{}).
		Where("period_start = ? AND period_end = ?", input.PeriodStart, input.PeriodEnd)
	if input.SupplierId > 0 {
		resultDB = resultDB.Where("supplier_id = ?", input.SupplierId)
	}
	var results []*SupplierEvaluation
	err = resultDB.Order("period_start DESC, score DESC, id DESC").Find(&results).Error
	return results, err
}

func buildSupplierEvaluationFromScorecard(scorecard SupplierScorecard, slaEvidence supplierEvaluationSlaEvidence, now int64) SupplierEvaluation {
	recommendation := supplierEvaluationRecommendation(scorecard.Score)
	if recommendation == SupplierEvaluationRecommendationAdmit && slaEvidence.Required && slaEvidence.Run == nil {
		recommendation = SupplierEvaluationRecommendationObserve
	}
	slaContractId := 0
	slaProbeRunId := 0
	slaGateSummaryJSON := ""
	if slaEvidence.Run != nil {
		slaContractId = slaEvidence.Run.ContractId
		slaProbeRunId = slaEvidence.Run.Id
		slaGateSummaryJSON = supplierEvaluationSlaGateSummary(*slaEvidence.Run)
	}
	return SupplierEvaluation{
		EvaluationKey:         supplierEvaluationKey(scorecard),
		EvaluationType:        SupplierEvaluationTypeAdmission,
		SupplierId:            scorecard.SupplierId,
		SupplierScorecardId:   scorecard.Id,
		SlaContractId:         slaContractId,
		SlaProbeRunId:         slaProbeRunId,
		SlaGateSummaryJSON:    slaGateSummaryJSON,
		PeriodStart:           scorecard.PeriodStart,
		PeriodEnd:             scorecard.PeriodEnd,
		Status:                SupplierEvaluationStatusDraft,
		Recommendation:        recommendation,
		Score:                 scorecard.Score,
		Grade:                 scorecard.Grade,
		TotalRequests:         scorecard.TotalRequests,
		SuccessRate:           scorecard.SuccessRate,
		AvgLatencyMs:          scorecard.AvgLatencyMs,
		CacheHitRate:          scorecard.CacheHitRate,
		GrossProfitQuota:      scorecard.GrossProfitQuota,
		SupplyHeadroomTokens:  scorecard.SupplyHeadroomTokens,
		AvgSupplyQualityScore: scorecard.AvgSupplyQualityScore,
		AvgUnitCostQuota:      scorecard.AvgUnitCostQuota,
		Reason:                supplierEvaluationReason(recommendation, scorecard, slaEvidence),
		GeneratedAt:           now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}

func supplierEvaluationRecommendation(score float64) string {
	switch {
	case score >= 85:
		return SupplierEvaluationRecommendationAdmit
	case score >= 70:
		return SupplierEvaluationRecommendationObserve
	default:
		return SupplierEvaluationRecommendationReject
	}
}

func supplierEvaluationReason(recommendation string, scorecard SupplierScorecard, slaEvidence supplierEvaluationSlaEvidence) string {
	slaSuffix := ""
	if slaEvidence.Required {
		if slaEvidence.Run != nil {
			slaSuffix = fmt.Sprintf("; SLA admission evidence run #%d contract #%d hard gate passed", slaEvidence.Run.Id, slaEvidence.Run.ContractId)
		} else {
			slaSuffix = "; active SLA contract exists but no passed admission probe run is recorded for this supplier"
		}
	}
	switch recommendation {
	case SupplierEvaluationRecommendationAdmit:
		return fmt.Sprintf("scorecard grade %s score %.3f meets admission threshold%s", scorecard.Grade, scorecard.Score, slaSuffix)
	case SupplierEvaluationRecommendationObserve:
		return fmt.Sprintf("scorecard grade %s score %.3f is below admission threshold or lacks required SLA evidence; keep supplier under observation%s", scorecard.Grade, scorecard.Score, slaSuffix)
	default:
		return fmt.Sprintf("scorecard grade %s score %.3f is below minimum admission threshold%s", scorecard.Grade, scorecard.Score, slaSuffix)
	}
}

func findSupplierEvaluationSlaEvidence(scorecard SupplierScorecard) (supplierEvaluationSlaEvidence, error) {
	required, err := hasActiveSlaContract()
	if err != nil {
		return supplierEvaluationSlaEvidence{}, err
	}
	evidence := supplierEvaluationSlaEvidence{Required: required}
	if !required {
		return evidence, nil
	}

	var run SlaProbeRun
	query := DB.Model(&SlaProbeRun{}).
		Joins("JOIN sla_probe_plans ON sla_probe_plans.id = sla_probe_runs.plan_id").
		Joins("JOIN sla_contracts ON sla_contracts.id = sla_probe_runs.contract_id").
		Where("sla_probe_runs.supplier_id = ?", scorecard.SupplierId).
		Where("sla_probe_runs.status = ?", SlaProbeRunStatusPassed).
		Where("sla_probe_runs.hard_gate_passed = ?", true).
		Where("sla_probe_plans.probe_type = ?", SlaProbeTypeAdmission).
		Where("sla_contracts.status = ?", SlaContractStatusActive)
	err = query.
		Order("sla_probe_runs.recorded_at DESC, sla_probe_runs.ended_at DESC, sla_probe_runs.id DESC").
		First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return evidence, nil
	}
	if err != nil {
		return evidence, err
	}
	evidence.Run = &run
	return evidence, nil
}

func hasActiveSlaContract() (bool, error) {
	var count int64
	err := DB.Model(&SlaContract{}).Where("status = ?", SlaContractStatusActive).Count(&count).Error
	return count > 0, err
}

func supplierEvaluationSlaGateSummary(run SlaProbeRun) string {
	summary := map[string]any{
		"contract_id":       run.ContractId,
		"probe_run_id":      run.Id,
		"run_key":           run.RunKey,
		"status":            run.Status,
		"hard_gate_passed":  run.HardGatePassed,
		"route_mode":        run.RouteMode,
		"model_name":        run.ModelName,
		"sla_tier":          run.SlaTier,
		"runner_version":    run.RunnerVersion,
		"runtime_ref":       run.RuntimeRef,
		"artifact_uri":      run.ArtifactURI,
		"artifact_sha256":   run.ArtifactSHA256,
		"recorded_at":       run.RecordedAt,
		"summary_json":      run.SummaryJSON,
		"failure_reasons":   run.FailureReasons,
		"soft_gate_warning": run.SoftGateWarnings,
	}
	encoded, err := json.Marshal(summary)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func UpdateSupplierEvaluationReview(id int, status string, reviewedBy int, reviewNote string) (*SupplierEvaluation, error) {
	status = normalizeSupplierEvaluationStatus(status)
	if status == "" || status == SupplierEvaluationStatusDraft {
		return nil, errors.New("review status must be approved or rejected")
	}
	var evaluation SupplierEvaluation
	if err := DB.Select("id", "applied_at").First(&evaluation, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if evaluation.AppliedAt > 0 {
		return nil, errors.New("applied supplier evaluation cannot be reviewed")
	}
	now := common.GetTimestamp()
	err := DB.Model(&SupplierEvaluation{}).Where("id = ?", id).Updates(map[string]any{
		"status":      status,
		"reviewed_at": now,
		"reviewed_by": reviewedBy,
		"review_note": strings.TrimSpace(reviewNote),
		"updated_at":  now,
	}).Error
	if err != nil {
		return nil, err
	}
	return GetSupplierEvaluationByID(id)
}

func ApplySupplierEvaluation(id int, appliedBy int, operatorNote string) (*SupplierEvaluation, error) {
	operatorNote = strings.TrimSpace(operatorNote)
	now := common.GetTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		var evaluation SupplierEvaluation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&evaluation, "id = ?", id).Error; err != nil {
			return err
		}
		if evaluation.Status != SupplierEvaluationStatusApproved {
			return errors.New("supplier evaluation must be approved before apply")
		}
		if evaluation.AppliedAt > 0 {
			return errors.New("supplier evaluation already applied")
		}

		var supplier Supplier
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&supplier, "id = ?", evaluation.SupplierId).Error; err != nil {
			return err
		}
		targetStatus, err := supplierEvaluationTargetSupplierStatus(evaluation.Recommendation, supplier.Status)
		if err != nil {
			return err
		}

		statusBefore := supplier.Status
		appliedNote := supplierEvaluationApplyNote(evaluation, appliedBy, operatorNote)
		supplierNotes := appendSupplierEvaluationNote(supplier.Notes, appliedNote)
		if err := tx.Model(&Supplier{}).Where("id = ?", supplier.Id).Updates(map[string]any{
			"status":       targetStatus,
			"notes":        supplierNotes,
			"updated_time": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&SupplierEvaluation{}).Where("id = ?", evaluation.Id).Updates(map[string]any{
			"applied_at":             now,
			"applied_by":             appliedBy,
			"applied_note":           appliedNote,
			"supplier_status_before": statusBefore,
			"supplier_status_after":  targetStatus,
			"updated_at":             now,
		}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	InitChannelCache()
	return GetSupplierEvaluationByID(id)
}

func supplierEvaluationTargetSupplierStatus(recommendation string, currentStatus int) (int, error) {
	switch recommendation {
	case SupplierEvaluationRecommendationAdmit:
		return common.ChannelStatusEnabled, nil
	case SupplierEvaluationRecommendationObserve:
		return currentStatus, nil
	case SupplierEvaluationRecommendationReject:
		return common.ChannelStatusManuallyDisabled, nil
	default:
		return 0, errors.New("invalid supplier evaluation recommendation")
	}
}

func supplierEvaluationApplyNote(evaluation SupplierEvaluation, appliedBy int, operatorNote string) string {
	note := fmt.Sprintf(
		"supplier_evaluation #%d applied by user #%d: recommendation=%s grade=%s score=%.3f",
		evaluation.Id,
		appliedBy,
		evaluation.Recommendation,
		evaluation.Grade,
		evaluation.Score,
	)
	if operatorNote == "" {
		return note
	}
	return fmt.Sprintf("%s; note=%s", note, operatorNote)
}

func appendSupplierEvaluationNote(existing string, addition string) string {
	existing = strings.TrimSpace(existing)
	if existing == "" {
		return addition
	}
	return existing + "\n" + addition
}

func GetSupplierEvaluationByID(id int) (*SupplierEvaluation, error) {
	var evaluation SupplierEvaluation
	err := DB.First(&evaluation, "id = ?", id).Error
	return &evaluation, err
}
