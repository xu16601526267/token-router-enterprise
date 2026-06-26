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
package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func enterprisePositiveInt(c *gin.Context, key string, fallback int, max int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil || value <= 0 {
		return fallback
	}
	if max > 0 && value > max {
		return max
	}
	return value
}

func enterprisePositiveInt64(c *gin.Context, key string, fallback int64) int64 {
	value, err := strconv.ParseInt(c.Query(key), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func enterprisePathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "资源 ID 无效")
		return 0, false
	}
	return id, true
}

func enterpriseCanManageUser(managerId int, managerRole int, userId int, targetRole int) bool {
	return managerRole == common.RoleRootUser || managerId == userId || managerRole > targetRole
}

func enterpriseCheckTokenPermission(c *gin.Context, tokenId int) (*model.EnterpriseTokenRecord, bool) {
	record, err := model.GetEnterpriseTokenByID(tokenId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "企业密钥不存在")
		} else {
			common.ApiError(c, err)
		}
		return nil, false
	}
	if !enterpriseCanManageUser(c.GetInt("id"), c.GetInt("role"), record.Token.UserId, record.UserRole) {
		common.ApiErrorMsg(c, "没有权限管理该用户的企业密钥")
		return nil, false
	}
	return record, true
}

func enterpriseCSVFilename(prefix string, startTimestamp int64, endTimestamp int64) string {
	format := func(timestamp int64) string {
		if timestamp <= 0 {
			return "unknown"
		}
		return time.Unix(timestamp, 0).Format("20060102")
	}
	return fmt.Sprintf("%s-%s-%s.csv", prefix, format(startTimestamp), format(endTimestamp))
}

func writeEnterpriseCSV(c *gin.Context, filename string, body []byte) {
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", body)
}

func enterpriseUsageFiltersFromQuery(c *gin.Context) service.EnterpriseUsageFilters {
	channelId, _ := strconv.Atoi(c.Query("channel_id"))
	return service.EnterpriseUsageFilters{
		Keyword:         strings.TrimSpace(c.Query("keyword")),
		ModelName:       strings.TrimSpace(c.Query("model_name")),
		Username:        strings.TrimSpace(c.Query("username")),
		Group:           strings.TrimSpace(c.Query("group")),
		Status:          strings.TrimSpace(c.Query("status")),
		ChannelId:       channelId,
		Page:            enterprisePositiveInt(c, "page", 1, 1000000),
		PageSize:        enterprisePositiveInt(c, "page_size", 50, 500),
		SortBy:          strings.TrimSpace(c.Query("sort_by")),
		SortOrder:       strings.TrimSpace(c.Query("sort_order")),
		TimeGranularity: strings.TrimSpace(c.Query("time_granularity")),
	}
}

func GetEnterpriseControlTower(c *gin.Context) {
	startTimestamp, endTimestamp := parseEnterpriseOverviewRange(c)
	data, err := service.GetEnterpriseControlTower(startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func GetEnterpriseChannels(c *gin.Context) {
	startTimestamp, endTimestamp := parseEnterpriseOverviewRange(c)
	status, _ := strconv.Atoi(c.Query("status"))
	supplierId, _ := strconv.Atoi(c.Query("supplier_id"))
	channelType, _ := strconv.Atoi(c.Query("type"))
	data, err := service.GetEnterpriseChannelCenterWithFilters(startTimestamp, endTimestamp, service.EnterpriseChannelFilters{
		Keyword:    strings.TrimSpace(c.Query("keyword")),
		Status:     status,
		SupplierId: supplierId,
		Type:       channelType,
		Group:      strings.TrimSpace(c.Query("group")),
		Page:       enterprisePositiveInt(c, "page", 1, 1000000),
		PageSize:   enterprisePositiveInt(c, "page_size", 50, 500),
		SortBy:     strings.TrimSpace(c.Query("sort_by")),
		SortOrder:  strings.TrimSpace(c.Query("sort_order")),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func ExportEnterpriseChannels(c *gin.Context) {
	startTimestamp, endTimestamp := parseEnterpriseOverviewRange(c)
	status, _ := strconv.Atoi(c.Query("status"))
	supplierId, _ := strconv.Atoi(c.Query("supplier_id"))
	channelType, _ := strconv.Atoi(c.Query("type"))
	body, err := service.BuildEnterpriseChannelsCSV(startTimestamp, endTimestamp, service.EnterpriseChannelFilters{
		Keyword:    strings.TrimSpace(c.Query("keyword")),
		Status:     status,
		SupplierId: supplierId,
		Type:       channelType,
		Group:      strings.TrimSpace(c.Query("group")),
		SortBy:     strings.TrimSpace(c.Query("sort_by")),
		SortOrder:  strings.TrimSpace(c.Query("sort_order")),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	writeEnterpriseCSV(c, enterpriseCSVFilename("enterprise-channels", startTimestamp, endTimestamp), body)
}

func GetEnterpriseChannelDetail(c *gin.Context) {
	id, ok := enterprisePathID(c)
	if !ok {
		return
	}
	startTimestamp, endTimestamp := parseEnterpriseOverviewRange(c)
	data, err := service.GetEnterpriseChannelDetail(id, startTimestamp, endTimestamp)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, "渠道不存在")
		} else {
			common.ApiError(c, err)
		}
		return
	}
	common.ApiSuccess(c, data)
}

func GetEnterpriseAPIKeys(c *gin.Context) {
	page := enterprisePositiveInt(c, "page", 1, 1000000)
	pageSize := enterprisePositiveInt(c, "page_size", 20, 200)
	status, _ := strconv.Atoi(c.Query("status"))
	userId, _ := strconv.Atoi(c.Query("user_id"))
	filters := model.EnterpriseTokenFilters{
		Keyword: strings.TrimSpace(c.Query("keyword")), Status: status,
		UserId: userId, Group: strings.TrimSpace(c.Query("group")),
		ModelLimitMode: strings.TrimSpace(c.Query("model_limit_mode")),
		CreatedStart:   enterprisePositiveInt64(c, "created_start", 0),
		CreatedEnd:     enterprisePositiveInt64(c, "created_end", 0),
		ManagerId:      c.GetInt("id"), ManagerRole: c.GetInt("role"),
		RestrictByManagerRole: true,
	}
	items, total, summary, err := service.ListEnterpriseAPIKeys(filters, (page-1)*pageSize, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, dto.EnterpriseAPIKeyPage{
		Items: items, Total: total, Page: page, PageSize: pageSize, Summary: summary,
	})
}

func ExportEnterpriseAPIKeys(c *gin.Context) {
	status, _ := strconv.Atoi(c.Query("status"))
	userId, _ := strconv.Atoi(c.Query("user_id"))
	filters := model.EnterpriseTokenFilters{
		Keyword: strings.TrimSpace(c.Query("keyword")), Status: status,
		UserId: userId, Group: strings.TrimSpace(c.Query("group")),
		ModelLimitMode: strings.TrimSpace(c.Query("model_limit_mode")),
		CreatedStart:   enterprisePositiveInt64(c, "created_start", 0),
		CreatedEnd:     enterprisePositiveInt64(c, "created_end", 0),
		ManagerId:      c.GetInt("id"), ManagerRole: c.GetInt("role"),
		RestrictByManagerRole: true,
	}
	body, err := service.BuildEnterpriseAPIKeysCSV(filters)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	writeEnterpriseCSV(c, fmt.Sprintf("enterprise-api-keys-%d.csv", time.Now().Unix()), body)
}

func GetEnterpriseAPIKeyUsers(c *gin.Context) {
	items, err := service.ListEnterpriseAPIKeyUsers(c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, items)
}

func CreateEnterpriseAPIKey(c *gin.Context) {
	var input dto.EnterpriseAPIKeyMutationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiErrorMsg(c, "企业密钥参数格式不正确")
		return
	}
	user, err := model.GetUserById(input.UserId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !enterpriseCanManageUser(c.GetInt("id"), c.GetInt("role"), user.Id, user.Role) {
		common.ApiErrorMsg(c, "没有权限为该用户创建企业密钥")
		return
	}
	secret, err := service.CreateEnterpriseAPIKey(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, secret)
}

func UpdateEnterpriseAPIKey(c *gin.Context) {
	id, ok := enterprisePathID(c)
	if !ok {
		return
	}
	if _, ok = enterpriseCheckTokenPermission(c, id); !ok {
		return
	}
	var input dto.EnterpriseAPIKeyMutationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiErrorMsg(c, "企业密钥参数格式不正确")
		return
	}
	item, err := service.UpdateEnterpriseAPIKey(id, input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func RotateEnterpriseAPIKey(c *gin.Context) {
	id, ok := enterprisePathID(c)
	if !ok {
		return
	}
	if _, ok = enterpriseCheckTokenPermission(c, id); !ok {
		return
	}
	secret, err := service.RotateEnterpriseAPIKey(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, secret)
}

func DeleteEnterpriseAPIKey(c *gin.Context) {
	id, ok := enterprisePathID(c)
	if !ok {
		return
	}
	if _, ok = enterpriseCheckTokenPermission(c, id); !ok {
		return
	}
	item, err := service.DeleteEnterpriseAPIKey(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func GetEnterpriseUsageAnalytics(c *gin.Context) {
	startTimestamp, endTimestamp := parseEnterpriseOverviewRange(c)
	data, err := service.GetEnterpriseUsageAnalyticsWithFilters(startTimestamp, endTimestamp, enterpriseUsageFiltersFromQuery(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func ExportEnterpriseUsageAnalytics(c *gin.Context) {
	startTimestamp, endTimestamp := parseEnterpriseOverviewRange(c)
	body, err := service.BuildEnterpriseUsageAnalyticsCSVWithFilters(startTimestamp, endTimestamp, enterpriseUsageFiltersFromQuery(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	writeEnterpriseCSV(c, enterpriseCSVFilename("enterprise-usage", startTimestamp, endTimestamp), body)
}

func GetEnterpriseUsers(c *gin.Context) {
	limit := enterprisePositiveInt(c, "limit", 250, 1000)
	data, err := service.GetEnterpriseUsers(limit, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func GetEnterpriseBilling(c *gin.Context) {
	startTimestamp, endTimestamp := parseEnterpriseOverviewRange(c)
	data, err := service.GetEnterpriseBilling(startTimestamp, endTimestamp, string(parseEnterpriseOverviewGranularity(c)))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func ExportEnterpriseBilling(c *gin.Context) {
	startTimestamp, endTimestamp := parseEnterpriseOverviewRange(c)
	body, err := service.BuildEnterpriseBillingCSV(startTimestamp, endTimestamp)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	writeEnterpriseCSV(c, enterpriseCSVFilename("enterprise-billing", startTimestamp, endTimestamp), body)
}

func GenerateEnterpriseBillingSettlement(c *gin.Context) {
	var input model.SettlementStatementGenerateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiErrorMsg(c, "结算单参数格式不正确")
		return
	}
	item, err := service.GenerateEnterpriseSettlementStatement(input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}
