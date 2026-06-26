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
    <div className='enterprise-page-header bg-card relative overflow-hidden rounded-md border px-4 py-3 shadow-[0_1px_2px_rgb(15_23_42/0.04)] sm:px-5'>
      <div className='bg-primary absolute inset-y-0 left-0 w-1' />
      <div className='relative flex flex-col justify-between gap-3 sm:flex-row sm:items-center'>
        <div className='min-w-0'>
          {eyebrow != null && (
            <p className='text-primary mb-1 text-[11px] font-semibold uppercase'>
              {eyebrow}
            </p>
          )}
          <h1 className='text-foreground text-[20px] leading-7 font-semibold sm:text-[22px]'>
            {title}
          </h1>
          <p className='text-muted-foreground mt-1 max-w-3xl text-xs leading-5 sm:text-[13px]'>
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
