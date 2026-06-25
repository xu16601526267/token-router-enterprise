package model

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SettlementSubjectSupplier = "supplier"
	SettlementSubjectUser     = "user"

	SettlementStatusDraft     = "draft"
	SettlementStatusConfirmed = "confirmed"
)

type SettlementStatement struct {
	Id                    int     `json:"id"`
	SubjectType           string  `json:"subject_type" gorm:"size:32;not null;uniqueIndex:uk_settlement_subject_period"`
	SupplierId            int     `json:"supplier_id" gorm:"index;default:0;uniqueIndex:uk_settlement_subject_period"`
	UserId                int     `json:"user_id" gorm:"index;default:0;uniqueIndex:uk_settlement_subject_period"`
	PeriodStart           int64   `json:"period_start" gorm:"bigint;not null;uniqueIndex:uk_settlement_subject_period"`
	PeriodEnd             int64   `json:"period_end" gorm:"bigint;not null;uniqueIndex:uk_settlement_subject_period"`
	TotalSellQuota        int64   `json:"total_sell_quota" gorm:"default:0"`
	TotalCostQuota        int64   `json:"total_cost_quota" gorm:"default:0"`
	GrossProfitQuota      int64   `json:"gross_profit_quota" gorm:"default:0"`
	TotalRequests         int64   `json:"total_requests" gorm:"default:0"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens" gorm:"default:0"`
	TotalCachedTokens     int64   `json:"total_cached_tokens" gorm:"default:0"`
	TotalCompletionTokens int64   `json:"total_completion_tokens" gorm:"default:0"`
	CacheHitRate          float64 `json:"cache_hit_rate" gorm:"default:0"`
	Status                string  `json:"status" gorm:"size:32;default:'draft';index"`
	GeneratedAt           int64   `json:"generated_at" gorm:"bigint;index"`
	CreatedAt             int64   `json:"created_at" gorm:"bigint;index"`
	UpdatedAt             int64   `json:"updated_at" gorm:"bigint;index"`
}

func (s *SettlementStatement) normalize() {
	s.SubjectType = strings.TrimSpace(s.SubjectType)
	s.Status = strings.TrimSpace(s.Status)
	if s.Status == "" {
		s.Status = SettlementStatusDraft
	}
	now := common.GetTimestamp()
	if s.GeneratedAt == 0 {
		s.GeneratedAt = now
	}
	if s.CreatedAt == 0 {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	s.GrossProfitQuota = s.TotalSellQuota - s.TotalCostQuota
	if s.TotalRequests <= 0 {
		s.CacheHitRate = 0
	}
}

type SettlementStatementGenerateInput struct {
	SubjectType string `json:"subject_type"`
	SupplierId  int    `json:"supplier_id"`
	UserId      int    `json:"user_id"`
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`
}

type SettlementStatementFilters struct {
	SubjectType string
	SupplierId  int
	UserId      int
	Status      string
	StartTime   int64
	EndTime     int64
}

type marginAggregate struct {
	TotalSellQuota        int64
	TotalCostQuota        int64
	TotalRequests         int64
	TotalPromptTokens     int64
	TotalCachedTokens     int64
	TotalCompletionTokens int64
	CacheHitCount         int64
}

func validateSettlementStatementInput(input SettlementStatementGenerateInput) error {
	input.SubjectType = strings.TrimSpace(input.SubjectType)
	if input.SubjectType != SettlementSubjectSupplier && input.SubjectType != SettlementSubjectUser {
		return errors.New("subject_type must be supplier or user")
	}
	if input.SubjectType == SettlementSubjectSupplier && input.SupplierId <= 0 {
		return errors.New("supplier_id is required for supplier settlement")
	}
	if input.SubjectType == SettlementSubjectUser && input.UserId <= 0 {
		return errors.New("user_id is required for user settlement")
	}
	if input.PeriodStart <= 0 || input.PeriodEnd <= 0 || input.PeriodEnd < input.PeriodStart {
		return errors.New("invalid settlement period")
	}
	return nil
}

func statementLedgerQuery(input SettlementStatementGenerateInput) *gorm.DB {
	db := DB.Model(&UsageLedger{}).
		Where("created_at >= ? AND created_at <= ?", input.PeriodStart, input.PeriodEnd).
		Where("status = ?", "success")
	if input.SubjectType == SettlementSubjectSupplier {
		db = db.Where("supplier_id = ?", input.SupplierId)
	}
	if input.SubjectType == SettlementSubjectUser {
		db = db.Where("user_id = ?", input.UserId)
	}
	return db
}

func scanMarginAggregate(db *gorm.DB) (marginAggregate, error) {
	var agg marginAggregate
	err := db.Select(`
		COUNT(*) AS total_requests,
		COALESCE(SUM(sell_quota), 0) AS total_sell_quota,
		COALESCE(SUM(cost_quota), 0) AS total_cost_quota,
		COALESCE(SUM(prompt_tokens), 0) AS total_prompt_tokens,
		COALESCE(SUM(cached_tokens), 0) AS total_cached_tokens,
		COALESCE(SUM(completion_tokens), 0) AS total_completion_tokens,
		COALESCE(SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END), 0) AS cache_hit_count`,
	).Scan(&agg).Error
	return agg, err
}

func GenerateSettlementStatement(input SettlementStatementGenerateInput) (*SettlementStatement, error) {
	input.SubjectType = strings.TrimSpace(input.SubjectType)
	if err := validateSettlementStatementInput(input); err != nil {
		return nil, err
	}
	agg, err := scanMarginAggregate(statementLedgerQuery(input))
	if err != nil {
		return nil, err
	}
	cacheHitRate := 0.0
	if agg.TotalRequests > 0 {
		cacheHitRate = float64(agg.CacheHitCount) / float64(agg.TotalRequests)
	}
	statement := &SettlementStatement{
		SubjectType:           input.SubjectType,
		SupplierId:            input.SupplierId,
		UserId:                input.UserId,
		PeriodStart:           input.PeriodStart,
		PeriodEnd:             input.PeriodEnd,
		TotalSellQuota:        agg.TotalSellQuota,
		TotalCostQuota:        agg.TotalCostQuota,
		TotalRequests:         agg.TotalRequests,
		TotalPromptTokens:     agg.TotalPromptTokens,
		TotalCachedTokens:     agg.TotalCachedTokens,
		TotalCompletionTokens: agg.TotalCompletionTokens,
		CacheHitRate:          cacheHitRate,
		Status:                SettlementStatusDraft,
	}
	statement.normalize()
	err = DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "subject_type"},
			{Name: "supplier_id"},
			{Name: "user_id"},
			{Name: "period_start"},
			{Name: "period_end"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"total_sell_quota",
			"total_cost_quota",
			"gross_profit_quota",
			"total_requests",
			"total_prompt_tokens",
			"total_cached_tokens",
			"total_completion_tokens",
			"cache_hit_rate",
			"status",
			"generated_at",
			"updated_at",
		}),
	}).Create(statement).Error
	if err != nil {
		return nil, err
	}
	return GetSettlementStatementBySubjectPeriod(input)
}

func GetSettlementStatementBySubjectPeriod(input SettlementStatementGenerateInput) (*SettlementStatement, error) {
	var statement SettlementStatement
	err := DB.Where("subject_type = ? AND supplier_id = ? AND user_id = ? AND period_start = ? AND period_end = ?",
		input.SubjectType, input.SupplierId, input.UserId, input.PeriodStart, input.PeriodEnd,
	).First(&statement).Error
	return &statement, err
}

func GetSettlementStatementByID(id int) (*SettlementStatement, error) {
	var statement SettlementStatement
	err := DB.First(&statement, "id = ?", id).Error
	return &statement, err
}

func SearchSettlementStatements(filters SettlementStatementFilters, offset int, limit int) ([]*SettlementStatement, int64, error) {
	db := DB.Model(&SettlementStatement{})
	if filters.SubjectType != "" {
		db = db.Where("subject_type = ?", strings.TrimSpace(filters.SubjectType))
	}
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if filters.UserId > 0 {
		db = db.Where("user_id = ?", filters.UserId)
	}
	if filters.Status != "" {
		db = db.Where("status = ?", strings.TrimSpace(filters.Status))
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
	var statements []*SettlementStatement
	err := db.Offset(offset).Limit(limit).Order("period_start DESC, id DESC").Find(&statements).Error
	return statements, total, err
}

func UsageLedgerFiltersForStatement(statement *SettlementStatement) UsageLedgerFilters {
	if statement == nil {
		return UsageLedgerFilters{}
	}
	filters := UsageLedgerFilters{
		SupplierId: statement.SupplierId,
		UserId:     statement.UserId,
		StartTime:  statement.PeriodStart,
		EndTime:    statement.PeriodEnd,
		Status:     "success",
	}
	if statement.SubjectType != SettlementSubjectSupplier {
		filters.SupplierId = 0
	}
	if statement.SubjectType != SettlementSubjectUser {
		filters.UserId = 0
	}
	return filters
}

type MarginSummaryFilters struct {
	GroupBy    string
	SupplierId int
	ChannelId  int
	UserId     int
	TokenId    int
	ModelName  string
	StartTime  int64
	EndTime    int64
}

type MarginSummaryRow struct {
	GroupKey              string  `json:"group_key"`
	BucketStart           int64   `json:"bucket_start,omitempty"`
	SupplierId            int     `json:"supplier_id,omitempty"`
	ChannelId             int     `json:"channel_id,omitempty"`
	UserId                int     `json:"user_id,omitempty"`
	ModelName             string  `json:"model_name,omitempty"`
	TotalRequests         int64   `json:"total_requests"`
	TotalSellQuota        int64   `json:"total_sell_quota"`
	TotalCostQuota        int64   `json:"total_cost_quota"`
	GrossProfitQuota      int64   `json:"gross_profit_quota"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCachedTokens     int64   `json:"total_cached_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	CacheHitCount         int64   `json:"cache_hit_count"`
	CacheHitRate          float64 `json:"cache_hit_rate"`
}

func normalizeMarginGroupBy(groupBy string) string {
	switch strings.TrimSpace(groupBy) {
	case "channel", "user", "model", "day":
		return strings.TrimSpace(groupBy)
	default:
		return "supplier"
	}
}

func SearchMarginSummary(filters MarginSummaryFilters) ([]MarginSummaryRow, error) {
	filters.GroupBy = normalizeMarginGroupBy(filters.GroupBy)
	db := DB.Model(&UsageLedger{}).Where("status = ?", "success")
	if filters.SupplierId > 0 {
		db = db.Where("supplier_id = ?", filters.SupplierId)
	}
	if filters.ChannelId > 0 {
		db = db.Where("channel_id = ?", filters.ChannelId)
	}
	if filters.UserId > 0 {
		db = db.Where("user_id = ?", filters.UserId)
	}
	if filters.TokenId > 0 {
		db = db.Where("token_id = ?", filters.TokenId)
	}
	if filters.ModelName != "" {
		db = db.Where("model_name = ?", strings.TrimSpace(filters.ModelName))
	}
	if filters.StartTime > 0 {
		db = db.Where("created_at >= ?", filters.StartTime)
	}
	if filters.EndTime > 0 {
		db = db.Where("created_at <= ?", filters.EndTime)
	}

	groupExpr := "supplier_id"
	selectPrefix := "supplier_id"
	switch filters.GroupBy {
	case "channel":
		groupExpr = "channel_id"
		selectPrefix = "channel_id"
	case "user":
		groupExpr = "user_id"
		selectPrefix = "user_id"
	case "model":
		groupExpr = "model_name"
		selectPrefix = "model_name"
	case "day":
		groupExpr = "(created_at / 86400) * 86400"
		selectPrefix = "(created_at / 86400) * 86400 AS bucket_start"
	}

	var rows []MarginSummaryRow
	err := db.Select(selectPrefix + `,
		COUNT(*) AS total_requests,
		COALESCE(SUM(sell_quota), 0) AS total_sell_quota,
		COALESCE(SUM(cost_quota), 0) AS total_cost_quota,
		COALESCE(SUM(prompt_tokens), 0) AS total_prompt_tokens,
		COALESCE(SUM(cached_tokens), 0) AS total_cached_tokens,
		COALESCE(SUM(completion_tokens), 0) AS total_completion_tokens,
		COALESCE(SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END), 0) AS cache_hit_count`,
	).Group(groupExpr).Order(groupExpr).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].GrossProfitQuota = rows[i].TotalSellQuota - rows[i].TotalCostQuota
		if rows[i].TotalRequests > 0 {
			rows[i].CacheHitRate = float64(rows[i].CacheHitCount) / float64(rows[i].TotalRequests)
		}
		switch filters.GroupBy {
		case "supplier":
			rows[i].GroupKey = strconv.Itoa(rows[i].SupplierId)
		case "channel":
			rows[i].GroupKey = strconv.Itoa(rows[i].ChannelId)
		case "user":
			rows[i].GroupKey = strconv.Itoa(rows[i].UserId)
		case "model":
			rows[i].GroupKey = rows[i].ModelName
		case "day":
			rows[i].GroupKey = strconv.FormatInt(rows[i].BucketStart, 10)
		}
	}
	return rows, nil
}
