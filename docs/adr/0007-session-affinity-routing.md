# ADR 0007: Session ID channel affinity routing

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P3 先度量再承诺 SLA、P9 流量即情报
- 关联架构：M2 会话亲和路由、KVCache 成本优化

## 背景

M1 已经证明需求端请求可以落入 `UsageLedger`，并记录 `SessionId`、cache 拆分、卖价与成本。下一步要让同一会话稳定打到同一上游 channel，才可能复用 KV cache，并让后续 cache 命中率报表有经营意义。

代码基线中已有 `service/channel_affinity.go` 与 `middleware/distributor.go` 的亲和框架：首次请求成功后记录 key → channel，后续请求在 channel 选择前优先读取该缓存。当前默认规则只覆盖 Codex/Claude CLI 的专用字段，未覆盖 token-router 架构文档要求的通用 `X-Session-Id`。

## 决策

复用现有 channel affinity 框架，新增默认规则 `token router session`：

1. 匹配 OpenAI compatible 主路径：`/v1/chat/completions`、`/v1/responses`、`/v1/messages`。
2. 亲和 key 来源优先级：
   - request header `X-Session-Id`
   - request header `X-Session-ID`
   - JSON body `session_id`
   - JSON body `prompt_cache_key`
   - JSON body `user`
3. cache key 包含 rule name、model name、using group 和 session value，避免同一 session 在不同模型/分组之间误复用不可用 channel。
4. 保留现有成功后记录语义：只有成功请求才写入亲和缓存；失败 channel 不成为新亲和目标。
5. 不改变 retry/fallback 语义：亲和 channel 不可用时，沿用现有清理缓存并退回随机/权重选择的路径。
6. 保留 Codex/Claude CLI 专用规则优先级；通用 session 规则排在它们之后，避免抢走专用 header pass-through template。

## 边界

本轮不做：

1. 不实现跨进程/跨节点强一致 sticky 表；现有 HybridCache 足够支撑当前 API-only 进程实测。
2. 不做 SLA 权重动态调度；M2 先证明同 session 同 channel。
3. 不做 UI 配置页；默认规则先进入后端配置默认值，后续可通过现有 option/config 机制覆盖。

## 验证

扩展 E2E 与进程模拟器：

1. seed 两个同模型、同供应商、同 mock `gb10-4t` 上游 channel。
2. 同一 `X-Session-Id` 连发请求。
3. 查询 `/api/usage_ledgers`，断言同一 session 下所有 ledger 的 `channel_id` 一致。
4. 保留原有断言：cache hit、`SellQuota > CostQuota`、业务 request id 幂等。
