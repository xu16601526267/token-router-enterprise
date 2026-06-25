package service

import (
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

type enterpriseChannelLogAggregate struct {
	ChannelId    int
	Requests     int64
	SuccessCount int64
	Latency      float64
}

func enterpriseChannelAggregates(startTimestamp int64, endTimestamp int64) (map[int]enterpriseChannelLogAggregate, error) {
	result := make(map[int]enterpriseChannelLogAggregate)
	if model.LOG_DB == nil {
		return result, nil
	}
	var rows []enterpriseChannelLogAggregate
	err := model.LOG_DB.Model(&model.Log{}).
		Select("channel AS channel_id, COUNT(*) AS requests, SUM(CASE WHEN type = ? THEN 1 ELSE 0 END) AS success_count, COALESCE(AVG(CASE WHEN type = ? THEN use_time ELSE NULL END), 0) AS latency", model.LogTypeConsume, model.LogTypeConsume).
		Where("created_at >= ? AND created_at <= ? AND type IN ?", startTimestamp, endTimestamp, []int{model.LogTypeConsume, model.LogTypeError}).
		Group("channel").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ChannelId] = row
	}
	return result, nil
}

func enterpriseChannelItem(channel model.Channel, supplier model.Supplier, logs enterpriseChannelLogAggregate) dto.EnterpriseChannelItem {
	successRate := 0.0
	if logs.Requests > 0 {
		successRate = float64(logs.SuccessCount) / float64(logs.Requests)
	} else if channel.Status == common.ChannelStatusEnabled {
		successRate = 1
	}
	priority := int64(0)
	if channel.Priority != nil {
		priority = *channel.Priority
	}
	weight := uint(0)
	if channel.Weight != nil {
		weight = *channel.Weight
	}
	tag := ""
	if channel.Tag != nil {
		tag = *channel.Tag
	}
	remark := ""
	if channel.Remark != nil {
		remark = *channel.Remark
	}
	return dto.EnterpriseChannelItem{
		Id: channel.Id, Name: channel.Name, Type: channel.Type, Status: channel.Status,
		SupplierId: channel.SupplierId, SupplierName: supplier.Name,
		SupplierType: supplier.Type, SupplierStatus: supplier.Status,
		Models: channel.Models, Group: channel.Group, Tag: tag, Remark: remark,
		Balance: channel.Balance, UsedQuota: channel.UsedQuota,
		ResponseTimeMs: channel.ResponseTime, AverageLatencyMs: logs.Latency * 1000,
		Requests: logs.Requests, SuccessRate: successRate, Priority: priority,
		Weight: weight, LastCheckedAt: channel.TestTime,
		BalanceUpdatedTime: channel.BalanceUpdatedTime,
	}
}

func GetEnterpriseChannelCenter(startTimestamp int64, endTimestamp int64) (*dto.EnterpriseChannelCenterData, error) {
	var channels []model.Channel
	if err := model.DB.Order("status DESC, priority DESC, used_quota DESC, id DESC").Find(&channels).Error; err != nil {
		return nil, err
	}
	var suppliers []model.Supplier
	if err := model.DB.Find(&suppliers).Error; err != nil {
		return nil, err
	}
	supplierMap := make(map[int]model.Supplier, len(suppliers))
	for _, supplier := range suppliers {
		supplierMap[supplier.Id] = supplier
	}
	aggregates, err := enterpriseChannelAggregates(startTimestamp, endTimestamp)
	if err != nil {
		return nil, err
	}
	items := make([]dto.EnterpriseChannelItem, 0, len(channels))
	summary := dto.EnterpriseChannelSummary{}
	var successWeight int64
	var latencyWeight int64
	for _, channel := range channels {
		item := enterpriseChannelItem(channel, supplierMap[channel.SupplierId], aggregates[channel.Id])
		items = append(items, item)
		if channel.Status == common.ChannelStatusEnabled {
			summary.EnabledChannels++
		}
		summary.TotalBalance += channel.Balance
		if channel.Status == common.ChannelStatusEnabled && channel.Balance > 0 && channel.Balance < 10 {
			summary.LowBalanceAlerts++
		}
		if item.Requests > 0 {
			summary.AverageSuccessRate += item.SuccessRate * float64(item.Requests)
			successWeight += item.Requests
		}
		if item.AverageLatencyMs > 0 {
			summary.AverageLatencyMs += item.AverageLatencyMs * float64(item.Requests)
			latencyWeight += item.Requests
		}
	}
	for _, supplier := range suppliers {
		if supplier.Status == common.ChannelStatusEnabled {
			summary.HealthySuppliers++
		}
	}
	if successWeight > 0 {
		summary.AverageSuccessRate /= float64(successWeight)
	}
	if latencyWeight > 0 {
		summary.AverageLatencyMs /= float64(latencyWeight)
	}
	return &dto.EnterpriseChannelCenterData{
		GeneratedAt: common.GetTimestamp(), Summary: summary, Items: items,
	}, nil
}

func GetEnterpriseChannelDetail(channelId int, startTimestamp int64, endTimestamp int64) (*dto.EnterpriseChannelDetail, error) {
	var channel model.Channel
	if err := model.DB.First(&channel, channelId).Error; err != nil {
		return nil, err
	}
	var supplier model.Supplier
	if channel.SupplierId > 0 {
		if err := model.DB.First(&supplier, channel.SupplierId).Error; err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	aggregates, err := enterpriseChannelAggregates(startTimestamp, endTimestamp)
	if err != nil {
		return nil, err
	}
	item := enterpriseChannelItem(channel, supplier, aggregates[channel.Id])

	models := make([]string, 0)
	for _, modelName := range strings.Split(channel.Models, ",") {
		modelName = strings.TrimSpace(modelName)
		if modelName != "" {
			models = append(models, modelName)
		}
	}
	sort.Strings(models)

	var supplierDetail *dto.EnterpriseSupplierDetail
	if supplier.Id > 0 {
		detail := dto.EnterpriseSupplierDetail{
			Id: supplier.Id, Name: supplier.Name, Type: supplier.Type,
			Status: supplier.Status, Notes: supplier.Notes, UpdatedTime: supplier.UpdatedTime,
		}
		_ = model.DB.Model(&model.Channel{}).Where("supplier_id = ?", supplier.Id).Count(&detail.ChannelCount).Error
		_ = model.DB.Model(&model.Channel{}).Select("COALESCE(SUM(balance), 0)").Where("supplier_id = ?", supplier.Id).Scan(&detail.TotalBalance).Error
		var scorecard model.SupplierScorecard
		if err := model.DB.Where("supplier_id = ?", supplier.Id).Order("period_end DESC, id DESC").First(&scorecard).Error; err == nil {
			detail.SuccessRate = scorecard.SuccessRate
			detail.LatencyMs = scorecard.AvgLatencyMs
			detail.Score = scorecard.Score
			detail.Grade = scorecard.Grade
		}
		if preference, err := model.GetActiveSupplierRoutePreferenceBySupplierID(supplier.Id); err == nil {
			detail.RouteWeight = preference.WeightPercent
		}
		supplierDetail = &detail
	}

	incidents := make([]dto.EnterpriseChannelIncident, 0, 8)
	if model.LOG_DB != nil {
		var logs []model.Log
		if err := model.LOG_DB.Where("channel = ? AND created_at >= ? AND created_at <= ? AND type = ?", channelId, startTimestamp, endTimestamp, model.LogTypeError).
			Order("id DESC").Limit(8).Find(&logs).Error; err != nil {
			return nil, err
		}
		for _, log := range logs {
			title := strings.TrimSpace(log.Content)
			if title == "" {
				title = "上游请求异常"
			}
			incidents = append(incidents, dto.EnterpriseChannelIncident{
				Id: log.Id, Title: title, Severity: "warning", Status: "open", CreatedAt: log.CreatedAt,
			})
		}
	}
	return &dto.EnterpriseChannelDetail{
		Channel: item, Supplier: supplierDetail, SupportedModels: models, Incidents: incidents,
	}, nil
}
