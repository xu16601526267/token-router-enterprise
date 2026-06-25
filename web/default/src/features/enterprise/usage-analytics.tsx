import { useQuery } from '@tanstack/react-query'
import {
  Activity,
  AlertTriangle,
  ArrowDownRight,
  ArrowUpRight,
  Boxes,
  Clock3,
  Coins,
  DatabaseZap,
  Download,
  Filter,
  Gauge,
  RefreshCw,
  Search,
  Sparkles,
  UsersRound,
} from 'lucide-react'
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
import { useMemo, useState, type ReactNode } from 'react'
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip as ChartTooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { toast } from 'sonner'

import {
  EnterprisePageHeader,
  EnterprisePanel,
  EnterpriseStatCard,
} from '@/components/enterprise'
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
import {
  formatCompactNumber,
  formatCurrencyUSD,
  formatLogQuota,
  formatNumber,
  formatTokens,
} from '@/lib/format'
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
  page_size: 50,
}

const PIE_COLORS = [
  'var(--chart-1)',
  'var(--chart-2)',
  'var(--chart-3)',
  'var(--chart-4)',
  'var(--chart-5)',
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

function BreakdownList(props: {
  items: EnterpriseUsageBreakdownItem[]
  emptyText: string
}) {
  if (props.items.length === 0) {
    return (
      <div className='text-muted-foreground flex min-h-52 items-center justify-center text-sm'>
        {props.emptyText}
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      {props.items.slice(0, 6).map((item, position) => (
        <div key={`${item.name}-${item.quota}`} className='space-y-2'>
          <div className='flex items-center justify-between gap-3 text-xs'>
            <div className='flex min-w-0 items-center gap-2.5'>
              <span className='bg-muted text-muted-foreground flex size-6 shrink-0 items-center justify-center rounded-lg text-[10px] font-semibold'>
                {position + 1}
              </span>
              <span className='truncate font-medium'>{item.name}</span>
            </div>
            <div className='shrink-0 text-right'>
              <p className='font-semibold tabular-nums'>
                {formatCurrencyUSD(item.cost)}
              </p>
              <p className='text-muted-foreground text-[10px]'>
                {formatPercent(item.share)}
              </p>
            </div>
          </div>
          <div className='bg-muted h-1.5 overflow-hidden rounded-full'>
            <div
              className='h-full rounded-full bg-linear-to-r from-blue-500 to-violet-500'
              style={{
                width: `${Math.max(3, Math.min(100, item.share * 100))}%`,
              }}
            />
          </div>
        </div>
      ))}
    </div>
  )
}

function LogStatusBadge(props: { status: string }) {
  const isSuccess = props.status === 'success'
  return (
    <Badge
      variant='outline'
      className={cn(
        'text-[10px]',
        isSuccess
          ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-600'
          : 'border-rose-500/20 bg-rose-500/10 text-rose-600'
      )}
    >
      {isSuccess ? '成功' : '失败'}
    </Badge>
  )
}

function UsageLogTable(props: { logs: EnterpriseUsageLogItem[] }) {
  if (props.logs.length === 0) {
    return (
      <div className='text-muted-foreground flex min-h-72 items-center justify-center text-sm'>
        当前筛选条件下暂无调用日志
      </div>
    )
  }

  return (
    <div className='overflow-x-auto'>
      <Table>
        <TableHeader>
          <TableRow className='bg-muted/35'>
            <TableHead className='min-w-36'>请求编号</TableHead>
            <TableHead className='min-w-36'>时间</TableHead>
            <TableHead className='min-w-28'>客户 / 用户</TableHead>
            <TableHead className='min-w-32'>模型</TableHead>
            <TableHead className='text-right'>输入</TableHead>
            <TableHead className='text-right'>输出</TableHead>
            <TableHead className='text-right'>成本</TableHead>
            <TableHead className='min-w-28'>渠道</TableHead>
            <TableHead className='text-right'>延迟</TableHead>
            <TableHead>状态</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {props.logs.map((log) => (
            <TableRow key={log.id} className='hover:bg-muted/25'>
              <TableCell className='text-muted-foreground font-mono text-[11px]'>
                {log.request_id || `log_${log.id}`}
              </TableCell>
              <TableCell className='text-muted-foreground text-xs'>
                {formatDateTime(log.created_at)}
              </TableCell>
              <TableCell>
                <p className='text-xs font-medium'>
                  {log.username || '系统调用'}
                </p>
                <p className='text-muted-foreground mt-0.5 text-[10px]'>
                  {log.group || log.token_name || '默认分组'}
                </p>
              </TableCell>
              <TableCell>
                <Badge
                  variant='secondary'
                  className='max-w-36 truncate text-[10px]'
                >
                  {log.model_name || '未知模型'}
                </Badge>
              </TableCell>
              <TableCell className='text-right text-xs tabular-nums'>
                {formatTokens(log.prompt_tokens)}
              </TableCell>
              <TableCell className='text-right text-xs tabular-nums'>
                {formatTokens(log.completion_tokens)}
              </TableCell>
              <TableCell className='text-right text-xs font-semibold tabular-nums'>
                {formatLogQuota(log.quota)}
              </TableCell>
              <TableCell className='max-w-32 truncate text-xs'>
                {log.channel_name ||
                  (log.channel_id > 0 ? `渠道 #${log.channel_id}` : '-')}
              </TableCell>
              <TableCell className='text-right text-xs tabular-nums'>
                {formatNumber(log.use_time_ms)} ms
              </TableCell>
              <TableCell>
                <LogStatusBadge status={log.status} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

export function EnterpriseUsageAnalytics(props: {
  classicContent?: ReactNode
}) {
  const [search, setSearch] = useState('')
  const [modelFilter, setModelFilter] = useState('all')
  const [statusFilter, setStatusFilter] = useState('all')
  const [page, setPage] = useState(1)
  const [exportingUsage, setExportingUsage] = useState(false)
  const range = useMemo(() => {
    const end = Math.floor(Date.now() / 1000)
    return { start: end - 7 * 24 * 60 * 60, end }
  }, [])
  const usageParams = useMemo(
    () => ({
      start_timestamp: range.start,
      end_timestamp: range.end,
      keyword: search.trim() || undefined,
      model_name: modelFilter !== 'all' ? modelFilter : undefined,
      status: statusFilter !== 'all' ? statusFilter : undefined,
      page,
      page_size: 50,
    }),
    [modelFilter, page, range.end, range.start, search, statusFilter]
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
    label: formatDate(point.timestamp),
    cost: point.quota / 500_000,
    errorRate: point.requests > 0 ? point.errors / point.requests : 0,
  }))
  const models = useMemo(
    () => [
      ...new Set([
        ...usage.by_model.map((item) => item.name).filter(Boolean),
        ...usage.recent_logs.map((log) => log.model_name).filter(Boolean),
      ]),
    ],
    [usage.by_model, usage.recent_logs]
  )
  const totalLogPages = Math.max(
    1,
    Math.ceil((usage.total_logs ?? 0) / (usage.page_size ?? 50))
  )

  const anomalyItems = useMemo(() => {
    const items: Array<{
      title: string
      detail: string
      tone: 'rose' | 'amber' | 'blue'
    }> = []
    if (metrics.error_rate >= 0.02) {
      items.push({
        title: '错误率超过企业告警线',
        detail: `当前错误率 ${formatPercent(metrics.error_rate)}，建议检查高失败渠道。`,
        tone: 'rose',
      })
    }
    if (metrics.average_latency_ms >= 800) {
      items.push({
        title: '平均延迟持续偏高',
        detail: `当前平均延迟 ${formatNumber(metrics.average_latency_ms)} ms，可启用低延迟路由。`,
        tone: 'amber',
      })
    }
    if (metrics.cache_hit_rate < 0.3 && metrics.total_requests > 0) {
      items.push({
        title: '缓存命中率存在优化空间',
        detail: `当前命中率 ${formatPercent(metrics.cache_hit_rate)}，建议检查缓存键和 TTL。`,
        tone: 'blue',
      })
    }
    if (items.length === 0) {
      items.push({
        title: '当前未发现显著异常',
        detail: '用量、延迟与失败率均处于可控范围。',
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

  const dateRangeLabel = `${formatDate(usage.range.start_timestamp || range.start)} - ${formatDate(usage.range.end_timestamp || range.end)}`
  const exportUsageCsv = async () => {
    setExportingUsage(true)
    try {
      await exportEnterpriseUsageAnalytics({
        ...usageParams,
      })
      toast.success('用量明细已导出')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '导出失败')
    } finally {
      setExportingUsage(false)
    }
  }

  return (
    <div className='enterprise-dashboard space-y-4 pb-2 sm:space-y-5'>
      <EnterprisePageHeader
        eyebrow='企业经营分析'
        title='用量日志与成本分析'
        description='按客户、部门、模型和渠道追踪请求、Token、延迟与成本，支持运营、财务和客户成功协同分析。'
        actions={
          <>
            <Badge
              variant='outline'
              className='bg-background/70 h-8 rounded-lg px-3 text-xs font-normal'
            >
              {dateRangeLabel}
            </Badge>
            <Button
              variant='outline'
              size='sm'
              onClick={() => void exportUsageCsv()}
              disabled={exportingUsage || usageQuery.isFetching}
            >
              <Download className='size-4' />
              {exportingUsage ? '导出中' : '导出明细'}
            </Button>
            <Button
              variant='outline'
              size='sm'
              onClick={() => void usageQuery.refetch()}
              disabled={usageQuery.isFetching}
            >
              <RefreshCw
                className={cn(
                  'size-4',
                  usageQuery.isFetching && 'animate-spin'
                )}
              />
              刷新数据
            </Button>
          </>
        }
      />

      <div className='grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-6'>
        <EnterpriseStatCard
          title='总请求数'
          value={formatCompactNumber(metrics.total_requests)}
          helper='近 7 天企业调用'
          icon={Activity}
          tone='blue'
          loading={usageQuery.isLoading}
        />
        <EnterpriseStatCard
          title='输入 Tokens'
          value={formatTokens(metrics.prompt_tokens)}
          helper='Prompt 消耗'
          icon={DatabaseZap}
          tone='violet'
          loading={usageQuery.isLoading}
        />
        <EnterpriseStatCard
          title='输出 Tokens'
          value={formatTokens(metrics.completion_tokens)}
          helper='Completion 消耗'
          icon={Boxes}
          tone='emerald'
          loading={usageQuery.isLoading}
        />
        <EnterpriseStatCard
          title='缓存命中率'
          value={formatPercent(metrics.cache_hit_rate)}
          helper='来自 Usage Ledger'
          icon={Gauge}
          tone='amber'
          loading={usageQuery.isLoading}
        />
        <EnterpriseStatCard
          title='总成本'
          value={formatCurrencyUSD(metrics.estimated_cost)}
          helper='按系统额度换算'
          icon={Coins}
          tone='violet'
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
      </div>

      <EnterprisePanel bodyClassName='p-3 sm:p-4'>
        <div className='grid gap-3 lg:grid-cols-[1.2fr_repeat(2,minmax(0,.7fr))_auto]'>
          <label className='relative block'>
            <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2' />
            <Input
              value={search}
              onChange={(event) => {
                setSearch(event.target.value)
                setPage(1)
              }}
              placeholder='搜索客户、用户、请求编号、模型或渠道'
              className='pl-9'
            />
          </label>
          <NativeSelect
            value={modelFilter}
            onChange={(event) => {
              setModelFilter(event.target.value)
              setPage(1)
            }}
          >
            <NativeSelectOption value='all'>全部模型</NativeSelectOption>
            {models.map((model) => (
              <NativeSelectOption key={model} value={model}>
                {model}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          <NativeSelect
            value={statusFilter}
            onChange={(event) => {
              setStatusFilter(event.target.value)
              setPage(1)
            }}
          >
            <NativeSelectOption value='all'>全部状态</NativeSelectOption>
            <NativeSelectOption value='success'>成功</NativeSelectOption>
            <NativeSelectOption value='error'>失败</NativeSelectOption>
          </NativeSelect>
          <Button
            variant='outline'
            onClick={() => {
              setSearch('')
              setModelFilter('all')
              setStatusFilter('all')
              setPage(1)
            }}
          >
            <Filter className='size-4' />
            重置筛选
          </Button>
        </div>
      </EnterprisePanel>

      <div className='grid gap-4 xl:grid-cols-[minmax(0,1.75fr)_minmax(290px,.65fr)]'>
        <div className='grid min-w-0 gap-4 lg:grid-cols-2'>
          <EnterprisePanel
            className='lg:col-span-2'
            title='Tokens 与成本趋势'
            description='输入、输出 Token 和成本按天汇总'
            action={<Badge variant='secondary'>按天</Badge>}
            bodyClassName='h-80 p-3 sm:p-5'
          >
            {chartData.length === 0 ? (
              <div className='text-muted-foreground flex h-full items-center justify-center text-sm'>
                当前时间范围内暂无趋势数据
              </div>
            ) : (
              <ResponsiveContainer width='100%' height='100%'>
                <AreaChart
                  data={chartData}
                  margin={{ top: 12, right: 12, left: -20, bottom: 0 }}
                >
                  <defs>
                    <linearGradient
                      id='usagePrompt'
                      x1='0'
                      y1='0'
                      x2='0'
                      y2='1'
                    >
                      <stop
                        offset='5%'
                        stopColor='var(--chart-1)'
                        stopOpacity={0.35}
                      />
                      <stop
                        offset='95%'
                        stopColor='var(--chart-1)'
                        stopOpacity={0.02}
                      />
                    </linearGradient>
                    <linearGradient
                      id='usageCompletion'
                      x1='0'
                      y1='0'
                      x2='0'
                      y2='1'
                    >
                      <stop
                        offset='5%'
                        stopColor='var(--chart-2)'
                        stopOpacity={0.3}
                      />
                      <stop
                        offset='95%'
                        stopColor='var(--chart-2)'
                        stopOpacity={0.02}
                      />
                    </linearGradient>
                  </defs>
                  <CartesianGrid
                    strokeDasharray='4 6'
                    vertical={false}
                    stroke='var(--border)'
                    opacity={0.7}
                  />
                  <XAxis
                    dataKey='label'
                    axisLine={false}
                    tickLine={false}
                    tick={{ fontSize: 11, fill: 'var(--muted-foreground)' }}
                  />
                  <YAxis
                    axisLine={false}
                    tickLine={false}
                    tickFormatter={(value) =>
                      formatCompactNumber(Number(value))
                    }
                    tick={{ fontSize: 11, fill: 'var(--muted-foreground)' }}
                  />
                  <ChartTooltip
                    contentStyle={{
                      borderRadius: 12,
                      borderColor: 'var(--border)',
                      background: 'var(--popover)',
                    }}
                    formatter={(value, name) => [
                      formatTokens(Number(value ?? 0)),
                      name === 'prompt_tokens' ? '输入 Tokens' : '输出 Tokens',
                    ]}
                  />
                  <Area
                    type='monotone'
                    dataKey='prompt_tokens'
                    stroke='var(--chart-1)'
                    fill='url(#usagePrompt)'
                    strokeWidth={2.2}
                  />
                  <Area
                    type='monotone'
                    dataKey='completion_tokens'
                    stroke='var(--chart-2)'
                    fill='url(#usageCompletion)'
                    strokeWidth={2.2}
                  />
                </AreaChart>
              </ResponsiveContainer>
            )}
          </EnterprisePanel>

          <EnterprisePanel
            title='模型成本分布'
            description='按模型查看费用与调用占比'
          >
            <BreakdownList
              items={usage.by_model}
              emptyText='暂无模型成本数据'
            />
          </EnterprisePanel>

          <EnterprisePanel
            title='成本中心占比'
            description='按用户分组聚合企业内部成本'
          >
            {usage.by_group.length === 0 ? (
              <div className='text-muted-foreground flex min-h-52 items-center justify-center text-sm'>
                暂无成本中心数据
              </div>
            ) : (
              <div className='grid min-h-52 grid-cols-[140px_minmax(0,1fr)] items-center gap-2'>
                <ResponsiveContainer width='100%' height={150}>
                  <PieChart>
                    <Pie
                      data={usage.by_group.slice(0, 5)}
                      dataKey='cost'
                      nameKey='name'
                      innerRadius={40}
                      outerRadius={64}
                      paddingAngle={3}
                    >
                      {usage.by_group.slice(0, 5).map((item, index) => (
                        <Cell
                          key={item.name}
                          fill={PIE_COLORS[index % PIE_COLORS.length]}
                        />
                      ))}
                    </Pie>
                    <ChartTooltip
                      formatter={(value) =>
                        formatCurrencyUSD(Number(value ?? 0))
                      }
                    />
                  </PieChart>
                </ResponsiveContainer>
                <div className='space-y-2.5'>
                  {usage.by_group.slice(0, 5).map((item, index) => (
                    <div
                      key={item.name}
                      className='flex items-center justify-between gap-3 text-xs'
                    >
                      <span className='flex min-w-0 items-center gap-2'>
                        <span
                          className='size-2 rounded-full'
                          style={{
                            background: PIE_COLORS[index % PIE_COLORS.length],
                          }}
                        />
                        <span className='truncate'>{item.name}</span>
                      </span>
                      <span className='shrink-0 font-semibold'>
                        {formatPercent(item.share)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </EnterprisePanel>

          <EnterprisePanel
            className='lg:col-span-2'
            title='错误率与延迟趋势'
            description='识别服务质量波动和异常时段'
            bodyClassName='h-64 p-3 sm:p-5'
          >
            {chartData.length === 0 ? (
              <div className='text-muted-foreground flex h-full items-center justify-center text-sm'>
                暂无服务质量趋势
              </div>
            ) : (
              <ResponsiveContainer width='100%' height='100%'>
                <BarChart
                  data={chartData}
                  margin={{ top: 12, right: 12, left: -20, bottom: 0 }}
                >
                  <CartesianGrid
                    strokeDasharray='4 6'
                    vertical={false}
                    stroke='var(--border)'
                    opacity={0.7}
                  />
                  <XAxis
                    dataKey='label'
                    axisLine={false}
                    tickLine={false}
                    tick={{ fontSize: 11, fill: 'var(--muted-foreground)' }}
                  />
                  <YAxis
                    axisLine={false}
                    tickLine={false}
                    tickFormatter={(value) =>
                      formatCompactNumber(Number(value))
                    }
                    tick={{ fontSize: 11, fill: 'var(--muted-foreground)' }}
                  />
                  <ChartTooltip
                    contentStyle={{
                      borderRadius: 12,
                      borderColor: 'var(--border)',
                      background: 'var(--popover)',
                    }}
                  />
                  <Bar
                    dataKey='errors'
                    name='错误请求'
                    fill='var(--chart-5)'
                    radius={[6, 6, 0, 0]}
                    maxBarSize={36}
                  />
                  <Bar
                    dataKey='average_latency_ms'
                    name='平均延迟 (ms)'
                    fill='var(--chart-3)'
                    radius={[6, 6, 0, 0]}
                    maxBarSize={36}
                  />
                </BarChart>
              </ResponsiveContainer>
            )}
          </EnterprisePanel>
        </div>

        <div className='space-y-4'>
          <EnterprisePanel
            title='成本异常检测'
            action={<Badge variant='outline'>{anomalyItems.length}</Badge>}
          >
            <div className='space-y-3'>
              {anomalyItems.map((item) => (
                <article
                  key={item.title}
                  className='border-border/70 bg-background/55 rounded-xl border p-3.5'
                >
                  <div className='flex gap-3'>
                    <span
                      className={cn(
                        'mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-xl',
                        item.tone === 'rose' && 'bg-rose-500/10 text-rose-600',
                        item.tone === 'amber' &&
                          'bg-amber-500/10 text-amber-600',
                        item.tone === 'blue' && 'bg-blue-500/10 text-blue-600'
                      )}
                    >
                      <AlertTriangle className='size-4' />
                    </span>
                    <div>
                      <p className='text-xs font-semibold'>{item.title}</p>
                      <p className='text-muted-foreground mt-1 text-[11px] leading-5'>
                        {item.detail}
                      </p>
                    </div>
                  </div>
                </article>
              ))}
            </div>
          </EnterprisePanel>

          <EnterprisePanel title='热点客户 / 用户' description='按成本贡献排序'>
            <BreakdownList items={usage.by_user} emptyText='暂无客户用量数据' />
          </EnterprisePanel>

          <EnterprisePanel title='优化建议'>
            <div className='space-y-3'>
              <div className='rounded-xl border border-blue-500/15 bg-blue-500/5 p-3.5'>
                <div className='flex items-center gap-2 text-xs font-semibold text-blue-700 dark:text-blue-300'>
                  <Sparkles className='size-4' />
                  提升缓存命中率
                </div>
                <p className='text-muted-foreground mt-1.5 text-[11px] leading-5'>
                  对高频 Prompt 建立稳定缓存键，减少重复上游调用和成本。
                </p>
              </div>
              <div className='rounded-xl border border-violet-500/15 bg-violet-500/5 p-3.5'>
                <div className='flex items-center gap-2 text-xs font-semibold text-violet-700 dark:text-violet-300'>
                  <ArrowDownRight className='size-4' />
                  优化模型组合
                </div>
                <p className='text-muted-foreground mt-1.5 text-[11px] leading-5'>
                  将低复杂度请求路由到更低成本模型，并保留高价值请求的质量兜底。
                </p>
              </div>
              <div className='rounded-xl border border-emerald-500/15 bg-emerald-500/5 p-3.5'>
                <div className='flex items-center gap-2 text-xs font-semibold text-emerald-700 dark:text-emerald-300'>
                  <ArrowUpRight className='size-4' />
                  建立部门预算线
                </div>
                <p className='text-muted-foreground mt-1.5 text-[11px] leading-5'>
                  对高增长成本中心设置预警线，提前发现预算透支风险。
                </p>
              </div>
            </div>
          </EnterprisePanel>

          <EnterprisePanel title='运行概况'>
            <div className='grid grid-cols-2 gap-3'>
              <div className='bg-muted/40 rounded-xl p-3'>
                <Clock3 className='size-4 text-violet-500' />
                <p className='mt-2 text-lg font-semibold'>
                  {formatNumber(metrics.average_latency_ms)} ms
                </p>
                <p className='text-muted-foreground text-[10px]'>平均延迟</p>
              </div>
              <div className='bg-muted/40 rounded-xl p-3'>
                <UsersRound className='size-4 text-blue-500' />
                <p className='mt-2 text-lg font-semibold'>
                  {formatNumber(usage.by_user.length)}
                </p>
                <p className='text-muted-foreground text-[10px]'>活跃主体</p>
              </div>
            </div>
          </EnterprisePanel>
        </div>
      </div>

      <EnterprisePanel
        title='用量日志'
        description={`共 ${formatNumber(usage.total_logs ?? 0)} 条匹配记录，当前展示 ${usage.recent_logs.length} 条`}
        action={
          <div className='flex flex-wrap items-center gap-2'>
            <Badge variant='outline'>
              第 {usage.page || page} / {totalLogPages} 页
            </Badge>
            <Button
              size='sm'
              variant='outline'
              onClick={() => setPage((value) => Math.max(1, value - 1))}
              disabled={page <= 1 || usageQuery.isFetching}
            >
              上一页
            </Button>
            <Button
              size='sm'
              variant='outline'
              onClick={() =>
                setPage((value) => Math.min(totalLogPages, value + 1))
              }
              disabled={page >= totalLogPages || usageQuery.isFetching}
            >
              下一页
            </Button>
          </div>
        }
        bodyClassName='p-0'
      >
        <UsageLogTable logs={usage.recent_logs} />
      </EnterprisePanel>

      {props.classicContent && (
        <EnterprisePanel
          title='完整调用明细'
          description='保留原系统的搜索、筛选、分页、详情与缓存诊断能力'
          bodyClassName='min-h-[560px] p-0'
        >
          {props.classicContent}
        </EnterprisePanel>
      )}
    </div>
  )
}
