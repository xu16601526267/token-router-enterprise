# ADR 0039: Supplier posture gates runtime routing

- 状态：Accepted
- 日期：2026-06-23
- 关联原则：P1 严选不做集市；P4 供应商优胜劣汰；P5 agent 出建议、人审执行；P8 平台只做数据与信任中枢
- 关联 ADR：0037 Supplier evaluation apply workflow；0026 Self-hosted routing policies

## 背景

ADR 0037 让 approved supplier evaluation 可以通过显式 apply 落到 `Supplier.status`。这补齐了“人审证据 -> supplier posture”的控制面，但 runtime 路由仍主要只检查 `Channel.status` 与 `Ability.enabled`。

如果一条 `reject` evaluation 被 apply，把 supplier 置为 disabled，而该 supplier 下的 channel 仍处于 enabled，则常规随机/亲和路由和 self-hosted routing policy 仍可能继续选中这些 channel。这会让“严选复评后禁用供应商”的人审动作没有真实运行时效果。

## 决策

1. runtime channel eligibility 必须同时满足：
   - `Channel.status = enabled`
   - channel 对应 group / model 的 ability enabled
   - 若 `Channel.supplier_id > 0`，则 `Supplier.status = enabled`
2. memory channel cache 初始化时只把 enabled supplier 下的 enabled channel 放入可选路由列表。
3. memory channel cache 在选路时继续按 supplier status 做防御性过滤，避免缓存短暂陈旧时继续选中 disabled supplier。
4. DB fallback 选路必须在计算 priority 前应用 supplier posture 约束，避免 disabled 高优先级 supplier 遮挡 enabled 低优先级 supplier。
5. supply routing policy 匹配与激活校验都必须拒绝 disabled supplier。
6. supplier evaluation apply 成功更新 `Supplier.status` 后刷新 memory channel cache，使 dashboard apply 能尽快影响 runtime 路由。

## 不做什么

1. 不自动修改 `Channel.status`。
2. 不删除或重写 `Ability`。
3. 不删除、禁用或重写已有 `SupplyRoutingPolicy`；匹配时自然跳过 disabled supplier。
4. 不阻止历史 usage ledger、billing、settlement 对既有事实做 channel / supplier 读取。
5. 不把 supplier missing 的 channel 当作可运行供应；有 `supplier_id` 但找不到 supplier 时 runtime 视为不可选。
6. 不触碰支付、打款、发票、结算状态。

## 验收

1. memory cache 选路不会返回 disabled supplier 下的 channel，并能 fallback 到 lower-priority enabled supplier channel。
2. DB fallback 选路不会因为 disabled supplier 的高优先级 channel 而返回 nil，能选中 lower-priority enabled supplier channel。
3. active supply routing policy 如果引用 disabled supplier，则请求匹配不到该 policy。
4. disabled supplier 的 self-hosted execution 不能激活 routing policy。
5. supplier evaluation apply 更新 supplier status 后刷新 memory channel cache。
6. Go focused tests 与 `go test ./...` 通过。
