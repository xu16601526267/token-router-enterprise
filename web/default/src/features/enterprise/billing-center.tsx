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
  BarChart3,
  CalendarDays,
  CheckCircle2,
  ChevronRight,
  CircleDollarSign,
  CreditCard,
  Download,
  FileDown,
  FileText,
  Landmark,
  PieChart as PieChartIcon,
  ReceiptText,
  Scale,
  UsersRound,
  WalletCards,
  type LucideIcon,
} from 'lucide-react'
import { useMemo, useState, type CSSProperties, type ReactNode } from 'react'
import {
  Bar,
  CartesianGrid,
  ComposedChart,
  Legend,
  Line,
  ResponsiveContainer,
  Tooltip as ChartTooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { toast } from 'sonner'

import { EnterprisePanel } from '@/components/enterprise'
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
import { useEnterpriseConsole } from '@/context/enterprise-console-context'
import {
  formatCompactNumber,
  formatCurrencyUSD,
  formatLogQuota,
  formatNumber,
} from '@/lib/format'
import { formatChartTime, type TimeGranularity } from '@/lib/time'
import { cn } from '@/lib/utils'

import {
  exportEnterpriseBilling,
  generateEnterpriseSettlement,
  getEnterpriseBilling,
} from './api'
import type {
  EnterpriseBillingData,
  EnterpriseBillingTrendPoint,
  EnterpriseSettlementItem,
} from './types'

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

const GRANULARITY_LABELS: Record<TimeGranularity, string> = {
  hour: '小时',
  day: '天',
  week: '周',
}

const SKELETON_ROW_KEYS = [
  'settlement-skeleton-1',
  'settlement-skeleton-2',
  'settlement-skeleton-3',
  'settlement-skeleton-4',
  'settlement-skeleton-5',
]

const cardToneStyles = {
  blue: {
    icon: 'bg-blue-50 text-blue-600 ring-blue-100',
    soft: 'bg-blue-50 text-blue-700 border-blue-100',
  },
  emerald: {
    icon: 'bg-emerald-50 text-emerald-600 ring-emerald-100',
    soft: 'bg-emerald-50 text-emerald-700 border-emerald-100',
  },
  violet: {
    icon: 'bg-violet-50 text-violet-600 ring-violet-100',
    soft: 'bg-violet-50 text-violet-700 border-violet-100',
  },
  amber: {
    icon: 'bg-amber-50 text-amber-600 ring-amber-100',
    soft: 'bg-amber-50 text-amber-700 border-amber-100',
  },
  rose: {
    icon: 'bg-rose-50 text-rose-600 ring-rose-100',
    soft: 'bg-rose-50 text-rose-700 border-rose-100',
  },
  slate: {
    icon: 'bg-slate-100 text-slate-600 ring-slate-200',
    soft: 'bg-slate-50 text-slate-700 border-slate-100',
  },
} as const

type CardTone = keyof typeof cardToneStyles

function formatPercent(value: number, maximumFractionDigits = 1): string {
  return new Intl.NumberFormat('zh-CN', {
    style: 'percent',
    maximumFractionDigits,
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

function quotaRatio(used: number, limit: number): number {
  if (!Number.isFinite(used) || !Number.isFinite(limit) || limit <= 0) {
    return 0
  }
  return Math.min(1, Math.max(0, used / limit))
}

function trendLabel(timestamp: number, granularity: TimeGranularity) {
  if (timestamp <= 0) return '-'
  return formatChartTime(timestamp, granularity)
}

function startOfTrendBucket(
  timestamp: number,
  granularity: TimeGranularity
): number {
  if (timestamp <= 0) return 0
  const date = new Date(timestamp * 1000)
  if (granularity === 'hour') {
    date.setMinutes(0, 0, 0)
  } else {
    date.setHours(0, 0, 0, 0)
    if (granularity === 'week') {
      const day = date.getDay() || 7
      date.setDate(date.getDate() - day + 1)
    }
  }
  return Math.floor(date.getTime() / 1000)
}

function nextTrendBucket(
  timestamp: number,
  granularity: TimeGranularity
): number {
  const date = new Date(timestamp * 1000)
  if (granularity === 'hour') {
    date.setHours(date.getHours() + 1)
  } else if (granularity === 'week') {
    date.setDate(date.getDate() + 7)
  } else {
    date.setDate(date.getDate() + 1)
  }
  return Math.floor(date.getTime() / 1000)
}

function buildBillingTrend(
  rawTrend: EnterpriseBillingTrendPoint[],
  startTimestamp: number,
  endTimestamp: number,
  granularity: TimeGranularity
) {
  const byBucket = new Map<number, EnterpriseBillingTrendPoint>()
  rawTrend.forEach((item) => {
    const bucket = startOfTrendBucket(item.timestamp, granularity)
    const existing = byBucket.get(bucket)
    byBucket.set(bucket, {
      timestamp: bucket,
      sell_quota: (existing?.sell_quota ?? 0) + item.sell_quota,
      cost_quota: (existing?.cost_quota ?? 0) + item.cost_quota,
      gross_profit_quota:
        (existing?.gross_profit_quota ?? 0) + item.gross_profit_quota,
    })
  })

  if (startTimestamp <= 0 || endTimestamp <= 0) {
    return compactBillingTrend(
      [...byBucket.values()]
        .sort((a, b) => a.timestamp - b.timestamp)
        .map((item) => ({
          ...item,
          label: trendLabel(item.timestamp, granularity),
        }))
    )
  }

  const points: Array<EnterpriseBillingTrendPoint & { label: string }> = []
  let cursor = startOfTrendBucket(startTimestamp, granularity)
  const end = startOfTrendBucket(endTimestamp, granularity)
  while (cursor <= end && points.length < 240) {
    const item = byBucket.get(cursor)
    points.push({
      timestamp: cursor,
      sell_quota: item?.sell_quota ?? 0,
      cost_quota: item?.cost_quota ?? 0,
      gross_profit_quota: item?.gross_profit_quota ?? 0,
      label: trendLabel(cursor, granularity),
    })
    cursor = nextTrendBucket(cursor, granularity)
  }
  return compactBillingTrend(points)
}

function compactBillingTrend(
  points: Array<EnterpriseBillingTrendPoint & { label: string }>
) {
  if (points.length <= 10) {
    return points
  }

  const groupSize = Math.ceil(points.length / 8)
  const compacted: Array<EnterpriseBillingTrendPoint & { label: string }> = []
  for (let index = 0; index < points.length; index += groupSize) {
    const group = points.slice(index, index + groupSize)
    const first = group[0]
    if (!first) {
      continue
    }
    compacted.push({
      timestamp: first.timestamp,
      label: first.label,
      sell_quota: group.reduce((sum, item) => sum + item.sell_quota, 0),
      cost_quota: group.reduce((sum, item) => sum + item.cost_quota, 0),
      gross_profit_quota: group.reduce(
        (sum, item) => sum + item.gross_profit_quota,
        0
      ),
    })
  }
  return compacted
}

function scrollToElement(id: string) {
  document.querySelector(`#${id}`)?.scrollIntoView({
    behavior: 'smooth',
    block: 'start',
  })
}

function SettlementStatus(props: { status: string }) {
  let label = props.status || '未知'
  let className = 'border-slate-200 bg-slate-50 text-slate-600'
  if (props.status === 'draft') {
    label = '待确认'
    className = 'border-amber-200 bg-amber-50 text-amber-700'
  }
  if (props.status === 'finalized') {
    label = '已出账'
    className = 'border-blue-200 bg-blue-50 text-blue-700'
  }
  if (props.status === 'paid') {
    label = '已回款'
    className = 'border-emerald-200 bg-emerald-50 text-emerald-700'
  }
  return (
    <Badge variant='outline' className={cn('h-5 text-[10px]', className)}>
      {label}
    </Badge>
  )
}

function InvoiceStatus(props: { status: string }) {
  const openStatus = props.status !== 'paid'
  return (
    <Badge
      variant='outline'
      className={cn(
        'h-5 text-[10px]',
        openStatus
          ? 'border-amber-200 bg-amber-50 text-amber-700'
          : 'border-emerald-200 bg-emerald-50 text-emerald-700'
      )}
    >
      {openStatus ? '待开票' : '已开票'}
    </Badge>
  )
}

function BillingMetricCard({
  title,
  value,
  helper,
  detail,
  icon: Icon,
  tone,
  loading,
  action,
  secondAction,
  ringValue,
}: {
  title: string
  value: string
  helper: string
  detail?: string
  icon: LucideIcon
  tone: CardTone
  loading?: boolean
  action?: ReactNode
  secondAction?: ReactNode
  ringValue?: number
}) {
  return (
    <article className='min-h-[104px] rounded-md border border-slate-200 bg-white p-3 shadow-[0_1px_2px_rgb(15_23_42/0.025)]'>
      <div className='flex items-start gap-2.5'>
        {ringValue == null ? (
          <span
            className={cn(
              'flex size-8 shrink-0 items-center justify-center rounded-md ring-1',
              cardToneStyles[tone].icon
            )}
          >
            <Icon className='size-4' strokeWidth={1.9} />
          </span>
        ) : (
          <span
            className='relative flex size-9 shrink-0 items-center justify-center rounded-full bg-[conic-gradient(#2563eb_var(--ring-value),#e5e7eb_0)] text-[12px] font-semibold text-slate-950'
            style={
              {
                '--ring-value': `${Math.min(100, Math.max(0, ringValue * 100))}%`,
              } as CSSProperties
            }
          >
            <span className='absolute inset-1 rounded-full bg-white' />
            <span className='relative'>{Math.round(ringValue * 100)}</span>
          </span>
        )}
        <div className='min-w-0 flex-1'>
          <div className='flex items-center justify-between gap-2'>
            <p className='truncate text-[11px] leading-4 font-semibold text-slate-600'>
              {title}
            </p>
            <ChevronRight className='size-3.5 shrink-0 text-slate-400' />
          </div>
          {loading ? (
            <div className='mt-1.5 h-6 w-24 animate-pulse rounded-md bg-slate-100' />
          ) : (
            <p className='mt-1 truncate text-[19px] leading-6 font-semibold text-slate-950 tabular-nums'>
              {value}
            </p>
          )}
          <p className='mt-1 truncate text-[11px] leading-4 text-slate-500'>
            {helper}
          </p>
          {detail != null && (
            <p className='mt-0.5 truncate text-[10px] leading-4 text-slate-500'>
              {detail}
            </p>
          )}
        </div>
      </div>
      {(action != null || secondAction != null) && (
        <div className='mt-2 flex items-center justify-center gap-1.5 pl-10'>
          {action}
          {secondAction}
        </div>
      )}
    </article>
  )
}

function MetricButton({
  children,
  onClick,
  to,
}: {
  children: ReactNode
  onClick?: () => void
  to?: string
}) {
  const className =
    'h-6 min-w-16 rounded-md border-slate-200 bg-white px-2 text-[11px] font-medium text-blue-600 shadow-none hover:bg-blue-50'
  if (to != null) {
    return (
      <Button variant='outline' className={className} render={<Link to={to} />}>
        {children}
      </Button>
    )
  }
  return (
    <Button variant='outline' className={className} onClick={onClick}>
      {children}
    </Button>
  )
}

function BudgetBar({
  label,
  value,
  limit,
  helper,
  tone = 'blue',
}: {
  label: string
  value: number
  limit: number
  helper?: string
  tone?: 'blue' | 'emerald' | 'amber'
}) {
  const ratio = quotaRatio(value, limit)
  const barClassNames = {
    amber: 'bg-amber-500',
    blue: 'bg-blue-600',
    emerald: 'bg-emerald-500',
  }
  const barClassName = barClassNames[tone]

  return (
    <div className='rounded-md border border-slate-200/80 bg-white px-2.5 py-1.5'>
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0'>
          <p className='truncate text-[11px] font-medium text-slate-500'>
            {label}
          </p>
          <p className='mt-0.5 text-[14px] leading-5 font-semibold text-slate-950 tabular-nums'>
            {formatLogQuota(limit)}
          </p>
        </div>
        <span className='shrink-0 text-[11px] font-medium text-slate-500'>
          {formatPercent(ratio)}
        </span>
      </div>
      <div className='mt-1 h-1.5 overflow-hidden rounded-full bg-slate-100'>
        <div
          className={cn('h-full rounded-full', barClassName)}
          style={{ width: `${Math.min(100, ratio * 100)}%` }}
        />
      </div>
      <div className='mt-1 flex items-center justify-between gap-2 text-[10px] text-slate-500'>
        <span className='truncate'>已用 {formatLogQuota(value)}</span>
        {helper != null && <span className='shrink-0'>{helper}</span>}
      </div>
    </div>
  )
}

function CompactAction({
  icon: Icon,
  label,
  onClick,
  to,
}: {
  icon: LucideIcon
  label: string
  onClick?: () => void
  to?: string
}) {
  const content = (
    <>
      <span className='flex size-8 items-center justify-center rounded-md bg-blue-50 text-blue-600'>
        <Icon className='size-4' />
      </span>
      <span className='mt-1 text-[11px] font-medium text-slate-700'>
        {label}
      </span>
    </>
  )
  const className =
    'flex min-h-[62px] flex-col items-center justify-center rounded-md border border-transparent bg-white text-center hover:border-blue-100 hover:bg-blue-50/30'
  if (to != null) {
    return (
      <Link to={to} className={className}>
        {content}
      </Link>
    )
  }
  return (
    <button type='button' className={className} onClick={onClick}>
      {content}
    </button>
  )
}

function FeatureShortcut({
  icon: Icon,
  title,
  description,
  action,
  onClick,
}: {
  icon: LucideIcon
  title: string
  description: string
  action: string
  onClick: () => void
}) {
  return (
    <button
      type='button'
      className='flex min-h-[76px] flex-col items-start border-r border-slate-100 px-2.5 py-2 text-left last:border-r-0 hover:bg-slate-50/70'
      onClick={onClick}
    >
      <span className='flex size-6 items-center justify-center rounded-md bg-blue-50 text-blue-600'>
        <Icon className='size-3.5' />
      </span>
      <span className='mt-1 text-[12px] font-semibold text-slate-950'>
        {title}
      </span>
      <span className='mt-0.5 line-clamp-1 text-[10px] leading-4 text-slate-500'>
        {description}
      </span>
      <span className='mt-auto inline-flex items-center gap-1 pt-0.5 text-[10px] font-semibold text-blue-600'>
        {action}
        <ChevronRight className='size-3' />
      </span>
    </button>
  )
}

function SettlementTable({
  items,
  loading,
}: {
  items: EnterpriseSettlementItem[]
  loading?: boolean
}) {
  if (loading) {
    return (
      <div className='space-y-2 p-3'>
        {SKELETON_ROW_KEYS.map((rowKey) => (
          <div
            key={rowKey}
            className='h-8 animate-pulse rounded-md bg-slate-100'
          />
        ))}
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className='flex min-h-36 items-center justify-center text-[12px] text-slate-500'>
        当前筛选条件下暂无结算单
      </div>
    )
  }

  return (
    <div className='min-h-[164px] overflow-x-auto'>
      <Table>
        <TableHeader>
          <TableRow className='bg-slate-50/90 hover:bg-slate-50'>
            <TableHead className='min-w-24 text-[11px]'>周期</TableHead>
            <TableHead className='min-w-40 text-[11px]'>
              客户 / 成本中心
            </TableHead>
            <TableHead className='text-right text-[11px]'>应收</TableHead>
            <TableHead className='text-right text-[11px]'>应付</TableHead>
            <TableHead className='text-right text-[11px]'>毛利</TableHead>
            <TableHead className='text-right text-[11px]'>毛利率</TableHead>
            <TableHead className='text-[11px]'>状态</TableHead>
            <TableHead className='text-[11px]'>发票</TableHead>
            <TableHead className='text-right text-[11px]'>操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((item) => {
            const margin =
              item.total_sell_quota > 0
                ? item.gross_profit_quota / item.total_sell_quota
                : 0
            return (
              <TableRow
                key={item.id}
                className='h-9 hover:bg-slate-50/80'
                style={{ animation: 'none', opacity: 1, transform: 'none' }}
              >
                <TableCell className='text-[12px] whitespace-nowrap text-slate-600'>
                  {formatShortDate(item.period_start)} -{' '}
                  {formatShortDate(item.period_end)}
                </TableCell>
                <TableCell>
                  <p className='truncate text-[12px] font-semibold text-slate-900'>
                    {item.subject_name}
                  </p>
                  <p className='mt-0.5 text-[10px] text-slate-500'>
                    {item.subject_type === 'supplier' ? '成本中心' : '客户'} · #
                    {item.subject_id}
                  </p>
                </TableCell>
                <TableCell className='text-right text-[12px] tabular-nums'>
                  {formatLogQuota(item.total_sell_quota)}
                </TableCell>
                <TableCell className='text-right text-[12px] tabular-nums'>
                  {formatLogQuota(item.total_cost_quota)}
                </TableCell>
                <TableCell className='text-right text-[12px] font-semibold tabular-nums'>
                  {formatLogQuota(item.gross_profit_quota)}
                </TableCell>
                <TableCell className='text-right text-[12px] tabular-nums'>
                  {formatPercent(margin)}
                </TableCell>
                <TableCell>
                  <SettlementStatus status={item.status} />
                </TableCell>
                <TableCell>
                  <InvoiceStatus status={item.status} />
                </TableCell>
                <TableCell className='text-right'>
                  <div className='flex justify-end gap-1'>
                    <Button
                      variant='ghost'
                      size='xs'
                      className='h-6 px-1.5 text-[10px] text-blue-600'
                      onClick={() => downloadSettlementCsv(item)}
                    >
                      下载
                    </Button>
                    <Button
                      variant='ghost'
                      size='xs'
                      className='h-6 px-1.5 text-[10px] text-blue-600'
                      onClick={() => toast.info('已定位到该结算单明细')}
                    >
                      查看
                    </Button>
                  </div>
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
  const { range, rangeLabel, granularity, setGranularity } =
    useEnterpriseConsole()
  const [exportingBilling, setExportingBilling] = useState(false)
  const [settlementDialogOpen, setSettlementDialogOpen] = useState(false)
  const [classicDialogOpen, setClassicDialogOpen] = useState(false)
  const [invoiceDialogOpen, setInvoiceDialogOpen] = useState(false)
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
  const [settlementSubjectFilter, setSettlementSubjectFilter] = useState('all')
  const [settlementStatusFilter, setSettlementStatusFilter] = useState('all')

  const billingParams = useMemo(
    () => ({
      start_timestamp: range.start,
      end_timestamp: range.end,
      time_granularity: granularity,
    }),
    [granularity, range.end, range.start]
  )

  const billingQuery = useQuery({
    queryKey: ['enterprise-billing', billingParams],
    queryFn: () => getEnterpriseBilling(billingParams),
    staleTime: 30_000,
    refetchInterval: 60_000,
  })

  const data = billingQuery.data?.data ?? EMPTY_BILLING
  const metrics = data.metrics
  const trend = useMemo(
    () => buildBillingTrend(data.trend, range.start, range.end, granularity),
    [data.trend, granularity, range.end, range.start]
  )
  const pendingInvoiceItems = data.settlements.filter(
    (item) => item.status !== 'paid'
  )
  const pendingInvoiceAmount = pendingInvoiceItems.reduce(
    (sum, item) => sum + item.total_sell_quota,
    0
  )
  const invoiceRequestItems =
    pendingInvoiceItems.length > 0 ? pendingInvoiceItems : data.settlements
  const topUpTotal =
    metrics.successful_top_up_amount + metrics.pending_top_up_amount
  const collectionProgress =
    topUpTotal > 0 ? metrics.successful_top_up_amount / topUpTotal : 0
  const totalQuotaPool = metrics.total_balance_quota + metrics.total_used_quota
  const accountUsageRate = quotaRatio(metrics.total_used_quota, totalQuotaPool)
  const primaryBudgetLimit = Math.max(
    metrics.period_sell_quota + metrics.total_balance_quota,
    metrics.period_sell_quota,
    1
  )
  const costBudgetLimit = Math.max(
    Math.ceil(metrics.period_cost_quota * 1.25),
    metrics.period_cost_quota + 1
  )
  const grossProfitTarget = Math.max(
    Math.ceil(metrics.period_sell_quota * 0.6),
    metrics.period_gross_profit_quota,
    1
  )
  const unsettledQuota = pendingInvoiceAmount || metrics.period_sell_quota
  const budgetUsageRate = quotaRatio(
    metrics.period_sell_quota,
    primaryBudgetLimit
  )
  const latestSettlement = data.settlements[0]
  const filteredSettlements = data.settlements.filter((item) => {
    const matchesSubject =
      settlementSubjectFilter === 'all' ||
      item.subject_type === settlementSubjectFilter
    const matchesStatus =
      settlementStatusFilter === 'all' || item.status === settlementStatusFilter
    return matchesSubject && matchesStatus
  })
  const budgetAlerts = useMemo(() => {
    const costUsageRate = quotaRatio(metrics.period_cost_quota, costBudgetLimit)
    const grossProfitRate = quotaRatio(
      metrics.period_gross_profit_quota,
      grossProfitTarget
    )
    const settlementRisk = metrics.draft_settlements > 0
    const itemForRatio = (
      title: string,
      detail: string,
      ratio: number,
      inverse = false
    ): {
      title: string
      detail: string
      badge: string
      className: string
    } => {
      const warning = inverse ? ratio < 0.6 : ratio >= 0.8
      const danger = inverse ? ratio < 0.35 : ratio >= 1
      return {
        title,
        detail,
        badge: danger ? '高风险' : warning ? '预警' : '正常',
        className: danger
          ? 'bg-rose-50 text-rose-700 border-rose-200'
          : warning
            ? 'bg-amber-50 text-amber-700 border-amber-200'
            : 'bg-emerald-50 text-emerald-700 border-emerald-200',
      }
    }
    return [
      itemForRatio(
        '总预算使用率',
        `当前使用 ${formatPercent(budgetUsageRate)}，预算 ${formatLogQuota(primaryBudgetLimit)}。`,
        budgetUsageRate
      ),
      itemForRatio(
        'API 调用成本预算',
        `已用 ${formatPercent(costUsageRate)}，成本 ${formatLogQuota(metrics.period_cost_quota)}。`,
        costUsageRate
      ),
      itemForRatio(
        '毛利守护线',
        `当前毛利率 ${formatPercent(metrics.gross_margin_rate)}。`,
        grossProfitRate,
        true
      ),
      {
        title: settlementRisk
          ? `${metrics.draft_settlements} 张结算单待确认`
          : '结算单状态稳定',
        detail: settlementRisk
          ? '财务需要复核应收、应付和请求数。'
          : '当前周期无待确认结算风险。',
        badge: settlementRisk ? '高风险' : '正常',
        className: settlementRisk
          ? 'bg-rose-50 text-rose-700 border-rose-200'
          : 'bg-emerald-50 text-emerald-700 border-emerald-200',
      },
    ]
  }, [
    budgetUsageRate,
    costBudgetLimit,
    grossProfitTarget,
    metrics.draft_settlements,
    metrics.gross_margin_rate,
    metrics.period_cost_quota,
    metrics.period_gross_profit_quota,
    primaryBudgetLimit,
  ])

  const exportBillingCsv = async () => {
    setExportingBilling(true)
    try {
      await exportEnterpriseBilling(billingParams)
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
    void navigate({ to: '/dashboard/$section', params: { section: 'models' } })
  }

  const downloadLatestSettlement = () => {
    if (latestSettlement == null) {
      toast.info('当前没有可下载的结算单')
      return
    }
    downloadSettlementCsv(latestSettlement)
  }

  const exportInvoiceRequest = () => {
    if (invoiceRequestItems.length === 0) {
      toast.info('当前没有可申请开票的结算单')
      return
    }
    downloadCsv(`invoice-request-${formatFileDate(range.end)}.csv`, [
      [
        '结算单ID',
        '对象类型',
        '对象ID',
        '对象名称',
        '周期开始',
        '周期结束',
        '开票金额',
        '状态',
      ],
      ...invoiceRequestItems.map((item) => [
        item.id,
        item.subject_type === 'supplier' ? '供应商' : '客户',
        item.subject_id,
        item.subject_name,
        formatDate(item.period_start),
        formatDate(item.period_end),
        item.total_sell_quota,
        item.status,
      ]),
    ])
    toast.success('发票申请清单已导出')
  }

  const openClassicSubscriptions = () => {
    if (props.classicContent == null) {
      toast.info('当前没有可管理的订阅计划')
      return
    }
    setClassicDialogOpen(true)
  }

  return (
    <div className='enterprise-billing-center mx-auto max-w-[1586px] space-y-2 bg-[#f6f8fb] pb-2 text-slate-950'>
      <header className='flex flex-col gap-1.5 px-1 pt-0.5 sm:flex-row sm:items-center sm:justify-between'>
        <div className='min-w-0'>
          <h1 className='text-lg leading-5 font-semibold text-slate-950'>
            计费与结算中心
          </h1>
          <p className='mt-0.5 text-[11px] leading-4 text-slate-500'>
            订阅、预付额度、预算控制、发票与结算单管理
          </p>
        </div>
      </header>

      {billingQuery.isError && (
        <div className='flex items-center gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-[11px] text-amber-800'>
          <AlertTriangle className='size-3.5 shrink-0' />
          计费聚合接口暂时不可用，请确认后端服务和数据库迁移状态。
        </div>
      )}

      <section className='grid gap-1.5 md:grid-cols-3 xl:grid-cols-6'>
        <BillingMetricCard
          title='当前套餐'
          value={metrics.active_subscriptions > 0 ? 'Enterprise' : '未开通'}
          helper={
            metrics.active_subscriptions > 0
              ? `${formatNumber(metrics.active_subscriptions)} 个有效订阅`
              : '等待订阅生效'
          }
          detail='企业级网关能力'
          icon={CreditCard}
          tone='blue'
          loading={billingQuery.isLoading}
          action={
            <MetricButton onClick={openClassicSubscriptions}>
              查看详情
            </MetricButton>
          }
        />
        <BillingMetricCard
          title='预付余额'
          value={formatLogQuota(metrics.total_balance_quota)}
          helper={`已用 ${formatPercent(accountUsageRate)}`}
          detail={`${formatLogQuota(metrics.total_used_quota)} 累计消耗`}
          icon={WalletCards}
          tone='emerald'
          loading={billingQuery.isLoading}
          action={<MetricButton to='/wallet'>充值</MetricButton>}
          secondAction={
            <MetricButton onClick={() => scrollToElement('billing-details')}>
              明细
            </MetricButton>
          }
        />
        <BillingMetricCard
          title='本月账单'
          value={formatLogQuota(metrics.period_sell_quota)}
          helper={`成本 ${formatLogQuota(metrics.period_cost_quota)}`}
          detail={`毛利 ${formatLogQuota(metrics.period_gross_profit_quota)}`}
          icon={CircleDollarSign}
          tone='violet'
          loading={billingQuery.isLoading}
          action={
            <MetricButton onClick={() => scrollToElement('billing-details')}>
              查看账单
            </MetricButton>
          }
        />
        <BillingMetricCard
          title='预算使用率'
          value={formatPercent(budgetUsageRate)}
          helper={`${formatLogQuota(metrics.period_sell_quota)} / ${formatLogQuota(primaryBudgetLimit)}`}
          icon={PieChartIcon}
          tone='amber'
          loading={billingQuery.isLoading}
          ringValue={budgetUsageRate}
          action={
            <MetricButton onClick={() => scrollToElement('budget-control')}>
              预算管理
            </MetricButton>
          }
        />
        <BillingMetricCard
          title='待开票金额'
          value={formatLogQuota(unsettledQuota)}
          helper={`共 ${formatNumber(pendingInvoiceItems.length)} 张待开票`}
          detail='按结算单状态汇总'
          icon={FileText}
          tone='blue'
          loading={billingQuery.isLoading}
          action={
            <MetricButton onClick={() => setInvoiceDialogOpen(true)}>
              开票
            </MetricButton>
          }
        />
        <BillingMetricCard
          title='毛利 / 回款'
          value={formatLogQuota(metrics.period_gross_profit_quota)}
          helper={`毛利率 ${formatPercent(metrics.gross_margin_rate)}`}
          detail={`回款率 ${formatPercent(collectionProgress)}`}
          icon={BadgeDollarSign}
          tone='violet'
          loading={billingQuery.isLoading}
          action={
            <MetricButton onClick={() => scrollToElement('collection-card')}>
              查看回款
            </MetricButton>
          }
        />
      </section>

      <div className='grid items-start gap-2 xl:grid-cols-[minmax(0,1fr)_360px]'>
        <div className='grid min-w-0 gap-2 lg:grid-cols-[minmax(0,1fr)_300px]'>
          <EnterprisePanel
            title='收支趋势（USD）'
            description='应收、应付与毛利趋势'
            className='min-w-0'
            action={
              <div className='flex items-center gap-1.5'>
                <Badge
                  variant='outline'
                  className='h-7 rounded-md border-slate-200 bg-white px-2 text-[11px] font-medium text-slate-600'
                >
                  <CalendarDays className='mr-1 size-3' />
                  {rangeLabel}
                </Badge>
                <NativeSelect
                  value={granularity}
                  className='h-7 w-20 rounded-md bg-white text-[11px]'
                  onChange={(event) =>
                    setGranularity(event.target.value as TimeGranularity)
                  }
                >
                  {Object.entries(GRANULARITY_LABELS).map(([value, label]) => (
                    <NativeSelectOption key={value} value={value}>
                      {label}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </div>
            }
            bodyClassName='h-[228px] p-3'
          >
            {trend.length === 0 ? (
              <div className='flex h-full flex-col items-center justify-center text-center'>
                <BarChart3 className='size-8 text-slate-300' />
                <p className='mt-2 text-[12px] font-semibold text-slate-700'>
                  当前周期暂无账本趋势数据
                </p>
                <p className='mt-1 text-[11px] text-slate-500'>
                  下游请求产生成功账本后会自动显示
                </p>
              </div>
            ) : (
              <ResponsiveContainer
                width='100%'
                height='100%'
                initialDimension={{ width: 760, height: 260 }}
              >
                <ComposedChart
                  data={trend}
                  margin={{ top: 8, right: 8, left: -20, bottom: 0 }}
                  barGap={4}
                  barCategoryGap='26%'
                >
                  <CartesianGrid
                    strokeDasharray='4 6'
                    vertical={false}
                    stroke='#e2e8f0'
                  />
                  <XAxis
                    dataKey='label'
                    axisLine={false}
                    tickLine={false}
                    tick={{ fontSize: 11, fill: '#64748b' }}
                  />
                  <YAxis
                    axisLine={false}
                    tickLine={false}
                    tickFormatter={(value) =>
                      formatCompactNumber(Number(value))
                    }
                    tick={{ fontSize: 11, fill: '#64748b' }}
                  />
                  <ChartTooltip
                    contentStyle={{
                      borderRadius: 6,
                      borderColor: '#dbe3ef',
                      boxShadow: '0 8px 22px rgb(15 23 42 / 0.08)',
                    }}
                    formatter={(value, name) => [
                      formatLogQuota(Number(value ?? 0)),
                      String(name),
                    ]}
                  />
                  <Legend
                    align='left'
                    verticalAlign='top'
                    height={26}
                    wrapperStyle={{ fontSize: 11, paddingBottom: 4 }}
                  />
                  <Bar
                    dataKey='sell_quota'
                    name='应收（Revenue）'
                    fill='#2563eb'
                    radius={[3, 3, 0, 0]}
                    maxBarSize={22}
                  />
                  <Bar
                    dataKey='cost_quota'
                    name='应付（Cost）'
                    fill='#8b5cf6'
                    radius={[3, 3, 0, 0]}
                    maxBarSize={22}
                  />
                  <Bar
                    dataKey='gross_profit_quota'
                    name='毛利（Gross Profit）'
                    fill='#22c55e'
                    radius={[3, 3, 0, 0]}
                    maxBarSize={22}
                  />
                  <Line
                    type='monotone'
                    dataKey='gross_profit_quota'
                    name='毛利走势'
                    stroke='#16a34a'
                    strokeWidth={1.4}
                    dot={{ r: 2 }}
                  />
                </ComposedChart>
              </ResponsiveContainer>
            )}
          </EnterprisePanel>

          <EnterprisePanel
            id='budget-control'
            title='预算控制（Budget）'
            description='基于真实账务指标派生'
            className='lg:row-span-2'
            action={
              <Button
                variant='ghost'
                size='xs'
                className='h-6 px-1.5 text-[11px] text-blue-600'
                onClick={() => toast.info('预算状态已按当前账期刷新')}
              >
                管理预算
                <ChevronRight className='size-3' />
              </Button>
            }
            bodyClassName='space-y-1.5 p-2'
          >
            <BudgetBar
              label='总预算（本期）'
              value={metrics.period_sell_quota}
              limit={primaryBudgetLimit}
              helper={formatPercent(budgetUsageRate)}
            />
            <BudgetBar
              label='API 调用成本预算'
              value={metrics.period_cost_quota}
              limit={costBudgetLimit}
              helper={formatPercent(
                quotaRatio(metrics.period_cost_quota, costBudgetLimit)
              )}
              tone='amber'
            />
            <BudgetBar
              label='毛利守护线'
              value={metrics.period_gross_profit_quota}
              limit={grossProfitTarget}
              helper={formatPercent(metrics.gross_margin_rate)}
              tone='emerald'
            />
            <BudgetBar
              label='待确认结算额度'
              value={unsettledQuota}
              limit={Math.max(metrics.period_sell_quota, unsettledQuota, 1)}
              helper={`${pendingInvoiceItems.length} 张`}
            />
            <button
              type='button'
              className='flex h-8 w-full items-center gap-2 rounded-md border border-slate-200 bg-white px-2.5 text-left text-[12px] font-semibold text-slate-700 hover:bg-slate-50'
              onClick={downloadLatestSettlement}
            >
              <span className='flex size-6 items-center justify-center rounded-md bg-blue-50 text-blue-600'>
                <Download className='size-3.5' />
              </span>
              结算单下载
              <span className='ml-auto text-[10px] font-medium text-slate-500'>
                支持按周期导出
              </span>
            </button>
          </EnterprisePanel>

          <div className='overflow-hidden rounded-md border border-slate-200 bg-white shadow-[0_1px_2px_rgb(15_23_42/0.025)]'>
            <div className='grid divide-y divide-slate-100 sm:grid-cols-5 sm:divide-x sm:divide-y-0'>
              <FeatureShortcut
                icon={Scale}
                title='成本中心分摊'
                description='按成本中心分摊费用，支持自定义分摊规则'
                action='去分摊'
                onClick={() => setSettlementSubjectFilter('supplier')}
              />
              <FeatureShortcut
                icon={UsersRound}
                title='客户账单'
                description='按客户生成账单，支持账期与对账'
                action='去查看'
                onClick={() => {
                  setSettlementSubjectFilter('user')
                  scrollToElement('billing-details')
                }}
              />
              <FeatureShortcut
                icon={WalletCards}
                title='内部充值'
                description='向企业预付余额充值，支持多种支付方式'
                action='去充值'
                onClick={() => void navigate({ to: '/wallet' })}
              />
              <FeatureShortcut
                icon={CreditCard}
                title='订阅计划'
                description='管理套餐与订阅，查看用量与权限'
                action='去管理'
                onClick={openClassicSubscriptions}
              />
              <FeatureShortcut
                icon={ReceiptText}
                title='发票状态'
                description='查看开票记录与状态，支持周期申请'
                action='去查看'
                onClick={() => setInvoiceDialogOpen(true)}
              />
            </div>
          </div>

          <EnterprisePanel
            id='billing-details'
            title='结算与账单明细'
            description='客户和供应商结算单统一视图'
            className='lg:col-span-2'
            bodyClassName='p-0'
            action={
              <div className='flex items-center gap-1.5'>
                <NativeSelect
                  value={settlementSubjectFilter}
                  className='h-7 w-32 rounded-md bg-white text-[11px]'
                  onChange={(event) =>
                    setSettlementSubjectFilter(event.target.value)
                  }
                >
                  <NativeSelectOption value='all'>
                    全部客户 / 成本中心
                  </NativeSelectOption>
                  <NativeSelectOption value='user'>客户账单</NativeSelectOption>
                  <NativeSelectOption value='supplier'>
                    成本中心
                  </NativeSelectOption>
                </NativeSelect>
                <NativeSelect
                  value={settlementStatusFilter}
                  className='h-7 w-24 rounded-md bg-white text-[11px]'
                  onChange={(event) =>
                    setSettlementStatusFilter(event.target.value)
                  }
                >
                  <NativeSelectOption value='all'>全部状态</NativeSelectOption>
                  <NativeSelectOption value='draft'>待确认</NativeSelectOption>
                  <NativeSelectOption value='finalized'>
                    已出账
                  </NativeSelectOption>
                  <NativeSelectOption value='paid'>已回款</NativeSelectOption>
                </NativeSelect>
                <Button
                  variant='outline'
                  className='h-7 rounded-md border-slate-200 bg-white px-2 text-[11px] text-slate-700 shadow-none'
                  onClick={() => void exportBillingCsv()}
                  disabled={exportingBilling}
                >
                  <Download className='size-3' />
                  导出
                </Button>
              </div>
            }
          >
            <SettlementTable
              items={filteredSettlements}
              loading={billingQuery.isLoading}
            />
            <div className='flex h-9 items-center justify-between border-t border-slate-100 px-3 text-[11px] text-slate-500'>
              <span>共 {formatNumber(filteredSettlements.length)} 条</span>
              <span>{rangeLabel}</span>
            </div>
          </EnterprisePanel>
        </div>

        <aside className='grid min-w-0 gap-2'>
          <EnterprisePanel
            title='预算预警'
            action={
              <Button
                variant='ghost'
                size='xs'
                className='h-6 px-1.5 text-[11px] text-blue-600'
                onClick={() => scrollToElement('budget-control')}
              >
                查看全部
                <ChevronRight className='size-3' />
              </Button>
            }
            bodyClassName='space-y-2 p-2.5'
          >
            {budgetAlerts.map((item) => (
              <div
                key={item.title}
                className='flex items-center gap-2 rounded-md border border-slate-100 bg-white px-2.5 py-2'
              >
                <span className='size-1.5 rounded-full bg-amber-400' />
                <div className='min-w-0 flex-1'>
                  <p className='truncate text-[11px] font-semibold text-slate-800'>
                    {item.title}
                  </p>
                  <p className='mt-0.5 truncate text-[10px] text-slate-500'>
                    {item.detail}
                  </p>
                </div>
                <Badge
                  variant='outline'
                  className={cn(
                    'h-5 shrink-0 rounded-md px-1.5 text-[10px]',
                    item.className
                  )}
                >
                  {item.badge}
                </Badge>
              </div>
            ))}
          </EnterprisePanel>

          <EnterprisePanel
            title='自动续费状态'
            action={
              <Button
                variant='ghost'
                size='xs'
                className='h-6 px-1.5 text-[11px] text-blue-600'
                onClick={openClassicSubscriptions}
              >
                管理订阅
                <ChevronRight className='size-3' />
              </Button>
            }
            bodyClassName='divide-y divide-slate-100 p-0'
          >
            <div className='flex items-center gap-2 px-3 py-2.5'>
              <span className='flex size-7 items-center justify-center rounded-md bg-blue-50 text-blue-600'>
                <CreditCard className='size-3.5' />
              </span>
              <div className='min-w-0 flex-1'>
                <p className='truncate text-[12px] font-semibold text-slate-800'>
                  Enterprise 年付套餐
                </p>
                <p className='mt-0.5 text-[10px] text-slate-500'>
                  {metrics.active_subscriptions > 0
                    ? `${metrics.active_subscriptions} 个订阅运行中`
                    : '暂无有效订阅'}
                </p>
              </div>
              <span
                className={cn(
                  'inline-flex items-center gap-1 text-[10px] font-semibold',
                  metrics.active_subscriptions > 0
                    ? 'text-emerald-600'
                    : 'text-slate-500'
                )}
              >
                <CheckCircle2 className='size-3' />
                {metrics.active_subscriptions > 0 ? '已启用' : '未启用'}
              </span>
            </div>
            <div className='flex items-center gap-2 px-3 py-2.5'>
              <span className='flex size-7 items-center justify-center rounded-md bg-blue-50 text-blue-600'>
                <WalletCards className='size-3.5' />
              </span>
              <div className='min-w-0 flex-1'>
                <p className='truncate text-[12px] font-semibold text-slate-800'>
                  预付额度自动补充
                </p>
                <p className='mt-0.5 text-[10px] text-slate-500'>
                  当前余额 {formatLogQuota(metrics.total_balance_quota)}
                </p>
              </div>
              <span className='inline-flex items-center gap-1 text-[10px] font-semibold text-emerald-600'>
                <CheckCircle2 className='size-3' />
                监控中
              </span>
            </div>
          </EnterprisePanel>

          <EnterprisePanel
            id='collection-card'
            title='收款进度（本月）'
            action={
              <Button
                variant='ghost'
                size='xs'
                className='h-6 px-1.5 text-[11px] text-blue-600'
                onClick={() => scrollToElement('billing-details')}
              >
                查看全部
                <ChevronRight className='size-3' />
              </Button>
            }
            bodyClassName='p-3'
          >
            <div className='flex items-center gap-4'>
              <div
                className='relative flex size-20 shrink-0 items-center justify-center rounded-full bg-[conic-gradient(#2563eb_var(--ring-value),#e2e8f0_0)]'
                style={
                  {
                    '--ring-value': `${Math.min(100, collectionProgress * 100)}%`,
                  } as CSSProperties
                }
              >
                <div className='flex size-14 flex-col items-center justify-center rounded-full bg-white'>
                  <span className='text-[14px] font-semibold text-slate-950 tabular-nums'>
                    {formatPercent(collectionProgress, 0)}
                  </span>
                  <span className='text-[9px] text-slate-500'>已回款</span>
                </div>
              </div>
              <dl className='grid flex-1 gap-1.5 text-[11px]'>
                <div className='flex items-center justify-between gap-2'>
                  <dt className='text-slate-500'>应收总额</dt>
                  <dd className='font-semibold text-slate-900'>
                    {formatLogQuota(metrics.period_sell_quota)}
                  </dd>
                </div>
                <div className='flex items-center justify-between gap-2'>
                  <dt className='text-slate-500'>已回款</dt>
                  <dd className='font-semibold text-slate-900'>
                    {formatCurrencyUSD(metrics.successful_top_up_amount)}
                  </dd>
                </div>
                <div className='flex items-center justify-between gap-2'>
                  <dt className='text-slate-500'>待回款</dt>
                  <dd className='font-semibold text-slate-900'>
                    {formatCurrencyUSD(metrics.pending_top_up_amount)}
                  </dd>
                </div>
              </dl>
            </div>
          </EnterprisePanel>

          <EnterprisePanel title='快速操作' bodyClassName='p-3'>
            <div className='grid grid-cols-4 gap-2'>
              <CompactAction icon={WalletCards} label='充值' to='/wallet' />
              <CompactAction
                icon={FileDown}
                label='导出账单'
                onClick={() => void exportBillingCsv()}
              />
              <CompactAction
                icon={ReceiptText}
                label='创建结算单'
                onClick={() => setSettlementDialogOpen(true)}
              />
              <CompactAction
                icon={FileText}
                label='申请发票'
                onClick={() => setInvoiceDialogOpen(true)}
              />
              <CompactAction icon={UsersRound} label='客户对账' to='/users' />
              <CompactAction
                icon={Download}
                label='下载结算单'
                onClick={downloadLatestSettlement}
              />
              <CompactAction
                icon={Landmark}
                label='预算管理'
                onClick={() => scrollToElement('budget-control')}
              />
              <CompactAction
                icon={BadgeDollarSign}
                label='费用分析'
                onClick={openFeeAnalysis}
              />
            </div>
          </EnterprisePanel>
        </aside>
      </div>

      {props.classicContent && (
        <Dialog open={classicDialogOpen} onOpenChange={setClassicDialogOpen}>
          <DialogContent className='max-h-[88vh] overflow-y-auto sm:max-w-6xl'>
            <DialogHeader>
              <DialogTitle>订阅计划管理</DialogTitle>
              <DialogDescription>
                管理套餐创建、支付平台关联、启停和编辑能力。
              </DialogDescription>
            </DialogHeader>
            {props.actions != null && (
              <div className='flex justify-end'>{props.actions}</div>
            )}
            <div className='rounded-md border border-slate-200 bg-slate-50/40 p-2'>
              {props.classicContent}
            </div>
          </DialogContent>
        </Dialog>
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
          <div className='grid gap-3'>
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

      <Dialog open={invoiceDialogOpen} onOpenChange={setInvoiceDialogOpen}>
        <DialogContent className='sm:max-w-3xl'>
          <DialogHeader>
            <DialogTitle>发票申请清单</DialogTitle>
            <DialogDescription>
              基于当前账期结算单生成开票申请，导出后可交由财务系统处理。
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-3'>
            <div className='grid gap-2 sm:grid-cols-3'>
              <div className='rounded-md border border-slate-200 bg-slate-50/70 px-3 py-2'>
                <p className='text-[11px] text-slate-500'>待开票金额</p>
                <p className='mt-1 text-base font-semibold text-slate-950 tabular-nums'>
                  {formatLogQuota(unsettledQuota)}
                </p>
              </div>
              <div className='rounded-md border border-slate-200 bg-slate-50/70 px-3 py-2'>
                <p className='text-[11px] text-slate-500'>待处理结算单</p>
                <p className='mt-1 text-base font-semibold text-slate-950 tabular-nums'>
                  {formatNumber(pendingInvoiceItems.length)}
                </p>
              </div>
              <div className='rounded-md border border-slate-200 bg-slate-50/70 px-3 py-2'>
                <p className='text-[11px] text-slate-500'>账期范围</p>
                <p className='mt-1 truncate text-sm font-semibold text-slate-950'>
                  {rangeLabel}
                </p>
              </div>
            </div>

            <div className='overflow-hidden rounded-md border border-slate-200'>
              <Table>
                <TableHeader>
                  <TableRow className='bg-slate-50'>
                    <TableHead className='h-8 text-[11px]'>结算单</TableHead>
                    <TableHead className='h-8 text-[11px]'>对象</TableHead>
                    <TableHead className='h-8 text-[11px]'>周期</TableHead>
                    <TableHead className='h-8 text-right text-[11px]'>
                      金额
                    </TableHead>
                    <TableHead className='h-8 text-right text-[11px]'>
                      状态
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {invoiceRequestItems.length === 0 ? (
                    <TableRow>
                      <TableCell
                        colSpan={5}
                        className='h-24 text-center text-[12px] text-slate-500'
                      >
                        当前账期暂无可开票结算单
                      </TableCell>
                    </TableRow>
                  ) : (
                    invoiceRequestItems.slice(0, 6).map((item) => (
                      <TableRow key={item.id} className='text-[12px]'>
                        <TableCell className='py-2 font-medium text-slate-900'>
                          #{item.id}
                        </TableCell>
                        <TableCell className='py-2'>
                          <div className='min-w-0'>
                            <p className='truncate font-medium text-slate-800'>
                              {item.subject_name || `ID ${item.subject_id}`}
                            </p>
                            <p className='text-[10px] text-slate-500'>
                              {item.subject_type === 'supplier'
                                ? '供应商 / 上游'
                                : '客户 / 下游'}
                            </p>
                          </div>
                        </TableCell>
                        <TableCell className='py-2 text-slate-600'>
                          {formatShortDate(item.period_start)} -{' '}
                          {formatShortDate(item.period_end)}
                        </TableCell>
                        <TableCell className='py-2 text-right font-semibold text-slate-900 tabular-nums'>
                          {formatLogQuota(item.total_sell_quota)}
                        </TableCell>
                        <TableCell className='py-2 text-right'>
                          <InvoiceStatus status={item.status} />
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
            {invoiceRequestItems.length > 6 && (
              <p className='text-[11px] text-slate-500'>
                仅预览前 6 条，导出会包含全部{' '}
                {formatNumber(invoiceRequestItems.length)} 条。
              </p>
            )}
          </div>
          <DialogFooter>
            <Button
              variant='outline'
              onClick={() => setInvoiceDialogOpen(false)}
            >
              关闭
            </Button>
            <Button
              onClick={exportInvoiceRequest}
              disabled={invoiceRequestItems.length === 0}
            >
              导出申请清单
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
