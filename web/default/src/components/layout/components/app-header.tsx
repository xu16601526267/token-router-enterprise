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
import {
  Building2,
  CalendarClock,
  Check,
  ChevronDown,
  HelpCircle,
  UserRound,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useLocation } from '@tanstack/react-router'

import { ConfigDrawer } from '@/components/config-drawer'
import { NotificationPopover } from '@/components/notification-popover'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { Search } from '@/components/search'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  ENTERPRISE_TIME_RANGE_OPTIONS,
  useEnterpriseConsole,
  type EnterpriseDateRange,
  type EnterpriseRangePreset,
} from '@/context/enterprise-console-context'
import { getPlatformTenants } from '@/features/enterprise/api'
import { useNotifications } from '@/hooks/use-notifications'
import { ROLE } from '@/lib/roles'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import type { TopNavLink } from '../types'
import { Header } from './header'

/**
 * General application Header component
 * Integrates navigation bar, search, configuration and profile functions
 *
 * @example
 * // Basic usage
 * <AppHeader />
 *
 * @example
 * // Custom navigation links
 * <AppHeader navLinks={customLinks} />
 *
 * @example
 * // Hide navigation bar and search box
 * <AppHeader showTopNav={false} showSearch={false} />
 *
 * @example
 * // Fully customize left and right content
 * <AppHeader
 *   leftContent={<CustomLeft />}
 *   rightContent={<CustomRight />}
 * />
 */
type AppHeaderProps = {
  /**
   * Custom navigation links, uses default global navigation or dynamically generated from backend if not provided
   */
  navLinks?: TopNavLink[]
  /**
   * Whether to show top navigation bar
   * @default true
   */
  showTopNav?: boolean
  /**
   * Left content, overrides TopNav if provided
   */
  leftContent?: React.ReactNode
  /**
   * Whether to show search box
   * @default true
   */
  showSearch?: boolean
  /**
   * Custom right content, overrides default right content if provided
   */
  rightContent?: React.ReactNode
  /**
   * Whether to show notification button
   * @default true
   */
  showNotifications?: boolean
  /**
   * Whether to show config drawer
   * @default true
   */
  showConfigDrawer?: boolean
  /**
   * Whether to show profile dropdown
   * @default true
   */
  showProfileDropdown?: boolean
}

function toDateTimeInputValue(timestamp: number): string {
  if (!timestamp) return ''
  const date = new Date(timestamp * 1000)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

function fromDateTimeInputValue(value: string): number | undefined {
  if (!value) return undefined
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return undefined
  return Math.floor(date.getTime() / 1000)
}

function formatRangeDateTime(timestamp: number): string {
  if (!timestamp) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(timestamp * 1000)
}

function formatDateRangeLabel(
  range: EnterpriseDateRange,
  rangePreset: EnterpriseRangePreset,
  rangeLabel: string
): string {
  if (rangePreset !== 'custom') return rangeLabel
  return `${formatRangeDateTime(range.start)} - ${formatRangeDateTime(range.end)}`
}

function EnterpriseDateRangeControl({
  controlClassName,
  selectedMarkClassName,
}: {
  controlClassName: string
  selectedMarkClassName: string
}) {
  const {
    range,
    rangePreset,
    rangeLabel,
    setRangePreset,
    setCustomRange,
  } = useEnterpriseConsole()
  const [open, setOpen] = useState(false)
  const [draftStart, setDraftStart] = useState(toDateTimeInputValue(range.start))
  const [draftEnd, setDraftEnd] = useState(toDateTimeInputValue(range.end))
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open) return
    setDraftStart(toDateTimeInputValue(range.start))
    setDraftEnd(toDateTimeInputValue(range.end))
    setError('')
  }, [open, range.end, range.start])

  const applyPreset = (preset: Exclude<EnterpriseRangePreset, 'custom'>) => {
    setRangePreset(preset)
    setError('')
    setOpen(false)
  }

  const applyCustomRange = () => {
    const start = fromDateTimeInputValue(draftStart)
    const end = fromDateTimeInputValue(draftEnd)

    if (!start || !end) {
      setError('请选择完整的开始和结束时间')
      return
    }
    if (end <= start) {
      setError('结束时间必须晚于开始时间')
      return
    }

    setCustomRange({ start, end })
    setError('')
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            type='button'
            variant='outline'
            className={cn(
              controlClassName,
              'max-w-[276px] gap-1.5 text-slate-600 tabular-nums'
            )}
          />
        }
      >
        <CalendarClock className='size-3.5 shrink-0' />
        <span className='truncate'>
          {formatDateRangeLabel(range, rangePreset, rangeLabel)}
        </span>
        <ChevronDown className='size-3.5 shrink-0 text-slate-400' />
      </PopoverTrigger>
      <PopoverContent
        align='start'
        className='w-[342px] rounded-md border-slate-200 p-2.5 shadow-[0_8px_22px_rgb(15_23_42/0.10)]'
      >
        <div className='space-y-2.5'>
          <div className='grid grid-cols-4 gap-1'>
            {ENTERPRISE_TIME_RANGE_OPTIONS.map((option) => (
              <Button
                key={option.value}
                type='button'
                variant='outline'
                className={cn(
                  'h-7 rounded-md px-1.5 text-[11px] font-medium shadow-none',
                  rangePreset === option.value
                    ? 'border-blue-200 bg-blue-50 text-blue-700'
                    : 'border-slate-200 bg-white text-slate-600 hover:bg-slate-50'
                )}
                onClick={() => applyPreset(option.value)}
              >
                {option.label}
                {rangePreset === option.value && (
                  <Check className={selectedMarkClassName} />
                )}
              </Button>
            ))}
          </div>

          <div className='grid gap-2'>
            <label className='grid gap-1 text-[11px] font-medium text-slate-600'>
              开始时间
              <Input
                type='datetime-local'
                value={draftStart}
                onChange={(event) => setDraftStart(event.target.value)}
                className='h-8 rounded-md border-slate-200 bg-white text-[12px] tabular-nums'
              />
            </label>
            <label className='grid gap-1 text-[11px] font-medium text-slate-600'>
              结束时间
              <Input
                type='datetime-local'
                value={draftEnd}
                onChange={(event) => setDraftEnd(event.target.value)}
                className='h-8 rounded-md border-slate-200 bg-white text-[12px] tabular-nums'
              />
            </label>
          </div>

          {error && <p className='text-[11px] text-rose-600'>{error}</p>}

          <div className='flex justify-end'>
            <Button
              type='button'
              className='h-7 rounded-md px-3 text-[11px] font-semibold'
              onClick={applyCustomRange}
            >
              应用时间段
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}

function EnterpriseHeaderContext() {
  const user = useAuthStore((state) => state.auth.user)
  const pathname = useLocation({ select: (location) => location.pathname })
  const {
    workspaceId,
    setWorkspaceId,
  } = useEnterpriseConsole()
  const isPersonalWorkbench = pathname.startsWith('/wallet')
  const isAdmin = (user?.role ?? 0) >= ROLE.ADMIN
  const rawGroup = user?.group?.trim()
  let workspace = '个人工作区'
  if (rawGroup && rawGroup !== 'default') {
    workspace = rawGroup
  } else if (isAdmin) {
    workspace = '趋境科技 / Enterprise'
  }
  const tenantsQuery = useQuery({
    queryKey: ['enterprise-header-tenants'],
    queryFn: () => getPlatformTenants({ status: 'active' }),
    enabled: isAdmin,
    staleTime: 60_000,
  })
  const tenantOptions =
    tenantsQuery.data?.data?.slice(0, 12).map((tenant) => ({
      value: `tenant:${tenant.id}`,
      label: tenant.name,
      description:
        tenant.industry ||
        tenant.type ||
        (tenant.status === 'active' ? '活跃租户' : tenant.status),
    })) ?? []
  const workspaceOptions = [
    {
      value: 'platform',
      label: workspace,
      description: isAdmin ? '平台总览' : '当前工作区',
    },
    ...tenantOptions,
  ]
  const selectedWorkspace =
    workspaceOptions.find((item) => item.value === workspaceId) ??
    workspaceOptions[0]
  const controlClassName =
    'h-8 rounded-md border-slate-200/80 bg-white/90 px-2.5 text-[12px] font-medium text-slate-700 shadow-none hover:border-slate-300 hover:bg-white aria-expanded:bg-white'
  const selectedMarkClassName = 'ms-auto size-3.5 text-blue-600'

  if (isPersonalWorkbench) {
    return (
      <div className='hidden min-w-0 items-center gap-2 lg:flex'>
        <div
          className={cn(
            controlClassName,
            'flex max-w-56 items-center justify-start gap-2'
          )}
        >
          <UserRound className='size-3.5 shrink-0 text-blue-600' />
          <span className='truncate'>个人工作台</span>
        </div>
      </div>
    )
  }

  return (
    <div className='hidden min-w-0 items-center gap-2 lg:flex'>
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger
          render={
            <Button
              variant='outline'
              className={cn(controlClassName, 'max-w-56 justify-start gap-2')}
            />
          }
        >
          <Building2 className='size-3.5 shrink-0 text-blue-600' />
          <span className='truncate'>{selectedWorkspace.label}</span>
          <ChevronDown className='ms-auto size-3.5 shrink-0 text-slate-400' />
        </DropdownMenuTrigger>
        <DropdownMenuContent className='w-64 rounded-md border-slate-200 p-1 shadow-[0_8px_22px_rgb(15_23_42/0.10)]'>
          <DropdownMenuGroup>
            <DropdownMenuLabel className='px-2 py-1 text-[11px]'>
              公司 / 租户
            </DropdownMenuLabel>
            {workspaceOptions.map((option) => (
              <DropdownMenuItem
                key={option.value}
                className='gap-2 rounded-md px-2 py-1.5 text-[12px]'
                onClick={() => setWorkspaceId(option.value)}
              >
                <span className='min-w-0 flex-1'>
                  <span className='block truncate font-medium text-slate-800'>
                    {option.label}
                  </span>
                  <span className='block truncate text-[10.5px] text-slate-500'>
                    {option.description}
                  </span>
                </span>
                {workspaceId === option.value && (
                  <Check className={selectedMarkClassName} />
                )}
              </DropdownMenuItem>
            ))}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>

      <EnterpriseDateRangeControl
        controlClassName={controlClassName}
        selectedMarkClassName={selectedMarkClassName}
      />
    </div>
  )
}

export function AppHeader({
  leftContent,
  showSearch = true,
  rightContent,
  showNotifications = true,
  showConfigDrawer = true,
  showProfileDropdown = true,
}: AppHeaderProps) {
  const notifications = useNotifications()

  return (
    <Header>
      <EnterpriseHeaderContext />

      {leftContent ? (
        <div className='ms-2 flex items-center'>{leftContent}</div>
      ) : null}

      {rightContent ?? (
        <div className='ms-auto flex min-w-0 items-center gap-1.5 sm:gap-2'>
          {showSearch && (
            <Search
              className='hidden h-8 border-slate-200 bg-white text-[12px] md:flex lg:w-64 xl:w-80'
              placeholder='搜索客户、密钥、渠道、模型'
            />
          )}
          {showNotifications && (
            <NotificationPopover
              open={notifications.popoverOpen}
              onOpenChange={notifications.setPopoverOpen}
              unreadCount={notifications.unreadCount}
              activeTab={notifications.activeTab}
              onTabChange={notifications.setActiveTab}
              notice={notifications.notice}
              announcements={notifications.announcements}
              loading={notifications.loading}
            />
          )}
          <Button
            variant='outline'
            size='icon-sm'
            className='hidden border-slate-200 bg-white text-slate-500 sm:inline-flex'
            aria-label='帮助中心'
          >
            <HelpCircle className='size-4' />
          </Button>
          {showConfigDrawer && <ConfigDrawer />}
          {showProfileDropdown && <ProfileDropdown />}
        </div>
      )}
    </Header>
  )
}
