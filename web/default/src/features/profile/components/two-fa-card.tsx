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
import { Shield, AlertTriangle, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useDialogs } from '@/hooks/use-dialog'

import { useTwoFA } from '../hooks'
import { TwoFABackupDialog } from './dialogs/two-fa-backup-dialog'
import { TwoFADisableDialog } from './dialogs/two-fa-disable-dialog'
import { TwoFASetupDialog } from './dialogs/two-fa-setup-dialog'

// ============================================================================
// Two-Factor Authentication Card Component
// ============================================================================

interface TwoFACardProps {
  loading: boolean
}

type DialogKey = 'setup' | 'disable' | 'backup'

export function TwoFACard({ loading: pageLoading }: TwoFACardProps) {
  const { t } = useTranslation()
  const { status, loading, refetch } = useTwoFA(!pageLoading)
  const dialogs = useDialogs<DialogKey>()

  if (pageLoading || loading) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardHeader className='p-3 sm:p-5'>
          <Skeleton className='h-6 w-48' />
          <Skeleton className='mt-2 h-4 w-64' />
        </CardHeader>
        <CardContent className='p-3 sm:p-5'>
          <Skeleton className='h-20 w-full' />
        </CardContent>
      </Card>
    )
  }

  return (
    <>
      <Card
        data-card-hover='false'
        className='gap-0 overflow-hidden rounded-md border-slate-200 py-0 shadow-[0_1px_2px_rgb(15_23_42/0.04)]'
      >
        <CardHeader className='border-b border-slate-100 bg-slate-50/65 p-3'>
          <CardTitle className='text-[14px] leading-5 font-semibold text-slate-900'>
            {t('双因素认证')}
          </CardTitle>
          <CardDescription className='text-[11px] leading-4 text-slate-500'>
            {t('为账号增加额外安全校验层。')}
          </CardDescription>
        </CardHeader>

        <CardContent className='p-3'>
          <div className='space-y-4'>
            {/* Status Section */}
            <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between xl:flex-col 2xl:flex-row'>
              <div className='flex items-start gap-3'>
                <div className='rounded-md bg-slate-50 p-2 text-slate-600 ring-1 ring-slate-100'>
                  <Shield className='size-4' />
                </div>
                <div className='space-y-1'>
                  <div className='flex items-center gap-2'>
                    <p className='text-[13px] font-semibold text-slate-900'>
                      {t('两步验证')}
                    </p>
                    {status.enabled ? (
                      <StatusBadge
                        label={t('已启用')}
                        variant='success'
                        showDot
                        copyable={false}
                      />
                    ) : (
                      <StatusBadge
                        label={t('未启用')}
                        variant='neutral'
                        showDot
                        copyable={false}
                      />
                    )}
                    {status.locked && (
                      <StatusBadge
                        label={t('已锁定')}
                        variant='danger'
                        showDot
                        copyable={false}
                      />
                    )}
                  </div>
                  <p className='text-xs text-slate-500'>
                    {status.enabled
                      ? t('剩余备用码：{{count}}', {
                          count: status.backup_codes_remaining,
                        })
                      : t('为账号增加额外安全校验层。')}
                  </p>
                </div>
              </div>

              {!status.enabled && (
                <Button
                  className='w-full sm:w-auto xl:w-full 2xl:w-auto'
                  onClick={() => dialogs.open('setup')}
                >
                  {t('启用')}
                </Button>
              )}
            </div>

            {/* Actions Section - Only show when enabled */}
            {status.enabled && (
              <div className='flex flex-col gap-3 border-t border-slate-100 pt-4 sm:flex-row xl:flex-col 2xl:flex-row'>
                <Button
                  variant='outline'
                  className='flex-1'
                  onClick={() => dialogs.open('backup')}
                >
                  <RefreshCw className='mr-2 h-4 w-4' />
                  {t('重新生成备用码')}
                </Button>
                <Button
                  variant='destructive'
                  className='flex-1'
                  onClick={() => dialogs.open('disable')}
                >
                  <AlertTriangle className='mr-2 h-4 w-4' />
                  {t('关闭 2FA')}
                </Button>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Dialogs */}
      <TwoFASetupDialog
        open={dialogs.isOpen('setup')}
        onOpenChange={(open) =>
          open ? dialogs.open('setup') : dialogs.close('setup')
        }
        onSuccess={refetch}
      />

      <TwoFADisableDialog
        open={dialogs.isOpen('disable')}
        onOpenChange={(open) =>
          open ? dialogs.open('disable') : dialogs.close('disable')
        }
        onSuccess={refetch}
      />

      <TwoFABackupDialog
        open={dialogs.isOpen('backup')}
        onOpenChange={(open) =>
          open ? dialogs.open('backup') : dialogs.close('backup')
        }
        onSuccess={refetch}
      />
    </>
  )
}
