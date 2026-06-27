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
  ArrowRight,
  BadgeCheck,
  BookOpen,
  Check,
  CircleDollarSign,
  Code2,
  Copy,
  CreditCard,
  ExternalLink,
  FileClock,
  FlaskConical,
  KeyRound,
  LockKeyhole,
  Mail,
  ReceiptText,
  RefreshCw,
  ShieldCheck,
  Sparkles,
  UserRound,
  Wallet,
  Zap,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip as ChartTooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { toast } from 'sonner'

import { EnterprisePanel, EnterpriseStatCard } from '@/components/enterprise'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { getUserQuotaDates } from '@/features/dashboard/api'
import type { QuotaDataItem } from '@/features/dashboard/types'
import { fetchTokenKey, getApiKeys } from '@/features/keys/api'
import { getSelfSubscriptionFull } from '@/features/subscriptions/api'
import { getTopupInfo, getUserBillingHistory } from '@/features/wallet/api'
import { usePayment } from '@/features/wallet/hooks/use-payment'
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

function formatDay(timestamp: number): string {
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
  }).format(timestamp * 1000)
}

function aggregateUsage(rows: QuotaDataItem[]) {
  const trend = new Map<
    number,
    { timestamp: number; requests: number; tokens: number; quota: number }
  >()
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
    const bucket =
      Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()) /
      1000
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
    trend: [...trend.values()]
      .sort((a, b) => a.timestamp - b.timestamp)
      .map((item) => ({ ...item, label: formatDay(item.timestamp) })),
    models: [...models.entries()]
      .map(([name, value]) => ({ name, ...value }))
      .sort((a, b) => b.requests - a.requests)
      .slice(0, 6),
  }
}

function startOfUtcDay(timestamp: number): number {
  const date = new Date(timestamp * 1000)
  return (
    Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()) /
    1000
  )
}

function fillDailyUsageTrend(
  rows: ReturnType<typeof aggregateUsage>['trend'],
  startTimestamp: number,
  endTimestamp: number
) {
  const byDayLabel = new Map(rows.map((item) => [item.label, item]))
  const start = startOfUtcDay(startTimestamp)
  const end = startOfUtcDay(endTimestamp)
  const filled: ReturnType<typeof aggregateUsage>['trend'] = []

  for (let timestamp = start; timestamp <= end; timestamp += 24 * 60 * 60) {
    const label = formatDay(timestamp)
    const existing = byDayLabel.get(label)
    filled.push(
      existing ?? {
        timestamp,
        label,
        requests: 0,
        tokens: 0,
        quota: 0,
      }
    )
  }

  return filled
}

function buildRanges() {
  const now = new Date()
  const end = Math.floor(now.getTime() / 1000)
  const monthStart = Math.floor(
    new Date(now.getFullYear(), now.getMonth(), 1).getTime() / 1000
  )

  return {
    month: { start: monthStart, end },
    trend: { start: end - 7 * 24 * 60 * 60, end },
  }
}

function formatPaymentStatus(status: string) {
  if (status === 'success') return '已完成'
  if (status === 'pending') return '处理中'
  return '已过期'
}

function paymentStatusClass(status: string) {
  if (status === 'success') {
    return 'border-emerald-200 bg-emerald-50 text-emerald-700'
  }
  if (status === 'pending') {
    return 'border-amber-200 bg-amber-50 text-amber-700'
  }
  return 'border-slate-200 bg-slate-50 text-slate-500'
}

const PERSONAL_STAT_CARD_CLASS = 'min-h-[88px] p-3'

function buildPersonalBaseUrl(configuredServerUrl?: string): string {
  if (configuredServerUrl != null && configuredServerUrl.length > 0) {
    return `${configuredServerUrl.replace(/\/$/, '')}/v1`
  }
  if (typeof window === 'undefined') return '/v1'
  return `${window.location.origin}/v1`
}

export function PersonalWorkbench() {
  const user = useAuthStore((state) => state.auth.user)
  const { copyToClipboard, copiedText } = useCopyToClipboard({
    successMessage: '已复制到剪贴板',
  })
  const { processing: paymentProcessing, processPayment } = usePayment()
  const [copyingKey, setCopyingKey] = useState(false)
  const [selectedTopupAmount, setSelectedTopupAmount] = useState<number | null>(
    null
  )
  const ranges = useMemo(buildRanges, [])

  const trendUsageQuery = useQuery({
    queryKey: [
      'personal-workbench-trend-usage',
      ranges.trend.start,
      ranges.trend.end,
    ],
    queryFn: () =>
      getUserQuotaDates(
        {
          start_timestamp: ranges.trend.start,
          end_timestamp: ranges.trend.end,
        },
        false
      ),
    staleTime: 30_000,
  })
  const monthUsageQuery = useQuery({
    queryKey: [
      'personal-workbench-month-usage',
      ranges.month.start,
      ranges.month.end,
    ],
    queryFn: () =>
      getUserQuotaDates(
        {
          start_timestamp: ranges.month.start,
          end_timestamp: ranges.month.end,
        },
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
  const topupInfoQuery = useQuery({
    queryKey: ['personal-workbench-topup-info'],
    queryFn: getTopupInfo,
    staleTime: 60_000,
    retry: false,
  })

  const trendRows = trendUsageQuery.data?.data
  const monthRows = monthUsageQuery.data?.data
  const trendUsage = useMemo(() => aggregateUsage(trendRows ?? []), [trendRows])
  const monthUsage = useMemo(() => aggregateUsage(monthRows ?? []), [monthRows])
  const trendChartRows = useMemo(
    () =>
      fillDailyUsageTrend(
        trendUsage.trend,
        ranges.trend.start,
        ranges.trend.end
      ),
    [ranges.trend.end, ranges.trend.start, trendUsage.trend]
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
  const topupInfo = topupInfoQuery.data?.data
  const paymentMethods = topupInfo?.pay_methods ?? []
  const amountOptions = useMemo(() => {
    const presets = (topupInfo?.amount_options ?? [])
      .filter((amount) => Number.isFinite(amount) && amount > 0)
      .slice(0, 5)

    if (presets.length > 0) return presets
    if (topupInfo?.min_topup && topupInfo.min_topup > 0) {
      return [topupInfo.min_topup]
    }
    return []
  }, [topupInfo])
  const activeTopupAmount =
    selectedTopupAmount ?? amountOptions[1] ?? amountOptions[0] ?? 0
  const primaryPaymentMethod =
    paymentMethods[0]?.type ?? (topupInfo?.enable_stripe_topup ? 'stripe' : '')
  const canTopup =
    activeTopupAmount > 0 &&
    primaryPaymentMethod.length > 0 &&
    (topupInfo?.enable_online_topup || topupInfo?.enable_stripe_topup) &&
    topupInfo?.payment_compliance_confirmed !== false
  const recentActivities = useMemo(
    () =>
      [...(trendRows ?? [])]
        .sort((a, b) => b.created_at - a.created_at)
        .slice(0, 4),
    [trendRows]
  )
  const modelTotalRequests = Math.max(
    trendUsage.models.reduce((sum, model) => sum + model.requests, 0),
    1
  )
  const configuredServerUrl = import.meta.env.VITE_REACT_APP_SERVER_URL as
    | string
    | undefined
  const baseUrl = buildPersonalBaseUrl(configuredServerUrl)

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

  const handleStartPayment = async () => {
    if (topupInfoQuery.isLoading) return
    if (!topupInfo) {
      toast.error('充值配置未加载')
      return
    }
    if (!topupInfo.enable_online_topup && !topupInfo.enable_stripe_topup) {
      toast.error('在线充值未启用')
      return
    }
    if (topupInfo.payment_compliance_confirmed === false) {
      toast.error('支付合规声明尚未确认')
      return
    }
    if (!primaryPaymentMethod || activeTopupAmount <= 0) {
      toast.error('未配置可用充值金额或支付方式')
      return
    }
    await processPayment(activeTopupAmount, primaryPaymentMethod)
  }

  const refreshAll = () => {
    void trendUsageQuery.refetch()
    void monthUsageQuery.refetch()
    void keysQuery.refetch()
    void billingQuery.refetch()
    void topupInfoQuery.refetch()
  }

  const refreshing =
    trendUsageQuery.isFetching ||
    monthUsageQuery.isFetching ||
    keysQuery.isFetching ||
    billingQuery.isFetching ||
    topupInfoQuery.isFetching
  let copyKeyButtonLabel = '复制 Key'
  if (copyingKey) {
    copyKeyButtonLabel = '读取中'
  } else if (copiedText) {
    copyKeyButtonLabel = '已复制'
  }

  return (
    <div className='personal-workbench enterprise-dashboard space-y-2 px-0.5 pt-2 pb-2 text-slate-950 sm:px-0'>
      <div className='flex flex-col gap-2 lg:flex-row lg:items-end lg:justify-between'>
        <div className='min-w-0'>
          <h1 className='text-[22px] leading-7 font-semibold tracking-normal text-slate-950'>
            个人工作台
          </h1>
          <p className='mt-0.5 text-[12px] leading-4 text-slate-500'>
            个人额度、API Key、自助充值与快捷体验
          </p>
        </div>
        <div className='flex flex-wrap items-center gap-2'>
          <Badge
            variant='outline'
            className='h-7 rounded-md border-slate-200 bg-white px-2.5 text-[11px] font-medium text-slate-600 shadow-[0_1px_2px_rgb(15_23_42/0.04)]'
          >
            当前分组 · {user?.group || '默认'}
          </Badge>
          <Button
            variant='outline'
            size='sm'
            className='h-7 rounded-md bg-white px-2.5 text-[11px]'
            disabled={refreshing}
            onClick={refreshAll}
          >
            <RefreshCw
              className={cn('size-3.5', refreshing && 'animate-spin')}
            />
            刷新
          </Button>
        </div>
      </div>

      <div className='grid gap-2 sm:grid-cols-2 xl:grid-cols-5'>
        <EnterpriseStatCard
          title='可用额度'
          value={formatQuota(user?.quota ?? 0)}
          helper='当前账户余额'
          icon={Wallet}
          tone='blue'
          className={PERSONAL_STAT_CARD_CLASS}
        />
        <EnterpriseStatCard
          title='本月调用'
          value={formatCompactNumber(monthUsage.requests)}
          helper={`${formatTokens(monthUsage.tokens)} Tokens`}
          icon={Zap}
          tone='emerald'
          loading={monthUsageQuery.isLoading}
          className={PERSONAL_STAT_CARD_CLASS}
        />
        <EnterpriseStatCard
          title='活跃 API Key'
          value={formatNumber(activeKeys.length)}
          helper={`共创建 ${keysQuery.data?.data?.total ?? keys.length} 个`}
          icon={KeyRound}
          tone='violet'
          loading={keysQuery.isLoading}
          className={PERSONAL_STAT_CARD_CLASS}
        />
        <EnterpriseStatCard
          title='最近充值'
          value={latestTopup ? formatCurrencyUSD(latestTopup.money) : '-'}
          helper={
            latestTopup
              ? formatTimestampToDate(latestTopup.create_time)
              : '暂无充值记录'
          }
          icon={CircleDollarSign}
          tone='amber'
          loading={billingQuery.isLoading}
          className={PERSONAL_STAT_CARD_CLASS}
        />
        <EnterpriseStatCard
          title='订阅状态'
          value={activeSubscription ? '使用中' : '免费版'}
          helper={
            activeSubscription
              ? `有效期至 ${formatTimestampToDate(activeSubscription.subscription.end_time)}`
              : '升级享更多权益'
          }
          icon={CreditCard}
          tone={activeSubscription ? 'emerald' : 'slate'}
          loading={subscriptionQuery.isLoading}
          className={PERSONAL_STAT_CARD_CLASS}
        />
      </div>

      <div className='grid gap-2 xl:grid-cols-[minmax(360px,0.95fr)_232px_minmax(420px,1.05fr)]'>
        <EnterprisePanel
          title='我的 API Key'
          description='默认密钥脱敏展示，复制时通过安全接口读取'
          action={
            <Button
              variant='ghost'
              size='sm'
              className='h-7 px-2 text-[11px] text-blue-600'
              render={<Link to='/keys' />}
            >
              管理全部
              <ArrowRight className='size-3.5' />
            </Button>
          }
          bodyClassName='space-y-2'
        >
          <div className='rounded-md border border-blue-100 bg-blue-50/55 p-2.5'>
            <div className='flex flex-col gap-2 md:flex-row md:items-center md:justify-between'>
              <div className='min-w-0'>
                <div className='flex items-center gap-2.5'>
                  <span className='flex size-7 shrink-0 items-center justify-center rounded-md bg-white text-blue-600 ring-1 ring-blue-100'>
                    <KeyRound className='size-4' />
                  </span>
                  <div className='min-w-0'>
                    <p className='truncate text-[13px] font-semibold text-slate-950'>
                      {preferredKey?.name || '尚未创建 API Key'}
                    </p>
                    <p className='mt-0.5 truncate font-mono text-[11px] text-slate-500'>
                      {preferredKey?.key || '创建密钥后即可开始调用'}
                    </p>
                  </div>
                </div>
              </div>
              {preferredKey ? (
                <Button
                  size='sm'
                  className='h-7 rounded-md px-2.5 text-[12px]'
                  disabled={copyingKey}
                  onClick={() => void handleCopyKey()}
                >
                  {copiedText ? (
                    <Check className='size-3.5' />
                  ) : (
                    <Copy className='size-3.5' />
                  )}
                  {copyKeyButtonLabel}
                </Button>
              ) : (
                <Button
                  size='sm'
                  className='h-7 rounded-md px-2.5 text-[12px]'
                  render={<Link to='/keys' />}
                >
                  创建 API Key
                </Button>
              )}
            </div>
          </div>

          <div className='grid gap-2 md:grid-cols-2'>
            <div className='rounded-md border border-slate-100 bg-slate-50/55 p-2.5'>
              <p className='text-[11px] font-medium text-slate-500'>Base URL</p>
              <div className='mt-2 flex items-center gap-2'>
                <code className='min-w-0 flex-1 truncate rounded-md bg-white px-2 py-1.5 text-[11px] text-slate-700 ring-1 ring-slate-100'>
                  {baseUrl}
                </code>
                <Button
                  variant='outline'
                  size='icon-sm'
                  className='rounded-md bg-white'
                  onClick={() => void copyToClipboard(baseUrl)}
                  aria-label='复制 Base URL'
                >
                  <Copy className='size-3.5' />
                </Button>
              </div>
            </div>
            <div className='rounded-md border border-slate-100 bg-slate-50/55 p-2.5'>
              <p className='text-[11px] font-medium text-slate-500'>密钥权限</p>
              <div className='mt-2 flex flex-wrap gap-1.5'>
                <Badge
                  variant='secondary'
                  className='rounded px-1.5 text-[10px]'
                >
                  {preferredKey?.group || '默认分组'}
                </Badge>
                <Badge
                  variant='secondary'
                  className='rounded px-1.5 text-[10px]'
                >
                  {preferredKey?.model_limits_enabled ? '限制模型' : '全部模型'}
                </Badge>
                <Badge
                  variant='secondary'
                  className='rounded px-1.5 text-[10px]'
                >
                  {preferredKey?.allow_ips ? 'IP 白名单' : '未限制 IP'}
                </Badge>
              </div>
            </div>
          </div>
        </EnterprisePanel>

        <EnterprisePanel
          title='快速入口'
          description='常用工具'
          bodyClassName='space-y-1.5'
        >
          {[
            {
              title: 'Playground',
              desc: '在线验证模型调用',
              icon: FlaskConical,
              tone: 'text-blue-600 bg-blue-50 ring-blue-100',
              href: '/playground',
            },
            {
              title: 'API 文档',
              desc: '查看接口与模型路径',
              icon: BookOpen,
              tone: 'text-violet-600 bg-violet-50 ring-violet-100',
              href: '/models',
            },
            {
              title: 'SDK & 示例',
              desc: '复制 OpenAI 兼容地址',
              icon: Code2,
              tone: 'text-emerald-600 bg-emerald-50 ring-emerald-100',
              href: '/pricing',
            },
          ].map((entry) => {
            const Icon = entry.icon
            return (
              <a
                key={entry.title}
                href={entry.href}
                className='group flex items-center gap-2 rounded-md border border-slate-100 bg-white px-2.5 py-2 transition-colors hover:border-blue-200 hover:bg-blue-50/35'
              >
                <span
                  className={cn(
                    'flex size-7 shrink-0 items-center justify-center rounded-md ring-1',
                    entry.tone
                  )}
                >
                  <Icon className='size-4' />
                </span>
                <span className='min-w-0 flex-1'>
                  <span className='block truncate text-[12px] font-semibold text-slate-900'>
                    {entry.title}
                  </span>
                  <span className='mt-0.5 block truncate text-[10px] text-slate-500'>
                    {entry.desc}
                  </span>
                </span>
                <ArrowRight className='size-3.5 text-slate-400 group-hover:text-blue-600' />
              </a>
            )
          })}
        </EnterprisePanel>

        <EnterprisePanel
          title='用量趋势'
          description={`近 7 天 · ${formatCompactNumber(trendUsage.requests)} 次请求`}
          action={
            <Badge
              variant='outline'
              className='h-6 rounded-md bg-white px-2 text-[11px] font-medium'
            >
              近 7 天
            </Badge>
          }
          bodyClassName='h-[160px] px-2 pb-2 pt-2 sm:px-3'
        >
          {trendChartRows.length > 0 ? (
            <ResponsiveContainer
              width='100%'
              height='100%'
              initialDimension={{ width: 420, height: 170 }}
            >
              <AreaChart
                data={trendChartRows}
                margin={{ top: 8, right: 10, left: -6, bottom: 0 }}
              >
                <defs>
                  <linearGradient
                    id='personalUsageGradient'
                    x1='0'
                    y1='0'
                    x2='0'
                    y2='1'
                  >
                    <stop offset='5%' stopColor='#3b82f6' stopOpacity={0.26} />
                    <stop
                      offset='95%'
                      stopColor='#3b82f6'
                      stopOpacity={0.025}
                    />
                  </linearGradient>
                </defs>
                <CartesianGrid
                  vertical={false}
                  stroke='#e2e8f0'
                  strokeDasharray='4 8'
                />
                <XAxis
                  dataKey='label'
                  axisLine={false}
                  tickLine={false}
                  dy={8}
                  tick={{ fill: '#64748b', fontSize: 10 }}
                />
                <YAxis
                  axisLine={false}
                  tickLine={false}
                  width={40}
                  tickFormatter={(value) => formatCompactNumber(Number(value))}
                  tick={{ fill: '#64748b', fontSize: 10 }}
                />
                <ChartTooltip
                  contentStyle={{
                    borderRadius: 6,
                    border: '1px solid #e2e8f0',
                    background: '#fff',
                    color: '#0f172a',
                    boxShadow: '0 12px 28px rgb(15 23 42 / 0.12)',
                    fontSize: 12,
                  }}
                  formatter={(value, name) => [
                    formatNumber(Number(value)),
                    name,
                  ]}
                />
                <Area
                  type='monotone'
                  dataKey='requests'
                  name='请求量'
                  stroke='#2563eb'
                  strokeWidth={2}
                  dot={{ r: 2, strokeWidth: 1, fill: '#fff' }}
                  activeDot={{ r: 3 }}
                  fill='url(#personalUsageGradient)'
                />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className='flex h-full flex-col items-center justify-center text-center'>
              <Sparkles className='mb-2 size-7 text-slate-300' />
              <p className='text-[13px] font-medium text-slate-900'>
                还没有调用记录
              </p>
              <p className='mt-1 text-[11px] text-slate-500'>
                创建 API Key 并发起请求后展示真实趋势。
              </p>
            </div>
          )}
        </EnterprisePanel>
      </div>

      <div className='grid items-start gap-2 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.9fr)_minmax(250px,0.72fr)]'>
        <EnterprisePanel
          title='最近活动'
          description='来自个人用量聚合记录'
          bodyClassName='p-0'
        >
          {recentActivities.length > 0 ? (
            <div className='overflow-x-auto'>
              <table className='w-full min-w-[620px] text-left text-[12px]'>
                <thead className='border-b border-slate-100 bg-slate-50/70 text-[11px] text-slate-500'>
                  <tr>
                    <th className='px-3 py-1.5 font-medium'>时间</th>
                    <th className='px-3 py-1.5 font-medium'>类型</th>
                    <th className='px-3 py-1.5 font-medium'>对象</th>
                    <th className='px-3 py-1.5 text-right font-medium'>消耗</th>
                    <th className='px-3 py-1.5 text-right font-medium'>状态</th>
                  </tr>
                </thead>
                <tbody className='divide-y divide-slate-100'>
                  {recentActivities.map((item, index) => {
                    const activityKey =
                      item.id == null || item.id <= 0
                        ? `${item.created_at}-${item.model_name ?? 'unknown'}-${item.count ?? 0}-${item.quota ?? 0}-${index}`
                        : `log-${item.id}`

                    return (
                      <tr key={activityKey}>
                        <td className='px-3 py-1.5 text-slate-500'>
                          {formatTimestampToDate(item.created_at)}
                        </td>
                        <td className='px-3 py-1.5 font-medium text-slate-900'>
                          模型调用
                        </td>
                        <td className='max-w-[220px] truncate px-3 py-1.5 text-slate-600'>
                          {item.model_name || '未分类模型'}
                        </td>
                        <td className='px-3 py-1.5 text-right font-semibold text-slate-900 tabular-nums'>
                          {formatQuota(item.quota ?? 0)}
                        </td>
                        <td className='px-3 py-1.5 text-right'>
                          <Badge className='rounded border-emerald-200 bg-emerald-50 px-1.5 text-[10px] text-emerald-700 shadow-none'>
                            成功
                          </Badge>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <div className='flex min-h-[112px] flex-col items-center justify-center text-center'>
              <FileClock className='mb-2 size-7 text-slate-300' />
              <p className='text-[13px] font-medium text-slate-900'>
                暂无活动记录
              </p>
              <p className='mt-1 text-[11px] text-slate-500'>
                真实请求产生后会自动出现在这里。
              </p>
            </div>
          )}
        </EnterprisePanel>

        <EnterprisePanel
          title='最近账单'
          description='当前账户充值订单'
          action={
            <span className='text-[11px] text-slate-500'>
              共 {formatNumber(billingQuery.data?.data?.total ?? 0)} 条
            </span>
          }
          bodyClassName='space-y-1.5'
        >
          {billingItems.length > 0 ? (
            billingItems.slice(0, 4).map((record) => (
              <div
                key={record.id}
                className='flex items-center gap-2 rounded-md border border-slate-100 bg-white p-2'
              >
                <span className='flex size-7 shrink-0 items-center justify-center rounded-md bg-emerald-50 text-emerald-600 ring-1 ring-emerald-100'>
                  <ReceiptText className='size-4' />
                </span>
                <div className='min-w-0 flex-1'>
                  <p className='truncate text-[12px] font-semibold text-slate-900'>
                    {record.payment_method || '余额充值'}
                  </p>
                  <p className='mt-0.5 truncate text-[10px] text-slate-500'>
                    {formatTimestampToDate(record.create_time)}
                  </p>
                </div>
                <div className='shrink-0 text-right'>
                  <p className='text-[12px] font-semibold text-slate-900 tabular-nums'>
                    {formatCurrencyUSD(record.money)}
                  </p>
                  <Badge
                    variant='outline'
                    className={cn(
                      'mt-1 rounded px-1.5 text-[10px]',
                      paymentStatusClass(record.status)
                    )}
                  >
                    {formatPaymentStatus(record.status)}
                  </Badge>
                </div>
              </div>
            ))
          ) : (
            <div className='flex min-h-[112px] flex-col items-center justify-center text-center'>
              <FileClock className='mb-2 size-7 text-slate-300' />
              <p className='text-[13px] font-medium text-slate-900'>
                暂无账单记录
              </p>
              <p className='mt-1 text-[11px] text-slate-500'>
                充值成功后会显示订单状态。
              </p>
            </div>
          )}
        </EnterprisePanel>

        <div className='grid gap-2'>
          <EnterprisePanel
            title='充值中心'
            description='按系统配置展示可用金额'
            bodyClassName='space-y-1 px-2 py-1.5'
          >
            {amountOptions.length > 0 ? (
              <div className='grid grid-cols-5 gap-1'>
                {amountOptions.map((amount) => {
                  const active = amount === activeTopupAmount
                  return (
                    <button
                      key={amount}
                      type='button'
                      className={cn(
                        'h-6 rounded-md border px-1 text-[11px] font-semibold transition-colors',
                        active
                          ? 'border-blue-300 bg-blue-50 text-blue-700'
                          : 'border-slate-100 bg-white text-slate-700 hover:border-blue-200'
                      )}
                      onClick={() => setSelectedTopupAmount(amount)}
                    >
                      ${amount}
                    </button>
                  )
                })}
              </div>
            ) : (
              <div className='rounded-md border border-dashed border-slate-200 bg-slate-50 px-3 py-2 text-[11px] text-slate-500'>
                暂未配置在线充值金额
              </div>
            )}
            <Button
              className='h-6 w-full rounded-md text-[11px]'
              disabled={!canTopup || paymentProcessing}
              onClick={() => void handleStartPayment()}
            >
              <CircleDollarSign className='size-3.5' />
              {paymentProcessing ? '处理中' : '立即充值'}
            </Button>
            <p className='truncate text-[10px] leading-4 text-slate-500'>
              {primaryPaymentMethod
                ? `默认支付方式：${paymentMethods[0]?.name || primaryPaymentMethod}`
                : '请先在系统设置中配置支付方式'}
            </p>
          </EnterprisePanel>

          <EnterprisePanel
            title='账户安全'
            description='当前登录账户状态'
            action={
              <Button
                variant='ghost'
                size='sm'
                className='h-6 px-1.5 text-[11px] text-blue-600'
                render={<Link to='/profile' />}
              >
                设置
                <ExternalLink className='size-3' />
              </Button>
            }
            bodyClassName='space-y-0.5 px-2 py-1.5'
          >
            {[
              {
                label: '登录账户',
                value: user?.username || '-',
                icon: UserRound,
                state: '正常',
              },
              {
                label: '邮箱绑定',
                value: user?.email || '未绑定',
                icon: Mail,
                state: user?.email ? '已绑定' : '待完善',
              },
              {
                label: '访问分组',
                value: user?.group || '默认分组',
                icon: ShieldCheck,
                state: '已生效',
              },
              {
                label: 'API Key',
                value: `${activeKeys.length} 个可用`,
                icon: LockKeyhole,
                state: activeKeys.length > 0 ? '正常' : '待创建',
              },
            ].map((item) => {
              const Icon = item.icon
              return (
                <div
                  key={item.label}
                  className='flex h-6 min-w-0 items-center gap-1.5 rounded-md border border-slate-100 bg-white px-1.5'
                >
                  <span className='flex size-[18px] shrink-0 items-center justify-center rounded bg-slate-50 text-slate-500 ring-1 ring-slate-100'>
                    <Icon className='size-3' />
                  </span>
                  <span className='shrink-0 text-[10px] text-slate-500'>
                    {item.label}
                  </span>
                  <span className='min-w-0 flex-1 truncate text-[11px] font-medium text-slate-900'>
                    {item.value}
                  </span>
                  <span className='shrink-0 text-[10px] font-medium text-emerald-600'>
                    {item.state}
                  </span>
                </div>
              )
            })}
          </EnterprisePanel>
        </div>
      </div>

      <section className='rounded-md border border-slate-200 bg-white px-3 py-2 shadow-[0_1px_2px_rgb(15_23_42/0.035)]'>
        <div className='flex flex-col gap-2 lg:flex-row lg:items-center'>
          <div className='min-w-40 shrink-0'>
            <h3 className='text-[13px] font-semibold text-slate-900'>
              常用模型（已授权）
            </h3>
            <p className='mt-0.5 text-[10px] text-slate-500'>
              按近 7 天请求量排序
            </p>
          </div>
          {trendUsage.models.length > 0 ? (
            <div className='grid flex-1 gap-1.5 md:grid-cols-3 xl:grid-cols-5'>
              {trendUsage.models.slice(0, 5).map((model, index) => {
                const percent = Math.round(
                  (model.requests / modelTotalRequests) * 100
                )
                return (
                  <div
                    key={model.name}
                    className='rounded-md border border-slate-100 bg-slate-50/35 px-1.5 py-1'
                  >
                    <div className='flex items-center gap-2'>
                      <span className='flex size-6 shrink-0 items-center justify-center rounded-md bg-violet-50 text-[10px] font-semibold text-violet-700 ring-1 ring-violet-100'>
                        {index + 1}
                      </span>
                      <div className='min-w-0 flex-1'>
                        <p className='truncate text-[12px] font-semibold text-slate-900'>
                          {model.name}
                        </p>
                        <p className='mt-0.5 text-[10px] text-slate-500'>
                          {formatTokens(model.tokens)} Tokens
                        </p>
                      </div>
                      <span className='text-[12px] font-semibold text-slate-900 tabular-nums'>
                        {formatCompactNumber(model.requests)}
                      </span>
                    </div>
                    <div className='mt-1 h-1.5 overflow-hidden rounded-full bg-slate-100'>
                      <div
                        className='h-full rounded-full bg-blue-500'
                        style={{ width: `${Math.max(percent, 4)}%` }}
                      />
                    </div>
                  </div>
                )
              })}
            </div>
          ) : (
            <div className='flex min-h-10 flex-1 items-center justify-center text-[12px] text-slate-500'>
              暂无模型用量
            </div>
          )}
        </div>
      </section>

      <section className='rounded-md border border-slate-200 bg-white px-3 py-2 shadow-[0_1px_2px_rgb(15_23_42/0.035)]'>
        <div className='flex flex-col gap-2 lg:flex-row lg:items-center'>
          <div className='min-w-28 shrink-0'>
            <h3 className='text-[13px] font-semibold text-slate-900'>
              快捷操作
            </h3>
            <p className='mt-0.5 text-[10px] text-slate-500'>
              围绕个人调用链路
            </p>
          </div>
          <div className='grid flex-1 gap-2 sm:grid-cols-2 lg:grid-cols-5'>
            <Button
              variant='outline'
              className='h-8 justify-start rounded-md bg-white text-[12px]'
              disabled={!preferredKey || copyingKey}
              onClick={() => void handleCopyKey()}
            >
              <Copy className='size-4 text-blue-600' />
              复制默认 Key
            </Button>
            <Button
              variant='outline'
              className='h-8 justify-start rounded-md bg-white text-[12px]'
              render={<Link to='/playground' />}
            >
              <FlaskConical className='size-4 text-violet-600' />
              去 Playground
            </Button>
            <Button
              variant='outline'
              className='h-8 justify-start rounded-md bg-white text-[12px]'
              onClick={() => void handleStartPayment()}
              disabled={!canTopup || paymentProcessing}
            >
              <CircleDollarSign className='size-4 text-emerald-600' />
              快速充值
            </Button>
            <Button
              variant='outline'
              className='h-8 justify-start rounded-md bg-white text-[12px]'
              render={<Link to='/profile' />}
            >
              <BadgeCheck className='size-4 text-slate-600' />
              更新资料
            </Button>
            <Button
              variant='outline'
              className='h-8 justify-start rounded-md bg-white text-[12px]'
              render={<Link to='/profile' />}
            >
              <ShieldCheck className='size-4 text-blue-600' />
              开启 MFA
            </Button>
          </div>
        </div>
      </section>
    </div>
  )
}
