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
  ArrowRight,
  BadgeDollarSign,
  Boxes,
  Building2,
  CircleDollarSign,
  Clock3,
  Gauge,
  KeyRound,
  Network,
  RefreshCw,
  Route,
  ShieldCheck,
  Sparkles,
  Users,
  WalletCards,
} from 'lucide-react'
import { useMemo } from 'react'
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip as ChartTooltip,
  XAxis,
  YAxis,
} from 'recharts'

import {
  EnterprisePageHeader,
  EnterprisePanel,
  EnterpriseStatCard,
} from '@/components/enterprise'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  formatCompactNumber,
  formatCurrencyUSD,
  formatNumber,
  formatTokens,
} from '@/lib/format'
import { cn } from '@/lib/utils'

import { getEnterpriseOverview } from './api'
import type {
  EnterpriseOverviewData,
  EnterpriseOverviewInsight,
  EnterpriseOverviewRankingItem,
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

function formatPercentage(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    style: 'percent',
    maximumFractionDigits: 2,
  }).format(Number.isFinite(value) ? value : 0)
}

function formatDate(timestamp: number): string {
  if (!timestamp) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
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

function severityMeta(severity: string) {
  if (severity === 'action') {
    return {
      label: '需处理',
      className: 'border-rose-500/20 bg-rose-500/10 text-rose-600',
      dot: 'bg-rose-500',
    }
  }
  if (severity === 'watch') {
    return {
      label: '需关注',
      className: 'border-amber-500/20 bg-amber-500/10 text-amber-600',
      dot: 'bg-amber-500',
    }
  }
  return {
    label: '信息',
    className: 'border-blue-500/20 bg-blue-500/10 text-blue-600',
    dot: 'bg-blue-500',
  }
}

function RankingList({
  items,
  valueLabel,
}: {
  items: EnterpriseOverviewRankingItem[]
  valueLabel: (item: EnterpriseOverviewRankingItem) => string
}) {
  if (items.length === 0) {
    return (
      <div className='text-muted-foreground flex min-h-48 items-center justify-center text-sm'>
        当前时间范围内暂无统计数据
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      {items.map((item, index) => (
        <div
          key={`${item.name}-${item.requests}-${item.tokens}-${item.quota}`}
          className='space-y-2'
        >
          <div className='flex items-center justify-between gap-3 text-xs'>
            <div className='flex min-w-0 items-center gap-2.5'>
              <span className='bg-muted text-muted-foreground flex size-6 shrink-0 items-center justify-center rounded-lg text-[10px] font-semibold'>
                {index + 1}
              </span>
              <span className='text-foreground truncate font-medium'>
                {item.name}
              </span>
            </div>
            <span className='text-foreground/80 shrink-0 font-semibold tabular-nums'>
              {valueLabel(item)}
            </span>
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

function InsightItem({ insight }: { insight: EnterpriseOverviewInsight }) {
  const meta = severityMeta(insight.severity)
  return (
    <article className='border-border/70 bg-background/60 hover:border-primary/20 hover:bg-background rounded-xl border p-3.5 transition-colors'>
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0'>
          <div className='flex items-center gap-2'>
            <span className={cn('size-1.5 shrink-0 rounded-full', meta.dot)} />
            <p className='truncate text-sm font-semibold'>{insight.title}</p>
          </div>
          <p className='text-muted-foreground mt-1.5 line-clamp-2 text-xs leading-5'>
            {insight.summary ||
              insight.recommended_action ||
              '等待运营团队确认处理方案'}
          </p>
        </div>
        <Badge
          variant='outline'
          className={cn('shrink-0 text-[10px]', meta.className)}
        >
          {meta.label}
        </Badge>
      </div>
      <div className='text-muted-foreground mt-3 flex items-center justify-between gap-2 text-[10px]'>
        <span className='truncate'>{insight.model_name || '全局范围'}</span>
        <span className='shrink-0'>{formatDateTime(insight.generated_at)}</span>
      </div>
    </article>
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
  const chartData = overview.trend.map((item) => ({
    ...item,
    label: formatDate(item.timestamp),
  }))
  const dateRangeLabel = `${formatDate(overview.range.start_timestamp || range.start)} - ${formatDate(overview.range.end_timestamp || range.end)}`
  const channelHealth =
    metrics.total_channels > 0
      ? metrics.healthy_channels / metrics.total_channels
      : 0

  return (
    <div className='enterprise-dashboard space-y-4 pb-2 sm:space-y-5'>
      <EnterprisePageHeader
        eyebrow='企业工作区'
        title='企业总览'
        description='AI 网关与 Token Router 经营驾驶舱，统一查看用量、成本、渠道健康度、路由治理和待办风险。'
        actions={
          <>
            <Badge
              variant='outline'
              className='bg-background/70 h-8 gap-1.5 rounded-lg px-3 text-xs font-normal'
            >
              <span className='size-1.5 rounded-full bg-emerald-500' />
              生产环境
            </Badge>
            <Badge
              variant='outline'
              className='bg-background/70 h-8 rounded-lg px-3 text-xs font-normal'
            >
              近 7 天 · {dateRangeLabel}
            </Badge>
            <Button
              variant='outline'
              size='sm'
              className='bg-background/70 h-8 rounded-lg'
              disabled={overviewQuery.isFetching}
              onClick={() => void overviewQuery.refetch()}
            >
              <RefreshCw
                className={cn(
                  'size-3.5',
                  overviewQuery.isFetching && 'animate-spin'
                )}
              />
              刷新数据
            </Button>
          </>
        }
      />

      {overviewQuery.isError && (
        <div className='flex items-center gap-2 rounded-xl border border-amber-500/25 bg-amber-500/8 px-4 py-3 text-xs text-amber-700 dark:text-amber-300'>
          <AlertTriangle className='size-4 shrink-0' />
          企业聚合接口暂时不可用，请确认后端已更新并完成数据库迁移。其余管理页面不受影响。
        </div>
      )}

      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-6'>
        <EnterpriseStatCard
          title='请求总量'
          value={formatCompactNumber(metrics.total_requests)}
          helper='近 7 天真实调用'
          icon={Activity}
          tone='blue'
          loading={overviewQuery.isLoading}
        />
        <EnterpriseStatCard
          title='调用成功率'
          value={formatPercentage(metrics.success_rate)}
          helper='消费日志口径'
          trend={metrics.success_rate >= 0.99 ? '运行健康' : '建议关注'}
          trendTone={metrics.success_rate >= 0.99 ? 'positive' : 'negative'}
          icon={ShieldCheck}
          tone='emerald'
          loading={overviewQuery.isLoading}
        />
        <EnterpriseStatCard
          title='平均响应耗时'
          value={`${formatNumber(metrics.average_latency_ms)} ms`}
          helper='成功请求平均值'
          icon={Clock3}
          tone='violet'
          loading={overviewQuery.isLoading}
        />
        <EnterpriseStatCard
          title='本期调用成本'
          value={formatCurrencyUSD(metrics.estimated_cost)}
          helper='按系统额度换算'
          icon={CircleDollarSign}
          tone='amber'
          loading={overviewQuery.isLoading}
        />
        <EnterpriseStatCard
          title='活跃企业用户'
          value={formatNumber(metrics.active_users)}
          helper={`共 ${formatNumber(metrics.total_users)} 个账户`}
          icon={Users}
          tone='blue'
          loading={overviewQuery.isLoading}
        />
        <EnterpriseStatCard
          title='渠道健康度'
          value={formatPercentage(channelHealth)}
          helper={`${metrics.healthy_channels}/${metrics.total_channels} 个渠道可用`}
          trend={
            metrics.low_balance_channels > 0
              ? `${metrics.low_balance_channels} 个低余额`
              : '余额正常'
          }
          trendTone={metrics.low_balance_channels > 0 ? 'negative' : 'positive'}
          icon={Network}
          tone={metrics.low_balance_channels > 0 ? 'rose' : 'emerald'}
          loading={overviewQuery.isLoading}
        />
      </div>

      <div className='grid gap-4 2xl:grid-cols-[minmax(0,1.7fr)_minmax(330px,0.8fr)]'>
        <EnterprisePanel
          title='请求与 Token 趋势'
          description='按天汇总实际网关调用，数据来自 quota_data 聚合表。'
          action={
            <div className='text-muted-foreground flex items-center gap-3 text-[11px]'>
              <span className='flex items-center gap-1.5'>
                <span className='size-2 rounded-full bg-blue-500' /> 请求量
              </span>
              <span className='flex items-center gap-1.5'>
                <span className='size-2 rounded-full bg-violet-500' /> Tokens
              </span>
            </div>
          }
          bodyClassName='h-[330px] px-2 pb-2 pt-4 sm:px-3'
        >
          {chartData.length > 0 ? (
            <ResponsiveContainer width='100%' height='100%'>
              <AreaChart
                data={chartData}
                margin={{ top: 12, right: 14, left: 4, bottom: 4 }}
              >
                <defs>
                  <linearGradient
                    id='enterpriseRequestsGradient'
                    x1='0'
                    y1='0'
                    x2='0'
                    y2='1'
                  >
                    <stop offset='5%' stopColor='#3b82f6' stopOpacity={0.28} />
                    <stop
                      offset='95%'
                      stopColor='#3b82f6'
                      stopOpacity={0.015}
                    />
                  </linearGradient>
                  <linearGradient
                    id='enterpriseTokensGradient'
                    x1='0'
                    y1='0'
                    x2='0'
                    y2='1'
                  >
                    <stop offset='5%' stopColor='#8b5cf6' stopOpacity={0.18} />
                    <stop offset='95%' stopColor='#8b5cf6' stopOpacity={0.01} />
                  </linearGradient>
                </defs>
                <CartesianGrid
                  vertical={false}
                  stroke='var(--border)'
                  strokeOpacity={0.75}
                />
                <XAxis
                  dataKey='label'
                  axisLine={false}
                  tickLine={false}
                  tick={{ fill: 'var(--muted-foreground)', fontSize: 11 }}
                  dy={8}
                />
                <YAxis
                  yAxisId='requests'
                  axisLine={false}
                  tickLine={false}
                  width={48}
                  tickFormatter={(value) => formatCompactNumber(Number(value))}
                  tick={{ fill: 'var(--muted-foreground)', fontSize: 11 }}
                />
                <YAxis
                  yAxisId='tokens'
                  orientation='right'
                  axisLine={false}
                  tickLine={false}
                  width={55}
                  tickFormatter={(value) => formatCompactNumber(Number(value))}
                  tick={{ fill: 'var(--muted-foreground)', fontSize: 11 }}
                />
                <ChartTooltip
                  cursor={{ stroke: 'var(--border)', strokeDasharray: '4 4' }}
                  contentStyle={{
                    borderRadius: 12,
                    border: '1px solid var(--border)',
                    background: 'var(--popover)',
                    color: 'var(--popover-foreground)',
                    boxShadow: '0 14px 35px rgb(15 23 42 / 0.12)',
                    fontSize: 12,
                  }}
                  formatter={(value, name) => [
                    name === 'Tokens'
                      ? formatTokens(Number(value))
                      : formatNumber(Number(value)),
                    name,
                  ]}
                />
                <Area
                  yAxisId='requests'
                  type='monotone'
                  dataKey='requests'
                  name='请求量'
                  stroke='#3b82f6'
                  strokeWidth={2.2}
                  fill='url(#enterpriseRequestsGradient)'
                />
                <Area
                  yAxisId='tokens'
                  type='monotone'
                  dataKey='tokens'
                  name='Tokens'
                  stroke='#8b5cf6'
                  strokeWidth={1.8}
                  fill='url(#enterpriseTokensGradient)'
                />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className='flex h-full flex-col items-center justify-center text-center'>
              <Activity className='text-muted-foreground/40 mb-3 size-8' />
              <p className='text-sm font-medium'>暂无趋势数据</p>
              <p className='text-muted-foreground mt-1 text-xs'>
                网关产生调用后，此处会自动展示真实请求趋势。
              </p>
            </div>
          )}
        </EnterprisePanel>

        <EnterprisePanel
          title='运营风险与待办'
          description={`${metrics.open_insights} 条运营洞察，${metrics.pending_approvals} 项待审批。`}
          action={
            <Button
              variant='ghost'
              size='sm'
              className='h-7 px-2 text-xs'
              render={<Link to='/token-router' />}
            >
              进入控制塔
              <ArrowRight className='size-3.5' />
            </Button>
          }
          bodyClassName='space-y-2.5'
        >
          {overview.insights.length > 0 ? (
            overview.insights
              .slice(0, 4)
              .map((insight) => (
                <InsightItem key={insight.id} insight={insight} />
              ))
          ) : (
            <div className='flex min-h-56 flex-col items-center justify-center text-center'>
              <ShieldCheck className='mb-3 size-9 text-emerald-500/70' />
              <p className='text-sm font-medium'>暂无未处理风险</p>
              <p className='text-muted-foreground mt-1 max-w-56 text-xs leading-5'>
                生成 Operating Insights 后，风险和建议会在这里集中呈现。
              </p>
            </div>
          )}
        </EnterprisePanel>
      </div>

      <div className='grid gap-4 xl:grid-cols-2 2xl:grid-cols-4'>
        <EnterprisePanel
          title='热门模型'
          description='按真实请求量排序'
          action={<Boxes className='text-muted-foreground size-4' />}
        >
          <RankingList
            items={overview.top_models}
            valueLabel={(item) => formatCompactNumber(item.requests)}
          />
        </EnterprisePanel>

        <EnterprisePanel
          title='客户 / 用户用量'
          description='按账户调用量排序'
          action={<Building2 className='text-muted-foreground size-4' />}
        >
          <RankingList
            items={overview.top_users}
            valueLabel={(item) => formatCompactNumber(item.requests)}
          />
        </EnterprisePanel>

        <EnterprisePanel
          title='治理资产'
          description='企业管理能力的实时覆盖情况'
          bodyClassName='grid grid-cols-2 gap-3'
        >
          {[
            {
              label: '活跃 API Keys',
              value: metrics.active_api_keys,
              icon: KeyRound,
              tone: 'text-blue-600 bg-blue-500/10',
            },
            {
              label: '供应商',
              value: `${metrics.healthy_suppliers}/${metrics.total_suppliers}`,
              icon: Building2,
              tone: 'text-emerald-600 bg-emerald-500/10',
            },
            {
              label: '生效路由策略',
              value: metrics.active_policies,
              icon: Route,
              tone: 'text-violet-600 bg-violet-500/10',
            },
            {
              label: '待审批事项',
              value: metrics.pending_approvals,
              icon: Sparkles,
              tone: 'text-amber-600 bg-amber-500/10',
            },
          ].map((item) => {
            const Icon = item.icon
            return (
              <div
                key={item.label}
                className='bg-background/55 rounded-xl border p-3.5'
              >
                <span
                  className={cn(
                    'flex size-8 items-center justify-center rounded-lg',
                    item.tone
                  )}
                >
                  <Icon className='size-4' />
                </span>
                <p className='mt-3 text-xl font-semibold tracking-tight'>
                  {item.value}
                </p>
                <p className='text-muted-foreground mt-0.5 text-[11px]'>
                  {item.label}
                </p>
              </div>
            )
          })}
        </EnterprisePanel>

        <EnterprisePanel
          title='经营指标'
          description='基于定价建议与额度口径聚合'
          bodyClassName='space-y-4'
        >
          <div className='bg-background/55 flex items-center justify-between rounded-xl border p-3.5'>
            <div>
              <p className='text-muted-foreground text-[11px]'>预估毛利</p>
              <p className='mt-1 text-xl font-semibold'>
                {formatCurrencyUSD(metrics.estimated_gross_profit)}
              </p>
            </div>
            <span className='flex size-9 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-600'>
              <BadgeDollarSign className='size-4.5' />
            </span>
          </div>
          <div className='bg-background/55 flex items-center justify-between rounded-xl border p-3.5'>
            <div>
              <p className='text-muted-foreground text-[11px]'>毛利率</p>
              <p className='mt-1 text-xl font-semibold'>
                {formatPercentage(metrics.gross_margin_rate)}
              </p>
            </div>
            <span className='flex size-9 items-center justify-center rounded-xl bg-violet-500/10 text-violet-600'>
              <Gauge className='size-4.5' />
            </span>
          </div>
          <div className='text-muted-foreground rounded-xl border border-dashed px-3.5 py-3 text-xs leading-5'>
            定价建议尚未生成时，毛利指标会显示为 0；不影响调用成本与用量统计。
          </div>
        </EnterprisePanel>
      </div>

      <EnterprisePanel
        title='核心渠道运行状态'
        description='按累计消耗排序，帮助运营团队优先关注主力渠道。'
        action={
          <Button
            variant='ghost'
            size='sm'
            className='h-7 px-2 text-xs'
            render={<Link to='/channels' />}
          >
            管理全部渠道
            <ArrowRight className='size-3.5' />
          </Button>
        }
        bodyClassName='overflow-x-auto p-0'
      >
        <table className='w-full min-w-[760px] text-left text-xs'>
          <thead className='bg-muted/35 text-muted-foreground border-b text-[11px] font-medium'>
            <tr>
              <th className='px-5 py-3 font-medium'>渠道</th>
              <th className='px-4 py-3 font-medium'>分组</th>
              <th className='px-4 py-3 font-medium'>响应耗时</th>
              <th className='px-4 py-3 font-medium'>余额</th>
              <th className='px-4 py-3 font-medium'>累计消耗</th>
              <th className='px-4 py-3 font-medium'>模型覆盖</th>
              <th className='px-5 py-3 text-right font-medium'>状态</th>
            </tr>
          </thead>
          <tbody className='divide-border/65 divide-y'>
            {overview.channels.length > 0 ? (
              overview.channels.map((channel) => {
                const modelCount = channel.models
                  ? channel.models.split(',').filter(Boolean).length
                  : 0
                return (
                  <tr
                    key={channel.id}
                    className='hover:bg-muted/25 transition-colors'
                  >
                    <td className='px-5 py-3.5'>
                      <div className='flex items-center gap-2.5'>
                        <span className='flex size-8 items-center justify-center rounded-lg bg-blue-500/10 text-blue-600'>
                          <Network className='size-4' />
                        </span>
                        <div className='min-w-0'>
                          <p className='max-w-56 truncate font-medium'>
                            {channel.name}
                          </p>
                          <p className='text-muted-foreground text-[10px]'>
                            ID {channel.id}
                          </p>
                        </div>
                      </div>
                    </td>
                    <td className='text-muted-foreground px-4 py-3.5'>
                      {channel.group || '默认分组'}
                    </td>
                    <td className='px-4 py-3.5 font-medium tabular-nums'>
                      {channel.response_time > 0
                        ? `${channel.response_time} ms`
                        : '-'}
                    </td>
                    <td className='px-4 py-3.5 tabular-nums'>
                      {formatCurrencyUSD(channel.balance)}
                    </td>
                    <td className='px-4 py-3.5 tabular-nums'>
                      {formatCompactNumber(channel.used_quota)}
                    </td>
                    <td className='text-muted-foreground px-4 py-3.5'>
                      {modelCount > 0 ? `${modelCount} 个模型` : '未配置'}
                    </td>
                    <td className='px-5 py-3.5 text-right'>
                      <Badge
                        variant='outline'
                        className={cn(
                          'text-[10px]',
                          channel.status === 1
                            ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-600'
                            : 'border-rose-500/20 bg-rose-500/10 text-rose-600'
                        )}
                      >
                        {channel.status === 1 ? '运行中' : '不可用'}
                      </Badge>
                    </td>
                  </tr>
                )
              })
            ) : (
              <tr>
                <td
                  colSpan={7}
                  className='text-muted-foreground px-5 py-12 text-center text-sm'
                >
                  暂无渠道数据，请先在渠道与供应商中心完成上游接入。
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </EnterprisePanel>

      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
        {[
          {
            title: '创建企业 API Key',
            description: '为客户、团队或应用分配独立访问凭证',
            icon: KeyRound,
            to: '/keys' as const,
          },
          {
            title: '配置路由策略',
            description: '管理供应商权重、降级和 SLA 路由',
            icon: Route,
            to: '/token-router' as const,
          },
          {
            title: '查看用量明细',
            description: '追踪模型、客户、渠道与成本消耗',
            icon: WalletCards,
            to: '/usage-logs' as const,
          },
          {
            title: '管理组织成员',
            description: '维护企业用户、权限与账户额度',
            icon: Users,
            to: '/users' as const,
          },
        ].map((item) => {
          const Icon = item.icon
          return (
            <Link
              key={item.title}
              to={item.to}
              className='group bg-card/90 hover:border-primary/25 flex items-center gap-3 rounded-2xl border p-4 shadow-sm transition-all hover:-translate-y-0.5 hover:shadow-md'
            >
              <span className='bg-primary/8 text-primary group-hover:bg-primary group-hover:text-primary-foreground flex size-10 shrink-0 items-center justify-center rounded-xl transition-colors'>
                <Icon className='size-4.5' />
              </span>
              <span className='min-w-0 flex-1'>
                <span className='block truncate text-sm font-semibold'>
                  {item.title}
                </span>
                <span className='text-muted-foreground mt-0.5 block truncate text-[11px]'>
                  {item.description}
                </span>
              </span>
              <ArrowRight className='text-muted-foreground group-hover:text-primary size-4 shrink-0 transition-transform group-hover:translate-x-0.5' />
            </Link>
          )
        })}
      </div>
    </div>
  )
}
