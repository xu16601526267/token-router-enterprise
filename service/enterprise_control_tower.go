package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

type enterpriseChannelTraffic struct {
	ChannelId    int
	Requests     int64
	SuccessCount int64
	Latency      float64
}

func GetEnterpriseControlTower(startTimestamp int64, endTimestamp int64) (*dto.EnterpriseControlTowerData, error) {
	policies, _, err := model.SearchSupplyRoutingPolicies(model.SupplyRoutingPolicyFilters{
		StartTime: startTimestamp,
		EndTime:   endTimestamp,
	}, 0, 100)
	if err != nil {
		return nil, err
	}

	var channels []model.Channel
	if err := model.DB.Order("status DESC, used_quota DESC, id DESC").Limit(24).Find(&channels).Error; err != nil {
		return nil, err
	}
	var suppliers []model.Supplier
	if err := model.DB.Order("status DESC, id DESC").Find(&suppliers).Error; err != nil {
		return nil, err
	}
	channelMap := make(map[int]model.Channel, len(channels))
	for _, channel := range channels {
		channelMap[channel.Id] = channel
	}
	supplierMap := make(map[int]model.Supplier, len(suppliers))
	for _, supplier := range suppliers {
		supplierMap[supplier.Id] = supplier
	}

	trafficMap := make(map[int]enterpriseChannelTraffic)
	if model.LOG_DB != nil {
		var rows []enterpriseChannelTraffic
		err := model.LOG_DB.Model(&model.Log{}).
			Select("channel AS channel_id, COUNT(*) AS requests, SUM(CASE WHEN type = ? THEN 1 ELSE 0 END) AS success_count, COALESCE(AVG(CASE WHEN type = ? THEN use_time ELSE NULL END), 0) AS latency", model.LogTypeConsume, model.LogTypeConsume).
			Where("created_at >= ? AND created_at <= ? AND type IN ?", startTimestamp, endTimestamp, []int{model.LogTypeConsume, model.LogTypeError}).
			Group("channel").Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			trafficMap[row.ChannelId] = row
		}
	}

	providerHealth := make([]dto.EnterpriseProviderHealth, 0, len(channels))
	metrics := dto.EnterpriseControlTowerMetrics{}
	for _, channel := range channels {
		traffic := trafficMap[channel.Id]
		successRate := 0.0
		if traffic.Requests > 0 {
			successRate = float64(traffic.SuccessCount) / float64(traffic.Requests)
		}
		if successRate == 0 && channel.Status == common.ChannelStatusEnabled {
			successRate = 1
		}
		supplier := supplierMap[channel.SupplierId]
		providerHealth = append(providerHealth, dto.EnterpriseProviderHealth{
			ChannelId: channel.Id, ChannelName: channel.Name, SupplierId: channel.SupplierId,
			SupplierName: supplier.Name, Status: channel.Status, Requests: traffic.Requests,
			SuccessRate: successRate, AverageLatencyMs: traffic.Latency * 1000,
			ResponseTimeMs: channel.ResponseTime, Balance: channel.Balance,
			Models: channel.Models, Region: channel.Group,
		})
		metrics.Requests += traffic.Requests
		if channel.Status == common.ChannelStatusEnabled {
			metrics.RealtimeSuccessRate += successRate * float64(traffic.Requests)
			metrics.AverageLatencyMs += traffic.Latency * 1000 * float64(traffic.SuccessCount)
		}
	}
	if metrics.Requests > 0 {
		metrics.RealtimeSuccessRate /= float64(metrics.Requests)
	}
	var successfulRequests int64
	for _, row := range trafficMap {
		successfulRequests += row.SuccessCount
	}
	if successfulRequests > 0 {
		metrics.AverageLatencyMs /= float64(successfulRequests)
	}

	policyItems := make([]dto.EnterpriseRoutingPolicyItem, 0, len(policies))
	for _, policy := range policies {
		if policy == nil {
			continue
		}
		channel := channelMap[policy.ChannelId]
		supplier := supplierMap[policy.SupplierId]
		name := strings.TrimSpace(policy.DecisionKey)
		if name == "" {
			name = fmt.Sprintf("路由策略 #%d", policy.Id)
		}
		policyItems = append(policyItems, dto.EnterpriseRoutingPolicyItem{
			Id: policy.Id, Name: name, SliceKey: policy.SliceKey,
			ModelName: policy.ModelName, SlaTier: policy.SlaTier, Track: policy.Track,
			ActionType: policy.ActionType, Status: policy.Status,
			SupplierId: policy.SupplierId, SupplierName: supplier.Name,
			ChannelId: policy.ChannelId, ChannelName: channel.Name,
			Priority: policy.Priority, TrafficPercent: policy.TrafficPercent,
			EffectiveFrom: policy.EffectiveFrom, EffectiveTo: policy.EffectiveTo,
			UpdatedAt: policy.UpdatedAt, Reason: policy.Reason,
		})
		if policy.Status == model.SupplyRoutingPolicyStatusActive {
			metrics.ActivePolicies++
		}
		if policy.ActivatedAt >= startTimestamp && policy.ActivatedAt <= endTimestamp {
			metrics.AutomaticSwitches++
		}
	}

	quotaRows, err := model.GetAllQuotaDates(startTimestamp, endTimestamp, "")
	if err != nil {
		return nil, err
	}
	type dailyValue struct{ requests, tokens int64 }
	daily := make(map[int64]dailyValue)
	for _, row := range quotaRows {
		if row == nil {
			continue
		}
		day := time.Unix(row.CreatedAt, 0).UTC()
		bucket := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC).Unix()
		value := daily[bucket]
		value.requests += int64(row.Count)
		value.tokens += int64(row.TokenUsed)
		daily[bucket] = value
		metrics.Tokens += int64(row.TokenUsed)
	}
	trend := make([]dto.EnterpriseControlTowerTrendPoint, 0, len(daily))
	for timestamp, value := range daily {
		trend = append(trend, dto.EnterpriseControlTowerTrendPoint{
			Timestamp: timestamp, Requests: value.requests,
			SuccessRate: metrics.RealtimeSuccessRate, LatencyMs: metrics.AverageLatencyMs,
		})
	}
	sort.Slice(trend, func(i, j int) bool { return trend[i].Timestamp < trend[j].Timestamp })

	recentChanges := make([]dto.EnterpriseControlTowerEvent, 0, 6)
	for _, policy := range policies {
		if policy == nil || len(recentChanges) >= 6 {
			continue
		}
		recentChanges = append(recentChanges, dto.EnterpriseControlTowerEvent{
			Id: policy.Id, Title: "路由策略已更新", Detail: fmt.Sprintf("%s · %s · 流量 %d%%", policy.ModelName, policy.Track, policy.TrafficPercent),
			Category: "routing_policy", Severity: "info", Status: policy.Status, CreatedAt: policy.UpdatedAt,
		})
	}

	var actionPlans []model.SupplyActionPlan
	if err := model.DB.Where("status IN ?", []string{model.SupplyActionPlanStatusPlanned, model.SupplyActionPlanStatusInProgress}).Order("opportunity_rank_score DESC, updated_at DESC").Limit(6).Find(&actionPlans).Error; err != nil {
		return nil, err
	}
	pendingActions := make([]dto.EnterpriseControlTowerEvent, 0, len(actionPlans))
	for _, plan := range actionPlans {
		pendingActions = append(pendingActions, dto.EnterpriseControlTowerEvent{
			Id: plan.Id, Title: "待执行动作：" + plan.ModelName,
			Detail: plan.Reason, Category: plan.ActionType, Severity: plan.OpportunityPriority,
			Status: plan.Status, CreatedAt: plan.UpdatedAt,
		})
	}

	var insights []model.OperatingInsight
	if err := model.DB.Where("status = ?", model.OperatingInsightStatusDraft).
		Order("generated_at DESC").Limit(6).Find(&insights).Error; err != nil {
		return nil, err
	}
	risks := make([]dto.EnterpriseControlTowerEvent, 0, len(insights))
	for _, insight := range insights {
		risks = append(risks, dto.EnterpriseControlTowerEvent{
			Id: insight.Id, Title: insight.Title, Detail: insight.Summary,
			Category: insight.Category, Severity: insight.Severity,
			Status: insight.Status, CreatedAt: insight.GeneratedAt,
		})
	}

	var pricingApprovals int64
	var evaluationApprovals int64
	var decisionApprovals int64
	_ = model.DB.Model(&model.PricingRecommendation{}).Where("status = ?", model.PricingRecommendationStatusDraft).Count(&pricingApprovals).Error
	_ = model.DB.Model(&model.SupplierEvaluation{}).Where("status = ?", model.SupplierEvaluationStatusDraft).Count(&evaluationApprovals).Error
	_ = model.DB.Model(&model.SupplyDecision{}).Where("status = ?", model.SupplyDecisionStatusDraft).Count(&decisionApprovals).Error
	metrics.PendingApprovals = pricingApprovals + evaluationApprovals + decisionApprovals

	return &dto.EnterpriseControlTowerData{
		GeneratedAt: common.GetTimestamp(),
		Range:       dto.EnterpriseControlTowerRange{StartTimestamp: startTimestamp, EndTimestamp: endTimestamp},
		Metrics:     metrics, Trend: trend, Policies: policyItems, ProviderHealth: providerHealth,
		RecentChanges: recentChanges, PendingActions: pendingActions, Risks: risks,
	}, nil
}
