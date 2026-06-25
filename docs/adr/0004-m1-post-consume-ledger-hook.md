# ADR 0004: M1 post-consume 台账插桩

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P2 无数据不承诺、P8 守住边界
- 关联架构：M1 计量闭环

## 背景

ADR 0003 已经新增 `RecordUsage`，并用单元测试验证了 cache-aware 成本计算与 request id 幂等。要形成真实计量闭环，必须把它接到 new-api 成功结算后的 post-consume 路径。

架构文档已核定三处挂载点：

1. `PostTextConsumeQuota`
2. `PostAudioConsumeQuota`
3. `PostWssConsumeQuota`

这些路径都在最终成功请求上执行，并已经完成 `UpdateChannelUsedQuota`，因此天然避开故障转移中的失败候选。

## 决策

在三条 post-consume 路径成功结算后异步调用 `RecordUsage`：

1. 写入失败只记录错误日志，不影响用户请求、扣费、日志或上游响应。
2. 只在 token 总量大于 0 的成功计量分支写入；上游无 usage 或 total token 为 0 时暂不写入事实台账。
3. 本轮不改计费公式、不改用户扣费、不改 channel 选择、不改亲和路由。
4. 使用 `RequestId` 幂等，重复调用不会重复落账。
5. 台账幂等键优先接受需求端显式传入的 `X-Token-Router-Request-Id`。缺省时才退回 relay 内部 request id。这样不改变 `X-Oneapi-Request-Id` 的链路日志语义，也能让需求端用稳定业务请求号做去重。

## 后续

这一步完成后，需求端模拟器可以通过真实 OpenAI 兼容请求触发 `UsageLedger` 写入。下一步再写 ADR 实现模拟器与 `gb10-4t` 简化供给链路。

## 验证

本 ADR 对应施工完成后，需要在 `aima2` 上运行：

1. `go test ./service`
2. `go test ./controller ./model ./router ./service`
