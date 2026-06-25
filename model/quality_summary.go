package model

import (
	"strconv"
	"strings"
)

type QualitySummaryFilters struct {
	GroupBy    string
	SupplierId int
	ChannelId  int
	UserId     int
	TokenId    int
	ModelName  string
	Status     string
	StartTime  int64
	EndTime    int64
}

type QualitySummaryRow struct {
	GroupKey              string  `json:"group_key"`
	BucketStart           int64   `json:"bucket_start,omitempty"`
	SupplierId            int     `json:"supplier_id,omitempty"`
	ChannelId             int     `json:"channel_id,omitempty"`
	UserId                int     `json:"user_id,omitempty"`
	ModelName             string  `json:"model_name,omitempty"`
	SlaTier               string  `json:"sla_tier,omitempty"`
	SupplyNode            string  `json:"supply_node,omitempty"`
	TotalRequests         int64   `json:"total_requests"`
	SuccessRequests       int64   `json:"success_requests"`
	ErrorRequests         int64   `json:"error_requests"`
	SuccessRate           float64 `json:"success_rate"`
	AvgLatencyMs          float64 `json:"avg_latency_ms"`
	MaxLatencyMs          int     `json:"max_latency_ms"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCachedTokens     int64   `json:"total_cached_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	TotalSellQuota        int64   `json:"total_sell_quota"`
	TotalCostQuota        int64   `json:"total_cost_quota"`
	GrossProfitQuota      int64   `json:"gross_profit_quota"`
	CacheHitCount         int64   `json:"cache_hit_count"`
	CacheHitRate          float64 `json:"cache_hit_rate"`
}

func normalizeQualityGroupBy(groupBy string) string {
	switch strings.TrimSpace(groupBy) {
	case "channel", "user", "model", "sla_tier", "supply_node", "day":
		return strings.TrimSpace(groupBy)
	default:
		return "supplier"
	}
}

func SearchQualitySummary(filters QualitySummaryFilters) ([]QualitySummaryRow, error) {
	filters.GroupBy = normalizeQualityGroupBy(filters.GroupBy)
	db := DB.Model(&UsageLedger{})
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
	if filters.Status != "" {
		db = db.Where("status = ?", strings.TrimSpace(filters.Status))
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
	case "sla_tier":
		groupExpr = "sla_tier"
		selectPrefix = "sla_tier"
	case "supply_node":
		groupExpr = "supply_node"
		selectPrefix = "supply_node"
	case "day":
		groupExpr = "(created_at / 86400) * 86400"
		selectPrefix = "(created_at / 86400) * 86400 AS bucket_start"
	}

	var rows []QualitySummaryRow
	err := db.Select(selectPrefix + `,
		COUNT(*) AS total_requests,
		COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0) AS success_requests,
		COALESCE(SUM(CASE WHEN status <> 'success' THEN 1 ELSE 0 END), 0) AS error_requests,
		COALESCE(AVG(latency_ms), 0) AS avg_latency_ms,
		COALESCE(MAX(latency_ms), 0) AS max_latency_ms,
		COALESCE(SUM(prompt_tokens), 0) AS total_prompt_tokens,
		COALESCE(SUM(cached_tokens), 0) AS total_cached_tokens,
		COALESCE(SUM(completion_tokens), 0) AS total_completion_tokens,
		COALESCE(SUM(sell_quota), 0) AS total_sell_quota,
		COALESCE(SUM(cost_quota), 0) AS total_cost_quota,
		COALESCE(SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END), 0) AS cache_hit_count`,
	).Group(groupExpr).Order(groupExpr).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].GrossProfitQuota = rows[i].TotalSellQuota - rows[i].TotalCostQuota
		if rows[i].TotalRequests > 0 {
			rows[i].SuccessRate = float64(rows[i].SuccessRequests) / float64(rows[i].TotalRequests)
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
		case "sla_tier":
			rows[i].GroupKey = rows[i].SlaTier
		case "supply_node":
			rows[i].GroupKey = rows[i].SupplyNode
		case "day":
			rows[i].GroupKey = strconv.FormatInt(rows[i].BucketStart, 10)
		}
	}
	return rows, nil
}
