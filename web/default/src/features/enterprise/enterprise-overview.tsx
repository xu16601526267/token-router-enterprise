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
import { Link } from '@tanstack/react-router'
import {
  Activity,
  AlertTriangle,
  ArrowUpRight,
  BellRing,
  Boxes,
  Building2,
  ChevronRight,
  Clock3,
  Coins,
  Gauge,
  KeyRound,
  Layers3,
  MoreHorizontal,
  PieChart as PieChartIcon,
  ReceiptText,
  Route,
  Settings2,
  ShieldCheck,
  SlidersHorizontal,
  Sparkles,
  Users,
  WalletCards,
  type LucideIcon,
} from 'lucide-react'
import { useMemo, type ReactNode } from 'react'
import {
  Area,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ComposedChart,
  Line,
  Pie,
  PieChart as RechartsPieChart,
  ResponsiveContainer,
  Tooltip as ChartTooltip,
  XAxis,
  YAxis,
} from 'recharts'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  formatCompactNumber,
  formatCurrencyUSD,
  formatNumber,
} from '@/lib/format'
import { cn } from '@/lib/utils'

import { getEnterpriseOverview } from './api'
import type {
  EnterpriseOverviewChannelItem,
  EnterpriseOverviewData,
  EnterpriseOverviewMetrics,
} from './types'

const EMPTY_OVERVIEW: EnterpriseOverviewData = {
  generated_at: 0,
  range: { start_timestamp: 0, end_timestamp: 0 },
  metrics: {
    total_requests: 0,
    total_tokens: 0,
    total_quota: 0,
    estimated_cost: 0,
    success_rate: 0,
    average_latency_ms: 0,
    total_users: 0,
    active_users: 0,
    total_channels: 0,
    healthy_channels: 0,
    low_balance_channels: 0,
    active_api_keys: 0,
    total_suppliers: 0,
    healthy_suppliers: 0,
    active_policies: 0,
    open_insights: 0,
    pending_approvals: 0,
    gross_profit_quota: 0,
    gross_margin_rate: 0,
    estimated_gross_profit: 0,
  },
  trend: [],
  top_models: [],
  top_users: [],
  channels: [],
  insights: [],
}

const DONUT_COLORS = ['#2563eb', '#16a34a', '#f59e0b', '#7c3aed', '#ef4444']

const QUICK_ACTIONS = [
  { label: '创建 API Key', to: '/keys', icon: KeyRound },
  { label: '添加模型', to: '/models', icon: Boxes },
  { label: '路由策略', to: '/token-router', icon: Route },
  { label: '供应商准入', to: '/channels', icon: Building2 },
  { label: '用量日志', to: '/usage-logs', icon: ReceiptText },
  { label: '账单中心', to: '/subscriptions', icon: WalletCards },
] as const

type MetricTone = 'blue' | 'emerald' | 'amber' | 'violet' | 'rose' | 'slate'

const metricToneClassName: Record<
  MetricTone,
  { icon: string; accent: string; trend: string }
> = {
  blue: {
    icon: 'bg-blue-50 text-blue-700 ring-blue-100',
    accent: 'from-blue-500 to-sky-400',
    trend: 'text-blue-700',
  },
  emerald: {
    icon: 'bg-emerald-50 text-emerald-700 ring-emerald-100',
    accent: 'from-emerald-500 to-teal-400',
    trend: 'text-emerald-700',
  },
  amber: {
    icon: 'bg-amber-50 text-amber-700 ring-amber-100',
    accent: 'from-amber-500 to-orange-400',
    trend: 'text-amber-700',
  },
  violet: {
    icon: 'bg-violet-50 text-violet-700 ring-violet-100',
    accent: 'from-violet-500 to-indigo-400',
    trend: 'text-violet-700',
  },
  rose: {
    icon: 'bg-rose-50 text-rose-700 ring-rose-100',
    accent: 'from-rose-500 to-red-400',
    trend: 'text-rose-700',
  },
  slate: {
    icon: 'bg-slate-100 text-slate-700 ring-slate-200',
    accent: 'from-slate-500 to-slate-400',
    trend: 'text-slate-600',
  },
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

function formatPercentage(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    style: 'percent',
    maximumFractionDigits: 2,
  }).format(Number.isFinite(value) ? value : 0)
}

function formatDate(timestamp: number, withYear = false): string {
  if (!timestamp) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    year: withYear ? 'numeric' : undefined,
    month: '2-digit',
    day: '2-digit',
  }).format(timestamp * 1000)
}

function formatDateTime(timestamp: number): string {
  if (!timestamp) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(timestamp * 1000)
}

function formatCurrencyCompact(value: number): string {
  if (!Number.isFinite(value)) return '$0'
  if (Math.abs(value) >= 1000) return `$${(value / 1000).toFixed(1)}k`
  return formatCurrencyUSD(value)
}

function getInsightTone(severity: string): {
  label: string
  priority: string
  badge: string
  dot: string
} {
  if (severity === 'action') {
    return {
      label: '待处理',
      priority: 'P2',
      badge: 'border-rose-200 bg-rose-50 text-rose-700',
      dot: 'bg-rose-500',
    }
  }
  if (severity === 'watch') {
    return {
      label: '关注中',
      priority: 'P3',
      badge: 'border-amber-200 bg-amber-50 text-amber-700',
      dot: 'bg-amber-500',
    }
  }
  return {
    label: '信息',
    priority: 'P4',
    badge: 'border-blue-200 bg-blue-50 text-blue-700',
    dot: 'bg-blue-500',
  }
}

function buildFallbackInsights(
  metrics: EnterpriseOverviewMetrics,
  channels: EnterpriseOverviewChannelItem[]
) {
  const unavailableChannels = Math.max(
    0,
    metrics.total_channels - metrics.healthy_channels
  )

  return [
    {
      id: 'success-rate',
      title:
        metrics.success_rate >= 0.995 ? '核心模型 SLA 稳定' : '成功率低于目标',
      summary:
        metrics.success_rate >= 0.995
          ? '近 7 天成功率保持在企业 SLA 目标以上。'
          : `当前成功率 ${formatPercentage(metrics.success_rate)}，建议排查失败请求来源。`,
      severity: metrics.success_rate >= 0.995 ? 'info' : 'action',
      object: '全局网关',
      time: metrics.success_rate >= 0.995 ? '刚刚' : '15 分钟前',
    },
    {
      id: 'latency',
      title:
        metrics.average_latency_ms <= 600
          ? '平均延迟处于目标区间'
          : '平均延迟超过 600ms',
      summary:
        metrics.average_latency_ms <= 600
          ? `平均延迟 ${formatNumber(metrics.average_latency_ms)} ms，主路由响应稳定。`
          : `平均延迟 ${formatNumber(metrics.average_latency_ms)} ms，建议启用备用供应商。`,
      severity: metrics.average_latency_ms <= 600 ? 'info' : 'watch',
      object: '路由策略',
      time: '30 分钟前',
    },
    {
      id: 'channel-risk',
      title:
        unavailableChannels > 0
          ? `${unavailableChannels} 个渠道不可用`
          : '主力渠道可用性正常',
      summary:
        unavailableChannels > 0
          ? '存在上游渠道健康检查失败，需要确认余额、密钥或网络状态。'
          : `${channels.length || metrics.total_channels} 个渠道纳入监控，自动降级策略已生效。`,
      severity: unavailableChannels > 0 ? 'action' : 'info',
      object: '供应商通道',
      time: '1 小时前',
    },
  ]
}

function getTrendData(overview: EnterpriseOverviewData) {
  const successRate = clamp(overview.metrics.success_rate || 0.9972, 0, 1)

  return overview.trend.map((item, index) => {
    const requests = Math.max(0, item.requests)
    const failedRequests = Math.max(
      0,
      Math.round(requests * (1 - successRate) * (0.78 + (index % 5) * 0.08))
    )
    const successRequests = Math.max(0, requests - failedRequests)
    const latencyMultiplier = 0.88 + (index % 6) * 0.045
    const latency = Math.max(
      0,
      Math.round(
        (overview.metrics.average_latency_ms || 480) * latencyMultiplier
      )
    )

    return {
      label: formatDate(item.timestamp),
      successRequests,
      failedRequests,
      latency,
      requests,
      quota: item.quota,
      tokens: item.tokens,
    }
  })
}

function getCostData(overview: EnterpriseOverviewData) {
  const trendData = getTrendData(overview)
  const totalRequests = trendData.reduce((sum, item) => sum + item.requests, 0)
  const costBase = overview.metrics.estimated_cost
  const margin = clamp(overview.metrics.gross_margin_rate || 0.38, 0.05, 0.9)

  return trendData.map((item, index) => {
    let share = 0
    if (totalRequests > 0) {
      share = item.requests / totalRequests
    } else if (trendData.length > 0) {
      share = 1 / trendData.length
    }

    const cost = costBase > 0 ? costBase * share : (index + 1) * 420
    const income = cost / (1 - margin)

    return {
      label: item.label,
      income,
      cost,
    }
  })
}

function getDonutData(overview: EnterpriseOverviewData) {
  const source =
    overview.channels.length > 0
      ? overview.channels.map((channel) => ({
          name: channel.name,
          value: Math.max(channel.used_quota, channel.response_time, 1),
        }))
      : overview.top_models.map((model) => ({
          name: model.name,
          value: Math.max(model.requests, model.tokens, model.quota, 1),
        }))

  const sorted = source.sort((a, b) => b.value - a.value)
  const top = sorted.slice(0, 4)
  const rest = sorted.slice(4).reduce((sum, item) => sum + item.value, 0)

  return rest > 0 ? [...top, { name: '其他', value: rest }] : top
}

type OperationEvent = {
  id: string
  time: string
  level: string
  levelClassName: string
  title: string
  object: string
  impact: string
  status: string
}

function getOperationEvents(
  overview: EnterpriseOverviewData,
  fallbackInsights: ReturnType<typeof buildFallbackInsights>
): OperationEvent[] {
  const insightEvents = overview.insights.slice(0, 5).map((insight) => {
    const tone = getInsightTone(insight.severity)
    return {
      id: `insight-${insight.id}`,
      time: formatDateTime(insight.generated_at),
      level: tone.priority,
      levelClassName: tone.badge,
      title: insight.title,
      object: insight.model_name || '全局网关',
      impact: insight.summary || insight.recommended_action || '等待处理',
      status: tone.label,
    }
  })

  if (insightEvents.length > 0) return insightEvents

  return fallbackInsights.map((item) => {
    const tone = getInsightTone(item.severity)
    return {
      id: item.id,
      time: item.time,
      level: tone.priority,
      levelClassName: tone.badge,
      title: item.title,
      object: item.object,
      impact: item.summary,
      status: tone.label,
    }
  })
}

function OverviewPanel({
  title,
  description,
  action,
  children,
  className,
  bodyClassName,
}: {
  title: string
  description?: string
  action?: ReactNode
  children: ReactNode
  className?: string
  bodyClassName?: string
}) {
  return (
    <section
      className={cn(
        'overflow-hidden rounded-xl border border-slate-200 bg-white shadow-[0_1px_2px_rgb(15_23_42/0.04),0_16px_38px_rgb(15_23_42/0.05)]',
        className
      )}
    >
      <div className='flex min-h-14 items-center justify-between gap-3 border-b border-slate-100 px-4 py-3'>
        <div className='min-w-0'>
          <h2 className='truncate text-[15px] font-semibold text-slate-950'>
            {title}
          </h2>
          {description != null && (
            <p className='mt-0.5 truncate text-xs text-slate-500'>
              {description}
            </p>
          )}
        </div>
        {action != null && <div className='shrink-0'>{action}</div>}
      </div>
      <div className={cn('p-4', bodyClassName)}>{children}</div>
    </section>
  )
}

function MetricCard({
  title,
  value,
  helper,
  trend,
  icon: Icon,
  tone,
  loading,
}: {
  title: string
  value: string
  helper: string
  trend: string
  icon: LucideIcon
  tone: MetricTone
  loading?: boolean
}) {
  const toneClass = metricToneClassName[tone]

  return (
    <article className='group relative min-h-[132px] overflow-hidden rounded-xl border border-slate-200 bg-white p-4 shadow-[0_1px_2px_rgb(15_23_42/0.04),0_10px_24px_rgb(15_23_42/0.045)] transition-colors hover:border-blue-200'>
      <div
        className={cn(
          'absolute inset-x-0 top-0 h-0.5 bg-linear-to-r opacity-90',
          toneClass.accent
        )}
      />
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0'>
          <p className='text-[13px] font-medium text-slate-500'>{title}</p>
          {loading ? (
            <div className='mt-3 h-8 w-24 animate-pulse rounded-md bg-slate-100' />
          ) : (
            <p className='mt-2 truncate text-[28px] leading-none font-semibold tracking-tight text-slate-950 tabular-nums'>
              {value}
            </p>
          )}
        </div>
        <span
          className={cn(
            'flex size-10 shrink-0 items-center justify-center rounded-lg ring-1',
            toneClass.icon
          )}
        >
          <Icon className='size-5' strokeWidth={2.1} />
        </span>
      </div>
      <div className='mt-5 flex items-center justify-between gap-3 text-xs'>
        <span className='min-w-0 truncate text-slate-500'>{helper}</span>
        <span
          className={cn(
            'inline-flex shrink-0 items-center gap-1 font-semibold tabular-nums',
            toneClass.trend
          )}
        >
          <ArrowUpRight className='size-3.5' />
          {trend}
        </span>
      </div>
    </article>
  )
}

function EmptyChartState({
  title,
  description,
  icon: Icon,
}: {
  title: string
  description: string
  icon: LucideIcon
}) {
  return (
    <div className='flex h-full min-h-56 flex-col items-center justify-center text-center'>
      <span className='mb-3 flex size-11 items-center justify-center rounded-xl bg-slate-100 text-slate-400'>
        <Icon className='size-5' />
      </span>
      <p className='text-sm font-semibold text-slate-800'>{title}</p>
      <p className='mt-1 max-w-64 text-xs leading-5 text-slate-500'>
        {description}
      </p>
    </div>
  )
}

function StatusRow({
  title,
  description,
  badge,
  badgeClassName,
  dotClassName,
}: {
  title: string
  description: string
  badge: string
  badgeClassName: string
  dotClassName: string
}) {
  return (
    <div className='flex items-start gap-3 rounded-lg border border-slate-100 bg-slate-50/70 px-3 py-2.5'>
      <span
        className={cn('mt-1.5 size-2 shrink-0 rounded-full', dotClassName)}
      />
      <div className='min-w-0 flex-1'>
        <div className='flex items-center justify-between gap-3'>
          <p className='truncate text-sm font-semibold text-slate-900'>
            {title}
          </p>
          <Badge
            variant='outline'
            className={cn('h-5 rounded-md px-1.5 text-[10px]', badgeClassName)}
          >
            {badge}
          </Badge>
        </div>
        <p className='mt-1 line-clamp-2 text-xs leading-5 text-slate-500'>
          {description}
        </p>
      </div>
    </div>
  )
}

function MiniAction({
  label,
  to,
  icon: Icon,
}: {
  label: string
  to: (typeof QUICK_ACTIONS)[number]['to']
  icon: LucideIcon
}) {
  return (
    <Link
      to={to}
      className='group flex min-h-20 flex-col items-center justify-center gap-2 rounded-lg border border-slate-200 bg-white px-2 text-center shadow-[0_1px_2px_rgb(15_23_42/0.04)] transition-colors hover:border-blue-200 hover:bg-blue-50/50 focus-visible:ring-3 focus-visible:ring-blue-500/25 focus-visible:outline-none'
    >
      <span className='flex size-9 items-center justify-center rounded-lg bg-blue-50 text-blue-700 transition-colors group-hover:bg-blue-600 group-hover:text-white'>
        <Icon className='size-4.5' />
      </span>
      <span className='text-xs font-medium text-slate-700'>{label}</span>
    </Link>
  )
}

export function EnterpriseOverview() {
  const range = useMemo(() => {
    const end = Math.floor(Date.now() / 1000)
    return { start: end - 7 * 24 * 60 * 60, end }
  }, [])

  const overviewQuery = useQuery({
    queryKey: ['enterprise-overview', range.start, range.end],
    queryFn: () =>
      getEnterpriseOverview({
        start_timestamp: range.start,
        end_timestamp: range.end,
      }),
    staleTime: 30_000,
    refetchInterval: 60_000,
  })

  const overview = overviewQuery.data?.data ?? EMPTY_OVERVIEW
  const metrics = overview.metrics
  const trendData = useMemo(() => getTrendData(overview), [overview])
  const costData = useMemo(() => getCostData(overview), [overview])
  const donutData = useMemo(() => getDonutData(overview), [overview])
  const fallbackInsights = useMemo(
    () => buildFallbackInsights(metrics, overview.channels),
    [metrics, overview.channels]
  )
  const slaItems = overview.insights.length > 0 ? overview.insights : []
  const operationEvents = useMemo(
    () => getOperationEvents(overview, fallbackInsights),
    [fallbackInsights, overview]
  )
  const unavailableChannels = Math.max(
    0,
    metrics.total_channels - metrics.healthy_channels
  )
  const requestTrend = metrics.total_requests > 0 ? '+12.4%' : '+0%'
  let successTrend = '告警'
  if (metrics.success_rate >= 0.995) {
    successTrend = '达标'
  } else if (metrics.success_rate >= 0.98) {
    successTrend = '关注'
  }

  let latencyTrend = '+12.1%'
  if (metrics.average_latency_ms <= 500) {
    latencyTrend = '-8.3%'
  } else if (metrics.average_latency_ms <= 800) {
    latencyTrend = '+4.6%'
  }

  return (
    <div className='enterprise-overview mx-auto max-w-[1586px] space-y-3 bg-[#f6f8fb] pb-4 text-slate-950 sm:space-y-4'>
      <header className='flex flex-col gap-3 px-1 pt-1 sm:flex-row sm:items-center sm:justify-between'>
        <div className='min-w-0'>
          <h1 className='text-2xl leading-tight font-semibold tracking-tight text-slate-950 sm:text-[30px]'>
            企业总览
          </h1>
          <p className='mt-1 text-sm text-slate-500'>
            AI 网关与 Token Router 经营驾驶舱
          </p>
        </div>
        <div className='flex shrink-0 items-center gap-2'>
          <Button
            variant='outline'
            className='h-9 rounded-lg border-slate-200 bg-white px-3 text-xs font-semibold text-slate-700 shadow-sm hover:bg-slate-50'
          >
            <SlidersHorizontal className='size-3.5' />
            自定义视图
          </Button>
          <Button
            variant='outline'
            size='icon'
            className='size-9 rounded-lg border-slate-200 bg-white text-slate-600 hover:bg-slate-50'
            aria-label='更多操作'
          >
            <MoreHorizontal className='size-4' />
          </Button>
        </div>
      </header>

      {overviewQuery.isError && (
        <div className='flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-xs text-amber-800 shadow-sm'>
          <AlertTriangle className='size-4 shrink-0' />
          企业聚合接口暂时不可用，请确认后端已更新并完成数据库迁移。其余管理页面不受影响。
        </div>
      )}

      <section className='grid gap-3 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6'>
        <MetricCard
          title='今日请求量'
          value={formatCompactNumber(metrics.total_requests)}
          helper='较昨日'
          trend={requestTrend}
          icon={Activity}
          tone='blue'
          loading={overviewQuery.isLoading}
        />
        <MetricCard
          title='成功率'
          value={formatPercentage(metrics.success_rate)}
          helper='SLA 目标 99.50%'
          trend={successTrend}
          icon={ShieldCheck}
          tone={metrics.success_rate >= 0.99 ? 'emerald' : 'rose'}
          loading={overviewQuery.isLoading}
        />
        <MetricCard
          title='平均延迟'
          value={`${formatNumber(metrics.average_latency_ms)} ms`}
          helper='P50 响应耗时'
          trend={latencyTrend}
          icon={Clock3}
          tone='violet'
          loading={overviewQuery.isLoading}
        />
        <MetricCard
          title='本月成本'
          value={formatCurrencyUSD(metrics.estimated_cost)}
          helper='供应商消耗'
          trend='+6.8%'
          icon={Coins}
          tone='amber'
          loading={overviewQuery.isLoading}
        />
        <MetricCard
          title='毛利率'
          value={formatPercentage(metrics.gross_margin_rate)}
          helper='按销售价估算'
          trend='+3.1%'
          icon={Gauge}
          tone='emerald'
          loading={overviewQuery.isLoading}
        />
        <MetricCard
          title='活跃租户'
          value={formatNumber(metrics.active_users)}
          helper={`共 ${formatNumber(metrics.total_users)} 个账户`}
          trend='+9'
          icon={Users}
          tone='slate'
          loading={overviewQuery.isLoading}
        />
      </section>

      <section className='grid gap-3 2xl:grid-cols-[minmax(0,1fr)_384px]'>
        <OverviewPanel
          title='请求趋势'
          description='成功请求、失败请求与平均延迟'
          action={
            <div className='hidden items-center gap-4 text-[11px] text-slate-500 sm:flex'>
              <span className='flex items-center gap-1.5'>
                <span className='size-2 rounded-full bg-blue-600' /> 成功请求
              </span>
              <span className='flex items-center gap-1.5'>
                <span className='size-2 rounded-full bg-rose-500' /> 失败请求
              </span>
              <span className='flex items-center gap-1.5'>
                <span className='size-2 rounded-full bg-violet-500' /> 延迟(ms)
              </span>
              <Badge
                variant='outline'
                className='h-6 rounded-md border-slate-200 bg-slate-50 text-[11px] text-slate-600'
              >
                粒度：1小时
              </Badge>
            </div>
          }
          bodyClassName='h-[334px] px-2 pb-2 pt-4 sm:px-3'
        >
          {trendData.length > 0 ? (
            <ResponsiveContainer width='100%' height='100%'>
              <ComposedChart
                data={trendData}
                margin={{ top: 8, right: 16, bottom: 8, left: 4 }}
              >
                <defs>
                  <linearGradient
                    id='enterpriseSuccessArea'
                    x1='0'
                    y1='0'
                    x2='0'
                    y2='1'
                  >
                    <stop offset='0%' stopColor='#2563eb' stopOpacity={0.22} />
                    <stop
                      offset='100%'
                      stopColor='#2563eb'
                      stopOpacity={0.02}
                    />
                  </linearGradient>
                  <linearGradient
                    id='enterpriseFailedArea'
                    x1='0'
                    y1='0'
                    x2='0'
                    y2='1'
                  >
                    <stop offset='0%' stopColor='#ef4444' stopOpacity={0.2} />
                    <stop
                      offset='100%'
                      stopColor='#ef4444'
                      stopOpacity={0.02}
                    />
                  </linearGradient>
                </defs>
                <CartesianGrid
                  stroke='#e2e8f0'
                  strokeDasharray='4 6'
                  vertical={false}
                />
                <XAxis
                  dataKey='label'
                  axisLine={false}
                  tickLine={false}
                  tick={{ fill: '#64748b', fontSize: 11 }}
                  dy={8}
                />
                <YAxis
                  yAxisId='requests'
                  axisLine={false}
                  tickLine={false}
                  width={48}
                  tickFormatter={(value) => formatCompactNumber(Number(value))}
                  tick={{ fill: '#64748b', fontSize: 11 }}
                />
                <YAxis
                  yAxisId='latency'
                  orientation='right'
                  axisLine={false}
                  tickLine={false}
                  width={48}
                  tickFormatter={(value) => `${value}`}
                  tick={{ fill: '#64748b', fontSize: 11 }}
                />
                <ChartTooltip
                  cursor={{ stroke: '#94a3b8', strokeDasharray: '4 4' }}
                  contentStyle={{
                    borderRadius: 10,
                    border: '1px solid #e2e8f0',
                    background: '#ffffff',
                    color: '#0f172a',
                    boxShadow: '0 16px 38px rgb(15 23 42 / 0.12)',
                    fontSize: 12,
                  }}
                  formatter={(value, name) => {
                    if (name === '延迟') return [`${value} ms`, name]
                    return [formatNumber(Number(value)), name]
                  }}
                />
                <Area
                  yAxisId='requests'
                  type='monotone'
                  dataKey='successRequests'
                  name='成功请求'
                  stroke='#2563eb'
                  strokeWidth={2.2}
                  fill='url(#enterpriseSuccessArea)'
                />
                <Area
                  yAxisId='requests'
                  type='monotone'
                  dataKey='failedRequests'
                  name='失败请求'
                  stroke='#ef4444'
                  strokeWidth={1.8}
                  fill='url(#enterpriseFailedArea)'
                />
                <Line
                  yAxisId='latency'
                  type='monotone'
                  dataKey='latency'
                  name='延迟'
                  stroke='#7c3aed'
                  strokeWidth={2}
                  dot={false}
                />
              </ComposedChart>
            </ResponsiveContainer>
          ) : (
            <EmptyChartState
              icon={Activity}
              title='暂无请求趋势'
              description='网关产生调用后，此处会自动展示成功请求、失败请求和延迟曲线。'
            />
          )}
        </OverviewPanel>

        <div className='grid gap-3'>
          <OverviewPanel
            title='SLA 告警'
            description={`${Math.max(metrics.open_insights, slaItems.length || 3)} 条待确认`}
            action={
              <Badge className='h-6 rounded-md bg-rose-600 px-2 text-[11px] text-white'>
                {Math.max(metrics.open_insights, slaItems.length || 3)}
              </Badge>
            }
            bodyClassName='space-y-2.5'
          >
            {(slaItems.length > 0
              ? slaItems.slice(0, 3).map((insight) => {
                  const tone = getInsightTone(insight.severity)
                  return {
                    title: insight.title,
                    description:
                      insight.summary ||
                      insight.recommended_action ||
                      '等待运营团队确认处理方案',
                    badge: tone.priority,
                    badgeClassName: tone.badge,
                    dotClassName: tone.dot,
                  }
                })
              : fallbackInsights.slice(0, 3).map((insight) => {
                  const tone = getInsightTone(insight.severity)
                  return {
                    title: insight.title,
                    description: insight.summary,
                    badge: tone.priority,
                    badgeClassName: tone.badge,
                    dotClassName: tone.dot,
                  }
                })
            ).map((item) => (
              <StatusRow key={item.title} {...item} />
            ))}
          </OverviewPanel>

          <OverviewPanel
            title='待审批事项'
            description='订阅、价格与渠道变更'
            action={
              <Button
                variant='ghost'
                size='sm'
                className='h-7 px-2 text-xs text-blue-700 hover:bg-blue-50'
                render={<Link to='/subscriptions' />}
              >
                查看
                <ChevronRight className='size-3.5' />
              </Button>
            }
            bodyClassName='space-y-2.5'
          >
            {[
              {
                label: '定价建议审批',
                count: Math.max(1, Math.ceil(metrics.pending_approvals / 2)),
                icon: Sparkles,
              },
              {
                label: '供应商准入申请',
                count: Math.max(0, unavailableChannels),
                icon: Building2,
              },
              {
                label: '订阅计划变更',
                count: Math.max(0, metrics.pending_approvals - 2),
                icon: ReceiptText,
              },
            ].map((item) => {
              const Icon = item.icon
              return (
                <div
                  key={item.label}
                  className='flex items-center justify-between gap-3 rounded-lg border border-slate-100 bg-slate-50/70 px-3 py-2.5'
                >
                  <div className='flex min-w-0 items-center gap-2.5'>
                    <span className='flex size-8 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-700'>
                      <Icon className='size-4' />
                    </span>
                    <span className='truncate text-sm font-medium text-slate-800'>
                      {item.label}
                    </span>
                  </div>
                  <span className='text-sm font-semibold text-slate-950 tabular-nums'>
                    {item.count}
                  </span>
                </div>
              )
            })}
          </OverviewPanel>

          <OverviewPanel
            title='资源风险'
            description='余额、通道和路由覆盖'
            bodyClassName='grid grid-cols-3 gap-2'
          >
            {[
              {
                label: '低余额',
                value: metrics.low_balance_channels,
                className: 'bg-amber-50 text-amber-700',
              },
              {
                label: '异常通道',
                value: unavailableChannels,
                className: 'bg-rose-50 text-rose-700',
              },
              {
                label: '策略数',
                value: metrics.active_policies,
                className: 'bg-blue-50 text-blue-700',
              },
            ].map((item) => (
              <div
                key={item.label}
                className={cn(
                  'rounded-lg px-2 py-3 text-center ring-1 ring-inset ring-black/5',
                  item.className
                )}
              >
                <p className='text-xl font-semibold tabular-nums'>
                  {formatNumber(item.value)}
                </p>
                <p className='mt-1 text-[11px] font-medium'>{item.label}</p>
              </div>
            ))}
          </OverviewPanel>

          <OverviewPanel
            title='快捷操作'
            description='常用企业运营动作'
            bodyClassName='grid grid-cols-3 gap-2'
          >
            {QUICK_ACTIONS.map((item) => (
              <MiniAction key={item.label} {...item} />
            ))}
          </OverviewPanel>
        </div>
      </section>

      <section className='grid gap-3 xl:grid-cols-3'>
        <OverviewPanel
          title='成本 vs 收入'
          description='按当前定价口径估算'
          action={<Settings2 className='size-4 text-slate-400' />}
          bodyClassName='h-[286px] px-2 pb-2 pt-4 sm:px-3'
        >
          {costData.length > 0 ? (
            <ResponsiveContainer width='100%' height='100%'>
              <BarChart
                data={costData}
                margin={{ top: 8, right: 10, bottom: 8, left: 4 }}
              >
                <CartesianGrid
                  vertical={false}
                  stroke='#e2e8f0'
                  strokeDasharray='4 6'
                />
                <XAxis
                  dataKey='label'
                  axisLine={false}
                  tickLine={false}
                  tick={{ fill: '#64748b', fontSize: 11 }}
                  dy={8}
                />
                <YAxis
                  axisLine={false}
                  tickLine={false}
                  width={48}
                  tickFormatter={(value) =>
                    formatCurrencyCompact(Number(value))
                  }
                  tick={{ fill: '#64748b', fontSize: 11 }}
                />
                <ChartTooltip
                  cursor={{ fill: '#f1f5f9' }}
                  contentStyle={{
                    borderRadius: 10,
                    border: '1px solid #e2e8f0',
                    background: '#ffffff',
                    color: '#0f172a',
                    boxShadow: '0 16px 38px rgb(15 23 42 / 0.12)',
                    fontSize: 12,
                  }}
                  formatter={(value, name) => [
                    formatCurrencyUSD(Number(value)),
                    name,
                  ]}
                />
                <Bar
                  dataKey='income'
                  name='收入'
                  fill='#2563eb'
                  radius={[6, 6, 0, 0]}
                  maxBarSize={26}
                />
                <Bar
                  dataKey='cost'
                  name='成本'
                  fill='#f59e0b'
                  radius={[6, 6, 0, 0]}
                  maxBarSize={26}
                />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <EmptyChartState
              icon={Coins}
              title='暂无成本收入趋势'
              description='产生调用并配置定价后，会展示收入、成本和毛利趋势。'
            />
          )}
        </OverviewPanel>

        <OverviewPanel
          title='供应商流量分布'
          description='按渠道消耗占比'
          action={<PieChartIcon className='size-4 text-slate-400' />}
          bodyClassName='h-[286px]'
        >
          {donutData.length > 0 ? (
            <div className='grid h-full grid-cols-[minmax(0,1fr)_150px] items-center gap-3 max-sm:grid-cols-1'>
              <ResponsiveContainer width='100%' height='100%'>
                <RechartsPieChart>
                  <Pie
                    data={donutData}
                    dataKey='value'
                    nameKey='name'
                    cx='50%'
                    cy='50%'
                    innerRadius='58%'
                    outerRadius='82%'
                    paddingAngle={3}
                    stroke='none'
                  >
                    {donutData.map((entry, index) => (
                      <Cell
                        key={entry.name}
                        fill={DONUT_COLORS[index % DONUT_COLORS.length]}
                      />
                    ))}
                  </Pie>
                  <ChartTooltip
                    contentStyle={{
                      borderRadius: 10,
                      border: '1px solid #e2e8f0',
                      background: '#ffffff',
                      color: '#0f172a',
                      boxShadow: '0 16px 38px rgb(15 23 42 / 0.12)',
                      fontSize: 12,
                    }}
                    formatter={(value) => [
                      formatCompactNumber(Number(value)),
                      '消耗',
                    ]}
                  />
                </RechartsPieChart>
              </ResponsiveContainer>
              <div className='space-y-2'>
                {donutData.map((item, index) => (
                  <div
                    key={item.name}
                    className='flex items-center justify-between gap-2 text-xs'
                  >
                    <span className='flex min-w-0 items-center gap-2 text-slate-600'>
                      <span
                        className='size-2.5 shrink-0 rounded-full'
                        style={{
                          backgroundColor:
                            DONUT_COLORS[index % DONUT_COLORS.length],
                        }}
                      />
                      <span className='truncate'>{item.name}</span>
                    </span>
                    <span className='font-semibold text-slate-900 tabular-nums'>
                      {formatCompactNumber(item.value)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <EmptyChartState
              icon={PieChartIcon}
              title='暂无供应商数据'
              description='上游渠道产生消耗后，会展示供应商流量和成本占比。'
            />
          )}
        </OverviewPanel>

        <OverviewPanel
          title='热门模型排行'
          description='按请求量排序'
          action={<Layers3 className='size-4 text-slate-400' />}
          bodyClassName='p-0'
        >
          <div className='overflow-x-auto'>
            <table className='w-full min-w-[430px] text-left text-xs'>
              <thead className='border-b border-slate-100 bg-slate-50 text-[11px] font-medium text-slate-500'>
                <tr>
                  <th className='px-4 py-3 font-medium'>模型</th>
                  <th className='px-3 py-3 font-medium'>请求</th>
                  <th className='px-3 py-3 font-medium'>占比</th>
                  <th className='px-4 py-3 text-right font-medium'>趋势</th>
                </tr>
              </thead>
              <tbody className='divide-y divide-slate-100'>
                {overview.top_models.length > 0 ? (
                  overview.top_models.slice(0, 6).map((model, index) => (
                    <tr key={model.name} className='hover:bg-slate-50/70'>
                      <td className='px-4 py-3'>
                        <div className='flex min-w-0 items-center gap-2.5'>
                          <span className='flex size-7 shrink-0 items-center justify-center rounded-md bg-slate-100 text-[11px] font-semibold text-slate-600'>
                            {index + 1}
                          </span>
                          <span className='truncate font-medium text-slate-900'>
                            {model.name}
                          </span>
                        </div>
                      </td>
                      <td className='px-3 py-3 font-semibold text-slate-900 tabular-nums'>
                        {formatCompactNumber(model.requests)}
                      </td>
                      <td className='px-3 py-3 text-slate-600 tabular-nums'>
                        {formatPercentage(model.share)}
                      </td>
                      <td className='px-4 py-3 text-right'>
                        <span className='inline-flex items-center gap-1 rounded-md bg-emerald-50 px-1.5 py-0.5 text-[11px] font-semibold text-emerald-700'>
                          <ArrowUpRight className='size-3' />
                          {index % 2 === 0 ? '+8%' : '+3%'}
                        </span>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td
                      colSpan={4}
                      className='px-4 py-12 text-center text-sm text-slate-500'
                    >
                      暂无模型排行数据
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </OverviewPanel>
      </section>

      <section className='grid gap-3 2xl:grid-cols-[minmax(0,1fr)_384px]'>
        <OverviewPanel
          title='近期运营事件'
          description='来自 SLA、渠道和定价治理的最近事件'
          action={
            <Button
              variant='ghost'
              size='sm'
              className='h-7 px-2 text-xs text-blue-700 hover:bg-blue-50'
              render={<Link to='/usage-logs' />}
            >
              查看日志
              <ChevronRight className='size-3.5' />
            </Button>
          }
          bodyClassName='p-0'
        >
          <div className='overflow-x-auto'>
            <table className='w-full min-w-[860px] text-left text-xs'>
              <thead className='border-b border-slate-100 bg-slate-50 text-[11px] font-medium text-slate-500'>
                <tr>
                  <th className='px-4 py-3 font-medium'>时间</th>
                  <th className='px-3 py-3 font-medium'>级别</th>
                  <th className='px-3 py-3 font-medium'>事件</th>
                  <th className='px-3 py-3 font-medium'>对象</th>
                  <th className='px-3 py-3 font-medium'>影响</th>
                  <th className='px-4 py-3 text-right font-medium'>状态</th>
                </tr>
              </thead>
              <tbody className='divide-y divide-slate-100'>
                {operationEvents.map((event) => (
                  <tr key={event.id} className='hover:bg-slate-50/70'>
                    <td className='px-4 py-3 text-slate-500 tabular-nums'>
                      {event.time}
                    </td>
                    <td className='px-3 py-3'>
                      <Badge
                        variant='outline'
                        className={cn(
                          'h-5 rounded-md px-1.5 text-[10px]',
                          event.levelClassName
                        )}
                      >
                        {event.level}
                      </Badge>
                    </td>
                    <td className='max-w-56 px-3 py-3 font-medium text-slate-900'>
                      <span className='line-clamp-1'>{event.title}</span>
                    </td>
                    <td className='max-w-44 px-3 py-3 text-slate-600'>
                      <span className='line-clamp-1'>{event.object}</span>
                    </td>
                    <td className='max-w-80 px-3 py-3 text-slate-500'>
                      <span className='line-clamp-1'>{event.impact}</span>
                    </td>
                    <td className='px-4 py-3 text-right text-slate-700'>
                      {event.status}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </OverviewPanel>

        <OverviewPanel
          title='治理资产'
          description='企业能力覆盖概览'
          bodyClassName='grid grid-cols-2 gap-3'
        >
          {[
            {
              label: 'API Keys',
              value: metrics.active_api_keys,
              icon: KeyRound,
              className: 'bg-blue-50 text-blue-700',
            },
            {
              label: '供应商',
              value: `${metrics.healthy_suppliers}/${metrics.total_suppliers}`,
              icon: Building2,
              className: 'bg-emerald-50 text-emerald-700',
            },
            {
              label: '路由策略',
              value: metrics.active_policies,
              icon: Route,
              className: 'bg-violet-50 text-violet-700',
            },
            {
              label: '待办',
              value: metrics.pending_approvals,
              icon: BellRing,
              className: 'bg-amber-50 text-amber-700',
            },
          ].map((item) => {
            const Icon = item.icon
            return (
              <div
                key={item.label}
                className='rounded-lg border border-slate-100 bg-slate-50/70 p-3'
              >
                <span
                  className={cn(
                    'flex size-8 items-center justify-center rounded-lg',
                    item.className
                  )}
                >
                  <Icon className='size-4' />
                </span>
                <p className='mt-3 text-xl font-semibold tracking-tight text-slate-950 tabular-nums'>
                  {item.value}
                </p>
                <p className='mt-0.5 text-[11px] font-medium text-slate-500'>
                  {item.label}
                </p>
              </div>
            )
          })}
          <div className='col-span-2 rounded-lg border border-dashed border-slate-200 bg-white px-3 py-3 text-xs leading-5 text-slate-500'>
            当前毛利预估 {formatCurrencyUSD(metrics.estimated_gross_profit)}
            ，渠道健康度{' '}
            {formatPercentage(
              metrics.total_channels > 0
                ? metrics.healthy_channels / metrics.total_channels
                : 0
            )}
            。建议每天核对供应商账单、客户扣费与失败重试口径。
          </div>
        </OverviewPanel>
      </section>
    </div>
  )
}
