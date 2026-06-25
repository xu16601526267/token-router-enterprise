# ADR 0005: API-only E2E simulator for gb10-4t supply

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P8 守住边界、P9 流量即情报
- 关联架构：M1 计量闭环、端到端验证方式

## 背景

当前后端 M0/M1 已完成：供应商、协议、台账、真实 post-consume 插桩都已落地。下一步要证明完整链条：

1. 简化供给侧 `gb10-4t` 能作为上游渠道接入。
2. 需求端模拟器能通过 OpenAI 兼容 API 发请求。
3. token-router 能完成正常鉴权、渠道分发、上游调用、下游扣费与台账落库。
4. 可以通过接口核验 `UsageLedger` 的 session、cache 拆分、双价和幂等数据。

上游 new-api 的 main 包依赖 `web/default/dist` 和 `web/classic/dist` 前端构建产物。先补全完整 dashboard 构建会扩大本阶段范围；本阶段要证明的是 API/计量/台账链路，而不是前端资源服务。

## 决策

新增 API-only E2E harness：

1. 使用 `httptest` 启动 token-router 的真实 API/relay 路由，不注册 web 静态资源。
2. 使用内存 SQLite 初始化真实 GORM 表。
3. 启动一个 OpenAI 兼容 mock 上游，命名为 `gb10-4t`，返回可控 usage：
   - 首次请求：`cached_tokens = 0`
   - 同一 `X-Session-Id` 后续请求：`cached_tokens > 0`
4. 需求端模拟器通过真实 HTTP 请求调用 `/v1/chat/completions`。
5. 通过 admin API `/api/usage_ledgers` 查询并断言：
   - 两次请求均落账。
   - 两次请求具有同一 `SessionId`。
   - 第二次 `CachedTokens > 0` 且 `CostQuota < SellQuota`。
   - request id 幂等不重复落账。

## 边界

本轮不做：

1. 不构建 dashboard 前端。
2. 不做真实支付、钱包、打款、发票。
3. 不接真实 `gb10-4t` 外部服务；先用同名 mock 供给跑通协议和数据链路。
4. 不声明 SLA 已可商用；本轮只证明可度量数据链路。

## 验证

本 ADR 对应施工完成后，在 `aima2` 上运行：

1. `go test ./tests/e2e`
2. `go test ./controller ./model ./router ./service ./tests/e2e`

