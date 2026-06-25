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

func enterpriseQuotaCurrency(quota int64) float64 {
	if common.QuotaPerUnit <= 0 {
		return 0
	}
	return float64(quota) / common.QuotaPerUnit
}

func enterpriseUsageBreakdown(db *gorm.DB, column string, startTimestamp int64, endTimestamp int64, totalQuota int64) ([]dto.EnterpriseUsageBreakdownItem, error) {
	allowed := map[string]bool{"model_name": true, "username": true, "group": true}
	if !allowed[column] {
		return nil, fmt.Errorf("unsupported enterprise usage breakdown: %s", column)
	}
	var rows []enterpriseUsageBreakdownRow
	err := db.Model(&model.Log{}).
		Select(column+" AS name, COALESCE(SUM(quota), 0) AS quota").
		Where("created_at >= ? AND created_at <= ? AND type = ?", startTimestamp, endTimestamp, model.LogTypeConsume).
		Group(column).Order("quota DESC").Limit(12).Scan(&rows).Error
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

func enterpriseUsageChannelBreakdown(db *gorm.DB, startTimestamp int64, endTimestamp int64, totalQuota int64) ([]dto.EnterpriseUsageBreakdownItem, error) {
	var rows []enterpriseUsageChannelRow
	err := db.Model(&model.Log{}).
		Select("channel AS channel_id, COALESCE(SUM(quota), 0) AS quota").
		Where("created_at >= ? AND created_at <= ? AND type = ?", startTimestamp, endTimestamp, model.LogTypeConsume).
		Group("channel").Order("quota DESC").Limit(12).Scan(&rows).Error
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
	data := &dto.EnterpriseUsageAnalyticsData{
		GeneratedAt: common.GetTimestamp(),
		Range:       dto.EnterpriseUsageRange{StartTimestamp: startTimestamp, EndTimestamp: endTimestamp},
		Trend:       []dto.EnterpriseUsageTrendPoint{}, ByModel: []dto.EnterpriseUsageBreakdownItem{},
		ByUser: []dto.EnterpriseUsageBreakdownItem{}, ByChannel: []dto.EnterpriseUsageBreakdownItem{},
		ByGroup: []dto.EnterpriseUsageBreakdownItem{}, RecentLogs: []dto.EnterpriseUsageLogItem{},
	}
	if model.LOG_DB == nil {
		return data, nil
	}

	var consumed enterpriseUsageAggregate
	if err := model.LOG_DB.Model(&model.Log{}).
		Select("COUNT(*) AS requests, COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, COALESCE(SUM(completion_tokens), 0) AS completion_tokens, COALESCE(SUM(quota), 0) AS quota, COALESCE(AVG(use_time), 0) AS latency").
		Where("created_at >= ? AND created_at <= ? AND type = ?", startTimestamp, endTimestamp, model.LogTypeConsume).
		Scan(&consumed).Error; err != nil {
		return nil, err
	}
	var errorCount int64
	if err := model.LOG_DB.Model(&model.Log{}).
		Where("created_at >= ? AND created_at <= ? AND type = ?", startTimestamp, endTimestamp, model.LogTypeError).
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

	type trendValue struct{ requests, errors, quota int64 }
	trendMap := make(map[int64]trendValue)
	var trendRows []struct {
		CreatedAt int64
		Type      int
		Quota     int
	}
	if err := model.LOG_DB.Model(&model.Log{}).Select("created_at", "type", "quota").
		Where("created_at >= ? AND created_at <= ? AND type IN ?", startTimestamp, endTimestamp, []int{model.LogTypeConsume, model.LogTypeError}).
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
			value.quota += int64(row.Quota)
		}
		trendMap[bucket] = value
	}
	for timestamp, value := range trendMap {
		data.Trend = append(data.Trend, dto.EnterpriseUsageTrendPoint{
			Timestamp: timestamp, Requests: value.requests, Errors: value.errors, Quota: value.quota,
		})
	}
	sort.Slice(data.Trend, func(i, j int) bool { return data.Trend[i].Timestamp < data.Trend[j].Timestamp })

	var err error
	if data.ByModel, err = enterpriseUsageBreakdown(model.LOG_DB, "model_name", startTimestamp, endTimestamp, consumed.Quota); err != nil {
		return nil, err
	}
	if data.ByUser, err = enterpriseUsageBreakdown(model.LOG_DB, "username", startTimestamp, endTimestamp, consumed.Quota); err != nil {
		return nil, err
	}
	if data.ByGroup, err = enterpriseUsageBreakdown(model.LOG_DB, "group", startTimestamp, endTimestamp, consumed.Quota); err != nil {
		return nil, err
	}
	if data.ByChannel, err = enterpriseUsageChannelBreakdown(model.LOG_DB, startTimestamp, endTimestamp, consumed.Quota); err != nil {
		return nil, err
	}

	var recent []model.Log
	if err := model.LOG_DB.Where("created_at >= ? AND created_at <= ? AND type IN ?", startTimestamp, endTimestamp, []int{model.LogTypeConsume, model.LogTypeError}).
		Order("id DESC").Limit(50).Find(&recent).Error; err != nil {
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
		status := "成功"
		if log.Type == model.LogTypeError {
			status = "失败"
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
