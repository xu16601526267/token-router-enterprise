# ADR 0015: Supply decision recommendations

- 日期：2026-06-22
- 状态：Accepted
- 关联原则：P5 人在 dashboard 决策、P6 做大正和、P8 平台只做数据与信任中枢、P9 流量即情报
- 关联架构：T2 供给决策建议、`TrafficProfile` L1 画像、供给三轨

## 背景

ADR 0014 已经把 `UsageLedger` 与 `SupplyCapacity` 聚合成 `TrafficProfile`。系统现在能回答某个切片在一个周期内的需求、峰值、cache、毛利、成功率和供给余量，但还没有把这些事实转成供给三轨的运营建议。

`traffic-and-supply.md` 的 T2 要求是：agent 出缺口 / ROI 分析，人在 dashboard 审批。当前阶段必须守住两个边界：

1. 建议可以自动生成，但不能自动调权、自动采购或自动注册自持算力。
2. 审批动作只记录人的选择，不直接改动资金、供应商或路由状态。

## 决策

新增 `SupplyDecision` 表与 admin API：

1. `POST /api/supply_decisions/generate`：从已物化 `TrafficProfile` 生成供给建议。
2. `GET /api/supply_decisions`：按周期、model、SLA、user、track、status 查询建议。
3. `POST /api/supply_decisions/:id/approve`：把 draft/rejected 建议标记为 approved，并记录 operator。
4. `POST /api/supply_decisions/:id/reject`：把 draft/approved 建议标记为 rejected，并记录 operator。
5. 第一版每个 profile 生成一条建议，使用确定性启发式：
   - `peak_tokens > supply_headroom_tokens`：建议 `third_party_recruit`，补第三方供给缺口。
   - 否则 `cache_hit_rate >= 0.5` 且有正毛利：建议 `self_hosted_evaluate`，进入自持算力 ROI 评估。
   - 否则有正毛利：建议 `self_operated_purchase`，进入自营采购评估。
   - 其它情况：建议 `third_party_probe`，维持轻资产观察。
6. `roi_score` 第一版为可排序运营分，不是财务承诺：
   `gross_profit_quota + cache_hit_rate * demand_tokens * avg_unit_cost_quota - gap_tokens * avg_unit_cost_quota`

## 边界

本轮不做：

1. 不自动修改 channel 权重、supplier 状态或 capacity。
2. 不创建采购单、打款单、发票或任何资金动作。
3. 不接真实 agent；generate API 先等价为确定性规则引擎，方便 E2E 复算。
4. 不承诺 ROI；`roi_score` 只是排序信号，后续可替换为更完整的成本模型。

## 影响

正向影响：

- L2 决策层有了可落库、可查询、可审批的最小闭环。
- `TrafficProfile` 不再只是报表数据，开始驱动供给三轨建议。
- 后续 dashboard 可以直接展示 draft / approved / rejected 建议，不需要重新扫画像表。

代价：

- 第一版规则简单，需要更多真实流量后校准阈值。
- 审批只留痕，不会让系统自动执行运营动作。

## 验证

本 ADR 对应施工完成后，需要证明：

1. `SupplyDecision` 已进入普通迁移和 fast migration。
2. `/api/supply_decisions/generate` 能基于 `gb10-4t` 的 `TrafficProfile` 生成 draft 建议。
3. `/api/supply_decisions` 能查询到该建议，并校验 track、type、ROI、status。
4. `/api/supply_decisions/:id/approve` 能把建议改为 approved 并记录 operator。
5. `token-router-sim run` 能在真实进程链路中核验 decision API。
6. `aima2` 上 focused Go 测试和 `go test ./...` 通过，README 记录证据。
