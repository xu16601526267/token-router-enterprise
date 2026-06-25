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
import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  BadgeCheck,
  BookOpen,
  Check,
  CircleDollarSign,
  Copy,
  CreditCard,
  ExternalLink,
  FileClock,
  FlaskConical,
  KeyRound,
  LockKeyhole,
  RefreshCw,
  Sparkles,
  UserRound,
  Wallet,
  Zap,
} from 'lucide-react'
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
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import {
  formatCompactNumber,
  formatCurrencyUSD,
  formatNumber,
  formatQuota,
  formatTimestampToDate,
  formatTokens,
} from '@/lib/format'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'
import { getUserQuotaDates } from '@/features/dashboard/api'
import { fetchTokenKey, getApiKeys } from '@/features/keys/api'
import { getSelfSubscriptionFull } from '@/features/subscriptions/api'
import { getUserBillingHistory } from '@/features/wallet/api'
import type { QuotaDataItem } from '@/features/dashboard/types'

function formatDay(timestamp: number): string {
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
  }).format(timestamp * 1000)
}

function aggregateUsage(rows: QuotaDataItem[]) {
  const trend = new Map<number, { timestamp: number; requests: number; tokens: number; quota: number }>()
  const models = new Map<string, { requests: number; tokens: number }>()
  let requests = 0
  let tokens = 0
  let quota = 0

  rows.forEach((row) => {
    const requestCount = row.count ?? 0
    const tokenCount = row.token_used ?? 0
    const quotaCount = row.quota ?? 0
    requests += requestCount
    tokens += tokenCount
    quota += quotaCount

    const date = new Date(row.created_at * 1000)
    const bucket = Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()) / 1000
    const current = trend.get(bucket) ?? {
      timestamp: bucket,
      requests: 0,
      tokens: 0,
      quota: 0,
    }
    current.requests += requestCount
    current.tokens += tokenCount
    current.quota += quotaCount
    trend.set(bucket, current)

    const modelName = row.model_name || '未分类模型'
    const model = models.get(modelName) ?? { requests: 0, tokens: 0 }
    model.requests += requestCount
    model.tokens += tokenCount
    models.set(modelName, model)
  })

  return {
    requests,
    tokens,
    quota,
    trend: Array.from(trend.values())
      .sort((a, b) => a.timestamp - b.timestamp)
      .map((item) => ({ ...item, label: formatDay(item.timestamp) })),
    models: Array.from(models.entries())
      .map(([name, value]) => ({ name, ...value }))
      .sort((a, b) => b.requests - a.requests)
      .slice(0, 6),
  }
}

export function PersonalWorkbench() {
  const user = useAuthStore((state) => state.auth.user)
  const { copyToClipboard, copiedText } = useCopyToClipboard({
    successMessage: '已复制到剪贴板',
  })
  const [copyingKey, setCopyingKey] = useState(false)
  const range = useMemo(() => {
    const end = Math.floor(Date.now() / 1000)
    return { start: end - 7 * 24 * 60 * 60, end }
  }, [])

  const usageQuery = useQuery({
    queryKey: ['personal-workbench-usage', range.start, range.end],
    queryFn: () =>
      getUserQuotaDates(
        { start_timestamp: range.start, end_timestamp: range.end },
        false
      ),
    staleTime: 30_000,
  })
  const keysQuery = useQuery({
    queryKey: ['personal-workbench-keys'],
    queryFn: () => getApiKeys({ p: 1, size: 100 }),
    staleTime: 30_000,
  })
  const subscriptionQuery = useQuery({
    queryKey: ['personal-workbench-subscriptions'],
    queryFn: getSelfSubscriptionFull,
    staleTime: 60_000,
    retry: false,
  })
  const billingQuery = useQuery({
    queryKey: ['personal-workbench-billing'],
    queryFn: () => getUserBillingHistory(1, 5),
    staleTime: 60_000,
    retry: false,
  })

  const usage = useMemo(
    () => aggregateUsage(usageQuery.data?.data ?? []),
    [usageQuery.data?.data]
  )
  const keys = keysQuery.data?.data?.items ?? []
  const activeKeys = keys.filter((key) => key.status === 1)
  const preferredKey = activeKeys[0] ?? keys[0]
  const subscriptions = subscriptionQuery.data?.data?.subscriptions ?? []
  const activeSubscription = subscriptions.find(
    (item) => item.subscription.status === 'active'
  )
  const billingItems = billingQuery.data?.data?.items ?? []
  const latestTopup = billingItems[0]
  const baseUrl =
    typeof window === 'undefined' ? '/v1' : `${window.location.origin}/v1`

  const handleCopyKey = async () => {
    if (!preferredKey) return
    setCopyingKey(true)
    try {
      const response = await fetchTokenKey(preferredKey.id)
      const key = response.data?.key
      if (response.success && key) {
        await copyToClipboard(`sk-${key.replace(/^sk-/, '')}`)
      }
    } finally {
      setCopyingKey(false)
    }
  }

  return (
    <div className='enterprise-dashboard space-y-4 pb-2 sm:space-y-5'>
      <EnterprisePageHeader
        eyebrow='个人中心'
        title={`你好，${user?.display_name || user?.username || '用户'}`}
        description='管理个人额度、API Key、订阅和调用记录；企业管理员可在同一套系统中切换到完整经营驾驶舱。'
        actions={
          <>
            <Badge variant='outline' className='h-8 rounded-lg bg-background/70 px-3 text-xs font-normal'>
              当前分组 · {user?.group || '默认'}
            </Badge>
            <Button
              variant='outline'
              size='sm'
              className='h-8 rounded-lg bg-background/70'
              disabled={usageQuery.isFetching || keysQuery.isFetching}
              onClick={() => {
                void usageQuery.refetch()
                void keysQuery.refetch()
              }}
            >
              <RefreshCw
                className={cn(
                  'size-3.5',
                  (usageQuery.isFetching || keysQuery.isFetching) && 'animate-spin'
                )}
              />
              刷新
            </Button>
          </>
        }
      />

      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-5'>
        <EnterpriseStatCard
          title='可用额度'
          value={formatQuota(user?.quota ?? 0)}
          helper='当前账户余额'
          icon={Wallet}
          tone='blue'
        />
        <EnterpriseStatCard
          title='近 7 天调用'
          value={formatCompactNumber(usage.requests)}
          helper={`${formatTokens(usage.tokens)} Tokens`}
          icon={Zap}
          tone='emerald'
          loading={usageQuery.isLoading}
        />
        <EnterpriseStatCard
          title='活跃 API Key'
          value={formatNumber(activeKeys.length)}
          helper={`共创建 ${keysQuery.data?.data?.total ?? keys.length} 个`}
          icon={KeyRound}
          tone='violet'
          loading={keysQuery.isLoading}
        />
        <EnterpriseStatCard
          title='最近充值'
          value={latestTopup ? formatCurrencyUSD(latestTopup.money) : '-'}
          helper={latestTopup ? formatTimestampToDate(latestTopup.create_time) : '暂无充值记录'}
          icon={CircleDollarSign}
          tone='amber'
          loading={billingQuery.isLoading}
        />
        <EnterpriseStatCard
          title='订阅状态'
          value={activeSubscription ? '使用中' : '未订阅'}
          helper={
            activeSubscription
              ? `有效期至 ${formatTimestampToDate(activeSubscription.subscription.end_time)}`
              : '可按需购买订阅计划'
          }
          icon={CreditCard}
          tone={activeSubscription ? 'emerald' : 'slate'}
          loading={subscriptionQuery.isLoading}
        />
      </div>

      <div className='grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(0,0.85fr)]'>
        <EnterprisePanel
          title='我的 API 接入'
          description='密钥默认脱敏展示，复制时通过安全接口临时读取。'
          action={
            <Button
              variant='ghost'
              size='sm'
              className='h-7 px-2 text-xs'
              render={<Link to='/keys' />}
            >
              管理全部密钥
              <ArrowRight className='size-3.5' />
            </Button>
          }
          bodyClassName='space-y-4'
        >
          <div className='rounded-xl border bg-[linear-gradient(135deg,color-mix(in_oklch,var(--primary)_6%,var(--background))_0%,var(--background)_100%)] p-4'>
            <div className='flex flex-col justify-between gap-3 sm:flex-row sm:items-center'>
              <div className='min-w-0'>
                <div className='flex items-center gap-2'>
                  <span className='flex size-8 items-center justify-center rounded-lg bg-primary/10 text-primary'>
                    <KeyRound className='size-4' />
                  </span>
                  <div>
                    <p className='text-sm font-semibold'>
                      {preferredKey?.name || '尚未创建 API Key'}
                    </p>
                    <p className='mt-0.5 font-mono text-[11px] text-muted-foreground'>
                      {preferredKey?.key || '创建密钥后即可开始调用'}
                    </p>
                  </div>
                </div>
              </div>
              {preferredKey ? (
                <Button
                  size='sm'
                  className='h-8 rounded-lg'
                  disabled={copyingKey}
                  onClick={() => void handleCopyKey()}
                >
                  {copiedText ? <Check className='size-3.5' /> : <Copy className='size-3.5' />}
                  {copyingKey ? '读取中' : copiedText ? '已复制' : '复制完整 Key'}
                </Button>
              ) : (
                <Button size='sm' className='h-8 rounded-lg' render={<Link to='/keys' />}>
                  创建 API Key
                </Button>
              )}
            </div>
          </div>

          <div className='grid gap-3 sm:grid-cols-2'>
            <div className='rounded-xl border bg-background/55 p-3.5'>
              <p className='text-[11px] font-medium text-muted-foreground'>Base URL</p>
              <div className='mt-2 flex items-center gap-2'>
                <code className='min-w-0 flex-1 truncate rounded-lg bg-muted px-2.5 py-2 text-[11px]'>
                  {baseUrl}
                </code>
                <Button
                  variant='outline'
                  size='icon-sm'
                  onClick={() => void copyToClipboard(baseUrl)}
                  aria-label='复制 Base URL'
                >
                  <Copy className='size-3.5' />
                </Button>
              </div>
            </div>
            <div className='rounded-xl border bg-background/55 p-3.5'>
              <p className='text-[11px] font-medium text-muted-foreground'>密钥权限</p>
              <div className='mt-2 flex flex-wrap gap-1.5'>
                <Badge variant='secondary' className='text-[10px]'>
                  {preferredKey?.group || '默认分组'}
                </Badge>
                <Badge variant='secondary' className='text-[10px]'>
                  {preferredKey?.model_limits_enabled ? '限制模型' : '全部授权模型'}
                </Badge>
                <Badge variant='secondary' className='text-[10px]'>
                  {preferredKey?.allow_ips ? '已配置 IP 白名单' : '未限制 IP'}
                </Badge>
              </div>
            </div>
          </div>

          <div className='grid gap-2 sm:grid-cols-3'>
            <a
              href='/playground'
              className='group flex items-center gap-3 rounded-xl border px-3.5 py-3 transition-colors hover:border-primary/25 hover:bg-muted/30'
            >
              <FlaskConical className='size-4 text-blue-600' />
              <div className='min-w-0 flex-1'>
                <p className='text-xs font-semibold'>在线调试台</p>
                <p className='truncate text-[10px] text-muted-foreground'>立即验证模型调用</p>
              </div>
              <ArrowRight className='size-3.5 text-muted-foreground' />
            </a>
            <a
              href='/pricing'
              className='group flex items-center gap-3 rounded-xl border px-3.5 py-3 transition-colors hover:border-primary/25 hover:bg-muted/30'
            >
              <BookOpen className='size-4 text-violet-600' />
              <div className='min-w-0 flex-1'>
                <p className='text-xs font-semibold'>模型与价格</p>
                <p className='truncate text-[10px] text-muted-foreground'>查看可用模型</p>
              </div>
              <ArrowRight className='size-3.5 text-muted-foreground' />
            </a>
            <a
              href='/wallet'
              className='group flex items-center gap-3 rounded-xl border px-3.5 py-3 transition-colors hover:border-primary/25 hover:bg-muted/30'
            >
              <Wallet className='size-4 text-emerald-600' />
              <div className='min-w-0 flex-1'>
                <p className='text-xs font-semibold'>充值与账单</p>
                <p className='truncate text-[10px] text-muted-foreground'>管理余额和记录</p>
              </div>
              <ArrowRight className='size-3.5 text-muted-foreground' />
            </a>
          </div>
        </EnterprisePanel>

        <EnterprisePanel
          title='近 7 天用量趋势'
          description={`共 ${formatCompactNumber(usage.requests)} 次请求 · ${formatTokens(usage.tokens)} Tokens`}
          bodyClassName='h-[340px] px-2 pb-2 pt-4 sm:px-3'
        >
          {usage.trend.length > 0 ? (
            <ResponsiveContainer width='100%' height='100%'>
              <AreaChart data={usage.trend} margin={{ top: 12, right: 14, left: 0, bottom: 4 }}>
                <defs>
                  <linearGradient id='personalUsageGradient' x1='0' y1='0' x2='0' y2='1'>
                    <stop offset='5%' stopColor='#6366f1' stopOpacity={0.28} />
                    <stop offset='95%' stopColor='#6366f1' stopOpacity={0.015} />
                  </linearGradient>
                </defs>
                <CartesianGrid vertical={false} stroke='var(--border)' strokeOpacity={0.75} />
                <XAxis
                  dataKey='label'
                  axisLine={false}
                  tickLine={false}
                  dy={8}
                  tick={{ fill: 'var(--muted-foreground)', fontSize: 11 }}
                />
                <YAxis
                  axisLine={false}
                  tickLine={false}
                  width={46}
                  tickFormatter={(value) => formatCompactNumber(Number(value))}
                  tick={{ fill: 'var(--muted-foreground)', fontSize: 11 }}
                />
                <ChartTooltip
                  contentStyle={{
                    borderRadius: 12,
                    border: '1px solid var(--border)',
                    background: 'var(--popover)',
                    color: 'var(--popover-foreground)',
                    boxShadow: '0 14px 35px rgb(15 23 42 / 0.12)',
                    fontSize: 12,
                  }}
                  formatter={(value, name) => [formatNumber(Number(value)), name]}
                />
                <Area
                  type='monotone'
                  dataKey='requests'
                  name='请求量'
                  stroke='#6366f1'
                  strokeWidth={2.2}
                  fill='url(#personalUsageGradient)'
                />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className='flex h-full flex-col items-center justify-center text-center'>
              <Sparkles className='mb-3 size-8 text-muted-foreground/40' />
              <p className='text-sm font-medium'>还没有调用记录</p>
              <p className='mt-1 text-xs text-muted-foreground'>创建 API Key 并发起请求后，这里会展示真实趋势。</p>
            </div>
          )}
        </EnterprisePanel>
      </div>

      <div className='grid gap-4 xl:grid-cols-3'>
        <EnterprisePanel title='常用模型' description='按近 7 天请求量排序'>
          {usage.models.length > 0 ? (
            <div className='space-y-3'>
              {usage.models.map((model, index) => (
                <div key={model.name} className='flex items-center gap-3 rounded-xl border bg-background/50 p-3'>
                  <span className='flex size-7 shrink-0 items-center justify-center rounded-lg bg-violet-500/10 text-[10px] font-semibold text-violet-600'>
                    {index + 1}
                  </span>
                  <div className='min-w-0 flex-1'>
                    <p className='truncate text-xs font-semibold'>{model.name}</p>
                    <p className='mt-0.5 text-[10px] text-muted-foreground'>
                      {formatTokens(model.tokens)} Tokens
                    </p>
                  </div>
                  <span className='text-xs font-semibold tabular-nums'>
                    {formatCompactNumber(model.requests)}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <div className='flex min-h-52 items-center justify-center text-sm text-muted-foreground'>暂无模型用量</div>
          )}
        </EnterprisePanel>

        <EnterprisePanel
          title='最近充值记录'
          description='仅展示当前账户的真实支付记录'
          action={
            <Button variant='ghost' size='sm' className='h-7 px-2 text-xs' render={<Link to='/wallet' />}>
              查看全部
              <ArrowRight className='size-3.5' />
            </Button>
          }
        >
          {billingItems.length > 0 ? (
            <div className='divide-y divide-border/60'>
              {billingItems.slice(0, 5).map((record) => (
                <div key={record.id} className='flex items-center gap-3 py-3 first:pt-0 last:pb-0'>
                  <span className='flex size-8 shrink-0 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-600'>
                    <CircleDollarSign className='size-4' />
                  </span>
                  <div className='min-w-0 flex-1'>
                    <p className='truncate text-xs font-medium'>{record.payment_method || '余额充值'}</p>
                    <p className='mt-0.5 text-[10px] text-muted-foreground'>
                      {formatTimestampToDate(record.create_time)}
                    </p>
                  </div>
                  <div className='text-right'>
                    <p className='text-xs font-semibold'>{formatCurrencyUSD(record.money)}</p>
                    <Badge
                      variant='outline'
                      className={cn(
                        'mt-1 text-[9px]',
                        record.status === 'success'
                          ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-600'
                          : 'border-amber-500/20 bg-amber-500/10 text-amber-600'
                      )}
                    >
                      {record.status === 'success' ? '已完成' : record.status === 'pending' ? '处理中' : '已过期'}
                    </Badge>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className='flex min-h-52 flex-col items-center justify-center text-center'>
              <FileClock className='mb-3 size-8 text-muted-foreground/40' />
              <p className='text-sm font-medium'>暂无充值记录</p>
              <p className='mt-1 text-xs text-muted-foreground'>充值完成后将在此展示。</p>
            </div>
          )}
        </EnterprisePanel>

        <EnterprisePanel title='账户与安全' description='来自当前登录账户的真实配置状态'>
          <div className='space-y-3'>
            {[
              {
                label: '登录账户',
                value: user?.username || '-',
                icon: UserRound,
                state: '正常',
              },
              {
                label: '邮箱',
                value: user?.email || '未绑定',
                icon: BadgeCheck,
                state: user?.email ? '已绑定' : '待完善',
              },
              {
                label: '访问分组',
                value: user?.group || '默认分组',
                icon: LockKeyhole,
                state: '已生效',
              },
              {
                label: 'API Key 状态',
                value: `${activeKeys.length} 个可用`,
                icon: KeyRound,
                state: activeKeys.length > 0 ? '正常' : '待创建',
              },
            ].map((item) => {
              const Icon = item.icon
              return (
                <div key={item.label} className='flex items-center gap-3 rounded-xl border bg-background/50 p-3'>
                  <span className='flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground'>
                    <Icon className='size-4' />
                  </span>
                  <div className='min-w-0 flex-1'>
                    <p className='text-[10px] text-muted-foreground'>{item.label}</p>
                    <p className='mt-0.5 truncate text-xs font-medium'>{item.value}</p>
                  </div>
                  <span className='shrink-0 text-[10px] font-medium text-emerald-600'>{item.state}</span>
                </div>
              )
            })}
          </div>
          <Button
            variant='outline'
            size='sm'
            className='mt-4 w-full rounded-lg'
            render={<Link to='/profile' />}
          >
            进入账户设置
            <ExternalLink className='size-3.5' />
          </Button>
        </EnterprisePanel>
      </div>
    </div>
  )
}
