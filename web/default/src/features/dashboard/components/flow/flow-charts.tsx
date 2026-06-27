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
import { VChart } from '@visactor/react-vchart'
import type { EventParamsDefinition, IVChart } from '@visactor/vchart'
import {
  Activity,
  ChevronRight,
  CircleAlert,
  EyeOff,
  GitBranch,
  Hash,
  Info,
  Loader2,
  Route,
  WalletCards,
} from 'lucide-react'
import {
  Fragment,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import { useTranslation } from 'react-i18next'

import { EnterpriseStatCard } from '@/components/enterprise'
import { MultiSelect } from '@/components/multi-select'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Toggle } from '@/components/ui/toggle'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { getFlowQuotaDates } from '@/features/dashboard/api'
import {
  buildDashboardFlowData,
  buildFlowSankeySpec,
  buildQueryParams,
  flowNodeFilterFromSankeyDatum,
  flowSankeyDatumValue,
  getDefaultDays,
  getFlowStages,
} from '@/features/dashboard/lib'
import {
  compactFlowSelectionLabel,
  flowDisplayState,
  requireSuccessfulFlowRows,
} from '@/features/dashboard/lib/flow-selection'
import type {
  DashboardFilters,
  FlowLinkSelection,
  FlowMetric,
  FlowNodeFilter,
  FlowNodeKind,
  FlowOverflowMode,
  FlowRole,
} from '@/features/dashboard/types'
import { formatQuota } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { computeTimeRange } from '@/lib/time'
import { useChartTheme } from '@/lib/use-chart-theme'
import { cn } from '@/lib/utils'
import { VCHART_OPTION } from '@/lib/vchart'
import { useAuthStore } from '@/stores/auth-store'

import { FlowNodeFilterControl } from './flow-node-filter'

interface FlowChartsProps {
  filters?: DashboardFilters
  // When false, sensitive node labels are masked in the rendered Sankey.
  sensitiveVisible?: boolean
}

const FLOW_METRIC_OPTIONS = [
  { value: 'quota', label: '按额度', icon: WalletCards },
  { value: 'tokens', label: '按 Tokens', icon: Hash },
  { value: 'requests', label: '按请求', icon: Activity },
] as const

const FLOW_METRIC_LABEL_KEYS: Record<FlowMetric, string> = {
  quota: '额度',
  tokens: 'Tokens',
  requests: '请求',
}

const FLOW_TOP_LIMIT_OPTIONS = [10, 20, 50, 100] as const

const DEFAULT_FLOW_TOP_NODE_LIMIT = 10

const FLOW_OVERFLOW_MODE_OPTIONS = [
  { value: 'aggregate', label: '合并为其他' },
  { value: 'hide', label: '隐藏' },
] as const

// A Sankey needs at least two columns to render any link.
const MIN_VISIBLE_STAGES = 2

const FLOW_STAGE_META: Record<
  FlowNodeKind,
  { labelKey: string; descKey: string }
> = {
  user: {
    labelKey: '用户',
    descKey: '发起请求的下游用户',
  },
  node: {
    labelKey: '节点',
    descKey: '处理请求的部署节点',
  },
  token: {
    labelKey: '密钥',
    descKey: '本次调用使用的 API Key',
  },
  group: {
    labelKey: '分组',
    descKey: '请求命中的用户分组',
  },
  model: {
    labelKey: '模型',
    descKey: '下游请求的模型',
  },
  channel: {
    labelKey: '渠道',
    descKey: '最终承载请求的上游渠道',
  },
}

const FLOW_STAGE_LABEL_KEYS: Record<FlowNodeKind, string> = {
  user: FLOW_STAGE_META.user.labelKey,
  node: FLOW_STAGE_META.node.labelKey,
  token: FLOW_STAGE_META.token.labelKey,
  group: FLOW_STAGE_META.group.labelKey,
  model: FLOW_STAGE_META.model.labelKey,
  channel: FLOW_STAGE_META.channel.labelKey,
}

const FLOW_OTHER_NODE_LABEL_KEYS: Record<FlowNodeKind, string> = {
  user: '其他用户',
  node: '其他节点',
  token: '其他密钥',
  group: '其他分组',
  model: '其他模型',
  channel: '其他渠道',
}

type FlowChartPointerEvent = EventParamsDefinition['pointerdown']

function chartRecordValue(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object'
    ? (value as Record<string, unknown>)
    : undefined
}

function looksLikeFlowDatum(value: unknown): boolean {
  const record = chartRecordValue(value)
  if (!record) return false
  return (
    (record.key !== undefined && record.kind !== undefined) ||
    (record.source !== undefined && record.target !== undefined)
  )
}

function chartGraphicDatum(value: unknown): unknown {
  const record = chartRecordValue(value)
  const context = chartRecordValue(record?.context)
  const data = context?.data
  if (Array.isArray(data)) return data[0]
  return data
}

function flowChartEventDatum(event: FlowChartPointerEvent): unknown {
  const record = chartRecordValue(event)
  if (!record) return undefined

  if (record.datum !== undefined && record.datum !== null) return record.datum

  const itemRecord = chartRecordValue(record.item)
  if (itemRecord?.datum !== undefined && itemRecord.datum !== null) {
    return itemRecord.datum
  }

  const graphicDatum = chartGraphicDatum(record.item)
  if (graphicDatum !== undefined && graphicDatum !== null) return graphicDatum

  const itemData = itemRecord?.data
  if (Array.isArray(itemData)) return itemData[0]
  if (itemData !== undefined && itemData !== null) return itemData

  return looksLikeFlowDatum(record) ? record : undefined
}

function flowNodeFilterKey(filter: FlowNodeFilter): string {
  return `${filter.kind}\u0000${filter.id}`
}

function isSameFlowNodeFilter(
  a: FlowNodeFilter | undefined,
  b: FlowNodeFilter
): boolean {
  return Boolean(a && a.kind === b.kind && a.id === b.id)
}

function toggleSelectedValue(values: string[], value: string): string[] {
  return values.includes(value)
    ? values.filter((item) => item !== value)
    : [...values, value]
}

function toggleSelectedNodeFilter(
  filters: FlowNodeFilter[],
  filter: FlowNodeFilter
): FlowNodeFilter[] {
  const key = flowNodeFilterKey(filter)
  const hasFilter = filters.some((item) => flowNodeFilterKey(item) === key)
  return hasFilter
    ? filters.filter((item) => flowNodeFilterKey(item) !== key)
    : [...filters, filter]
}

function formatFlowMetricNumber(value: number): string {
  return Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(
    value
  )
}

export function FlowCharts(props: FlowChartsProps) {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const chartInstanceRef = useRef<IVChart | null>(null)
  const user = useAuthStore((state) => state.auth.user)
  const isRoot = Boolean(user?.role && user.role >= ROLE.SUPER_ADMIN)
  const isAdmin = Boolean(user?.role && user.role >= ROLE.ADMIN)
  let flowRole: FlowRole = 'user'
  if (isRoot) {
    flowRole = 'root'
  } else if (isAdmin) {
    flowRole = 'admin'
  }
  const [metric, setMetric] = useState<FlowMetric>('quota')
  const [topNodeLimit, setTopNodeLimit] = useState(DEFAULT_FLOW_TOP_NODE_LIMIT)
  const [overflowMode, setOverflowMode] =
    useState<FlowOverflowMode>('aggregate')
  const [selectedUsers, setSelectedUsers] = useState<string[]>([])
  const [selectedNodes, setSelectedNodes] = useState<FlowNodeFilter[]>([])
  const [activeFlowNode, setActiveFlowNode] = useState<
    FlowNodeFilter | undefined
  >()
  const [activeFlowLink, setActiveFlowLink] = useState<
    FlowLinkSelection | undefined
  >()
  const [hiddenStages, setHiddenStages] = useState<FlowNodeKind[]>([])

  const stages = useMemo(() => getFlowStages(flowRole), [flowRole])
  const visibleStages = useMemo(
    () => stages.filter((stage) => !hiddenStages.includes(stage)),
    [stages, hiddenStages]
  )
  useEffect(() => {
    const visible = new Set(visibleStages)
    setSelectedNodes((prev) => {
      const next = prev.filter((filter) => visible.has(filter.kind))
      return next.length === prev.length ? prev : next
    })
    setActiveFlowNode((prev) =>
      prev && visible.has(prev.kind) ? prev : undefined
    )
    // The graph reshapes when columns are toggled, so any highlighted edge may
    // no longer exist. Drop the link selection rather than leave it dangling.
    setActiveFlowLink(undefined)
  }, [visibleStages])
  const toggleStage = (stage: FlowNodeKind) => {
    setHiddenStages((prev) => {
      const hidden = new Set(prev)
      if (hidden.has(stage)) {
        hidden.delete(stage)
      } else {
        const remaining = stages.filter((item) => !hidden.has(item)).length
        if (remaining <= MIN_VISIBLE_STAGES) return prev
        hidden.add(stage)
      }
      return stages.filter((item) => hidden.has(item))
    })
  }

  const timeRange = useMemo(
    () =>
      computeTimeRange(
        getDefaultDays(props.filters?.time_granularity),
        props.filters?.start_timestamp,
        props.filters?.end_timestamp
      ),
    [
      props.filters?.end_timestamp,
      props.filters?.start_timestamp,
      props.filters?.time_granularity,
    ]
  )
  const flowQueryParams = useMemo(
    () => buildQueryParams(timeRange, props.filters),
    [props.filters, timeRange]
  )

  const {
    data: flowRows,
    error: flowError,
    isError,
    isLoading,
  } = useQuery({
    queryKey: ['dashboard', 'flow', flowQueryParams, flowRole],
    queryFn: () => getFlowQuotaDates(flowQueryParams, isAdmin),
    select: (res) => requireSuccessfulFlowRows(res, '请稍后重试。'),
    staleTime: 60_000,
  })

  const maskSensitive = props.sensitiveVisible === false
  const flowData = useMemo(
    () =>
      buildDashboardFlowData(isLoading ? [] : (flowRows ?? []), metric, {
        role: flowRole,
        selectedUsers,
        selectedNodes,
        activeNode: activeFlowNode,
        activeLink: activeFlowLink,
        visibleStages,
        topNodeLimit,
        overflowMode,
        maskSensitive,
        deletedTokenLabel: (tokenId) => `已删除 (${tokenId})`,
        otherNodeLabel: (kind) => FLOW_OTHER_NODE_LABEL_KEYS[kind],
      }),
    [
      flowRole,
      flowRows,
      isLoading,
      metric,
      overflowMode,
      activeFlowNode,
      activeFlowLink,
      selectedNodes,
      selectedUsers,
      topNodeLimit,
      visibleStages,
      maskSensitive,
      t,
    ]
  )
  const userFilterOptions = useMemo(
    () =>
      flowData.filterOptions.users.map((user) => ({
        label: `${user.label} · ${user.valueLabel}`,
        value: user.value,
      })),
    [flowData.filterOptions.users]
  )
  const nodeFilterStages = useMemo(
    () => visibleStages.filter((stage) => stage !== 'user'),
    [visibleStages]
  )
  const nodeFilterOptions = useMemo(
    () =>
      flowData.filterOptions.nodes.filter((option) => option.kind !== 'user'),
    [flowData.filterOptions.nodes]
  )
  const metricLabel = FLOW_METRIC_LABEL_KEYS[metric]
  const formatNodeMetricValue = useCallback(
    (value: number) =>
      metric === 'quota' ? formatQuota(value) : formatFlowMetricNumber(value),
    [metric]
  )
  // Explicit filters (the chips/dropdown control) narrow the rows that feed the
  // chart. They are intentionally independent from the click-to-highlight state
  // below so selecting a filter never dims a node, it removes unrelated rows.
  const toggleFlowNodeFilter = useCallback((filter: FlowNodeFilter) => {
    if (filter.kind === 'user') {
      setSelectedUsers((prev) => toggleSelectedValue(prev, filter.id))
      return
    }
    setSelectedNodes((prev) => toggleSelectedNodeFilter(prev, filter))
  }, [])
  const removeFlowNodeFilter = useCallback((filter: FlowNodeFilter) => {
    if (filter.kind === 'user') {
      setSelectedUsers((prev) => prev.filter((item) => item !== filter.id))
      return
    }
    const key = flowNodeFilterKey(filter)
    setSelectedNodes((prev) =>
      prev.filter((item) => flowNodeFilterKey(item) !== key)
    )
  }, [])
  const clearFlowNodeFilters = useCallback(() => {
    setSelectedNodes([])
  }, [])
  // Clicking a node only drives the highlight: keep every node/link on screen
  // but emphasize the full paths through the clicked node and dim the rest.
  // Clicking the active node again, or clicking empty space, clears it.
  const handleChartPointerDown = useCallback((event: FlowChartPointerEvent) => {
    const datum = flowChartEventDatum(event)
    const filter = flowNodeFilterFromSankeyDatum(datum)
    if (filter) {
      setActiveFlowLink(undefined)
      setActiveFlowNode((prev) =>
        isSameFlowNodeFilter(prev, filter) ? undefined : filter
      )
      return
    }

    const source = flowSankeyDatumValue(datum, 'source')
    const target = flowSankeyDatumValue(datum, 'target')
    if (typeof source === 'string' && typeof target === 'string') {
      setActiveFlowNode(undefined)
      setActiveFlowLink((prev) =>
        prev && prev.source === source && prev.target === target
          ? undefined
          : { source, target }
      )
      return
    }

    setActiveFlowNode(undefined)
    setActiveFlowLink(undefined)
    chartInstanceRef.current?.clearState('selected')
    chartInstanceRef.current?.clearState('blur')
  }, [])
  const chartTitle = '流量路径'
  const flowSpec = useMemo(
    () =>
      buildFlowSankeySpec(flowData.flow, chartTitle, formatQuota, {
        quota: '额度',
        tokens: 'Tokens',
        requests: '请求',
        share: '占比',
      }),
    [chartTitle, flowData.flow]
  )
  const chartTheme = resolvedTheme === 'dark' ? 'dark' : 'light'
  const chartKey = [
    metric,
    topNodeLimit,
    overflowMode,
    flowRole,
    activeFlowNode ? flowNodeFilterKey(activeFlowNode) : '',
    activeFlowLink
      ? `${activeFlowLink.source}\u0000${activeFlowLink.target}`
      : '',
    selectedNodes.map(flowNodeFilterKey).join(','),
    selectedUsers.join(','),
    visibleStages.join(','),
    maskSensitive ? 'masked' : 'plain',
    flowRows?.length ?? 0,
    resolvedTheme,
  ].join('-')
  const displayState = flowDisplayState({
    isLoading,
    isError,
    linkCount: flowData.flow.links.length,
    themeReady,
  })
  const flowErrorMessage =
    flowError instanceof Error ? flowError.message : '请稍后重试。'
  let chartContent = (
    <VChart
      key={`flow-${chartKey}`}
      spec={{
        ...flowSpec,
        theme: chartTheme,
        background: 'transparent',
      }}
      option={VCHART_OPTION}
      onReady={(instance: IVChart) => {
        chartInstanceRef.current = instance
      }}
      onPointerDown={handleChartPointerDown}
    />
  )
  if (displayState === 'loading') {
    chartContent = <Skeleton className='h-full w-full' />
  } else if (displayState === 'error') {
    chartContent = (
      <div className='flex h-full items-center justify-center p-4'>
        <Alert variant='destructive' className='max-w-md'>
          <CircleAlert />
          <AlertTitle>{t('Failed to load')}</AlertTitle>
          <AlertDescription>{flowErrorMessage}</AlertDescription>
        </Alert>
      </div>
    )
  } else if (displayState === 'empty') {
    chartContent = (
      <Empty className='h-full border-0 py-12'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Route />
          </EmptyMedia>
          <EmptyTitle>暂无流量链路数据</EmptyTitle>
          <EmptyDescription>
            当前筛选范围内没有可用于构建链路的调用记录。
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  const selectedNodeCount = selectedUsers.length + selectedNodes.length
  const activeSelectionText = activeFlowNode
    ? `${FLOW_STAGE_LABEL_KEYS[activeFlowNode.kind]}：${activeFlowNode.id}`
    : activeFlowLink
      ? '已聚焦一条链路'
      : selectedNodeCount > 0
        ? `${selectedNodeCount} 个筛选条件`
        : '全量链路'

  return (
    <div className='enterprise-dashboard flex min-h-0 flex-col gap-3 overflow-auto pb-5 text-slate-950'>
      <div className='flex flex-wrap items-end justify-between gap-3'>
        <div className='min-w-0'>
          <p className='text-[11px] font-semibold text-blue-600'>
            网关调用链路
          </p>
          <h1 className='mt-1 text-[22px] leading-7 font-bold tracking-normal text-slate-950'>
            流量链路分析
          </h1>
          <p className='mt-1 text-[12px] text-slate-500'>
            从下游用户、密钥、分组、模型到上游渠道的真实调用路径。
          </p>
        </div>
        <div className='rounded-md border border-slate-200 bg-white px-3 py-2 text-right shadow-[0_1px_2px_rgb(15_23_42/0.035)]'>
          <p className='text-[11px] text-slate-500'>当前视图</p>
          <p className='mt-0.5 text-xs font-semibold text-slate-900'>
            {activeSelectionText}
          </p>
        </div>
      </div>

      <div className='grid gap-2 sm:grid-cols-2 xl:grid-cols-4'>
        <EnterpriseStatCard
          title='请求量'
          value={formatFlowMetricNumber(flowData.summary.requests)}
          helper='当前筛选'
          icon={Activity}
          tone='blue'
          loading={isLoading}
        />
        <EnterpriseStatCard
          title='消耗额度'
          value={formatQuota(flowData.summary.quota)}
          helper='按售价口径'
          icon={WalletCards}
          tone='amber'
          loading={isLoading}
        />
        <EnterpriseStatCard
          title='Token 用量'
          value={formatFlowMetricNumber(flowData.summary.tokens)}
          helper='输入与输出合计'
          icon={Hash}
          tone='violet'
          loading={isLoading}
        />
        <EnterpriseStatCard
          title='链路规模'
          value={`${flowData.flow.nodes.length} 节点`}
          helper={`${flowData.flow.links.length} 条连接`}
          icon={GitBranch}
          tone='emerald'
          loading={isLoading}
        />
      </div>

      <div className='flex flex-col gap-2 rounded-md border border-slate-200 bg-white p-3 shadow-[0_1px_2px_rgb(15_23_42/0.035)] xl:flex-row xl:items-end xl:justify-between'>
        <div className='flex min-w-0 flex-wrap items-end gap-2'>
          <div className='flex min-w-0 flex-col gap-1.5'>
            <div className='flex items-center gap-1.5'>
              <span className='text-muted-foreground text-xs font-medium'>
                流宽口径
              </span>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <button
                        type='button'
                        className='text-muted-foreground/60 hover:text-foreground flex size-5 shrink-0 items-center justify-center rounded-md'
                        aria-label='流宽口径'
                      />
                    }
                  >
                    <Info className='size-3.5' />
                  </TooltipTrigger>
                  <TooltipContent className='max-w-[14rem]'>
                    选择链路宽度按额度、Token 还是请求次数计算。
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
            <Tabs
              value={metric}
              onValueChange={(value) => setMetric(value as FlowMetric)}
              className='shrink-0'
            >
              <TabsList aria-label='流宽口径'>
                {FLOW_METRIC_OPTIONS.map((option) => {
                  const Icon = option.icon
                  return (
                    <TabsTrigger
                      key={option.value}
                      value={option.value}
                      className='gap-1.5 px-2.5 text-xs'
                    >
                      <Icon data-icon='inline-start' aria-hidden='true' />
                      {option.label}
                    </TabsTrigger>
                  )
                })}
              </TabsList>
            </Tabs>
          </div>

          <div className='flex min-w-0 flex-col gap-1.5'>
            <span className='text-muted-foreground text-xs font-medium'>
              节点数量
            </span>
            <Tabs
              value={String(topNodeLimit)}
              onValueChange={(value) => setTopNodeLimit(Number(value))}
              className='shrink-0'
            >
              <TabsList aria-label='节点数量'>
                {FLOW_TOP_LIMIT_OPTIONS.map((limit) => (
                  <TabsTrigger
                    key={limit}
                    value={String(limit)}
                    className='px-2.5 text-xs'
                  >
                    Top {limit}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </div>

          <div className='flex min-w-0 flex-col gap-1.5'>
            <span className='text-muted-foreground text-xs font-medium'>
              超出节点
            </span>
            <Tabs
              value={overflowMode}
              onValueChange={(value) =>
                setOverflowMode(value as FlowOverflowMode)
              }
              className='shrink-0'
            >
              <TabsList aria-label='超出节点'>
                {FLOW_OVERFLOW_MODE_OPTIONS.map((option) => (
                  <TabsTrigger
                    key={option.value}
                    value={option.value}
                    className='px-2.5 text-xs'
                  >
                    {option.label}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </div>

          <FlowNodeFilterControl
            stages={nodeFilterStages}
            stageLabels={FLOW_STAGE_LABEL_KEYS}
            metricLabel={metricLabel}
            formatMetricValue={formatNodeMetricValue}
            options={nodeFilterOptions}
            selectedNodes={selectedNodes}
            onToggleNode={toggleFlowNodeFilter}
            onRemoveNode={removeFlowNodeFilter}
            onClearNodes={clearFlowNodeFilters}
          />
        </div>

        <div className='flex min-w-0 items-center gap-2 xl:justify-end'>
          {isAdmin && (
            <div className='flex min-w-0 flex-col gap-2 sm:flex-row xl:w-[min(24rem,34vw)]'>
              <MultiSelect
                options={userFilterOptions}
                selected={selectedUsers}
                onChange={setSelectedUsers}
                placeholder='全部用户'
                emptyText='暂无用户'
                maxVisibleChips={2}
                renderSelectedSummary={(values) =>
                  compactFlowSelectionLabel(values.length)
                }
              />
            </div>
          )}
          {isLoading && (
            <Loader2 className='text-muted-foreground size-4 animate-spin' />
          )}
        </div>
      </div>

      <div className='overflow-hidden rounded-md border border-slate-200 bg-white shadow-[0_1px_2px_rgb(15_23_42/0.035)]'>
        <div className='flex w-full flex-col gap-2 border-b border-slate-100 bg-slate-50/65 px-3 py-2 lg:flex-row lg:items-center lg:justify-between'>
          <div className='flex min-w-0 items-center gap-2'>
            <GitBranch className='text-muted-foreground/60 size-4 shrink-0' />
            <div>
              <div className='text-[13px] font-semibold text-slate-900'>
                {chartTitle}
              </div>
              <p className='text-[11px] text-slate-500'>
                点击节点或连线可聚焦完整上下游路径。
              </p>
            </div>
          </div>
          <TooltipProvider>
            <div className='flex min-w-0 items-center gap-1 overflow-x-auto pb-1 lg:justify-end lg:pb-0'>
              <Tooltip>
                <TooltipTrigger
                  render={
                    <button
                      type='button'
                      className='text-muted-foreground/60 hover:text-foreground flex size-6 shrink-0 items-center justify-center rounded-md'
                      aria-label='显示或隐藏链路阶段'
                    />
                  }
                >
                  <Info className='size-3.5' />
                </TooltipTrigger>
                <TooltipContent className='max-w-[16rem]'>
                  点击阶段可显示或隐藏该列。
                </TooltipContent>
              </Tooltip>
              {stages.map((stage, index) => {
                const meta = FLOW_STAGE_META[stage]
                const visible = !hiddenStages.includes(stage)
                return (
                  <Fragment key={stage}>
                    {index > 0 && (
                      <ChevronRight className='text-muted-foreground/40 size-3.5 shrink-0' />
                    )}
                    <Tooltip>
                      <TooltipTrigger
                        render={
                          <Toggle
                            variant='outline'
                            size='sm'
                            pressed={visible}
                            onPressedChange={() => toggleStage(stage)}
                            aria-label={meta.labelKey}
                            className={cn('shrink-0', !visible && 'opacity-50')}
                          />
                        }
                      >
                        {!visible && <EyeOff className='size-3' />}
                        {meta.labelKey}
                      </TooltipTrigger>
                      <TooltipContent>{meta.descKey}</TooltipContent>
                    </Tooltip>
                  </Fragment>
                )
              })}
            </div>
          </TooltipProvider>
        </div>
        <div className='h-[320px] p-1.5 sm:h-[380px] sm:p-2 2xl:h-[440px]'>
          {chartContent}
        </div>
      </div>

      <div className='grid gap-3 lg:grid-cols-3'>
        <section className='rounded-md border border-slate-200 bg-white p-3 shadow-[0_1px_2px_rgb(15_23_42/0.03)]'>
          <div className='flex items-center justify-between gap-2'>
            <div>
              <h3 className='text-[13px] font-semibold text-slate-900'>
                链路阶段
              </h3>
              <p className='mt-0.5 text-[11px] text-slate-500'>
                当前参与构图的路径列。
              </p>
            </div>
            <span className='rounded-md bg-blue-50 px-2 py-1 text-[11px] font-semibold text-blue-700'>
              {visibleStages.length}/{stages.length}
            </span>
          </div>
          <div className='mt-3 flex flex-wrap gap-1.5'>
            {stages.map((stage) => {
              const visible = visibleStages.includes(stage)
              return (
                <span
                  key={stage}
                  className={cn(
                    'rounded-md border px-2 py-1 text-[11px] font-medium',
                    visible
                      ? 'border-blue-100 bg-blue-50 text-blue-700'
                      : 'border-slate-200 bg-slate-50 text-slate-400'
                  )}
                >
                  {FLOW_STAGE_LABEL_KEYS[stage]}
                </span>
              )
            })}
          </div>
        </section>

        <section className='rounded-md border border-slate-200 bg-white p-3 shadow-[0_1px_2px_rgb(15_23_42/0.03)]'>
          <div className='flex items-center justify-between gap-2'>
            <div>
              <h3 className='text-[13px] font-semibold text-slate-900'>
                筛选状态
              </h3>
              <p className='mt-0.5 text-[11px] text-slate-500'>
                用户和节点筛选会直接影响构图数据。
              </p>
            </div>
            <span className='rounded-md bg-slate-100 px-2 py-1 text-[11px] font-semibold text-slate-600'>
              {selectedNodeCount || 0}
            </span>
          </div>
          <div className='mt-3 grid grid-cols-2 gap-2'>
            <div className='rounded-md border border-slate-200 bg-slate-50/70 px-2.5 py-2'>
              <p className='text-[11px] text-slate-500'>用户筛选</p>
              <p className='mt-1 text-sm font-semibold text-slate-900 tabular-nums'>
                {selectedUsers.length}
              </p>
            </div>
            <div className='rounded-md border border-slate-200 bg-slate-50/70 px-2.5 py-2'>
              <p className='text-[11px] text-slate-500'>节点筛选</p>
              <p className='mt-1 text-sm font-semibold text-slate-900 tabular-nums'>
                {selectedNodes.length}
              </p>
            </div>
          </div>
        </section>

        <section className='rounded-md border border-slate-200 bg-white p-3 shadow-[0_1px_2px_rgb(15_23_42/0.03)]'>
          <div className='flex items-center justify-between gap-2'>
            <div>
              <h3 className='text-[13px] font-semibold text-slate-900'>
                数据状态
              </h3>
              <p className='mt-0.5 text-[11px] text-slate-500'>
                来源于真实调用流水聚合接口。
              </p>
            </div>
            <span
              className={cn(
                'rounded-md px-2 py-1 text-[11px] font-semibold',
                isError
                  ? 'bg-rose-50 text-rose-700'
                  : isLoading
                    ? 'bg-amber-50 text-amber-700'
                    : 'bg-emerald-50 text-emerald-700'
              )}
            >
              {isError ? '异常' : isLoading ? '加载中' : '已同步'}
            </span>
          </div>
          <div className='mt-3 grid grid-cols-2 gap-2'>
            <div className='rounded-md border border-slate-200 bg-slate-50/70 px-2.5 py-2'>
              <p className='text-[11px] text-slate-500'>原始行数</p>
              <p className='mt-1 text-sm font-semibold text-slate-900 tabular-nums'>
                {formatFlowMetricNumber(flowRows?.length ?? 0)}
              </p>
            </div>
            <div className='rounded-md border border-slate-200 bg-slate-50/70 px-2.5 py-2'>
              <p className='text-[11px] text-slate-500'>接口</p>
              <p className='mt-1 truncate text-sm font-semibold text-slate-900'>
                {isAdmin ? '全局' : '个人'}
              </p>
            </div>
          </div>
        </section>
      </div>
    </div>
  )
}
