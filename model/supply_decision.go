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
	SupplyDecisionStatusDraft    = "draft"
	SupplyDecisionStatusApproved = "approved"
	SupplyDecisionStatusRejected = "rejected"

	SupplyDecisionTrackThirdParty   = "third_party"
	SupplyDecisionTrackSelfOperated = "self_operated"
	SupplyDecisionTrackSelfHosted   = "self_hosted"

	SupplyDecisionTypeThirdPartyRecruit  = "third_party_recruit"
	SupplyDecisionTypeSelfOperatedBuy    = "self_operated_purchase"
	SupplyDecisionTypeSelfHostedEvaluate = "self_hosted_evaluate"
	SupplyDecisionTypeThirdPartyProbe    = "third_party_probe"

	SupplyDecisionSourceProfile  = "profile"
	SupplyDecisionSourceForecast = "forecast"
)

type SupplyDecision struct {
	Id                    int     `json:"id"`
	DecisionKey           string  `json:"decision_key" gorm:"size:512;not null;uniqueIndex:uk_supply_decision_key"`
	TrafficProfileId      int     `json:"traffic_profile_id" gorm:"index;default:0"`
	TrafficForecastId     int     `json:"traffic_forecast_id" gorm:"index;default:0"`
	DecisionSource        string  `json:"decision_source" gorm:"size:32;not null;default:'profile';index"`
	SliceKey              string  `json:"slice_key" gorm:"size:256;not null;index"`
	ModelName             string  `json:"model_name" gorm:"size:128;not null;index"`
	SlaTier               string  `json:"sla_tier" gorm:"size:64;not null;default:'default';index"`
	UserId                int     `json:"user_id" gorm:"index;default:0"`
	PeriodStart           int64   `json:"period_start" gorm:"bigint;not null;index"`
	PeriodEnd             int64   `json:"period_end" gorm:"bigint;not null;index"`
	ForecastTargetStart   int64   `json:"forecast_target_period_start" gorm:"bigint;default:0;index"`
	ForecastTargetEnd     int64   `json:"forecast_target_period_end" gorm:"bigint;default:0;index"`
	ForecastConfidence    float64 `json:"forecast_confidence" gorm:"default:0"`
	ForecastMethod        string  `json:"forecast_method" gorm:"size:64;default:'';index"`
	DecisionType          string  `json:"decision_type" gorm:"size:64;not null;index"`
	Track                 string  `json:"track" gorm:"size:64;not null;index"`
	Status                string  `json:"status" gorm:"size:32;not null;default:'draft';index"`
	DemandTokens          int64   `json:"demand_tokens" gorm:"default:0"`
	PeakTokens            int64   `json:"peak_tokens" gorm:"default:0"`
	SupplyHeadroomTokens  int64   `json:"supply_headroom_tokens" gorm:"default:0"`
	GapTokens             int64   `json:"gap_tokens" gorm:"default:0"`
	RecommendedCapacity   int64   `json:"recommended_capacity" gorm:"default:0"`
	CacheHitRate          float64 `json:"cache_hit_rate" gorm:"default:0"`
	SlaMetRate            float64 `json:"sla_met_rate" gorm:"default:0"`
	GrossProfitQuota      int64   `json:"gross_profit_quota" gorm:"default:0"`
	AvgSupplyQualityScore float64 `json:"avg_supply_quality_score" gorm:"default:0"`
	AvgUnitCostQuota      float64 `json:"avg_unit_cost_quota" gorm:"default:0"`
	RoiScore              float64 `json:"roi_score" gorm:"default:0"`
	Reason                string  `json:"reason" gorm:"type:text"`
	GeneratedAt           int64   `json:"generated_at" gorm:"bigint;index"`
	ReviewedAt            int64   `json:"reviewed_at" gorm:"bigint;default:0"`
	ReviewedBy            int     `json:"reviewed_by" gorm:"default:0;index"`
	ReviewNote            string  `json:"review_note,omitempty" gorm:"type:text"`
	CreatedAt             int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt             int64   `json:"updated_at" gorm:"bigint"`
}

type SupplyDecisionGenerateInput struct {
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
	ModelName   string `json:"model_name"`
	SlaTier     string `json:"sla_tier"`
	UserId      int    `json:"user_id"`
}

type SupplyDecisionFilters struct {
	ModelName    string
	SlaTier      string
	UserId       int
	Status       string
	Track        string
	DecisionType string
	StartTime    int64
	EndTime      int64
}

type SupplyDecisionReviewInput struct {
	ReviewNote string `json:"review_note"`
}

func validateSupplyDecisionGenerateInput(input SupplyDecisionGenerateInput) error {
	if input.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if input.PeriodEnd <= input.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	return nil
}

func normalizeSupplyDecisionStatus(status string) string {
	switch strings.TrimSpace(status) {
	case SupplyDecisionStatusApproved:
		return SupplyDecisionStatusApproved
	case SupplyDecisionStatusRejected:
		return SupplyDecisionStatusRejected
	default:
		return SupplyDecisionStatusDraft
	}
}

func supplyDecisionKey(profile TrafficProfile) string {
	return fmt.Sprintf("profile:%s|period:%d-%d", supplyDecisionSliceKey(profile), profile.PeriodStart, profile.PeriodEnd)
}

func supplyDecisionSliceKey(profile TrafficProfile) string {
	sliceKey := strings.TrimSpace(profile.SliceKey)
	if sliceKey == "" {
		return trafficProfileSliceKey(profile.ModelName, profile.SlaTier, profile.UserId)
	}
	return sliceKey
}

func SearchSupplyDecisions(filters SupplyDecisionFilters, offset int, limit int) ([]*SupplyDecision, int64, error) {
	db := DB.Model(&SupplyDecision{})
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if strings.TrimSpace(filters.SlaTier) != "" {
		db = db.Where("sla_tier = ?", normalizeTrafficProfileSlaTier(filters.SlaTier))
	}
	if filters.UserId > 0 {
		db = db.Where("user_id = ?", filters.UserId)
	}
	if strings.TrimSpace(filters.Status) != "" {
		db = db.Where("status = ?", normalizeSupplyDecisionStatus(filters.Status))
	}
	if strings.TrimSpace(filters.Track) != "" {
		db = db.Where("track = ?", strings.TrimSpace(filters.Track))
	}
	if strings.TrimSpace(filters.DecisionType) != "" {
		db = db.Where("decision_type = ?", strings.TrimSpace(filters.DecisionType))
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
	var decisions []*SupplyDecision
	err := db.Offset(offset).Limit(limit).Order("period_start DESC, id DESC").Find(&decisions).Error
	return decisions, total, err
}

func GenerateSupplyDecisions(input SupplyDecisionGenerateInput) ([]*SupplyDecision, error) {
	if err := validateSupplyDecisionGenerateInput(input); err != nil {
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
	if len(profiles) == 0 {
		return []*SupplyDecision{}, nil
	}

	forecasts, err := loadTrafficForecastsForSupplyDecisions(profiles, input)
	if err != nil {
		return nil, err
	}

	decisions := make([]SupplyDecision, 0, len(profiles))
	now := common.GetTimestamp()
	for _, profile := range profiles {
		decision := buildSupplyDecisionFromProfile(profile, forecasts[supplyDecisionSliceKey(profile)], now)
		decisions = append(decisions, decision)
	}

	err = DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "decision_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"traffic_profile_id",
			"traffic_forecast_id",
			"decision_source",
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
			"decision_type",
			"track",
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
			"reason",
			"generated_at",
			"updated_at",
		}),
	}).Create(&decisions).Error
	if err != nil {
		return nil, err
	}

	resultDB := DB.Model(&SupplyDecision{}).
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
	var results []*SupplyDecision
	err = resultDB.Order("period_start DESC, id DESC").Find(&results).Error
	return results, err
}

func loadTrafficForecastsForSupplyDecisions(profiles []TrafficProfile, input SupplyDecisionGenerateInput) (map[string]*TrafficForecast, error) {
	if len(profiles) == 0 {
		return map[string]*TrafficForecast{}, nil
	}
	sliceKeys := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		sliceKeys = append(sliceKeys, supplyDecisionSliceKey(profile))
	}

	forecastDB := DB.Model(&TrafficForecast{}).
		Where("source_period_start = ? AND source_period_end = ?", input.PeriodStart, input.PeriodEnd).
		Where("slice_key IN ?", sliceKeys)
	if strings.TrimSpace(input.ModelName) != "" {
		forecastDB = forecastDB.Where("model_name = ?", strings.TrimSpace(input.ModelName))
	}
	if strings.TrimSpace(input.SlaTier) != "" {
		forecastDB = forecastDB.Where("sla_tier = ?", normalizeTrafficProfileSlaTier(input.SlaTier))
	}
	if input.UserId > 0 {
		forecastDB = forecastDB.Where("user_id = ?", input.UserId)
	}

	var rows []TrafficForecast
	if err := forecastDB.Order("generated_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]*TrafficForecast, len(rows))
	for i := range rows {
		row := rows[i]
		if _, exists := result[row.SliceKey]; exists {
			continue
		}
		result[row.SliceKey] = &row
	}
	return result, nil
}

func buildSupplyDecisionFromProfile(profile TrafficProfile, forecast *TrafficForecast, now int64) SupplyDecision {
	sliceKey := supplyDecisionSliceKey(profile)
	decisionSource := SupplyDecisionSourceProfile
	trafficForecastId := 0
	forecastTargetStart := int64(0)
	forecastTargetEnd := int64(0)
	forecastConfidence := float64(0)
	forecastMethod := ""
	demandTokens := profile.DemandTokens
	peakTokens := profile.PeakTokens
	supplyHeadroomTokens := profile.SupplyHeadroomTokens
	gapTokens := peakTokens - supplyHeadroomTokens
	cacheHitRate := profile.CacheHitRate
	slaMetRate := profile.SlaMetRate
	grossProfitQuota := profile.GrossProfitQuota
	avgUnitCostQuota := profile.AvgUnitCostQuota
	reasonPrefix := ""

	if forecast != nil {
		decisionSource = SupplyDecisionSourceForecast
		trafficForecastId = forecast.Id
		forecastTargetStart = forecast.TargetPeriodStart
		forecastTargetEnd = forecast.TargetPeriodEnd
		forecastConfidence = forecast.Confidence
		forecastMethod = forecast.Method
		demandTokens = forecast.ForecastDemandTokens
		peakTokens = forecast.ForecastPeakTokens
		supplyHeadroomTokens = forecast.ForecastHeadroomTokens
		gapTokens = forecast.ForecastGapTokens
		cacheHitRate = forecast.CacheHitRate
		slaMetRate = forecast.SlaMetRate
		grossProfitQuota = forecast.GrossProfitQuota
		avgUnitCostQuota = forecast.AvgUnitCostQuota
		reasonPrefix = fmt.Sprintf(
			"forecast-informed target=%d-%d confidence=%.2f method=%s: ",
			forecast.TargetPeriodStart,
			forecast.TargetPeriodEnd,
			forecast.Confidence,
			forecast.Method,
		)
	}

	if gapTokens < 0 {
		gapTokens = 0
	}

	decisionType := SupplyDecisionTypeThirdPartyProbe
	track := SupplyDecisionTrackThirdParty
	recommendedCapacity := peakTokens
	reason := "keep third-party supply in observation mode"

	switch {
	case gapTokens > 0:
		decisionType = SupplyDecisionTypeThirdPartyRecruit
		track = SupplyDecisionTrackThirdParty
		recommendedCapacity = gapTokens
		reason = "peak demand exceeds supply headroom"
	case cacheHitRate >= 0.5 && grossProfitQuota > 0:
		decisionType = SupplyDecisionTypeSelfHostedEvaluate
		track = SupplyDecisionTrackSelfHosted
		recommendedCapacity = demandTokens
		reason = "cache locality and positive gross profit make this slice a self-hosted candidate"
	case grossProfitQuota > 0:
		decisionType = SupplyDecisionTypeSelfOperatedBuy
		track = SupplyDecisionTrackSelfOperated
		recommendedCapacity = maxInt64(demandTokens, peakTokens)
		reason = "positive gross profit makes this slice a self-operated purchase candidate"
	}

	if recommendedCapacity < 0 {
		recommendedCapacity = 0
	}
	roiScore := float64(grossProfitQuota) +
		cacheHitRate*float64(demandTokens)*avgUnitCostQuota -
		float64(gapTokens)*avgUnitCostQuota
	if math.IsNaN(roiScore) || math.IsInf(roiScore, 0) {
		roiScore = 0
	}

	return SupplyDecision{
		DecisionKey:           supplyDecisionKey(profile),
		TrafficProfileId:      profile.Id,
		TrafficForecastId:     trafficForecastId,
		DecisionSource:        decisionSource,
		SliceKey:              sliceKey,
		ModelName:             profile.ModelName,
		SlaTier:               normalizeTrafficProfileSlaTier(profile.SlaTier),
		UserId:                profile.UserId,
		PeriodStart:           profile.PeriodStart,
		PeriodEnd:             profile.PeriodEnd,
		ForecastTargetStart:   forecastTargetStart,
		ForecastTargetEnd:     forecastTargetEnd,
		ForecastConfidence:    forecastConfidence,
		ForecastMethod:        forecastMethod,
		DecisionType:          decisionType,
		Track:                 track,
		Status:                SupplyDecisionStatusDraft,
		DemandTokens:          demandTokens,
		PeakTokens:            peakTokens,
		SupplyHeadroomTokens:  supplyHeadroomTokens,
		GapTokens:             gapTokens,
		RecommendedCapacity:   recommendedCapacity,
		CacheHitRate:          cacheHitRate,
		SlaMetRate:            slaMetRate,
		GrossProfitQuota:      grossProfitQuota,
		AvgSupplyQualityScore: profile.AvgSupplyQualityScore,
		AvgUnitCostQuota:      avgUnitCostQuota,
		RoiScore:              roiScore,
		Reason:                reasonPrefix + reason,
		GeneratedAt:           now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}

func UpdateSupplyDecisionReview(id int, status string, reviewedBy int, reviewNote string) (*SupplyDecision, error) {
	status = normalizeSupplyDecisionStatus(status)
	if status == SupplyDecisionStatusDraft {
		return nil, errors.New("review status must be approved or rejected")
	}
	now := common.GetTimestamp()
	err := DB.Model(&SupplyDecision{}).Where("id = ?", id).Updates(map[string]any{
		"status":      status,
		"reviewed_at": now,
		"reviewed_by": reviewedBy,
		"review_note": strings.TrimSpace(reviewNote),
		"updated_at":  now,
	}).Error
	if err != nil {
		return nil, err
	}
	return GetSupplyDecisionByID(id)
}

func GetSupplyDecisionByID(id int) (*SupplyDecision, error) {
	var decision SupplyDecision
	err := DB.First(&decision, "id = ?", id).Error
	return &decision, err
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
