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
  blue: 'bg-blue-500/10 text-blue-600 dark:text-blue-300',
  violet: 'bg-violet-500/10 text-violet-600 dark:text-violet-300',
  emerald: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-300',
  amber: 'bg-amber-500/10 text-amber-600 dark:text-amber-300',
  rose: 'bg-rose-500/10 text-rose-600 dark:text-rose-300',
  slate: 'bg-slate-500/10 text-slate-600 dark:text-slate-300',
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
    <article className='enterprise-stat-card group relative min-h-28 overflow-hidden rounded-2xl border bg-card/95 p-4 shadow-[0_1px_2px_rgb(15_23_42/0.03),0_10px_30px_rgb(15_23_42/0.035)] transition-transform duration-200 hover:-translate-y-0.5 sm:p-4.5'>
      <div className='pointer-events-none absolute inset-x-6 -top-px h-px bg-linear-to-r from-transparent via-primary/25 to-transparent opacity-0 transition-opacity group-hover:opacity-100' />
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0'>
          <p className='truncate text-xs font-medium text-muted-foreground'>
            {title}
          </p>
          {loading ? (
            <div className='mt-2 h-8 w-24 animate-pulse rounded-lg bg-muted' />
          ) : (
            <p className='mt-1.5 truncate text-2xl font-semibold tracking-[-0.035em] text-foreground'>
              {value}
            </p>
          )}
        </div>
        <span
          className={cn(
            'flex size-9 shrink-0 items-center justify-center rounded-xl',
            toneStyles[tone]
          )}
        >
          <Icon className='size-4.5' strokeWidth={1.8} />
        </span>
      </div>
      <div className='mt-3 flex min-h-4 items-center gap-1.5 text-[11px]'>
        {trend != null && (
          <span
            className={cn(
              'font-semibold',
              trendTone === 'positive' && 'text-emerald-600 dark:text-emerald-400',
              trendTone === 'negative' && 'text-rose-600 dark:text-rose-400',
              trendTone === 'neutral' && 'text-foreground/70'
            )}
          >
            {trend}
          </span>
        )}
        {helper != null && (
          <span className='truncate text-muted-foreground'>{helper}</span>
        )}
      </div>
    </article>
  )
}
