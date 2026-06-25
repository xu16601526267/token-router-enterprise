import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  AlertTriangle,
  ArrowRight,
  Bot,
  CheckCircle2,
  Clock3,
  GitBranch,
  Network,
  RefreshCw,
  Route,
  ShieldCheck,
  Sparkles,
  Zap,
} from 'lucide-react'
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { useTranslation } from 'react-i18next'
import {
  EnterprisePageHeader,
  EnterprisePanel,
  EnterpriseStatCard,
} from '@/components/enterprise'
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
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'
import { getControlTower } from '../control-tower-api'
import type {
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

function eventTone(severity: string): string {
  if (severity === 'action' || severity === 'high' || severity === 'P1') {
    return 'bg-rose-500/10 text-rose-600 dark:text-rose-300'
  }
  if (severity === 'watch' || severity === 'medium' || severity === 'P2') {
    return 'bg-amber-500/10 text-amber-600 dark:text-amber-300'
  }
  return 'bg-blue-500/10 text-blue-600 dark:text-blue-300'
}

function providerStatus(provider: ProviderHealth): {
  label: string
  className: string
} {
  if (provider.status !== 1) {
    return {
      label: '已停用',
      className: 'bg-muted text-muted-foreground',
    }
  }
  if (provider.success_rate > 0 && provider.success_rate < 0.98) {
    return {
      label: '需关注',
      className: 'bg-amber-500/10 text-amber-600 dark:text-amber-300',
    }
  }
  return {
    label: '健康',
    className: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-300',
  }
}

function EventList(props: {
  events: ControlTowerEvent[]
  emptyText: string
}) {
  if (props.events.length === 0) {
    return (
      <div className='flex min-h-36 items-center justify-center text-sm text-muted-foreground'>
        {props.emptyText}
      </div>
    )
  }
  return (
    <div className='space-y-1'>
      {props.events.map((event) => (
        <div
          key={`${event.category}-${event.id}`}
          className='flex items-start gap-3 rounded-xl px-2.5 py-2.5 transition-colors hover:bg-muted/45'
        >
          <span
            className={cn(
              'mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-lg',
              eventTone(event.severity)
            )}
          >
            <AlertTriangle className='size-3.5' aria-hidden='true' />
          </span>
          <div className='min-w-0 flex-1'>
            <div className='flex items-start justify-between gap-2'>
              <p className='truncate text-sm font-medium'>{event.title}</p>
              <span className='shrink-0 text-[11px] text-muted-foreground'>
                {event.created_at > 0
                  ? dayjs.unix(event.created_at).format('MM-DD HH:mm')
                  : '—'}
              </span>
            </div>
            <p className='mt-0.5 line-clamp-2 text-xs leading-5 text-muted-foreground'>
              {event.detail || '暂无补充说明'}
            </p>
          </div>
        </div>
      ))}
    </div>
  )
}

function ProviderNode(props: { provider: ProviderHealth; index: number }) {
  const status = providerStatus(props.provider)
  const latency =
    props.provider.average_latency_ms || props.provider.response_time_ms
  return (
    <div className='relative flex items-center gap-3 rounded-xl border bg-background/85 p-3 shadow-sm'>
      <span
        className={cn(
          'absolute -left-1.5 top-1/2 size-3 -translate-y-1/2 rounded-full border-2 border-background',
          props.index === 0 ? 'bg-primary' : 'bg-muted-foreground/45'
        )}
      />
      <span className='flex size-9 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary'>
        <Bot className='size-4' aria-hidden='true' />
      </span>
      <div className='min-w-0 flex-1'>
        <div className='flex items-center gap-2'>
          <p className='truncate text-sm font-semibold'>
            {props.provider.channel_name}
          </p>
          <Badge className={cn('border-0', status.className)}>
            {status.label}
          </Badge>
        </div>
        <p className='mt-1 truncate text-[11px] text-muted-foreground'>
          {props.provider.supplier_name || '未绑定供应商'} ·{' '}
          {props.provider.region || '默认分组'}
        </p>
      </div>
      <div className='shrink-0 text-right text-[11px]'>
        <p className='font-semibold'>{formatPercent(props.provider.success_rate)}</p>
        <p className='mt-0.5 text-muted-foreground'>{latency.toFixed(0)} ms</p>
      </div>
    </div>
  )
}

function PolicyStatus(props: { policy: RoutingPolicyItem }) {
  const active = props.policy.status === 'active'
  return (
    <Badge
      className={cn(
        'border-0',
        active
          ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-300'
          : 'bg-muted text-muted-foreground'
      )}
    >
      {active ? '运行中' : '已停用'}
    </Badge>
  )
}

export function ControlTower() {
  const { t } = useTranslation()
  const endTimestamp = Math.floor(Date.now() / 1000)
  const startTimestamp = endTimestamp - 7 * 24 * 60 * 60
  const query = useQuery({
    queryKey: ['enterprise-control-tower', startTimestamp, endTimestamp],
    queryFn: () =>
      getControlTower({
        start_timestamp: startTimestamp,
        end_timestamp: endTimestamp,
      }),
    refetchInterval: 60_000,
  })
  const data = query.data?.data
  const metrics = data?.metrics
  const providers = useMemo(
    () =>
      [...(data?.provider_health ?? [])]
        .sort((a, b) => b.requests - a.requests)
        .slice(0, 4),
    [data?.provider_health]
  )
  const policies = data?.policies ?? []
  const primaryPolicy = policies.find((item) => item.status === 'active')
  const trend = (data?.trend ?? []).map((item) => ({
    ...item,
    date: dayjs.unix(item.timestamp).format('MM-DD'),
  }))

  return (
    <div className='flex flex-col gap-4 pb-5'>
      <EnterprisePageHeader
        eyebrow='企业级路由治理'
        title='智能路由控制塔'
        description='统一观测实时流量、供应商健康、SLA、路由策略和待执行动作。原有 Token Router 能力完整保留。'
        actions={
          <>
            <Button
              variant='outline'
              size='sm'
              onClick={() => query.refetch()}
              disabled={query.isFetching}
            >
              <RefreshCw
                className={cn('size-4', query.isFetching && 'animate-spin')}
              />
              刷新数据
            </Button>
            <Button size='sm'>
              <Sparkles className='size-4' />
              新建路由策略
            </Button>
          </>
        }
      />

      <div className='grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-5'>
        <EnterpriseStatCard
          title='活跃路由策略'
          value={formatCount(metrics?.active_policies ?? 0)}
          helper='已生效策略'
          icon={Route}
          tone='blue'
          loading={query.isLoading}
        />
        <EnterpriseStatCard
          title='实时成功率'
          value={formatPercent(metrics?.realtime_success_rate ?? 0)}
          helper='近 7 天请求'
          icon={ShieldCheck}
          tone='emerald'
          loading={query.isLoading}
        />
        <EnterpriseStatCard
          title='平均延迟'
          value={`${(metrics?.average_latency_ms ?? 0).toFixed(0)} ms`}
          helper='成功请求平均值'
          icon={Clock3}
          tone='violet'
          loading={query.isLoading}
        />
        <EnterpriseStatCard
          title='自动切换次数'
          value={formatCount(metrics?.automatic_switches ?? 0)}
          helper='本统计周期'
          icon={Zap}
          tone='amber'
          loading={query.isLoading}
        />
        <EnterpriseStatCard
          title='待审批事项'
          value={formatCount(metrics?.pending_approvals ?? 0)}
          helper='定价、供应商与决策'
          icon={CheckCircle2}
          tone='rose'
          loading={query.isLoading}
        />
      </div>

      <div className='grid grid-cols-1 gap-4 2xl:grid-cols-[minmax(0,1.7fr)_minmax(320px,0.8fr)]'>
        <EnterprisePanel
          title='路由拓扑与实时流量'
          description='请求从租户入口经过策略匹配，再分配至健康的供应商与渠道。'
          action={
            <Badge variant='outline'>
              {formatCount(metrics?.requests ?? 0)} 请求
            </Badge>
          }
          bodyClassName='p-4 sm:p-5'
        >
          <div className='grid min-h-[390px] items-center gap-5 lg:grid-cols-[minmax(180px,0.7fr)_52px_minmax(220px,0.9fr)_52px_minmax(280px,1.2fr)]'>
            <div className='rounded-2xl border bg-gradient-to-br from-primary/8 via-background to-violet-500/8 p-4'>
              <span className='flex size-10 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-lg shadow-primary/20'>
                <Network className='size-5' />
              </span>
              <p className='mt-4 text-sm font-semibold'>企业客户端流量</p>
              <p className='mt-1 text-2xl font-semibold tracking-tight'>
                {formatCount(metrics?.requests ?? 0)}
              </p>
              <p className='text-xs text-muted-foreground'>近 7 天总请求</p>
              <div className='mt-4 space-y-2 text-xs'>
                <div className='flex justify-between rounded-lg bg-background/80 px-3 py-2'>
                  <span>接口调用</span>
                  <span className='font-medium'>62%</span>
                </div>
                <div className='flex justify-between rounded-lg bg-background/80 px-3 py-2'>
                  <span>内部服务</span>
                  <span className='font-medium'>28%</span>
                </div>
                <div className='flex justify-between rounded-lg bg-background/80 px-3 py-2'>
                  <span>在线调试</span>
                  <span className='font-medium'>10%</span>
                </div>
              </div>
            </div>

            <div className='hidden items-center lg:flex'>
              <div className='h-px flex-1 bg-gradient-to-r from-primary/20 to-primary' />
              <ArrowRight className='size-4 text-primary' />
            </div>

            <div className='rounded-2xl border border-primary/20 bg-primary/[0.035] p-4 shadow-[0_12px_40px_rgb(59_130_246/0.08)]'>
              <div className='flex items-start justify-between gap-2'>
                <span className='flex size-10 items-center justify-center rounded-xl bg-violet-500/10 text-violet-600'>
                  <GitBranch className='size-5' />
                </span>
                <Badge className='border-0 bg-emerald-500/10 text-emerald-600'>
                  运行中
                </Badge>
              </div>
              <p className='mt-4 text-sm font-semibold'>智能路由策略</p>
              <p className='mt-1 truncate text-xs text-muted-foreground'>
                {primaryPolicy?.name || '默认智能路由'}
              </p>
              <div className='mt-4 grid grid-cols-2 gap-2'>
                <div className='rounded-xl border bg-background/80 p-3'>
                  <p className='text-[11px] text-muted-foreground'>成功率</p>
                  <p className='mt-1 text-sm font-semibold'>
                    {formatPercent(metrics?.realtime_success_rate ?? 0)}
                  </p>
                </div>
                <div className='rounded-xl border bg-background/80 p-3'>
                  <p className='text-[11px] text-muted-foreground'>策略优先级</p>
                  <p className='mt-1 text-sm font-semibold'>
                    {primaryPolicy?.priority ?? 100}
                  </p>
                </div>
              </div>
              <div className='mt-3 rounded-xl border border-dashed bg-background/55 p-3 text-xs text-muted-foreground'>
                加权路由 · 自动降级 · 区域感知 · SLA 守护
              </div>
            </div>

            <div className='hidden items-center lg:flex'>
              <div className='h-px flex-1 bg-gradient-to-r from-primary to-violet-400/30' />
              <ArrowRight className='size-4 text-primary' />
            </div>

            <div className='space-y-2.5'>
              {providers.length === 0 ? (
                <div className='flex min-h-56 items-center justify-center rounded-2xl border border-dashed text-sm text-muted-foreground'>
                  暂无供应商流量数据
                </div>
              ) : (
                providers.map((provider, index) => (
                  <ProviderNode
                    key={provider.channel_id}
                    provider={provider}
                    index={index}
                  />
                ))
              )}
            </div>
          </div>
        </EnterprisePanel>

        <div className='grid gap-4'>
          <EnterprisePanel
            title='实时健康状态'
            description='按当前请求量排序的主要渠道。'
            bodyClassName='p-3'
          >
            <div className='space-y-1'>
              {providers.map((provider) => {
                const status = providerStatus(provider)
                const latency =
                  provider.average_latency_ms || provider.response_time_ms
                return (
                  <div
                    key={provider.channel_id}
                    className='flex items-center gap-3 rounded-xl px-2.5 py-2.5 hover:bg-muted/45'
                  >
                    <span
                      className={cn(
                        'size-2 rounded-full',
                        status.label === '健康'
                          ? 'bg-emerald-500'
                          : status.label === '需关注'
                            ? 'bg-amber-500'
                            : 'bg-muted-foreground'
                      )}
                    />
                    <div className='min-w-0 flex-1'>
                      <p className='truncate text-sm font-medium'>
                        {provider.channel_name}
                      </p>
                      <p className='truncate text-[11px] text-muted-foreground'>
                        {provider.supplier_name || '未绑定供应商'}
                      </p>
                    </div>
                    <div className='text-right text-xs'>
                      <p className='font-semibold'>
                        {formatPercent(provider.success_rate)}
                      </p>
                      <p className='text-muted-foreground'>
                        {latency.toFixed(0)} ms
                      </p>
                    </div>
                  </div>
                )
              })}
            </div>
          </EnterprisePanel>

          <EnterprisePanel title='风险提醒' bodyClassName='p-2.5'>
            <EventList events={data?.risks ?? []} emptyText='当前没有未处理风险' />
          </EnterprisePanel>
        </div>
      </div>

      <div className='grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(320px,0.6fr)]'>
        <EnterprisePanel
          title='请求与路由趋势'
          description='展示近 7 天请求量，辅助判断流量波动与扩容窗口。'
          action={<Badge variant='outline'>自动刷新 60 秒</Badge>}
        >
          <div className='h-72'>
            <ResponsiveContainer width='100%' height='100%'>
              <AreaChart data={trend} margin={{ left: -16, right: 8 }}>
                <defs>
                  <linearGradient id='controlTowerRequests' x1='0' x2='0' y1='0' y2='1'>
                    <stop offset='5%' stopColor='var(--primary)' stopOpacity={0.35} />
                    <stop offset='95%' stopColor='var(--primary)' stopOpacity={0.02} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray='4 4' vertical={false} />
                <XAxis dataKey='date' tickLine={false} axisLine={false} />
                <YAxis tickLine={false} axisLine={false} tickFormatter={formatCount} />
                <Tooltip
                  formatter={(value) => [formatCount(Number(value ?? 0)), '请求量']}
                />
                <Area
                  dataKey='requests'
                  type='monotone'
                  stroke='var(--primary)'
                  strokeWidth={2.5}
                  fill='url(#controlTowerRequests)'
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </EnterprisePanel>

        <div className='grid gap-4 md:grid-cols-2 xl:grid-cols-1'>
          <EnterprisePanel title='最近策略变更' bodyClassName='p-2.5'>
            <EventList
              events={data?.recent_changes ?? []}
              emptyText='暂无策略变更记录'
            />
          </EnterprisePanel>
          <EnterprisePanel title='待执行动作' bodyClassName='p-2.5'>
            <EventList
              events={data?.pending_actions ?? []}
              emptyText='当前没有待执行动作'
            />
          </EnterprisePanel>
        </div>
      </div>

      <EnterprisePanel
        title='活跃路由策略'
        description='与原有策略、决策、SLA 和执行记录兼容，仍可在下方其他页签中进行深度治理。'
        action={<Badge variant='outline'>{policies.length} 条策略</Badge>}
        bodyClassName='p-0'
      >
        <div className='overflow-x-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>策略名称</TableHead>
                <TableHead>模型 / 切片</TableHead>
                <TableHead>主渠道</TableHead>
                <TableHead>供应商</TableHead>
                <TableHead>流量占比</TableHead>
                <TableHead>优先级</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>更新时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {policies.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className='h-28 text-center text-muted-foreground'>
                    暂无路由策略，可在“路由策略”页签创建或激活策略。
                  </TableCell>
                </TableRow>
              ) : (
                policies.slice(0, 10).map((policy) => (
                  <TableRow key={policy.id}>
                    <TableCell>
                      <div className='max-w-56'>
                        <p className='truncate font-medium'>{policy.name}</p>
                        <p className='truncate text-xs text-muted-foreground'>
                          {policy.track || '默认轨道'}
                        </p>
                      </div>
                    </TableCell>
                    <TableCell>
                      <p className='font-medium'>{policy.model_name || '全部模型'}</p>
                      <p className='text-xs text-muted-foreground'>
                        {policy.slice_key || '全局'}
                      </p>
                    </TableCell>
                    <TableCell>{policy.channel_name || `#${policy.channel_id}`}</TableCell>
                    <TableCell>{policy.supplier_name || '未绑定'}</TableCell>
                    <TableCell>{policy.traffic_percent}%</TableCell>
                    <TableCell>{policy.priority}</TableCell>
                    <TableCell>
                      <PolicyStatus policy={policy} />
                    </TableCell>
                    <TableCell>
                      {policy.updated_at > 0
                        ? dayjs.unix(policy.updated_at).format('MM-DD HH:mm')
                        : '—'}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </EnterprisePanel>

      <p className='sr-only'>{t('Overview')}</p>
    </div>
  )
}
