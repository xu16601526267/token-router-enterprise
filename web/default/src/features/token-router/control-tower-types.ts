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
export type ControlTowerRange = {
  start_timestamp: number
  end_timestamp: number
}

export type ControlTowerMetrics = {
  active_policies: number
  realtime_success_rate: number
  average_latency_ms: number
  automatic_switches: number
  pending_approvals: number
  requests: number
  tokens: number
}

export type ControlTowerTrendPoint = {
  timestamp: number
  requests: number
  success_rate: number
  latency_ms: number
}

export type RoutingPolicyItem = {
  id: number
  name: string
  slice_key: string
  model_name: string
  sla_tier: string
  track: string
  action_type: string
  status: string
  supplier_id: number
  supplier_name: string
  channel_id: number
  channel_name: string
  priority: number
  traffic_percent: number
  effective_from: number
  effective_to: number
  updated_at: number
  reason: string
}

export type ProviderHealth = {
  channel_id: number
  channel_name: string
  supplier_id: number
  supplier_name: string
  status: number
  requests: number
  success_rate: number
  average_latency_ms: number
  response_time_ms: number
  balance: number
  models: string
  region: string
}

export type ControlTowerEvent = {
  id: number
  title: string
  detail: string
  category: string
  severity: string
  status: string
  created_at: number
}

export type ControlTowerData = {
  generated_at: number
  range: ControlTowerRange
  metrics: ControlTowerMetrics
  trend: ControlTowerTrendPoint[]
  policies: RoutingPolicyItem[]
  provider_health: ProviderHealth[]
  recent_changes: ControlTowerEvent[]
  pending_actions: ControlTowerEvent[]
  risks: ControlTowerEvent[]
}

export type ControlTowerResponse = {
  success: boolean
  message?: string
  data?: ControlTowerData
}
