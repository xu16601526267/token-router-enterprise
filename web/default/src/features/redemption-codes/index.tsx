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
import { EnterprisePageHeader, EnterprisePanel } from '@/components/enterprise'
import { SectionPageLayout } from '@/components/layout'

import { RedemptionsDialogs } from './components/redemptions-dialogs'
import { RedemptionsPrimaryButtons } from './components/redemptions-primary-buttons'
import { RedemptionsProvider } from './components/redemptions-provider'
import { RedemptionsTable } from './components/redemptions-table'

export function Redemptions() {
  return (
    <RedemptionsProvider>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Content>
          <div className='mx-auto flex h-full max-w-[1586px] flex-col overflow-hidden bg-[#f6f8fb] text-slate-950'>
            <EnterprisePageHeader
              eyebrow='组织与计费'
              title='兑换码管理'
              description='批量生成、启停、核销与追踪客户充值兑换额度'
              actions={<RedemptionsPrimaryButtons />}
            />
            <EnterprisePanel
              className='flex min-h-0 flex-1 flex-col'
              bodyClassName='flex min-h-0 flex-1 flex-col p-2'
              title='兑换码列表'
              description='按名称、状态、额度、有效期和核销用户追踪兑换码'
            >
              <RedemptionsTable />
            </EnterprisePanel>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <RedemptionsDialogs />
    </RedemptionsProvider>
  )
}
