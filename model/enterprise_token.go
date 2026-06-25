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
package model

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

type EnterpriseTokenFilters struct {
	Keyword               string
	Status                int
	UserId                int
	Group                 string
	ManagerId             int
	ManagerRole           int
	RestrictByManagerRole bool
}

type EnterpriseTokenRecord struct {
	Token
	Username    string `gorm:"column:username"`
	DisplayName string `gorm:"column:display_name"`
	Email       string `gorm:"column:email"`
	UserGroup   string `gorm:"column:user_group"`
	UserRole    int    `gorm:"column:user_role"`
}

func applyEnterpriseTokenFilters(query *gorm.DB, filters EnterpriseTokenFilters) *gorm.DB {
	now := time.Now().Unix()
	if filters.Keyword != "" {
		keyword := strings.TrimSpace(filters.Keyword)
		keyKeyword := strings.TrimPrefix(keyword, "sk-")
		like := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(keyword) + "%"
		keyLike := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(keyKeyword) + "%"
		query = query.Where(`tokens.name LIKE ? ESCAPE '!' OR tokens.key LIKE ? ESCAPE '!' OR users.username LIKE ? ESCAPE '!' OR users.display_name LIKE ? ESCAPE '!' OR users.email LIKE ? ESCAPE '!'`, like, keyLike, like, like, like)
	}
	if filters.UserId > 0 {
		query = query.Where("tokens.user_id = ?", filters.UserId)
	}
	if filters.Group != "" {
		query = query.Where("tokens.group = ?", strings.TrimSpace(filters.Group))
	}
	switch filters.Status {
	case common.TokenStatusEnabled:
		query = query.Where("tokens.status = ? AND (tokens.expired_time = -1 OR tokens.expired_time > ?) AND (tokens.unlimited_quota = ? OR tokens.remain_quota > 0)", common.TokenStatusEnabled, now, true)
	case common.TokenStatusDisabled:
		query = query.Where("tokens.status = ?", common.TokenStatusDisabled)
	case common.TokenStatusExpired:
		query = query.Where("tokens.status = ? OR (tokens.status = ? AND tokens.expired_time != -1 AND tokens.expired_time <= ?)", common.TokenStatusExpired, common.TokenStatusEnabled, now)
	case common.TokenStatusExhausted:
		query = query.Where("tokens.status = ? OR (tokens.status = ? AND tokens.unlimited_quota = ? AND tokens.remain_quota <= 0)", common.TokenStatusExhausted, common.TokenStatusEnabled, false)
	}
	if filters.RestrictByManagerRole && filters.ManagerRole < common.RoleRootUser {
		query = query.Where("users.role < ? OR users.id = ?", filters.ManagerRole, filters.ManagerId)
	}
	return query
}

func SearchEnterpriseTokens(filters EnterpriseTokenFilters, offset int, limit int) ([]EnterpriseTokenRecord, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	base := DB.Model(&Token{}).Joins("LEFT JOIN users ON users.id = tokens.user_id")
	base = applyEnterpriseTokenFilters(base, filters)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []EnterpriseTokenRecord
	err := base.Select("tokens.*, users.username AS username, users.display_name AS display_name, users.email AS email, users.`group` AS user_group, users.role AS user_role").
		Order("tokens.id DESC").Offset(offset).Limit(limit).Scan(&records).Error
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func GetEnterpriseTokenByID(id int) (*EnterpriseTokenRecord, error) {
	var record EnterpriseTokenRecord
	err := DB.Model(&Token{}).
		Joins("LEFT JOIN users ON users.id = tokens.user_id").
		Select("tokens.*, users.username AS username, users.display_name AS display_name, users.email AS email, users.`group` AS user_group, users.role AS user_role").
		Where("tokens.id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func RotateEnterpriseTokenKey(token *Token, key string) error {
	oldKey := token.Key
	now := common.GetTimestamp()
	if err := DB.Model(token).Select("key", "accessed_time").Updates(map[string]interface{}{
		"key": key, "accessed_time": now,
	}).Error; err != nil {
		return err
	}
	token.Key = key
	token.AccessedTime = now
	if common.RedisEnabled {
		gopool.Go(func() {
			if oldKey != "" {
				_ = cacheDeleteToken(oldKey)
			}
			_ = cacheSetToken(*token)
		})
	}
	return nil
}
