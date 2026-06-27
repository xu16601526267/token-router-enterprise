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
import { Shield, Key, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'
import { useDialogs } from '@/hooks/use-dialog'

import type { UserProfile } from '../types'
import { AccessTokenDialog } from './dialogs/access-token-dialog'
import { ChangePasswordDialog } from './dialogs/change-password-dialog'
import { DeleteAccountDialog } from './dialogs/delete-account-dialog'

// ============================================================================
// Profile Security Card Component
// ============================================================================

interface ProfileSecurityCardProps {
  profile: UserProfile | null
  loading: boolean
}

type DialogKey = 'password' | 'token' | 'delete'

export function ProfileSecurityCard({
  profile,
  loading,
}: ProfileSecurityCardProps) {
  const { t } = useTranslation()
  const dialogs = useDialogs<DialogKey>()

  if (loading) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
          <Skeleton className='h-6 w-32' />
          <Skeleton className='mt-2 h-4 w-48' />
        </CardHeader>
        <CardContent className='space-y-3 p-3 sm:p-5'>
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className='h-16 w-full' />
          ))}
        </CardContent>
      </Card>
    )
  }

  if (!profile) return null

  const securityActions = [
    {
      icon: Shield,
      title: t('修改密码'),
      description: t('更新登录密码，保持账号安全'),
      action: () => dialogs.open('password'),
      variant: 'default' as const,
    },
    {
      icon: Key,
      title: t('访问令牌'),
      description: t('生成和管理个人访问 Token'),
      action: () => dialogs.open('token'),
      variant: 'default' as const,
    },
    {
      icon: Trash2,
      title: t('删除账号'),
      description: t('永久删除账号和相关数据'),
      action: () => dialogs.open('delete'),
      variant: 'destructive' as const,
    },
  ]

  return (
    <>
      <TitledCard
        title={t('安全设置')}
        description={t('管理账号安全设置和访问权限')}
        icon={<Shield className='h-4 w-4' />}
        disableHoverEffect
        className='rounded-md border-slate-200 shadow-[0_1px_2px_rgb(15_23_42/0.04)]'
        headerClassName='bg-slate-50/65 p-3 !pb-3'
        contentClassName='p-3'
        titleClassName='text-[14px] leading-5 font-semibold text-slate-900'
        descriptionClassName='text-[11px] leading-4 text-slate-500'
      >
        <div className='grid grid-cols-1 gap-2.5 sm:gap-3 md:grid-cols-3'>
          {securityActions.map((item) => (
            <button
              key={item.title}
              type='button'
              onClick={item.action}
              className={`flex items-center gap-3 rounded-md border border-slate-200 bg-white p-3 text-left transition-colors hover:border-blue-200 hover:bg-blue-50/40 md:flex-col md:gap-2 md:text-center ${
                item.variant === 'destructive' ? 'border-destructive/30' : ''
              }`}
            >
              <div
                className={`rounded-md p-2 ${
                  item.variant === 'destructive'
                    ? 'bg-destructive/10 text-destructive'
                    : 'bg-muted'
                }`}
              >
                <item.icon className='h-5 w-5' />
              </div>
              <div className='min-w-0 md:contents'>
                <p className='text-sm font-medium'>{item.title}</p>
                <p className='text-muted-foreground line-clamp-1 text-xs md:line-clamp-none'>
                  {item.description}
                </p>
              </div>
            </button>
          ))}
        </div>
      </TitledCard>

      {/* Dialogs */}
      <ChangePasswordDialog
        open={dialogs.isOpen('password')}
        onOpenChange={(open) =>
          open ? dialogs.open('password') : dialogs.close('password')
        }
        username={profile.username}
      />

      <AccessTokenDialog
        open={dialogs.isOpen('token')}
        onOpenChange={(open) =>
          open ? dialogs.open('token') : dialogs.close('token')
        }
      />

      <DeleteAccountDialog
        open={dialogs.isOpen('delete')}
        onOpenChange={(open) =>
          open ? dialogs.open('delete') : dialogs.close('delete')
        }
        username={profile.username}
      />
    </>
  )
}
