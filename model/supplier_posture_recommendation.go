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
	SupplierPostureRecommendationStatusDraft    = "draft"
	SupplierPostureRecommendationStatusApproved = "approved"
	SupplierPostureRecommendationStatusRejected = "rejected"
	SupplierPostureRecommendationStatusApplied  = "applied"

	SupplierPostureRecommendationActionObserve   = "observe"
	SupplierPostureRecommendationActionBoost     = "boost"
	SupplierPostureRecommendationActionDowngrade = "downgrade"
	SupplierPostureRecommendationActionDisable   = "disable"

	SupplierPostureRecommendationBoostMinScore = 90
)

type SupplierPostureRecommendation struct {
	Id                    int     `json:"id"`
	RecommendationKey     string  `json:"recommendation_key" gorm:"size:512;not null;uniqueIndex:uk_supplier_posture_recommendation_key"`
	SupplierId            int     `json:"supplier_id" gorm:"not null;index"`
	SupplierScorecardId   int     `json:"supplier_scorecard_id" gorm:"not null;index"`
	PeriodStart           int64   `json:"period_start" gorm:"bigint;not null;index"`
	PeriodEnd             int64   `json:"period_end" gorm:"bigint;not null;index"`
	Status                string  `json:"status" gorm:"size:32;not null;default:'draft';index"`
	RecommendedAction     string  `json:"recommended_action" gorm:"size:32;not null;index"`
	Score                 float64 `json:"score" gorm:"default:0;index"`
	Grade                 string  `json:"grade" gorm:"size:8;not null;default:'D';index"`
	TotalRequests         int64   `json:"total_requests" gorm:"default:0"`
	SuccessRate           float64 `json:"success_rate" gorm:"default:0"`
	AvgLatencyMs          float64 `json:"avg_latency_ms" gorm:"default:0"`
	SupplyHeadroomTokens  int64   `json:"supply_headroom_tokens" gorm:"default:0"`
	AvgSupplyQualityScore float64 `json:"avg_supply_quality_score" gorm:"default:0"`
	QualityInsightCount   int     `json:"quality_insight_count" gorm:"default:0"`
	CapacityInsightCount  int     `json:"capacity_insight_count" gorm:"default:0"`
	ActionInsightCount    int     `json:"action_insight_count" gorm:"default:0"`
	SupplierStatusCurrent int     `json:"supplier_status_current" gorm:"default:0;index"`
	Reason                string  `json:"reason" gorm:"type:text"`
	EvidenceJSON          string  `json:"evidence_json" gorm:"type:text"`
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

type SupplierPostureRecommendationGenerateInput struct {
	PeriodStart int64 `json:"period_start"`
	PeriodEnd   int64 `json:"period_end"`
	SupplierId  int   `json:"supplier_id"`
}

type SupplierPostureRecommendationFilters struct {
	SupplierId        int
	Status            string
	RecommendedAction string
	Grade             string
	StartTime         int64
	EndTime           int64
}

type SupplierPostureRecommendationReviewInput struct {
	ReviewNote string `json:"review_note"`
}

type SupplierPostureRecommendationApplyInput struct {
	OperatorNote string `json:"operator_note"`
}

type supplierPostureInsightEvidence struct {
	QualityInsightIDs  []int
	CapacityInsightIDs []int
	ActionInsightIDs   []int
}

func validateSupplierPostureRecommendationGenerateInput(input SupplierPostureRecommendationGenerateInput) error {
	if input.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if input.PeriodEnd <= input.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	return nil
}

func normalizeSupplierPostureRecommendationStatus(value string) string {
	switch strings.TrimSpace(value) {
	case SupplierPostureRecommendationStatusDraft:
		return SupplierPostureRecommendationStatusDraft
	case SupplierPostureRecommendationStatusApproved:
		return SupplierPostureRecommendationStatusApproved
	case SupplierPostureRecommendationStatusRejected:
		return SupplierPostureRecommendationStatusRejected
	case SupplierPostureRecommendationStatusApplied:
		return SupplierPostureRecommendationStatusApplied
	default:
		return ""
	}
}

func normalizeSupplierPostureRecommendationAction(value string) string {
	switch strings.TrimSpace(value) {
	case SupplierPostureRecommendationActionObserve:
		return SupplierPostureRecommendationActionObserve
	case SupplierPostureRecommendationActionBoost:
		return SupplierPostureRecommendationActionBoost
	case SupplierPostureRecommendationActionDowngrade:
		return SupplierPostureRecommendationActionDowngrade
	case SupplierPostureRecommendationActionDisable:
		return SupplierPostureRecommendationActionDisable
	default:
		return ""
	}
}

func supplierPostureRecommendationKey(scorecard SupplierScorecard) string {
	return fmt.Sprintf("posture:scorecard:%d", scorecard.Id)
}

func SearchSupplierPostureRecommendations(filters SupplierPostureRecommendationFilters, offset int, limit int) ([]*SupplierPostureRecommendation, int64, error) {
	db := DB.Model(&SupplierPostureRecommendation{})
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if status := normalizeSupplierPostureRecommendationStatus(filters.Status); status != "" {
		db = db.Where("status = ?", status)
	}
	if action := normalizeSupplierPostureRecommendationAction(filters.RecommendedAction); action != "" {
		db = db.Where("recommended_action = ?", action)
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
	var recommendations []*SupplierPostureRecommendation
	err := db.Offset(offset).Limit(limit).Order("period_start DESC, score ASC, id DESC").Find(&recommendations).Error
	return recommendations, total, err
}

func GenerateSupplierPostureRecommendations(input SupplierPostureRecommendationGenerateInput) ([]*SupplierPostureRecommendation, error) {
	if err := validateSupplierPostureRecommendationGenerateInput(input); err != nil {
		return nil, err
	}

	scorecardDB := DB.Model(&SupplierScorecard{}).
		Where("period_start = ? AND period_end = ?", input.PeriodStart, input.PeriodEnd)
	if input.SupplierId > 0 {
		scorecardDB = scorecardDB.Where("supplier_id = ?", input.SupplierId)
	}

	var scorecards []SupplierScorecard
	if err := scorecardDB.Order("score ASC, id ASC").Find(&scorecards).Error; err != nil {
		return nil, err
	}
	if len(scorecards) == 0 {
		return []*SupplierPostureRecommendation{}, nil
	}

	supplierIDs := make([]int, 0, len(scorecards))
	for _, scorecard := range scorecards {
		supplierIDs = append(supplierIDs, scorecard.SupplierId)
	}
	suppliers, err := loadSuppliersByID(supplierIDs)
	if err != nil {
		return nil, err
	}
	insights, err := loadSupplierPostureInsightEvidence(input, supplierIDs)
	if err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	recommendations := make([]SupplierPostureRecommendation, 0, len(scorecards))
	for _, scorecard := range scorecards {
		supplier, ok := suppliers[scorecard.SupplierId]
		if !ok {
			continue
		}
		recommendations = append(recommendations, buildSupplierPostureRecommendation(scorecard, supplier, insights[scorecard.SupplierId], now))
	}
	if len(recommendations) == 0 {
		return []*SupplierPostureRecommendation{}, nil
	}

	err = DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "recommendation_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"supplier_id",
			"supplier_scorecard_id",
			"period_start",
			"period_end",
			"recommended_action",
			"score",
			"grade",
			"total_requests",
			"success_rate",
			"avg_latency_ms",
			"supply_headroom_tokens",
			"avg_supply_quality_score",
			"quality_insight_count",
			"capacity_insight_count",
			"action_insight_count",
			"supplier_status_current",
			"reason",
			"evidence_json",
			"generated_at",
			"updated_at",
		}),
	}).Create(&recommendations).Error
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(recommendations))
	for _, recommendation := range recommendations {
		keys = append(keys, recommendation.RecommendationKey)
	}
	var results []*SupplierPostureRecommendation
	err = DB.Model(&SupplierPostureRecommendation{}).
		Where("recommendation_key IN ?", keys).
		Order("period_start DESC, score ASC, id DESC").
		Find(&results).Error
	return results, err
}

func loadSuppliersByID(ids []int) (map[int]Supplier, error) {
	if len(ids) == 0 {
		return map[int]Supplier{}, nil
	}
	var suppliers []Supplier
	if err := DB.Where("id IN ?", ids).Find(&suppliers).Error; err != nil {
		return nil, err
	}
	result := make(map[int]Supplier, len(suppliers))
	for _, supplier := range suppliers {
		result[supplier.Id] = supplier
	}
	return result, nil
}

func loadSupplierPostureInsightEvidence(input SupplierPostureRecommendationGenerateInput, supplierIDs []int) (map[int]supplierPostureInsightEvidence, error) {
	result := make(map[int]supplierPostureInsightEvidence, len(supplierIDs))
	if len(supplierIDs) == 0 {
		return result, nil
	}
	var insights []OperatingInsight
	err := DB.Model(&OperatingInsight{}).
		Where("status = ?", OperatingInsightStatusDraft).
		Where("period_end >= ? AND period_start <= ?", input.PeriodStart, input.PeriodEnd).
		Where("category IN ?", []string{OperatingInsightCategoryQualityWatch, OperatingInsightCategoryCapacityRisk}).
		Order("id ASC").
		Find(&insights).Error
	if err != nil {
		return nil, err
	}
	for _, insight := range insights {
		for _, supplierID := range supplierIDs {
			if !supplierPostureInsightMatchesSupplier(insight, supplierID) {
				continue
			}
			evidence := result[supplierID]
			switch insight.Category {
			case OperatingInsightCategoryQualityWatch:
				evidence.QualityInsightIDs = append(evidence.QualityInsightIDs, insight.Id)
			case OperatingInsightCategoryCapacityRisk:
				evidence.CapacityInsightIDs = append(evidence.CapacityInsightIDs, insight.Id)
			}
			if insight.Severity == OperatingInsightSeverityAction {
				evidence.ActionInsightIDs = append(evidence.ActionInsightIDs, insight.Id)
			}
			result[supplierID] = evidence
		}
	}
	return result, nil
}

func supplierPostureInsightMatchesSupplier(insight OperatingInsight, supplierID int) bool {
	supplierToken := fmt.Sprintf("supplier:%d", supplierID)
	if strings.Contains(insight.SliceKey, supplierToken+"|") ||
		strings.Contains(insight.SliceKey, "|"+supplierToken+"|") ||
		strings.HasSuffix(insight.SliceKey, "|"+supplierToken) {
		return true
	}
	if insight.SlaProbeRunId <= 0 {
		return false
	}
	var run SlaProbeRun
	err := DB.Select("id", "supplier_id").First(&run, "id = ?", insight.SlaProbeRunId).Error
	return err == nil && run.SupplierId == supplierID
}

func buildSupplierPostureRecommendation(scorecard SupplierScorecard, supplier Supplier, insightEvidence supplierPostureInsightEvidence, now int64) SupplierPostureRecommendation {
	action := supplierPostureRecommendedAction(scorecard, supplier.Status, insightEvidence)
	evidence := supplierPostureEvidenceJSON(scorecard, supplier, insightEvidence)
	return SupplierPostureRecommendation{
		RecommendationKey:     supplierPostureRecommendationKey(scorecard),
		SupplierId:            scorecard.SupplierId,
		SupplierScorecardId:   scorecard.Id,
		PeriodStart:           scorecard.PeriodStart,
		PeriodEnd:             scorecard.PeriodEnd,
		Status:                SupplierPostureRecommendationStatusDraft,
		RecommendedAction:     action,
		Score:                 scorecard.Score,
		Grade:                 scorecard.Grade,
		TotalRequests:         scorecard.TotalRequests,
		SuccessRate:           scorecard.SuccessRate,
		AvgLatencyMs:          scorecard.AvgLatencyMs,
		SupplyHeadroomTokens:  scorecard.SupplyHeadroomTokens,
		AvgSupplyQualityScore: scorecard.AvgSupplyQualityScore,
		QualityInsightCount:   len(insightEvidence.QualityInsightIDs),
		CapacityInsightCount:  len(insightEvidence.CapacityInsightIDs),
		ActionInsightCount:    len(insightEvidence.ActionInsightIDs),
		SupplierStatusCurrent: supplier.Status,
		Reason:                supplierPostureRecommendationReason(action, scorecard, supplier.Status, insightEvidence),
		EvidenceJSON:          evidence,
		GeneratedAt:           now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}

func supplierPostureRecommendedAction(scorecard SupplierScorecard, supplierStatus int, insightEvidence supplierPostureInsightEvidence) string {
	if scorecard.Score < 50 ||
		scorecard.Grade == SupplierScorecardGradeD ||
		len(insightEvidence.ActionInsightIDs) >= 2 {
		return SupplierPostureRecommendationActionDisable
	}
	if scorecard.Score < 70 ||
		scorecard.Grade == SupplierScorecardGradeC ||
		len(insightEvidence.ActionInsightIDs) >= 1 ||
		len(insightEvidence.QualityInsightIDs)+len(insightEvidence.CapacityInsightIDs) >= 2 {
		return SupplierPostureRecommendationActionDowngrade
	}
	if supplierStatus == common.ChannelStatusEnabled &&
		scorecard.TotalRequests > 0 &&
		scorecard.Score >= SupplierPostureRecommendationBoostMinScore &&
		scorecard.Grade == SupplierScorecardGradeA &&
		len(insightEvidence.QualityInsightIDs) == 0 &&
		len(insightEvidence.CapacityInsightIDs) == 0 &&
		len(insightEvidence.ActionInsightIDs) == 0 {
		return SupplierPostureRecommendationActionBoost
	}
	return SupplierPostureRecommendationActionObserve
}

func supplierPostureRecommendationReason(action string, scorecard SupplierScorecard, supplierStatus int, insightEvidence supplierPostureInsightEvidence) string {
	statusPrefix := ""
	if supplierStatus != common.ChannelStatusEnabled {
		statusPrefix = fmt.Sprintf("supplier status is already %d; ", supplierStatus)
	}
	switch action {
	case SupplierPostureRecommendationActionDisable:
		return fmt.Sprintf("%sscorecard grade %s score %.3f with %d action-severity supplier insights meets disable review threshold", statusPrefix, scorecard.Grade, scorecard.Score, len(insightEvidence.ActionInsightIDs))
	case SupplierPostureRecommendationActionDowngrade:
		return fmt.Sprintf("%sscorecard grade %s score %.3f with %d quality and %d capacity supplier insights meets downgrade review threshold", statusPrefix, scorecard.Grade, scorecard.Score, len(insightEvidence.QualityInsightIDs), len(insightEvidence.CapacityInsightIDs))
	case SupplierPostureRecommendationActionBoost:
		return fmt.Sprintf("%sscorecard grade %s score %.3f with no open supplier posture insights meets boost review threshold", statusPrefix, scorecard.Grade, scorecard.Score)
	default:
		return fmt.Sprintf("%sscorecard grade %s score %.3f does not meet boost, downgrade, or disable threshold", statusPrefix, scorecard.Grade, scorecard.Score)
	}
}

func supplierPostureEvidenceJSON(scorecard SupplierScorecard, supplier Supplier, insightEvidence supplierPostureInsightEvidence) string {
	encoded, err := json.Marshal(map[string]any{
		"supplier_id":            supplier.Id,
		"supplier_name":          supplier.Name,
		"supplier_type":          supplier.Type,
		"supplier_status":        supplier.Status,
		"scorecard_id":           scorecard.Id,
		"score":                  scorecard.Score,
		"grade":                  scorecard.Grade,
		"quality_insight_ids":    insightEvidence.QualityInsightIDs,
		"capacity_insight_ids":   insightEvidence.CapacityInsightIDs,
		"action_insight_ids":     insightEvidence.ActionInsightIDs,
		"quality_insight_count":  len(insightEvidence.QualityInsightIDs),
		"capacity_insight_count": len(insightEvidence.CapacityInsightIDs),
		"action_insight_count":   len(insightEvidence.ActionInsightIDs),
	})
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func UpdateSupplierPostureRecommendationReview(id int, status string, reviewedBy int, reviewNote string) (*SupplierPostureRecommendation, error) {
	status = normalizeSupplierPostureRecommendationStatus(status)
	if status == "" || status == SupplierPostureRecommendationStatusDraft || status == SupplierPostureRecommendationStatusApplied {
		return nil, errors.New("review status must be approved or rejected")
	}
	var recommendation SupplierPostureRecommendation
	if err := DB.Select("id", "applied_at", "status").First(&recommendation, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if recommendation.AppliedAt > 0 || recommendation.Status == SupplierPostureRecommendationStatusApplied {
		return nil, errors.New("applied supplier posture recommendation cannot be reviewed")
	}
	now := common.GetTimestamp()
	err := DB.Model(&SupplierPostureRecommendation{}).Where("id = ?", id).Updates(map[string]any{
		"status":      status,
		"reviewed_at": now,
		"reviewed_by": reviewedBy,
		"review_note": strings.TrimSpace(reviewNote),
		"updated_at":  now,
	}).Error
	if err != nil {
		return nil, err
	}
	return GetSupplierPostureRecommendationByID(id)
}

func ApplySupplierPostureRecommendation(id int, appliedBy int, operatorNote string) (*SupplierPostureRecommendation, error) {
	operatorNote = strings.TrimSpace(operatorNote)
	now := common.GetTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		var recommendation SupplierPostureRecommendation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&recommendation, "id = ?", id).Error; err != nil {
			return err
		}
		if recommendation.Status != SupplierPostureRecommendationStatusApproved {
			return errors.New("supplier posture recommendation must be approved before apply")
		}
		if recommendation.AppliedAt > 0 {
			return errors.New("supplier posture recommendation already applied")
		}

		var supplier Supplier
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&supplier, "id = ?", recommendation.SupplierId).Error; err != nil {
			return err
		}
		targetStatus, err := supplierPostureTargetSupplierStatus(recommendation.RecommendedAction, supplier.Status)
		if err != nil {
			return err
		}

		statusBefore := supplier.Status
		appliedNote := supplierPostureApplyNote(recommendation, appliedBy, operatorNote)
		supplierNotes := appendSupplierPostureNote(supplier.Notes, appliedNote)
		if err := tx.Model(&Supplier{}).Where("id = ?", supplier.Id).Updates(map[string]any{
			"status":       targetStatus,
			"notes":        supplierNotes,
			"updated_time": now,
		}).Error; err != nil {
			return err
		}
		if err := applySupplierRoutePreferenceForPostureTx(tx, recommendation, appliedBy, operatorNote, now); err != nil {
			return err
		}
		if err := tx.Model(&SupplierPostureRecommendation{}).Where("id = ?", recommendation.Id).Updates(map[string]any{
			"status":                 SupplierPostureRecommendationStatusApplied,
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
	return GetSupplierPostureRecommendationByID(id)
}

func supplierPostureTargetSupplierStatus(action string, currentStatus int) (int, error) {
	switch action {
	case SupplierPostureRecommendationActionObserve, SupplierPostureRecommendationActionBoost, SupplierPostureRecommendationActionDowngrade:
		return currentStatus, nil
	case SupplierPostureRecommendationActionDisable:
		return common.ChannelStatusManuallyDisabled, nil
	default:
		return 0, errors.New("invalid supplier posture recommendation action")
	}
}

func supplierPostureApplyNote(recommendation SupplierPostureRecommendation, appliedBy int, operatorNote string) string {
	note := fmt.Sprintf(
		"supplier_posture_recommendation #%d applied by user #%d: action=%s grade=%s score=%.3f",
		recommendation.Id,
		appliedBy,
		recommendation.RecommendedAction,
		recommendation.Grade,
		recommendation.Score,
	)
	if operatorNote == "" {
		return note
	}
	return fmt.Sprintf("%s; note=%s", note, operatorNote)
}

func appendSupplierPostureNote(existing string, addition string) string {
	existing = strings.TrimSpace(existing)
	if existing == "" {
		return addition
	}
	return existing + "\n" + addition
}

func GetSupplierPostureRecommendationByID(id int) (*SupplierPostureRecommendation, error) {
	var recommendation SupplierPostureRecommendation
	err := DB.First(&recommendation, "id = ?", id).Error
	return &recommendation, err
}
