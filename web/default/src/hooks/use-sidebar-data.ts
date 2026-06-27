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
import {
  Activity,
  BadgeDollarSign,
  BarChart3,
  Box,
  Building2,
  FileText,
  FlaskConical,
  KeyRound,
  LayoutDashboard,
  ListTodo,
  Radio,
  ReceiptText,
  Route,
  Settings,
  ShieldCheck,
  Ticket,
  UserRound,
  Users,
  Wallet,
  Workflow,
} from 'lucide-react'

import type { SidebarData } from '@/components/layout/types'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

/**
 * 根导航使用中文产品文案，并根据角色提供两套信息架构：
 * - 管理员：B 端经营、资源治理、组织与计费优先；
 * - 普通用户：个人工作台、开发接入与自助服务优先。
 *
 * URL 与旧系统保持一致，因此不会破坏既有权限、收藏和接口调用。
 */
export function useSidebarData(): SidebarData {
  const role = useAuthStore((state) => state.auth.user?.role ?? ROLE.USER)
  const isAdmin = role >= ROLE.ADMIN

  if (!isAdmin) {
    return {
      navGroups: [
        {
          id: 'overview',
          title: '总览',
          items: [
            {
              title: '总览',
              url: '/dashboard/overview',
              icon: LayoutDashboard,
            },
            {
              title: '仪表盘',
              url: '/dashboard/models',
              icon: Activity,
            },
          ],
        },
        {
          id: 'development',
          title: '开发者工具',
          items: [
            {
              title: 'API Keys',
              url: '/keys',
              icon: KeyRound,
            },
            {
              title: 'Playground',
              url: '/playground',
              icon: FlaskConical,
            },
            {
              title: 'Models',
              url: '/models',
              icon: Box,
            },
            {
              title: 'Usage Logs',
              url: '/usage-logs/common',
              icon: FileText,
            },
          ],
        },
        {
          id: 'personal',
          title: '账户与资源',
          items: [
            {
              title: 'Wallet',
              url: '/wallet',
              icon: Wallet,
            },
            {
              title: '订阅与账单',
              url: '/subscriptions',
              icon: ReceiptText,
            },
          ],
        },
        {
          id: 'profile',
          title: '个人中心',
          items: [
            {
              title: 'Profile',
              url: '/profile',
              icon: UserRound,
            },
            {
              title: '安全设置',
              url: '/profile',
              icon: ShieldCheck,
            },
          ],
        },
      ],
    }
  }

  return {
    navGroups: [
      {
        id: 'enterprise-console',
        title: '企业工作台',
        items: [
          {
            title: '企业总览',
            url: '/dashboard/overview',
            icon: LayoutDashboard,
          },
          {
            title: 'B端客户',
            url: '/tenants',
            icon: Building2,
          },
          {
            title: '模型经营分析',
            url: '/dashboard/models',
            icon: BarChart3,
          },
          {
            title: '流量链路分析',
            url: '/dashboard/flow',
            icon: Workflow,
          },
          {
            title: '用户用量分析',
            url: '/dashboard/users',
            icon: Activity,
          },
        ],
      },
      {
        id: 'console',
        title: '业务运营',
        items: [
          {
            title: '接口密钥',
            url: '/keys',
            icon: KeyRound,
          },
          {
            title: '用量与成本日志',
            url: '/usage-logs/common',
            icon: FileText,
          },
          {
            title: '异步任务日志',
            url: '/usage-logs/task',
            activeUrls: ['/usage-logs/drawing'],
            configUrls: ['/usage-logs/drawing', '/usage-logs/task'],
            icon: ListTodo,
          },
        ],
      },
      {
        id: 'resource-governance',
        title: '资源与路由',
        items: [
          {
            title: '渠道与供应商',
            url: '/channels',
            icon: Radio,
          },
          {
            title: '智能路由控制塔',
            url: '/token-router',
            icon: Route,
          },
          {
            title: '模型资产',
            url: '/models/metadata',
            icon: Box,
          },
        ],
      },
      {
        id: 'admin',
        title: '组织与计费',
        items: [
          {
            title: '用户与权限',
            url: '/users',
            icon: Users,
          },
          {
            title: '计费与结算',
            url: '/subscriptions',
            icon: BadgeDollarSign,
          },
          {
            title: '兑换码管理',
            url: '/redemption-codes',
            icon: Ticket,
          },
          {
            title: '钱包与充值',
            url: '/wallet',
            icon: Wallet,
          },
          {
            title: '系统设置',
            url: '/system-settings/site',
            activeUrls: ['/system-settings'],
            icon: Settings,
          },
        ],
      },
      {
        id: 'chat',
        title: '开发与体验',
        items: [
          {
            title: '在线调试台',
            url: '/playground',
            icon: FlaskConical,
          },
        ],
      },
      {
        id: 'personal',
        title: '个人中心',
        items: [
          {
            title: '个人资料',
            url: '/profile',
            icon: UserRound,
          },
          {
            title: '安全与身份',
            url: '/profile',
            icon: ShieldCheck,
          },
          {
            title: '账单凭证',
            url: '/wallet',
            icon: ReceiptText,
          },
        ],
      },
    ],
  }
}
