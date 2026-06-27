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
  AlarmClock,
  Clock3,
  CreditCard,
  Edit,
  KeyRound,
  MoreHorizontal,
  Plus,
  Power,
  PowerOff,
  Search,
  ShieldAlert,
  Trash2,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import dayjs from '@/lib/dayjs'
import { formatNumber, formatQuota, formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import { getApiKeys, updateApiKeyStatus } from './api'
import { ApiKeyCell, ModelLimitsCell } from './components/api-keys-cells'
import { useApiKeys } from './components/api-keys-provider'
import {
  API_KEY_STATUS,
  API_KEY_STATUSES,
  ERROR_MESSAGES,
  SUCCESS_MESSAGES,
} from './constants'
import type { ApiKey } from './types'

const PERSONAL_KEYS_PAGE_SIZE = 100
const PERSONAL_KEYS_SKELETON_ROWS = [
  'personal-key-skeleton-name',
  'personal-key-skeleton-key',
  'personal-key-skeleton-quota',
  'personal-key-skeleton-status',
]

function formatRelativeTime(timestamp: number) {
  if (!timestamp || timestamp <= 0) return '-'
  const diffMinutes = dayjs().diff(dayjs(timestamp * 1000), 'minute')
  if (diffMinutes < 1) return '刚刚'
  if (diffMinutes < 60) return `${diffMinutes} 分钟前`
  const diffHours = Math.floor(diffMinutes / 60)
  if (diffHours < 24) return `${diffHours} 小时前`
  return `${Math.floor(diffHours / 24)} 天前`
}

function isExpiringSoon(apiKey: ApiKey) {
  if (apiKey.expired_time <= 0) return false
  const expiresAt = dayjs(apiKey.expired_time * 1000)
  return expiresAt.isAfter(dayjs()) && expiresAt.diff(dayjs(), 'day') <= 7
}

function quotaSummary(apiKeys: ApiKey[]) {
  const limitedKeys = apiKeys.filter((apiKey) => !apiKey.unlimited_quota)
  const hasUnlimited = apiKeys.some((apiKey) => apiKey.unlimited_quota)
  const remaining = limitedKeys.reduce(
    (sum, apiKey) => sum + apiKey.remain_quota,
    0
  )

  return {
    value: hasUnlimited ? '无限制' : formatQuota(remaining),
    helper: hasUnlimited
      ? `${limitedKeys.length} 个限额 Key`
      : `${limitedKeys.length} 个限额 Key 可用`,
  }
}

function PersonalApiKeyStatCard({
  title,
  value,
  helper,
  icon: Icon,
  tone,
  loading,
}: {
  title: string
  value: string
  helper: string
  icon: typeof KeyRound
  tone: 'blue' | 'green' | 'amber' | 'violet'
  loading?: boolean
}) {
  const toneClass = {
    blue: 'bg-blue-50 text-blue-600 ring-blue-100',
    green: 'bg-emerald-50 text-emerald-600 ring-emerald-100',
    amber: 'bg-orange-50 text-orange-600 ring-orange-100',
    violet: 'bg-violet-50 text-violet-600 ring-violet-100',
  }[tone]

  return (
    <article className='min-h-[104px] rounded-md border border-slate-200 bg-white p-4 shadow-[0_1px_2px_rgb(15_23_42/0.035)]'>
      <div className='flex items-center gap-3'>
        <span
          className={cn(
            'flex size-10 shrink-0 items-center justify-center rounded-md ring-1',
            toneClass
          )}
        >
          <Icon className='size-5' />
        </span>
        <div className='min-w-0'>
          <p className='truncate text-[13px] font-semibold text-slate-600'>
            {title}
          </p>
          {loading ? (
            <Skeleton className='mt-2 h-7 w-24 rounded-md' />
          ) : (
            <p className='mt-1 truncate text-[25px] leading-7 font-semibold text-slate-950'>
              {value}
            </p>
          )}
          <p className='mt-1 truncate text-[12px] font-medium text-emerald-600'>
            {helper}
          </p>
        </div>
      </div>
    </article>
  )
}

function PersonalKeyActions({ apiKey }: { apiKey: ApiKey }) {
  const { t } = useTranslation()
  const { setOpen, setCurrentRow, triggerRefresh } = useApiKeys()
  const [toggling, setToggling] = useState(false)
  const isEnabled = apiKey.status === API_KEY_STATUS.ENABLED

  const handleToggleStatus = async () => {
    const nextStatus = isEnabled
      ? API_KEY_STATUS.DISABLED
      : API_KEY_STATUS.ENABLED

    setToggling(true)
    try {
      const result = await updateApiKeyStatus(apiKey.id, nextStatus)
      if (result.success) {
        toast.success(
          t(
            isEnabled
              ? SUCCESS_MESSAGES.API_KEY_DISABLED
              : SUCCESS_MESSAGES.API_KEY_ENABLED
          )
        )
        triggerRefresh()
        return
      }
      toast.error(result.message || t(ERROR_MESSAGES.STATUS_UPDATE_FAILED))
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setToggling(false)
    }
  }

  return (
    <div className='flex items-center justify-end gap-1'>
      <Button
        variant='ghost'
        size='icon-sm'
        className={cn(
          'size-7 rounded-md',
          isEnabled
            ? 'text-rose-500 hover:text-rose-600'
            : 'text-emerald-600 hover:text-emerald-700'
        )}
        disabled={toggling}
        aria-label={isEnabled ? t('Disable') : t('Enable')}
        onClick={() => void handleToggleStatus()}
      >
        {isEnabled ? (
          <PowerOff className='size-3.5' />
        ) : (
          <Power className='size-3.5' />
        )}
      </Button>
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger
          render={
            <Button
              variant='ghost'
              size='icon-sm'
              className='size-7 rounded-md text-slate-500'
            />
          }
        >
          <MoreHorizontal className='size-3.5' />
          <span className='sr-only'>{t('Open menu')}</span>
        </DropdownMenuTrigger>
        <DropdownMenuContent align='end' className='w-36'>
          <DropdownMenuItem
            onClick={() => {
              setCurrentRow(apiKey)
              setOpen('update')
            }}
          >
            <Edit className='size-3.5' />
            {t('Edit')}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            className='text-rose-600 focus:text-rose-600'
            onClick={() => {
              setCurrentRow(apiKey)
              setOpen('delete')
            }}
          >
            <Trash2 className='size-3.5' />
            {t('Delete')}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}

function PersonalApiKeysTable({
  apiKeys,
  loading,
}: {
  apiKeys: ApiKey[]
  loading: boolean
}) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <div className='space-y-2 p-4'>
        {PERSONAL_KEYS_SKELETON_ROWS.map((row) => (
          <Skeleton key={row} className='h-9 w-full rounded-md' />
        ))}
      </div>
    )
  }

  if (apiKeys.length === 0) {
    return (
      <div className='flex min-h-44 flex-col items-center justify-center gap-2 px-4 text-center'>
        <KeyRound className='size-8 text-slate-300' />
        <p className='text-[14px] font-semibold text-slate-800'>
          {t('No API Keys Found')}
        </p>
        <p className='text-[12px] text-slate-500'>
          {t(
            'No API keys available. Create your first API key to get started.'
          )}
        </p>
      </div>
    )
  }

  return (
    <div className='overflow-x-auto px-4 pb-3'>
      <table className='w-full min-w-[980px] border-separate border-spacing-0 text-left'>
        <thead>
          <tr className='text-[12px] font-semibold text-slate-500'>
            <th className='h-10 border-b border-slate-200 bg-slate-50/80 px-3'>
              名称
            </th>
            <th className='h-10 border-b border-slate-200 bg-slate-50/80 px-3'>
              Key ID
            </th>
            <th className='h-10 border-b border-slate-200 bg-slate-50/80 px-3'>
              授权模型
            </th>
            <th className='h-10 border-b border-slate-200 bg-slate-50/80 px-3'>
              额度
            </th>
            <th className='h-10 border-b border-slate-200 bg-slate-50/80 px-3'>
              创建时间
            </th>
            <th className='h-10 border-b border-slate-200 bg-slate-50/80 px-3'>
              最近使用
            </th>
            <th className='h-10 border-b border-slate-200 bg-slate-50/80 px-3'>
              状态
            </th>
            <th className='h-10 border-b border-slate-200 bg-slate-50/80 px-3 text-right'>
              操作
            </th>
          </tr>
        </thead>
        <tbody>
          {apiKeys.map((apiKey) => {
            const statusConfig = API_KEY_STATUSES[apiKey.status]
            const totalQuota = apiKey.remain_quota + apiKey.used_quota
            const quotaText = apiKey.unlimited_quota
              ? '无限制'
              : `${formatQuota(apiKey.remain_quota)} / ${formatQuota(totalQuota)}`

            return (
              <tr key={apiKey.id} className='group text-[13px] text-slate-700'>
                <td className='h-12 border-b border-slate-100 px-3'>
                  <div className='font-semibold text-slate-900'>
                    {apiKey.name || '-'}
                  </div>
                  <div className='mt-0.5 text-[11px] text-slate-500'>
                    {apiKey.group || '默认分组'}
                  </div>
                </td>
                <td className='h-12 border-b border-slate-100 px-3'>
                  <ApiKeyCell apiKey={apiKey} />
                </td>
                <td className='h-12 border-b border-slate-100 px-3'>
                  <ModelLimitsCell apiKey={apiKey} />
                </td>
                <td className='h-12 border-b border-slate-100 px-3 font-medium text-slate-800'>
                  {quotaText}
                </td>
                <td className='h-12 border-b border-slate-100 px-3 text-slate-600'>
                  {formatTimestampToDate(apiKey.created_time).slice(0, 10)}
                </td>
                <td className='h-12 border-b border-slate-100 px-3 text-slate-600'>
                  {formatRelativeTime(apiKey.accessed_time)}
                </td>
                <td className='h-12 border-b border-slate-100 px-3'>
                  {statusConfig ? (
                    <StatusBadge
                      label={t(statusConfig.label)}
                      variant={statusConfig.variant}
                      copyable={false}
                    />
                  ) : (
                    '-'
                  )}
                </td>
                <td className='h-12 border-b border-slate-100 px-3'>
                  <PersonalKeyActions apiKey={apiKey} />
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

export function PersonalApiKeys() {
  const { t } = useTranslation()
  const { setOpen, refreshTrigger } = useApiKeys()
  const [search, setSearch] = useState('')

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['personal-api-keys', refreshTrigger],
    queryFn: async () => {
      const result = await getApiKeys({ p: 1, size: PERSONAL_KEYS_PAGE_SIZE })
      if (!result.success) {
        toast.error(result.message || t(ERROR_MESSAGES.LOAD_FAILED))
        return { items: [], total: 0 }
      }
      return {
        items: result.data?.items ?? [],
        total: result.data?.total ?? 0,
      }
    },
    placeholderData: (previousData) => previousData,
  })

  const apiKeys = useMemo(() => data?.items ?? [], [data?.items])
  const filteredKeys = useMemo(() => {
    const keyword = search.trim().toLowerCase()
    if (!keyword) return apiKeys
    return apiKeys.filter((apiKey) => {
      return [apiKey.name, apiKey.key, apiKey.group, apiKey.model_limits]
        .filter((value): value is string => typeof value === 'string')
        .some((value) => value.toLowerCase().includes(keyword))
    })
  }, [apiKeys, search])

  const activeKeys = apiKeys.filter(
    (apiKey) => apiKey.status === API_KEY_STATUS.ENABLED
  ).length
  const quota = quotaSummary(apiKeys)
  const expiringSoon = apiKeys.filter(isExpiringSoon).length
  const abnormalKeys = apiKeys.filter(
    (apiKey) => apiKey.status !== API_KEY_STATUS.ENABLED
  ).length

  return (
    <div className='personal-api-keys mx-auto w-full max-w-[1586px] space-y-4 bg-[#f6f8fb] px-5 pt-8 pb-3 text-slate-950 sm:px-8'>
      <div className='flex flex-col gap-3 md:flex-row md:items-start md:justify-between'>
        <div className='min-w-0'>
          <h1 className='text-[24px] leading-8 font-semibold tracking-normal text-slate-950'>
            我的 API Keys
          </h1>
          <p className='mt-1 text-[13px] text-slate-500'>
            C端个人用户管理自己的密钥、额度、过期时间和调用限制
          </p>
        </div>
        <Button
          className='h-9 rounded-md bg-blue-600 px-4 text-[13px] font-semibold text-white shadow-sm hover:bg-blue-700'
          onClick={() => setOpen('create')}
        >
          <Plus className='size-4' />
          创建 Key
        </Button>
      </div>

      <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-4'>
        <PersonalApiKeyStatCard
          title='活跃 Key'
          value={formatNumber(activeKeys)}
          helper={`共 ${formatNumber(apiKeys.length)} 个 Key`}
          icon={KeyRound}
          tone='blue'
          loading={isLoading}
        />
        <PersonalApiKeyStatCard
          title='可用额度'
          value={quota.value}
          helper={quota.helper}
          icon={CreditCard}
          tone='green'
          loading={isLoading}
        />
        <PersonalApiKeyStatCard
          title='即将过期'
          value={formatNumber(expiringSoon)}
          helper='7天内'
          icon={AlarmClock}
          tone='amber'
          loading={isLoading}
        />
        <PersonalApiKeyStatCard
          title='需处理 Key'
          value={formatNumber(abnormalKeys)}
          helper={abnormalKeys > 0 ? '请检查状态' : '安全'}
          icon={ShieldAlert}
          tone='violet'
          loading={isLoading}
        />
      </div>

      <section className='overflow-hidden rounded-md border border-slate-200 bg-white shadow-[0_1px_2px_rgb(15_23_42/0.035)]'>
        <div className='flex flex-col gap-2 border-b border-slate-100 px-4 py-3 md:flex-row md:items-center md:justify-between'>
          <div className='min-w-0'>
            <h2 className='text-[15px] font-semibold text-slate-950'>
              密钥列表
            </h2>
            <p className='mt-0.5 text-[12px] text-slate-500'>
              当前展示 {formatNumber(filteredKeys.length)} 条
              {isFetching && !isLoading ? '，正在同步最新数据' : ''}
            </p>
          </div>
          <div className='relative w-full md:w-[320px]'>
            <Search className='pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-slate-400' />
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder='搜索名称、Key、分组、模型'
              className='h-8 rounded-md border-slate-200 bg-white pl-9 text-[12px]'
            />
          </div>
        </div>
        <PersonalApiKeysTable apiKeys={filteredKeys} loading={isLoading} />
        <div className='flex items-center justify-between border-t border-slate-100 px-4 py-2 text-[12px] text-slate-500'>
          <span>
            总计 {formatNumber(data?.total ?? apiKeys.length)} 条，当前取前{' '}
            {PERSONAL_KEYS_PAGE_SIZE} 条
          </span>
          <span className='inline-flex items-center gap-1'>
            <Clock3 className='size-3.5' />
            数据来自 /api/token
          </span>
        </div>
      </section>
    </div>
  )
}
