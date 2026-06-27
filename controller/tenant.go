package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func pathTenantId(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("tenant_id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "tenant_id is invalid")
		return 0, false
	}
	return id, true
}

func pathStatementId(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("statement_id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "statement_id is invalid")
		return 0, false
	}
	return id, true
}

func pathTenantObjectId(c *gin.Context, param string) (int, bool) {
	id, err := strconv.Atoi(c.Param(param))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, param+" is invalid")
		return 0, false
	}
	return id, true
}

func tenantListLimit(c *gin.Context) int {
	return enterprisePositiveInt(c, "limit", 50, 500)
}

func tenantListOffset(c *gin.Context) int {
	return parseOptionalIntQuery(c, "offset")
}

func GetMyWorkspaces(c *gin.Context) {
	workspaces := []gin.H{{
		"scope_type": model.ScopePersonal,
		"scope_id":   c.GetInt("id"),
		"role":       "owner",
	}}
	if c.GetInt("role") >= common.RoleAdminUser {
		workspaces = append(workspaces, gin.H{
			"scope_type": model.ScopePlatform,
			"scope_id":   0,
			"role":       "admin",
		})
	}
	var rows []struct {
		model.Tenant
		MemberRole string `json:"member_role"`
	}
	if err := model.DB.Table("tenants").
		Select("tenants.*, tenant_members.role AS member_role").
		Joins("JOIN tenant_members ON tenant_members.tenant_id = tenants.id").
		Where("tenant_members.user_id = ? AND tenant_members.status = ? AND tenants.status <> ?", c.GetInt("id"), model.TenantMemberStatusActive, model.TenantStatusDisabled).
		Order("tenants.id DESC").
		Scan(&rows).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	for _, row := range rows {
		tenant := row.Tenant
		workspaces = append(workspaces, gin.H{
			"scope_type": model.ScopeTenant,
			"scope_id":   tenant.Id,
			"role":       row.MemberRole,
			"tenant":     tenant,
		})
	}
	common.ApiSuccess(c, workspaces)
}

func CreatePlatformTenant(c *gin.Context) {
	var input service.TenantCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	tenant, err := service.CreateTenantWithDefaults(input, c.GetInt("id"), c.ClientIP())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, tenant)
}

func GetPlatformTenants(c *gin.Context) {
	var tenants []model.Tenant
	query := model.DB.Model(&model.Tenant{}).Order("id DESC")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&tenants).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, tenants)
}

func GetPlatformTenant360(c *gin.Context) {
	tenantId, ok := pathTenantId(c)
	if !ok {
		return
	}
	tenant, err := model.GetTenantById(tenantId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var memberCount, endCustomerCount, appCount, keyCount, ledgerCount int64
	_ = model.DB.Model(&model.TenantMember{}).Where("tenant_id = ?", tenantId).Count(&memberCount).Error
	_ = model.DB.Model(&model.TenantEndCustomer{}).Where("tenant_id = ?", tenantId).Count(&endCustomerCount).Error
	_ = model.DB.Model(&model.TenantApp{}).Where("tenant_id = ?", tenantId).Count(&appCount).Error
	_ = model.DB.Model(&model.Token{}).Where("tenant_id = ?", tenantId).Count(&keyCount).Error
	_ = model.DB.Model(&model.UsageLedger{}).Where("tenant_id = ?", tenantId).Count(&ledgerCount).Error
	config, _ := model.GetBillingConfigByTenantId(tenantId)
	credit, _ := model.GetCreditAccountByTenantId(tenantId)
	common.ApiSuccess(c, gin.H{
		"tenant":             tenant,
		"billing_config":     config,
		"credit_account":     credit,
		"members":            memberCount,
		"end_customers":      endCustomerCount,
		"apps":               appCount,
		"api_keys":           keyCount,
		"usage_ledger_count": ledgerCount,
	})
}

func UpdatePlatformTenantStatus(c *gin.Context) {
	tenantId, ok := pathTenantId(c)
	if !ok {
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	status, err := service.NormalizeTenantStatus(input.Status)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DB.Model(&model.Tenant{}).Where("id = ?", tenantId).Updates(map[string]interface{}{"status": status, "updated_at": common.GetTimestamp()}).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: tenantId, ActorId: c.GetInt("id"), Action: "tenant.status.update", Target: "tenant", After: common.GetJsonString(gin.H{"status": status}), Ip: c.ClientIP()})
	common.ApiSuccess(c, true)
}

func UpdatePlatformTenantBillingConfig(c *gin.Context) {
	tenantId, ok := pathTenantId(c)
	if !ok {
		return
	}
	var input model.BillingConfig
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.SetTenantBillingConfig(tenantId, &input, c.GetInt("id"), c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}
	config, _ := model.GetBillingConfigByTenantId(tenantId)
	credit, _ := model.GetCreditAccountByTenantId(tenantId)
	common.ApiSuccess(c, gin.H{"billing_config": config, "credit_account": credit})
}

func UpsertPlatformTenantModelPolicy(c *gin.Context) {
	tenantId, ok := pathTenantId(c)
	if !ok {
		return
	}
	var input model.TenantModelPolicy
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	input.TenantId = tenantId
	if err := service.UpsertTenantModelPolicy(&input, c.GetInt("id"), c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, input)
}

func GetPlatformTenantModelPolicies(c *gin.Context) {
	tenantId, ok := pathTenantId(c)
	if !ok {
		return
	}
	var policies []model.TenantModelPolicy
	if err := model.DB.Where("tenant_id = ?", tenantId).Order("id DESC").Find(&policies).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, policies)
}

func CreatePlatformFrontChannel(c *gin.Context) {
	var input model.FrontChannel
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.CreateFrontChannel(&input, c.GetInt("id"), c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, input)
}

func GetPlatformFrontChannels(c *gin.Context) {
	var channels []model.FrontChannel
	if err := model.DB.Order("id DESC").Find(&channels).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, channels)
}

func UpdatePlatformFrontChannel(c *gin.Context) {
	id, ok := pathTenantObjectId(c, "id")
	if !ok {
		return
	}
	var input model.FrontChannel
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	updates := map[string]interface{}{
		"name":              input.Name,
		"type":              input.Type,
		"domain":            input.Domain,
		"landing_page":      input.LandingPage,
		"owner":             input.Owner,
		"pricing_policy_id": input.PricingPolicyId,
		"utm":               input.Utm,
		"status":            input.Status,
		"updated_at":        common.GetTimestamp(),
	}
	if input.Status == "" {
		delete(updates, "status")
	}
	if err := model.DB.Model(&model.FrontChannel{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopePlatform, ActorId: c.GetInt("id"), Action: "front_channel.update", Target: "front_channel:" + strconv.Itoa(id), After: common.MapToJsonStr(updates), Ip: c.ClientIP()})
	common.ApiSuccess(c, true)
}

func GetScopedAuditLogs(c *gin.Context) {
	var logs []model.AuditLog
	query := model.DB.Model(&model.AuditLog{}).Order("id DESC").Limit(200)
	if scopeType := c.Query("scope_type"); scopeType != "" {
		query = query.Where("scope_type = ?", scopeType)
	}
	if scopeId, _ := strconv.Atoi(c.Query("scope_id")); scopeId > 0 {
		query = query.Where("scope_id = ?", scopeId)
	}
	if err := query.Find(&logs).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, logs)
}

func GetTenantOverview(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	summary, err := service.GetTenantUsageSummary(tenantId, parseOptionalInt64Query(c, "start_time"), parseOptionalInt64Query(c, "end_time"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	credit, _ := model.GetCreditAccountByTenantId(tenantId)
	config, _ := model.GetBillingConfigByTenantId(tenantId)
	common.ApiSuccess(c, gin.H{"usage": summary, "credit_account": credit, "billing_config": config})
}

func GetTenantMembers(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var members []model.TenantMember
	query := model.DB.Where("tenant_id = ?", tenantId).Order("id DESC")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if role := c.Query("role"); role != "" {
		query = query.Where("role = ?", role)
	}
	if err := query.Offset(tenantListOffset(c)).Limit(tenantListLimit(c)).Find(&members).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, members)
}

func CreateTenantMember(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var input model.TenantMember
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	input.TenantId = tenantId
	if err := input.Upsert(); err != nil {
		common.ApiError(c, err)
		return
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: tenantId, ActorId: c.GetInt("id"), Action: "tenant.member.upsert", Target: "tenant_member", After: common.GetJsonString(input), Ip: c.ClientIP()})
	common.ApiSuccess(c, input)
}

func UpdateTenantMember(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	memberId, ok := pathTenantObjectId(c, "id")
	if !ok {
		return
	}
	var input struct {
		Role                 string `json:"role"`
		Status               string `json:"status"`
		DepartmentId         int    `json:"department_id"`
		PermissionTemplateId int    `json:"permission_template_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	updates := map[string]interface{}{
		"role":                   input.Role,
		"status":                 input.Status,
		"department_id":          input.DepartmentId,
		"permission_template_id": input.PermissionTemplateId,
		"updated_at":             common.GetTimestamp(),
	}
	if input.Role == "" {
		delete(updates, "role")
	}
	if input.Status == "" {
		delete(updates, "status")
	}
	if err := model.DB.Model(&model.TenantMember{}).Where("id = ? AND tenant_id = ?", memberId, tenantId).Updates(updates).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: tenantId, ActorId: c.GetInt("id"), Action: "tenant.member.update", Target: "tenant_member:" + strconv.Itoa(memberId), After: common.MapToJsonStr(updates), Ip: c.ClientIP()})
	common.ApiSuccess(c, true)
}

func GetTenantEndCustomers(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var customers []model.TenantEndCustomer
	query := model.DB.Where("tenant_id = ?", tenantId).Order("id DESC")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if customerType := c.Query("customer_type"); customerType != "" {
		query = query.Where("customer_type = ?", customerType)
	}
	if err := query.Offset(tenantListOffset(c)).Limit(tenantListLimit(c)).Find(&customers).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, customers)
}

func CreateTenantEndCustomer(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var input model.TenantEndCustomer
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	input.TenantId = tenantId
	if err := service.CreateTenantEndCustomer(&input, c.GetInt("id"), c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, input)
}

func UpdateTenantEndCustomer(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	customerId, ok := pathTenantObjectId(c, "id")
	if !ok {
		return
	}
	var input struct {
		CustomerType  string `json:"customer_type"`
		QuotaPolicyId int    `json:"quota_policy_id"`
		Status        string `json:"status"`
		ExternalId    string `json:"external_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	updates := map[string]interface{}{
		"customer_type":   input.CustomerType,
		"quota_policy_id": input.QuotaPolicyId,
		"status":          input.Status,
		"external_id":     input.ExternalId,
		"updated_at":      common.GetTimestamp(),
	}
	if input.CustomerType == "" {
		delete(updates, "customer_type")
	}
	if input.Status == "" {
		delete(updates, "status")
	}
	if err := model.DB.Model(&model.TenantEndCustomer{}).Where("id = ? AND tenant_id = ?", customerId, tenantId).Updates(updates).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: tenantId, ActorId: c.GetInt("id"), Action: "tenant.end_customer.update", Target: "tenant_end_customer:" + strconv.Itoa(customerId), After: common.MapToJsonStr(updates), Ip: c.ClientIP()})
	common.ApiSuccess(c, true)
}

func GetTenantApps(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var apps []model.TenantApp
	query := model.DB.Where("tenant_id = ?", tenantId).Order("id DESC")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if env := c.Query("env"); env != "" {
		query = query.Where("env = ?", env)
	}
	if err := query.Offset(tenantListOffset(c)).Limit(tenantListLimit(c)).Find(&apps).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, apps)
}

func CreateTenantApp(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var input model.TenantApp
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	input.TenantId = tenantId
	if err := service.CreateTenantApp(&input, c.GetInt("id"), c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, input)
}

func UpdateTenantApp(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	appId, ok := pathTenantObjectId(c, "id")
	if !ok {
		return
	}
	var input model.TenantApp
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	updates := map[string]interface{}{
		"name":        input.Name,
		"env":         input.Env,
		"owner_id":    input.OwnerId,
		"webhook_url": input.WebhookUrl,
		"ip_policy":   input.IpPolicy,
		"status":      input.Status,
		"updated_at":  common.GetTimestamp(),
	}
	if input.Name == "" {
		delete(updates, "name")
	}
	if input.Env == "" {
		delete(updates, "env")
	}
	if input.Status == "" {
		delete(updates, "status")
	}
	if err := model.DB.Model(&model.TenantApp{}).Where("id = ? AND tenant_id = ?", appId, tenantId).Updates(updates).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: tenantId, ActorId: c.GetInt("id"), Action: "tenant.app.update", Target: "tenant_app:" + strconv.Itoa(appId), After: common.MapToJsonStr(updates), Ip: c.ClientIP()})
	common.ApiSuccess(c, true)
}

func GetTenantModelPolicies(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var policies []model.TenantModelPolicy
	if err := model.DB.Where("tenant_id = ?", tenantId).Order("id DESC").Find(&policies).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, policies)
}

func CreateTenantAPIKey(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var input service.TenantAPIKeyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	token, secret, err := service.CreateTenantAPIKey(tenantId, input, c.GetInt("id"), c.ClientIP())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"token": token, "secret_key": secret})
}

func GetTenantAPIKeys(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var tokens []model.Token
	if err := model.DB.Where("tenant_id = ?", tenantId).Order("id DESC").Find(&tokens).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	for i := range tokens {
		tokens[i].Clean()
	}
	common.ApiSuccess(c, tokens)
}

func UpdateTenantAPIKeyStatus(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	tokenId, ok := pathTenantObjectId(c, "id")
	if !ok {
		return
	}
	var input struct {
		Status         *int   `json:"status"`
		RotationStatus string `json:"rotation_status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	updates := map[string]interface{}{}
	if input.Status != nil {
		updates["status"] = *input.Status
	}
	if input.RotationStatus != "" {
		updates["rotation_status"] = input.RotationStatus
	}
	if len(updates) == 0 {
		common.ApiErrorMsg(c, "no api key status updates")
		return
	}
	if err := model.DB.Model(&model.Token{}).Where("id = ? AND tenant_id = ?", tokenId, tenantId).Updates(updates).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	_ = model.RecordScopedAuditLog(&model.AuditLog{ScopeType: model.ScopeTenant, ScopeId: tenantId, ActorId: c.GetInt("id"), Action: "tenant.api_key.status_update", Target: "token:" + strconv.Itoa(tokenId), After: common.MapToJsonStr(updates), Ip: c.ClientIP()})
	common.ApiSuccess(c, true)
}

func GetTenantUsageLedgers(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	filters := model.UsageLedgerFilters{
		TenantId:      tenantId,
		AppId:         parseOptionalIntQuery(c, "app_id"),
		EndCustomerId: parseOptionalIntQuery(c, "end_customer_id"),
		TokenId:       parseOptionalIntQuery(c, "token_id"),
		ModelName:     c.Query("model_name"),
		BillingMode:   c.Query("billing_mode"),
		StartTime:     parseOptionalInt64Query(c, "start_time"),
		EndTime:       parseOptionalInt64Query(c, "end_time"),
	}
	items, total, err := model.SearchUsageLedgers(filters, parseOptionalIntQuery(c, "offset"), enterprisePositiveInt(c, "limit", 50, 500))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total})
}

func GetTenantBillingStatements(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var statements []model.BillingStatement
	query := model.DB.Where("tenant_id = ?", tenantId).Order("period_start DESC, id DESC")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Offset(tenantListOffset(c)).Limit(tenantListLimit(c)).Find(&statements).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, statements)
}

func GetTenantBillingStatementDetail(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	statementId, ok := pathStatementId(c)
	if !ok {
		return
	}
	var statement model.BillingStatement
	if err := model.DB.Where("id = ? AND tenant_id = ?", statementId, tenantId).First(&statement).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	var payments []model.PaymentRecord
	_ = model.DB.Where("statement_id = ? AND tenant_id = ?", statementId, tenantId).Order("id DESC").Find(&payments).Error
	var invoices []model.Invoice
	_ = model.DB.Where("statement_id = ? AND tenant_id = ?", statementId, tenantId).Order("id DESC").Find(&invoices).Error
	ledgers, total, err := model.SearchUsageLedgers(model.UsageLedgerFilters{
		TenantId:  tenantId,
		StartTime: statement.PeriodStart,
		EndTime:   statement.PeriodEnd,
		Status:    "success",
	}, tenantListOffset(c), tenantListLimit(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"statement": statement,
		"payments":  payments,
		"invoices":  invoices,
		"ledgers":   ledgers,
		"total":     total,
	})
}

func GenerateTenantBillingStatement(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var input struct {
		PeriodStart int64 `json:"period_start"`
		PeriodEnd   int64 `json:"period_end"`
		Adjustment  int64 `json:"adjustment"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	statement, err := service.GenerateTenantBillingStatement(tenantId, input.PeriodStart, input.PeriodEnd, input.Adjustment, c.GetInt("id"), c.ClientIP())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, statement)
}

func ConfirmTenantBillingStatement(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	statementId, ok := pathStatementId(c)
	if !ok {
		return
	}
	statement, err := service.ConfirmTenantBillingStatement(tenantId, statementId, c.GetInt("id"), c.ClientIP())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, statement)
}

func RegisterTenantBillingPayment(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	statementId, ok := pathStatementId(c)
	if !ok {
		return
	}
	var input struct {
		Amount        int64  `json:"amount"`
		Method        string `json:"method"`
		InvoiceNo     string `json:"invoice_no"`
		InvoiceStatus string `json:"invoice_status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	statement, err := service.RegisterTenantPaymentAndInvoice(tenantId, statementId, input.Amount, input.Method, input.InvoiceNo, input.InvoiceStatus, c.GetInt("id"), c.ClientIP())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, statement)
}

func CreateTenantRoutingPreference(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var input model.TenantRoutingPreference
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	input.TenantId = tenantId
	if err := service.CreateTenantRoutingPreference(&input, c.GetInt("id"), c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, input)
}

func GetTenantRoutingPreferences(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var prefs []model.TenantRoutingPreference
	query := model.DB.Where("tenant_id = ?", tenantId).Order("id DESC")
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if modelName := c.Query("model_name"); modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}
	if err := query.Offset(tenantListOffset(c)).Limit(tenantListLimit(c)).Find(&prefs).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, prefs)
}

func ReviewTenantRoutingPreference(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	preferenceId, err := strconv.Atoi(c.Param("id"))
	if err != nil || preferenceId <= 0 {
		common.ApiErrorMsg(c, "routing preference id is invalid")
		return
	}
	var input struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		common.ApiError(c, err)
		return
	}
	pref, err := service.ReviewTenantRoutingPreference(tenantId, preferenceId, input.Status, c.GetInt("id"), input.Note, c.ClientIP())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, pref)
}

func GetTenantAuditLogs(c *gin.Context) {
	tenantId := common.GetContextKeyInt(c, constant.ContextKeyTenantId)
	var logs []model.AuditLog
	query := model.DB.Where("scope_type = ? AND scope_id = ?", model.ScopeTenant, tenantId).Order("id DESC")
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	if err := query.Offset(tenantListOffset(c)).Limit(tenantListLimit(c)).Find(&logs).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, logs)
}
