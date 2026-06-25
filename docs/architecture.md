# token-router：双边 Token API 中转站（数据支撑型）

## Context（背景与定位）

双边 token 中转：上游接企业提供的 token 流量，下游把统一 OpenAI 兼容 API 卖给其他企业。

**关键定性（决定整份方案）**：

1. **平台不发生真实资金流。** 上、下游结算都走**线下公司级财务**；线上平台只做**数据支撑**——精准计量 + 产出成本/收入/毛利的**对账数据**供财务线下结算。→ 砍掉支付网关、真实充值扣费、打款执行、发票开具/收款全部模块。
2. **成本的核心杠杆是 KV cache 命中率，不是 token×单价。** 因此 **session ID 的记录与分发**是平台核心：同一会话要"亲和路由"到同一上游节点才能复用 KV cache，且成本核算必须 **cache-aware**（命中 token 成本极低）。对应 org 的 Mooncake/SGLang/KTransformers KVCache 中心化后端。

一句话：**这是一个"会话亲和路由 + cache-aware 计量 + 对账数据导出"的系统，不是一个支付/钱包系统。**

---

## 一、基线与范围（保持简单）

- **基线**：公开 `QuantumNous/new-api`（Go 单体：路由 + 计量 + 数据），不上 Higress。
- **从 kvcache-ai/new-api 移植**：`service/channel_affinity.go`（会话/规则亲和路由——**KVCache 复用的关键**）；可选 `relay/channel/codex/`（ChatGPT/Codex OAuth 供给）。
- **Higress**：仅作未来可选数据面（高并发/云原生时再议），本期不做。
- **平台做**：上游渠道接入、会话亲和路由、计量、双价（成本/卖价）台账、对账数据与报表导出。
- **平台不做**：支付网关、真实充值扣费、打款执行、发票/收款（→ 线下财务）。下游"额度"仅作**用量配额/风控上限**，非真实钱包扣款。

直接复用 new-api 现成能力：渠道接入/分组/权重/重试/故障转移、令牌/用户/分组、模型倍率（含 `CacheRatio`/`CacheCreationRatio`）、调用日志（带 `channel_id`、`usage.prompt_tokens_details.cached_tokens`）、按维度聚合。

---

## 二、核心：Session ID 与 KVCache 成本

1. **会话标识**：下游传入（header `X-Session-Id`，或 OpenAI `user`/会话字段）；缺失则路由器分配并回写。每次调用必记 `SessionId`。
2. **分发上游**：构造上游请求时把 `SessionId`（缓存键）透传给上游推理引擎（KTransformers/SGLang/Mooncake），用于 prefix cache 定位/复用。挂载点：relay 请求构造处（`relay/` 下 channel adaptor 的请求封装）。
3. **会话亲和路由**：同 `SessionId` 用一致性哈希固定到同一上游渠道/节点/缓存池 → 最大化 KV cache 命中。**移植并复用 kvcache fork 的 `channel_affinity.go`**，把亲和键设为 SessionId。
4. **cache-aware 成本**：每调用记录 `cached_tokens` / fresh prompt / completion；成本 = `fresh_prefill × 成本价 + cached × 极低倍率 + completion × 成本价`，复用 new-api 的 `CacheRatio`/`CacheCreationRatio` 与 `usage` 里的 cache 拆分。**这是成本能算准的前提。**
5. **命中率报表**：按 session/客户/模型/上游统计 cache 命中率，用于成本优化与定价。

---

## 三、数据模型（精简，全部 GORM，金额用既有 quota 整数单位）

| 表 | 关键字段 | 作用 |
|---|---|---|
| `Supplier`（`model/supplier.go`）| Name、Status、Notes（**仅业务标识，无银行/税号等资金字段**）| 上游结算对手（线下财务对应）|
| `Channel.SupplierId`（改 `model/channel.go`）| `SupplierId int gorm:"index;default:0"` | 成本归集连接点 |
| `SupplierAgreement`（`model/supplier_agreement.go`）| SupplierId、EffectiveFrom/To、ModelName、Cost*Ratio（含 **CostCacheRatio**）或单价、Priority | 协议成本价（cache-aware）|
| `UsageLedger`（`model/usage_ledger.go`）| RequestId(唯一,幂等)、**SessionId**、SupplierId、ChannelId、UserId、TokenId、ModelName、**CachedTokens/PromptTokens/CompletionTokens**、**SellQuota**、**CostQuota**、CacheHit、CreatedAt | 双价台账（每调用一行，对账事实基础）|
| `SettlementStatement`（`model/settlement.go`）| SupplierId/UserId、PeriodStart/End、Total{Cost,Sell}Quota、TotalRequests、CacheHitRate、Status(draft/confirmed)| **对账数据**（供线下财务；无 payout/发票状态机）|

> 不建 `Payout`/`Reconciliation`/发票表——资金动作在线下。`SettlementStatement` 只是"这期该跟某上游/某客户对多少账"的数据 + 导出。
> 迁移登记 `model/main.go`（`migrateDB`+`migrateDBFast`）；CRUD/路由克隆 `vendor_meta.go` + `vendorRoute` 模板。

### 供给侧遥测与画像事实层（部分实现）

`SupplyCapacity` 记录 supplier / node / model / period 的额定容量、已用量、余量、质量分和单位成本。ADR
[`0012-supply-capacity-snapshots`](adr/0012-supply-capacity-snapshots.md)
先提供人工/外部写入的周期快照；ADR
[`0050-ledger-backed-supply-capacity-usage-refresh`](adr/0050-ledger-backed-supply-capacity-usage-refresh.md)
新增 `POST /api/supply_capacities/refresh_usage`，把同周期 successful `UsageLedger` 的 `prompt_tokens + completion_tokens` 回填为 `used_tokens`，并重算 headroom / utilization。该 refresh 只补消耗事实，不探测真实硬件容量，不改写 quality / unit cost，不触发调权或路由。
ADR [`0058-supply-capacity-telemetry-evidence`](adr/0058-supply-capacity-telemetry-evidence.md)
新增 `SupplyCapacityTelemetry` 与 `/api/supply_capacity_telemetries/record|GET`：节点或外部系统可以记录 exact-period capacity / used / GPU utilization / quality / unit cost / source ref，系统在同一事务里 upsert `SupplyCapacity` 并复制最近 telemetry evidence。ADR
[`0064-upstream-capacity-telemetry-collector`](adr/0064-upstream-capacity-telemetry-collector.md)
新增 `/api/supply_capacity_telemetries/collect`，从已配置 channel upstream 的固定 `/token-router/telemetry/capacity` endpoint 主动拉取容量遥测并复用同一 record/upsert 语义。ADR
[`0065-supply-capacity-telemetry-sweep`](adr/0065-supply-capacity-telemetry-sweep.md)
新增 `/api/supply_capacity_telemetries/sweep`，从已有 capacity snapshots 批量选择同 supplier / model 的 enabled channel 执行 collect，并返回 collected / skipped 结果，供部署侧 cron 或后续 scheduler 调用。这里的 token utilization 仍由 used/capacity 计算，GPU utilization 是独立硬件证据；这些 API 不创建 supplier/channel、不激活 routing policy、不改价、不触碰账单或结算。
ADR [`0066-supply-telemetry-sweep-runner`](adr/0066-supply-telemetry-sweep-runner.md)
新增 `token-router-supply telemetry sweep` 一次性 runner，调用同一个 admin sweep API，并用 `--fail-on-skip` / `--min-collected` 为 cron / systemd timer 提供退出码语义。该 runner 是部署侧调度入口，不是 API server 内部后台 worker，也不注册 fleet agent、不自动调权、不禁用 channel、不激活 routing policy、不触碰资金动作。
ADR [`0070-supply-telemetry-fleet-agent`](adr/0070-supply-telemetry-fleet-agent.md)
新增 `SupplyTelemetryAgent` 与 `/api/supply_telemetry_agents/heartbeat|sweep_result|GET`，并把 `token-router-supply telemetry agent` 做成常驻部署侧 loop：heartbeat 后复用 sweep API，再记录最近 sweep status / counts。该 agent 证明部署侧采集进程存活和最近采集结果，不是 API server 内置 worker，不执行远程命令，不自动调权、禁用 channel、激活 routing policy 或触碰资金动作。
ADR [`0071-api-server-supply-telemetry-worker`](adr/0071-api-server-supply-telemetry-worker.md)
新增显式 opt-in 的 API server 内置供给遥测 worker：只有 `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_ENABLED=true` 且 master node 时启动；worker 周期调用同一个 sweep model，并把 heartbeat / sweep result 写入 `SupplyTelemetryAgent`。该 worker 适合小部署先不拆独立进程，但仍只采集 telemetry evidence，不自动创建 supplier/channel、不调权、不禁用、不激活 routing policy、不触碰账单、结算或资金动作。
ADR [`0080-deployable-operations-review-runner`](adr/0080-deployable-operations-review-runner.md)
新增 `token-router-supply review once`：按依赖顺序调用 scorecard、posture、traffic profile/forecast、pricing、decision、opportunity、operating insight 的既有 generate API，输出每步 count 与 `total_generated`，供 cron / systemd timer 刷新 agent-readable review set。ADR [`0083-operations-review-agent-loop`](adr/0083-operations-review-agent-loop.md)
又新增 `token-router-supply review agent`：常驻 interval loop 复用同一 review chain，支持 `--once` 单轮 smoke；未显式传入 period 时每轮重新计算最近 1h source window，避免长期进程重复 startup period。review runner / agent 都只生成既有 read model，不 approve / apply / activate / disable / complete，不创建 action plan，不调权、不改价、不路由、不触碰账单、结算或资金动作。
ADR [`0084-systemd-deployment-templates`](adr/0084-systemd-deployment-templates.md)
新增 `deploy/systemd/`，把 `token-router-api`、telemetry resident agent 或 sweep timer、review resident agent 或 once timer、以及 `/etc/token-router/token-router.env` 示例固化为 repo-native 部署模板。operator 需要在 resident 和 timer 模式之间二选一；这些 unit 只启动现有进程，不新增自动执行语义，不内置 admin token，不触碰路由、价格、账单、结算或资金动作。
ADR [`0085-gb10-process-smoke-runner`](adr/0085-gb10-process-smoke-runner.md)
新增 `deploy/smoke/token-router-gb10-process-smoke.sh`，把 strict `gb10-4t` mock、seed SQLite、API-only server、demand simulator 和 `review agent --once --min-generated 1` 串成一个 repo-native process smoke，并把 binaries、SQLite、logs 和 summary 留在证据目录。该 runner 只复现当前闭环，不安装 systemd、不使用生产 token、不新增自动 approve / apply / activate / disable / 调权 / 改价语义。

`SupplierScorecard` 和 `TrafficProfile` 继续读取 `SupplyCapacity` 的 capacity / used / headroom；ADR
[`0052-traffic-forecast-materialization`](adr/0052-traffic-forecast-materialization.md)
新增 `TrafficForecast`，把多个 `TrafficProfile` 物化为下一周期的 forecast（第一版 moving average：demand 均值、peak 最大值、latest headroom、gap、confidence）；ADR
[`0063-recency-weighted-traffic-forecast`](adr/0063-recency-weighted-traffic-forecast.md)
把默认生成方法升级为 `weighted_moving_average`：同 slice 内按 profile 时间顺序使用线性 recency weight，demand / cache / SLA / gross profit / unit cost 加权，peak 仍取 max observed，headroom 仍取 latest profile；ADR
[`0067-seasonal-anomaly-traffic-forecast`](adr/0067-seasonal-anomaly-traffic-forecast.md)
新增可选 `seasonal_period_count` / `anomaly_guard` forecast 生成路径：默认仍是 `weighted_moving_average`，传入 seasonal/anomaly 参数时生成 `method=seasonal_anomaly_adjusted`，并在 `TrafficForecast` 保存 baseline demand、trend delta、seasonal index / demand、anomaly spike/drop ratio 等可复算 evidence；ADR
[`0053-traffic-forecast-dashboard`](adr/0053-traffic-forecast-dashboard.md)
把 forecast source window、target window、confidence、gap 和 generate 操作暴露到默认 `/token-router` dashboard；ADR
[`0054-forecast-informed-supply-decisions`](adr/0054-forecast-informed-supply-decisions.md)
让 `GenerateSupplyDecisions` 在同 slice/source period 存在 forecast 时使用 forecast demand / peak / headroom / gap，并把 forecast id、target period、confidence、method 作为只读证据写入 `SupplyDecision`；ADR
[`0055-supply-expansion-opportunities`](adr/0055-supply-expansion-opportunities.md)
新增 `SupplyExpansionOpportunity` read model，把 approved/draft decision 进一步物化为 third-party gap / self-operated bulk / self-hosted cache opportunity，并记录 locality、stability、headroom risk 和 rank score。决策 key 仍绑定 source `TrafficProfile`，重复生成保留已有 review 状态，opportunity 只做排序分析，路由策略和价格仍不自动改写。因此真实业务流量可以进入：
ADR [`0056-supply-expansion-opportunity-dashboard`](adr/0056-supply-expansion-opportunity-dashboard.md)
把该 read model 暴露到默认 `/token-router` dashboard 的 `Opportunities` tab，支持 opportunity type / priority 筛选、generate 操作、forecast/profile source evidence 和 rank signals 展示；该界面仍只生成/查询分析记录，不创建 action plan、不激活 routing policy、不触碰供应商、容量、价格、账单、结算或资金动作。
ADR [`0057-action-plan-opportunity-evidence`](adr/0057-action-plan-opportunity-evidence.md)
让 `SupplyActionPlan` 在从 approved `SupplyDecision` 生成时复制已存在的 `SupplyExpansionOpportunity` id/key/type/priority/cluster/rank score，避免 L2 排序证据在 operator handoff 中丢失；approved decision 仍是唯一入口，action plan generate 不自动生成 opportunity、不自动创建 supplier/channel/capacity、不自动激活 routing policy。
ADR [`0060-self-hosted-cost-basis-evidence`](adr/0060-self-hosted-cost-basis-evidence.md)
新增 `SupplyCostProfile` 与 `/api/supply_cost_profiles/record|GET`，记录 self-hosted supplier / node / model / period 的固定成本、可变单位成本、容量和来源证据，并计算 amortized unit cost。`SupplyExpansionOpportunity` 在生成 self-hosted opportunity 时读取匹配 cost profile，把 cost profile id、self-hosted unit cost、unit savings、total savings 写入 read model，并把 savings 加入 rank score；没有 cost profile 时原 rank 不变。该证据不修改 `SupplyDecision` ROI、不改价、不结算、不采购、不自动路由。
ADR [`0061-self-hosted-cost-profile-dashboard`](adr/0061-self-hosted-cost-profile-dashboard.md)
把该成本证据暴露到默认 `/token-router` dashboard：新增 `Cost Profiles` tab 查询/记录 `SupplyCostProfile`，并在 `Opportunities` tab 展示 linked cost profile、self-hosted unit cost、unit savings 和 total savings。该界面是 operator 审核成本基准与机会排序证据的工作面，不自动创建 supplier/channel/capacity/action plan/routing policy，不改价、不采购、不结算。
ADR [`0068-prepaid-supply-lot-drawdown`](adr/0068-prepaid-supply-lot-drawdown.md)
新增 `SupplyPrepaidLot` 与 `/api/supply_prepaid_lots/record|GET|refresh_usage`，记录 self-operated supplier 的线下预付采购批次、token 库存、单位成本、source ref / external ref，并从 matching successful `UsageLedger` 回填 drawdown、remaining、drawdown rate 和 usage ledger source evidence。该证据是自营库存 / 资金核销 read model，不创建支付、钱包、采购单、打款、发票或真实资金状态，不自动路由或改价。
ADR [`0069-prepaid-lot-dashboard`](adr/0069-prepaid-lot-dashboard.md)
把该自营预付证据暴露到默认 `/token-router` dashboard：新增 `Prepaid Lots` tab 查询/记录 `SupplyPrepaidLot`，展示 purchased / drawdown / remaining、total/unit cost、source evidence、recorded metadata 和 usage ledger drawdown source，并提供显式 refresh drawdown。该界面是 operator 核对线下预付批次与业务核销的工作面，不创建支付、采购审批、容量、路由、账单或结算。
ADR [`0072-supplier-posture-recommendations`](adr/0072-supplier-posture-recommendations.md)
新增 `SupplierPostureRecommendation` 与 `/api/supplier_posture_recommendations` generate/query/approve/reject/apply API：从 `SupplierScorecard` 与 open `OperatingInsight` quality/capacity evidence 生成 observe / downgrade / disable 建议。generate 只写 draft evidence；approve/reject 只做人审；apply 只允许 approved recommendation 显式写入 `Supplier.status`/`Supplier.notes`，其中 disable 通过既有 runtime supplier gate 生效。ADR [`0078-supplier-posture-boost-recommendations`](adr/0078-supplier-posture-boost-recommendations.md) 又把强 supplier 的 positive lane 接入同一模型：enabled supplier 且 grade A、score >= 90、非零请求、无 open posture insight 时生成 `boost` draft；approved apply 不改 supplier status，只写入 `SupplierRoutePreference(weight_percent=150)`。
ADR [`0073-supplier-posture-dashboard`](adr/0073-supplier-posture-dashboard.md)
把该运行期姿态建议暴露到默认 `/token-router` dashboard：新增 `Posture` tab 查询/生成 `SupplierPostureRecommendation`，支持 status / recommended action / grade 筛选，并展示 scorecard evidence、open insight counts、runtime request/latency/supply evidence、review/apply audit 与 supplier status before/after。该界面只提供显式 generate / approve / reject / apply，不自动禁用、不触碰 channel/routing/pricing/billing/settlement/funds。
ADR [`0074-supplier-route-preference-overlay`](adr/0074-supplier-route-preference-overlay.md)
新增 `SupplierRoutePreference` 与 `/api/supplier_route_preferences` 查询 API：approved posture `downgrade` apply 会创建/更新 active `weight_percent=25` 的 supplier-level route preference；`observe` 或 `disable` apply 会清除 active preference。normal channel selection 在 memory cache 与 DB fallback 中把 active preference 作为候选权重 multiplier 叠加；self-hosted `SupplyRoutingPolicy` 对落入 `traffic_percent` bucket 的请求仍先于普通选择。该 overlay 不改写 `Channel.weight`、`Ability.weight`、routing policy、pricing、billing、settlement 或 funds。
ADR [`0075-supplier-route-preference-dashboard`](adr/0075-supplier-route-preference-dashboard.md)
把 active `SupplierRoutePreference` 暴露到同一个 `Posture` dashboard：页面查询 `GET /api/supplier_route_preferences?status=active`，展示 active preference 汇总、只读 preference panel、source recommendation、weight、effective window、operator note 和 reason，并在对应 posture recommendation row 标记 `Route Preference Active`。该界面只核对 ADR 0074 的路由 overlay 结果，不提供手工权重编辑，不自动生成或应用 preference，不改写 channel/ability/routing policy/pricing/billing/settlement/funds。
ADR [`0076-supplier-route-preference-operator-controls`](adr/0076-supplier-route-preference-operator-controls.md)
把 `SupplierRoutePreference` 从只读核对面推进到 bounded manual control：新增 `POST /api/supplier_route_preferences/activate` 和 `POST /api/supplier_route_preferences/:supplier_id/disable`，operator 可对 enabled supplier 设置 `weight_percent=1..100`、reason、operator note 与可选 effective window；manual preference 使用 `source_posture_recommendation_id=0`，activate/disable 后刷新 runtime channel cache。该控制只降低或恢复 supplier-level normal channel selection 权重，不允许高于 baseline 的 boost，不改写 channel/ability/routing policy/pricing/billing/settlement/funds。
ADR [`0077-bounded-supplier-route-preference-boost`](adr/0077-bounded-supplier-route-preference-boost.md)
把 manual `SupplierRoutePreference.weight_percent` 上限从 `100` 提升到 `200`：`100` 仍是 baseline，`1..99` 是降权，`101..200` 是 bounded boost。memory cache 与 DB fallback 使用同一 multiplier；posture-driven downgrade 仍固定 `25`，不会自动生成 boost。该控制仍需 operator reason / note / effective window 与 enabled supplier，不改写 channel/ability/routing policy/pricing/billing/settlement/funds。
ADR [`0078-supplier-posture-boost-recommendations`](adr/0078-supplier-posture-boost-recommendations.md)
新增 posture-driven `boost` recommendation：generate 只产生 draft evidence，approve/reject 仍只做人审，apply approved `boost` 才创建/更新 active `SupplierRoutePreference(weight_percent=150)`；approved `downgrade` 仍固定 `25`，`observe` / `disable` 仍清除 active preference。dashboard 已能按 `Boost` action 过滤和展示该 recommendation。
ADR [`0079-process-simulator-posture-route-preference-proof`](adr/0079-process-simulator-posture-route-preference-proof.md)
把 real-process `token-router-sim run` 接入同一 posture review 链路：基于 gb10 grade A scorecard 生成 `boost` draft，查询 `recommended_action=boost`，approve/apply 后回读 active `SupplierRoutePreference(weight_percent=150)`，并在最终输出中验证 `supplier_posture_verified=true` 与 `supplier_route_preference_verified=true`。
ADR [`0081-self-hosted-routing-canary-percent`](adr/0081-self-hosted-routing-canary-percent.md)
给 self-hosted `SupplyRoutingPolicy` 增加 `traffic_percent`，operator 激活 policy 时可设置 `1..100` 的 deterministic session canary；默认 `100` 保持既有 hard override。落入 bucket 的请求优先走 self-hosted policy，未落入 bucket 的请求回到普通 channel selection 且不写 policy miss insight；命中 policy 但 channel/supplier/ability 不可用时仍会 fallback 并生成 policy miss insight。该能力不做自动 promotion/rollback，不自动激活 policy，不改写 channel/ability/SupplierRoutePreference/pricing/billing/settlement/funds。
ADR [`0082-routing-policy-canary-dashboard-control`](adr/0082-routing-policy-canary-dashboard-control.md)
把 ADR 0081 的 canary percent 暴露到默认 `/token-router` dashboard：`Routing Policies` tab 的 self-hosted execution source 不再一键隐式 `100%` 激活，而是打开 activation dialog，operator 显式填写 `traffic_percent=1..100` 与 note；policy 表和 execution source row 同步展示当前 traffic share。该界面仍只调用既有人审 activation API，不改变 backend routing 语义，不自动 promotion/rollback，不改写 supplier/channel/capacity/pricing/billing/settlement/funds。

```text
UsageLedger -> supply capacity usage refresh -> SupplyCapacityTelemetry evidence / SupplyTelemetryAgent heartbeat / API worker sweep -> token-router-supply review once/agent -> systemd deployment templates -> gb10 process smoke runner -> SupplierScorecard / SupplierPostureRecommendation + SupplierRoutePreference overlay + Posture dashboard visibility / bounded operator controls + TrafficProfile -> TrafficForecast -> SupplyDecision -> SupplyCostProfile / SupplyPrepaidLot evidence -> SupplyExpansionOpportunity -> SupplyActionPlan evidence -> SupplyActionExecution drawdown -> SupplyRoutingPolicy canary -> Routing Policies dashboard canary control
```

这把供给侧 used/headroom 从 seed 或人工猜测推进到同一条需求台账事实链，并让 capacity snapshot、upstream-collected / swept telemetry、deployable sweep runner、resident telemetry agent、opt-in API worker、deployable operations review runner/agent、systemd deployment templates、gb10 process smoke runner、recency-weighted forecast、seasonal/anomaly forecast evidence、supplier posture recommendation + dashboard + route preference overlay + dashboard visibility / bounded operator controls、self-hosted opportunity、self-operated prepaid lot drawdown、execution drawdown、self-hosted routing canary 与 dashboard canary activation control 带上可审计 telemetry / cost / usage ledger source。P1/P4 的运行期供应商降级/禁用已有 evidence-backed 人审入口；approved `downgrade` 已能降低 normal channel selection 权重，approved `boost` 已能把 strong scorecard supplier 提升到 `150%` normal-routing 候选权重，operator 也能在 `Posture` dashboard 中手工 activate/disable `1..200` bounded route preference；approved `disable` 仍进入 runtime supplier gate。P9 的下一周期经营假设、L2 机会排序、自持摊销成本证据、自营预付批次核销与 dashboard、执行级库存 / 算力核销 read model 已接入供给建议链路；self-hosted policy 已可在 passed runtime SLA gate 后按 deterministic session bucket 小流量 canary，且 dashboard 激活面可以选择并回看 traffic share；供给遥测已有可被 cron / systemd timer 调用的一次性 runner、部署侧常驻 agent，以及显式启用的 API server 内置 worker；经营复盘已有 `review once` timer 入口和 `review agent` 常驻 loop 可刷新 agent-readable review set；`deploy/systemd/` 已把这些进程组合成可迁移到专属服务器的 unit/timer 模板，`deploy/smoke/` 已把同一闭环变成部署前可复现的真实进程检查。更复杂 ML / 外部数据 forecast、agent 自动调权、自动 promotion/rollback 和自动执行边界仍是后续工作。

### SLA 准入测量扩展（部分实现）

官方 SLA 准入不能只复用 `SupplierScorecard` 的运营均分。后续按 ADR
[`0038-sla-measurement-automation`](adr/0038-sla-measurement-automation.md)
增加独立测量层；ADR [`0040-sla-measurement-evidence-apis`](adr/0040-sla-measurement-evidence-apis.md)
已先落 contract / plan / run 证据 API；ADR
[`0042-token-router-sla-probe-runner`](adr/0042-token-router-sla-probe-runner.md)
已实现第一版 `token-router-sla probe run`，用于从计划执行 OpenAI-compatible
chat probe、写出 artifact、计算 SHA256 并回写 `SlaProbeRun`；ADR
[`0044-token-router-sla-workflow-cli`](adr/0044-token-router-sla-workflow-cli.md)
已补齐 `contract import` 与 `plan generate` CLI 编排；ADR
[`0043-sla-evidence-gated-supplier-evaluations`](adr/0043-sla-evidence-gated-supplier-evaluations.md)
已把 passed admission run 作为 active SLA contract 下 `SupplierEvaluation.admit` 的硬前置；ADR
[`0046-supplier-evaluation-sla-evidence-dashboard`](adr/0046-supplier-evaluation-sla-evidence-dashboard.md)
把 linked contract/run/gate summary 暴露到 Evaluations 表，方便 operator 直接核对 admit 的 SLA 证据来源；ADR
[`0047-sla-probe-run-operating-insights`](adr/0047-sla-probe-run-operating-insights.md)
把 failed / invalid / cancelled 或 hard-gate 未通过的 `SlaProbeRun` 接入 `OperatingInsight` 的 quality_watch 复盘面；ADR
[`0048-sla-evidence-gated-self-hosted-routing-policy`](adr/0048-sla-evidence-gated-self-hosted-routing-policy.md)
要求 self-hosted `SupplyRoutingPolicy` 激活前必须有同 supplier/channel/model/SLA tier 的 passed runtime run；ADR
[`0049-real-process-routing-sla-evidence-e2e`](adr/0049-real-process-routing-sla-evidence-e2e.md)
把该门禁接入真实进程 simulator，验证缺 evidence 拒绝、API 写入 runtime evidence、policy 持久化 evidence 和后续 self-hosted ledger；ADR
[`0051-supply-routing-policy-miss-insights`](adr/0051-supply-routing-policy-miss-insights.md)
把 active policy 的 channel/supplier/ability 不可用 fallback 写入 `OperatingInsight`，让静默 policy miss 进入 operator 复盘面；ADR
[`0059-capacity-telemetry-operating-insights`](adr/0059-capacity-telemetry-operating-insights.md)
把缺失 / 过期 telemetry、高 GPU utilization 或低 token headroom 的 `SupplyCapacity` 也写入 `OperatingInsight(category=capacity_risk)`，让供给侧硬件风险进入同一复盘面。

| 表 / 工具 | 关键字段 | 作用 |
|---|---|---|
| `SlaContract` | contract key、model、source/hash、effective period、input/output/cache/stream/error/availability profile | Kimi/GLM 等模型官方 SLA 合同的版本化数据源 |
| `SlaProbePlan` | contract、supplier/channel、probe type、route mode、prompt suite、cache profile、schedule | 从合同生成供应商准入或运行时抽查计划 |
| `SlaProbeRun` / `SlaProbeSample` | run summary、TTFT/TPOT/OTPS、cache、streaming usage、failure taxonomy、artifact hash | 自动化测量证据，可被 `SupplierEvaluation` 和 `OperatingInsight` 引用 |
| `SupplierEvaluation` | `sla_contract_id`、`sla_probe_run_id`、`sla_gate_summary_json` | active contract 存在时，`admit` 必须引用 latest passed admission run；缺失证据则只能 `observe` |
| `OperatingInsight` | `sla_contract_id`、`sla_probe_run_id`、`sla_probe_run_key`、`sla_probe_status`、hard gate、failure reasons、artifact hash、runtime ref；policy miss insight key / summary；capacity telemetry risk key / summary | failed / invalid / cancelled 或 hard-gate failed SLA run 的经营复盘入口；active routing policy channel/supplier/ability 不可用时的 fallback 复盘入口；capacity telemetry 缺失 / stale / hot GPU / low headroom 的供给风险复盘入口；只提示 operator review，不自动调权/禁用/改价 |
| `SupplyRoutingPolicy` | `sla_contract_id`、`sla_probe_run_id`、`sla_probe_run_key`、artifact hash、runtime ref、`traffic_percent` | self-hosted routing policy 激活门禁与 deterministic canary；必须引用 passed runtime run，不把 admission run 当作生产路由证据；未落入 canary bucket 的流量走普通 channel selection；dashboard activation dialog 可显式提交并展示 `traffic_percent` |
| `token-router-sla` | `contract import`、`plan generate`、`probe run --record` | 独立 runner，负责编排和精确采集，不在 API handler 里跑长 benchmark |

真实进程验证使用 `token-router-sim run --expect-sla-evidence` 断言 supplier evaluation 已引用 CLI 写回的 passed admission run。`gb10-4t` mock supply 支持 `stream=true` SSE response，供 `token-router-sla probe run` 采集 TTFT，并暴露 `/token-router/telemetry/capacity` 供 capacity telemetry collect/sweep 拉取节点容量。posture 阶段会基于 gb10 grade A scorecard 生成/查询/approve/apply `boost` recommendation，回读 active `SupplierRoutePreference(weight_percent=150)` 并验证 `supplier_posture_verified=true` / `supplier_route_preference_verified=true`；routing policy 阶段由 simulator 先验证缺 runtime evidence 的 activation 会失败，再通过 SLA evidence API 写入 runtime_light / direct_upstream passed run，并验证 `routing_sla_evidence_verified=true`，随后用 `traffic_percent=50` 验证 included session 路由到 self-hosted、excluded session fallback 到普通 channel selection，并输出 `supply_routing_policy_canary_verified=true`；policy miss 阶段会禁用 active policy channel，验证请求 fallback 到普通 channel selection 且输出 `policy_miss_insight_verified=true`；capacity telemetry 阶段会通过 sweep 拉取 gb10-4t mock telemetry 并验证 `capacity_telemetry_sweep_verified=true`，再记录 hot node telemetry 并验证 `capacity_telemetry_insight_verified=true`；部署侧 runner 阶段会单独执行 `token-router-supply telemetry sweep --min-collected 1` 并回读 `source_ref=gb10-4t-mock-capacity` 的 telemetry / capacity evidence；fleet agent 阶段会执行 `token-router-supply telemetry agent --once --min-collected 1`，并回读 `SupplyTelemetryAgent.last_sweep_status=ok`、`collected=1` 与同一 `gb10-4t-mock-capacity` telemetry/capacity evidence；API worker 阶段会在 `TOKEN_ROUTER_SUPPLY_TELEMETRY_WORKER_ENABLED=true` 下启动 API-only server，不启动外部 agent，回读 `SupplyTelemetryAgent.agent_key=api-server:aima2:adr0071`、`last_sweep_status=ok`、`collected=1` 和同一 telemetry/capacity evidence；经营复盘 agent 阶段会在同一类 strict mock + API-only process 后执行 `token-router-supply review agent --once --min-generated 1`，输出 `status=ok`、rolling source period 和 `total_generated=12`，证明 resident loop 调度的是同一组 live generate APIs；`deploy/smoke/token-router-gb10-process-smoke.sh` 已把 strict mock、seed、API、demand simulator 与 review agent 单轮 proof 固化为一条命令。前端 `Evaluations` tab 将 scorecard 指标标为 Runtime Evidence，并单独展示 SLA Evidence linked badge、contract/run id、run key、hard gate、artifact SHA256 短码和 runtime ref；`Operating Insights` tab 也展示 linked SLA run evidence，用于发现 runtime probe 失败、invalid、cancelled、hard-gate failed、routing policy miss 或 capacity telemetry risk 的复盘项；`Routing Policies` tab 展示 self-hosted policy 的 passed runtime evidence，缺少该 evidence 的 execution 不能激活 policy。

这层只产出准入和抽查证据，不自动创建 supplier/channel，不自动调权，不自动禁用，不触碰资金动作。

---

## 四、计量挂载点（已核实 new-api 真实代码）

- 三条同构后结算路径：`PostTextConsumeQuota`（`service/text_quota.go`）、`PostAudioConsumeQuota`/`PostWssConsumeQuota`（`service/quota.go`），均在 `model.UpdateChannelUsedQuota(...)` 后。
- 新建 `service/usage_record.go: RecordUsage(ctx, relayInfo, usage, sellQuota)`，在三处异步调用（`gopool.Go`）：`Channel→SupplierId`（走 `channel_cache.go`）→ 匹配协议价 → 复刻 `calculateTextQuotaSummary()` 公式算 **cache-aware** `CostQuota`（不乘下游 GroupRatio）→ 写 `UsageLedger`（带 `SessionId`、cache 拆分、RequestId 幂等）。
- 故障转移：只有最终成功那次进入 post-consume，成本天然落到真正出流量的上游。

---

## 五、下游（简化）

- 客户 = `User` + 令牌分组（方案 A，按 user 即按客户，报表零成本）；不接支付。
- 额度作**用量配额/风控上限**（复用 `Token.RemainQuota`/`UnlimitedQuota` 语义，但不绑真实钱包）。
- 多租户组织隔离（`Organization`）为**未来可选**，本期不做。

---

## 六、报表与对账数据导出

- **毛利报表**：`GROUP BY` `UsageLedger`：`SUM(SellQuota) 收入, SUM(CostQuota) 成本, 差额 毛利, AVG(CacheHit) 命中率`，维度 supplier/channel/user/model/时间。
- **对账数据导出**：按周期×供应商、周期×客户出 `SettlementStatement` + 明细 CSV/API，交线下财务。
- 新建 `controller/report.go` + 前端报表页（复用现有 dashboard 风格）。

---

## 七、里程碑（精简）

- **M0 数据骨架**：`Supplier`/`SupplierAgreement`/`UsageLedger`（含 `SessionId`）+ `Channel.SupplierId` + 双迁移 + CRUD + 前端表单。
- **M1 计量闭环（关键）**：`RecordUsage`（三处插桩，cache-aware 成本）+ session ID 记录与上游透传。→ **闭环：接上游 → 卖下游 → 每笔落一行带 session 与 cache 拆分的双价台账。**
- **M2 会话亲和路由**：移植 `channel_affinity.go`，亲和键=SessionId；cache 命中率报表。
- **M3 对账/毛利报表导出**：`SettlementStatement` 周期汇总 + CSV/API 导出给财务。
- **M4（可选）**：下游多租户、Higress 数据面演进。

### 最关键文件
- `service/text_quota.go`（`PostTextConsumeQuota` + `calculateTextQuotaSummary` 成本蓝本）、`service/quota.go`（另两插桩点）
- `relay/` channel adaptor 请求构造处（session ID 透传上游）
- `service/channel_affinity.go`（从 kvcache fork 移植，会话亲和路由）
- `model/channel.go`（加 `SupplierId`）、`model/log.go`/`usedata.go`（cache 字段来源、幂等键）
- `model/vendor_meta.go` + `controller/vendor_meta.go` + `router/api-router.go`(`vendorRoute`) + `model/main.go` — 新表克隆/注册模板

---

## 八、验证方式（端到端）

1. `docker-compose up` 起 new-api（MySQL+Redis）。
2. 建 `Supplier`、给某 `Channel` 设 `SupplierId`、配协议价（成本倍率 < 卖价倍率，且 `CostCacheRatio` 远小于 1）。
3. 建下游 user/token。
4. 带 `X-Session-Id` 连发两次**相同前缀**请求。**断言**：两次 `UsageLedger` 都带同一 `SessionId`；第二次 `CachedTokens>0`、`CostQuota` 显著低于第一次（cache 生效）；`SellQuota>CostQuota`。
5. **亲和路由**：同 session 多次请求落同一 channel；换 session 可落不同 channel。
6. 跑报表 → 毛利与 cache 命中率按 supplier/user/model 正确；导出对账 CSV。
7. **幂等**：重复 RequestId 不重复计。

---

## 附：调研依据（new-api / kvcache-ai / Higress）

- **kvcache-ai/new-api**：`QuantumNous/new-api`（原 `Calcium-Ion/new-api`，~39.5k star）的内部私有 fork，加了 Codex/ChatGPT OAuth 接入、`Vendor` 元数据、`funding_source`、`channel_affinity`，但无结算/无多租户。→ 基线用公开上游，按需移植其 `channel_affinity.go` / `relay/channel/codex/`。
- **new-api 可复用底座**：Channel（渠道）、Token（令牌）、User/Group、Pricing（含 cache 倍率）、Log（含 `cached_tokens`、`channel_id`）、QuotaData 聚合；Go + GORM + MySQL/PG + Redis + Docker。
- **Higress**：强数据面 AI 网关（ai-proxy 协议转换、ai-statistics、ai-token-ratelimit、ai-quota），但**零账务**（quota 仅 token 水位、无金额、无结算）。适合未来高并发数据面，不解决本业务核心（计量与对账）。
