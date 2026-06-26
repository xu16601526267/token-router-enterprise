package service

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	BillingSourceTenantPostpaid = "tenant_postpaid"
	BillingSourceTenantMixed    = "tenant_mixed"
)

type TenantCreateInput struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Industry     string `json:"industry"`
	OwnerUserId  int    `json:"owner_user_id"`
	BrandConfig  string `json:"brand_config"`
	Domain       string `json:"domain"`
	ContractNo   string `json:"contract_no"`
	BillingMode  string `json:"billing_mode"`
	CreditLimit  int64  `json:"credit_limit"`
	StatementDay int    `json:"statement_day"`
	PaymentTerms int    `json:"payment_terms"`
}

type TenantAPIKeyInput struct {
	UserId             int     `json:"user_id"`
	Name               string  `json:"name"`
	AppId              int     `json:"app_id"`
	EndCustomerId      int     `json:"end_customer_id"`
	OwnerScope         string  `json:"owner_scope"`
	ModelPolicyId      int     `json:"model_policy_id"`
	ExpiredTime        int64   `json:"expired_time"`
	RemainQuota        int     `json:"remain_quota"`
	UnlimitedQuota     bool    `json:"unlimited_quota"`
	ModelLimitsEnabled bool    `json:"model_limits_enabled"`
	ModelLimits        string  `json:"model_limits"`
	AllowIps           *string `json:"allow_ips"`
	Group              string  `json:"group"`
	RateLimit          string  `json:"rate_limit"`
}

type TenantUsageSummary struct {
	TenantId      int   `json:"tenant_id"`
	RequestCount  int64 `json:"request_count"`
	SellQuota     int64 `json:"sell_quota"`
	CostQuota     int64 `json:"cost_quota"`
	PostpaidQuota int64 `json:"postpaid_quota"`
}

func normalizeBillingMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case model.BillingModePostpaid:
		return model.BillingModePostpaid
	case model.BillingModeMixed:
		return model.BillingModeMixed
	default:
		return model.BillingModePrepaid
	}
}

func CreateTenantWithDefaults(input TenantCreateInput, actorId int, ip string) (*model.Tenant, error) {
	tenant := &model.Tenant{
		Name:        input.Name,
		Type:        input.Type,
		Industry:    input.Industry,
		OwnerUserId: input.OwnerUserId,
		BrandConfig: input.BrandConfig,
		Domain:      input.Domain,
		ContractNo:  input.ContractNo,
	}
	if err := tenant.Insert(); err != nil {
		return nil, err
	}
	if input.OwnerUserId > 0 {
		member := &model.TenantMember{
			TenantId: tenant.Id,
			UserId:   input.OwnerUserId,
			Role:     model.TenantRoleOwner,
			Status:   model.TenantMemberStatusActive,
		}
		if err := member.Upsert(); err != nil {
			return nil, err
		}
	}
	mode := normalizeBillingMode(input.BillingMode)
	config := &model.BillingConfig{
		TenantId:     tenant.Id,
		BillingMode:  mode,
		CreditLimit:  input.CreditLimit,
		StatementDay: input.StatementDay,
		PaymentTerms: input.PaymentTerms,
	}
	if err := SetTenantBillingConfig(tenant.Id, config, actorId, ip); err != nil {
		return nil, err
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{
		ScopeType: model.ScopePlatform,
		ActorId:   actorId,
		Action:    "tenant.create",
		Target:    fmt.Sprintf("tenant:%d", tenant.Id),
		After:     common.GetJsonString(tenant),
		Ip:        ip,
	})
	return tenant, nil
}

func SetTenantBillingConfig(tenantId int, config *model.BillingConfig, actorId int, ip string) error {
	if tenantId <= 0 {
		return errors.New("tenant_id is required")
	}
	if _, err := model.GetTenantById(tenantId); err != nil {
		return err
	}
	config.TenantId = tenantId
	config.BillingMode = normalizeBillingMode(config.BillingMode)
	if err := model.UpsertBillingConfig(config); err != nil {
		return err
	}
	account := &model.CreditAccount{TenantId: tenantId, CreditLimit: config.CreditLimit, Status: model.CreditAccountStatusActive}
	if existing, err := model.GetCreditAccountByTenantId(tenantId); err == nil {
		account.UnbilledAmount = existing.UnbilledAmount
		account.BilledUnpaidAmount = existing.BilledUnpaidAmount
		account.OverdueAmount = existing.OverdueAmount
		account.Status = existing.Status
	}
	if err := model.UpsertCreditAccount(account); err != nil {
		return err
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{
		ScopeType: model.ScopeTenant,
		ScopeId:   tenantId,
		ActorId:   actorId,
		Action:    "tenant.billing_config.update",
		Target:    fmt.Sprintf("tenant:%d", tenantId),
		After:     common.GetJsonString(config),
		Ip:        ip,
	})
	return nil
}

func UpsertTenantModelPolicy(policy *model.TenantModelPolicy, actorId int, ip string) error {
	if policy == nil {
		return errors.New("tenant model policy is nil")
	}
	if policy.TenantId <= 0 {
		return errors.New("tenant_id is required")
	}
	policy.ModelName = strings.TrimSpace(policy.ModelName)
	policy.Alias = strings.TrimSpace(policy.Alias)
	if policy.ModelName == "" && policy.ModelId <= 0 {
		return errors.New("model_name or model_id is required")
	}
	now := common.GetTimestamp()
	if policy.CreatedAt == 0 {
		policy.CreatedAt = now
	}
	policy.UpdatedAt = now
	return model.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "model_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"model_id", "visible", "price_plan_id", "rate_limit", "alias", "enabled", "updated_at"}),
	}).Create(policy).Error
}

func EnsureTenantModelAllowed(tenantId int, modelPolicyId int, modelName string) error {
	if tenantId <= 0 {
		return nil
	}
	modelName = strings.TrimSpace(modelName)
	query := model.DB.Model(&model.TenantModelPolicy{}).Where("tenant_id = ? AND enabled = ? AND visible = ?", tenantId, true, true)
	if modelPolicyId > 0 {
		query = query.Where("id = ?", modelPolicyId)
	}
	var policies []model.TenantModelPolicy
	if err := query.Find(&policies).Error; err != nil {
		return err
	}
	if len(policies) == 0 {
		return fmt.Errorf("tenant %d has no enabled model policy", tenantId)
	}
	for _, policy := range policies {
		if policy.ModelName == "" || policy.ModelName == modelName || policy.Alias == modelName {
			return nil
		}
	}
	return fmt.Errorf("tenant %d is not authorized to use model %s", tenantId, modelName)
}

func CreateTenantEndCustomer(customer *model.TenantEndCustomer, actorId int, ip string) error {
	if customer == nil {
		return errors.New("tenant end customer is nil")
	}
	if customer.TenantId <= 0 || customer.UserId <= 0 {
		return errors.New("tenant_id and user_id are required")
	}
	customer.CustomerType = strings.TrimSpace(customer.CustomerType)
	customer.Status = strings.TrimSpace(customer.Status)
	if customer.CustomerType == "" {
		customer.CustomerType = "user"
	}
	if customer.Status == "" {
		customer.Status = model.TenantStatusActive
	}
	now := common.GetTimestamp()
	if customer.CreatedAt == 0 {
		customer.CreatedAt = now
	}
	customer.UpdatedAt = now
	err := model.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"customer_type", "quota_policy_id", "status", "external_id", "updated_at"}),
	}).Create(customer).Error
	if err == nil {
		_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: customer.TenantId, ActorId: actorId, Action: "tenant.end_customer.upsert", Target: fmt.Sprintf("user:%d", customer.UserId), After: common.GetJsonString(customer), Ip: ip})
	}
	return err
}

func CreateTenantApp(app *model.TenantApp, actorId int, ip string) error {
	if app == nil {
		return errors.New("tenant app is nil")
	}
	if app.TenantId <= 0 || strings.TrimSpace(app.Name) == "" {
		return errors.New("tenant_id and name are required")
	}
	app.Name = strings.TrimSpace(app.Name)
	app.Env = strings.TrimSpace(app.Env)
	app.Status = strings.TrimSpace(app.Status)
	if app.Env == "" {
		app.Env = "prod"
	}
	if app.Status == "" {
		app.Status = model.TenantStatusActive
	}
	now := common.GetTimestamp()
	if app.CreatedAt == 0 {
		app.CreatedAt = now
	}
	app.UpdatedAt = now
	if err := model.DB.Create(app).Error; err != nil {
		return err
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: app.TenantId, ActorId: actorId, Action: "tenant.app.create", Target: fmt.Sprintf("app:%d", app.Id), After: common.GetJsonString(app), Ip: ip})
	return nil
}

func CreateTenantAPIKey(tenantId int, input TenantAPIKeyInput, actorId int, ip string) (*model.Token, string, error) {
	if tenantId <= 0 {
		return nil, "", errors.New("tenant_id is required")
	}
	if input.UserId <= 0 {
		return nil, "", errors.New("user_id is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, "", errors.New("token name is required")
	}
	if input.AppId > 0 {
		var app model.TenantApp
		if err := model.DB.Where("id = ? AND tenant_id = ?", input.AppId, tenantId).First(&app).Error; err != nil {
			return nil, "", errors.New("tenant app not found")
		}
	}
	if input.EndCustomerId > 0 {
		var customer model.TenantEndCustomer
		if err := model.DB.Where("id = ? AND tenant_id = ?", input.EndCustomerId, tenantId).First(&customer).Error; err != nil {
			return nil, "", errors.New("tenant end customer not found")
		}
	}
	if input.ModelPolicyId > 0 {
		var policy model.TenantModelPolicy
		if err := model.DB.Where("id = ? AND tenant_id = ? AND enabled = ?", input.ModelPolicyId, tenantId, true).First(&policy).Error; err != nil {
			return nil, "", errors.New("tenant model policy not found")
		}
	}
	key, err := common.GenerateKey()
	if err != nil {
		return nil, "", err
	}
	ownerScope := strings.TrimSpace(input.OwnerScope)
	if ownerScope == "" {
		if input.EndCustomerId > 0 {
			ownerScope = model.TokenOwnerScopeEndCustomer
		} else if input.AppId > 0 {
			ownerScope = model.TokenOwnerScopeApp
		} else {
			ownerScope = model.TokenOwnerScopeTenant
		}
	}
	if input.ExpiredTime == 0 {
		input.ExpiredTime = -1
	}
	token := &model.Token{
		UserId:             input.UserId,
		Key:                key,
		Name:               strings.TrimSpace(input.Name),
		Status:             common.TokenStatusEnabled,
		CreatedTime:        common.GetTimestamp(),
		AccessedTime:       common.GetTimestamp(),
		ExpiredTime:        input.ExpiredTime,
		RemainQuota:        input.RemainQuota,
		UnlimitedQuota:     input.UnlimitedQuota,
		ModelLimitsEnabled: input.ModelLimitsEnabled,
		ModelLimits:        input.ModelLimits,
		AllowIps:           input.AllowIps,
		Group:              input.Group,
		TenantId:           tenantId,
		AppId:              input.AppId,
		EndCustomerId:      input.EndCustomerId,
		OwnerScope:         ownerScope,
		ModelPolicyId:      input.ModelPolicyId,
		RateLimit:          strings.TrimSpace(input.RateLimit),
		RotationStatus:     "active",
	}
	if token.Group == "" {
		user, err := model.GetUserById(input.UserId, false)
		if err == nil {
			token.Group = user.Group
		}
	}
	if err := token.Insert(); err != nil {
		return nil, "", err
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: tenantId, ActorId: actorId, Action: "tenant.api_key.create", Target: fmt.Sprintf("token:%d", token.Id), After: common.GetJsonString(token), Ip: ip})
	return token, "sk-" + key, nil
}

func TenantBillingMode(tenantId int) string {
	if tenantId <= 0 {
		return model.BillingModePrepaid
	}
	config, err := model.GetBillingConfigByTenantId(tenantId)
	if err != nil || strings.TrimSpace(config.BillingMode) == "" {
		return model.BillingModePrepaid
	}
	return normalizeBillingMode(config.BillingMode)
}

func billingPeriodFromUnix(ts int64) string {
	if ts <= 0 {
		ts = common.GetTimestamp()
	}
	return time.Unix(ts, 0).Format("2006-01")
}

func TenantPostpaidQuota(mode string, sellQuota int, prepaidConsumed int) int {
	if sellQuota <= 0 {
		return 0
	}
	switch normalizeBillingMode(mode) {
	case model.BillingModePostpaid:
		return sellQuota
	case model.BillingModeMixed:
		if sellQuota > prepaidConsumed {
			return sellQuota - prepaidConsumed
		}
	}
	return 0
}

func EnsureTenantCreditAvailable(tenantId int, quota int) error {
	if tenantId <= 0 || quota <= 0 {
		return nil
	}
	config, err := model.GetBillingConfigByTenantId(tenantId)
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(config.OverCreditPolicy)) {
	case "allow", "alert", "manual_review":
		return nil
	}
	account, err := model.GetCreditAccountByTenantId(tenantId)
	if err != nil {
		return err
	}
	if account.Status != model.CreditAccountStatusActive {
		return fmt.Errorf("tenant %d credit account is %s", tenantId, account.Status)
	}
	if int64(quota) > account.AvailableCredit {
		return fmt.Errorf("tenant %d credit quota insufficient, available=%d, required=%d", tenantId, account.AvailableCredit, quota)
	}
	return nil
}

func ApplyTenantLedgerCredit(ledger *model.UsageLedger) error {
	if ledger == nil || ledger.TenantId <= 0 || ledger.PostpaidQuota <= 0 {
		return nil
	}
	return retryTenantDBWrite(func() error {
		return model.DB.Transaction(func(tx *gorm.DB) error {
			return applyTenantLedgerCreditTx(tx, ledger)
		})
	})
}

func applyTenantLedgerCreditTx(tx *gorm.DB, ledger *model.UsageLedger) error {
	if ledger == nil || ledger.TenantId <= 0 || ledger.PostpaidQuota <= 0 {
		return nil
	}
	if tx == nil {
		return errors.New("tenant credit db is required")
	}
	var account model.CreditAccount
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("tenant_id = ?", ledger.TenantId).First(&account).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		account = model.CreditAccount{TenantId: ledger.TenantId, Status: model.CreditAccountStatusActive}
		var config model.BillingConfig
		if configErr := tx.Where("tenant_id = ?", ledger.TenantId).First(&config).Error; configErr != nil {
			if !errors.Is(configErr, gorm.ErrRecordNotFound) {
				return configErr
			}
		} else {
			account.CreditLimit = config.CreditLimit
		}
		if err := tx.Create(&account).Error; err != nil {
			return err
		}
	}
	now := common.GetTimestamp()
	result := tx.Model(&model.CreditAccount{}).Where("tenant_id = ?", ledger.TenantId).Updates(map[string]interface{}{
		"unbilled_amount": gorm.Expr("unbilled_amount + ?", ledger.PostpaidQuota),
		"available_credit": gorm.Expr(
			"credit_limit - (unbilled_amount + ? + billed_unpaid_amount + overdue_amount)",
			ledger.PostpaidQuota,
		),
		"updated_at": now,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("tenant %d credit account update affected no rows", ledger.TenantId)
	}
	return nil
}

func retryTenantDBWrite(operation func() error) error {
	if operation == nil {
		return nil
	}
	var err error
	for attempt := 0; attempt < 8; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}
		if !isRetryableTenantDBWriteError(err) {
			return err
		}
		time.Sleep(time.Duration(10*(1<<attempt)) * time.Millisecond)
	}
	return err
}

func isRetryableTenantDBWriteError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database table is locked") ||
		strings.Contains(message, "sqlite_busy")
}

func GenerateTenantBillingStatement(tenantId int, periodStart int64, periodEnd int64, adjustment int64, actorId int, ip string) (*model.BillingStatement, error) {
	if tenantId <= 0 {
		return nil, errors.New("tenant_id is required")
	}
	if periodStart <= 0 || periodEnd <= 0 || periodEnd < periodStart {
		return nil, errors.New("invalid billing period")
	}
	var aggregate struct {
		Amount int64
	}
	if err := model.DB.Model(&model.UsageLedger{}).
		Select("COALESCE(SUM(postpaid_quota), 0) AS amount").
		Where("tenant_id = ? AND created_at >= ? AND created_at <= ? AND status = ?", tenantId, periodStart, periodEnd, "success").
		Scan(&aggregate).Error; err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	config, _ := model.GetBillingConfigByTenantId(tenantId)
	paymentTerms := 30
	if config != nil && config.PaymentTerms > 0 {
		paymentTerms = config.PaymentTerms
	}
	statement := &model.BillingStatement{
		TenantId:    tenantId,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Amount:      aggregate.Amount,
		Adjustment:  adjustment,
		Payable:     aggregate.Amount + adjustment,
		Status:      model.BillingStatementStatusDraft,
		DueDate:     now + int64(paymentTerms*24*60*60),
		GeneratedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if statement.Payable < 0 {
		statement.Payable = 0
	}
	err := model.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "period_start"}, {Name: "period_end"}},
		DoUpdates: clause.AssignmentColumns([]string{"amount", "adjustment", "payable", "status", "due_date", "generated_at", "updated_at"}),
	}).Create(statement).Error
	if err != nil {
		return nil, err
	}
	if err := model.DB.Where("tenant_id = ? AND period_start = ? AND period_end = ?", tenantId, periodStart, periodEnd).First(statement).Error; err != nil {
		return nil, err
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: tenantId, ActorId: actorId, Action: "tenant.billing_statement.generate", Target: fmt.Sprintf("statement:%d", statement.Id), After: common.GetJsonString(statement), Ip: ip})
	return statement, nil
}

func ConfirmTenantBillingStatement(tenantId int, statementId int, actorId int, ip string) (*model.BillingStatement, error) {
	statement, err := getTenantStatement(tenantId, statementId)
	if err != nil {
		return nil, err
	}
	if statement.Status != model.BillingStatementStatusDraft && statement.Status != model.BillingStatementStatusAdjusted {
		return nil, fmt.Errorf("statement status %s cannot be confirmed", statement.Status)
	}
	now := common.GetTimestamp()
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(statement).Updates(map[string]interface{}{"status": model.BillingStatementStatusConfirmed, "confirmed_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		var account model.CreditAccount
		if err := tx.Where("tenant_id = ?", tenantId).First(&account).Error; err != nil {
			return err
		}
		account.UnbilledAmount -= statement.Payable
		if account.UnbilledAmount < 0 {
			account.UnbilledAmount = 0
		}
		account.BilledUnpaidAmount += statement.Payable
		account.Recalculate()
		return tx.Model(&account).Select("unbilled_amount", "billed_unpaid_amount", "available_credit", "updated_at").Updates(map[string]interface{}{
			"unbilled_amount":      account.UnbilledAmount,
			"billed_unpaid_amount": account.BilledUnpaidAmount,
			"available_credit":     account.AvailableCredit,
			"updated_at":           now,
		}).Error
	}); err != nil {
		return nil, err
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: tenantId, ActorId: actorId, Action: "tenant.billing_statement.confirm", Target: fmt.Sprintf("statement:%d", statementId), Ip: ip})
	return getTenantStatement(tenantId, statementId)
}

func RegisterTenantPaymentAndInvoice(tenantId int, statementId int, amount int64, method string, invoiceNo string, invoiceStatus string, actorId int, ip string) (*model.BillingStatement, error) {
	statement, err := getTenantStatement(tenantId, statementId)
	if err != nil {
		return nil, err
	}
	if amount <= 0 {
		return nil, errors.New("payment amount must be positive")
	}
	now := common.GetTimestamp()
	if invoiceStatus == "" {
		invoiceStatus = "issued"
	}
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		payment := &model.PaymentRecord{StatementId: statementId, TenantId: tenantId, Amount: amount, Method: method, PaidAt: now, OperatorId: actorId, CreatedAt: now}
		if err := tx.Create(payment).Error; err != nil {
			return err
		}
		if invoiceNo != "" || invoiceStatus != "" {
			invoice := &model.Invoice{StatementId: statementId, TenantId: tenantId, Amount: amount, InvoiceNo: invoiceNo, InvoiceStatus: invoiceStatus, OperatorId: actorId, CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(invoice).Error; err != nil {
				return err
			}
		}
		var paid struct{ Amount int64 }
		if err := tx.Model(&model.PaymentRecord{}).Select("COALESCE(SUM(amount), 0) AS amount").Where("statement_id = ?", statementId).Scan(&paid).Error; err != nil {
			return err
		}
		status := model.BillingStatementStatusInvoiced
		if paid.Amount >= statement.Payable {
			status = model.BillingStatementStatusPaid
		}
		if err := tx.Model(statement).Updates(map[string]interface{}{"status": status, "updated_at": now}).Error; err != nil {
			return err
		}
		var account model.CreditAccount
		if err := tx.Where("tenant_id = ?", tenantId).First(&account).Error; err != nil {
			return err
		}
		account.BilledUnpaidAmount -= amount
		if account.BilledUnpaidAmount < 0 {
			account.BilledUnpaidAmount = 0
		}
		if account.OverdueAmount > 0 {
			if account.OverdueAmount >= amount {
				account.OverdueAmount -= amount
			} else {
				account.OverdueAmount = 0
			}
		}
		account.Recalculate()
		return tx.Model(&account).Select("billed_unpaid_amount", "overdue_amount", "available_credit", "updated_at").Updates(map[string]interface{}{
			"billed_unpaid_amount": account.BilledUnpaidAmount,
			"overdue_amount":       account.OverdueAmount,
			"available_credit":     account.AvailableCredit,
			"updated_at":           now,
		}).Error
	}); err != nil {
		return nil, err
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: tenantId, ActorId: actorId, Action: "tenant.billing_statement.payment", Target: fmt.Sprintf("statement:%d", statementId), Ip: ip})
	return getTenantStatement(tenantId, statementId)
}

func MarkOverdueTenantStatements(now int64) error {
	if now <= 0 {
		now = common.GetTimestamp()
	}
	var statements []model.BillingStatement
	if err := model.DB.Where("status IN ? AND due_date > 0 AND due_date < ?", []string{model.BillingStatementStatusConfirmed, model.BillingStatementStatusInvoiced}, now).Find(&statements).Error; err != nil {
		return err
	}
	for _, statement := range statements {
		if err := model.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&statement).Updates(map[string]interface{}{"status": model.BillingStatementStatusOverdue, "updated_at": now}).Error; err != nil {
				return err
			}
			var account model.CreditAccount
			if err := tx.Where("tenant_id = ?", statement.TenantId).First(&account).Error; err != nil {
				return err
			}
			account.OverdueAmount += statement.Payable
			account.Recalculate()
			return tx.Model(&account).Select("overdue_amount", "available_credit", "updated_at").Updates(map[string]interface{}{
				"overdue_amount":   account.OverdueAmount,
				"available_credit": account.AvailableCredit,
				"updated_at":       now,
			}).Error
		}); err != nil {
			return err
		}
	}
	return nil
}

func getTenantStatement(tenantId int, statementId int) (*model.BillingStatement, error) {
	if tenantId <= 0 || statementId <= 0 {
		return nil, errors.New("tenant_id and statement_id are required")
	}
	var statement model.BillingStatement
	if err := model.DB.Where("id = ? AND tenant_id = ?", statementId, tenantId).First(&statement).Error; err != nil {
		return nil, err
	}
	return &statement, nil
}

func GetTenantUsageSummary(tenantId int, startTime int64, endTime int64) (*TenantUsageSummary, error) {
	summary := &TenantUsageSummary{TenantId: tenantId}
	query := model.DB.Model(&model.UsageLedger{}).Where("tenant_id = ? AND status = ?", tenantId, "success")
	if startTime > 0 {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("created_at <= ?", endTime)
	}
	err := query.Select("COUNT(*) AS request_count, COALESCE(SUM(sell_quota), 0) AS sell_quota, COALESCE(SUM(cost_quota), 0) AS cost_quota, COALESCE(SUM(postpaid_quota), 0) AS postpaid_quota").Scan(summary).Error
	return summary, err
}

func CreateFrontChannel(channel *model.FrontChannel, actorId int, ip string) error {
	if channel == nil {
		return errors.New("front channel is nil")
	}
	channel.Name = strings.TrimSpace(channel.Name)
	if channel.Name == "" {
		return errors.New("front channel name is required")
	}
	channel.Status = strings.TrimSpace(channel.Status)
	if channel.Status == "" {
		channel.Status = model.FrontChannelStatusActive
	}
	now := common.GetTimestamp()
	if channel.CreatedAt == 0 {
		channel.CreatedAt = now
	}
	channel.UpdatedAt = now
	if err := model.DB.Create(channel).Error; err != nil {
		return err
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopePlatform, ActorId: actorId, Action: "front_channel.create", Target: fmt.Sprintf("front_channel:%d", channel.Id), After: common.GetJsonString(channel), Ip: ip})
	return nil
}

func CreateTenantRoutingPreference(pref *model.TenantRoutingPreference, actorId int, ip string) error {
	if pref == nil {
		return errors.New("tenant routing preference is nil")
	}
	if pref.TenantId <= 0 {
		return errors.New("tenant_id is required")
	}
	pref.Status = strings.TrimSpace(pref.Status)
	if pref.Status == "" {
		pref.Status = model.TenantRoutingStatusDraft
	}
	pref.RequestedBy = actorId
	now := common.GetTimestamp()
	if pref.CreatedAt == 0 {
		pref.CreatedAt = now
	}
	pref.UpdatedAt = now
	if err := model.DB.Create(pref).Error; err != nil {
		return err
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: pref.TenantId, ActorId: actorId, Action: "tenant.routing_preference.create", Target: fmt.Sprintf("tenant_routing:%d", pref.Id), After: common.GetJsonString(pref), Ip: ip})
	return nil
}

func ReviewTenantRoutingPreference(tenantId int, preferenceId int, status string, actorId int, note string, ip string) (*model.TenantRoutingPreference, error) {
	var pref model.TenantRoutingPreference
	if err := model.DB.Where("id = ? AND tenant_id = ?", preferenceId, tenantId).First(&pref).Error; err != nil {
		return nil, err
	}
	status = strings.TrimSpace(status)
	if status != model.TenantRoutingStatusApproved && status != model.TenantRoutingStatusApplied && status != model.TenantRoutingStatusDisabled && status != model.TenantRoutingStatusRolledBack {
		return nil, errors.New("invalid routing preference status")
	}
	updates := map[string]interface{}{"status": status, "approved_by": actorId, "operator_note": strings.TrimSpace(note), "updated_at": common.GetTimestamp()}
	if status == model.TenantRoutingStatusApplied {
		updates["applied_at"] = common.GetTimestamp()
	}
	if err := model.DB.Model(&pref).Updates(updates).Error; err != nil {
		return nil, err
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: tenantId, ActorId: actorId, Action: "tenant.routing_preference.review", Target: fmt.Sprintf("tenant_routing:%d", pref.Id), After: common.MapToJsonStr(updates), Ip: ip})
	if err := model.DB.First(&pref, pref.Id).Error; err != nil {
		return nil, err
	}
	return &pref, nil
}

func GetTenantAppliedRoutingChannelForRequest(c *gin.Context, tenantId int, modelName string, usingGroup string, requestPath string) (*model.Channel, string, bool) {
	if tenantId <= 0 || strings.TrimSpace(modelName) == "" {
		return nil, "", false
	}
	var prefs []model.TenantRoutingPreference
	if err := model.DB.
		Where("tenant_id = ? AND status = ? AND preferred_channel_id > 0", tenantId, model.TenantRoutingStatusApplied).
		Where("(model_name = '' OR model_name = ?)", modelName).
		Order(clause.Expr{SQL: "CASE WHEN model_name = ? THEN 0 ELSE 1 END", Vars: []interface{}{modelName}}).
		Order("id DESC").
		Limit(5).
		Find(&prefs).Error; err != nil {
		common.SysLog("failed to load tenant routing preferences: " + err.Error())
		return nil, "", false
	}
	for _, pref := range prefs {
		channel, err := model.CacheGetChannel(pref.PreferredChannelId)
		if err != nil || channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		if !model.IsChannelSupplierEnabled(channel) {
			continue
		}
		if pref.PreferredSupplierId > 0 && channel.SupplierId != pref.PreferredSupplierId {
			continue
		}
		if !tenantRoutingChannelSupportsRequestPath(channel, requestPath) {
			continue
		}
		if usingGroup == "auto" {
			userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
			for _, group := range GetUserAutoGroup(userGroup) {
				if tenantRoutingChannelEnabledForGroupModel(group, modelName, channel.Id) {
					c.Set("tenant_routing_preference_id", pref.Id)
					c.Set("tenant_routing_preference_channel_id", channel.Id)
					return channel, group, true
				}
			}
			continue
		}
		if tenantRoutingChannelEnabledForGroupModel(usingGroup, modelName, channel.Id) {
			c.Set("tenant_routing_preference_id", pref.Id)
			c.Set("tenant_routing_preference_channel_id", channel.Id)
			return channel, usingGroup, true
		}
	}
	return nil, "", false
}

func tenantRoutingChannelEnabledForGroupModel(group string, modelName string, channelID int) bool {
	if group == "" || modelName == "" || channelID <= 0 {
		return false
	}
	if tenantRoutingAbilityExists(group, modelName, channelID) {
		return true
	}
	normalized := ratio_setting.FormatMatchingModelName(modelName)
	return normalized != "" && normalized != modelName && tenantRoutingAbilityExists(group, normalized, channelID)
}

func tenantRoutingAbilityExists(group string, modelName string, channelID int) bool {
	var count int64
	err := model.DB.Model(&model.Ability{}).
		Where(tenantRoutingAbilityGroupColumn()+" = ? AND abilities.model = ? AND abilities.channel_id = ? AND abilities.enabled = ?", group, modelName, channelID, true).
		Count(&count).Error
	return err == nil && count > 0
}

func tenantRoutingAbilityGroupColumn() string {
	if common.UsingPostgreSQL {
		return `abilities."group"`
	}
	return "abilities.`group`"
}

func tenantRoutingChannelSupportsRequestPath(channel *model.Channel, requestPath string) bool {
	if channel == nil {
		return false
	}
	if channel.Type != constant.ChannelTypeAdvancedCustom {
		return true
	}
	config := channel.GetOtherSettings().AdvancedCustom
	return config != nil && config.SupportsPath(requestPath)
}

type TenantCreditBillingSession struct {
	relayInfo *relaycommon.RelayInfo
	prepaid   relaycommon.BillingSettler
	mode      string
}

func NewTenantPostpaidBillingSession(relayInfo *relaycommon.RelayInfo, mode string) *TenantCreditBillingSession {
	if relayInfo != nil {
		relayInfo.BillingSource = BillingSourceTenantPostpaid
		if normalizeBillingMode(mode) == model.BillingModeMixed {
			relayInfo.BillingSource = BillingSourceTenantMixed
		}
		relayInfo.FinalPreConsumedQuota = 0
	}
	return &TenantCreditBillingSession{relayInfo: relayInfo, mode: normalizeBillingMode(mode)}
}

func NewTenantMixedBillingSession(relayInfo *relaycommon.RelayInfo, prepaid relaycommon.BillingSettler) *TenantCreditBillingSession {
	if relayInfo != nil {
		relayInfo.BillingSource = BillingSourceTenantMixed
	}
	return &TenantCreditBillingSession{relayInfo: relayInfo, prepaid: prepaid, mode: model.BillingModeMixed}
}

func (s *TenantCreditBillingSession) Settle(actualQuota int) error {
	if s == nil {
		return nil
	}
	preConsumed := s.GetPreConsumedQuota()
	if s.prepaid != nil {
		if actualQuota <= preConsumed {
			return s.prepaid.Settle(actualQuota)
		}
		return s.prepaid.Settle(preConsumed)
	}
	return nil
}

func (s *TenantCreditBillingSession) Refund(c *gin.Context) {
	if s != nil && s.prepaid != nil {
		s.prepaid.Refund(c)
	}
}

func (s *TenantCreditBillingSession) NeedsRefund() bool {
	return s != nil && s.prepaid != nil && s.prepaid.NeedsRefund()
}

func (s *TenantCreditBillingSession) GetPreConsumedQuota() int {
	if s != nil && s.prepaid != nil {
		return s.prepaid.GetPreConsumedQuota()
	}
	return 0
}

func (s *TenantCreditBillingSession) Reserve(targetQuota int) error {
	if s != nil && s.prepaid != nil {
		return s.prepaid.Reserve(targetQuota)
	}
	if s != nil && s.relayInfo != nil {
		return EnsureTenantCreditAvailable(s.relayInfo.TenantId, targetQuota)
	}
	return nil
}

func PrepareTenantBillingSession(c *gin.Context, preConsumedQuota int, relayInfo *relaycommon.RelayInfo) (*TenantCreditBillingSession, *types.NewAPIError) {
	if relayInfo == nil || relayInfo.TenantId <= 0 {
		return nil, nil
	}
	mode := TenantBillingMode(relayInfo.TenantId)
	relayInfo.TenantBillingMode = mode
	switch mode {
	case model.BillingModePostpaid:
		if err := EnsureTenantCreditAvailable(relayInfo.TenantId, preConsumedQuota); err != nil {
			return nil, types.NewErrorWithStatusCode(err, types.ErrorCodeInsufficientUserQuota, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}
		return NewTenantPostpaidBillingSession(relayInfo, mode), nil
	case model.BillingModeMixed:
		session, apiErr := NewBillingSession(c, relayInfo, preConsumedQuota)
		if apiErr != nil {
			if apiErr.GetErrorCode() == types.ErrorCodeInsufficientUserQuota || apiErr.GetErrorCode() == types.ErrorCodePreConsumeTokenQuotaFailed {
				if err := EnsureTenantCreditAvailable(relayInfo.TenantId, preConsumedQuota); err != nil {
					return nil, types.NewErrorWithStatusCode(err, types.ErrorCodeInsufficientUserQuota, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
				}
				return NewTenantPostpaidBillingSession(relayInfo, mode), nil
			}
			return nil, apiErr
		}
		return NewTenantMixedBillingSession(relayInfo, session), nil
	default:
		return nil, nil
	}
}

func DecorateTenantLedger(ledger *model.UsageLedger, relayInfo *relaycommon.RelayInfo) {
	if ledger == nil || relayInfo == nil || relayInfo.TenantId <= 0 {
		return
	}
	mode := relayInfo.TenantBillingMode
	if mode == "" {
		mode = TenantBillingMode(relayInfo.TenantId)
	}
	prepaidConsumed := relayInfo.FinalPreConsumedQuota
	if relayInfo.Billing != nil {
		prepaidConsumed = relayInfo.Billing.GetPreConsumedQuota()
	}
	ledger.TenantId = relayInfo.TenantId
	ledger.AppId = relayInfo.AppId
	ledger.EndCustomerId = relayInfo.EndCustomerId
	ledger.BillingMode = mode
	ledger.BillingPeriod = billingPeriodFromUnix(ledger.CreatedAt)
	ledger.PostpaidQuota = TenantPostpaidQuota(mode, ledger.SellQuota, prepaidConsumed)
	ledger.PriceSnapshot = common.GetJsonString(map[string]interface{}{
		"model":            relayInfo.OriginModelName,
		"group":            relayInfo.UsingGroup,
		"billing_mode":     mode,
		"prepaid_consumed": prepaidConsumed,
		"model_ratio":      relayInfo.PriceData.ModelRatio,
		"completion_ratio": relayInfo.PriceData.CompletionRatio,
		"group_ratio":      relayInfo.PriceData.GroupRatioInfo.GroupRatio,
		"use_price":        relayInfo.PriceData.UsePrice,
		"model_price":      relayInfo.PriceData.ModelPrice,
	})
}
