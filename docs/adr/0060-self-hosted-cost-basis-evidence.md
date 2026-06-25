# ADR 0060: self-hosted cost basis evidence

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P1 商业闭环；P2 无数据不承诺；P5 人审；P6 成本透明；P8 守住边界；P9 流量即情报
- 关联 ADR：0012 Supply capacity snapshots；0054 Forecast-informed supply decisions；0055 Supply expansion opportunities；0058 Supply capacity telemetry evidence

## 背景

当前 `SupplyDecision` 和 `SupplyExpansionOpportunity` 已经能从 traffic profile / forecast 识别 self-hosted candidate，并用 cache locality、gross profit、headroom risk 给机会排序。

剩余缺口是：self-hosted ROI 仍然只使用流量画像里的平均单位供给成本，不能记录 GB10 / 自托管节点的硬件折旧、固定运营成本、可变成本和摊销后的单位成本。这样 operator 能看到“适合自托管”，但看不到这条建议是否真的有硬件成本依据。

## 决策

新增 `SupplyCostProfile` 作为只读经营证据，记录某个 supplier / supply node / model / period 的自托管成本基准：

1. 固定成本 quota、可变单位成本 quota、容量 token。
2. 基于 `fixed_cost_quota / capacity_tokens + variable_unit_cost_quota` 计算出的 `amortized_unit_cost_quota`。
3. 来源类型、来源引用、观测时间、记录人和备注，用于审计。

新增管理 API：

- `POST /api/supply_cost_profiles/record`：幂等记录成本证据。
- `GET /api/supply_cost_profiles/`：按 supplier / node / model / source / period 查询。

`/api/supply_expansion_opportunities/generate` 在生成 self-hosted opportunity 时读取同周期重叠的成本证据。若存在匹配 profile：

- opportunity 保存 `self_hosted_cost_profile_id`。
- 保存 `self_hosted_unit_cost_quota`、`self_hosted_savings_unit_quota`、`self_hosted_savings_quota`。
- `rank_score` 在原 ranking 基础上增加 `self_hosted_savings_quota`。
- reason 追加 cost profile 来源和节省证据。

若没有匹配成本证据，已有 opportunity 行为、rank 和 reason 保持不变。

## 边界

1. 不修改 `SupplyDecision` 的 ROI 公式，避免把未审核成本证据提前混入决策生成。
2. 不改价、不结算、不生成账单、不自动采购、不自动创建或激活 routing policy。
3. 不把 cost profile 视为 SLA 或真实可用容量证明；容量与 SLA 仍由 capacity telemetry / SLA evidence 负责。
4. 不引入后台采集器；本轮只提供记录、查询、生成机会时使用的证据通道。
5. 不要求所有 self-hosted candidate 必须有成本证据；缺失时保持原始机会排序，并由后续 insight / dashboard 显示缺口。

## 验收

1. model 测试覆盖 cost profile 幂等记录、摊销单位成本计算、查询，以及无成本证据时 opportunity rank 不变。
2. model / E2E 覆盖存在成本证据时 self-hosted opportunity 保存成本字段，并将 savings 加入 rank。
3. 真实进程 simulator 记录 GB10 成本 profile，生成 opportunity，并输出 `self_hosted_cost_profile_verified=true`。
4. API 证据可查询 `/api/supply_cost_profiles/` 与 `/api/supply_expansion_opportunities/`，证明成本来源、摊销单位成本、节省额和 rank 一致。
5. README / architecture / traffic docs / product principles 记录本轮能力与边界。
