package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupEnterpriseAnalyticsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	return setupEnterpriseUsersTestDB(t)
}

func TestEnterpriseUsageAnalyticsTrendAndStatusAreClientFriendly(t *testing.T) {
	db := setupEnterpriseAnalyticsTestDB(t)
	require.NoError(t, db.Create(&model.Channel{Id: 7, Name: "stable-upstream", Status: common.ChannelStatusEnabled}).Error)

	start := int64(1_700_000_000)
	require.NoError(t, db.Create(&model.Log{
		UserId: 1, CreatedAt: start + 60, Type: model.LogTypeConsume,
		Username: "alice", TokenName: "prod-key", ModelName: "gpt-test",
		Group: "vip", Quota: 300, PromptTokens: 10, CompletionTokens: 20,
		UseTime: 2, ChannelId: 7, Ip: "203.0.113.8", RequestId: "req-success",
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		UserId: 1, CreatedAt: start + 120, Type: model.LogTypeError,
		Username: "alice", TokenName: "prod-key", ModelName: "gpt-test",
		Group: "vip", ChannelId: 7, Ip: "203.0.113.8", RequestId: "req-error",
	}).Error)
	require.NoError(t, db.Create(&model.UsageLedger{
		RequestId: "ledger-success", CreatedAt: start + 60, Status: "success",
		PromptTokens: 10, CompletionTokens: 20, SellQuota: 300, CostQuota: 120,
		CacheHit: true,
	}).Error)

	data, err := GetEnterpriseUsageAnalytics(start, start+3600)

	require.NoError(t, err)
	require.NotNil(t, data)
	assert.EqualValues(t, 2, data.Metrics.TotalRequests)
	assert.EqualValues(t, 1, data.Metrics.ErrorRequests)
	assert.EqualValues(t, 10, data.Metrics.PromptTokens)
	assert.EqualValues(t, 20, data.Metrics.CompletionTokens)
	require.Len(t, data.Trend, 1)
	assert.EqualValues(t, 10, data.Trend[0].PromptTokens)
	assert.EqualValues(t, 20, data.Trend[0].CompletionTokens)
	assert.EqualValues(t, 2000, data.Trend[0].AverageLatencyMs)
	assert.InDelta(t, 1, data.Trend[0].CacheHitRate, 0.0001)

	statuses := map[string]bool{}
	requestTypes := map[string]bool{}
	for _, item := range data.RecentLogs {
		statuses[item.Status] = true
		requestTypes[item.RequestType] = true
	}
	assert.True(t, statuses["success"])
	assert.True(t, statuses["error"])
	assert.True(t, requestTypes["chat"])
}

func TestBuildEnterpriseUsageAnalyticsCSVIncludesFullLogSection(t *testing.T) {
	db := setupEnterpriseAnalyticsTestDB(t)
	start := int64(1_700_000_000)
	require.NoError(t, db.Create(&model.Log{
		UserId: 1, CreatedAt: start + 60, Type: model.LogTypeConsume,
		Username: "alice", ModelName: "gpt-test", Group: "vip",
		Quota: 300, PromptTokens: 10, CompletionTokens: 20, RequestId: "req-success",
	}).Error)

	body, err := BuildEnterpriseUsageAnalyticsCSV(start, start+3600)

	require.NoError(t, err)
	text := string(body)
	assert.True(t, strings.HasPrefix(text, "\uFEFF"))
	assert.Contains(t, text, "用量指标")
	assert.Contains(t, text, "调用日志")
	assert.Contains(t, text, "req-success")
	assert.Contains(t, text, "请求类型")
	assert.Contains(t, text, "chat")
	assert.Contains(t, text, "success")
}

func TestEnterpriseUsageAnalyticsFiltersAndPaginatesLogs(t *testing.T) {
	db := setupEnterpriseAnalyticsTestDB(t)
	require.NoError(t, db.Create(&model.Channel{Id: 11, Name: "filtered-upstream", Status: common.ChannelStatusEnabled}).Error)
	start := int64(1_700_000_000)
	require.NoError(t, db.Create(&model.Log{
		UserId: 1, CreatedAt: start + 10, Type: model.LogTypeConsume,
		Username: "alice", TokenName: "prod-key", ModelName: "gpt-filter",
		Group: "vip", Quota: 100, PromptTokens: 10, CompletionTokens: 20,
		UseTime: 1, ChannelId: 11, Ip: "203.0.113.10", RequestId: "req-needle-a",
		Content: "needle success",
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		UserId: 1, CreatedAt: start + 20, Type: model.LogTypeError,
		Username: "alice", TokenName: "prod-key", ModelName: "gpt-filter",
		Group: "vip", ChannelId: 11, Ip: "203.0.113.10", RequestId: "req-needle-b",
		Content: "needle error",
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		UserId: 2, CreatedAt: start + 30, Type: model.LogTypeConsume,
		Username: "bob", TokenName: "dev-key", ModelName: "gpt-other",
		Group: "default", Quota: 500, RequestId: "req-other",
		Content: "unmatched",
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		UserId: 3, CreatedAt: start + 40, Type: model.LogTypeConsume,
		Username: "carol", TokenName: "embed-key", ModelName: "text-embedding-3-small",
		Group: "default", Quota: 200, RequestId: "req-embedding",
		Content: "embedding request",
	}).Error)

	data, err := GetEnterpriseUsageAnalyticsWithFilters(start, start+3600, EnterpriseUsageFilters{
		Keyword: "needle", ModelName: "gpt-filter", Page: 1, PageSize: 1,
		SortBy: "created_at", SortOrder: "asc",
	})

	require.NoError(t, err)
	require.NotNil(t, data)
	assert.EqualValues(t, 2, data.TotalLogs)
	assert.EqualValues(t, 2, data.Metrics.TotalRequests)
	assert.EqualValues(t, 1, data.Metrics.ErrorRequests)
	require.Len(t, data.RecentLogs, 1)
	assert.Equal(t, "req-needle-a", data.RecentLogs[0].RequestId)
	assert.Equal(t, "filtered-upstream", data.RecentLogs[0].ChannelName)

	successOnly, err := GetEnterpriseUsageAnalyticsWithFilters(start, start+3600, EnterpriseUsageFilters{
		Keyword: "needle", Status: "success",
	})

	require.NoError(t, err)
	assert.EqualValues(t, 1, successOnly.TotalLogs)
	assert.EqualValues(t, 1, successOnly.Metrics.TotalRequests)
	assert.EqualValues(t, 0, successOnly.Metrics.ErrorRequests)

	embeddingOnly, err := GetEnterpriseUsageAnalyticsWithFilters(start, start+3600, EnterpriseUsageFilters{
		RequestType: "embedding",
	})

	require.NoError(t, err)
	assert.EqualValues(t, 1, embeddingOnly.TotalLogs)
	assert.EqualValues(t, 1, embeddingOnly.Metrics.TotalRequests)
	require.Len(t, embeddingOnly.RecentLogs, 1)
	assert.Equal(t, "embedding", embeddingOnly.RecentLogs[0].RequestType)

	chatOnly, err := GetEnterpriseUsageAnalyticsWithFilters(start, start+3600, EnterpriseUsageFilters{
		RequestType: "chat",
	})

	require.NoError(t, err)
	assert.EqualValues(t, 3, chatOnly.TotalLogs)
	assert.EqualValues(t, 3, chatOnly.Metrics.TotalRequests)
}

func TestEnterpriseChannelCenterFiltersAndCSV(t *testing.T) {
	db := setupEnterpriseAnalyticsTestDB(t)
	start := int64(1_700_000_000)
	require.NoError(t, db.Create(&model.Supplier{
		Id: 9, Name: "Acme Supplier", Type: model.SupplierTypeThirdParty, Status: common.ChannelStatusEnabled,
	}).Error)
	tag := "premium"
	remark := "primary route"
	priority := int64(20)
	weight := uint(80)
	require.NoError(t, db.Create(&model.Channel{
		Id: 21, Name: "alpha-route", Key: "sk-alpha", Status: common.ChannelStatusEnabled,
		SupplierId: 9, Models: "gpt-alpha,gpt-beta", Group: "vip", Tag: &tag,
		Remark: &remark, Balance: 50, Priority: &priority, Weight: &weight,
	}).Error)
	require.NoError(t, db.Create(&model.Channel{
		Id: 22, Name: "beta-route", Key: "sk-beta", Status: common.ChannelStatusManuallyDisabled,
		SupplierId: 9, Models: "gpt-beta", Group: "default",
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		CreatedAt: start + 30, Type: model.LogTypeConsume, ChannelId: 21,
		ModelName: "gpt-alpha", RequestId: "channel-success", UseTime: 2,
	}).Error)
	require.NoError(t, db.Create(&model.Log{
		CreatedAt: start + 60, Type: model.LogTypeError, ChannelId: 21,
		ModelName: "gpt-alpha", RequestId: "channel-error",
	}).Error)

	data, err := GetEnterpriseChannelCenterWithFilters(start, start+3600, EnterpriseChannelFilters{
		Keyword: "alpha", Status: common.ChannelStatusEnabled, Page: 1, PageSize: 1,
	})

	require.NoError(t, err)
	require.NotNil(t, data)
	assert.EqualValues(t, 1, data.Total)
	require.Len(t, data.Items, 1)
	assert.Equal(t, "alpha-route", data.Items[0].Name)
	assert.Equal(t, "Acme Supplier", data.Items[0].SupplierName)
	assert.EqualValues(t, 2, data.Items[0].Requests)
	assert.InDelta(t, 0.5, data.Items[0].SuccessRate, 0.0001)

	body, err := BuildEnterpriseChannelsCSV(start, start+3600, EnterpriseChannelFilters{Keyword: "alpha"})

	require.NoError(t, err)
	text := string(body)
	assert.Contains(t, text, "渠道与供应商指标")
	assert.Contains(t, text, "alpha-route")
	assert.NotContains(t, text, "beta-route")
}

func TestGenerateEnterpriseSettlementStatementReturnsEnterpriseItem(t *testing.T) {
	db := setupEnterpriseAnalyticsTestDB(t)
	start := int64(1_700_000_000)
	user := createEnterpriseUser(t, db, "settlement-user", common.RoleCommonUser, common.UserStatusEnabled, "tenant")
	require.NoError(t, db.Create(&model.UsageLedger{
		RequestId: "settlement-ledger-1", CreatedAt: start + 60, Status: "success",
		UserId: user.Id, SellQuota: 1000, CostQuota: 400,
		PromptTokens: 100, CompletionTokens: 50, CachedTokens: 10,
	}).Error)

	item, err := GenerateEnterpriseSettlementStatement(model.SettlementStatementGenerateInput{
		SubjectType: model.SettlementSubjectUser,
		UserId:      user.Id,
		PeriodStart: start,
		PeriodEnd:   start + 3600,
	})

	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Positive(t, item.Id)
	assert.Equal(t, model.SettlementSubjectUser, item.SubjectType)
	assert.Equal(t, "settlement-user", item.SubjectName)
	assert.EqualValues(t, 1000, item.TotalSellQuota)
	assert.EqualValues(t, 400, item.TotalCostQuota)
	assert.EqualValues(t, 600, item.GrossProfitQuota)
	assert.EqualValues(t, 1, item.TotalRequests)
	assert.Equal(t, model.SettlementStatusDraft, item.Status)
}

func TestBuildEnterpriseAPIKeysCSVIsScopedAndMasked(t *testing.T) {
	db := setupEnterpriseUsersTestDB(t)
	admin := createEnterpriseUser(t, db, "admin-export", common.RoleAdminUser, common.UserStatusEnabled, "ops")
	customer := createEnterpriseUser(t, db, "customer-export", common.RoleCommonUser, common.UserStatusEnabled, "tenant")
	peerAdmin := createEnterpriseUser(t, db, "peer-admin-export", common.RoleAdminUser, common.UserStatusEnabled, "ops")
	createEnterpriseUserToken(t, db, customer.Id, "customer-export-key", common.TokenStatusEnabled)
	createEnterpriseUserToken(t, db, peerAdmin.Id, "peer-admin-export-key", common.TokenStatusEnabled)

	body, err := BuildEnterpriseAPIKeysCSV(model.EnterpriseTokenFilters{
		ManagerId: admin.Id, ManagerRole: admin.Role, RestrictByManagerRole: true,
	})

	require.NoError(t, err)
	text := string(body)
	assert.Contains(t, text, "customer-export-key")
	assert.NotContains(t, text, "peer-admin-export-key")
	assert.NotContains(t, text, "customer-export-key-raw-key")
	assert.Contains(t, text, "sk-cust")
}
