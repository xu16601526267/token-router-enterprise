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
  EnterpriseChannelCenterData,
  EnterpriseChannelDetail,
  EnterpriseChannelResponse,
} from './enterprise-types'

type RangeQuery = {
  start_timestamp: number
  end_timestamp: number
  keyword?: string
  status?: number
  supplier_id?: number
  type?: number
  group?: string
  page?: number
  page_size?: number
  sort_by?: string
  sort_order?: string
}

async function downloadCsv(
  path: string,
  params: RangeQuery,
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

export async function getEnterpriseChannelCenter(
  params: RangeQuery
): Promise<EnterpriseChannelResponse<EnterpriseChannelCenterData>> {
  const response = await api.get('/api/enterprise/channels', { params })
  return response.data
}

export async function getEnterpriseChannelDetail(
  id: number,
  params: RangeQuery
): Promise<EnterpriseChannelResponse<EnterpriseChannelDetail>> {
  const response = await api.get(`/api/enterprise/channels/${id}`, { params })
  return response.data
}

export async function exportEnterpriseChannelCenter(
  params: RangeQuery
): Promise<void> {
  await downloadCsv(
    '/api/enterprise/channels/export',
    params,
    'enterprise-channels.csv'
  )
}
