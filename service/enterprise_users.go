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
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

type enterpriseUserTokenCount struct {
	UserId int
	Count  int64
}

type enterpriseNamedCount struct {
	Name  string
	Count int64
}

func enterpriseRoleLabel(role int) string {
	switch role {
	case common.RoleRootUser:
		return "超级管理员"
	case common.RoleAdminUser:
		return "管理员"
	default:
		return "普通用户"
	}
}

func GetEnterpriseUsers(limit int) (*dto.EnterpriseUsersData, error) {
	if limit <= 0 || limit > 1000 {
		limit = 250
	}
	var users []model.User
	if err := model.DB.Omit("password").Order("role DESC, status DESC, last_login_at DESC, id DESC").Limit(limit).Find(&users).Error; err != nil {
		return nil, err
	}

	var tokenRows []enterpriseUserTokenCount
	if err := model.DB.Model(&model.Token{}).
		Select("user_id, COUNT(*) AS count").Group("user_id").Scan(&tokenRows).Error; err != nil {
		return nil, err
	}
	tokenCount := make(map[int]int64, len(tokenRows))
	for _, row := range tokenRows {
		tokenCount[row.UserId] = row.Count
	}

	items := make([]dto.EnterpriseUserItem, 0, len(users))
	for _, user := range users {
		items = append(items, dto.EnterpriseUserItem{
			Id: user.Id, Username: user.Username, DisplayName: user.DisplayName,
			Email: user.Email, Group: user.Group, Role: user.Role, Status: user.Status,
			APIKeyCount: tokenCount[user.Id], Quota: user.Quota, UsedQuota: user.UsedQuota,
			RequestCount: user.RequestCount, LastLoginAt: user.LastLoginAt,
		})
	}

	summary := dto.EnterpriseUserSummary{}
	if err := model.DB.Model(&model.User{}).Count(&summary.TotalUsers).Error; err != nil {
		return nil, err
	}
	if err := model.DB.Model(&model.User{}).Where("status = ?", common.UserStatusEnabled).Count(&summary.ActiveUsers).Error; err != nil {
		return nil, err
	}
	if err := model.DB.Model(&model.User{}).Where("role >= ?", common.RoleAdminUser).Count(&summary.AdminUsers).Error; err != nil {
		return nil, err
	}
	if err := model.DB.Model(&model.User{}).Where("status = ?", common.UserStatusDisabled).Count(&summary.DisabledUsers).Error; err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	if err := model.DB.Model(&model.Token{}).
		Where("status = ? AND (expired_time = -1 OR expired_time > ?) AND (unlimited_quota = ? OR remain_quota > 0)", common.TokenStatusEnabled, now, true).
		Count(&summary.ActiveAPIKeys).Error; err != nil {
		return nil, err
	}
	if err := model.DB.Model(&model.User{}).Distinct("`group`").Count(&summary.Groups).Error; err != nil {
		// PostgreSQL does not accept backticks, retry with GORM's plain column.
		if retryErr := model.DB.Model(&model.User{}).Distinct("group").Count(&summary.Groups).Error; retryErr != nil {
			return nil, retryErr
		}
	}

	var roleRows []struct {
		Role  int
		Count int64
	}
	if err := model.DB.Model(&model.User{}).Select("role, COUNT(*) AS count").Group("role").Order("role DESC").Scan(&roleRows).Error; err != nil {
		return nil, err
	}
	roleCounts := make([]dto.EnterpriseCountItem, 0, len(roleRows))
	for _, row := range roleRows {
		roleCounts = append(roleCounts, dto.EnterpriseCountItem{Name: enterpriseRoleLabel(row.Role), Count: row.Count})
	}

	// Let GORM quote the physical group column by selecting into a stable alias.
	groupColumn := "`group`"
	if common.UsingPostgreSQL {
		groupColumn = `"group"`
	}
	var groupRows []enterpriseNamedCount
	if err := model.DB.Model(&model.User{}).
		Select(groupColumn + " AS name, COUNT(*) AS count").Group(groupColumn).Order("count DESC").Scan(&groupRows).Error; err != nil {
		return nil, err
	}
	groupCounts := make([]dto.EnterpriseCountItem, 0, len(groupRows))
	for _, row := range groupRows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			name = "默认分组"
		}
		groupCounts = append(groupCounts, dto.EnterpriseCountItem{Name: name, Count: row.Count})
	}
	if len(groupCounts) == 0 && summary.TotalUsers > 0 {
		groupCounts = append(groupCounts, dto.EnterpriseCountItem{Name: "默认分组", Count: summary.TotalUsers})
	}
	if summary.Groups == 0 {
		summary.Groups = int64(len(groupCounts))
	}
	_ = strconv.IntSize // keep architecture-neutral integer conversions explicit above

	return &dto.EnterpriseUsersData{
		GeneratedAt: common.GetTimestamp(), Summary: summary, Users: items,
		RoleCounts: roleCounts, GroupCounts: groupCounts,
	}, nil
}
