package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func enterpriseClearTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	for _, table := range []string{
		"tokens",
		"users",
		"logs",
		"channels",
		"usage_ledgers",
		"top_ups",
		"user_subscriptions",
		"settlement_statements",
		"suppliers",
	} {
		require.NoError(t, db.Exec("DELETE FROM "+table).Error)
	}
}

func setupEnterpriseUsersTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalRedisEnabled := common.RedisEnabled

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db := model.DB
	require.NotNil(t, db)
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Token{},
		&model.Log{},
		&model.Channel{},
		&model.UsageLedger{},
		&model.TopUp{},
		&model.UserSubscription{},
		&model.SettlementStatement{},
		&model.Supplier{},
	))
	enterpriseClearTestTables(t, db)

	t.Cleanup(func() {
		enterpriseClearTestTables(t, db)
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		common.RedisEnabled = originalRedisEnabled
	})

	return db
}

func createEnterpriseUser(t *testing.T, db *gorm.DB, username string, role int, status int, group string) model.User {
	t.Helper()

	user := model.User{
		Username: username,
		Password: "password",
		Role:     role,
		Status:   status,
		Group:    group,
		AffCode:  username + "-aff",
	}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func createEnterpriseUserToken(t *testing.T, db *gorm.DB, userId int, name string, status int) {
	t.Helper()

	require.NoError(t, db.Create(&model.Token{
		UserId:         userId,
		Key:            name + "-raw-key",
		Status:         status,
		Name:           name,
		CreatedTime:    1,
		AccessedTime:   1,
		ExpiredTime:    -1,
		RemainQuota:    100,
		UnlimitedQuota: true,
		Group:          "default",
	}).Error)
}

func TestGetEnterpriseUsersRootSeesAllUsers(t *testing.T) {
	db := setupEnterpriseUsersTestDB(t)
	root := createEnterpriseUser(t, db, "root", common.RoleRootUser, common.UserStatusEnabled, "root")
	admin := createEnterpriseUser(t, db, "admin", common.RoleAdminUser, common.UserStatusEnabled, "ops")
	customer := createEnterpriseUser(t, db, "customer", common.RoleCommonUser, common.UserStatusEnabled, "tenant")
	createEnterpriseUserToken(t, db, root.Id, "root-key", common.TokenStatusEnabled)
	createEnterpriseUserToken(t, db, admin.Id, "admin-key", common.TokenStatusEnabled)
	createEnterpriseUserToken(t, db, customer.Id, "customer-key", common.TokenStatusEnabled)

	data, err := GetEnterpriseUsers(50, root.Id, root.Role)

	require.NoError(t, err)
	require.NotNil(t, data)
	assert.EqualValues(t, 3, data.Summary.TotalUsers)
	assert.EqualValues(t, 3, data.Summary.ActiveAPIKeys)
	assert.Len(t, data.Users, 3)
}

func TestGetEnterpriseUsersAdminScopeExcludesRootAndPeerAdmins(t *testing.T) {
	db := setupEnterpriseUsersTestDB(t)
	root := createEnterpriseUser(t, db, "root", common.RoleRootUser, common.UserStatusEnabled, "root")
	admin := createEnterpriseUser(t, db, "admin", common.RoleAdminUser, common.UserStatusEnabled, "ops")
	peerAdmin := createEnterpriseUser(t, db, "peer-admin", common.RoleAdminUser, common.UserStatusEnabled, "ops")
	customerA := createEnterpriseUser(t, db, "customer-a", common.RoleCommonUser, common.UserStatusEnabled, "tenant-a")
	customerB := createEnterpriseUser(t, db, "customer-b", common.RoleCommonUser, common.UserStatusDisabled, "tenant-b")
	createEnterpriseUserToken(t, db, root.Id, "root-key", common.TokenStatusEnabled)
	createEnterpriseUserToken(t, db, admin.Id, "admin-key", common.TokenStatusEnabled)
	createEnterpriseUserToken(t, db, peerAdmin.Id, "peer-admin-key", common.TokenStatusEnabled)
	createEnterpriseUserToken(t, db, customerA.Id, "customer-a-key", common.TokenStatusEnabled)
	createEnterpriseUserToken(t, db, customerB.Id, "customer-b-key", common.TokenStatusDisabled)

	data, err := GetEnterpriseUsers(50, admin.Id, admin.Role)

	require.NoError(t, err)
	require.NotNil(t, data)
	assert.EqualValues(t, 3, data.Summary.TotalUsers)
	assert.EqualValues(t, 2, data.Summary.ActiveUsers)
	assert.EqualValues(t, 1, data.Summary.AdminUsers)
	assert.EqualValues(t, 1, data.Summary.DisabledUsers)
	assert.EqualValues(t, 2, data.Summary.ActiveAPIKeys)

	usernames := make(map[string]struct{}, len(data.Users))
	for _, user := range data.Users {
		usernames[user.Username] = struct{}{}
	}
	assert.Contains(t, usernames, "admin")
	assert.Contains(t, usernames, "customer-a")
	assert.Contains(t, usernames, "customer-b")
	assert.NotContains(t, usernames, "root")
	assert.NotContains(t, usernames, "peer-admin")
}
