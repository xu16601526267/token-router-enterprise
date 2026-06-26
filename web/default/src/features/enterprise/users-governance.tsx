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
  Building2,
  Check,
  ChevronRight,
  CircleOff,
  Clock3,
  KeyRound,
  RefreshCw,
  Search,
  ShieldCheck,
  ShieldEllipsis,
  UserCog,
  UserRoundCheck,
  UsersRound,
} from 'lucide-react'
import { useMemo, useState, type ReactNode } from 'react'

import {
  EnterprisePageHeader,
  EnterprisePanel,
  EnterpriseStatCard,
} from '@/components/enterprise'
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
import { formatLogQuota, formatNumber } from '@/lib/format'
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

const ROLE_META: Record<number, { label: string; className: string }> = {
  [ROLE.SUPER_ADMIN]: {
    label: '超级管理员',
    className: 'border-violet-500/20 bg-violet-500/10 text-violet-600',
  },
  [ROLE.ADMIN]: {
    label: '管理员',
    className: 'border-blue-500/20 bg-blue-500/10 text-blue-600',
  },
  [ROLE.USER]: {
    label: '普通用户',
    className: 'border-slate-500/20 bg-slate-500/10 text-slate-600',
  },
}

function initials(user: EnterpriseUserItem): string {
  const name = user.display_name || user.username
  if (name.length === 0) return 'U'
  return name.slice(0, 2).toUpperCase()
}

function formatDateTime(timestamp: number): string {
  if (timestamp <= 0) return '从未登录'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(timestamp * 1000)
}

function roleMeta(role: number) {
  return ROLE_META[role] ?? ROLE_META[ROLE.USER]
}

function UserStatusBadge(props: { status: number }) {
  const active = props.status === 1
  return (
    <Badge
      variant='outline'
      className={cn(
        'text-[10px]',
        active
          ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-600'
          : 'border-slate-500/20 bg-slate-500/10 text-slate-600'
      )}
    >
      {active ? '活跃' : '已停用'}
    </Badge>
  )
}

function PermissionCell(props: { allowed: boolean; partial?: boolean }) {
  if (props.allowed) {
    return (
      <span className='mx-auto flex size-6 items-center justify-center rounded-md bg-emerald-500/10 text-emerald-600'>
        <Check className='size-3.5' />
      </span>
    )
  }
  if (props.partial) {
    return (
      <span className='mx-auto flex size-6 items-center justify-center rounded-md bg-amber-500/10 text-amber-600'>
        <ShieldEllipsis className='size-3.5' />
      </span>
    )
  }
  return (
    <span className='bg-muted text-muted-foreground mx-auto flex size-6 items-center justify-center rounded-md'>
      <CircleOff className='size-3.5' />
    </span>
  )
}

export function EnterpriseUsersGovernance(props: {
  actions?: ReactNode
  classicTable?: ReactNode
}) {
  const [activeGroup, setActiveGroup] = useState('all')
  const [search, setSearch] = useState('')
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null)
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
  const scrollToClassicUsers = () => {
    document
      .querySelector('#classic-users-management')
      ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  return (
    <div className='enterprise-dashboard space-y-3 pb-2'>
      <EnterprisePageHeader
        eyebrow='组织治理'
        title='用户、团队与权限'
        description='在一个工作区内查看组织分组、角色权限、API 资产和访问风险，并继续使用原有用户管理能力。'
        actions={
          <div className='flex flex-wrap items-center gap-2'>
            <Button
              variant='outline'
              size='sm'
              onClick={() => void usersQuery.refetch()}
              disabled={usersQuery.isFetching}
            >
              <RefreshCw
                className={cn(
                  'size-4',
                  usersQuery.isFetching && 'animate-spin'
                )}
              />
              刷新
            </Button>
            {props.actions}
          </div>
        }
      />

      <div className='grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-5'>
        <EnterpriseStatCard
          title='总用户数'
          value={formatNumber(summary.total_users)}
          helper='组织全部成员'
          icon={UsersRound}
          tone='blue'
          loading={usersQuery.isLoading}
        />
        <EnterpriseStatCard
          title='活跃席位'
          value={formatNumber(summary.active_users)}
          helper={
            summary.total_users > 0
              ? `${Math.round((summary.active_users / summary.total_users) * 100)}% 使用率`
              : '暂无席位'
          }
          icon={UserRoundCheck}
          tone='emerald'
          loading={usersQuery.isLoading}
        />
        <EnterpriseStatCard
          title='管理员账号'
          value={formatNumber(summary.admin_users)}
          helper='含超级管理员'
          icon={ShieldCheck}
          tone='violet'
          loading={usersQuery.isLoading}
        />
        <EnterpriseStatCard
          title='组织分组'
          value={formatNumber(summary.groups)}
          helper='用于权限与计费隔离'
          icon={Building2}
          tone='amber'
          loading={usersQuery.isLoading}
        />
        <EnterpriseStatCard
          title='活跃 API Keys'
          value={formatNumber(summary.active_api_keys)}
          helper={`${summary.disabled_users} 个停用账号`}
          icon={KeyRound}
          tone='blue'
          loading={usersQuery.isLoading}
        />
      </div>

      <div className='grid min-h-[680px] gap-3 xl:grid-cols-[240px_minmax(0,1fr)_330px]'>
        <EnterprisePanel
          title='组织结构'
          description='按现有用户分组聚合'
          bodyClassName='p-3'
        >
          <div className='space-y-2'>
            <button
              type='button'
              onClick={() => setActiveGroup('all')}
              className={cn(
                'flex w-full items-center justify-between rounded-md px-3 py-2.5 text-left text-xs transition-colors',
                activeGroup === 'all'
                  ? 'bg-primary/10 font-semibold text-primary'
                  : 'hover:bg-muted/60'
              )}
            >
              <span className='flex items-center gap-2'>
                <Building2 className='size-4' />
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
                  'flex w-full items-center justify-between rounded-md px-3 py-2.5 text-left text-xs transition-colors',
                  activeGroup === group.name
                    ? 'bg-primary/10 font-semibold text-primary'
                    : 'hover:bg-muted/60'
                )}
              >
                <span className='flex min-w-0 items-center gap-2'>
                  <ChevronRight className='size-3.5 shrink-0' />
                  <span className='truncate'>{group.name}</span>
                </span>
                <span className='text-muted-foreground tabular-nums'>
                  {group.count}
                </span>
              </button>
            ))}
          </div>
          <div className='text-muted-foreground mt-5 rounded-md border border-dashed p-3 text-[11px] leading-5'>
            当前兼容旧系统的用户分组字段。后续可直接扩展为多级组织树，而不会影响现有账号。
          </div>
        </EnterprisePanel>

        <EnterprisePanel
          title={
            activeGroup === 'all'
              ? `全部成员 (${filteredUsers.length})`
              : `${activeGroup} (${filteredUsers.length})`
          }
          description='选择成员后可在右侧查看权限与资产概况'
          action={
            <label className='relative block w-56 max-w-full'>
              <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-3.5 -translate-y-1/2' />
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder='搜索成员'
                className='h-8 pl-8 text-xs'
              />
            </label>
          }
          bodyClassName='p-0'
        >
          <div className='overflow-x-auto'>
            <Table>
              <TableHeader>
                <TableRow className='bg-muted/35'>
                  <TableHead className='min-w-44'>成员</TableHead>
                  <TableHead className='min-w-28'>分组</TableHead>
                  <TableHead>角色</TableHead>
                  <TableHead className='text-right'>API Keys</TableHead>
                  <TableHead className='text-right'>已用额度</TableHead>
                  <TableHead className='min-w-32'>最近登录</TableHead>
                  <TableHead>状态</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredUsers.map((user) => {
                  const meta = roleMeta(user.role)
                  const selected = selectedUser?.id === user.id
                  return (
                    <TableRow
                      key={user.id}
                      onClick={() => setSelectedUserId(user.id)}
                      className={cn(
                        'cursor-pointer hover:bg-muted/30',
                        selected && 'bg-primary/5'
                      )}
                    >
                      <TableCell>
                        <div className='flex items-center gap-3'>
                          <Avatar className='size-8'>
                            <AvatarFallback className='bg-primary/10 text-primary text-[10px] font-semibold'>
                              {initials(user)}
                            </AvatarFallback>
                          </Avatar>
                          <div className='min-w-0'>
                            <p className='truncate text-xs font-semibold'>
                              {user.display_name || user.username}
                            </p>
                            <p className='text-muted-foreground truncate text-[10px]'>
                              {user.email || user.username}
                            </p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className='text-xs'>
                        {user.group || '默认分组'}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant='outline'
                          className={cn('text-[10px]', meta.className)}
                        >
                          {meta.label}
                        </Badge>
                      </TableCell>
                      <TableCell className='text-right text-xs font-medium tabular-nums'>
                        {user.api_key_count}
                      </TableCell>
                      <TableCell className='text-right text-xs tabular-nums'>
                        {formatLogQuota(user.used_quota)}
                      </TableCell>
                      <TableCell className='text-muted-foreground text-xs'>
                        {formatDateTime(user.last_login_at)}
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
            <div className='text-muted-foreground flex min-h-72 items-center justify-center text-sm'>
              当前筛选条件下没有成员
            </div>
          )}
        </EnterprisePanel>

        <div className='space-y-4'>
          <EnterprisePanel title='成员权限概览'>
            {selectedUser ? (
              <div>
                <div className='border-border/60 flex items-center gap-3 border-b pb-4'>
                  <Avatar className='size-11'>
                    <AvatarFallback className='bg-primary/10 text-primary text-xs font-semibold'>
                      {initials(selectedUser)}
                    </AvatarFallback>
                  </Avatar>
                  <div className='min-w-0'>
                    <p className='truncate text-sm font-semibold'>
                      {selectedUser.display_name || selectedUser.username}
                    </p>
                    <p className='text-muted-foreground truncate text-xs'>
                      {selectedUser.email || selectedUser.username}
                    </p>
                  </div>
                </div>
                <dl className='mt-4 grid grid-cols-2 gap-3 text-xs'>
                  <div className='bg-muted/40 rounded-md p-3'>
                    <dt className='text-muted-foreground'>角色</dt>
                    <dd className='mt-1 font-semibold'>
                      {roleMeta(selectedUser.role).label}
                    </dd>
                  </div>
                  <div className='bg-muted/40 rounded-md p-3'>
                    <dt className='text-muted-foreground'>API Keys</dt>
                    <dd className='mt-1 font-semibold'>
                      {selectedUser.api_key_count}
                    </dd>
                  </div>
                  <div className='bg-muted/40 rounded-md p-3'>
                    <dt className='text-muted-foreground'>累计请求</dt>
                    <dd className='mt-1 font-semibold'>
                      {formatNumber(selectedUser.request_count)}
                    </dd>
                  </div>
                  <div className='bg-muted/40 rounded-md p-3'>
                    <dt className='text-muted-foreground'>可用额度</dt>
                    <dd className='mt-1 font-semibold'>
                      {formatLogQuota(selectedUser.quota)}
                    </dd>
                  </div>
                </dl>
                <Button
                  variant='outline'
                  className='mt-4 w-full'
                  size='sm'
                  disabled={!props.classicTable}
                  onClick={scrollToClassicUsers}
                >
                  <UserCog className='size-4' />
                  在经典管理中编辑
                </Button>
              </div>
            ) : (
              <div className='text-muted-foreground flex min-h-52 items-center justify-center text-sm'>
                请选择成员
              </div>
            )}
          </EnterprisePanel>

          <EnterprisePanel
            title='权限策略矩阵'
            description='沿用现有角色模型，避免破坏历史权限'
          >
            <div className='overflow-hidden rounded-md border'>
              <Table>
                <TableHeader>
                  <TableRow className='bg-muted/35'>
                    <TableHead className='text-[10px]'>模块</TableHead>
                    <TableHead className='text-center text-[10px]'>
                      超级管理员
                    </TableHead>
                    <TableHead className='text-center text-[10px]'>
                      管理员
                    </TableHead>
                    <TableHead className='text-center text-[10px]'>
                      用户
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {[
                    ['接口密钥', true, true, true],
                    ['渠道管理', true, true, false],
                    ['智能路由', true, true, false],
                    ['用户管理', true, true, false],
                    ['计费结算', true, false, false],
                    ['系统设置', true, false, false],
                  ].map((row) => (
                    <TableRow key={String(row[0])}>
                      <TableCell className='text-[10px] font-medium'>
                        {row[0]}
                      </TableCell>
                      <TableCell>
                        <PermissionCell allowed={Boolean(row[1])} />
                      </TableCell>
                      <TableCell>
                        <PermissionCell
                          allowed={Boolean(row[2])}
                          partial={row[0] === '计费结算'}
                        />
                      </TableCell>
                      <TableCell>
                        <PermissionCell allowed={Boolean(row[3])} />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </EnterprisePanel>

          <EnterprisePanel title='安全与合规'>
            <div className='space-y-3 text-xs'>
              <div className='bg-muted/35 flex items-center justify-between rounded-md p-3'>
                <span className='flex items-center gap-2'>
                  <ShieldCheck className='size-4 text-violet-500' />
                  高权限账号
                </span>
                <Badge variant='outline'>{summary.admin_users}</Badge>
              </div>
              <div className='bg-muted/35 flex items-center justify-between rounded-md p-3'>
                <span className='flex items-center gap-2'>
                  <Clock3 className='size-4 text-amber-500' />
                  30 天未登录
                </span>
                <Badge
                  variant='outline'
                  className={staleUsers > 0 ? 'text-amber-600' : ''}
                >
                  {staleUsers}
                </Badge>
              </div>
              <div className='bg-muted/35 flex items-center justify-between rounded-md p-3'>
                <span className='flex items-center gap-2'>
                  <CircleOff className='size-4 text-slate-500' />
                  停用账号
                </span>
                <Badge variant='outline'>{summary.disabled_users}</Badge>
              </div>
            </div>
          </EnterprisePanel>
        </div>
      </div>

      {props.classicTable && (
        <EnterprisePanel
          id='classic-users-management'
          title='经典用户管理'
          description='保留原系统完整的新增、编辑、额度调整、启停和删除能力'
          bodyClassName='min-h-[540px] p-0'
        >
          {props.classicTable}
        </EnterprisePanel>
      )}
    </div>
  )
}
