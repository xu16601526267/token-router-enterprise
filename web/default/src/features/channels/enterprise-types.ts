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
export type EnterpriseChannelSummary = {
  enabled_channels: number
  healthy_suppliers: number
  average_success_rate: number
  average_latency_ms: number
  total_balance: number
  low_balance_alerts: number
}

export type EnterpriseChannelItem = {
  id: number
  name: string
  type: number
  status: number
  supplier_id: number
  supplier_name: string
  supplier_type: string
  supplier_status: number
  models: string
  group: string
  tag: string
  remark: string
  balance: number
  used_quota: number
  response_time_ms: number
  average_latency_ms: number
  requests: number
  success_rate: number
  priority: number
  weight: number
  last_checked_at: number
  balance_updated_time: number
}

export type EnterpriseChannelCenterData = {
  generated_at: number
  summary: EnterpriseChannelSummary
  items: EnterpriseChannelItem[]
  total: number
  page: number
  page_size: number
}

export type EnterpriseSupplierDetail = {
  id: number
  name: string
  type: string
  status: number
  notes: string
  updated_time: number
  channel_count: number
  total_balance: number
  success_rate: number
  latency_ms: number
  score: number
  grade: string
  route_weight: number
}

export type EnterpriseChannelIncident = {
  id: number
  title: string
  severity: string
  status: string
  created_at: number
}

export type EnterpriseChannelDetail = {
  channel: EnterpriseChannelItem
  supplier: EnterpriseSupplierDetail | null
  supported_models: string[]
  incidents: EnterpriseChannelIncident[]
}

export type EnterpriseChannelResponse<T> = {
  success: boolean
  message?: string
  data?: T
}
