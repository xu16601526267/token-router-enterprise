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
import { Building2, CalendarClock, CircleDot } from 'lucide-react'

import { ConfigDrawer } from '@/components/config-drawer'
import { LanguageSwitcher } from '@/components/language-switcher'
import { NotificationPopover } from '@/components/notification-popover'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { Search } from '@/components/search'
import { useNotifications } from '@/hooks/use-notifications'
import { useTopNavLinks } from '@/hooks/use-top-nav-links'
import { ROLE } from '@/lib/roles'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import { defaultTopNavLinks } from '../config/top-nav.config'
import type { TopNavLink } from '../types'
import { Header } from './header'
import { SystemBrand } from './system-brand'
import { TopNav } from './top-nav'

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
  const workspace =
    user?.group?.trim() || (isAdmin ? '企业工作区' : '个人工作区')
  const dateLabel = new Intl.DateTimeFormat('zh-CN', {
    weekday: 'short',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date())
  const envLabel =
    import.meta.env.MODE === 'production' ? '生产环境' : '测试环境'

  return (
    <div className='ms-2 hidden min-w-0 items-center gap-2 xl:flex'>
      <span className='bg-muted/70 text-muted-foreground inline-flex h-7 max-w-52 items-center gap-1.5 rounded-md border px-2 text-xs'>
        <Building2 className='text-primary size-3.5 shrink-0' />
        <span className='truncate'>{workspace}</span>
      </span>
      <span
        className={cn(
          'inline-flex h-7 items-center gap-1.5 rounded-md border px-2 text-xs',
          isAdmin
            ? 'border-emerald-500/20 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
            : 'border-blue-500/20 bg-blue-500/10 text-blue-700 dark:text-blue-300'
        )}
      >
        <CircleDot className='size-3.5' />
        {envLabel}
      </span>
      <span className='text-muted-foreground inline-flex h-7 items-center gap-1.5 rounded-md border px-2 text-xs'>
        <CalendarClock className='size-3.5' />
        {dateLabel}
      </span>
    </div>
  )
}

export function AppHeader({
  navLinks = defaultTopNavLinks,
  showTopNav = true,
  leftContent,
  showSearch = true,
  rightContent,
  showNotifications = true,
  showConfigDrawer = true,
  showProfileDropdown = true,
}: AppHeaderProps) {
  // Prioritize dynamically generated links from backend
  const dynamicLinks = useTopNavLinks()
  const links = dynamicLinks.length > 0 ? dynamicLinks : navLinks

  // Notifications hook
  const notifications = useNotifications()

  return (
    <Header>
      <SystemBrand variant='inline' />
      <EnterpriseHeaderContext />

      {leftContent ? (
        <div className='ms-2 flex items-center'>{leftContent}</div>
      ) : null}

      {rightContent ?? (
        <div className='ms-auto flex items-center gap-1 sm:gap-2'>
          {showTopNav && (
            <div className='me-1 hidden lg:block'>
              <TopNav links={links} />
            </div>
          )}
          {showSearch && <Search />}
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
          <LanguageSwitcher />
          {showConfigDrawer && <ConfigDrawer />}
          {showProfileDropdown && <ProfileDropdown />}
        </div>
      )}
    </Header>
  )
}
