# ADR 0029: Router-assigned session IDs

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P3 先度量再承诺 SLA、P6 做大正和、P9 流量即情报
- 关联架构：M1 session ID 记录与上游透传、M2 会话亲和路由、T0 `UsageLedger` 事实层

## 背景

架构文档明确要求：

1. 下游可通过 `X-Session-Id` 或 OpenAI `user` / 会话字段传入 session。
2. 缺失时 router 分配 session，并回写给下游。
3. 每次调用必须记录 `SessionId`。

ADR 0028 已经补齐默认上游透传：只要请求中存在 session/cache key，relay 会把它写入上游 `X-Session-Id` / `session_id`。但如果下游既没有 header，也没有 body 中的 `session_id`、`prompt_cache_key` 或 `user`，当前请求仍会以空 session 进入 affinity / upstream / ledger。

这会违反“每次调用必记 SessionId”，也让缺省客户端无法拿到可复用的 session key。

## 决策

在 distributor 早期为 OpenAI-compatible 请求解析并规范化 effective session id：

1. 解析优先级：
   - request header `X-Session-Id`
   - request header `X-Session-ID`
   - request header `session_id`
   - request header `X-Prompt-Cache-Key`
   - JSON body `session_id`
   - JSON body `prompt_cache_key`
   - JSON body `user`
2. 若解析到非空值，将它写入当前请求 header：
   - `X-Session-Id`
   - `session_id`
3. 若没有解析到非空值，则基于当前 request id 生成 `trsess_<request_id>`，同样写入 request header。
4. 无论来源是 client 传入、body 字段还是 router 生成，都向下游响应写回 `X-Session-Id`。
5. 因为规范化发生在 channel affinity 选择之前，`GetPreferredChannelByAffinity`、upstream propagation、`RecordUsage` 会读取同一个 effective session。

## 边界

本轮不做：

1. 不把 generated session 写入 JSON body。
2. 不为 multipart/form-data 解析 body session；缺 header 时直接生成。
3. 不引入跨请求服务端 session 存储；下游若要复用 cache，需要读取响应 header 并在后续请求带回。
4. 不改变 request id 幂等逻辑。
5. 不改变 routing policy、成本公式或 supplier/channel/capacity 状态。

## 影响

正向影响：

- `UsageLedger.SessionId` 对成功请求不再为空。
- 缺省客户端也能从响应 header 获得后续可复用的 session key。
- 本地 affinity、上游 KV cache key、ledger 事实层使用同一个 session 值，减少“本地有 session、上游无 session”的漂移。

代价：

- 不带 session 的每个新请求会得到不同的 generated session；只有客户端回传响应 header 后才会跨请求复用。
- generated session 包含 request id 衍生值，便于追踪，但不是业务会话语义。

## 验证

本 ADR 对应施工完成后，需要证明：

1. helper 对 header / JSON body / 缺省三种情况都能产出 effective session。
2. 缺省请求响应包含 `X-Session-Id`，上游 mock 收到非空 session header。
3. 缺省请求写入 `UsageLedger.SessionId`，可按响应 header 查询回读。
4. 已有显式 `X-Session-Id` 行为、affinity、routing policy 和 cache-aware 成本不回退。
5. `aima2` focused Go 测试、`go test ./...` 和真实进程 simulator 通过，README 记录证据。
