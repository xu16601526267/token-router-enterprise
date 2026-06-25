package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SupplyActionExecutionStatusRecorded = "recorded"

	SupplyActionExecutionDrawdownSourceUsageLedger = "usage_ledger"
)

type SupplyActionExecution struct {
	Id                    int     `json:"id"`
	SupplyActionPlanId    int     `json:"supply_action_plan_id" gorm:"not null;uniqueIndex:uk_supply_action_execution_plan"`
	SupplyDecisionId      int     `json:"supply_decision_id" gorm:"not null;index"`
	DecisionKey           string  `json:"decision_key" gorm:"size:512;not null;index"`
	TrafficProfileId      int     `json:"traffic_profile_id" gorm:"index;default:0"`
	SliceKey              string  `json:"slice_key" gorm:"size:256;not null;index"`
	ModelName             string  `json:"model_name" gorm:"size:128;not null;index"`
	SlaTier               string  `json:"sla_tier" gorm:"size:64;not null;default:'default';index"`
	UserId                int     `json:"user_id" gorm:"index;default:0"`
	PeriodStart           int64   `json:"period_start" gorm:"bigint;not null;index"`
	PeriodEnd             int64   `json:"period_end" gorm:"bigint;not null;index"`
	DecisionType          string  `json:"decision_type" gorm:"size:64;not null;index"`
	Track                 string  `json:"track" gorm:"size:64;not null;index"`
	ActionType            string  `json:"action_type" gorm:"size:64;not null;index"`
	ExecutionStatus       string  `json:"execution_status" gorm:"size:32;not null;default:'recorded';index"`
	SupplierId            int     `json:"supplier_id" gorm:"index;default:0"`
	ChannelId             int     `json:"channel_id" gorm:"index;default:0"`
	SupplyCapacityId      int     `json:"supply_capacity_id" gorm:"index;default:0"`
	RecommendedCapacity   int64   `json:"recommended_capacity" gorm:"default:0"`
	ActualCapacityTokens  int64   `json:"actual_capacity_tokens" gorm:"default:0"`
	GapTokens             int64   `json:"gap_tokens" gorm:"default:0"`
	RoiScore              float64 `json:"roi_score" gorm:"default:0"`
	UnitCostQuota         float64 `json:"unit_cost_quota" gorm:"default:0"`
	DrawdownTokens        int64   `json:"drawdown_tokens" gorm:"default:0"`
	DrawdownRequestCount  int64   `json:"drawdown_request_count" gorm:"default:0"`
	RemainingTokens       int64   `json:"remaining_tokens" gorm:"default:0"`
	DrawdownRate          float64 `json:"drawdown_rate" gorm:"default:0"`
	DrawdownSourceType    string  `json:"drawdown_source_type" gorm:"size:64;default:'';index"`
	DrawdownSourceRef     string  `json:"drawdown_source_ref" gorm:"size:256;default:'';index"`
	DrawdownRefreshedAt   int64   `json:"drawdown_refreshed_at" gorm:"bigint;default:0;index"`
	EffectiveFrom         int64   `json:"effective_from" gorm:"bigint;default:0;index"`
	EffectiveTo           int64   `json:"effective_to" gorm:"bigint;default:0;index"`
	ExternalRef           string  `json:"external_ref" gorm:"size:256;default:'';index"`
	OperatorNote          string  `json:"operator_note" gorm:"type:text"`
	ActionPlanCompletedAt int64   `json:"action_plan_completed_at" gorm:"bigint;default:0;index"`
	ActionPlanCompletedBy int     `json:"action_plan_completed_by" gorm:"default:0;index"`
	RecordedAt            int64   `json:"recorded_at" gorm:"bigint;index"`
	RecordedBy            int     `json:"recorded_by" gorm:"default:0;index"`
	CreatedAt             int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt             int64   `json:"updated_at" gorm:"bigint"`
}

type SupplyActionExecutionRecordInput struct {
	SupplyActionPlanId   int     `json:"supply_action_plan_id"`
	ExecutionStatus      string  `json:"execution_status"`
	SupplierId           int     `json:"supplier_id"`
	ChannelId            int     `json:"channel_id"`
	SupplyCapacityId     int     `json:"supply_capacity_id"`
	ActualCapacityTokens int64   `json:"actual_capacity_tokens"`
	UnitCostQuota        float64 `json:"unit_cost_quota"`
	EffectiveFrom        int64   `json:"effective_from"`
	EffectiveTo          int64   `json:"effective_to"`
	ExternalRef          string  `json:"external_ref"`
	OperatorNote         string  `json:"operator_note"`
}

type SupplyActionExecutionFilters struct {
	ExecutionId        int
	SupplyActionPlanId int
	SupplyDecisionId   int
	ExecutionStatus    string
	Track              string
	SupplierId         int
	ChannelId          int
	SupplyCapacityId   int
	StartTime          int64
	EndTime            int64
}

type SupplyActionExecutionUsageRefreshInput struct {
	ExecutionId        int    `json:"execution_id"`
	SupplyActionPlanId int    `json:"supply_action_plan_id"`
	SupplyDecisionId   int    `json:"supply_decision_id"`
	ExecutionStatus    string `json:"execution_status"`
	Track              string `json:"track"`
	SupplierId         int    `json:"supplier_id"`
	ChannelId          int    `json:"channel_id"`
	SupplyCapacityId   int    `json:"supply_capacity_id"`
	StartTime          int64  `json:"start_timestamp"`
	EndTime            int64  `json:"end_timestamp"`
}

func normalizeSupplyActionExecutionStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "", SupplyActionExecutionStatusRecorded:
		return SupplyActionExecutionStatusRecorded
	default:
		return ""
	}
}

func validateSupplyActionExecutionInput(input SupplyActionExecutionRecordInput) error {
	if input.SupplyActionPlanId <= 0 {
		return errors.New("supply_action_plan_id is required")
	}
	if normalizeSupplyActionExecutionStatus(input.ExecutionStatus) == "" {
		return errors.New("invalid supply action execution status")
	}
	if input.ActualCapacityTokens < 0 {
		return errors.New("actual_capacity_tokens cannot be negative")
	}
	if input.UnitCostQuota < 0 {
		return errors.New("unit_cost_quota cannot be negative")
	}
	if input.EffectiveFrom < 0 || input.EffectiveTo < 0 {
		return errors.New("effective time cannot be negative")
	}
	if input.EffectiveTo > 0 && input.EffectiveFrom > 0 && input.EffectiveTo <= input.EffectiveFrom {
		return errors.New("effective_to must be greater than effective_from")
	}
	return nil
}

func SearchSupplyActionExecutions(filters SupplyActionExecutionFilters, offset int, limit int) ([]*SupplyActionExecution, int64, error) {
	db := DB.Model(&SupplyActionExecution{})
	if filters.ExecutionId > 0 {
		db = db.Where("id = ?", filters.ExecutionId)
	}
	if filters.SupplyActionPlanId > 0 {
		db = db.Where("supply_action_plan_id = ?", filters.SupplyActionPlanId)
	}
	if filters.SupplyDecisionId > 0 {
		db = db.Where("supply_decision_id = ?", filters.SupplyDecisionId)
	}
	if status := normalizeSupplyActionExecutionStatus(filters.ExecutionStatus); status != "" {
		db = db.Where("execution_status = ?", status)
	}
	if strings.TrimSpace(filters.Track) != "" {
		db = db.Where("track = ?", strings.TrimSpace(filters.Track))
	}
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if filters.ChannelId > 0 {
		db = db.Where("channel_id = ?", filters.ChannelId)
	}
	if filters.SupplyCapacityId > 0 {
		db = db.Where("supply_capacity_id = ?", filters.SupplyCapacityId)
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
	var executions []*SupplyActionExecution
	err := db.Offset(offset).Limit(limit).Order("recorded_at DESC, id DESC").Find(&executions).Error
	return executions, total, err
}

func RecordSupplyActionExecution(input SupplyActionExecutionRecordInput, recordedBy int) (*SupplyActionExecution, error) {
	if err := validateSupplyActionExecutionInput(input); err != nil {
		return nil, err
	}

	var plan SupplyActionPlan
	if err := DB.First(&plan, input.SupplyActionPlanId).Error; err != nil {
		return nil, err
	}
	if plan.Status != SupplyActionPlanStatusCompleted {
		return nil, errors.New("supply action plan must be completed before execution can be recorded")
	}
	if err := validateSupplyActionExecutionReferences(input); err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	execution := buildSupplyActionExecutionFromPlan(plan, input, recordedBy, now)
	if err := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "supply_action_plan_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"supply_decision_id",
			"decision_key",
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
			"execution_status",
			"supplier_id",
			"channel_id",
			"supply_capacity_id",
			"recommended_capacity",
			"actual_capacity_tokens",
			"gap_tokens",
			"roi_score",
			"unit_cost_quota",
			"drawdown_tokens",
			"drawdown_request_count",
			"remaining_tokens",
			"drawdown_rate",
			"drawdown_source_type",
			"drawdown_source_ref",
			"drawdown_refreshed_at",
			"effective_from",
			"effective_to",
			"external_ref",
			"operator_note",
			"action_plan_completed_at",
			"action_plan_completed_by",
			"recorded_at",
			"recorded_by",
			"updated_at",
		}),
	}).Create(&execution).Error; err != nil {
		return nil, err
	}

	var result SupplyActionExecution
	err := DB.Where("supply_action_plan_id = ?", input.SupplyActionPlanId).First(&result).Error
	return &result, err
}

func RefreshSupplyActionExecutionUsage(input SupplyActionExecutionUsageRefreshInput) ([]*SupplyActionExecution, error) {
	executions, err := findSupplyActionExecutionsForUsageRefresh(input)
	if err != nil {
		return nil, err
	}
	if len(executions) == 0 {
		return []*SupplyActionExecution{}, nil
	}

	now := common.GetTimestamp()
	updated := make([]*SupplyActionExecution, 0, len(executions))
	for i := range executions {
		execution := executions[i]
		drawdown, err := usageDrawdownForSupplyActionExecution(execution)
		if err != nil {
			return nil, err
		}
		execution.DrawdownTokens = drawdown.UsedTokens
		execution.DrawdownRequestCount = drawdown.RequestCount
		execution.RemainingTokens = execution.ActualCapacityTokens - drawdown.UsedTokens
		if execution.ActualCapacityTokens > 0 {
			execution.DrawdownRate = float64(drawdown.UsedTokens) / float64(execution.ActualCapacityTokens)
		} else {
			execution.DrawdownRate = 0
		}
		execution.DrawdownSourceType = SupplyActionExecutionDrawdownSourceUsageLedger
		execution.DrawdownSourceRef = buildSupplyActionExecutionDrawdownSourceRef(execution)
		execution.DrawdownRefreshedAt = now
		execution.UpdatedAt = now
		if err := DB.Model(&SupplyActionExecution{}).
			Where("id = ?", execution.Id).
			Updates(map[string]any{
				"drawdown_tokens":        execution.DrawdownTokens,
				"drawdown_request_count": execution.DrawdownRequestCount,
				"remaining_tokens":       execution.RemainingTokens,
				"drawdown_rate":          execution.DrawdownRate,
				"drawdown_source_type":   execution.DrawdownSourceType,
				"drawdown_source_ref":    execution.DrawdownSourceRef,
				"drawdown_refreshed_at":  execution.DrawdownRefreshedAt,
				"updated_at":             execution.UpdatedAt,
			}).Error; err != nil {
			return nil, err
		}
		var saved SupplyActionExecution
		if err := DB.First(&saved, execution.Id).Error; err != nil {
			return nil, err
		}
		updated = append(updated, &saved)
	}
	return updated, nil
}

func findSupplyActionExecutionsForUsageRefresh(input SupplyActionExecutionUsageRefreshInput) ([]SupplyActionExecution, error) {
	executions, _, err := SearchSupplyActionExecutions(SupplyActionExecutionFilters{
		ExecutionId:        input.ExecutionId,
		SupplyActionPlanId: input.SupplyActionPlanId,
		SupplyDecisionId:   input.SupplyDecisionId,
		ExecutionStatus:    input.ExecutionStatus,
		Track:              input.Track,
		SupplierId:         input.SupplierId,
		ChannelId:          input.ChannelId,
		SupplyCapacityId:   input.SupplyCapacityId,
		StartTime:          input.StartTime,
		EndTime:            input.EndTime,
	}, 0, 100000)
	if err != nil {
		return nil, err
	}
	result := make([]SupplyActionExecution, 0, len(executions))
	for _, execution := range executions {
		result = append(result, *execution)
	}
	return result, nil
}

type supplyActionExecutionUsageDrawdown struct {
	UsedTokens   int64
	RequestCount int64
}

func usageDrawdownForSupplyActionExecution(execution SupplyActionExecution) (supplyActionExecutionUsageDrawdown, error) {
	start, end := supplyActionExecutionUsageWindow(execution)
	if execution.SupplierId <= 0 || start <= 0 || end <= 0 || end < start {
		return supplyActionExecutionUsageDrawdown{}, nil
	}

	db := DB.Model(&UsageLedger{}).
		Where("supplier_id = ?", execution.SupplierId).
		Where("status = ?", "success").
		Where("created_at >= ? AND created_at <= ?", start, end)
	if execution.ChannelId > 0 {
		db = db.Where("channel_id = ?", execution.ChannelId)
	}
	if strings.TrimSpace(execution.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(execution.ModelName))
	}
	if strings.TrimSpace(execution.SlaTier) != "" {
		db = db.Where("(sla_tier = ? OR sla_tier = '')", strings.TrimSpace(execution.SlaTier))
	}
	if execution.UserId > 0 {
		db = db.Where("(user_id = ? OR user_id = 0)", execution.UserId)
	}
	if execution.SupplyCapacityId > 0 {
		supplyNode, err := supplyNodeForSupplyActionExecution(execution.SupplyCapacityId)
		if err != nil {
			return supplyActionExecutionUsageDrawdown{}, err
		}
		if supplyNode != "" {
			db = db.Where("supply_node = ?", supplyNode)
		}
	}

	var row supplyActionExecutionUsageDrawdown
	err := db.Select("COALESCE(SUM(prompt_tokens + completion_tokens), 0) AS used_tokens, COUNT(*) AS request_count").Scan(&row).Error
	return row, err
}

func supplyActionExecutionUsageWindow(execution SupplyActionExecution) (int64, int64) {
	start := execution.EffectiveFrom
	if start <= 0 {
		start = execution.PeriodStart
	}
	end := execution.EffectiveTo
	if end <= 0 {
		end = execution.PeriodEnd
	}
	return start, end
}

func supplyNodeForSupplyActionExecution(capacityID int) (string, error) {
	var capacity SupplyCapacity
	err := DB.Select("supply_node").First(&capacity, capacityID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(capacity.SupplyNode), nil
}

func buildSupplyActionExecutionDrawdownSourceRef(execution SupplyActionExecution) string {
	start, end := supplyActionExecutionUsageWindow(execution)
	return fmt.Sprintf("usage_ledger:execution:%d:%d:%d", execution.Id, start, end)
}

func validateSupplyActionExecutionReferences(input SupplyActionExecutionRecordInput) error {
	if input.SupplierId > 0 {
		if _, err := GetSupplierByID(input.SupplierId); err != nil {
			return err
		}
	}
	if input.ChannelId > 0 {
		var channel Channel
		if err := DB.First(&channel, input.ChannelId).Error; err != nil {
			return err
		}
	}
	if input.SupplyCapacityId > 0 {
		if _, err := GetSupplyCapacityByID(input.SupplyCapacityId); err != nil {
			return err
		}
	}
	return nil
}

func buildSupplyActionExecutionFromPlan(plan SupplyActionPlan, input SupplyActionExecutionRecordInput, recordedBy int, now int64) SupplyActionExecution {
	return SupplyActionExecution{
		SupplyActionPlanId:    plan.Id,
		SupplyDecisionId:      plan.SupplyDecisionId,
		DecisionKey:           plan.DecisionKey,
		TrafficProfileId:      plan.TrafficProfileId,
		SliceKey:              plan.SliceKey,
		ModelName:             plan.ModelName,
		SlaTier:               plan.SlaTier,
		UserId:                plan.UserId,
		PeriodStart:           plan.PeriodStart,
		PeriodEnd:             plan.PeriodEnd,
		DecisionType:          plan.DecisionType,
		Track:                 plan.Track,
		ActionType:            plan.ActionType,
		ExecutionStatus:       normalizeSupplyActionExecutionStatus(input.ExecutionStatus),
		SupplierId:            input.SupplierId,
		ChannelId:             input.ChannelId,
		SupplyCapacityId:      input.SupplyCapacityId,
		RecommendedCapacity:   plan.RecommendedCapacity,
		ActualCapacityTokens:  input.ActualCapacityTokens,
		GapTokens:             plan.GapTokens,
		RoiScore:              plan.RoiScore,
		UnitCostQuota:         input.UnitCostQuota,
		EffectiveFrom:         input.EffectiveFrom,
		EffectiveTo:           input.EffectiveTo,
		ExternalRef:           strings.TrimSpace(input.ExternalRef),
		OperatorNote:          strings.TrimSpace(input.OperatorNote),
		ActionPlanCompletedAt: plan.CompletedAt,
		ActionPlanCompletedBy: plan.StatusUpdatedBy,
		RecordedAt:            now,
		RecordedBy:            recordedBy,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}
