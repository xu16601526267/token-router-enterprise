package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/require"
)

const (
	b2b2cTenantCount     = 3
	b2b2cCustomersPerB   = 4
	b2b2cKeysPerCustomer = 2
	b2b2cRequestsPerKey  = 2
)

type e2eAccessActor struct {
	id          int
	accessToken string
}

type e2eTenantScenario struct {
	tenant      model.Tenant
	owner       e2eAccessActor
	app         model.TenantApp
	modelPolicy model.TenantModelPolicy
	customers   []e2eCustomerScenario
}

type e2eCustomerScenario struct {
	userID   int
	customer model.TenantEndCustomer
	keys     []e2eKeyScenario
}

type e2eKeyScenario struct {
	token  model.Token
	secret string
}

type e2eTenantUsageLedgersResponse struct {
	Items []*model.UsageLedger `json:"items"`
	Total int64                `json:"total"`
}

type e2eTenantAPIKeyResponse struct {
	Token     model.Token `json:"token"`
	SecretKey string      `json:"secret_key"`
}

type e2eTenantOverviewResponse struct {
	Usage         service.TenantUsageSummary `json:"usage"`
	CreditAccount model.CreditAccount        `json:"credit_account"`
	BillingConfig model.BillingConfig        `json:"billing_config"`
}

type e2eChatResult struct {
	requestID string
	status    int
	err       error
}

func TestTenantB2B2CMultiTenantMultiCustomerMultiKeyConcurrentFullChain(t *testing.T) {
	supply := newGB10MockSupply(t)
	tokenRouter := setupTokenRouterE2E(t, supply)
	periodStart := common.GetTimestamp() - 3600

	intruder := seedE2EAccessActor(t, 9000, "b2b2c-intruder", common.RoleCommonUser)
	scenarios := make([]e2eTenantScenario, 0, b2b2cTenantCount)
	for tenantIndex := 0; tenantIndex < b2b2cTenantCount; tenantIndex++ {
		scenarios = append(scenarios, createB2B2CTenantScenario(t, tokenRouter.URL, tenantIndex))
	}
	createAppliedTenantRoutingPreferenceE2E(t, tokenRouter.URL, scenarios[0])

	e2eExpectAPIFailure(t, tokenRouter.URL, http.MethodGet,
		fmt.Sprintf("/api/tenant/%d/usage_ledgers?limit=10", scenarios[1].tenant.Id),
		scenarios[0].owner, nil, "没有权限访问该租户")

	results := make(chan e2eChatResult, b2b2cTenantCount*b2b2cCustomersPerB*b2b2cKeysPerCustomer*b2b2cRequestsPerKey)
	var wg sync.WaitGroup
	for tenantIndex, scenario := range scenarios {
		for customerIndex, customer := range scenario.customers {
			for keyIndex, key := range customer.keys {
				sessionID := fmt.Sprintf("trsess_b2b2c_b%d_c%d_k%d", tenantIndex, customerIndex, keyIndex)
				for requestIndex := 0; requestIndex < b2b2cRequestsPerKey; requestIndex++ {
					requestID := fmt.Sprintf("b2b2c-b%d-c%d-k%d-r%d", tenantIndex, customerIndex, keyIndex, requestIndex)
					wg.Add(1)
					go func(secret string, sessionID string, requestID string) {
						defer wg.Done()
						status, err := e2eTenantChatCompletion(tokenRouter.URL, secret, sessionID, requestID, e2eModelName)
						results <- e2eChatResult{requestID: requestID, status: status, err: err}
					}(key.secret, sessionID, requestID)
				}
			}
		}
	}
	wg.Wait()
	close(results)
	for result := range results {
		require.NoError(t, result.err, result.requestID)
		require.Equal(t, http.StatusOK, result.status, result.requestID)
	}

	expectedPerTenant := b2b2cCustomersPerB * b2b2cKeysPerCustomer * b2b2cRequestsPerKey
	for tenantIndex, scenario := range scenarios {
		ledgers := waitTenantLedgerCountE2E(t, scenario.tenant.Id, expectedPerTenant)
		assertTenantLedgerBreakdownE2E(t, scenario, ledgers)
		if tenantIndex == 0 {
			for _, ledger := range ledgers {
				require.Equal(t, 3, ledger.ChannelId)
				require.Equal(t, 2, ledger.SupplierId)
				require.Equal(t, "gb10-4t-self-hosted", ledger.SupplyNode)
			}
		}

		apiLedgers := e2eAPIRequest[e2eTenantUsageLedgersResponse](t, tokenRouter.URL, http.MethodGet,
			fmt.Sprintf("/api/tenant/%d/usage_ledgers?limit=200", scenario.tenant.Id), scenario.owner, nil)
		require.Equal(t, int64(expectedPerTenant), apiLedgers.Total)
		require.Len(t, apiLedgers.Items, expectedPerTenant)

		overview := e2eAPIRequest[e2eTenantOverviewResponse](t, tokenRouter.URL, http.MethodGet,
			fmt.Sprintf("/api/tenant/%d/overview", scenario.tenant.Id), scenario.owner, nil)
		totalSell, totalPostpaid := sumLedgerQuotaE2E(ledgers)
		require.Equal(t, int64(expectedPerTenant), overview.Usage.RequestCount)
		require.Equal(t, totalSell, overview.Usage.SellQuota)
		require.Equal(t, totalPostpaid, overview.Usage.PostpaidQuota)
		waitTenantCreditAccountE2E(t, scenario.tenant.Id, func(account *model.CreditAccount) bool {
			return account.UnbilledAmount == totalPostpaid && account.AvailableCredit == account.CreditLimit-totalPostpaid
		})

		statement := e2eAPIRequest[model.BillingStatement](t, tokenRouter.URL, http.MethodPost,
			fmt.Sprintf("/api/tenant/%d/billing/statements/generate", scenario.tenant.Id),
			scenario.owner, map[string]any{
				"period_start": periodStart,
				"period_end":   common.GetTimestamp() + 3600,
			})
		require.Equal(t, totalPostpaid, statement.Amount)
		require.Equal(t, totalPostpaid, statement.Payable)
		require.Equal(t, model.BillingStatementStatusDraft, statement.Status)

		confirmed := e2eAPIRequest[model.BillingStatement](t, tokenRouter.URL, http.MethodPost,
			fmt.Sprintf("/api/tenant/%d/billing/statements/%d/confirm", scenario.tenant.Id, statement.Id),
			scenario.owner, nil)
		require.Equal(t, model.BillingStatementStatusConfirmed, confirmed.Status)
		waitTenantCreditAccountE2E(t, scenario.tenant.Id, func(account *model.CreditAccount) bool {
			return account.UnbilledAmount == 0 && account.BilledUnpaidAmount == totalPostpaid
		})

		paid := e2eAPIRequest[model.BillingStatement](t, tokenRouter.URL, http.MethodPost,
			fmt.Sprintf("/api/tenant/%d/billing/statements/%d/payment", scenario.tenant.Id, statement.Id),
			scenario.owner, map[string]any{
				"amount":         totalPostpaid,
				"method":         "bank_transfer",
				"invoice_no":     fmt.Sprintf("INV-B2B2C-%d", scenario.tenant.Id),
				"invoice_status": "issued",
			})
		require.Equal(t, model.BillingStatementStatusPaid, paid.Status)
		waitTenantCreditAccountE2E(t, scenario.tenant.Id, func(account *model.CreditAccount) bool {
			return account.UnbilledAmount == 0 && account.BilledUnpaidAmount == 0 && account.AvailableCredit == account.CreditLimit
		})

		auditLogs := e2eAPIRequest[[]model.AuditLog](t, tokenRouter.URL, http.MethodGet,
			fmt.Sprintf("/api/tenant/%d/audit_logs", scenario.tenant.Id), scenario.owner, nil)
		require.NotEmpty(t, auditLogs)
	}

	disabledKey := scenarios[0].customers[0].keys[0]
	e2eAPIRequest[bool](t, tokenRouter.URL, http.MethodPatch,
		fmt.Sprintf("/api/tenant/%d/api_keys/%d/status", scenarios[0].tenant.Id, disabledKey.token.Id),
		scenarios[0].owner, map[string]any{"status": common.TokenStatusDisabled})
	status, err := e2eTenantChatCompletion(tokenRouter.URL, disabledKey.secret, "b2b2c-disabled-key", "b2b2c-disabled-key-r0", e2eModelName)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, status)

	activeKey := scenarios[1].customers[0].keys[0]
	status, err = e2eTenantChatCompletion(tokenRouter.URL, activeKey.secret, "b2b2c-forbidden-model", "b2b2c-forbidden-model-r0", "gpt-forbidden")
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, status)

	e2eExpectAPIFailure(t, tokenRouter.URL, http.MethodGet,
		fmt.Sprintf("/api/tenant/%d/usage_ledgers?limit=10", scenarios[0].tenant.Id),
		intruder, nil, "没有权限访问该租户")
}

func createB2B2CTenantScenario(t *testing.T, baseURL string, tenantIndex int) e2eTenantScenario {
	t.Helper()
	owner := seedE2EAccessActor(t, 9100+tenantIndex, fmt.Sprintf("b2b2c-b%d-owner", tenantIndex), common.RoleCommonUser)
	scenario := e2eTenantScenario{owner: owner}
	scenario.tenant = e2eAPIRequest[model.Tenant](t, baseURL, http.MethodPost, "/api/platform/tenants",
		e2eAccessActor{id: 1, accessToken: e2eAdminToken}, service.TenantCreateInput{
			Name:         fmt.Sprintf("B2B2C Tenant %d", tenantIndex),
			Type:         "enterprise",
			Industry:     "ai-router-e2e",
			OwnerUserId:  owner.id,
			BillingMode:  model.BillingModePostpaid,
			CreditLimit:  10_000_000,
			StatementDay: 1,
			PaymentTerms: 30,
		})
	scenario.modelPolicy = e2eAPIRequest[model.TenantModelPolicy](t, baseURL, http.MethodPost,
		fmt.Sprintf("/api/platform/tenants/%d/model_policies", scenario.tenant.Id),
		e2eAccessActor{id: 1, accessToken: e2eAdminToken}, model.TenantModelPolicy{
			ModelName: e2eModelName,
			Visible:   true,
			Enabled:   true,
		})
	policies := e2eAPIRequest[[]model.TenantModelPolicy](t, baseURL, http.MethodGet,
		fmt.Sprintf("/api/tenant/%d/model_policies", scenario.tenant.Id), owner, nil)
	require.Len(t, policies, 1)
	scenario.modelPolicy = policies[0]

	workspaces := e2eAPIRequest[[]map[string]any](t, baseURL, http.MethodGet, "/api/workspaces", owner, nil)
	require.True(t, e2eWorkspaceContainsTenant(workspaces, scenario.tenant.Id))

	scenario.app = e2eAPIRequest[model.TenantApp](t, baseURL, http.MethodPost,
		fmt.Sprintf("/api/tenant/%d/apps", scenario.tenant.Id), owner, model.TenantApp{
			Name:    fmt.Sprintf("tenant-%d-production-app", tenantIndex),
			Env:     "prod",
			OwnerId: owner.id,
			Status:  model.TenantStatusActive,
		})

	for customerIndex := 0; customerIndex < b2b2cCustomersPerB; customerIndex++ {
		userID := 10_000 + tenantIndex*100 + customerIndex
		seedE2EPlainUser(t, userID, fmt.Sprintf("b2b2c-b%d-c%d", tenantIndex, customerIndex))
		customer := e2eAPIRequest[model.TenantEndCustomer](t, baseURL, http.MethodPost,
			fmt.Sprintf("/api/tenant/%d/end_customers", scenario.tenant.Id), owner, model.TenantEndCustomer{
				UserId:       userID,
				CustomerType: "employee",
				Status:       model.TenantStatusActive,
				ExternalId:   fmt.Sprintf("tenant-%d-employee-%d", tenantIndex, customerIndex),
			})
		customerScenario := e2eCustomerScenario{userID: userID, customer: customer}
		for keyIndex := 0; keyIndex < b2b2cKeysPerCustomer; keyIndex++ {
			keyResp := e2eAPIRequest[e2eTenantAPIKeyResponse](t, baseURL, http.MethodPost,
				fmt.Sprintf("/api/tenant/%d/api_keys", scenario.tenant.Id), owner, service.TenantAPIKeyInput{
					UserId:         userID,
					Name:           fmt.Sprintf("b%d-c%d-key%d", tenantIndex, customerIndex, keyIndex),
					AppId:          scenario.app.Id,
					EndCustomerId:  customer.Id,
					OwnerScope:     model.TokenOwnerScopeEndCustomer,
					ModelPolicyId:  scenario.modelPolicy.Id,
					UnlimitedQuota: true,
					Group:          "default",
				})
			require.NotEmpty(t, keyResp.SecretKey)
			customerScenario.keys = append(customerScenario.keys, e2eKeyScenario{token: keyResp.Token, secret: keyResp.SecretKey})
		}
		scenario.customers = append(scenario.customers, customerScenario)
	}

	customers := e2eAPIRequest[[]model.TenantEndCustomer](t, baseURL, http.MethodGet,
		fmt.Sprintf("/api/tenant/%d/end_customers?limit=50", scenario.tenant.Id), owner, nil)
	require.Len(t, customers, b2b2cCustomersPerB)
	keys := e2eAPIRequest[[]model.Token](t, baseURL, http.MethodGet,
		fmt.Sprintf("/api/tenant/%d/api_keys", scenario.tenant.Id), owner, nil)
	require.Len(t, keys, b2b2cCustomersPerB*b2b2cKeysPerCustomer)
	return scenario
}

func createAppliedTenantRoutingPreferenceE2E(t *testing.T, baseURL string, scenario e2eTenantScenario) {
	t.Helper()
	pref := e2eAPIRequest[model.TenantRoutingPreference](t, baseURL, http.MethodPost,
		fmt.Sprintf("/api/tenant/%d/routing_preferences", scenario.tenant.Id), scenario.owner, model.TenantRoutingPreference{
			ModelName:           e2eModelName,
			PreferredSupplierId: 2,
			PreferredChannelId:  3,
			Reason:              "e2e tenant prefers self-hosted channel",
		})
	applied := e2eAPIRequest[model.TenantRoutingPreference](t, baseURL, http.MethodPost,
		fmt.Sprintf("/api/tenant/%d/routing_preferences/%d/review", scenario.tenant.Id, pref.Id),
		scenario.owner, map[string]any{"status": model.TenantRoutingStatusApplied, "note": "apply in b2b2c e2e"})
	require.Equal(t, model.TenantRoutingStatusApplied, applied.Status)
}

func seedE2EAccessActor(t *testing.T, id int, username string, role int) e2eAccessActor {
	t.Helper()
	accessToken := fmt.Sprintf("access-token-b2b2c-%06d", id)
	user := &model.User{
		Id:          id,
		Username:    username,
		Password:    "unused-password",
		Role:        role,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		Quota:       1_000_000,
		AffCode:     fmt.Sprintf("aff-b2b2c-%d", id),
		AccessToken: &accessToken,
	}
	require.NoError(t, model.DB.Create(user).Error)
	return e2eAccessActor{id: id, accessToken: accessToken}
}

func seedE2EPlainUser(t *testing.T, id int, username string) {
	t.Helper()
	user := &model.User{
		Id:       id,
		Username: username,
		Password: "unused-password",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		Quota:    1_000_000,
		AffCode:  fmt.Sprintf("aff-b2b2c-%d", id),
	}
	require.NoError(t, model.DB.Create(user).Error)
}

func e2eAPIRequest[T any](t *testing.T, baseURL string, method string, path string, actor e2eAccessActor, body any) T {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(method, baseURL+path, bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+actor.accessToken)
	req.Header.Set("New-Api-User", strconv.Itoa(actor.id))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    T      `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.True(t, envelope.Success, envelope.Message)
	return envelope.Data
}

func e2eExpectAPIFailure(t *testing.T, baseURL string, method string, path string, actor e2eAccessActor, body any, messageContains string) {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(method, baseURL+path, bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+actor.accessToken)
	req.Header.Set("New-Api-User", strconv.Itoa(actor.id))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var envelope struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&envelope))
	require.False(t, envelope.Success)
	require.Contains(t, envelope.Message, messageContains)
}

func e2eTenantChatCompletion(baseURL string, secret string, sessionID string, requestID string, modelName string) (int, error) {
	body := map[string]any{
		"model": modelName,
		"messages": []map[string]string{{
			"role":    "user",
			"content": "hello from b2b2c tenant e2e",
		}},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("X-Session-Id", sessionID)
	req.Header.Set(common.UsageLedgerRequestIdHeader, requestID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		var decoded map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
			return resp.StatusCode, err
		}
		if decoded["choices"] == nil {
			return resp.StatusCode, fmt.Errorf("chat response has no choices")
		}
	}
	return resp.StatusCode, nil
}

func waitTenantLedgerCountE2E(t *testing.T, tenantID int, expected int) []*model.UsageLedger {
	t.Helper()
	var ledgers []*model.UsageLedger
	require.Eventually(t, func() bool {
		items, total, err := model.SearchUsageLedgers(model.UsageLedgerFilters{TenantId: tenantID, Status: "success"}, 0, expected+10)
		if err != nil {
			return false
		}
		ledgers = items
		return total == int64(expected) && len(items) == expected
	}, 5*time.Second, 50*time.Millisecond)
	return ledgers
}

func waitTenantCreditAccountE2E(t *testing.T, tenantID int, condition func(account *model.CreditAccount) bool) model.CreditAccount {
	t.Helper()
	var latest model.CreditAccount
	var lastErr error
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		account, err := model.GetCreditAccountByTenantId(tenantID)
		if err != nil {
			lastErr = err
			time.Sleep(50 * time.Millisecond)
			continue
		}
		latest = *account
		if condition(account) {
			return latest
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.Failf(t, "tenant credit account condition not met", "tenant=%d latest credit account=%+v err=%v", tenantID, latest, lastErr)
	return latest
}

func assertTenantLedgerBreakdownE2E(t *testing.T, scenario e2eTenantScenario, ledgers []*model.UsageLedger) {
	t.Helper()
	expectedPerCustomer := b2b2cKeysPerCustomer * b2b2cRequestsPerKey
	customerCounts := map[int]int{}
	tokenCounts := map[int]int{}
	for _, ledger := range ledgers {
		require.Equal(t, scenario.tenant.Id, ledger.TenantId)
		require.Equal(t, scenario.app.Id, ledger.AppId)
		require.Equal(t, model.BillingModePostpaid, ledger.BillingMode)
		require.Equal(t, ledger.SellQuota, ledger.PostpaidQuota)
		require.Greater(t, ledger.SellQuota, 0)
		customerCounts[ledger.EndCustomerId]++
		tokenCounts[ledger.TokenId]++
	}
	for _, customer := range scenario.customers {
		require.Equal(t, expectedPerCustomer, customerCounts[customer.customer.Id])
		for _, key := range customer.keys {
			require.Equal(t, b2b2cRequestsPerKey, tokenCounts[key.token.Id])
		}
	}
}

func sumLedgerQuotaE2E(ledgers []*model.UsageLedger) (int64, int64) {
	var sell int64
	var postpaid int64
	for _, ledger := range ledgers {
		sell += int64(ledger.SellQuota)
		postpaid += int64(ledger.PostpaidQuota)
	}
	return sell, postpaid
}

func e2eWorkspaceContainsTenant(workspaces []map[string]any, tenantID int) bool {
	for _, workspace := range workspaces {
		if workspace["scope_type"] != model.ScopeTenant {
			continue
		}
		scopeID, ok := workspace["scope_id"].(float64)
		if ok && int(scopeID) == tenantID {
			return true
		}
	}
	return false
}
