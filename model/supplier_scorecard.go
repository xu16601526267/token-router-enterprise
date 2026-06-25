package model

import (
	"errors"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm/clause"
)

const (
	SupplierScorecardGradeA = "A"
	SupplierScorecardGradeB = "B"
	SupplierScorecardGradeC = "C"
	SupplierScorecardGradeD = "D"
)

type SupplierScorecard struct {
	Id                    int     `json:"id"`
	SupplierId            int     `json:"supplier_id" gorm:"not null;uniqueIndex:uk_supplier_scorecard_period,priority:1;index"`
	PeriodStart           int64   `json:"period_start" gorm:"bigint;not null;uniqueIndex:uk_supplier_scorecard_period,priority:2;index"`
	PeriodEnd             int64   `json:"period_end" gorm:"bigint;not null;uniqueIndex:uk_supplier_scorecard_period,priority:3;index"`
	TotalRequests         int64   `json:"total_requests" gorm:"default:0"`
	SuccessRequests       int64   `json:"success_requests" gorm:"default:0"`
	ErrorRequests         int64   `json:"error_requests" gorm:"default:0"`
	SuccessRate           float64 `json:"success_rate" gorm:"default:0"`
	AvgLatencyMs          float64 `json:"avg_latency_ms" gorm:"default:0"`
	MaxLatencyMs          int     `json:"max_latency_ms" gorm:"default:0"`
	CacheHitCount         int64   `json:"cache_hit_count" gorm:"default:0"`
	CacheHitRate          float64 `json:"cache_hit_rate" gorm:"default:0"`
	TotalSellQuota        int64   `json:"total_sell_quota" gorm:"default:0"`
	TotalCostQuota        int64   `json:"total_cost_quota" gorm:"default:0"`
	GrossProfitQuota      int64   `json:"gross_profit_quota" gorm:"default:0"`
	SupplyCapacityTokens  int64   `json:"supply_capacity_tokens" gorm:"default:0"`
	SupplyUsedTokens      int64   `json:"supply_used_tokens" gorm:"default:0"`
	SupplyHeadroomTokens  int64   `json:"supply_headroom_tokens" gorm:"default:0"`
	AvgSupplyQualityScore float64 `json:"avg_supply_quality_score" gorm:"default:0"`
	AvgUnitCostQuota      float64 `json:"avg_unit_cost_quota" gorm:"default:0"`
	Score                 float64 `json:"score" gorm:"default:0;index"`
	Grade                 string  `json:"grade" gorm:"size:8;not null;default:'D';index"`
	GeneratedAt           int64   `json:"generated_at" gorm:"bigint;index"`
	CreatedAt             int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt             int64   `json:"updated_at" gorm:"bigint"`
}

type SupplierScorecardGenerateInput struct {
	PeriodStart int64 `json:"period_start"`
	PeriodEnd   int64 `json:"period_end"`
	SupplierId  int   `json:"supplier_id"`
}

type SupplierScorecardFilters struct {
	SupplierId int
	Grade      string
	StartTime  int64
	EndTime    int64
}

type supplierScorecardCapacityAggregate struct {
	CapacityTokens int64
	UsedTokens     int64
	HeadroomTokens int64
	QualitySum     float64
	UnitCostSum    float64
	Count          int64
}

func validateSupplierScorecardGenerateInput(input SupplierScorecardGenerateInput) error {
	if input.PeriodStart <= 0 {
		return errors.New("period_start is required")
	}
	if input.PeriodEnd <= input.PeriodStart {
		return errors.New("period_end must be greater than period_start")
	}
	return nil
}

func normalizeSupplierScorecardGrade(grade string) string {
	switch strings.ToUpper(strings.TrimSpace(grade)) {
	case SupplierScorecardGradeA:
		return SupplierScorecardGradeA
	case SupplierScorecardGradeB:
		return SupplierScorecardGradeB
	case SupplierScorecardGradeC:
		return SupplierScorecardGradeC
	case SupplierScorecardGradeD:
		return SupplierScorecardGradeD
	default:
		return ""
	}
}

func SearchSupplierScorecards(filters SupplierScorecardFilters, offset int, limit int) ([]*SupplierScorecard, int64, error) {
	db := DB.Model(&SupplierScorecard{})
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
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
	var scorecards []*SupplierScorecard
	err := db.Offset(offset).Limit(limit).Order("period_start DESC, score DESC, id DESC").Find(&scorecards).Error
	return scorecards, total, err
}

func GenerateSupplierScorecards(input SupplierScorecardGenerateInput) ([]*SupplierScorecard, error) {
	if err := validateSupplierScorecardGenerateInput(input); err != nil {
		return nil, err
	}

	qualityRows, err := SearchQualitySummary(QualitySummaryFilters{
		GroupBy:    "supplier",
		SupplierId: input.SupplierId,
		StartTime:  input.PeriodStart,
		EndTime:    input.PeriodEnd,
	})
	if err != nil {
		return nil, err
	}
	if len(qualityRows) == 0 {
		return []*SupplierScorecard{}, nil
	}

	capacityBySupplier, err := supplierScorecardCapacityBySupplier(input)
	if err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	scorecards := make([]SupplierScorecard, 0, len(qualityRows))
	for _, row := range qualityRows {
		if row.SupplierId <= 0 {
			continue
		}
		scorecard := buildSupplierScorecard(row, capacityBySupplier[row.SupplierId], input.PeriodStart, input.PeriodEnd, now)
		scorecards = append(scorecards, scorecard)
	}
	if len(scorecards) == 0 {
		return []*SupplierScorecard{}, nil
	}

	err = DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "supplier_id"},
			{Name: "period_start"},
			{Name: "period_end"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"total_requests",
			"success_requests",
			"error_requests",
			"success_rate",
			"avg_latency_ms",
			"max_latency_ms",
			"cache_hit_count",
			"cache_hit_rate",
			"total_sell_quota",
			"total_cost_quota",
			"gross_profit_quota",
			"supply_capacity_tokens",
			"supply_used_tokens",
			"supply_headroom_tokens",
			"avg_supply_quality_score",
			"avg_unit_cost_quota",
			"score",
			"grade",
			"generated_at",
			"updated_at",
		}),
	}).Create(&scorecards).Error
	if err != nil {
		return nil, err
	}

	resultDB := DB.Model(&SupplierScorecard{}).
		Where("period_start = ? AND period_end = ?", input.PeriodStart, input.PeriodEnd)
	if input.SupplierId > 0 {
		resultDB = resultDB.Where("supplier_id = ?", input.SupplierId)
	}
	var results []*SupplierScorecard
	err = resultDB.Order("period_start DESC, score DESC, id DESC").Find(&results).Error
	return results, err
}

func supplierScorecardCapacityBySupplier(input SupplierScorecardGenerateInput) (map[int]supplierScorecardCapacityAggregate, error) {
	db := DB.Model(&SupplyCapacity{}).
		Where("period_end >= ? AND period_start <= ?", input.PeriodStart, input.PeriodEnd)
	if input.SupplierId > 0 {
		db = db.Where("supplier_id = ?", input.SupplierId)
	}

	var capacities []SupplyCapacity
	if err := db.Find(&capacities).Error; err != nil {
		return nil, err
	}

	bySupplier := make(map[int]supplierScorecardCapacityAggregate)
	for _, capacity := range capacities {
		if capacity.SupplierId <= 0 {
			continue
		}
		agg := bySupplier[capacity.SupplierId]
		agg.CapacityTokens += capacity.CapacityTokens
		agg.UsedTokens += capacity.UsedTokens
		agg.HeadroomTokens += capacity.HeadroomTokens
		agg.QualitySum += capacity.QualityScore
		agg.UnitCostSum += capacity.UnitCostQuota
		agg.Count++
		bySupplier[capacity.SupplierId] = agg
	}
	return bySupplier, nil
}

func buildSupplierScorecard(row QualitySummaryRow, capacity supplierScorecardCapacityAggregate, periodStart int64, periodEnd int64, now int64) SupplierScorecard {
	scorecard := SupplierScorecard{
		SupplierId:           row.SupplierId,
		PeriodStart:          periodStart,
		PeriodEnd:            periodEnd,
		TotalRequests:        row.TotalRequests,
		SuccessRequests:      row.SuccessRequests,
		ErrorRequests:        row.ErrorRequests,
		SuccessRate:          row.SuccessRate,
		AvgLatencyMs:         row.AvgLatencyMs,
		MaxLatencyMs:         row.MaxLatencyMs,
		CacheHitCount:        row.CacheHitCount,
		CacheHitRate:         row.CacheHitRate,
		TotalSellQuota:       row.TotalSellQuota,
		TotalCostQuota:       row.TotalCostQuota,
		GrossProfitQuota:     row.GrossProfitQuota,
		SupplyCapacityTokens: capacity.CapacityTokens,
		SupplyUsedTokens:     capacity.UsedTokens,
		SupplyHeadroomTokens: capacity.HeadroomTokens,
		GeneratedAt:          now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if capacity.Count > 0 {
		scorecard.AvgSupplyQualityScore = capacity.QualitySum / float64(capacity.Count)
		scorecard.AvgUnitCostQuota = capacity.UnitCostSum / float64(capacity.Count)
	}
	scorecard.Score = supplierScorecardScore(scorecard)
	scorecard.Grade = supplierScorecardGrade(scorecard.Score)
	return scorecard
}

func supplierScorecardScore(scorecard SupplierScorecard) float64 {
	qualityScore := clampFloat(scorecard.AvgSupplyQualityScore, 0, 100)
	latencyScore := 1 - scorecard.AvgLatencyMs/5000
	latencyScore = clampFloat(latencyScore, 0, 1)
	marginScore := 0.0
	if scorecard.GrossProfitQuota > 0 {
		marginScore = 1
	}
	score := scorecard.SuccessRate*40 +
		scorecard.CacheHitRate*20 +
		qualityScore*0.2 +
		latencyScore*10 +
		marginScore*10
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0
	}
	return clampFloat(score, 0, 100)
}

func supplierScorecardGrade(score float64) string {
	switch {
	case score >= 85:
		return SupplierScorecardGradeA
	case score >= 70:
		return SupplierScorecardGradeB
	case score >= 55:
		return SupplierScorecardGradeC
	default:
		return SupplierScorecardGradeD
	}
}

func clampFloat(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
