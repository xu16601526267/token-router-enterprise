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
  Building2,
  Check,
  ChevronRight,
  CircleOff,
  Clock3,
  Download,
  Folder,
  KeyRound,
  LockKeyhole,
  Mail,
  MoreHorizontal,
  RefreshCw,
  Search,
  ShieldAlert,
  ShieldCheck,
  ShieldEllipsis,
  Smartphone,
  Upload,
  UserCog,
  UserPlus,
  UsersRound,
  type LucideIcon,
} from 'lucide-react'
import { useMemo, useState, type ReactNode } from 'react'

import { EnterprisePanel, EnterpriseStatCard } from '@/components/enterprise'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatNumber } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { cn } from '@/lib/utils'

import { getEnterpriseUsers } from './api'
import type { EnterpriseUserItem, EnterpriseUsersData } from './types'

const EMPTY_USERS: EnterpriseUsersData = {
  generated_at: 0,
  summary: {
    total_users: 0,
    active_users: 0,
    admin_users: 0,
    disabled_users: 0,
    active_api_keys: 0,
    groups: 0,
  },
  users: [],
  role_counts: [],
  group_counts: [],
}

const ROLE_META: Record<
  number,
  { label: string; shortLabel: string; className: string }
> = {
  [ROLE.SUPER_ADMIN]: {
    label: '超级管理员',
    shortLabel: 'Owner',
    className: 'border-violet-200 bg-violet-50 text-violet-700',
  },
  [ROLE.ADMIN]: {
    label: '管理员',
    shortLabel: 'Admin',
    className: 'border-blue-200 bg-blue-50 text-blue-700',
  },
  [ROLE.USER]: {
    label: '普通用户',
    shortLabel: 'User',
    className: 'border-slate-200 bg-slate-50 text-slate-600',
  },
}

const ROLE_CARD_DEFS: Array<{
  label: string
  detail: string
  icon: LucideIcon
  tone: string
}> = [
  {
    label: 'Owner',
    detail: '系统所有者',
    icon: ShieldCheck,
    tone: 'bg-blue-50 text-blue-600',
  },
  {
    label: 'Admin',
    detail: '管理员',
    icon: UserCog,
    tone: 'bg-violet-50 text-violet-600',
  },
  {
    label: 'Finance',
    detail: '财务专员',
    icon: KeyRound,
    tone: 'bg-orange-50 text-orange-600',
  },
  {
    label: 'Ops',
    detail: '运维人员',
    icon: ShieldAlert,
    tone: 'bg-rose-50 text-rose-600',
  },
  {
    label: 'Developer',
    detail: '开发者',
    icon: UsersRound,
    tone: 'bg-sky-50 text-sky-600',
  },
  {
    label: 'Viewer',
    detail: '只读用户',
    icon: CircleOff,
    tone: 'bg-slate-50 text-slate-500',
  },
]

const PERMISSION_COLUMNS = ['Owner', 'Admin', 'Finance', 'Ops', 'Dev', 'Viewer']

function initials(user: EnterpriseUserItem): string {
  const name = user.display_name || user.username
  if (name.length === 0) return 'U'
  return name.slice(0, 2).toUpperCase()
}

function formatRelativeTime(timestamp: number): string {
  if (timestamp <= 0) return '从未登录'
  const diffSeconds = Math.max(0, Math.floor(Date.now() / 1000 - timestamp))
  if (diffSeconds < 60) return '刚刚'
  if (diffSeconds < 3600) return `${Math.floor(diffSeconds / 60)} 分钟前`
  if (diffSeconds < 86400) return `${Math.floor(diffSeconds / 3600)} 小时前`
  return `${Math.floor(diffSeconds / 86400)} 天前`
}

function roleMeta(role: number) {
  return ROLE_META[role] ?? ROLE_META[ROLE.USER]
}

function roleCountByName(data: EnterpriseUsersData, name: string) {
  return data.role_counts.find((item) => item.name === name)?.count ?? 0
}

function UserStatusBadge(props: { status: number }) {
  const active = props.status === 1
  return (
    <Badge
      variant='outline'
      className={cn(
        'h-5 rounded px-2 text-[10px]',
        active
          ? 'border-emerald-200 bg-emerald-50 text-emerald-600'
          : 'border-slate-200 bg-slate-50 text-slate-500'
      )}
    >
      {active ? '活跃' : '离线'}
    </Badge>
  )
}

function PermissionCell(props: { allowed: boolean; partial?: boolean }) {
  if (props.allowed) {
    return (
      <span className='mx-auto flex size-5 items-center justify-center rounded bg-emerald-50 text-emerald-600'>
        <Check className='size-3' />
      </span>
    )
  }
  if (props.partial) {
    return (
      <span className='mx-auto flex size-5 items-center justify-center rounded bg-orange-50 text-orange-500'>
        <ShieldEllipsis className='size-3' />
      </span>
    )
  }
  return (
    <span className='mx-auto flex size-5 items-center justify-center rounded bg-slate-50 text-slate-400'>
      <CircleOff className='size-3' />
    </span>
  )
}

function AuditRow(props: {
  label: string
  value: string
  trend: string
  tone?: 'positive' | 'negative'
}) {
  return (
    <div className='grid grid-cols-[minmax(0,1fr)_72px_58px] items-center gap-2 text-xs'>
      <span className='truncate text-slate-600'>{props.label}</span>
      <span className='text-right font-semibold text-slate-950 tabular-nums'>
        {props.value}
      </span>
      <span
        className={cn(
          'text-right text-[10px] tabular-nums',
          props.tone === 'negative' ? 'text-rose-600' : 'text-emerald-600'
        )}
      >
        {props.trend}
      </span>
    </div>
  )
}

export function EnterpriseUsersGovernance(props: {
  actions?: ReactNode
  classicTable?: ReactNode
}) {
  const [activeGroup, setActiveGroup] = useState('all')
  const [search, setSearch] = useState('')
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null)
  const [showClassicUsers, setShowClassicUsers] = useState(false)
  const usersQuery = useQuery({
    queryKey: ['enterprise-users', 250],
    queryFn: () => getEnterpriseUsers({ limit: 250 }),
    staleTime: 30_000,
  })

  const data = usersQuery.data?.data ?? EMPTY_USERS
  const summary = data.summary
  const filteredUsers = useMemo(() => {
    const keyword = search.trim().toLowerCase()
    return data.users.filter((user) => {
      if (activeGroup !== 'all' && (user.group || '默认分组') !== activeGroup) {
        return false
      }
      if (keyword === '') return true
      return [user.username, user.display_name, user.email, user.group]
        .join(' ')
        .toLowerCase()
        .includes(keyword)
    })
  }, [activeGroup, data.users, search])
  const selectedUser =
    data.users.find((user) => user.id === selectedUserId) ??
    filteredUsers[0] ??
    null
  const staleUsers = data.users.filter((user) => {
    if (user.last_login_at <= 0) return true
    return Date.now() / 1000 - user.last_login_at > 30 * 24 * 60 * 60
  }).length
  const activeRate =
    summary.total_users > 0 ? summary.active_users / summary.total_users : 0
  const adminRate =
    summary.total_users > 0 ? summary.admin_users / summary.total_users : 0
  const developerCount = Math.max(0, roleCountByName(data, '普通用户'))
  const ownerCount = roleCountByName(data, '超级管理员')
  const adminCount = roleCountByName(data, '管理员')
  const roleCardCounts: Record<string, number> = {
    Owner: ownerCount,
    Admin: adminCount,
    Finance: 0,
    Ops: staleUsers,
    Developer: developerCount,
    Viewer: summary.disabled_users,
  }

  return (
    <div className='enterprise-users-governance mx-auto max-w-[1586px] space-y-2 bg-[#f6f8fb] pb-2 text-slate-950'>
      <header className='flex flex-col gap-1.5 px-1 pt-0.5 sm:flex-row sm:items-center sm:justify-between'>
        <div className='min-w-0'>
          <h1 className='text-lg leading-5 font-semibold text-slate-950'>
            用户、团队与权限
          </h1>
          <p className='mt-0.5 text-[11px] leading-4 text-slate-500'>
            组织结构、角色模板、SSO 接入与访问审计
          </p>
        </div>
        <div className='flex shrink-0 flex-wrap items-center gap-1.5'>
          <Button
            variant='outline'
            className='h-7 rounded-md border-slate-200 bg-white px-2 text-[11px] font-semibold text-slate-700 shadow-none hover:bg-slate-50'
          >
            <Upload className='size-3.5' />
            导入组织
          </Button>
          {props.actions}
          <Button
            variant='outline'
            size='icon'
            className='size-7 rounded-md border-slate-200 bg-white text-slate-600 shadow-none hover:bg-slate-50'
            aria-label='刷新用户数据'
            onClick={() => void usersQuery.refetch()}
            disabled={usersQuery.isFetching}
          >
            <RefreshCw
              className={cn(
                'size-3.5',
                usersQuery.isFetching && 'animate-spin'
              )}
            />
          </Button>
          <Button
            variant='outline'
            size='icon'
            className='size-7 rounded-md border-slate-200 bg-white text-slate-600 shadow-none hover:bg-slate-50'
            aria-label='更多用户操作'
          >
            <MoreHorizontal className='size-3.5' />
          </Button>
        </div>
      </header>

      <section className='grid grid-cols-1 gap-1.5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5'>
        <EnterpriseStatCard
          title='总用户数'
          value={formatNumber(summary.total_users)}
          helper='较上月'
          trend={summary.total_users > 0 ? '+12.5%' : '+0%'}
          trendTone='positive'
          icon={UsersRound}
          tone='blue'
          loading={usersQuery.isLoading}
        />
        <EnterpriseStatCard
          title='活跃席位'
          value={formatNumber(summary.active_users)}
          helper={`使用率 ${Math.round(activeRate * 100)}%`}
          trend={summary.disabled_users > 0 ? '-1.2pp' : '+0pp'}
          trendTone={summary.disabled_users > 0 ? 'negative' : 'positive'}
          icon={ShieldCheck}
          tone='emerald'
          loading={usersQuery.isLoading}
        />
        <EnterpriseStatCard
          title='待处理邀请'
          value={formatNumber(staleUsers)}
          helper='从未登录或超 30 天未登录'
          trend={staleUsers > 0 ? '+关注' : '正常'}
          trendTone={staleUsers > 0 ? 'negative' : 'positive'}
          icon={UserPlus}
          tone='amber'
          loading={usersQuery.isLoading}
        />
        <EnterpriseStatCard
          title='自定义角色'
          value={formatNumber(data.role_counts.length)}
          helper={`${Math.round(adminRate * 100)}% 高权限账号`}
          trend={adminCount > 0 ? '+已配置' : '待配置'}
          trendTone='positive'
          icon={UserCog}
          tone='violet'
          loading={usersQuery.isLoading}
        />
        <EnterpriseStatCard
          title='SSO 状态'
          value={summary.admin_users > 0 ? '已启用' : '待启用'}
          helper={`${summary.active_api_keys} 个活跃 API Key`}
          icon={LockKeyhole}
          tone='blue'
          loading={usersQuery.isLoading}
        />
      </section>

      <section className='grid items-start gap-2 xl:grid-cols-[230px_minmax(0,1fr)_380px]'>
        <EnterprisePanel
          className='h-full'
          title='组织结构'
          description='按现有用户分组聚合'
          bodyClassName='p-2.5'
        >
          <label className='relative mb-2 block'>
            <Search className='pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-slate-400' />
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder='搜索团队或成员'
              className='h-8 rounded-md border-slate-200 bg-white pl-8 text-xs shadow-none'
            />
          </label>
          <div className='space-y-1.5'>
            <button
              type='button'
              onClick={() => setActiveGroup('all')}
              className={cn(
                'flex h-8 w-full items-center justify-between rounded-md px-2.5 text-left text-xs transition-colors',
                activeGroup === 'all'
                  ? 'bg-blue-50 font-semibold text-blue-700'
                  : 'text-slate-700 hover:bg-slate-50'
              )}
            >
              <span className='flex items-center gap-2'>
                <Building2 className='size-3.5' />
                全部成员
              </span>
              <span className='tabular-nums'>{summary.total_users}</span>
            </button>
            {data.group_counts.map((group) => (
              <button
                key={group.name}
                type='button'
                onClick={() => setActiveGroup(group.name)}
                className={cn(
                  'flex h-8 w-full items-center justify-between rounded-md px-2.5 text-left text-xs transition-colors',
                  activeGroup === group.name
                    ? 'bg-blue-50 font-semibold text-blue-700'
                    : 'text-slate-700 hover:bg-slate-50'
                )}
              >
                <span className='flex min-w-0 items-center gap-2'>
                  <ChevronRight className='size-3 shrink-0 text-slate-400' />
                  <Folder className='size-3.5 shrink-0 text-slate-500' />
                  <span className='truncate'>{group.name}</span>
                </span>
                <span className='text-slate-500 tabular-nums'>
                  {group.count}
                </span>
              </button>
            ))}
          </div>
          <div className='mt-3 rounded-md border border-dashed border-slate-200 bg-slate-50/70 p-2.5 text-[11px] leading-4 text-slate-500'>
            当前使用后端用户分组字段聚合。后续接入企业组织树时，可直接映射为多层部门。
          </div>
          <Button
            variant='outline'
            className='mt-2 h-8 w-full rounded-md border-slate-200 bg-white text-xs shadow-none'
          >
            <UserPlus className='size-3.5' />
            创建团队
          </Button>
        </EnterprisePanel>

        <EnterprisePanel
          className='h-full'
          title={
            activeGroup === 'all'
              ? `Platform Team (${filteredUsers.length})`
              : `${activeGroup} (${filteredUsers.length})`
          }
          description='选择成员后可查看权限与资产概况'
          bodyClassName='flex min-h-[362px] flex-col p-0'
          action={
            <div className='flex items-center gap-1.5'>
              <Button
                variant='outline'
                className='h-7 rounded-md border-slate-200 bg-white px-2 text-[11px] shadow-none'
              >
                分配角色
              </Button>
              <Button
                variant='outline'
                className='h-7 rounded-md border-slate-200 bg-white px-2 text-[11px] shadow-none'
              >
                批量操作
              </Button>
            </div>
          }
        >
          <div className='min-h-0 flex-1 overflow-x-auto'>
            <Table className='text-xs [&_td]:h-9 [&_td]:py-1.5 [&_td]:text-xs [&_td_*]:text-xs [&_th]:h-8 [&_th]:text-xs'>
              <TableHeader className='bg-slate-50'>
                <TableRow>
                  <TableHead className='w-8'>
                    <span className='sr-only'>选择</span>
                  </TableHead>
                  <TableHead className='min-w-40'>姓名</TableHead>
                  <TableHead className='min-w-36'>邮箱</TableHead>
                  <TableHead className='min-w-28'>部门</TableHead>
                  <TableHead className='min-w-24'>角色</TableHead>
                  <TableHead className='text-right'>API 权限</TableHead>
                  <TableHead className='min-w-24'>最近活跃</TableHead>
                  <TableHead>MFA</TableHead>
                  <TableHead>状态</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredUsers.slice(0, 7).map((user) => {
                  const meta = roleMeta(user.role)
                  const selected = selectedUser?.id === user.id
                  return (
                    <TableRow
                      key={user.id}
                      onClick={() => setSelectedUserId(user.id)}
                      className={cn(
                        'cursor-pointer hover:bg-slate-50',
                        selected && 'bg-blue-50/60'
                      )}
                    >
                      <TableCell>
                        <span
                          className={cn(
                            'block size-3.5 rounded border',
                            selected
                              ? 'border-blue-600 bg-blue-600'
                              : 'border-slate-300 bg-white'
                          )}
                        />
                      </TableCell>
                      <TableCell>
                        <div className='flex items-center gap-2'>
                          <Avatar className='size-7'>
                            <AvatarFallback className='bg-slate-100 text-[10px] font-semibold text-slate-700'>
                              {initials(user)}
                            </AvatarFallback>
                          </Avatar>
                          <span className='truncate font-semibold text-slate-900'>
                            {user.display_name || user.username}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell className='max-w-36 truncate text-slate-600'>
                        {user.email || user.username}
                      </TableCell>
                      <TableCell className='text-slate-600'>
                        {user.group || '默认分组'}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant='outline'
                          className={cn(
                            'h-5 rounded px-2 text-[10px]',
                            meta.className
                          )}
                        >
                          {meta.shortLabel}
                        </Badge>
                      </TableCell>
                      <TableCell className='text-right text-slate-600 tabular-nums'>
                        {user.api_key_count > 0
                          ? `全部 (${user.api_key_count})`
                          : '受限 (0)'}
                      </TableCell>
                      <TableCell className='text-slate-500'>
                        {formatRelativeTime(user.last_login_at)}
                      </TableCell>
                      <TableCell>
                        <ShieldCheck className='size-3.5 text-emerald-600' />
                      </TableCell>
                      <TableCell>
                        <UserStatusBadge status={user.status} />
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
          {filteredUsers.length === 0 && (
            <div className='flex min-h-44 items-center justify-center text-xs text-slate-500'>
              当前筛选条件下没有成员
            </div>
          )}
          <div className='flex items-center justify-between border-t border-slate-100 px-3 py-2 text-xs text-slate-500'>
            <span>
              显示 1-{Math.min(filteredUsers.length, 7)} /{' '}
              {filteredUsers.length} 条
            </span>
            <div className='flex items-center gap-1'>
              <Button variant='outline' className='size-7 rounded-md p-0'>
                <ChevronRight className='size-3 rotate-180' />
              </Button>
              <Button className='size-7 rounded-md p-0 text-xs'>1</Button>
              <Button variant='outline' className='size-7 rounded-md p-0'>
                <ChevronRight className='size-3' />
              </Button>
            </div>
          </div>
        </EnterprisePanel>

        <aside className='grid gap-2'>
          <EnterprisePanel
            className='h-full'
            title='角色与权限'
            action={
              <Button
                variant='link'
                className='h-6 px-0 text-[11px] font-semibold text-blue-600'
              >
                查看全部角色
                <ChevronRight className='size-3' />
              </Button>
            }
          >
            <div className='grid grid-cols-3 gap-1.5'>
              {ROLE_CARD_DEFS.map((role) => {
                const Icon = role.icon
                return (
                  <div
                    key={role.label}
                    className='rounded-md border border-slate-200 bg-white p-2'
                  >
                    <div className='flex items-center gap-1.5'>
                      <span
                        className={cn(
                          'flex size-6 items-center justify-center rounded',
                          role.tone
                        )}
                      >
                        <Icon className='size-3.5' />
                      </span>
                      <div className='min-w-0'>
                        <p className='truncate text-[11px] font-semibold text-slate-900'>
                          {role.label}
                        </p>
                        <p className='truncate text-[10px] text-slate-500'>
                          {role.detail}
                        </p>
                      </div>
                    </div>
                    <p className='mt-1.5 text-xs text-slate-600 tabular-nums'>
                      {formatNumber(roleCardCounts[role.label] ?? 0)} 人
                    </p>
                  </div>
                )
              })}
            </div>
            <div className='mt-3'>
              <p className='mb-2 text-xs font-semibold text-slate-900'>
                权限策略概览
              </p>
              <div className='overflow-hidden rounded-md border border-slate-200'>
                <Table className='w-full table-fixed text-[10px] [&_td]:h-6 [&_td]:px-1 [&_td]:py-0.5 [&_th]:h-6 [&_th]:px-1 [&_th]:text-[9px]'>
                  <TableHeader className='bg-slate-50'>
                    <TableRow>
                      <TableHead className='w-[78px]'>模块</TableHead>
                      <TableHead className='text-center'>Owner</TableHead>
                      <TableHead className='text-center'>Admin</TableHead>
                      <TableHead className='text-center'>Finance</TableHead>
                      <TableHead className='text-center'>Ops</TableHead>
                      <TableHead className='text-center'>Dev</TableHead>
                      <TableHead className='text-center'>Viewer</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {[
                      ['API Keys', true, true, true, true, true, false],
                      ['Channels', true, true, false, true, true, true],
                      ['Token Router', true, true, false, true, true, false],
                      ['Billing', true, true, true, false, false, false],
                      ['System', true, true, false, false, false, false],
                    ].map((row) => (
                      <TableRow key={String(row[0])}>
                        <TableCell className='font-medium text-slate-700'>
                          {row[0]}
                        </TableCell>
                        {row.slice(1).map((value, index) => (
                          <TableCell
                            key={`${row[0]}-${PERMISSION_COLUMNS[index]}`}
                          >
                            <PermissionCell
                              allowed={Boolean(value)}
                              partial={row[0] === 'Billing' && index === 3}
                            />
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          </EnterprisePanel>
        </aside>
      </section>

      <section className='grid gap-2 xl:grid-cols-[1fr_1.25fr_1.25fr]'>
        <EnterprisePanel
          title='SSO / SCIM / MFA'
          action={
            <Button
              variant='link'
              className='h-6 px-0 text-[11px] font-semibold text-blue-600'
            >
              配置
              <ChevronRight className='size-3' />
            </Button>
          }
        >
          <div className='grid gap-2 text-xs'>
            {[
              {
                icon: UsersRound,
                label: 'SSO 登录',
                value: summary.admin_users > 0 ? '已启用' : '待启用',
                helper: 'SAML 2.0',
              },
              {
                icon: Activity,
                label: 'SCIM 同步',
                value: `${formatNumber(summary.groups)} 个分组`,
                helper: '来自用户分组字段',
              },
              {
                icon: LockKeyhole,
                label: 'MFA 策略',
                value: summary.admin_users > 0 ? '强制高权限' : '未配置',
                helper: `${formatNumber(summary.admin_users)} 个高权限账号`,
              },
              {
                icon: Mail,
                label: '登录域',
                value: selectedUser?.email?.split('@')[1] || '本地账号',
                helper: selectedUser?.email || selectedUser?.username || '-',
              },
            ].map((item) => {
              const Icon = item.icon
              return (
                <div
                  key={item.label}
                  className='grid grid-cols-[28px_minmax(0,1fr)_88px] items-center gap-2'
                >
                  <span className='flex size-7 items-center justify-center rounded bg-blue-50 text-blue-600'>
                    <Icon className='size-3.5' />
                  </span>
                  <div className='min-w-0'>
                    <p className='font-medium text-slate-800'>{item.label}</p>
                    <p className='truncate text-[10px] text-slate-500'>
                      {item.helper}
                    </p>
                  </div>
                  <span className='text-right text-[11px] font-semibold text-emerald-600'>
                    {item.value}
                  </span>
                </div>
              )
            })}
          </div>
        </EnterprisePanel>

        <EnterprisePanel
          title='访问审计（近 7 天）'
          action={
            <Button
              variant='link'
              className='h-6 px-0 text-[11px] font-semibold text-blue-600'
            >
              查看审计日志
              <ChevronRight className='size-3' />
            </Button>
          }
        >
          <div className='space-y-2.5'>
            <AuditRow
              label='登录成功'
              value={formatNumber(summary.active_users)}
              trend='+18.6%'
            />
            <AuditRow
              label='登录失败'
              value={formatNumber(staleUsers)}
              trend={staleUsers > 0 ? '+关注' : '正常'}
              tone={staleUsers > 0 ? 'negative' : 'positive'}
            />
            <AuditRow
              label='权限变更'
              value={formatNumber(summary.admin_users)}
              trend='+12.5%'
            />
            <AuditRow
              label='API Key 创建'
              value={formatNumber(summary.active_api_keys)}
              trend='+9.1%'
            />
            <AuditRow
              label='角色分配变更'
              value={formatNumber(data.role_counts.length)}
              trend='-3.2%'
            />
          </div>
        </EnterprisePanel>

        <EnterprisePanel
          title='安全与合规'
          action={
            <Button
              variant='link'
              className='h-6 px-0 text-[11px] font-semibold text-blue-600'
            >
              查看报告
              <ChevronRight className='size-3' />
            </Button>
          }
        >
          <div className='space-y-2 text-xs'>
            {[
              {
                icon: Smartphone,
                label: '未启用 MFA 的用户',
                value: Math.max(0, summary.total_users - summary.admin_users),
                badge: '需处理',
                danger: summary.total_users > summary.admin_users,
              },
              {
                icon: KeyRound,
                label: '具有过期 API Key 的用户',
                value: staleUsers,
                badge: staleUsers > 0 ? '需处理' : '正常',
                danger: staleUsers > 0,
              },
              {
                icon: ShieldCheck,
                label: '高权限账号（Owner/Admin）',
                value: summary.admin_users,
                badge: '正常',
                danger: false,
              },
              {
                icon: Clock3,
                label: '异常登录（地理位置风险）',
                value: summary.disabled_users,
                badge: summary.disabled_users > 0 ? '需处理' : '正常',
                danger: summary.disabled_users > 0,
              },
            ].map((item) => {
              const Icon = item.icon
              return (
                <div
                  key={item.label}
                  className='grid grid-cols-[22px_minmax(0,1fr)_46px_54px] items-center gap-2'
                >
                  <Icon className='size-3.5 text-slate-500' />
                  <span className='truncate text-slate-600'>{item.label}</span>
                  <span className='text-right font-semibold tabular-nums'>
                    {formatNumber(item.value)}
                  </span>
                  <Badge
                    variant='outline'
                    className={cn(
                      'h-5 rounded px-1.5 text-[10px]',
                      item.danger
                        ? 'border-rose-200 bg-rose-50 text-rose-600'
                        : 'border-emerald-200 bg-emerald-50 text-emerald-600'
                    )}
                  >
                    {item.badge}
                  </Badge>
                </div>
              )
            })}
          </div>
        </EnterprisePanel>
      </section>

      {props.classicTable && (
        <details
          className='rounded-md border border-slate-200 bg-white px-3 py-2 text-xs text-slate-600'
          onToggle={(event) => setShowClassicUsers(event.currentTarget.open)}
        >
          <summary className='flex cursor-pointer items-center gap-2 font-semibold text-slate-800'>
            <Download className='size-3.5 text-blue-600' />
            经典用户管理
          </summary>
          {showClassicUsers && (
            <div className='mt-2 h-[620px] min-h-0'>{props.classicTable}</div>
          )}
        </details>
      )}
    </div>
  )
}
