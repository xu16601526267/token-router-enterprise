# 流量记录、分析与供给延展设计

> 配套 [`product-principles.md`](product-principles.md) 的 **P9（流量即情报）** 与 **供给三轨**。
> 本文设计：怎么记录供需两端流量 → 怎么分析 → 怎么驱动"招募更多第三方 / 自营采购 / 自持算力"的供给决策。
> 原则：**全部长在 [`architecture.md`](architecture.md) 的 `UsageLedger` 事实层之上，不改主链路**；先记录、再分析、最后才放权决策。

---

## 一、为什么要记录两端流量

平台不押资金、起步轻资产，**流量数据就是供给战略的情报**。三个扩张决策全靠它：

1. **该不该多接第三方？** 哪些模型 / 切片的需求 > 现有供给余量（供给缺口）？
2. **该不该自营采购？** 哪些切片量大且稳，值得垫资锁一批低成本容量？
3. **该不该自持算力？** 哪些切片 **cache 局部性高 + 量稳 + SLA 愿付溢价**，自建推理吃下它边际收益最高？

没有供需两端的流量画像，这三问只能拍脑袋。

---

## 二、三层数据（自底向上）

| 层 | 是什么 | 现状 |
|---|---|---|
| **L0 事实层** | `UsageLedger`：每调用一行（session、cache 拆分、双价、幂等） | architecture 已设计 |
| **L1 遥测 + 画像层** | 质量遥测（延迟 / 成败）+ 供给容量遥测 + 需求 / 供给聚合画像 | **已部分实现**：`SupplyCapacity`、`SupplyCapacityTelemetry`（record + upstream collect + sweep + deployable sweep runner / resident agent / opt-in API worker）、`SupplyTelemetryAgent`、`SupplierScorecard`、`TrafficProfile`、`TrafficForecast`，并可把供给 telemetry risk 写入 `OperatingInsight`；`deploy/systemd/` 已提供 API / telemetry runner 的 service + timer 模板，`deploy/smoke/` 已提供 strict gb10 process smoke，方便从 `aima2` 迁移到专属服务器前先复现闭环 |
| **L2 决策层** | 供给缺口分析、自持 ROI 分析、供应商姿态建议 → 供给三轨建议 | **已部分实现**：`SupplyDecision` + `SupplyExpansionOpportunity` + `SupplierPostureRecommendation` + `SupplierRoutePreference` + self-hosted `SupplyCostProfile` evidence + self-operated `SupplyPrepaidLot` drawdown evidence；`token-router-supply review once` 可一次性刷新 scorecard、posture、traffic、forecast、pricing、decision、opportunity 与 operating insight review read models，`token-router-supply review agent` 可用常驻 interval loop 周期刷新同一 review set；`deploy/systemd/` 已提供 review runner 的 resident service 或 timer 模板，`deploy/smoke/token-router-gb10-process-smoke.sh` 已把 live review agent 单轮 proof 纳入同一真实进程 smoke；dashboard 已暴露 opportunity type / priority / rank signals、cost profile 记录/查询、self-hosted savings evidence、prepaid lot 记录/查询/ledger drawdown refresh，以及 supplier posture recommendation query/generate/review/apply 和 active route preference 可见性 / bounded manual activate-disable / `1..200` boost；approved `downgrade` / `boost` apply 会落到 route preference overlay，`SupplyActionPlan` 会把已有 opportunity evidence 带入人审后的工作项，recorded self-hosted execution 可在人审后以 `traffic_percent` deterministic canary 激活路由（agent 出、人审，见 P5） |

L0 是计费级事实、已存在；L1 / L2 是为"分析 + 供给决策"新加的，主链路不动。

---

## 三、记录什么

### 需求侧（每请求，多数挂在 `UsageLedger` 上）
- 已有：SessionId、模型、cache 拆分、SellQuota、UserId / TokenId。
- **补**：请求时延（TTFT / 总时延）、成败 / 错误码、客户要求的 **SLA 档位**。
  → 支撑 SLA 度量（P3）、需求画像、cache 局部性统计。

### 供给侧（每请求 + 周期快照）
- 每请求补：出流量的**节点 / 缓存池**、上游时延与成败 → 供给质量分。
- 周期快照（`SupplyCapacity` + `SupplyCapacityTelemetry`）：每供应商 / 节点的**额定容量、token 利用率、GPU 利用率、余量、质量分、单位成本和遥测来源**；telemetry 可以由外部 record，也可以由 collect 从已配置 channel upstream 的固定 endpoint 拉取，或由 sweep 扫描已有 capacity snapshots 后批量 collect；部署侧可用 `token-router-supply telemetry sweep` 作为 cron / systemd timer 的 one-shot runner，也可用 `token-router-supply telemetry agent` 常驻写入 agent heartbeat 与最近 sweep 摘要；`deploy/systemd/` 已提供这两种部署模板，operator 二选一；小部署可显式设置 `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_ENABLED=true` 让 API server 内置 worker 周期执行同一 sweep。
  → 判断"供给缺口 / 该不该扩"的关键。`UsageLedger` 只记了"消耗"，没记"还剩多少"。

### 画像聚合（L1，按切片）
- **切片** = 模型 × 客户 / SLA 档 × 时间窗（可加 session 局部性桶）。
- 每切片产出：需求量与峰谷、cache 命中率分布、毛利、达成 SLA、对应供给余量。

---

## 四、分析能做什么（L2）

- **需求预测**：按切片预测量与峰谷 → 容量规划；当前默认 `TrafficForecast.method=weighted_moving_average` 对近期 profile 给更高线性权重，peak 仍取历史 max observed 以保持容量风险保守；显式传入 `seasonal_period_count` / `anomaly_guard` 时可生成 `seasonal_anomaly_adjusted` forecast，并记录 baseline、trend、seasonal index 与 anomaly spike/drop evidence。
- **cache 局部性聚类**：找出高复用切片 → 自持算力的首选目标；当前由 `SupplyExpansionOpportunity.cluster_key=high_cache_stable` 和 `locality_score` / `stability_score` 物化。
- **供给缺口检测**：需求 > 供给余量的切片 → 该招募 / 扩容；当前由 `SupplyExpansionOpportunity.opportunity_type=third_party_gap` 和 `headroom_risk_score` 物化。
- **自持 ROI**：切片量 × cache 命中潜力 ×（第三方成本 − 自持摊销成本）− SLA 违约成本 → 排序自持优先级；当前 `SupplyDecision.roi_score` + locality/stability/headroom risk 先形成基础 `rank_score`，若存在匹配 `SupplyCostProfile`，`SupplyExpansionOpportunity` 会记录 self-hosted 摊销单位成本、单位节省、总节省，并把总节省加入 rank score。SLA 违约成本与自动执行仍未接入。
- **供应商对比与供给风险复盘**：质量分 / 单位成本 / 余量，喂给评级（P4）与分流；缺 telemetry、stale telemetry、高 GPU utilization、低 token headroom 会进入 `OperatingInsight(category=capacity_risk)`，`SupplierPostureRecommendation` 会把 scorecard 与 open quality/capacity insight 汇总成 observe / boost / downgrade / disable 人审建议；dashboard 负责显式 generate / approve / reject / apply，approved `disable` apply 才会写入 supplier runtime status，approved `downgrade` apply 会创建 `SupplierRoutePreference(weight_percent=25)` 并在 normal channel selection 中降低该 supplier 候选权重，approved `boost` apply 会创建 `SupplierRoutePreference(weight_percent=150)` 并提高 strong supplier 候选权重；operator 也可在 `Posture` dashboard 对 enabled supplier 手工 activate/disable bounded route preference，active preference 会展示来源 recommendation/manual、weight、effective window 和 operator evidence。

---

## 五、怎么驱动供给三轨（决策闭环）

```
L0 事实 → L1 画像 → L2 分析（缺口 / ROI） → 三轨建议
                                          ├─ 招募第三方（缺口、长尾、试探）
                                          ├─ 自营采购（量大且稳，锁低成本批量）
                                          └─ 自持算力（高 cache 局部性 + 稳 + 溢价）
                                                   ↓
                                       dashboard：agent 出建议，人决策批准（P5）
                                                   ↓
                                  action plan：复制 opportunity evidence，交给 operator
                                                   ↓
                                  生效：加供应商 / 下采购单（线下） / 调度自有算力
```

自持算力落地，即把 org 的 Mooncake / SGLang / KTransformers 后端注册成一个"自持 supplier"，亲和路由把目标切片的 session 固定过去，**cache 全归自己**。
当前 `SupplyRoutingPolicy` 已支持 passed runtime SLA evidence 后的 `traffic_percent` canary：落入 deterministic session bucket 的请求才走 self-hosted policy，bucket 外请求继续走普通 channel selection，避免自持接入从 0 直接跳到 100%。

---

## 六、数据模型增量（最小、forward-compatible）

| 表 / 字段 | 作用 |
|---|---|
| `Supplier.Type`（第三方 / 自营 / 自持） | 区分三轨，成本基准与核销方式不同 |
| `UsageLedger` 补 `LatencyMs`、`Status`、`SlaTier`、`SupplyNode` | 质量遥测 + 供给定位（主链路不变，仅多记几列） |
| `SupplyCapacity`（供应商 / 节点 × 周期：Capacity、Utilization、GpuUtilization、Headroom、QualityScore、UnitCost、TelemetrySource） | 供给余量与质量快照——缺口 / 扩容判断的依据 |
| `SupplyCapacityTelemetry`（供应商 / 节点 × 周期 × source ref：Capacity、Used、GpuUtilization、QualityScore、UnitCost、ObservedAt） | 供给容量快照的可审计来源；record API 幂等 upsert exact-period capacity，collect API 可从 channel upstream 拉取同一格式遥测，sweep API 可批量扫描已有 capacity 并返回 collected/skipped；`token-router-supply telemetry sweep` / `telemetry agent` / API worker 可让部署侧周期调用同一 sweep 语义；不自动创建 supplier/channel、不自动路由、不调权 |
| `SupplyTelemetryAgent`（AgentKey、Hostname、RuntimeRef、LastHeartbeatAt、LastSweepStatus、LastSweepCounts） | 部署侧供给遥测 agent 或 API worker 的存活和最近 sweep 摘要；heartbeat / sweep_result API 幂等 upsert，不把 heartbeat-only 证据当成真实容量采集成功，不执行远程命令、不调权、不自动路由 |
| `SupplyCostProfile`（自持供应商 / 节点 × 周期 × source ref：FixedCost、VariableUnitCost、Capacity、AmortizedUnitCost、ObservedAt） | 自持算力成本基准证据；record API 幂等 upsert，只给 self-hosted opportunity read model 提供摊销单位成本和 savings，不自动采购、改价、路由或结算 |
| `SupplyPrepaidLot`（自营供应商 / 节点 × 周期 × source ref：PurchasedTokens、UnitCost、TotalCost、Drawdown、Remaining、UsageLedger source） | 自营预付批次 / 资金核销 read model；record API 幂等 upsert 线下采购凭据，refresh API 只从 matching successful `UsageLedger` 回填 drawdown / remaining / source evidence；dashboard 已可显式记录、查询、刷新核销；不创建支付、钱包、采购单、打款、发票或真实资金状态 |
| `SupplyActionExecution` drawdown（DrawdownTokens、RequestCount、RemainingTokens、DrawdownRate、UsageLedger source） | completed action plan 的执行级库存 / 算力核销 read model；refresh API 只从 matching successful `UsageLedger` 回填，不改 actual capacity、unit cost、capacity snapshot、routing、settlement 或资金状态 |
| `OperatingInsight` capacity risk（capacity telemetry reason：missing / stale / high_gpu / low_headroom） | 把供给 telemetry 风险变成 agent/operator 可读的复盘项；只记录 draft / acknowledged / dismissed，不自动执行 |
| `SupplierPostureRecommendation`（供应商 × scorecard period：Scorecard、Open quality/capacity insights、RecommendedAction、Review/Apply audit） | 供应商运行期姿态建议；generate 可产出 observe / boost / downgrade / disable draft，其中 `boost` 要求 enabled supplier、grade A、score >= 90、非零请求且无 open posture insight；approve/reject 人审留痕，apply 仅允许 approved recommendation 显式写 `Supplier.status`/`notes` 并触发对应 route preference overlay；dashboard 已可 query/generate/review/apply/filter action 并展示 runtime evidence、status before/after，以及对应 active route preference badge；`disable` 进入既有 runtime gate，`downgrade` / `boost` 创建或更新 active route preference |
| `SupplierRoutePreference`（供应商当前路由偏好：SourcePostureRecommendation、Status、WeightPercent、Effective window、Apply/Clear audit） | supplier-level route weight overlay；approved posture `downgrade` apply 写入 active `weight_percent=25`，approved posture `boost` apply 写入 active `weight_percent=150`，`observe` / `disable` apply 清除 active preference；manual activate 使用 `source_posture_recommendation_id=0` 并要求 enabled supplier、`weight_percent=1..200`、reason、operator note / effective window；`100` 是 baseline，`1..99` 降权，`101..200` bounded boost；normal channel selection 的 memory cache 与 DB fallback 会叠加该 multiplier，activate/disable 后刷新 runtime channel cache；`Posture` dashboard 已展示 active preference 来源、权重、生效窗口、operator note 和 reason，并提供 bounded manual set/disable；不改写 `Channel.weight`、`Ability.weight`、routing policy、pricing、billing、settlement 或 funds |
| `SupplyRoutingPolicy.traffic_percent`（recorded self-hosted execution → active policy） | self-hosted routing policy 的 deterministic session canary share；activation 仍要求同 supplier/channel/model/SLA tier 的 passed runtime SLA evidence，`traffic_percent=1..100`，`100` 是 hard override，`1..99` 是小流量 canary；未落入 bucket 的请求走普通 channel selection 且不写 policy miss insight；`Routing Policies` dashboard activation dialog 已可显式提交并展示该 traffic share；不自动 promotion/rollback、不改写 supplier/channel/capacity/route preference/pricing/billing/settlement/funds |
| `TrafficProfile`（切片 × 周期：Demand、PeakRatio、CacheHitDist、Margin、SlaMet、SupplyHeadroom） | L1 画像物化（也可先用视图，量大再落表） |
| `TrafficForecast`（切片 × source/target 周期：WeightedDemand、MaxPeak、LatestHeadroom、Confidence、Method、Seasonal/AnomalyEvidence） | L2 下一周期经营假设；默认 `weighted_moving_average`，按 source profile recency 加权 demand/cache/SLA/毛利/单位成本，peak 取 max observed，headroom 取 latest profile；可选 `seasonal_anomaly_adjusted` 保存 baseline demand、trend delta、seasonal index / demand、anomaly status / ratio / profile id，不自动执行供给动作 |
| `SupplyDecision`（切片、建议轨道、ROI、状态 draft / approved） | L2 建议 + 人审记录（dashboard） |
| `SupplyExpansionOpportunity`（decision × period：OpportunityType、Priority、ClusterKey、Locality/Stability/HeadroomRisk、SelfHostedCostProfile/Savings、RankScore） | L2 机会排序 read model，dashboard 可按 opportunity type / priority 查询并展示 source evidence、rank signals 与 self-hosted cost evidence / savings，方便 agent/operator 找到自持优先目标或供给缺口，不自动执行 |
| `SupplyActionPlan` opportunity evidence（OpportunityId、Type、Priority、ClusterKey、RankScore） | 人审后工作项保留 L2 排序证据，避免 operator handoff 只剩 decision 字段；不允许 opportunity 绕过 approved decision 直接执行 |

> 不新建实时流处理 / 数仓——起步用 GORM 聚合 + 周期任务物化即可，和 architecture 同栈。

---

## 七、里程碑（叠加在 architecture M0–M4 之上）

- **T0 遥测补齐**：`UsageLedger` 加延迟 / 成败 / SLA 档 / 节点；建 `SupplyCapacity` 周期快照与 `SupplyCapacityTelemetry` 证据记录。
- **T1 画像 + dashboard**：切片画像 + 供给余量看板（纯只读，先让人看懂供需）。
- **T2 供给决策建议**：`token-router-supply review once` 或 `token-router-supply review agent` 刷新 agent-readable scorecard / posture / profile / forecast / pricing / decision / opportunity / insight read models，dashboard 展示建议，人审批（P5）。
- **T3 自持算力接入**：org 推理后端注册成"自持 supplier"，亲和路由定向切片，已能在 passed runtime SLA gate 后用 `traffic_percent` deterministic canary 小流量导入，并已在 dashboard activation dialog 中让 operator 显式选择 / 回看 canary share；`deploy/smoke/token-router-gb10-process-smoke.sh` 可在迁移前复现 strict mock + API + demand + review agent 闭环；仍需继续验证更长期 cache 自有、成本摊销和自动 promotion/rollback 边界。

---

## 八、与现有架构的接缝

- **不改主链路**：计量仍走 architecture 的三处 post-consume 插桩，只多记几列。
- **复用亲和路由**：自持算力靠同一套 `service/channel_affinity.go`（亲和键 = SessionId）把切片固定到自有节点。
- **复用成本模型**：自营 / 自持只是 `Supplier.Type` 不同的成本基准，cache-aware 单位成本公式不变。
