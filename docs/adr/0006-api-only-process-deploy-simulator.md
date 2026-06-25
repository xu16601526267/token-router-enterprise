# ADR 0006: API-only process deployment and simulator CLI

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P8 守住边界、P9 流量即情报
- 关联架构：M1 计量闭环、端到端验证方式

## 背景

ADR 0005 已用 `httptest` 证明核心链路：需求端模拟器请求 → token-router relay → `gb10-4t` mock 供给 → `UsageLedger` 查询核验。下一步要更接近部署现实：在 `aima2` 上启动真实进程，通过真实 TCP HTTP 端口跑同一条链路。

当前上游 `main.go` 嵌入 `web/default/dist` 和 `web/classic/dist`，完整构建需要前端产物。token-router 当前阶段要先验证 API/计量/台账闭环，不应让 dashboard 静态资源阻塞后端链路实测。

## 决策

新增两个小入口：

1. `cmd/token-router-api`：API-only 服务进程。
   - 初始化 Go 后端运行所需资源：env、logger、ratio settings、HTTP client、token encoders、DB、option map、log DB、Redis（可为空禁用）、i18n。
   - 只注册 `/api` 和 `/v1` relay 路由，不注册 dashboard/web 静态资源。
   - 不启动自动测活、自动更新、异步任务轮询等生产后台任务。
2. `cmd/token-router-sim`：端到端模拟器 CLI。
   - `mock-supply`：启动 OpenAI compatible `gb10-4t` mock 上游。
   - `seed`：向同一个 SQLite/MySQL/Postgres DB 写入 root admin、需求端用户/token、`gb10-4t` supplier/channel/agreement/ability，并持久化 `gpt-test` 的模型、completion、cache、group 倍率配置。
   - `run`：通过真实 HTTP 调 `/v1/chat/completions` 两次，再调用 `/api/usage_ledgers` 核验台账。

真实进程验证中 `seed` 与 `token-router-api` 是两个独立进程，不能依赖同进程内存里的 `ratio_setting`。因此模拟器 seed 必须把定价配置写入 `options` 表，由 API 进程启动时通过 `model.InitOptionMap()` 加载。

## 边界

本轮仍不做：

1. 不构建 dashboard 前端。
2. 不接真实外部 `gb10-4t` 服务；先用同协议 mock 供给验证链路。
3. 不做支付/钱包/打款/发票。
4. 不把 API-only 入口作为最终生产部署形态；它是当前后端链路验证与后续迁移的轻量运行面。

## 验证

在 `aima2` 上执行：

1. 构建 `token-router-api` 和 `token-router-sim`。
2. 启动 `token-router-sim mock-supply`。
3. 执行 `token-router-sim seed`。
4. 启动 `token-router-api`。
5. 执行 `token-router-sim run`，断言：
   - 两次需求端请求均成功。
   - `UsageLedger` 返回两条同 session 记录。
   - 第二条或其中一条 `CachedTokens > 0`。
   - `SellQuota > CostQuota`。
   - 重复 request id 不增加台账条数。
