package model

import (
	"errors"
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	SupplyRoutingPolicyStatusActive   = "active"
	SupplyRoutingPolicyStatusDisabled = "disabled"
)

const (
	SupplyRoutingPolicyMissReasonChannelMissing   = "channel_missing"
	SupplyRoutingPolicyMissReasonChannelDisabled  = "channel_disabled"
	SupplyRoutingPolicyMissReasonSupplierDisabled = "supplier_disabled"
	SupplyRoutingPolicyMissReasonSupplierMismatch = "supplier_mismatch"
	SupplyRoutingPolicyMissReasonCannotServeModel = "cannot_serve_model"
)

type SupplyRoutingPolicy struct {
	Id                      int    `json:"id"`
	SupplyActionExecutionId int    `json:"supply_action_execution_id" gorm:"not null;uniqueIndex:uk_supply_routing_policy_execution"`
	SupplyActionPlanId      int    `json:"supply_action_plan_id" gorm:"not null;index"`
	SupplyDecisionId        int    `json:"supply_decision_id" gorm:"not null;index"`
	DecisionKey             string `json:"decision_key" gorm:"size:512;not null;index"`
	TrafficProfileId        int    `json:"traffic_profile_id" gorm:"index;default:0"`
	SliceKey                string `json:"slice_key" gorm:"size:256;not null;index"`
	ModelName               string `json:"model_name" gorm:"size:128;not null;index"`
	SlaTier                 string `json:"sla_tier" gorm:"size:64;not null;default:'default';index"`
	UserId                  int    `json:"user_id" gorm:"index;default:0"`
	PeriodStart             int64  `json:"period_start" gorm:"bigint;not null;index"`
	PeriodEnd               int64  `json:"period_end" gorm:"bigint;not null;index"`
	Track                   string `json:"track" gorm:"size:64;not null;index"`
	ActionType              string `json:"action_type" gorm:"size:64;not null;index"`
	Status                  string `json:"status" gorm:"size:32;not null;default:'active';index"`
	SupplierId              int    `json:"supplier_id" gorm:"index;default:0"`
	ChannelId               int    `json:"channel_id" gorm:"index;default:0"`
	SupplyCapacityId        int    `json:"supply_capacity_id" gorm:"index;default:0"`
	SlaContractId           int    `json:"sla_contract_id" gorm:"default:0;index"`
	SlaProbeRunId           int    `json:"sla_probe_run_id" gorm:"default:0;index"`
	SlaProbeRunKey          string `json:"sla_probe_run_key" gorm:"size:512"`
	SlaArtifactSHA256       string `json:"sla_artifact_sha256" gorm:"size:128"`
	SlaRuntimeRef           string `json:"sla_runtime_ref" gorm:"size:256"`
	EffectiveFrom           int64  `json:"effective_from" gorm:"bigint;default:0;index"`
	EffectiveTo             int64  `json:"effective_to" gorm:"bigint;default:0;index"`
	Priority                int    `json:"priority" gorm:"default:100;index"`
	TrafficPercent          int    `json:"traffic_percent" gorm:"default:100;index"`
	Reason                  string `json:"reason" gorm:"type:text"`
	ActivatedAt             int64  `json:"activated_at" gorm:"bigint;index"`
	ActivatedBy             int    `json:"activated_by" gorm:"default:0;index"`
	DisabledAt              int64  `json:"disabled_at" gorm:"bigint;default:0;index"`
	DisabledBy              int    `json:"disabled_by" gorm:"default:0;index"`
	OperatorNote            string `json:"operator_note" gorm:"type:text"`
	CreatedAt               int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt               int64  `json:"updated_at" gorm:"bigint"`
}

type SupplyRoutingPolicyActivateInput struct {
	SupplyActionExecutionId int    `json:"supply_action_execution_id"`
	Priority                int    `json:"priority"`
	TrafficPercent          int    `json:"traffic_percent"`
	OperatorNote            string `json:"operator_note"`
}

type SupplyRoutingPolicyDisableInput struct {
	OperatorNote string `json:"operator_note"`
}

type SupplyRoutingPolicyFilters struct {
	SupplyActionExecutionId int
	SupplyActionPlanId      int
	SupplyDecisionId        int
	Status                  string
	Track                   string
	SupplierId              int
	ChannelId               int
	SupplyCapacityId        int
	StartTime               int64
	EndTime                 int64
}

type SupplyRoutingPolicyMatchInput struct {
	Group     string
	ModelName string
	SlaTier   string
	UserId    int
	RouteKey  string
	Now       int64
}

type SupplyRoutingPolicyMiss struct {
	Policy SupplyRoutingPolicy
	Group  string
	Reason string
}

func normalizeSupplyRoutingPolicyStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "", SupplyRoutingPolicyStatusActive:
		return SupplyRoutingPolicyStatusActive
	case SupplyRoutingPolicyStatusDisabled:
		return SupplyRoutingPolicyStatusDisabled
	default:
		return ""
	}
}

func normalizeRoutingSlaTier(slaTier string) string {
	slaTier = strings.TrimSpace(slaTier)
	if slaTier == "" {
		return "default"
	}
	return slaTier
}

func normalizeSupplyRoutingTrafficPercent(percent int) int {
	if percent <= 0 {
		return 100
	}
	if percent > 100 {
		return 100
	}
	return percent
}

func normalizeSupplyRoutingPolicyActivationTrafficPercent(percent int) (int, error) {
	if percent == 0 {
		return 100, nil
	}
	if percent < 0 || percent > 100 {
		return 0, errors.New("traffic_percent must be between 1 and 100")
	}
	return percent, nil
}

func SearchSupplyRoutingPolicies(filters SupplyRoutingPolicyFilters, offset int, limit int) ([]*SupplyRoutingPolicy, int64, error) {
	db := DB.Model(&SupplyRoutingPolicy{})
	if filters.SupplyActionExecutionId > 0 {
		db = db.Where("supply_action_execution_id = ?", filters.SupplyActionExecutionId)
	}
	if filters.SupplyActionPlanId > 0 {
		db = db.Where("supply_action_plan_id = ?", filters.SupplyActionPlanId)
	}
	if filters.SupplyDecisionId > 0 {
		db = db.Where("supply_decision_id = ?", filters.SupplyDecisionId)
	}
	if status := normalizeSupplyRoutingPolicyStatus(filters.Status); status != "" {
		db = db.Where("status = ?", status)
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
	var policies []*SupplyRoutingPolicy
	err := db.Offset(offset).Limit(limit).Order("status ASC, priority DESC, activated_at DESC, id DESC").Find(&policies).Error
	return policies, total, err
}

func FindActiveSupplyRoutingPolicyForRequest(input SupplyRoutingPolicyMatchInput) (*SupplyRoutingPolicy, error) {
	policy, _, err := ResolveActiveSupplyRoutingPolicyForRequest(input)
	return policy, err
}

func ResolveActiveSupplyRoutingPolicyForRequest(input SupplyRoutingPolicyMatchInput) (*SupplyRoutingPolicy, *SupplyRoutingPolicyMiss, error) {
	modelName := strings.TrimSpace(input.ModelName)
	if modelName == "" {
		return nil, nil, nil
	}
	now := input.Now
	if now <= 0 {
		now = common.GetTimestamp()
	}
	slaTier := normalizeRoutingSlaTier(input.SlaTier)

	db := DB.Model(&SupplyRoutingPolicy{}).
		Where("status = ?", SupplyRoutingPolicyStatusActive).
		Where("model_name = ?", modelName).
		Where("(sla_tier = '' OR sla_tier = ?)", slaTier).
		Where("(user_id = 0 OR user_id = ?)", input.UserId).
		Where("(effective_from = 0 OR effective_from <= ?)", now).
		Where("(effective_to = 0 OR effective_to >= ?)", now)

	var policies []*SupplyRoutingPolicy
	if err := db.Order("priority DESC, activated_at DESC, id DESC").Limit(8).Find(&policies).Error; err != nil {
		return nil, nil, err
	}
	var firstMiss *SupplyRoutingPolicyMiss
	for _, policy := range policies {
		if !supplyRoutingPolicyIncludesRouteKey(policy, input.RouteKey) {
			continue
		}
		reason := supplyRoutingPolicyMissReason(input, modelName, policy)
		if reason != "" {
			if firstMiss == nil && policy != nil {
				firstMiss = &SupplyRoutingPolicyMiss{
					Policy: *policy,
					Group:  strings.TrimSpace(input.Group),
					Reason: reason,
				}
			}
			continue
		}
		return policy, nil, nil
	}
	return nil, firstMiss, nil
}

func supplyRoutingPolicyIncludesRouteKey(policy *SupplyRoutingPolicy, routeKey string) bool {
	if policy == nil {
		return false
	}
	percent := normalizeSupplyRoutingTrafficPercent(policy.TrafficPercent)
	if percent >= 100 {
		return true
	}
	routeKey = strings.TrimSpace(routeKey)
	if routeKey == "" {
		return false
	}
	return supplyRoutingPolicyTrafficBucket(policy.Id, routeKey) <= percent
}

func supplyRoutingPolicyTrafficBucket(policyID int, routeKey string) int {
	hasher := fnv.New32a()
	_, _ = fmt.Fprintf(hasher, "%d|%s", policyID, strings.TrimSpace(routeKey))
	return int(hasher.Sum32()%100) + 1
}

func supplyRoutingPolicyMissReason(input SupplyRoutingPolicyMatchInput, modelName string, policy *SupplyRoutingPolicy) string {
	if policy == nil {
		return ""
	}
	if policy.ChannelId <= 0 {
		return SupplyRoutingPolicyMissReasonChannelMissing
	}
	channel, err := CacheGetChannel(policy.ChannelId)
	if err != nil || channel == nil {
		return SupplyRoutingPolicyMissReasonChannelMissing
	}
	if channel.Status != common.ChannelStatusEnabled {
		return SupplyRoutingPolicyMissReasonChannelDisabled
	}
	if !IsChannelSupplierEnabled(channel) {
		return SupplyRoutingPolicyMissReasonSupplierDisabled
	}
	if policy.SupplierId > 0 && channel.SupplierId != policy.SupplierId {
		return SupplyRoutingPolicyMissReasonSupplierMismatch
	}
	if strings.TrimSpace(input.Group) != "" && !IsChannelEnabledForGroupModel(input.Group, modelName, policy.ChannelId) {
		return SupplyRoutingPolicyMissReasonCannotServeModel
	}
	return ""
}

func ActivateSupplyRoutingPolicy(input SupplyRoutingPolicyActivateInput, activatedBy int) (*SupplyRoutingPolicy, error) {
	if input.SupplyActionExecutionId <= 0 {
		return nil, errors.New("supply_action_execution_id is required")
	}
	trafficPercent, err := normalizeSupplyRoutingPolicyActivationTrafficPercent(input.TrafficPercent)
	if err != nil {
		return nil, err
	}
	var execution SupplyActionExecution
	if err := DB.First(&execution, input.SupplyActionExecutionId).Error; err != nil {
		return nil, err
	}
	priority := input.Priority
	if priority == 0 {
		priority = 100
	}
	now := common.GetTimestamp()
	slaEvidence, err := validateSupplyRoutingPolicyExecution(execution, now)
	if err != nil {
		return nil, err
	}
	policy := buildSupplyRoutingPolicyFromExecution(execution, slaEvidence, priority, trafficPercent, strings.TrimSpace(input.OperatorNote), activatedBy, now)
	var existing SupplyRoutingPolicy
	err = DB.Where("supply_action_execution_id = ?", execution.Id).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := DB.Create(&policy).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		if existing.CreatedAt > 0 {
			policy.CreatedAt = existing.CreatedAt
		}
		if err := DB.Model(&existing).Updates(map[string]interface{}{
			"supply_action_plan_id": policy.SupplyActionPlanId,
			"supply_decision_id":    policy.SupplyDecisionId,
			"decision_key":          policy.DecisionKey,
			"traffic_profile_id":    policy.TrafficProfileId,
			"slice_key":             policy.SliceKey,
			"model_name":            policy.ModelName,
			"sla_tier":              policy.SlaTier,
			"user_id":               policy.UserId,
			"period_start":          policy.PeriodStart,
			"period_end":            policy.PeriodEnd,
			"track":                 policy.Track,
			"action_type":           policy.ActionType,
			"status":                policy.Status,
			"supplier_id":           policy.SupplierId,
			"channel_id":            policy.ChannelId,
			"supply_capacity_id":    policy.SupplyCapacityId,
			"sla_contract_id":       policy.SlaContractId,
			"sla_probe_run_id":      policy.SlaProbeRunId,
			"sla_probe_run_key":     policy.SlaProbeRunKey,
			"sla_artifact_sha256":   policy.SlaArtifactSHA256,
			"sla_runtime_ref":       policy.SlaRuntimeRef,
			"effective_from":        policy.EffectiveFrom,
			"effective_to":          policy.EffectiveTo,
			"priority":              policy.Priority,
			"traffic_percent":       policy.TrafficPercent,
			"reason":                policy.Reason,
			"activated_at":          policy.ActivatedAt,
			"activated_by":          policy.ActivatedBy,
			"disabled_at":           int64(0),
			"disabled_by":           0,
			"operator_note":         policy.OperatorNote,
			"created_at":            policy.CreatedAt,
			"updated_at":            policy.UpdatedAt,
		}).Error; err != nil {
			return nil, err
		}
	}
	var result SupplyRoutingPolicy
	err = DB.Where("supply_action_execution_id = ?", execution.Id).First(&result).Error
	return &result, err
}

func DisableSupplyRoutingPolicy(id int, input SupplyRoutingPolicyDisableInput, disabledBy int) (*SupplyRoutingPolicy, error) {
	if id <= 0 {
		return nil, errors.New("routing policy id is required")
	}
	var policy SupplyRoutingPolicy
	if err := DB.First(&policy, id).Error; err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	policy.Status = SupplyRoutingPolicyStatusDisabled
	policy.DisabledAt = now
	policy.DisabledBy = disabledBy
	policy.OperatorNote = strings.TrimSpace(input.OperatorNote)
	policy.UpdatedAt = now
	if err := DB.Save(&policy).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

func validateSupplyRoutingPolicyExecution(execution SupplyActionExecution, now int64) (*SlaProbeRun, error) {
	if execution.ExecutionStatus != SupplyActionExecutionStatusRecorded {
		return nil, errors.New("supply action execution must be recorded before routing policy activation")
	}
	if execution.Track != SupplierTypeSelfHosted {
		return nil, errors.New("only self-hosted executions can activate routing policies")
	}
	if execution.ChannelId <= 0 {
		return nil, errors.New("execution channel_id is required for routing policy activation")
	}
	if execution.SupplierId <= 0 {
		return nil, errors.New("execution supplier_id is required for routing policy activation")
	}
	supplier, err := GetSupplierByID(execution.SupplierId)
	if err != nil {
		return nil, err
	}
	if supplier.Type != SupplierTypeSelfHosted {
		return nil, fmt.Errorf("supplier_id=%d is not self-hosted", execution.SupplierId)
	}
	if supplier.Status != common.ChannelStatusEnabled {
		return nil, fmt.Errorf("supplier_id=%d is not enabled", execution.SupplierId)
	}
	channel, err := CacheGetChannel(execution.ChannelId)
	if err != nil {
		return nil, err
	}
	if channel.Status != common.ChannelStatusEnabled {
		return nil, fmt.Errorf("channel_id=%d is not enabled", execution.ChannelId)
	}
	if channel.SupplierId != execution.SupplierId {
		return nil, fmt.Errorf("channel_id=%d does not belong to supplier_id=%d", execution.ChannelId, execution.SupplierId)
	}
	if !IsChannelEnabledForAnyGroupModel(strings.Split(channel.Group, ","), execution.ModelName, execution.ChannelId) {
		return nil, fmt.Errorf("channel_id=%d cannot serve model %s", execution.ChannelId, execution.ModelName)
	}
	run, err := findSupplyRoutingPolicySlaEvidence(execution, now)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, errors.New("passed runtime SLA probe run is required for routing policy activation")
	}
	return run, nil
}

func findSupplyRoutingPolicySlaEvidence(execution SupplyActionExecution, now int64) (*SlaProbeRun, error) {
	if now <= 0 {
		now = common.GetTimestamp()
	}
	var run SlaProbeRun
	err := DB.Model(&SlaProbeRun{}).
		Joins("JOIN sla_probe_plans ON sla_probe_plans.id = sla_probe_runs.plan_id").
		Joins("JOIN sla_contracts ON sla_contracts.id = sla_probe_runs.contract_id").
		Where("sla_probe_runs.supplier_id = ?", execution.SupplierId).
		Where("sla_probe_runs.channel_id = ?", execution.ChannelId).
		Where("sla_probe_runs.model_name = ?", strings.TrimSpace(execution.ModelName)).
		Where("sla_probe_runs.sla_tier = ?", normalizeRoutingSlaTier(execution.SlaTier)).
		Where("sla_probe_runs.status = ?", SlaProbeRunStatusPassed).
		Where("sla_probe_runs.hard_gate_passed = ?", true).
		Where("sla_probe_plans.probe_type IN ?", []string{SlaProbeTypeRuntimeLight, SlaProbeTypeRuntimeDeep}).
		Where("sla_contracts.status = ?", SlaContractStatusActive).
		Where("(sla_contracts.effective_from = 0 OR sla_contracts.effective_from <= ?)", now).
		Where("(sla_contracts.effective_to = 0 OR sla_contracts.effective_to >= ?)", now).
		Order("sla_probe_runs.recorded_at DESC, sla_probe_runs.ended_at DESC, sla_probe_runs.id DESC").
		First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func buildSupplyRoutingPolicyFromExecution(execution SupplyActionExecution, slaEvidence *SlaProbeRun, priority int, trafficPercent int, note string, activatedBy int, now int64) SupplyRoutingPolicy {
	var slaContractId int
	var slaProbeRunId int
	var slaProbeRunKey string
	var slaArtifactSHA256 string
	var slaRuntimeRef string
	if slaEvidence != nil {
		slaContractId = slaEvidence.ContractId
		slaProbeRunId = slaEvidence.Id
		slaProbeRunKey = slaEvidence.RunKey
		slaArtifactSHA256 = slaEvidence.ArtifactSHA256
		slaRuntimeRef = slaEvidence.RuntimeRef
	}
	return SupplyRoutingPolicy{
		SupplyActionExecutionId: execution.Id,
		SupplyActionPlanId:      execution.SupplyActionPlanId,
		SupplyDecisionId:        execution.SupplyDecisionId,
		DecisionKey:             execution.DecisionKey,
		TrafficProfileId:        execution.TrafficProfileId,
		SliceKey:                execution.SliceKey,
		ModelName:               execution.ModelName,
		SlaTier:                 normalizeRoutingSlaTier(execution.SlaTier),
		UserId:                  execution.UserId,
		PeriodStart:             execution.PeriodStart,
		PeriodEnd:               execution.PeriodEnd,
		Track:                   execution.Track,
		ActionType:              execution.ActionType,
		Status:                  SupplyRoutingPolicyStatusActive,
		SupplierId:              execution.SupplierId,
		ChannelId:               execution.ChannelId,
		SupplyCapacityId:        execution.SupplyCapacityId,
		SlaContractId:           slaContractId,
		SlaProbeRunId:           slaProbeRunId,
		SlaProbeRunKey:          slaProbeRunKey,
		SlaArtifactSHA256:       slaArtifactSHA256,
		SlaRuntimeRef:           slaRuntimeRef,
		EffectiveFrom:           execution.EffectiveFrom,
		EffectiveTo:             execution.EffectiveTo,
		Priority:                priority,
		TrafficPercent:          normalizeSupplyRoutingTrafficPercent(trafficPercent),
		Reason:                  execution.OperatorNote,
		ActivatedAt:             now,
		ActivatedBy:             activatedBy,
		DisabledAt:              0,
		DisabledBy:              0,
		OperatorNote:            note,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}
