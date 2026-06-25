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
}

export type EnterpriseApiKeySummary = {
  total: number
  active: number
  expiring_soon: number
  exhausted: number
  disabled: number
  active_users: number
  total_used_quota: number
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
