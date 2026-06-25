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
import type { ReactNode } from 'react'

export function EnterprisePageHeader({
  eyebrow,
  title,
  description,
  actions,
}: {
  eyebrow?: string
  title: string
  description: string
  actions?: ReactNode
}) {
  return (
    <div className='relative overflow-hidden rounded-2xl border bg-[linear-gradient(122deg,color-mix(in_oklch,var(--card)_98%,var(--primary)_2%)_0%,var(--card)_54%,color-mix(in_oklch,var(--card)_93%,var(--primary)_7%)_100%)] px-5 py-5 shadow-[0_14px_45px_rgb(15_23_42/0.045)] sm:px-6 sm:py-6'>
      <div className='pointer-events-none absolute -top-28 right-[-5rem] size-64 rounded-full bg-primary/8 blur-3xl' />
      <div className='pointer-events-none absolute right-[20%] -bottom-28 size-52 rounded-full bg-violet-500/8 blur-3xl' />
      <div className='relative flex flex-col justify-between gap-4 sm:flex-row sm:items-end'>
        <div className='min-w-0'>
          {eyebrow != null && (
            <p className='mb-2 text-[11px] font-semibold tracking-[0.18em] text-primary uppercase'>
              {eyebrow}
            </p>
          )}
          <h1 className='text-2xl font-semibold tracking-[-0.035em] text-foreground sm:text-[28px]'>
            {title}
          </h1>
          <p className='mt-1.5 max-w-2xl text-sm leading-6 text-muted-foreground'>
            {description}
          </p>
        </div>
        {actions != null && (
          <div className='flex shrink-0 flex-wrap items-center gap-2'>
            {actions}
          </div>
        )}
      </div>
    </div>
  )
}
