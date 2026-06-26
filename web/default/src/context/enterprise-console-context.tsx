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
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

import type { TimeGranularity } from '@/lib/time'

export type EnterpriseRangePreset = 'today' | '7d' | '30d' | 'month' | 'custom'

export type EnterpriseDateRange = {
  start: number
  end: number
}

type EnterpriseConsolePreferences = {
  workspaceId: string
  rangePreset: EnterpriseRangePreset
  customRange?: EnterpriseDateRange
  granularity: TimeGranularity
}

type EnterpriseConsoleContextValue = EnterpriseConsolePreferences & {
  range: EnterpriseDateRange
  rangeLabel: string
  setWorkspaceId: (workspaceId: string) => void
  setRangePreset: (rangePreset: EnterpriseRangePreset) => void
  setCustomRange: (range: EnterpriseDateRange) => void
  setGranularity: (granularity: TimeGranularity) => void
}

const STORAGE_KEY = 'enterprise-console-preferences'

const DEFAULT_PREFERENCES: EnterpriseConsolePreferences = {
  workspaceId: 'platform',
  rangePreset: '30d',
  granularity: 'day',
}

export const ENTERPRISE_TIME_RANGE_OPTIONS: Array<{
  value: Exclude<EnterpriseRangePreset, 'custom'>
  label: string
  description: string
}> = [
  {
    value: 'today',
    label: '今天',
    description: '从今日 00:00 到当前时间',
  },
  {
    value: '7d',
    label: '最近 7 天',
    description: '默认经营观察窗口',
  },
  {
    value: '30d',
    label: '最近 30 天',
    description: '月度趋势与账单复核',
  },
  {
    value: 'month',
    label: '本月',
    description: '从本月 1 日到当前时间',
  },
]

function isEnterpriseRangePreset(
  value: unknown
): value is EnterpriseRangePreset {
  return (
    value === 'today' ||
    value === '7d' ||
    value === '30d' ||
    value === 'month' ||
    value === 'custom'
  )
}

function isTimeGranularity(value: unknown): value is TimeGranularity {
  return value === 'hour' || value === 'day' || value === 'week'
}

function isEnterpriseDateRange(value: unknown): value is EnterpriseDateRange {
  if (value == null || typeof value !== 'object') return false
  const maybeRange = value as Partial<EnterpriseDateRange>
  return (
    typeof maybeRange.start === 'number' &&
    typeof maybeRange.end === 'number' &&
    Number.isFinite(maybeRange.start) &&
    Number.isFinite(maybeRange.end) &&
    Number(maybeRange.end) > Number(maybeRange.start)
  )
}

function readPreferences(): EnterpriseConsolePreferences {
  if (typeof window === 'undefined') return DEFAULT_PREFERENCES

  try {
    const parsed = JSON.parse(
      window.localStorage.getItem(STORAGE_KEY) ?? '{}'
    ) as Partial<EnterpriseConsolePreferences>

    return {
      workspaceId:
        typeof parsed.workspaceId === 'string' && parsed.workspaceId.length > 0
          ? parsed.workspaceId
          : DEFAULT_PREFERENCES.workspaceId,
      rangePreset: isEnterpriseRangePreset(parsed.rangePreset)
        ? parsed.rangePreset
        : DEFAULT_PREFERENCES.rangePreset,
      customRange: isEnterpriseDateRange(parsed.customRange)
        ? parsed.customRange
        : undefined,
      granularity: isTimeGranularity(parsed.granularity)
        ? parsed.granularity
        : DEFAULT_PREFERENCES.granularity,
    }
  } catch {
    return DEFAULT_PREFERENCES
  }
}

function startOfLocalDay(date: Date): number {
  return Math.floor(
    new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime() /
      1000
  )
}

function startOfLocalMonth(date: Date): number {
  return Math.floor(new Date(date.getFullYear(), date.getMonth(), 1).getTime() / 1000)
}

export function getEnterpriseRangeForPreset(
  preset: Exclude<EnterpriseRangePreset, 'custom'>,
  now = new Date()
): EnterpriseDateRange {
  const end = Math.floor(now.getTime() / 1000)

  if (preset === 'today') {
    return { start: startOfLocalDay(now), end }
  }

  if (preset === '30d') {
    return { start: end - 30 * 24 * 60 * 60, end }
  }

  if (preset === 'month') {
    return { start: startOfLocalMonth(now), end }
  }

  return { start: end - 7 * 24 * 60 * 60, end }
}

function getShortDateLabel(now: Date): string {
  const date = new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
  }).format(now)
  const weekday = new Intl.DateTimeFormat('zh-CN', {
    weekday: 'short',
  }).format(now)

  return `${date}${weekday}`
}

function getRangeLabel(preset: EnterpriseRangePreset, now: Date): string {
  if (preset === 'custom') return '自定义时间段'

  const today = getShortDateLabel(now)
  const option = ENTERPRISE_TIME_RANGE_OPTIONS.find((item) => item.value === preset)
  const label = option?.label ?? '最近 30 天'

  return `${label} · ${today}`
}

const EnterpriseConsoleContext =
  createContext<EnterpriseConsoleContextValue | null>(null)

export function EnterpriseConsoleProvider({
  children,
}: {
  children: ReactNode
}) {
  const [preferences, setPreferences] = useState(readPreferences)
  const now = useMemo(() => new Date(), [preferences.rangePreset, preferences.customRange])

  useEffect(() => {
    if (typeof window === 'undefined') return
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences))
  }, [preferences])

  const setWorkspaceId = useCallback((workspaceId: string) => {
    setPreferences((current) => ({ ...current, workspaceId }))
  }, [])

  const setRangePreset = useCallback((rangePreset: EnterpriseRangePreset) => {
    setPreferences((current) => ({
      ...current,
      rangePreset,
    }))
  }, [])

  const setCustomRange = useCallback((range: EnterpriseDateRange) => {
    setPreferences((current) => ({
      ...current,
      rangePreset: 'custom',
      customRange: range,
    }))
  }, [])

  const setGranularity = useCallback((granularity: TimeGranularity) => {
    setPreferences((current) => ({ ...current, granularity }))
  }, [])

  const range = useMemo(
    () =>
      preferences.rangePreset === 'custom' &&
      isEnterpriseDateRange(preferences.customRange)
        ? preferences.customRange
        : getEnterpriseRangeForPreset(
            preferences.rangePreset === 'custom' ? '30d' : preferences.rangePreset,
            now
          ),
    [now, preferences.customRange, preferences.rangePreset]
  )

  const value = useMemo<EnterpriseConsoleContextValue>(
    () => ({
      ...preferences,
      range,
      rangeLabel: getRangeLabel(preferences.rangePreset, now),
      setWorkspaceId,
      setRangePreset,
      setCustomRange,
      setGranularity,
    }),
    [
      now,
      preferences,
      range,
      setCustomRange,
      setGranularity,
      setRangePreset,
      setWorkspaceId,
    ]
  )

  return (
    <EnterpriseConsoleContext.Provider value={value}>
      {children}
    </EnterpriseConsoleContext.Provider>
  )
}

export function useEnterpriseConsole() {
  const context = useContext(EnterpriseConsoleContext)

  if (!context) {
    throw new Error(
      'useEnterpriseConsole must be used within EnterpriseConsoleProvider'
    )
  }

  return context
}
