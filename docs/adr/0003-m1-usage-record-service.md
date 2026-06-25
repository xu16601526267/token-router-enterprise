# ADR 0003: M1 台账记录服务先行

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P8 守住边界、P9 流量即情报
- 关联架构：M1 计量闭环、T0 遥测补齐

## 背景

M0 已经落地 `Supplier`、`SupplierAgreement`、`UsageLedger` 和 `Channel.SupplierId`。M1 的完整目标是三条 post-consume 路径成功结算后写入一行 cache-aware 双价台账。

直接把三处插桩、成本计算、session 解析和幂等写入一次性合并，会让问题定位变差。更稳的顺序是先实现可单测的 `RecordUsage` 服务，再把三条路径接入它。

## 决策

新增 `service.RecordUsage(ctx, relayInfo, usage, sellQuota)`：

1. 从 `relayInfo` 读取 request/channel/user/token/model/timing。
2. 从请求头或上下文读取 `SessionId`，优先级为 `X-Session-Id`、`session_id`、`prompt_cache_key`。
3. 通过 `Channel.SupplierId` 归集供应商。
4. 通过当前有效 `SupplierAgreement` 计算 cache-aware `CostQuota`。
5. 写入 `UsageLedger`，以 `RequestId` 做幂等唯一键。

成本计算规则：

1. 不乘下游 group ratio。
2. ratio 模式按 `(fresh_prompt + cached * CostCacheRatio + cache_creation * CostCacheCreationRatio + completion * CostCompletionRatio) * CostModelRatio`。
3. price 模式按 `CostModelPrice * QuotaPerUnit`。
4. 有 token 且成本倍率非零时，四舍五入后最低记 1 个 quota 单位。

本轮不做：

1. 不接入 `PostTextConsumeQuota` / `PostAudioConsumeQuota` / `PostWssConsumeQuota`。
2. 不改变用户扣费逻辑。
3. 不做 session 亲和路由，只记录 session。
4. 不做报表和对账汇总。

## 验证

本 ADR 对应施工完成后，需要：

1. 新增单元测试覆盖 cache-aware 成本计算。
2. 新增单元测试覆盖 `RecordUsage` 写入、session 解析和重复 request id 幂等。
3. 在 `aima2` 上至少运行 `go test ./service`。

