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
import { getRouteApi, useNavigate } from '@tanstack/react-router'
import { useCallback, useMemo } from 'react'

import { EnterprisePageHeader, EnterprisePanel } from '@/components/enterprise'
import { SectionPageLayout } from '@/components/layout'
import type { NavGroup } from '@/components/layout/types'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { EnterpriseUsageAnalytics } from '@/features/enterprise/usage-analytics'
import { CacheStatsDialog } from '@/features/system-settings/general/channel-affinity/cache-stats-dialog'
import { useIsAdmin } from '@/hooks/use-admin'
import { useSidebarConfig } from '@/hooks/use-sidebar-config'

import { UserInfoDialog } from './components/dialogs/user-info-dialog'
import {
  UsageLogsProvider,
  useUsageLogsContext,
} from './components/usage-logs-provider'
import { UsageLogsTable } from './components/usage-logs-table'
import { PersonalUsageDashboard } from './personal-usage-dashboard'
import {
  isUsageLogsSectionId,
  USAGE_LOGS_DEFAULT_SECTION,
  type UsageLogsSectionId,
} from './section-registry'

const route = getRouteApi('/_authenticated/usage-logs/$section')
const TASK_LOG_SECTIONS = ['drawing', 'task'] as const

const SECTION_META: Record<
  UsageLogsSectionId,
  { title: string; description: string; panelTitle: string }
> = {
  common: {
    title: '用量与成本日志',
    description: '按客户、部门、模型、渠道追踪请求、Token 与成本',
    panelTitle: '用量日志',
  },
  drawing: {
    title: '绘图任务日志',
    description: '跟踪绘图代理任务的提交、队列、结果与失败原因',
    panelTitle: '绘图任务列表',
  },
  task: {
    title: '异步任务日志',
    description: '跟踪音乐、视频等异步生成任务的提交、渠道、进度与结果',
    panelTitle: '异步任务列表',
  },
}

function UsageLogsContent() {
  const navigate = useNavigate()
  const isAdmin = useIsAdmin()
  const params = route.useParams()
  const activeCategory: UsageLogsSectionId =
    params.section && isUsageLogsSectionId(params.section)
      ? params.section
      : USAGE_LOGS_DEFAULT_SECTION
  const {
    selectedUserId,
    userInfoDialogOpen,
    setUserInfoDialogOpen,
    affinityTarget,
    affinityDialogOpen,
    setAffinityDialogOpen,
  } = useUsageLogsContext()
  const tabNavGroups = useMemo<NavGroup[]>(
    () => [
      {
        title: '任务日志',
        items: TASK_LOG_SECTIONS.map((section) => ({
          title: SECTION_META[section].title,
          url: `/usage-logs/${section}`,
        })),
      },
    ],
    []
  )
  const filteredTabGroups = useSidebarConfig(tabNavGroups)
  const visibleSections = useMemo(
    () =>
      (filteredTabGroups[0]?.items ?? [])
        .map((item) => {
          if (!('url' in item) || typeof item.url !== 'string') return null
          return item.url.split('/').pop() ?? null
        })
        .filter((section): section is UsageLogsSectionId =>
          Boolean(section && isUsageLogsSectionId(section))
        ),
    [filteredTabGroups]
  )

  const handleSectionChange = useCallback(
    (section: string) => {
      void navigate({
        to: '/usage-logs/$section',
        params: { section: section as UsageLogsSectionId },
      })
    },
    [navigate]
  )

  const pageMeta = SECTION_META[activeCategory]
  const showTaskSwitcher =
    activeCategory !== 'common' && visibleSections.length > 1
  let content = null

  if (isAdmin && activeCategory === 'common') {
    content = (
      <div className='h-full min-h-0 overflow-auto'>
        <EnterpriseUsageAnalytics
          classicContent={
            <div className='h-[620px] min-h-0'>
              <UsageLogsTable logCategory={activeCategory} />
            </div>
          }
        />
      </div>
    )
  } else if (activeCategory === 'common') {
    content = (
      <div className='h-full min-h-0 overflow-auto'>
        <PersonalUsageDashboard />
      </div>
    )
  } else {
    content = (
      <div className='mx-auto flex h-full max-w-[1586px] flex-col overflow-hidden bg-[#f6f8fb] text-slate-950'>
        <EnterprisePageHeader
          eyebrow='业务运营'
          title={pageMeta.title}
          description={pageMeta.description}
          actions={
            showTaskSwitcher ? (
              <Tabs value={activeCategory} onValueChange={handleSectionChange}>
                <TabsList className='h-8 rounded-md border border-slate-200 bg-white p-1 shadow-none'>
                  {visibleSections.map((section) => (
                    <TabsTrigger
                      key={section}
                      value={section}
                      className='h-6 rounded px-2.5 text-[12px] font-semibold text-slate-600 data-[state=active]:bg-blue-50 data-[state=active]:text-blue-700 data-[state=active]:shadow-none'
                    >
                      {SECTION_META[section].title}
                    </TabsTrigger>
                  ))}
                </TabsList>
              </Tabs>
            ) : null
          }
        />
        <EnterprisePanel
          className='flex min-h-0 flex-1 flex-col'
          bodyClassName='flex min-h-0 flex-1 flex-col p-2'
          title={pageMeta.panelTitle}
          description='按提交时间、渠道、用户、任务 ID 和状态排查异步任务执行链路'
        >
          <UsageLogsTable logCategory={activeCategory} />
        </EnterprisePanel>
      </div>
    )
  }

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Content>{content}</SectionPageLayout.Content>
      </SectionPageLayout>

      <UserInfoDialog
        userId={selectedUserId}
        open={userInfoDialogOpen}
        onOpenChange={setUserInfoDialogOpen}
      />

      <CacheStatsDialog
        open={affinityDialogOpen}
        onOpenChange={setAffinityDialogOpen}
        target={
          affinityTarget
            ? {
                rule_name: affinityTarget.rule_name || '',
                using_group:
                  affinityTarget.using_group ||
                  affinityTarget.selected_group ||
                  '',
                key_hint: affinityTarget.key_hint || '',
                key_fp: affinityTarget.key_fp || '',
              }
            : null
        }
      />
    </>
  )
}

export function UsageLogs() {
  return (
    <UsageLogsProvider>
      <UsageLogsContent />
    </UsageLogsProvider>
  )
}
