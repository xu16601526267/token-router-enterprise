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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { getUserQuotaDates } from '@/features/dashboard/api'
import { useModelStatCardsConfig } from '@/features/dashboard/hooks/use-dashboard-config'
import {
  buildQueryParams,
  calculateDashboardStats,
  getDefaultDays,
} from '@/features/dashboard/lib'
import type {
  QuotaDataItem,
  DashboardFilters,
} from '@/features/dashboard/types'
import { formatCompactNumber, formatNumber, formatQuota } from '@/lib/format'
import { computeTimeRange } from '@/lib/time'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

interface LogStatCardsProps {
  filters?: DashboardFilters
  onDataUpdate?: (data: QuotaDataItem[], loading: boolean) => void
}

const MAX_INLINE_STAT_CHARS = 9

function formatStatNumber(value: number, locale: Intl.LocalesArgument) {
  const fullValue = formatNumber(value, locale)
  const displayValue =
    fullValue.length > MAX_INLINE_STAT_CHARS
      ? formatCompactNumber(value, locale)
      : fullValue

  return {
    displayValue,
    fullValue,
  }
}

export function LogStatCards(props: LogStatCardsProps) {
  const { i18n } = useTranslation()
  const statCardsConfig = useModelStatCardsConfig()
  const user = useAuthStore((state) => state.auth.user)
  const isAdmin = !!(user?.role && user.role >= 10)
  const [stats, setStats] = useState<{
    totalQuota: number
    totalCount: number
    totalTokens: number
  } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  const [timeRangeMinutes, setTimeRangeMinutes] = useState(0)

  const { filters, onDataUpdate } = props

  useEffect(() => {
    const abortController = new AbortController()
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true)

    setError(false)
    onDataUpdate?.([], true)

    const timeRange = computeTimeRange(
      getDefaultDays(filters?.time_granularity),
      filters?.start_timestamp,
      filters?.end_timestamp
    )
    const timeDiff = (timeRange.end_timestamp - timeRange.start_timestamp) / 60
    setTimeRangeMinutes(timeDiff)

    void (async () => {
      try {
        const res = await getUserQuotaDates(
          buildQueryParams(timeRange, filters),
          isAdmin
        )
        if (abortController.signal.aborted) return
        const data = res?.data || []
        setStats(calculateDashboardStats(data))
        onDataUpdate?.(data, false)
      } catch {
        if (abortController.signal.aborted) return
        setStats(null)
        setError(true)
        onDataUpdate?.([], false)
      } finally {
        if (!abortController.signal.aborted) {
          setLoading(false)
        }
      }
    })()

    return () => {
      abortController.abort()
    }
  }, [filters, isAdmin, onDataUpdate])

  const adaptedStats = {
    rpm: stats?.totalCount ?? 0,
    quota: stats?.totalQuota ?? 0,
    tpm: stats?.totalTokens ?? 0,
  }

  const items = statCardsConfig.map((config) => {
    const rawValue = config.getValue(adaptedStats, timeRangeMinutes)
    const locale = i18n.resolvedLanguage || i18n.language
    const formatted =
      config.key === 'quota'
        ? {
            displayValue: formatQuota(rawValue),
            fullValue: formatQuota(rawValue),
          }
        : formatStatNumber(rawValue, locale)

    return {
      title: config.title,
      value: formatted.displayValue,
      fullValue: formatted.fullValue,
      desc: config.description,
      icon: config.icon,
    }
  })

  return (
    <div className='grid min-w-0 grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-5'>
      {items.map((it, idx) => {
        const Icon = it.icon
        return (
          <div
            key={it.title}
            className={cn(
              'min-h-[84px] min-w-0 rounded-md border border-slate-200 bg-white p-3 shadow-[0_1px_2px_rgb(15_23_42/0.035)]',
              idx === items.length - 1 &&
                items.length % 2 !== 0 &&
                'col-span-2 sm:col-span-1'
            )}
          >
            <div className='flex min-w-0 items-center gap-2'>
              <span className='flex size-8 shrink-0 items-center justify-center rounded-md bg-blue-50 text-blue-600 ring-1 ring-blue-100'>
                <Icon className='size-4' strokeWidth={1.9} />
              </span>
              <div className='truncate text-[11px] leading-4 font-medium text-slate-500'>
                {it.title}
              </div>
            </div>

            {loading ? (
              <div className='mt-2 flex flex-col gap-1.5 pl-10'>
                <Skeleton className='h-7 w-20' />
                <Skeleton className='h-3.5 w-28' />
              </div>
            ) : error ? (
              <>
                <div className='mt-1.5 pl-10 text-[19px] leading-6 font-semibold tracking-tight text-slate-400 tabular-nums'>
                  --
                </div>
                <div className='mt-1 pl-10 text-[11px] text-slate-400'>
                  {it.desc}
                </div>
              </>
            ) : (
              <>
                <div
                  className='mt-1.5 max-w-full truncate pl-10 text-[19px] leading-6 font-semibold tracking-tight text-slate-950 tabular-nums'
                  title={it.fullValue}
                >
                  {it.value}
                </div>
                <div className='mt-1 pl-10 text-[11px] text-slate-500'>
                  {it.desc}
                </div>
              </>
            )}
          </div>
        )
      })}
    </div>
  )
}
