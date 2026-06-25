# ADR 0051: supply routing policy miss insights

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P2 无数据不承诺；P3 先度量再承诺 SLA；P5 人审；P8 守住边界；P9 流量即情报
- 关联 ADR：0026 Self-hosted routing policies；0027 Supply routing policy dashboard；0034 Operating insights；0039 Supplier posture runtime gating；0048 SLA evidence gated self-hosted routing policy

## 背景

ADR 0026 明确允许 active `SupplyRoutingPolicy` 在匹配 model / SLA / user / period 时优先影响选路，也明确 policy channel 不可用时要回到现有 channel selection，避免可用性被单条自持 policy 阻断。

ADR 0039 和 ADR 0048 又加强了运行时门禁：disabled supplier、disabled channel、无 ability 或缺 runtime SLA evidence 都不能被 policy 使用。但当前 fallback 是静默的。operator 只能看到 policy 仍为 active，却看不到请求实际没有进入 policy channel，P9 的“流量即情报”在 routing policy miss 上断开。

## 决策

把 policy miss 变成经营复盘信号：

1. 将 policy lookup 拆成“解析可用 policy”和“解释 miss reason”两个步骤。
2. 当 active policy 匹配请求切片，但 policy channel 因以下原因不可用时，继续 fallback 到普通 channel selection：
   - `channel_missing`
   - `channel_disabled`
   - `supplier_disabled`
   - `supplier_mismatch`
   - `cannot_serve_model`
3. 在实际 fallback 点写入一条 `OperatingInsight`：
   - category = `quality_watch`
   - severity = `watch`
   - status = `draft`
   - insight key 按 policy、group、reason、小时窗口幂等 upsert，避免每个请求刷屏。
4. insight 引用 policy 的 `traffic_profile_id`、`supply_decision_id`、model、SLA tier、user、supplier/channel/capacity/SLA evidence 摘要。
5. simulator 在真实进程链路中制造 active policy channel unavailable 场景，并验证 fallback 后能查询到 policy miss insight。

## 边界

1. 不禁用 `SupplyRoutingPolicy`。
2. 不修改 supplier、channel、ability、capacity、pricing、billing 或 settlement。
3. 不让 policy miss 直接导致请求失败；普通 fallback 行为保持不变。
4. 不新增后台告警系统或定时任务；第一版只把 miss 写入现有 Operating Insights 复盘面。
5. 不把 insight 当成自动化执行命令；是否 disable policy 仍由 operator 人审决定。

## 验收

1. model 测试证明 active policy 引用 disabled channel 时，policy lookup 返回 miss reason 并写入幂等 `OperatingInsight`。
2. service / HTTP e2e 证明请求会 fallback 到普通 channel，同时持久化 policy miss insight。
3. 真实进程 simulator 输出 `policy_miss_insight_verified=true`。
4. focused Go tests 与 aima2 真实进程验证通过，README / architecture / product principles 记录本轮证据。
