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
  Bot,
  Box,
  FileText,
  FlaskConical,
  KeyRound,
  LayoutDashboard,
  ListTodo,
  MessageSquare,
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
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import type { SidebarData } from '@/components/layout/types'

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
          id: 'console',
          title: '个人工作台',
          items: [
            {
              title: '个人总览',
              url: '/dashboard/overview',
              icon: Activity,
            },
            {
              title: '用量分析',
              url: '/dashboard/models',
              icon: BarChart3,
            },
          ],
        },
        {
          id: 'development',
          title: '开发与体验',
          items: [
            {
              title: '接口密钥',
              url: '/keys',
              icon: KeyRound,
            },
            {
              title: '在线调试台',
              url: '/playground',
              icon: FlaskConical,
            },
            {
              title: '智能对话',
              icon: MessageSquare,
              type: 'chat-presets',
            },
            {
              title: '模型与价格',
              url: '/pricing',
              icon: Bot,
            },
          ],
        },
        {
          id: 'usage',
          title: '调用记录',
          items: [
            {
              title: '用量日志',
              url: '/usage-logs/common',
              icon: FileText,
            },
            {
              title: '任务日志',
              url: '/usage-logs/task',
              activeUrls: ['/usage-logs/drawing'],
              configUrls: ['/usage-logs/drawing', '/usage-logs/task'],
              icon: ListTodo,
            },
          ],
        },
        {
          id: 'personal',
          title: '账户服务',
          items: [
            {
              title: '钱包与充值',
              url: '/wallet',
              icon: Wallet,
            },
            {
              title: '个人资料',
              url: '/profile',
              icon: UserRound,
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
          {
            title: '智能对话',
            icon: MessageSquare,
            type: 'chat-presets',
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
