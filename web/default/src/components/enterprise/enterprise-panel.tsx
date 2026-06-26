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
        'enterprise-panel overflow-hidden rounded-md border bg-card shadow-[0_1px_2px_rgb(15_23_42/0.04)]',
        className
      )}
      {...props}
    >
      {(title != null || description != null || action != null) && (
        <header className='border-border/80 bg-muted/20 flex min-h-11 items-center justify-between gap-3 border-b px-3 py-2.5 sm:px-4'>
          <div className='min-w-0'>
            {title != null && (
              <h3 className='text-foreground truncate text-[13px] font-semibold sm:text-sm'>
                {title}
              </h3>
            )}
            {description != null && (
              <p className='text-muted-foreground mt-0.5 text-[11px] leading-4'>
                {description}
              </p>
            )}
          </div>
          {action != null && <div className='shrink-0'>{action}</div>}
        </header>
      )}
      <div className={cn('p-3 sm:p-4', bodyClassName)}>{children}</div>
    </section>
  )
}
