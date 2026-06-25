import { api } from '@/lib/api'
import type {
  EnterpriseChannelCenterData,
  EnterpriseChannelDetail,
  EnterpriseChannelResponse,
} from './enterprise-types'

type RangeQuery = {
  start_timestamp: number
  end_timestamp: number
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
