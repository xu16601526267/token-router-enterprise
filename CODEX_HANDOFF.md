# Token Router 企业版改造：Codex 接手说明

## 1. 交付内容

本目录是原仓库的完整源码快照，已经合入当前阶段的企业版前后端改造。未包含 `node_modules`、编译产物、数据库文件或本地密钥。

视觉目标必须以 `docs/codex-handoff/ui-reference/` 下 8 张 Image 2 设计图为准，不要退回旧版通用后台风格。

左侧导航要求：

- 全中文显示；
- 管理员进入 B 端企业后台；
- 普通用户保留 C 端个人工作台；
- 原有功能、路由和数据结构尽量兼容，不做无必要的破坏性重写。

## 2. 技术栈

- 后端：Go、Gin、GORM
- 前端：React 19、TypeScript、TanStack Router、TanStack Query、Tailwind、Base UI / shadcn 风格组件
- 前端包管理：Bun（锁文件位于 `web/bun.lock`）
- 项目 Go 版本：见 `go.mod`，当前要求 Go 1.25.1

## 3. 当前新增后端文件

- `controller/enterprise.go`
- `controller/enterprise_operations.go`
- `dto/enterprise.go`
- `model/enterprise_token.go`
- `service/enterprise_api_key.go`
- `service/enterprise_control_tower.go`
- `service/enterprise_channels.go`
- `service/enterprise_usage.go`
- `service/enterprise_users.go`
- `service/enterprise_billing.go`

企业接口路由集中在 `router/api-router.go` 的 `/api/enterprise/*` 管理员路由组。

## 4. 当前新增前端文件

企业设计基础组件：

- `web/default/src/components/enterprise/`

企业页面及数据层：

- `web/default/src/features/enterprise/enterprise-overview.tsx`
- `web/default/src/features/enterprise/personal-workbench.tsx`
- `web/default/src/features/enterprise/usage-analytics.tsx`
- `web/default/src/features/enterprise/users-governance.tsx`
- `web/default/src/features/enterprise/billing-center.tsx`
- `web/default/src/features/enterprise/api.ts`
- `web/default/src/features/enterprise/types.ts`

优先级 1—3 页面：

- `web/default/src/features/token-router/components/control-tower.tsx`
- `web/default/src/features/keys/enterprise-api-keys.tsx`
- `web/default/src/features/channels/enterprise-channels-center.tsx`

接入点：

- `web/default/src/features/dashboard/components/overview/overview-dashboard.tsx`
- `web/default/src/features/token-router/index.tsx`
- `web/default/src/features/keys/index.tsx`
- `web/default/src/features/channels/index.tsx`
- `web/default/src/features/usage-logs/index.tsx`
- `web/default/src/features/users/index.tsx`
- `web/default/src/features/subscriptions/index.tsx`
- `web/default/src/hooks/use-sidebar-data.ts`

## 5. Image 2 设计图与页面映射

1. `01-企业总览.png` → 企业管理员首页
2. `02-渠道与供应商中心.png` → Channels 企业视图
3. `03-Token-Router控制塔.png` → Token Router 控制塔
4. `04-API-Keys与客户接入.png` → 企业 API Key / 客户接入
5. `05-用量日志与成本分析.png` → Usage Logs 企业分析视图
6. `06-用户团队与权限.png` → Users 企业治理视图
7. `07-计费与结算中心.png` → Subscriptions / Billing 企业视图
8. `08-个人工作台.png` → 普通用户 C 端首页

视觉原则：深色企业侧栏、浅色内容区、蓝紫主色、圆角卡片、专业图表、高信息密度、明确的审批/SLA/成本/治理信息层级。

## 6. 继续开发时必须先做的验证

```bash
# 前端
cd web
bun install
cd default
bun run typecheck
bun run build

# 后端（需要 Go 1.25.1）
cd ../../
go test ./...
go build ./...
```

然后逐项核对：

- 企业接口响应结构与前端 TypeScript 类型完全一致；
- 所有企业页均使用真实接口，不使用静态 mock 作为最终数据；
- 加载、空数据、错误、无权限、分页和筛选状态齐全；
- 管理员与普通用户的导航、首页和权限分流正确；
- 旧版功能仍可访问，数据库升级无破坏性；
- 桌面端优先，同时补齐常用宽度下的响应式表现。

## 7. 当前状态说明

这是“完整源码 + 当前改造进度”的接手包，不代表已经达到生产完成状态。后端全量编译、真实数据联调、UI 像素级收口、端到端测试和部署验证仍需继续。具体剩余任务见：

`docs/codex-handoff/Token_Router_企业版_剩余待完成任务清单.docx`
