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

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

type enterpriseBillingUserAggregate struct {
	BalanceQuota int64
	UsedQuota    int64
}

type enterpriseBillingTopUpAggregate struct {
	Successful float64
	Pending    float64
}

type enterpriseTopUpRow struct {
	Id              int
	UserId          int
	Username        string
	Money           float64
	PaymentMethod   string
	PaymentProvider string
	Status          string
	CreateTime      int64
}

func GetEnterpriseBilling(startTimestamp int64, endTimestamp int64) (*dto.EnterpriseBillingData, error) {
	data := &dto.EnterpriseBillingData{
		GeneratedAt:  common.GetTimestamp(),
		Range:        dto.EnterpriseBillingRange{StartTimestamp: startTimestamp, EndTimestamp: endTimestamp},
		Trend:        []dto.EnterpriseBillingTrendPoint{},
		Settlements:  []dto.EnterpriseSettlementItem{},
		RecentTopups: []dto.EnterpriseTopUpItem{},
	}

	var userAggregate enterpriseBillingUserAggregate
	if err := model.DB.Model(&model.User{}).
		Select("COALESCE(SUM(quota), 0) AS balance_quota, COALESCE(SUM(used_quota), 0) AS used_quota").
		Scan(&userAggregate).Error; err != nil {
		return nil, err
	}
	data.Metrics.TotalBalanceQuota = userAggregate.BalanceQuota
	data.Metrics.TotalUsedQuota = userAggregate.UsedQuota

	var usageAggregate struct {
		SellQuota int64
		CostQuota int64
	}
	if err := model.DB.Model(&model.UsageLedger{}).
		Select("COALESCE(SUM(sell_quota), 0) AS sell_quota, COALESCE(SUM(cost_quota), 0) AS cost_quota").
		Where("created_at >= ? AND created_at <= ? AND status = ?", startTimestamp, endTimestamp, "success").
		Scan(&usageAggregate).Error; err != nil {
		return nil, err
	}
	data.Metrics.PeriodSellQuota = usageAggregate.SellQuota
	data.Metrics.PeriodCostQuota = usageAggregate.CostQuota
	data.Metrics.PeriodGrossProfitQuota = usageAggregate.SellQuota - usageAggregate.CostQuota
	if usageAggregate.SellQuota > 0 {
		data.Metrics.GrossMarginRate = float64(data.Metrics.PeriodGrossProfitQuota) / float64(usageAggregate.SellQuota)
	}

	trendRows, err := model.SearchMarginSummary(model.MarginSummaryFilters{
		GroupBy: "day", StartTime: startTimestamp, EndTime: endTimestamp,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range trendRows {
		data.Trend = append(data.Trend, dto.EnterpriseBillingTrendPoint{
			Timestamp: row.BucketStart, SellQuota: row.TotalSellQuota,
			CostQuota: row.TotalCostQuota, GrossProfitQuota: row.GrossProfitQuota,
		})
	}

	var topUpAggregate enterpriseBillingTopUpAggregate
	if err := model.DB.Model(&model.TopUp{}).
		Select("COALESCE(SUM(CASE WHEN status = ? THEN money ELSE 0 END), 0) AS successful, COALESCE(SUM(CASE WHEN status = ? THEN money ELSE 0 END), 0) AS pending", common.TopUpStatusSuccess, common.TopUpStatusPending).
		Where("create_time >= ? AND create_time <= ?", startTimestamp, endTimestamp).
		Scan(&topUpAggregate).Error; err != nil {
		return nil, err
	}
	data.Metrics.SuccessfulTopUpAmount = topUpAggregate.Successful
	data.Metrics.PendingTopUpAmount = topUpAggregate.Pending

	if err := model.DB.Model(&model.UserSubscription{}).
		Where("status = ? AND end_time >= ?", "active", common.GetTimestamp()).
		Count(&data.Metrics.ActiveSubscriptions).Error; err != nil {
		return nil, err
	}
	if err := model.DB.Model(&model.SettlementStatement{}).
		Where("status = ?", model.SettlementStatusDraft).
		Count(&data.Metrics.DraftSettlements).Error; err != nil {
		return nil, err
	}

	statements, _, err := model.SearchSettlementStatements(model.SettlementStatementFilters{
		StartTime: startTimestamp, EndTime: endTimestamp,
	}, 0, 20)
	if err != nil {
		return nil, err
	}
	userIds := make([]int, 0)
	supplierIds := make([]int, 0)
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
	userNames := make(map[int]string)
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
	supplierNames := make(map[int]string)
	if len(supplierIds) > 0 {
		var suppliers []model.Supplier
		if err := model.DB.Select("id", "name").Where("id IN ?", supplierIds).Find(&suppliers).Error; err == nil {
			for _, supplier := range suppliers {
				supplierNames[supplier.Id] = supplier.Name
			}
		}
	}
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
		data.Settlements = append(data.Settlements, dto.EnterpriseSettlementItem{
			Id: statement.Id, SubjectType: statement.SubjectType, SubjectId: subjectId,
			SubjectName: subjectName, PeriodStart: statement.PeriodStart, PeriodEnd: statement.PeriodEnd,
			TotalSellQuota: statement.TotalSellQuota, TotalCostQuota: statement.TotalCostQuota,
			GrossProfitQuota: statement.GrossProfitQuota, TotalRequests: statement.TotalRequests,
			Status: statement.Status,
		})
	}

	var topUps []enterpriseTopUpRow
	if err := model.DB.Table("top_ups").
		Select("top_ups.id, top_ups.user_id, users.username, top_ups.money, top_ups.payment_method, top_ups.payment_provider, top_ups.status, top_ups.create_time").
		Joins("LEFT JOIN users ON users.id = top_ups.user_id").
		Order("top_ups.id DESC").Limit(12).Scan(&topUps).Error; err != nil {
		// GORM's default pluralization for TopUp can differ on older databases.
		if fallbackErr := model.DB.Model(&model.TopUp{}).
			Select("top_ups.id, top_ups.user_id, users.username, top_ups.money, top_ups.payment_method, top_ups.payment_provider, top_ups.status, top_ups.create_time").
			Joins("LEFT JOIN users ON users.id = top_ups.user_id").
			Order("top_ups.id DESC").Limit(12).Scan(&topUps).Error; fallbackErr != nil {
			return nil, fallbackErr
		}
	}
	for _, topUp := range topUps {
		data.RecentTopups = append(data.RecentTopups, dto.EnterpriseTopUpItem{
			Id: topUp.Id, UserId: topUp.UserId, Username: topUp.Username,
			Money: topUp.Money, PaymentMethod: topUp.PaymentMethod,
			PaymentProvider: topUp.PaymentProvider, Status: topUp.Status, CreateTime: topUp.CreateTime,
		})
	}
	return data, nil
}
