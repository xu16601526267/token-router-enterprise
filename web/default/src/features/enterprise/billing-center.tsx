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
import { Link, useNavigate } from '@tanstack/react-router'
import {
  AlertTriangle,
  BadgeDollarSign,
  CircleDollarSign,
  CreditCard,
  Download,
  FileText,
  Landmark,
  PieChart as PieChartIcon,
  ReceiptText,
  RefreshCw,
  Scale,
  Sparkles,
  WalletCards,
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
import { useMemo, useState, type CSSProperties, type ReactNode } from 'react'
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
} from '@/lib/format'
import { cn } from '@/lib/utils'

import {
  exportEnterpriseBilling,
  generateEnterpriseSettlement,
  getEnterpriseBilling,
} from './api'
import type { EnterpriseBillingData, EnterpriseSettlementItem } from './types'

type CsvValue = boolean | null | number | string | undefined

const EMPTY_BILLING: EnterpriseBillingData = {
  generated_at: 0,
  range: { start_timestamp: 0, end_timestamp: 0 },
  metrics: {
    total_balance_quota: 0,
    total_used_quota: 0,
    period_sell_quota: 0,
    period_cost_quota: 0,
    period_gross_profit_quota: 0,
    gross_margin_rate: 0,
    successful_top_up_amount: 0,
    pending_top_up_amount: 0,
    active_subscriptions: 0,
    draft_settlements: 0,
  },
  trend: [],
  settlements: [],
  recent_topups: [],
}

function formatPercent(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    style: 'percent',
    maximumFractionDigits: 1,
  }).format(Number.isFinite(value) ? value : 0)
}

function formatDate(timestamp: number): string {
  if (timestamp <= 0) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(timestamp * 1000)
}

function formatShortDate(timestamp: number): string {
  if (timestamp <= 0) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
  }).format(timestamp * 1000)
}

function formatFileDate(timestamp: number): string {
  if (timestamp <= 0) return 'unknown'
  return new Date(timestamp * 1000).toISOString().slice(0, 10)
}

function dateInputValue(timestamp: number): string {
  if (timestamp <= 0) return ''
  return new Date(timestamp * 1000).toISOString().slice(0, 10)
}

function dateInputTimestamp(value: string, endOfDay = false): number {
  const parts = value.split('-').map((part) => Number(part))
  if (parts.length !== 3 || parts.some((part) => !Number.isFinite(part))) {
    return 0
  }
  const [year, month, day] = parts
  const date = new Date(
    year,
    month - 1,
    day,
    endOfDay ? 23 : 0,
    endOfDay ? 59 : 0,
    endOfDay ? 59 : 0
  )
  return Math.floor(date.getTime() / 1000)
}

function csvCell(value: CsvValue): string {
  const text = String(value ?? '')
  if (!/[",\n\r]/.test(text)) return text
  return `"${text.replaceAll('"', '""')}"`
}

function downloadCsv(filename: string, rows: CsvValue[][]) {
  const csv = rows.map((row) => row.map(csvCell).join(',')).join('\n')
  const blob = new Blob([`\uFEFF${csv}`], {
    type: 'text/csv;charset=utf-8',
  })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

function downloadSettlementCsv(item: EnterpriseSettlementItem) {
  const margin =
    item.total_sell_quota > 0
      ? item.gross_profit_quota / item.total_sell_quota
      : 0
  downloadCsv(`settlement-${item.id}-${formatFileDate(item.period_end)}.csv`, [
    [
      '结算单ID',
      '周期开始',
      '周期结束',
      '对象类型',
      '对象ID',
      '对象名称',
      '应收额度',
      '应付额度',
      '毛利额度',
      '毛利率',
      '请求数',
      '状态',
    ],
    [
      item.id,
      formatDate(item.period_start),
      formatDate(item.period_end),
      item.subject_type === 'supplier' ? '供应商' : '客户',
      item.subject_id,
      item.subject_name,
      item.total_sell_quota,
      item.total_cost_quota,
      item.gross_profit_quota,
      formatPercent(margin),
      item.total_requests,
      item.status,
    ],
  ])
  toast.success('结算单已导出')
}

function SettlementStatus(props: { status: string }) {
  let label = props.status
  let className = 'border-slate-500/20 bg-slate-500/10 text-slate-600'
  if (props.status === 'draft') {
    label = '草稿'
    className = 'border-amber-500/20 bg-amber-500/10 text-amber-600'
  }
  if (props.status === 'finalized') {
    label = '已确认'
    className = 'border-blue-500/20 bg-blue-500/10 text-blue-600'
  }
  if (props.status === 'paid') {
    label = '已结算'
    className = 'border-emerald-500/20 bg-emerald-500/10 text-emerald-600'
  }
  return (
    <Badge variant='outline' className={cn('text-[10px]', className)}>
      {label || '未知'}
    </Badge>
  )
}

function SettlementTable(props: { items: EnterpriseSettlementItem[] }) {
  if (props.items.length === 0) {
    return (
      <div className='text-muted-foreground flex min-h-72 items-center justify-center text-sm'>
        暂无结算单，系统会在生成结算数据后显示在这里
      </div>
    )
  }
  return (
    <div className='overflow-x-auto'>
      <Table>
        <TableHeader>
          <TableRow className='bg-muted/35'>
            <TableHead className='min-w-32'>周期</TableHead>
            <TableHead className='min-w-40'>客户 / 供应商</TableHead>
            <TableHead className='text-right'>应收</TableHead>
            <TableHead className='text-right'>应付</TableHead>
            <TableHead className='text-right'>毛利</TableHead>
            <TableHead className='text-right'>毛利率</TableHead>
            <TableHead className='text-right'>请求数</TableHead>
            <TableHead>状态</TableHead>
            <TableHead className='text-right'>操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {props.items.map((item) => {
            const margin =
              item.total_sell_quota > 0
                ? item.gross_profit_quota / item.total_sell_quota
                : 0
            return (
              <TableRow key={item.id} className='hover:bg-muted/25'>
                <TableCell className='text-xs'>
                  {formatShortDate(item.period_start)} -{' '}
                  {formatShortDate(item.period_end)}
                </TableCell>
                <TableCell>
                  <p className='text-xs font-semibold'>{item.subject_name}</p>
                  <p className='text-muted-foreground mt-0.5 text-[10px]'>
                    {item.subject_type === 'supplier'
                      ? '供应商结算'
                      : '客户结算'}{' '}
                    · #{item.subject_id}
                  </p>
                </TableCell>
                <TableCell className='text-right text-xs tabular-nums'>
                  {formatLogQuota(item.total_sell_quota)}
                </TableCell>
                <TableCell className='text-right text-xs tabular-nums'>
                  {formatLogQuota(item.total_cost_quota)}
                </TableCell>
                <TableCell className='text-right text-xs font-semibold tabular-nums'>
                  {formatLogQuota(item.gross_profit_quota)}
                </TableCell>
                <TableCell className='text-right text-xs tabular-nums'>
                  {formatPercent(margin)}
                </TableCell>
                <TableCell className='text-right text-xs tabular-nums'>
                  {formatCompactNumber(item.total_requests)}
                </TableCell>
                <TableCell>
                  <SettlementStatus status={item.status} />
                </TableCell>
                <TableCell className='text-right'>
                  <Button
                    variant='ghost'
                    size='icon-sm'
                    aria-label='下载结算单'
                    onClick={() => downloadSettlementCsv(item)}
                  >
                    <Download className='size-3.5' />
                  </Button>
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}

export function EnterpriseBillingCenter(props: {
  actions?: ReactNode
  classicContent?: ReactNode
}) {
  const navigate = useNavigate()
  const [exportingBilling, setExportingBilling] = useState(false)
  const range = useMemo(() => {
    const end = Math.floor(Date.now() / 1000)
    return { start: end - 30 * 24 * 60 * 60, end }
  }, [])
  const [settlementDialogOpen, setSettlementDialogOpen] = useState(false)
  const [settlementSubjectType, setSettlementSubjectType] = useState<
    'user' | 'supplier'
  >('user')
  const [settlementSubjectId, setSettlementSubjectId] = useState('')
  const [settlementPeriodStart, setSettlementPeriodStart] = useState(() =>
    dateInputValue(range.start)
  )
  const [settlementPeriodEnd, setSettlementPeriodEnd] = useState(() =>
    dateInputValue(range.end)
  )
  const [generatingSettlement, setGeneratingSettlement] = useState(false)
  const billingQuery = useQuery({
    queryKey: ['enterprise-billing', range.start, range.end],
    queryFn: () =>
      getEnterpriseBilling({
        start_timestamp: range.start,
        end_timestamp: range.end,
      }),
    staleTime: 30_000,
  })
  const data = billingQuery.data?.data ?? EMPTY_BILLING
  const metrics = data.metrics
  const totalAllocated = metrics.total_balance_quota + metrics.total_used_quota
  const usageRate =
    totalAllocated > 0 ? metrics.total_used_quota / totalAllocated : 0
  const trend = data.trend.map((item) => ({
    ...item,
    label: formatShortDate(item.timestamp),
  }))
  const settlementGrossProfit = data.settlements.reduce(
    (sum, item) => sum + item.gross_profit_quota,
    0
  )
  const collectionProgress =
    metrics.successful_top_up_amount + metrics.pending_top_up_amount > 0
      ? metrics.successful_top_up_amount /
        (metrics.successful_top_up_amount + metrics.pending_top_up_amount)
      : 0
  const exportBillingCsv = async () => {
    setExportingBilling(true)
    try {
      await exportEnterpriseBilling({
        start_timestamp: range.start,
        end_timestamp: range.end,
      })
      toast.success('账单数据已导出')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '导出失败')
    } finally {
      setExportingBilling(false)
    }
  }
  const submitSettlement = async () => {
    const subjectId = Number(settlementSubjectId)
    if (!Number.isInteger(subjectId) || subjectId <= 0) {
      toast.error('请输入有效的客户或供应商 ID')
      return
    }
    const periodStart = dateInputTimestamp(settlementPeriodStart)
    const periodEnd = dateInputTimestamp(settlementPeriodEnd, true)
    if (periodStart <= 0 || periodEnd <= 0 || periodEnd < periodStart) {
      toast.error('请选择有效的结算周期')
      return
    }
    setGeneratingSettlement(true)
    try {
      const result = await generateEnterpriseSettlement({
        subject_type: settlementSubjectType,
        supplier_id:
          settlementSubjectType === 'supplier' ? subjectId : undefined,
        user_id: settlementSubjectType === 'user' ? subjectId : undefined,
        period_start: periodStart,
        period_end: periodEnd,
      })
      if (!result.success) {
        throw new Error(result.message || '生成失败')
      }
      toast.success(`结算单 #${result.data?.id ?? subjectId} 已生成`)
      setSettlementDialogOpen(false)
      void billingQuery.refetch()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '生成失败')
    } finally {
      setGeneratingSettlement(false)
    }
  }
  const openFeeAnalysis = () => {
    toast.info('已打开模型调用分析，可按模型核对用量与费用')
    void navigate({ to: '/dashboard/$section', params: { section: 'models' } })
  }

  return (
    <div className='enterprise-dashboard space-y-4 pb-2 sm:space-y-5'>
      <EnterprisePageHeader
        eyebrow='组织与计费'
        title='计费与结算中心'
        description='统一管理订阅、预付额度、经营毛利、充值流水与客户/供应商结算单，同时保留旧版订阅管理。'
        actions={
          <div className='flex flex-wrap items-center gap-2'>
            <Button
              variant='outline'
              size='sm'
              onClick={() => void billingQuery.refetch()}
              disabled={billingQuery.isFetching}
            >
              <RefreshCw
                className={cn(
                  'size-4',
                  billingQuery.isFetching && 'animate-spin'
                )}
              />
              刷新
            </Button>
            {props.actions}
          </div>
        }
      />

      <div className='grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-6'>
        <EnterpriseStatCard
          title='活跃订阅'
          value={formatNumber(metrics.active_subscriptions)}
          helper='当前有效套餐'
          icon={CreditCard}
          tone='blue'
          loading={billingQuery.isLoading}
        />
        <EnterpriseStatCard
          title='企业可用额度'
          value={formatLogQuota(metrics.total_balance_quota)}
          helper='全部账户汇总'
          icon={WalletCards}
          tone='emerald'
          loading={billingQuery.isLoading}
        />
        <EnterpriseStatCard
          title='本期应收'
          value={formatLogQuota(metrics.period_sell_quota)}
          helper='用量账本应收'
          icon={CircleDollarSign}
          tone='violet'
          loading={billingQuery.isLoading}
        />
        <EnterpriseStatCard
          title='本期应付'
          value={formatLogQuota(metrics.period_cost_quota)}
          helper='供应成本汇总'
          icon={Landmark}
          tone='amber'
          loading={billingQuery.isLoading}
        />
        <EnterpriseStatCard
          title='毛利率'
          value={formatPercent(metrics.gross_margin_rate)}
          helper={formatLogQuota(metrics.period_gross_profit_quota)}
          icon={PieChartIcon}
          tone='violet'
          loading={billingQuery.isLoading}
        />
        <EnterpriseStatCard
          title='待确认结算单'
          value={formatNumber(metrics.draft_settlements)}
          helper='需要财务复核'
          icon={ReceiptText}
          tone={metrics.draft_settlements > 0 ? 'rose' : 'emerald'}
          loading={billingQuery.isLoading}
        />
      </div>

      <div className='grid gap-4 xl:grid-cols-[minmax(0,1.65fr)_minmax(300px,.65fr)]'>
        <div className='space-y-4'>
          <EnterprisePanel
            title='收支与毛利趋势'
            description='基于 Usage Ledger 的应收、应付与毛利数据'
            action={<Badge variant='secondary'>近 30 天</Badge>}
            bodyClassName='h-80 p-3 sm:p-5'
          >
            {trend.length === 0 ? (
              <div className='text-muted-foreground flex h-full items-center justify-center text-sm'>
                当前周期暂无账本趋势数据
              </div>
            ) : (
              <ResponsiveContainer width='100%' height='100%'>
                <BarChart
                  data={trend}
                  margin={{ top: 12, right: 12, left: -18, bottom: 0 }}
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
                    formatter={(value) => formatLogQuota(Number(value ?? 0))}
                  />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Bar
                    dataKey='sell_quota'
                    name='应收'
                    fill='var(--chart-1)'
                    radius={[6, 6, 0, 0]}
                    maxBarSize={34}
                  />
                  <Bar
                    dataKey='cost_quota'
                    name='应付'
                    fill='var(--chart-2)'
                    radius={[6, 6, 0, 0]}
                    maxBarSize={34}
                  />
                  <Bar
                    dataKey='gross_profit_quota'
                    name='毛利'
                    fill='var(--chart-3)'
                    radius={[6, 6, 0, 0]}
                    maxBarSize={34}
                  />
                </BarChart>
              </ResponsiveContainer>
            )}
          </EnterprisePanel>

          <div className='grid gap-4 md:grid-cols-3'>
            <EnterprisePanel
              title='额度使用率'
              description='组织全部账户额度概况'
            >
              <div className='flex items-center gap-4'>
                <div
                  className='relative flex size-24 shrink-0 items-center justify-center rounded-full bg-[conic-gradient(var(--primary)_var(--usage-angle),var(--muted)_0)]'
                  style={
                    {
                      '--usage-angle': `${Math.min(100, usageRate * 100)}%`,
                    } as CSSProperties
                  }
                >
                  <div className='bg-card flex size-18 items-center justify-center rounded-full text-lg font-semibold'>
                    {formatPercent(usageRate)}
                  </div>
                </div>
                <dl className='min-w-0 space-y-2 text-xs'>
                  <div>
                    <dt className='text-muted-foreground'>累计已用</dt>
                    <dd className='font-semibold'>
                      {formatLogQuota(metrics.total_used_quota)}
                    </dd>
                  </div>
                  <div>
                    <dt className='text-muted-foreground'>当前余额</dt>
                    <dd className='font-semibold'>
                      {formatLogQuota(metrics.total_balance_quota)}
                    </dd>
                  </div>
                </dl>
              </div>
            </EnterprisePanel>

            <EnterprisePanel
              title='充值回款'
              description='本期成功与待处理充值'
            >
              <div className='space-y-3'>
                <div className='flex items-end justify-between'>
                  <div>
                    <p className='text-2xl font-semibold'>
                      {formatPercent(collectionProgress)}
                    </p>
                    <p className='text-muted-foreground text-[11px]'>
                      成功回款占比
                    </p>
                  </div>
                  <Scale className='size-8 text-emerald-500/80' />
                </div>
                <div className='bg-muted h-2 overflow-hidden rounded-full'>
                  <div
                    className='h-full rounded-full bg-emerald-500'
                    style={{
                      width: `${Math.min(100, collectionProgress * 100)}%`,
                    }}
                  />
                </div>
                <div className='grid grid-cols-2 gap-2 text-xs'>
                  <div className='bg-muted/40 rounded-lg p-2'>
                    <p className='text-muted-foreground'>已完成</p>
                    <p className='mt-1 font-semibold'>
                      {formatCurrencyUSD(metrics.successful_top_up_amount)}
                    </p>
                  </div>
                  <div className='bg-muted/40 rounded-lg p-2'>
                    <p className='text-muted-foreground'>待处理</p>
                    <p className='mt-1 font-semibold'>
                      {formatCurrencyUSD(metrics.pending_top_up_amount)}
                    </p>
                  </div>
                </div>
              </div>
            </EnterprisePanel>

            <EnterprisePanel title='结算毛利' description='最近结算单累计毛利'>
              <div className='flex h-full flex-col justify-between gap-4'>
                <div>
                  <p className='text-2xl font-semibold tracking-tight'>
                    {formatLogQuota(settlementGrossProfit)}
                  </p>
                  <p className='text-muted-foreground mt-1 text-xs'>
                    {data.settlements.length} 张结算单
                  </p>
                </div>
                <div className='text-muted-foreground rounded-xl border border-violet-500/15 bg-violet-500/5 p-3 text-[11px] leading-5'>
                  毛利来自结算单应收减应付，不包含支付通道手续费。
                </div>
              </div>
            </EnterprisePanel>
          </div>
        </div>

        <div className='space-y-4'>
          <EnterprisePanel title='预算与结算预警'>
            <div className='space-y-3'>
              {usageRate >= 0.8 && (
                <div className='rounded-xl border border-rose-500/15 bg-rose-500/5 p-3.5'>
                  <p className='flex items-center gap-2 text-xs font-semibold text-rose-600'>
                    <AlertTriangle className='size-4' />
                    组织额度使用率超过 80%
                  </p>
                  <p className='text-muted-foreground mt-1.5 text-[11px] leading-5'>
                    建议检查高用量客户并补充企业余额。
                  </p>
                </div>
              )}
              {metrics.draft_settlements > 0 && (
                <div className='rounded-xl border border-amber-500/15 bg-amber-500/5 p-3.5'>
                  <p className='flex items-center gap-2 text-xs font-semibold text-amber-600'>
                    <FileText className='size-4' />
                    {metrics.draft_settlements} 张结算单待确认
                  </p>
                  <p className='text-muted-foreground mt-1.5 text-[11px] leading-5'>
                    请财务复核应收、应付和请求明细。
                  </p>
                </div>
              )}
              {usageRate < 0.8 && metrics.draft_settlements === 0 && (
                <div className='rounded-xl border border-emerald-500/15 bg-emerald-500/5 p-3.5'>
                  <p className='flex items-center gap-2 text-xs font-semibold text-emerald-600'>
                    <Sparkles className='size-4' />
                    当前财务状态稳定
                  </p>
                  <p className='text-muted-foreground mt-1.5 text-[11px] leading-5'>
                    额度和结算流程暂未发现高风险事项。
                  </p>
                </div>
              )}
            </div>
          </EnterprisePanel>

          <EnterprisePanel title='最近充值流水'>
            <div className='space-y-2.5'>
              {data.recent_topups.slice(0, 6).map((topup) => (
                <div
                  key={topup.id}
                  className='bg-muted/35 flex items-center justify-between gap-3 rounded-xl p-3'
                >
                  <div className='min-w-0'>
                    <p className='truncate text-xs font-semibold'>
                      {topup.username || `用户 #${topup.user_id}`}
                    </p>
                    <p className='text-muted-foreground mt-0.5 truncate text-[10px]'>
                      {topup.payment_provider ||
                        topup.payment_method ||
                        '内部充值'}{' '}
                      · {formatDate(topup.create_time)}
                    </p>
                  </div>
                  <div className='shrink-0 text-right'>
                    <p className='text-xs font-semibold'>
                      {formatCurrencyUSD(topup.money)}
                    </p>
                    <Badge variant='outline' className='mt-1 text-[9px]'>
                      {topup.status || '未知'}
                    </Badge>
                  </div>
                </div>
              ))}
              {data.recent_topups.length === 0 && (
                <p className='text-muted-foreground py-8 text-center text-sm'>
                  暂无充值流水
                </p>
              )}
            </div>
          </EnterprisePanel>

          <EnterprisePanel title='快捷操作'>
            <div className='grid grid-cols-2 gap-2'>
              <Button
                variant='outline'
                className='h-auto flex-col gap-2 py-3'
                render={<Link to='/wallet' />}
              >
                <WalletCards className='size-4 text-emerald-500' />
                <span className='text-xs'>企业充值</span>
              </Button>
              <Button
                variant='outline'
                className='h-auto flex-col gap-2 py-3'
                onClick={() => setSettlementDialogOpen(true)}
              >
                <ReceiptText className='size-4 text-violet-500' />
                <span className='text-xs'>生成结算单</span>
              </Button>
              <Button
                variant='outline'
                className='h-auto flex-col gap-2 py-3'
                onClick={() => void exportBillingCsv()}
                disabled={exportingBilling}
              >
                <Download className='size-4 text-blue-500' />
                <span className='text-xs'>
                  {exportingBilling ? '导出中' : '导出账单'}
                </span>
              </Button>
              <Button
                variant='outline'
                className='h-auto flex-col gap-2 py-3'
                onClick={openFeeAnalysis}
              >
                <BadgeDollarSign className='size-4 text-amber-500' />
                <span className='text-xs'>费用分析</span>
              </Button>
            </div>
          </EnterprisePanel>
        </div>
      </div>

      <EnterprisePanel
        title='结算与账单明细'
        description='客户和供应商结算单统一视图'
        action={
          <Badge variant='outline'>共 {data.settlements.length} 条</Badge>
        }
        bodyClassName='p-0'
      >
        <SettlementTable items={data.settlements} />
      </EnterprisePanel>

      {props.classicContent && (
        <EnterprisePanel
          title='订阅计划管理'
          description='保留原有套餐创建、支付平台关联、启停和编辑能力'
          bodyClassName='min-h-[520px] p-0'
        >
          {props.classicContent}
        </EnterprisePanel>
      )}

      <Dialog
        open={settlementDialogOpen}
        onOpenChange={setSettlementDialogOpen}
      >
        <DialogContent className='sm:max-w-lg'>
          <DialogHeader>
            <DialogTitle>生成结算单</DialogTitle>
            <DialogDescription>
              按客户或供应商生成当前周期结算单，重复生成会覆盖同周期草稿数据。
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4'>
            <label className='grid gap-2 text-sm'>
              <span className='font-medium'>结算对象</span>
              <NativeSelect
                value={settlementSubjectType}
                onChange={(event) =>
                  setSettlementSubjectType(
                    event.target.value as 'user' | 'supplier'
                  )
                }
              >
                <NativeSelectOption value='user'>
                  客户 / 下游用户
                </NativeSelectOption>
                <NativeSelectOption value='supplier'>
                  供应商 / 上游
                </NativeSelectOption>
              </NativeSelect>
            </label>
            <label className='grid gap-2 text-sm'>
              <span className='font-medium'>
                {settlementSubjectType === 'supplier' ? '供应商 ID' : '用户 ID'}
              </span>
              <Input
                inputMode='numeric'
                value={settlementSubjectId}
                onChange={(event) => setSettlementSubjectId(event.target.value)}
                placeholder={
                  settlementSubjectType === 'supplier' ? '例如 12' : '例如 1001'
                }
              />
            </label>
            <div className='grid gap-3 sm:grid-cols-2'>
              <label className='grid gap-2 text-sm'>
                <span className='font-medium'>周期开始</span>
                <Input
                  type='date'
                  value={settlementPeriodStart}
                  onChange={(event) =>
                    setSettlementPeriodStart(event.target.value)
                  }
                />
              </label>
              <label className='grid gap-2 text-sm'>
                <span className='font-medium'>周期结束</span>
                <Input
                  type='date'
                  value={settlementPeriodEnd}
                  onChange={(event) =>
                    setSettlementPeriodEnd(event.target.value)
                  }
                />
              </label>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => setSettlementDialogOpen(false)}
              disabled={generatingSettlement}
            >
              取消
            </Button>
            <Button
              onClick={() => void submitSettlement()}
              disabled={generatingSettlement}
            >
              {generatingSettlement ? '生成中' : '生成结算单'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
