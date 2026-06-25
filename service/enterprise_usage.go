/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type enterpriseUsageAggregate struct {
	Requests         int64
	PromptTokens     int64
	CompletionTokens int64
	Quota            int64
	Latency          float64
}

type enterpriseUsageBreakdownRow struct {
	Name  string
	Quota int64
}

type enterpriseUsageChannelRow struct {
	ChannelId int
	Quota     int64
}

type EnterpriseUsageFilters struct {
	Keyword   string
	ModelName string
	Username  string
	Group     string
	Status    string
	ChannelId int
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
}

func normalizeEnterprisePage(page int, pageSize int, defaultPageSize int, maxPageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if maxPageSize > 0 && pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func enterpriseLikePattern(value string) string {
	replacer := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")
	return "%" + replacer.Replace(value) + "%"
}

func enterpriseQuotaCurrency(quota int64) float64 {
	if common.QuotaPerUnit <= 0 {
		return 0
	}
	return float64(quota) / common.QuotaPerUnit
}

func enterpriseUsageLogQuery(db *gorm.DB, startTimestamp int64, endTimestamp int64, filters EnterpriseUsageFilters, logTypes []int) *gorm.DB {
	query := db.Model(&model.Log{}).
		Where("created_at >= ? AND created_at <= ?", startTimestamp, endTimestamp)
	if len(logTypes) > 0 {
		query = query.Where("type IN ?", logTypes)
	}
	if filters.Status == "success" {
		query = query.Where("type = ?", model.LogTypeConsume)
	} else if filters.Status == "error" {
		query = query.Where("type = ?", model.LogTypeError)
	}
	if filters.ModelName != "" {
		query = query.Where("model_name = ?", filters.ModelName)
	}
	if filters.Username != "" {
		query = query.Where("username = ?", filters.Username)
	}
	if filters.Group != "" {
		query = query.Where(model.LogGroupColumn()+" = ?", filters.Group)
	}
	if filters.ChannelId > 0 {
		query = query.Where("channel_id = ?", filters.ChannelId)
	}
	if keyword := strings.TrimSpace(filters.Keyword); keyword != "" {
		like := enterpriseLikePattern(keyword)
		query = query.Where(
			"request_id LIKE ? ESCAPE '!' OR username LIKE ? ESCAPE '!' OR token_name LIKE ? ESCAPE '!' OR model_name LIKE ? ESCAPE '!' OR ip LIKE ? ESCAPE '!' OR content LIKE ? ESCAPE '!'",
			like, like, like, like, like, like,
		)
	}
	return query
}

func enterpriseUsageBreakdown(db *gorm.DB, column string, startTimestamp int64, endTimestamp int64, totalQuota int64, filters EnterpriseUsageFilters) ([]dto.EnterpriseUsageBreakdownItem, error) {
	allowed := map[string]string{"model_name": "model_name", "username": "username", "group": model.LogGroupColumn()}
	sqlColumn, ok := allowed[column]
	if !ok {
		return nil, fmt.Errorf("unsupported enterprise usage breakdown: %s", column)
	}
	var rows []enterpriseUsageBreakdownRow
	err := enterpriseUsageLogQuery(db, startTimestamp, endTimestamp, filters, []int{model.LogTypeConsume}).
		Select(sqlColumn + " AS name, COALESCE(SUM(quota), 0) AS quota").
		Clauses(clause.GroupBy{Columns: []clause.Column{{Name: sqlColumn, Raw: true}}}).
		Order("quota DESC").Limit(12).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]dto.EnterpriseUsageBreakdownItem, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			if column == "group" {
				name = "默认分组"
			} else {
				name = "未分类"
			}
		}
		share := 0.0
		if totalQuota > 0 {
			share = float64(row.Quota) / float64(totalQuota)
		}
		items = append(items, dto.EnterpriseUsageBreakdownItem{
			Name: name, Quota: row.Quota, Cost: enterpriseQuotaCurrency(row.Quota), Share: share,
		})
	}
	return items, nil
}

func enterpriseUsageChannelBreakdown(db *gorm.DB, startTimestamp int64, endTimestamp int64, totalQuota int64, filters EnterpriseUsageFilters) ([]dto.EnterpriseUsageBreakdownItem, error) {
	var rows []enterpriseUsageChannelRow
	err := enterpriseUsageLogQuery(db, startTimestamp, endTimestamp, filters, []int{model.LogTypeConsume}).
		Select("channel_id, COALESCE(SUM(quota), 0) AS quota").
		Group("channel_id").Order("quota DESC").Limit(12).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		if row.ChannelId > 0 {
			ids = append(ids, row.ChannelId)
		}
	}
	channelNames := make(map[int]string, len(ids))
	if len(ids) > 0 {
		var channels []model.Channel
		if err := model.DB.Select("id", "name").Where("id IN ?", ids).Find(&channels).Error; err == nil {
			for _, channel := range channels {
				channelNames[channel.Id] = channel.Name
			}
		}
	}
	items := make([]dto.EnterpriseUsageBreakdownItem, 0, len(rows))
	for _, row := range rows {
		name := channelNames[row.ChannelId]
		if name == "" {
			if row.ChannelId > 0 {
				name = "渠道 #" + strconv.Itoa(row.ChannelId)
			} else {
				name = "未记录渠道"
			}
		}
		share := 0.0
		if totalQuota > 0 {
			share = float64(row.Quota) / float64(totalQuota)
		}
		items = append(items, dto.EnterpriseUsageBreakdownItem{
			Name: name, Quota: row.Quota, Cost: enterpriseQuotaCurrency(row.Quota), Share: share,
		})
	}
	return items, nil
}

func GetEnterpriseUsageAnalytics(startTimestamp int64, endTimestamp int64) (*dto.EnterpriseUsageAnalyticsData, error) {
	return GetEnterpriseUsageAnalyticsWithFilters(startTimestamp, endTimestamp, EnterpriseUsageFilters{})
}

func GetEnterpriseUsageAnalyticsWithFilters(startTimestamp int64, endTimestamp int64, filters EnterpriseUsageFilters) (*dto.EnterpriseUsageAnalyticsData, error) {
	page, pageSize := normalizeEnterprisePage(filters.Page, filters.PageSize, 50, 500)
	filters.Page = page
	filters.PageSize = pageSize
	data := &dto.EnterpriseUsageAnalyticsData{
		GeneratedAt: common.GetTimestamp(),
		Range:       dto.EnterpriseUsageRange{StartTimestamp: startTimestamp, EndTimestamp: endTimestamp},
		Trend:       []dto.EnterpriseUsageTrendPoint{}, ByModel: []dto.EnterpriseUsageBreakdownItem{},
		ByUser: []dto.EnterpriseUsageBreakdownItem{}, ByChannel: []dto.EnterpriseUsageBreakdownItem{},
		ByGroup: []dto.EnterpriseUsageBreakdownItem{}, RecentLogs: []dto.EnterpriseUsageLogItem{},
		Page: page, PageSize: pageSize,
	}
	if model.LOG_DB == nil {
		return data, nil
	}

	var consumed enterpriseUsageAggregate
	if err := enterpriseUsageLogQuery(model.LOG_DB, startTimestamp, endTimestamp, filters, []int{model.LogTypeConsume}).
		Select("COUNT(*) AS requests, COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, COALESCE(SUM(completion_tokens), 0) AS completion_tokens, COALESCE(SUM(quota), 0) AS quota, COALESCE(AVG(use_time), 0) AS latency").
		Scan(&consumed).Error; err != nil {
		return nil, err
	}
	var errorCount int64
	if err := enterpriseUsageLogQuery(model.LOG_DB, startTimestamp, endTimestamp, filters, []int{model.LogTypeError}).
		Count(&errorCount).Error; err != nil {
		return nil, err
	}
	data.Metrics.TotalRequests = consumed.Requests + errorCount
	data.Metrics.PromptTokens = consumed.PromptTokens
	data.Metrics.CompletionTokens = consumed.CompletionTokens
	data.Metrics.TotalTokens = consumed.PromptTokens + consumed.CompletionTokens
	data.Metrics.TotalQuota = consumed.Quota
	data.Metrics.EstimatedCost = enterpriseQuotaCurrency(consumed.Quota)
	data.Metrics.ErrorRequests = errorCount
	data.Metrics.AverageLatencyMs = consumed.Latency * 1000
	if data.Metrics.TotalRequests > 0 {
		data.Metrics.ErrorRate = float64(errorCount) / float64(data.Metrics.TotalRequests)
	}

	var cacheAggregate struct {
		Total int64
		Hits  int64
	}
	_ = model.DB.Model(&model.UsageLedger{}).
		Select("COUNT(*) AS total, COALESCE(SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END), 0) AS hits").
		Where("created_at >= ? AND created_at <= ?", startTimestamp, endTimestamp).
		Scan(&cacheAggregate).Error
	if cacheAggregate.Total > 0 {
		data.Metrics.CacheHitRate = float64(cacheAggregate.Hits) / float64(cacheAggregate.Total)
	}

	type trendValue struct {
		requests         int64
		errors           int64
		promptTokens     int64
		completionTokens int64
		quota            int64
		latencyMs        int64
		latencyRequests  int64
	}
	trendMap := make(map[int64]trendValue)
	var trendRows []struct {
		CreatedAt        int64
		Type             int
		PromptTokens     int
		CompletionTokens int
		Quota            int
		UseTime          int
	}
	if err := model.LOG_DB.Model(&model.Log{}).Select("created_at", "type", "prompt_tokens", "completion_tokens", "quota", "use_time").
		Scopes(func(db *gorm.DB) *gorm.DB {
			return enterpriseUsageLogQuery(db, startTimestamp, endTimestamp, filters, []int{model.LogTypeConsume, model.LogTypeError})
		}).
		Find(&trendRows).Error; err != nil {
		return nil, err
	}
	for _, row := range trendRows {
		stamp := time.Unix(row.CreatedAt, 0).UTC()
		bucket := time.Date(stamp.Year(), stamp.Month(), stamp.Day(), 0, 0, 0, 0, time.UTC).Unix()
		value := trendMap[bucket]
		value.requests++
		if row.Type == model.LogTypeError {
			value.errors++
		} else {
			value.promptTokens += int64(row.PromptTokens)
			value.completionTokens += int64(row.CompletionTokens)
			value.quota += int64(row.Quota)
			value.latencyMs += int64(row.UseTime * 1000)
			value.latencyRequests++
		}
		trendMap[bucket] = value
	}
	for timestamp, value := range trendMap {
		averageLatencyMs := 0.0
		if value.latencyRequests > 0 {
			averageLatencyMs = float64(value.latencyMs) / float64(value.latencyRequests)
		}
		data.Trend = append(data.Trend, dto.EnterpriseUsageTrendPoint{
			Timestamp: timestamp, Requests: value.requests, Errors: value.errors,
			PromptTokens: value.promptTokens, CompletionTokens: value.completionTokens,
			Quota: value.quota, AverageLatencyMs: averageLatencyMs,
		})
	}
	sort.Slice(data.Trend, func(i, j int) bool { return data.Trend[i].Timestamp < data.Trend[j].Timestamp })

	var err error
	if data.ByModel, err = enterpriseUsageBreakdown(model.LOG_DB, "model_name", startTimestamp, endTimestamp, consumed.Quota, filters); err != nil {
		return nil, err
	}
	if data.ByUser, err = enterpriseUsageBreakdown(model.LOG_DB, "username", startTimestamp, endTimestamp, consumed.Quota, filters); err != nil {
		return nil, err
	}
	if data.ByGroup, err = enterpriseUsageBreakdown(model.LOG_DB, "group", startTimestamp, endTimestamp, consumed.Quota, filters); err != nil {
		return nil, err
	}
	if data.ByChannel, err = enterpriseUsageChannelBreakdown(model.LOG_DB, startTimestamp, endTimestamp, consumed.Quota, filters); err != nil {
		return nil, err
	}

	var recent []model.Log
	if err := enterpriseUsageLogQuery(model.LOG_DB, startTimestamp, endTimestamp, filters, []int{model.LogTypeConsume, model.LogTypeError}).
		Count(&data.TotalLogs).Error; err != nil {
		return nil, err
	}
	order := "id"
	switch strings.ToLower(strings.TrimSpace(filters.SortBy)) {
	case "created_at":
		order = "created_at"
	case "quota":
		order = "quota"
	case "use_time":
		order = "use_time"
	}
	if strings.ToLower(strings.TrimSpace(filters.SortOrder)) == "asc" {
		order += " ASC"
	} else {
		order += " DESC"
	}
	if err := enterpriseUsageLogQuery(model.LOG_DB, startTimestamp, endTimestamp, filters, []int{model.LogTypeConsume, model.LogTypeError}).
		Order(order).Limit(pageSize).Offset((page - 1) * pageSize).Find(&recent).Error; err != nil {
		return nil, err
	}
	channelIds := make([]int, 0, len(recent))
	for _, log := range recent {
		if log.ChannelId > 0 {
			channelIds = append(channelIds, log.ChannelId)
		}
	}
	channelNames := make(map[int]string)
	if len(channelIds) > 0 {
		var channels []model.Channel
		if err := model.DB.Select("id", "name").Where("id IN ?", channelIds).Find(&channels).Error; err == nil {
			for _, channel := range channels {
				channelNames[channel.Id] = channel.Name
			}
		}
	}
	for _, log := range recent {
		status := "success"
		if log.Type == model.LogTypeError {
			status = "error"
		}
		data.RecentLogs = append(data.RecentLogs, dto.EnterpriseUsageLogItem{
			Id: log.Id, RequestId: log.RequestId, CreatedAt: log.CreatedAt,
			Username: log.Username, Group: log.Group, TokenName: log.TokenName,
			ModelName: log.ModelName, PromptTokens: log.PromptTokens,
			CompletionTokens: log.CompletionTokens, Quota: log.Quota,
			ChannelId: log.ChannelId, ChannelName: channelNames[log.ChannelId],
			UseTimeMs: log.UseTime * 1000, Status: status, Ip: log.Ip,
		})
	}
	return data, nil
}
