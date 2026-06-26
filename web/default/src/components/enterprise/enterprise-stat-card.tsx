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
import type { LucideIcon } from 'lucide-react'

import { cn } from '@/lib/utils'

type EnterpriseStatTone =
  | 'blue'
  | 'violet'
  | 'emerald'
  | 'amber'
  | 'rose'
  | 'slate'

const toneStyles: Record<EnterpriseStatTone, string> = {
  blue: 'bg-blue-50 text-blue-700 ring-blue-100 dark:bg-blue-500/10 dark:text-blue-300 dark:ring-blue-500/20',
  violet:
    'bg-violet-50 text-violet-700 ring-violet-100 dark:bg-violet-500/10 dark:text-violet-300 dark:ring-violet-500/20',
  emerald:
    'bg-emerald-50 text-emerald-700 ring-emerald-100 dark:bg-emerald-500/10 dark:text-emerald-300 dark:ring-emerald-500/20',
  amber:
    'bg-amber-50 text-amber-700 ring-amber-100 dark:bg-amber-500/10 dark:text-amber-300 dark:ring-amber-500/20',
  rose: 'bg-rose-50 text-rose-700 ring-rose-100 dark:bg-rose-500/10 dark:text-rose-300 dark:ring-rose-500/20',
  slate:
    'bg-slate-100 text-slate-700 ring-slate-200 dark:bg-slate-500/10 dark:text-slate-300 dark:ring-slate-500/20',
}

type EnterpriseStatCardProps = {
  title: string
  value: string
  helper?: string
  trend?: string
  trendTone?: 'positive' | 'negative' | 'neutral'
  icon: LucideIcon
  tone?: EnterpriseStatTone
  loading?: boolean
}

export function EnterpriseStatCard({
  title,
  value,
  helper,
  trend,
  trendTone = 'neutral',
  icon: Icon,
  tone = 'blue',
  loading = false,
}: EnterpriseStatCardProps) {
  return (
    <article className='enterprise-stat-card bg-card relative min-h-[92px] overflow-hidden rounded-md border p-3 shadow-[0_1px_2px_rgb(15_23_42/0.04)]'>
      <div className='bg-primary/80 pointer-events-none absolute inset-y-0 left-0 w-0.5' />
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0'>
          <p className='text-muted-foreground truncate text-[11px] font-medium'>
            {title}
          </p>
          {loading ? (
            <div className='bg-muted mt-2 h-7 w-24 animate-pulse rounded-md' />
          ) : (
            <p className='text-foreground mt-1 truncate text-[22px] leading-7 font-semibold tabular-nums'>
              {value}
            </p>
          )}
        </div>
        <span
          className={cn(
            'flex size-8 shrink-0 items-center justify-center rounded-md ring-1',
            toneStyles[tone]
          )}
        >
          <Icon className='size-4' strokeWidth={1.8} />
        </span>
      </div>
      <div className='mt-2 flex min-h-4 items-center gap-1.5 text-[11px]'>
        {trend != null && (
          <span
            className={cn(
              'font-semibold',
              trendTone === 'positive' &&
                'text-emerald-600 dark:text-emerald-400',
              trendTone === 'negative' && 'text-rose-600 dark:text-rose-400',
              trendTone === 'neutral' && 'text-foreground/70'
            )}
          >
            {trend}
          </span>
        )}
        {helper != null && (
          <span className='text-muted-foreground truncate'>{helper}</span>
        )}
      </div>
    </article>
  )
}
