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
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

const enterpriseExportLimit = 10000

func newEnterpriseCSV() (*bytes.Buffer, *csv.Writer) {
	buffer := &bytes.Buffer{}
	buffer.WriteString("\xEF\xBB\xBF")
	return buffer, csv.NewWriter(buffer)
}

func enterpriseWriteCSVSection(writer *csv.Writer, title string, header []string, rows [][]string) error {
	if title != "" {
		if err := writer.Write([]string{title}); err != nil {
			return err
		}
	}
	if len(header) > 0 {
		if err := writer.Write(header); err != nil {
			return err
		}
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return writer.Write([]string{})
}

func enterpriseFlushCSV(buffer *bytes.Buffer, writer *csv.Writer) ([]byte, error) {
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func enterpriseFormatUnixDate(timestamp int64) string {
	if timestamp <= 0 {
		return ""
	}
	return time.Unix(timestamp, 0).Format("2006-01-02")
}

func enterpriseFormatUnixDateTime(timestamp int64) string {
	if timestamp <= 0 {
		return ""
	}
	return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
}

func enterpriseFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}

func enterpriseFormatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
}

func enterpriseFormatBool(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func enterpriseAPIKeyStatusLabel(status int) string {
	switch status {
	case common.TokenStatusEnabled:
		return "启用"
	case common.TokenStatusDisabled:
		return "禁用"
	case common.TokenStatusExpired:
		return "过期"
	case common.TokenStatusExhausted:
		return "额度耗尽"
	default:
		return fmt.Sprintf("未知(%d)", status)
	}
}

func enterpriseUsageLogItems(startTimestamp int64, endTimestamp int64, limit int) ([]dto.EnterpriseUsageLogItem, error) {
	return enterpriseUsageLogItemsWithFilters(startTimestamp, endTimestamp, EnterpriseUsageFilters{}, limit)
}

func enterpriseUsageLogItemsWithFilters(startTimestamp int64, endTimestamp int64, filters EnterpriseUsageFilters, limit int) ([]dto.EnterpriseUsageLogItem, error) {
	items := []dto.EnterpriseUsageLogItem{}
	if model.LOG_DB == nil {
		return items, nil
	}
	if limit <= 0 || limit > enterpriseExportLimit {
		limit = enterpriseExportLimit
	}
	var logs []model.Log
	if err := enterpriseUsageLogQuery(model.LOG_DB, startTimestamp, endTimestamp, filters, []int{model.LogTypeConsume, model.LogTypeError}).
		Order("id DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	channelIds := make([]int, 0, len(logs))
	for _, log := range logs {
		if log.ChannelId > 0 {
			channelIds = append(channelIds, log.ChannelId)
		}
	}
	channelNames := map[int]string{}
	if len(channelIds) > 0 {
		var channels []model.Channel
		if err := model.DB.Select("id", "name").Where("id IN ?", channelIds).Find(&channels).Error; err == nil {
			for _, channel := range channels {
				channelNames[channel.Id] = channel.Name
			}
		}
	}
	items = make([]dto.EnterpriseUsageLogItem, 0, len(logs))
	for _, log := range logs {
		status := "success"
		if log.Type == model.LogTypeError {
			status = "error"
		}
		items = append(items, dto.EnterpriseUsageLogItem{
			Id: log.Id, RequestId: log.RequestId, CreatedAt: log.CreatedAt,
			Username: log.Username, Group: log.Group, TokenName: log.TokenName,
			ModelName: log.ModelName, RequestType: enterpriseUsageRequestType(log.ModelName), PromptTokens: log.PromptTokens,
			CompletionTokens: log.CompletionTokens, Quota: log.Quota,
			ChannelId: log.ChannelId, ChannelName: channelNames[log.ChannelId],
			UseTimeMs: log.UseTime * 1000, Status: status, Ip: log.Ip,
		})
	}
	return items, nil
}

func enterpriseBreakdownCSVRows(items []dto.EnterpriseUsageBreakdownItem) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Name,
			enterpriseFormatInt(item.Quota),
			enterpriseFormatFloat(item.Cost),
			enterpriseFormatFloat(item.Share),
		})
	}
	return rows
}

func BuildEnterpriseUsageAnalyticsCSV(startTimestamp int64, endTimestamp int64) ([]byte, error) {
	return BuildEnterpriseUsageAnalyticsCSVWithFilters(startTimestamp, endTimestamp, EnterpriseUsageFilters{})
}

func BuildEnterpriseUsageAnalyticsCSVWithFilters(startTimestamp int64, endTimestamp int64, filters EnterpriseUsageFilters) ([]byte, error) {
	data, err := GetEnterpriseUsageAnalyticsWithFilters(startTimestamp, endTimestamp, filters)
	if err != nil {
		return nil, err
	}
	logs, err := enterpriseUsageLogItemsWithFilters(startTimestamp, endTimestamp, filters, enterpriseExportLimit)
	if err != nil {
		return nil, err
	}
	buffer, writer := newEnterpriseCSV()
	metrics := data.Metrics
	if err := enterpriseWriteCSVSection(writer, "用量指标", []string{"指标", "值"}, [][]string{
		{"统计开始", enterpriseFormatUnixDateTime(data.Range.StartTimestamp)},
		{"统计结束", enterpriseFormatUnixDateTime(data.Range.EndTimestamp)},
		{"总请求数", enterpriseFormatInt(metrics.TotalRequests)},
		{"错误请求数", enterpriseFormatInt(metrics.ErrorRequests)},
		{"错误率", enterpriseFormatFloat(metrics.ErrorRate)},
		{"输入 Tokens", enterpriseFormatInt(metrics.PromptTokens)},
		{"输出 Tokens", enterpriseFormatInt(metrics.CompletionTokens)},
		{"总 Tokens", enterpriseFormatInt(metrics.TotalTokens)},
		{"总额度", enterpriseFormatInt(metrics.TotalQuota)},
		{"估算成本 USD", enterpriseFormatFloat(metrics.EstimatedCost)},
		{"平均延迟 ms", enterpriseFormatFloat(metrics.AverageLatencyMs)},
		{"缓存命中率", enterpriseFormatFloat(metrics.CacheHitRate)},
	}); err != nil {
		return nil, err
	}

	trendRows := make([][]string, 0, len(data.Trend))
	for _, item := range data.Trend {
		trendRows = append(trendRows, []string{
			enterpriseFormatUnixDate(item.Timestamp),
			enterpriseFormatInt(item.Requests),
			enterpriseFormatInt(item.Errors),
			enterpriseFormatInt(item.PromptTokens),
			enterpriseFormatInt(item.CompletionTokens),
			enterpriseFormatInt(item.Quota),
			enterpriseFormatFloat(item.AverageLatencyMs),
			enterpriseFormatFloat(item.CacheHitRate),
		})
	}
	if err := enterpriseWriteCSVSection(writer, "每日趋势", []string{"日期", "请求数", "错误数", "输入 Tokens", "输出 Tokens", "额度", "平均延迟 ms", "缓存命中率"}, trendRows); err != nil {
		return nil, err
	}
	breakdownHeader := []string{"名称", "额度", "成本 USD", "占比"}
	for _, section := range []struct {
		title string
		items []dto.EnterpriseUsageBreakdownItem
	}{
		{"按模型成本", data.ByModel},
		{"按用户成本", data.ByUser},
		{"按渠道成本", data.ByChannel},
		{"按分组成本", data.ByGroup},
	} {
		if err := enterpriseWriteCSVSection(writer, section.title, breakdownHeader, enterpriseBreakdownCSVRows(section.items)); err != nil {
			return nil, err
		}
	}

	logRows := make([][]string, 0, len(logs))
	for _, item := range logs {
		logRows = append(logRows, []string{
			enterpriseFormatInt(int64(item.Id)),
			item.RequestId,
			enterpriseFormatUnixDateTime(item.CreatedAt),
			item.Username,
			item.Group,
			item.TokenName,
			item.ModelName,
			item.RequestType,
			enterpriseFormatInt(int64(item.PromptTokens)),
			enterpriseFormatInt(int64(item.CompletionTokens)),
			enterpriseFormatInt(int64(item.Quota)),
			enterpriseFormatInt(int64(item.ChannelId)),
			item.ChannelName,
			enterpriseFormatInt(int64(item.UseTimeMs)),
			item.Status,
			item.Ip,
		})
	}
	if err := enterpriseWriteCSVSection(writer, "调用日志（最多 10000 条）", []string{"日志ID", "请求编号", "时间", "用户", "分组", "密钥名称", "模型", "请求类型", "输入 Tokens", "输出 Tokens", "额度", "渠道ID", "渠道名称", "延迟 ms", "状态", "IP"}, logRows); err != nil {
		return nil, err
	}
	return enterpriseFlushCSV(buffer, writer)
}

func BuildEnterpriseAPIKeysCSV(filters model.EnterpriseTokenFilters) ([]byte, error) {
	buffer, writer := newEnterpriseCSV()
	rows := [][]string{}
	offset := 0
	pageSize := 200
	for offset < enterpriseExportLimit {
		records, total, err := model.SearchEnterpriseTokens(filters, offset, pageSize)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			item := enterpriseAPIKeyItem(record)
			rows = append(rows, []string{
				enterpriseFormatInt(int64(item.Id)),
				enterpriseFormatInt(int64(item.UserId)),
				item.Username,
				item.DisplayName,
				item.Email,
				item.UserGroup,
				item.Name,
				item.MaskedKey,
				enterpriseAPIKeyStatusLabel(item.Status),
				enterpriseAPIKeyStatusLabel(item.EffectiveStatus),
				item.Group,
				enterpriseFormatBool(item.UnlimitedQuota),
				enterpriseFormatInt(int64(item.RemainQuota)),
				enterpriseFormatInt(int64(item.UsedQuota)),
				enterpriseFormatBool(item.ModelLimitsEnabled),
				item.ModelLimits,
				valueOrEmpty(item.AllowIps),
				enterpriseFormatBool(item.CrossGroupRetry),
				enterpriseFormatUnixDateTime(item.CreatedTime),
				enterpriseFormatUnixDateTime(item.AccessedTime),
				enterpriseFormatUnixDateTime(item.ExpiredTime),
			})
		}
		offset += len(records)
		if len(records) == 0 || int64(offset) >= total {
			break
		}
	}
	if err := enterpriseWriteCSVSection(writer, "企业 API Key 清单", []string{"密钥ID", "用户ID", "用户名", "显示名", "邮箱", "用户分组", "密钥名称", "脱敏Key", "配置状态", "有效状态", "路由分组", "无限额度", "剩余额度", "已用额度", "启用模型白名单", "模型白名单", "IP 白名单", "跨组重试", "创建时间", "最近使用", "过期时间"}, rows); err != nil {
		return nil, err
	}
	return enterpriseFlushCSV(buffer, writer)
}

func BuildEnterpriseChannelsCSV(startTimestamp int64, endTimestamp int64, filters EnterpriseChannelFilters) ([]byte, error) {
	filters.Page = 1
	filters.PageSize = enterpriseExportLimit
	data, err := GetEnterpriseChannelCenterWithFilters(startTimestamp, endTimestamp, filters)
	if err != nil {
		return nil, err
	}

	buffer, writer := newEnterpriseCSV()
	summary := data.Summary
	if err := enterpriseWriteCSVSection(writer, "渠道与供应商指标", []string{"指标", "值"}, [][]string{
		{"统计开始", enterpriseFormatUnixDateTime(startTimestamp)},
		{"统计结束", enterpriseFormatUnixDateTime(endTimestamp)},
		{"启用渠道数", enterpriseFormatInt(summary.EnabledChannels)},
		{"健康供应商数", enterpriseFormatInt(summary.HealthySuppliers)},
		{"平均成功率", enterpriseFormatFloat(summary.AverageSuccessRate)},
		{"平均延迟 ms", enterpriseFormatFloat(summary.AverageLatencyMs)},
		{"总余额 USD", enterpriseFormatFloat(summary.TotalBalance)},
		{"低余额告警", enterpriseFormatInt(summary.LowBalanceAlerts)},
		{"匹配渠道数", enterpriseFormatInt(data.Total)},
	}); err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(data.Items))
	for _, item := range data.Items {
		rows = append(rows, []string{
			enterpriseFormatInt(int64(item.Id)),
			item.Name,
			enterpriseFormatInt(int64(item.Type)),
			enterpriseAPIKeyStatusLabel(item.Status),
			enterpriseFormatInt(int64(item.SupplierId)),
			item.SupplierName,
			item.SupplierType,
			enterpriseAPIKeyStatusLabel(item.SupplierStatus),
			item.Models,
			item.Group,
			item.Tag,
			item.Remark,
			enterpriseFormatFloat(item.Balance),
			enterpriseFormatInt(int64(item.UsedQuota)),
			enterpriseFormatFloat(item.SuccessRate),
			enterpriseFormatFloat(item.AverageLatencyMs),
			enterpriseFormatInt(item.Requests),
			enterpriseFormatInt(item.Priority),
			enterpriseFormatInt(int64(item.Weight)),
			enterpriseFormatUnixDateTime(item.LastCheckedAt),
			enterpriseFormatUnixDateTime(item.BalanceUpdatedTime),
		})
	}
	if err := enterpriseWriteCSVSection(writer, "渠道明细（最多 10000 条）", []string{"渠道ID", "渠道名称", "渠道类型", "渠道状态", "供应商ID", "供应商名称", "供应商类型", "供应商状态", "模型", "路由分组", "标签", "备注", "余额 USD", "已用额度", "成功率", "平均延迟 ms", "请求数", "优先级", "权重", "最后检查", "余额更新时间"}, rows); err != nil {
		return nil, err
	}
	return enterpriseFlushCSV(buffer, writer)
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func enterpriseBillingSettlementItems(startTimestamp int64, endTimestamp int64) ([]dto.EnterpriseSettlementItem, error) {
	statements, _, err := model.SearchSettlementStatements(model.SettlementStatementFilters{
		StartTime: startTimestamp, EndTime: endTimestamp,
	}, 0, enterpriseExportLimit)
	if err != nil {
		return nil, err
	}
	userIds := make([]int, 0, len(statements))
	supplierIds := make([]int, 0, len(statements))
	for _, statement := range statements {
		if statement == nil {
			continue
		}
		if statement.UserId > 0 {
			userIds = append(userIds, statement.UserId)
		}
		if statement.SupplierId > 0 {
			supplierIds = append(supplierIds, statement.SupplierId)
		}
	}
	userNames := map[int]string{}
	if len(userIds) > 0 {
		var users []model.User
		if err := model.DB.Select("id", "username", "display_name").Where("id IN ?", userIds).Find(&users).Error; err == nil {
			for _, user := range users {
				name := user.DisplayName
				if name == "" {
					name = user.Username
				}
				userNames[user.Id] = name
			}
		}
	}
	supplierNames := map[int]string{}
	if len(supplierIds) > 0 {
		var suppliers []model.Supplier
		if err := model.DB.Select("id", "name").Where("id IN ?", supplierIds).Find(&suppliers).Error; err == nil {
			for _, supplier := range suppliers {
				supplierNames[supplier.Id] = supplier.Name
			}
		}
	}
	items := make([]dto.EnterpriseSettlementItem, 0, len(statements))
	for _, statement := range statements {
		if statement == nil {
			continue
		}
		subjectId := statement.UserId
		subjectName := userNames[statement.UserId]
		if statement.SubjectType == model.SettlementSubjectSupplier {
			subjectId = statement.SupplierId
			subjectName = supplierNames[statement.SupplierId]
		}
		if subjectName == "" {
			subjectName = fmt.Sprintf("%s #%d", statement.SubjectType, subjectId)
		}
		items = append(items, dto.EnterpriseSettlementItem{
			Id: statement.Id, SubjectType: statement.SubjectType, SubjectId: subjectId,
			SubjectName: subjectName, PeriodStart: statement.PeriodStart, PeriodEnd: statement.PeriodEnd,
			TotalSellQuota: statement.TotalSellQuota, TotalCostQuota: statement.TotalCostQuota,
			GrossProfitQuota: statement.GrossProfitQuota, TotalRequests: statement.TotalRequests,
			Status: statement.Status,
		})
	}
	return items, nil
}

func enterpriseBillingTopUpItems(startTimestamp int64, endTimestamp int64) ([]dto.EnterpriseTopUpItem, error) {
	var topUps []enterpriseTopUpRow
	query := model.DB.Table("top_ups").
		Select("top_ups.id, top_ups.user_id, users.username, top_ups.money, top_ups.payment_method, top_ups.payment_provider, top_ups.status, top_ups.create_time").
		Joins("LEFT JOIN users ON users.id = top_ups.user_id").
		Where("top_ups.create_time >= ? AND top_ups.create_time <= ?", startTimestamp, endTimestamp).
		Order("top_ups.id DESC").Limit(enterpriseExportLimit)
	if err := query.Scan(&topUps).Error; err != nil {
		return nil, err
	}
	items := make([]dto.EnterpriseTopUpItem, 0, len(topUps))
	for _, topUp := range topUps {
		items = append(items, dto.EnterpriseTopUpItem{
			Id: topUp.Id, UserId: topUp.UserId, Username: topUp.Username,
			Money: topUp.Money, PaymentMethod: topUp.PaymentMethod,
			PaymentProvider: topUp.PaymentProvider, Status: topUp.Status, CreateTime: topUp.CreateTime,
		})
	}
	return items, nil
}

func BuildEnterpriseBillingCSV(startTimestamp int64, endTimestamp int64) ([]byte, error) {
	data, err := GetEnterpriseBilling(startTimestamp, endTimestamp, "day")
	if err != nil {
		return nil, err
	}
	settlements, err := enterpriseBillingSettlementItems(startTimestamp, endTimestamp)
	if err != nil {
		return nil, err
	}
	topUps, err := enterpriseBillingTopUpItems(startTimestamp, endTimestamp)
	if err != nil {
		return nil, err
	}

	buffer, writer := newEnterpriseCSV()
	metrics := data.Metrics
	if err := enterpriseWriteCSVSection(writer, "账单指标", []string{"指标", "值"}, [][]string{
		{"统计开始", enterpriseFormatUnixDateTime(data.Range.StartTimestamp)},
		{"统计结束", enterpriseFormatUnixDateTime(data.Range.EndTimestamp)},
		{"活跃订阅", enterpriseFormatInt(metrics.ActiveSubscriptions)},
		{"企业可用额度", enterpriseFormatInt(metrics.TotalBalanceQuota)},
		{"企业已用额度", enterpriseFormatInt(metrics.TotalUsedQuota)},
		{"本期应收额度", enterpriseFormatInt(metrics.PeriodSellQuota)},
		{"本期应付额度", enterpriseFormatInt(metrics.PeriodCostQuota)},
		{"本期毛利额度", enterpriseFormatInt(metrics.PeriodGrossProfitQuota)},
		{"本期毛利率", enterpriseFormatFloat(metrics.GrossMarginRate)},
		{"成功充值金额", enterpriseFormatFloat(metrics.SuccessfulTopUpAmount)},
		{"待处理充值金额", enterpriseFormatFloat(metrics.PendingTopUpAmount)},
		{"待确认结算单", enterpriseFormatInt(metrics.DraftSettlements)},
	}); err != nil {
		return nil, err
	}
	trendRows := make([][]string, 0, len(data.Trend))
	for _, item := range data.Trend {
		trendRows = append(trendRows, []string{
			enterpriseFormatUnixDate(item.Timestamp),
			enterpriseFormatInt(item.SellQuota),
			enterpriseFormatInt(item.CostQuota),
			enterpriseFormatInt(item.GrossProfitQuota),
		})
	}
	if err := enterpriseWriteCSVSection(writer, "收支趋势", []string{"日期", "应收额度", "应付额度", "毛利额度"}, trendRows); err != nil {
		return nil, err
	}
	settlementRows := make([][]string, 0, len(settlements))
	for _, item := range settlements {
		settlementRows = append(settlementRows, []string{
			enterpriseFormatInt(int64(item.Id)),
			enterpriseFormatUnixDate(item.PeriodStart),
			enterpriseFormatUnixDate(item.PeriodEnd),
			item.SubjectType,
			enterpriseFormatInt(int64(item.SubjectId)),
			item.SubjectName,
			enterpriseFormatInt(item.TotalSellQuota),
			enterpriseFormatInt(item.TotalCostQuota),
			enterpriseFormatInt(item.GrossProfitQuota),
			enterpriseFormatInt(item.TotalRequests),
			item.Status,
		})
	}
	if err := enterpriseWriteCSVSection(writer, "结算单（最多 10000 条）", []string{"结算单ID", "周期开始", "周期结束", "对象类型", "对象ID", "对象名称", "应收额度", "应付额度", "毛利额度", "请求数", "状态"}, settlementRows); err != nil {
		return nil, err
	}
	topUpRows := make([][]string, 0, len(topUps))
	for _, item := range topUps {
		topUpRows = append(topUpRows, []string{
			enterpriseFormatInt(int64(item.Id)),
			enterpriseFormatInt(int64(item.UserId)),
			item.Username,
			enterpriseFormatFloat(item.Money),
			item.PaymentMethod,
			item.PaymentProvider,
			item.Status,
			enterpriseFormatUnixDateTime(item.CreateTime),
		})
	}
	if err := enterpriseWriteCSVSection(writer, "充值流水（最多 10000 条）", []string{"充值ID", "用户ID", "用户名", "金额", "支付方式", "支付平台", "状态", "创建时间"}, topUpRows); err != nil {
		return nil, err
	}
	return enterpriseFlushCSV(buffer, writer)
}
