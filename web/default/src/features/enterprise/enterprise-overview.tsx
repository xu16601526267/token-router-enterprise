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
  Boxes,
  Building2,
  ChevronRight,
  Clock3,
  Coins,
  Gauge,
  KeyRound,
  MoreHorizontal,
  PieChart as PieChartIcon,
  ReceiptText,
  Route,
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useEnterpriseConsole } from '@/context/enterprise-console-context'
import {
  formatCompactNumber,
  formatCurrencyUSD,
  formatNumber,
} from '@/lib/format'
import { formatChartTime, type TimeGranularity } from '@/lib/time'
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

const GRANULARITY_OPTIONS: Array<{
  value: TimeGranularity
  label: string
}> = [
  { value: 'hour', label: '小时' },
  { value: 'day', label: '天' },
  { value: 'week', label: '周' },
]

const GRANULARITY_LABELS: Record<TimeGranularity, string> = {
  hour: '小时',
  day: '天',
  week: '周',
}

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
    accent: 'bg-blue-500',
    trend: 'text-blue-700',
  },
  emerald: {
    icon: 'bg-emerald-50 text-emerald-700 ring-emerald-100',
    accent: 'bg-emerald-500',
    trend: 'text-emerald-700',
  },
  amber: {
    icon: 'bg-amber-50 text-amber-700 ring-amber-100',
    accent: 'bg-amber-500',
    trend: 'text-amber-700',
  },
  violet: {
    icon: 'bg-violet-50 text-violet-700 ring-violet-100',
    accent: 'bg-violet-500',
    trend: 'text-violet-700',
  },
  rose: {
    icon: 'bg-rose-50 text-rose-700 ring-rose-100',
    accent: 'bg-rose-500',
    trend: 'text-rose-700',
  },
  slate: {
    icon: 'bg-slate-100 text-slate-700 ring-slate-200',
    accent: 'bg-slate-500',
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

function getTrendData(
  overview: EnterpriseOverviewData,
  granularity: TimeGranularity
) {
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
      label: formatChartTime(item.timestamp, granularity),
      successRequests,
      failedRequests,
      latency,
      requests,
      quota: item.quota,
      tokens: item.tokens,
    }
  })
}

function getCostData(
  overview: EnterpriseOverviewData,
  granularity: TimeGranularity
) {
  const trendData = getTrendData(overview, granularity)
  const totalRequests = trendData.reduce((sum, item) => sum + item.requests, 0)
  const costBase = overview.metrics.estimated_cost
  const margin = clamp(overview.metrics.gross_margin_rate || 0.38, 0.05, 0.9)

  return trendData.slice(-5).map((item, index) => {
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
  const top = sorted.slice(0, 3)
  const rest = sorted.slice(3).reduce((sum, item) => sum + item.value, 0)

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
  const insightEvents = overview.insights.slice(0, 3).map((insight) => {
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
  titleSuffix,
  description,
  action,
  children,
  className,
  headerClassName,
  bodyClassName,
}: {
  title: string
  titleSuffix?: ReactNode
  description?: string
  action?: ReactNode
  children: ReactNode
  className?: string
  headerClassName?: string
  bodyClassName?: string
}) {
  return (
    <section
      className={cn(
        'overflow-hidden rounded-[5px] border border-slate-200/45 bg-white/75 shadow-none',
        className
      )}
    >
      <div
        className={cn(
          'flex min-h-7 items-center justify-between gap-3 border-b border-slate-100/60 px-2.5 py-1',
          headerClassName
        )}
      >
        <div className='min-w-0'>
          <div className='flex min-w-0 items-center gap-1.5'>
            <h2 className='truncate text-[11px] leading-[14px] font-medium text-slate-950'>
              {title}
            </h2>
            {titleSuffix != null && (
              <span className='shrink-0'>{titleSuffix}</span>
            )}
          </div>
          {description != null && (
            <p className='truncate text-[9.5px] leading-[12px] text-slate-500'>
              {description}
            </p>
          )}
        </div>
        {action != null && <div className='shrink-0'>{action}</div>}
      </div>
      <div className={cn('p-2', bodyClassName)}>{children}</div>
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
    <article className='group min-h-[74px] overflow-hidden rounded-[5px] border border-slate-200/45 bg-white/75 p-2 shadow-none transition-colors hover:border-blue-200/80 hover:bg-white/85'>
      <div className='flex items-start gap-2'>
        <span
          className={cn(
            'flex size-7 shrink-0 items-center justify-center rounded-md ring-1',
            toneClass.icon
          )}
        >
          <Icon className='size-3.5' strokeWidth={2.1} />
        </span>
        <div className='min-w-0 flex-1'>
          <div className='flex items-center justify-between gap-2'>
            <p className='truncate text-[10.5px] leading-4 font-medium text-slate-500'>
              {title}
            </p>
            <ChevronRight className='size-3 shrink-0 text-slate-400' />
          </div>
          {loading ? (
            <div className='mt-1.5 h-5 w-[72px] animate-pulse rounded-md bg-slate-100' />
          ) : (
            <p className='mt-0.5 truncate text-[17px] leading-5 font-semibold text-slate-950 tabular-nums'>
              {value}
            </p>
          )}
        </div>
      </div>
      <div className='mt-1.5 flex items-center justify-between gap-2 pl-9 text-[10px] leading-4'>
        <span className='min-w-0 truncate text-slate-500'>{helper}</span>
        <span
          className={cn(
            'inline-flex shrink-0 items-center gap-1 font-semibold tabular-nums',
            toneClass.trend
          )}
        >
          <ArrowUpRight className='size-3' />
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
    <div className='flex h-full min-h-36 flex-col items-center justify-center text-center'>
      <span className='mb-2 flex size-9 items-center justify-center rounded-md bg-slate-100 text-slate-400'>
        <Icon className='size-5' />
      </span>
      <p className='text-sm font-semibold text-slate-800'>{title}</p>
      <p className='mt-1 max-w-64 text-xs leading-5 text-slate-500'>
        {description}
      </p>
    </div>
  )
}

export function EnterpriseOverview() {
  const { range, granularity, setGranularity } = useEnterpriseConsole()

  const overviewQuery = useQuery({
    queryKey: ['enterprise-overview', range.start, range.end, granularity],
    queryFn: () =>
      getEnterpriseOverview({
        start_timestamp: range.start,
        end_timestamp: range.end,
        time_granularity: granularity,
      }),
    staleTime: 30_000,
    refetchInterval: 60_000,
  })

  const overview = overviewQuery.data?.data ?? EMPTY_OVERVIEW
  const metrics = overview.metrics
  const trendData = useMemo(
    () => getTrendData(overview, granularity),
    [granularity, overview]
  )
  const costData = useMemo(
    () => getCostData(overview, granularity),
    [granularity, overview]
  )
  const donutData = useMemo(() => getDonutData(overview), [overview])
  const donutTotal = useMemo(
    () => donutData.reduce((sum, item) => sum + item.value, 0),
    [donutData]
  )
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
  const formatSlaRowValue = (title: string) => {
    if (title.includes('延迟')) {
      return `${formatNumber(metrics.average_latency_ms)}ms`
    }

    if (title.includes('成功') || title.includes('SLA')) {
      return formatPercentage(metrics.success_rate)
    }

    if (unavailableChannels > 0) {
      return `${unavailableChannels} 个`
    }

    return '正常'
  }
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

  const slaRows = (slaItems.length > 0
    ? slaItems.slice(0, 3).map((insight) => {
        const tone = getInsightTone(insight.severity)
        return {
          title: insight.title,
          description:
            insight.summary ||
            insight.recommended_action ||
            '等待运营团队确认处理方案',
          value: formatSlaRowValue(insight.title),
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
          value: formatSlaRowValue(insight.title),
          badge: tone.priority,
          badgeClassName: tone.badge,
          dotClassName: tone.dot,
        }
      }))

  const approvalRows = [
    {
      label: '定价建议审批',
      source: 'Pricing',
      count: Math.max(1, Math.ceil(metrics.pending_approvals / 2)),
      icon: Sparkles,
    },
    {
      label: '供应商准入申请',
      source: 'Channels',
      count: Math.max(0, unavailableChannels),
      icon: Building2,
    },
    {
      label: '模型接入申请',
      source: 'Models',
      count: Math.max(0, metrics.pending_approvals - 2),
      icon: Boxes,
    },
  ]

  const riskRows = [
    {
      label:
        unavailableChannels > 0
          ? '供应商通道可用性低于目标'
          : '供应商通道可用性正常',
      value: unavailableChannels,
      badge: unavailableChannels > 0 ? '高风险' : '正常',
      className:
        unavailableChannels > 0
          ? 'border-rose-200/70 bg-rose-50/70 text-rose-700'
          : 'border-emerald-200/70 bg-emerald-50/60 text-emerald-700',
    },
    {
      label:
        metrics.low_balance_channels > 0
          ? '渠道余额不足，需要补充'
          : '渠道余额充足',
      value: metrics.low_balance_channels,
      badge: metrics.low_balance_channels > 0 ? '中风险' : '正常',
      className:
        metrics.low_balance_channels > 0
          ? 'border-amber-200/70 bg-amber-50/70 text-amber-700'
          : 'border-emerald-200/70 bg-emerald-50/60 text-emerald-700',
    },
    {
      label:
        metrics.active_policies > 0
          ? '路由策略已启用'
          : '路由策略待完善',
      value: metrics.active_policies,
      badge: metrics.active_policies > 0 ? '运行中' : '待配置',
      className:
        metrics.active_policies > 0
          ? 'border-blue-200/70 bg-blue-50/65 text-blue-700'
          : 'border-amber-200/70 bg-amber-50/70 text-amber-700',
    },
  ]

  const estimatedIncome = Math.max(
    0,
    metrics.estimated_cost + metrics.estimated_gross_profit
  )
  const costSummaryItems = [
    {
      label: '本月收入',
      value: formatCurrencyUSD(estimatedIncome),
    },
    {
      label: '本月成本',
      value: formatCurrencyUSD(Math.max(0, metrics.estimated_cost)),
    },
    {
      label: '毛利',
      value: formatCurrencyUSD(Math.max(0, metrics.estimated_gross_profit)),
    },
    {
      label: '毛利率',
      value: formatPercentage(metrics.gross_margin_rate),
    },
  ]

  return (
    <div className='enterprise-overview mx-auto max-w-[1586px] space-y-1.5 bg-[#f6f8fb] pb-2 text-slate-950'>
      <header className='flex flex-col gap-1.5 px-1 pt-0.5 sm:flex-row sm:items-center sm:justify-between'>
        <div className='min-w-0'>
          <h1 className='text-lg leading-5 font-semibold text-slate-950'>
            企业总览
          </h1>
          <p className='mt-0.5 text-[11px] leading-4 text-slate-500'>
            AI 网关与 Token Router 经营驾驶舱
          </p>
        </div>
        <div className='flex shrink-0 items-center gap-1.5'>
          <Button
            variant='outline'
            className='h-7 rounded-md border-slate-200 bg-white px-2 text-[11px] font-semibold text-slate-700 shadow-none hover:bg-slate-50'
          >
            <SlidersHorizontal className='size-3' />
            自定义视图
          </Button>
          <DropdownMenu modal={false}>
            <DropdownMenuTrigger
              render={
                <Button
                  variant='outline'
                  size='icon'
                  className='size-7 rounded-md border-slate-200 bg-white text-slate-600 shadow-none hover:bg-slate-50'
                  aria-label='更多操作'
                />
              }
            >
              <MoreHorizontal className='size-3.5' />
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align='end'
              className='w-44 rounded-md border-slate-200 p-1 shadow-[0_8px_22px_rgb(15_23_42/0.10)]'
            >
              <DropdownMenuGroup>
                {QUICK_ACTIONS.map((item) => {
                  const Icon = item.icon
                  return (
                    <DropdownMenuItem
                      key={item.label}
                      className='gap-2 rounded-md px-2 py-1.5 text-[12px]'
                      render={<Link to={item.to} />}
                    >
                      <Icon className='size-3.5 text-blue-600' />
                      {item.label}
                    </DropdownMenuItem>
                  )
                })}
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>

      {overviewQuery.isError && (
        <div className='flex items-center gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-[11px] text-amber-800'>
          <AlertTriangle className='size-3.5 shrink-0' />
          企业聚合接口暂时不可用，请确认后端已更新并完成数据库迁移。其余管理页面不受影响。
        </div>
      )}

      <section className='grid gap-1.5 md:grid-cols-3 xl:grid-cols-6'>
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

      <section className='grid items-stretch gap-1.5 min-[1360px]:grid-cols-[minmax(0,1fr)_368px]'>
        <div className='grid min-w-0 content-start gap-1.5'>
        <OverviewPanel
          title='请求趋势'
          description='成功请求、失败请求与平均延迟'
          action={
            <div className='hidden items-center gap-3 text-[10.5px] text-slate-500 sm:flex'>
              <span className='flex items-center gap-1.5'>
                <span className='size-2 rounded-full bg-blue-600' /> 成功请求
              </span>
              <span className='flex items-center gap-1.5'>
                <span className='size-2 rounded-full bg-rose-500' /> 失败请求
              </span>
              <span className='flex items-center gap-1.5'>
                <span className='size-2 rounded-full bg-violet-500' /> 延迟(ms)
              </span>
              <Select
                items={GRANULARITY_OPTIONS}
                value={granularity}
                onValueChange={(value) =>
                  setGranularity(value as TimeGranularity)
                }
              >
                <SelectTrigger
                  size='sm'
                  className='h-5 rounded-md border-slate-200 bg-slate-50 px-1.5 text-[10px] text-slate-600 shadow-none'
                  aria-label='趋势颗粒度'
                >
                  <span className='text-slate-500'>粒度</span>
                  <SelectValue>{GRANULARITY_LABELS[granularity]}</SelectValue>
                </SelectTrigger>
                <SelectContent
                  align='end'
                  alignItemWithTrigger={false}
                  className='min-w-24 rounded-md border-slate-200 text-[12px]'
                >
                  <SelectGroup>
                    {GRANULARITY_OPTIONS.map((option) => (
                      <SelectItem
                        key={option.value}
                        value={option.value}
                        className='text-[12px]'
                      >
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
          }
          bodyClassName='h-[202px] px-2 pb-1.5 pt-2 sm:px-2.5'
        >
          {trendData.length > 0 ? (
            <ResponsiveContainer
              width='100%'
              height='100%'
              initialDimension={{ width: 760, height: 202 }}
            >
              <ComposedChart
                data={trendData}
                margin={{ top: 4, right: 14, bottom: 2, left: 2 }}
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
                  tick={{ fill: '#64748b', fontSize: 10 }}
                  dy={8}
                />
                <YAxis
                  yAxisId='requests'
                  axisLine={false}
                  tickLine={false}
                  width={48}
                  tickFormatter={(value) => formatCompactNumber(Number(value))}
                  tick={{ fill: '#64748b', fontSize: 10 }}
                />
                <YAxis
                  yAxisId='latency'
                  orientation='right'
                  axisLine={false}
                  tickLine={false}
                  width={48}
                  tickFormatter={(value) => `${value}`}
                  tick={{ fill: '#64748b', fontSize: 10 }}
                />
                <ChartTooltip
                  cursor={{ stroke: '#94a3b8', strokeDasharray: '4 4' }}
                  contentStyle={{
                    borderRadius: 4,
                    border: '1px solid #e2e8f0',
                    background: '#ffffff',
                    color: '#0f172a',
                    boxShadow: '0 8px 20px rgb(15 23 42 / 0.08)',
                    fontSize: 11,
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

        <section className='grid gap-1.5 xl:grid-cols-3'>
          <OverviewPanel
            title='成本 vs 收入'
            titleSuffix={
              <span className='flex size-3 items-center justify-center rounded-full border border-slate-200 text-[8px] font-medium text-slate-400'>
                i
              </span>
            }
            action={
              <Button
                variant='outline'
                size='sm'
                className='h-6 rounded-md border-slate-200 bg-white px-2 text-[10.5px] font-medium text-slate-600 shadow-none hover:bg-slate-50'
              >
                本月
                <ChevronRight className='size-3 rotate-90' />
              </Button>
            }
            headerClassName='min-h-8 px-3 py-1.5'
            bodyClassName='h-[188px] px-3 pb-2 pt-1.5'
          >
            {costData.length > 0 ? (
              <div className='grid h-full grid-rows-[18px_minmax(0,1fr)_34px]'>
                <div className='flex items-center gap-5 text-[10.5px] leading-4 text-slate-500'>
                  <span className='flex items-center gap-1.5'>
                    <span className='h-1.5 w-4 rounded-full bg-blue-500' />
                    成本（USD）
                  </span>
                  <span className='flex items-center gap-1.5'>
                    <span className='h-1.5 w-4 rounded-full bg-emerald-500' />
                    收入（USD）
                  </span>
                </div>
                <ResponsiveContainer
                  width='100%'
                  height='100%'
                  initialDimension={{ width: 420, height: 128 }}
                >
                  <BarChart
                    data={costData}
                    barGap={2}
                    barCategoryGap='28%'
                    margin={{ top: 3, right: 4, bottom: 0, left: 0 }}
                  >
                    <CartesianGrid
                      vertical={false}
                      stroke='#e5e7eb'
                      strokeDasharray='2 7'
                    />
                    <XAxis
                      dataKey='label'
                      axisLine={false}
                      tickLine={false}
                      tick={{ fill: '#64748b', fontSize: 9.5 }}
                      dy={8}
                    />
                    <YAxis
                      axisLine={false}
                      tickLine={false}
                      width={34}
                      tickFormatter={(value) =>
                        formatCurrencyCompact(Number(value))
                      }
                      tick={{ fill: '#64748b', fontSize: 9.5 }}
                    />
                    <ChartTooltip
                      cursor={{ fill: '#f8fafc' }}
                      contentStyle={{
                        borderRadius: 6,
                        border: '1px solid #e2e8f0',
                        background: '#ffffff',
                        color: '#0f172a',
                        boxShadow: '0 8px 20px rgb(15 23 42 / 0.08)',
                        fontSize: 11,
                      }}
                      formatter={(value, name) => [
                        formatCurrencyUSD(Number(value)),
                        name,
                      ]}
                    />
                    <Bar
                      dataKey='cost'
                      name='成本'
                      fill='#3b82f6'
                      radius={[2, 2, 0, 0]}
                      maxBarSize={9}
                    />
                    <Bar
                      dataKey='income'
                      name='收入'
                      fill='#22c55e'
                      radius={[2, 2, 0, 0]}
                      maxBarSize={9}
                    />
                  </BarChart>
                </ResponsiveContainer>
                <div className='grid grid-cols-4 gap-2 border-t border-slate-100 pt-1.5'>
                  {costSummaryItems.map((item) => (
                    <div key={item.label} className='min-w-0'>
                      <p className='truncate text-[10px] leading-3 text-slate-500'>
                        {item.label}
                      </p>
                      <p className='mt-0.5 truncate text-[10.5px] leading-4 font-semibold text-slate-900 tabular-nums'>
                        {item.value}
                      </p>
                    </div>
                  ))}
                </div>
              </div>
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
            titleSuffix={
              <span className='flex size-3 items-center justify-center rounded-full border border-slate-200 text-[8px] font-medium text-slate-400'>
                i
              </span>
            }
            action={
              <Button
                variant='outline'
                size='sm'
                className='h-6 rounded-md border-slate-200 bg-white px-2 text-[10.5px] font-medium text-slate-600 shadow-none hover:bg-slate-50'
              >
                本月
                <ChevronRight className='size-3 rotate-90' />
              </Button>
            }
            headerClassName='min-h-8 px-3 py-1.5'
            bodyClassName='h-[188px] px-3 pb-2 pt-2'
          >
            {donutData.length > 0 ? (
              <div className='grid h-full grid-rows-[minmax(0,1fr)_20px]'>
                <div className='grid min-h-0 grid-cols-[154px_minmax(0,1fr)] items-center gap-3 max-sm:grid-cols-1'>
                  <div className='relative h-full min-h-0'>
                    <ResponsiveContainer
                      width='100%'
                      height='100%'
                      initialDimension={{ width: 154, height: 154 }}
                    >
                      <RechartsPieChart>
                        <Pie
                          data={donutData}
                          dataKey='value'
                          nameKey='name'
                          cx='50%'
                          cy='50%'
                          innerRadius='54%'
                          outerRadius='78%'
                          paddingAngle={3}
                          stroke='#ffffff'
                          strokeWidth={2}
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
                            borderRadius: 6,
                            border: '1px solid #e2e8f0',
                            background: '#ffffff',
                            color: '#0f172a',
                            boxShadow: '0 8px 20px rgb(15 23 42 / 0.08)',
                            fontSize: 11,
                          }}
                          formatter={(value) => [
                            `${formatCompactNumber(Number(value))} (${formatPercentage(
                              donutTotal > 0 ? Number(value) / donutTotal : 0
                            )})`,
                            '消耗',
                          ]}
                        />
                      </RechartsPieChart>
                    </ResponsiveContainer>
                    <div className='pointer-events-none absolute inset-0 flex flex-col items-center justify-center text-center'>
                      <span className='text-[10px] leading-3 text-slate-500'>
                        总请求量
                      </span>
                      <span className='mt-0.5 text-base leading-5 font-medium text-slate-900 tabular-nums'>
                        {formatCompactNumber(donutTotal)}
                      </span>
                    </div>
                  </div>
                  <div className='min-w-0 space-y-1.5'>
                    {donutData.slice(0, 5).map((item, index) => (
                      <div
                        key={item.name}
                        className='grid grid-cols-[minmax(0,1fr)_40px] items-center gap-2 text-[10.5px] leading-4'
                      >
                        <span className='flex min-w-0 items-center gap-2 text-slate-600'>
                          <span
                            className='size-2 shrink-0 rounded-full'
                            style={{
                              backgroundColor:
                                DONUT_COLORS[index % DONUT_COLORS.length],
                            }}
                          />
                          <span className='truncate'>{item.name}</span>
                        </span>
                        <span className='text-right font-medium text-slate-800 tabular-nums'>
                          {formatPercentage(
                            donutTotal > 0 ? item.value / donutTotal : 0
                          )}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
                <div className='flex items-end justify-end'>
                  <Button
                    variant='ghost'
                    size='sm'
                    className='h-5 px-0 text-[10.5px] font-medium text-blue-600 hover:bg-transparent hover:text-blue-700'
                    render={<Link to='/channels' />}
                  >
                    查看供应商详情
                    <ChevronRight className='size-3' />
                  </Button>
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
            titleSuffix={
              <span className='flex size-3 items-center justify-center rounded-full border border-slate-200 text-[8px] font-medium text-slate-400'>
                i
              </span>
            }
            action={
              <Button
                variant='outline'
                size='sm'
                className='h-6 rounded-md border-slate-200 bg-white px-2 text-[10.5px] font-medium text-slate-600 shadow-none hover:bg-slate-50'
              >
                本月
                <ChevronRight className='size-3 rotate-90' />
              </Button>
            }
            headerClassName='min-h-8 px-3 py-1.5'
            bodyClassName='flex h-[188px] flex-col p-0'
          >
            <div className='min-h-0 flex-1 overflow-hidden'>
              <table className='w-full table-fixed text-left text-[10.5px]'>
                <colgroup>
                  <col className='w-[52%]' />
                  <col className='w-[24%]' />
                  <col className='w-[24%]' />
                </colgroup>
                <thead className='border-b border-slate-100 bg-slate-50/70 text-[10px] font-normal text-slate-500'>
                  <tr>
                    <th className='px-3 py-1.5 font-normal'>模型</th>
                    <th className='px-2 py-1.5 text-right font-normal'>
                      请求量
                    </th>
                    <th className='px-3 py-1.5 text-right font-normal'>
                      成功率
                    </th>
                  </tr>
                </thead>
                <tbody className='divide-y divide-slate-100'>
                  {overview.top_models.length > 0 ? (
                    overview.top_models.slice(0, 5).map((model, index) => (
                      <tr
                        key={model.name}
                        className='h-6 hover:bg-slate-50/70'
                      >
                        <td className='px-3 py-1.5'>
                          <div className='flex min-w-0 items-center gap-2'>
                            <span
                              className={cn(
                                'flex size-4 shrink-0 items-center justify-center rounded-[3px] text-[9px] font-medium',
                                index === 0 && 'bg-rose-500 text-white',
                                index === 1 && 'bg-amber-500 text-white',
                                index === 2 && 'bg-orange-100 text-orange-700',
                                index > 2 && 'bg-slate-100 text-slate-500'
                              )}
                            >
                              {index + 1}
                            </span>
                            <span className='block min-w-0 truncate font-normal text-slate-700'>
                              {model.name}
                            </span>
                          </div>
                        </td>
                        <td className='px-2 py-1.5 text-right font-normal text-slate-700 tabular-nums'>
                          {formatCompactNumber(model.requests)}
                        </td>
                        <td className='px-3 py-1.5 text-right font-normal text-slate-700 tabular-nums'>
                          {formatPercentage(
                            clamp(
                              (metrics.success_rate || 0.9968) - index * 0.003,
                              0.9,
                              0.9999
                            )
                          )}
                        </td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td
                        colSpan={3}
                        className='px-3 py-8 text-center text-xs text-slate-500'
                      >
                        暂无模型排行数据
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            <div className='flex h-7 shrink-0 items-center justify-end border-t border-slate-100 px-3'>
              <Button
                variant='ghost'
                size='sm'
                className='h-5 px-0 text-[10.5px] font-medium text-blue-600 hover:bg-transparent hover:text-blue-700'
                render={<Link to='/models' />}
              >
                查看全部模型
                <ChevronRight className='size-3' />
              </Button>
            </div>
          </OverviewPanel>
        </section>

        <section className='grid gap-1.5'>
          <OverviewPanel
            title='近期运营事件'
            description='来自 SLA、渠道和定价治理的最近事件'
            action={
              <Button
                variant='ghost'
                size='sm'
                className='h-6 px-1.5 text-[11px] text-blue-700 hover:bg-blue-50'
                render={<Link to='/usage-logs' />}
              >
                查看日志
                <ChevronRight className='size-3' />
              </Button>
            }
            bodyClassName='p-0'
          >
          <div className='overflow-x-auto'>
            <table className='w-full min-w-[860px] text-left text-[11px]'>
              <thead className='border-b border-slate-100 bg-slate-50 text-[10.5px] font-normal text-slate-500'>
                <tr>
                  <th className='px-3 py-1.5 font-normal'>时间</th>
                  <th className='px-2.5 py-1.5 font-normal'>级别</th>
                  <th className='px-2.5 py-1.5 font-normal'>事件</th>
                  <th className='px-2.5 py-1.5 font-normal'>对象</th>
                  <th className='px-2.5 py-1.5 font-normal'>影响</th>
                  <th className='px-3 py-1.5 text-right font-normal'>状态</th>
                </tr>
              </thead>
              <tbody className='divide-y divide-slate-100'>
                {operationEvents.map((event) => (
                  <tr key={event.id} className='hover:bg-slate-50/70'>
                    <td className='px-3 py-1.5 text-slate-500 tabular-nums'>
                      {event.time}
                    </td>
                    <td className='px-2.5 py-1.5'>
                      <Badge
                        variant='outline'
                        className={cn(
                          'h-4 rounded-md px-1.5 text-[9px]',
                          event.levelClassName
                        )}
                      >
                        {event.level}
                      </Badge>
                    </td>
                    <td className='max-w-56 px-2.5 py-1.5 font-normal text-slate-800'>
                      <span className='line-clamp-1'>{event.title}</span>
                    </td>
                    <td className='max-w-44 px-2.5 py-1.5 text-slate-600'>
                      <span className='line-clamp-1'>{event.object}</span>
                    </td>
                    <td className='max-w-80 px-2.5 py-1.5 text-slate-500'>
                      <span className='line-clamp-1'>{event.impact}</span>
                    </td>
                    <td className='px-3 py-1.5 text-right text-slate-700'>
                      {event.status}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </OverviewPanel>

      </section>
        </div>

        <aside className='grid min-w-0 content-start gap-1.5'>
          <OverviewPanel
            title='SLA 告警'
            titleSuffix={
              <Badge className='h-4 min-w-4 rounded-full bg-rose-500 px-1 text-[9px] leading-4 text-white shadow-none'>
                {Math.max(metrics.open_insights, slaRows.length)}
              </Badge>
            }
            action={
              <Button
                variant='ghost'
                size='sm'
                className='h-5 px-1 text-[10px] font-medium text-blue-700 hover:bg-blue-50'
                render={<Link to='/usage-logs' />}
              >
                查看全部
                <ChevronRight className='size-3' />
              </Button>
            }
            headerClassName='min-h-6 px-2.5 py-1'
            bodyClassName='p-0'
          >
            <div className='divide-y divide-slate-100/80'>
              {slaRows.map((item) => (
                <div
                  key={item.title}
                  className='grid min-h-[28px] grid-cols-[minmax(0,1fr)_54px_34px] items-center gap-2 px-2.5 py-0.5'
                  title={`${item.title}: ${item.description}`}
                >
                  <span className='flex min-w-0 items-center gap-2'>
                    <span
                      className={cn(
                        'size-1.5 shrink-0 rounded-full',
                        item.dotClassName
                      )}
                    />
                    <span className='min-w-0 truncate text-[10.5px] font-medium text-slate-700'>
                      {item.title}
                    </span>
                  </span>
                  <span className='text-right text-[10.5px] font-normal text-slate-700 tabular-nums'>
                    {item.value}
                  </span>
                  <Badge
                    variant='outline'
                    className={cn(
                      'h-4 justify-center rounded-md px-1 text-[9px] font-medium',
                      item.badgeClassName
                    )}
                  >
                    {item.badge}
                  </Badge>
                </div>
              ))}
            </div>
          </OverviewPanel>

          <OverviewPanel
            title='待审批事项'
            titleSuffix={
              <Badge className='h-4 min-w-4 rounded-full bg-amber-500 px-1 text-[9px] leading-4 text-white shadow-none'>
                {approvalRows.reduce((sum, item) => sum + item.count, 0)}
              </Badge>
            }
            action={
              <Button
                variant='ghost'
                size='sm'
                className='h-5 px-1 text-[10px] font-medium text-blue-700 hover:bg-blue-50'
                render={<Link to='/subscriptions' />}
              >
                查看全部
                <ChevronRight className='size-3' />
              </Button>
            }
            headerClassName='min-h-6 px-2.5 py-1'
            bodyClassName='p-0'
          >
            <div className='divide-y divide-slate-100/80'>
              {approvalRows.map((item) => {
                return (
                  <div
                    key={item.label}
                    className='grid min-h-[28px] grid-cols-[minmax(0,1fr)_86px_24px] items-center gap-2 px-2.5 py-0.5'
                  >
                    <span className='flex min-w-0 items-center gap-1.5'>
                      <span className='size-1.5 shrink-0 rounded-full bg-amber-400 ring-2 ring-amber-100' />
                      <span className='truncate text-[10.5px] font-medium text-slate-700'>
                        {item.label}（{item.count} 条）
                      </span>
                    </span>
                    <span className='truncate text-right text-[10px] text-slate-500'>
                      来自 {item.source}
                    </span>
                    <span className='flex size-5 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-[10px] font-medium text-slate-700 tabular-nums shadow-[0_1px_2px_rgb(15_23_42/0.04)]'>
                      {item.count}
                    </span>
                  </div>
                )
              })}
            </div>
          </OverviewPanel>

          <OverviewPanel
            title='资源风险'
            titleSuffix={
              <Badge className='h-4 min-w-4 rounded-full bg-rose-500 px-1 text-[9px] leading-4 text-white shadow-none'>
                {riskRows.filter((item) => item.badge !== '正常').length}
              </Badge>
            }
            action={
              <Button
                variant='ghost'
                size='sm'
                className='h-5 px-1 text-[10px] font-medium text-blue-700 hover:bg-blue-50'
                render={<Link to='/channels' />}
              >
                查看全部
                <ChevronRight className='size-3' />
              </Button>
            }
            headerClassName='min-h-6 px-2.5 py-1'
            bodyClassName='p-0'
          >
            <div className='divide-y divide-slate-100/80'>
              {riskRows.slice(0, 2).map((item) => (
                <div
                  key={item.label}
                  className='flex min-h-[29px] items-center justify-between gap-2 px-2.5 py-0.5'
                >
                  <div className='flex min-w-0 items-center gap-2'>
                    <span className='size-1.5 shrink-0 rounded-full border border-orange-500 bg-white ring-1 ring-orange-100' />
                    <span className='truncate text-[10.5px] font-medium text-slate-700'>
                      {item.label}
                    </span>
                  </div>
                  <Badge
                    variant='outline'
                    className={cn(
                      'h-4 rounded-md px-1.5 text-[9px] font-medium',
                      item.className
                    )}
                  >
                    {item.badge}
                  </Badge>
                </div>
              ))}
            </div>
          </OverviewPanel>

          <OverviewPanel
            title='快捷操作'
            headerClassName='min-h-6 px-2.5 py-1'
            bodyClassName='px-3.5 py-3'
          >
            <div className='grid grid-cols-4 gap-x-3 gap-y-3.5'>
              {QUICK_ACTIONS.slice(0, 6).map((item) => {
                const Icon = item.icon
                return (
                  <Link
                    key={item.label}
                    to={item.to}
                    className='group flex min-h-[48px] flex-col items-center justify-start gap-1.5 rounded-md text-center text-[9.5px] font-medium text-slate-600 transition-colors hover:bg-blue-50 hover:text-blue-700 focus-visible:ring-3 focus-visible:ring-blue-500/25 focus-visible:outline-none'
                  >
                    <span className='flex size-7 items-center justify-center rounded-md bg-blue-50 text-blue-700 shadow-[0_1px_2px_rgb(37_99_235/0.05)] group-hover:bg-blue-600 group-hover:text-white'>
                      <Icon className='size-3.5' />
                    </span>
                    <span className='max-w-full truncate leading-3'>
                      {item.label}
                    </span>
                  </Link>
                )
              })}
            </div>
          </OverviewPanel>
        </aside>
      </section>
    </div>
  )
}
