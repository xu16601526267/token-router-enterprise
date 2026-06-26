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
import {
  Activity,
  AlertTriangle,
  Boxes,
  CheckCircle2,
  CircleDollarSign,
  Clock3,
  Download,
  Gauge,
  Radio,
  RefreshCw,
  Search,
  ServerCog,
  ShieldCheck,
  Tags,
} from 'lucide-react'
import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  EnterprisePageHeader,
  EnterprisePanel,
  EnterpriseStatCard,
} from '@/components/enterprise'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

import { ChannelsTable } from './components/channels-table'
import {
  exportEnterpriseChannelCenter,
  getEnterpriseChannelCenter,
  getEnterpriseChannelDetail,
} from './enterprise-api'
import type { EnterpriseChannelItem } from './enterprise-types'
import { getChannelTypeLabel } from './lib/channel-utils'

type EnterpriseChannelsCenterProps = {
  actions?: ReactNode
  retryBadge?: ReactNode
}

const EMPTY_CHANNEL_ITEMS: EnterpriseChannelItem[] = []

function formatPercent(value: number): string {
  return `${(value * 100).toFixed(2)}%`
}

function formatMoney(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 2,
  }).format(value)
}

function statusConfig(item: EnterpriseChannelItem): {
  label: string
  className: string
  dot: string
} {
  if (item.status !== 1) {
    return {
      label: '已停用',
      className: 'bg-muted text-muted-foreground',
      dot: 'bg-muted-foreground',
    }
  }
  if (item.success_rate > 0 && item.success_rate < 0.98) {
    return {
      label: '告警',
      className: 'bg-amber-500/10 text-amber-600 dark:text-amber-300',
      dot: 'bg-amber-500',
    }
  }
  if (item.balance > 0 && item.balance < 10) {
    return {
      label: '低余额',
      className: 'bg-rose-500/10 text-rose-600 dark:text-rose-300',
      dot: 'bg-rose-500',
    }
  }
  return {
    label: '健康',
    className: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-300',
    dot: 'bg-emerald-500',
  }
}

function modelBadges(models: string) {
  const values = models
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
  return (
    <div className='flex max-w-60 flex-wrap gap-1'>
      {values.slice(0, 2).map((model) => (
        <Badge key={model} variant='secondary' className='max-w-32 truncate'>
          {model}
        </Badge>
      ))}
      {values.length > 2 && (
        <Badge variant='outline'>+{values.length - 2}</Badge>
      )}
      {values.length === 0 && (
        <span className='text-muted-foreground text-xs'>未配置</span>
      )}
    </div>
  )
}

export function EnterpriseChannelsCenter(props: EnterpriseChannelsCenterProps) {
  const { t } = useTranslation()
  const range = useMemo(() => {
    const end = Math.floor(Date.now() / 1000)
    return { start: end - 7 * 24 * 60 * 60, end }
  }, [])
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState('all')
  const [supplier, setSupplier] = useState('all')
  const [group, setGroup] = useState('all')
  const [page, setPage] = useState(1)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [exportingChannels, setExportingChannels] = useState(false)

  const queryParams = useMemo(
    () => ({
      start_timestamp: range.start,
      end_timestamp: range.end,
      keyword: keyword.trim() || undefined,
      status: status !== 'all' ? Number(status) : undefined,
      supplier_id: supplier !== 'all' ? Number(supplier) : undefined,
      group: group !== 'all' ? group : undefined,
      page,
      page_size: 50,
    }),
    [group, keyword, page, range.end, range.start, status, supplier]
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
  const items = data?.items ?? EMPTY_CHANNEL_ITEMS
  const supplierOptions = useMemo(
    () => [
      ...new Map(
        items
          .filter((item) => item.supplier_id > 0)
          .map((item) => [
            item.supplier_id,
            item.supplier_name || `供应商 #${item.supplier_id}`,
          ])
      ).entries(),
    ],
    [items]
  )
  const groupOptions = useMemo(
    () => [...new Set(items.map((item) => item.group).filter(Boolean))].sort(),
    [items]
  )
  const selected = items.find((item) => item.id === selectedId) ?? null
  const detail = detailQuery.data?.data
  const total = data?.total ?? items.length
  const totalPages = Math.max(1, Math.ceil(total / (data?.page_size ?? 50)))

  let supplierProfileContent = (
    <p className='text-muted-foreground mt-3 rounded-md border border-dashed p-4 text-xs'>
      当前渠道尚未绑定供应商，可在经典配置视图中补充。
    </p>
  )
  if (detailQuery.isLoading) {
    supplierProfileContent = (
      <div className='bg-muted mt-3 h-24 animate-pulse rounded-md' />
    )
  } else if (detail?.supplier) {
    supplierProfileContent = (
      <div className='bg-muted/20 mt-3 rounded-md border p-3 text-xs'>
        <div className='flex items-center justify-between gap-3'>
          <div>
            <p className='font-semibold'>{detail.supplier.name}</p>
            <p className='text-muted-foreground mt-0.5'>
              {detail.supplier.type}
            </p>
          </div>
          <Badge variant='outline'>等级 {detail.supplier.grade || '—'}</Badge>
        </div>
        <div className='mt-3 grid grid-cols-3 gap-2 text-center'>
          <div className='bg-background rounded-md p-2'>
            <p className='text-muted-foreground'>评分</p>
            <p className='mt-1 font-semibold'>
              {detail.supplier.score.toFixed(1)}
            </p>
          </div>
          <div className='bg-background rounded-md p-2'>
            <p className='text-muted-foreground'>渠道数</p>
            <p className='mt-1 font-semibold'>
              {detail.supplier.channel_count}
            </p>
          </div>
          <div className='bg-background rounded-md p-2'>
            <p className='text-muted-foreground'>路由权重</p>
            <p className='mt-1 font-semibold'>
              {detail.supplier.route_weight || 100}%
            </p>
          </div>
        </div>
      </div>
    )
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

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Content>
        <Tabs
          defaultValue='enterprise'
          className='flex h-full min-h-0 flex-col gap-3'
        >
          <div className='flex shrink-0 flex-col gap-3'>
            <EnterprisePageHeader
              eyebrow='上游资源治理'
              title='渠道与供应商中心'
              description='统一管理上游供应商、渠道健康度、余额、模型覆盖和路由优先级；原有渠道编辑与批量操作完整保留。'
              actions={
                <>
                  {props.retryBadge}
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() => centerQuery.refetch()}
                    disabled={centerQuery.isFetching}
                  >
                    <RefreshCw
                      className={cn(
                        'size-4',
                        centerQuery.isFetching && 'animate-spin'
                      )}
                    />
                    刷新运营数据
                  </Button>
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() => void exportChannelsCsv()}
                    disabled={exportingChannels}
                  >
                    <Download className='size-4' />
                    {exportingChannels ? '导出中' : '导出渠道'}
                  </Button>
                  {props.actions}
                </>
              }
            />
            <TabsList className='w-fit'>
              <TabsTrigger value='enterprise'>企业运营视图</TabsTrigger>
              <TabsTrigger value='classic'>经典配置视图</TabsTrigger>
            </TabsList>
          </div>

          <TabsContent
            value='enterprise'
            className='min-h-0 flex-1 overflow-auto pb-5'
          >
            <div className='flex flex-col gap-3'>
              <div className='grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-5'>
                <EnterpriseStatCard
                  title='启用渠道数'
                  value={String(summary?.enabled_channels ?? 0)}
                  helper='可参与路由'
                  icon={Radio}
                  tone='blue'
                  loading={centerQuery.isLoading}
                />
                <EnterpriseStatCard
                  title='健康供应商'
                  value={String(summary?.healthy_suppliers ?? 0)}
                  helper='状态正常'
                  icon={ShieldCheck}
                  tone='emerald'
                  loading={centerQuery.isLoading}
                />
                <EnterpriseStatCard
                  title='平均成功率'
                  value={formatPercent(summary?.average_success_rate ?? 0)}
                  helper='近 7 天请求'
                  icon={CheckCircle2}
                  tone='violet'
                  loading={centerQuery.isLoading}
                />
                <EnterpriseStatCard
                  title='平均延迟'
                  value={`${(summary?.average_latency_ms ?? 0).toFixed(0)} ms`}
                  helper='近 7 天成功请求'
                  icon={Clock3}
                  tone='amber'
                  loading={centerQuery.isLoading}
                />
                <EnterpriseStatCard
                  title='低余额告警'
                  value={String(summary?.low_balance_alerts ?? 0)}
                  helper={`${formatMoney(summary?.total_balance ?? 0)} 总余额`}
                  icon={AlertTriangle}
                  tone='rose'
                  loading={centerQuery.isLoading}
                />
              </div>

              <EnterprisePanel bodyClassName='p-3 sm:p-4'>
                <div className='flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between'>
                  <div className='flex flex-1 flex-col gap-2 sm:flex-row sm:flex-wrap'>
                    <div className='relative min-w-60 flex-1 xl:max-w-md'>
                      <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2' />
                      <Input
                        className='pl-9'
                        value={keyword}
                        placeholder='搜索渠道、供应商、模型或标签'
                        onChange={(event) => {
                          setKeyword(event.target.value)
                          setPage(1)
                        }}
                      />
                    </div>
                    <NativeSelect
                      value={status}
                      className='w-full sm:w-36'
                      onChange={(event) => {
                        setStatus(event.target.value)
                        setPage(1)
                      }}
                    >
                      <NativeSelectOption value='all'>
                        全部状态
                      </NativeSelectOption>
                      <NativeSelectOption value='1'>已启用</NativeSelectOption>
                      <NativeSelectOption value='2'>已停用</NativeSelectOption>
                    </NativeSelect>
                    <NativeSelect
                      value={supplier}
                      className='w-full sm:w-44'
                      onChange={(event) => {
                        setSupplier(event.target.value)
                        setPage(1)
                      }}
                    >
                      <NativeSelectOption value='all'>
                        全部供应商
                      </NativeSelectOption>
                      {supplierOptions.map(([id, name]) => (
                        <NativeSelectOption key={id} value={id}>
                          {name}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                    <NativeSelect
                      value={group}
                      className='w-full sm:w-40'
                      onChange={(event) => {
                        setGroup(event.target.value)
                        setPage(1)
                      }}
                    >
                      <NativeSelectOption value='all'>
                        全部路由分组
                      </NativeSelectOption>
                      {groupOptions.map((value) => (
                        <NativeSelectOption key={value} value={value}>
                          {value}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </div>
                  <div className='text-muted-foreground flex flex-wrap items-center gap-2 text-xs'>
                    <span>
                      共 {total} 个渠道，第 {data?.page ?? page} / {totalPages}{' '}
                      页
                    </span>
                    <Button
                      size='sm'
                      variant='outline'
                      onClick={() => setPage((value) => Math.max(1, value - 1))}
                      disabled={page <= 1 || centerQuery.isFetching}
                    >
                      上一页
                    </Button>
                    <Button
                      size='sm'
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
              </EnterprisePanel>

              <div className='grid min-h-0 gap-3 2xl:grid-cols-[minmax(0,1fr)_380px]'>
                <EnterprisePanel className='min-w-0' bodyClassName='p-0'>
                  <div className='overflow-x-auto'>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>渠道 / 供应商</TableHead>
                          <TableHead>上游类型</TableHead>
                          <TableHead>覆盖模型</TableHead>
                          <TableHead>路由分组</TableHead>
                          <TableHead>余额</TableHead>
                          <TableHead>成功率</TableHead>
                          <TableHead>延迟</TableHead>
                          <TableHead>标签</TableHead>
                          <TableHead>状态</TableHead>
                          <TableHead>最后检查</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {items.length === 0 ? (
                          <TableRow>
                            <TableCell
                              colSpan={10}
                              className='text-muted-foreground h-40 text-center'
                            >
                              {centerQuery.isLoading
                                ? '正在加载渠道数据…'
                                : '没有符合筛选条件的渠道'}
                            </TableCell>
                          </TableRow>
                        ) : (
                          items.map((item) => {
                            const config = statusConfig(item)
                            const latency =
                              item.average_latency_ms || item.response_time_ms
                            return (
                              <TableRow
                                key={item.id}
                                className={cn(
                                  'cursor-pointer',
                                  selectedId === item.id && 'bg-primary/[0.045]'
                                )}
                                onClick={() => setSelectedId(item.id)}
                              >
                                <TableCell>
                                  <div className='max-w-52'>
                                    <div className='flex items-center gap-2'>
                                      <span
                                        className={cn(
                                          'size-2 rounded-full',
                                          config.dot
                                        )}
                                      />
                                      <p className='truncate font-medium'>
                                        {item.name}
                                      </p>
                                    </div>
                                    <p className='text-muted-foreground mt-0.5 truncate pl-4 text-xs'>
                                      {item.supplier_name || '未绑定供应商'}
                                    </p>
                                  </div>
                                </TableCell>
                                <TableCell>
                                  {getChannelTypeLabel(item.type)}
                                </TableCell>
                                <TableCell>
                                  {modelBadges(item.models)}
                                </TableCell>
                                <TableCell>
                                  <Badge variant='outline'>
                                    {item.group || 'default'}
                                  </Badge>
                                </TableCell>
                                <TableCell>
                                  <p
                                    className={cn(
                                      'font-medium',
                                      item.balance > 0 &&
                                        item.balance < 10 &&
                                        'text-rose-600'
                                    )}
                                  >
                                    {formatMoney(item.balance)}
                                  </p>
                                </TableCell>
                                <TableCell>
                                  {formatPercent(item.success_rate)}
                                </TableCell>
                                <TableCell>{latency.toFixed(0)} ms</TableCell>
                                <TableCell>
                                  {item.tag ? (
                                    <Badge variant='secondary'>
                                      {item.tag}
                                    </Badge>
                                  ) : (
                                    '—'
                                  )}
                                </TableCell>
                                <TableCell>
                                  <Badge
                                    className={cn('border-0', config.className)}
                                  >
                                    {config.label}
                                  </Badge>
                                </TableCell>
                                <TableCell>
                                  {item.last_checked_at > 0
                                    ? dayjs.unix(item.last_checked_at).fromNow()
                                    : '未检查'}
                                </TableCell>
                              </TableRow>
                            )
                          })
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </EnterprisePanel>

                <EnterprisePanel
                  title={selected ? selected.name : '渠道运行详情'}
                  description={
                    selected
                      ? `${selected.supplier_name || '未绑定供应商'} · ${selected.group || 'default'}`
                      : '选择左侧渠道查看供应商、健康与事件信息。'
                  }
                  action={
                    selected ? (
                      <Badge
                        className={cn(
                          'border-0',
                          statusConfig(selected).className
                        )}
                      >
                        {statusConfig(selected).label}
                      </Badge>
                    ) : null
                  }
                >
                  {!selected ? (
                    <div className='flex min-h-80 flex-col items-center justify-center text-center'>
                      <span className='bg-primary/10 text-primary flex size-12 items-center justify-center rounded-md'>
                        <ServerCog className='size-5' />
                      </span>
                      <p className='mt-3 text-sm font-medium'>
                        供应商与渠道治理
                      </p>
                      <p className='text-muted-foreground mt-1 max-w-64 text-xs leading-5'>
                        查看账户健康、供应商评分、路由权重、模型覆盖和近期异常。
                      </p>
                    </div>
                  ) : (
                    <div className='space-y-3'>
                      <div className='grid grid-cols-2 gap-3 text-xs'>
                        <div className='rounded-md border p-3'>
                          <Activity className='size-4 text-emerald-600' />
                          <p className='text-muted-foreground mt-2'>
                            近 7 天成功率
                          </p>
                          <p className='mt-1 text-base font-semibold'>
                            {formatPercent(selected.success_rate)}
                          </p>
                        </div>
                        <div className='rounded-md border p-3'>
                          <Gauge className='size-4 text-violet-600' />
                          <p className='text-muted-foreground mt-2'>平均延迟</p>
                          <p className='mt-1 text-base font-semibold'>
                            {(
                              selected.average_latency_ms ||
                              selected.response_time_ms
                            ).toFixed(0)}{' '}
                            ms
                          </p>
                        </div>
                        <div className='rounded-md border p-3'>
                          <CircleDollarSign className='size-4 text-amber-600' />
                          <p className='text-muted-foreground mt-2'>账户余额</p>
                          <p className='mt-1 text-base font-semibold'>
                            {formatMoney(selected.balance)}
                          </p>
                        </div>
                        <div className='rounded-md border p-3'>
                          <Boxes className='size-4 text-blue-600' />
                          <p className='text-muted-foreground mt-2'>路由权重</p>
                          <p className='mt-1 text-base font-semibold'>
                            {selected.weight || 0}%
                          </p>
                        </div>
                      </div>

                      <div>
                        <div className='flex items-center gap-2 text-sm font-medium'>
                          <ShieldCheck className='text-primary size-4' />
                          供应商画像
                        </div>
                        {supplierProfileContent}
                      </div>

                      <div>
                        <div className='flex items-center gap-2 text-sm font-medium'>
                          <Tags className='text-primary size-4' />
                          支持模型
                        </div>
                        <div className='mt-2 flex flex-wrap gap-1.5'>
                          {(
                            detail?.supported_models ??
                            selected.models.split(',').filter(Boolean)
                          )
                            .slice(0, 10)
                            .map((model) => (
                              <Badge key={model} variant='secondary'>
                                {model}
                              </Badge>
                            ))}
                        </div>
                      </div>

                      <div>
                        <div className='flex items-center justify-between gap-2'>
                          <div className='flex items-center gap-2 text-sm font-medium'>
                            <AlertTriangle className='size-4 text-amber-600' />
                            近期异常
                          </div>
                          <span className='text-muted-foreground text-[11px]'>
                            近 7 天
                          </span>
                        </div>
                        <div className='mt-2 space-y-1'>
                          {(detail?.incidents ?? []).length === 0 ? (
                            <p className='rounded-md bg-emerald-500/5 p-3 text-xs text-emerald-700 dark:text-emerald-300'>
                              未发现近期错误事件。
                            </p>
                          ) : (
                            detail?.incidents.slice(0, 5).map((incident) => (
                              <div
                                key={incident.id}
                                className='rounded-md border px-3 py-2.5'
                              >
                                <p className='line-clamp-2 text-xs font-medium'>
                                  {incident.title}
                                </p>
                                <p className='text-muted-foreground mt-1 text-[11px]'>
                                  {dayjs
                                    .unix(incident.created_at)
                                    .format('MM-DD HH:mm')}
                                </p>
                              </div>
                            ))
                          )}
                        </div>
                      </div>
                    </div>
                  )}
                </EnterprisePanel>
              </div>
            </div>
          </TabsContent>

          <TabsContent
            value='classic'
            className='min-h-0 flex-1 overflow-auto pb-5'
          >
            <EnterprisePanel
              title='经典渠道配置'
              description='保留原系统全部渠道增删改查、测试、余额更新、多 Key 管理和上游模型操作。'
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
