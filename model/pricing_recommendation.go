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
	PricingRecommendationStatusDraft    = "draft"
	PricingRecommendationStatusApproved = "approved"
	PricingRecommendationStatusRejected = "rejected"

	PricingRecommendationActionRaisePrice   = "raise_price"
	PricingRecommendationActionKeepPrice    = "keep_price"
	PricingRecommendationActionShareSavings = "share_savings"
)

type PricingRecommendation struct {
	Id                        int     `json:"id"`
	RecommendationKey         string  `json:"recommendation_key" gorm:"size:512;not null;uniqueIndex:uk_pricing_recommendation_key"`
	TrafficProfileId          int     `json:"traffic_profile_id" gorm:"index;default:0"`
	SliceKey                  string  `json:"slice_key" gorm:"size:256;not null;index"`
	ModelName                 string  `json:"model_name" gorm:"size:128;not null;index"`
	SlaTier                   string  `json:"sla_tier" gorm:"size:64;not null;default:'default';index"`
	UserId                    int     `json:"user_id" gorm:"index;default:0"`
	PeriodStart               int64   `json:"period_start" gorm:"bigint;not null;index"`
	PeriodEnd                 int64   `json:"period_end" gorm:"bigint;not null;index"`
	Status                    string  `json:"status" gorm:"size:32;not null;default:'draft';index"`
	Action                    string  `json:"action" gorm:"size:32;not null;index"`
	RequestCount              int64   `json:"request_count" gorm:"default:0"`
	DemandTokens              int64   `json:"demand_tokens" gorm:"default:0"`
	PeakTokens                int64   `json:"peak_tokens" gorm:"default:0"`
	SupplyHeadroomTokens      int64   `json:"supply_headroom_tokens" gorm:"default:0"`
	CacheHitRate              float64 `json:"cache_hit_rate" gorm:"default:0"`
	SlaMetRate                float64 `json:"sla_met_rate" gorm:"default:0"`
	AvgLatencyMs              float64 `json:"avg_latency_ms" gorm:"default:0"`
	MaxLatencyMs              int     `json:"max_latency_ms" gorm:"default:0"`
	TotalSellQuota            int64   `json:"total_sell_quota" gorm:"default:0"`
	TotalCostQuota            int64   `json:"total_cost_quota" gorm:"default:0"`
	GrossProfitQuota          int64   `json:"gross_profit_quota" gorm:"default:0"`
	CurrentUnitPriceQuota     float64 `json:"current_unit_price_quota" gorm:"default:0"`
	CurrentUnitCostQuota      float64 `json:"current_unit_cost_quota" gorm:"default:0"`
	CurrentMarginRate         float64 `json:"current_margin_rate" gorm:"default:0"`
	RecommendedUnitPriceQuota float64 `json:"recommended_unit_price_quota" gorm:"default:0"`
	RecommendedMarginRate     float64 `json:"recommended_margin_rate" gorm:"default:0"`
	AvgSupplyQualityScore     float64 `json:"avg_supply_quality_score" gorm:"default:0"`
	AvgUnitCostQuota          float64 `json:"avg_unit_cost_quota" gorm:"default:0"`
	Reason                    string  `json:"reason" gorm:"type:text"`
	GeneratedAt               int64   `json:"generated_at" gorm:"bigint;index"`
	ReviewedAt                int64   `json:"reviewed_at" gorm:"bigint;default:0"`
	ReviewedBy                int     `json:"reviewed_by" gorm:"default:0;index"`
	ReviewNote                string  `json:"review_note,omitempty" gorm:"type:text"`
	CreatedAt                 int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt                 int64   `json:"updated_at" gorm:"bigint"`
}

type PricingRecommendationGenerateInput struct {
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
	ModelName   string `json:"model_name"`
	SlaTier     string `json:"sla_tier"`
	UserId      int    `json:"user_id"`
}

type PricingRecommendationFilters struct {
	ModelName string
	SlaTier   string
	UserId    int
	Status    string
	Action    string
	StartTime int64
	EndTime   int64
}

type PricingRecommendationReviewInput struct {
	ReviewNote string `json:"review_note"`
}

func validatePricingRecommendationGenerateInput(input PricingRecommendationGenerateInput) error {
	if input.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if input.PeriodEnd <= input.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	return nil
}

func normalizePricingRecommendationStatus(value string) string {
	switch strings.TrimSpace(value) {
	case PricingRecommendationStatusDraft:
		return PricingRecommendationStatusDraft
	case PricingRecommendationStatusApproved:
		return PricingRecommendationStatusApproved
	case PricingRecommendationStatusRejected:
		return PricingRecommendationStatusRejected
	default:
		return ""
	}
}

func normalizePricingRecommendationAction(value string) string {
	switch strings.TrimSpace(value) {
	case PricingRecommendationActionRaisePrice:
		return PricingRecommendationActionRaisePrice
	case PricingRecommendationActionKeepPrice:
		return PricingRecommendationActionKeepPrice
	case PricingRecommendationActionShareSavings:
		return PricingRecommendationActionShareSavings
	default:
		return ""
	}
}

func pricingRecommendationKey(profile TrafficProfile) string {
	return fmt.Sprintf("pricing:profile:%s|period:%d-%d", profile.SliceKey, profile.PeriodStart, profile.PeriodEnd)
}

func SearchPricingRecommendations(filters PricingRecommendationFilters, offset int, limit int) ([]*PricingRecommendation, int64, error) {
	db := DB.Model(&PricingRecommendation{})
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if strings.TrimSpace(filters.SlaTier) != "" {
		db = db.Where("sla_tier = ?", normalizeTrafficProfileSlaTier(filters.SlaTier))
	}
	if filters.UserId > 0 {
		db = db.Where("user_id = ?", filters.UserId)
	}
	if status := normalizePricingRecommendationStatus(filters.Status); status != "" {
		db = db.Where("status = ?", status)
	}
	if action := normalizePricingRecommendationAction(filters.Action); action != "" {
		db = db.Where("action = ?", action)
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
	var recommendations []*PricingRecommendation
	err := db.Offset(offset).Limit(limit).Order("period_start DESC, id DESC").Find(&recommendations).Error
	return recommendations, total, err
}

func GeneratePricingRecommendations(input PricingRecommendationGenerateInput) ([]*PricingRecommendation, error) {
	if err := validatePricingRecommendationGenerateInput(input); err != nil {
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
		return []*PricingRecommendation{}, nil
	}

	now := common.GetTimestamp()
	recommendations := make([]PricingRecommendation, 0, len(profiles))
	for _, profile := range profiles {
		recommendations = append(recommendations, buildPricingRecommendationFromProfile(profile, now))
	}

	err := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "recommendation_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"traffic_profile_id",
			"slice_key",
			"model_name",
			"sla_tier",
			"user_id",
			"period_start",
			"period_end",
			"action",
			"request_count",
			"demand_tokens",
			"peak_tokens",
			"supply_headroom_tokens",
			"cache_hit_rate",
			"sla_met_rate",
			"avg_latency_ms",
			"max_latency_ms",
			"total_sell_quota",
			"total_cost_quota",
			"gross_profit_quota",
			"current_unit_price_quota",
			"current_unit_cost_quota",
			"current_margin_rate",
			"recommended_unit_price_quota",
			"recommended_margin_rate",
			"avg_supply_quality_score",
			"avg_unit_cost_quota",
			"reason",
			"generated_at",
			"updated_at",
		}),
	}).Create(&recommendations).Error
	if err != nil {
		return nil, err
	}

	resultDB := DB.Model(&PricingRecommendation{}).
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
	var results []*PricingRecommendation
	err = resultDB.Order("period_start DESC, id DESC").Find(&results).Error
	return results, err
}

func buildPricingRecommendationFromProfile(profile TrafficProfile, now int64) PricingRecommendation {
	currentUnitPrice := pricingRecommendationQuotaPerToken(profile.TotalSellQuota, profile.DemandTokens)
	currentUnitCost := pricingRecommendationQuotaPerToken(profile.TotalCostQuota, profile.DemandTokens)
	currentMarginRate := pricingRecommendationMarginRate(currentUnitPrice, currentUnitCost)
	recommendedUnitPrice := currentUnitPrice
	action := PricingRecommendationActionKeepPrice
	reason := "unit economics and SLA evidence support keeping current pricing under observation"

	switch {
	case profile.TotalSellQuota <= 0 || profile.GrossProfitQuota <= 0:
		action = PricingRecommendationActionRaisePrice
		recommendedUnitPrice = math.Max(currentUnitPrice*1.10, pricingRecommendationPriceForMargin(currentUnitCost, 0.30))
		reason = "observed gross profit is non-positive; raise price or lower SLA before expanding commitment"
	case profile.SlaMetRate < 0.95:
		action = PricingRecommendationActionRaisePrice
		recommendedUnitPrice = math.Max(currentUnitPrice*1.10, pricingRecommendationPriceForMargin(currentUnitCost, 0.35))
		reason = "SLA attainment is below threshold; price must cover reliability risk before a stronger promise"
	case normalizeTrafficProfileSlaTier(profile.SlaTier) != trafficProfileDefaultSlaTier && profile.PeakTokens > 0 && profile.SupplyHeadroomTokens < profile.PeakTokens:
		action = PricingRecommendationActionRaisePrice
		recommendedUnitPrice = math.Max(currentUnitPrice*1.10, pricingRecommendationPriceForMargin(currentUnitCost, 0.35))
		reason = "higher SLA tier has insufficient supply headroom; raise price or reduce commitment"
	case currentMarginRate >= 0.45 && profile.CacheHitRate >= 0.5 && profile.SlaMetRate >= 0.99:
		action = PricingRecommendationActionShareSavings
		recommendedUnitPrice = math.Max(currentUnitPrice*0.90, pricingRecommendationPriceForMargin(currentUnitCost, 0.30))
		reason = "cache locality and stable SLA create room to share efficiency savings"
	}

	if currentUnitPrice <= 0 && recommendedUnitPrice <= 0 && currentUnitCost > 0 {
		recommendedUnitPrice = pricingRecommendationPriceForMargin(currentUnitCost, 0.30)
	}
	if math.IsNaN(recommendedUnitPrice) || math.IsInf(recommendedUnitPrice, 0) || recommendedUnitPrice < 0 {
		recommendedUnitPrice = 0
	}

	return PricingRecommendation{
		RecommendationKey:         pricingRecommendationKey(profile),
		TrafficProfileId:          profile.Id,
		SliceKey:                  profile.SliceKey,
		ModelName:                 profile.ModelName,
		SlaTier:                   normalizeTrafficProfileSlaTier(profile.SlaTier),
		UserId:                    profile.UserId,
		PeriodStart:               profile.PeriodStart,
		PeriodEnd:                 profile.PeriodEnd,
		Status:                    PricingRecommendationStatusDraft,
		Action:                    action,
		RequestCount:              profile.RequestCount,
		DemandTokens:              profile.DemandTokens,
		PeakTokens:                profile.PeakTokens,
		SupplyHeadroomTokens:      profile.SupplyHeadroomTokens,
		CacheHitRate:              profile.CacheHitRate,
		SlaMetRate:                profile.SlaMetRate,
		AvgLatencyMs:              profile.AvgLatencyMs,
		MaxLatencyMs:              profile.MaxLatencyMs,
		TotalSellQuota:            profile.TotalSellQuota,
		TotalCostQuota:            profile.TotalCostQuota,
		GrossProfitQuota:          profile.GrossProfitQuota,
		CurrentUnitPriceQuota:     currentUnitPrice,
		CurrentUnitCostQuota:      currentUnitCost,
		CurrentMarginRate:         currentMarginRate,
		RecommendedUnitPriceQuota: recommendedUnitPrice,
		RecommendedMarginRate:     pricingRecommendationMarginRate(recommendedUnitPrice, currentUnitCost),
		AvgSupplyQualityScore:     profile.AvgSupplyQualityScore,
		AvgUnitCostQuota:          profile.AvgUnitCostQuota,
		Reason:                    reason,
		GeneratedAt:               now,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
}

func UpdatePricingRecommendationReview(id int, status string, reviewedBy int, reviewNote string) (*PricingRecommendation, error) {
	status = normalizePricingRecommendationStatus(status)
	if status == "" || status == PricingRecommendationStatusDraft {
		return nil, errors.New("review status must be approved or rejected")
	}
	now := common.GetTimestamp()
	err := DB.Model(&PricingRecommendation{}).Where("id = ?", id).Updates(map[string]any{
		"status":      status,
		"reviewed_at": now,
		"reviewed_by": reviewedBy,
		"review_note": strings.TrimSpace(reviewNote),
		"updated_at":  now,
	}).Error
	if err != nil {
		return nil, err
	}
	return GetPricingRecommendationByID(id)
}

func GetPricingRecommendationByID(id int) (*PricingRecommendation, error) {
	var recommendation PricingRecommendation
	err := DB.First(&recommendation, "id = ?", id).Error
	return &recommendation, err
}

func pricingRecommendationQuotaPerToken(quota int64, tokens int64) float64 {
	if tokens <= 0 {
		return 0
	}
	value := float64(quota) / float64(tokens)
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}

func pricingRecommendationMarginRate(unitPrice float64, unitCost float64) float64 {
	if unitPrice <= 0 {
		return 0
	}
	value := (unitPrice - unitCost) / unitPrice
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}

func pricingRecommendationPriceForMargin(unitCost float64, marginRate float64) float64 {
	if unitCost <= 0 {
		return 0
	}
	if marginRate < 0 {
		marginRate = 0
	}
	if marginRate >= 0.95 {
		marginRate = 0.95
	}
	value := unitCost / (1 - marginRate)
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}
