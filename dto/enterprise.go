/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package dto

// Enterprise API key governance ------------------------------------------------

type EnterpriseAPIKeyMutationInput struct {
	UserId             int     `json:"user_id"`
	Name               string  `json:"name"`
	Status             int     `json:"status"`
	ExpiredTime        int64   `json:"expired_time"`
	RemainQuota        int     `json:"remain_quota"`
	UnlimitedQuota     bool    `json:"unlimited_quota"`
	ModelLimitsEnabled bool    `json:"model_limits_enabled"`
	ModelLimits        string  `json:"model_limits"`
	AllowIps           *string `json:"allow_ips"`
	Group              string  `json:"group"`
	CrossGroupRetry    bool    `json:"cross_group_retry"`
	RateLimit          string  `json:"rate_limit"`
}

type EnterpriseAPIKeyItem struct {
	Id                 int     `json:"id"`
	UserId             int     `json:"user_id"`
	Name               string  `json:"name"`
	MaskedKey          string  `json:"masked_key"`
	Status             int     `json:"status"`
	EffectiveStatus    int     `json:"effective_status"`
	CreatedTime        int64   `json:"created_time"`
	AccessedTime       int64   `json:"accessed_time"`
	ExpiredTime        int64   `json:"expired_time"`
	RemainQuota        int     `json:"remain_quota"`
	UsedQuota          int     `json:"used_quota"`
	UnlimitedQuota     bool    `json:"unlimited_quota"`
	ModelLimitsEnabled bool    `json:"model_limits_enabled"`
	ModelLimits        string  `json:"model_limits"`
	AllowIps           *string `json:"allow_ips"`
	Group              string  `json:"group"`
	CrossGroupRetry    bool    `json:"cross_group_retry"`
	RateLimit          string  `json:"rate_limit"`
	RecentFailureCount int64   `json:"recent_failure_count"`
	Username           string  `json:"username"`
	DisplayName        string  `json:"display_name"`
	Email              string  `json:"email"`
	UserGroup          string  `json:"user_group"`
}

type EnterpriseAPIKeySummary struct {
	Total          int64 `json:"total"`
	Active         int64 `json:"active"`
	ExpiringSoon   int64 `json:"expiring_soon"`
	Exhausted      int64 `json:"exhausted"`
	Disabled       int64 `json:"disabled"`
	ActiveUsers    int64 `json:"active_users"`
	TotalUsedQuota int64 `json:"total_used_quota"`
}

type EnterpriseAPIKeyPage struct {
	Items    []EnterpriseAPIKeyItem  `json:"items"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
	Summary  EnterpriseAPIKeySummary `json:"summary"`
}

type EnterpriseAPIKeyUser struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Group       string `json:"group"`
	Status      int    `json:"status"`
	Role        int    `json:"role"`
}

type EnterpriseAPIKeySecret struct {
	Item      EnterpriseAPIKeyItem `json:"item"`
	SecretKey string               `json:"secret_key"`
}

// Channel and supplier center --------------------------------------------------

type EnterpriseChannelSummary struct {
	EnabledChannels    int64   `json:"enabled_channels"`
	HealthySuppliers   int64   `json:"healthy_suppliers"`
	TotalSuppliers     int64   `json:"total_suppliers"`
	AverageSuccessRate float64 `json:"average_success_rate"`
	AverageLatencyMs   float64 `json:"average_latency_ms"`
	TotalBalance       float64 `json:"total_balance"`
	LowBalanceAlerts   int64   `json:"low_balance_alerts"`
}

type EnterpriseChannelItem struct {
	Id                 int     `json:"id"`
	Name               string  `json:"name"`
	Type               int     `json:"type"`
	Status             int     `json:"status"`
	SupplierId         int     `json:"supplier_id"`
	SupplierName       string  `json:"supplier_name"`
	SupplierType       string  `json:"supplier_type"`
	SupplierStatus     int     `json:"supplier_status"`
	Models             string  `json:"models"`
	Group              string  `json:"group"`
	Tag                string  `json:"tag"`
	Remark             string  `json:"remark"`
	Balance            float64 `json:"balance"`
	UsedQuota          int64   `json:"used_quota"`
	ResponseTimeMs     int     `json:"response_time_ms"`
	AverageLatencyMs   float64 `json:"average_latency_ms"`
	Requests           int64   `json:"requests"`
	SuccessRate        float64 `json:"success_rate"`
	Priority           int64   `json:"priority"`
	Weight             uint    `json:"weight"`
	LastCheckedAt      int64   `json:"last_checked_at"`
	BalanceUpdatedTime int64   `json:"balance_updated_time"`
}

type EnterpriseChannelCenterData struct {
	GeneratedAt int64                    `json:"generated_at"`
	Summary     EnterpriseChannelSummary `json:"summary"`
	Items       []EnterpriseChannelItem  `json:"items"`
	Total       int64                    `json:"total"`
	Page        int                      `json:"page"`
	PageSize    int                      `json:"page_size"`
}

type EnterpriseSupplierDetail struct {
	Id           int     `json:"id"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Status       int     `json:"status"`
	Notes        string  `json:"notes"`
	UpdatedTime  int64   `json:"updated_time"`
	ChannelCount int64   `json:"channel_count"`
	TotalBalance float64 `json:"total_balance"`
	SuccessRate  float64 `json:"success_rate"`
	LatencyMs    float64 `json:"latency_ms"`
	Score        float64 `json:"score"`
	Grade        string  `json:"grade"`
	RouteWeight  int     `json:"route_weight"`
}

type EnterpriseChannelIncident struct {
	Id        int    `json:"id"`
	Title     string `json:"title"`
	Severity  string `json:"severity"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

type EnterpriseChannelDetail struct {
	Channel         EnterpriseChannelItem       `json:"channel"`
	Supplier        *EnterpriseSupplierDetail   `json:"supplier"`
	SupportedModels []string                    `json:"supported_models"`
	Incidents       []EnterpriseChannelIncident `json:"incidents"`
}

// Token Router control tower ---------------------------------------------------

type EnterpriseControlTowerRange struct {
	StartTimestamp int64 `json:"start_timestamp"`
	EndTimestamp   int64 `json:"end_timestamp"`
}

type EnterpriseControlTowerMetrics struct {
	ActivePolicies      int64   `json:"active_policies"`
	RealtimeSuccessRate float64 `json:"realtime_success_rate"`
	AverageLatencyMs    float64 `json:"average_latency_ms"`
	AutomaticSwitches   int64   `json:"automatic_switches"`
	PendingApprovals    int64   `json:"pending_approvals"`
	Requests            int64   `json:"requests"`
	Tokens              int64   `json:"tokens"`
}

type EnterpriseControlTowerTrendPoint struct {
	Timestamp   int64   `json:"timestamp"`
	Requests    int64   `json:"requests"`
	SuccessRate float64 `json:"success_rate"`
	LatencyMs   float64 `json:"latency_ms"`
}

type EnterpriseRoutingPolicyItem struct {
	Id             int    `json:"id"`
	Name           string `json:"name"`
	SliceKey       string `json:"slice_key"`
	ModelName      string `json:"model_name"`
	SlaTier        string `json:"sla_tier"`
	Track          string `json:"track"`
	ActionType     string `json:"action_type"`
	Status         string `json:"status"`
	SupplierId     int    `json:"supplier_id"`
	SupplierName   string `json:"supplier_name"`
	ChannelId      int    `json:"channel_id"`
	ChannelName    string `json:"channel_name"`
	Priority       int    `json:"priority"`
	TrafficPercent int    `json:"traffic_percent"`
	EffectiveFrom  int64  `json:"effective_from"`
	EffectiveTo    int64  `json:"effective_to"`
	UpdatedAt      int64  `json:"updated_at"`
	Reason         string `json:"reason"`
}

type EnterpriseProviderHealth struct {
	ChannelId        int     `json:"channel_id"`
	ChannelName      string  `json:"channel_name"`
	SupplierId       int     `json:"supplier_id"`
	SupplierName     string  `json:"supplier_name"`
	Status           int     `json:"status"`
	Requests         int64   `json:"requests"`
	SuccessRate      float64 `json:"success_rate"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	ResponseTimeMs   int     `json:"response_time_ms"`
	Balance          float64 `json:"balance"`
	Models           string  `json:"models"`
	Region           string  `json:"region"`
}

type EnterpriseControlTowerEvent struct {
	Id        int    `json:"id"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	Category  string `json:"category"`
	Severity  string `json:"severity"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

type EnterpriseControlTowerData struct {
	GeneratedAt    int64                              `json:"generated_at"`
	Range          EnterpriseControlTowerRange        `json:"range"`
	Metrics        EnterpriseControlTowerMetrics      `json:"metrics"`
	Trend          []EnterpriseControlTowerTrendPoint `json:"trend"`
	Policies       []EnterpriseRoutingPolicyItem      `json:"policies"`
	ProviderHealth []EnterpriseProviderHealth         `json:"provider_health"`
	RecentChanges  []EnterpriseControlTowerEvent      `json:"recent_changes"`
	PendingActions []EnterpriseControlTowerEvent      `json:"pending_actions"`
	Risks          []EnterpriseControlTowerEvent      `json:"risks"`
}

// Usage and cost analytics -----------------------------------------------------

type EnterpriseUsageRange struct {
	StartTimestamp int64 `json:"start_timestamp"`
	EndTimestamp   int64 `json:"end_timestamp"`
}

type EnterpriseUsageMetrics struct {
	TotalRequests    int64   `json:"total_requests"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalQuota       int64   `json:"total_quota"`
	EstimatedCost    float64 `json:"estimated_cost"`
	ErrorRequests    int64   `json:"error_requests"`
	ErrorRate        float64 `json:"error_rate"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
}

type EnterpriseUsageTrendPoint struct {
	Timestamp        int64   `json:"timestamp"`
	Requests         int64   `json:"requests"`
	Errors           int64   `json:"errors"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	Quota            int64   `json:"quota"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
}

type EnterpriseUsageBreakdownItem struct {
	Id    int     `json:"id,omitempty"`
	Name  string  `json:"name"`
	Quota int64   `json:"quota"`
	Cost  float64 `json:"cost"`
	Share float64 `json:"share"`
}

type EnterpriseUsageLogItem struct {
	Id               int    `json:"id"`
	RequestId        string `json:"request_id"`
	CreatedAt        int64  `json:"created_at"`
	Username         string `json:"username"`
	Group            string `json:"group"`
	TokenName        string `json:"token_name"`
	ModelName        string `json:"model_name"`
	RequestType      string `json:"request_type"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	Quota            int    `json:"quota"`
	ChannelId        int    `json:"channel_id"`
	ChannelName      string `json:"channel_name"`
	UseTimeMs        int    `json:"use_time_ms"`
	Status           string `json:"status"`
	Ip               string `json:"ip"`
}

type EnterpriseUsageAnalyticsData struct {
	GeneratedAt int64                          `json:"generated_at"`
	Range       EnterpriseUsageRange           `json:"range"`
	Metrics     EnterpriseUsageMetrics         `json:"metrics"`
	Trend       []EnterpriseUsageTrendPoint    `json:"trend"`
	ByModel     []EnterpriseUsageBreakdownItem `json:"by_model"`
	ByUser      []EnterpriseUsageBreakdownItem `json:"by_user"`
	ByChannel   []EnterpriseUsageBreakdownItem `json:"by_channel"`
	ByGroup     []EnterpriseUsageBreakdownItem `json:"by_group"`
	RecentLogs  []EnterpriseUsageLogItem       `json:"recent_logs"`
	TotalLogs   int64                          `json:"total_logs"`
	Page        int                            `json:"page"`
	PageSize    int                            `json:"page_size"`
}

// Users and access governance --------------------------------------------------

type EnterpriseUserSummary struct {
	TotalUsers    int64 `json:"total_users"`
	ActiveUsers   int64 `json:"active_users"`
	AdminUsers    int64 `json:"admin_users"`
	DisabledUsers int64 `json:"disabled_users"`
	ActiveAPIKeys int64 `json:"active_api_keys"`
	Groups        int64 `json:"groups"`
}

type EnterpriseUserItem struct {
	Id           int    `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Email        string `json:"email"`
	Group        string `json:"group"`
	Role         int    `json:"role"`
	Status       int    `json:"status"`
	APIKeyCount  int64  `json:"api_key_count"`
	Quota        int    `json:"quota"`
	UsedQuota    int    `json:"used_quota"`
	RequestCount int    `json:"request_count"`
	LastLoginAt  int64  `json:"last_login_at"`
}

type EnterpriseCountItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type EnterpriseUsersData struct {
	GeneratedAt int64                 `json:"generated_at"`
	Summary     EnterpriseUserSummary `json:"summary"`
	Users       []EnterpriseUserItem  `json:"users"`
	RoleCounts  []EnterpriseCountItem `json:"role_counts"`
	GroupCounts []EnterpriseCountItem `json:"group_counts"`
}

// Billing and settlement -------------------------------------------------------

type EnterpriseBillingRange struct {
	StartTimestamp  int64  `json:"start_timestamp"`
	EndTimestamp    int64  `json:"end_timestamp"`
	TimeGranularity string `json:"time_granularity,omitempty"`
}

type EnterpriseBillingMetrics struct {
	TotalBalanceQuota      int64   `json:"total_balance_quota"`
	TotalUsedQuota         int64   `json:"total_used_quota"`
	PeriodSellQuota        int64   `json:"period_sell_quota"`
	PeriodCostQuota        int64   `json:"period_cost_quota"`
	PeriodGrossProfitQuota int64   `json:"period_gross_profit_quota"`
	GrossMarginRate        float64 `json:"gross_margin_rate"`
	SuccessfulTopUpAmount  float64 `json:"successful_top_up_amount"`
	PendingTopUpAmount     float64 `json:"pending_top_up_amount"`
	ActiveSubscriptions    int64   `json:"active_subscriptions"`
	DraftSettlements       int64   `json:"draft_settlements"`
}

type EnterpriseBillingTrendPoint struct {
	Timestamp        int64 `json:"timestamp"`
	SellQuota        int64 `json:"sell_quota"`
	CostQuota        int64 `json:"cost_quota"`
	GrossProfitQuota int64 `json:"gross_profit_quota"`
}

type EnterpriseSettlementItem struct {
	Id               int    `json:"id"`
	SubjectType      string `json:"subject_type"`
	SubjectId        int    `json:"subject_id"`
	SubjectName      string `json:"subject_name"`
	PeriodStart      int64  `json:"period_start"`
	PeriodEnd        int64  `json:"period_end"`
	TotalSellQuota   int64  `json:"total_sell_quota"`
	TotalCostQuota   int64  `json:"total_cost_quota"`
	GrossProfitQuota int64  `json:"gross_profit_quota"`
	TotalRequests    int64  `json:"total_requests"`
	Status           string `json:"status"`
}

type EnterpriseTopUpItem struct {
	Id              int     `json:"id"`
	UserId          int     `json:"user_id"`
	Username        string  `json:"username"`
	Money           float64 `json:"money"`
	PaymentMethod   string  `json:"payment_method"`
	PaymentProvider string  `json:"payment_provider"`
	Status          string  `json:"status"`
	CreateTime      int64   `json:"create_time"`
}

type EnterpriseBillingData struct {
	GeneratedAt  int64                         `json:"generated_at"`
	Range        EnterpriseBillingRange        `json:"range"`
	Metrics      EnterpriseBillingMetrics      `json:"metrics"`
	Trend        []EnterpriseBillingTrendPoint `json:"trend"`
	Settlements  []EnterpriseSettlementItem    `json:"settlements"`
	RecentTopups []EnterpriseTopUpItem         `json:"recent_topups"`
}
