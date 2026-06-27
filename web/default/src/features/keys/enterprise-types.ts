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
export type EnterpriseApiKeyInput = {
  user_id: number
  name: string
  status: number
  expired_time: number
  remain_quota: number
  unlimited_quota: boolean
  model_limits_enabled: boolean
  model_limits: string
  allow_ips: string | null
  group: string
  cross_group_retry: boolean
  rate_limit: string
}

export type EnterpriseApiKeyItem = EnterpriseApiKeyInput & {
  id: number
  masked_key: string
  effective_status: number
  created_time: number
  accessed_time: number
  used_quota: number
  username: string
  display_name: string
  email: string
  user_group: string
  recent_failure_count: number
}

export type EnterpriseApiKeySummary = {
  total: number
  active: number
  expiring_soon: number
  exhausted: number
  disabled: number
  active_users: number
  total_used_quota: number
  rate_limit_hits: number
}

export type EnterpriseApiKeyPage = {
  items: EnterpriseApiKeyItem[]
  total: number
  page: number
  page_size: number
  summary: EnterpriseApiKeySummary
}

export type EnterpriseApiKeyUser = {
  id: number
  username: string
  display_name: string
  email: string
  group: string
  status: number
  role: number
}

export type EnterpriseApiKeySecret = {
  item: EnterpriseApiKeyItem
  secret_key: string
}

export type EnterpriseApiResponse<T> = {
  success: boolean
  message?: string
  data?: T
}
