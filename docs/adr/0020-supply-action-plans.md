# ADR 0020: Supply action plans from approved decisions

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P5 人在回路、P8 平台只做数据与信任中枢、P9 流量即情报
- 关联架构：T2 供给决策闭环、供给三轨、`SupplyDecision` 人审记录

## 背景

ADR 0015 / 0016 已经完成 `TrafficProfile` → `SupplyDecision` → dashboard 人审留痕。当前 approved decision 仍只是一个审批状态，无法稳定交给线下运营或后续执行器处理。

`traffic-and-supply.md` 的闭环是：L2 分析产生三轨建议，人批准后，进入“加供应商 / 下采购单（线下） / 调度自有算力”的生效阶段。平台软件不能直接执行资金动作、采购动作或自动调权，但应该把“人已经批准、下一步该做什么”变成结构化、幂等、可查询的 action plan。

## 决策

新增 `SupplyActionPlan` 事实表与 admin API：

1. `POST /api/supply_action_plans/generate`：从 approved `SupplyDecision` 生成 action plans。
2. `GET /api/supply_action_plans`：按周期、decision、track、status 查询 action plans。
3. 每个 approved decision 最多对应一个 action plan，以 `supply_decision_id` 幂等 upsert。
4. action type 按 decision type 映射：
   - `third_party_recruit` → `recruit_third_party`
   - `self_operated_purchase` → `prepare_self_operated_purchase`
   - `self_hosted_evaluate` → `evaluate_self_hosted_capacity`
   - `third_party_probe` → `keep_third_party_observation`
5. action plan 复制核心事实：slice、model、SLA、user、period、track、recommended capacity、gap、ROI、审批人和审批时间。

## 边界

本轮不做：

1. 不自动创建 supplier / channel。
2. 不自动修改 channel weight、supplier status 或 capacity。
3. 不创建采购单、打款单、发票或真实资金状态。
4. 不接真实 agent；action plan 是 approved decision 的结构化后续工作项。
5. 不做 dashboard 展示；本轮先完成后端事实层、API、E2E 和真实进程模拟器。

## 影响

正向影响：

- T2 不再停在“建议已批准”，而是有了可交付运营的结构化工作项。
- 后续 dashboard 可以展示 action plans，而不用重新解释 approved decisions。
- 后续自营采购、自持算力接入可以只读取 approved action plans，并继续守住人审边界。

代价：

- 第一版 action plan 只有 planned 状态，不做完整任务流。
- action plan 的执行仍在线下或后续系统中完成，当前软件只提供数据证据。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `SupplyActionPlan` 进入普通迁移和 fast migration。
2. `/api/supply_action_plans/generate` 只从 approved decisions 生成 action plan。
3. `/api/supply_action_plans` 能查询回读，并校验 action type、track、recommended capacity、ROI、source review fields。
4. `token-router-sim run` 能在真实进程链路中核验 action plan API。
5. `aima2` 上 focused Go 测试和 `go test ./...` 通过，README 记录证据。
