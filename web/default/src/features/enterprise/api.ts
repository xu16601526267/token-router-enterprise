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
  EnterpriseOverviewData,
} from './types'

export async function getEnterpriseOverview(params: {
  start_timestamp: number
  end_timestamp: number
}): Promise<EnterpriseApiResponse<EnterpriseOverviewData>> {
  const res = await api.get('/api/enterprise/overview', { params })
  return res.data
}

export async function getEnterpriseUsageAnalytics(params: {
  start_timestamp: number
  end_timestamp: number
}): Promise<import('./types').EnterpriseApiResponse<import('./types').EnterpriseUsageAnalyticsData>> {
  const res = await api.get('/api/enterprise/usage-analytics', { params })
  return res.data
}

export async function getEnterpriseUsers(params: {
  limit?: number
} = {}): Promise<import('./types').EnterpriseApiResponse<import('./types').EnterpriseUsersData>> {
  const res = await api.get('/api/enterprise/users', { params })
  return res.data
}

export async function getEnterpriseBilling(params: {
  start_timestamp: number
  end_timestamp: number
}): Promise<import('./types').EnterpriseApiResponse<import('./types').EnterpriseBillingData>> {
  const res = await api.get('/api/enterprise/billing', { params })
  return res.data
}
