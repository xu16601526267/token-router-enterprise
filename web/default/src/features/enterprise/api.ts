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
import { api } from '@/lib/api'
import type {
  EnterpriseApiResponse,
  EnterpriseBillingData,
  EnterpriseOverviewData,
  EnterpriseUsageAnalyticsData,
  EnterpriseUsersData,
} from './types'

type EnterpriseRangeParams = {
  start_timestamp: number
  end_timestamp: number
}

async function downloadEnterpriseCsv(
  path: string,
  params: Record<string, number | string | undefined>,
  fallbackFilename: string
): Promise<void> {
  const response = await api.get(path, {
    params,
    responseType: 'blob',
    skipBusinessError: true,
    disableDuplicate: true,
  })
  const disposition = String(response.headers['content-disposition'] ?? '')
  const filenameMatch = disposition.match(/filename="?([^";]+)"?/i)
  const filename = filenameMatch?.[1] ?? fallbackFilename
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

export async function getEnterpriseOverview(
  params: EnterpriseRangeParams
): Promise<EnterpriseApiResponse<EnterpriseOverviewData>> {
  const res = await api.get('/api/enterprise/overview', { params })
  return res.data
}

export async function getEnterpriseUsageAnalytics(
  params: EnterpriseRangeParams
): Promise<EnterpriseApiResponse<EnterpriseUsageAnalyticsData>> {
  const res = await api.get('/api/enterprise/usage-analytics', { params })
  return res.data
}

export async function getEnterpriseUsers(params: {
  limit?: number
} = {}): Promise<EnterpriseApiResponse<EnterpriseUsersData>> {
  const res = await api.get('/api/enterprise/users', { params })
  return res.data
}

export async function getEnterpriseBilling(
  params: EnterpriseRangeParams
): Promise<EnterpriseApiResponse<EnterpriseBillingData>> {
  const res = await api.get('/api/enterprise/billing', { params })
  return res.data
}

export async function exportEnterpriseUsageAnalytics(
  params: EnterpriseRangeParams
): Promise<void> {
  await downloadEnterpriseCsv(
    '/api/enterprise/usage-analytics/export',
    params,
    'enterprise-usage.csv'
  )
}

export async function exportEnterpriseBilling(
  params: EnterpriseRangeParams
): Promise<void> {
  await downloadEnterpriseCsv(
    '/api/enterprise/billing/export',
    params,
    'enterprise-billing.csv'
  )
}
