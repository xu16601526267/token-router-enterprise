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
import type { HTMLAttributes, ReactNode } from 'react'
import { cn } from '@/lib/utils'

type EnterprisePanelProps = HTMLAttributes<HTMLDivElement> & {
  title?: ReactNode
  description?: ReactNode
  action?: ReactNode
  bodyClassName?: string
}

export function EnterprisePanel({
  title,
  description,
  action,
  bodyClassName,
  className,
  children,
  ...props
}: EnterprisePanelProps) {
  return (
    <section
      className={cn(
        'enterprise-panel overflow-hidden rounded-2xl border bg-card/95 shadow-[0_1px_2px_rgb(15_23_42/0.03),0_12px_36px_rgb(15_23_42/0.04)] backdrop-blur-sm',
        className
      )}
      {...props}
    >
      {(title != null || description != null || action != null) && (
        <header className='flex min-h-14 items-start justify-between gap-4 border-b border-border/65 px-4 py-3.5 sm:px-5'>
          <div className='min-w-0'>
            {title != null && (
              <h3 className='truncate text-sm font-semibold tracking-tight text-foreground sm:text-[15px]'>
                {title}
              </h3>
            )}
            {description != null && (
              <p className='mt-0.5 text-xs leading-5 text-muted-foreground'>
                {description}
              </p>
            )}
          </div>
          {action != null && <div className='shrink-0'>{action}</div>}
        </header>
      )}
      <div className={cn('p-4 sm:p-5', bodyClassName)}>{children}</div>
    </section>
  )
}
