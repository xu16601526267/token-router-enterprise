# ADR 0028: Default upstream session propagation

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P3 先度量再承诺 SLA、P6 做大正和、P9 流量即情报
- 关联架构：M1 session ID 记录与上游透传、M2 会话亲和路由、T3 自持算力 cache 归集

## 背景

架构文档要求两件事同时成立：

1. 平台记录每次请求的 `SessionId`，用于 `UsageLedger`、画像和对账。
2. 构造上游请求时把同一个 session/cache key 透传给上游推理引擎，用于 prefix / KV cache 定位。

当前代码已经完成 `UsageLedger.SessionId` 记录和 `token router session` affinity rule；但默认通用规则只把 `X-Session-Id` 用作本地 channel affinity key，没有默认把它发给上游。只有特定 Codex/Claude affinity template 或手工 channel header override 会透传相关 header。

这会留下一个隐患：本地看起来同 session 亲和到了同一 channel，但上游引擎未必拿到同一个 cache key，cache 命中可能不可复现。真实 E2E mock 也可能因为空 session key 复用而误判成功。

## 决策

在 relay 构造上游 HTTP 请求时增加默认 session header propagation：

1. 从当前请求解析 session/cache key，优先级与 `UsageLedger` 接近：
   - request header `X-Session-Id`
   - request header `X-Session-ID`
   - request header `session_id`
   - request header `X-Prompt-Cache-Key`
   - JSON body `session_id`
   - JSON body `prompt_cache_key`
   - JSON body `user`
2. 命中非空值时，在上游请求 header 写入：
   - `X-Session-Id: <value>`
   - `session_id: <value>`
3. 写入发生在 adaptor 默认 header 之后、channel header override 之前，因此显式 channel override 仍可覆盖或删除。
4. 不修改请求 JSON body，不生成缺失 session id，不改变本地 affinity cache key。
5. 仅在能从请求中提取到非空 session/cache key 时生效。

## 边界

本轮不做：

1. 不为没有 session 的请求自动生成 session id。
2. 不把 session key 写入 body 字段，避免破坏不同上游 schema。
3. 不把所有下游 header wildcard 透传给上游；仍只传这两个明确 cache/session header。
4. 不改变 Codex/Claude 专用 pass-through template。
5. 不改变 `UsageLedger` 成本计算、session affinity 选择或 routing policy 选择。

## 影响

正向影响：

- 架构的“记录 session + 上游透传”闭环成立，cache-aware 成本不再只依赖本地 ledger 事实。
- `gb10-4t` mock 可以严格断言上游收到真实 client session，避免空 session 误判 cache 命中。
- T3 self-hosted route 命中后，上游 self-hosted channel 也收到同一 session/cache key。

代价：

- 部分上游会看到两个额外 header；它们不含 token/API key，风险低。
- 仍需后续按真实 Mooncake / SGLang / KTransformers 接口确认最终 cache key 字段名称；本轮先保证通用 header 层可用。

## 验证

本 ADR 对应施工完成后，需要证明：

1. relay header helper 能从 header 和 JSON body 提取 session/cache key。
2. `DoApiRequest` 默认向上游写入 `X-Session-Id` 和 `session_id`。
3. channel header override 优先级仍高于默认 session header。
4. `gb10-4t` httptest 与真实进程 mock 严格校验上游收到非空、正确的 session header。
5. `aima2` focused Go 测试、`go test ./...` 和真实进程 simulator 通过，README 记录证据。
