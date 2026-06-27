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
import { useQuery } from '@tanstack/react-query'
import { VChart } from '@visactor/react-vchart'
import {
  BarChart3,
  CalendarDays,
  Coins,
  Hash,
  Loader2,
  TrendingUp,
  Users,
  type LucideIcon,
} from 'lucide-react'
import { useEffect, useMemo, useState, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useTheme } from '@/context/theme-provider'
import { getUserQuotaDataByUsers } from '@/features/dashboard/api'
import {
  TIME_GRANULARITY_OPTIONS,
  TIME_RANGE_PRESETS,
} from '@/features/dashboard/constants'
import {
  getDefaultDays,
  saveGranularity,
  processUserChartData,
} from '@/features/dashboard/lib'
import type {
  ProcessedUserChartData,
  QuotaDataItem,
  UserChartsFilters,
} from '@/features/dashboard/types'
import { formatCompactNumber, formatNumber, formatQuota } from '@/lib/format'
import { getRollingDateRange, type TimeGranularity } from '@/lib/time'
import { cn } from '@/lib/utils'
import { VCHART_OPTION } from '@/lib/vchart'

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

const USER_CHARTS: {
  value: string
  labelKey: string
  specKey: keyof ProcessedUserChartData
}[] = [
  {
    value: 'rank',
    labelKey: '用户消耗排行',
    specKey: 'spec_user_rank',
  },
  {
    value: 'trend',
    labelKey: '用户消耗趋势',
    specKey: 'spec_user_trend',
  },
]

const TOP_USER_LIMIT_OPTIONS = [5, 10, 20, 50]

type UserSummaryRow = {
  username: string
  quota: number
  requests: number
  tokens: number
}

function buildUserSummary(data: QuotaDataItem[]) {
  const rows = new Map<string, UserSummaryRow>()
  const activeDays = new Set<string>()

  for (const item of data) {
    const username = item.username || `user-${item.user_id ?? 'unknown'}`
    const current =
      rows.get(username) ??
      ({
        username,
        quota: 0,
        requests: 0,
        tokens: 0,
      } satisfies UserSummaryRow)
    current.quota += Number(item.quota) || 0
    current.requests += Number(item.count) || 0
    current.tokens += Number(item.token_used) || 0
    rows.set(username, current)

    if (item.created_at) {
      activeDays.add(new Date(item.created_at * 1000).toDateString())
    }
  }

  const topUsers = [...rows.values()].sort((a, b) => b.quota - a.quota)
  return {
    topUsers,
    totalQuota: topUsers.reduce((sum, item) => sum + item.quota, 0),
    totalRequests: topUsers.reduce((sum, item) => sum + item.requests, 0),
    totalTokens: topUsers.reduce((sum, item) => sum + item.tokens, 0),
    userCount: rows.size,
    activeDays: activeDays.size,
  }
}

function CompactStatCard(props: {
  title: string
  value: string
  helper: string
  icon: LucideIcon
  tone: 'blue' | 'emerald' | 'violet' | 'amber'
  loading?: boolean
}) {
  const Icon = props.icon
  const toneClassName = {
    blue: 'bg-blue-50 text-blue-600 ring-blue-100',
    emerald: 'bg-emerald-50 text-emerald-600 ring-emerald-100',
    violet: 'bg-violet-50 text-violet-600 ring-violet-100',
    amber: 'bg-amber-50 text-amber-600 ring-amber-100',
  }[props.tone]

  return (
    <div className='min-h-[84px] rounded-md border border-slate-200 bg-white p-3 shadow-[0_1px_2px_rgb(15_23_42/0.035)]'>
      <div className='flex items-start gap-2.5'>
        <span
          className={cn(
            'flex size-8 shrink-0 items-center justify-center rounded-md ring-1',
            toneClassName
          )}
        >
          <Icon className='size-4' aria-hidden='true' />
        </span>
        <div className='min-w-0 flex-1'>
          <p className='truncate text-[11px] font-medium text-slate-500'>
            {props.title}
          </p>
          {props.loading ? (
            <Skeleton className='mt-2 h-6 w-20' />
          ) : (
            <p className='mt-1 truncate text-[20px] leading-6 font-semibold text-slate-950 tabular-nums'>
              {props.value}
            </p>
          )}
          <p className='mt-1 truncate text-[11px] text-slate-500'>
            {props.helper}
          </p>
        </div>
      </div>
    </div>
  )
}

function ChartShell(props: {
  title: string
  description: string
  icon: LucideIcon
  loading?: boolean
  empty?: boolean
  emptyText: string
  children: React.ReactNode
}) {
  const Icon = props.icon

  return (
    <section className='overflow-hidden rounded-md border border-slate-200 bg-white shadow-[0_1px_2px_rgb(15_23_42/0.035)]'>
      <div className='flex min-h-11 items-center justify-between gap-2 border-b border-slate-100 bg-slate-50/65 px-3 py-2'>
        <div className='flex min-w-0 items-center gap-2'>
          <span className='flex size-7 shrink-0 items-center justify-center rounded-md bg-blue-50 text-blue-600 ring-1 ring-blue-100'>
            <Icon className='size-3.5' aria-hidden='true' />
          </span>
          <div className='min-w-0'>
            <h2 className='truncate text-[13px] font-semibold text-slate-900'>
              {props.title}
            </h2>
            <p className='truncate text-[11px] text-slate-500'>
              {props.description}
            </p>
          </div>
        </div>
        {props.loading && (
          <Loader2 className='size-4 shrink-0 animate-spin text-slate-400' />
        )}
      </div>
      <div className='h-[260px] p-2'>
        {props.loading ? (
          <Skeleton className='h-full w-full' />
        ) : props.empty ? (
          <div className='flex h-full flex-col items-center justify-center rounded-md border border-dashed border-slate-200 bg-slate-50/45 px-4 text-center'>
            <Icon className='size-7 text-slate-300' aria-hidden='true' />
            <p className='mt-2 text-[13px] font-semibold text-slate-700'>
              {props.emptyText}
            </p>
            <p className='mt-1 max-w-[18rem] text-[11px] leading-5 text-slate-500'>
              调整时间范围、颗粒度或等待新的调用日志同步后会自动更新。
            </p>
          </div>
        ) : (
          props.children
        )}
      </div>
    </section>
  )
}

interface UserChartsProps {
  filters: UserChartsFilters
  onFiltersChange: (filters: UserChartsFilters) => void
}

export function UserCharts(props: UserChartsProps) {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const [themeReady, setThemeReady] = useState(false)
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)

  // The selection is owned by the dashboard parent so it persists across
  // sub-section switches; the rolling window is derived from the chosen range.
  const timeGranularity = props.filters.timeGranularity
  const selectedRange = props.filters.selectedRange
  const topUserLimit = props.filters.topUserLimit
  const onFiltersChange = props.onFiltersChange

  const timeRange = useMemo(() => {
    const { start, end } = getRollingDateRange(selectedRange)
    return {
      start_timestamp: Math.floor(start.getTime() / 1000),
      end_timestamp: Math.floor(end.getTime() / 1000),
    }
  }, [selectedRange])

  const handleRangeChange = useCallback(
    (days: number) => {
      onFiltersChange({ ...props.filters, selectedRange: days })
    },
    [onFiltersChange, props.filters]
  )

  const handleGranularityChange = useCallback(
    (g: TimeGranularity) => {
      saveGranularity(g)
      onFiltersChange({
        ...props.filters,
        timeGranularity: g,
        selectedRange: getDefaultDays(g),
      })
    },
    [onFiltersChange, props.filters]
  )

  const handleTopUserLimitChange = useCallback(
    (limit: number) => {
      onFiltersChange({ ...props.filters, topUserLimit: limit })
    },
    [onFiltersChange, props.filters]
  )

  useEffect(() => {
    const updateTheme = async () => {
      setThemeReady(false)
      if (!themeManagerPromise) {
        themeManagerPromise = import('@visactor/vchart').then(
          (m) => m.ThemeManager
        )
      }
      const ThemeManager = await themeManagerPromise
      themeManagerRef.current = ThemeManager
      ThemeManager.setCurrentTheme(resolvedTheme === 'dark' ? 'dark' : 'light')
      setThemeReady(true)
    }
    updateTheme()
  }, [resolvedTheme])

  const { data: userData, isLoading } = useQuery({
    queryKey: ['dashboard', 'user-quota', timeRange],
    queryFn: () => getUserQuotaDataByUsers(timeRange),
    select: (res) => (res.success ? res.data : []),
    staleTime: 60_000,
  })

  const summary = useMemo(
    () => buildUserSummary(isLoading ? [] : (userData ?? [])),
    [userData, isLoading]
  )
  const isEmpty = !isLoading && summary.topUsers.length === 0

  const chartData = useMemo(
    () =>
      processUserChartData(
        isLoading ? [] : (userData ?? []),
        timeGranularity,
        t,
        topUserLimit
      ),
    [userData, isLoading, timeGranularity, t, topUserLimit]
  )

  return (
    <div className='enterprise-dashboard space-y-3 text-slate-950'>
      <div className='flex flex-col gap-2 rounded-md border border-slate-200 bg-white p-3 shadow-[0_1px_2px_rgb(15_23_42/0.035)] xl:flex-row xl:items-center xl:justify-between'>
        <div className='min-w-0'>
          <p className='text-[11px] font-semibold text-blue-600'>
            用户用量归因
          </p>
          <p className='mt-0.5 text-[12px] text-slate-500'>
            按真实调用日志聚合用户消耗、调用次数和 Token 贡献。
          </p>
        </div>
        <div className='flex min-w-0 flex-wrap items-center gap-2 xl:justify-end'>
          <Tabs
            value={String(selectedRange)}
            onValueChange={(value) => handleRangeChange(Number(value))}
            className='shrink-0'
          >
            <TabsList>
              {TIME_RANGE_PRESETS.map((preset) => (
                <TabsTrigger
                  key={preset.days}
                  value={String(preset.days)}
                  className='px-2.5 text-xs'
                >
                  {t(preset.label)}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>

          <Tabs
            value={timeGranularity}
            onValueChange={(value) =>
              handleGranularityChange(value as TimeGranularity)
            }
            className='shrink-0'
          >
            <TabsList>
              {TIME_GRANULARITY_OPTIONS.map((opt) => (
                <TabsTrigger
                  key={opt.value}
                  value={opt.value}
                  className='px-2.5 text-xs'
                >
                  {t(opt.label)}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>

          <Tabs
            value={String(topUserLimit)}
            onValueChange={(value) => handleTopUserLimitChange(Number(value))}
            className='shrink-0'
          >
            <TabsList>
              <span className='px-2 text-xs font-medium whitespace-nowrap text-slate-500'>
                Top 用户
              </span>
              {TOP_USER_LIMIT_OPTIONS.map((limit) => (
                <TabsTrigger
                  key={limit}
                  value={String(limit)}
                  className='px-2.5 text-xs'
                >
                  {t('Top {{count}}', { count: limit })}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>

          {isLoading && (
            <Loader2 className='size-4 animate-spin text-slate-400' />
          )}
        </div>
      </div>

      <div className='grid min-w-0 gap-2 sm:grid-cols-2 xl:grid-cols-4'>
        <CompactStatCard
          title='参与用户'
          value={formatNumber(summary.userCount)}
          helper={`近 ${selectedRange} 天去重用户`}
          icon={Users}
          tone='blue'
          loading={isLoading}
        />
        <CompactStatCard
          title='消耗额度'
          value={formatQuota(summary.totalQuota)}
          helper='按售价口径汇总'
          icon={Coins}
          tone='amber'
          loading={isLoading}
        />
        <CompactStatCard
          title='请求次数'
          value={formatCompactNumber(summary.totalRequests)}
          helper='成功调用日志聚合'
          icon={TrendingUp}
          tone='emerald'
          loading={isLoading}
        />
        <CompactStatCard
          title='Token 用量'
          value={formatCompactNumber(summary.totalTokens)}
          helper={`${summary.activeDays} 个活跃日期`}
          icon={Hash}
          tone='violet'
          loading={isLoading}
        />
      </div>

      <div className='grid gap-3 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]'>
        {USER_CHARTS.map((chart) => {
          const spec = chartData[chart.specKey]
          const chartIcon = chart.value === 'rank' ? BarChart3 : CalendarDays

          return (
            <ChartShell
              key={chart.value}
              title={t(chart.labelKey)}
              description={
                chart.value === 'rank'
                  ? '按用户总消耗排序'
                  : '按时间颗粒度观察用户变化'
              }
              icon={chartIcon}
              loading={isLoading}
              empty={isEmpty}
              emptyText={
                chart.value === 'rank'
                  ? '当前范围暂无用户排行'
                  : '当前范围暂无用户趋势'
              }
            >
              {themeReady && spec && (
                <VChart
                  key={`user-${chart.value}-${topUserLimit}-${resolvedTheme}-${summary.totalQuota}`}
                  spec={{
                    ...spec,
                    theme: resolvedTheme === 'dark' ? 'dark' : 'light',
                    title: { visible: false },
                    background: 'transparent',
                  }}
                  option={VCHART_OPTION}
                />
              )}
            </ChartShell>
          )
        })}
      </div>

      <section className='overflow-hidden rounded-md border border-slate-200 bg-white shadow-[0_1px_2px_rgb(15_23_42/0.035)]'>
        <div className='flex min-h-11 items-center justify-between gap-2 border-b border-slate-100 bg-slate-50/65 px-3 py-2'>
          <div>
            <h2 className='text-[13px] font-semibold text-slate-900'>
              用户贡献明细
            </h2>
            <p className='text-[11px] text-slate-500'>
              按消耗额度排序，展示请求和 Token 贡献。
            </p>
          </div>
          <span className='rounded-md bg-blue-50 px-2 py-1 text-[11px] font-semibold text-blue-700'>
            Top {topUserLimit}
          </span>
        </div>
        <div className='divide-y divide-slate-100'>
          {isLoading ? (
            Array.from({ length: 4 }).map((_, index) => (
              <div key={index} className='grid grid-cols-4 gap-3 px-3 py-2'>
                <Skeleton className='h-5 w-32' />
                <Skeleton className='h-5 w-20' />
                <Skeleton className='h-5 w-20' />
                <Skeleton className='h-5 w-20' />
              </div>
            ))
          ) : isEmpty ? (
            <div className='px-3 py-8 text-center text-xs text-slate-500'>
              当前筛选范围暂无用户用量明细。
            </div>
          ) : (
            summary.topUsers.slice(0, topUserLimit).map((row, index) => {
              const share =
                summary.totalQuota > 0 ? row.quota / summary.totalQuota : 0
              return (
                <div
                  key={row.username}
                  className='grid grid-cols-[48px_minmax(0,1.3fr)_minmax(120px,0.8fr)_minmax(120px,0.8fr)_minmax(120px,0.8fr)] items-center gap-3 px-3 py-2 text-[12px]'
                >
                  <span className='flex size-6 items-center justify-center rounded-md bg-slate-100 text-xs font-semibold text-slate-600'>
                    {index + 1}
                  </span>
                  <div className='min-w-0'>
                    <p className='truncate font-semibold text-slate-900'>
                      {row.username}
                    </p>
                    <div className='mt-1 h-1.5 overflow-hidden rounded-full bg-slate-100'>
                      <div
                        className='h-full rounded-full bg-blue-500'
                        style={{ width: `${Math.max(4, share * 100)}%` }}
                      />
                    </div>
                  </div>
                  <span className='font-semibold text-slate-900 tabular-nums'>
                    {formatQuota(row.quota)}
                  </span>
                  <span className='text-slate-600 tabular-nums'>
                    {formatCompactNumber(row.requests)} 请求
                  </span>
                  <span className='text-slate-600 tabular-nums'>
                    {formatCompactNumber(row.tokens)} Tokens
                  </span>
                </div>
              )
            })
          )}
        </div>
      </section>
    </div>
  )
}
