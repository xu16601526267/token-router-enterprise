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

	statuses := map[string]bool{}
	for _, item := range data.RecentLogs {
		statuses[item.Status] = true
	}
	assert.True(t, statuses["success"])
	assert.True(t, statuses["error"])
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
	assert.Contains(t, text, "success")
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
