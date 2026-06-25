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
    <div className='bg-card relative overflow-hidden rounded-2xl border px-5 py-5 shadow-[0_1px_2px_rgb(15_23_42/0.03),0_12px_36px_rgb(15_23_42/0.04)] sm:px-6 sm:py-6'>
      <div className='absolute inset-x-0 top-0 h-1 bg-[linear-gradient(90deg,var(--primary),color-mix(in_oklch,var(--primary)_40%,transparent),transparent)]' />
      <div className='relative flex flex-col justify-between gap-4 sm:flex-row sm:items-end'>
        <div className='min-w-0'>
          {eyebrow != null && (
            <p className='text-primary mb-2 text-[11px] font-semibold tracking-[0.18em] uppercase'>
              {eyebrow}
            </p>
          )}
          <h1 className='text-foreground text-2xl font-semibold tracking-[-0.035em] sm:text-[28px]'>
            {title}
          </h1>
          <p className='text-muted-foreground mt-1.5 max-w-2xl text-sm leading-6'>
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
