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

export async function exportEnterpriseApiKeys(
  params: EnterpriseApiKeyQuery
): Promise<void> {
  const response = await api.get('/api/enterprise/api-keys/export', {
    params,
    responseType: 'blob',
    skipBusinessError: true,
    disableDuplicate: true,
  })
  const disposition = String(response.headers['content-disposition'] ?? '')
  const filenameMatch = disposition.match(/filename="?([^";]+)"?/i)
  const filename = filenameMatch?.[1] ?? 'enterprise-api-keys.csv'
  const blob = new Blob([response.data], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
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
