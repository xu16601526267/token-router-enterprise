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
  ChevronDown,
  CircleDot,
  HelpCircle,
} from 'lucide-react'

import { ConfigDrawer } from '@/components/config-drawer'
import { NotificationPopover } from '@/components/notification-popover'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { Search } from '@/components/search'
import { Button } from '@/components/ui/button'
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

function EnterpriseHeaderContext() {
  const user = useAuthStore((state) => state.auth.user)
  const isAdmin = (user?.role ?? 0) >= ROLE.ADMIN
  const rawGroup = user?.group?.trim()
  const workspace =
    rawGroup && rawGroup !== 'default'
      ? rawGroup
      : isAdmin
        ? '趋境科技 / Enterprise'
        : '个人工作区'
  const dateLabel = new Intl.DateTimeFormat('zh-CN', {
    weekday: 'short',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date())
  const envLabel =
    import.meta.env.MODE === 'production' ? '生产环境' : '测试环境'

  return (
    <div className='hidden min-w-0 items-center gap-2 lg:flex'>
      <button
        type='button'
        className='inline-flex h-8 max-w-56 items-center gap-2 rounded-md border border-slate-200 bg-white px-2.5 text-xs font-medium text-slate-700 shadow-sm transition-colors hover:bg-slate-50'
      >
        <Building2 className='size-3.5 shrink-0 text-blue-600' />
        <span className='truncate'>{workspace}</span>
        <ChevronDown className='size-3.5 shrink-0 text-slate-400' />
      </button>
      <span
        className={cn(
          'inline-flex h-8 items-center gap-1.5 rounded-md border px-2.5 text-xs font-medium',
          isAdmin
            ? 'border-emerald-200 bg-emerald-50 text-emerald-700'
            : 'border-blue-200 bg-blue-50 text-blue-700'
        )}
      >
        <CircleDot className='size-3.5' />
        {envLabel}
      </span>
      <span className='inline-flex h-8 items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 text-xs font-medium text-slate-600 shadow-sm'>
        <CalendarClock className='size-3.5' />
        最近 7 天 · {dateLabel}
      </span>
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
              className='hidden h-8 border-slate-200 bg-white text-xs shadow-sm md:flex lg:w-64 xl:w-80'
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
            className='hidden border-slate-200 bg-white text-slate-500 shadow-sm sm:inline-flex'
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
