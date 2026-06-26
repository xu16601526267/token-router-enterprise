package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ScopePlatform = "platform"
	ScopeTenant   = "tenant"
	ScopePersonal = "personal"

	TenantStatusActive    = "active"
	TenantStatusSuspended = "suspended"
	TenantStatusDisabled  = "disabled"

	TenantMemberStatusActive   = "active"
	TenantMemberStatusInvited  = "invited"
	TenantMemberStatusDisabled = "disabled"

	TenantRoleOwner     = "owner"
	TenantRoleAdmin     = "admin"
	TenantRoleFinance   = "finance"
	TenantRoleOps       = "ops"
	TenantRoleDeveloper = "developer"
	TenantRoleViewer    = "viewer"

	TokenOwnerScopePersonal    = ScopePersonal
	TokenOwnerScopeTenant      = ScopeTenant
	TokenOwnerScopeApp         = "app"
	TokenOwnerScopeEndCustomer = "end_customer"

	BillingModePrepaid  = "prepaid"
	BillingModePostpaid = "postpaid"
	BillingModeMixed    = "mixed"

	BillingStatementStatusDraft     = "draft"
	BillingStatementStatusConfirmed = "confirmed"
	BillingStatementStatusInvoiced  = "invoiced"
	BillingStatementStatusPaid      = "paid"
	BillingStatementStatusOverdue   = "overdue"
	BillingStatementStatusAdjusted  = "adjusted"

	CreditAccountStatusActive    = "active"
	CreditAccountStatusSuspended = "suspended"

	FrontChannelStatusActive   = "active"
	FrontChannelStatusDisabled = "disabled"

	TenantRoutingStatusDraft      = "draft"
	TenantRoutingStatusApproved   = "approved"
	TenantRoutingStatusApplied    = "applied"
	TenantRoutingStatusDisabled   = "disabled"
	TenantRoutingStatusRolledBack = "rolled_back"
)

type Tenant struct {
	Id          int    `json:"id"`
	Name        string `json:"name" gorm:"size:128;not null;index"`
	Type        string `json:"type" gorm:"size:32;default:'enterprise';index"`
	Status      string `json:"status" gorm:"size:32;default:'active';index"`
	Industry    string `json:"industry" gorm:"size:64;default:'';index"`
	OwnerUserId int    `json:"owner_user_id" gorm:"index"`
	BrandConfig string `json:"brand_config" gorm:"type:text"`
	Domain      string `json:"domain" gorm:"size:128;default:'';index"`
	ContractNo  string `json:"contract_no" gorm:"size:128;default:'';index"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint;index"`
}

func (t *Tenant) normalize() {
	t.Name = strings.TrimSpace(t.Name)
	t.Type = strings.TrimSpace(t.Type)
	t.Status = strings.TrimSpace(t.Status)
	t.Industry = strings.TrimSpace(t.Industry)
	t.Domain = strings.TrimSpace(t.Domain)
	t.ContractNo = strings.TrimSpace(t.ContractNo)
	if t.Type == "" {
		t.Type = "enterprise"
	}
	if t.Status == "" {
		t.Status = TenantStatusActive
	}
	now := common.GetTimestamp()
	if t.CreatedAt == 0 {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
}

func (t *Tenant) Insert() error {
	t.normalize()
	if t.Name == "" {
		return errors.New("tenant name is required")
	}
	return DB.Create(t).Error
}

func GetTenantById(id int) (*Tenant, error) {
	var tenant Tenant
	if err := DB.First(&tenant, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

type TenantMember struct {
	Id                   int    `json:"id"`
	TenantId             int    `json:"tenant_id" gorm:"not null;index;uniqueIndex:uk_tenant_member"`
	UserId               int    `json:"user_id" gorm:"not null;index;uniqueIndex:uk_tenant_member"`
	Role                 string `json:"role" gorm:"size:32;not null;index"`
	DepartmentId         int    `json:"department_id" gorm:"index;default:0"`
	Status               string `json:"status" gorm:"size:32;default:'active';index"`
	PermissionTemplateId int    `json:"permission_template_id" gorm:"index;default:0"`
	JoinedAt             int64  `json:"joined_at" gorm:"bigint;index"`
	CreatedAt            int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt            int64  `json:"updated_at" gorm:"bigint;index"`
}

func (m *TenantMember) normalize() {
	m.Role = strings.ToLower(strings.TrimSpace(m.Role))
	m.Status = strings.TrimSpace(m.Status)
	if m.Role == "" {
		m.Role = TenantRoleViewer
	}
	if m.Status == "" {
		m.Status = TenantMemberStatusActive
	}
	now := common.GetTimestamp()
	if m.JoinedAt == 0 {
		m.JoinedAt = now
	}
	if m.CreatedAt == 0 {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
}

func (m *TenantMember) Upsert() error {
	m.normalize()
	if m.TenantId <= 0 || m.UserId <= 0 {
		return errors.New("tenant member requires tenant_id and user_id")
	}
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role", "department_id", "status", "permission_template_id", "joined_at", "updated_at"}),
	}).Create(m).Error
}

func GetTenantMember(tenantId int, userId int) (*TenantMember, error) {
	var member TenantMember
	if err := DB.Where("tenant_id = ? AND user_id = ?", tenantId, userId).First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func IsTenantRoleAtLeast(role string, required string) bool {
	rank := map[string]int{
		TenantRoleViewer:    10,
		TenantRoleDeveloper: 20,
		TenantRoleOps:       30,
		TenantRoleFinance:   35,
		TenantRoleAdmin:     40,
		TenantRoleOwner:     50,
	}
	return rank[strings.ToLower(strings.TrimSpace(role))] >= rank[strings.ToLower(strings.TrimSpace(required))]
}

func TenantRoleAllowed(role string, allowed []string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if len(allowed) == 0 {
		return true
	}
	for _, item := range allowed {
		if role == strings.ToLower(strings.TrimSpace(item)) {
			return true
		}
	}
	return false
}

type TenantEndCustomer struct {
	Id            int    `json:"id"`
	TenantId      int    `json:"tenant_id" gorm:"not null;index;uniqueIndex:uk_tenant_end_customer"`
	UserId        int    `json:"user_id" gorm:"not null;index;uniqueIndex:uk_tenant_end_customer"`
	CustomerType  string `json:"customer_type" gorm:"size:32;default:'user';index"`
	QuotaPolicyId int    `json:"quota_policy_id" gorm:"index;default:0"`
	Status        string `json:"status" gorm:"size:32;default:'active';index"`
	ExternalId    string `json:"external_id" gorm:"size:128;default:'';index"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt     int64  `json:"updated_at" gorm:"bigint;index"`
}

type TenantApp struct {
	Id         int    `json:"id"`
	TenantId   int    `json:"tenant_id" gorm:"not null;index"`
	Name       string `json:"name" gorm:"size:128;not null;index"`
	Env        string `json:"env" gorm:"size:32;default:'prod';index"`
	OwnerId    int    `json:"owner_id" gorm:"index;default:0"`
	WebhookUrl string `json:"webhook_url" gorm:"size:255;default:''"`
	IpPolicy   string `json:"ip_policy" gorm:"type:text"`
	Status     string `json:"status" gorm:"size:32;default:'active';index"`
	CreatedAt  int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt  int64  `json:"updated_at" gorm:"bigint;index"`
}

type TenantModelPolicy struct {
	Id          int    `json:"id"`
	TenantId    int    `json:"tenant_id" gorm:"not null;index;uniqueIndex:uk_tenant_model_policy"`
	ModelId     int    `json:"model_id" gorm:"index;default:0"`
	ModelName   string `json:"model_name" gorm:"size:128;default:'';index;uniqueIndex:uk_tenant_model_policy"`
	Visible     bool   `json:"visible" gorm:"default:true;index"`
	PricePlanId int    `json:"price_plan_id" gorm:"index;default:0"`
	RateLimit   string `json:"rate_limit" gorm:"size:128;default:''"`
	Alias       string `json:"alias" gorm:"size:128;default:'';index"`
	Enabled     bool   `json:"enabled" gorm:"default:true;index"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint;index"`
}

type FrontChannel struct {
	Id              int    `json:"id"`
	Name            string `json:"name" gorm:"size:128;not null;index"`
	Type            string `json:"type" gorm:"size:64;default:'';index"`
	Domain          string `json:"domain" gorm:"size:128;default:'';index"`
	LandingPage     string `json:"landing_page" gorm:"size:255;default:''"`
	Owner           string `json:"owner" gorm:"size:128;default:'';index"`
	PricingPolicyId int    `json:"pricing_policy_id" gorm:"index;default:0"`
	Utm             string `json:"utm" gorm:"type:text"`
	Status          string `json:"status" gorm:"size:32;default:'active';index"`
	CreatedAt       int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt       int64  `json:"updated_at" gorm:"bigint;index"`
}

type BillingConfig struct {
	Id               int    `json:"id"`
	TenantId         int    `json:"tenant_id" gorm:"not null;uniqueIndex"`
	BillingMode      string `json:"billing_mode" gorm:"size:32;default:'prepaid';index"`
	BillingCycle     string `json:"billing_cycle" gorm:"size:32;default:'monthly'"`
	StatementDay     int    `json:"statement_day" gorm:"default:1"`
	PaymentTerms     int    `json:"payment_terms" gorm:"default:30"`
	CreditLimit      int64  `json:"credit_limit" gorm:"default:0"`
	OverCreditPolicy string `json:"over_credit_policy" gorm:"size:64;default:'block'"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt        int64  `json:"updated_at" gorm:"bigint;index"`
}

type CreditAccount struct {
	Id                 int    `json:"id"`
	TenantId           int    `json:"tenant_id" gorm:"not null;uniqueIndex"`
	CreditLimit        int64  `json:"credit_limit" gorm:"default:0"`
	UnbilledAmount     int64  `json:"unbilled_amount" gorm:"default:0"`
	BilledUnpaidAmount int64  `json:"billed_unpaid_amount" gorm:"default:0"`
	OverdueAmount      int64  `json:"overdue_amount" gorm:"default:0"`
	AvailableCredit    int64  `json:"available_credit" gorm:"default:0"`
	Status             string `json:"status" gorm:"size:32;default:'active';index"`
	CreatedAt          int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt          int64  `json:"updated_at" gorm:"bigint;index"`
}

func (a *CreditAccount) Recalculate() {
	a.AvailableCredit = a.CreditLimit - a.UnbilledAmount - a.BilledUnpaidAmount - a.OverdueAmount
}

type BillingStatement struct {
	Id          int    `json:"id"`
	TenantId    int    `json:"tenant_id" gorm:"not null;index;uniqueIndex:uk_billing_statement_period"`
	PeriodStart int64  `json:"period_start" gorm:"bigint;not null;uniqueIndex:uk_billing_statement_period"`
	PeriodEnd   int64  `json:"period_end" gorm:"bigint;not null;uniqueIndex:uk_billing_statement_period"`
	Amount      int64  `json:"amount" gorm:"default:0"`
	Adjustment  int64  `json:"adjustment" gorm:"default:0"`
	Payable     int64  `json:"payable" gorm:"default:0"`
	Status      string `json:"status" gorm:"size:32;default:'draft';index"`
	ConfirmedAt int64  `json:"confirmed_at" gorm:"bigint;default:0"`
	DueDate     int64  `json:"due_date" gorm:"bigint;default:0;index"`
	GeneratedAt int64  `json:"generated_at" gorm:"bigint;index"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint;index"`
}

type PaymentRecord struct {
	Id          int    `json:"id"`
	StatementId int    `json:"statement_id" gorm:"not null;index"`
	TenantId    int    `json:"tenant_id" gorm:"index"`
	Amount      int64  `json:"amount" gorm:"default:0"`
	Method      string `json:"method" gorm:"size:64;default:''"`
	PaidAt      int64  `json:"paid_at" gorm:"bigint;index"`
	OperatorId  int    `json:"operator_id" gorm:"index;default:0"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;index"`
}

type Invoice struct {
	Id            int    `json:"id"`
	StatementId   int    `json:"statement_id" gorm:"not null;index"`
	TenantId      int    `json:"tenant_id" gorm:"index"`
	Amount        int64  `json:"amount" gorm:"default:0"`
	InvoiceNo     string `json:"invoice_no" gorm:"size:128;default:'';index"`
	InvoiceStatus string `json:"invoice_status" gorm:"size:32;default:'none';index"`
	OperatorId    int    `json:"operator_id" gorm:"index;default:0"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt     int64  `json:"updated_at" gorm:"bigint;index"`
}

type TenantRoutingPreference struct {
	Id                  int    `json:"id"`
	TenantId            int    `json:"tenant_id" gorm:"not null;index"`
	ModelName           string `json:"model_name" gorm:"size:128;default:'';index"`
	SlaTier             string `json:"sla_tier" gorm:"size:64;default:'';index"`
	PreferredSupplierId int    `json:"preferred_supplier_id" gorm:"index;default:0"`
	PreferredChannelId  int    `json:"preferred_channel_id" gorm:"index;default:0"`
	Status              string `json:"status" gorm:"size:32;default:'draft';index"`
	Reason              string `json:"reason" gorm:"type:text"`
	RequestedBy         int    `json:"requested_by" gorm:"index;default:0"`
	ApprovedBy          int    `json:"approved_by" gorm:"index;default:0"`
	AppliedAt           int64  `json:"applied_at" gorm:"bigint;default:0"`
	RollbackFromId      int    `json:"rollback_from_id" gorm:"index;default:0"`
	OperatorNote        string `json:"operator_note" gorm:"type:text"`
	CreatedAt           int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt           int64  `json:"updated_at" gorm:"bigint;index"`
}

type AuditLog struct {
	Id        int    `json:"id"`
	ScopeType string `json:"scope_type" gorm:"size:32;not null;index"`
	ScopeId   int    `json:"scope_id" gorm:"index;default:0"`
	ActorId   int    `json:"actor_id" gorm:"index;default:0"`
	Action    string `json:"action" gorm:"size:128;not null;index"`
	Target    string `json:"target" gorm:"size:128;default:'';index"`
	Before    string `json:"before" gorm:"type:text"`
	After     string `json:"after" gorm:"type:text"`
	Ip        string `json:"ip" gorm:"size:64;default:'';index"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index"`
}

func RecordScopedAuditLog(log *AuditLog) error {
	if log == nil {
		return errors.New("audit log is nil")
	}
	log.ScopeType = strings.TrimSpace(log.ScopeType)
	log.Action = strings.TrimSpace(log.Action)
	if log.ScopeType == "" {
		log.ScopeType = ScopePlatform
	}
	if log.Action == "" {
		return errors.New("audit action is required")
	}
	if log.CreatedAt == 0 {
		log.CreatedAt = common.GetTimestamp()
	}
	return DB.Create(log).Error
}

func tenantNowCreateUpdate(created *int64, updated *int64) {
	now := common.GetTimestamp()
	if created != nil && *created == 0 {
		*created = now
	}
	if updated != nil {
		*updated = now
	}
}

func UpsertBillingConfig(config *BillingConfig) error {
	if config == nil {
		return errors.New("billing config is nil")
	}
	config.BillingMode = strings.TrimSpace(config.BillingMode)
	config.BillingCycle = strings.TrimSpace(config.BillingCycle)
	config.OverCreditPolicy = strings.TrimSpace(config.OverCreditPolicy)
	if config.BillingMode == "" {
		config.BillingMode = BillingModePrepaid
	}
	if config.BillingCycle == "" {
		config.BillingCycle = "monthly"
	}
	if config.StatementDay <= 0 {
		config.StatementDay = 1
	}
	if config.PaymentTerms <= 0 {
		config.PaymentTerms = 30
	}
	if config.OverCreditPolicy == "" {
		config.OverCreditPolicy = "block"
	}
	tenantNowCreateUpdate(&config.CreatedAt, &config.UpdatedAt)
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"billing_mode", "billing_cycle", "statement_day", "payment_terms", "credit_limit", "over_credit_policy", "updated_at"}),
	}).Create(config).Error
}

func GetBillingConfigByTenantId(tenantId int) (*BillingConfig, error) {
	var config BillingConfig
	if err := DB.Where("tenant_id = ?", tenantId).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func GetCreditAccountByTenantId(tenantId int) (*CreditAccount, error) {
	var account CreditAccount
	if err := DB.Where("tenant_id = ?", tenantId).First(&account).Error; err != nil {
		return nil, err
	}
	account.Recalculate()
	return &account, nil
}

func UpsertCreditAccount(account *CreditAccount) error {
	if account == nil {
		return errors.New("credit account is nil")
	}
	if account.Status == "" {
		account.Status = CreditAccountStatusActive
	}
	account.Recalculate()
	tenantNowCreateUpdate(&account.CreatedAt, &account.UpdatedAt)
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"credit_limit", "unbilled_amount", "billed_unpaid_amount", "overdue_amount", "available_credit", "status", "updated_at"}),
	}).Create(account).Error
}

func IsTenantActive(tenantId int) (bool, error) {
	if tenantId <= 0 {
		return true, nil
	}
	tenant, err := GetTenantById(tenantId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, err
		}
		return false, err
	}
	return tenant.Status == TenantStatusActive, nil
}
