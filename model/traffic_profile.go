package model

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm/clause"
)

const trafficProfileDefaultSlaTier = "default"

type TrafficProfile struct {
	Id                    int     `json:"id"`
	SliceKey              string  `json:"slice_key" gorm:"size:256;not null;uniqueIndex:uk_traffic_profile_slice_period,priority:1"`
	ModelName             string  `json:"model_name" gorm:"size:128;not null;index"`
	SlaTier               string  `json:"sla_tier" gorm:"size:64;not null;default:'default';index"`
	UserId                int     `json:"user_id" gorm:"index;default:0"`
	PeriodStart           int64   `json:"period_start" gorm:"bigint;not null;uniqueIndex:uk_traffic_profile_slice_period,priority:2;index"`
	PeriodEnd             int64   `json:"period_end" gorm:"bigint;not null;uniqueIndex:uk_traffic_profile_slice_period,priority:3;index"`
	RequestCount          int64   `json:"request_count" gorm:"default:0"`
	SuccessRequestCount   int64   `json:"success_request_count" gorm:"default:0"`
	DemandTokens          int64   `json:"demand_tokens" gorm:"default:0"`
	PeakTokens            int64   `json:"peak_tokens" gorm:"default:0"`
	PeakRatio             float64 `json:"peak_ratio" gorm:"default:0"`
	UniqueSessions        int64   `json:"unique_sessions" gorm:"default:0"`
	CacheHitCount         int64   `json:"cache_hit_count" gorm:"default:0"`
	CacheHitRate          float64 `json:"cache_hit_rate" gorm:"default:0"`
	TotalCachedTokens     int64   `json:"total_cached_tokens" gorm:"default:0"`
	SlaMetRate            float64 `json:"sla_met_rate" gorm:"default:0"`
	AvgLatencyMs          float64 `json:"avg_latency_ms" gorm:"default:0"`
	MaxLatencyMs          int     `json:"max_latency_ms" gorm:"default:0"`
	TotalSellQuota        int64   `json:"total_sell_quota" gorm:"default:0"`
	TotalCostQuota        int64   `json:"total_cost_quota" gorm:"default:0"`
	GrossProfitQuota      int64   `json:"gross_profit_quota" gorm:"default:0"`
	SupplyCapacityTokens  int64   `json:"supply_capacity_tokens" gorm:"default:0"`
	SupplyUsedTokens      int64   `json:"supply_used_tokens" gorm:"default:0"`
	SupplyHeadroomTokens  int64   `json:"supply_headroom_tokens" gorm:"default:0"`
	AvgSupplyQualityScore float64 `json:"avg_supply_quality_score" gorm:"default:0"`
	AvgUnitCostQuota      float64 `json:"avg_unit_cost_quota" gorm:"default:0"`
	GeneratedAt           int64   `json:"generated_at" gorm:"bigint;index"`
	CreatedAt             int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt             int64   `json:"updated_at" gorm:"bigint"`
}

type TrafficProfileGenerateInput struct {
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
	ModelName   string `json:"model_name"`
	SlaTier     string `json:"sla_tier"`
	UserId      int    `json:"user_id"`
}

type TrafficProfileFilters struct {
	ModelName string
	SlaTier   string
	UserId    int
	StartTime int64
	EndTime   int64
}

type trafficProfileAccumulator struct {
	profile        TrafficProfile
	sessionSet     map[string]struct{}
	hourTokens     map[int64]int64
	latencyTotalMs int64
}

type trafficProfileSupplyAggregate struct {
	CapacityTokens int64
	UsedTokens     int64
	HeadroomTokens int64
	QualitySum     float64
	UnitCostSum    float64
	Count          int64
}

func normalizeTrafficProfileSlaTier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return trafficProfileDefaultSlaTier
	}
	return value
}

func trafficProfileSliceKey(modelName string, slaTier string, userId int) string {
	return fmt.Sprintf("model:%s|sla:%s|user:%d", strings.TrimSpace(modelName), normalizeTrafficProfileSlaTier(slaTier), userId)
}

func validateTrafficProfileGenerateInput(input TrafficProfileGenerateInput) error {
	if input.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if input.PeriodEnd <= input.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	return nil
}

func SearchTrafficProfiles(filters TrafficProfileFilters, offset int, limit int) ([]*TrafficProfile, int64, error) {
	db := DB.Model(&TrafficProfile{})
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if strings.TrimSpace(filters.SlaTier) != "" {
		db = db.Where("sla_tier = ?", normalizeTrafficProfileSlaTier(filters.SlaTier))
	}
	if filters.UserId > 0 {
		db = db.Where("user_id = ?", filters.UserId)
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
	var profiles []*TrafficProfile
	if err := db.Offset(offset).Limit(limit).Order("period_start DESC, id DESC").Find(&profiles).Error; err != nil {
		return nil, 0, err
	}
	return profiles, total, nil
}

func GenerateTrafficProfiles(input TrafficProfileGenerateInput) ([]*TrafficProfile, error) {
	if err := validateTrafficProfileGenerateInput(input); err != nil {
		return nil, err
	}

	db := DB.Model(&UsageLedger{}).
		Where("created_at >= ? AND created_at <= ?", input.PeriodStart, input.PeriodEnd)
	if strings.TrimSpace(input.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(input.ModelName))
	}
	if strings.TrimSpace(input.SlaTier) != "" {
		slaTier := normalizeTrafficProfileSlaTier(input.SlaTier)
		if slaTier == trafficProfileDefaultSlaTier {
			db = db.Where("(sla_tier = ? OR sla_tier = '')", slaTier)
		} else {
			db = db.Where("sla_tier = ?", slaTier)
		}
	}
	if input.UserId > 0 {
		db = db.Where("user_id = ?", input.UserId)
	}

	var ledgers []UsageLedger
	if err := db.Order("created_at ASC, id ASC").Find(&ledgers).Error; err != nil {
		return nil, err
	}
	if len(ledgers) == 0 {
		return []*TrafficProfile{}, nil
	}

	supplyByModel, err := trafficProfileSupplyByModel(input)
	if err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	accumulators := make(map[string]*trafficProfileAccumulator)
	for _, ledger := range ledgers {
		modelName := strings.TrimSpace(ledger.ModelName)
		slaTier := normalizeTrafficProfileSlaTier(ledger.SlaTier)
		sliceKey := trafficProfileSliceKey(modelName, slaTier, ledger.UserId)
		acc, ok := accumulators[sliceKey]
		if !ok {
			acc = &trafficProfileAccumulator{
				profile: TrafficProfile{
					SliceKey:    sliceKey,
					ModelName:   modelName,
					SlaTier:     slaTier,
					UserId:      ledger.UserId,
					PeriodStart: input.PeriodStart,
					PeriodEnd:   input.PeriodEnd,
					GeneratedAt: now,
					CreatedAt:   now,
					UpdatedAt:   now,
				},
				sessionSet: make(map[string]struct{}),
				hourTokens: make(map[int64]int64),
			}
			accumulators[sliceKey] = acc
		}

		requestTokens := int64(ledger.PromptTokens + ledger.CompletionTokens)
		acc.profile.RequestCount++
		if ledger.Status == "success" {
			acc.profile.SuccessRequestCount++
		}
		acc.profile.DemandTokens += requestTokens
		acc.profile.TotalCachedTokens += int64(ledger.CachedTokens)
		acc.profile.TotalSellQuota += int64(ledger.SellQuota)
		acc.profile.TotalCostQuota += int64(ledger.CostQuota)
		if ledger.CacheHit {
			acc.profile.CacheHitCount++
		}
		if strings.TrimSpace(ledger.SessionId) != "" {
			acc.sessionSet[strings.TrimSpace(ledger.SessionId)] = struct{}{}
		}
		if ledger.LatencyMs > acc.profile.MaxLatencyMs {
			acc.profile.MaxLatencyMs = ledger.LatencyMs
		}
		acc.latencyTotalMs += int64(ledger.LatencyMs)
		hourBucket := (ledger.CreatedAt / 3600) * 3600
		acc.hourTokens[hourBucket] += requestTokens
	}

	profiles := make([]TrafficProfile, 0, len(accumulators))
	for _, acc := range accumulators {
		finalizeTrafficProfileAccumulator(acc, supplyByModel[acc.profile.ModelName])
		profiles = append(profiles, acc.profile)
	}

	err = DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "slice_key"},
			{Name: "period_start"},
			{Name: "period_end"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"model_name",
			"sla_tier",
			"user_id",
			"request_count",
			"success_request_count",
			"demand_tokens",
			"peak_tokens",
			"peak_ratio",
			"unique_sessions",
			"cache_hit_count",
			"cache_hit_rate",
			"total_cached_tokens",
			"sla_met_rate",
			"avg_latency_ms",
			"max_latency_ms",
			"total_sell_quota",
			"total_cost_quota",
			"gross_profit_quota",
			"supply_capacity_tokens",
			"supply_used_tokens",
			"supply_headroom_tokens",
			"avg_supply_quality_score",
			"avg_unit_cost_quota",
			"generated_at",
			"updated_at",
		}),
	}).Create(&profiles).Error
	if err != nil {
		return nil, err
	}

	resultDB := DB.Model(&TrafficProfile{}).
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
	var results []*TrafficProfile
	err = resultDB.Order("period_start DESC, id DESC").Find(&results).Error
	return results, err
}

func trafficProfileSupplyByModel(input TrafficProfileGenerateInput) (map[string]trafficProfileSupplyAggregate, error) {
	db := DB.Model(&SupplyCapacity{}).
		Where("period_end >= ? AND period_start <= ?", input.PeriodStart, input.PeriodEnd)
	if strings.TrimSpace(input.ModelName) != "" {
		db = db.Where("(model_name = ? OR model_name = '')", strings.TrimSpace(input.ModelName))
	}

	var capacities []SupplyCapacity
	if err := db.Find(&capacities).Error; err != nil {
		return nil, err
	}

	byModel := make(map[string]trafficProfileSupplyAggregate)
	for _, capacity := range capacities {
		modelName := strings.TrimSpace(capacity.ModelName)
		if modelName == "" {
			continue
		}
		agg := byModel[modelName]
		agg.CapacityTokens += capacity.CapacityTokens
		agg.UsedTokens += capacity.UsedTokens
		agg.HeadroomTokens += capacity.HeadroomTokens
		agg.QualitySum += capacity.QualityScore
		agg.UnitCostSum += capacity.UnitCostQuota
		agg.Count++
		byModel[modelName] = agg
	}
	return byModel, nil
}

func finalizeTrafficProfileAccumulator(acc *trafficProfileAccumulator, supply trafficProfileSupplyAggregate) {
	profile := &acc.profile
	profile.GrossProfitQuota = profile.TotalSellQuota - profile.TotalCostQuota
	profile.UniqueSessions = int64(len(acc.sessionSet))
	if profile.RequestCount > 0 {
		profile.CacheHitRate = float64(profile.CacheHitCount) / float64(profile.RequestCount)
		profile.SlaMetRate = float64(profile.SuccessRequestCount) / float64(profile.RequestCount)
		profile.AvgLatencyMs = float64(acc.latencyTotalMs) / float64(profile.RequestCount)
	}

	var activeBucketCount int64
	for _, tokens := range acc.hourTokens {
		activeBucketCount++
		if tokens > profile.PeakTokens {
			profile.PeakTokens = tokens
		}
	}
	if activeBucketCount > 0 && profile.DemandTokens > 0 {
		avgActiveBucketTokens := float64(profile.DemandTokens) / float64(activeBucketCount)
		profile.PeakRatio = float64(profile.PeakTokens) / avgActiveBucketTokens
		if math.IsNaN(profile.PeakRatio) || math.IsInf(profile.PeakRatio, 0) {
			profile.PeakRatio = 0
		}
	}

	profile.SupplyCapacityTokens = supply.CapacityTokens
	profile.SupplyUsedTokens = supply.UsedTokens
	profile.SupplyHeadroomTokens = supply.HeadroomTokens
	if supply.Count > 0 {
		profile.AvgSupplyQualityScore = supply.QualitySum / float64(supply.Count)
		profile.AvgUnitCostQuota = supply.UnitCostSum / float64(supply.Count)
	}
}
