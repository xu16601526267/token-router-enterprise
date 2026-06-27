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
import type { TFunction } from 'i18next'

import type { StatusBadgeProps } from '@/components/status-badge'

// ============================================================================
// Redemption Status Configuration
// ============================================================================

export const REDEMPTION_STATUS = {
  ENABLED: 1,
  DISABLED: 2,
  USED: 3,
} as const

export const REDEMPTION_STATUS_VALUES = Object.values(REDEMPTION_STATUS).map(
  (value) => String(value)
) as `${number}`[]

// labelKey values are user-facing labels; components can still pass them through
// i18n for deployments that provide overrides.
export const REDEMPTION_STATUSES: Record<
  number,
  Pick<StatusBadgeProps, 'variant'> & {
    labelKey: string
    value: number
  }
> = {
  [REDEMPTION_STATUS.ENABLED]: {
    labelKey: '未使用',
    variant: 'success',
    value: REDEMPTION_STATUS.ENABLED,
  },
  [REDEMPTION_STATUS.DISABLED]: {
    labelKey: '已禁用',
    variant: 'neutral',
    value: REDEMPTION_STATUS.DISABLED,
  },
  [REDEMPTION_STATUS.USED]: {
    labelKey: '已使用',
    variant: 'neutral',
    value: REDEMPTION_STATUS.USED,
  },
} as const

// Virtual status filter value for expired redemption codes
// Note: "Expired" is not a real DB status, it's computed from expired_time
export const REDEMPTION_FILTER_EXPIRED = 'expired'

export function getRedemptionStatusOptions(t: TFunction) {
  return [
    ...Object.values(REDEMPTION_STATUSES).map((config) => ({
      label: t(config.labelKey),
      value: String(config.value),
    })),
    {
      label: t('已过期'),
      value: REDEMPTION_FILTER_EXPIRED,
    },
  ]
}

// ============================================================================
// Validation Constants
// ============================================================================

export const REDEMPTION_VALIDATION = {
  NAME_MIN_LENGTH: 1,
  NAME_MAX_LENGTH: 20,
  COUNT_MIN: 1,
  COUNT_MAX: 100,
} as const

// ============================================================================
// Error Messages
// ============================================================================

// i18n keys; use t(ERROR_MESSAGES.xxx) when displaying. For form schema with interpolation use getRedemptionFormErrorMessages(t).
export const ERROR_MESSAGES = {
  UNEXPECTED: '发生未知错误',
  LOAD_FAILED: '兑换码加载失败',
  SEARCH_FAILED: '兑换码搜索失败',
  CREATE_FAILED: '兑换码创建失败',
  UPDATE_FAILED: '兑换码更新失败',
  DELETE_FAILED: '兑换码删除失败',
  DELETE_INVALID_FAILED: '失效兑换码清理失败',
  STATUS_UPDATE_FAILED: '兑换码状态更新失败',
  NAME_LENGTH_INVALID: '名称长度需在 {{min}} 到 {{max}} 个字符之间',
  COUNT_INVALID: '生成数量需在 {{min}} 到 {{max}} 之间',
  EXPIRED_TIME_INVALID: '过期时间不能早于当前时间',
} as const

/** For form schema only: returns translated messages with interpolation. */
export function getRedemptionFormErrorMessages(t: TFunction) {
  return {
    NAME_LENGTH_INVALID: t(ERROR_MESSAGES.NAME_LENGTH_INVALID, {
      min: REDEMPTION_VALIDATION.NAME_MIN_LENGTH,
      max: REDEMPTION_VALIDATION.NAME_MAX_LENGTH,
    }),
    COUNT_INVALID: t(ERROR_MESSAGES.COUNT_INVALID, {
      min: REDEMPTION_VALIDATION.COUNT_MIN,
      max: REDEMPTION_VALIDATION.COUNT_MAX,
    }),
    EXPIRED_TIME_INVALID: t(ERROR_MESSAGES.EXPIRED_TIME_INVALID),
  } as const
}

// ============================================================================
// Success Messages (i18n keys; use t(SUCCESS_MESSAGES.xxx) when displaying)
// ============================================================================

export const SUCCESS_MESSAGES = {
  REDEMPTION_CREATED: '兑换码创建成功',
  REDEMPTION_UPDATED: '兑换码更新成功',
  REDEMPTION_DELETED: '兑换码删除成功',
  REDEMPTION_ENABLED: '兑换码已启用',
  REDEMPTION_DISABLED: '兑换码已禁用',
  COPY_SUCCESS: '已复制到剪贴板',
} as const
