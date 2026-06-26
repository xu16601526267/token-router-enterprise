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
import {
  Activity,
  AlertTriangle,
  ArrowDownRight,
  ArrowUpRight,
  Boxes,
  CalendarDays,
  Clock3,
  Coins,
  DatabaseZap,
  Download,
  Eye,
  Filter,
  Gauge,
  MoreHorizontal,
  RefreshCw,
  Search,
  SlidersHorizontal,
  Sparkles,
  UsersRound,
  type LucideIcon,
} from 'lucide-react'
import { useMemo, useState, type ReactNode } from 'react'
import {
  Bar,
  CartesianGrid,
  Cell,
  ComposedChart,
  Line,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip as ChartTooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { toast } from 'sonner'

import { EnterprisePanel, EnterpriseStatCard } from '@/components/enterprise'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useEnterpriseConsole } from '@/context/enterprise-console-context'
import {
  formatCompactNumber,
  formatCurrencyUSD,
  formatLogQuota,
  formatNumber,
  formatTokens,
} from '@/lib/format'
import type { TimeGranularity } from '@/lib/time'
import { cn } from '@/lib/utils'

import {
  exportEnterpriseUsageAnalytics,
  getEnterpriseUsageAnalytics,
} from './api'
import type {
  EnterpriseUsageAnalyticsData,
  EnterpriseUsageBreakdownItem,
  EnterpriseUsageLogItem,
} from './types'

const EMPTY_USAGE: EnterpriseUsageAnalyticsData = {
  generated_at: 0,
  range: { start_timestamp: 0, end_timestamp: 0 },
  metrics: {
    total_requests: 0,
    prompt_tokens: 0,
    completion_tokens: 0,
    total_tokens: 0,
    total_quota: 0,
    estimated_cost: 0,
    error_requests: 0,
    error_rate: 0,
    average_latency_ms: 0,
    cache_hit_rate: 0,
  },
  trend: [],
  by_model: [],
  by_user: [],
  by_channel: [],
  by_group: [],
  recent_logs: [],
  total_logs: 0,
  page: 1,
  page_size: 20,
}

const PIE_COLORS = ['#2563eb', '#8b5cf6', '#22c55e', '#f59e0b', '#ef4444']

type UsageInsightTone = 'rose' | 'amber' | 'blue'

const OPTIMIZATION_ICONS: Record<UsageInsightTone, LucideIcon> = {
  rose: AlertTriangle,
  amber: ArrowDownRight,
  blue: Sparkles,
}

const GRANULARITY_LABELS: Record<TimeGranularity, string> = {
  hour: '小时',
  day: '天',
  week: '周',
}

const REQUEST_TYPE_LABELS: Record<string, string> = {
  chat: '对话',
  embedding: 'Embedding',
  rerank: 'Rerank',
  image: '图像',
  audio: '音频',
}

const REQUEST_TYPE_OPTIONS = [
  { value: 'chat', label: '对话' },
  { value: 'embedding', label: 'Embedding' },
  { value: 'rerank', label: 'Rerank' },
  { value: 'image', label: '图像' },
  { value: 'audio', label: '音频' },
]

function formatPercent(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    style: 'percent',
    maximumFractionDigits: 2,
  }).format(Number.isFinite(value) ? value : 0)
}

function formatDate(timestamp: number): string {
  if (timestamp <= 0) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
  }).format(timestamp * 1000)
}

function formatDateTime(timestamp: number): string {
  if (timestamp <= 0) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(timestamp * 1000)
}

function formatTrendLabel(timestamp: number, granularity: TimeGranularity) {
  if (timestamp <= 0) return '-'
  const date = new Date(timestamp * 1000)
  if (granularity === 'hour') {
    return new Intl.DateTimeFormat('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      hour12: false,
    }).format(date)
  }
  return formatDate(timestamp)
}

function logUnitPrice(log: EnterpriseUsageLogItem) {
  const tokens = log.prompt_tokens + log.completion_tokens
  if (tokens <= 0) return '-'
  return `${formatLogQuota((log.quota / tokens) * 1000)} / 1K`
}

function modelChips(items: EnterpriseUsageBreakdownItem[]) {
  if (items.length === 0) return '全部模型策略'
  if (items.length === 1) return items[0]?.name || '全部模型策略'
  return `${items[0]?.name || '模型'} +${items.length - 1}`
}

function getSortLabel(sortBy: string) {
  if (sortBy === 'quota') return '成本'
  if (sortBy === 'use_time') return '延迟'
  return '时间'
}

function formatRequestType(value: string) {
  return REQUEST_TYPE_LABELS[value] ?? '对话'
}

function LogStatusBadge(props: { status: string }) {
  const isSuccess = props.status === 'success'
  return (
    <Badge
      variant='outline'
      className={cn(
        'h-5 rounded px-2 text-[10px]',
        isSuccess
          ? 'border-emerald-200 bg-emerald-50 text-emerald-600'
          : 'border-rose-200 bg-rose-50 text-rose-600'
      )}
    >
      {isSuccess ? '成功' : '失败'}
    </Badge>
  )
}

function TopBreakdownList(props: {
  items: EnterpriseUsageBreakdownItem[]
  value: 'cost' | 'quota'
  emptyText: string
  limit?: number
}) {
  if (props.items.length === 0) {
    return (
      <div className='flex min-h-24 items-center justify-center text-xs text-slate-500'>
        {props.emptyText}
      </div>
    )
  }

  return (
    <div className='space-y-2.5'>
      {props.items.slice(0, props.limit ?? 6).map((item, index) => (
        <div
          key={`${item.name}-${item.quota}`}
          className='grid grid-cols-[22px_minmax(0,1fr)_72px] items-center gap-2 text-xs'
        >
          <span className='flex size-5 items-center justify-center rounded bg-slate-100 text-[10px] font-semibold text-slate-500'>
            {index + 1}
          </span>
          <div className='min-w-0'>
            <div className='flex items-center justify-between gap-2'>
              <span className='truncate font-semibold text-slate-800'>
                {item.name}
              </span>
              <span className='text-[10px] text-slate-500'>
                {formatPercent(item.share)}
              </span>
            </div>
            <div className='mt-1 h-1.5 overflow-hidden rounded-full bg-slate-100'>
              <div
                className='h-full rounded-full bg-blue-500'
                style={{
                  width: `${Math.max(4, Math.min(100, item.share * 100))}%`,
                }}
              />
            </div>
          </div>
          <span className='text-right text-[11px] font-semibold text-slate-900 tabular-nums'>
            {props.value === 'cost'
              ? formatCurrencyUSD(item.cost)
              : formatLogQuota(item.quota)}
          </span>
        </div>
      ))}
    </div>
  )
}

function CostDonutPanel(props: {
  title: string
  subtitle?: string
  items: EnterpriseUsageBreakdownItem[]
  centerText: string
  compact?: boolean
}) {
  return (
    <div
      className={cn(
        'grid h-full min-h-0 items-center gap-2',
        props.compact
          ? 'grid-cols-[104px_minmax(0,1fr)]'
          : 'grid-cols-[118px_minmax(0,1fr)]'
      )}
    >
      <div className={cn('relative', props.compact ? 'h-[112px]' : 'h-[124px]')}>
        {props.items.length === 0 ? (
          <div className='flex h-full items-center justify-center rounded-md bg-slate-50 text-xs text-slate-400'>
            暂无数据
          </div>
        ) : (
          <>
            <ResponsiveContainer
              width='100%'
              height='100%'
              initialDimension={{ width: 118, height: 124 }}
            >
              <PieChart>
                <Pie
                  data={props.items.slice(0, 6)}
                  dataKey='cost'
                  nameKey='name'
                  innerRadius={34}
                  outerRadius={52}
                  paddingAngle={3}
                >
                  {props.items.slice(0, 6).map((item, index) => (
                    <Cell
                      key={item.name}
                      fill={PIE_COLORS[index % PIE_COLORS.length]}
                    />
                  ))}
                </Pie>
                <ChartTooltip
                  formatter={(value) => formatCurrencyUSD(Number(value ?? 0))}
                />
              </PieChart>
            </ResponsiveContainer>
            <div className='pointer-events-none absolute inset-0 flex flex-col items-center justify-center'>
              <span className='text-[13px] font-semibold text-slate-950'>
                {props.centerText}
              </span>
              <span className='text-[10px] text-slate-500'>
                {props.subtitle || '总成本'}
              </span>
            </div>
          </>
        )}
      </div>
      <TopBreakdownList
        items={props.items}
        value='cost'
        emptyText={`暂无${props.title}数据`}
        limit={props.compact ? 3 : 6}
      />
    </div>
  )
}

function UsageLogTable(props: {
  logs: EnterpriseUsageLogItem[]
  page: number
  totalPages: number
  totalLogs: number
  fetching: boolean
  onPrev: () => void
  onNext: () => void
}) {
  if (props.logs.length === 0) {
    return (
      <div className='flex min-h-72 items-center justify-center text-xs text-slate-500'>
        当前筛选条件下暂无调用日志
      </div>
    )
  }

  return (
    <>
      <div className='overflow-x-auto'>
        <Table className='w-full table-fixed text-[11px] [&_td]:h-7 [&_td]:px-2 [&_td]:py-1 [&_td]:text-[11px] [&_td_*]:text-[11px] [&_th]:h-7 [&_th]:px-2 [&_th]:text-[11px] [&_th_*]:text-[11px]'>
          <TableHeader className='bg-slate-50'>
            <TableRow>
              <TableHead className='w-[11%]'>请求 ID</TableHead>
              <TableHead className='w-[9%]'>时间</TableHead>
              <TableHead className='w-[8%]'>客户</TableHead>
              <TableHead className='w-[8%]'>用户</TableHead>
              <TableHead className='w-[10%]'>模型</TableHead>
              <TableHead className='w-[5%]'>类型</TableHead>
              <TableHead className='w-[7%] text-right'>输入</TableHead>
              <TableHead className='w-[7%] text-right'>输出</TableHead>
              <TableHead className='w-[8%] text-right'>单价</TableHead>
              <TableHead className='w-[7%] text-right'>总成本</TableHead>
              <TableHead className='w-[8%]'>渠道</TableHead>
              <TableHead className='w-[6%] text-right'>延迟</TableHead>
              <TableHead className='w-[5%]'>状态</TableHead>
              <TableHead className='w-[4%] text-right'>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {props.logs.map((log) => (
              <TableRow key={log.id} className='hover:bg-blue-50/40'>
                <TableCell className='truncate font-mono text-[11px] text-blue-600'>
                  {log.request_id || `log_${log.id}`}
                </TableCell>
                <TableCell className='truncate text-slate-500'>
                  {formatDateTime(log.created_at)}
                </TableCell>
                <TableCell className='truncate'>
                  <span className='truncate font-semibold text-slate-800'>
                    {log.group || log.username || '默认客户'}
                  </span>
                </TableCell>
                <TableCell className='truncate text-slate-600'>
                  {log.username || 'system'}
                </TableCell>
                <TableCell className='truncate'>
                  <Badge
                    variant='outline'
                    className='h-5 max-w-full truncate rounded px-1.5 text-[10px]'
                  >
                    {log.model_name || '未知模型'}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Badge
                    variant='outline'
                    className='h-5 rounded px-1.5 text-[10px] text-slate-600'
                  >
                    {formatRequestType(log.request_type)}
                  </Badge>
                </TableCell>
                <TableCell className='text-right tabular-nums'>
                  {formatTokens(log.prompt_tokens)}
                </TableCell>
                <TableCell className='text-right tabular-nums'>
                  {formatTokens(log.completion_tokens)}
                </TableCell>
                <TableCell className='text-right text-[11px] tabular-nums'>
                  {logUnitPrice(log)}
                </TableCell>
                <TableCell className='text-right font-semibold tabular-nums'>
                  {formatLogQuota(log.quota)}
                </TableCell>
                <TableCell className='truncate'>
                  {log.channel_name ||
                    (log.channel_id > 0 ? `渠道 #${log.channel_id}` : '-')}
                </TableCell>
                <TableCell className='text-right tabular-nums'>
                  {formatNumber(log.use_time_ms)} ms
                </TableCell>
                <TableCell>
                  <LogStatusBadge status={log.status} />
                </TableCell>
                <TableCell className='text-right'>
                  <Button
                    variant='ghost'
                    size='icon-xs'
                    aria-label='查看日志详情'
                  >
                    <Eye className='size-3.5' />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      <div className='flex items-center justify-between border-t border-slate-100 px-3 py-2'>
        <span className='text-xs text-slate-500'>
          共 {formatNumber(props.totalLogs)} 条
        </span>
        <div className='flex items-center gap-2'>
          <Button
            size='xs'
            variant='outline'
            disabled={props.page <= 1 || props.fetching}
            onClick={props.onPrev}
          >
            上一页
          </Button>
          <span className='text-xs text-slate-500'>
            第 {props.page} / {props.totalPages} 页
          </span>
          <Button
            size='xs'
            variant='outline'
            disabled={props.page >= props.totalPages || props.fetching}
            onClick={props.onNext}
          >
            下一页
          </Button>
        </div>
      </div>
    </>
  )
}

export function EnterpriseUsageAnalytics(props: {
  classicContent?: ReactNode
}) {
  const { range, rangeLabel, granularity, setGranularity } =
    useEnterpriseConsole()
  const [search, setSearch] = useState('')
  const [modelFilter, setModelFilter] = useState('all')
  const [userFilter, setUserFilter] = useState('all')
  const [groupFilter, setGroupFilter] = useState('all')
  const [channelFilter, setChannelFilter] = useState('all')
  const [statusFilter, setStatusFilter] = useState('all')
  const [requestTypeFilter, setRequestTypeFilter] = useState('all')
  const [sortBy, setSortBy] = useState('created_at')
  const [page, setPage] = useState(1)
  const [exportingUsage, setExportingUsage] = useState(false)
  const [showClassicLogs, setShowClassicLogs] = useState(false)

  const usageParams = useMemo(
    () => ({
      start_timestamp: range.start,
      end_timestamp: range.end,
      time_granularity: granularity,
      keyword: search.trim() || undefined,
      model_name: modelFilter !== 'all' ? modelFilter : undefined,
      username: userFilter !== 'all' ? userFilter : undefined,
      group: groupFilter !== 'all' ? groupFilter : undefined,
      channel_id:
        channelFilter !== 'all'
          ? Number.parseInt(channelFilter, 10)
          : undefined,
      status: statusFilter !== 'all' ? statusFilter : undefined,
      request_type:
        requestTypeFilter !== 'all' ? requestTypeFilter : undefined,
      page,
      page_size: 20,
      sort_by: sortBy,
      sort_order: 'desc',
    }),
    [
      range.start,
      range.end,
      granularity,
      search,
      modelFilter,
      userFilter,
      groupFilter,
      channelFilter,
      statusFilter,
      requestTypeFilter,
      page,
      sortBy,
    ]
  )

  const usageQuery = useQuery({
    queryKey: ['enterprise-usage', usageParams],
    queryFn: () => getEnterpriseUsageAnalytics(usageParams),
    staleTime: 30_000,
    refetchInterval: 60_000,
  })

  const usage = usageQuery.data?.data ?? EMPTY_USAGE
  const metrics = usage.metrics
  const chartData = usage.trend.map((point) => ({
    ...point,
    label: formatTrendLabel(point.timestamp, granularity),
    inputTokens: point.prompt_tokens,
    outputTokens: point.completion_tokens,
    cacheRate: point.cache_hit_rate,
    errorRate: point.requests > 0 ? point.errors / point.requests : 0,
  }))
  const totalLogPages = Math.max(
    1,
    Math.ceil((usage.total_logs ?? 0) / (usage.page_size ?? 20))
  )
  const sortLabel = getSortLabel(sortBy)

  const modelOptions = useMemo(
    () => [
      ...new Set([
        ...usage.by_model.map((item) => item.name).filter(Boolean),
        ...usage.recent_logs.map((log) => log.model_name).filter(Boolean),
      ]),
    ],
    [usage.by_model, usage.recent_logs]
  )
  const userOptions = useMemo(
    () => [
      ...new Set([
        ...usage.by_user.map((item) => item.name).filter(Boolean),
        ...usage.recent_logs.map((log) => log.username).filter(Boolean),
      ]),
    ],
    [usage.by_user, usage.recent_logs]
  )
  const groupOptions = useMemo(
    () => [
      ...new Set([
        ...usage.by_group.map((item) => item.name).filter(Boolean),
        ...usage.recent_logs.map((log) => log.group).filter(Boolean),
      ]),
    ],
    [usage.by_group, usage.recent_logs]
  )
  const channelOptions = useMemo(
    () =>
      usage.by_channel
        .filter((item) => item.id != null && item.id > 0)
        .map((item) => ({ id: item.id ?? 0, name: item.name })),
    [usage.by_channel]
  )
  const modelRanking = useMemo(
    () => [...usage.by_model].sort((a, b) => b.quota - a.quota),
    [usage.by_model]
  )

  const anomalyItems = useMemo(() => {
    const items: Array<{
      title: string
      detail: string
      tone: 'rose' | 'amber' | 'blue'
    }> = []
    if (metrics.error_rate >= 0.02) {
      items.push({
        title: '检测到错误率异常',
        detail: `错误率 ${formatPercent(metrics.error_rate)}，建议优先排查失败请求来源。`,
        tone: 'rose',
      })
    }
    if (metrics.average_latency_ms >= 800) {
      items.push({
        title: '平均延迟高于目标',
        detail: `平均延迟 ${formatNumber(metrics.average_latency_ms)} ms，可检查慢渠道或模型。`,
        tone: 'amber',
      })
    }
    if (metrics.cache_hit_rate < 0.3 && metrics.total_requests > 0) {
      items.push({
        title: '缓存命中率偏低',
        detail: `命中率 ${formatPercent(metrics.cache_hit_rate)}，仍有缓存优化空间。`,
        tone: 'blue',
      })
    }
    if (items.length === 0) {
      items.push({
        title: '未发现显著成本异常',
        detail: '请求、成本、失败率与延迟均处于当前筛选范围的正常状态。',
        tone: 'blue',
      })
    }
    return items
  }, [
    metrics.average_latency_ms,
    metrics.cache_hit_rate,
    metrics.error_rate,
    metrics.total_requests,
  ])

  const resetFilters = () => {
    setSearch('')
    setModelFilter('all')
    setUserFilter('all')
    setGroupFilter('all')
    setChannelFilter('all')
    setStatusFilter('all')
    setRequestTypeFilter('all')
    setSortBy('created_at')
    setPage(1)
  }

  const exportUsageCsv = async () => {
    setExportingUsage(true)
    try {
      await exportEnterpriseUsageAnalytics(usageParams)
      toast.success('用量明细已导出')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '导出失败')
    } finally {
      setExportingUsage(false)
    }
  }

  return (
    <div className='enterprise-usage-analytics mx-auto max-w-[1586px] space-y-2 bg-[#f6f8fb] pb-2 text-slate-950'>
      <header className='flex flex-col gap-1.5 px-1 pt-0.5 sm:flex-row sm:items-center sm:justify-between'>
        <div className='min-w-0'>
          <h1 className='text-lg leading-5 font-semibold text-slate-950'>
            用量日志与成本分析
          </h1>
          <p className='mt-0.5 text-[11px] leading-4 text-slate-500'>
            按客户、部门、模型、渠道追踪请求、Token 与成本
          </p>
        </div>
        <div className='flex shrink-0 items-center gap-1.5'>
          <Button
            variant='outline'
            className='h-7 rounded-md border-slate-200 bg-white px-2 text-[11px] font-semibold text-slate-700 shadow-none hover:bg-slate-50'
            onClick={() => void exportUsageCsv()}
            disabled={exportingUsage || usageQuery.isFetching}
          >
            <Download className='size-3' />
            {exportingUsage ? '导出中' : '导出报表'}
          </Button>
          <Button
            variant='outline'
            className='h-7 rounded-md border-slate-200 bg-white px-2 text-[11px] font-semibold text-slate-700 shadow-none hover:bg-slate-50'
          >
            <SlidersHorizontal className='size-3' />
            自定义视图
          </Button>
          <Button
            variant='outline'
            size='icon'
            className='size-7 rounded-md border-slate-200 bg-white text-slate-600 shadow-none hover:bg-slate-50'
            aria-label='刷新用量数据'
            onClick={() => void usageQuery.refetch()}
            disabled={usageQuery.isFetching}
          >
            <RefreshCw
              className={cn(
                'size-3.5',
                usageQuery.isFetching && 'animate-spin'
              )}
            />
          </Button>
        </div>
      </header>

      <section className='grid gap-1.5 md:grid-cols-3 xl:grid-cols-6'>
        <EnterpriseStatCard
          title='总请求数'
          value={formatCompactNumber(metrics.total_requests)}
          helper='当前筛选范围'
          icon={Activity}
          tone='blue'
          loading={usageQuery.isLoading}
        />
        <EnterpriseStatCard
          title='输入 Tokens'
          value={formatTokens(metrics.prompt_tokens)}
          helper='Prompt 消耗'
          icon={DatabaseZap}
          tone='amber'
          loading={usageQuery.isLoading}
        />
        <EnterpriseStatCard
          title='输出 Tokens'
          value={formatTokens(metrics.completion_tokens)}
          helper='Completion 消耗'
          icon={Boxes}
          tone='violet'
          loading={usageQuery.isLoading}
        />
        <EnterpriseStatCard
          title='缓存命中率'
          value={formatPercent(metrics.cache_hit_rate)}
          helper='Usage Ledger 统计'
          icon={Gauge}
          tone='emerald'
          loading={usageQuery.isLoading}
        />
        <EnterpriseStatCard
          title='总成本'
          value={formatCurrencyUSD(metrics.estimated_cost)}
          helper='按额度换算'
          icon={Coins}
          tone='amber'
          loading={usageQuery.isLoading}
        />
        <EnterpriseStatCard
          title='错误率'
          value={formatPercent(metrics.error_rate)}
          helper={`${formatNumber(metrics.error_requests)} 次失败`}
          icon={AlertTriangle}
          tone={metrics.error_rate >= 0.02 ? 'rose' : 'emerald'}
          loading={usageQuery.isLoading}
        />
      </section>

      <EnterprisePanel bodyClassName='p-2'>
        <div className='grid gap-1.5 min-[1420px]:grid-cols-[170px_repeat(6,minmax(104px,1fr))_minmax(190px,1.5fr)_76px_76px_92px]'>
          <div className='flex h-8 items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2 text-[11px] font-medium text-slate-700'>
            <CalendarDays className='size-3.5 text-slate-400' />
            <span className='truncate'>{rangeLabel}</span>
          </div>
          <NativeSelect
            value={userFilter}
            className='h-8 rounded-md bg-white text-xs'
            onChange={(event) => {
              setUserFilter(event.target.value)
              setPage(1)
            }}
          >
            <NativeSelectOption value='all'>全部客户</NativeSelectOption>
            {userOptions.map((user) => (
              <NativeSelectOption key={user} value={user}>
                {user}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          <NativeSelect
            value={groupFilter}
            className='h-8 rounded-md bg-white text-xs'
            onChange={(event) => {
              setGroupFilter(event.target.value)
              setPage(1)
            }}
          >
            <NativeSelectOption value='all'>全部部门</NativeSelectOption>
            {groupOptions.map((group) => (
              <NativeSelectOption key={group} value={group}>
                {group}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          <NativeSelect
            value={channelFilter}
            className='h-8 rounded-md bg-white text-xs'
            onChange={(event) => {
              setChannelFilter(event.target.value)
              setPage(1)
            }}
          >
            <NativeSelectOption value='all'>全部供应商</NativeSelectOption>
            {channelOptions.map((channel) => (
              <NativeSelectOption key={channel.id} value={String(channel.id)}>
                {channel.name}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          <NativeSelect
            value={modelFilter}
            className='h-8 rounded-md bg-white text-xs'
            onChange={(event) => {
              setModelFilter(event.target.value)
              setPage(1)
            }}
          >
            <NativeSelectOption value='all'>全部模型策略</NativeSelectOption>
            {modelOptions.map((model) => (
              <NativeSelectOption key={model} value={model}>
                {model}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          <NativeSelect
            value={statusFilter}
            className='h-8 rounded-md bg-white text-xs'
            onChange={(event) => {
              setStatusFilter(event.target.value)
              setPage(1)
            }}
          >
            <NativeSelectOption value='all'>全部状态</NativeSelectOption>
            <NativeSelectOption value='success'>成功</NativeSelectOption>
            <NativeSelectOption value='error'>失败</NativeSelectOption>
          </NativeSelect>
          <NativeSelect
            value={requestTypeFilter}
            className='h-8 rounded-md bg-white text-xs'
            onChange={(event) => {
              setRequestTypeFilter(event.target.value)
              setPage(1)
            }}
          >
            <NativeSelectOption value='all'>全部请求类型</NativeSelectOption>
            {REQUEST_TYPE_OPTIONS.map((item) => (
              <NativeSelectOption key={item.value} value={item.value}>
                {item.label}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          <div className='relative'>
            <Search className='pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-slate-400' />
            <Input
              value={search}
              className='h-8 rounded-md bg-white pl-8 text-xs'
              placeholder='搜索客户 / 请求 / 模型 / 渠道'
              onChange={(event) => {
                setSearch(event.target.value)
                setPage(1)
              }}
            />
          </div>
          <Button
            variant='outline'
            className='h-8 rounded-md border-slate-200 bg-white px-2 text-xs font-semibold text-slate-700 shadow-none hover:bg-slate-50'
            onClick={resetFilters}
          >
            重置
          </Button>
          <Button
            className='h-8 rounded-md bg-blue-600 px-2 text-xs font-semibold text-white shadow-none hover:bg-blue-700'
            onClick={() => void usageQuery.refetch()}
            disabled={usageQuery.isFetching}
          >
            <Filter className='size-3.5' />
            筛选
          </Button>
          <Button
            variant='outline'
            className='h-8 rounded-md border-slate-200 bg-white px-2 text-xs font-semibold text-slate-700 shadow-none hover:bg-slate-50'
          >
            保存视图
          </Button>
        </div>
      </EnterprisePanel>

      <section className='grid items-start gap-2 min-[1360px]:grid-cols-[minmax(0,1fr)_350px]'>
        <div className='grid min-w-0 gap-2'>
          <div className='grid gap-2 min-[1180px]:grid-cols-[minmax(360px,1.28fr)_minmax(260px,.82fr)_minmax(280px,.9fr)]'>
            <EnterprisePanel
              title='Tokens 趋势'
              description='输入 Tokens、输出 Tokens 与缓存命中率'
              action={
                <NativeSelect
                  value={granularity}
                  className='h-6 rounded-md bg-white text-[11px]'
                  onChange={(event) =>
                    setGranularity(event.target.value as TimeGranularity)
                  }
                >
                  <NativeSelectOption value='hour'>按小时</NativeSelectOption>
                  <NativeSelectOption value='day'>按天</NativeSelectOption>
                  <NativeSelectOption value='week'>按周</NativeSelectOption>
                </NativeSelect>
              }
              bodyClassName='h-36 p-2'
            >
              {chartData.length === 0 ? (
                <div className='flex h-full items-center justify-center text-xs text-slate-500'>
                  当前时间范围内暂无趋势数据
                </div>
              ) : (
                <ResponsiveContainer
                  width='100%'
                  height='100%'
                  initialDimension={{ width: 520, height: 144 }}
                >
                  <ComposedChart
                    data={chartData}
                    margin={{ top: 8, right: 6, left: -18, bottom: 0 }}
                  >
                    <CartesianGrid
                      strokeDasharray='4 6'
                      vertical={false}
                      stroke='#dbe3ef'
                    />
                    <XAxis
                      dataKey='label'
                      axisLine={false}
                      tickLine={false}
                      tick={{ fontSize: 10, fill: '#64748b' }}
                    />
                    <YAxis
                      axisLine={false}
                      tickLine={false}
                      tickFormatter={(value) =>
                        formatCompactNumber(Number(value))
                      }
                      tick={{ fontSize: 10, fill: '#64748b' }}
                    />
                    <ChartTooltip
                      contentStyle={{
                        borderRadius: 6,
                        borderColor: '#e2e8f0',
                        background: '#fff',
                        fontSize: 12,
                      }}
                      formatter={(value, name) => {
                        if (name === 'cacheRate') {
                          return [
                            formatPercent(Number(value ?? 0)),
                            '缓存命中率',
                          ]
                        }
                        return [
                          formatTokens(Number(value ?? 0)),
                          name === 'inputTokens'
                            ? '输入 Tokens'
                            : '输出 Tokens',
                        ]
                      }}
                    />
                    <Bar
                      dataKey='inputTokens'
                      name='输入 Tokens'
                      fill='#3b82f6'
                      radius={[3, 3, 0, 0]}
                      maxBarSize={24}
                    />
                    <Bar
                      dataKey='outputTokens'
                      name='输出 Tokens'
                      fill='#8b5cf6'
                      radius={[3, 3, 0, 0]}
                      maxBarSize={24}
                    />
                    <Line
                      type='monotone'
                      dataKey='cacheRate'
                      name='缓存命中率'
                      yAxisId={0}
                      stroke='#22c55e'
                      dot={false}
                      strokeWidth={1.8}
                    />
                  </ComposedChart>
                </ResponsiveContainer>
              )}
            </EnterprisePanel>

            <EnterprisePanel
              title='成本分布'
              description='按模型策略聚合'
              bodyClassName='h-36 p-2'
              action={
                <Badge
                  variant='outline'
                  className='h-5 rounded px-2 text-[10px]'
                >
                  {modelChips(usage.by_model)}
                </Badge>
              }
            >
              <CostDonutPanel
                title='成本分布'
                items={usage.by_model}
                centerText={formatCurrencyUSD(metrics.estimated_cost)}
                compact
              />
            </EnterprisePanel>

            <EnterprisePanel
              title='模型用量排行'
              description='按 Token 成本排序'
              bodyClassName='h-36 p-2'
              action={
                <NativeSelect
                  value={sortBy}
                  className='h-6 rounded-md bg-white text-[11px]'
                  onChange={(event) => setSortBy(event.target.value)}
                >
                  <NativeSelectOption value='created_at'>
                    最新
                  </NativeSelectOption>
                  <NativeSelectOption value='quota'>成本</NativeSelectOption>
                  <NativeSelectOption value='use_time'>延迟</NativeSelectOption>
                </NativeSelect>
              }
            >
              <TopBreakdownList
                items={modelRanking}
                value='quota'
                emptyText='暂无模型用量数据'
                limit={4}
              />
            </EnterprisePanel>
          </div>

          <EnterprisePanel
            title='错误率趋势'
            description='错误率与错误请求数按当前粒度聚合'
            action={
              <Badge variant='outline' className='h-5 rounded px-2 text-[10px]'>
                {GRANULARITY_LABELS[granularity]}
              </Badge>
            }
            bodyClassName='h-24 p-2'
          >
            {chartData.length === 0 ? (
              <div className='flex h-full items-center justify-center text-xs text-slate-500'>
                暂无服务质量趋势
              </div>
            ) : (
              <ResponsiveContainer
                width='100%'
                height='100%'
                initialDimension={{ width: 920, height: 96 }}
              >
                <ComposedChart
                  data={chartData}
                  margin={{ top: 8, right: 8, left: -18, bottom: 0 }}
                >
                  <CartesianGrid
                    strokeDasharray='4 6'
                    vertical={false}
                    stroke='#dbe3ef'
                  />
                  <XAxis
                    dataKey='label'
                    axisLine={false}
                    tickLine={false}
                    tick={{ fontSize: 10, fill: '#64748b' }}
                  />
                  <YAxis
                    axisLine={false}
                    tickLine={false}
                    tick={{ fontSize: 10, fill: '#64748b' }}
                  />
                  <ChartTooltip
                    contentStyle={{
                      borderRadius: 6,
                      borderColor: '#e2e8f0',
                      background: '#fff',
                      fontSize: 12,
                    }}
                    formatter={(value, name) =>
                      name === 'errorRate'
                        ? [formatPercent(Number(value ?? 0)), '错误率']
                        : [formatNumber(Number(value ?? 0)), '错误请求数']
                    }
                  />
                  <Bar
                    dataKey='errors'
                    name='错误请求数'
                    fill='#3b82f6'
                    radius={[2, 2, 0, 0]}
                    maxBarSize={12}
                  />
                  <Line
                    type='monotone'
                    dataKey='errorRate'
                    name='错误率'
                    stroke='#ef4444'
                    dot={false}
                    strokeWidth={1.6}
                  />
                </ComposedChart>
              </ResponsiveContainer>
            )}
          </EnterprisePanel>

          <EnterprisePanel
            title='用量日志'
            description={`当前展示 ${usage.recent_logs.length} 条，按 ${sortLabel} 排序`}
            bodyClassName='p-0'
          >
            <UsageLogTable
              logs={usage.recent_logs}
              page={page}
              totalPages={totalLogPages}
              totalLogs={usage.total_logs ?? 0}
              fetching={usageQuery.isFetching}
              onPrev={() => setPage((value) => Math.max(1, value - 1))}
              onNext={() =>
                setPage((value) => Math.min(totalLogPages, value + 1))
              }
            />
          </EnterprisePanel>
        </div>

        <aside className='grid gap-2'>
          <EnterprisePanel
            title='成本中心控制'
            description='按部门 / 分组监测预算'
            action={
              <Badge variant='outline' className='h-5 rounded px-2 text-[10px]'>
                实时
              </Badge>
            }
            bodyClassName='h-36 p-2'
          >
            <CostDonutPanel
              title='成本中心'
              items={usage.by_group}
              centerText={formatCurrencyUSD(metrics.estimated_cost)}
              compact
            />
          </EnterprisePanel>

          <EnterprisePanel
            title='热点客户'
            description='按成本贡献排序'
            action={
              <Button
                variant='ghost'
                size='xs'
                className='h-6 px-1.5 text-[11px] text-blue-600'
              >
                查看全部
                <MoreHorizontal className='size-3.5' />
              </Button>
            }
          >
            <TopBreakdownList
              items={usage.by_user}
              value='cost'
              emptyText='暂无客户用量数据'
              limit={5}
            />
          </EnterprisePanel>

          <EnterprisePanel title='优化建议'>
            <div className='space-y-2'>
              {anomalyItems.map((item) => {
                const Icon = OPTIMIZATION_ICONS[item.tone]
                return (
                  <article
                    key={item.title}
                    className={cn(
                      'rounded-md border p-2.5',
                      item.tone === 'rose' &&
                        'border-rose-100 bg-rose-50/70 text-rose-700',
                      item.tone === 'amber' &&
                        'border-amber-100 bg-amber-50/70 text-amber-700',
                      item.tone === 'blue' &&
                        'border-blue-100 bg-blue-50/70 text-blue-700'
                    )}
                  >
                    <div className='flex items-start gap-2'>
                      <span className='mt-0.5 flex size-7 shrink-0 items-center justify-center rounded bg-white/70'>
                        <Icon className='size-3.5' />
                      </span>
                      <div className='min-w-0'>
                        <p className='text-xs font-semibold'>{item.title}</p>
                        <p className='mt-1 text-[11px] leading-4 text-slate-600'>
                          {item.detail}
                        </p>
                      </div>
                    </div>
                  </article>
                )
              })}
              <article className='rounded-md border border-emerald-100 bg-emerald-50/70 p-2.5 text-emerald-700'>
                <div className='flex items-start gap-2'>
                  <span className='mt-0.5 flex size-7 shrink-0 items-center justify-center rounded bg-white/70'>
                    <ArrowUpRight className='size-3.5' />
                  </span>
                  <div className='min-w-0'>
                    <p className='text-xs font-semibold'>优化模型组合</p>
                    <p className='mt-1 text-[11px] leading-4 text-slate-600'>
                      Top 模型 {usage.by_model[0]?.name || '暂无'} 占比{' '}
                      {formatPercent(usage.by_model[0]?.share ?? 0)}
                      ，可按低复杂度请求拆分路由。
                    </p>
                  </div>
                </div>
              </article>
            </div>
          </EnterprisePanel>

          <EnterprisePanel title='运行概况'>
            <div className='grid grid-cols-2 gap-2'>
              <div className='rounded-md border border-slate-100 bg-slate-50 p-2'>
                <Clock3 className='size-3.5 text-violet-500' />
                <p className='mt-1.5 text-base font-semibold tabular-nums'>
                  {formatNumber(metrics.average_latency_ms)} ms
                </p>
                <p className='text-[10px] text-slate-500'>平均延迟</p>
              </div>
              <div className='rounded-md border border-slate-100 bg-slate-50 p-2'>
                <UsersRound className='size-3.5 text-blue-500' />
                <p className='mt-1.5 text-base font-semibold tabular-nums'>
                  {formatNumber(usage.by_user.length)}
                </p>
                <p className='text-[10px] text-slate-500'>活跃主体</p>
              </div>
            </div>
          </EnterprisePanel>
        </aside>
      </section>

      {props.classicContent && (
        <details
          className='rounded-md border border-slate-200 bg-white px-3 py-2 text-xs text-slate-600'
          onToggle={(event) => setShowClassicLogs(event.currentTarget.open)}
        >
          <summary className='flex cursor-pointer items-center gap-2 font-semibold text-slate-800'>
            <SlidersHorizontal className='size-3.5 text-blue-600' />
            高级日志工作台
          </summary>
          {showClassicLogs && (
            <div className='mt-2 h-[620px] min-h-0'>{props.classicContent}</div>
          )}
        </details>
      )}
    </div>
  )
}
