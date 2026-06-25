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
	TrafficForecastMethodMovingAverage         = "moving_average"
	TrafficForecastMethodWeightedMovingAverage = "weighted_moving_average"
	TrafficForecastMethodSeasonalAnomaly       = "seasonal_anomaly_adjusted"

	TrafficForecastAnomalyNotEvaluated        = "not_evaluated"
	TrafficForecastAnomalyInsufficientHistory = "insufficient_history"
	TrafficForecastAnomalyNormal              = "normal"
	TrafficForecastAnomalySpike               = "spike"
	TrafficForecastAnomalyDrop                = "drop"

	defaultTrafficForecastAnomalyThresholdRate = 2.0
)

type TrafficForecast struct {
	Id                     int     `json:"id"`
	ForecastKey            string  `json:"forecast_key" gorm:"size:512;not null;uniqueIndex:uk_traffic_forecast_key"`
	SliceKey               string  `json:"slice_key" gorm:"size:256;not null;index"`
	ModelName              string  `json:"model_name" gorm:"size:128;not null;index"`
	SlaTier                string  `json:"sla_tier" gorm:"size:64;not null;default:'default';index"`
	UserId                 int     `json:"user_id" gorm:"index;default:0"`
	SourcePeriodStart      int64   `json:"source_period_start" gorm:"bigint;not null;index"`
	SourcePeriodEnd        int64   `json:"source_period_end" gorm:"bigint;not null;index"`
	TargetPeriodStart      int64   `json:"target_period_start" gorm:"bigint;not null;index"`
	TargetPeriodEnd        int64   `json:"target_period_end" gorm:"bigint;not null;index"`
	SourceProfileCount     int64   `json:"source_profile_count" gorm:"default:0"`
	ObservedRequestCount   int64   `json:"observed_request_count" gorm:"default:0"`
	ObservedDemandTokens   int64   `json:"observed_demand_tokens" gorm:"default:0"`
	ObservedPeakTokens     int64   `json:"observed_peak_tokens" gorm:"default:0"`
	BaselineDemandTokens   int64   `json:"baseline_demand_tokens" gorm:"default:0"`
	ForecastDemandTokens   int64   `json:"forecast_demand_tokens" gorm:"default:0"`
	ForecastPeakTokens     int64   `json:"forecast_peak_tokens" gorm:"default:0"`
	ForecastHeadroomTokens int64   `json:"forecast_headroom_tokens" gorm:"default:0"`
	ForecastGapTokens      int64   `json:"forecast_gap_tokens" gorm:"default:0"`
	TrendDemandDeltaTokens int64   `json:"trend_demand_delta_tokens" gorm:"default:0"`
	TrendDemandDeltaRate   float64 `json:"trend_demand_delta_rate" gorm:"default:0"`
	SeasonalPeriodCount    int     `json:"seasonal_period_count" gorm:"default:0"`
	SeasonalIndex          float64 `json:"seasonal_index" gorm:"default:1"`
	SeasonalDemandTokens   int64   `json:"seasonal_demand_tokens" gorm:"default:0"`
	AnomalyStatus          string  `json:"anomaly_status" gorm:"size:64;default:'not_evaluated';index"`
	AnomalyProfileId       int     `json:"anomaly_profile_id" gorm:"default:0;index"`
	AnomalyDemandRatio     float64 `json:"anomaly_demand_ratio" gorm:"default:0"`
	CacheHitRate           float64 `json:"cache_hit_rate" gorm:"default:0"`
	SlaMetRate             float64 `json:"sla_met_rate" gorm:"default:0"`
	GrossProfitQuota       int64   `json:"gross_profit_quota" gorm:"default:0"`
	AvgUnitCostQuota       float64 `json:"avg_unit_cost_quota" gorm:"default:0"`
	Confidence             float64 `json:"confidence" gorm:"default:0"`
	Method                 string  `json:"method" gorm:"size:64;not null;index"`
	Reason                 string  `json:"reason" gorm:"type:text"`
	GeneratedAt            int64   `json:"generated_at" gorm:"bigint;index"`
	CreatedAt              int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt              int64   `json:"updated_at" gorm:"bigint"`
}

type TrafficForecastGenerateInput struct {
	PeriodStart          int64   `json:"period_start"`
	PeriodEnd            int64   `json:"period_end"`
	TargetPeriodStart    int64   `json:"target_period_start"`
	TargetPeriodEnd      int64   `json:"target_period_end"`
	ModelName            string  `json:"model_name"`
	SlaTier              string  `json:"sla_tier"`
	UserId               int     `json:"user_id"`
	SeasonalPeriodCount  int     `json:"seasonal_period_count"`
	AnomalyGuard         bool    `json:"anomaly_guard"`
	AnomalyThresholdRate float64 `json:"anomaly_threshold_rate"`
}

type TrafficForecastFilters struct {
	ModelName         string
	SlaTier           string
	UserId            int
	Method            string
	SourcePeriodStart int64
	SourcePeriodEnd   int64
	TargetPeriodStart int64
	TargetPeriodEnd   int64
}

type trafficForecastAccumulator struct {
	forecast             TrafficForecast
	weightedDemandSum    int64
	weightedCacheSum     float64
	weightedSlaSum       float64
	weightedProfitSum    int64
	weightedUnitCostSum  float64
	weightSum            int64
	latestProfileEnd     int64
	latestProfileId      int
	firstDemandTokens    int64
	firstProfileSeen     bool
	latestDemandTokens   int64
	profileDemands       []int64
	profileIds           []int
	seasonalPeriodCount  int
	anomalyGuard         bool
	anomalyThresholdRate float64
}

func validateTrafficForecastGenerateInput(input TrafficForecastGenerateInput) error {
	if input.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if input.PeriodEnd <= input.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	if input.TargetPeriodEnd > 0 && input.TargetPeriodStart <= 0 {
		return errors.New("target_period_start is required when target_period_end is set")
	}
	if input.TargetPeriodStart > 0 && input.TargetPeriodEnd <= input.TargetPeriodStart {
		return errors.New("target_period_end must be greater than target_period_start")
	}
	if input.SeasonalPeriodCount == 1 || input.SeasonalPeriodCount < 0 {
		return errors.New("seasonal_period_count must be 0 or greater than 1")
	}
	if input.AnomalyThresholdRate < 0 {
		return errors.New("anomaly_threshold_rate cannot be negative")
	}
	return nil
}

func SearchTrafficForecasts(filters TrafficForecastFilters, offset int, limit int) ([]*TrafficForecast, int64, error) {
	db := DB.Model(&TrafficForecast{})
	if strings.TrimSpace(filters.ModelName) != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if strings.TrimSpace(filters.SlaTier) != "" {
		db = db.Where("sla_tier = ?", normalizeTrafficProfileSlaTier(filters.SlaTier))
	}
	if filters.UserId > 0 {
		db = db.Where("user_id = ?", filters.UserId)
	}
	if strings.TrimSpace(filters.Method) != "" {
		db = db.Where("method = ?", strings.TrimSpace(filters.Method))
	}
	if filters.SourcePeriodStart > 0 {
		db = db.Where("source_period_end >= ?", filters.SourcePeriodStart)
	}
	if filters.SourcePeriodEnd > 0 {
		db = db.Where("source_period_start <= ?", filters.SourcePeriodEnd)
	}
	if filters.TargetPeriodStart > 0 {
		db = db.Where("target_period_end >= ?", filters.TargetPeriodStart)
	}
	if filters.TargetPeriodEnd > 0 {
		db = db.Where("target_period_start <= ?", filters.TargetPeriodEnd)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var forecasts []*TrafficForecast
	err := db.Offset(offset).Limit(limit).Order("target_period_start DESC, id DESC").Find(&forecasts).Error
	return forecasts, total, err
}

func GenerateTrafficForecasts(input TrafficForecastGenerateInput) ([]*TrafficForecast, error) {
	if err := validateTrafficForecastGenerateInput(input); err != nil {
		return nil, err
	}
	targetStart, targetEnd := trafficForecastTargetPeriod(input)

	profileDB := DB.Model(&TrafficProfile{}).
		Where("period_start >= ? AND period_end <= ?", input.PeriodStart, input.PeriodEnd)
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
		return []*TrafficForecast{}, nil
	}

	now := common.GetTimestamp()
	accumulators := make(map[string]*trafficForecastAccumulator)
	for _, profile := range profiles {
		sliceKey := strings.TrimSpace(profile.SliceKey)
		if sliceKey == "" {
			sliceKey = trafficProfileSliceKey(profile.ModelName, profile.SlaTier, profile.UserId)
		}
		acc, ok := accumulators[sliceKey]
		if !ok {
			acc = &trafficForecastAccumulator{
				forecast: TrafficForecast{
					ForecastKey:       trafficForecastKey(sliceKey, input.PeriodStart, input.PeriodEnd, targetStart, targetEnd),
					SliceKey:          sliceKey,
					ModelName:         profile.ModelName,
					SlaTier:           normalizeTrafficProfileSlaTier(profile.SlaTier),
					UserId:            profile.UserId,
					SourcePeriodStart: input.PeriodStart,
					SourcePeriodEnd:   input.PeriodEnd,
					TargetPeriodStart: targetStart,
					TargetPeriodEnd:   targetEnd,
					Method:            trafficForecastMethod(input),
					SeasonalIndex:     1,
					AnomalyStatus:     TrafficForecastAnomalyNotEvaluated,
					GeneratedAt:       now,
					CreatedAt:         now,
					UpdatedAt:         now,
				},
				seasonalPeriodCount:  input.SeasonalPeriodCount,
				anomalyGuard:         input.AnomalyGuard,
				anomalyThresholdRate: normalizeTrafficForecastAnomalyThreshold(input.AnomalyThresholdRate),
			}
			accumulators[sliceKey] = acc
		}
		if !acc.firstProfileSeen {
			acc.firstDemandTokens = profile.DemandTokens
			acc.firstProfileSeen = true
		}
		acc.forecast.SourceProfileCount++
		weight := acc.forecast.SourceProfileCount
		acc.weightSum += weight
		acc.profileDemands = append(acc.profileDemands, profile.DemandTokens)
		acc.profileIds = append(acc.profileIds, profile.Id)
		acc.latestDemandTokens = profile.DemandTokens
		acc.forecast.ObservedRequestCount += profile.RequestCount
		acc.forecast.ObservedDemandTokens += profile.DemandTokens
		acc.weightedDemandSum += profile.DemandTokens * weight
		if profile.PeakTokens > acc.forecast.ObservedPeakTokens {
			acc.forecast.ObservedPeakTokens = profile.PeakTokens
			acc.forecast.ForecastPeakTokens = profile.PeakTokens
		}
		acc.weightedCacheSum += profile.CacheHitRate * float64(weight)
		acc.weightedSlaSum += profile.SlaMetRate * float64(weight)
		acc.weightedProfitSum += profile.GrossProfitQuota * weight
		acc.weightedUnitCostSum += profile.AvgUnitCostQuota * float64(weight)
		if profile.PeriodEnd > acc.latestProfileEnd || (profile.PeriodEnd == acc.latestProfileEnd && profile.Id > acc.latestProfileId) {
			acc.latestProfileEnd = profile.PeriodEnd
			acc.latestProfileId = profile.Id
			acc.forecast.ForecastHeadroomTokens = profile.SupplyHeadroomTokens
		}
	}

	forecasts := make([]TrafficForecast, 0, len(accumulators))
	for _, acc := range accumulators {
		finalizeTrafficForecastAccumulator(acc)
		forecasts = append(forecasts, acc.forecast)
	}

	err := DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "forecast_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"slice_key",
			"model_name",
			"sla_tier",
			"user_id",
			"source_period_start",
			"source_period_end",
			"target_period_start",
			"target_period_end",
			"source_profile_count",
			"observed_request_count",
			"observed_demand_tokens",
			"observed_peak_tokens",
			"baseline_demand_tokens",
			"forecast_demand_tokens",
			"forecast_peak_tokens",
			"forecast_headroom_tokens",
			"forecast_gap_tokens",
			"trend_demand_delta_tokens",
			"trend_demand_delta_rate",
			"seasonal_period_count",
			"seasonal_index",
			"seasonal_demand_tokens",
			"anomaly_status",
			"anomaly_profile_id",
			"anomaly_demand_ratio",
			"cache_hit_rate",
			"sla_met_rate",
			"gross_profit_quota",
			"avg_unit_cost_quota",
			"confidence",
			"method",
			"reason",
			"generated_at",
			"updated_at",
		}),
	}).Create(&forecasts).Error
	if err != nil {
		return nil, err
	}

	resultDB := DB.Model(&TrafficForecast{}).
		Where("source_period_start = ? AND source_period_end = ?", input.PeriodStart, input.PeriodEnd).
		Where("target_period_start = ? AND target_period_end = ?", targetStart, targetEnd)
	if strings.TrimSpace(input.ModelName) != "" {
		resultDB = resultDB.Where("model_name = ?", strings.TrimSpace(input.ModelName))
	}
	if strings.TrimSpace(input.SlaTier) != "" {
		resultDB = resultDB.Where("sla_tier = ?", normalizeTrafficProfileSlaTier(input.SlaTier))
	}
	if input.UserId > 0 {
		resultDB = resultDB.Where("user_id = ?", input.UserId)
	}
	var results []*TrafficForecast
	err = resultDB.Order("target_period_start DESC, id DESC").Find(&results).Error
	return results, err
}

func trafficForecastTargetPeriod(input TrafficForecastGenerateInput) (int64, int64) {
	if input.TargetPeriodStart > 0 && input.TargetPeriodEnd > input.TargetPeriodStart {
		return input.TargetPeriodStart, input.TargetPeriodEnd
	}
	duration := input.PeriodEnd - input.PeriodStart
	return input.PeriodEnd, input.PeriodEnd + duration
}

func trafficForecastKey(sliceKey string, sourceStart int64, sourceEnd int64, targetStart int64, targetEnd int64) string {
	return fmt.Sprintf("forecast:%s|source:%d-%d|target:%d-%d", sliceKey, sourceStart, sourceEnd, targetStart, targetEnd)
}

func trafficForecastMethod(input TrafficForecastGenerateInput) string {
	if input.SeasonalPeriodCount > 1 || input.AnomalyGuard {
		return TrafficForecastMethodSeasonalAnomaly
	}
	return TrafficForecastMethodWeightedMovingAverage
}

func normalizeTrafficForecastAnomalyThreshold(value float64) float64 {
	if value <= 0 {
		return defaultTrafficForecastAnomalyThresholdRate
	}
	return value
}

func finalizeTrafficForecastAccumulator(acc *trafficForecastAccumulator) {
	if acc == nil || acc.forecast.SourceProfileCount <= 0 {
		return
	}
	count := acc.forecast.SourceProfileCount
	if acc.weightSum <= 0 {
		return
	}
	acc.forecast.BaselineDemandTokens = ceilDivInt64(acc.weightedDemandSum, acc.weightSum)
	acc.forecast.ForecastDemandTokens = acc.forecast.BaselineDemandTokens
	acc.forecast.CacheHitRate = acc.weightedCacheSum / float64(acc.weightSum)
	acc.forecast.SlaMetRate = acc.weightedSlaSum / float64(acc.weightSum)
	acc.forecast.GrossProfitQuota = ceilDivInt64(acc.weightedProfitSum, acc.weightSum)
	acc.forecast.AvgUnitCostQuota = acc.weightedUnitCostSum / float64(acc.weightSum)
	acc.forecast.Confidence = math.Min(float64(count)/3.0, 1)
	acc.forecast.SeasonalDemandTokens = acc.forecast.BaselineDemandTokens
	if acc.firstProfileSeen {
		acc.forecast.TrendDemandDeltaTokens = acc.latestDemandTokens - acc.firstDemandTokens
		if acc.firstDemandTokens > 0 {
			acc.forecast.TrendDemandDeltaRate = float64(acc.forecast.TrendDemandDeltaTokens) / float64(acc.firstDemandTokens)
		}
	}
	if acc.forecast.Method == TrafficForecastMethodSeasonalAnomaly {
		applySeasonalTrafficForecastAdjustment(acc)
		applyTrafficForecastAnomalyGuard(acc)
	}
	acc.forecast.ForecastGapTokens = acc.forecast.ForecastPeakTokens - acc.forecast.ForecastHeadroomTokens
	if acc.forecast.ForecastGapTokens < 0 {
		acc.forecast.ForecastGapTokens = 0
	}
	acc.forecast.Reason = trafficForecastReason(acc)
}

func ceilDivInt64(value int64, divisor int64) int64 {
	if divisor <= 0 {
		return 0
	}
	if value >= 0 {
		return (value + divisor - 1) / divisor
	}
	return value / divisor
}

func applySeasonalTrafficForecastAdjustment(acc *trafficForecastAccumulator) {
	acc.forecast.SeasonalIndex = 1
	if acc.seasonalPeriodCount > 1 {
		acc.forecast.SeasonalPeriodCount = acc.seasonalPeriodCount
	}
	if acc.seasonalPeriodCount <= 1 || len(acc.profileDemands) < acc.seasonalPeriodCount {
		return
	}
	overallAverage := float64(acc.forecast.ObservedDemandTokens) / float64(len(acc.profileDemands))
	if overallAverage <= 0 {
		return
	}
	targetBucket := len(acc.profileDemands) % acc.seasonalPeriodCount
	var bucketSum int64
	var bucketCount int64
	for index, demand := range acc.profileDemands {
		if index%acc.seasonalPeriodCount == targetBucket {
			bucketSum += demand
			bucketCount++
		}
	}
	if bucketCount <= 0 {
		return
	}
	bucketAverage := float64(bucketSum) / float64(bucketCount)
	acc.forecast.SeasonalIndex = bucketAverage / overallAverage
	acc.forecast.SeasonalDemandTokens = ceilFloatToInt64(float64(acc.forecast.BaselineDemandTokens) * acc.forecast.SeasonalIndex)
	acc.forecast.ForecastDemandTokens = acc.forecast.SeasonalDemandTokens
}

func applyTrafficForecastAnomalyGuard(acc *trafficForecastAccumulator) {
	if !acc.anomalyGuard {
		acc.forecast.AnomalyStatus = TrafficForecastAnomalyNotEvaluated
		return
	}
	if len(acc.profileDemands) < 3 {
		acc.forecast.AnomalyStatus = TrafficForecastAnomalyInsufficientHistory
		return
	}
	latestDemand := acc.profileDemands[len(acc.profileDemands)-1]
	latestProfileID := acc.profileIds[len(acc.profileIds)-1]
	var previousSum int64
	for _, demand := range acc.profileDemands[:len(acc.profileDemands)-1] {
		previousSum += demand
	}
	previousAverage := float64(previousSum) / float64(len(acc.profileDemands)-1)
	if previousAverage <= 0 {
		acc.forecast.AnomalyStatus = TrafficForecastAnomalyInsufficientHistory
		return
	}
	ratio := float64(latestDemand) / previousAverage
	acc.forecast.AnomalyProfileId = latestProfileID
	acc.forecast.AnomalyDemandRatio = ratio
	threshold := normalizeTrafficForecastAnomalyThreshold(acc.anomalyThresholdRate)
	switch {
	case ratio >= threshold:
		acc.forecast.AnomalyStatus = TrafficForecastAnomalySpike
		acc.forecast.ForecastDemandTokens = ceilFloatToInt64((float64(acc.forecast.ForecastDemandTokens) + previousAverage) / 2)
	case ratio <= 1/threshold:
		acc.forecast.AnomalyStatus = TrafficForecastAnomalyDrop
		acc.forecast.ForecastDemandTokens = ceilFloatToInt64((float64(acc.forecast.ForecastDemandTokens) + previousAverage) / 2)
	default:
		acc.forecast.AnomalyStatus = TrafficForecastAnomalyNormal
	}
}

func trafficForecastReason(acc *trafficForecastAccumulator) string {
	if acc.forecast.Method != TrafficForecastMethodSeasonalAnomaly {
		return fmt.Sprintf("recency-weighted moving average over %d traffic profile(s); weight_sum=%d latest_weight=%d; peak uses max observed demand and headroom uses latest source profile", acc.forecast.SourceProfileCount, acc.weightSum, acc.forecast.SourceProfileCount)
	}
	return fmt.Sprintf("seasonal/anomaly adjusted forecast over %d traffic profile(s); baseline=%d seasonal_period_count=%d seasonal_index=%.4f seasonal_demand=%d anomaly_status=%s anomaly_ratio=%.4f; peak uses max observed demand and headroom uses latest source profile",
		acc.forecast.SourceProfileCount,
		acc.forecast.BaselineDemandTokens,
		acc.forecast.SeasonalPeriodCount,
		acc.forecast.SeasonalIndex,
		acc.forecast.SeasonalDemandTokens,
		acc.forecast.AnomalyStatus,
		acc.forecast.AnomalyDemandRatio,
	)
}

func ceilFloatToInt64(value float64) int64 {
	if value <= 0 {
		return 0
	}
	return int64(math.Ceil(value))
}
