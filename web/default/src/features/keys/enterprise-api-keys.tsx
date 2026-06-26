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
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import {
  EnterprisePageHeader,
  EnterprisePanel,
  EnterpriseStatCard,
} from '@/components/enterprise'
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
} from './enterprise-types'

const TOKEN_STATUS_ENABLED = 1
const TOKEN_STATUS_DISABLED = 2
const TOKEN_STATUS_EXPIRED = 3
const EMPTY_API_KEY_ITEMS: EnterpriseApiKeyItem[] = []
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
    return <Badge variant='outline'>全部模型</Badge>
  }
  const models = item.model_limits
    .split(',')
    .map((model) => model.trim())
    .filter(Boolean)
  return (
    <div className='flex max-w-56 flex-wrap gap-1'>
      {models.slice(0, 2).map((model) => (
        <Badge key={model} variant='secondary' className='max-w-32 truncate'>
          {model}
        </Badge>
      ))}
      {models.length > 2 && (
        <Badge variant='outline'>+{models.length - 2}</Badge>
      )}
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
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState('all')
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
      page_size: 20,
      keyword: keyword.trim() || undefined,
      status: status === 'all' ? undefined : Number(status),
    }),
    [page, keyword, status]
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

  useEffect(() => {
    if (selected && !items.some((item) => item.id === selected.id)) {
      setSelected(null)
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
            eyebrow='企业客户接入治理'
            title='接口密钥与客户接入'
            description='面向企业客户统一发放、轮换和审计 API Key，并管理额度、模型范围、IP 白名单与路由分组。'
            actions={
              <>
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => void exportKeys()}
                  disabled={exporting || keysQuery.isFetching}
                >
                  <Download className='size-4' />
                  {exporting ? '导出中' : '导出'}
                </Button>
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => keysQuery.refetch()}
                  disabled={keysQuery.isFetching}
                >
                  <RefreshCw
                    className={cn(
                      'size-4',
                      keysQuery.isFetching && 'animate-spin'
                    )}
                  />
                  刷新
                </Button>
                <Button size='sm' onClick={openCreate}>
                  <Plus className='size-4' />
                  创建企业密钥
                </Button>
              </>
            }
          />

          <div className='grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-5'>
            <EnterpriseStatCard
              title='密钥总数'
              value={String(summary?.total ?? 0)}
              helper='全部企业密钥'
              icon={KeyRound}
              tone='blue'
              loading={keysQuery.isLoading}
            />
            <EnterpriseStatCard
              title='活跃密钥'
              value={String(summary?.active ?? 0)}
              helper='可正常调用'
              icon={ShieldCheck}
              tone='emerald'
              loading={keysQuery.isLoading}
            />
            <EnterpriseStatCard
              title='活跃客户'
              value={String(summary?.active_users ?? 0)}
              helper='已分配密钥用户'
              icon={Users}
              tone='violet'
              loading={keysQuery.isLoading}
            />
            <EnterpriseStatCard
              title='即将到期'
              value={String(summary?.expiring_soon ?? 0)}
              helper='未来 7 天'
              icon={CalendarClock}
              tone='amber'
              loading={keysQuery.isLoading}
            />
            <EnterpriseStatCard
              title='额度异常'
              value={String(
                (summary?.exhausted ?? 0) + (summary?.disabled ?? 0)
              )}
              helper='耗尽或禁用'
              icon={AlertTriangle}
              tone='rose'
              loading={keysQuery.isLoading}
            />
          </div>

          <EnterprisePanel bodyClassName='p-3 sm:p-4'>
            <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
              <div className='flex flex-1 flex-col gap-2 sm:flex-row'>
                <div className='relative max-w-md flex-1'>
                  <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2' />
                  <Input
                    value={keyword}
                    className='pl-9'
                    placeholder='搜索客户、邮箱、密钥名称或完整 Key'
                    onChange={(event) => {
                      setKeyword(event.target.value)
                      setPage(1)
                    }}
                  />
                </div>
                <NativeSelect
                  value={status}
                  className='w-full sm:w-40'
                  onChange={(event) => {
                    setStatus(event.target.value)
                    setPage(1)
                  }}
                >
                  <NativeSelectOption value='all'>全部状态</NativeSelectOption>
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
              </div>
              <p className='text-muted-foreground text-xs'>
                共 {pageData?.total ?? 0} 条 · 密钥明文不在列表中返回
              </p>
            </div>
          </EnterprisePanel>

          <div className='grid min-h-0 gap-3 2xl:grid-cols-[minmax(0,1fr)_360px]'>
            <EnterprisePanel bodyClassName='p-0' className='min-w-0'>
              <div className='overflow-x-auto'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>客户 / 租户</TableHead>
                      <TableHead>密钥名称</TableHead>
                      <TableHead>授权模型</TableHead>
                      <TableHead>额度</TableHead>
                      <TableHead>IP 白名单</TableHead>
                      <TableHead>最近使用</TableHead>
                      <TableHead>状态</TableHead>
                      <TableHead className='w-32'>操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.length === 0 ? (
                      <TableRow>
                        <TableCell
                          colSpan={8}
                          className='text-muted-foreground h-40 text-center'
                        >
                          {keysQuery.isLoading
                            ? '正在加载企业密钥…'
                            : '没有符合条件的企业密钥'}
                        </TableCell>
                      </TableRow>
                    ) : (
                      items.map((item) => {
                        const statusConfig = tokenStatus(item.effective_status)
                        const ipCount = item.allow_ips
                          ? item.allow_ips.split('\n').filter(Boolean).length
                          : 0
                        return (
                          <TableRow
                            key={item.id}
                            className={cn(
                              'cursor-pointer',
                              selected?.id === item.id && 'bg-primary/[0.045]'
                            )}
                            onClick={() => setSelected(item)}
                          >
                            <TableCell>
                              <div className='max-w-48'>
                                <p className='truncate font-medium'>
                                  {item.display_name || item.username}
                                </p>
                                <p className='text-muted-foreground truncate text-xs'>
                                  {item.email || item.user_group}
                                </p>
                              </div>
                            </TableCell>
                            <TableCell>
                              <p className='max-w-44 truncate font-medium'>
                                {item.name}
                              </p>
                              <code className='text-muted-foreground text-[11px]'>
                                {item.masked_key}
                              </code>
                            </TableCell>
                            <TableCell>{modelChips(item)}</TableCell>
                            <TableCell>
                              <p className='text-sm font-medium'>
                                {item.unlimited_quota
                                  ? '无限额度'
                                  : formatLogQuota(item.remain_quota)}
                              </p>
                              <p className='text-muted-foreground text-[11px]'>
                                已用 {formatLogQuota(item.used_quota)}
                              </p>
                            </TableCell>
                            <TableCell>
                              {ipCount > 0 ? `${ipCount} 条` : '未限制'}
                            </TableCell>
                            <TableCell>
                              {item.accessed_time > 0
                                ? dayjs.unix(item.accessed_time).fromNow()
                                : '从未使用'}
                            </TableCell>
                            <TableCell>
                              <Badge
                                className={cn(
                                  'border-0',
                                  statusConfig.className
                                )}
                              >
                                {statusConfig.label}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <div className='flex items-center gap-1'>
                                <Button
                                  size='icon-sm'
                                  variant='ghost'
                                  aria-label='编辑密钥'
                                  onClick={(event) => {
                                    event.stopPropagation()
                                    openEdit(item)
                                  }}
                                >
                                  <Edit3 className='size-4' />
                                </Button>
                                <Button
                                  size='icon-sm'
                                  variant='ghost'
                                  aria-label='轮换密钥'
                                  onClick={(event) => {
                                    event.stopPropagation()
                                    setSelected(item)
                                    setConfirmAction('rotate')
                                  }}
                                >
                                  <RotateCcw className='size-4' />
                                </Button>
                                <Button
                                  size='icon-sm'
                                  variant='ghost'
                                  className='text-destructive'
                                  aria-label='删除密钥'
                                  onClick={(event) => {
                                    event.stopPropagation()
                                    setSelected(item)
                                    setConfirmAction('delete')
                                  }}
                                >
                                  <Trash2 className='size-4' />
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
              <div className='flex items-center justify-between border-t px-4 py-3'>
                <Button
                  size='sm'
                  variant='outline'
                  disabled={page <= 1}
                  onClick={() => setPage((value) => Math.max(1, value - 1))}
                >
                  上一页
                </Button>
                <span className='text-muted-foreground text-xs'>
                  第 {page} 页
                </span>
                <Button
                  size='sm'
                  variant='outline'
                  disabled={items.length < 20}
                  onClick={() => setPage((value) => value + 1)}
                >
                  下一页
                </Button>
              </div>
            </EnterprisePanel>

            <EnterprisePanel
              title={selected ? '接入与治理详情' : '选择一条密钥'}
              description={
                selected
                  ? '仅展示脱敏信息与治理配置。'
                  : '从左侧列表选择密钥查看详情。'
              }
              action={
                selected ? (
                  <MoreHorizontal className='text-muted-foreground size-4' />
                ) : null
              }
            >
              {!selected ? (
                <div className='flex min-h-72 flex-col items-center justify-center text-center'>
                  <span className='bg-primary/10 text-primary flex size-12 items-center justify-center rounded-md'>
                    <KeyRound className='size-5' />
                  </span>
                  <p className='mt-3 text-sm font-medium'>企业密钥治理</p>
                  <p className='text-muted-foreground mt-1 max-w-64 text-xs leading-5'>
                    查看 Base URL、路由分组、模型范围、到期时间和最近使用情况。
                  </p>
                </div>
              ) : (
                <div className='space-y-3'>
                  <div>
                    <p className='text-muted-foreground text-xs'>客户 / 用户</p>
                    <p className='mt-1 font-semibold'>
                      {selected.display_name || selected.username}
                    </p>
                    <p className='text-muted-foreground text-xs'>
                      {selected.email}
                    </p>
                  </div>
                  <div className='bg-muted/25 rounded-md border p-3'>
                    <p className='text-muted-foreground text-[11px]'>
                      Base URL
                    </p>
                    <div className='mt-1 flex items-center gap-2'>
                      <code className='min-w-0 flex-1 truncate text-xs'>
                        /v1
                      </code>
                      <Button
                        size='icon-sm'
                        variant='ghost'
                        onClick={() =>
                          navigator.clipboard.writeText(
                            `${window.location.origin}/v1`
                          )
                        }
                      >
                        <Copy className='size-3.5' />
                      </Button>
                    </div>
                  </div>
                  <div className='grid grid-cols-2 gap-3 text-xs'>
                    <div className='rounded-md border p-3'>
                      <p className='text-muted-foreground'>路由分组</p>
                      <p className='mt-1 font-medium'>
                        {selected.group || '继承用户'}
                      </p>
                    </div>
                    <div className='rounded-md border p-3'>
                      <p className='text-muted-foreground'>到期时间</p>
                      <p className='mt-1 font-medium'>
                        {selected.expired_time > 0
                          ? dayjs
                              .unix(selected.expired_time)
                              .format('YYYY-MM-DD')
                          : '永不过期'}
                      </p>
                    </div>
                    <div className='rounded-md border p-3'>
                      <p className='text-muted-foreground'>IP 白名单</p>
                      <p className='mt-1 font-medium'>
                        {selected.allow_ips
                          ? `${selected.allow_ips.split('\n').filter(Boolean).length} 条规则`
                          : '未限制'}
                      </p>
                    </div>
                    <div className='rounded-md border p-3'>
                      <p className='text-muted-foreground'>跨组重试</p>
                      <p className='mt-1 font-medium'>
                        {selected.cross_group_retry ? '允许' : '不允许'}
                      </p>
                    </div>
                  </div>
                  <div>
                    <p className='text-xs font-medium'>授权模型</p>
                    <div className='mt-2'>{modelChips(selected)}</div>
                  </div>
                  <div className='flex gap-2 border-t pt-4'>
                    <Button
                      size='sm'
                      className='flex-1'
                      onClick={() => openEdit(selected)}
                    >
                      <Edit3 className='size-4' />
                      编辑配置
                    </Button>
                    <Button
                      size='sm'
                      variant='outline'
                      onClick={() => setConfirmAction('rotate')}
                    >
                      <RotateCcw className='size-4' />
                      轮换
                    </Button>
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
            users={usersQuery.data?.data ?? []}
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
