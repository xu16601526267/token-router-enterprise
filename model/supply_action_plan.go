package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm/clause"
)

const (
	SupplyActionPlanStatusPlanned    = "planned"
	SupplyActionPlanStatusInProgress = "in_progress"
	SupplyActionPlanStatusCompleted  = "completed"
	SupplyActionPlanStatusCancelled  = "cancelled"

	SupplyActionTypeRecruitThirdParty          = "recruit_third_party"
	SupplyActionTypePrepareSelfOperatedBuy     = "prepare_self_operated_purchase"
	SupplyActionTypeEvaluateSelfHostedCapacity = "evaluate_self_hosted_capacity"
	SupplyActionTypeKeepThirdPartyObservation  = "keep_third_party_observation"
)

type SupplyActionPlan struct {
	Id                           int     `json:"id"`
	SupplyDecisionId             int     `json:"supply_decision_id" gorm:"not null;uniqueIndex:uk_supply_action_plan_decision"`
	DecisionKey                  string  `json:"decision_key" gorm:"size:512;not null;index"`
	SupplyExpansionOpportunityId int     `json:"supply_expansion_opportunity_id" gorm:"default:0;index"`
	OpportunityKey               string  `json:"opportunity_key" gorm:"size:768;default:'';index"`
	OpportunityType              string  `json:"opportunity_type" gorm:"size:64;default:'';index"`
	OpportunityPriority          string  `json:"opportunity_priority" gorm:"size:32;default:'';index"`
	OpportunityClusterKey        string  `json:"opportunity_cluster_key" gorm:"size:64;default:'';index"`
	OpportunityRankScore         float64 `json:"opportunity_rank_score" gorm:"default:0;index"`
	TrafficProfileId             int     `json:"traffic_profile_id" gorm:"index;default:0"`
	SliceKey                     string  `json:"slice_key" gorm:"size:256;not null;index"`
	ModelName                    string  `json:"model_name" gorm:"size:128;not null;index"`
	SlaTier                      string  `json:"sla_tier" gorm:"size:64;not null;default:'default';index"`
	UserId                       int     `json:"user_id" gorm:"index;default:0"`
	PeriodStart                  int64   `json:"period_start" gorm:"bigint;not null;index"`
	PeriodEnd                    int64   `json:"period_end" gorm:"bigint;not null;index"`
	DecisionType                 string  `json:"decision_type" gorm:"size:64;not null;index"`
	Track                        string  `json:"track" gorm:"size:64;not null;index"`
	ActionType                   string  `json:"action_type" gorm:"size:64;not null;index"`
	Status                       string  `json:"status" gorm:"size:32;not null;default:'planned';index"`
	RecommendedCapacity          int64   `json:"recommended_capacity" gorm:"default:0"`
	GapTokens                    int64   `json:"gap_tokens" gorm:"default:0"`
	RoiScore                     float64 `json:"roi_score" gorm:"default:0"`
	Reason                       string  `json:"reason" gorm:"type:text"`
	SourceReviewedAt             int64   `json:"source_reviewed_at" gorm:"bigint;default:0;index"`
	SourceReviewedBy             int     `json:"source_reviewed_by" gorm:"default:0;index"`
	GeneratedAt                  int64   `json:"generated_at" gorm:"bigint;index"`
	StartedAt                    int64   `json:"started_at" gorm:"bigint;default:0;index"`
	CompletedAt                  int64   `json:"completed_at" gorm:"bigint;default:0;index"`
	CancelledAt                  int64   `json:"cancelled_at" gorm:"bigint;default:0;index"`
	StatusUpdatedAt              int64   `json:"status_updated_at" gorm:"bigint;default:0;index"`
	StatusUpdatedBy              int     `json:"status_updated_by" gorm:"default:0;index"`
	OperatorNote                 string  `json:"operator_note" gorm:"type:text"`
	CreatedAt                    int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt                    int64   `json:"updated_at" gorm:"bigint"`
}

type SupplyActionPlanGenerateInput struct {
	DecisionId  int    `json:"decision_id"`
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
	Track       string `json:"track"`
}

type SupplyActionPlanFilters struct {
	DecisionId int
	Status     string
	Track      string
	StartTime  int64
	EndTime    int64
}

type SupplyActionPlanStatusInput struct {
	Status       string `json:"status"`
	OperatorNote string `json:"operator_note"`
}

func validateSupplyActionPlanGenerateInput(input SupplyActionPlanGenerateInput) error {
	if input.DecisionId > 0 {
		return nil
	}
	if input.PeriodStart <= 0 {
		return errors.New("period_start is required when decision_id is not provided")
	}
	if input.PeriodEnd <= input.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	return nil
}

func normalizeSupplyActionPlanStatus(status string) string {
	switch strings.TrimSpace(status) {
	case SupplyActionPlanStatusPlanned:
		return SupplyActionPlanStatusPlanned
	case SupplyActionPlanStatusInProgress:
		return SupplyActionPlanStatusInProgress
	case SupplyActionPlanStatusCompleted:
		return SupplyActionPlanStatusCompleted
	case SupplyActionPlanStatusCancelled:
		return SupplyActionPlanStatusCancelled
	default:
		return ""
	}
}

func canTransitionSupplyActionPlanStatus(from string, to string) bool {
	from = normalizeSupplyActionPlanStatus(from)
	to = normalizeSupplyActionPlanStatus(to)
	if from == "" {
		from = SupplyActionPlanStatusPlanned
	}
	if to == "" {
		return false
	}
	if from == to {
		return true
	}
	switch from {
	case SupplyActionPlanStatusPlanned:
		return to == SupplyActionPlanStatusInProgress ||
			to == SupplyActionPlanStatusCompleted ||
			to == SupplyActionPlanStatusCancelled
	case SupplyActionPlanStatusInProgress:
		return to == SupplyActionPlanStatusCompleted ||
			to == SupplyActionPlanStatusCancelled
	default:
		return false
	}
}

func SearchSupplyActionPlans(filters SupplyActionPlanFilters, offset int, limit int) ([]*SupplyActionPlan, int64, error) {
	db := DB.Model(&SupplyActionPlan{})
	if filters.DecisionId > 0 {
		db = db.Where("supply_decision_id = ?", filters.DecisionId)
	}
	if status := normalizeSupplyActionPlanStatus(filters.Status); status != "" {
		db = db.Where("status = ?", status)
	}
	if strings.TrimSpace(filters.Track) != "" {
		db = db.Where("track = ?", strings.TrimSpace(filters.Track))
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
	var plans []*SupplyActionPlan
	err := db.Offset(offset).Limit(limit).Order("opportunity_rank_score DESC, period_start DESC, id DESC").Find(&plans).Error
	return plans, total, err
}

func GenerateSupplyActionPlans(input SupplyActionPlanGenerateInput) ([]*SupplyActionPlan, error) {
	if err := validateSupplyActionPlanGenerateInput(input); err != nil {
		return nil, err
	}

	decisionDB := DB.Model(&SupplyDecision{}).Where("status = ?", SupplyDecisionStatusApproved)
	if input.DecisionId > 0 {
		decisionDB = decisionDB.Where("id = ?", input.DecisionId)
	} else {
		decisionDB = decisionDB.Where("period_end >= ? AND period_start <= ?", input.PeriodStart, input.PeriodEnd)
	}
	if strings.TrimSpace(input.Track) != "" {
		decisionDB = decisionDB.Where("track = ?", strings.TrimSpace(input.Track))
	}

	var decisions []SupplyDecision
	if err := decisionDB.Order("period_start ASC, id ASC").Find(&decisions).Error; err != nil {
		return nil, err
	}
	if len(decisions) == 0 {
		return []*SupplyActionPlan{}, nil
	}
	opportunities, err := loadSupplyExpansionOpportunitiesForActionPlans(decisions)
	if err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	plans := make([]SupplyActionPlan, 0, len(decisions))
	for _, decision := range decisions {
		plans = append(plans, buildSupplyActionPlanFromDecision(decision, opportunities[decision.Id], now))
	}

	err = DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "supply_decision_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"decision_key",
			"supply_expansion_opportunity_id",
			"opportunity_key",
			"opportunity_type",
			"opportunity_priority",
			"opportunity_cluster_key",
			"opportunity_rank_score",
			"traffic_profile_id",
			"slice_key",
			"model_name",
			"sla_tier",
			"user_id",
			"period_start",
			"period_end",
			"decision_type",
			"track",
			"action_type",
			"recommended_capacity",
			"gap_tokens",
			"roi_score",
			"reason",
			"source_reviewed_at",
			"source_reviewed_by",
			"generated_at",
			"updated_at",
		}),
	}).Create(&plans).Error
	if err != nil {
		return nil, err
	}

	decisionIds := make([]int, 0, len(decisions))
	for _, decision := range decisions {
		decisionIds = append(decisionIds, decision.Id)
	}
	var results []*SupplyActionPlan
	err = DB.Model(&SupplyActionPlan{}).Where("supply_decision_id IN ?", decisionIds).Order("opportunity_rank_score DESC, period_start DESC, id DESC").Find(&results).Error
	return results, err
}

func UpdateSupplyActionPlanStatus(id int, input SupplyActionPlanStatusInput, updatedBy int) (*SupplyActionPlan, error) {
	status := normalizeSupplyActionPlanStatus(input.Status)
	if status == "" {
		return nil, errors.New("invalid supply action plan status")
	}

	var plan SupplyActionPlan
	if err := DB.First(&plan, id).Error; err != nil {
		return nil, err
	}
	if !canTransitionSupplyActionPlanStatus(plan.Status, status) {
		return nil, errors.New("invalid supply action plan status transition")
	}

	now := common.GetTimestamp()
	updates := map[string]interface{}{
		"status":            status,
		"status_updated_at": now,
		"status_updated_by": updatedBy,
		"operator_note":     strings.TrimSpace(input.OperatorNote),
		"updated_at":        now,
	}
	switch status {
	case SupplyActionPlanStatusInProgress:
		if plan.StartedAt == 0 {
			updates["started_at"] = now
		}
	case SupplyActionPlanStatusCompleted:
		if plan.StartedAt == 0 {
			updates["started_at"] = now
		}
		updates["completed_at"] = now
	case SupplyActionPlanStatusCancelled:
		updates["cancelled_at"] = now
	}

	if err := DB.Model(&plan).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := DB.First(&plan, id).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

func loadSupplyExpansionOpportunitiesForActionPlans(decisions []SupplyDecision) (map[int]*SupplyExpansionOpportunity, error) {
	decisionIDs := make([]int, 0, len(decisions))
	seen := map[int]struct{}{}
	for _, decision := range decisions {
		if decision.Id <= 0 {
			continue
		}
		if _, ok := seen[decision.Id]; ok {
			continue
		}
		seen[decision.Id] = struct{}{}
		decisionIDs = append(decisionIDs, decision.Id)
	}
	if len(decisionIDs) == 0 {
		return map[int]*SupplyExpansionOpportunity{}, nil
	}

	var rows []SupplyExpansionOpportunity
	if err := DB.Where("supply_decision_id IN ?", decisionIDs).Order("rank_score DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[int]*SupplyExpansionOpportunity, len(rows))
	for i := range rows {
		row := rows[i]
		if _, exists := result[row.SupplyDecisionId]; exists {
			continue
		}
		result[row.SupplyDecisionId] = &row
	}
	return result, nil
}

func buildSupplyActionPlanFromDecision(decision SupplyDecision, opportunity *SupplyExpansionOpportunity, now int64) SupplyActionPlan {
	actionType, reason := supplyActionTypeAndReason(decision)
	plan := SupplyActionPlan{
		SupplyDecisionId:    decision.Id,
		DecisionKey:         decision.DecisionKey,
		TrafficProfileId:    decision.TrafficProfileId,
		SliceKey:            decision.SliceKey,
		ModelName:           decision.ModelName,
		SlaTier:             normalizeTrafficProfileSlaTier(decision.SlaTier),
		UserId:              decision.UserId,
		PeriodStart:         decision.PeriodStart,
		PeriodEnd:           decision.PeriodEnd,
		DecisionType:        decision.DecisionType,
		Track:               decision.Track,
		ActionType:          actionType,
		Status:              SupplyActionPlanStatusPlanned,
		RecommendedCapacity: decision.RecommendedCapacity,
		GapTokens:           decision.GapTokens,
		RoiScore:            decision.RoiScore,
		Reason:              reason,
		SourceReviewedAt:    decision.ReviewedAt,
		SourceReviewedBy:    decision.ReviewedBy,
		GeneratedAt:         now,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if opportunity != nil {
		plan.SupplyExpansionOpportunityId = opportunity.Id
		plan.OpportunityKey = opportunity.OpportunityKey
		plan.OpportunityType = opportunity.OpportunityType
		plan.OpportunityPriority = opportunity.Priority
		plan.OpportunityClusterKey = opportunity.ClusterKey
		plan.OpportunityRankScore = opportunity.RankScore
	}
	return plan
}

func supplyActionTypeAndReason(decision SupplyDecision) (string, string) {
	switch decision.DecisionType {
	case SupplyDecisionTypeThirdPartyRecruit:
		return SupplyActionTypeRecruitThirdParty, "approved decision requires third-party supplier recruitment; execute onboarding outside the platform"
	case SupplyDecisionTypeSelfOperatedBuy:
		return SupplyActionTypePrepareSelfOperatedBuy, "approved decision requires self-operated capacity purchase; execute procurement offline and record capacity after purchase"
	case SupplyDecisionTypeSelfHostedEvaluate:
		return SupplyActionTypeEvaluateSelfHostedCapacity, "approved decision requires self-hosted capacity evaluation; register infrastructure only after operator approval"
	default:
		return SupplyActionTypeKeepThirdPartyObservation, "approved decision keeps this slice in observation mode without supply mutation"
	}
}
