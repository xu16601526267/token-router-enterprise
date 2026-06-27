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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  CalendarClock,
  Check,
  Copy,
  Download,
  Edit3,
  KeyRound,
  MoreHorizontal,
  Plus,
  RefreshCw,
  RotateCcw,
  Search,
  ShieldCheck,
  Trash2,
  Users,
  type LucideIcon,
} from 'lucide-react'
import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { EnterprisePageHeader, EnterprisePanel } from '@/components/enterprise'
import { SectionPageLayout } from '@/components/layout'
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
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import dayjs from '@/lib/dayjs'
import { formatLogQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

import {
  createEnterpriseApiKey,
  deleteEnterpriseApiKey,
  exportEnterpriseApiKeys,
  getEnterpriseApiKeys,
  getEnterpriseApiKeyUsers,
  rotateEnterpriseApiKey,
  updateEnterpriseApiKey,
} from './enterprise-api'
import type {
  EnterpriseApiKeyInput,
  EnterpriseApiKeyItem,
  EnterpriseApiKeySecret,
  EnterpriseApiKeyUser,
} from './enterprise-types'

const TOKEN_STATUS_ENABLED = 1
const TOKEN_STATUS_DISABLED = 2
const TOKEN_STATUS_EXPIRED = 3
const EMPTY_API_KEY_ITEMS: EnterpriseApiKeyItem[] = []
const EMPTY_API_KEY_USERS: EnterpriseApiKeyUser[] = []
const TOKEN_STATUS_EXHAUSTED = 4

type ApiKeyFormState = {
  userId: string
  name: string
  status: string
  expiredAt: string
  remainQuota: string
  unlimitedQuota: boolean
  modelLimitsEnabled: boolean
  modelLimits: string
  allowIps: string
  group: string
  crossGroupRetry: boolean
  rateLimit: string
}

const emptyForm: ApiKeyFormState = {
  userId: '',
  name: '',
  status: String(TOKEN_STATUS_ENABLED),
  expiredAt: '',
  remainQuota: '1000000',
  unlimitedQuota: true,
  modelLimitsEnabled: false,
  modelLimits: '',
  allowIps: '',
  group: '',
  crossGroupRetry: false,
  rateLimit: '',
}

function tokenStatus(status: number): {
  label: string
  className: string
} {
  if (status === TOKEN_STATUS_ENABLED) {
    return {
      label: '活跃',
      className: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-300',
    }
  }
  if (status === TOKEN_STATUS_DISABLED) {
    return {
      label: '已禁用',
      className: 'bg-muted text-muted-foreground',
    }
  }
  if (status === TOKEN_STATUS_EXPIRED) {
    return {
      label: '已过期',
      className: 'bg-amber-500/10 text-amber-600 dark:text-amber-300',
    }
  }
  return {
    label: '额度耗尽',
    className: 'bg-rose-500/10 text-rose-600 dark:text-rose-300',
  }
}

function modelChips(item: EnterpriseApiKeyItem) {
  if (!item.model_limits_enabled) {
    return (
      <Badge variant='outline' className='h-5 rounded px-2 text-[10px]'>
        全部模型
      </Badge>
    )
  }
  const models = item.model_limits
    .split(',')
    .map((model) => model.trim())
    .filter(Boolean)
  return (
    <div className='flex max-w-56 flex-wrap gap-1'>
      {models.slice(0, 2).map((model) => (
        <Badge
          key={model}
          variant='secondary'
          className='h-5 max-w-32 rounded px-2 text-[10px]'
        >
          {model}
        </Badge>
      ))}
      {models.length > 2 && (
        <Badge variant='outline' className='h-5 rounded px-2 text-[10px]'>
          +{models.length - 2}
        </Badge>
      )}
    </div>
  )
}

function formatCount(value: number): string {
  return new Intl.NumberFormat('zh-CN', { notation: 'compact' }).format(value)
}

function formatCompactQuota(value: number): string {
  const formatted = formatLogQuota(value)
  if (formatted.length <= 8) {
    return formatted
  }
  const numeric = Number(formatted.replace(/^\$/, ''))
  if (Number.isFinite(numeric) && Math.abs(numeric) < 1) {
    const compact = numeric.toFixed(3).replace(/0+$/, '').replace(/\.$/, '')
    return `$${compact}`
  }
  return formatted
}

function dateInputToTimestamp(value: string, boundary: 'start' | 'end') {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return undefined
  }
  const parsed = dayjs(value)
  if (!parsed.isValid()) {
    return undefined
  }
  return boundary === 'start'
    ? parsed.startOf('day').unix()
    : parsed.endOf('day').unix()
}

function formatRelativeTimestamp(timestamp: number): string {
  if (timestamp <= 0) return '从未使用'
  const diffSeconds = Math.max(0, dayjs().unix() - timestamp)
  if (diffSeconds < 60) return '刚刚'
  if (diffSeconds < 3600) return `${Math.floor(diffSeconds / 60)} 分钟前`
  if (diffSeconds < 86400) return `${Math.floor(diffSeconds / 3600)} 小时前`
  if (diffSeconds < 30 * 86400) {
    return `${Math.floor(diffSeconds / 86400)} 天前`
  }
  return dayjs.unix(timestamp).format('YYYY-MM-DD')
}

function keyOwner(item: EnterpriseApiKeyItem) {
  return item.display_name || item.username || `用户 #${item.user_id}`
}

function environmentLabel(value: string) {
  const normalized = value.toLowerCase()
  if (
    normalized.includes('prod') ||
    normalized.includes('production') ||
    value.includes('生产')
  ) {
    return '生产'
  }
  if (
    normalized.includes('stag') ||
    normalized.includes('preview') ||
    value.includes('预发')
  ) {
    return '预发'
  }
  if (
    normalized.includes('dev') ||
    normalized.includes('test') ||
    normalized.includes('sandbox') ||
    value.includes('开发')
  ) {
    return '开发'
  }
  return value || '默认'
}

function environmentBadgeClass(label: string) {
  if (label === '生产') {
    return 'border-emerald-200 bg-emerald-50 text-emerald-600'
  }
  if (label === '预发') {
    return 'border-blue-200 bg-blue-50 text-blue-600'
  }
  if (label === '开发') {
    return 'border-violet-200 bg-violet-50 text-violet-600'
  }
  return 'border-slate-200 bg-slate-50 text-slate-600'
}

function MetricTile(props: {
  title: string
  value: string
  helper: string
  icon: LucideIcon
  tone: 'blue' | 'violet' | 'amber' | 'emerald' | 'rose'
  loading?: boolean
}) {
  const toneClass = {
    blue: 'bg-blue-50 text-blue-600 ring-blue-100',
    violet: 'bg-violet-50 text-violet-600 ring-violet-100',
    amber: 'bg-amber-50 text-amber-600 ring-amber-100',
    emerald: 'bg-emerald-50 text-emerald-600 ring-emerald-100',
    rose: 'bg-rose-50 text-rose-600 ring-rose-100',
  }[props.tone]
  const Icon = props.icon
  return (
    <div className='rounded-md border border-slate-200 bg-white px-3 py-2.5 shadow-[0_1px_2px_rgb(15_23_42/0.035)]'>
      <div className='flex items-start gap-2.5'>
        <span
          className={cn(
            'flex size-8 shrink-0 items-center justify-center rounded-md ring-1',
            toneClass
          )}
        >
          <Icon className='size-4' aria-hidden='true' />
        </span>
        <div className='min-w-0 flex-1'>
          <p className='truncate text-[12px] font-medium text-slate-600'>
            {props.title}
          </p>
          <p className='mt-1 text-[22px] leading-6 font-semibold text-slate-950'>
            {props.loading ? '...' : props.value}
          </p>
          <p className='mt-1 truncate text-[11px] text-slate-500'>
            {props.helper}
          </p>
        </div>
      </div>
    </div>
  )
}

function FilterField(props: {
  label: string
  children: ReactNode
  className?: string
}) {
  return (
    <label
      className={cn(
        'flex h-8 min-w-0 items-center gap-2 rounded-md border border-slate-200 bg-white px-2 text-xs',
        props.className
      )}
    >
      <span className='shrink-0 text-[11px] font-medium text-slate-500'>
        {props.label}
      </span>
      <div className='min-w-0 flex-1'>{props.children}</div>
    </label>
  )
}

function DetailField(props: {
  label: string
  value: string
  copyValue?: string
}) {
  return (
    <div className='space-y-1'>
      <p className='text-[11px] font-medium text-slate-500'>{props.label}</p>
      <div className='flex min-h-8 items-center gap-2 rounded-md border border-slate-200 bg-slate-50/70 px-2'>
        <code className='min-w-0 flex-1 truncate text-[11px] text-slate-800'>
          {props.value}
        </code>
        {props.copyValue != null && (
          <Button
            size='icon-xs'
            variant='ghost'
            aria-label={`复制${props.label}`}
            onClick={() => {
              void navigator.clipboard.writeText(props.copyValue ?? props.value)
              toast.success('已复制')
            }}
          >
            <Copy className='size-3' />
          </Button>
        )}
      </div>
    </div>
  )
}

function CompactSummaryItem(props: {
  label: string
  value: string
  helper?: string
  tone?: 'blue' | 'emerald' | 'amber' | 'rose' | 'slate'
}) {
  const toneClass = {
    blue: 'bg-blue-50 text-blue-700 ring-blue-100',
    emerald: 'bg-emerald-50 text-emerald-700 ring-emerald-100',
    amber: 'bg-amber-50 text-amber-700 ring-amber-100',
    rose: 'bg-rose-50 text-rose-700 ring-rose-100',
    slate: 'bg-slate-50 text-slate-700 ring-slate-100',
  }[props.tone ?? 'slate']
  return (
    <div className='min-w-0 rounded-md border border-slate-100 bg-slate-50/45 px-2.5 py-2'>
      <p className='text-[11px] leading-4 font-medium text-slate-500'>
        {props.label}
      </p>
      <p
        className={cn(
          'mt-1 block max-w-full truncate rounded px-1 text-[13px] leading-5 font-semibold tabular-nums ring-1',
          toneClass
        )}
      >
        {props.value}
      </p>
      {props.helper != null && (
        <p className='mt-1 truncate text-[11px] leading-4 text-slate-500'>
          {props.helper}
        </p>
      )}
    </div>
  )
}

function ThinProgress(props: { value: number; tone?: 'blue' | 'emerald' }) {
  return (
    <div className='h-1.5 overflow-hidden rounded-full bg-slate-100'>
      <div
        className={cn(
          'h-full rounded-full',
          props.tone === 'emerald' ? 'bg-emerald-500' : 'bg-blue-500'
        )}
        style={{ width: `${Math.min(100, Math.max(0, props.value))}%` }}
      />
    </div>
  )
}

function toForm(item: EnterpriseApiKeyItem): ApiKeyFormState {
  return {
    userId: String(item.user_id),
    name: item.name,
    status: String(item.status === TOKEN_STATUS_ENABLED ? 1 : 2),
    expiredAt:
      item.expired_time > 0
        ? dayjs.unix(item.expired_time).format('YYYY-MM-DDTHH:mm')
        : '',
    remainQuota: String(item.remain_quota),
    unlimitedQuota: item.unlimited_quota,
    modelLimitsEnabled: item.model_limits_enabled,
    modelLimits: item.model_limits,
    allowIps: item.allow_ips ?? '',
    group: item.group,
    crossGroupRetry: item.cross_group_retry,
    rateLimit: item.rate_limit ?? '',
  }
}

function toInput(form: ApiKeyFormState): EnterpriseApiKeyInput {
  return {
    user_id: Number(form.userId),
    name: form.name.trim(),
    status: Number(form.status),
    expired_time: form.expiredAt
      ? Math.floor(new Date(form.expiredAt).getTime() / 1000)
      : -1,
    remain_quota: Number(form.remainQuota) || 0,
    unlimited_quota: form.unlimitedQuota,
    model_limits_enabled: form.modelLimitsEnabled,
    model_limits: form.modelLimits,
    allow_ips: form.allowIps.trim() || null,
    group: form.group.trim(),
    cross_group_retry: form.crossGroupRetry,
    rate_limit: form.rateLimit.trim(),
  }
}

function ApiKeyFormDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  editing: EnterpriseApiKeyItem | null
  users: Array<{
    id: number
    username: string
    display_name: string
    email: string
    group: string
  }>
  saving: boolean
  onSubmit: (input: EnterpriseApiKeyInput) => void
}) {
  const [form, setForm] = useState<ApiKeyFormState>(emptyForm)

  useEffect(() => {
    if (!props.open) return
    if (props.editing) {
      setForm(toForm(props.editing))
      return
    }
    setForm({
      ...emptyForm,
      userId: props.users[0] ? String(props.users[0].id) : '',
    })
  }, [props.open, props.editing, props.users])

  const update = <K extends keyof ApiKeyFormState>(
    key: K,
    value: ApiKeyFormState[K]
  ) => setForm((current) => ({ ...current, [key]: value }))
  let submitLabel = '创建密钥'
  if (props.saving) {
    submitLabel = '保存中…'
  } else if (props.editing) {
    submitLabel = '保存修改'
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>
            {props.editing ? '编辑企业密钥' : '创建企业密钥'}
          </DialogTitle>
          <DialogDescription>
            配置归属客户、额度、模型白名单、IP
            白名单和路由分组。密钥明文仅在创建或轮换后展示一次。
          </DialogDescription>
        </DialogHeader>

        <div className='grid gap-3 sm:grid-cols-2'>
          <Field>
            <FieldLabel htmlFor='enterprise-key-user'>
              归属用户 / 客户
            </FieldLabel>
            <NativeSelect
              id='enterprise-key-user'
              className='w-full'
              value={form.userId}
              disabled={props.editing != null}
              onChange={(event) => update('userId', event.target.value)}
            >
              {props.users.map((user) => (
                <NativeSelectOption key={user.id} value={user.id}>
                  {user.display_name || user.username} · {user.group}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </Field>
          <Field>
            <FieldLabel htmlFor='enterprise-key-name'>密钥名称</FieldLabel>
            <Input
              id='enterprise-key-name'
              value={form.name}
              placeholder='例如：生产环境主密钥'
              onChange={(event) => update('name', event.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor='enterprise-key-status'>状态</FieldLabel>
            <NativeSelect
              id='enterprise-key-status'
              className='w-full'
              value={form.status}
              onChange={(event) => update('status', event.target.value)}
            >
              <NativeSelectOption value={TOKEN_STATUS_ENABLED}>
                启用
              </NativeSelectOption>
              <NativeSelectOption value={TOKEN_STATUS_DISABLED}>
                禁用
              </NativeSelectOption>
            </NativeSelect>
          </Field>
          <Field>
            <FieldLabel htmlFor='enterprise-key-expiry'>到期时间</FieldLabel>
            <Input
              id='enterprise-key-expiry'
              type='datetime-local'
              value={form.expiredAt}
              onChange={(event) => update('expiredAt', event.target.value)}
            />
          </Field>
          <Field>
            <div className='flex items-center justify-between gap-3'>
              <FieldLabel htmlFor='enterprise-key-unlimited'>
                无限额度
              </FieldLabel>
              <Switch
                id='enterprise-key-unlimited'
                checked={form.unlimitedQuota}
                onCheckedChange={(checked) => update('unlimitedQuota', checked)}
              />
            </div>
            <Input
              type='number'
              min={0}
              disabled={form.unlimitedQuota}
              value={form.remainQuota}
              onChange={(event) => update('remainQuota', event.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor='enterprise-key-group'>路由分组</FieldLabel>
            <Input
              id='enterprise-key-group'
              value={form.group}
              placeholder='留空则继承用户分组'
              onChange={(event) => update('group', event.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor='enterprise-key-rate-limit'>
              QPS / 速率限制
            </FieldLabel>
            <Input
              id='enterprise-key-rate-limit'
              value={form.rateLimit}
              placeholder='留空继承租户，例如 200'
              onChange={(event) => update('rateLimit', event.target.value)}
            />
          </Field>
          <Field className='sm:col-span-2'>
            <div className='flex items-center justify-between gap-3'>
              <FieldLabel htmlFor='enterprise-key-model-limit'>
                模型白名单
              </FieldLabel>
              <Switch
                id='enterprise-key-model-limit'
                checked={form.modelLimitsEnabled}
                onCheckedChange={(checked) =>
                  update('modelLimitsEnabled', checked)
                }
              />
            </div>
            <Textarea
              value={form.modelLimits}
              disabled={!form.modelLimitsEnabled}
              placeholder='gpt-4o, claude-3-5-sonnet, deepseek-v3'
              onChange={(event) => update('modelLimits', event.target.value)}
            />
          </Field>
          <Field className='sm:col-span-2'>
            <FieldLabel htmlFor='enterprise-key-ip-list'>IP 白名单</FieldLabel>
            <Textarea
              id='enterprise-key-ip-list'
              value={form.allowIps}
              placeholder={
                '每行一个 IP 或 CIDR，例如：\n10.0.0.0/8\n203.0.113.10'
              }
              onChange={(event) => update('allowIps', event.target.value)}
            />
          </Field>
          <Field orientation='horizontal' className='sm:col-span-2'>
            <div>
              <FieldLabel htmlFor='enterprise-key-cross-retry'>
                允许跨分组重试
              </FieldLabel>
              <p className='text-muted-foreground text-xs'>
                仅在自动分组路由策略下生效。
              </p>
            </div>
            <Switch
              id='enterprise-key-cross-retry'
              checked={form.crossGroupRetry}
              onCheckedChange={(checked) => update('crossGroupRetry', checked)}
            />
          </Field>
        </div>

        <DialogFooter>
          <Button variant='outline' onClick={() => props.onOpenChange(false)}>
            取消
          </Button>
          <Button
            disabled={props.saving || !form.userId || !form.name.trim()}
            onClick={() => props.onSubmit(toInput(form))}
          >
            {submitLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function SecretDialog(props: {
  secret: EnterpriseApiKeySecret | null
  onOpenChange: (open: boolean) => void
}) {
  const [copied, setCopied] = useState(false)
  const copy = async () => {
    if (!props.secret) return
    await navigator.clipboard.writeText(props.secret.secret_key)
    setCopied(true)
    toast.success('密钥已复制，请妥善保存')
  }
  return (
    <Dialog open={props.secret != null} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>请立即保存密钥</DialogTitle>
          <DialogDescription>
            出于安全考虑，完整密钥只展示这一次。关闭后只能通过轮换生成新密钥。
          </DialogDescription>
        </DialogHeader>
        <div className='rounded-md border border-amber-500/25 bg-amber-500/5 p-4'>
          <div className='flex items-start gap-3'>
            <AlertTriangle className='mt-0.5 size-5 shrink-0 text-amber-600' />
            <div className='min-w-0 flex-1'>
              <p className='text-sm font-medium'>{props.secret?.item.name}</p>
              <code className='bg-background mt-3 block rounded-md px-3 py-3 text-xs break-all'>
                {props.secret?.secret_key}
              </code>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button onClick={copy}>
            {copied ? (
              <Check className='size-4' />
            ) : (
              <Copy className='size-4' />
            )}
            {copied ? '已复制' : '复制完整密钥'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function EnterpriseApiKeys() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState('all')
  const [tenantFilter, setTenantFilter] = useState('all')
  const [groupFilter, setGroupFilter] = useState('all')
  const [modelLimitMode, setModelLimitMode] = useState('any')
  const [createdStart, setCreatedStart] = useState('')
  const [createdEnd, setCreatedEnd] = useState('')
  const [detailTab, setDetailTab] = useState<
    'access' | 'quota' | 'security' | 'audit'
  >('access')
  const [selected, setSelected] = useState<EnterpriseApiKeyItem | null>(null)
  const [editing, setEditing] = useState<EnterpriseApiKeyItem | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [secret, setSecret] = useState<EnterpriseApiKeySecret | null>(null)
  const [confirmAction, setConfirmAction] = useState<
    'rotate' | 'delete' | null
  >(null)
  const [exporting, setExporting] = useState(false)

  const params = useMemo(
    () => ({
      page,
      page_size: pageSize,
      keyword: keyword.trim() || undefined,
      status: status === 'all' ? undefined : Number(status),
      user_id: tenantFilter === 'all' ? undefined : Number(tenantFilter),
      group: groupFilter === 'all' ? undefined : groupFilter,
      model_limit_mode: modelLimitMode === 'any' ? undefined : modelLimitMode,
      created_start: dateInputToTimestamp(createdStart, 'start'),
      created_end: dateInputToTimestamp(createdEnd, 'end'),
    }),
    [
      page,
      pageSize,
      keyword,
      status,
      tenantFilter,
      groupFilter,
      modelLimitMode,
      createdStart,
      createdEnd,
    ]
  )
  const keysQuery = useQuery({
    queryKey: ['enterprise-api-keys', params],
    queryFn: () => getEnterpriseApiKeys(params),
  })
  const usersQuery = useQuery({
    queryKey: ['enterprise-api-key-users'],
    queryFn: getEnterpriseApiKeyUsers,
  })
  const pageData = keysQuery.data?.data
  const summary = pageData?.summary
  const items = pageData?.items ?? EMPTY_API_KEY_ITEMS
  const totalPages = Math.max(1, Math.ceil((pageData?.total ?? 0) / pageSize))
  const users = usersQuery.data?.data ?? EMPTY_API_KEY_USERS
  const groupOptions = useMemo(() => {
    const groups = new Set<string>()
    users.forEach((user) => {
      if (user.group) groups.add(user.group)
    })
    items.forEach((item) => {
      if (item.group) groups.add(item.group)
      if (item.user_group) groups.add(item.user_group)
    })
    return [...groups].sort((a, b) => a.localeCompare(b))
  }, [items, users])
  const visibleActiveCount = useMemo(
    () =>
      items.filter((item) => item.effective_status === TOKEN_STATUS_ENABLED)
        .length,
    [items]
  )
  const visibleRestrictedCount = useMemo(
    () =>
      items.filter(
        (item) =>
          item.effective_status !== TOKEN_STATUS_ENABLED ||
          item.model_limits_enabled ||
          Boolean(item.allow_ips?.trim())
      ).length,
    [items]
  )
  const visibleModelLimitedCount = useMemo(
    () => items.filter((item) => item.model_limits_enabled).length,
    [items]
  )
  const visibleIpRestrictedCount = useMemo(
    () => items.filter((item) => Boolean(item.allow_ips?.trim())).length,
    [items]
  )
  const visibleFailureCount = useMemo(
    () =>
      items.reduce(
        (total, item) => total + Math.max(0, item.recent_failure_count ?? 0),
        0
      ),
    [items]
  )
  const visibleUsedQuota = useMemo(
    () =>
      items.reduce((total, item) => total + Math.max(0, item.used_quota), 0),
    [items]
  )
  const environmentStats = useMemo(() => {
    const groups = new Map<string, number>()
    items.forEach((item) => {
      const label = environmentLabel(item.group || item.user_group)
      groups.set(label, (groups.get(label) ?? 0) + 1)
    })
    return [...groups.entries()]
      .sort((left, right) => right[1] - left[1])
      .slice(0, 4)
  }, [items])
  const selectedStatus = selected
    ? tokenStatus(selected.effective_status)
    : null
  const selectedIpCount = selected?.allow_ips
    ? selected.allow_ips.split('\n').filter(Boolean).length
    : 0
  const selectedQuotaTotal = selected
    ? Math.max(0, selected.used_quota) + Math.max(0, selected.remain_quota)
    : 0
  let selectedQuotaUsage = 0
  if (selected && !selected.unlimited_quota && selectedQuotaTotal > 0) {
    selectedQuotaUsage = Math.round(
      (Math.max(0, selected.used_quota) / selectedQuotaTotal) * 100
    )
  }
  let selectedSecurityLevel = '未选择'
  if (selected?.allow_ips?.trim()) {
    selectedSecurityLevel = 'IP 白名单'
  } else if (selected?.model_limits_enabled) {
    selectedSecurityLevel = '模型白名单'
  } else if (selected) {
    selectedSecurityLevel = '基础访问'
  }
  const selectedBaseUrl =
    typeof window === 'undefined' ? '/v1' : `${window.location.origin}/v1`
  const selectedClientId = selected
    ? `tr_client_${selected.id}_${selected.masked_key.replace(/^sk-/, '')}`
    : ''
  const sdkSnippet = selected
    ? [
        `curl -X POST ${selectedBaseUrl}/chat/completions \\`,
        `  -H "Authorization: Bearer ${selected.masked_key}" \\`,
        '  -H "Content-Type: application/json" \\',
        "  -d '{",
        '    "model": "gpt-4o",',
        '    "messages": [{"role": "user", "content": "Hello"}]',
        "  }'",
      ].join('\n')
    : ''
  const auditEvents = selected
    ? [
        selected.accessed_time > 0
          ? {
              time: selected.accessed_time,
              title: 'API Key 使用',
              detail: `${keyOwner(selected)} 最近一次调用`,
              status: '成功',
            }
          : null,
        {
          time: selected.created_time,
          title: '创建 API Key',
          detail: `${selected.name} 已创建`,
          status: '活跃',
        },
        selected.expired_time > 0
          ? {
              time: selected.expired_time,
              title: '到期策略',
              detail: `密钥将在 ${dayjs.unix(selected.expired_time).format('YYYY-MM-DD HH:mm')} 到期`,
              status:
                selected.expired_time <= dayjs().unix() ? '已过期' : '计划',
            }
          : null,
      ].filter(
        (
          event
        ): event is {
          time: number
          title: string
          detail: string
          status: string
        } => event != null
      )
    : []

  const resetFilters = () => {
    setKeyword('')
    setStatus('all')
    setTenantFilter('all')
    setGroupFilter('all')
    setModelLimitMode('any')
    setCreatedStart('')
    setCreatedEnd('')
    setPage(1)
  }

  useEffect(() => {
    if (items.length === 0) {
      if (selected) setSelected(null)
      return
    }
    if (!selected || !items.some((item) => item.id === selected.id)) {
      setSelected(items[0])
    }
  }, [items, selected])

  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey: ['enterprise-api-keys'] })
  }
  const exportKeys = async () => {
    setExporting(true)
    try {
      await exportEnterpriseApiKeys(params)
      toast.success('企业密钥清单已导出')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '导出失败')
    } finally {
      setExporting(false)
    }
  }
  const saveMutation = useMutation({
    mutationFn: async (input: EnterpriseApiKeyInput) => {
      if (editing) {
        return updateEnterpriseApiKey(editing.id, input)
      }
      return createEnterpriseApiKey(input)
    },
    onSuccess: async (response) => {
      if (!response.success || !response.data) return
      if ('secret_key' in response.data) {
        setSecret(response.data)
      } else {
        setSelected(response.data)
      }
      setFormOpen(false)
      setEditing(null)
      toast.success('企业密钥已保存')
      await refresh()
    },
  })
  const rotateMutation = useMutation({
    mutationFn: (id: number) => rotateEnterpriseApiKey(id),
    onSuccess: async (response) => {
      if (response.success && response.data) {
        setSecret(response.data)
        toast.success('密钥已轮换，旧密钥立即失效')
        await refresh()
      }
      setConfirmAction(null)
    },
  })
  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteEnterpriseApiKey(id),
    onSuccess: async (response) => {
      if (response.success) {
        toast.success('企业密钥已删除')
        setSelected(null)
        await refresh()
      }
      setConfirmAction(null)
    },
  })

  const openCreate = () => {
    setEditing(null)
    setFormOpen(true)
  }
  const openEdit = (item: EnterpriseApiKeyItem) => {
    setEditing(item)
    setFormOpen(true)
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Content>
        <div className='flex h-full min-h-0 flex-col gap-3 overflow-auto pb-5'>
          <EnterprisePageHeader
            title='API Keys 与客户接入'
            description='面向企业客户的密钥发放、租户接入与访问控制'
          />

          <div className='grid min-h-0 gap-2 xl:grid-cols-[minmax(0,1fr)_330px] 2xl:grid-cols-[minmax(0,1fr)_384px]'>
            <div className='flex min-w-0 flex-col gap-2'>
              <div className='grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-5'>
                <MetricTile
                  title='活跃 API Keys'
                  value={formatCount(summary?.active ?? 0)}
                  helper='可正常调用'
                  icon={KeyRound}
                  tone='blue'
                  loading={keysQuery.isLoading}
                />
                <MetricTile
                  title='活跃租户'
                  value={formatCount(summary?.active_users ?? 0)}
                  helper='已分配密钥用户'
                  icon={Users}
                  tone='violet'
                  loading={keysQuery.isLoading}
                />
                <MetricTile
                  title='即将过期密钥'
                  value={formatCount(summary?.expiring_soon ?? 0)}
                  helper='未来 7 天'
                  icon={CalendarClock}
                  tone='amber'
                  loading={keysQuery.isLoading}
                />
                <MetricTile
                  title='本月调用额度'
                  value={formatLogQuota(summary?.total_used_quota ?? 0)}
                  helper='按当前权限汇总'
                  icon={ShieldCheck}
                  tone='emerald'
                  loading={keysQuery.isLoading}
                />
                <MetricTile
                  title='限流命中次数'
                  value={formatCount(summary?.rate_limit_hits ?? 0)}
                  helper='近 24h 触发'
                  icon={AlertTriangle}
                  tone='rose'
                  loading={keysQuery.isLoading}
                />
              </div>

              <EnterprisePanel bodyClassName='p-2.5'>
                <div className='grid gap-2 md:grid-cols-2 xl:grid-cols-[minmax(126px,0.9fr)_minmax(126px,0.9fr)_minmax(138px,0.95fr)_minmax(118px,0.78fr)] 2xl:grid-cols-[minmax(126px,0.9fr)_minmax(126px,0.9fr)_minmax(138px,0.95fr)_minmax(118px,0.78fr)_minmax(178px,1.18fr)_minmax(230px,1.45fr)_64px]'>
                  <FilterField label='租户'>
                    <NativeSelect
                      value={tenantFilter}
                      className='h-6 w-full rounded border-0 bg-transparent px-0 text-xs shadow-none'
                      onChange={(event) => {
                        setTenantFilter(event.target.value)
                        setPage(1)
                      }}
                    >
                      <NativeSelectOption value='all'>
                        全部租户
                      </NativeSelectOption>
                      {users.map((user) => (
                        <NativeSelectOption
                          key={user.id}
                          value={String(user.id)}
                        >
                          {user.display_name || user.username}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </FilterField>
                  <FilterField label='环境'>
                    <NativeSelect
                      value={groupFilter}
                      className='h-6 w-full rounded border-0 bg-transparent px-0 text-xs shadow-none'
                      onChange={(event) => {
                        setGroupFilter(event.target.value)
                        setPage(1)
                      }}
                    >
                      <NativeSelectOption value='all'>
                        全部环境
                      </NativeSelectOption>
                      {groupOptions.map((group) => (
                        <NativeSelectOption key={group} value={group}>
                          {environmentLabel(group)}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </FilterField>
                  <FilterField label='授权模型'>
                    <NativeSelect
                      value={modelLimitMode}
                      className='h-6 w-full rounded border-0 bg-transparent px-0 text-xs shadow-none'
                      onChange={(event) => {
                        setModelLimitMode(event.target.value)
                        setPage(1)
                      }}
                    >
                      <NativeSelectOption value='any'>全部</NativeSelectOption>
                      <NativeSelectOption value='all'>
                        全量模型
                      </NativeSelectOption>
                      <NativeSelectOption value='restricted'>
                        白名单模型
                      </NativeSelectOption>
                    </NativeSelect>
                  </FilterField>
                  <FilterField label='状态'>
                    <NativeSelect
                      value={status}
                      className='h-6 w-full rounded border-0 bg-transparent px-0 text-xs shadow-none'
                      onChange={(event) => {
                        setStatus(event.target.value)
                        setPage(1)
                      }}
                    >
                      <NativeSelectOption value='all'>全部</NativeSelectOption>
                      <NativeSelectOption value={TOKEN_STATUS_ENABLED}>
                        活跃
                      </NativeSelectOption>
                      <NativeSelectOption value={TOKEN_STATUS_DISABLED}>
                        已禁用
                      </NativeSelectOption>
                      <NativeSelectOption value={TOKEN_STATUS_EXPIRED}>
                        已过期
                      </NativeSelectOption>
                      <NativeSelectOption value={TOKEN_STATUS_EXHAUSTED}>
                        额度耗尽
                      </NativeSelectOption>
                    </NativeSelect>
                  </FilterField>
                  <FilterField label='日期'>
                    <div className='flex min-w-0 items-center gap-1'>
                      <CalendarClock className='size-3.5 shrink-0 text-slate-400' />
                      <Input
                        value={createdStart}
                        placeholder='开始'
                        aria-label='开始日期，格式 YYYY-MM-DD'
                        className='h-6 min-w-0 rounded border-0 px-1 text-[11px] shadow-none focus-visible:ring-0'
                        onChange={(event) => {
                          setCreatedStart(event.target.value)
                          setPage(1)
                        }}
                      />
                      <span className='text-[11px] text-slate-400'>~</span>
                      <Input
                        value={createdEnd}
                        placeholder='结束'
                        aria-label='结束日期，格式 YYYY-MM-DD'
                        className='h-6 min-w-0 rounded border-0 px-1 text-[11px] shadow-none focus-visible:ring-0'
                        onChange={(event) => {
                          setCreatedEnd(event.target.value)
                          setPage(1)
                        }}
                      />
                    </div>
                  </FilterField>
                  <div className='relative'>
                    <Search className='pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-slate-400' />
                    <Input
                      value={keyword}
                      className='h-8 rounded-md pl-8 text-xs'
                      placeholder='搜索客户 / 租户 / 密钥名称 / Key ID'
                      onChange={(event) => {
                        setKeyword(event.target.value)
                        setPage(1)
                      }}
                    />
                  </div>
                  <Button
                    variant='outline'
                    size='sm'
                    className='h-8'
                    onClick={resetFilters}
                  >
                    重置
                  </Button>
                </div>
              </EnterprisePanel>

              <EnterprisePanel bodyClassName='p-0' className='min-w-0'>
                <div className='flex items-center justify-between gap-2 border-b border-slate-100 px-3 py-2'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <Button
                      size='sm'
                      className='h-8 bg-blue-600 text-white hover:bg-blue-700'
                      onClick={openCreate}
                    >
                      <Plus className='size-3.5' />
                      创建 API Key
                    </Button>
                    <Button
                      size='sm'
                      variant='outline'
                      className='h-8'
                      onClick={() => void exportKeys()}
                      disabled={exporting || keysQuery.isFetching}
                    >
                      <Download className='size-3.5' />
                      批量导出
                    </Button>
                    <Button
                      size='sm'
                      variant='outline'
                      className='h-8'
                      disabled={!selected}
                      onClick={() => setConfirmAction('rotate')}
                    >
                      <RotateCcw className='size-3.5' />
                      轮换密钥
                    </Button>
                    <Button
                      size='sm'
                      variant='outline'
                      className='h-8 border-rose-100 bg-rose-50 text-rose-600 hover:bg-rose-100 hover:text-rose-700'
                      disabled={!selected}
                      onClick={() => setConfirmAction('delete')}
                    >
                      <Trash2 className='size-3.5' />
                      吊销访问
                    </Button>
                  </div>
                  <div className='flex items-center gap-2'>
                    <Button variant='outline' size='sm' className='h-8'>
                      <MoreHorizontal className='size-3.5' />
                      列设置
                    </Button>
                    <Button
                      variant='outline'
                      size='icon-sm'
                      aria-label='刷新密钥列表'
                      onClick={() => keysQuery.refetch()}
                      disabled={keysQuery.isFetching}
                    >
                      <RefreshCw
                        className={cn(
                          'size-3.5',
                          keysQuery.isFetching && 'animate-spin'
                        )}
                      />
                    </Button>
                  </div>
                </div>
                <div className='overflow-x-auto'>
                  <Table className='text-xs [&_td]:h-12 [&_td]:py-1.5 [&_td]:text-xs [&_td_*]:text-xs [&_th]:h-8 [&_th]:text-xs [&_th_*]:text-xs'>
                    <TableHeader className='bg-slate-50'>
                      <TableRow>
                        <TableHead className='w-8'>
                          <span className='sr-only'>选择</span>
                        </TableHead>
                        <TableHead>客户 / 租户</TableHead>
                        <TableHead>环境</TableHead>
                        <TableHead>密钥名称 / Key ID</TableHead>
                        <TableHead>授权模型</TableHead>
                        <TableHead>额度 / 配额</TableHead>
                        <TableHead>QPS 限流</TableHead>
                        <TableHead>IP 白名单</TableHead>
                        <TableHead>最近使用</TableHead>
                        <TableHead>状态</TableHead>
                        <TableHead className='w-28 text-right'>操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {items.length === 0 ? (
                        <TableRow>
                          <TableCell
                            colSpan={11}
                            className='h-44 text-center text-xs text-slate-500'
                          >
                            {keysQuery.isLoading
                              ? '正在加载企业密钥...'
                              : '没有符合条件的企业密钥'}
                          </TableCell>
                        </TableRow>
                      ) : (
                        items.map((item) => {
                          const statusConfig = tokenStatus(
                            item.effective_status
                          )
                          const ipCount = item.allow_ips
                            ? item.allow_ips.split('\n').filter(Boolean).length
                            : 0
                          const envLabel = environmentLabel(
                            item.group || item.user_group
                          )
                          return (
                            <TableRow
                              key={item.id}
                              className={cn(
                                'cursor-pointer',
                                selected?.id === item.id &&
                                  'bg-blue-50/70 ring-1 ring-inset ring-blue-200'
                              )}
                              onClick={() => {
                                setSelected(item)
                                setDetailTab('access')
                              }}
                            >
                              <TableCell>
                                <input
                                  type='checkbox'
                                  aria-label={`选择 ${item.name}`}
                                  checked={selected?.id === item.id}
                                  onChange={() => setSelected(item)}
                                  onClick={(event) => event.stopPropagation()}
                                  className='size-3.5 rounded border-slate-300'
                                />
                              </TableCell>
                              <TableCell>
                                <div className='max-w-44'>
                                  <p className='truncate font-semibold text-slate-900'>
                                    {keyOwner(item)}
                                  </p>
                                  <p className='truncate text-[11px] text-slate-500'>
                                    {item.username ||
                                      item.email ||
                                      item.user_group}
                                  </p>
                                </div>
                              </TableCell>
                              <TableCell>
                                <Badge
                                  variant='outline'
                                  className={cn(
                                    'h-5 rounded px-2 text-[10px]',
                                    environmentBadgeClass(envLabel)
                                  )}
                                >
                                  {envLabel}
                                </Badge>
                              </TableCell>
                              <TableCell>
                                <p className='max-w-44 truncate font-semibold text-slate-900'>
                                  {item.name}
                                </p>
                                <div className='flex items-center gap-1'>
                                  <code className='max-w-36 truncate text-[11px] text-slate-500'>
                                    {item.masked_key}
                                  </code>
                                  <Button
                                    size='icon-xs'
                                    variant='ghost'
                                    aria-label='复制脱敏 Key ID'
                                    onClick={(event) => {
                                      event.stopPropagation()
                                      void navigator.clipboard.writeText(
                                        item.masked_key
                                      )
                                      toast.success('已复制脱敏 Key ID')
                                    }}
                                  >
                                    <Copy className='size-3' />
                                  </Button>
                                </div>
                              </TableCell>
                              <TableCell>{modelChips(item)}</TableCell>
                              <TableCell>
                                <p className='font-semibold text-slate-900'>
                                  {item.unlimited_quota
                                    ? '无限额度'
                                    : formatLogQuota(item.remain_quota)}
                                </p>
                                <p className='text-[11px] text-slate-500'>
                                  已用 {formatLogQuota(item.used_quota)}
                                </p>
                              </TableCell>
                              <TableCell>
                                {item.rate_limit?.trim() || '继承租户'}
                              </TableCell>
                              <TableCell>
                                {ipCount > 0 ? `${ipCount} 条` : '0 条'}
                              </TableCell>
                              <TableCell>
                                {item.accessed_time > 0
                                  ? formatRelativeTimestamp(item.accessed_time)
                                  : '从未使用'}
                              </TableCell>
                              <TableCell>
                                <Badge
                                  className={cn(
                                    'h-5 rounded px-2 text-[10px]',
                                    statusConfig.className
                                  )}
                                >
                                  {statusConfig.label}
                                </Badge>
                              </TableCell>
                              <TableCell className='text-right'>
                                <div className='flex justify-end gap-1'>
                                  <Button
                                    size='icon-xs'
                                    variant='ghost'
                                    aria-label='编辑密钥'
                                    onClick={(event) => {
                                      event.stopPropagation()
                                      openEdit(item)
                                    }}
                                  >
                                    <Edit3 className='size-3.5' />
                                  </Button>
                                  <Button
                                    size='icon-xs'
                                    variant='ghost'
                                    aria-label='轮换密钥'
                                    onClick={(event) => {
                                      event.stopPropagation()
                                      setSelected(item)
                                      setConfirmAction('rotate')
                                    }}
                                  >
                                    <RotateCcw className='size-3.5' />
                                  </Button>
                                  <Button
                                    size='icon-xs'
                                    variant='ghost'
                                    className='text-rose-600'
                                    aria-label='删除密钥'
                                    onClick={(event) => {
                                      event.stopPropagation()
                                      setSelected(item)
                                      setConfirmAction('delete')
                                    }}
                                  >
                                    <Trash2 className='size-3.5' />
                                  </Button>
                                </div>
                              </TableCell>
                            </TableRow>
                          )
                        })
                      )}
                    </TableBody>
                  </Table>
                </div>
                <div className='grid gap-3 border-t border-slate-100 bg-slate-50/20 px-3 py-2.5 xl:grid-cols-[1.05fr_0.95fr_1fr]'>
                  <div className='min-w-0'>
                    <div className='flex items-center justify-between gap-2'>
                      <p className='text-xs font-semibold text-slate-900'>
                        当前筛选概览
                      </p>
                      <span className='text-[11px] text-slate-500'>
                        第 {page} 页 / 共 {totalPages} 页
                      </span>
                    </div>
                    <div className='mt-2 grid grid-cols-3 gap-2'>
                      <CompactSummaryItem
                        label='可调用'
                        value={formatCount(summary?.active ?? 0)}
                        helper={`当前页 ${visibleActiveCount}`}
                        tone='emerald'
                      />
                      <CompactSummaryItem
                        label='受限/失效'
                        value={formatCount(
                          (summary?.disabled ?? 0) + (summary?.exhausted ?? 0)
                        )}
                        helper={`当前页 ${visibleRestrictedCount}`}
                        tone='rose'
                      />
                      <CompactSummaryItem
                        label='即将过期'
                        value={formatCount(summary?.expiring_soon ?? 0)}
                        helper='未来 7 天'
                        tone='amber'
                      />
                    </div>
                  </div>
                  <div className='min-w-0 border-slate-100 xl:border-l xl:pl-3'>
                    <p className='text-xs font-semibold text-slate-900'>
                      环境与策略覆盖
                    </p>
                    <div className='mt-2 space-y-1.5'>
                      {(environmentStats.length > 0
                        ? environmentStats
                        : [['无数据', 0] as [string, number]]
                      ).map(([label, count]) => (
                        <div
                          key={label}
                          className='grid grid-cols-[54px_minmax(0,1fr)_28px] items-center gap-2 text-[11px]'
                        >
                          <span className='truncate font-medium text-slate-600'>
                            {label}
                          </span>
                          <ThinProgress
                            value={
                              items.length > 0
                                ? (count / items.length) * 100
                                : 0
                            }
                            tone={label === '生产' ? 'emerald' : 'blue'}
                          />
                          <span className='text-right text-slate-500 tabular-nums'>
                            {count}
                          </span>
                        </div>
                      ))}
                    </div>
                    <p className='mt-2 truncate text-[11px] text-slate-500'>
                      模型白名单 {visibleModelLimitedCount} · IP 白名单{' '}
                      {visibleIpRestrictedCount}
                    </p>
                  </div>
                  <div className='min-w-0 border-slate-100 xl:border-l xl:pl-3'>
                    <p className='text-xs font-semibold text-slate-900'>
                      调用与额度状态
                    </p>
                    <div className='mt-2 grid grid-cols-3 gap-2'>
                      <CompactSummaryItem
                        label='可见用量'
                        value={formatCompactQuota(visibleUsedQuota)}
                        helper='当前页'
                        tone='blue'
                      />
                      <CompactSummaryItem
                        label='24h 失败'
                        value={formatCount(visibleFailureCount)}
                        helper='当前页'
                        tone={visibleFailureCount > 0 ? 'rose' : 'emerald'}
                      />
                      <CompactSummaryItem
                        label='已分配客户'
                        value={formatCount(summary?.active_users ?? 0)}
                        helper={`可选 ${users.length}`}
                        tone='slate'
                      />
                    </div>
                  </div>
                </div>
                <div className='flex items-center justify-between border-t border-slate-100 px-3 py-2'>
                  <span className='text-xs text-slate-500'>
                    共 {pageData?.total ?? 0} 条 · 密钥明文不在列表中返回
                  </span>
                  <div className='flex items-center gap-2'>
                    <Button
                      size='xs'
                      variant='outline'
                      disabled={page <= 1}
                      onClick={() => setPage((value) => Math.max(1, value - 1))}
                    >
                      ‹
                    </Button>
                    {[...Array(Math.min(totalPages, 5))].map((_, index) => {
                      const pageNumber = index + 1
                      return (
                        <Button
                          key={pageNumber}
                          size='xs'
                          variant={page === pageNumber ? 'default' : 'ghost'}
                          className={cn(
                            'size-7 px-0',
                            page === pageNumber &&
                              'bg-blue-600 text-white hover:bg-blue-700'
                          )}
                          onClick={() => setPage(pageNumber)}
                        >
                          {pageNumber}
                        </Button>
                      )
                    })}
                    {totalPages > 5 && (
                      <span className='text-xs text-slate-400'>...</span>
                    )}
                    <Button
                      size='xs'
                      variant='outline'
                      disabled={page >= totalPages}
                      onClick={() => setPage((value) => value + 1)}
                    >
                      ›
                    </Button>
                    <NativeSelect
                      value={String(pageSize)}
                      className='h-7 w-24 rounded-md text-xs'
                      onChange={(event) => {
                        setPageSize(Number(event.target.value))
                        setPage(1)
                      }}
                    >
                      {[10, 20, 50].map((size) => (
                        <NativeSelectOption key={size} value={String(size)}>
                          {size} 条 / 页
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </div>
                </div>
              </EnterprisePanel>
            </div>

            <EnterprisePanel
              className='min-w-0 xl:sticky xl:top-3 xl:self-start'
              bodyClassName='p-0 xl:max-h-[calc(100vh-88px)] xl:overflow-y-auto'
            >
              {!selected ? (
                <div className='flex min-h-[520px] flex-col items-center justify-center text-center'>
                  <span className='flex size-11 items-center justify-center rounded-md bg-blue-50 text-blue-600'>
                    <KeyRound className='size-5' />
                  </span>
                  <p className='mt-3 text-sm font-semibold text-slate-900'>
                    企业密钥治理
                  </p>
                  <p className='mt-1 max-w-64 text-xs leading-5 text-slate-500'>
                    查看 Base URL、SDK 示例、额度、安全策略和审计状态。
                  </p>
                </div>
              ) : (
                <div className='min-h-[520px]'>
                  <div className='flex items-start justify-between gap-3 border-b border-slate-100 p-3'>
                    <div className='flex min-w-0 items-start gap-3'>
                      <span className='flex size-9 shrink-0 items-center justify-center rounded-md bg-blue-50 text-blue-600 ring-1 ring-blue-100'>
                        <KeyRound className='size-4' />
                      </span>
                      <div className='min-w-0'>
                        <p className='truncate text-sm font-semibold text-slate-900'>
                          {keyOwner(selected)}
                        </p>
                        <p className='mt-0.5 truncate text-[11px] text-slate-500'>
                          {selected.username ||
                            selected.email ||
                            selected.user_group ||
                            '未绑定用户'}
                        </p>
                      </div>
                    </div>
                    <div className='flex shrink-0 items-center gap-2'>
                      {selectedStatus != null && (
                        <Badge
                          className={cn(
                            'h-5 rounded px-2 text-[10px]',
                            selectedStatus.className
                          )}
                        >
                          {selectedStatus.label}
                        </Badge>
                      )}
                      <Button
                        variant='outline'
                        size='icon-sm'
                        aria-label='更多密钥操作'
                      >
                        <MoreHorizontal className='size-3.5' />
                      </Button>
                    </div>
                  </div>
                  <div className='flex border-b border-slate-100 px-3'>
                    {[
                      ['access', '接入详情'],
                      ['quota', '额度与用量'],
                      ['security', '安全与策略'],
                      ['audit', '审计日志'],
                    ].map(([value, label]) => (
                      <button
                        key={value}
                        type='button'
                        className={cn(
                          'h-9 border-b-2 px-2 text-xs font-medium',
                          detailTab === value
                            ? 'border-blue-600 text-blue-600'
                            : 'border-transparent text-slate-500 hover:text-slate-900'
                        )}
                        onClick={() =>
                          setDetailTab(
                            value as 'access' | 'quota' | 'security' | 'audit'
                          )
                        }
                      >
                        {label}
                      </button>
                    ))}
                  </div>

                  <div className='space-y-2.5 p-3'>
                    {detailTab === 'access' && (
                      <>
                        <div className='grid grid-cols-3 gap-2 border-b border-slate-100 pb-2.5'>
                          <div>
                            <p className='text-[11px] text-slate-500'>
                              剩余额度
                            </p>
                            <p className='mt-0.5 truncate text-[13px] font-semibold text-slate-950'>
                              {selected.unlimited_quota
                                ? '无限'
                                : formatLogQuota(selected.remain_quota)}
                            </p>
                          </div>
                          <div>
                            <p className='text-[11px] text-slate-500'>
                              安全边界
                            </p>
                            <p className='mt-0.5 truncate text-[13px] font-semibold text-slate-950'>
                              {selectedSecurityLevel}
                            </p>
                          </div>
                          <div>
                            <p className='text-[11px] text-slate-500'>
                              24h 失败
                            </p>
                            <p
                              className={cn(
                                'mt-0.5 truncate text-[13px] font-semibold',
                                (selected.recent_failure_count ?? 0) > 0
                                  ? 'text-rose-600'
                                  : 'text-emerald-600'
                              )}
                            >
                              {formatCount(selected.recent_failure_count ?? 0)}
                            </p>
                          </div>
                        </div>
                        <DetailField
                          label='Base URL'
                          value={selectedBaseUrl}
                          copyValue={selectedBaseUrl}
                        />
                        <DetailField
                          label='Webhook 回调地址'
                          value='未配置回调地址'
                        />
                        <DetailField
                          label='Client ID'
                          value={selectedClientId}
                          copyValue={selectedClientId}
                        />
                        <div>
                          <div className='flex items-center justify-between'>
                            <p className='text-xs font-semibold text-slate-900'>
                              SDK 快速接入
                            </p>
                            <div className='flex gap-1'>
                              {['cURL', 'Python', 'Node.js', 'Java'].map(
                                (item) => (
                                  <Badge
                                    key={item}
                                    variant='outline'
                                    className='h-5 rounded px-2 text-[10px]'
                                  >
                                    {item}
                                  </Badge>
                                )
                              )}
                            </div>
                          </div>
                          <div className='mt-2 rounded-md border border-slate-200 bg-slate-50 p-2'>
                            <pre className='max-h-36 overflow-auto text-[11px] leading-5 whitespace-pre-wrap text-slate-700'>
                              {sdkSnippet}
                            </pre>
                            <div className='mt-2 flex justify-end'>
                              <Button
                                size='xs'
                                variant='outline'
                                onClick={() => {
                                  void navigator.clipboard.writeText(sdkSnippet)
                                  toast.success('SDK 示例已复制')
                                }}
                              >
                                <Copy className='size-3' />
                                复制
                              </Button>
                            </div>
                          </div>
                        </div>
                        <div className='border-t border-slate-100 pt-3'>
                          <p className='text-xs font-semibold text-slate-900'>
                            回调与状态
                          </p>
                          <div className='mt-2 grid grid-cols-3 gap-2 text-[11px]'>
                            <div>
                              <p className='text-slate-500'>访问状态</p>
                              <p
                                className={cn(
                                  'mt-1 inline-flex items-center gap-1 font-semibold',
                                  selected.effective_status ===
                                    TOKEN_STATUS_ENABLED
                                    ? 'text-emerald-600'
                                    : 'text-rose-600'
                                )}
                              >
                                <span
                                  className={cn(
                                    'size-1.5 rounded-full',
                                    selected.effective_status ===
                                      TOKEN_STATUS_ENABLED
                                      ? 'bg-emerald-500'
                                      : 'bg-rose-500'
                                  )}
                                />
                                {selectedStatus?.label ?? '未知'}
                              </p>
                            </div>
                            <div>
                              <p className='text-slate-500'>最近调用</p>
                              <p className='mt-1 font-semibold text-slate-800'>
                                {selected.accessed_time > 0
                                  ? dayjs
                                      .unix(selected.accessed_time)
                                      .format('YYYY-MM-DD HH:mm')
                                  : '暂无调用'}
                              </p>
                            </div>
                            <div>
                              <p className='text-slate-500'>失败次数(24h)</p>
                              <p className='mt-1 font-semibold text-slate-800'>
                                {formatCount(
                                  selected.recent_failure_count ?? 0
                                )}
                              </p>
                            </div>
                          </div>
                        </div>
                        <div className='border-t border-slate-100 pt-3'>
                          <div className='flex items-center justify-between'>
                            <p className='text-xs font-semibold text-slate-900'>
                              最近审计活动
                            </p>
                            <button
                              type='button'
                              className='text-[11px] font-medium text-blue-600 hover:underline'
                              onClick={() => setDetailTab('audit')}
                            >
                              查看完整审计日志 →
                            </button>
                          </div>
                          <div className='mt-2 space-y-2'>
                            {auditEvents.slice(0, 4).map((event) => (
                              <div
                                key={`access-${event.title}-${event.time}`}
                                className='grid grid-cols-[94px_minmax(0,1fr)_42px] items-center gap-2 text-[11px]'
                              >
                                <span className='text-slate-500'>
                                  {event.time > 0
                                    ? dayjs
                                        .unix(event.time)
                                        .format('YYYY-MM-DD HH:mm')
                                    : '-'}
                                </span>
                                <span className='truncate text-slate-700'>
                                  {event.title}
                                </span>
                                <Badge className='h-5 rounded bg-emerald-50 px-2 text-[10px] text-emerald-600'>
                                  {event.status}
                                </Badge>
                              </div>
                            ))}
                          </div>
                        </div>
                      </>
                    )}

                    {detailTab === 'quota' && (
                      <>
                        <div className='grid grid-cols-2 gap-2 text-xs'>
                          <div className='rounded-md border border-slate-100 bg-slate-50/40 p-2.5'>
                            <p className='text-slate-500'>剩余额度</p>
                            <p className='mt-1 text-lg font-semibold text-slate-950'>
                              {selected.unlimited_quota
                                ? '无限'
                                : formatLogQuota(selected.remain_quota)}
                            </p>
                          </div>
                          <div className='rounded-md border border-slate-100 bg-slate-50/40 p-2.5'>
                            <p className='text-slate-500'>已用额度</p>
                            <p className='mt-1 text-lg font-semibold text-slate-950'>
                              {formatLogQuota(selected.used_quota)}
                            </p>
                          </div>
                          <div className='rounded-md border border-slate-100 bg-slate-50/40 p-2.5'>
                            <p className='text-slate-500'>最近调用</p>
                            <p className='mt-1 font-semibold text-slate-950'>
                              {selected.accessed_time > 0
                                ? dayjs
                                    .unix(selected.accessed_time)
                                    .format('YYYY-MM-DD HH:mm')
                                : '从未使用'}
                            </p>
                          </div>
                          <div className='rounded-md border border-slate-100 bg-slate-50/40 p-2.5'>
                            <p className='text-slate-500'>过期时间</p>
                            <p className='mt-1 font-semibold text-slate-950'>
                              {selected.expired_time > 0
                                ? dayjs
                                    .unix(selected.expired_time)
                                    .format('YYYY-MM-DD HH:mm')
                                : '永不过期'}
                            </p>
                          </div>
                        </div>
                        <div className='rounded-md border border-slate-100 bg-slate-50/30 p-2.5'>
                          <div className='flex items-center justify-between text-[11px]'>
                            <span className='font-medium text-slate-600'>
                              配额消耗
                            </span>
                            <span className='font-semibold text-slate-900'>
                              {selected.unlimited_quota
                                ? '无限额度'
                                : `${selectedQuotaUsage}%`}
                            </span>
                          </div>
                          <div className='mt-2'>
                            <ThinProgress
                              value={
                                selected.unlimited_quota
                                  ? 0
                                  : selectedQuotaUsage
                              }
                              tone='emerald'
                            />
                          </div>
                          <p className='mt-2 text-[11px] text-slate-500'>
                            按已用额度和剩余额度实时估算，不展示密钥明文。
                          </p>
                        </div>
                        <Button
                          size='sm'
                          className='w-full'
                          onClick={() => openEdit(selected)}
                        >
                          <Edit3 className='size-3.5' />
                          调整额度与有效期
                        </Button>
                      </>
                    )}

                    {detailTab === 'security' && (
                      <>
                        <div className='rounded-md border border-slate-100 bg-slate-50/35 p-2.5'>
                          <p className='text-xs font-semibold text-slate-900'>
                            授权模型
                          </p>
                          <div className='mt-2'>{modelChips(selected)}</div>
                        </div>
                        <div className='rounded-md border border-slate-100 bg-slate-50/35 p-2.5'>
                          <div className='flex items-center justify-between gap-2'>
                            <p className='text-xs font-semibold text-slate-900'>
                              IP 白名单
                            </p>
                            <Badge
                              variant='outline'
                              className='h-5 rounded px-2 text-[10px]'
                            >
                              {selectedIpCount} 条
                            </Badge>
                          </div>
                          <p className='mt-2 text-xs leading-5 whitespace-pre-wrap text-slate-600'>
                            {selected.allow_ips || '未限制来源 IP'}
                          </p>
                        </div>
                        <div className='grid grid-cols-2 gap-2 text-xs'>
                          <div className='rounded-md border border-slate-100 bg-slate-50/35 p-2.5'>
                            <p className='text-slate-500'>路由分组</p>
                            <p className='mt-1 font-semibold text-slate-950'>
                              {selected.group || '继承用户'}
                            </p>
                          </div>
                          <div className='rounded-md border border-slate-100 bg-slate-50/35 p-2.5'>
                            <p className='text-slate-500'>速率限制</p>
                            <p className='mt-1 font-semibold text-slate-950'>
                              {selected.rate_limit?.trim() || '继承租户'}
                            </p>
                          </div>
                          <div className='rounded-md border border-slate-100 bg-slate-50/35 p-2.5'>
                            <p className='text-slate-500'>跨组重试</p>
                            <p className='mt-1 font-semibold text-slate-950'>
                              {selected.cross_group_retry ? '允许' : '不允许'}
                            </p>
                          </div>
                        </div>
                      </>
                    )}

                    {detailTab === 'audit' && (
                      <div className='space-y-2'>
                        {auditEvents.map((event) => (
                          <div
                            key={`${event.title}-${event.time}`}
                            className='grid grid-cols-[96px_minmax(0,1fr)_44px] gap-2 rounded-md border border-slate-100 px-2 py-2 text-xs'
                          >
                            <span className='text-slate-500'>
                              {event.time > 0
                                ? dayjs.unix(event.time).format('MM-DD HH:mm')
                                : '-'}
                            </span>
                            <span className='min-w-0'>
                              <span className='block truncate font-semibold text-slate-900'>
                                {event.title}
                              </span>
                              <span className='block truncate text-[11px] text-slate-500'>
                                {event.detail}
                              </span>
                            </span>
                            <Badge className='h-5 rounded bg-emerald-50 px-2 text-[10px] text-emerald-600'>
                              {event.status}
                            </Badge>
                          </div>
                        ))}
                      </div>
                    )}

                    <div className='flex gap-2 border-t border-slate-100 pt-3'>
                      <Button
                        size='sm'
                        className='flex-1 bg-blue-600 text-white hover:bg-blue-700'
                        onClick={() => openEdit(selected)}
                      >
                        <Edit3 className='size-3.5' />
                        编辑配置
                      </Button>
                      <Button
                        size='sm'
                        variant='outline'
                        onClick={() => setConfirmAction('rotate')}
                      >
                        <RotateCcw className='size-3.5' />
                        轮换
                      </Button>
                    </div>
                  </div>
                </div>
              )}
            </EnterprisePanel>
          </div>

          <ApiKeyFormDialog
            open={formOpen}
            onOpenChange={(open) => {
              setFormOpen(open)
              if (!open) setEditing(null)
            }}
            editing={editing}
            users={users}
            saving={saveMutation.isPending}
            onSubmit={(input) => saveMutation.mutate(input)}
          />
          <SecretDialog
            secret={secret}
            onOpenChange={(open) => {
              if (!open) setSecret(null)
            }}
          />
          <ConfirmDialog
            open={confirmAction != null}
            onOpenChange={(open) => {
              if (!open) setConfirmAction(null)
            }}
            title={
              confirmAction === 'delete' ? '删除企业密钥？' : '轮换企业密钥？'
            }
            desc={
              confirmAction === 'delete'
                ? `删除后“${selected?.name ?? ''}”将立即失效且无法恢复。`
                : `轮换后“${selected?.name ?? ''}”的旧密钥将立即失效，完整新密钥只显示一次。`
            }
            destructive={confirmAction === 'delete'}
            confirmText={confirmAction === 'delete' ? '确认删除' : '确认轮换'}
            isLoading={rotateMutation.isPending || deleteMutation.isPending}
            handleConfirm={() => {
              if (!selected) return
              if (confirmAction === 'delete') {
                deleteMutation.mutate(selected.id)
              } else {
                rotateMutation.mutate(selected.id)
              }
            }}
          />
          <span className='sr-only'>{t('API Keys')}</span>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
