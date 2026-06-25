package controller

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const enterpriseOverviewMaxRangeSeconds = int64(31 * 24 * 60 * 60)

type enterpriseOverviewRange struct {
	StartTimestamp int64 `json:"start_timestamp"`
	EndTimestamp   int64 `json:"end_timestamp"`
}

type enterpriseOverviewMetrics struct {
	TotalRequests        int64   `json:"total_requests"`
	TotalTokens          int64   `json:"total_tokens"`
	TotalQuota           int64   `json:"total_quota"`
	EstimatedCost        float64 `json:"estimated_cost"`
	SuccessRate          float64 `json:"success_rate"`
	AverageLatencyMs     float64 `json:"average_latency_ms"`
	TotalUsers           int64   `json:"total_users"`
	ActiveUsers          int64   `json:"active_users"`
	TotalChannels        int64   `json:"total_channels"`
	HealthyChannels      int64   `json:"healthy_channels"`
	LowBalanceChannels   int64   `json:"low_balance_channels"`
	ActiveAPIKeys        int64   `json:"active_api_keys"`
	TotalSuppliers       int64   `json:"total_suppliers"`
	HealthySuppliers     int64   `json:"healthy_suppliers"`
	ActivePolicies       int64   `json:"active_policies"`
	OpenInsights         int64   `json:"open_insights"`
	PendingApprovals     int64   `json:"pending_approvals"`
	GrossProfitQuota     int64   `json:"gross_profit_quota"`
	GrossMarginRate      float64 `json:"gross_margin_rate"`
	EstimatedGrossProfit float64 `json:"estimated_gross_profit"`
}

type enterpriseOverviewTrendPoint struct {
	Timestamp int64 `json:"timestamp"`
	Requests  int64 `json:"requests"`
	Tokens    int64 `json:"tokens"`
	Quota     int64 `json:"quota"`
}

type enterpriseOverviewRankingItem struct {
	Name     string  `json:"name"`
	Requests int64   `json:"requests"`
	Tokens   int64   `json:"tokens"`
	Quota    int64   `json:"quota"`
	Share    float64 `json:"share"`
}

type enterpriseOverviewChannelItem struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Status       int     `json:"status"`
	ResponseTime int     `json:"response_time"`
	Balance      float64 `json:"balance"`
	UsedQuota    int64   `json:"used_quota"`
	Models       string  `json:"models"`
	Group        string  `json:"group"`
}

type enterpriseOverviewInsight struct {
	ID                int     `json:"id"`
	Title             string  `json:"title"`
	Summary           string  `json:"summary"`
	Severity          string  `json:"severity"`
	Category          string  `json:"category"`
	ModelName         string  `json:"model_name"`
	RecommendedAction string  `json:"recommended_action"`
	SLAMetRate        float64 `json:"sla_met_rate"`
	GeneratedAt       int64   `json:"generated_at"`
}

type enterpriseOverviewData struct {
	GeneratedAt int64                           `json:"generated_at"`
	Range       enterpriseOverviewRange         `json:"range"`
	Metrics     enterpriseOverviewMetrics       `json:"metrics"`
	Trend       []enterpriseOverviewTrendPoint  `json:"trend"`
	TopModels   []enterpriseOverviewRankingItem `json:"top_models"`
	TopUsers    []enterpriseOverviewRankingItem `json:"top_users"`
	Channels    []enterpriseOverviewChannelItem `json:"channels"`
	Insights    []enterpriseOverviewInsight     `json:"insights"`
}

type enterpriseOverviewAggregate struct {
	Requests int64
	Tokens   int64
	Quota    int64
}

func parseEnterpriseOverviewRange(c *gin.Context) (int64, int64) {
	now := time.Now().Unix()
	endTimestamp := now
	startTimestamp := now - 7*24*60*60

	if raw := c.Query("end_timestamp"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			endTimestamp = parsed
		}
	}
	if raw := c.Query("start_timestamp"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			startTimestamp = parsed
		}
	}

	if endTimestamp <= startTimestamp {
		startTimestamp = endTimestamp - 7*24*60*60
	}
	if endTimestamp-startTimestamp > enterpriseOverviewMaxRangeSeconds {
		startTimestamp = endTimestamp - enterpriseOverviewMaxRangeSeconds
	}
	return startTimestamp, endTimestamp
}

func quotaToCurrency(quota int64) float64 {
	if common.QuotaPerUnit <= 0 {
		return 0
	}
	return float64(quota) / common.QuotaPerUnit
}

func rankingItemsFromMap(values map[string]enterpriseOverviewAggregate, totalRequests int64, limit int) []enterpriseOverviewRankingItem {
	items := make([]enterpriseOverviewRankingItem, 0, len(values))
	for name, value := range values {
		if name == "" {
			name = "未分类"
		}
		share := 0.0
		if totalRequests > 0 {
			share = float64(value.Requests) / float64(totalRequests)
		}
		items = append(items, enterpriseOverviewRankingItem{
			Name:     name,
			Requests: value.Requests,
			Tokens:   value.Tokens,
			Quota:    value.Quota,
			Share:    share,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Requests == items[j].Requests {
			return items[i].Quota > items[j].Quota
		}
		return items[i].Requests > items[j].Requests
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

// GetEnterpriseOverview returns a read-only, admin-scoped aggregation used by
// the B2B control cockpit. It intentionally reuses existing tables and does not
// change legacy API semantics, making the enterprise UI additive and backwards
// compatible with the original console.
func GetEnterpriseOverview(c *gin.Context) {
	startTimestamp, endTimestamp := parseEnterpriseOverviewRange(c)
	quotaRows, quotaErr := model.GetAllQuotaDates(startTimestamp, endTimestamp, "")
	if quotaErr != nil {
		common.ApiError(c, quotaErr)
		return
	}

	metrics := enterpriseOverviewMetrics{}
	trendMap := make(map[int64]enterpriseOverviewAggregate)
	modelMap := make(map[string]enterpriseOverviewAggregate)
	for _, row := range quotaRows {
		if row == nil {
			continue
		}
		metrics.TotalRequests += int64(row.Count)
		metrics.TotalTokens += int64(row.TokenUsed)
		metrics.TotalQuota += int64(row.Quota)

		day := time.Unix(row.CreatedAt, 0).UTC()
		bucket := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC).Unix()
		trendValue := trendMap[bucket]
		trendValue.Requests += int64(row.Count)
		trendValue.Tokens += int64(row.TokenUsed)
		trendValue.Quota += int64(row.Quota)
		trendMap[bucket] = trendValue

		modelValue := modelMap[row.ModelName]
		modelValue.Requests += int64(row.Count)
		modelValue.Tokens += int64(row.TokenUsed)
		modelValue.Quota += int64(row.Quota)
		modelMap[row.ModelName] = modelValue
	}
	metrics.EstimatedCost = quotaToCurrency(metrics.TotalQuota)

	trend := make([]enterpriseOverviewTrendPoint, 0, len(trendMap))
	for timestamp, value := range trendMap {
		trend = append(trend, enterpriseOverviewTrendPoint{
			Timestamp: timestamp,
			Requests:  value.Requests,
			Tokens:    value.Tokens,
			Quota:     value.Quota,
		})
	}
	sort.Slice(trend, func(i, j int) bool { return trend[i].Timestamp < trend[j].Timestamp })

	userRows, _ := model.GetQuotaDataGroupByUser(startTimestamp, endTimestamp)
	userMap := make(map[string]enterpriseOverviewAggregate)
	for _, row := range userRows {
		if row == nil {
			continue
		}
		value := userMap[row.Username]
		value.Requests += int64(row.Count)
		value.Tokens += int64(row.TokenUsed)
		value.Quota += int64(row.Quota)
		userMap[row.Username] = value
	}

	_ = model.DB.Model(&model.User{}).Count(&metrics.TotalUsers).Error
	_ = model.DB.Model(&model.User{}).Where("status = ?", common.UserStatusEnabled).Count(&metrics.ActiveUsers).Error
	_ = model.DB.Model(&model.Channel{}).Count(&metrics.TotalChannels).Error
	_ = model.DB.Model(&model.Channel{}).Where("status = ?", common.ChannelStatusEnabled).Count(&metrics.HealthyChannels).Error
	_ = model.DB.Model(&model.Channel{}).Where("status = ? AND balance > 0 AND balance < ?", common.ChannelStatusEnabled, 10).Count(&metrics.LowBalanceChannels).Error
	_ = model.DB.Model(&model.Token{}).Where("status = ?", common.TokenStatusEnabled).Count(&metrics.ActiveAPIKeys).Error
	_ = model.DB.Model(&model.Supplier{}).Count(&metrics.TotalSuppliers).Error
	_ = model.DB.Model(&model.Supplier{}).Where("status = ?", common.ChannelStatusEnabled).Count(&metrics.HealthySuppliers).Error
	_ = model.DB.Model(&model.SupplyRoutingPolicy{}).Where("status = ?", model.SupplyRoutingPolicyStatusActive).Count(&metrics.ActivePolicies).Error
	_ = model.DB.Model(&model.OperatingInsight{}).Where("status = ?", model.OperatingInsightStatusDraft).Count(&metrics.OpenInsights).Error

	var pricingApprovals int64
	var supplierApprovals int64
	var decisionApprovals int64
	_ = model.DB.Model(&model.PricingRecommendation{}).Where("status = ?", model.PricingRecommendationStatusDraft).Count(&pricingApprovals).Error
	_ = model.DB.Model(&model.SupplierEvaluation{}).Where("status = ?", model.SupplierEvaluationStatusDraft).Count(&supplierApprovals).Error
	_ = model.DB.Model(&model.SupplyDecision{}).Where("status = ?", model.SupplyDecisionStatusDraft).Count(&decisionApprovals).Error
	metrics.PendingApprovals = pricingApprovals + supplierApprovals + decisionApprovals

	if model.LOG_DB != nil {
		var consumeCount int64
		var errorCount int64
		_ = model.LOG_DB.Model(&model.Log{}).
			Where("created_at >= ? AND created_at <= ? AND type = ?", startTimestamp, endTimestamp, model.LogTypeConsume).
			Count(&consumeCount).Error
		_ = model.LOG_DB.Model(&model.Log{}).
			Where("created_at >= ? AND created_at <= ? AND type = ?", startTimestamp, endTimestamp, model.LogTypeError).
			Count(&errorCount).Error
		if consumeCount+errorCount > 0 {
			metrics.SuccessRate = float64(consumeCount) / float64(consumeCount+errorCount)
		}
		_ = model.LOG_DB.Model(&model.Log{}).
			Select("COALESCE(AVG(use_time), 0)").
			Where("created_at >= ? AND created_at <= ? AND type = ?", startTimestamp, endTimestamp, model.LogTypeConsume).
			Scan(&metrics.AverageLatencyMs).Error
	}
	if metrics.SuccessRate == 0 && metrics.TotalRequests > 0 {
		metrics.SuccessRate = 1
	}

	var marginSummary struct {
		SellQuota        int64
		CostQuota        int64
		GrossProfitQuota int64
	}
	_ = model.DB.Model(&model.PricingRecommendation{}).
		Select("COALESCE(SUM(total_sell_quota), 0) AS sell_quota, COALESCE(SUM(total_cost_quota), 0) AS cost_quota, COALESCE(SUM(gross_profit_quota), 0) AS gross_profit_quota").
		Where("period_end >= ? AND period_start <= ?", startTimestamp, endTimestamp).
		Scan(&marginSummary).Error
	metrics.GrossProfitQuota = marginSummary.GrossProfitQuota
	metrics.EstimatedGrossProfit = quotaToCurrency(marginSummary.GrossProfitQuota)
	if marginSummary.SellQuota > 0 {
		metrics.GrossMarginRate = float64(marginSummary.GrossProfitQuota) / float64(marginSummary.SellQuota)
	}

	channelModels := make([]model.Channel, 0, 6)
	_ = model.DB.Model(&model.Channel{}).
		Select("id, name, status, response_time, balance, used_quota, models, `group`").
		Order("used_quota DESC").Limit(6).Find(&channelModels).Error
	channels := make([]enterpriseOverviewChannelItem, 0, len(channelModels))
	for _, channel := range channelModels {
		channels = append(channels, enterpriseOverviewChannelItem{
			ID:           channel.Id,
			Name:         channel.Name,
			Status:       channel.Status,
			ResponseTime: channel.ResponseTime,
			Balance:      channel.Balance,
			UsedQuota:    channel.UsedQuota,
			Models:       channel.Models,
			Group:        channel.Group,
		})
	}

	insightModels := make([]model.OperatingInsight, 0, 6)
	_ = model.DB.Model(&model.OperatingInsight{}).
		Where("status = ?", model.OperatingInsightStatusDraft).
		Order("CASE severity WHEN 'action' THEN 1 WHEN 'watch' THEN 2 ELSE 3 END, generated_at DESC").
		Limit(6).Find(&insightModels).Error
	insights := make([]enterpriseOverviewInsight, 0, len(insightModels))
	for _, insight := range insightModels {
		insights = append(insights, enterpriseOverviewInsight{
			ID:                insight.Id,
			Title:             insight.Title,
			Summary:           insight.Summary,
			Severity:          insight.Severity,
			Category:          insight.Category,
			ModelName:         insight.ModelName,
			RecommendedAction: insight.RecommendedAction,
			SLAMetRate:        insight.SlaMetRate,
			GeneratedAt:       insight.GeneratedAt,
		})
	}

	data := enterpriseOverviewData{
		GeneratedAt: time.Now().Unix(),
		Range: enterpriseOverviewRange{
			StartTimestamp: startTimestamp,
			EndTimestamp:   endTimestamp,
		},
		Metrics:   metrics,
		Trend:     trend,
		TopModels: rankingItemsFromMap(modelMap, metrics.TotalRequests, 6),
		TopUsers:  rankingItemsFromMap(userMap, metrics.TotalRequests, 6),
		Channels:  channels,
		Insights:  insights,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
}
