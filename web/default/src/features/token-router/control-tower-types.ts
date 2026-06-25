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
