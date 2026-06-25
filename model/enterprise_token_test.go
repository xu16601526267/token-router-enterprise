package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func enterpriseTokenClearTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	require.NoError(t, db.Exec("DELETE FROM tokens").Error)
	require.NoError(t, db.Exec("DELETE FROM users").Error)
}

func setupEnterpriseTokenTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalRedisEnabled := common.RedisEnabled

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db := DB
	require.NotNil(t, db)
	LOG_DB = db
	require.NoError(t, db.AutoMigrate(&User{}, &Token{}))
	enterpriseTokenClearTestTables(t, db)

	t.Cleanup(func() {
		enterpriseTokenClearTestTables(t, db)
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		common.RedisEnabled = originalRedisEnabled
	})

	return db
}

func TestSearchEnterpriseTokensUsesDialectQuotedGroupColumn(t *testing.T) {
	db := setupEnterpriseTokenTestDB(t)

	user := User{
		Username:    "enterprise-user",
		Password:    "password",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		DisplayName: "Enterprise User",
		Email:       "enterprise@example.com",
		Group:       "tenant-vip",
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&Token{
		UserId:         user.Id,
		Key:            "enterprise-test-key",
		Status:         common.TokenStatusEnabled,
		Name:           "production-key",
		CreatedTime:    1,
		AccessedTime:   1,
		ExpiredTime:    -1,
		RemainQuota:    1000,
		UnlimitedQuota: true,
		Group:          "api-prod",
	}).Error)

	common.UsingSQLite = false
	common.UsingPostgreSQL = true
	records, total, err := SearchEnterpriseTokens(EnterpriseTokenFilters{Group: "api-prod"}, 0, 20)

	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, records, 1)
	require.Equal(t, "tenant-vip", records[0].UserGroup)
	require.Equal(t, "production-key", records[0].Token.Name)
}
