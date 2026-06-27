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
  AlertTriangle,
  ArrowUpRight,
  BadgeDollarSign,
  Boxes,
  BrainCircuit,
  Clock3,
  DatabaseZap,
  FileQuestion,
  Layers3,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  ShieldCheck,
  SlidersHorizontal,
} from 'lucide-react'
import { useMemo, useState } from 'react'

import {
  EnterprisePageHeader,
  EnterprisePanel,
  EnterpriseStatCard,
} from '@/components/enterprise'
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
import { useEnterpriseConsole } from '@/context/enterprise-console-context'
import { getEnterpriseOverview } from '@/features/enterprise/api'
import type {
  EnterpriseOverviewChannelItem,
  EnterpriseOverviewRankingItem,
} from '@/features/enterprise/types'
import {
  getOptionValue,
  useSystemOptions,
} from '@/features/system-settings/hooks/use-system-options'
import { cn } from '@/lib/utils'

import { getMissingModels, getModels, getVendors, searchModels } from '../api'
import { modelsQueryKeys, vendorsQueryKeys } from '../lib'
import type { Model, Vendor } from '../types'
import { useModels } from './models-provider'

type ModelTypeFilter =
  | 'all'
  | 'text'
  | 'vision'
  | 'reasoning'
  | 'embedding'
  | 'audio'

type VisibilityFilter = 'all' | 'global' | 'restricted'

type PricingMaps = {
  prices: Record<string, number>
  ratios: Record<string, number>
}

type CandidateModelRow = {
  key: string
  name: string
  channels: string[]
  requests: number
  tokens: number
  quota: number
  source: 'missing' | 'channel' | 'usage'
}

type ModelCatalogRow =
  | {
      type: 'configured'
      model: Model
    }
  | {
      type: 'candidate'
      candidate: CandidateModelRow
    }

const MODEL_PAGE_SIZE = 200
const TABLE_LIMIT = 8
const EMPTY_MODELS: Model[] = []
const EMPTY_VENDORS: Vendor[] = []
const EMPTY_MISSING_MODELS: string[] = []
const EMPTY_TOP_MODELS: EnterpriseOverviewRankingItem[] = []
const EMPTY_OVERVIEW_CHANNELS: EnterpriseOverviewChannelItem[] = []

const SYSTEM_OPTION_DEFAULTS = {
  ModelPrice: '',
  ModelRatio: '',
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat('zh-CN', {
    maximumFractionDigits: 1,
    notation: Math.abs(value) >= 10000 ? 'compact' : 'standard',
  }).format(value)
}

function formatPercent(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return '--'
  return `${(value * 100).toFixed(1)}%`
}

function parseJsonMap(raw: string): Record<string, number> {
  if (!raw.trim()) return {}
  try {
    const parsed = JSON.parse(raw) as unknown
    if (parsed == null || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return {}
    }

    return Object.entries(parsed as Record<string, unknown>).reduce(
      (result, [key, value]) => {
        const numberValue =
          typeof value === 'number' ? value : Number(String(value).trim())
        if (Number.isFinite(numberValue)) {
          result[key] = numberValue
        }
        return result
      },
      {} as Record<string, number>
    )
  } catch {
    return {}
  }
}

function getModelKeyValue(
  source: Record<string, number>,
  modelName: string
): number | undefined {
  if (modelName in source) return source[modelName]
  const lowerName = modelName.toLowerCase()
  const matchedEntry = Object.entries(source).find(
    ([key]) => key.toLowerCase() === lowerName
  )
  return matchedEntry?.[1]
}

function getModelPricingLabel(modelName: string, pricing: PricingMaps): string {
  const price = getModelKeyValue(pricing.prices, modelName)
  if (price != null) {
    return `$${price.toFixed(price >= 1 ? 2 : 4)}`
  }

  const ratio = getModelKeyValue(pricing.ratios, modelName)
  if (ratio != null) {
    return `${ratio.toFixed(ratio >= 1 ? 2 : 4)}x`
  }

  return '未配置'
}

function getPricingLabel(model: Model, pricing: PricingMaps): string {
  return getModelPricingLabel(model.model_name, pricing)
}

function splitText(value?: string): string[] {
  if (!value) return []
  return value
    .split(/[,，\s]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function getCapabilitiesFromText(sourceText: string, tagText?: string): string[] {
  const source = `${sourceText} ${tagText ?? ''}`.toLowerCase()
  const tags = splitText(tagText)
  const capabilities = new Set<string>()

  if (/vision|image|图片|视觉|multimodal|omni/.test(source)) {
    capabilities.add('视觉')
  }
  if (/reason|r1|thinking|推理/.test(source)) {
    capabilities.add('推理')
  }
  if (/embed|embedding|向量/.test(source)) {
    capabilities.add('向量')
  }
  if (/audio|tts|whisper|语音|音频/.test(source)) {
    capabilities.add('语音')
  }
  if (/code|coder|代码/.test(source)) {
    capabilities.add('代码')
  }

  tags.slice(0, 2).forEach((tag) => capabilities.add(tag))

  if (capabilities.size === 0) {
    capabilities.add('文本')
  }

  return [...capabilities].slice(0, 4)
}

function getModelCapabilities(model: Model): string[] {
  return getCapabilitiesFromText(
    [model.model_name, model.description ?? '', model.endpoints ?? ''].join(' '),
    model.tags
  )
}

function getCandidateCapabilities(modelName: string): string[] {
  return getCapabilitiesFromText(modelName)
}

function matchesModelType(model: Model, typeFilter: ModelTypeFilter): boolean {
  if (typeFilter === 'all') return true
  const text = [
    model.model_name,
    model.description ?? '',
    model.tags ?? '',
    model.endpoints ?? '',
  ]
    .join(' ')
    .toLowerCase()

  if (typeFilter === 'vision') {
    return /vision|image|图片|视觉|multimodal|omni/.test(text)
  }
  if (typeFilter === 'reasoning') {
    return /reason|r1|thinking|推理/.test(text)
  }
  if (typeFilter === 'embedding') {
    return /embed|embedding|向量/.test(text)
  }
  if (typeFilter === 'audio') {
    return /audio|tts|whisper|语音|音频/.test(text)
  }
  return !/embed|embedding|image|vision|audio|tts|whisper/.test(text)
}

function matchesModelNameType(
  modelName: string,
  typeFilter: ModelTypeFilter
): boolean {
  if (typeFilter === 'all') return true
  const text = modelName.toLowerCase()

  if (typeFilter === 'vision') {
    return /vision|image|图片|视觉|multimodal|omni/.test(text)
  }
  if (typeFilter === 'reasoning') {
    return /reason|r1|thinking|推理/.test(text)
  }
  if (typeFilter === 'embedding') {
    return /embed|embedding|向量/.test(text)
  }
  if (typeFilter === 'audio') {
    return /audio|tts|whisper|语音|音频/.test(text)
  }
  return !/embed|embedding|image|vision|audio|tts|whisper/.test(text)
}

function matchesVisibility(
  model: Model,
  visibility: VisibilityFilter
): boolean {
  if (visibility === 'all') return true
  const groups = model.enable_groups ?? []
  if (visibility === 'restricted') return groups.length > 0
  return groups.length === 0
}

function modelStatus(model: Model): {
  label: string
  className: string
  dotClassName: string
} {
  if (model.status === 1) {
    return {
      label: '已上架',
      className: 'border-emerald-200 bg-emerald-50 text-emerald-700',
      dotClassName: 'bg-emerald-500',
    }
  }

  return {
    label: '已停用',
    className: 'border-slate-200 bg-slate-50 text-slate-500',
    dotClassName: 'bg-slate-400',
  }
}

function getVendorName(model: Model, vendorMap: Map<number, Vendor>): string {
  if (model.vendor_id == null) return '未绑定供应商'
  return vendorMap.get(model.vendor_id)?.name ?? `供应商 #${model.vendor_id}`
}

function buildEndpointSummary(model: Model): string {
  const endpoints = splitText(model.endpoints)
  const channels = model.bound_channels ?? []

  if (channels.length > 0) {
    return `${channels.length} 条渠道`
  }

  if (endpoints.length > 0) {
    return `${endpoints.length} 个端点`
  }

  return '待绑定渠道'
}

function getVisibleScope(model: Model): string {
  const groups = model.enable_groups ?? []
  if (groups.length === 0) return '全局可见'
  if (groups.length <= 2) return groups.join(' / ')
  return `${groups.slice(0, 2).join(' / ')} +${groups.length - 2}`
}

function getModelRiskCount(model: Model): number {
  let count = 0
  if (model.status !== 1) count += 1
  if (model.vendor_id == null) count += 1
  if ((model.bound_channels?.length ?? 0) === 0 && !model.endpoints) count += 1
  return count
}

function normalizeModelName(value: string): string {
  return value.trim()
}

function buildCandidateModelRows({
  missingModels,
  topModels,
  channels,
  configuredNames,
}: {
  missingModels: string[]
  topModels: EnterpriseOverviewRankingItem[]
  channels: EnterpriseOverviewChannelItem[]
  configuredNames: Set<string>
}): CandidateModelRow[] {
  const rows = new Map<string, CandidateModelRow>()

  const ensureRow = (
    rawName: string,
    source: CandidateModelRow['source']
  ): CandidateModelRow | null => {
    const name = normalizeModelName(rawName)
    if (!name) return null

    const key = name.toLowerCase()
    if (configuredNames.has(key)) return null

    const existing = rows.get(key)
    if (existing) {
      if (existing.source === 'channel' && source !== 'channel') {
        existing.source = source
      }
      return existing
    }

    const row: CandidateModelRow = {
      key,
      name,
      channels: [],
      requests: 0,
      tokens: 0,
      quota: 0,
      source,
    }
    rows.set(key, row)
    return row
  }

  missingModels.forEach((modelName) => ensureRow(modelName, 'missing'))

  channels.forEach((channel) => {
    splitText(channel.models).forEach((modelName) => {
      const row = ensureRow(modelName, 'channel')
      if (!row) return
      if (!row.channels.includes(channel.name)) {
        row.channels.push(channel.name)
      }
    })
  })

  topModels.forEach((model) => {
    const row = ensureRow(model.name, 'usage')
    if (!row) return
    row.requests += model.requests
    row.tokens += model.tokens
    row.quota += model.quota
    if (row.source === 'channel') {
      row.source = 'usage'
    }
  })

  return [...rows.values()].sort((left, right) => {
    if (right.requests !== left.requests) return right.requests - left.requests
    if (right.channels.length !== left.channels.length) {
      return right.channels.length - left.channels.length
    }
    return left.name.localeCompare(right.name)
  })
}

function getCandidateSourceLabel(candidate: CandidateModelRow): string {
  if (candidate.source === 'missing') return '真实请求发现'
  if (candidate.source === 'usage') return '用量排行发现'
  return '渠道声明发现'
}

function getCandidateChannelLabel(candidate: CandidateModelRow): string {
  if (candidate.channels.length === 0) return getCandidateSourceLabel(candidate)
  if (candidate.channels.length <= 2) return candidate.channels.join(' / ')
  return `${candidate.channels.slice(0, 2).join(' / ')} +${
    candidate.channels.length - 2
  }`
}

function TableEmptyState({
  loading,
  onCreate,
  onSync,
}: {
  loading: boolean
  onCreate: () => void
  onSync: () => void
}) {
  return (
    <TableRow>
      <TableCell colSpan={6} className='h-[300px] text-center'>
        {loading ? (
          <div className='flex flex-col items-center justify-center gap-2 text-[12px] text-slate-500'>
            <RefreshCw className='size-5 animate-spin text-blue-500' />
            正在加载模型资产
          </div>
        ) : (
          <div className='mx-auto flex max-w-sm flex-col items-center justify-center gap-2 text-center'>
            <span className='flex size-10 items-center justify-center rounded-md bg-blue-50 text-blue-600 ring-1 ring-blue-100'>
              <FileQuestion className='size-5' />
            </span>
            <div>
              <p className='text-sm font-semibold text-slate-900'>
                暂无模型资产
              </p>
              <p className='mt-1 text-[12px] leading-5 text-slate-500'>
                当前筛选条件下没有模型。可以从上游同步，也可以手动新增模型后绑定供应商与渠道。
              </p>
            </div>
            <div className='mt-1 flex items-center gap-2'>
              <Button
                size='sm'
                variant='outline'
                className='h-7 rounded-md px-2 text-[12px]'
                onClick={onSync}
              >
                <RefreshCw className='size-3.5' />
                同步模型
              </Button>
              <Button
                size='sm'
                className='h-7 rounded-md px-2 text-[12px]'
                onClick={onCreate}
              >
                <Plus className='size-3.5' />
                新增模型
              </Button>
            </div>
          </div>
        )}
      </TableCell>
    </TableRow>
  )
}

export function EnterpriseModelCenter({
  onSectionChange,
}: {
  onSectionChange: (section: string) => void
}) {
  const { range, granularity } = useEnterpriseConsole()
  const { setOpen, setCurrentRow } = useModels()
  const [keyword, setKeyword] = useState('')
  const [typeFilter, setTypeFilter] = useState<ModelTypeFilter>('all')
  const [vendorFilter, setVendorFilter] = useState('all')
  const [visibilityFilter, setVisibilityFilter] =
    useState<VisibilityFilter>('all')
  const [statusFilter, setStatusFilter] = useState('all')

  const baseModelParams = useMemo(
    () => ({
      p: 1,
      page_size: MODEL_PAGE_SIZE,
      ...(vendorFilter !== 'all' ? { vendor: vendorFilter } : {}),
      ...(statusFilter !== 'all' ? { status: statusFilter } : {}),
    }),
    [statusFilter, vendorFilter]
  )
  const trimmedKeyword = keyword.trim()

  const summaryQuery = useQuery({
    queryKey: modelsQueryKeys.list({ p: 1, page_size: MODEL_PAGE_SIZE }),
    queryFn: () => getModels({ p: 1, page_size: MODEL_PAGE_SIZE }),
    staleTime: 30_000,
  })

  const modelsQuery = useQuery({
    queryKey: modelsQueryKeys.list({
      ...baseModelParams,
      keyword: trimmedKeyword,
    }),
    queryFn: () =>
      trimmedKeyword.length > 0
        ? searchModels({ ...baseModelParams, keyword: trimmedKeyword })
        : getModels(baseModelParams),
    staleTime: 30_000,
  })

  const vendorsQuery = useQuery({
    queryKey: vendorsQueryKeys.list({ p: 1, page_size: 1000 }),
    queryFn: () => getVendors({ p: 1, page_size: 1000 }),
    staleTime: 60_000,
  })

  const missingModelsQuery = useQuery({
    queryKey: modelsQueryKeys.missing(),
    queryFn: getMissingModels,
    staleTime: 60_000,
  })

  const overviewQuery = useQuery({
    queryKey: [
      'model-center-enterprise-overview',
      range.start,
      range.end,
      granularity,
    ],
    queryFn: () =>
      getEnterpriseOverview({
        start_timestamp: range.start,
        end_timestamp: range.end,
        time_granularity: granularity,
      }),
    staleTime: 30_000,
  })

  const systemOptionsQuery = useSystemOptions()
  const pricing = useMemo<PricingMaps>(() => {
    const options = getOptionValue(
      systemOptionsQuery.data?.data,
      SYSTEM_OPTION_DEFAULTS
    )
    return {
      prices: parseJsonMap(options.ModelPrice),
      ratios: parseJsonMap(options.ModelRatio),
    }
  }, [systemOptionsQuery.data?.data])

  const vendors = useMemo(
    () => vendorsQuery.data?.data?.items ?? EMPTY_VENDORS,
    [vendorsQuery.data?.data?.items]
  )
  const vendorMap = useMemo(
    () => new Map(vendors.map((vendor) => [vendor.id, vendor])),
    [vendors]
  )

  const summaryModels = useMemo(
    () => summaryQuery.data?.data?.items ?? EMPTY_MODELS,
    [summaryQuery.data?.data?.items]
  )
  const missingModelNames = useMemo(
    () => missingModelsQuery.data?.data ?? EMPTY_MISSING_MODELS,
    [missingModelsQuery.data?.data]
  )
  const topModels = useMemo(
    () => overviewQuery.data?.data?.top_models ?? EMPTY_TOP_MODELS,
    [overviewQuery.data?.data?.top_models]
  )
  const overviewChannels = useMemo(
    () => overviewQuery.data?.data?.channels ?? EMPTY_OVERVIEW_CHANNELS,
    [overviewQuery.data?.data?.channels]
  )
  const configuredNames = useMemo(
    () =>
      new Set(
        summaryModels.map((model) => normalizeModelName(model.model_name).toLowerCase())
      ),
    [summaryModels]
  )
  const candidateRows = useMemo(
    () =>
      buildCandidateModelRows({
        missingModels: missingModelNames,
        topModels,
        channels: overviewChannels,
        configuredNames,
      }),
    [configuredNames, missingModelNames, overviewChannels, topModels]
  )
  const listedCount = summaryModels.filter((model) => model.status === 1).length
  const trustedCount = summaryModels.filter(
    (model) =>
      model.sync_official === 1 || (model.enable_groups?.length ?? 0) > 0
  ).length
  const pricedCount = summaryModels.filter(
    (model) => getPricingLabel(model, pricing) !== '未配置'
  ).length
  const abnormalCount = summaryModels.filter(
    (model) => getModelRiskCount(model) > 0
  ).length
  const totalModels = summaryQuery.data?.data?.total ?? summaryModels.length
  const modelCatalogCount = totalModels + candidateRows.length
  const missingCount = missingModelNames.length
  const activeVendorCount = vendors.filter(
    (vendor) => vendor.status === 1
  ).length
  const grossMarginRate =
    overviewQuery.data?.data?.metrics.gross_margin_rate ?? null

  const tableModels = useMemo(
    () => modelsQuery.data?.data?.items ?? EMPTY_MODELS,
    [modelsQuery.data?.data?.items]
  )
  const filteredModels = useMemo(
    () =>
      tableModels.filter(
        (model) =>
          matchesModelType(model, typeFilter) &&
          matchesVisibility(model, visibilityFilter)
      ),
    [tableModels, typeFilter, visibilityFilter]
  )
  const filteredCandidateRows = useMemo(
    () =>
      candidateRows.filter((candidate) => {
        const keywordMatched =
          trimmedKeyword.length === 0 ||
          candidate.name.toLowerCase().includes(trimmedKeyword.toLowerCase()) ||
          candidate.channels.some((channel) =>
            channel.toLowerCase().includes(trimmedKeyword.toLowerCase())
          )

        return (
          keywordMatched &&
          matchesModelNameType(candidate.name, typeFilter) &&
          vendorFilter === 'all' &&
          statusFilter === 'all' &&
          visibilityFilter !== 'restricted'
        )
      }),
    [candidateRows, statusFilter, trimmedKeyword, typeFilter, vendorFilter, visibilityFilter]
  )
  const filteredCatalogRows = useMemo<ModelCatalogRow[]>(
    () => [
      ...filteredModels.map((model) => ({
        type: 'configured' as const,
        model,
      })),
      ...filteredCandidateRows.map((candidate) => ({
        type: 'candidate' as const,
        candidate,
      })),
    ],
    [filteredCandidateRows, filteredModels]
  )
  const visibleRows = filteredCatalogRows.slice(0, TABLE_LIMIT)
  const filteredOutCount = Math.max(0, filteredCatalogRows.length - TABLE_LIMIT)
  const hasActiveFilters =
    trimmedKeyword.length > 0 ||
    typeFilter !== 'all' ||
    vendorFilter !== 'all' ||
    visibilityFilter !== 'all' ||
    statusFilter !== 'all'
  const isLoading =
    modelsQuery.isLoading ||
    summaryQuery.isLoading ||
    vendorsQuery.isLoading ||
    missingModelsQuery.isLoading ||
    overviewQuery.isLoading

  const openCreateModel = () => {
    setCurrentRow(null)
    setOpen('create-model')
  }

  const openCreateCandidateModel = (modelName: string) => {
    setCurrentRow({ model_name: modelName } as unknown as Model)
    setOpen('create-model')
  }

  const openSyncWizard = () => {
    setOpen('sync-wizard')
  }

  const openEditModel = (model: Model) => {
    setCurrentRow(model)
    setOpen('update-model')
  }

  return (
    <div className='enterprise-model-center mx-auto flex h-full max-w-[1586px] flex-col overflow-hidden bg-[#f6f8fb] text-slate-950'>
      <EnterprisePageHeader
        title='模型中心'
        description='全局模型目录、能力标签、展示名称、价格、可见范围与上架策略'
        actions={
          <>
            <Button
              variant='outline'
              className='h-8 rounded-md border-slate-200 bg-white px-2.5 text-[12px] font-semibold text-slate-700 shadow-none'
              onClick={openSyncWizard}
            >
              <RefreshCw className='size-3.5' />
              同步模型
            </Button>
            <Button
              className='h-8 rounded-md bg-blue-600 px-2.5 text-[12px] font-semibold text-white shadow-none hover:bg-blue-700'
              onClick={openCreateModel}
            >
              <Plus className='size-3.5' />
              新增模型
            </Button>
          </>
        }
      />

      <div className='flex min-h-0 flex-1 flex-col overflow-auto px-1 pb-2'>
        <section className='grid gap-1.5 md:grid-cols-3 xl:grid-cols-5'>
          <EnterpriseStatCard
            title='全局模型'
            value={formatNumber(modelCatalogCount)}
            helper={`已上架 ${formatNumber(listedCount)} · 待接入 ${formatNumber(
              candidateRows.length
            )}`}
            icon={Boxes}
            tone='blue'
            loading={summaryQuery.isLoading || missingModelsQuery.isLoading}
          />
          <EnterpriseStatCard
            title='企业可信'
            value={formatNumber(trustedCount)}
            helper='官方同步或授权策略'
            icon={ShieldCheck}
            tone='emerald'
            loading={summaryQuery.isLoading}
          />
          <EnterpriseStatCard
            title='平均毛利'
            value={formatPercent(grossMarginRate)}
            helper='来自企业总览账务'
            trend={overviewQuery.isError ? '不可用' : undefined}
            trendTone={overviewQuery.isError ? 'negative' : 'neutral'}
            icon={BadgeDollarSign}
            tone='amber'
            loading={overviewQuery.isLoading}
          />
          <EnterpriseStatCard
            title='待接入模型'
            value={formatNumber(missingCount)}
            helper={`${formatNumber(activeVendorCount)} 个供应商启用`}
            icon={DatabaseZap}
            tone='violet'
            loading={missingModelsQuery.isLoading || vendorsQuery.isLoading}
          />
          <EnterpriseStatCard
            title='异常模型'
            value={formatNumber(abnormalCount)}
            helper={`${formatNumber(pricedCount)} 个已配置价格`}
            icon={AlertTriangle}
            tone={abnormalCount > 0 ? 'rose' : 'slate'}
            loading={summaryQuery.isLoading || systemOptionsQuery.isLoading}
          />
        </section>

        <EnterprisePanel
          className='mt-1.5 flex min-h-[620px] flex-1 flex-col'
          bodyClassName='flex min-h-0 flex-1 flex-col p-0'
          title='模型目录'
          description='按供应来源、能力标签、价格与可见范围核对下游可用资产'
          action={
            <Button
              variant='outline'
              className='h-7 rounded-md border-slate-200 bg-white px-2 text-[11px] font-semibold text-slate-700 shadow-none'
              onClick={() => onSectionChange('deployments')}
            >
              <Layers3 className='size-3.5' />
              部署视图
            </Button>
          }
        >
          <div className='grid gap-2 border-b border-slate-100 bg-white px-3 py-2 min-[1280px]:grid-cols-[126px_126px_150px_126px_minmax(220px,1fr)]'>
            <NativeSelect
              size='sm'
              value={typeFilter}
              onChange={(event) =>
                setTypeFilter(event.target.value as ModelTypeFilter)
              }
              aria-label='模型类型'
              className='w-full'
            >
              <NativeSelectOption value='all'>模型类型 全部</NativeSelectOption>
              <NativeSelectOption value='text'>文本</NativeSelectOption>
              <NativeSelectOption value='vision'>视觉</NativeSelectOption>
              <NativeSelectOption value='reasoning'>推理</NativeSelectOption>
              <NativeSelectOption value='embedding'>向量</NativeSelectOption>
              <NativeSelectOption value='audio'>语音</NativeSelectOption>
            </NativeSelect>
            <NativeSelect
              size='sm'
              value={vendorFilter}
              onChange={(event) => setVendorFilter(event.target.value)}
              aria-label='供应商'
              className='w-full'
            >
              <NativeSelectOption value='all'>供应商 全部</NativeSelectOption>
              {vendors.map((vendor) => (
                <NativeSelectOption key={vendor.id} value={String(vendor.id)}>
                  {vendor.name}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            <NativeSelect
              size='sm'
              value={visibilityFilter}
              onChange={(event) =>
                setVisibilityFilter(event.target.value as VisibilityFilter)
              }
              aria-label='可见范围'
              className='w-full'
            >
              <NativeSelectOption value='all'>可见范围 全部</NativeSelectOption>
              <NativeSelectOption value='global'>全局可见</NativeSelectOption>
              <NativeSelectOption value='restricted'>
                企业授权
              </NativeSelectOption>
            </NativeSelect>
            <NativeSelect
              size='sm'
              value={statusFilter}
              onChange={(event) => setStatusFilter(event.target.value)}
              aria-label='状态'
              className='w-full'
            >
              <NativeSelectOption value='all'>状态 全部</NativeSelectOption>
              <NativeSelectOption value='1'>已上架</NativeSelectOption>
              <NativeSelectOption value='0'>已停用</NativeSelectOption>
            </NativeSelect>
            <div className='relative min-w-0'>
              <Search className='pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-slate-400' />
              <Input
                value={keyword}
                onChange={(event) => setKeyword(event.target.value)}
                placeholder='搜索模型、标签、端点'
                className='h-7 rounded-md border-slate-200 bg-slate-50 pl-8 text-[12px] shadow-none placeholder:text-slate-400'
              />
            </div>
          </div>

          {modelsQuery.isError || vendorsQuery.isError ? (
            <div className='mx-3 mt-2 flex items-center gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-[11px] text-amber-800'>
              <AlertTriangle className='size-3.5 shrink-0' />
              模型或供应商接口暂时不可用，请确认后端服务状态后重试。
            </div>
          ) : null}

          <div className='min-h-0 flex-1 overflow-auto'>
            <Table>
              <TableHeader className='sticky top-0 z-10 bg-slate-50/95 backdrop-blur'>
                <TableRow className='border-slate-100'>
                  <TableHead className='h-9 w-[28%] px-3 text-[11px] font-semibold text-slate-500'>
                    模型
                  </TableHead>
                  <TableHead className='h-9 text-[11px] font-semibold text-slate-500'>
                    能力
                  </TableHead>
                  <TableHead className='h-9 text-[11px] font-semibold text-slate-500'>
                    供应来源
                  </TableHead>
                  <TableHead className='h-9 text-[11px] font-semibold text-slate-500'>
                    价格
                  </TableHead>
                  <TableHead className='h-9 text-[11px] font-semibold text-slate-500'>
                    范围样本
                  </TableHead>
                  <TableHead className='h-9 text-right text-[11px] font-semibold text-slate-500'>
                    状态
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {visibleRows.length === 0 ? (
                  <TableEmptyState
                    loading={isLoading}
                    onCreate={openCreateModel}
                    onSync={openSyncWizard}
                  />
                ) : (
                  visibleRows.map((row) => {
                    if (row.type === 'candidate') {
                      const candidate = row.candidate
                      const candidatePricing = getModelPricingLabel(
                        candidate.name,
                        pricing
                      )

                      return (
                        <TableRow
                          key={`candidate-${candidate.key}`}
                          className='group border-slate-100 bg-amber-50/20 hover:bg-amber-50/40'
                        >
                          <TableCell className='px-3 py-2.5 align-top'>
                            <div className='flex min-w-0 items-start gap-2.5'>
                              <span className='flex size-8 shrink-0 items-center justify-center rounded-md bg-amber-50 text-amber-600 ring-1 ring-amber-100'>
                                <DatabaseZap className='size-4' />
                              </span>
                              <div className='min-w-0'>
                                <div className='flex min-w-0 items-center gap-1.5'>
                                  <p className='truncate text-[13px] font-semibold text-slate-950'>
                                    {candidate.name}
                                  </p>
                                  <Badge className='h-5 rounded bg-amber-50 px-1.5 text-[10px] text-amber-700 ring-1 ring-amber-100'>
                                    待接入
                                  </Badge>
                                </div>
                                <p className='mt-0.5 line-clamp-1 text-[11px] text-slate-500'>
                                  {getCandidateSourceLabel(candidate)}
                                </p>
                              </div>
                            </div>
                          </TableCell>
                          <TableCell className='py-2.5 align-top'>
                            <div className='flex max-w-[260px] flex-wrap gap-1'>
                              {getCandidateCapabilities(candidate.name).map(
                                (capability) => (
                                  <Badge
                                    key={capability}
                                    variant='outline'
                                    className='h-5 rounded border-slate-200 bg-white px-1.5 text-[10px] text-slate-600'
                                  >
                                    {capability}
                                  </Badge>
                                )
                              )}
                            </div>
                          </TableCell>
                          <TableCell className='py-2.5 align-top'>
                            <p className='text-[12px] font-semibold text-slate-800'>
                              {getCandidateChannelLabel(candidate)}
                            </p>
                            <p className='mt-0.5 text-[11px] text-slate-500'>
                              {candidate.channels.length > 0
                                ? `${formatNumber(candidate.channels.length)} 条渠道声明`
                                : '尚未绑定供应商'}
                            </p>
                          </TableCell>
                          <TableCell className='py-2.5 align-top'>
                            <p
                              className={cn(
                                'text-[12px] font-semibold tabular-nums',
                                candidatePricing === '未配置'
                                  ? 'text-amber-600'
                                  : 'text-slate-900'
                              )}
                            >
                              {candidatePricing}
                            </p>
                            <p className='mt-0.5 text-[11px] text-slate-500'>
                              待补资产配置
                            </p>
                          </TableCell>
                          <TableCell className='py-2.5 align-top'>
                            <p className='text-[12px] font-semibold text-slate-800'>
                              {candidate.requests > 0
                                ? `${formatNumber(candidate.requests)} 请求`
                                : '暂无样本'}
                            </p>
                            <p className='mt-0.5 text-[11px] text-slate-500'>
                              {candidate.tokens > 0
                                ? `${formatNumber(candidate.tokens)} tokens`
                                : '等待路由接入'}
                            </p>
                          </TableCell>
                          <TableCell className='py-2.5 pr-3 align-top'>
                            <div className='flex justify-end'>
                              <div className='flex items-center gap-1.5'>
                                <Badge
                                  variant='outline'
                                  className='h-5 rounded border-amber-200 bg-amber-50 px-1.5 text-[10px] text-amber-700'
                                >
                                  <span className='size-1.5 rounded-full bg-amber-500' />
                                  待配置
                                </Badge>
                                <Button
                                  variant='ghost'
                                  className='h-7 rounded-md px-2 text-[11px] font-semibold text-blue-600 opacity-80 group-hover:opacity-100 hover:bg-white hover:text-blue-700'
                                  onClick={() =>
                                    openCreateCandidateModel(candidate.name)
                                  }
                                  aria-label={`配置 ${candidate.name}`}
                                >
                                  <Plus className='size-3.5' />
                                  配置
                                </Button>
                              </div>
                            </div>
                          </TableCell>
                        </TableRow>
                      )
                    }

                    const model = row.model
                    const status = modelStatus(model)
                    const risks = getModelRiskCount(model)
                    return (
                      <TableRow
                        key={model.id}
                        className='group border-slate-100 hover:bg-blue-50/35'
                      >
                        <TableCell className='px-3 py-2.5 align-top'>
                          <div className='flex min-w-0 items-start gap-2.5'>
                            <span className='flex size-8 shrink-0 items-center justify-center rounded-md bg-blue-50 text-blue-600 ring-1 ring-blue-100'>
                              {model.icon ? (
                                <img
                                  src={model.icon}
                                  alt=''
                                  className='size-5 rounded object-cover'
                                />
                              ) : (
                                <BrainCircuit className='size-4' />
                              )}
                            </span>
                            <div className='min-w-0'>
                              <div className='flex min-w-0 items-center gap-1.5'>
                                <p className='truncate text-[13px] font-semibold text-slate-950'>
                                  {model.model_name}
                                </p>
                                {model.sync_official === 1 && (
                                  <Badge className='h-5 rounded bg-blue-50 px-1.5 text-[10px] text-blue-700'>
                                    官方
                                  </Badge>
                                )}
                              </div>
                              <p className='mt-0.5 line-clamp-1 text-[11px] text-slate-500'>
                                {model.description ||
                                  buildEndpointSummary(model)}
                              </p>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell className='py-2.5 align-top'>
                          <div className='flex max-w-[260px] flex-wrap gap-1'>
                            {getModelCapabilities(model).map((capability) => (
                              <Badge
                                key={capability}
                                variant='outline'
                                className='h-5 rounded border-slate-200 bg-white px-1.5 text-[10px] text-slate-600'
                              >
                                {capability}
                              </Badge>
                            ))}
                          </div>
                        </TableCell>
                        <TableCell className='py-2.5 align-top'>
                          <p className='text-[12px] font-semibold text-slate-800'>
                            {getVendorName(model, vendorMap)}
                          </p>
                          <p className='mt-0.5 text-[11px] text-slate-500'>
                            {buildEndpointSummary(model)}
                          </p>
                        </TableCell>
                        <TableCell className='py-2.5 align-top'>
                          <p
                            className={cn(
                              'text-[12px] font-semibold tabular-nums',
                              getPricingLabel(model, pricing) === '未配置'
                                ? 'text-amber-600'
                                : 'text-slate-900'
                            )}
                          >
                            {getPricingLabel(model, pricing)}
                          </p>
                          <p className='mt-0.5 text-[11px] text-slate-500'>
                            系统定价
                          </p>
                        </TableCell>
                        <TableCell className='py-2.5 align-top'>
                          <p className='text-[12px] font-semibold text-slate-800'>
                            暂无样本
                          </p>
                          <p className='mt-0.5 text-[11px] text-slate-500'>
                            {getVisibleScope(model)}
                          </p>
                        </TableCell>
                        <TableCell className='py-2.5 pr-3 align-top'>
                          <div className='flex justify-end'>
                            <div className='flex items-center gap-1.5'>
                              {risks > 0 && (
                                <Badge
                                  variant='outline'
                                  className='h-5 rounded border-amber-200 bg-amber-50 px-1.5 text-[10px] text-amber-700'
                                >
                                  {risks} 风险
                                </Badge>
                              )}
                              <Badge
                                variant='outline'
                                className={cn(
                                  'h-5 rounded px-1.5 text-[10px]',
                                  status.className
                                )}
                              >
                                <span
                                  className={cn(
                                    'size-1.5 rounded-full',
                                    status.dotClassName
                                  )}
                                />
                                {status.label}
                              </Badge>
                              <Button
                                variant='ghost'
                                size='icon'
                                className='size-7 rounded-md text-slate-500 opacity-70 group-hover:opacity-100 hover:bg-white hover:text-blue-600'
                                onClick={() => openEditModel(model)}
                                aria-label={`编辑 ${model.model_name}`}
                              >
                                <Pencil className='size-3.5' />
                              </Button>
                            </div>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </div>

          <div className='flex min-h-9 items-center justify-between gap-2 border-t border-slate-100 bg-slate-50/70 px-3 py-2 text-[11px] text-slate-500'>
            <div className='flex min-w-0 items-center gap-2'>
              <SlidersHorizontal className='size-3.5 shrink-0' />
              <span className='truncate'>
                {hasActiveFilters
                  ? `当前筛选 ${formatNumber(filteredCatalogRows.length)} 个模型`
                  : `全量目录 ${formatNumber(modelCatalogCount)} 个模型`}
                {filteredOutCount > 0 ? `，表格展示前 ${TABLE_LIMIT} 个` : ''}
              </span>
            </div>
            <div className='flex shrink-0 items-center gap-2'>
              <span className='hidden items-center gap-1 sm:flex'>
                <Clock3 className='size-3' />
                30 秒缓存
              </span>
              <Button
                variant='ghost'
                className='h-6 rounded-md px-1.5 text-[11px] font-semibold text-blue-600 hover:bg-blue-50 hover:text-blue-700'
                onClick={() => onSectionChange('deployments')}
              >
                查看部署
                <ArrowUpRight className='size-3' />
              </Button>
            </div>
          </div>
        </EnterprisePanel>
      </div>
    </div>
  )
}
