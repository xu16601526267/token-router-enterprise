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
import { Activity, BarChart3, WalletCards } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { getUserAvatarFallback, getUserAvatarStyle } from '@/lib/avatar'
import { formatCompactNumber, formatQuota } from '@/lib/format'
import { getRoleLabel } from '@/lib/roles'

import { getDisplayName } from '../lib'
import type { UserProfile } from '../types'

// ============================================================================
// Profile Header Component
// ============================================================================

interface ProfileHeaderProps {
  profile: UserProfile | null
  loading: boolean
}

export function ProfileHeader({ profile, loading }: ProfileHeaderProps) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardContent className='p-4 sm:p-5'>
          <div className='flex flex-col items-center gap-4 text-center sm:flex-row sm:text-left'>
            <Skeleton className='h-16 w-16 rounded-2xl' />
            <div className='space-y-3'>
              <div className='flex flex-col items-center gap-2 sm:flex-row sm:justify-start'>
                <Skeleton className='h-8 w-48' />
                <Skeleton className='h-5 w-16' />
              </div>
              <div className='flex flex-col items-center gap-1 sm:flex-row sm:justify-start sm:gap-4'>
                <Skeleton className='h-4 w-24' />
                <Skeleton className='h-4 w-40' />
                <Skeleton className='h-4 w-20' />
              </div>
            </div>
          </div>
        </CardContent>
        <div className='border-t'>
          <div className='divide-border/60 grid grid-cols-1 divide-y sm:grid-cols-3 sm:divide-x sm:divide-y-0'>
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className='px-4 py-3.5 sm:px-5 sm:py-4'>
                <Skeleton className='h-3.5 w-20' />
                <Skeleton className='mt-2 h-7 w-28' />
                <Skeleton className='mt-1.5 h-3.5 w-24' />
              </div>
            ))}
          </div>
        </div>
      </Card>
    )
  }

  if (!profile) return null

  const displayName = getDisplayName(profile)
  const avatarName = profile.username || displayName
  const avatarFallback = getUserAvatarFallback(avatarName)
  const avatarFallbackStyle = getUserAvatarStyle(avatarName)
  const roleLabel = getRoleLabel(profile.role)
  const stats = [
    {
      label: t('当前余额'),
      value: formatQuota(profile.quota),
      description: t('剩余额度'),
      icon: WalletCards,
    },
    {
      label: t('累计用量'),
      value: formatQuota(profile.used_quota),
      description: t('已消费额度'),
      icon: BarChart3,
    },
    {
      label: t('API 请求'),
      value: formatCompactNumber(profile.request_count),
      description: t('累计请求次数'),
      icon: Activity,
    },
  ]

  return (
    <Card
      data-card-hover='false'
      className='gap-0 overflow-hidden rounded-md border-slate-200 py-0 shadow-[0_1px_2px_rgb(15_23_42/0.04)]'
    >
      <CardContent className='p-3 sm:p-4'>
        <div className='flex items-center gap-3 text-left sm:gap-4'>
          <Avatar className='ring-background h-12 w-12 rounded-md text-sm ring-2 sm:h-14 sm:w-14 sm:text-base sm:ring-2'>
            <AvatarFallback
              className='rounded-md font-semibold text-white'
              style={avatarFallbackStyle}
            >
              {avatarFallback}
            </AvatarFallback>
          </Avatar>

          <div className='min-w-0 flex-1 space-y-1.5 sm:space-y-3'>
            <div className='flex min-w-0 items-center gap-2'>
              <h1 className='truncate text-[22px] leading-7 font-semibold text-slate-950'>
                {displayName}
              </h1>
              <StatusBadge
                label={roleLabel}
                variant='neutral'
                copyable={false}
              />
              <StatusBadge
                label={`${t('用户 ID')} ${profile.id}`}
                variant='info'
                copyText={String(profile.id)}
              />
            </div>

            <div className='flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs text-slate-500 sm:gap-x-3'>
              <span className='truncate'>@{profile.username}</span>
              {profile.email && (
                <>
                  <span>•</span>
                  <span className='truncate'>{profile.email}</span>
                </>
              )}
              {profile.group && (
                <>
                  <span>•</span>
                  <span className='truncate'>{profile.group}</span>
                </>
              )}
            </div>
          </div>
        </div>
      </CardContent>
      <div className='border-t'>
        <div className='grid grid-cols-3 divide-x divide-slate-100'>
          {stats.map((item) => (
            <div key={item.label} className='min-w-0 px-3 py-2.5 sm:px-4 sm:py-3'>
              <div className='flex items-center gap-2'>
                <item.icon className='size-3.5 shrink-0 text-slate-400' />
                <div className='truncate text-[11px] font-semibold text-slate-500'>
                  {item.label}
                </div>
              </div>

              <div className='mt-1 truncate font-mono text-xl font-bold tracking-tight text-slate-950 tabular-nums'>
                {item.value}
              </div>
              <div className='mt-0.5 hidden text-xs text-slate-400 md:block'>
                {item.description}
              </div>
            </div>
          ))}
        </div>
      </div>
    </Card>
  )
}
