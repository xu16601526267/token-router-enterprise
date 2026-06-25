import { api } from '@/lib/api'
import type {
  EnterpriseApiKeyInput,
  EnterpriseApiKeyItem,
  EnterpriseApiKeyPage,
  EnterpriseApiKeySecret,
  EnterpriseApiKeyUser,
  EnterpriseApiResponse,
} from './enterprise-types'

export type EnterpriseApiKeyQuery = {
  page?: number
  page_size?: number
  keyword?: string
  status?: number
  user_id?: number
  group?: string
}

export async function getEnterpriseApiKeys(
  params: EnterpriseApiKeyQuery
): Promise<EnterpriseApiResponse<EnterpriseApiKeyPage>> {
  const response = await api.get('/api/enterprise/api-keys', { params })
  return response.data
}

export async function getEnterpriseApiKeyUsers(): Promise<
  EnterpriseApiResponse<EnterpriseApiKeyUser[]>
> {
  const response = await api.get('/api/enterprise/api-keys/users')
  return response.data
}

export async function createEnterpriseApiKey(
  input: EnterpriseApiKeyInput
): Promise<EnterpriseApiResponse<EnterpriseApiKeySecret>> {
  const response = await api.post('/api/enterprise/api-keys', input)
  return response.data
}

export async function updateEnterpriseApiKey(
  id: number,
  input: EnterpriseApiKeyInput
): Promise<EnterpriseApiResponse<EnterpriseApiKeyItem>> {
  const response = await api.put(`/api/enterprise/api-keys/${id}`, input)
  return response.data
}

export async function rotateEnterpriseApiKey(
  id: number
): Promise<EnterpriseApiResponse<EnterpriseApiKeySecret>> {
  const response = await api.post(`/api/enterprise/api-keys/${id}/rotate`)
  return response.data
}

export async function deleteEnterpriseApiKey(
  id: number
): Promise<EnterpriseApiResponse<EnterpriseApiKeyItem>> {
  const response = await api.delete(`/api/enterprise/api-keys/${id}`)
  return response.data
}
