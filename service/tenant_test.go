package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func tenantTestUser(t *testing.T, id int, username string, quota int, role int) *model.User {
	t.Helper()
	user := &model.User{
		Id:       id,
		Username: username,
		Password: "password123",
		Role:     role,
		Status:   common.UserStatusEnabled,
		Quota:    quota,
		Group:    "default",
		AffCode:  fmt.Sprintf("aff-%d-%s", id, username),
	}
	require.NoError(t, model.DB.Create(user).Error)
	return user
}

func tenantTestContext(requestID string) *gin.Context {
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set(common.UsageLedgerRequestIdHeader, requestID)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	ctx.Set(common.RequestIdKey, requestID)
	return ctx
}

func tenantTestSupply(t *testing.T, modelName string) (*model.Supplier, *model.Channel) {
	t.Helper()
	supplier := &model.Supplier{Name: "supplier-" + modelName, Type: model.SupplierTypeThirdParty}
	require.NoError(t, supplier.Insert())
	channel := &model.Channel{
		Id:         1000 + supplier.Id,
		Name:       "channel-" + modelName,
		Key:        "sk-test",
		Status:     common.ChannelStatusEnabled,
		SupplierId: supplier.Id,
	}
	require.NoError(t, model.DB.Create(channel).Error)
	agreement := &model.SupplierAgreement{
		SupplierId:             supplier.Id,
		ModelName:              modelName,
		CostModelRatio:         1,
		CostCompletionRatio:    2,
		CostCacheRatio:         0.1,
		CostCacheCreationRatio: 0.5,
		Status:                 1,
	}
	require.NoError(t, agreement.Insert())
	return supplier, channel
}

func tenantTestCreatePostpaidTenant(t *testing.T, ownerUserID int, name string, creditLimit int64) *model.Tenant {
	t.Helper()
	tenant, err := CreateTenantWithDefaults(TenantCreateInput{
		Name:         name,
		Type:         "enterprise",
		OwnerUserId:  ownerUserID,
		BillingMode:  model.BillingModePostpaid,
		CreditLimit:  creditLimit,
		StatementDay: 1,
		PaymentTerms: 15,
	}, 1, "127.0.0.1")
	require.NoError(t, err)
	return tenant
}

func tenantTestCreatePolicy(t *testing.T, tenantID int, modelName string) *model.TenantModelPolicy {
	t.Helper()
	policy := &model.TenantModelPolicy{
		TenantId:  tenantID,
		ModelName: modelName,
		Visible:   true,
		Enabled:   true,
		Alias:     modelName + "-alias",
	}
	require.NoError(t, UpsertTenantModelPolicy(policy, 1, "127.0.0.1"))
	require.NotZero(t, policy.Id)
	return policy
}

func tenantTestCreateAppAndCustomer(t *testing.T, tenantID int, userID int, suffix string) (*model.TenantApp, *model.TenantEndCustomer) {
	t.Helper()
	customer := &model.TenantEndCustomer{
		TenantId:     tenantID,
		UserId:       userID,
		CustomerType: "employee",
		Status:       model.TenantStatusActive,
		ExternalId:   "ext-" + suffix,
	}
	require.NoError(t, CreateTenantEndCustomer(customer, userID, "127.0.0.1"))
	require.NotZero(t, customer.Id)
	app := &model.TenantApp{
		TenantId: tenantID,
		Name:     "app-" + suffix,
		Env:      "prod",
		OwnerId:  userID,
		Status:   model.TenantStatusActive,
	}
	require.NoError(t, CreateTenantApp(app, userID, "127.0.0.1"))
	require.NotZero(t, app.Id)
	return app, customer
}

func TestTenantPostpaidB2B2CLedgerStatementPaymentClosure(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)

	tenantTestUser(t, 1, "platform-root", 0, common.RoleRootUser)
	owner := tenantTestUser(t, 2, "b-owner", 0, common.RoleCommonUser)
	endUser := tenantTestUser(t, 3, "b-employee-c", 0, common.RoleCommonUser)
	tenant := tenantTestCreatePostpaidTenant(t, owner.Id, "Acme Enterprise", 1000)
	policy := tenantTestCreatePolicy(t, tenant.Id, "gpt-test")
	app, customer := tenantTestCreateAppAndCustomer(t, tenant.Id, endUser.Id, "acme-c1")

	token, secret, err := CreateTenantAPIKey(tenant.Id, TenantAPIKeyInput{
		UserId:         endUser.Id,
		Name:           "acme-c1-key",
		AppId:          app.Id,
		EndCustomerId:  customer.Id,
		ModelPolicyId:  policy.Id,
		UnlimitedQuota: true,
	}, owner.Id, "127.0.0.1")
	require.NoError(t, err)
	require.NotEmpty(t, secret)
	require.Equal(t, tenant.Id, token.TenantId)
	require.Equal(t, app.Id, token.AppId)
	require.Equal(t, customer.Id, token.EndCustomerId)
	require.Equal(t, model.TokenOwnerScopeEndCustomer, token.OwnerScope)

	_, channel := tenantTestSupply(t, "gpt-test")
	ctx := tenantTestContext("tenant-ledger-closure-1")
	relayInfo := &relaycommon.RelayInfo{
		UserId:            endUser.Id,
		TokenId:           token.Id,
		TenantId:          tenant.Id,
		AppId:             app.Id,
		EndCustomerId:     customer.Id,
		ModelPolicyId:     policy.Id,
		TenantBillingMode: model.BillingModePostpaid,
		RequestId:         "tenant-ledger-closure-1",
		OriginModelName:   "gpt-test",
		StartTime:         time.Now().Add(-time.Second),
		ChannelMeta:       &relaycommon.ChannelMeta{ChannelId: channel.Id, ChannelBaseUrl: "upstream"},
	}
	apiErr := PreConsumeBilling(ctx, 200, relayInfo)
	require.Nil(t, apiErr)
	require.Equal(t, BillingSourceTenantPostpaid, relayInfo.BillingSource)
	require.NoError(t, SettleBilling(ctx, relayInfo, 200))

	ledger, err := RecordUsage(ctx, relayInfo, &dto.Usage{PromptTokens: 100, CompletionTokens: 20}, 200)
	require.NoError(t, err)
	require.Equal(t, tenant.Id, ledger.TenantId)
	require.Equal(t, app.Id, ledger.AppId)
	require.Equal(t, customer.Id, ledger.EndCustomerId)
	require.Equal(t, model.BillingModePostpaid, ledger.BillingMode)
	require.Equal(t, 200, ledger.PostpaidQuota)
	require.NotEmpty(t, ledger.PriceSnapshot)

	_, err = RecordUsage(ctx, relayInfo, &dto.Usage{PromptTokens: 100, CompletionTokens: 20}, 999)
	require.NoError(t, err)

	var ledgerCount int64
	require.NoError(t, model.DB.Model(&model.UsageLedger{}).Where("tenant_id = ?", tenant.Id).Count(&ledgerCount).Error)
	require.Equal(t, int64(1), ledgerCount)

	account, err := model.GetCreditAccountByTenantId(tenant.Id)
	require.NoError(t, err)
	require.Equal(t, int64(200), account.UnbilledAmount)
	require.Equal(t, int64(800), account.AvailableCredit)

	statement, err := GenerateTenantBillingStatement(tenant.Id, common.GetTimestamp()-60, common.GetTimestamp()+60, 0, owner.Id, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, int64(200), statement.Amount)
	require.Equal(t, model.BillingStatementStatusDraft, statement.Status)

	statement, err = ConfirmTenantBillingStatement(tenant.Id, statement.Id, owner.Id, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, model.BillingStatementStatusConfirmed, statement.Status)
	account, err = model.GetCreditAccountByTenantId(tenant.Id)
	require.NoError(t, err)
	require.Equal(t, int64(0), account.UnbilledAmount)
	require.Equal(t, int64(200), account.BilledUnpaidAmount)

	statement, err = RegisterTenantPaymentAndInvoice(tenant.Id, statement.Id, 200, "bank_transfer", "INV-ACME-001", "issued", owner.Id, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, model.BillingStatementStatusPaid, statement.Status)
	account, err = model.GetCreditAccountByTenantId(tenant.Id)
	require.NoError(t, err)
	require.Equal(t, int64(0), account.BilledUnpaidAmount)
	require.Equal(t, int64(1000), account.AvailableCredit)

	var payments int64
	require.NoError(t, model.DB.Model(&model.PaymentRecord{}).Where("statement_id = ?", statement.Id).Count(&payments).Error)
	require.Equal(t, int64(1), payments)
	var invoices int64
	require.NoError(t, model.DB.Model(&model.Invoice{}).Where("statement_id = ?", statement.Id).Count(&invoices).Error)
	require.Equal(t, int64(1), invoices)
	var auditCount int64
	require.NoError(t, model.DB.Model(&model.AuditLog{}).Where("scope_type = ? AND scope_id = ?", model.ScopeTenant, tenant.Id).Count(&auditCount).Error)
	require.GreaterOrEqual(t, auditCount, int64(6))
}

func TestTenantBillingStatementImmutableAfterConfirmAndRejectsOverpay(t *testing.T) {
	truncate(t)

	owner := tenantTestUser(t, 1, "immutable-owner", 0, common.RoleCommonUser)
	tenant := tenantTestCreatePostpaidTenant(t, owner.Id, "Immutable Tenant", 1000)
	now := common.GetTimestamp()
	ledger := &model.UsageLedger{
		RequestId:     "immutable-ledger-1",
		TenantId:      tenant.Id,
		ModelName:     "gpt-immutable",
		SellQuota:     300,
		PostpaidQuota: 300,
		BillingMode:   model.BillingModePostpaid,
		BillingPeriod: billingPeriodFromUnix(now),
		Status:        "success",
		CreatedAt:     now,
	}
	_, err := insertUsageLedgerWithTenantCredit(ledger)
	require.NoError(t, err)

	statement, err := GenerateTenantBillingStatement(tenant.Id, now-60, now+60, 0, owner.Id, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, int64(300), statement.Payable)
	statement, err = ConfirmTenantBillingStatement(tenant.Id, statement.Id, owner.Id, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, model.BillingStatementStatusConfirmed, statement.Status)

	_, err = insertUsageLedgerWithTenantCredit(&model.UsageLedger{
		RequestId:     "immutable-ledger-2",
		TenantId:      tenant.Id,
		ModelName:     "gpt-immutable",
		SellQuota:     100,
		PostpaidQuota: 100,
		BillingMode:   model.BillingModePostpaid,
		BillingPeriod: billingPeriodFromUnix(now),
		Status:        "success",
		CreatedAt:     now + 1,
	})
	require.NoError(t, err)
	_, err = GenerateTenantBillingStatement(tenant.Id, now-60, now+60, 0, owner.Id, "127.0.0.1")
	require.Error(t, err)
	saved, err := getTenantStatement(tenant.Id, statement.Id)
	require.NoError(t, err)
	require.Equal(t, model.BillingStatementStatusConfirmed, saved.Status)
	require.Equal(t, int64(300), saved.Payable)

	_, err = RegisterTenantPaymentAndInvoice(tenant.Id, statement.Id, 301, "bank_transfer", "INV-OVERPAY", "issued", owner.Id, "127.0.0.1")
	require.Error(t, err)
	var payments int64
	require.NoError(t, model.DB.Model(&model.PaymentRecord{}).Where("statement_id = ?", statement.Id).Count(&payments).Error)
	require.Equal(t, int64(0), payments)

	saved, err = RegisterTenantPaymentAndInvoice(tenant.Id, statement.Id, 100, "bank_transfer", "INV-PARTIAL", "issued", owner.Id, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, model.BillingStatementStatusInvoiced, saved.Status)
	_, err = RegisterTenantPaymentAndInvoice(tenant.Id, statement.Id, 250, "bank_transfer", "INV-OVERPAY-2", "issued", owner.Id, "127.0.0.1")
	require.Error(t, err)
	saved, err = RegisterTenantPaymentAndInvoice(tenant.Id, statement.Id, 200, "bank_transfer", "INV-FINAL", "issued", owner.Id, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, model.BillingStatementStatusPaid, saved.Status)
	account, err := model.GetCreditAccountByTenantId(tenant.Id)
	require.NoError(t, err)
	require.Equal(t, int64(0), account.BilledUnpaidAmount)
	require.Equal(t, int64(100), account.UnbilledAmount)
	require.Equal(t, int64(900), account.AvailableCredit)
}

func TestTenantCreditReservationLifecycle(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)

	owner := tenantTestUser(t, 1, "reservation-owner", 0, common.RoleCommonUser)
	endUser := tenantTestUser(t, 2, "reservation-user", 0, common.RoleCommonUser)
	tenant := tenantTestCreatePostpaidTenant(t, owner.Id, "Reservation Tenant", 1000)
	policy := tenantTestCreatePolicy(t, tenant.Id, "gpt-reserve")
	app, customer := tenantTestCreateAppAndCustomer(t, tenant.Id, endUser.Id, "reserve")
	token, _, err := CreateTenantAPIKey(tenant.Id, TenantAPIKeyInput{
		UserId:         endUser.Id,
		Name:           "reserve-key",
		AppId:          app.Id,
		EndCustomerId:  customer.Id,
		ModelPolicyId:  policy.Id,
		UnlimitedQuota: true,
	}, owner.Id, "127.0.0.1")
	require.NoError(t, err)
	_, channel := tenantTestSupply(t, "gpt-reserve")

	ctx := tenantTestContext("reservation-success")
	relayInfo := &relaycommon.RelayInfo{
		UserId:            endUser.Id,
		TokenId:           token.Id,
		TenantId:          tenant.Id,
		AppId:             app.Id,
		EndCustomerId:     customer.Id,
		ModelPolicyId:     policy.Id,
		TenantBillingMode: model.BillingModePostpaid,
		RequestId:         "reservation-success",
		OriginModelName:   "gpt-reserve",
		StartTime:         time.Now().Add(-time.Second),
		ChannelMeta:       &relaycommon.ChannelMeta{ChannelId: channel.Id, ChannelBaseUrl: "upstream"},
	}
	apiErr := PreConsumeBilling(ctx, 200, relayInfo)
	require.Nil(t, apiErr)
	account, err := model.GetCreditAccountByTenantId(tenant.Id)
	require.NoError(t, err)
	require.Equal(t, int64(200), account.ReservedAmount)
	require.Equal(t, int64(800), account.AvailableCredit)

	require.NoError(t, SettleBilling(ctx, relayInfo, 150))
	account, err = model.GetCreditAccountByTenantId(tenant.Id)
	require.NoError(t, err)
	require.Equal(t, int64(150), account.ReservedAmount)
	require.Equal(t, int64(850), account.AvailableCredit)

	_, err = RecordUsage(ctx, relayInfo, &dto.Usage{PromptTokens: 40, CompletionTokens: 10}, 150)
	require.NoError(t, err)
	account, err = model.GetCreditAccountByTenantId(tenant.Id)
	require.NoError(t, err)
	require.Equal(t, int64(0), account.ReservedAmount)
	require.Equal(t, int64(150), account.UnbilledAmount)
	require.Equal(t, int64(850), account.AvailableCredit)

	refundCtx := tenantTestContext("reservation-refund")
	refundRelay := &relaycommon.RelayInfo{TenantId: tenant.Id, TenantBillingMode: model.BillingModePostpaid, OriginModelName: "gpt-reserve", RequestId: "reservation-refund"}
	apiErr = PreConsumeBilling(refundCtx, 100, refundRelay)
	require.Nil(t, apiErr)
	account, err = model.GetCreditAccountByTenantId(tenant.Id)
	require.NoError(t, err)
	require.Equal(t, int64(100), account.ReservedAmount)
	require.Equal(t, int64(750), account.AvailableCredit)
	refundRelay.Billing.Refund(refundCtx)
	require.Eventually(t, func() bool {
		account, err := model.GetCreditAccountByTenantId(tenant.Id)
		return err == nil && account.ReservedAmount == 0 && account.AvailableCredit == 850
	}, time.Second, 10*time.Millisecond)
}

func TestMarkOverdueTenantStatementsMovesOnlyRemainingBalance(t *testing.T) {
	truncate(t)

	owner := tenantTestUser(t, 1, "overdue-owner", 0, common.RoleCommonUser)
	tenant := tenantTestCreatePostpaidTenant(t, owner.Id, "Overdue Tenant", 1000)
	now := common.GetTimestamp()
	_, err := insertUsageLedgerWithTenantCredit(&model.UsageLedger{
		RequestId:     "overdue-ledger",
		TenantId:      tenant.Id,
		ModelName:     "gpt-overdue",
		SellQuota:     300,
		PostpaidQuota: 300,
		BillingMode:   model.BillingModePostpaid,
		BillingPeriod: billingPeriodFromUnix(now),
		Status:        "success",
		CreatedAt:     now,
	})
	require.NoError(t, err)
	statement, err := GenerateTenantBillingStatement(tenant.Id, now-60, now+60, 0, owner.Id, "127.0.0.1")
	require.NoError(t, err)
	statement, err = ConfirmTenantBillingStatement(tenant.Id, statement.Id, owner.Id, "127.0.0.1")
	require.NoError(t, err)
	statement, err = RegisterTenantPaymentAndInvoice(tenant.Id, statement.Id, 100, "bank_transfer", "INV-PARTIAL-OVERDUE", "issued", owner.Id, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, model.BillingStatementStatusInvoiced, statement.Status)
	require.NoError(t, model.DB.Model(&model.BillingStatement{}).Where("id = ?", statement.Id).Update("due_date", now-1).Error)

	require.NoError(t, MarkOverdueTenantStatements(now))
	statement, err = getTenantStatement(tenant.Id, statement.Id)
	require.NoError(t, err)
	require.Equal(t, model.BillingStatementStatusOverdue, statement.Status)
	account, err := model.GetCreditAccountByTenantId(tenant.Id)
	require.NoError(t, err)
	require.Equal(t, int64(0), account.BilledUnpaidAmount)
	require.Equal(t, int64(200), account.OverdueAmount)
	require.Equal(t, int64(800), account.AvailableCredit)

	statement, err = RegisterTenantPaymentAndInvoice(tenant.Id, statement.Id, 50, "bank_transfer", "INV-OVERDUE-PARTIAL", "issued", owner.Id, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, model.BillingStatementStatusOverdue, statement.Status)
	account, err = model.GetCreditAccountByTenantId(tenant.Id)
	require.NoError(t, err)
	require.Equal(t, int64(150), account.OverdueAmount)
	require.Equal(t, int64(850), account.AvailableCredit)

	statement, err = RegisterTenantPaymentAndInvoice(tenant.Id, statement.Id, 150, "bank_transfer", "INV-OVERDUE-FINAL", "issued", owner.Id, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, model.BillingStatementStatusPaid, statement.Status)
	account, err = model.GetCreditAccountByTenantId(tenant.Id)
	require.NoError(t, err)
	require.Equal(t, int64(0), account.OverdueAmount)
	require.Equal(t, int64(1000), account.AvailableCredit)
}

func TestTenantIsolationAndModelPolicy(t *testing.T) {
	truncate(t)

	tenantTestUser(t, 1, "platform-root", 0, common.RoleRootUser)
	user := tenantTestUser(t, 2, "tenant-user", 0, common.RoleCommonUser)
	tenantA := tenantTestCreatePostpaidTenant(t, user.Id, "Tenant A", 1000)
	tenantB := tenantTestCreatePostpaidTenant(t, 0, "Tenant B", 1000)
	policyA := tenantTestCreatePolicy(t, tenantA.Id, "gpt-a")
	policyB := tenantTestCreatePolicy(t, tenantB.Id, "gpt-b")

	require.NoError(t, EnsureTenantModelAllowed(tenantA.Id, policyA.Id, "gpt-a"))
	require.NoError(t, EnsureTenantModelAllowed(tenantA.Id, policyA.Id, "gpt-a-alias"))
	require.Error(t, EnsureTenantModelAllowed(tenantA.Id, policyA.Id, "gpt-b"))
	require.Error(t, EnsureTenantModelAllowed(tenantA.Id, policyB.Id, "gpt-b"))

	member, err := model.GetTenantMember(tenantA.Id, user.Id)
	require.NoError(t, err)
	require.Equal(t, model.TenantRoleOwner, member.Role)
	_, err = model.GetTenantMember(tenantB.Id, user.Id)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestTenantMixedBillingDecoratesOnlyOverageAsPostpaid(t *testing.T) {
	truncate(t)

	owner := tenantTestUser(t, 1, "mixed-owner", 0, common.RoleCommonUser)
	tenant, err := CreateTenantWithDefaults(TenantCreateInput{
		Name:         "Mixed Enterprise",
		OwnerUserId:  owner.Id,
		BillingMode:  model.BillingModeMixed,
		CreditLimit:  1000,
		StatementDay: 1,
		PaymentTerms: 30,
	}, owner.Id, "127.0.0.1")
	require.NoError(t, err)

	ledger := &model.UsageLedger{RequestId: "mixed-overage", ModelName: "gpt-mixed", SellQuota: 350, CreatedAt: common.GetTimestamp()}
	DecorateTenantLedger(ledger, &relaycommon.RelayInfo{
		TenantId:              tenant.Id,
		TenantBillingMode:     model.BillingModeMixed,
		FinalPreConsumedQuota: 120,
		OriginModelName:       "gpt-mixed",
	})
	require.Equal(t, model.BillingModeMixed, ledger.BillingMode)
	require.Equal(t, 230, ledger.PostpaidQuota)
	require.NotEmpty(t, ledger.BillingPeriod)
	require.NoError(t, ApplyTenantLedgerCredit(ledger))

	account, err := model.GetCreditAccountByTenantId(tenant.Id)
	require.NoError(t, err)
	require.Equal(t, int64(230), account.UnbilledAmount)
	require.Equal(t, int64(770), account.AvailableCredit)
}

func TestTenantCreditPrecheckBlocksOnlyWhenOverPolicyBlocks(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)

	owner := tenantTestUser(t, 1, "credit-owner", 0, common.RoleCommonUser)
	tenant := tenantTestCreatePostpaidTenant(t, owner.Id, "Credit Block Tenant", 50)
	ctx := tenantTestContext("credit-precheck")
	relayInfo := &relaycommon.RelayInfo{TenantId: tenant.Id, OriginModelName: "gpt-credit"}

	apiErr := PreConsumeBilling(ctx, 100, relayInfo)
	require.NotNil(t, apiErr)
	require.Equal(t, types.ErrorCodeInsufficientUserQuota, apiErr.GetErrorCode())

	for _, policy := range []string{"warn", "allow"} {
		require.NoError(t, SetTenantBillingConfig(tenant.Id, &model.BillingConfig{
			BillingMode:      model.BillingModePostpaid,
			CreditLimit:      50,
			OverCreditPolicy: policy,
		}, owner.Id, "127.0.0.1"))
		relayInfo = &relaycommon.RelayInfo{TenantId: tenant.Id, OriginModelName: "gpt-credit"}
		apiErr = PreConsumeBilling(ctx, 100, relayInfo)
		require.Nil(t, apiErr, policy)
		require.Equal(t, BillingSourceTenantPostpaid, relayInfo.BillingSource, policy)
	}
}

func TestNormalizeTenantStatusAllowsOnlyKnownStatuses(t *testing.T) {
	status, err := NormalizeTenantStatus(" suspended ")
	require.NoError(t, err)
	require.Equal(t, model.TenantStatusSuspended, status)

	status, err = NormalizeTenantStatus("DISABLED")
	require.NoError(t, err)
	require.Equal(t, model.TenantStatusDisabled, status)

	_, err = NormalizeTenantStatus("archived")
	require.Error(t, err)
}

func TestFrontChannelAndTenantRoutingPreferenceAudited(t *testing.T) {
	truncate(t)

	platform := tenantTestUser(t, 1, "routing-platform", 0, common.RoleRootUser)
	owner := tenantTestUser(t, 2, "routing-owner", 0, common.RoleCommonUser)
	tenant := tenantTestCreatePostpaidTenant(t, owner.Id, "Routing Tenant", 1000)

	frontChannel := &model.FrontChannel{
		Name:            "official-site",
		Type:            "landing_page",
		Domain:          "example.com",
		LandingPage:     "/enterprise",
		Owner:           "growth",
		PricingPolicyId: 9,
		Utm:             `{"source":"official"}`,
	}
	require.NoError(t, CreateFrontChannel(frontChannel, platform.Id, "127.0.0.1"))
	require.NotZero(t, frontChannel.Id)
	require.Equal(t, model.FrontChannelStatusActive, frontChannel.Status)

	pref := &model.TenantRoutingPreference{
		TenantId:            tenant.Id,
		ModelName:           "gpt-test",
		SlaTier:             "premium",
		PreferredSupplierId: 11,
		PreferredChannelId:  22,
		Reason:              "prefer low latency",
	}
	require.NoError(t, CreateTenantRoutingPreference(pref, owner.Id, "127.0.0.1"))
	require.Equal(t, model.TenantRoutingStatusDraft, pref.Status)

	applied, err := ReviewTenantRoutingPreference(tenant.Id, pref.Id, model.TenantRoutingStatusApplied, platform.Id, "approved", "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, model.TenantRoutingStatusApplied, applied.Status)
	require.NotZero(t, applied.AppliedAt)

	rolledBack, err := ReviewTenantRoutingPreference(tenant.Id, pref.Id, model.TenantRoutingStatusRolledBack, platform.Id, "rollback", "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, model.TenantRoutingStatusRolledBack, rolledBack.Status)

	var platformAudits int64
	require.NoError(t, model.DB.Model(&model.AuditLog{}).Where("scope_type = ? AND action = ?", model.ScopePlatform, "front_channel.create").Count(&platformAudits).Error)
	require.Equal(t, int64(1), platformAudits)
	var tenantAudits int64
	require.NoError(t, model.DB.Model(&model.AuditLog{}).Where("scope_type = ? AND scope_id = ? AND action LIKE ?", model.ScopeTenant, tenant.Id, "tenant.routing_preference.%").Count(&tenantAudits).Error)
	require.Equal(t, int64(3), tenantAudits)
}

func TestTenantAppliedRoutingPreferenceSelectsUsableChannel(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)

	owner := tenantTestUser(t, 1, "route-owner", 0, common.RoleCommonUser)
	tenant := tenantTestCreatePostpaidTenant(t, owner.Id, "Route Tenant", 1000)
	_, channel := tenantTestSupply(t, "gpt-route")
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     "default",
		Model:     "gpt-route",
		ChannelId: channel.Id,
		Enabled:   true,
	}).Error)
	pref := &model.TenantRoutingPreference{
		TenantId:           tenant.Id,
		ModelName:          "gpt-route",
		PreferredChannelId: channel.Id,
		Status:             model.TenantRoutingStatusApplied,
	}
	require.NoError(t, serviceCreateAppliedTenantRoutingPreferenceForTest(pref))

	ctx := tenantTestContext("route-pref")
	selected, group, found := GetTenantAppliedRoutingChannelForRequest(ctx, tenant.Id, "gpt-route", "default", "/v1/chat/completions")
	require.True(t, found)
	require.Equal(t, channel.Id, selected.Id)
	require.Equal(t, "default", group)

	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("status", common.ChannelStatusManuallyDisabled).Error)
	selected, _, found = GetTenantAppliedRoutingChannelForRequest(ctx, tenant.Id, "gpt-route", "default", "/v1/chat/completions")
	require.False(t, found)
	require.Nil(t, selected)
}

func serviceCreateAppliedTenantRoutingPreferenceForTest(pref *model.TenantRoutingPreference) error {
	pref.RequestedBy = 1
	pref.ApprovedBy = 1
	pref.AppliedAt = common.GetTimestamp()
	pref.CreatedAt = common.GetTimestamp()
	pref.UpdatedAt = common.GetTimestamp()
	return model.DB.Create(pref).Error
}

func TestTenantConcurrentMultiBMultiCAccountingIsolation(t *testing.T) {
	truncate(t)
	gin.SetMode(gin.TestMode)

	tenantTestUser(t, 1, "platform-root", 0, common.RoleRootUser)
	_, channel := tenantTestSupply(t, "gpt-concurrent")

	type callCase struct {
		tenantID      int
		appID         int
		endCustomerID int
		userID        int
		tokenID       int
		policyID      int
		requestID     string
		sellQuota     int
	}
	var cases []callCase
	expectedByTenant := map[int]int64{}

	for b := 0; b < 3; b++ {
		ownerID := 10 + b
		owner := tenantTestUser(t, ownerID, fmt.Sprintf("owner-%d", b), 0, common.RoleCommonUser)
		tenant := tenantTestCreatePostpaidTenant(t, owner.Id, fmt.Sprintf("Tenant-%d", b), 100000)
		policy := tenantTestCreatePolicy(t, tenant.Id, "gpt-concurrent")
		for c := 0; c < 3; c++ {
			userID := 100 + b*10 + c
			endUser := tenantTestUser(t, userID, fmt.Sprintf("tenant-%d-c-%d", b, c), 0, common.RoleCommonUser)
			app, customer := tenantTestCreateAppAndCustomer(t, tenant.Id, endUser.Id, fmt.Sprintf("%d-%d", b, c))
			for k := 0; k < 2; k++ {
				token, _, err := CreateTenantAPIKey(tenant.Id, TenantAPIKeyInput{
					UserId:         endUser.Id,
					Name:           fmt.Sprintf("key-%d-%d-%d", b, c, k),
					AppId:          app.Id,
					EndCustomerId:  customer.Id,
					ModelPolicyId:  policy.Id,
					UnlimitedQuota: true,
				}, owner.Id, "127.0.0.1")
				require.NoError(t, err)
				sellQuota := 100 + b*20 + c*5 + k
				cases = append(cases, callCase{
					tenantID:      tenant.Id,
					appID:         app.Id,
					endCustomerID: customer.Id,
					userID:        endUser.Id,
					tokenID:       token.Id,
					policyID:      policy.Id,
					requestID:     fmt.Sprintf("tenant-%d-c-%d-key-%d", b, c, k),
					sellQuota:     sellQuota,
				})
				expectedByTenant[tenant.Id] += int64(sellQuota)
			}
		}
	}

	errCh := make(chan error, len(cases))
	var wg sync.WaitGroup
	for _, item := range cases {
		item := item
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := tenantTestContext(item.requestID)
			relayInfo := &relaycommon.RelayInfo{
				UserId:            item.userID,
				TokenId:           item.tokenID,
				TenantId:          item.tenantID,
				AppId:             item.appID,
				EndCustomerId:     item.endCustomerID,
				ModelPolicyId:     item.policyID,
				TenantBillingMode: model.BillingModePostpaid,
				RequestId:         item.requestID,
				OriginModelName:   "gpt-concurrent",
				StartTime:         time.Now().Add(-time.Millisecond),
				ChannelMeta:       &relaycommon.ChannelMeta{ChannelId: channel.Id, ChannelBaseUrl: "upstream"},
			}
			if apiErr := PreConsumeBilling(ctx, item.sellQuota, relayInfo); apiErr != nil {
				errCh <- apiErr
				return
			}
			if err := SettleBilling(ctx, relayInfo, item.sellQuota); err != nil {
				errCh <- err
				return
			}
			_, err := RecordUsage(ctx, relayInfo, &dto.Usage{PromptTokens: 50, CompletionTokens: 10}, item.sellQuota)
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	var totalLedgers int64
	require.NoError(t, model.DB.Model(&model.UsageLedger{}).Count(&totalLedgers).Error)
	require.Equal(t, int64(len(cases)), totalLedgers)
	for tenantID, expected := range expectedByTenant {
		summary, err := GetTenantUsageSummary(tenantID, 0, 0)
		require.NoError(t, err)
		require.Equal(t, int64(6), summary.RequestCount)
		require.Equal(t, expected, summary.SellQuota)
		require.Equal(t, expected, summary.PostpaidQuota)
		account, err := model.GetCreditAccountByTenantId(tenantID)
		require.NoError(t, err)
		require.Equal(t, expected, account.UnbilledAmount)
		require.Equal(t, int64(100000)-expected, account.AvailableCredit)
	}
}
