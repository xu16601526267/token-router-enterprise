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
  Activity,
  AlertTriangle,
  Boxes,
  Building2,
  CreditCard,
  FileCheck2,
  KeyRound,
  Plus,
  RefreshCw,
  Search,
  Settings2,
  ShieldCheck,
  SlidersHorizontal,
  UserRound,
  Users,
  WalletCards,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'

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
import dayjs from '@/lib/dayjs'
import { formatNumber, formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

import {
  createPlatformTenant,
  getPlatformTenant360,
  getPlatformTenantModelPolicies,
  getPlatformTenants,
  updatePlatformTenantBillingConfig,
  updatePlatformTenantStatus,
  upsertPlatformTenantModelPolicy,
} from './api'
import type {
  PlatformTenant,
  PlatformTenant360,
  PlatformTenantBillingConfigInput,
  PlatformTenantCreateInput,
  TenantBillingMode,
  TenantModelPolicy,
  TenantModelPolicyInput,
  TenantStatus,
} from './types'

const EMPTY_TENANTS: PlatformTenant[] = []
const EMPTY_POLICIES: TenantModelPolicy[] = []

const statusOptions = [
  { value: 'active', label: '正常' },
  { value: 'suspended', label: '暂停' },
  { value: 'disabled', label: '禁用' },
] as const

const billingModeOptions = [
  { value: 'prepaid', label: '预付费' },
  { value: 'postpaid', label: '后付费' },
  { value: 'mixed', label: '混合计费' },
] as const

const defaultCreateForm = {
  name: '',
  type: 'enterprise',
  industry: '',
  ownerUserId: '',
  domain: '',
  contractNo: '',
  billingMode: 'postpaid' as TenantBillingMode,
  creditLimit: '50000000',
  statementDay: '1',
  paymentTerms: '30',
}

type CreateTenantFormState = typeof defaultCreateForm

const defaultBillingForm = {
  billingMode: 'postpaid' as TenantBillingMode,
  billingCycle: 'monthly',
  statementDay: '1',
  paymentTerms: '30',
  creditLimit: '50000000',
  overCreditPolicy: 'block',
}

type BillingFormState = typeof defaultBillingForm

const defaultPolicyForm = {
  modelName: '',
  alias: '',
  rateLimit: '',
  modelId: '',
  pricePlanId: '',
  visible: true,
  enabled: true,
}

type ModelPolicyFormState = typeof defaultPolicyForm

function toInt(value: string, fallback = 0): number {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return fallback
  return Math.max(0, Math.trunc(parsed))
}

function formatDate(timestamp: number): string {
  if (!timestamp) return '-'
  return dayjs.unix(timestamp).format('YYYY-MM-DD')
}

function statusMeta(status: string): { label: string; className: string } {
  if (status === 'active') {
    return {
      label: '正常',
      className: 'bg-emerald-500/10 text-emerald-700',
    }
  }
  if (status === 'suspended') {
    return {
      label: '暂停',
      className: 'bg-amber-500/10 text-amber-700',
    }
  }
  return {
    label: '禁用',
    className: 'bg-slate-500/10 text-slate-600',
  }
}

function billingModeLabel(mode?: string): string {
  return (
    billingModeOptions.find((item) => item.value === mode)?.label ?? '未配置'
  )
}

function overCreditPolicyLabel(policy?: string): string {
  if (policy === 'warn') return '预警放行'
  if (policy === 'allow') return '允许超额'
  return '超额拦截'
}

function quotaText(value?: number | null): string {
  return formatQuota(value ?? 0)
}

function TenantStatusBadge({ status }: { status: string }) {
  const meta = statusMeta(status)
  return (
    <Badge variant='outline' className={cn('border-0', meta.className)}>
      {meta.label}
    </Badge>
  )
}

function CompactMetric({
  label,
  value,
  helper,
}: {
  label: string
  value: string
  helper?: string
}) {
  return (
    <div className='bg-background rounded-md border px-3 py-2'>
      <p className='text-muted-foreground text-[11px]'>{label}</p>
      <p className='mt-1 truncate text-sm font-semibold tabular-nums'>
        {value}
      </p>
      {helper != null && (
        <p className='text-muted-foreground mt-0.5 truncate text-[11px]'>
          {helper}
        </p>
      )}
    </div>
  )
}

function CreateTenantDialog({
  open,
  onOpenChange,
  saving,
  onSubmit,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  saving: boolean
  onSubmit: (payload: PlatformTenantCreateInput) => void
}) {
  const [form, setForm] = useState<CreateTenantFormState>(defaultCreateForm)

  useEffect(() => {
    if (open) setForm(defaultCreateForm)
  }, [open])

  const update = <K extends keyof CreateTenantFormState>(
    key: K,
    value: CreateTenantFormState[K]
  ) => setForm((current) => ({ ...current, [key]: value }))

  const submit = () => {
    const name = form.name.trim()
    if (!name) {
      toast.error('请输入客户名称')
      return
    }
    onSubmit({
      name,
      type: form.type.trim() || 'enterprise',
      industry: form.industry.trim(),
      owner_user_id: toInt(form.ownerUserId),
      domain: form.domain.trim(),
      contract_no: form.contractNo.trim(),
      billing_mode: form.billingMode,
      credit_limit: toInt(form.creditLimit),
      statement_day: toInt(form.statementDay, 1),
      payment_terms: toInt(form.paymentTerms, 30),
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[88vh] overflow-y-auto rounded-md sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>新建 B 端客户</DialogTitle>
          <DialogDescription className='text-xs leading-5'>
            同步建立租户、默认结算配置和授信账户。负责人用户 ID 可后续补充。
          </DialogDescription>
        </DialogHeader>

        <div className='grid gap-3 sm:grid-cols-2'>
          <Field className='gap-1.5 sm:col-span-2'>
            <FieldLabel className='text-xs font-medium'>客户名称 *</FieldLabel>
            <Input
              value={form.name}
              placeholder='例如：华东事业群 / 某某科技'
              onChange={(event) => update('name', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>客户类型</FieldLabel>
            <Input
              value={form.type}
              onChange={(event) => update('type', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>行业</FieldLabel>
            <Input
              value={form.industry}
              placeholder='AI 应用 / 教育 / 金融'
              onChange={(event) => update('industry', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>
              负责人用户 ID
            </FieldLabel>
            <Input
              value={form.ownerUserId}
              inputMode='numeric'
              onChange={(event) => update('ownerUserId', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>绑定域名</FieldLabel>
            <Input
              value={form.domain}
              placeholder='customer.example.com'
              onChange={(event) => update('domain', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>合同编号</FieldLabel>
            <Input
              value={form.contractNo}
              onChange={(event) => update('contractNo', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>结算模式</FieldLabel>
            <NativeSelect
              className='w-full'
              value={form.billingMode}
              onChange={(event) =>
                update('billingMode', event.target.value as TenantBillingMode)
              }
            >
              {billingModeOptions.map((item) => (
                <NativeSelectOption key={item.value} value={item.value}>
                  {item.label}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>授信额度</FieldLabel>
            <Input
              value={form.creditLimit}
              inputMode='numeric'
              onChange={(event) => update('creditLimit', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>出账日</FieldLabel>
            <Input
              value={form.statementDay}
              inputMode='numeric'
              onChange={(event) => update('statementDay', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>账期天数</FieldLabel>
            <Input
              value={form.paymentTerms}
              inputMode='numeric'
              onChange={(event) => update('paymentTerms', event.target.value)}
            />
          </Field>
        </div>

        <DialogFooter className='rounded-b-md'>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={saving}
          >
            取消
          </Button>
          <Button onClick={submit} disabled={saving}>
            {saving ? '创建中' : '创建客户'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function BillingConfigDialog({
  open,
  tenant,
  detail,
  saving,
  onOpenChange,
  onSubmit,
}: {
  open: boolean
  tenant: PlatformTenant | null
  detail: PlatformTenant360 | undefined
  saving: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (payload: PlatformTenantBillingConfigInput) => void
}) {
  const [form, setForm] = useState<BillingFormState>(defaultBillingForm)

  useEffect(() => {
    if (!open) return
    const config = detail?.billing_config
    const account = detail?.credit_account
    setForm({
      billingMode:
        (config?.billing_mode as TenantBillingMode | undefined) ??
        defaultBillingForm.billingMode,
      billingCycle: config?.billing_cycle || defaultBillingForm.billingCycle,
      statementDay: String(
        config?.statement_day || defaultBillingForm.statementDay
      ),
      paymentTerms: String(
        config?.payment_terms || defaultBillingForm.paymentTerms
      ),
      creditLimit: String(
        config?.credit_limit ??
          account?.credit_limit ??
          defaultBillingForm.creditLimit
      ),
      overCreditPolicy:
        config?.over_credit_policy || defaultBillingForm.overCreditPolicy,
    })
  }, [detail, open])

  const update = <K extends keyof BillingFormState>(
    key: K,
    value: BillingFormState[K]
  ) => setForm((current) => ({ ...current, [key]: value }))

  const submit = () => {
    onSubmit({
      billing_mode: form.billingMode,
      billing_cycle: form.billingCycle.trim() || 'monthly',
      statement_day: toInt(form.statementDay, 1),
      payment_terms: toInt(form.paymentTerms, 30),
      credit_limit: toInt(form.creditLimit),
      over_credit_policy: form.overCreditPolicy.trim() || 'block',
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[88vh] overflow-y-auto rounded-md sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>结算与授信配置</DialogTitle>
          <DialogDescription className='text-xs leading-5'>
            {tenant?.name ?? '当前客户'} 的计费模式、账期、授信额度和超额策略。
          </DialogDescription>
        </DialogHeader>

        <div className='grid gap-3 sm:grid-cols-2'>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>结算模式</FieldLabel>
            <NativeSelect
              className='w-full'
              value={form.billingMode}
              onChange={(event) =>
                update('billingMode', event.target.value as TenantBillingMode)
              }
            >
              {billingModeOptions.map((item) => (
                <NativeSelectOption key={item.value} value={item.value}>
                  {item.label}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>结算周期</FieldLabel>
            <NativeSelect
              className='w-full'
              value={form.billingCycle}
              onChange={(event) => update('billingCycle', event.target.value)}
            >
              <NativeSelectOption value='monthly'>月结</NativeSelectOption>
              <NativeSelectOption value='weekly'>周结</NativeSelectOption>
              <NativeSelectOption value='manual'>手工出账</NativeSelectOption>
            </NativeSelect>
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>授信额度</FieldLabel>
            <Input
              value={form.creditLimit}
              inputMode='numeric'
              onChange={(event) => update('creditLimit', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>超额策略</FieldLabel>
            <NativeSelect
              className='w-full'
              value={form.overCreditPolicy}
              onChange={(event) =>
                update('overCreditPolicy', event.target.value)
              }
            >
              <NativeSelectOption value='block'>超额拦截</NativeSelectOption>
              <NativeSelectOption value='warn'>预警放行</NativeSelectOption>
              <NativeSelectOption value='allow'>允许超额</NativeSelectOption>
            </NativeSelect>
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>出账日</FieldLabel>
            <Input
              value={form.statementDay}
              inputMode='numeric'
              onChange={(event) => update('statementDay', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>账期天数</FieldLabel>
            <Input
              value={form.paymentTerms}
              inputMode='numeric'
              onChange={(event) => update('paymentTerms', event.target.value)}
            />
          </Field>
        </div>

        <DialogFooter className='rounded-b-md'>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={saving}
          >
            取消
          </Button>
          <Button onClick={submit} disabled={saving}>
            {saving ? '保存中' : '保存配置'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function ModelPolicyDialog({
  open,
  tenant,
  saving,
  onOpenChange,
  onSubmit,
}: {
  open: boolean
  tenant: PlatformTenant | null
  saving: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (payload: TenantModelPolicyInput) => void
}) {
  const [form, setForm] = useState<ModelPolicyFormState>(defaultPolicyForm)

  useEffect(() => {
    if (open) setForm(defaultPolicyForm)
  }, [open])

  const update = <K extends keyof ModelPolicyFormState>(
    key: K,
    value: ModelPolicyFormState[K]
  ) => setForm((current) => ({ ...current, [key]: value }))

  const submit = () => {
    const modelName = form.modelName.trim()
    if (!modelName) {
      toast.error('请输入模型名称')
      return
    }
    onSubmit({
      model_name: modelName,
      alias: form.alias.trim(),
      rate_limit: form.rateLimit.trim(),
      model_id: toInt(form.modelId),
      price_plan_id: toInt(form.pricePlanId),
      visible: form.visible,
      enabled: form.enabled,
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-h-[88vh] overflow-y-auto rounded-md sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>添加模型授权</DialogTitle>
          <DialogDescription className='text-xs leading-5'>
            {tenant?.name ?? '当前客户'} 可见、可用的模型范围和限速策略。
          </DialogDescription>
        </DialogHeader>

        <div className='grid gap-3 sm:grid-cols-2'>
          <Field className='gap-1.5 sm:col-span-2'>
            <FieldLabel className='text-xs font-medium'>模型名称 *</FieldLabel>
            <Input
              value={form.modelName}
              placeholder='例如：gpt-4o-mini'
              onChange={(event) => update('modelName', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>显示别名</FieldLabel>
            <Input
              value={form.alias}
              placeholder='客户侧展示名称'
              onChange={(event) => update('alias', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>限速策略</FieldLabel>
            <Input
              value={form.rateLimit}
              placeholder='例如：1000 rpm'
              onChange={(event) => update('rateLimit', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>模型 ID</FieldLabel>
            <Input
              value={form.modelId}
              inputMode='numeric'
              onChange={(event) => update('modelId', event.target.value)}
            />
          </Field>
          <Field className='gap-1.5'>
            <FieldLabel className='text-xs font-medium'>价格方案 ID</FieldLabel>
            <Input
              value={form.pricePlanId}
              inputMode='numeric'
              onChange={(event) => update('pricePlanId', event.target.value)}
            />
          </Field>
          <div className='rounded-md border px-3 py-2'>
            <div className='flex items-center justify-between gap-3'>
              <div>
                <p className='text-xs font-medium'>客户侧可见</p>
                <p className='text-muted-foreground text-[11px]'>
                  控制模型目录展示。
                </p>
              </div>
              <Switch
                checked={form.visible}
                onCheckedChange={(checked) => update('visible', checked)}
              />
            </div>
          </div>
          <div className='rounded-md border px-3 py-2'>
            <div className='flex items-center justify-between gap-3'>
              <div>
                <p className='text-xs font-medium'>允许调用</p>
                <p className='text-muted-foreground text-[11px]'>
                  关闭后会从调用链路拦截。
                </p>
              </div>
              <Switch
                checked={form.enabled}
                onCheckedChange={(checked) => update('enabled', checked)}
              />
            </div>
          </div>
        </div>

        <DialogFooter className='rounded-b-md'>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={saving}
          >
            取消
          </Button>
          <Button onClick={submit} disabled={saving}>
            {saving ? '保存中' : '保存授权'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function TenantDetailPanel({
  tenant,
  detail,
  policies,
  loading,
  policyLoading,
  statusSaving,
  onOpenBilling,
  onOpenPolicy,
  onUpdateStatus,
}: {
  tenant: PlatformTenant | null
  detail: PlatformTenant360 | undefined
  policies: TenantModelPolicy[]
  loading: boolean
  policyLoading: boolean
  statusSaving: boolean
  onOpenBilling: () => void
  onOpenPolicy: () => void
  onUpdateStatus: (status: TenantStatus) => void
}) {
  if (!tenant) {
    return (
      <EnterprisePanel
        title='客户 360'
        description='选择一个 B 端客户查看账户、账务、授权和下游规模。'
      >
        <div className='flex min-h-80 flex-col items-center justify-center text-center'>
          <span className='bg-primary/10 text-primary flex size-12 items-center justify-center rounded-md'>
            <Building2 className='size-5' />
          </span>
          <p className='mt-3 text-sm font-medium'>暂无客户数据</p>
          <p className='text-muted-foreground mt-1 max-w-64 text-xs leading-5'>
            新建客户后，可在这里完成状态治理、授信配置和模型授权。
          </p>
        </div>
      </EnterprisePanel>
    )
  }

  const config = detail?.billing_config
  const account = detail?.credit_account
  const status = statusMeta(tenant.status)

  return (
    <EnterprisePanel
      title='客户 360'
      description='账户、下游、密钥、授信和模型授权的聚合视图。'
      action={
        <div className='flex items-center gap-1.5'>
          <Button size='sm' variant='outline' onClick={onOpenBilling}>
            <CreditCard className='size-4' />
            结算
          </Button>
          <Button size='sm' onClick={onOpenPolicy}>
            <Boxes className='size-4' />
            模型
          </Button>
        </div>
      }
    >
      <div className='space-y-3'>
        <div className='bg-background rounded-md border p-3'>
          <div className='flex items-start justify-between gap-3'>
            <div className='min-w-0'>
              <p className='truncate text-sm font-semibold'>{tenant.name}</p>
              <p className='text-muted-foreground mt-1 truncate text-xs'>
                {tenant.industry || '未填写行业'} · ID {tenant.id}
              </p>
            </div>
            <Badge
              variant='outline'
              className={cn('border-0', status.className)}
            >
              {status.label}
            </Badge>
          </div>
          <div className='mt-3 grid grid-cols-2 gap-2 text-xs'>
            <CompactMetric
              label='负责人'
              value={tenant.owner_user_id ? String(tenant.owner_user_id) : '-'}
              helper='Owner User ID'
            />
            <CompactMetric
              label='合同'
              value={tenant.contract_no || '-'}
              helper={formatDate(tenant.created_at)}
            />
          </div>
        </div>

        <div className='grid grid-cols-2 gap-2'>
          <CompactMetric
            label='成员'
            value={loading ? '...' : formatNumber(detail?.members ?? 0)}
            helper='B 端员工'
          />
          <CompactMetric
            label='C端客户'
            value={loading ? '...' : formatNumber(detail?.end_customers ?? 0)}
            helper='下游终端'
          />
          <CompactMetric
            label='应用'
            value={loading ? '...' : formatNumber(detail?.apps ?? 0)}
            helper='业务应用'
          />
          <CompactMetric
            label='API Key'
            value={loading ? '...' : formatNumber(detail?.api_keys ?? 0)}
            helper='可调用密钥'
          />
        </div>

        <div className='bg-background rounded-md border p-3'>
          <div className='flex items-center justify-between gap-3'>
            <div>
              <p className='text-xs font-semibold'>结算与授信</p>
              <p className='text-muted-foreground mt-0.5 text-[11px]'>
                {billingModeLabel(config?.billing_mode)} ·{' '}
                {overCreditPolicyLabel(config?.over_credit_policy)}
              </p>
            </div>
            <WalletCards className='text-muted-foreground size-4' />
          </div>
          <div className='mt-3 grid grid-cols-2 gap-2'>
            <CompactMetric
              label='授信额度'
              value={quotaText(account?.credit_limit ?? config?.credit_limit)}
            />
            <CompactMetric
              label='可用授信'
              value={quotaText(account?.available_credit)}
            />
            <CompactMetric
              label='未出账'
              value={quotaText(account?.unbilled_amount)}
            />
            <CompactMetric
              label='逾期'
              value={quotaText(account?.overdue_amount)}
            />
          </div>
          <p className='text-muted-foreground mt-2 text-[11px]'>
            出账日 {config?.statement_day ?? '-'} 日 · 账期{' '}
            {config?.payment_terms ?? '-'} 天
          </p>
        </div>

        <div className='bg-background rounded-md border p-3'>
          <div className='flex items-center justify-between gap-2'>
            <div>
              <p className='text-xs font-semibold'>模型授权</p>
              <p className='text-muted-foreground mt-0.5 text-[11px]'>
                {policyLoading ? '正在加载' : `${policies.length} 条授权策略`}
              </p>
            </div>
            <Button size='sm' variant='outline' onClick={onOpenPolicy}>
              新增
            </Button>
          </div>
          <div className='mt-2 max-h-48 space-y-2 overflow-auto pr-1'>
            {policies.length === 0 ? (
              <div className='text-muted-foreground rounded-md border border-dashed px-3 py-6 text-center text-xs'>
                {policyLoading ? '正在加载模型授权…' : '尚未配置模型授权'}
              </div>
            ) : (
              policies.slice(0, 6).map((policy) => (
                <div
                  key={policy.id}
                  className='flex items-center justify-between gap-3 rounded-md border px-2.5 py-2'
                >
                  <div className='min-w-0'>
                    <p className='truncate text-xs font-medium'>
                      {policy.alias || policy.model_name}
                    </p>
                    <p className='text-muted-foreground truncate text-[11px]'>
                      {policy.model_name} · {policy.rate_limit || '不限速'}
                    </p>
                  </div>
                  <Badge
                    variant='outline'
                    className={cn(
                      'border-0',
                      policy.enabled && policy.visible
                        ? 'bg-emerald-500/10 text-emerald-700'
                        : 'bg-slate-500/10 text-slate-600'
                    )}
                  >
                    {policy.enabled && policy.visible ? '可用' : '受限'}
                  </Badge>
                </div>
              ))
            )}
          </div>
        </div>

        <div className='grid grid-cols-3 gap-2 border-t pt-3'>
          {statusOptions.map((item) => (
            <Button
              key={item.value}
              size='sm'
              variant={tenant.status === item.value ? 'default' : 'outline'}
              disabled={statusSaving || tenant.status === item.value}
              onClick={() => onUpdateStatus(item.value)}
            >
              {item.label}
            </Button>
          ))}
        </div>
      </div>
    </EnterprisePanel>
  )
}

export function EnterpriseTenantCenter() {
  const queryClient = useQueryClient()
  const [keyword, setKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | TenantStatus>('all')
  const [selectedTenantId, setSelectedTenantId] = useState<number | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [billingOpen, setBillingOpen] = useState(false)
  const [policyOpen, setPolicyOpen] = useState(false)

  const tenantsQuery = useQuery({
    queryKey: ['platform-tenants', statusFilter],
    queryFn: () =>
      getPlatformTenants(
        statusFilter === 'all' ? {} : { status: statusFilter }
      ),
  })

  const tenants = tenantsQuery.data?.data ?? EMPTY_TENANTS

  useEffect(() => {
    if (tenants.length === 0) {
      setSelectedTenantId(null)
      return
    }
    if (
      selectedTenantId == null ||
      !tenants.some((tenant) => tenant.id === selectedTenantId)
    ) {
      setSelectedTenantId(tenants[0].id)
    }
  }, [selectedTenantId, tenants])

  const detailQuery = useQuery({
    queryKey: ['platform-tenant-360', selectedTenantId],
    enabled: selectedTenantId != null,
    queryFn: () => getPlatformTenant360(selectedTenantId ?? 0),
  })

  const policiesQuery = useQuery({
    queryKey: ['platform-tenant-model-policies', selectedTenantId],
    enabled: selectedTenantId != null,
    queryFn: () => getPlatformTenantModelPolicies(selectedTenantId ?? 0),
  })

  const selectedTenant =
    detailQuery.data?.data?.tenant ??
    tenants.find((tenant) => tenant.id === selectedTenantId) ??
    null
  const selectedDetail = detailQuery.data?.data
  const policies = policiesQuery.data?.data ?? EMPTY_POLICIES

  const filteredTenants = useMemo(() => {
    const value = keyword.trim().toLowerCase()
    if (!value) return tenants
    return tenants.filter((tenant) =>
      [
        tenant.name,
        tenant.industry,
        tenant.domain,
        tenant.contract_no,
        String(tenant.id),
        String(tenant.owner_user_id),
      ]
        .filter(Boolean)
        .some((field) => field.toLowerCase().includes(value))
    )
  }, [keyword, tenants])

  const summary = useMemo(() => {
    const active = tenants.filter((tenant) => tenant.status === 'active').length
    const suspended = tenants.filter(
      (tenant) => tenant.status === 'suspended'
    ).length
    const disabled = tenants.filter(
      (tenant) => tenant.status === 'disabled'
    ).length
    return { active, suspended, disabled }
  }, [tenants])

  const invalidateTenantData = async (tenantId?: number | null) => {
    await queryClient.invalidateQueries({ queryKey: ['platform-tenants'] })
    if (tenantId != null) {
      await queryClient.invalidateQueries({
        queryKey: ['platform-tenant-360', tenantId],
      })
      await queryClient.invalidateQueries({
        queryKey: ['platform-tenant-model-policies', tenantId],
      })
    }
  }

  const createMutation = useMutation({
    mutationFn: (payload: PlatformTenantCreateInput) =>
      createPlatformTenant(payload),
    onSuccess: async (response) => {
      if (!response.success || !response.data) return
      toast.success('B 端客户已创建')
      setCreateOpen(false)
      setSelectedTenantId(response.data.id)
      await invalidateTenantData(response.data.id)
    },
  })

  const statusMutation = useMutation({
    mutationFn: ({
      tenantId,
      status,
    }: {
      tenantId: number
      status: TenantStatus
    }) => updatePlatformTenantStatus(tenantId, status),
    onSuccess: async (response, variables) => {
      if (!response.success) return
      toast.success('客户状态已更新')
      await invalidateTenantData(variables.tenantId)
    },
  })

  const billingMutation = useMutation({
    mutationFn: ({
      tenantId,
      payload,
    }: {
      tenantId: number
      payload: PlatformTenantBillingConfigInput
    }) => updatePlatformTenantBillingConfig(tenantId, payload),
    onSuccess: async (response, variables) => {
      if (!response.success) return
      toast.success('结算配置已保存')
      setBillingOpen(false)
      await invalidateTenantData(variables.tenantId)
    },
  })

  const policyMutation = useMutation({
    mutationFn: ({
      tenantId,
      payload,
    }: {
      tenantId: number
      payload: TenantModelPolicyInput
    }) => upsertPlatformTenantModelPolicy(tenantId, payload),
    onSuccess: async (response, variables) => {
      if (!response.success) return
      toast.success('模型授权已保存')
      setPolicyOpen(false)
      await invalidateTenantData(variables.tenantId)
    },
  })

  const refresh = () => {
    void invalidateTenantData(selectedTenantId)
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Content>
        <div className='enterprise-dashboard flex h-full min-h-0 flex-col gap-3 overflow-auto pb-5'>
          <EnterprisePageHeader
            eyebrow='平台方租户经营'
            title='B端客户 / 租户中心'
            description='按企业客户维护租户、授信、账期、模型授权和下游规模，支撑 A-B-C 多租户调用链路。'
            actions={
              <>
                <Button
                  variant='outline'
                  size='sm'
                  onClick={refresh}
                  disabled={tenantsQuery.isFetching || detailQuery.isFetching}
                >
                  <RefreshCw
                    className={cn(
                      'size-4',
                      (tenantsQuery.isFetching || detailQuery.isFetching) &&
                        'animate-spin'
                    )}
                  />
                  刷新
                </Button>
                <Button size='sm' onClick={() => setCreateOpen(true)}>
                  <Plus className='size-4' />
                  新建客户
                </Button>
              </>
            }
          />

          <div className='grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-5'>
            <EnterpriseStatCard
              title='客户数'
              value={String(tenants.length)}
              helper='当前筛选'
              icon={Building2}
              tone='blue'
              loading={tenantsQuery.isLoading}
            />
            <EnterpriseStatCard
              title='正常客户'
              value={String(summary.active)}
              helper='可正常调用'
              icon={ShieldCheck}
              tone='emerald'
              loading={tenantsQuery.isLoading}
            />
            <EnterpriseStatCard
              title='暂停 / 禁用'
              value={String(summary.suspended + summary.disabled)}
              helper='需运营处理'
              icon={AlertTriangle}
              tone='amber'
              loading={tenantsQuery.isLoading}
            />
            <EnterpriseStatCard
              title='当前客户 C端'
              value={String(selectedDetail?.end_customers ?? 0)}
              helper='下游终端客户'
              icon={Users}
              tone='violet'
              loading={detailQuery.isLoading}
            />
            <EnterpriseStatCard
              title='当前客户 Key'
              value={String(selectedDetail?.api_keys ?? 0)}
              helper='可调用密钥'
              icon={KeyRound}
              tone='slate'
              loading={detailQuery.isLoading}
            />
          </div>

          <EnterprisePanel bodyClassName='p-3'>
            <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
              <div className='flex flex-1 flex-col gap-2 sm:flex-row'>
                <div className='relative max-w-lg flex-1'>
                  <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2' />
                  <Input
                    value={keyword}
                    className='pl-9'
                    placeholder='搜索客户、行业、域名、合同、负责人 ID'
                    onChange={(event) => setKeyword(event.target.value)}
                  />
                </div>
                <NativeSelect
                  className='w-full sm:w-40'
                  value={statusFilter}
                  onChange={(event) =>
                    setStatusFilter(event.target.value as 'all' | TenantStatus)
                  }
                >
                  <NativeSelectOption value='all'>全部状态</NativeSelectOption>
                  {statusOptions.map((item) => (
                    <NativeSelectOption key={item.value} value={item.value}>
                      {item.label}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </div>
              <div className='text-muted-foreground flex items-center gap-2 text-xs'>
                <SlidersHorizontal className='size-4' />
                <span>
                  已筛选 {filteredTenants.length} / {tenants.length} 个客户
                </span>
              </div>
            </div>
          </EnterprisePanel>

          <div className='grid min-h-0 gap-3 2xl:grid-cols-[minmax(0,1fr)_390px]'>
            <EnterprisePanel
              title='B端客户列表'
              description='平台方维护企业客户、合同、负责人和租户状态。'
              bodyClassName='p-0'
              className='min-w-0'
              action={
                <Button
                  size='sm'
                  variant='outline'
                  onClick={() => setCreateOpen(true)}
                >
                  <Plus className='size-4' />
                  新建
                </Button>
              }
            >
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>客户</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>类型 / 行业</TableHead>
                    <TableHead>域名</TableHead>
                    <TableHead>合同</TableHead>
                    <TableHead>负责人</TableHead>
                    <TableHead>创建时间</TableHead>
                    <TableHead className='w-24'>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredTenants.length === 0 ? (
                    <TableRow>
                      <TableCell
                        colSpan={8}
                        className='text-muted-foreground h-44 text-center'
                      >
                        {tenantsQuery.isLoading
                          ? '正在加载 B 端客户…'
                          : '没有符合条件的 B 端客户'}
                      </TableCell>
                    </TableRow>
                  ) : (
                    filteredTenants.map((tenant) => (
                      <TableRow
                        key={tenant.id}
                        className={cn(
                          'cursor-pointer',
                          tenant.id === selectedTenantId && 'bg-primary/[0.05]'
                        )}
                        onClick={() => setSelectedTenantId(tenant.id)}
                      >
                        <TableCell>
                          <div className='max-w-56'>
                            <p className='truncate font-medium'>
                              {tenant.name}
                            </p>
                            <p className='text-muted-foreground truncate text-[11px]'>
                              ID {tenant.id} · {tenant.domain || '未绑定域名'}
                            </p>
                          </div>
                        </TableCell>
                        <TableCell>
                          <TenantStatusBadge status={tenant.status} />
                        </TableCell>
                        <TableCell>
                          <div className='max-w-40'>
                            <p className='truncate'>{tenant.type || '-'}</p>
                            <p className='text-muted-foreground truncate text-[11px]'>
                              {tenant.industry || '未填写行业'}
                            </p>
                          </div>
                        </TableCell>
                        <TableCell>
                          <p className='max-w-44 truncate'>
                            {tenant.domain || '-'}
                          </p>
                        </TableCell>
                        <TableCell>
                          <p className='max-w-36 truncate'>
                            {tenant.contract_no || '-'}
                          </p>
                        </TableCell>
                        <TableCell>
                          {tenant.owner_user_id > 0 ? (
                            <span className='inline-flex items-center gap-1.5'>
                              <UserRound className='text-muted-foreground size-3.5' />
                              {tenant.owner_user_id}
                            </span>
                          ) : (
                            '-'
                          )}
                        </TableCell>
                        <TableCell>{formatDate(tenant.created_at)}</TableCell>
                        <TableCell>
                          <Button
                            size='sm'
                            variant='outline'
                            onClick={(event) => {
                              event.stopPropagation()
                              setSelectedTenantId(tenant.id)
                              setBillingOpen(true)
                            }}
                          >
                            结算
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
              <div className='text-muted-foreground flex items-center justify-between border-t px-4 py-3 text-xs'>
                <span>列表按创建时间倒序；点击行查看客户 360。</span>
                <span>{tenantsQuery.isFetching ? '同步中…' : '已同步'}</span>
              </div>
            </EnterprisePanel>

            <TenantDetailPanel
              tenant={selectedTenant}
              detail={selectedDetail}
              policies={policies}
              loading={detailQuery.isLoading}
              policyLoading={policiesQuery.isLoading}
              statusSaving={statusMutation.isPending}
              onOpenBilling={() => setBillingOpen(true)}
              onOpenPolicy={() => setPolicyOpen(true)}
              onUpdateStatus={(status) => {
                if (!selectedTenantId) return
                statusMutation.mutate({ tenantId: selectedTenantId, status })
              }}
            />
          </div>

          <div className='grid gap-3 xl:grid-cols-3'>
            <EnterprisePanel
              title='多租户链路'
              description='A 平台方为多个 B 端客户开租户，B 下再挂 C 端用户和 Key。'
            >
              <div className='space-y-2 text-xs'>
                <div className='bg-background flex items-center gap-2 rounded-md border px-3 py-2'>
                  <Building2 className='text-primary size-4' />
                  <span className='font-medium'>A 平台方</span>
                  <span className='text-muted-foreground ml-auto'>
                    统一渠道与结算
                  </span>
                </div>
                <div className='bg-background flex items-center gap-2 rounded-md border px-3 py-2'>
                  <FileCheck2 className='size-4 text-emerald-600' />
                  <span className='font-medium'>B 企业客户</span>
                  <span className='text-muted-foreground ml-auto'>
                    租户、账期、模型策略
                  </span>
                </div>
                <div className='bg-background flex items-center gap-2 rounded-md border px-3 py-2'>
                  <KeyRound className='size-4 text-violet-600' />
                  <span className='font-medium'>C 终端 / Key</span>
                  <span className='text-muted-foreground ml-auto'>
                    调用、用量、对账
                  </span>
                </div>
              </div>
            </EnterprisePanel>
            <EnterprisePanel
              title='账务口径'
              description='客户 360 中展示授信、未出账、未收款和逾期金额。'
            >
              <div className='grid grid-cols-2 gap-2'>
                <CompactMetric
                  label='请求流水'
                  value={formatNumber(selectedDetail?.usage_ledger_count ?? 0)}
                  helper='usage ledger'
                />
                <CompactMetric
                  label='结算模式'
                  value={billingModeLabel(
                    selectedDetail?.billing_config?.billing_mode
                  )}
                />
                <CompactMetric
                  label='账期'
                  value={`${selectedDetail?.billing_config?.payment_terms ?? '-'} 天`}
                />
                <CompactMetric
                  label='超额策略'
                  value={overCreditPolicyLabel(
                    selectedDetail?.billing_config?.over_credit_policy
                  )}
                />
              </div>
            </EnterprisePanel>
            <EnterprisePanel
              title='治理状态'
              description='状态、模型、授信三个维度决定客户是否能持续调用。'
            >
              <div className='grid grid-cols-3 gap-2 text-center text-xs'>
                <div className='bg-background rounded-md border px-2 py-3'>
                  <Activity className='mx-auto mb-1 size-4 text-emerald-600' />
                  <p className='font-medium'>状态</p>
                  <p className='text-muted-foreground mt-0.5'>
                    {selectedTenant
                      ? statusMeta(selectedTenant.status).label
                      : '-'}
                  </p>
                </div>
                <div className='bg-background rounded-md border px-2 py-3'>
                  <Settings2 className='mx-auto mb-1 size-4 text-blue-600' />
                  <p className='font-medium'>模型</p>
                  <p className='text-muted-foreground mt-0.5'>
                    {policies.length} 条
                  </p>
                </div>
                <div className='bg-background rounded-md border px-2 py-3'>
                  <CreditCard className='mx-auto mb-1 size-4 text-amber-600' />
                  <p className='font-medium'>授信</p>
                  <p className='text-muted-foreground mt-0.5'>
                    {quotaText(
                      selectedDetail?.credit_account?.available_credit
                    )}
                  </p>
                </div>
              </div>
            </EnterprisePanel>
          </div>

          <CreateTenantDialog
            open={createOpen}
            onOpenChange={setCreateOpen}
            saving={createMutation.isPending}
            onSubmit={(payload) => createMutation.mutate(payload)}
          />
          <BillingConfigDialog
            open={billingOpen}
            tenant={selectedTenant}
            detail={selectedDetail}
            saving={billingMutation.isPending}
            onOpenChange={setBillingOpen}
            onSubmit={(payload) => {
              if (!selectedTenantId) return
              billingMutation.mutate({ tenantId: selectedTenantId, payload })
            }}
          />
          <ModelPolicyDialog
            open={policyOpen}
            tenant={selectedTenant}
            saving={policyMutation.isPending}
            onOpenChange={setPolicyOpen}
            onSubmit={(payload) => {
              if (!selectedTenantId) return
              policyMutation.mutate({ tenantId: selectedTenantId, payload })
            }}
          />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
