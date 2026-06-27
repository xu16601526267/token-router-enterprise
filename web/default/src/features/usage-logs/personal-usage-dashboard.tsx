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
import { getRouteApi } from '@tanstack/react-router'
import {
  Activity,
  Download,
  KeyRound,
  ListFilter,
  Search,
  ShieldCheck,
  Sparkles,
  TableProperties,
  WalletCards,
} from 'lucide-react'
import { useMemo } from 'react'
import {
  Area,
  Bar,
  CartesianGrid,
  ComposedChart,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import { EnterprisePanel, EnterpriseStatCard } from '@/components/enterprise'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { getUserQuotaDates } from '@/features/dashboard/api'
import type { QuotaDataItem } from '@/features/dashboard/types'
import { getApiKeys } from '@/features/keys/api'
import { getUserModels } from '@/lib/api'
import dayjs from '@/lib/dayjs'
import {
  formatCompactNumber,
  formatLogQuota,
  formatTimestampToDate,
  formatTokens,
} from '@/lib/format'
import { cn } from '@/lib/utils'

import { getUserLogs, getUserLogStats } from './api'
import { DEFAULT_LOGS_DATA, LOG_TYPE_ENUM, LOG_TYPES } from './constants'
import type { UsageLog } from './data/schema'
import type { GetLogsParams } from './types'

const route = getRouteApi('/_authenticated/usage-logs/$section')

type StatusFilter = 'all' | 'success' | 'error'
type DatePreset = '7' | '14' | '30'

const PAGE_SIZE = 5
const statusOptions: Array<{ label: string; value: StatusFilter }> = [
  { label: '状态 全部', value: 'all' },
  { label: '成功请求', value: 'success' },
  { label: '错误请求', value: 'error' },
]
const dateOptions: Array<{ label: string; value: DatePreset }> = [
  { label: '日期 近7天', value: '7' },
  { label: '日期 近14天', value: '14' },
  { label: '日期 近30天', value: '30' },
]

function toDateRange(search: Record<string, unknown>) {
  const endMs =
    typeof search.endTime === 'number' ? search.endTime : Date.now() + 3600_000
  const startMs =
    typeof search.startTime === 'number'
      ? search.startTime
      : endMs - 7 * 24 * 3600_000

  return {
    startMs,
    endMs,
    startSec: Math.floor(startMs / 1000),
    endSec: Math.floor(endMs / 1000),
  }
}

function inferDatePreset(search: Record<string, unknown>): DatePreset {
  if (
    typeof search.startTime !== 'number' ||
    typeof search.endTime !== 'number'
  ) {
    return '7'
  }
  const days = Math.round((search.endTime - search.startTime) / 86_400_000)
  if (days <= 7) return '7'
  if (days <= 14) return '14'
  return '30'
}

function inferStatusFilter(type?: string[]): StatusFilter {
  if (type?.includes(String(LOG_TYPE_ENUM.ERROR))) return 'error'
  if (type?.includes(String(LOG_TYPE_ENUM.CONSUME))) return 'success'
  return 'all'
}

function statusToLogType(status: StatusFilter) {
  return status === 'error' ? LOG_TYPE_ENUM.ERROR : LOG_TYPE_ENUM.CONSUME
}

function selectClassName() {
  return 'h-8 rounded-md border border-slate-200 bg-white px-2.5 text-[12px] font-medium text-slate-700 shadow-[0_1px_1px_rgb(15_23_42/0.03)] outline-none transition-colors hover:border-blue-200 focus:border-blue-300 focus:ring-2 focus:ring-blue-100'
}

function aggregateUsage(rows: QuotaDataItem[]) {
  return rows.reduce(
    (sum, row) => ({
      requests: sum.requests + (row.count ?? 0),
      tokens: sum.tokens + (row.token_used ?? 0),
      quota: sum.quota + (row.quota ?? 0),
    }),
    { requests: 0, tokens: 0, quota: 0 }
  )
}

function buildTrend(rows: QuotaDataItem[]) {
  const map = new Map<
    string,
    { date: string; requests: number; tokens: number; quota: number }
  >()

  for (const row of rows) {
    const key = dayjs((row.created_at || 0) * 1000).format('MM-DD')
    const current = map.get(key) ?? {
      date: key,
      requests: 0,
      tokens: 0,
      quota: 0,
    }
    current.requests += row.count ?? 0
    current.tokens += row.token_used ?? 0
    current.quota += row.quota ?? 0
    map.set(key, current)
  }

  return [...map.values()].sort((a, b) => a.date.localeCompare(b.date))
}

function buildModelRanking(rows: QuotaDataItem[]) {
  const map = new Map<
    string,
    { model: string; quota: number; requests: number }
  >()
  for (const row of rows) {
    const model = row.model_name || '未标记模型'
    const current = map.get(model) ?? { model, quota: 0, requests: 0 }
    current.quota += row.quota ?? 0
    current.requests += row.count ?? 0
    map.set(model, current)
  }
  const total = [...map.values()].reduce((sum, item) => sum + item.quota, 0)
  return [...map.values()]
    .sort((a, b) => b.quota - a.quota)
    .slice(0, 3)
    .map((item) => ({
      ...item,
      share: total > 0 ? Math.round((item.quota / total) * 100) : 0,
    }))
}

function logTypeLabel(type: number) {
  return LOG_TYPES.find((item) => item.value === type)?.label ?? 'Unknown'
}

function buildSuggestions(args: {
  avgTokens: number
  errorRate: number
  ranking: ReturnType<typeof buildModelRanking>
}) {
  const suggestions: Array<{ title: string; value: string }> = []

  if (args.ranking[0] && args.ranking[0].share >= 60) {
    suggestions.push({
      title: `${args.ranking[0].model} 占比偏高`,
      value: `占 ${args.ranking[0].share}%`,
    })
  }
  if (args.avgTokens >= 3000) {
    suggestions.push({
      title: '长上下文请求较多',
      value: '建议压缩',
    })
  }
  if (args.errorRate >= 1) {
    suggestions.push({
      title: '错误请求需要排查',
      value: `${args.errorRate.toFixed(1)}%`,
    })
  }
  if (suggestions.length === 0) {
    suggestions.push({
      title: '暂无明显异常',
      value: '保持观察',
    })
  }

  return suggestions.slice(0, 3)
}

function csvEscape(value: unknown) {
  const text = String(value ?? '')
  if (!/[",\n]/.test(text)) {
    return text
  }
  return `"${text.replaceAll('"', '""')}"`
}

export function PersonalUsageDashboard() {
  const search = route.useSearch()
  const navigate = route.useNavigate()
  const statusFilter = inferStatusFilter(search.type)
  const datePreset = inferDatePreset(search)
  const range = toDateRange(search)
  const page = search.page ?? 1
  const requestKeyword = search.requestId ?? ''

  const queryBase = useMemo<GetLogsParams>(
    () => ({
      start_timestamp: range.startSec,
      end_timestamp: range.endSec,
      model_name: search.model || undefined,
      token_name: search.token || undefined,
      request_id: requestKeyword || undefined,
    }),
    [range.endSec, range.startSec, requestKeyword, search.model, search.token]
  )

  const logType = statusToLogType(statusFilter)

  const logsQuery = useQuery({
    queryKey: ['personal-usage-logs', queryBase, logType, page],
    queryFn: async () => {
      const result = await getUserLogs({
        ...queryBase,
        type: logType,
        p: page,
        page_size: PAGE_SIZE,
      })
      return result.success
        ? (result.data ?? DEFAULT_LOGS_DATA)
        : DEFAULT_LOGS_DATA
    },
  })

  const statsQuery = useQuery({
    queryKey: ['personal-usage-stats', queryBase],
    queryFn: async () => {
      const result = await getUserLogStats({
        ...queryBase,
        type: LOG_TYPE_ENUM.CONSUME,
      })
      return result.success ? result.data : undefined
    },
  })

  const errorsQuery = useQuery({
    queryKey: ['personal-usage-errors', queryBase],
    queryFn: async () => {
      const result = await getUserLogs({
        ...queryBase,
        type: LOG_TYPE_ENUM.ERROR,
        p: 1,
        page_size: 1,
      })
      return result.success ? (result.data?.total ?? 0) : 0
    },
  })

  const quotaQuery = useQuery({
    queryKey: ['personal-usage-quota-trend', range.startSec, range.endSec],
    queryFn: async () => {
      const result = await getUserQuotaDates(
        {
          start_timestamp: range.startSec,
          end_timestamp: range.endSec,
        },
        false
      )
      return result.success ? result.data : []
    },
  })

  const modelsQuery = useQuery({
    queryKey: ['personal-usage-models'],
    queryFn: getUserModels,
    staleTime: 300_000,
  })

  const keysQuery = useQuery({
    queryKey: ['personal-usage-keys'],
    queryFn: () => getApiKeys({ p: 1, size: 100 }),
    staleTime: 120_000,
  })

  const quotaRows = quotaQuery.data ?? []
  const usage = aggregateUsage(quotaRows)
  const trend = buildTrend(quotaRows)
  const ranking = buildModelRanking(quotaRows)
  const totalRequests = usage.requests
  const errorTotal = errorsQuery.data ?? 0
  const denominator = totalRequests + errorTotal
  const errorRate = denominator > 0 ? (errorTotal / denominator) * 100 : 0
  const avgTokens = totalRequests > 0 ? usage.tokens / totalRequests : 0
  const suggestions = buildSuggestions({ avgTokens, errorRate, ranking })
  const logs = (logsQuery.data?.items ?? []) as UsageLog[]
  const totalLogs = logsQuery.data?.total ?? 0
  const pageCount = Math.max(1, Math.ceil(totalLogs / PAGE_SIZE))
  const isLoading =
    logsQuery.isLoading || statsQuery.isLoading || quotaQuery.isLoading

  const patchSearch = (patch: Record<string, unknown>) => {
    void navigate({
      to: '/usage-logs/$section',
      params: { section: 'common' },
      search: {
        ...search,
        ...patch,
        page: 1,
      },
    })
  }

  const handleDateChange = (value: DatePreset) => {
    const endTime = Date.now() + 3600_000
    const startTime = endTime - Number(value) * 24 * 3600_000
    patchSearch({ startTime, endTime })
  }

  const handleExport = () => {
    const rows = [
      ['请求ID', '时间', '模型', 'Tokens', '费用', '延迟', '状态'],
      ...logs.map((log) => [
        log.request_id || `log_${log.id}`,
        formatTimestampToDate(log.created_at),
        log.model_name || '-',
        (log.prompt_tokens ?? 0) + (log.completion_tokens ?? 0),
        formatLogQuota(log.quota || 0),
        `${Math.round((log.use_time || 0) * 1000)}ms`,
        logTypeLabel(log.type),
      ]),
    ]
      .map((row) => row.map(csvEscape).join(','))
      .join('\n')
    const blob = new Blob([rows], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `usage-logs-${dayjs().format('YYYYMMDD-HHmmss')}.csv`
    link.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className='personal-usage-dashboard enterprise-dashboard space-y-2 px-4 pt-2 pb-2 text-slate-950 sm:px-8'>
      <div className='flex flex-wrap items-end justify-between gap-3'>
        <div>
          <h1 className='text-[22px] leading-7 font-semibold tracking-normal text-slate-950'>
            我的用量日志
          </h1>
          <p className='mt-0.5 text-[12px] text-slate-500'>
            C端个人视角只展示自己的请求日志、费用和错误
          </p>
        </div>
        <div className='flex flex-wrap items-center gap-2'>
          <Badge
            variant='outline'
            className='h-7 rounded-md border-slate-200 bg-white px-2.5 text-[11px] font-medium text-slate-600'
          >
            {dayjs(range.startMs).format('YYYY-MM-DD')} ~{' '}
            {dayjs(range.endMs).format('YYYY-MM-DD')}
          </Badge>
          <Button
            variant='outline'
            size='sm'
            className='h-8 rounded-md bg-white px-2.5 text-[12px]'
            onClick={handleExport}
            disabled={logs.length === 0}
          >
            <Download className='size-3.5' />
            导出
          </Button>
        </div>
      </div>

      <div className='grid grid-cols-1 gap-2 md:grid-cols-2 xl:grid-cols-4'>
        <EnterpriseStatCard
          title='总请求'
          value={formatCompactNumber(totalRequests)}
          helper='当前周期'
          trend='近区间'
          icon={Activity}
          tone='blue'
          loading={isLoading}
          className='min-h-[86px]'
        />
        <EnterpriseStatCard
          title='Tokens'
          value={formatTokens(usage.tokens)}
          helper='输入 + 输出'
          trend={`${Math.round(avgTokens).toLocaleString()} / 请求`}
          icon={TableProperties}
          tone='violet'
          loading={isLoading}
          className='min-h-[86px]'
        />
        <EnterpriseStatCard
          title='总费用'
          value={formatLogQuota(statsQuery.data?.quota ?? usage.quota)}
          helper='按实际计费记录'
          trend='本周期'
          icon={WalletCards}
          tone='amber'
          loading={isLoading}
          className='min-h-[86px]'
        />
        <EnterpriseStatCard
          title='错误率'
          value={`${errorRate.toFixed(1)}%`}
          helper={errorTotal > 0 ? `${errorTotal} 条错误` : '健康'}
          trend={errorTotal > 0 ? '需关注' : '正常'}
          trendTone={errorTotal > 0 ? 'negative' : 'positive'}
          icon={ShieldCheck}
          tone={errorTotal > 0 ? 'rose' : 'emerald'}
          loading={isLoading}
          className='min-h-[86px]'
        />
      </div>

      <EnterprisePanel className='border-slate-200/80 shadow-[0_1px_2px_rgb(15_23_42/0.03)]'>
        <div className='flex flex-wrap items-center gap-2'>
          <span className='flex h-8 items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 text-[12px] font-medium text-slate-600'>
            <ListFilter className='size-3.5 text-slate-400' />
            筛选
          </span>
          <select
            aria-label='模型筛选'
            className={selectClassName()}
            value={search.model ?? ''}
            onChange={(event) =>
              patchSearch({ model: event.target.value || undefined })
            }
          >
            <option value=''>模型 全部</option>
            {(modelsQuery.data?.data ?? []).slice(0, 80).map((model) => (
              <option key={model} value={model}>
                {model}
              </option>
            ))}
          </select>
          <select
            aria-label='Key 筛选'
            className={selectClassName()}
            value={search.token ?? ''}
            onChange={(event) =>
              patchSearch({ token: event.target.value || undefined })
            }
          >
            <option value=''>Key 全部</option>
            {(keysQuery.data?.data?.items ?? []).map((key) => (
              <option key={key.id} value={key.name}>
                {key.name}
              </option>
            ))}
          </select>
          <select
            aria-label='状态筛选'
            className={selectClassName()}
            value={statusFilter}
            onChange={(event) =>
              patchSearch({
                type:
                  event.target.value === 'all'
                    ? undefined
                    : [
                        String(
                          statusToLogType(event.target.value as StatusFilter)
                        ),
                      ],
              })
            }
          >
            {statusOptions.map((item) => (
              <option key={item.value} value={item.value}>
                {item.label}
              </option>
            ))}
          </select>
          <select
            aria-label='日期筛选'
            className={selectClassName()}
            value={datePreset}
            onChange={(event) =>
              handleDateChange(event.target.value as DatePreset)
            }
          >
            {dateOptions.map((item) => (
              <option key={item.value} value={item.value}>
                {item.label}
              </option>
            ))}
          </select>
          <div className='relative min-w-[220px] flex-1 sm:max-w-[300px]'>
            <Search className='pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-slate-400' />
            <Input
              value={requestKeyword}
              onChange={(event) =>
                patchSearch({ requestId: event.target.value || undefined })
              }
              placeholder='搜索请求ID'
              className='h-8 rounded-md border-slate-200 bg-white pl-8 text-[12px] shadow-[0_1px_1px_rgb(15_23_42/0.03)]'
            />
          </div>
        </div>
      </EnterprisePanel>

      <div className='grid grid-cols-1 gap-2 xl:grid-cols-[minmax(0,1fr)_360px]'>
        <div className='space-y-2'>
          <EnterprisePanel
            title='用量趋势'
            description='请求量、Token 与费用变化'
            bodyClassName='h-[236px] px-2 pb-2 pt-3'
          >
            {trend.length === 0 ? (
              <div className='flex h-full items-center justify-center text-[12px] text-slate-400'>
                当前筛选条件下暂无趋势数据
              </div>
            ) : (
              <ResponsiveContainer width='100%' height='100%'>
                <ComposedChart
                  data={trend}
                  margin={{ top: 4, right: 8, bottom: 0, left: 2 }}
                >
                  <defs>
                    <linearGradient
                      id='personalUsageBlue'
                      x1='0'
                      y1='0'
                      x2='0'
                      y2='1'
                    >
                      <stop
                        offset='5%'
                        stopColor='#3b82f6'
                        stopOpacity={0.24}
                      />
                      <stop
                        offset='95%'
                        stopColor='#3b82f6'
                        stopOpacity={0.02}
                      />
                    </linearGradient>
                  </defs>
                  <CartesianGrid
                    stroke='#e5e7eb'
                    strokeDasharray='4 8'
                    vertical={false}
                  />
                  <XAxis
                    dataKey='date'
                    axisLine={false}
                    tickLine={false}
                    tick={{ fill: '#64748b', fontSize: 11 }}
                    dy={8}
                  />
                  <YAxis
                    axisLine={false}
                    tickLine={false}
                    tick={{ fill: '#94a3b8', fontSize: 11 }}
                    width={46}
                    tickFormatter={(value) =>
                      formatCompactNumber(Number(value))
                    }
                  />
                  <Tooltip
                    cursor={{ stroke: '#bfdbfe', strokeWidth: 1 }}
                    contentStyle={{
                      border: '1px solid #e2e8f0',
                      borderRadius: 6,
                      boxShadow: '0 8px 24px rgba(15, 23, 42, 0.08)',
                      fontSize: 12,
                    }}
                    formatter={(value, name) => {
                      if (name === 'quota') {
                        return [formatLogQuota(Number(value)), '费用']
                      }
                      if (name === 'tokens') {
                        return [formatTokens(Number(value)), 'Tokens']
                      }
                      return [Number(value).toLocaleString(), '请求']
                    }}
                  />
                  <Bar
                    dataKey='requests'
                    fill='#4f7cff'
                    radius={[3, 3, 0, 0]}
                    barSize={10}
                  />
                  <Area
                    type='monotone'
                    dataKey='tokens'
                    fill='url(#personalUsageBlue)'
                    stroke='#2563eb'
                    strokeWidth={2}
                  />
                  <Line
                    type='monotone'
                    dataKey='quota'
                    stroke='#8b5cf6'
                    strokeWidth={1.8}
                    dot={false}
                  />
                </ComposedChart>
              </ResponsiveContainer>
            )}
          </EnterprisePanel>

          <EnterprisePanel
            title='日志明细'
            description='最近请求、费用、延迟与状态'
            bodyClassName='p-0'
          >
            <div className='overflow-x-auto'>
              <table className='min-w-full text-left text-[12px]'>
                <thead className='border-b border-slate-100 bg-slate-50/65 text-[11px] font-semibold text-slate-500'>
                  <tr>
                    <th className='px-3 py-2'>请求ID</th>
                    <th className='px-3 py-2'>时间</th>
                    <th className='px-3 py-2'>模型</th>
                    <th className='px-3 py-2 text-right'>Tokens</th>
                    <th className='px-3 py-2 text-right'>费用</th>
                    <th className='px-3 py-2 text-right'>延迟</th>
                    <th className='px-3 py-2 text-right'>状态</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.length === 0 ? (
                    <tr>
                      <td
                        colSpan={7}
                        className='h-[108px] px-3 text-center text-[12px] text-slate-400'
                      >
                        当前筛选条件下暂无请求日志
                      </td>
                    </tr>
                  ) : (
                    logs.map((log) => {
                      const tokenCount =
                        (log.prompt_tokens ?? 0) + (log.completion_tokens ?? 0)
                      const isError = log.type === LOG_TYPE_ENUM.ERROR
                      return (
                        <tr
                          key={log.id}
                          className='border-b border-slate-100 last:border-0 hover:bg-slate-50/70'
                        >
                          <td className='max-w-[180px] truncate px-3 py-2 font-medium text-slate-700'>
                            {log.request_id || `log_${log.id}`}
                          </td>
                          <td className='px-3 py-2 whitespace-nowrap text-slate-500'>
                            {formatTimestampToDate(log.created_at).slice(5, 16)}
                          </td>
                          <td className='max-w-[170px] truncate px-3 py-2 text-slate-700'>
                            {log.model_name || '-'}
                          </td>
                          <td className='px-3 py-2 text-right text-slate-700 tabular-nums'>
                            {tokenCount.toLocaleString()}
                          </td>
                          <td className='px-3 py-2 text-right text-slate-700 tabular-nums'>
                            {formatLogQuota(log.quota || 0)}
                          </td>
                          <td className='px-3 py-2 text-right text-slate-500 tabular-nums'>
                            {Math.round((log.use_time || 0) * 1000)}ms
                          </td>
                          <td className='px-3 py-2 text-right'>
                            <Badge
                              variant='outline'
                              className={cn(
                                'h-5 rounded-md px-1.5 text-[11px]',
                                isError
                                  ? 'border-rose-200 bg-rose-50 text-rose-700'
                                  : 'border-emerald-200 bg-emerald-50 text-emerald-700'
                              )}
                            >
                              {isError ? '错误' : '成功'}
                            </Badge>
                          </td>
                        </tr>
                      )
                    })
                  )}
                </tbody>
              </table>
            </div>
            <div className='flex items-center justify-between border-t border-slate-100 px-3 py-2 text-[11px] text-slate-500'>
              <span>
                共 {totalLogs.toLocaleString()} 条，当前第 {page} / {pageCount}{' '}
                页
              </span>
              <div className='flex items-center gap-1.5'>
                <Button
                  variant='outline'
                  size='sm'
                  className='h-7 rounded-md bg-white px-2 text-[11px]'
                  disabled={page <= 1}
                  onClick={() =>
                    void navigate({
                      to: '/usage-logs/$section',
                      params: { section: 'common' },
                      search: { ...search, page: Math.max(1, page - 1) },
                    })
                  }
                >
                  上一页
                </Button>
                <Button
                  variant='outline'
                  size='sm'
                  className='h-7 rounded-md bg-white px-2 text-[11px]'
                  disabled={page >= pageCount}
                  onClick={() =>
                    void navigate({
                      to: '/usage-logs/$section',
                      params: { section: 'common' },
                      search: { ...search, page: page + 1 },
                    })
                  }
                >
                  下一页
                </Button>
              </div>
            </div>
          </EnterprisePanel>
        </div>

        <div className='space-y-2'>
          <EnterprisePanel
            title='费用排行'
            description='按模型消费占比'
            bodyClassName='space-y-2'
            action={
              <span className='text-[11px] font-medium text-blue-600'>
                查看全部 ›
              </span>
            }
          >
            {ranking.length === 0 ? (
              <div className='py-8 text-center text-[12px] text-slate-400'>
                暂无模型消费数据
              </div>
            ) : (
              ranking.map((item) => (
                <div key={item.model} className='space-y-1.5'>
                  <div className='flex items-center justify-between gap-3 text-[12px]'>
                    <span className='truncate font-medium text-slate-700'>
                      {item.model}
                    </span>
                    <span className='shrink-0 text-slate-900 tabular-nums'>
                      {formatLogQuota(item.quota)}
                    </span>
                  </div>
                  <div className='flex items-center gap-2'>
                    <div className='h-1.5 flex-1 overflow-hidden rounded-full bg-slate-100'>
                      <div
                        className='h-full rounded-full bg-blue-500'
                        style={{ width: `${Math.max(4, item.share)}%` }}
                      />
                    </div>
                    <span className='w-9 text-right text-[11px] text-slate-500 tabular-nums'>
                      {item.share}%
                    </span>
                  </div>
                </div>
              ))
            )}
          </EnterprisePanel>

          <EnterprisePanel
            title='优化建议'
            description='基于当前用量自动计算'
            bodyClassName='space-y-2'
            action={<Sparkles className='size-4 text-blue-500' />}
          >
            {suggestions.map((item) => (
              <div
                key={item.title}
                className='grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2 border-b border-slate-100 pb-2 last:border-0 last:pb-0'
              >
                <span className='flex size-7 items-center justify-center rounded-md bg-blue-50 text-blue-600 ring-1 ring-blue-100'>
                  <KeyRound className='size-3.5' />
                </span>
                <span className='truncate text-[12px] text-slate-700'>
                  {item.title}
                </span>
                <span className='text-[12px] font-medium text-slate-900'>
                  {item.value}
                </span>
              </div>
            ))}
          </EnterprisePanel>
        </div>
      </div>
    </div>
  )
}
