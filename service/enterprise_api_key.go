package service

import (
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func normalizeEnterpriseModels(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '\n' })
	seen := make(map[string]struct{}, len(parts))
	models := make([]string, 0, len(parts))
	for _, part := range parts {
		modelName := strings.TrimSpace(part)
		if modelName == "" {
			continue
		}
		if _, ok := seen[modelName]; ok {
			continue
		}
		seen[modelName] = struct{}{}
		models = append(models, modelName)
	}
	sort.Strings(models)
	return strings.Join(models, ",")
}

func normalizeEnterpriseAllowIps(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	parts := strings.FieldsFunc(*value, func(r rune) bool { return r == ',' || r == '\n' || r == ';' })
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if net.ParseIP(item) == nil {
			if _, _, err := net.ParseCIDR(item); err != nil {
				return nil, fmt.Errorf("无效的 IP 或 CIDR：%s", item)
			}
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	normalized := strings.Join(result, "\n")
	return &normalized, nil
}

func validateEnterpriseAPIKeyInput(input *dto.EnterpriseAPIKeyMutationInput, isCreate bool) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Group = strings.TrimSpace(input.Group)
	input.RateLimit = strings.TrimSpace(input.RateLimit)
	input.ModelLimits = normalizeEnterpriseModels(input.ModelLimits)
	if isCreate && input.UserId <= 0 {
		return errors.New("请选择归属用户")
	}
	if input.Name == "" {
		return errors.New("密钥名称不能为空")
	}
	if len(input.Name) > 50 {
		return errors.New("密钥名称不能超过 50 个字符")
	}
	if len(input.RateLimit) > 128 {
		return errors.New("速率限制配置不能超过 128 个字符")
	}
	if input.Status == 0 {
		input.Status = common.TokenStatusEnabled
	}
	if input.Status != common.TokenStatusEnabled && input.Status != common.TokenStatusDisabled {
		return errors.New("企业密钥仅支持启用或禁用状态")
	}
	if input.ExpiredTime == 0 {
		input.ExpiredTime = -1
	}
	if input.ExpiredTime != -1 && input.ExpiredTime <= time.Now().Unix() {
		return errors.New("到期时间必须晚于当前时间")
	}
	if !input.UnlimitedQuota {
		if input.RemainQuota <= 0 {
			return errors.New("有限额度密钥的剩余额度必须大于 0")
		}
		maxQuota := int(1_000_000_000 * common.QuotaPerUnit)
		if input.RemainQuota > maxQuota {
			return fmt.Errorf("剩余额度不能超过 %d", maxQuota)
		}
	}
	if input.ModelLimitsEnabled && input.ModelLimits == "" {
		return errors.New("开启模型白名单后至少需要选择一个模型")
	}
	allowIps, err := normalizeEnterpriseAllowIps(input.AllowIps)
	if err != nil {
		return err
	}
	input.AllowIps = allowIps
	return nil
}

func enterpriseEffectiveTokenStatus(token model.Token) int {
	if token.Status != common.TokenStatusEnabled {
		return token.Status
	}
	now := time.Now().Unix()
	if token.ExpiredTime != -1 && token.ExpiredTime <= now {
		return common.TokenStatusExpired
	}
	if !token.UnlimitedQuota && token.RemainQuota <= 0 {
		return common.TokenStatusExhausted
	}
	return common.TokenStatusEnabled
}

func enterpriseAPIKeyItem(record model.EnterpriseTokenRecord) dto.EnterpriseAPIKeyItem {
	return dto.EnterpriseAPIKeyItem{
		Id:                 record.Token.Id,
		UserId:             record.Token.UserId,
		Name:               record.Token.Name,
		MaskedKey:          "sk-" + model.MaskTokenKey(record.Token.Key),
		Status:             record.Token.Status,
		EffectiveStatus:    enterpriseEffectiveTokenStatus(record.Token),
		CreatedTime:        record.Token.CreatedTime,
		AccessedTime:       record.Token.AccessedTime,
		ExpiredTime:        record.Token.ExpiredTime,
		RemainQuota:        record.Token.RemainQuota,
		UsedQuota:          record.Token.UsedQuota,
		UnlimitedQuota:     record.Token.UnlimitedQuota,
		ModelLimitsEnabled: record.Token.ModelLimitsEnabled,
		ModelLimits:        record.Token.ModelLimits,
		AllowIps:           record.Token.AllowIps,
		Group:              record.Token.Group,
		CrossGroupRetry:    record.Token.CrossGroupRetry,
		RateLimit:          record.Token.RateLimit,
		Username:           record.Username,
		DisplayName:        record.DisplayName,
		Email:              record.Email,
		UserGroup:          record.UserGroup,
	}
}

func countEnterpriseAPIKeyRateLimitHits(filters model.EnterpriseTokenFilters, since int64) int64 {
	query := model.LOG_DB.Model(&model.Log{}).
		Where("type = ? AND created_at >= ? AND token_id > 0", model.LogTypeError, since).
		Where("(other LIKE ? OR content LIKE ? OR content LIKE ? OR content LIKE ?)",
			`%"status_code":429%`, "%rate limit%", "%Too Many Requests%", "%请求数限制%")

	if filters.RestrictByManagerRole && filters.ManagerRole < common.RoleRootUser {
		var tokenIds []int
		if err := model.DB.Model(&model.Token{}).
			Joins("LEFT JOIN users ON users.id = tokens.user_id").
			Where("users.role < ? OR users.id = ?", filters.ManagerRole, filters.ManagerId).
			Pluck("tokens.id", &tokenIds).Error; err != nil || len(tokenIds) == 0 {
			return 0
		}
		query = query.Where("token_id IN ?", tokenIds)
	}

	var count int64
	_ = query.Count(&count).Error
	return count
}

func ListEnterpriseAPIKeys(filters model.EnterpriseTokenFilters, offset int, limit int) ([]dto.EnterpriseAPIKeyItem, int64, dto.EnterpriseAPIKeySummary, error) {
	records, total, err := model.SearchEnterpriseTokens(filters, offset, limit)
	if err != nil {
		return nil, 0, dto.EnterpriseAPIKeySummary{}, err
	}
	now := time.Now().Unix()
	tokenIds := make([]int, 0, len(records))
	for _, record := range records {
		tokenIds = append(tokenIds, record.Token.Id)
	}
	recentFailures := map[int]int64{}
	if len(tokenIds) > 0 {
		var rows []struct {
			TokenId int
			Count   int64
		}
		_ = model.LOG_DB.Model(&model.Log{}).
			Select("token_id, COUNT(*) AS count").
			Where("token_id IN ? AND type = ? AND created_at >= ?", tokenIds, model.LogTypeError, now-24*60*60).
			Group("token_id").
			Scan(&rows).Error
		for _, row := range rows {
			recentFailures[row.TokenId] = row.Count
		}
	}
	items := make([]dto.EnterpriseAPIKeyItem, 0, len(records))
	for _, record := range records {
		item := enterpriseAPIKeyItem(record)
		item.RecentFailureCount = recentFailures[record.Token.Id]
		items = append(items, item)
	}

	base := model.DB.Model(&model.Token{})
	if filters.RestrictByManagerRole && filters.ManagerRole < common.RoleRootUser {
		var ids []int
		if err := model.DB.Model(&model.User{}).Where("role < ? OR id = ?", filters.ManagerRole, filters.ManagerId).Pluck("id", &ids).Error; err != nil {
			return nil, 0, dto.EnterpriseAPIKeySummary{}, err
		}
		if len(ids) == 0 {
			return items, total, dto.EnterpriseAPIKeySummary{}, nil
		}
		base = base.Where("user_id IN ?", ids)
	}
	summary := dto.EnterpriseAPIKeySummary{}
	_ = base.Count(&summary.Total).Error
	_ = base.Where("status = ? AND (expired_time = -1 OR expired_time > ?) AND (unlimited_quota = ? OR remain_quota > 0)", common.TokenStatusEnabled, now, true).Count(&summary.Active).Error
	_ = base.Where("status = ? AND expired_time > ? AND expired_time <= ?", common.TokenStatusEnabled, now, now+7*24*60*60).Count(&summary.ExpiringSoon).Error
	_ = base.Where("status = ? OR (status = ? AND unlimited_quota = ? AND remain_quota <= 0)", common.TokenStatusExhausted, common.TokenStatusEnabled, false).Count(&summary.Exhausted).Error
	_ = base.Where("status = ?", common.TokenStatusDisabled).Count(&summary.Disabled).Error
	_ = base.Distinct("user_id").Count(&summary.ActiveUsers).Error
	var quotaSum struct{ Value int64 }
	_ = base.Select("COALESCE(SUM(used_quota), 0) AS value").Scan(&quotaSum).Error
	summary.TotalUsedQuota = quotaSum.Value
	summary.RateLimitHits = countEnterpriseAPIKeyRateLimitHits(filters, now-24*60*60)
	return items, total, summary, nil
}

func ListEnterpriseAPIKeyUsers(managerId int, managerRole int) ([]dto.EnterpriseAPIKeyUser, error) {
	query := model.DB.Model(&model.User{}).Omit("password").Order("username ASC")
	if managerRole < common.RoleRootUser {
		query = query.Where("role < ? OR id = ?", managerRole, managerId)
	}
	var users []model.User
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	items := make([]dto.EnterpriseAPIKeyUser, 0, len(users))
	for _, user := range users {
		items = append(items, dto.EnterpriseAPIKeyUser{
			Id: user.Id, Username: user.Username, DisplayName: user.DisplayName,
			Email: user.Email, Group: user.Group, Status: user.Status, Role: user.Role,
		})
	}
	return items, nil
}

func CreateEnterpriseAPIKey(input dto.EnterpriseAPIKeyMutationInput) (*dto.EnterpriseAPIKeySecret, error) {
	if err := validateEnterpriseAPIKeyInput(&input, true); err != nil {
		return nil, err
	}
	user, err := model.GetUserById(input.UserId, false)
	if err != nil {
		return nil, err
	}
	count, err := model.CountUserTokens(input.UserId)
	if err != nil {
		return nil, err
	}
	if int(count) >= operation_setting.GetMaxUserTokens() {
		return nil, fmt.Errorf("该用户已达到最大密钥数量限制（%d）", operation_setting.GetMaxUserTokens())
	}
	key, err := common.GenerateKey()
	if err != nil {
		return nil, err
	}
	group := input.Group
	if group == "" {
		group = user.Group
	}
	token := model.Token{
		UserId: input.UserId, Key: key, Name: input.Name, Status: input.Status,
		CreatedTime: common.GetTimestamp(), AccessedTime: common.GetTimestamp(),
		ExpiredTime: input.ExpiredTime, RemainQuota: input.RemainQuota,
		UnlimitedQuota: input.UnlimitedQuota, ModelLimitsEnabled: input.ModelLimitsEnabled,
		ModelLimits: input.ModelLimits, AllowIps: input.AllowIps, Group: group,
		CrossGroupRetry: input.CrossGroupRetry, RateLimit: input.RateLimit,
	}
	if err := token.Insert(); err != nil {
		return nil, err
	}
	record := model.EnterpriseTokenRecord{Token: token, Username: user.Username, DisplayName: user.DisplayName, Email: user.Email, UserGroup: user.Group, UserRole: user.Role}
	return &dto.EnterpriseAPIKeySecret{Item: enterpriseAPIKeyItem(record), SecretKey: "sk-" + key}, nil
}

func UpdateEnterpriseAPIKey(id int, input dto.EnterpriseAPIKeyMutationInput) (*dto.EnterpriseAPIKeyItem, error) {
	if err := validateEnterpriseAPIKeyInput(&input, false); err != nil {
		return nil, err
	}
	record, err := model.GetEnterpriseTokenByID(id)
	if err != nil {
		return nil, err
	}
	token := record.Token
	token.Name = input.Name
	token.Status = input.Status
	token.ExpiredTime = input.ExpiredTime
	token.RemainQuota = input.RemainQuota
	token.UnlimitedQuota = input.UnlimitedQuota
	token.ModelLimitsEnabled = input.ModelLimitsEnabled
	token.ModelLimits = input.ModelLimits
	token.AllowIps = input.AllowIps
	token.Group = input.Group
	token.CrossGroupRetry = input.CrossGroupRetry
	token.RateLimit = input.RateLimit
	if err := token.Update(); err != nil {
		return nil, err
	}
	record.Token = token
	item := enterpriseAPIKeyItem(*record)
	return &item, nil
}

func RotateEnterpriseAPIKey(id int) (*dto.EnterpriseAPIKeySecret, error) {
	record, err := model.GetEnterpriseTokenByID(id)
	if err != nil {
		return nil, err
	}
	key, err := common.GenerateKey()
	if err != nil {
		return nil, err
	}
	if err := model.RotateEnterpriseTokenKey(&record.Token, key); err != nil {
		return nil, err
	}
	item := enterpriseAPIKeyItem(*record)
	return &dto.EnterpriseAPIKeySecret{Item: item, SecretKey: "sk-" + key}, nil
}

func DeleteEnterpriseAPIKey(id int) (*dto.EnterpriseAPIKeyItem, error) {
	record, err := model.GetEnterpriseTokenByID(id)
	if err != nil {
		return nil, err
	}
	item := enterpriseAPIKeyItem(*record)
	if err := record.Token.Delete(); err != nil {
		return nil, err
	}
	return &item, nil
}
