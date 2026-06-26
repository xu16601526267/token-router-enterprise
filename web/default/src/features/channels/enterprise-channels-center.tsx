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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Activity,
  AlertTriangle,
  ArrowUpRight,
  BarChart3,
  Boxes,
  CheckCircle2,
  CircleDollarSign,
  Download,
  Gauge,
  ListChecks,
  MoreHorizontal,
  Plus,
  Power,
  PowerOff,
  Radio,
  RefreshCw,
  Search,
  ServerCog,
  ShieldCheck,
  SlidersHorizontal,
  Tags,
  TestTube2,
  Waypoints,
  WalletCards,
} from 'lucide-react'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { EnterprisePageHeader, EnterprisePanel } from '@/components/enterprise'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
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
import { Tabs, TabsContent } from '@/components/ui/tabs'
import { useEnterpriseConsole } from '@/context/enterprise-console-context'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

import {
  testChannel,
  updateAllChannelsBalance,
  updateChannelBalance,
} from './api'
import { useChannels } from './components/channels-provider'
import { ChannelsTable } from './components/channels-table'
import { CHANNEL_STATUS } from './constants'
import {
  exportEnterpriseChannelCenter,
  getEnterpriseChannelCenter,
  getEnterpriseChannelDetail,
} from './enterprise-api'
import type { EnterpriseChannelItem } from './enterprise-types'
import {
  handleBatchDisable,
  handleBatchEnable,
  handleTestAllChannels,
} from './lib'
import { getChannelTypeLabel } from './lib/channel-utils'

type EnterpriseChannelsCenterProps = {
  actions?: ReactNode
  retryBadge?: ReactNode
}

type SummaryTone = 'blue' | 'emerald' | 'violet' | 'amber' | 'rose'

type DetailTab =
  | 'overview'
  | 'balance'
  | 'models'
  | 'routing'
  | 'events'
  | 'sla'

const EMPTY_CHANNEL_ITEMS: EnterpriseChannelItem[] = []

const toneStyles: Record<
  SummaryTone,
  { icon: string; value: string; trend: string }
> = {
  blue: {
    icon: 'bg-blue-50 text-blue-600 ring-blue-100',
    value: 'text-slate-950',
    trend: 'text-emerald-600',
  },
  emerald: {
    icon: 'bg-emerald-50 text-emerald-600 ring-emerald-100',
    value: 'text-slate-950',
    trend: 'text-emerald-600',
  },
  violet: {
    icon: 'bg-violet-50 text-violet-600 ring-violet-100',
    value: 'text-slate-950',
    trend: 'text-emerald-600',
  },
  amber: {
    icon: 'bg-amber-50 text-amber-600 ring-amber-100',
    value: 'text-slate-950',
    trend: 'text-amber-600',
  },
  rose: {
    icon: 'bg-rose-50 text-rose-600 ring-rose-100',
    value: 'text-slate-950',
    trend: 'text-rose-600',
  },
}

function formatPercent(value: number): string {
  return `${(value * 100).toFixed(2)}%`
}

function formatMoney(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 2,
  }).format(value)
}

function formatCompactNumber(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    maximumFractionDigits: 1,
    notation: Math.abs(value) >= 10000 ? 'compact' : 'standard',
  }).format(value)
}

function splitModels(models: string): string[] {
  return models
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function formatTimestamp(timestamp: number): string {
  if (timestamp <= 0) return '未检查'
  const now = dayjs()
  const target = dayjs.unix(timestamp)
  const minutes = now.diff(target, 'minute')
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  const hours = now.diff(target, 'hour')
  if (hours < 24) return `${hours} 小时前`
  const days = now.diff(target, 'day')
  if (days < 30) return `${days} 天前`
  return target.format('MM-DD HH:mm')
}

function statusConfig(item: EnterpriseChannelItem): {
  label: string
  className: string
  dot: string
  severity: 'healthy' | 'warning' | 'danger' | 'muted'
} {
  if (item.status !== CHANNEL_STATUS.ENABLED) {
    return {
      label:
        item.status === CHANNEL_STATUS.AUTO_DISABLED ? '自动停用' : '已停用',
      className: 'bg-slate-100 text-slate-600',
      dot: 'bg-slate-400',
      severity: 'muted',
    }
  }
  if (item.balance > 0 && item.balance < 10) {
    return {
      label: '低余额',
      className: 'bg-rose-50 text-rose-600',
      dot: 'bg-rose-500',
      severity: 'danger',
    }
  }
  if (item.success_rate > 0 && item.success_rate < 0.98) {
    return {
      label: '告警',
      className: 'bg-amber-50 text-amber-600',
      dot: 'bg-amber-500',
      severity: 'warning',
    }
  }
  return {
    label: '健康',
    className: 'bg-emerald-50 text-emerald-600',
    dot: 'bg-emerald-500',
    severity: 'healthy',
  }
}

function getSupplierInitial(item: EnterpriseChannelItem): string {
  const name = item.supplier_name || item.name || 'S'
  return name.trim().slice(0, 1).toUpperCase()
}

function getSelectedTargetIds(
  selectedIds: number[],
  selected: EnterpriseChannelItem | null
): number[] {
  if (selectedIds.length > 0) {
    return selectedIds
  }
  if (selected != null) {
    return [selected.id]
  }
  return []
}

function getAvailabilityTone(
  severity?: ReturnType<typeof statusConfig>['severity']
): 'emerald' | 'amber' | 'rose' {
  if (severity === 'danger') {
    return 'rose'
  }
  if (severity === 'warning') {
    return 'amber'
  }
  return 'emerald'
}

function modelBadges(models: string, compact = false) {
  const values = splitModels(models)
  const visibleCount = compact ? 2 : 3
  return (
    <div className='flex max-w-[240px] flex-wrap gap-1'>
      {values.slice(0, visibleCount).map((model) => (
        <Badge
          key={model}
          variant='outline'
          className='h-5 max-w-28 truncate rounded px-1.5 text-[10px] font-medium'
        >
          {model}
        </Badge>
      ))}
      {values.length > visibleCount && (
        <Badge variant='secondary' className='h-5 rounded px-1.5 text-[10px]'>
          +{values.length - visibleCount}
        </Badge>
      )}
      {values.length === 0 && (
        <span className='text-muted-foreground text-[11px]'>未配置</span>
      )}
    </div>
  )
}

function ChannelSummaryCard({
  title,
  value,
  helper,
  trend,
  tone,
  icon: Icon,
  loading,
}: {
  title: string
  value: string
  helper: string
  trend?: string
  tone: SummaryTone
  icon: typeof Radio
  loading?: boolean
}) {
  const styles = toneStyles[tone]
  return (
    <article className='min-h-[84px] rounded-md border border-slate-200/80 bg-white px-3 py-2.5 shadow-[0_1px_2px_rgb(15_23_42/0.035)]'>
      <div className='flex items-start justify-between gap-2'>
        <div className='flex min-w-0 items-start gap-2.5'>
          <span
            className={cn(
              'flex size-8 shrink-0 items-center justify-center rounded-md ring-1',
              styles.icon
            )}
          >
            <Icon className='size-4' strokeWidth={1.9} />
          </span>
          <div className='min-w-0'>
            <p className='truncate text-[11px] leading-4 font-medium text-slate-500'>
              {title}
            </p>
            {loading ? (
              <div className='mt-1.5 h-6 w-20 animate-pulse rounded bg-slate-100' />
            ) : (
              <p
                className={cn(
                  'mt-0.5 truncate text-[18px] leading-6 font-semibold tabular-nums',
                  styles.value
                )}
              >
                {value}
              </p>
            )}
          </div>
        </div>
        <ArrowUpRight className='mt-1 size-3.5 shrink-0 text-slate-400' />
      </div>
      <div className='mt-1.5 flex min-h-4 items-center gap-1.5 pl-10 text-[11px]'>
        <span className='text-slate-500'>{helper}</span>
        {trend != null && (
          <span className={cn('font-semibold', styles.trend)}>{trend}</span>
        )}
      </div>
    </article>
  )
}

function DetailMetric({
  label,
  value,
  helper,
  tone = 'slate',
}: {
  label: string
  value: string
  helper?: string
  tone?: 'slate' | 'emerald' | 'amber' | 'rose' | 'blue'
}) {
  return (
    <div
      className={cn(
        'rounded-md border px-2.5 py-1.5',
        tone === 'slate' && 'border-slate-200 bg-slate-50/60',
        tone === 'emerald' && 'border-emerald-100 bg-emerald-50/60',
        tone === 'amber' && 'border-amber-100 bg-amber-50/60',
        tone === 'rose' && 'border-rose-100 bg-rose-50/60',
        tone === 'blue' && 'border-blue-100 bg-blue-50/60'
      )}
    >
      <p className='text-[10px] leading-4 text-slate-500'>{label}</p>
      <p className='mt-0.5 truncate text-[14px] leading-5 font-semibold text-slate-950 tabular-nums'>
        {value}
      </p>
      {helper != null && (
        <p className='mt-0.5 truncate text-[10px] leading-4 text-slate-500'>
          {helper}
        </p>
      )}
    </div>
  )
}

function DetailSection({
  title,
  meta,
  icon: Icon,
  children,
}: {
  title: string
  meta?: string
  icon: typeof Activity
  children: ReactNode
}) {
  return (
    <section>
      <div className='mb-1 flex items-center justify-between gap-2'>
        <div className='flex min-w-0 items-center gap-1.5'>
          <Icon className='size-3.5 shrink-0 text-blue-600' />
          <h3 className='truncate text-[12px] leading-4 font-semibold text-slate-950'>
            {title}
          </h3>
        </div>
        {meta != null && (
          <span className='shrink-0 text-[10px] leading-4 text-slate-500'>
            {meta}
          </span>
        )}
      </div>
      {children}
    </section>
  )
}

function MiniProgressRow({
  label,
  value,
  helper,
  max,
  tone,
}: {
  label: string
  value: number
  helper: string
  max: number
  tone: 'blue' | 'emerald' | 'amber' | 'rose'
}) {
  const percent = max > 0 ? Math.min(100, Math.max(4, (value / max) * 100)) : 4
  return (
    <div className='rounded-md border border-slate-200 bg-white px-2.5 py-1.5'>
      <div className='flex items-center justify-between gap-2'>
        <span className='text-[10px] leading-4 text-slate-500'>{label}</span>
        <span className='text-[11px] leading-4 font-semibold text-slate-900 tabular-nums'>
          {helper}
        </span>
      </div>
      <div className='mt-1 h-1.5 overflow-hidden rounded-full bg-slate-100'>
        <div
          className={cn(
            'h-full rounded-full',
            tone === 'blue' && 'bg-blue-500',
            tone === 'emerald' && 'bg-emerald-500',
            tone === 'amber' && 'bg-amber-500',
            tone === 'rose' && 'bg-rose-500'
          )}
          style={{ width: `${percent}%` }}
        />
      </div>
    </div>
  )
}

export function EnterpriseChannelsCenter(props: EnterpriseChannelsCenterProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { setOpen, setCurrentRow } = useChannels()
  const { range, rangeLabel } = useEnterpriseConsole()
  const [activeTab, setActiveTab] = useState('enterprise')
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState('all')
  const [supplier, setSupplier] = useState('all')
  const [group, setGroup] = useState('all')
  const [type, setType] = useState('all')
  const [page, setPage] = useState(1)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [detailTab, setDetailTab] = useState<DetailTab>('overview')
  const [exportingChannels, setExportingChannels] = useState(false)
  const [busyAction, setBusyAction] = useState<string | null>(null)
  const rangeDays = useMemo(() => {
    const seconds = Math.max(0, range.end - range.start)
    return Math.max(1, Math.ceil(seconds / (24 * 60 * 60)))
  }, [range.end, range.start])

  const queryParams = useMemo(
    () => ({
      start_timestamp: range.start,
      end_timestamp: range.end,
      keyword: keyword.trim() || undefined,
      status: status !== 'all' ? Number(status) : undefined,
      supplier_id: supplier !== 'all' ? Number(supplier) : undefined,
      type: type !== 'all' ? Number(type) : undefined,
      group: group !== 'all' ? group : undefined,
      page,
      page_size: 50,
      sort_by: 'priority',
      sort_order: 'desc',
    }),
    [group, keyword, page, range.end, range.start, status, supplier, type]
  )

  const centerQuery = useQuery({
    queryKey: ['enterprise-channel-center', queryParams],
    queryFn: () => getEnterpriseChannelCenter(queryParams),
  })
  const detailQuery = useQuery({
    queryKey: ['enterprise-channel-detail', selectedId, range.start, range.end],
    queryFn: () =>
      getEnterpriseChannelDetail(selectedId ?? 0, {
        start_timestamp: range.start,
        end_timestamp: range.end,
      }),
    enabled: selectedId != null,
  })

  const data = centerQuery.data?.data
  const summary = data?.summary
  const rawItems = data?.items ?? EMPTY_CHANNEL_ITEMS
  const items = rawItems
  const supplierOptions = useMemo(
    () => [
      ...new Map(
        rawItems
          .filter((item) => item.supplier_id > 0)
          .map((item) => [
            item.supplier_id,
            item.supplier_name || `供应商 #${item.supplier_id}`,
          ])
      ).entries(),
    ],
    [rawItems]
  )
  const groupOptions = useMemo(
    () =>
      [...new Set(rawItems.map((item) => item.group).filter(Boolean))].sort(),
    [rawItems]
  )
  const typeOptions = useMemo(
    () =>
      [
        ...new Map(
          rawItems.map((item) => [item.type, getChannelTypeLabel(item.type)])
        ).entries(),
      ].sort((a, b) => a[1].localeCompare(b[1])),
    [rawItems]
  )
  const selected = items.find((item) => item.id === selectedId) ?? null
  const detail = detailQuery.data?.data
  const selectedModels = useMemo(
    () =>
      selected
        ? (detail?.supported_models ?? splitModels(selected.models))
        : [],
    [detail?.supported_models, selected]
  )
  const selectedIncidents = detail?.incidents ?? []
  const total = data?.total ?? rawItems.length
  const fallbackSupplierTotal = supplierOptions.length
  const totalSuppliers =
    summary?.total_suppliers != null && summary.total_suppliers > 0
      ? summary.total_suppliers
      : fallbackSupplierTotal
  const totalPages = Math.max(1, Math.ceil(total / (data?.page_size ?? 50)))
  const allPageSelected =
    items.length > 0 && items.every((item) => selectedIds.includes(item.id))
  const partialPageSelected =
    selectedIds.length > 0 &&
    items.some((item) => selectedIds.includes(item.id))

  useEffect(() => {
    if (items.length === 0) {
      setSelectedId(null)
      setSelectedIds([])
      return
    }
    if (selectedId == null || !items.some((item) => item.id === selectedId)) {
      setSelectedId(items[0].id)
    }
    setSelectedIds((ids) =>
      ids.filter((id) => items.some((item) => item.id === id))
    )
  }, [items, selectedId])

  const invalidateChannelViews = async () => {
    await queryClient.invalidateQueries({
      queryKey: ['enterprise-channel-center'],
    })
    await queryClient.invalidateQueries({
      queryKey: ['enterprise-channel-detail'],
    })
    await queryClient.invalidateQueries({ queryKey: ['channels'] })
    await centerQuery.refetch()
    if (selectedId != null) {
      await detailQuery.refetch()
    }
  }

  const exportChannelsCsv = async () => {
    setExportingChannels(true)
    try {
      await exportEnterpriseChannelCenter(queryParams)
      toast.success('渠道数据已导出')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '导出失败')
    } finally {
      setExportingChannels(false)
    }
  }

  const updateAllBalances = async () => {
    setBusyAction('update-all-balances')
    try {
      const response = await updateAllChannelsBalance()
      if (response.success) {
        toast.success('已开始批量更新渠道余额')
        await invalidateChannelViews()
      } else {
        toast.error(response.message || '批量更新余额失败')
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '批量更新余额失败')
    } finally {
      setBusyAction(null)
    }
  }

  const testAllEnabledChannels = async () => {
    setBusyAction('test-all')
    try {
      await handleTestAllChannels(
        queryClient,
        () => void invalidateChannelViews()
      )
    } finally {
      setBusyAction(null)
    }
  }

  const bulkEnable = async () => {
    setBusyAction('bulk-enable')
    try {
      await handleBatchEnable(selectedIds, queryClient, () => {
        setSelectedIds([])
        void invalidateChannelViews()
      })
    } finally {
      setBusyAction(null)
    }
  }

  const bulkDisable = async () => {
    setBusyAction('bulk-disable')
    try {
      await handleBatchDisable(selectedIds, queryClient, () => {
        setSelectedIds([])
        void invalidateChannelViews()
      })
    } finally {
      setBusyAction(null)
    }
  }

  const refreshSelectedBalances = async () => {
    const targets = getSelectedTargetIds(selectedIds, selected)
    if (targets.length === 0) {
      toast.error('请先选择渠道')
      return
    }
    setBusyAction('selected-balance')
    try {
      const results = await Promise.allSettled(
        targets.map((id) => updateChannelBalance(id))
      )
      const success = results.filter(
        (result) => result.status === 'fulfilled' && result.value.success
      ).length
      const failed = results.length - success
      if (success > 0) {
        toast.success(`已更新 ${success} 个渠道余额`)
      }
      if (failed > 0) {
        toast.error(`${failed} 个渠道余额更新失败`)
      }
      await invalidateChannelViews()
    } finally {
      setBusyAction(null)
    }
  }

  const testSelectedChannels = async () => {
    const targets = getSelectedTargetIds(selectedIds, selected)
    if (targets.length === 0) {
      toast.error('请先选择渠道')
      return
    }
    setBusyAction('selected-test')
    try {
      const results = await Promise.allSettled(
        targets.map((id) => testChannel(id))
      )
      const success = results.filter(
        (result) => result.status === 'fulfilled' && result.value.success
      ).length
      const failed = results.length - success
      if (success > 0) {
        toast.success(`测试通过 ${success} 个渠道`)
      }
      if (failed > 0) {
        toast.error(`${failed} 个渠道测试失败`)
      }
      await invalidateChannelViews()
    } finally {
      setBusyAction(null)
    }
  }

  const toggleSelectPage = (checked: boolean) => {
    if (checked) {
      setSelectedIds((ids) => [
        ...new Set([...ids, ...items.map((item) => item.id)]),
      ])
      return
    }
    setSelectedIds((ids) =>
      ids.filter((id) => !items.some((item) => item.id === id))
    )
  }

  const toggleSelected = (id: number, checked: boolean) => {
    setSelectedIds((ids) =>
      checked ? [...new Set([...ids, id])] : ids.filter((value) => value !== id)
    )
  }

  const selectedStatus = selected ? statusConfig(selected) : null
  const selectedLatency = selected
    ? selected.average_latency_ms || selected.response_time_ms
    : 0
  const availabilityTone = getAvailabilityTone(selectedStatus?.severity)
  const enabledItems = useMemo(
    () => items.filter((item) => item.status === CHANNEL_STATUS.ENABLED),
    [items]
  )
  const riskyItems = useMemo(
    () =>
      items.filter((item) => {
        const severity = statusConfig(item).severity
        return severity === 'warning' || severity === 'danger'
      }),
    [items]
  )
  const modelCoverage = useMemo(() => {
    const counts = new Map<string, number>()
    for (const item of items) {
      for (const model of splitModels(item.models)) {
        counts.set(model, (counts.get(model) ?? 0) + 1)
      }
    }
    return [...counts.entries()]
      .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
      .slice(0, 5)
  }, [items])
  const routeLeaders = useMemo(
    () =>
      [...items]
        .sort(
          (a, b) =>
            Number(b.priority) - Number(a.priority) ||
            Number(b.weight) - Number(a.weight) ||
            b.used_quota - a.used_quota
        )
        .slice(0, 3),
    [items]
  )
  const selectedPeerItems = useMemo(() => {
    if (!selected) return []
    return items.filter((item) =>
      selected.supplier_id > 0
        ? item.supplier_id === selected.supplier_id
        : item.id === selected.id
    )
  }, [items, selected])
  const supplierBalance = useMemo(() => {
    if (detail?.supplier?.total_balance != null) {
      return detail.supplier.total_balance
    }
    return selectedPeerItems.reduce((sum, item) => sum + item.balance, 0)
  }, [detail?.supplier?.total_balance, selectedPeerItems])
  const averageVisibleBalance = useMemo(
    () =>
      items.length > 0
        ? items.reduce((sum, item) => sum + item.balance, 0) / items.length
        : 0,
    [items]
  )
  const maxBalanceForBars = Math.max(
    selected?.balance ?? 0,
    supplierBalance,
    averageVisibleBalance,
    1
  )
  let supplierProfileContent: ReactNode = (
    <p className='rounded-md border border-dashed border-slate-200 p-3 text-xs leading-5 text-slate-500'>
      当前渠道尚未绑定供应商，可在经典配置中补充供应商信息。
    </p>
  )
  if (detailQuery.isLoading) {
    supplierProfileContent = (
      <div className='h-24 animate-pulse rounded-md bg-slate-100' />
    )
  } else if (detail?.supplier) {
    supplierProfileContent = (
      <div className='rounded-md border border-slate-200 bg-slate-50/40 p-2.5'>
        <div className='flex items-center justify-between gap-2'>
          <div className='min-w-0'>
            <p className='truncate text-[12px] font-semibold text-slate-950'>
              {detail.supplier.name}
            </p>
            <p className='mt-0.5 truncate text-[10px] text-slate-500'>
              {detail.supplier.type || '供应商'} ·{' '}
              {detail.supplier.notes || '暂无备注'}
            </p>
          </div>
          <Badge variant='outline' className='h-5 rounded px-1.5 text-[10px]'>
            等级 {detail.supplier.grade || '-'}
          </Badge>
        </div>
        <div className='mt-2 grid grid-cols-3 gap-2 text-center'>
          <DetailMetric label='评分' value={detail.supplier.score.toFixed(1)} />
          <DetailMetric
            label='渠道数'
            value={String(detail.supplier.channel_count)}
          />
          <DetailMetric
            label='权重'
            value={`${detail.supplier.route_weight || selected?.weight || 0}%`}
          />
        </div>
      </div>
    )
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Content>
        <Tabs
          value={activeTab}
          onValueChange={setActiveTab}
          className='flex h-full min-h-0 flex-col gap-2.5'
        >
          <div className='flex shrink-0 flex-col'>
            <EnterprisePageHeader
              eyebrow='资源与路由'
              title='渠道与供应商中心'
              description='统一管理上游供应商、渠道健康度、配额与模型覆盖。'
              actions={
                <>
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() => void updateAllBalances()}
                    disabled={busyAction === 'update-all-balances'}
                  >
                    <WalletCards className='size-3.5' />
                    批量更新余额
                  </Button>
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() => void exportChannelsCsv()}
                    disabled={exportingChannels}
                  >
                    <Download className='size-3.5' />
                    {exportingChannels ? '导出中' : '导出报表'}
                  </Button>
                  <Button
                    size='sm'
                    className='bg-blue-600 text-white hover:bg-blue-700'
                    onClick={() => {
                      setCurrentRow(null)
                      setOpen('create-channel')
                    }}
                  >
                    <Plus className='size-3.5' />
                    新增渠道
                  </Button>
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={
                        <Button
                          size='icon-sm'
                          variant='outline'
                          aria-label='更多渠道操作'
                        />
                      }
                    >
                      <MoreHorizontal className='size-3.5' />
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                      align='end'
                      className='w-44 rounded-md border-slate-200'
                    >
                      <DropdownMenuItem
                        className='text-xs'
                        onClick={() => void centerQuery.refetch()}
                      >
                        <RefreshCw className='size-3.5' />
                        刷新数据
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        className='text-xs'
                        onClick={() => void testAllEnabledChannels()}
                      >
                        <TestTube2 className='size-3.5' />
                        全量健康检查
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        className='text-xs'
                        onClick={() => setActiveTab('classic')}
                      >
                        <Gauge className='size-3.5' />
                        经典配置
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </>
              }
            />
          </div>

          <TabsContent
            value='enterprise'
            className='min-h-0 flex-1 overflow-auto pb-4'
          >
            <div className='grid min-h-full gap-2.5 xl:grid-cols-[minmax(0,1fr)_344px]'>
              <div className='flex min-w-0 flex-col gap-2.5'>
                <div className='grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-5'>
                  <ChannelSummaryCard
                    title='启用渠道数'
                    value={String(summary?.enabled_channels ?? 0)}
                    helper='较上月'
                    trend='+ 4'
                    icon={Radio}
                    tone='blue'
                    loading={centerQuery.isLoading}
                  />
                  <ChannelSummaryCard
                    title='健康供应商'
                    value={`${summary?.healthy_suppliers ?? 0} / ${totalSuppliers}`}
                    helper='可参与路由'
                    trend='+ 3'
                    icon={ShieldCheck}
                    tone='emerald'
                    loading={centerQuery.isLoading}
                  />
                  <ChannelSummaryCard
                    title='平均可用性'
                    value={formatPercent(summary?.average_success_rate ?? 0)}
                    helper={`${rangeDays} 天窗口`}
                    trend='实时'
                    icon={CheckCircle2}
                    tone='violet'
                    loading={centerQuery.isLoading}
                  />
                  <ChannelSummaryCard
                    title='本期供给成本'
                    value={formatMoney(summary?.total_balance ?? 0)}
                    helper='余额口径'
                    trend='+ 9.21%'
                    icon={CircleDollarSign}
                    tone='amber'
                    loading={centerQuery.isLoading}
                  />
                  <ChannelSummaryCard
                    title='低余额告警'
                    value={String(summary?.low_balance_alerts ?? 0)}
                    helper='需补充余额'
                    trend={summary?.low_balance_alerts ? '需处理' : '正常'}
                    icon={AlertTriangle}
                    tone='rose'
                    loading={centerQuery.isLoading}
                  />
                </div>

                <EnterprisePanel bodyClassName='p-2'>
                  <div className='flex flex-col gap-2 xl:flex-row xl:items-center xl:justify-between'>
                    <div className='grid flex-1 grid-cols-1 gap-2 md:grid-cols-2 xl:grid-cols-[minmax(220px,1fr)_128px_128px_140px_140px]'>
                      <div className='relative min-w-0'>
                        <Search className='pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-slate-400' />
                        <Input
                          className='h-8 rounded-md pl-8 text-xs'
                          value={keyword}
                          placeholder='搜索供应商、渠道、模型、标签'
                          onChange={(event) => {
                            setKeyword(event.target.value)
                            setPage(1)
                          }}
                        />
                      </div>
                      <NativeSelect
                        value={type}
                        className='h-8 rounded-md text-xs'
                        onChange={(event) => {
                          setType(event.target.value)
                          setPage(1)
                        }}
                      >
                        <NativeSelectOption value='all'>
                          上游类型
                        </NativeSelectOption>
                        {typeOptions.map(([id, label]) => (
                          <NativeSelectOption key={id} value={String(id)}>
                            {label}
                          </NativeSelectOption>
                        ))}
                      </NativeSelect>
                      <NativeSelect
                        value={group}
                        className='h-8 rounded-md text-xs'
                        onChange={(event) => {
                          setGroup(event.target.value)
                          setPage(1)
                        }}
                      >
                        <NativeSelectOption value='all'>
                          全部区域
                        </NativeSelectOption>
                        {groupOptions.map((value) => (
                          <NativeSelectOption key={value} value={value}>
                            {value}
                          </NativeSelectOption>
                        ))}
                      </NativeSelect>
                      <NativeSelect
                        value={status}
                        className='h-8 rounded-md text-xs'
                        onChange={(event) => {
                          setStatus(event.target.value)
                          setPage(1)
                        }}
                      >
                        <NativeSelectOption value='all'>
                          全部状态
                        </NativeSelectOption>
                        <NativeSelectOption
                          value={String(CHANNEL_STATUS.ENABLED)}
                        >
                          已启用
                        </NativeSelectOption>
                        <NativeSelectOption
                          value={String(CHANNEL_STATUS.MANUAL_DISABLED)}
                        >
                          已停用
                        </NativeSelectOption>
                      </NativeSelect>
                      <NativeSelect
                        value={supplier}
                        className='h-8 rounded-md text-xs'
                        onChange={(event) => {
                          setSupplier(event.target.value)
                          setPage(1)
                        }}
                      >
                        <NativeSelectOption value='all'>
                          全部供应商
                        </NativeSelectOption>
                        {supplierOptions.map(([id, name]) => (
                          <NativeSelectOption key={id} value={String(id)}>
                            {name}
                          </NativeSelectOption>
                        ))}
                      </NativeSelect>
                    </div>
                    <div className='flex shrink-0 items-center justify-between gap-2 text-xs text-slate-500 xl:justify-end'>
                      <Button
                        size='sm'
                        variant='outline'
                        className='h-8'
                        onClick={() => {
                          setKeyword('')
                          setStatus('all')
                          setSupplier('all')
                          setGroup('all')
                          setType('all')
                          setPage(1)
                        }}
                      >
                        <SlidersHorizontal className='size-3.5' />
                        重置筛选
                      </Button>
                      <span className='whitespace-nowrap'>
                        {rangeLabel} · 共 {total} 条 · 第 {data?.page ?? page}/
                        {totalPages} 页
                      </span>
                    </div>
                  </div>
                </EnterprisePanel>

                <EnterprisePanel className='min-w-0' bodyClassName='p-0'>
                  <div className='flex min-h-10 flex-wrap items-center justify-between gap-2 border-b border-slate-100 bg-white px-3 py-2'>
                    <div className='flex flex-wrap items-center gap-2 text-xs'>
                      <span className='font-medium text-slate-900'>
                        {selectedIds.length > 0
                          ? `已选择 ${selectedIds.length} 项`
                          : '渠道运行列表'}
                      </span>
                      <Button
                        size='xs'
                        variant='outline'
                        onClick={() => void bulkEnable()}
                        disabled={
                          selectedIds.length === 0 ||
                          busyAction === 'bulk-enable'
                        }
                      >
                        <Power className='size-3' />
                        启用
                      </Button>
                      <Button
                        size='xs'
                        variant='outline'
                        onClick={() => void bulkDisable()}
                        disabled={
                          selectedIds.length === 0 ||
                          busyAction === 'bulk-disable'
                        }
                      >
                        <PowerOff className='size-3' />
                        停用
                      </Button>
                      <Button
                        size='xs'
                        variant='outline'
                        onClick={() => void testSelectedChannels()}
                        disabled={
                          (selectedIds.length === 0 && selected == null) ||
                          busyAction === 'selected-test'
                        }
                      >
                        <TestTube2 className='size-3' />
                        测试
                      </Button>
                      <Button
                        size='xs'
                        variant='outline'
                        onClick={() => void refreshSelectedBalances()}
                        disabled={
                          (selectedIds.length === 0 && selected == null) ||
                          busyAction === 'selected-balance'
                        }
                      >
                        <WalletCards className='size-3' />
                        更新余额
                      </Button>
                      <Button
                        size='xs'
                        variant='ghost'
                        onClick={() => setActiveTab('classic')}
                      >
                        <MoreHorizontal className='size-3' />
                        更多操作
                      </Button>
                    </div>
                    <div className='flex items-center gap-2'>
                      <Button
                        size='xs'
                        variant='outline'
                        onClick={() => void testAllEnabledChannels()}
                        disabled={busyAction === 'test-all'}
                      >
                        全量健康检查
                      </Button>
                      <Button
                        size='xs'
                        variant='outline'
                        onClick={() =>
                          setPage((value) => Math.max(1, value - 1))
                        }
                        disabled={page <= 1 || centerQuery.isFetching}
                      >
                        上一页
                      </Button>
                      <Button
                        size='xs'
                        variant='outline'
                        onClick={() =>
                          setPage((value) => Math.min(totalPages, value + 1))
                        }
                        disabled={page >= totalPages || centerQuery.isFetching}
                      >
                        下一页
                      </Button>
                    </div>
                  </div>
                  <div
                    className={cn(
                      'overflow-auto',
                      items.length <= 4
                        ? 'max-h-[230px]'
                        : 'max-h-[calc(100vh-350px)] min-h-[430px]'
                    )}
                  >
                    <Table className='min-w-[920px]'>
                      <TableHeader className='sticky top-0 z-10 bg-slate-50/95 backdrop-blur'>
                        <TableRow className='h-9'>
                          <TableHead className='w-9 px-2'>
                            <Checkbox
                              checked={allPageSelected}
                              data-indeterminate={
                                !allPageSelected && partialPageSelected
                                  ? 'true'
                                  : undefined
                              }
                              onCheckedChange={(checked) =>
                                toggleSelectPage(checked === true)
                              }
                              aria-label='选择当前页渠道'
                            />
                          </TableHead>
                          <TableHead className='px-2 text-[11px] font-medium text-slate-500'>
                            供应商 / 渠道
                          </TableHead>
                          <TableHead className='px-2 text-[11px] font-medium text-slate-500'>
                            上游类型
                          </TableHead>
                          <TableHead className='px-2 text-[11px] font-medium text-slate-500'>
                            覆盖模型
                          </TableHead>
                          <TableHead className='px-2 text-[11px] font-medium text-slate-500'>
                            区域
                          </TableHead>
                          <TableHead className='px-2 text-[11px] font-medium text-slate-500'>
                            限额 / 配额
                          </TableHead>
                          <TableHead className='px-2 text-[11px] font-medium text-slate-500'>
                            余额
                          </TableHead>
                          <TableHead className='px-2 text-[11px] font-medium text-slate-500'>
                            成功率
                          </TableHead>
                          <TableHead className='px-2 text-[11px] font-medium text-slate-500'>
                            延迟 P95
                          </TableHead>
                          <TableHead className='px-2 text-[11px] font-medium text-slate-500'>
                            标签
                          </TableHead>
                          <TableHead className='px-2 text-[11px] font-medium text-slate-500'>
                            状态
                          </TableHead>
                          <TableHead className='px-2 text-[11px] font-medium text-slate-500'>
                            最后检查
                          </TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {items.length === 0 ? (
                          <TableRow>
                            <TableCell
                              colSpan={12}
                              className='h-52 text-center text-xs text-slate-500'
                            >
                              {centerQuery.isLoading
                                ? '正在加载渠道数据...'
                                : '没有符合筛选条件的渠道'}
                            </TableCell>
                          </TableRow>
                        ) : (
                          items.map((item) => {
                            const config = statusConfig(item)
                            const latency =
                              item.average_latency_ms || item.response_time_ms
                            const checked = selectedIds.includes(item.id)
                            return (
                              <TableRow
                                key={item.id}
                                className={cn(
                                  'h-[52px] cursor-pointer border-slate-100 text-xs hover:bg-blue-50/40',
                                  selectedId === item.id && 'bg-blue-50/70'
                                )}
                                onClick={() => {
                                  setSelectedId(item.id)
                                  setDetailTab('overview')
                                }}
                              >
                                <TableCell className='px-2 py-2'>
                                  <Checkbox
                                    checked={checked}
                                    onClick={(event) => event.stopPropagation()}
                                    onCheckedChange={(nextChecked) =>
                                      toggleSelected(
                                        item.id,
                                        nextChecked === true
                                      )
                                    }
                                    aria-label={`选择渠道 ${item.name}`}
                                  />
                                </TableCell>
                                <TableCell className='px-2 py-2'>
                                  <div className='flex min-w-48 items-center gap-2'>
                                    <span className='flex size-6 shrink-0 items-center justify-center rounded-full bg-slate-100 text-[10px] font-semibold text-slate-600'>
                                      {getSupplierInitial(item)}
                                    </span>
                                    <div className='min-w-0'>
                                      <p className='truncate text-[12px] font-semibold text-slate-950'>
                                        {item.supplier_name || item.name}
                                      </p>
                                      <p className='truncate text-[10px] text-slate-500'>
                                        {item.name}
                                      </p>
                                    </div>
                                  </div>
                                </TableCell>
                                <TableCell className='px-2 py-2 text-slate-700'>
                                  <div>
                                    <p>{getChannelTypeLabel(item.type)}</p>
                                    <p className='mt-0.5 text-[10px] text-slate-400'>
                                      {item.supplier_type || '上游渠道'}
                                    </p>
                                  </div>
                                </TableCell>
                                <TableCell className='px-2 py-2'>
                                  {modelBadges(item.models, true)}
                                </TableCell>
                                <TableCell className='px-2 py-2'>
                                  <span className='text-slate-700'>
                                    {item.group || 'default'}
                                  </span>
                                </TableCell>
                                <TableCell className='px-2 py-2'>
                                  <div className='text-slate-700'>
                                    <p>
                                      已用{' '}
                                      {formatCompactNumber(item.used_quota)}
                                    </p>
                                    <p className='text-[10px] text-slate-400'>
                                      权重 {item.weight || 0}%
                                    </p>
                                  </div>
                                </TableCell>
                                <TableCell className='px-2 py-2'>
                                  <p
                                    className={cn(
                                      'font-semibold tabular-nums',
                                      item.balance > 0 &&
                                        item.balance < 10 &&
                                        'text-rose-600'
                                    )}
                                  >
                                    {formatMoney(item.balance)}
                                  </p>
                                </TableCell>
                                <TableCell className='px-2 py-2'>
                                  <span
                                    className={cn(
                                      'font-semibold tabular-nums',
                                      item.success_rate >= 0.98
                                        ? 'text-emerald-600'
                                        : 'text-rose-600'
                                    )}
                                  >
                                    {formatPercent(item.success_rate)}
                                  </span>
                                </TableCell>
                                <TableCell className='px-2 py-2'>
                                  <span
                                    className={cn(
                                      'tabular-nums',
                                      latency > 600
                                        ? 'text-amber-600'
                                        : 'text-slate-700'
                                    )}
                                  >
                                    {latency.toFixed(0)}ms
                                  </span>
                                </TableCell>
                                <TableCell className='px-2 py-2'>
                                  {item.tag ? (
                                    <Badge
                                      variant='secondary'
                                      className='h-5 rounded px-1.5 text-[10px]'
                                    >
                                      {item.tag}
                                    </Badge>
                                  ) : (
                                    <span className='text-slate-400'>-</span>
                                  )}
                                </TableCell>
                                <TableCell className='px-2 py-2'>
                                  <Badge
                                    className={cn(
                                      'h-5 rounded border-0 px-1.5 text-[10px] font-medium',
                                      config.className
                                    )}
                                  >
                                    <span
                                      className={cn(
                                        'mr-1 size-1.5 rounded-full',
                                        config.dot
                                      )}
                                    />
                                    {config.label}
                                  </Badge>
                                </TableCell>
                                <TableCell className='px-2 py-2 text-[11px] text-slate-500'>
                                  {formatTimestamp(item.last_checked_at)}
                                </TableCell>
                              </TableRow>
                            )
                          })
                        )}
                      </TableBody>
                    </Table>
                  </div>
                  {items.length > 0 && (
                    <div className='grid border-t border-slate-100 bg-slate-50/50 sm:grid-cols-3'>
                      <div className='border-b border-slate-100 px-3 py-2 sm:border-r sm:border-b-0'>
                        <div className='flex items-center justify-between gap-2'>
                          <span className='text-[11px] font-medium text-slate-500'>
                            渠道可用状态
                          </span>
                          <span className='text-[12px] font-semibold text-slate-950 tabular-nums'>
                            {enabledItems.length}/{items.length}
                          </span>
                        </div>
                        <div className='mt-1.5 flex h-1.5 overflow-hidden rounded-full bg-slate-200'>
                          <span
                            className='bg-emerald-500'
                            style={{
                              width: `${items.length > 0 ? (enabledItems.length / items.length) * 100 : 0}%`,
                            }}
                          />
                          <span
                            className='bg-amber-400'
                            style={{
                              width: `${items.length > 0 ? (riskyItems.length / items.length) * 100 : 0}%`,
                            }}
                          />
                        </div>
                      </div>
                      <div className='border-b border-slate-100 px-3 py-2 sm:border-r sm:border-b-0'>
                        <div className='flex items-center justify-between gap-2'>
                          <span className='text-[11px] font-medium text-slate-500'>
                            模型覆盖 Top
                          </span>
                          <span className='text-[11px] text-slate-500'>
                            {modelCoverage.length} 类
                          </span>
                        </div>
                        <div className='mt-1.5 flex gap-1 overflow-hidden'>
                          {modelCoverage.slice(0, 3).map(([model, count]) => (
                            <Badge
                              key={model}
                              variant='secondary'
                              className='h-5 min-w-0 rounded px-1.5 text-[10px]'
                            >
                              <span className='truncate'>{model}</span>
                              <span className='ml-1 text-slate-500'>
                                {count}
                              </span>
                            </Badge>
                          ))}
                          {modelCoverage.length === 0 && (
                            <span className='text-[11px] text-slate-400'>
                              暂无模型配置
                            </span>
                          )}
                        </div>
                      </div>
                      <div className='px-3 py-2'>
                        <div className='flex items-center justify-between gap-2'>
                          <span className='text-[11px] font-medium text-slate-500'>
                            路由优先级
                          </span>
                          <span className='text-[11px] text-slate-500'>
                            按优先级/权重
                          </span>
                        </div>
                        <div className='mt-1.5 flex items-center gap-1.5 overflow-hidden'>
                          {routeLeaders.map((item, index) => (
                            <span
                              key={item.id}
                              className='inline-flex min-w-0 items-center gap-1 rounded bg-white px-1.5 py-0.5 text-[10px] font-medium text-slate-700 ring-1 ring-slate-200'
                            >
                              <span className='text-slate-400'>
                                {index + 1}
                              </span>
                              <span className='truncate'>
                                {item.supplier_name || item.name}
                              </span>
                            </span>
                          ))}
                        </div>
                      </div>
                    </div>
                  )}
                </EnterprisePanel>
              </div>

              <aside className='min-w-0'>
                <EnterprisePanel
                  className='xl:sticky xl:top-0'
                  bodyClassName='p-0'
                >
                  {!selected ? (
                    <div className='flex min-h-[520px] flex-col items-center justify-center px-6 text-center xl:min-h-[calc(100vh-172px)]'>
                      <span className='flex size-11 items-center justify-center rounded-md bg-blue-50 text-blue-600'>
                        <ServerCog className='size-5' />
                      </span>
                      <p className='mt-3 text-sm font-semibold text-slate-950'>
                        选择渠道查看详情
                      </p>
                      <p className='mt-1 text-xs leading-5 text-slate-500'>
                        右侧承载供应商健康、余额、模型覆盖、事件与快捷操作。
                      </p>
                    </div>
                  ) : (
                    <div className='min-h-[520px] xl:min-h-[calc(100vh-172px)]'>
                      <div className='border-b border-slate-100 px-3 py-3'>
                        <div className='flex items-start justify-between gap-3'>
                          <div className='min-w-0'>
                            <div className='flex items-center gap-2'>
                              <span className='flex size-7 shrink-0 items-center justify-center rounded-full bg-slate-100 text-[11px] font-semibold text-slate-600'>
                                {getSupplierInitial(selected)}
                              </span>
                              <div className='min-w-0'>
                                <p className='truncate text-[14px] font-semibold text-slate-950'>
                                  {selected.supplier_name || selected.name}
                                </p>
                                <p className='truncate text-[11px] text-slate-500'>
                                  {selected.name} · #{selected.id}
                                </p>
                              </div>
                            </div>
                          </div>
                          <Badge
                            className={cn(
                              'h-6 rounded border-0 px-2 text-[11px]',
                              selectedStatus?.className
                            )}
                          >
                            <span
                              className={cn(
                                'mr-1 size-1.5 rounded-full',
                                selectedStatus?.dot
                              )}
                            />
                            {selectedStatus?.label}
                          </Badge>
                        </div>
                        <div className='mt-3 grid grid-cols-3 gap-2'>
                          <DetailMetric
                            label='可用性'
                            value={formatPercent(selected.success_rate)}
                            helper={`${rangeDays} 天`}
                            tone={availabilityTone}
                          />
                          <DetailMetric
                            label='延迟 P95'
                            value={`${selectedLatency.toFixed(0)}ms`}
                            helper='成功请求'
                            tone='blue'
                          />
                          <DetailMetric
                            label='余额'
                            value={formatMoney(selected.balance)}
                            helper='USD'
                            tone={
                              selected.balance > 0 && selected.balance < 10
                                ? 'rose'
                                : 'slate'
                            }
                          />
                        </div>
                      </div>

                      <div className='flex gap-0.5 overflow-x-auto border-b border-slate-100 px-2.5 py-2'>
                        {[
                          ['overview', '概览'],
                          ['balance', '配额与余额'],
                          ['models', '支持模型'],
                          ['routing', '路由策略'],
                          ['events', '事件记录'],
                          ['sla', 'SLA'],
                        ].map(([value, label]) => (
                          <button
                            key={value}
                            type='button'
                            className={cn(
                              'h-7 shrink-0 rounded px-2 text-[11px] font-medium text-slate-500 transition-colors hover:bg-slate-100',
                              'px-1.5 text-[10px]',
                              detailTab === value && 'bg-blue-50 text-blue-600'
                            )}
                            onClick={() => setDetailTab(value as DetailTab)}
                          >
                            {label}
                          </button>
                        ))}
                      </div>

                      <div className='space-y-2.5 px-3 py-2.5'>
                        {detailTab === 'overview' && (
                          <div className='space-y-2.5'>
                            <DetailSection
                              title='账户健康'
                              meta={detailQuery.isFetching ? '更新中' : '实时'}
                              icon={Activity}
                            >
                              <div className='grid grid-cols-2 gap-2'>
                                <DetailMetric
                                  label='成功率'
                                  value={formatPercent(selected.success_rate)}
                                  helper={`${formatCompactNumber(selected.requests)} 请求`}
                                  tone='emerald'
                                />
                                <DetailMetric
                                  label='平均延迟'
                                  value={`${selectedLatency.toFixed(0)}ms`}
                                  helper='近窗口'
                                  tone='blue'
                                />
                                <DetailMetric
                                  label='已用额度'
                                  value={formatCompactNumber(
                                    selected.used_quota
                                  )}
                                  helper='累计口径'
                                />
                                <DetailMetric
                                  label='最后检查'
                                  value={formatTimestamp(
                                    selected.last_checked_at
                                  )}
                                  helper={
                                    selected.balance_updated_time > 0
                                      ? `余额 ${formatTimestamp(
                                          selected.balance_updated_time
                                        )}`
                                      : '余额未同步'
                                  }
                                />
                              </div>
                            </DetailSection>

                            <DetailSection
                              title='余额快照'
                              meta='USD'
                              icon={BarChart3}
                            >
                              <div className='space-y-1.5'>
                                <MiniProgressRow
                                  label='当前渠道余额'
                                  value={selected.balance}
                                  helper={formatMoney(selected.balance)}
                                  max={maxBalanceForBars}
                                  tone={
                                    selected.balance > 0 &&
                                    selected.balance < 10
                                      ? 'rose'
                                      : 'emerald'
                                  }
                                />
                                <MiniProgressRow
                                  label='供应商余额'
                                  value={supplierBalance}
                                  helper={formatMoney(supplierBalance)}
                                  max={maxBalanceForBars}
                                  tone='blue'
                                />
                              </div>
                            </DetailSection>

                            <DetailSection
                              title='近期事件'
                              meta={
                                selectedIncidents.length > 1
                                  ? `近 ${rangeDays} 天 · +${selectedIncidents.length - 1}`
                                  : `近 ${rangeDays} 天`
                              }
                              icon={ListChecks}
                            >
                              <div className='space-y-1.5'>
                                {selectedIncidents.length === 0 ? (
                                  <div className='rounded-md border border-emerald-100 bg-emerald-50/70 px-2.5 py-2 text-[11px] leading-4 text-emerald-700'>
                                    当前窗口未发现错误事件。
                                  </div>
                                ) : (
                                  selectedIncidents
                                    .slice(0, 1)
                                    .map((incident) => (
                                      <div
                                        key={incident.id}
                                        className='rounded-md border border-slate-200 bg-white px-2 py-1.5'
                                      >
                                        <div className='flex items-start justify-between gap-2'>
                                          <p className='line-clamp-1 text-[11px] leading-4 font-medium text-slate-900'>
                                            {incident.title}
                                          </p>
                                          <Badge
                                            variant='outline'
                                            className='h-5 rounded px-1.5 text-[10px]'
                                          >
                                            {incident.status}
                                          </Badge>
                                        </div>
                                        <p className='mt-0.5 text-[10px] text-slate-500'>
                                          {dayjs
                                            .unix(incident.created_at)
                                            .format('MM-DD HH:mm')}
                                        </p>
                                      </div>
                                    ))
                                )}
                              </div>
                            </DetailSection>

                            <DetailSection
                              title='支持模型'
                              meta={`${selectedModels.length} 个模型`}
                              icon={Tags}
                            >
                              <div className='flex max-h-[48px] flex-wrap gap-1 overflow-auto rounded-md border border-slate-200 bg-slate-50/50 p-1.5'>
                                {selectedModels.length === 0 ? (
                                  <span className='text-[11px] text-slate-400'>
                                    暂无模型配置
                                  </span>
                                ) : (
                                  selectedModels.slice(0, 10).map((model) => (
                                    <Badge
                                      key={model}
                                      variant='secondary'
                                      className='h-5 rounded px-1.5 text-[10px]'
                                    >
                                      {model}
                                    </Badge>
                                  ))
                                )}
                              </div>
                            </DetailSection>

                            <DetailSection
                              title='路由优先级'
                              meta='同组分流'
                              icon={Waypoints}
                            >
                              <div className='grid grid-cols-3 gap-2 rounded-md border border-blue-100 bg-blue-50/50 p-1.5'>
                                <div>
                                  <p className='text-[10px] text-blue-600/80'>
                                    优先级
                                  </p>
                                  <p className='mt-0.5 text-[13px] font-semibold text-slate-950 tabular-nums'>
                                    {selected.priority || 0}
                                  </p>
                                </div>
                                <div>
                                  <p className='text-[10px] text-blue-600/80'>
                                    权重
                                  </p>
                                  <p className='mt-0.5 text-[13px] font-semibold text-slate-950 tabular-nums'>
                                    {selected.weight || 0}%
                                  </p>
                                </div>
                                <div className='min-w-0'>
                                  <p className='text-[10px] text-blue-600/80'>
                                    分组
                                  </p>
                                  <p className='mt-0.5 truncate text-[13px] font-semibold text-slate-950'>
                                    {selected.group || 'default'}
                                  </p>
                                </div>
                              </div>
                            </DetailSection>

                          </div>
                        )}

                        {detailTab === 'balance' && (
                          <div className='space-y-3'>
                            <div>
                              <div className='mb-2 flex items-center justify-between'>
                                <h3 className='text-[13px] font-semibold text-slate-950'>
                                  配额与余额
                                </h3>
                                <span className='text-[10px] text-slate-500'>
                                  余额口径
                                </span>
                              </div>
                              <div className='grid grid-cols-2 gap-2'>
                                <DetailMetric
                                  label='当前余额'
                                  value={formatMoney(selected.balance)}
                                  helper='USD'
                                  tone={
                                    selected.balance > 0 &&
                                    selected.balance < 10
                                      ? 'rose'
                                      : 'emerald'
                                  }
                                />
                                <DetailMetric
                                  label='已用额度'
                                  value={formatCompactNumber(
                                    selected.used_quota
                                  )}
                                  helper='累计口径'
                                  tone='blue'
                                />
                                <DetailMetric
                                  label='路由权重'
                                  value={`${selected.weight || 0}%`}
                                  helper='同组分流'
                                />
                                <DetailMetric
                                  label='余额同步'
                                  value={
                                    selected.balance_updated_time > 0
                                      ? formatTimestamp(
                                          selected.balance_updated_time
                                        )
                                      : '未同步'
                                  }
                                  helper='最后同步时间'
                                />
                              </div>
                            </div>
                            <div className='rounded-md border border-slate-200 bg-slate-50/60 p-2.5'>
                              <div className='flex items-center justify-between text-[11px] text-slate-500'>
                                <span>低余额阈值</span>
                                <span
                                  className={cn(
                                    'font-semibold',
                                    selected.balance > 0 &&
                                      selected.balance < 10
                                      ? 'text-rose-600'
                                      : 'text-emerald-600'
                                  )}
                                >
                                  {selected.balance > 0 && selected.balance < 10
                                    ? '需补充余额'
                                    : '余额正常'}
                                </span>
                              </div>
                              <div className='mt-2 h-1.5 overflow-hidden rounded-full bg-slate-200'>
                                <div
                                  className={cn(
                                    'h-full rounded-full',
                                    selected.balance > 0 &&
                                      selected.balance < 10
                                      ? 'bg-rose-500'
                                      : 'bg-emerald-500'
                                  )}
                                  style={{
                                    width: `${Math.min(100, Math.max(4, selected.balance))}%`,
                                  }}
                                />
                              </div>
                            </div>
                          </div>
                        )}

                        {detailTab === 'models' && (
                          <div>
                            <div className='mb-2 flex items-center justify-between'>
                              <div className='flex items-center gap-2 text-[13px] font-semibold text-slate-950'>
                                <Tags className='size-3.5 text-blue-600' />
                                支持模型
                              </div>
                              <Badge
                                variant='outline'
                                className='h-5 rounded px-1.5 text-[10px]'
                              >
                                {
                                  (
                                    detail?.supported_models ??
                                    splitModels(selected.models)
                                  ).length
                                }
                                个模型
                              </Badge>
                            </div>
                            <div className='flex max-h-[340px] flex-wrap gap-1.5 overflow-auto rounded-md border border-slate-200 bg-slate-50/50 p-2'>
                              {(
                                detail?.supported_models ??
                                splitModels(selected.models)
                              ).map((model) => (
                                <Badge
                                  key={model}
                                  variant='secondary'
                                  className='h-6 rounded px-2 text-[11px]'
                                >
                                  {model}
                                </Badge>
                              ))}
                            </div>
                          </div>
                        )}

                        {detailTab === 'events' && (
                          <div>
                            <div className='mb-2 flex items-center justify-between'>
                              <div className='flex items-center gap-2 text-[13px] font-semibold text-slate-950'>
                                <AlertTriangle className='size-3.5 text-amber-600' />
                                近期事件
                              </div>
                              <span className='text-[10px] text-slate-500'>
                                近 {rangeDays} 天
                              </span>
                            </div>
                            <div className='space-y-1.5'>
                              {(detail?.incidents ?? []).length === 0 ? (
                                <p className='rounded-md bg-emerald-50 p-3 text-xs leading-5 text-emerald-700'>
                                  未发现近期错误事件。
                                </p>
                              ) : (
                                detail?.incidents
                                  .slice(0, 8)
                                  .map((incident) => (
                                    <div
                                      key={incident.id}
                                      className='rounded-md border border-slate-200 px-2.5 py-2'
                                    >
                                      <div className='flex items-start justify-between gap-2'>
                                        <p className='line-clamp-2 text-[11px] leading-4 font-medium text-slate-900'>
                                          {incident.title}
                                        </p>
                                        <Badge
                                          variant='outline'
                                          className='h-5 rounded px-1.5 text-[10px]'
                                        >
                                          {incident.status}
                                        </Badge>
                                      </div>
                                      <p className='mt-1 text-[10px] text-slate-500'>
                                        {dayjs
                                          .unix(incident.created_at)
                                          .format('MM-DD HH:mm')}
                                      </p>
                                    </div>
                                  ))
                              )}
                            </div>
                          </div>
                        )}

                        {detailTab === 'routing' && (
                          <div className='space-y-3'>
                            <div className='grid grid-cols-2 gap-2'>
                              <DetailMetric
                                label='路由优先级'
                                value={String(selected.priority || 0)}
                                helper='数值越大越优先'
                                tone='blue'
                              />
                              <DetailMetric
                                label='路由权重'
                                value={`${selected.weight || 0}%`}
                                helper='同组分流'
                                tone='emerald'
                              />
                            </div>
                            <div className='rounded-md border border-blue-100 bg-blue-50/60 p-2.5'>
                              <div className='flex items-center gap-2 text-[12px] font-semibold text-blue-700'>
                                <Boxes className='size-3.5' />
                                分组与标签
                              </div>
                              <div className='mt-2 flex flex-wrap gap-1.5'>
                                <Badge
                                  variant='outline'
                                  className='h-5 rounded bg-white px-1.5 text-[10px]'
                                >
                                  {selected.group || 'default'}
                                </Badge>
                                {selected.tag && (
                                  <Badge
                                    variant='secondary'
                                    className='h-5 rounded px-1.5 text-[10px]'
                                  >
                                    {selected.tag}
                                  </Badge>
                                )}
                              </div>
                            </div>
                            <DetailSection title='供应商画像' icon={ShieldCheck}>
                              {supplierProfileContent}
                            </DetailSection>
                            <p className='rounded-md border border-slate-200 bg-slate-50/60 p-2.5 text-[11px] leading-5 text-slate-500'>
                              策略编辑、模型映射、多
                              Key、状态码风控等高级操作保留在经典配置视图中，避免企业视图重复维护复杂表单。
                            </p>
                          </div>
                        )}

                        {detailTab === 'sla' && (
                          <div className='space-y-3'>
                            <div className='grid grid-cols-2 gap-2'>
                              <DetailMetric
                                label='成功率目标'
                                value='98.00%'
                                helper={`当前 ${formatPercent(selected.success_rate)}`}
                                tone={
                                  selected.success_rate >= 0.98
                                    ? 'emerald'
                                    : 'rose'
                                }
                              />
                              <DetailMetric
                                label='延迟 P95 目标'
                                value='800ms'
                                helper={`当前 ${selectedLatency.toFixed(0)}ms`}
                                tone={
                                  selectedLatency > 800 ? 'rose' : 'emerald'
                                }
                              />
                              <DetailMetric
                                label='错误事件'
                                value={String(detail?.incidents.length ?? 0)}
                                helper={`${rangeDays} 天窗口`}
                                tone={
                                  (detail?.incidents.length ?? 0) > 0
                                    ? 'amber'
                                    : 'emerald'
                                }
                              />
                              <DetailMetric
                                label='请求样本'
                                value={formatCompactNumber(selected.requests)}
                                helper='用于计算 SLA'
                                tone='blue'
                              />
                            </div>
                            <div className='rounded-md border border-slate-200 bg-white p-2.5'>
                              <div className='flex items-center justify-between'>
                                <span className='text-[12px] font-semibold text-slate-900'>
                                  执行状态
                                </span>
                                <Badge
                                  className={cn(
                                    'h-5 rounded border-0 px-1.5 text-[10px]',
                                    selectedStatus?.className
                                  )}
                                >
                                  {selectedStatus?.label}
                                </Badge>
                              </div>
                              <p className='mt-1.5 text-[11px] leading-5 text-slate-500'>
                                成功率、延迟和错误事件来自当前全局时间窗口，
                                与渠道列表和导出报表保持同一口径。
                              </p>
                            </div>
                          </div>
                        )}
                      </div>

                      <div className='mt-1 border-t border-slate-100 px-3 py-3'>
                        <div className='grid grid-cols-2 gap-2'>
                          <Button
                            size='sm'
                            variant='outline'
                            onClick={() => void testSelectedChannels()}
                            disabled={busyAction === 'selected-test'}
                          >
                            <TestTube2 className='size-3.5' />
                            测试
                          </Button>
                          <Button
                            size='sm'
                            variant='outline'
                            onClick={() => void refreshSelectedBalances()}
                            disabled={busyAction === 'selected-balance'}
                          >
                            <WalletCards className='size-3.5' />
                            更新余额
                          </Button>
                          <Button
                            size='sm'
                            variant='outline'
                            onClick={() => setActiveTab('classic')}
                          >
                            <Gauge className='size-3.5' />
                            编辑
                          </Button>
                          <Button
                            size='sm'
                            variant='outline'
                            onClick={() => void exportChannelsCsv()}
                          >
                            <Download className='size-3.5' />
                            导出
                          </Button>
                        </div>
                      </div>
                    </div>
                  )}
                </EnterprisePanel>
              </aside>
            </div>
          </TabsContent>

          <TabsContent
            value='classic'
            className='min-h-0 flex-1 overflow-auto pb-5'
          >
            <EnterprisePanel
              title='经典渠道配置'
              description='保留原系统全部渠道增删改查、测试、余额更新、多 Key 管理和上游模型操作。'
              action={
                <div className='flex items-center gap-2'>
                  {props.actions}
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() => setActiveTab('enterprise')}
                  >
                    返回运营视图
                  </Button>
                </div>
              }
              bodyClassName='p-0'
            >
              <ChannelsTable />
            </EnterprisePanel>
          </TabsContent>
          <span className='sr-only'>{t('Channels')}</span>
        </Tabs>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
