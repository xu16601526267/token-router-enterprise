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
export type EnterpriseOverviewMetrics = {
  total_requests: number
  total_tokens: number
  total_quota: number
  estimated_cost: number
  success_rate: number
  average_latency_ms: number
  total_users: number
  active_users: number
  total_channels: number
  healthy_channels: number
  low_balance_channels: number
  active_api_keys: number
  total_suppliers: number
  healthy_suppliers: number
  active_policies: number
  open_insights: number
  pending_approvals: number
  gross_profit_quota: number
  gross_margin_rate: number
  estimated_gross_profit: number
}

export type EnterpriseOverviewTrendPoint = {
  timestamp: number
  requests: number
  tokens: number
  quota: number
}

export type EnterpriseOverviewRankingItem = {
  name: string
  requests: number
  tokens: number
  quota: number
  share: number
}

export type EnterpriseOverviewChannelItem = {
  id: number
  name: string
  status: number
  response_time: number
  balance: number
  used_quota: number
  models: string
  group: string
}

export type EnterpriseOverviewInsight = {
  id: number
  title: string
  summary: string
  severity: string
  category: string
  model_name: string
  recommended_action: string
  sla_met_rate: number
  generated_at: number
}

export type EnterpriseOverviewData = {
  generated_at: number
  range: {
    start_timestamp: number
    end_timestamp: number
    time_granularity?: string
  }
  metrics: EnterpriseOverviewMetrics
  trend: EnterpriseOverviewTrendPoint[]
  top_models: EnterpriseOverviewRankingItem[]
  top_users: EnterpriseOverviewRankingItem[]
  channels: EnterpriseOverviewChannelItem[]
  insights: EnterpriseOverviewInsight[]
}

export type EnterpriseApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}

export type EnterpriseUsageMetrics = {
  total_requests: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  total_quota: number
  estimated_cost: number
  error_requests: number
  error_rate: number
  average_latency_ms: number
  cache_hit_rate: number
}

export type EnterpriseUsageTrendPoint = {
  timestamp: number
  requests: number
  errors: number
  prompt_tokens: number
  completion_tokens: number
  quota: number
  average_latency_ms: number
}

export type EnterpriseUsageBreakdownItem = {
  id?: number
  name: string
  quota: number
  cost: number
  share: number
}

export type EnterpriseUsageLogItem = {
  id: number
  request_id: string
  created_at: number
  username: string
  group: string
  token_name: string
  model_name: string
  prompt_tokens: number
  completion_tokens: number
  quota: number
  channel_id: number
  channel_name: string
  use_time_ms: number
  status: string
  ip: string
}

export type EnterpriseUsageAnalyticsData = {
  generated_at: number
  range: { start_timestamp: number; end_timestamp: number }
  metrics: EnterpriseUsageMetrics
  trend: EnterpriseUsageTrendPoint[]
  by_model: EnterpriseUsageBreakdownItem[]
  by_user: EnterpriseUsageBreakdownItem[]
  by_channel: EnterpriseUsageBreakdownItem[]
  by_group: EnterpriseUsageBreakdownItem[]
  recent_logs: EnterpriseUsageLogItem[]
  total_logs: number
  page: number
  page_size: number
}

export type EnterpriseUserItem = {
  id: number
  username: string
  display_name: string
  email: string
  group: string
  role: number
  status: number
  api_key_count: number
  quota: number
  used_quota: number
  request_count: number
  last_login_at: number
}

export type EnterpriseCountItem = {
  name: string
  count: number
}

export type EnterpriseUsersData = {
  generated_at: number
  summary: {
    total_users: number
    active_users: number
    admin_users: number
    disabled_users: number
    active_api_keys: number
    groups: number
  }
  users: EnterpriseUserItem[]
  role_counts: EnterpriseCountItem[]
  group_counts: EnterpriseCountItem[]
}

export type EnterpriseBillingTrendPoint = {
  timestamp: number
  sell_quota: number
  cost_quota: number
  gross_profit_quota: number
}

export type EnterpriseSettlementItem = {
  id: number
  subject_type: string
  subject_id: number
  subject_name: string
  period_start: number
  period_end: number
  total_sell_quota: number
  total_cost_quota: number
  gross_profit_quota: number
  total_requests: number
  status: string
}

export type EnterpriseTopUpItem = {
  id: number
  user_id: number
  username: string
  money: number
  payment_method: string
  payment_provider: string
  status: string
  create_time: number
}

export type EnterpriseBillingData = {
  generated_at: number
  range: { start_timestamp: number; end_timestamp: number }
  metrics: {
    total_balance_quota: number
    total_used_quota: number
    period_sell_quota: number
    period_cost_quota: number
    period_gross_profit_quota: number
    gross_margin_rate: number
    successful_top_up_amount: number
    pending_top_up_amount: number
    active_subscriptions: number
    draft_settlements: number
  }
  trend: EnterpriseBillingTrendPoint[]
  settlements: EnterpriseSettlementItem[]
  recent_topups: EnterpriseTopUpItem[]
}

export type TenantStatus = 'active' | 'suspended' | 'disabled'

export type TenantBillingMode = 'prepaid' | 'postpaid' | 'mixed'

export type PlatformTenant = {
  id: number
  name: string
  type: string
  status: TenantStatus | string
  industry: string
  owner_user_id: number
  brand_config: string
  domain: string
  contract_no: string
  created_at: number
  updated_at: number
}

export type TenantBillingConfig = {
  id: number
  tenant_id: number
  billing_mode: TenantBillingMode | string
  billing_cycle: string
  statement_day: number
  payment_terms: number
  credit_limit: number
  over_credit_policy: string
  created_at: number
  updated_at: number
}

export type TenantCreditAccount = {
  id: number
  tenant_id: number
  credit_limit: number
  unbilled_amount: number
  billed_unpaid_amount: number
  overdue_amount: number
  available_credit: number
  status: string
  created_at: number
  updated_at: number
}

export type PlatformTenant360 = {
  tenant: PlatformTenant
  billing_config: TenantBillingConfig | null
  credit_account: TenantCreditAccount | null
  members: number
  end_customers: number
  apps: number
  api_keys: number
  usage_ledger_count: number
}

export type PlatformTenantCreateInput = {
  name: string
  type: string
  industry?: string
  owner_user_id?: number
  brand_config?: string
  domain?: string
  contract_no?: string
  billing_mode: TenantBillingMode
  credit_limit?: number
  statement_day?: number
  payment_terms?: number
}

export type PlatformTenantBillingConfigInput = {
  billing_mode: TenantBillingMode
  billing_cycle: string
  statement_day: number
  payment_terms: number
  credit_limit: number
  over_credit_policy: string
}

export type TenantModelPolicy = {
  id: number
  tenant_id: number
  model_id: number
  model_name: string
  visible: boolean
  price_plan_id: number
  rate_limit: string
  alias: string
  enabled: boolean
  created_at: number
  updated_at: number
}

export type TenantModelPolicyInput = {
  model_id?: number
  model_name: string
  visible: boolean
  price_plan_id?: number
  rate_limit?: string
  alias?: string
  enabled: boolean
}
