# ADR 0055: supply expansion opportunities

- 日期：2026-06-23
- 状态：Accepted
- 关联原则：P5 agent 出洞察、人审；P7 理念可证伪；P8 守住边界；P9 流量即情报
- 关联 ADR：0015 Supply decision recommendations；0034 Operating insights；0052 Traffic forecast materialization；0054 Forecast-informed supply decisions

## 背景

`TrafficProfile`、`TrafficForecast` 和 `SupplyDecision` 已经能把历史流量、下一周期 forecast 和三轨供给建议串起来。`traffic-and-supply.md` 里还明确要求 L2 分析能做 cache 局部性聚类、供给缺口检测和自持 ROI 排序，用来回答哪些切片值得招募第三方、哪些值得自营采购、哪些值得自持算力。

当前 `SupplyDecision` 只保存单条建议本身；operator 或 agent 如果要按机会优先级排序，仍需要临时组合 decision、forecast、profile、cache 和 headroom 指标。为了让 P9 的“流量即情报”更接近可运营的数据面，需要新增一个可查询、可复算的机会排序层。

## 决策

新增 `SupplyExpansionOpportunity` 物化表与 admin API：

1. `POST /api/supply_expansion_opportunities/generate` 从指定 source period 的 `SupplyDecision` 生成机会排序记录。
2. 每条 opportunity 绑定 `supply_decision_id`、`traffic_profile_id`、可选 `traffic_forecast_id`，并复制 slice、source period、forecast target window、decision source/status、track/type 等证据字段。
3. 机会类型按既有 decision 归类：
   - `third_party_gap`：forecast/profile 显示 peak demand 超过 supply headroom。
   - `self_operated_bulk`：正毛利但 cache 局部性不足以优先自持。
   - `self_hosted_cache`：cache locality、稳定性和 ROI 支持评估自持算力。
   - `third_party_probe`：其他观察型第三方机会。
4. 记录可解释排序信号：`locality_score`、`stability_score`、`headroom_risk_score`、`roi_score`、`rank_score`、`cluster_key`、`priority` 和 reason。
5. repeated generate 按 `opportunity_key` 幂等刷新可复算 evidence，不新增 review 状态机；真正的人审仍在 `SupplyDecision` / `SupplyActionPlan` / `SupplyRoutingPolicy` 链路。

该层是 agent/operator 的 read model，不自动 approve/reject，不自动创建 action plan，不自动创建 supplier/channel/capacity，不激活 routing policy，不改价，不触碰账单、结算或资金动作。

## 不做什么

1. 不引入 LLM、外部 agent workflow、机器学习聚类或后台定时任务。
2. 不替代 `SupplyDecision` 的人审状态；opportunity 只是排序视图。
3. 不做真实硬件 quota、GPU 利用率、卡时成本摊销或库存核销。
4. 不把 opportunity 直接作为路由准入证据；self-hosted routing policy 仍必须走 completed action execution 和 passed runtime SLA evidence。

## 验收

1. model 测试证明 forecast-informed self-hosted decision 能生成 `self_hosted_cache` opportunity，并计算 locality/stability/rank。
2. model 测试证明 forecast gap decision 能生成 `third_party_gap` opportunity。
3. HTTP E2E 和 `token-router-sim run` 覆盖 generate/query API，并证明 gb10-4t 链路会产出可查询 opportunity。
4. 在 `aima2` 通过相关 Go tests 与真实进程 simulator。
