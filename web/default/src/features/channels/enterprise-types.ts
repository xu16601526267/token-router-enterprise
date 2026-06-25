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
