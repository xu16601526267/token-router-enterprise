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
import { memo } from 'react'

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from '@/components/ui/card'
import { cn } from '@/lib/utils'

type SettingsCardProps = {
  title: string
  description?: string
  children: React.ReactNode
  className?: string
}

export const SettingsCard = memo(function SettingsCard({
  title,
  description,
  children,
  className,
}: SettingsCardProps) {
  return (
    <Card
      className={cn(
        'overflow-hidden rounded-md border bg-card shadow-[0_1px_2px_rgb(15_23_42/0.04)]',
        className
      )}
    >
      <CardHeader className='border-border/80 bg-muted/20 px-4 py-3'>
        <CardTitle className='text-sm leading-5 font-semibold'>
          {title}
        </CardTitle>
        {description && (
          <CardDescription className='text-xs leading-4'>
            {description}
          </CardDescription>
        )}
      </CardHeader>
      <CardContent className='p-4'>{children}</CardContent>
    </Card>
  )
})
