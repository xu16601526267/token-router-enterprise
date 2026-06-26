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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  ArrowRight,
  CheckCircle2,
  Clock3,
  Download,
  FileText,
  GitBranch,
  MoreHorizontal,
  Network,
  RefreshCw,
  Route,
  ShieldCheck,
  SlidersHorizontal,
  Target,
  Zap,
  type LucideIcon,
} from 'lucide-react'
import { useMemo, type ReactNode } from 'react'
import {
  Area,
  AreaChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { toast } from 'sonner'

import { EnterprisePageHeader, EnterprisePanel } from '@/components/enterprise'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useEnterpriseConsole } from '@/context/enterprise-console-context'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

import {
  acknowledgeOperatingInsight,
  disableSupplyRoutingPolicy,
  dismissOperatingInsight,
  updateSupplyActionPlanStatus,
} from '../api'
import { getControlTower } from '../control-tower-api'
import type {
  ControlTowerData,
  ControlTowerEvent,
  ProviderHealth,
  RoutingPolicyItem,
} from '../control-tower-types'

function formatCount(value: number): string {
  return new Intl.NumberFormat('zh-CN', { notation: 'compact' }).format(value)
}

function formatPercent(value: number): string {
  return `${(value * 100).toFixed(2)}%`
}

function formatLatency(value: number): string {
  if (value <= 0) {
    return '0ms'
  }
  if (value < 10) {
    return `${value.toFixed(2)}ms`
  }
  return `${value.toFixed(0)}ms`
}

function csvEscape(value: string | number): string {
  const text = String(value ?? '')
  if (/[",\n\r]/.test(text)) {
    return `"${text.replaceAll('"', '""')}"`
  }
  return text
}

function exportControlTowerCsv(data: ControlTowerData | undefined) {
  if (!data) {
    toast.error('暂无可导出的控制塔数据')
    return
  }

  const rows: Array<Array<string | number>> = [
    ['section', 'name', 'metric', 'value', 'status'],
    ['metrics', 'requests', 'requests', data.metrics.requests, ''],
    ['metrics', 'tokens', 'tokens', data.metrics.tokens, ''],
    [
      'metrics',
      'success_rate',
      'success_rate',
      formatPercent(data.metrics.realtime_success_rate),
      '',
    ],
    [
      'metrics',
      'average_latency',
      'latency_ms',
      data.metrics.average_latency_ms.toFixed(2),
      '',
    ],
  ]

  data.provider_health.forEach((provider) => {
    rows.push([
      'provider',
      provider.channel_name,
      provider.supplier_name || '未绑定供应商',
      provider.requests,
      provider.status === 1 ? 'enabled' : 'disabled',
    ])
  })
  data.policies.forEach((policy) => {
    rows.push([
      'policy',
      policy.name,
      policy.model_name || '全部模型',
      policy.traffic_percent,
      policy.status,
    ])
  })
  data.pending_actions.forEach((event) => {
    rows.push([
      'action',
      event.title,
      event.category,
      event.detail,
      event.status,
    ])
  })
  data.risks.forEach((event) => {
    rows.push(['risk', event.title, event.severity, event.detail, event.status])
  })

  const csv = rows.map((row) => row.map(csvEscape).join(',')).join('\n')
  const blob = new Blob([`\uFEFF${csv}`], {
    type: 'text/csv;charset=utf-8;',
  })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `token-router-control-tower-${dayjs().format('YYYYMMDD-HHmmss')}.csv`
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
  URL.revokeObjectURL(url)
  toast.success('控制塔数据已导出')
}

function eventTone(severity: string): string {
  if (severity === 'action' || severity === 'high' || severity === 'P1') {
    return 'bg-rose-50 text-rose-600 ring-1 ring-rose-100'
  }
  if (severity === 'watch' || severity === 'medium' || severity === 'P2') {
    return 'bg-amber-50 text-amber-600 ring-1 ring-amber-100'
  }
  return 'bg-blue-50 text-blue-600 ring-1 ring-blue-100'
}

function providerStatus(provider: ProviderHealth): {
  label: string
  className: string
  dotClassName: string
} {
  if (provider.status !== 1) {
    return {
      label: '已停用',
      className: 'bg-slate-100 text-slate-500',
      dotClassName: 'bg-slate-300',
    }
  }
  if (provider.success_rate > 0 && provider.success_rate < 0.98) {
    return {
      label: '需关注',
      className: 'bg-amber-50 text-amber-600',
      dotClassName: 'bg-amber-400',
    }
  }
  return {
    label: '健康良好',
    className: 'bg-emerald-50 text-emerald-600',
    dotClassName: 'bg-emerald-500',
  }
}

function policyStatus(policy: RoutingPolicyItem): {
  label: string
  className: string
} {
  if (policy.status === 'active') {
    return {
      label: '运行中',
      className: 'bg-emerald-50 text-emerald-600 ring-1 ring-emerald-100',
    }
  }
  return {
    label: '未启用',
    className: 'bg-slate-100 text-slate-500 ring-1 ring-slate-200',
  }
}

function approvalStatus(policy: RoutingPolicyItem): {
  label: string
  className: string
} {
  if (policy.status === 'active') {
    return {
      label: '已通过',
      className: 'bg-emerald-50 text-emerald-600 ring-1 ring-emerald-100',
    }
  }
  return {
    label: '待审批',
    className: 'bg-amber-50 text-amber-600 ring-1 ring-amber-100',
  }
}

function MetricTile(props: {
  title: string
  value: string
  helper: string
  icon: LucideIcon
  tone: 'blue' | 'emerald' | 'violet' | 'orange' | 'rose'
  loading?: boolean
}) {
  const toneClass = {
    blue: 'bg-blue-50 text-blue-600 ring-blue-100',
    emerald: 'bg-emerald-50 text-emerald-600 ring-emerald-100',
    violet: 'bg-violet-50 text-violet-600 ring-violet-100',
    orange: 'bg-orange-50 text-orange-600 ring-orange-100',
    rose: 'bg-rose-50 text-rose-600 ring-rose-100',
  }[props.tone]
  const Icon = props.icon

  return (
    <div className='rounded-md border border-slate-200 bg-white px-3 py-2.5 shadow-[0_1px_2px_rgb(15_23_42/0.035)]'>
      <div className='flex items-start gap-2.5'>
        <span
          className={cn(
            'flex size-8 shrink-0 items-center justify-center rounded-md ring-1',
            toneClass
          )}
        >
          <Icon className='size-4' aria-hidden='true' />
        </span>
        <div className='min-w-0 flex-1'>
          <div className='flex items-center justify-between gap-2'>
            <p className='truncate text-[12px] font-medium text-slate-600'>
              {props.title}
            </p>
            <ArrowRight className='size-3.5 shrink-0 text-slate-400' />
          </div>
          <p className='mt-1 text-[22px] leading-6 font-semibold tracking-normal text-slate-950'>
            {props.loading ? '...' : props.value}
          </p>
          <p className='mt-1 text-[11px] text-slate-500'>{props.helper}</p>
        </div>
      </div>
    </div>
  )
}

function EventList(props: {
  events: ControlTowerEvent[]
  emptyText: string
  maxItems?: number
  renderAction?: (event: ControlTowerEvent) => ReactNode
}) {
  const events = props.events.slice(0, props.maxItems ?? 4)
  if (events.length === 0) {
    return (
      <div className='flex min-h-12 items-center gap-2 rounded-md border border-slate-200 bg-slate-50/65 px-2.5 text-xs text-slate-500'>
        <span className='flex size-5 shrink-0 items-center justify-center rounded-full bg-emerald-50 text-emerald-600 ring-1 ring-emerald-100'>
          <CheckCircle2 className='size-3' aria-hidden='true' />
        </span>
        <span className='truncate'>{props.emptyText}</span>
      </div>
    )
  }
  return (
    <div className='space-y-1'>
      {events.map((event) => {
        const action = props.renderAction?.(event)
        return (
          <div
            key={`${event.category}-${event.id}`}
            className='flex items-start gap-2 rounded-md px-1.5 py-1.5 transition-colors hover:bg-slate-50'
          >
            <span
              className={cn(
                'mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full',
                eventTone(event.severity)
              )}
            >
              <AlertTriangle className='size-3' aria-hidden='true' />
            </span>
            <div className='min-w-0 flex-1'>
              <div className='flex items-start justify-between gap-2'>
                <p className='truncate text-xs font-semibold text-slate-900'>
                  {event.title}
                </p>
                <span className='shrink-0 text-[10px] text-slate-400'>
                  {event.created_at > 0
                    ? dayjs.unix(event.created_at).format('MM-DD HH:mm')
                    : '-'}
                </span>
              </div>
              <p className='mt-0.5 line-clamp-1 text-[11px] text-slate-500'>
                {event.detail || '暂无补充说明'}
              </p>
              {action && (
                <div className='mt-1 flex flex-wrap gap-1.5'>{action}</div>
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}

function HealthRow(props: { provider: ProviderHealth }) {
  const status = providerStatus(props.provider)
  const latency =
    props.provider.average_latency_ms || props.provider.response_time_ms
  return (
    <div className='grid grid-cols-[minmax(0,1fr)_54px_48px_54px] items-center gap-2 rounded-md px-1.5 py-1.5 hover:bg-slate-50'>
      <div className='flex min-w-0 items-center gap-2'>
        <span className={cn('size-1.5 rounded-full', status.dotClassName)} />
        <div className='min-w-0'>
          <p className='truncate text-xs font-semibold text-slate-900'>
            {props.provider.supplier_name || props.provider.channel_name}
          </p>
          <p className='truncate text-[10px] text-slate-500'>
            {props.provider.channel_name}
          </p>
        </div>
      </div>
      <div className='text-right text-[11px] font-medium text-slate-700'>
        {formatPercent(props.provider.success_rate)}
      </div>
      <div className='text-right text-[11px] text-slate-500'>
        {formatCount(props.provider.requests)}
      </div>
      <div className='text-right text-[11px] text-slate-500'>
        {formatLatency(latency)}
      </div>
    </div>
  )
}

function ProviderRouteNode(props: {
  provider: ProviderHealth
  requestTotal: number
  index: number
}) {
  const status = providerStatus(props.provider)
  const trafficShare =
    props.requestTotal > 0 ? props.provider.requests / props.requestTotal : 0
  const latency =
    props.provider.average_latency_ms || props.provider.response_time_ms

  return (
    <div className='rounded-md border border-slate-200 bg-white px-3 py-2 shadow-[0_1px_1px_rgb(15_23_42/0.03)]'>
      <div className='flex items-center justify-between gap-2'>
        <div className='flex min-w-0 items-center gap-2'>
          <span
            className={cn(
              'flex size-6 shrink-0 items-center justify-center rounded-md text-[11px] font-semibold',
              props.index === 0
                ? 'bg-blue-50 text-blue-600'
                : 'bg-slate-100 text-slate-500'
            )}
          >
            {props.index === 0 ? '主' : '备'}
          </span>
          <p className='truncate text-xs font-semibold text-slate-900'>
            {props.provider.channel_name}
          </p>
        </div>
        <Badge
          className={cn('h-4 rounded px-1.5 text-[10px]', status.className)}
        >
          {status.label}
        </Badge>
      </div>
      <div className='mt-2 grid grid-cols-3 gap-1 text-[11px]'>
        <div>
          <p className='text-slate-400'>流量</p>
          <p className='font-semibold text-slate-800'>
            {trafficShare > 0 ? formatPercent(trafficShare) : '0%'}
          </p>
        </div>
        <div>
          <p className='text-slate-400'>成功率</p>
          <p className='font-semibold text-slate-800'>
            {formatPercent(props.provider.success_rate)}
          </p>
        </div>
        <div>
          <p className='text-slate-400'>延迟</p>
          <p className='font-semibold text-slate-800'>
            {formatLatency(latency)}
          </p>
        </div>
      </div>
    </div>
  )
}

function RuleItem(props: { icon: LucideIcon; title: string; detail: string }) {
  const Icon = props.icon
  return (
    <div className='flex gap-2 border-b border-slate-100 px-2.5 py-2 last:border-b-0'>
      <span className='mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-md bg-blue-50 text-blue-600'>
        <Icon className='size-3.5' aria-hidden='true' />
      </span>
      <div className='min-w-0'>
        <p className='text-xs font-semibold text-slate-900'>{props.title}</p>
        <p className='mt-0.5 line-clamp-2 text-[11px] leading-4 text-slate-500'>
          {props.detail}
        </p>
      </div>
    </div>
  )
}

function TrendMiniChart(props: {
  trend: Array<{ date: string; requests: number; success_rate: number }>
}) {
  if (props.trend.length === 0) {
    return (
      <div className='flex h-12 items-center justify-center rounded-md bg-slate-50 text-[11px] text-slate-400'>
        暂无趋势
      </div>
    )
  }
  return (
    <div className='h-12'>
      <ResponsiveContainer
        width='100%'
        height='100%'
        initialDimension={{ width: 260, height: 48 }}
      >
        <AreaChart data={props.trend} margin={{ left: 0, right: 0, top: 4 }}>
          <defs>
            <linearGradient
              id='controlTowerMiniTrend'
              x1='0'
              x2='0'
              y1='0'
              y2='1'
            >
              <stop offset='5%' stopColor='#2563eb' stopOpacity={0.22} />
              <stop offset='95%' stopColor='#2563eb' stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <XAxis dataKey='date' hide />
          <YAxis hide />
          <Tooltip
            formatter={(value) => [formatCount(Number(value ?? 0)), '请求量']}
            labelFormatter={(label) => `日期 ${label}`}
          />
          <Area
            dataKey='requests'
            type='monotone'
            stroke='#2563eb'
            strokeWidth={1.8}
            fill='url(#controlTowerMiniTrend)'
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}

function firstModelName(models: string): string {
  const text = models.trim()
  if (!text) {
    return '全部模型'
  }
  return text.split(',')[0]?.trim() || text
}

function buildRuntimePolicies(
  providers: ProviderHealth[],
  requestTotal: number,
  generatedAt: number,
  range: { start: number; end: number }
): RoutingPolicyItem[] {
  const activeProviders = providers.filter((provider) => provider.status === 1)
  if (activeProviders.length === 0) {
    return []
  }
  const primary = activeProviders[0]
  const trafficPercent =
    requestTotal > 0 && primary.requests > 0
      ? Math.max(1, Math.round((primary.requests / requestTotal) * 100))
      : 100
  return [
    {
      id: 0,
      name: '默认渠道路由池',
      slice_key: 'global',
      model_name: firstModelName(primary.models || ''),
      sla_tier: 'default',
      track: 'runtime',
      action_type: 'runtime_pool',
      status: 'active',
      supplier_id: primary.supplier_id,
      supplier_name: primary.supplier_name,
      channel_id: primary.channel_id,
      channel_name: primary.channel_name,
      priority: 1,
      traffic_percent: trafficPercent,
      effective_from: range.start,
      effective_to: range.end,
      updated_at: generatedAt,
      reason: '由当前启用渠道和请求日志实时派生',
    },
  ]
}

function TopologyConnectors(props: { providerCount: number }) {
  const targets = [86, 142, 198, 254].slice(0, Math.max(props.providerCount, 1))
  const strokeForIndex = (index: number): string => {
    if (index === 0) {
      return '#22c55e'
    }
    if (index === 1) {
      return '#3b82f6'
    }
    return '#94a3b8'
  }
  return (
    <svg
      className='pointer-events-none absolute inset-0 z-0 hidden h-full w-full sm:block'
      viewBox='0 0 1000 320'
      preserveAspectRatio='none'
      aria-hidden='true'
    >
      {targets.map((target, index) => (
        <path
          key={target}
          d={`M520 160 C610 ${160 + (target - 160) * 0.3} 640 ${target} 720 ${target}`}
          fill='none'
          stroke={strokeForIndex(index)}
          strokeDasharray={index === 0 ? undefined : '7 7'}
          strokeLinecap='round'
          strokeWidth={index === 0 ? '2.2' : '1.8'}
        />
      ))}
    </svg>
  )
}

function buildRecentChanges(
  events: ControlTowerEvent[] | undefined,
  primaryPolicy: RoutingPolicyItem | undefined
): ControlTowerEvent[] {
  if (events && events.length > 0) {
    return events
  }
  if (!primaryPolicy) {
    return []
  }
  const isRuntimePolicy = primaryPolicy.id === 0
  return [
    {
      id: primaryPolicy.id,
      title: isRuntimePolicy ? '默认路由池运行中' : '路由策略已更新',
      detail: `${primaryPolicy.model_name || '全部模型'} · ${primaryPolicy.channel_name || '默认渠道'} · 流量 ${primaryPolicy.traffic_percent}%`,
      category: isRuntimePolicy ? 'runtime_pool' : 'routing_policy',
      severity: 'info',
      status: primaryPolicy.status,
      created_at: primaryPolicy.updated_at,
    },
  ]
}

export function ControlTower(props: {
  nav?: ReactNode
  onOpenRoutingPolicies?: () => void
}) {
  const queryClient = useQueryClient()
  const { range, rangeLabel } = useEnterpriseConsole()
  const query = useQuery({
    queryKey: ['enterprise-control-tower', range.start, range.end],
    queryFn: () =>
      getControlTower({
        start_timestamp: range.start,
        end_timestamp: range.end,
      }),
    refetchInterval: 60_000,
  })
  const data = query.data?.data
  const metrics = data?.metrics
  const requestTotal = metrics?.requests ?? 0
  const providers = useMemo(
    () =>
      [...(data?.provider_health ?? [])]
        .sort((a, b) => b.requests - a.requests)
        .slice(0, 4),
    [data?.provider_health]
  )
  const rawPolicies = data?.policies ?? []
  const hasActiveRawPolicy = rawPolicies.some(
    (policy) => policy.status === 'active'
  )
  const runtimePolicies = useMemo(
    () =>
      buildRuntimePolicies(providers, requestTotal, data?.generated_at ?? 0, {
        start: range.start,
        end: range.end,
      }),
    [data?.generated_at, providers, range.end, range.start, requestTotal]
  )
  const policies = hasActiveRawPolicy
    ? rawPolicies
    : [...runtimePolicies, ...rawPolicies]
  const primaryPolicy =
    policies.find((policy) => policy.status === 'active') ?? policies[0]
  const backupProvider = providers[1]
  const activePolicyCount = Math.max(
    metrics?.active_policies ?? 0,
    policies.filter((policy) => policy.status === 'active').length
  )
  const recentChanges = buildRecentChanges(data?.recent_changes, primaryPolicy)
  const trend = (data?.trend ?? []).map((item) => ({
    ...item,
    date: dayjs.unix(item.timestamp).format('MM-DD'),
  }))
  const invalidateControlTower = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['enterprise-control-tower'] }),
      queryClient.invalidateQueries({
        queryKey: ['token-router', 'operating-insights'],
      }),
      queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-action-plans'],
      }),
      queryClient.invalidateQueries({
        queryKey: ['token-router', 'supply-routing-policies'],
      }),
      queryClient.invalidateQueries({
        queryKey: ['token-router', 'routing-source-executions'],
      }),
    ])
  }
  const acknowledgeRisk = useMutation({
    mutationFn: async (id: number) => {
      const result = await acknowledgeOperatingInsight(id, {
        review_note: 'acknowledged from enterprise control tower',
      })
      if (!result.success) {
        throw new Error(result.message || '确认风险失败')
      }
      return result
    },
    onSuccess: async () => {
      toast.success('风险已确认')
      await invalidateControlTower()
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '请求失败')
    },
  })
  const dismissRisk = useMutation({
    mutationFn: async (id: number) => {
      const result = await dismissOperatingInsight(id, {
        review_note: 'dismissed from enterprise control tower',
      })
      if (!result.success) {
        throw new Error(result.message || '忽略风险失败')
      }
      return result
    },
    onSuccess: async () => {
      toast.success('风险已忽略')
      await invalidateControlTower()
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '请求失败')
    },
  })
  const updateActionStatus = useMutation({
    mutationFn: async (event: ControlTowerEvent) => {
      const nextStatus =
        event.status === 'in_progress' ? 'completed' : 'in_progress'
      const result = await updateSupplyActionPlanStatus(event.id, {
        status: nextStatus,
        operator_note: `updated from enterprise control tower: ${nextStatus}`,
      })
      if (!result.success) {
        throw new Error(result.message || '更新待执行动作失败')
      }
      return result
    },
    onSuccess: async () => {
      toast.success('待执行动作已更新')
      await invalidateControlTower()
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '请求失败')
    },
  })
  const disablePolicy = useMutation({
    mutationFn: async (id: number) => {
      const result = await disableSupplyRoutingPolicy(id, {
        operator_note: 'disabled from enterprise control tower',
      })
      if (!result.success) {
        throw new Error(result.message || '停用路由策略失败')
      }
      return result
    },
    onSuccess: async () => {
      toast.success('路由策略已停用')
      await invalidateControlTower()
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : '请求失败')
    },
  })

  return (
    <div className='flex flex-col gap-2 pb-4'>
      <EnterprisePageHeader
        title='Token Router 控制塔'
        description='路由策略、流量分配、SLA 与执行治理'
        actions={
          <div className='flex flex-wrap items-center justify-end gap-2'>
            <Button
              variant='outline'
              size='sm'
              onClick={props.onOpenRoutingPolicies}
              disabled={props.onOpenRoutingPolicies == null}
            >
              <FileText className='size-3.5' />
              策略模板
            </Button>
            <Button
              variant='outline'
              size='sm'
              onClick={() => exportControlTowerCsv(data)}
            >
              <Download className='size-3.5' />
              导出
            </Button>
            <Button
              variant='outline'
              size='icon-sm'
              aria-label='刷新控制塔数据'
              onClick={() => query.refetch()}
              disabled={query.isFetching}
            >
              <RefreshCw
                className={cn('size-3.5', query.isFetching && 'animate-spin')}
              />
            </Button>
            <Button variant='outline' size='icon-sm' aria-label='更多操作'>
              <MoreHorizontal className='size-3.5' />
            </Button>
          </div>
        }
      />

      {query.isError && (
        <div className='rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-xs text-rose-700'>
          控制塔数据加载失败，请刷新后重试。
        </div>
      )}

      <div className='grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-5'>
        <MetricTile
          title='活跃路由策略'
          value={formatCount(activePolicyCount)}
          helper={hasActiveRawPolicy ? rangeLabel : '默认路由池'}
          icon={Route}
          tone='blue'
          loading={query.isLoading}
        />
        <MetricTile
          title='实时成功率'
          value={formatPercent(metrics?.realtime_success_rate ?? 0)}
          helper='按请求日志聚合'
          icon={ShieldCheck}
          tone='emerald'
          loading={query.isLoading}
        />
        <MetricTile
          title='平均延迟'
          value={formatLatency(metrics?.average_latency_ms ?? 0)}
          helper='成功请求平均值'
          icon={Clock3}
          tone='violet'
          loading={query.isLoading}
        />
        <MetricTile
          title='自动切换次数'
          value={formatCount(metrics?.automatic_switches ?? 0)}
          helper='策略激活记录'
          icon={Zap}
          tone='orange'
          loading={query.isLoading}
        />
        <MetricTile
          title='待审批策略'
          value={formatCount(metrics?.pending_approvals ?? 0)}
          helper='定价、评估与决策'
          icon={CheckCircle2}
          tone='rose'
          loading={query.isLoading}
        />
      </div>

      {props.nav != null && <div className='min-w-0'>{props.nav}</div>}

      <div className='grid grid-cols-1 items-stretch gap-2 xl:grid-cols-[minmax(0,1fr)_360px]'>
        <div className='grid min-w-0 gap-2'>
          <EnterprisePanel
            title='路由拓扑与流量路径'
            description='请求从租户入口进入策略匹配，再按健康、区域、SLA 与权重分配到渠道。'
            action={
              <div className='flex items-center gap-2 text-[11px] text-slate-500'>
                <span className='inline-flex items-center gap-1'>
                  <span className='size-1.5 rounded-full bg-blue-500' />
                  实时流量
                </span>
                <span className='inline-flex items-center gap-1'>
                  <span className='size-1.5 rounded-full bg-emerald-500' />
                  健康良好
                </span>
                <span className='inline-flex items-center gap-1'>
                  <span className='size-1.5 rounded-full bg-amber-400' />
                  需关注
                </span>
              </div>
            }
            bodyClassName='p-3'
          >
            <div className='grid gap-3 lg:grid-cols-[minmax(0,1.35fr)_minmax(286px,0.72fr)]'>
              <div className='relative grid min-h-[300px] gap-2 overflow-hidden sm:grid-cols-[164px_1fr_1.2fr]'>
                <TopologyConnectors providerCount={providers.length} />
                <div className='relative z-10 self-center rounded-md border border-slate-200 bg-white p-3 shadow-[0_1px_1px_rgb(15_23_42/0.03)]'>
                  <div className='flex items-center gap-2'>
                    <span className='flex size-7 items-center justify-center rounded-md bg-blue-50 text-blue-600'>
                      <Network className='size-4' />
                    </span>
                    <div>
                      <p className='text-xs font-semibold text-slate-900'>
                        客户请求流量
                      </p>
                      <p className='text-[11px] text-slate-500'>API / 控制台</p>
                    </div>
                  </div>
                  <p className='mt-3 text-2xl leading-7 font-semibold text-slate-950'>
                    {formatCount(metrics?.requests ?? 0)}
                  </p>
                  <div className='mt-2 grid grid-cols-2 gap-1 text-[11px]'>
                    <div className='rounded bg-slate-50 px-2 py-1.5'>
                      <p className='text-slate-400'>Tokens</p>
                      <p className='font-semibold text-slate-800'>
                        {formatCount(metrics?.tokens ?? 0)}
                      </p>
                    </div>
                    <div className='rounded bg-slate-50 px-2 py-1.5'>
                      <p className='text-slate-400'>成功率</p>
                      <p className='font-semibold text-slate-800'>
                        {formatPercent(metrics?.realtime_success_rate ?? 0)}
                      </p>
                    </div>
                  </div>
                  <div className='mt-2'>
                    <TrendMiniChart trend={trend} />
                  </div>
                </div>

                <div className='relative z-10 flex items-center justify-center px-1'>
                  <div className='hidden h-px flex-1 border-t border-dashed border-emerald-400 sm:block' />
                  <div className='w-full rounded-md border border-blue-100 bg-blue-50/60 p-3 shadow-[0_1px_1px_rgb(15_23_42/0.03)]'>
                    <div className='flex items-start justify-between gap-2'>
                      <span className='flex size-7 items-center justify-center rounded-md bg-blue-600 text-white'>
                        <GitBranch className='size-4' />
                      </span>
                      <Badge className='h-5 rounded bg-emerald-50 px-2 text-[10px] text-emerald-600 ring-1 ring-emerald-100'>
                        运行中
                      </Badge>
                    </div>
                    <p className='mt-2 truncate text-xs font-semibold text-slate-900'>
                      智能路由策略
                    </p>
                    <p className='mt-0.5 truncate text-[11px] text-slate-500'>
                      {primaryPolicy?.name || '暂无活跃策略'}
                    </p>
                    <div className='mt-3 grid grid-cols-2 gap-1.5 text-[11px]'>
                      <div className='rounded border border-blue-100 bg-white/80 px-2 py-1.5'>
                        <p className='text-slate-400'>命中率</p>
                        <p className='font-semibold text-slate-800'>
                          {primaryPolicy
                            ? `${primaryPolicy.traffic_percent}%`
                            : '0%'}
                        </p>
                      </div>
                      <div className='rounded border border-blue-100 bg-white/80 px-2 py-1.5'>
                        <p className='text-slate-400'>成功率</p>
                        <p className='font-semibold text-slate-800'>
                          {formatPercent(metrics?.realtime_success_rate ?? 0)}
                        </p>
                      </div>
                    </div>
                    <div className='mt-2 rounded border border-slate-200 bg-white/80 px-2 py-1.5 text-[11px] text-slate-500'>
                      优先级 {primaryPolicy?.priority ?? '-'} ·{' '}
                      {primaryPolicy?.track || '默认轨道'}
                    </div>
                  </div>
                  <div className='hidden h-px flex-1 border-t border-dashed border-blue-400 sm:block' />
                </div>

                <div className='relative z-10 space-y-2 self-center'>
                  {providers.length === 0 ? (
                    <div className='flex min-h-36 items-center justify-center rounded-md border border-dashed border-slate-200 bg-slate-50/60 text-xs text-slate-500'>
                      暂无供应商流量数据
                    </div>
                  ) : (
                    providers.map((provider, index) => (
                      <ProviderRouteNode
                        key={provider.channel_id}
                        provider={provider}
                        requestTotal={requestTotal}
                        index={index}
                      />
                    ))
                  )}
                  <div className='rounded-md border border-dashed border-slate-200 bg-slate-50/70 px-3 py-2 text-[11px] text-slate-500'>
                    全局兜底策略：
                    {backupProvider?.channel_name || '未配置备用路由'}
                  </div>
                </div>
              </div>

              <div className='rounded-md border border-slate-200 bg-white'>
                <div className='flex items-center justify-between border-b border-slate-100 px-3 py-2'>
                  <div>
                    <p className='text-xs font-semibold text-slate-900'>
                      路由规则
                    </p>
                    <p className='text-[11px] text-slate-500'>
                      {primaryPolicy?.name || '当前无策略'}
                    </p>
                  </div>
                  <Button
                    variant='ghost'
                    size='xs'
                    onClick={props.onOpenRoutingPolicies}
                    disabled={props.onOpenRoutingPolicies == null}
                  >
                    编辑策略
                  </Button>
                </div>
                <RuleItem
                  icon={SlidersHorizontal}
                  title='加权路由'
                  detail={
                    primaryPolicy
                      ? `${primaryPolicy.channel_name || '默认渠道'} · 流量 ${primaryPolicy.traffic_percent}% · ${primaryPolicy.model_name || '全部模型'}`
                      : '暂无可用策略，请在路由策略页签创建。'
                  }
                />
                <RuleItem
                  icon={Zap}
                  title='降级策略'
                  detail={`待执行动作 ${data?.pending_actions.length ?? 0} 条，异常风险 ${data?.risks.length ?? 0} 条。`}
                />
                <RuleItem
                  icon={Target}
                  title='区域感知'
                  detail={
                    providers
                      .map((provider) => provider.region)
                      .filter(Boolean)
                      .slice(0, 3)
                      .join(' / ') || '当前渠道未配置区域分组。'
                  }
                />
                <RuleItem
                  icon={ShieldCheck}
                  title='SLA 阈值'
                  detail={`当前成功率 ${formatPercent(metrics?.realtime_success_rate ?? 0)}，平均延迟 ${formatLatency(metrics?.average_latency_ms ?? 0)}。`}
                />
              </div>
            </div>
            <div className='mt-2 flex items-center justify-between text-[11px] text-slate-500'>
              <span>
                最后更新时间：
                {data?.generated_at
                  ? dayjs.unix(data.generated_at).format('HH:mm:ss')
                  : '-'}
              </span>
              <button
                type='button'
                className='font-medium text-blue-600 hover:underline'
                onClick={props.onOpenRoutingPolicies}
              >
                查看策略拓扑
              </button>
            </div>
          </EnterprisePanel>

          <EnterprisePanel
            title='活跃路由策略'
            description={`共 ${policies.length} 条策略，当前展示前 8 条`}
            bodyClassName='p-0'
          >
            <Table className='text-xs [&_td]:h-9 [&_td]:py-1.5 [&_td]:text-xs [&_td_*]:text-xs [&_th]:h-8 [&_th]:text-xs [&_th_*]:text-xs'>
              <TableHeader className='bg-slate-50'>
                <TableRow>
                  <TableHead>策略名</TableHead>
                  <TableHead>作用对象</TableHead>
                  <TableHead>命中率</TableHead>
                  <TableHead>当前主路由</TableHead>
                  <TableHead>备用路由</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>审批状态</TableHead>
                  <TableHead>更新时间</TableHead>
                  <TableHead className='text-right'>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {policies.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={9}
                      className='h-24 text-center text-xs text-slate-500'
                    >
                      暂无路由策略，可在“路由策略”页签创建或激活策略。
                    </TableCell>
                  </TableRow>
                ) : (
                  policies.slice(0, 8).map((policy, index) => {
                    const status = policyStatus(policy)
                    const approval = approvalStatus(policy)
                    return (
                      <TableRow key={policy.id || `runtime-${policy.channel_id}`}>
                        <TableCell>
                          <div className='max-w-[190px]'>
                            <p className='truncate font-semibold text-slate-900'>
                              {policy.name}
                            </p>
                            <p className='truncate text-[11px] text-slate-500'>
                              {policy.track || '默认轨道'}
                            </p>
                          </div>
                        </TableCell>
                        <TableCell>
                          <p className='font-medium text-slate-800'>
                            {policy.model_name || '全部模型'}
                          </p>
                          <p className='text-[11px] text-slate-500'>
                            {policy.slice_key || '全局'}
                          </p>
                        </TableCell>
                        <TableCell>
                          <span className='font-semibold text-slate-900'>
                            {policy.traffic_percent}%
                          </span>
                        </TableCell>
                        <TableCell>
                          {policy.channel_name || `#${policy.channel_id}`}
                        </TableCell>
                        <TableCell>
                          {providers[index + 1]?.channel_name ||
                            backupProvider?.channel_name ||
                            '-'}
                        </TableCell>
                        <TableCell>
                          <Badge
                            className={cn(
                              'h-5 rounded px-2 text-[10px]',
                              status.className
                            )}
                          >
                            {status.label}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Badge
                            className={cn(
                              'h-5 rounded px-2 text-[10px]',
                              approval.className
                            )}
                          >
                            {approval.label}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          {policy.updated_at > 0
                            ? dayjs
                                .unix(policy.updated_at)
                                .format('MM-DD HH:mm')
                            : '-'}
                        </TableCell>
                        <TableCell className='text-right'>
                          <Button
                            variant='ghost'
                            size='xs'
                            disabled={
                              policy.id <= 0 ||
                              policy.status !== 'active' ||
                              disablePolicy.isPending
                            }
                            onClick={() => disablePolicy.mutate(policy.id)}
                          >
                            停用
                          </Button>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </EnterprisePanel>
        </div>

        <aside className='grid h-full min-w-0 content-start gap-2'>
          <EnterprisePanel
            title='实时健康状态'
            description='按当前请求量排序'
            action={
              <Button variant='ghost' size='xs' onClick={() => query.refetch()}>
                查看全部
              </Button>
            }
            bodyClassName='p-2.5'
          >
            <div className='space-y-0.5'>
              <div className='grid grid-cols-[minmax(0,1fr)_54px_48px_54px] gap-2 px-1.5 pb-1 text-right text-[10px] text-slate-400'>
                <span className='text-left'>供应商</span>
                <span>成功率</span>
                <span>请求</span>
                <span>延迟</span>
              </div>
              {providers.length === 0 ? (
                <div className='rounded-md border border-dashed border-slate-200 bg-slate-50/60 py-6 text-center text-xs text-slate-500'>
                  暂无健康数据
                </div>
              ) : (
                providers.map((provider) => (
                  <HealthRow key={provider.channel_id} provider={provider} />
                ))
              )}
            </div>
          </EnterprisePanel>

          <EnterprisePanel
            title='最近策略变更'
            action={
              <Button
                variant='ghost'
                size='xs'
                onClick={props.onOpenRoutingPolicies}
              >
                查看全部
              </Button>
            }
            bodyClassName='p-2'
          >
            <EventList
              events={recentChanges}
              emptyText='暂无策略变更记录'
              maxItems={3}
              renderAction={(event) =>
                event.status === 'active' && event.id > 0 ? (
                  <Button
                    size='xs'
                    variant='outline'
                    disabled={disablePolicy.isPending}
                    onClick={() => disablePolicy.mutate(event.id)}
                  >
                    停用
                  </Button>
                ) : null
              }
            />
          </EnterprisePanel>

          <EnterprisePanel
            title='待执行动作'
            action={
              <Badge className='h-5 rounded bg-rose-600 px-2 text-white'>
                {data?.pending_actions.length ?? 0}
              </Badge>
            }
            bodyClassName='p-2'
          >
            <EventList
              events={data?.pending_actions ?? []}
              emptyText='当前没有待执行动作'
              maxItems={3}
              renderAction={(event) => (
                <Button
                  size='xs'
                  variant='outline'
                  disabled={updateActionStatus.isPending}
                  onClick={() => updateActionStatus.mutate(event)}
                >
                  {event.status === 'in_progress' ? '标记完成' : '开始执行'}
                </Button>
              )}
            />
          </EnterprisePanel>

          <EnterprisePanel
            title='风险提醒'
            action={
              <Button variant='ghost' size='xs'>
                查看全部
              </Button>
            }
            bodyClassName='p-2'
          >
            <EventList
              events={data?.risks ?? []}
              emptyText='当前没有未处理风险'
              maxItems={3}
              renderAction={(event) => (
                <>
                  <Button
                    size='xs'
                    variant='outline'
                    disabled={
                      acknowledgeRisk.isPending || dismissRisk.isPending
                    }
                    onClick={() => acknowledgeRisk.mutate(event.id)}
                  >
                    确认
                  </Button>
                  <Button
                    size='xs'
                    variant='ghost'
                    disabled={
                      acknowledgeRisk.isPending || dismissRisk.isPending
                    }
                    onClick={() => dismissRisk.mutate(event.id)}
                  >
                    忽略
                  </Button>
                </>
              )}
            />
          </EnterprisePanel>
        </aside>
      </div>
    </div>
  )
}
