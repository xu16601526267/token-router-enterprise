import { api } from '@/lib/api'
import type { ControlTowerResponse } from './control-tower-types'

export async function getControlTower(params: {
  start_timestamp: number
  end_timestamp: number
}): Promise<ControlTowerResponse> {
  const response = await api.get('/api/enterprise/control-tower', { params })
  return response.data
}
