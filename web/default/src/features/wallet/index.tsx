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
  Banknote,
  CheckCircle2,
  CreditCard,
  Gem,
  Gift,
  Loader2,
  MailCheck,
  ReceiptText,
  RefreshCw,
  ShieldCheck,
  Smartphone,
  WalletCards,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'

import { EnterprisePanel, EnterpriseStatCard } from '@/components/enterprise'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import { getUserQuotaDates } from '@/features/dashboard/api'
import type { QuotaDataItem } from '@/features/dashboard/types'
import {
  getPublicPlans,
  getSelfSubscriptionFull,
} from '@/features/subscriptions/api'
import { SubscriptionPurchaseDialog } from '@/features/subscriptions/components/dialogs/subscription-purchase-dialog'
import type {
  PlanRecord,
  UserSubscriptionRecord,
} from '@/features/subscriptions/types'
import { get2FAStatus, getSelf } from '@/lib/api'
import {
  formatCompactNumber,
  formatQuota,
  formatTimestampToDate,
} from '@/lib/format'
import { cn } from '@/lib/utils'

import { getUserBillingHistory } from './api'
import { BillingHistoryDialog } from './components/dialogs/billing-history-dialog'
import { CreemConfirmDialog } from './components/dialogs/creem-confirm-dialog'
import { PaymentConfirmDialog } from './components/dialogs/payment-confirm-dialog'
import {
  useCreemPayment,
  usePayment,
  useRedemption,
  useTopupInfo,
  useWaffoPayment,
  useWaffoPancakePayment,
} from './hooks'
import { formatCurrency, getMinTopupAmount } from './lib'
import type {
  CreemProduct,
  PaymentMethod,
  TopupRecord,
  UserWalletData,
  WaffoPayMethod,
} from './types'

interface WalletProps {
  initialShowHistory?: boolean
}

type WalletUser = UserWalletData & {
  display_name?: string
  email?: string
  phone?: string
  group?: string
  status?: number
}

type RechargeMethod =
  | {
      id: string
      label: string
      kind: 'standard'
      method: PaymentMethod
    }
  | {
      id: string
      label: string
      kind: 'waffo'
      method: WaffoPayMethod
      index: number
    }
  | {
      id: string
      label: string
      kind: 'waffo-pancake'
    }

function buildMonthRange() {
  const now = new Date()
  return {
    start: Math.floor(
      new Date(now.getFullYear(), now.getMonth(), 1).getTime() / 1000
    ),
    end: Math.floor(now.getTime() / 1000),
  }
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

function formatSubscriptionPreference(value?: string) {
  switch (value) {
    case 'subscription_first':
      return '订阅优先'
    case 'wallet_first':
      return '钱包优先'
    case 'subscription_only':
      return '仅订阅'
    case 'wallet_only':
      return '仅钱包'
    default:
      return '默认'
  }
}

function formatBillingStatus(status: string) {
  if (status === 'success') return '已支付'
  if (status === 'pending') return '待确认'
  if (status === 'expired') return '已过期'
  return status || '-'
}

function billingStatusClass(status: string) {
  if (status === 'success') {
    return 'border-emerald-200 bg-emerald-50 text-emerald-700'
  }
  if (status === 'pending') {
    return 'border-amber-200 bg-amber-50 text-amber-700'
  }
  return 'border-slate-200 bg-slate-50 text-slate-500'
}

function formatBillingTime(record: TopupRecord) {
  const ts = record.complete_time || record.create_time
  return formatTimestampToDate(ts)
}

function getEpayMethods(payMethods: PaymentMethod[] = []): PaymentMethod[] {
  return payMethods.filter(
    (method) =>
      method?.type &&
      method.type !== 'stripe' &&
      method.type !== 'creem' &&
      method.type !== 'waffo'
  )
}

function getActiveSubscription(items: UserSubscriptionRecord[]) {
  return items.find((item) => {
    const status = item.subscription.status
    const endTime = item.subscription.end_time
    return status === 'active' && (!endTime || endTime > Date.now() / 1000)
  })
}

function subscriptionUsagePercent(item?: UserSubscriptionRecord | null) {
  if (!item) return 0
  const total = Number(item.subscription.amount_total || 0)
  const used = Number(item.subscription.amount_used || 0)
  if (total <= 0) return 0
  return Math.min(100, Math.round((used / total) * 100))
}

function planDurationLabel(plan: PlanRecord) {
  const p = plan.plan
  const unitMap: Record<string, string> = {
    year: '年',
    month: '月',
    day: '天',
    hour: '小时',
    custom: '自定义',
  }
  if (p.duration_unit === 'custom') {
    return `${p.custom_seconds || 0} 秒`
  }
  return `${p.duration_value || 1} ${unitMap[p.duration_unit] || p.duration_unit}`
}

function sortPlans(plans: PlanRecord[]) {
  return [...plans].sort((a, b) => {
    const sortOrder = (a.plan.sort_order || 0) - (b.plan.sort_order || 0)
    if (sortOrder !== 0) return sortOrder
    return a.plan.price_amount - b.plan.price_amount
  })
}

function buildPurchaseCountMap(subscriptions: UserSubscriptionRecord[]) {
  const map = new Map<number, number>()
  for (const item of subscriptions) {
    const planId = item.subscription.plan_id
    if (!planId) continue
    map.set(planId, (map.get(planId) || 0) + 1)
  }
  return map
}

function maskPhone(value?: string) {
  if (!value) return '未绑定'
  if (value.length <= 7) return value
  return `${value.slice(0, 3)}****${value.slice(-4)}`
}

const statCardClass = 'min-h-[86px] p-3'

export function Wallet(props: WalletProps) {
  const [topupAmount, setTopupAmount] = useState(0)
  const [selectedMethodId, setSelectedMethodId] = useState('')
  const [pendingPaymentMethod, setPendingPaymentMethod] =
    useState<PaymentMethod | null>(null)
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false)
  const [billingDialogOpen, setBillingDialogOpen] = useState(
    Boolean(props.initialShowHistory)
  )
  const [redemptionCode, setRedemptionCode] = useState('')
  const [selectedCreemProduct, setSelectedCreemProduct] =
    useState<CreemProduct | null>(null)
  const [creemDialogOpen, setCreemDialogOpen] = useState(false)
  const [selectedPlan, setSelectedPlan] = useState<PlanRecord | null>(null)
  const [purchaseOpen, setPurchaseOpen] = useState(false)

  const monthRange = useMemo(buildMonthRange, [])
  const { topupInfo, presetAmounts, loading: topupLoading } = useTopupInfo()
  const {
    amount: paymentAmount,
    calculating,
    processing,
    calculatePaymentAmount,
    processPayment,
  } = usePayment()
  const { redeeming, redeemCode } = useRedemption()
  const { processing: creemProcessing, processCreemPayment } = useCreemPayment()
  const { processWaffoPayment } = useWaffoPayment()
  const { processing: pancakeProcessing, processWaffoPancakePayment } =
    useWaffoPancakePayment()

  const userQuery = useQuery({
    queryKey: ['wallet-self'],
    queryFn: getSelf,
    staleTime: 30_000,
  })
  const usageQuery = useQuery({
    queryKey: ['wallet-month-usage', monthRange.start, monthRange.end],
    queryFn: () =>
      getUserQuotaDates(
        {
          start_timestamp: monthRange.start,
          end_timestamp: monthRange.end,
        },
        false
      ),
    staleTime: 30_000,
  })
  const billingQuery = useQuery({
    queryKey: ['wallet-billing-history'],
    queryFn: () => getUserBillingHistory(1, 5),
    staleTime: 30_000,
    retry: false,
  })
  const subscriptionQuery = useQuery({
    queryKey: ['wallet-self-subscription'],
    queryFn: getSelfSubscriptionFull,
    staleTime: 30_000,
    retry: false,
  })
  const plansQuery = useQuery({
    queryKey: ['wallet-public-plans'],
    queryFn: getPublicPlans,
    staleTime: 60_000,
    retry: false,
  })
  const twoFaQuery = useQuery({
    queryKey: ['wallet-2fa-status'],
    queryFn: get2FAStatus,
    staleTime: 60_000,
    retry: false,
  })

  const user = userQuery.data?.data as WalletUser | undefined
  const monthUsage = useMemo(
    () => aggregateUsage(usageQuery.data?.data ?? []),
    [usageQuery.data?.data]
  )
  const billingRecords = billingQuery.data?.data?.items ?? []
  const subscriptions = subscriptionQuery.data?.data?.subscriptions ?? []
  const allSubscriptions =
    subscriptionQuery.data?.data?.all_subscriptions ?? subscriptions
  const billingPreference = subscriptionQuery.data?.data?.billing_preference
  const activeSubscription = getActiveSubscription(subscriptions)
  const purchaseCountMap = useMemo(
    () => buildPurchaseCountMap(allSubscriptions),
    [allSubscriptions]
  )
  const plans = useMemo(
    () => sortPlans(plansQuery.data?.data ?? []),
    [plansQuery.data?.data]
  )
  const activePlan = plans.find(
    (item) => item.plan.id === activeSubscription?.subscription.plan_id
  )
  const recommendedPlans = plans.slice(0, 3)
  const usagePercent = subscriptionUsagePercent(activeSubscription)
  const epayMethods = useMemo(
    () => getEpayMethods(topupInfo?.pay_methods),
    [topupInfo?.pay_methods]
  )

  const rechargeMethods = useMemo<RechargeMethod[]>(() => {
    const methods: RechargeMethod[] = []
    for (const method of topupInfo?.pay_methods ?? []) {
      if (method.type === 'waffo') continue
      methods.push({
        id: `standard:${method.type}`,
        label: method.name || method.type,
        kind: 'standard',
        method,
      })
    }
    if (topupInfo?.enable_waffo_topup) {
      ;(topupInfo.waffo_pay_methods ?? []).forEach((method, index) => {
        methods.push({
          id: `waffo:${index}`,
          label: method.name || `Waffo ${index + 1}`,
          kind: 'waffo',
          method,
          index,
        })
      })
    }
    if (topupInfo?.enable_waffo_pancake_topup) {
      methods.push({
        id: 'waffo-pancake',
        label: 'Waffo Pancake',
        kind: 'waffo-pancake',
      })
    }
    return methods
  }, [topupInfo])

  const amountOptions = useMemo(() => {
    const fromPreset = presetAmounts
      .map((item) => item.value)
      .filter((item) => Number.isFinite(item) && item > 0)
      .slice(0, 4)
    if (fromPreset.length > 0) {
      return fromPreset
    }
    const minTopup = getMinTopupAmount(topupInfo)
    if (minTopup > 0) {
      return [minTopup, minTopup * 2, minTopup * 5, minTopup * 10]
    }
    return [10, 50, 100]
  }, [presetAmounts, topupInfo])

  const selectedMethod = rechargeMethods.find(
    (method) => method.id === selectedMethodId
  )
  const canRecharge =
    topupAmount > 0 &&
    selectedMethod != null &&
    topupInfo?.payment_compliance_confirmed !== false
  const refreshing =
    userQuery.isFetching ||
    usageQuery.isFetching ||
    billingQuery.isFetching ||
    subscriptionQuery.isFetching ||
    plansQuery.isFetching ||
    topupLoading
  const mfaEnabled = Boolean(
    (twoFaQuery.data?.data as { enabled?: boolean } | undefined)?.enabled
  )

  useEffect(() => {
    if (topupAmount <= 0 && amountOptions.length > 0) {
      setTopupAmount(amountOptions[Math.min(1, amountOptions.length - 1)])
    }
  }, [amountOptions, topupAmount])

  useEffect(() => {
    if (!selectedMethodId && rechargeMethods.length > 0) {
      setSelectedMethodId(rechargeMethods[0].id)
    }
  }, [rechargeMethods, selectedMethodId])

  useEffect(() => {
    if (props.initialShowHistory) {
      setBillingDialogOpen(true)
      window.history.replaceState({}, '', window.location.pathname)
    }
  }, [props.initialShowHistory])

  const refreshAll = useCallback(() => {
    void userQuery.refetch()
    void usageQuery.refetch()
    void billingQuery.refetch()
    void subscriptionQuery.refetch()
    void plansQuery.refetch()
  }, [billingQuery, plansQuery, subscriptionQuery, usageQuery, userQuery])

  const handleRedeem = async () => {
    const success = await redeemCode(redemptionCode)
    if (success) {
      setRedemptionCode('')
      refreshAll()
    }
  }

  const handleStartRecharge = async () => {
    if (!topupInfo) {
      toast.error('充值配置未加载')
      return
    }
    if (topupInfo.payment_compliance_confirmed === false) {
      toast.error('支付合规声明尚未确认')
      return
    }
    if (!selectedMethod || topupAmount <= 0) {
      toast.error('请选择充值金额和支付方式')
      return
    }

    if (selectedMethod.kind === 'waffo') {
      const success = await processWaffoPayment(
        topupAmount,
        selectedMethod.index
      )
      if (success) refreshAll()
      return
    }

    if (selectedMethod.kind === 'waffo-pancake') {
      await processWaffoPancakePayment(topupAmount)
      return
    }

    setPendingPaymentMethod(selectedMethod.method)
    await calculatePaymentAmount(topupAmount, selectedMethod.method.type)
    setConfirmDialogOpen(true)
  }

  const handlePaymentConfirm = async () => {
    if (!pendingPaymentMethod) return
    const success = await processPayment(topupAmount, pendingPaymentMethod.type)
    if (success) {
      setConfirmDialogOpen(false)
      refreshAll()
    }
  }

  const handleCreemProductSelect = (product: CreemProduct) => {
    setSelectedCreemProduct(product)
    setCreemDialogOpen(true)
  }

  const handleCreemConfirm = async () => {
    if (!selectedCreemProduct) return
    const success = await processCreemPayment(selectedCreemProduct.productId)
    if (success) {
      setCreemDialogOpen(false)
      setSelectedCreemProduct(null)
      refreshAll()
    }
  }

  const handleOpenPurchase = (plan: PlanRecord) => {
    setSelectedPlan(plan)
    setPurchaseOpen(true)
  }

  return (
    <>
      <div className='personal-wallet enterprise-dashboard space-y-2 px-4 pt-2 pb-2 text-slate-950 sm:px-8'>
        <div className='flex flex-col gap-2 lg:flex-row lg:items-end lg:justify-between'>
          <div className='min-w-0'>
            <h1 className='text-[22px] leading-7 font-semibold tracking-normal text-slate-950'>
              钱包 / 账单 / 订阅
            </h1>
            <p className='mt-0.5 text-[12px] leading-4 text-slate-500'>
              C端以预付费和订阅为主，保留充值、消费明细和发票信息
            </p>
          </div>
          <div className='flex flex-wrap items-center gap-2'>
            <Badge
              variant='outline'
              className='h-7 rounded-md border-slate-200 bg-white px-2.5 text-[11px] font-medium text-slate-600 shadow-[0_1px_2px_rgb(15_23_42/0.04)]'
            >
              当前分组 · {user?.group || 'default'}
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

        <div className='grid gap-2 sm:grid-cols-2 xl:grid-cols-4'>
          <EnterpriseStatCard
            title='可用余额'
            value={formatQuota(user?.quota ?? 0)}
            helper='立即可用'
            icon={WalletCards}
            tone='blue'
            loading={userQuery.isLoading}
            className={statCardClass}
          />
          <EnterpriseStatCard
            title='本月消费'
            value={formatQuota(monthUsage.quota)}
            helper={`${formatCompactNumber(monthUsage.requests)} 次请求`}
            icon={Banknote}
            tone='amber'
            loading={usageQuery.isLoading}
            className={statCardClass}
          />
          <EnterpriseStatCard
            title='订阅套餐'
            value={activePlan?.plan.title || '免费版'}
            helper={activeSubscription ? '当前有效' : '可升级'}
            icon={Gem}
            tone={activeSubscription ? 'violet' : 'slate'}
            loading={subscriptionQuery.isLoading || plansQuery.isLoading}
            className={statCardClass}
          />
          <EnterpriseStatCard
            title='计费偏好'
            value={formatSubscriptionPreference(billingPreference)}
            helper={
              activeSubscription ? `订阅已用 ${usagePercent}%` : '钱包扣费'
            }
            icon={CreditCard}
            tone='emerald'
            loading={subscriptionQuery.isLoading}
            className={statCardClass}
          />
        </div>

        <div className='grid gap-2 xl:grid-cols-[minmax(0,1.45fr)_minmax(320px,0.72fr)]'>
          <div className='space-y-2'>
            <EnterprisePanel
              title='充值中心'
              description='选择金额、支付方式或使用兑换码'
              bodyClassName='space-y-2.5'
            >
              <div className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
                {amountOptions.map((amount) => (
                  <button
                    key={amount}
                    type='button'
                    className={cn(
                      'h-9 rounded-md border px-3 text-left text-[12px] font-semibold transition-colors',
                      amount === topupAmount
                        ? 'border-blue-500 bg-blue-600 text-white shadow-[0_6px_16px_rgb(37_99_235/0.18)]'
                        : 'border-slate-200 bg-white text-slate-800 hover:border-blue-200 hover:bg-blue-50/45'
                    )}
                    onClick={() => setTopupAmount(amount)}
                  >
                    {formatCurrency(amount)}
                  </button>
                ))}
              </div>

              <div className='grid gap-2 lg:grid-cols-[minmax(180px,0.45fr)_minmax(0,1fr)]'>
                <Input
                  type='number'
                  min={getMinTopupAmount(topupInfo)}
                  value={topupAmount}
                  onChange={(event) => {
                    const value = Number(event.target.value)
                    setTopupAmount(Number.isFinite(value) ? value : 0)
                  }}
                  className='h-8 rounded-md bg-white text-[12px]'
                  aria-label='自定义充值金额'
                />
                <div className='flex flex-wrap gap-1.5'>
                  {rechargeMethods.length === 0 ? (
                    <div className='flex h-8 items-center rounded-md border border-slate-200 bg-slate-50 px-3 text-[11px] text-slate-500'>
                      暂无可用在线支付方式
                    </div>
                  ) : (
                    rechargeMethods.map((method) => (
                      <button
                        key={method.id}
                        type='button'
                        className={cn(
                          'h-8 rounded-md border px-2.5 text-[11px] font-medium transition-colors',
                          method.id === selectedMethodId
                            ? 'border-blue-500 bg-blue-50 text-blue-700'
                            : 'border-slate-200 bg-white text-slate-600 hover:border-blue-200 hover:text-blue-700'
                        )}
                        onClick={() => setSelectedMethodId(method.id)}
                      >
                        {method.label}
                      </button>
                    ))
                  )}
                </div>
              </div>

              <Button
                className='h-9 w-full rounded-md bg-blue-600 text-[12px] font-semibold hover:bg-blue-700'
                disabled={
                  !canRecharge || calculating || processing || pancakeProcessing
                }
                onClick={() => void handleStartRecharge()}
              >
                {calculating || processing || pancakeProcessing ? (
                  <Loader2 className='size-3.5 animate-spin' />
                ) : (
                  <CreditCard className='size-3.5' />
                )}
                立即充值
              </Button>

              {topupInfo?.enable_creem_topup &&
              (topupInfo.creem_products?.length ?? 0) > 0 ? (
                <div className='grid gap-1.5 md:grid-cols-2'>
                  {topupInfo.creem_products?.slice(0, 2).map((product) => (
                    <button
                      key={product.productId}
                      type='button'
                      className='flex items-center justify-between gap-2 rounded-md border border-violet-100 bg-violet-50/45 px-2.5 py-2 text-left transition-colors hover:border-violet-200'
                      onClick={() => handleCreemProductSelect(product)}
                    >
                      <span className='min-w-0'>
                        <span className='block truncate text-[12px] font-semibold text-slate-900'>
                          {product.name}
                        </span>
                        <span className='mt-0.5 block text-[10px] text-slate-500'>
                          {formatQuota(product.quota)}
                        </span>
                      </span>
                      <span className='text-[12px] font-semibold text-violet-700'>
                        {product.currency === 'EUR' ? '€' : '$'}
                        {product.price}
                      </span>
                    </button>
                  ))}
                </div>
              ) : null}

              <div className='grid gap-2 md:grid-cols-[minmax(0,1fr)_auto]'>
                <Input
                  value={redemptionCode}
                  onChange={(event) => setRedemptionCode(event.target.value)}
                  placeholder='输入兑换码'
                  className='h-8 rounded-md bg-white text-[12px]'
                  aria-label='兑换码'
                />
                <Button
                  variant='outline'
                  className='h-8 rounded-md bg-white px-3 text-[12px]'
                  disabled={!redemptionCode.trim() || redeeming}
                  onClick={() => void handleRedeem()}
                >
                  {redeeming ? (
                    <Loader2 className='size-3.5 animate-spin' />
                  ) : (
                    <Gift className='size-3.5' />
                  )}
                  兑换
                </Button>
              </div>
            </EnterprisePanel>

            <EnterprisePanel
              title='账单记录'
              description='最近充值与账单状态'
              action={
                <Button
                  variant='ghost'
                  size='sm'
                  className='h-7 px-2 text-[11px] text-blue-600'
                  onClick={() => setBillingDialogOpen(true)}
                >
                  查看全部
                  <ArrowRight className='size-3.5' />
                </Button>
              }
              bodyClassName='p-0'
            >
              <div className='overflow-x-auto'>
                <table className='w-full min-w-[680px] text-left text-[12px]'>
                  <thead className='border-b border-slate-100 bg-slate-50/55 text-[11px] text-slate-500'>
                    <tr>
                      <th className='px-3 py-2 font-medium'>时间</th>
                      <th className='px-3 py-2 font-medium'>金额</th>
                      <th className='px-3 py-2 font-medium'>支付方式</th>
                      <th className='px-3 py-2 font-medium'>状态</th>
                      <th className='px-3 py-2 text-right font-medium'>操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {billingRecords.length === 0 ? (
                      <tr>
                        <td
                          colSpan={5}
                          className='px-3 py-8 text-center text-[12px] text-slate-500'
                        >
                          暂无账单记录
                        </td>
                      </tr>
                    ) : (
                      billingRecords.map((record) => (
                        <tr
                          key={record.id}
                          className='border-b border-slate-100 last:border-0'
                        >
                          <td className='px-3 py-2.5 text-slate-600'>
                            {formatBillingTime(record)}
                          </td>
                          <td className='px-3 py-2.5 font-medium text-slate-900'>
                            {formatQuota(record.amount)}
                          </td>
                          <td className='px-3 py-2.5 text-slate-600'>
                            {record.payment_method || '-'}
                          </td>
                          <td className='px-3 py-2.5'>
                            <Badge
                              variant='outline'
                              className={cn(
                                'h-5 rounded px-1.5 text-[10px]',
                                billingStatusClass(record.status)
                              )}
                            >
                              {formatBillingStatus(record.status)}
                            </Badge>
                          </td>
                          <td className='px-3 py-2.5 text-right'>
                            <Button
                              variant='ghost'
                              size='sm'
                              className='h-6 px-1.5 text-[11px] text-blue-600'
                              onClick={() => setBillingDialogOpen(true)}
                            >
                              查看
                            </Button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </EnterprisePanel>
          </div>

          <div className='space-y-2'>
            <EnterprisePanel
              title='订阅权益'
              description='套餐、额度与可升级项'
              action={
                <Button
                  variant='ghost'
                  size='sm'
                  className='h-7 px-2 text-[11px] text-blue-600'
                  render={<Link to='/pricing' />}
                >
                  查看全部
                  <ArrowRight className='size-3.5' />
                </Button>
              }
              bodyClassName='space-y-2.5'
            >
              <div className='rounded-md border border-slate-100 bg-slate-50/55 p-2.5'>
                <div className='flex items-center justify-between gap-2'>
                  <div className='min-w-0'>
                    <p className='truncate text-[13px] font-semibold text-slate-950'>
                      {activePlan?.plan.title || '免费版'}
                    </p>
                    <p className='mt-0.5 text-[11px] text-slate-500'>
                      {activeSubscription
                        ? `有效期至 ${formatTimestampToDate(activeSubscription.subscription.end_time)}`
                        : '当前基础模型可用'}
                    </p>
                  </div>
                  <Badge
                    variant='outline'
                    className='h-6 rounded-md border-emerald-200 bg-emerald-50 px-2 text-[11px] text-emerald-700'
                  >
                    当前
                  </Badge>
                </div>
                {activeSubscription ? (
                  <div className='mt-2'>
                    <div className='mb-1 flex items-center justify-between text-[10px] text-slate-500'>
                      <span>额度使用</span>
                      <span>{usagePercent}%</span>
                    </div>
                    <Progress value={usagePercent} className='h-1.5' />
                  </div>
                ) : null}
              </div>

              <div className='divide-y divide-slate-100'>
                {recommendedPlans.length === 0 ? (
                  <div className='rounded-md border border-dashed border-slate-200 px-3 py-5 text-center text-[12px] text-slate-500'>
                    暂无可购买订阅套餐
                  </div>
                ) : (
                  recommendedPlans.map((plan) => {
                    const isCurrent = activePlan?.plan.id === plan.plan.id
                    return (
                      <div
                        key={plan.plan.id}
                        className='grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 py-2 first:pt-0 last:pb-0'
                      >
                        <div className='min-w-0'>
                          <p className='truncate text-[12px] font-medium text-slate-900'>
                            {plan.plan.title}
                          </p>
                          <p className='mt-0.5 truncate text-[11px] text-slate-500'>
                            {formatQuota(plan.plan.total_amount)} ·{' '}
                            {planDurationLabel(plan)}
                          </p>
                        </div>
                        {isCurrent ? (
                          <span className='text-[11px] font-medium text-slate-500'>
                            当前
                          </span>
                        ) : (
                          <Button
                            variant='ghost'
                            size='sm'
                            className='h-6 px-1.5 text-[11px] text-blue-600'
                            onClick={() => handleOpenPurchase(plan)}
                          >
                            升级
                          </Button>
                        )}
                      </div>
                    )
                  })
                )}
              </div>
            </EnterprisePanel>

            <EnterprisePanel
              title='账户安全'
              description='支付与账单通知状态'
              action={
                <Button
                  variant='ghost'
                  size='sm'
                  className='h-7 px-2 text-[11px] text-blue-600'
                  render={<Link to='/profile' />}
                >
                  查看全部
                  <ArrowRight className='size-3.5' />
                </Button>
              }
              bodyClassName='space-y-2'
            >
              {[
                {
                  label: '邮箱验证',
                  value: user?.email ? '已绑定' : '未绑定',
                  icon: MailCheck,
                },
                {
                  label: '手机绑定',
                  value: maskPhone(user?.phone),
                  icon: Smartphone,
                },
                {
                  label: 'MFA',
                  value: mfaEnabled ? '已开启' : '未开启',
                  icon: ShieldCheck,
                },
              ].map((item) => {
                const Icon = item.icon
                return (
                  <div
                    key={item.label}
                    className='grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2 border-b border-slate-100 pb-2 last:border-0 last:pb-0'
                  >
                    <span className='flex size-7 items-center justify-center rounded-md bg-slate-50 text-slate-500 ring-1 ring-slate-100'>
                      <Icon className='size-3.5' />
                    </span>
                    <span className='text-[12px] text-slate-600'>
                      {item.label}
                    </span>
                    <span className='text-[12px] font-medium text-slate-900'>
                      {item.value}
                    </span>
                  </div>
                )
              })}
            </EnterprisePanel>

            <EnterprisePanel
              title='发票与凭证'
              description='记录保留与下载'
              bodyClassName='space-y-2'
            >
              <div className='grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2 rounded-md border border-slate-100 bg-slate-50/55 p-2.5'>
                <span className='flex size-8 items-center justify-center rounded-md bg-white text-blue-600 ring-1 ring-blue-100'>
                  <ReceiptText className='size-4' />
                </span>
                <div className='min-w-0'>
                  <p className='truncate text-[12px] font-semibold text-slate-900'>
                    最近账单
                  </p>
                  <p className='mt-0.5 truncate text-[11px] text-slate-500'>
                    {billingRecords[0]
                      ? `${formatBillingTime(billingRecords[0])} · ${formatBillingStatus(billingRecords[0].status)}`
                      : '暂无可下载账单'}
                  </p>
                </div>
                <Button
                  variant='outline'
                  size='sm'
                  className='h-7 rounded-md bg-white px-2 text-[11px]'
                  onClick={() => setBillingDialogOpen(true)}
                >
                  查看
                </Button>
              </div>
              <div className='flex items-center gap-1.5 text-[11px] text-emerald-600'>
                <CheckCircle2 className='size-3.5' />
                账单数据来自 /api/user/topup/self
              </div>
            </EnterprisePanel>
          </div>
        </div>
      </div>

      <PaymentConfirmDialog
        open={confirmDialogOpen}
        onOpenChange={setConfirmDialogOpen}
        onConfirm={handlePaymentConfirm}
        topupAmount={topupAmount}
        paymentAmount={paymentAmount}
        paymentMethod={pendingPaymentMethod ?? undefined}
        calculating={calculating}
        processing={processing}
        discountRate={1}
        usdExchangeRate={1}
      />
      <BillingHistoryDialog
        open={billingDialogOpen}
        onOpenChange={setBillingDialogOpen}
      />
      <CreemConfirmDialog
        open={creemDialogOpen}
        onOpenChange={setCreemDialogOpen}
        onConfirm={handleCreemConfirm}
        product={selectedCreemProduct}
        processing={creemProcessing}
      />
      <SubscriptionPurchaseDialog
        open={purchaseOpen}
        onOpenChange={setPurchaseOpen}
        plan={selectedPlan}
        enableStripe={topupInfo?.enable_stripe_topup}
        enableCreem={topupInfo?.enable_creem_topup}
        enableWaffoPancake={topupInfo?.enable_waffo_pancake_topup}
        enableOnlineTopUp={topupInfo?.enable_online_topup}
        epayMethods={epayMethods}
        purchaseLimit={selectedPlan?.plan.max_purchase_per_user}
        purchaseCount={
          selectedPlan ? purchaseCountMap.get(selectedPlan.plan.id) || 0 : 0
        }
        userQuota={user?.quota}
        onPurchaseSuccess={() => {
          refreshAll()
          toast.success('订阅状态已刷新')
        }}
      />
    </>
  )
}
