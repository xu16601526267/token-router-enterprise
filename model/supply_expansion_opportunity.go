package model

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm/clause"
)

const (
	SupplyExpansionOpportunityTypeThirdPartyGap   = "third_party_gap"
	SupplyExpansionOpportunityTypeThirdPartyProbe = "third_party_probe"
	SupplyExpansionOpportunityTypeSelfOperated    = "self_operated_bulk"
	SupplyExpansionOpportunityTypeSelfHosted      = "self_hosted_cache"

	SupplyExpansionOpportunityPriorityInfo   = "info"
	SupplyExpansionOpportunityPriorityWatch  = "watch"
	SupplyExpansionOpportunityPriorityAction = "action"

	SupplyExpansionOpportunityClusterCapacityGap     = "capacity_gap"
	SupplyExpansionOpportunityClusterHighCacheStable = "high_cache_stable"
	SupplyExpansionOpportunityClusterPositiveMargin  = "positive_margin"
	SupplyExpansionOpportunityClusterObserve         = "observe"
)

type SupplyExpansionOpportunity struct {
	Id                         int     `json:"id"`
	OpportunityKey             string  `json:"opportunity_key" gorm:"size:768;not null;uniqueIndex:uk_supply_expansion_opportunity_key"`
	SupplyDecisionId           int     `json:"supply_decision_id" gorm:"index;default:0"`
	TrafficProfileId           int     `json:"traffic_profile_id" gorm:"index;default:0"`
	TrafficForecastId          int     `json:"traffic_forecast_id" gorm:"index;default:0"`
	DecisionSource             string  `json:"decision_source" gorm:"size:32;not null;default:'profile';index"`
	DecisionStatus             string  `json:"decision_status" gorm:"size:32;not null;default:'draft';index"`
	SliceKey                   string  `json:"slice_key" gorm:"size:256;not null;index"`
	ModelName                  string  `json:"model_name" gorm:"size:128;not null;index"`
	SlaTier                    string  `json:"sla_tier" gorm:"size:64;not null;default:'default';index"`
	UserId                     int     `json:"user_id" gorm:"index;default:0"`
	PeriodStart                int64   `json:"period_start" gorm:"bigint;not null;index"`
	PeriodEnd                  int64   `json:"period_end" gorm:"bigint;not null;index"`
	ForecastTargetStart        int64   `json:"forecast_target_period_start" gorm:"bigint;default:0;index"`
	ForecastTargetEnd          int64   `json:"forecast_target_period_end" gorm:"bigint;default:0;index"`
	ForecastConfidence         float64 `json:"forecast_confidence" gorm:"default:0"`
	ForecastMethod             string  `json:"forecast_method" gorm:"size:64;default:'';index"`
	OpportunityType            string  `json:"opportunity_type" gorm:"size:64;not null;index"`
	Track                      string  `json:"track" gorm:"size:64;not null;index"`
	DecisionType               string  `json:"decision_type" gorm:"size:64;not null;index"`
	Priority                   string  `json:"priority" gorm:"size:32;not null;index"`
	ClusterKey                 string  `json:"cluster_key" gorm:"size:64;not null;index"`
	DemandTokens               int64   `json:"demand_tokens" gorm:"default:0"`
	PeakTokens                 int64   `json:"peak_tokens" gorm:"default:0"`
	SupplyHeadroomTokens       int64   `json:"supply_headroom_tokens" gorm:"default:0"`
	GapTokens                  int64   `json:"gap_tokens" gorm:"default:0"`
	RecommendedCapacity        int64   `json:"recommended_capacity" gorm:"default:0"`
	CacheHitRate               float64 `json:"cache_hit_rate" gorm:"default:0"`
	SlaMetRate                 float64 `json:"sla_met_rate" gorm:"default:0"`
	GrossProfitQuota           int64   `json:"gross_profit_quota" gorm:"default:0"`
	AvgSupplyQualityScore      float64 `json:"avg_supply_quality_score" gorm:"default:0"`
	AvgUnitCostQuota           float64 `json:"avg_unit_cost_quota" gorm:"default:0"`
	RoiScore                   float64 `json:"roi_score" gorm:"default:0"`
	SelfHostedCostProfileId    int     `json:"self_hosted_cost_profile_id" gorm:"default:0;index"`
	SelfHostedUnitCostQuota    float64 `json:"self_hosted_unit_cost_quota" gorm:"default:0"`
	SelfHostedSavingsUnitQuota float64 `json:"self_hosted_savings_unit_quota" gorm:"default:0"`
	SelfHostedSavingsQuota     float64 `json:"self_hosted_savings_quota" gorm:"default:0"`
	PeakRatio                  float64 `json:"peak_ratio" gorm:"default:0"`
	UniqueSessions             int64   `json:"unique_sessions" gorm:"default:0"`
	LocalityScore              float64 `json:"locality_score" gorm:"default:0"`
	StabilityScore             float64 `json:"stability_score" gorm:"default:0"`
	HeadroomRiskScore          float64 `json:"headroom_risk_score" gorm:"default:0"`
	RankScore                  float64 `json:"rank_score" gorm:"default:0;index"`
	Reason                     string  `json:"reason" gorm:"type:text"`
	GeneratedAt                int64   `json:"generated_at" gorm:"bigint;index"`
	CreatedAt                  int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt                  int64   `json:"updated_at" gorm:"bigint"`
}

type SupplyExpansionOpportunityGenerateInput struct {
	PeriodStart    int64  `json:"period_start"`
	PeriodEnd      int64  `json:"period_end"`
	ModelName      string `json:"model_name"`
	SlaTier        string `json:"sla_tier"`
	UserId         int    `json:"user_id"`
	DecisionStatus string `json:"decision_status"`
	Track          string `json:"track"`
}

type SupplyExpansionOpportunityFilters struct {
	ModelName       string
	SlaTier         string
	UserId          int
	DecisionStatus  string
	Track           string
	OpportunityType string
	Priority        string
	ClusterKey      string
	StartTime       int64
	EndTime         int64
}

func validateSupplyExpansionOpportunityGenerateInput(input SupplyExpansionOpportunityGenerateInput) error {
	if input.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if input.PeriodEnd <= input.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	return nil
}

func supplyExpansionOpportunityKey(decision SupplyDecision) string {
	return fmt.Sprintf("supply_expansion:decision:%s", decision.DecisionKey)
}

func normalizeSupplyExpansionOpportunityType(value string) string {
	switch strings.TrimSpace(value) {
	case SupplyExpansionOpportunityTypeThirdPartyGap:
		return SupplyExpansionOpportunityTypeThirdPartyGap
	case SupplyExpansionOpportunityTypeThirdPartyProbe:
		return SupplyExpansionOpportunityTypeThirdPartyProbe
	case SupplyExpansionOpportunityTypeSelfOperated:
		return SupplyExpansionOpportunityTypeSelfOperated
	case SupplyExpansionOpportunityTypeSelfHosted:
		return SupplyExpansionOpportunityTypeSelfHosted
	default:
		return ""
	}
}

func normalizeSupplyExpansionOpportunityPriority(value string) string {
	switch strings.TrimSpace(value) {
	case SupplyExpansionOpportunityPriorityInfo:
		return SupplyExpansionOpportunityPriorityInfo
	case SupplyExpansionOpportunityPriorityWatch:
		return SupplyExpansionOpportunityPriorityWatch
	case SupplyExpansionOpportunityPriorityAction:
		return SupplyExpansionOpportunityPriorityAction
	default:
		return ""
	}
}

func normalizeSupplyExpansionOpportunityCluster(value string) string {
	switch strings.TrimSpace(value) {
	case SupplyExpansionOpportunityClusterCapacityGap:
		return SupplyExpansionOpportunityClusterCapacityGap
	case SupplyExpansionOpportunityClusterHighCacheStable:
		return SupplyExpansionOpportunityClusterHighCacheStable
	case SupplyExpansionOpportunityClusterPositiveMargin:
		return SupplyExpansionOpportunityClusterPositiveMargin
	case SupplyExpansionOpportunityClusterObserve:
		return SupplyExpansionOpportunityClusterObserve
	default:
		return ""
	}
}

func SearchSupplyExpansionOpportunities(filters SupplyExpansionOpportunityFilters, offset int, limit int) ([]*SupplyExpansionOpportunity, int64, error) {
	db := DB.Model(&SupplyExpansionOpportunity{})
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if strings.TrimSpace(filters.SlaTier) != "" {
		db = db.Where("sla_tier = ?", normalizeTrafficProfileSlaTier(filters.SlaTier))
	}
	if filters.UserId > 0 {
		db = db.Where("user_id = ?", filters.UserId)
	}
	if strings.TrimSpace(filters.DecisionStatus) != "" {
		db = db.Where("decision_status = ?", normalizeSupplyDecisionStatus(filters.DecisionStatus))
	}
	if strings.TrimSpace(filters.Track) != "" {
		db = db.Where("track = ?", strings.TrimSpace(filters.Track))
	}
	if opportunityType := normalizeSupplyExpansionOpportunityType(filters.OpportunityType); opportunityType != "" {
		db = db.Where("opportunity_type = ?", opportunityType)
	}
	if priority := normalizeSupplyExpansionOpportunityPriority(filters.Priority); priority != "" {
		db = db.Where("priority = ?", priority)
	}
	if cluster := normalizeSupplyExpansionOpportunityCluster(filters.ClusterKey); cluster != "" {
		db = db.Where("cluster_key = ?", cluster)
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
	var opportunities []*SupplyExpansionOpportunity
	err := db.Offset(offset).Limit(limit).Order("rank_score DESC, period_start DESC, id DESC").Find(&opportunities).Error
	return opportunities, total, err
}

func GenerateSupplyExpansionOpportunities(input SupplyExpansionOpportunityGenerateInput) ([]*SupplyExpansionOpportunity, error) {
	if err := validateSupplyExpansionOpportunityGenerateInput(input); err != nil {
		return nil, err
	}

	decisionDB := DB.Model(&SupplyDecision{}).
		Where("period_start = ? AND period_end = ?", input.PeriodStart, input.PeriodEnd)
	if strings.TrimSpace(input.ModelName) != "" {
		decisionDB = decisionDB.Where("model_name = ?", strings.TrimSpace(input.ModelName))
	}
	if strings.TrimSpace(input.SlaTier) != "" {
		decisionDB = decisionDB.Where("sla_tier = ?", normalizeTrafficProfileSlaTier(input.SlaTier))
	}
	if input.UserId > 0 {
		decisionDB = decisionDB.Where("user_id = ?", input.UserId)
	}
	if strings.TrimSpace(input.DecisionStatus) != "" {
		decisionDB = decisionDB.Where("status = ?", normalizeSupplyDecisionStatus(input.DecisionStatus))
	}
	if strings.TrimSpace(input.Track) != "" {
		decisionDB = decisionDB.Where("track = ?", strings.TrimSpace(input.Track))
	}

	var decisions []SupplyDecision
	if err := decisionDB.Order("period_start ASC, id ASC").Find(&decisions).Error; err != nil {
		return nil, err
	}
	if len(decisions) == 0 {
		return []*SupplyExpansionOpportunity{}, nil
	}

	profiles, err := loadTrafficProfilesForSupplyExpansionOpportunities(decisions)
	if err != nil {
		return nil, err
	}
	costProfiles, err := loadSupplyCostProfilesForSupplyExpansionOpportunities(decisions)
	if err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	opportunities := make([]SupplyExpansionOpportunity, 0, len(decisions))
	for _, decision := range decisions {
		opportunities = append(opportunities, buildSupplyExpansionOpportunity(
			decision,
			profiles[decision.TrafficProfileId],
			costProfiles[decision.Id],
			now,
		))
	}

	err = DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "opportunity_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"supply_decision_id",
			"traffic_profile_id",
			"traffic_forecast_id",
			"decision_source",
			"decision_status",
			"slice_key",
			"model_name",
			"sla_tier",
			"user_id",
			"period_start",
			"period_end",
			"forecast_target_start",
			"forecast_target_end",
			"forecast_confidence",
			"forecast_method",
			"opportunity_type",
			"track",
			"decision_type",
			"priority",
			"cluster_key",
			"demand_tokens",
			"peak_tokens",
			"supply_headroom_tokens",
			"gap_tokens",
			"recommended_capacity",
			"cache_hit_rate",
			"sla_met_rate",
			"gross_profit_quota",
			"avg_supply_quality_score",
			"avg_unit_cost_quota",
			"roi_score",
			"self_hosted_cost_profile_id",
			"self_hosted_unit_cost_quota",
			"self_hosted_savings_unit_quota",
			"self_hosted_savings_quota",
			"peak_ratio",
			"unique_sessions",
			"locality_score",
			"stability_score",
			"headroom_risk_score",
			"rank_score",
			"reason",
			"generated_at",
			"updated_at",
		}),
	}).Create(&opportunities).Error
	if err != nil {
		return nil, err
	}

	resultDB := DB.Model(&SupplyExpansionOpportunity{}).
		Where("period_start = ? AND period_end = ?", input.PeriodStart, input.PeriodEnd)
	if strings.TrimSpace(input.ModelName) != "" {
		resultDB = resultDB.Where("model_name = ?", strings.TrimSpace(input.ModelName))
	}
	if strings.TrimSpace(input.SlaTier) != "" {
		resultDB = resultDB.Where("sla_tier = ?", normalizeTrafficProfileSlaTier(input.SlaTier))
	}
	if input.UserId > 0 {
		resultDB = resultDB.Where("user_id = ?", input.UserId)
	}
	if strings.TrimSpace(input.DecisionStatus) != "" {
		resultDB = resultDB.Where("decision_status = ?", normalizeSupplyDecisionStatus(input.DecisionStatus))
	}
	if strings.TrimSpace(input.Track) != "" {
		resultDB = resultDB.Where("track = ?", strings.TrimSpace(input.Track))
	}
	var results []*SupplyExpansionOpportunity
	err = resultDB.Order("rank_score DESC, id DESC").Find(&results).Error
	return results, err
}

func loadTrafficProfilesForSupplyExpansionOpportunities(decisions []SupplyDecision) (map[int]*TrafficProfile, error) {
	profileIDs := make([]int, 0, len(decisions))
	seen := map[int]struct{}{}
	for _, decision := range decisions {
		if decision.TrafficProfileId <= 0 {
			continue
		}
		if _, ok := seen[decision.TrafficProfileId]; ok {
			continue
		}
		seen[decision.TrafficProfileId] = struct{}{}
		profileIDs = append(profileIDs, decision.TrafficProfileId)
	}
	if len(profileIDs) == 0 {
		return map[int]*TrafficProfile{}, nil
	}
	var rows []TrafficProfile
	if err := DB.Where("id IN ?", profileIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[int]*TrafficProfile, len(rows))
	for i := range rows {
		row := rows[i]
		result[row.Id] = &row
	}
	return result, nil
}

func loadSupplyCostProfilesForSupplyExpansionOpportunities(decisions []SupplyDecision) (map[int]*SupplyCostProfile, error) {
	modelNames := make([]string, 0, len(decisions))
	seenModels := map[string]struct{}{}
	minStart := int64(0)
	maxEnd := int64(0)
	for _, decision := range decisions {
		if decision.Track != SupplyDecisionTrackSelfHosted {
			continue
		}
		modelName := strings.TrimSpace(decision.ModelName)
		if modelName == "" {
			continue
		}
		if _, ok := seenModels[modelName]; !ok {
			seenModels[modelName] = struct{}{}
			modelNames = append(modelNames, modelName)
		}
		if minStart == 0 || decision.PeriodStart < minStart {
			minStart = decision.PeriodStart
		}
		if decision.PeriodEnd > maxEnd {
			maxEnd = decision.PeriodEnd
		}
	}
	if len(modelNames) == 0 || minStart == 0 || maxEnd == 0 {
		return map[int]*SupplyCostProfile{}, nil
	}

	var rows []SupplyCostProfile
	if err := DB.Where("model_name IN ?", modelNames).
		Where("period_end >= ? AND period_start <= ?", minStart, maxEnd).
		Order("observed_at DESC, id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[int]*SupplyCostProfile, len(decisions))
	for _, decision := range decisions {
		if decision.Track != SupplyDecisionTrackSelfHosted {
			continue
		}
		for i := range rows {
			if rows[i].ModelName != decision.ModelName {
				continue
			}
			if rows[i].PeriodEnd < decision.PeriodStart || rows[i].PeriodStart > decision.PeriodEnd {
				continue
			}
			result[decision.Id] = &rows[i]
			break
		}
	}
	return result, nil
}

func buildSupplyExpansionOpportunity(decision SupplyDecision, profile *TrafficProfile, costProfile *SupplyCostProfile, now int64) SupplyExpansionOpportunity {
	peakRatio := float64(0)
	uniqueSessions := int64(0)
	if profile != nil {
		peakRatio = profile.PeakRatio
		uniqueSessions = profile.UniqueSessions
	}
	localityScore := clampFloat(decision.CacheHitRate, 0, 1)
	stabilityScore := supplyExpansionStabilityScore(peakRatio)
	headroomRiskScore := supplyExpansionHeadroomRiskScore(decision.GapTokens, decision.PeakTokens)
	opportunityType, priority, clusterKey, reason := classifySupplyExpansionOpportunity(decision, localityScore, stabilityScore)
	rankScore := supplyExpansionRankScore(decision.RoiScore, localityScore, stabilityScore, headroomRiskScore)
	costProfileId, selfHostedUnitCostQuota, selfHostedSavingsUnitQuota, selfHostedSavingsQuota := supplyExpansionSelfHostedCostEvidence(decision, costProfile)
	if costProfileId > 0 {
		rankScore += selfHostedSavingsQuota
		reason = fmt.Sprintf(
			"%s; cost profile %s amortized unit cost %.4f vs baseline %.4f gives %.2f quota savings",
			reason,
			costProfile.SourceRef,
			selfHostedUnitCostQuota,
			decision.AvgUnitCostQuota,
			selfHostedSavingsQuota,
		)
	}

	if math.IsNaN(rankScore) || math.IsInf(rankScore, 0) {
		rankScore = 0
	}

	return SupplyExpansionOpportunity{
		OpportunityKey:             supplyExpansionOpportunityKey(decision),
		SupplyDecisionId:           decision.Id,
		TrafficProfileId:           decision.TrafficProfileId,
		TrafficForecastId:          decision.TrafficForecastId,
		DecisionSource:             decision.DecisionSource,
		DecisionStatus:             decision.Status,
		SliceKey:                   decision.SliceKey,
		ModelName:                  decision.ModelName,
		SlaTier:                    normalizeTrafficProfileSlaTier(decision.SlaTier),
		UserId:                     decision.UserId,
		PeriodStart:                decision.PeriodStart,
		PeriodEnd:                  decision.PeriodEnd,
		ForecastTargetStart:        decision.ForecastTargetStart,
		ForecastTargetEnd:          decision.ForecastTargetEnd,
		ForecastConfidence:         decision.ForecastConfidence,
		ForecastMethod:             decision.ForecastMethod,
		OpportunityType:            opportunityType,
		Track:                      decision.Track,
		DecisionType:               decision.DecisionType,
		Priority:                   priority,
		ClusterKey:                 clusterKey,
		DemandTokens:               decision.DemandTokens,
		PeakTokens:                 decision.PeakTokens,
		SupplyHeadroomTokens:       decision.SupplyHeadroomTokens,
		GapTokens:                  decision.GapTokens,
		RecommendedCapacity:        decision.RecommendedCapacity,
		CacheHitRate:               decision.CacheHitRate,
		SlaMetRate:                 decision.SlaMetRate,
		GrossProfitQuota:           decision.GrossProfitQuota,
		AvgSupplyQualityScore:      decision.AvgSupplyQualityScore,
		AvgUnitCostQuota:           decision.AvgUnitCostQuota,
		RoiScore:                   decision.RoiScore,
		SelfHostedCostProfileId:    costProfileId,
		SelfHostedUnitCostQuota:    selfHostedUnitCostQuota,
		SelfHostedSavingsUnitQuota: selfHostedSavingsUnitQuota,
		SelfHostedSavingsQuota:     selfHostedSavingsQuota,
		PeakRatio:                  peakRatio,
		UniqueSessions:             uniqueSessions,
		LocalityScore:              localityScore,
		StabilityScore:             stabilityScore,
		HeadroomRiskScore:          headroomRiskScore,
		RankScore:                  rankScore,
		Reason:                     reason,
		GeneratedAt:                now,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
}

func classifySupplyExpansionOpportunity(decision SupplyDecision, localityScore float64, stabilityScore float64) (string, string, string, string) {
	if decision.GapTokens > 0 || decision.DecisionType == SupplyDecisionTypeThirdPartyRecruit {
		return SupplyExpansionOpportunityTypeThirdPartyGap,
			SupplyExpansionOpportunityPriorityAction,
			SupplyExpansionOpportunityClusterCapacityGap,
			"forecast/profile peak demand exceeds supply headroom; recruit or expand third-party supply before stronger commitment"
	}
	if decision.Track == SupplyDecisionTrackSelfHosted {
		return SupplyExpansionOpportunityTypeSelfHosted,
			SupplyExpansionOpportunityPriorityAction,
			SupplyExpansionOpportunityClusterHighCacheStable,
			fmt.Sprintf("cache locality score %.2f and stability score %.2f make this slice a self-hosted expansion candidate", localityScore, stabilityScore)
	}
	if decision.Track == SupplyDecisionTrackSelfOperated {
		return SupplyExpansionOpportunityTypeSelfOperated,
			SupplyExpansionOpportunityPriorityWatch,
			SupplyExpansionOpportunityClusterPositiveMargin,
			"positive gross profit and demand profile make this slice a self-operated bulk purchase candidate"
	}
	return SupplyExpansionOpportunityTypeThirdPartyProbe,
		SupplyExpansionOpportunityPriorityInfo,
		SupplyExpansionOpportunityClusterObserve,
		"keep third-party supply under observation until demand, cache locality, or margin justify expansion"
}

func supplyExpansionStabilityScore(peakRatio float64) float64 {
	if peakRatio <= 0 {
		return 0
	}
	if peakRatio < 1 {
		return 1
	}
	return clampFloat(1/peakRatio, 0, 1)
}

func supplyExpansionHeadroomRiskScore(gapTokens int64, peakTokens int64) float64 {
	if gapTokens <= 0 || peakTokens <= 0 {
		return 0
	}
	return clampFloat(float64(gapTokens)/float64(peakTokens), 0, 1)
}

func supplyExpansionRankScore(roiScore float64, localityScore float64, stabilityScore float64, headroomRiskScore float64) float64 {
	return roiScore + localityScore*100 + stabilityScore*50 + headroomRiskScore*150
}

func supplyExpansionSelfHostedCostEvidence(decision SupplyDecision, costProfile *SupplyCostProfile) (int, float64, float64, float64) {
	if decision.Track != SupplyDecisionTrackSelfHosted || costProfile == nil || costProfile.Id <= 0 {
		return 0, 0, 0, 0
	}
	unitCost := costProfile.AmortizedUnitCostQuota
	savingsUnit := decision.AvgUnitCostQuota - unitCost
	savingsQuota := savingsUnit * float64(decision.DemandTokens)
	if math.IsNaN(savingsQuota) || math.IsInf(savingsQuota, 0) {
		savingsQuota = 0
	}
	return costProfile.Id, unitCost, savingsUnit, savingsQuota
}
