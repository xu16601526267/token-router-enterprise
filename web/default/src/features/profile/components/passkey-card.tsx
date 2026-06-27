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
import { AlertTriangle, KeyRound, Loader2, ShieldAlert } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { StatusBadge } from '@/components/status-badge'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { usePasskeyManagement } from '@/features/auth/passkey'
import {
  SecureVerificationDialog,
  useSecureVerification,
  type VerificationMethod,
  type VerificationMethods,
} from '@/features/auth/secure-verification'
import dayjs from '@/lib/dayjs'

interface PasskeyCardProps {
  loading: boolean
}

export function PasskeyCard({ loading: pageLoading }: PasskeyCardProps) {
  const { t } = useTranslation()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [restrictedMethod, setRestrictedMethod] =
    useState<VerificationMethod | null>(null)

  const {
    status,
    loading,
    registering,
    removing,
    supported,
    enabled,
    lastUsed,
    register,
    remove,
  } = usePasskeyManagement()

  const {
    open: verificationOpen,
    setOpen: setVerificationOpen,
    methods: verificationMethods,
    state: verificationState,
    startVerification,
    executeVerification,
    cancel: cancelVerification,
    setCode,
    switchMethod,
    fetchVerificationMethods,
  } = useSecureVerification({
    onSuccess: () => {
      setRestrictedMethod(null)
    },
  })

  const dialogMethods = useMemo<VerificationMethods>(() => {
    if (!restrictedMethod) return verificationMethods
    return {
      ...verificationMethods,
      has2FA: restrictedMethod === '2fa' && verificationMethods.has2FA,
      hasPasskey:
        restrictedMethod === 'passkey' && verificationMethods.hasPasskey,
    }
  }, [restrictedMethod, verificationMethods])

  const handleRegister = useCallback(async () => {
    if (!supported) {
      toast.info(t('This device does not support Passkey'))
      return
    }

    const methods = await fetchVerificationMethods()
    if (!methods.has2FA) {
      // Without 2FA enabled, register directly. The browser-level Passkey prompt
      // is itself a strong proof of presence, so no extra verification is needed.
      await register()
      return
    }

    setRestrictedMethod('2fa')
    await startVerification(register, {
      preferredMethod: '2fa',
      title: t('Security verification'),
      description: t(
        'Confirm your identity with Two-factor Authentication before registering a Passkey.'
      ),
    })
  }, [fetchVerificationMethods, register, startVerification, supported, t])

  const handleRemove = useCallback(async () => {
    const methods = await fetchVerificationMethods()
    const required: VerificationMethod | null = methods.has2FA
      ? '2fa'
      : methods.hasPasskey
        ? 'passkey'
        : null

    if (!required) {
      toast.error(
        t(
          'Please enable Two-factor Authentication or Passkey before proceeding'
        )
      )
      return
    }

    if (required === 'passkey' && !methods.passkeySupported) {
      toast.info(t('This device does not support Passkey'))
      return
    }

    setConfirmOpen(false)
    setRestrictedMethod(required)
    await startVerification(remove, {
      preferredMethod: required,
      title: t('Security verification'),
      description: t(
        'Confirm your identity before removing this Passkey from your account.'
      ),
    })
  }, [fetchVerificationMethods, remove, startVerification, t])

  const handleVerificationCancel = useCallback(() => {
    setRestrictedMethod(null)
    cancelVerification()
  }, [cancelVerification])

  const handleVerificationOpenChange = useCallback(
    (next: boolean) => {
      if (!next) {
        setRestrictedMethod(null)
      }
      setVerificationOpen(next)
    },
    [setVerificationOpen]
  )

  // Adapt the hook's `Promise<unknown>` return into the dialog's
  // `void | Promise<void>` signature without losing error propagation
  // semantics (errors are surfaced via toast inside the hook).
  const handleDialogVerify = useCallback(
    async (method: VerificationMethod, code?: string) => {
      try {
        await executeVerification(method, code)
      } catch {
        // Errors are already surfaced by useSecureVerification via toast.
      }
    },
    [executeVerification]
  )

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

  const formattedLastUsed =
    lastUsed && !Number.isNaN(Date.parse(lastUsed))
      ? dayjs(lastUsed).fromNow()
      : t('暂未使用')

  const showUnsupportedNotice = !supported && !enabled

  return (
    <>
      <Card
        data-card-hover='false'
        className='gap-0 overflow-hidden rounded-md border-slate-200 py-0 shadow-[0_1px_2px_rgb(15_23_42/0.04)]'
      >
        <CardHeader className='border-b border-slate-100 bg-slate-50/65 p-3'>
          <CardTitle className='text-[14px] leading-5 font-semibold text-slate-900'>
            {t('通行密钥登录')}
          </CardTitle>
          <CardDescription className='text-[11px] leading-4 text-slate-500'>
            {t('无需输入密码，使用设备凭据完成登录。')}
          </CardDescription>
        </CardHeader>

        <CardContent className='p-3'>
          <div className='space-y-4'>
            <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between xl:flex-col 2xl:flex-row'>
              <div className='flex items-start gap-3'>
                <div className='rounded-md bg-slate-50 p-2 text-slate-600 ring-1 ring-slate-100'>
                  <KeyRound className='size-4' />
                </div>
                <div className='space-y-1'>
                  <div className='flex flex-wrap items-center gap-2'>
                    <p className='text-[13px] font-semibold text-slate-900'>
                      {t('Passkey 认证')}
                    </p>
                    <StatusBadge
                      label={enabled ? t('已启用') : t('未启用')}
                      variant={enabled ? 'success' : 'neutral'}
                      showDot
                      copyable={false}
                    />
                    {status?.backup_eligible !== undefined && (
                      <StatusBadge
                        label={
                          status.backup_eligible
                            ? status.backup_state
                              ? t('已备份')
                              : t('未备份')
                            : t('无备份')
                        }
                        variant={
                          status.backup_eligible
                            ? status.backup_state
                              ? 'success'
                              : 'warning'
                            : 'neutral'
                        }
                        showDot
                        copyable={false}
                      />
                    )}
                  </div>
                  <p className='text-xs text-slate-500'>
                    {t('最近使用:')} {formattedLastUsed}
                  </p>
                </div>
              </div>

              {!enabled && (
                <Button
                  className='w-full sm:w-auto xl:w-full 2xl:w-auto'
                  onClick={handleRegister}
                  disabled={!supported || registering}
                >
                  {registering && (
                    <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                  )}
                  {t('启用 Passkey')}
                </Button>
              )}
            </div>

            {enabled && (
              <div className='flex flex-col gap-3 border-t border-slate-100 pt-4 sm:flex-row xl:flex-col 2xl:flex-row'>
                <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
                  <AlertDialogTrigger
                    render={
                      <Button
                        variant='destructive'
                        className='flex-1'
                        disabled={removing}
                      />
                    }
                  >
                    {removing ? (
                      <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                    ) : (
                      <AlertTriangle className='mr-2 h-4 w-4' />
                    )}
                    {t('移除 Passkey')}
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>
                        {t('移除 Passkey？')}
                      </AlertDialogTitle>
                      <AlertDialogDescription>
                        {t(
                          '移除后下次登录需要使用密码。后续可以重新注册 Passkey。'
                        )}
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel disabled={removing}>
                        {t('取消')}
                      </AlertDialogCancel>
                      <AlertDialogAction
                        className='bg-destructive text-destructive-foreground'
                        disabled={removing}
                        onClick={(event) => {
                          event.preventDefault()
                          handleRemove()
                        }}
                      >
                        {t('移除')}
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            )}

            {showUnsupportedNotice && (
              <div className='flex items-start gap-3 rounded-md bg-amber-50 p-3 text-xs text-amber-700'>
                <ShieldAlert className='mt-0.5 h-4 w-4 flex-shrink-0 text-amber-500' />
                <div>
                  <p className='text-foreground font-medium'>
                    {t('当前设备不支持 Passkey')}
                  </p>
                  <p>
                    {t(
                      '请使用支持生物识别或安全密钥的浏览器和设备注册 Passkey。'
                    )}
                  </p>
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <SecureVerificationDialog
        open={verificationOpen}
        onOpenChange={handleVerificationOpenChange}
        methods={dialogMethods}
        state={verificationState}
        onVerify={handleDialogVerify}
        onCancel={handleVerificationCancel}
        onCodeChange={setCode}
        onMethodChange={switchMethod}
      />
    </>
  )
}
