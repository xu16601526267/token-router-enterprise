package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm/clause"
)

const (
	OperatingInsightStatusDraft        = "draft"
	OperatingInsightStatusAcknowledged = "acknowledged"
	OperatingInsightStatusDismissed    = "dismissed"

	OperatingInsightSeverityInfo   = "info"
	OperatingInsightSeverityWatch  = "watch"
	OperatingInsightSeverityAction = "action"

	OperatingInsightCategoryCacheEfficiency = "cache_efficiency"
	OperatingInsightCategoryCapacityRisk    = "capacity_risk"
	OperatingInsightCategoryPricingRisk     = "pricing_risk"
	OperatingInsightCategoryQualityWatch    = "quality_watch"
	OperatingInsightCategorySteadyState     = "steady_state"

	capacityTelemetryFreshnessSeconds = int64(3600)
	capacityTelemetryHighGpuThreshold = 0.9
	capacityTelemetryLowHeadroomRatio = 0.1
)

const (
	capacityTelemetryRiskMissingTelemetry = "missing_telemetry"
	capacityTelemetryRiskStaleTelemetry   = "stale_telemetry"
	capacityTelemetryRiskHighGpu          = "high_gpu"
	capacityTelemetryRiskLowHeadroom      = "low_headroom"
)

type OperatingInsight struct {
	Id                          int     `json:"id"`
	InsightKey                  string  `json:"insight_key" gorm:"size:512;not null;uniqueIndex:uk_operating_insight_key"`
	TrafficProfileId            int     `json:"traffic_profile_id" gorm:"index;default:0"`
	SupplyDecisionId            int     `json:"supply_decision_id" gorm:"index;default:0"`
	PricingRecommendationId     int     `json:"pricing_recommendation_id" gorm:"index;default:0"`
	SliceKey                    string  `json:"slice_key" gorm:"size:256;not null;index"`
	ModelName                   string  `json:"model_name" gorm:"size:128;not null;index"`
	SlaTier                     string  `json:"sla_tier" gorm:"size:64;not null;default:'default';index"`
	UserId                      int     `json:"user_id" gorm:"index;default:0"`
	PeriodStart                 int64   `json:"period_start" gorm:"bigint;not null;index"`
	PeriodEnd                   int64   `json:"period_end" gorm:"bigint;not null;index"`
	Status                      string  `json:"status" gorm:"size:32;not null;default:'draft';index"`
	Severity                    string  `json:"severity" gorm:"size:32;not null;default:'info';index"`
	Category                    string  `json:"category" gorm:"size:64;not null;index"`
	Title                       string  `json:"title" gorm:"size:255;not null"`
	Summary                     string  `json:"summary" gorm:"type:text"`
	RecommendedAction           string  `json:"recommended_action" gorm:"type:text"`
	DemandTokens                int64   `json:"demand_tokens" gorm:"default:0"`
	PeakTokens                  int64   `json:"peak_tokens" gorm:"default:0"`
	SupplyHeadroomTokens        int64   `json:"supply_headroom_tokens" gorm:"default:0"`
	CacheHitRate                float64 `json:"cache_hit_rate" gorm:"default:0"`
	SlaMetRate                  float64 `json:"sla_met_rate" gorm:"default:0"`
	GrossProfitQuota            int64   `json:"gross_profit_quota" gorm:"default:0"`
	AvgUnitCostQuota            float64 `json:"avg_unit_cost_quota" gorm:"default:0"`
	SupplyDecisionTrack         string  `json:"supply_decision_track" gorm:"size:64;default:'';index"`
	SupplyDecisionType          string  `json:"supply_decision_type" gorm:"size:64;default:'';index"`
	SupplyDecisionStatus        string  `json:"supply_decision_status" gorm:"size:32;default:'';index"`
	SupplyDecisionRoiScore      float64 `json:"supply_decision_roi_score" gorm:"default:0"`
	PricingRecommendationAction string  `json:"pricing_recommendation_action" gorm:"size:32;default:'';index"`
	PricingRecommendationStatus string  `json:"pricing_recommendation_status" gorm:"size:32;default:'';index"`
	RecommendedUnitPriceQuota   float64 `json:"recommended_unit_price_quota" gorm:"default:0"`
	RecommendedMarginRate       float64 `json:"recommended_margin_rate" gorm:"default:0"`
	SlaContractId               int     `json:"sla_contract_id" gorm:"default:0;index"`
	SlaProbeRunId               int     `json:"sla_probe_run_id" gorm:"default:0;index"`
	SlaProbeRunKey              string  `json:"sla_probe_run_key" gorm:"size:512;default:'';index"`
	SlaProbeStatus              string  `json:"sla_probe_status" gorm:"size:32;default:'';index"`
	SlaHardGatePassed           bool    `json:"sla_hard_gate_passed" gorm:"default:false;index"`
	SlaFailureReasons           string  `json:"sla_failure_reasons" gorm:"type:text"`
	SlaArtifactSHA256           string  `json:"sla_artifact_sha256" gorm:"size:128"`
	SlaRuntimeRef               string  `json:"sla_runtime_ref" gorm:"size:256"`
	GeneratedAt                 int64   `json:"generated_at" gorm:"bigint;index"`
	ReviewedAt                  int64   `json:"reviewed_at" gorm:"bigint;default:0"`
	ReviewedBy                  int     `json:"reviewed_by" gorm:"default:0;index"`
	ReviewNote                  string  `json:"review_note,omitempty" gorm:"type:text"`
	CreatedAt                   int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt                   int64   `json:"updated_at" gorm:"bigint"`
}

type OperatingInsightGenerateInput struct {
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
	ModelName   string `json:"model_name"`
	SlaTier     string `json:"sla_tier"`
	UserId      int    `json:"user_id"`
}

type OperatingInsightFilters struct {
	ModelName string
	SlaTier   string
	UserId    int
	Status    string
	Severity  string
	Category  string
	StartTime int64
	EndTime   int64
}

type OperatingInsightReviewInput struct {
	ReviewNote string `json:"review_note"`
}

func validateOperatingInsightGenerateInput(input OperatingInsightGenerateInput) error {
	if input.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if input.PeriodEnd <= input.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	return nil
}

func normalizeOperatingInsightStatus(value string) string {
	switch strings.TrimSpace(value) {
	case OperatingInsightStatusDraft:
		return OperatingInsightStatusDraft
	case OperatingInsightStatusAcknowledged:
		return OperatingInsightStatusAcknowledged
	case OperatingInsightStatusDismissed:
		return OperatingInsightStatusDismissed
	default:
		return ""
	}
}

func normalizeOperatingInsightSeverity(value string) string {
	switch strings.TrimSpace(value) {
	case OperatingInsightSeverityInfo:
		return OperatingInsightSeverityInfo
	case OperatingInsightSeverityWatch:
		return OperatingInsightSeverityWatch
	case OperatingInsightSeverityAction:
		return OperatingInsightSeverityAction
	default:
		return ""
	}
}

func normalizeOperatingInsightCategory(value string) string {
	switch strings.TrimSpace(value) {
	case OperatingInsightCategoryCacheEfficiency:
		return OperatingInsightCategoryCacheEfficiency
	case OperatingInsightCategoryCapacityRisk:
		return OperatingInsightCategoryCapacityRisk
	case OperatingInsightCategoryPricingRisk:
		return OperatingInsightCategoryPricingRisk
	case OperatingInsightCategoryQualityWatch:
		return OperatingInsightCategoryQualityWatch
	case OperatingInsightCategorySteadyState:
		return OperatingInsightCategorySteadyState
	default:
		return ""
	}
}

func operatingInsightKey(profile TrafficProfile) string {
	return fmt.Sprintf("operating:profile:%s|period:%d-%d", profile.SliceKey, profile.PeriodStart, profile.PeriodEnd)
}

func operatingInsightSlaProbeRunKey(run SlaProbeRun) string {
	return fmt.Sprintf("operating:sla_probe_run:%s", strings.TrimSpace(run.RunKey))
}

func operatingInsightSupplyRoutingPolicyMissKey(policy SupplyRoutingPolicy, group string, reason string, periodStart int64) string {
	return fmt.Sprintf("operating:supply_routing_policy_miss:policy:%d|group:%s|reason:%s|period:%d", policy.Id, normalizeSupplyRoutingPolicyMissGroup(group), reason, periodStart)
}

func SearchOperatingInsights(filters OperatingInsightFilters, offset int, limit int) ([]*OperatingInsight, int64, error) {
	db := DB.Model(&OperatingInsight{})
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if strings.TrimSpace(filters.SlaTier) != "" {
		db = db.Where("sla_tier = ?", normalizeTrafficProfileSlaTier(filters.SlaTier))
	}
	if filters.UserId > 0 {
		db = db.Where("user_id = ?", filters.UserId)
	}
	if status := normalizeOperatingInsightStatus(filters.Status); status != "" {
		db = db.Where("status = ?", status)
	}
	if severity := normalizeOperatingInsightSeverity(filters.Severity); severity != "" {
		db = db.Where("severity = ?", severity)
	}
	if category := normalizeOperatingInsightCategory(filters.Category); category != "" {
		db = db.Where("category = ?", category)
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
	var insights []*OperatingInsight
	err := db.Offset(offset).Limit(limit).Order("period_start DESC, severity DESC, id DESC").Find(&insights).Error
	return insights, total, err
}

func GenerateOperatingInsights(input OperatingInsightGenerateInput) ([]*OperatingInsight, error) {
	if err := validateOperatingInsightGenerateInput(input); err != nil {
		return nil, err
	}

	profileDB := DB.Model(&TrafficProfile{}).
		Where("period_start = ? AND period_end = ?", input.PeriodStart, input.PeriodEnd)
	if strings.TrimSpace(input.ModelName) != "" {
		profileDB = profileDB.Where("model_name = ?", strings.TrimSpace(input.ModelName))
	}
	if strings.TrimSpace(input.SlaTier) != "" {
		profileDB = profileDB.Where("sla_tier = ?", normalizeTrafficProfileSlaTier(input.SlaTier))
	}
	if input.UserId > 0 {
		profileDB = profileDB.Where("user_id = ?", input.UserId)
	}

	var profiles []TrafficProfile
	if err := profileDB.Order("period_start ASC, id ASC").Find(&profiles).Error; err != nil {
		return nil, err
	}
	decisions, err := loadSupplyDecisionsForProfiles(profiles)
	if err != nil {
		return nil, err
	}
	recommendations, err := loadPricingRecommendationsForProfiles(profiles)
	if err != nil {
		return nil, err
	}
	runs, err := loadSlaProbeRunsForOperatingInsights(input)
	if err != nil {
		return nil, err
	}
	capacities, err := loadSupplyCapacitiesForOperatingInsights(input)
	if err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	insights := make([]OperatingInsight, 0, len(profiles)+len(runs)+len(capacities))
	insightKeys := make([]string, 0, len(profiles)+len(runs)+len(capacities))
	for _, profile := range profiles {
		decision := decisions[supplyDecisionKey(profile)]
		recommendation := recommendations[pricingRecommendationKey(profile)]
		insight := buildOperatingInsightFromProfile(profile, decision, recommendation, now)
		insights = append(insights, insight)
		insightKeys = append(insightKeys, insight.InsightKey)
	}
	for _, run := range runs {
		insight := buildOperatingInsightFromSlaProbeRun(run, now)
		insights = append(insights, insight)
		insightKeys = append(insightKeys, insight.InsightKey)
	}
	for _, capacity := range capacities {
		reason := capacityTelemetryRiskReason(capacity, input, now)
		if reason == "" {
			continue
		}
		insight := buildOperatingInsightFromSupplyCapacityTelemetry(capacity, input, reason, now)
		insights = append(insights, insight)
		insightKeys = append(insightKeys, insight.InsightKey)
	}
	if len(insights) == 0 {
		return []*OperatingInsight{}, nil
	}

	err = DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "insight_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"traffic_profile_id",
			"supply_decision_id",
			"pricing_recommendation_id",
			"slice_key",
			"model_name",
			"sla_tier",
			"user_id",
			"period_start",
			"period_end",
			"severity",
			"category",
			"title",
			"summary",
			"recommended_action",
			"demand_tokens",
			"peak_tokens",
			"supply_headroom_tokens",
			"cache_hit_rate",
			"sla_met_rate",
			"gross_profit_quota",
			"avg_unit_cost_quota",
			"supply_decision_track",
			"supply_decision_type",
			"supply_decision_status",
			"supply_decision_roi_score",
			"pricing_recommendation_action",
			"pricing_recommendation_status",
			"recommended_unit_price_quota",
			"recommended_margin_rate",
			"sla_contract_id",
			"sla_probe_run_id",
			"sla_probe_run_key",
			"sla_probe_status",
			"sla_hard_gate_passed",
			"sla_failure_reasons",
			"sla_artifact_sha256",
			"sla_runtime_ref",
			"generated_at",
			"updated_at",
		}),
	}).Create(&insights).Error
	if err != nil {
		return nil, err
	}

	var results []*OperatingInsight
	err = DB.Model(&OperatingInsight{}).
		Where("insight_key IN ?", insightKeys).
		Order("period_start DESC, severity DESC, id DESC").
		Find(&results).Error
	return results, err
}

func loadSupplyDecisionsForProfiles(profiles []TrafficProfile) (map[string]*SupplyDecision, error) {
	if len(profiles) == 0 {
		return map[string]*SupplyDecision{}, nil
	}
	keys := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		keys = append(keys, supplyDecisionKey(profile))
	}
	var rows []SupplyDecision
	if err := DB.Where("decision_key IN ?", keys).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]*SupplyDecision, len(rows))
	for i := range rows {
		row := rows[i]
		result[row.DecisionKey] = &row
	}
	return result, nil
}

func loadPricingRecommendationsForProfiles(profiles []TrafficProfile) (map[string]*PricingRecommendation, error) {
	if len(profiles) == 0 {
		return map[string]*PricingRecommendation{}, nil
	}
	keys := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		keys = append(keys, pricingRecommendationKey(profile))
	}
	var rows []PricingRecommendation
	if err := DB.Where("recommendation_key IN ?", keys).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]*PricingRecommendation, len(rows))
	for i := range rows {
		row := rows[i]
		result[row.RecommendationKey] = &row
	}
	return result, nil
}

func loadSlaProbeRunsForOperatingInsights(input OperatingInsightGenerateInput) ([]SlaProbeRun, error) {
	if input.UserId > 0 {
		return []SlaProbeRun{}, nil
	}
	db := DB.Model(&SlaProbeRun{}).
		Where("started_at >= ? AND started_at <= ?", input.PeriodStart, input.PeriodEnd).
		Where("(status IN ? OR (status = ? AND hard_gate_passed = ?))", []string{
			SlaProbeRunStatusFailed,
			SlaProbeRunStatusInvalid,
			SlaProbeRunStatusCancelled,
		}, SlaProbeRunStatusPassed, false)
	if strings.TrimSpace(input.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(input.ModelName))
	}
	if strings.TrimSpace(input.SlaTier) != "" {
		db = db.Where("sla_tier = ?", normalizeTrafficProfileSlaTier(input.SlaTier))
	}
	var runs []SlaProbeRun
	err := db.Order("started_at ASC, id ASC").Find(&runs).Error
	return runs, err
}

func loadSupplyCapacitiesForOperatingInsights(input OperatingInsightGenerateInput) ([]SupplyCapacity, error) {
	if input.UserId > 0 {
		return []SupplyCapacity{}, nil
	}
	db := DB.Model(&SupplyCapacity{}).
		Where("status = ?", 1).
		Where("period_end >= ? AND period_start <= ?", input.PeriodStart, input.PeriodEnd)
	if strings.TrimSpace(input.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(input.ModelName))
	}
	var capacities []SupplyCapacity
	err := db.Order("period_start ASC, id ASC").Find(&capacities).Error
	return capacities, err
}

func buildOperatingInsightFromProfile(profile TrafficProfile, decision *SupplyDecision, recommendation *PricingRecommendation, now int64) OperatingInsight {
	insight := OperatingInsight{
		InsightKey:           operatingInsightKey(profile),
		TrafficProfileId:     profile.Id,
		SliceKey:             profile.SliceKey,
		ModelName:            profile.ModelName,
		SlaTier:              normalizeTrafficProfileSlaTier(profile.SlaTier),
		UserId:               profile.UserId,
		PeriodStart:          profile.PeriodStart,
		PeriodEnd:            profile.PeriodEnd,
		Status:               OperatingInsightStatusDraft,
		DemandTokens:         profile.DemandTokens,
		PeakTokens:           profile.PeakTokens,
		SupplyHeadroomTokens: profile.SupplyHeadroomTokens,
		CacheHitRate:         profile.CacheHitRate,
		SlaMetRate:           profile.SlaMetRate,
		GrossProfitQuota:     profile.GrossProfitQuota,
		AvgUnitCostQuota:     profile.AvgUnitCostQuota,
		GeneratedAt:          now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if decision != nil {
		insight.SupplyDecisionId = decision.Id
		insight.SupplyDecisionTrack = decision.Track
		insight.SupplyDecisionType = decision.DecisionType
		insight.SupplyDecisionStatus = decision.Status
		insight.SupplyDecisionRoiScore = decision.RoiScore
	}
	if recommendation != nil {
		insight.PricingRecommendationId = recommendation.Id
		insight.PricingRecommendationAction = recommendation.Action
		insight.PricingRecommendationStatus = recommendation.Status
		insight.RecommendedUnitPriceQuota = recommendation.RecommendedUnitPriceQuota
		insight.RecommendedMarginRate = recommendation.RecommendedMarginRate
	}
	applyOperatingInsightRule(&insight, profile, decision, recommendation)
	return insight
}

func applyOperatingInsightRule(insight *OperatingInsight, profile TrafficProfile, decision *SupplyDecision, recommendation *PricingRecommendation) {
	hasCapacityGap := profile.PeakTokens > 0 && profile.SupplyHeadroomTokens < profile.PeakTokens
	if decision != nil && decision.GapTokens > 0 {
		hasCapacityGap = true
	}
	isThirdPartyRecruit := decision != nil &&
		decision.Track == SupplyDecisionTrackThirdParty &&
		decision.DecisionType == SupplyDecisionTypeThirdPartyRecruit
	isRaisePrice := recommendation != nil && recommendation.Action == PricingRecommendationActionRaisePrice
	isShareSavings := recommendation != nil && recommendation.Action == PricingRecommendationActionShareSavings
	isSelfHosted := decision != nil && decision.Track == SupplyDecisionTrackSelfHosted

	switch {
	case hasCapacityGap || isThirdPartyRecruit:
		insight.Category = OperatingInsightCategoryCapacityRisk
		insight.Severity = OperatingInsightSeverityAction
		insight.Title = "Supply headroom is below peak demand"
		insight.Summary = "Peak demand exceeds available supply headroom; capacity must be addressed before stronger SLA or traffic expansion."
		insight.RecommendedAction = "Review third-party recruitment or capacity expansion before approving stronger commitments for this slice."
	case isRaisePrice:
		insight.Category = OperatingInsightCategoryPricingRisk
		insight.Severity = OperatingInsightSeverityAction
		insight.Title = "Pricing does not cover current SLA or cost risk"
		insight.Summary = "The linked pricing recommendation asks for a higher unit price or lower commitment based on margin, cost, or SLA evidence."
		insight.RecommendedAction = "Review the pricing recommendation before expanding traffic or committing this SLA tier."
	case profile.SlaMetRate < 0.95:
		insight.Category = OperatingInsightCategoryQualityWatch
		insight.Severity = OperatingInsightSeverityWatch
		insight.Title = "SLA evidence needs observation"
		insight.Summary = "Measured SLA attainment is below the operating threshold; keep the slice under observation."
		insight.RecommendedAction = "Delay commercial SLA promises until quality evidence improves."
	case profile.CacheHitRate >= 0.5 && profile.GrossProfitQuota > 0 && (isSelfHosted || isShareSavings):
		insight.Category = OperatingInsightCategoryCacheEfficiency
		insight.Severity = OperatingInsightSeverityAction
		insight.Title = "Cache efficiency creates positive-sum room"
		insight.Summary = "High cache hit rate and positive gross profit support coordinated self-hosted evaluation and price sharing decisions."
		insight.RecommendedAction = "Review self-hosted capacity and pricing savings together before scaling this slice."
	default:
		insight.Category = OperatingInsightCategorySteadyState
		insight.Severity = OperatingInsightSeverityInfo
		insight.Title = "Slice is steady under current evidence"
		insight.Summary = "Demand, supply, pricing, and SLA evidence do not require immediate operator action."
		insight.RecommendedAction = "Keep the slice under normal monitoring."
	}
}

func buildOperatingInsightFromSlaProbeRun(run SlaProbeRun, now int64) OperatingInsight {
	status := normalizeSlaProbeRunStatus(run.Status)
	if status == "" {
		status = run.Status
	}
	severity := OperatingInsightSeverityAction
	if status == SlaProbeRunStatusCancelled {
		severity = OperatingInsightSeverityWatch
	}

	failureReasons := strings.TrimSpace(run.FailureReasons)
	if failureReasons == "" && !run.HardGatePassed {
		failureReasons = "hard gate did not pass"
	}

	title := "SLA probe run needs operator review"
	summary := fmt.Sprintf("SLA probe run %s for %s/%s finished with status %s.", run.RunKey, run.ModelName, normalizeTrafficProfileSlaTier(run.SlaTier), status)
	if failureReasons != "" {
		summary = fmt.Sprintf("%s Failure evidence: %s", summary, failureReasons)
	}

	recommendedAction := "Review the probe artifact before promising or expanding this SLA tier."
	if status == SlaProbeRunStatusCancelled {
		recommendedAction = "Reschedule the probe or confirm why it was cancelled before relying on this SLA evidence."
	}

	periodEnd := run.EndedAt
	if periodEnd <= 0 {
		periodEnd = run.StartedAt
	}

	return OperatingInsight{
		InsightKey:        operatingInsightSlaProbeRunKey(run),
		SliceKey:          fmt.Sprintf("sla_probe:%s|supplier:%d|channel:%d", run.RunKey, run.SupplierId, run.ChannelId),
		ModelName:         run.ModelName,
		SlaTier:           normalizeTrafficProfileSlaTier(run.SlaTier),
		PeriodStart:       run.StartedAt,
		PeriodEnd:         periodEnd,
		Status:            OperatingInsightStatusDraft,
		Severity:          severity,
		Category:          OperatingInsightCategoryQualityWatch,
		Title:             title,
		Summary:           summary,
		RecommendedAction: recommendedAction,
		SlaMetRate:        boolRate(run.HardGatePassed),
		SlaContractId:     run.ContractId,
		SlaProbeRunId:     run.Id,
		SlaProbeRunKey:    run.RunKey,
		SlaProbeStatus:    status,
		SlaHardGatePassed: run.HardGatePassed,
		SlaFailureReasons: failureReasons,
		SlaArtifactSHA256: run.ArtifactSHA256,
		SlaRuntimeRef:     run.RuntimeRef,
		GeneratedAt:       now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func operatingInsightSupplyCapacityTelemetryKey(capacity SupplyCapacity, reason string) string {
	return fmt.Sprintf(
		"operating:capacity_telemetry:supplier:%d|node:%s|model:%s|reason:%s|period:%d-%d",
		capacity.SupplierId,
		capacity.SupplyNode,
		capacity.ModelName,
		reason,
		capacity.PeriodStart,
		capacity.PeriodEnd,
	)
}

func capacityTelemetryRiskReason(capacity SupplyCapacity, input OperatingInsightGenerateInput, now int64) string {
	if capacity.LastTelemetryId <= 0 || capacity.TelemetryObservedAt <= 0 {
		return capacityTelemetryRiskMissingTelemetry
	}
	if capacity.GpuUtilizationRate >= capacityTelemetryHighGpuThreshold {
		return capacityTelemetryRiskHighGpu
	}
	if capacity.CapacityTokens > 0 && float64(capacity.HeadroomTokens) <= float64(capacity.CapacityTokens)*capacityTelemetryLowHeadroomRatio {
		return capacityTelemetryRiskLowHeadroom
	}
	referenceTime := input.PeriodEnd
	if referenceTime <= 0 || (now > 0 && now < referenceTime) {
		referenceTime = now
	}
	if referenceTime > 0 && referenceTime-capacity.TelemetryObservedAt > capacityTelemetryFreshnessSeconds {
		return capacityTelemetryRiskStaleTelemetry
	}
	return ""
}

func buildOperatingInsightFromSupplyCapacityTelemetry(capacity SupplyCapacity, input OperatingInsightGenerateInput, reason string, now int64) OperatingInsight {
	severity := OperatingInsightSeverityWatch
	title := "Supply capacity telemetry needs review"
	summary := fmt.Sprintf("Supply capacity for supplier #%d node %s model %s needs telemetry review.", capacity.SupplierId, capacity.SupplyNode, capacity.ModelName)
	recommendedAction := "Review the supply node telemetry before relying on this capacity snapshot for stronger commitments."

	switch reason {
	case capacityTelemetryRiskHighGpu:
		severity = OperatingInsightSeverityAction
		title = "Supply node GPU utilization is high"
		summary = fmt.Sprintf("Supply capacity for supplier #%d node %s model %s reports GPU utilization %.2f with %d headroom tokens.", capacity.SupplierId, capacity.SupplyNode, capacity.ModelName, capacity.GpuUtilizationRate, capacity.HeadroomTokens)
		recommendedAction = "Review load placement and expansion options before routing more traffic to this node."
	case capacityTelemetryRiskLowHeadroom:
		severity = OperatingInsightSeverityAction
		title = "Supply node token headroom is low"
		summary = fmt.Sprintf("Supply capacity for supplier #%d node %s model %s has %d headroom tokens out of %d capacity tokens.", capacity.SupplierId, capacity.SupplyNode, capacity.ModelName, capacity.HeadroomTokens, capacity.CapacityTokens)
		recommendedAction = "Review demand placement or capacity expansion before committing more traffic to this node."
	case capacityTelemetryRiskStaleTelemetry:
		title = "Supply capacity telemetry is stale"
		summary = fmt.Sprintf("Supply capacity for supplier #%d node %s model %s last observed telemetry at %d from %s/%s.", capacity.SupplierId, capacity.SupplyNode, capacity.ModelName, capacity.TelemetryObservedAt, capacity.TelemetrySourceType, capacity.TelemetrySourceRef)
		recommendedAction = "Refresh node telemetry before using this capacity snapshot for operating decisions."
	case capacityTelemetryRiskMissingTelemetry:
		title = "Supply capacity has no telemetry evidence"
		summary = fmt.Sprintf("Supply capacity for supplier #%d node %s model %s has no linked telemetry evidence.", capacity.SupplierId, capacity.SupplyNode, capacity.ModelName)
		recommendedAction = "Record node telemetry before relying on this capacity snapshot for operating decisions."
	}

	return OperatingInsight{
		InsightKey:           operatingInsightSupplyCapacityTelemetryKey(capacity, reason),
		SliceKey:             limitOperatingInsightString(fmt.Sprintf("capacity:supplier:%d|node:%s|model:%s|reason:%s", capacity.SupplierId, capacity.SupplyNode, capacity.ModelName, reason), 256),
		ModelName:            capacity.ModelName,
		SlaTier:              normalizeTrafficProfileSlaTier(input.SlaTier),
		PeriodStart:          capacity.PeriodStart,
		PeriodEnd:            capacity.PeriodEnd,
		Status:               OperatingInsightStatusDraft,
		Severity:             severity,
		Category:             OperatingInsightCategoryCapacityRisk,
		Title:                title,
		Summary:              summary,
		RecommendedAction:    recommendedAction,
		PeakTokens:           capacity.UsedTokens,
		SupplyHeadroomTokens: capacity.HeadroomTokens,
		AvgUnitCostQuota:     capacity.UnitCostQuota,
		GeneratedAt:          now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func RecordSupplyRoutingPolicyMissInsight(miss *SupplyRoutingPolicyMiss, now int64) (*OperatingInsight, error) {
	if miss == nil || miss.Policy.Id <= 0 || strings.TrimSpace(miss.Reason) == "" {
		return nil, nil
	}
	if now <= 0 {
		now = common.GetTimestamp()
	}
	insight := buildOperatingInsightFromSupplyRoutingPolicyMiss(*miss, now)
	err := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "insight_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"traffic_profile_id",
			"supply_decision_id",
			"slice_key",
			"model_name",
			"sla_tier",
			"user_id",
			"period_start",
			"period_end",
			"severity",
			"category",
			"title",
			"summary",
			"recommended_action",
			"supply_decision_track",
			"sla_contract_id",
			"sla_probe_run_id",
			"sla_probe_run_key",
			"sla_probe_status",
			"sla_hard_gate_passed",
			"sla_artifact_sha256",
			"sla_runtime_ref",
			"generated_at",
			"updated_at",
		}),
	}).Create(&insight).Error
	if err != nil {
		return nil, err
	}
	var result OperatingInsight
	err = DB.Where("insight_key = ?", insight.InsightKey).First(&result).Error
	return &result, err
}

func buildOperatingInsightFromSupplyRoutingPolicyMiss(miss SupplyRoutingPolicyMiss, now int64) OperatingInsight {
	policy := miss.Policy
	periodStart, periodEnd := supplyRoutingPolicyMissInsightPeriod(now)
	group := normalizeSupplyRoutingPolicyMissGroup(miss.Group)
	reason := strings.TrimSpace(miss.Reason)
	reasonSummary := supplyRoutingPolicyMissReasonSummary(reason)
	sliceKey := limitOperatingInsightString(fmt.Sprintf("routing_policy_miss:policy:%d|group:%s|reason:%s", policy.Id, group, reason), 256)
	summary := fmt.Sprintf("Active supply routing policy #%d for %s/%s could not use channel #%d in group %s because %s; traffic can fall back to normal channel selection.", policy.Id, policy.ModelName, normalizeTrafficProfileSlaTier(policy.SlaTier), policy.ChannelId, group, reasonSummary)
	recommendedAction := "Review the policy channel, supplier posture, and model ability before relying on this self-hosted route."
	if policy.SlaProbeRunId > 0 {
		recommendedAction = fmt.Sprintf("%s Confirm the linked runtime SLA run #%d is still valid for the current channel posture.", recommendedAction, policy.SlaProbeRunId)
	}
	slaProbeStatus := ""
	if policy.SlaProbeRunId > 0 {
		slaProbeStatus = SlaProbeRunStatusPassed
	}

	return OperatingInsight{
		InsightKey:          operatingInsightSupplyRoutingPolicyMissKey(policy, group, reason, periodStart),
		TrafficProfileId:    policy.TrafficProfileId,
		SupplyDecisionId:    policy.SupplyDecisionId,
		SliceKey:            sliceKey,
		ModelName:           policy.ModelName,
		SlaTier:             normalizeTrafficProfileSlaTier(policy.SlaTier),
		UserId:              policy.UserId,
		PeriodStart:         periodStart,
		PeriodEnd:           periodEnd,
		Status:              OperatingInsightStatusDraft,
		Severity:            OperatingInsightSeverityWatch,
		Category:            OperatingInsightCategoryQualityWatch,
		Title:               "Supply routing policy is falling back",
		Summary:             summary,
		RecommendedAction:   recommendedAction,
		SupplyDecisionTrack: policy.Track,
		SlaMetRate:          boolRate(policy.SlaProbeRunId > 0),
		SlaContractId:       policy.SlaContractId,
		SlaProbeRunId:       policy.SlaProbeRunId,
		SlaProbeRunKey:      policy.SlaProbeRunKey,
		SlaProbeStatus:      slaProbeStatus,
		SlaHardGatePassed:   policy.SlaProbeRunId > 0,
		SlaArtifactSHA256:   policy.SlaArtifactSHA256,
		SlaRuntimeRef:       policy.SlaRuntimeRef,
		GeneratedAt:         now,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

func supplyRoutingPolicyMissInsightPeriod(now int64) (int64, int64) {
	const windowSeconds int64 = 3600
	if now <= 0 {
		now = common.GetTimestamp()
	}
	periodStart := now - now%windowSeconds
	return periodStart, periodStart + windowSeconds
}

func normalizeSupplyRoutingPolicyMissGroup(group string) string {
	group = strings.TrimSpace(group)
	if group == "" {
		return "unknown"
	}
	return group
}

func supplyRoutingPolicyMissReasonSummary(reason string) string {
	switch strings.TrimSpace(reason) {
	case SupplyRoutingPolicyMissReasonChannelMissing:
		return "the policy channel no longer exists"
	case SupplyRoutingPolicyMissReasonChannelDisabled:
		return "the policy channel is disabled"
	case SupplyRoutingPolicyMissReasonSupplierDisabled:
		return "the policy supplier is disabled"
	case SupplyRoutingPolicyMissReasonSupplierMismatch:
		return "the policy supplier does not match the channel owner"
	case SupplyRoutingPolicyMissReasonCannotServeModel:
		return "the channel cannot serve this group/model"
	default:
		return strings.TrimSpace(reason)
	}
}

func limitOperatingInsightString(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

func boolRate(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func UpdateOperatingInsightReview(id int, status string, reviewedBy int, reviewNote string) (*OperatingInsight, error) {
	status = normalizeOperatingInsightStatus(status)
	if status == "" || status == OperatingInsightStatusDraft {
		return nil, errors.New("review status must be acknowledged or dismissed")
	}
	now := common.GetTimestamp()
	err := DB.Model(&OperatingInsight{}).Where("id = ?", id).Updates(map[string]any{
		"status":      status,
		"reviewed_at": now,
		"reviewed_by": reviewedBy,
		"review_note": strings.TrimSpace(reviewNote),
		"updated_at":  now,
	}).Error
	if err != nil {
		return nil, err
	}
	return GetOperatingInsightByID(id)
}

func GetOperatingInsightByID(id int) (*OperatingInsight, error) {
	var insight OperatingInsight
	err := DB.First(&insight, "id = ?", id).Error
	return &insight, err
}
