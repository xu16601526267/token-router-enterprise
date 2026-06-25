# ADR 0034: Operating insights

- 状态：Accepted
- 日期：2026-06-22
- 关联原则：P5 agent 出洞察、人审；P6 做大正和；P7 理念是假设、需数据复盘；P8 平台只做数据与信任中枢；P9 流量即情报
- 关联架构：`TrafficProfile` L1 画像、`SupplyDecision` / `PricingRecommendation` T2 建议、人审 dashboard 模式

## 背景

当前系统已经能从 `UsageLedger` 和 `SupplyCapacity` 生成 `TrafficProfile`，再分别生成供给三轨建议与 SLA/价格建议。这些建议已经可以被 operator 审批，但它们仍分散在不同表里：

1. 供给侧建议回答“该补第三方、自营采购，还是评估自持算力”。
2. 价格建议回答“该提价、维持，还是分享 cache 节省”。
3. 供应商评估回答“某供应商是否适合入选或继续观察”。

产品原则 P5 / P7 还需要一个更高一层的经营洞察入口：agent 或规则引擎把多个事实和建议合成一条可解释的 hypothesis，operator 可以 acknowledge / dismiss，并留下复盘痕迹。第一版仍不接真实 LLM，先用确定性规则保证可复算、可测试、可审计。

## 决策

新增 `OperatingInsight` 物化表与 admin API：

1. `POST /api/operating_insights/generate`：基于已物化 `TrafficProfile`，关联同一 profile 的 `SupplyDecision` 与 `PricingRecommendation`，生成 draft insight。
2. `GET /api/operating_insights`：按周期、model、SLA、user、category、severity、status 查询 insight。
3. `POST /api/operating_insights/:id/acknowledge`：记录 operator 已读 / 接受该洞察。
4. `POST /api/operating_insights/:id/dismiss`：记录 operator 驳回该洞察。

第一版每个 `TrafficProfile` 最多生成一条 slice-level insight，字段复制关键证据：

- profile：slice、model、SLA、user、period、demand、peak、cache hit、SLA met、gross profit、supply headroom、unit cost。
- supply decision：id、track、decision type、status、ROI。
- pricing recommendation：id、action、status、recommended unit price、recommended margin。

分类与 severity 使用确定性启发式：

- `cache_efficiency` / `action`：cache hit 高、毛利为正，并且供给建议是 self-hosted 或价格建议是 share_savings。表示“cache 创造正和空间，可以同时推进自持评估和让利”。
- `capacity_risk` / `action`：供给缺口大或供给建议是 third-party recruit。表示“先补供给，不要贸然承诺更高 SLA”。
- `pricing_risk` / `action`：价格建议是 raise_price。表示“当前 SLA/成本/毛利组合需要改价或降承诺”。
- `quality_watch` / `watch`：SLA 达成不足。表示“质量未稳定前只观察，不承诺 SLA”。
- `steady_state` / `info`：其余切片继续观察。

`insight_key` 以 profile slice + period 唯一。重复 generate 刷新事实字段、分类和建议文本，但保留已 review 的 status、reviewed_at、reviewed_by、review_note。

## 不做什么

1. 不自动 approve / reject 任何 `SupplyDecision`、`PricingRecommendation` 或 `SupplierEvaluation`。
2. 不自动修改 `ModelPrice`、套餐、账单、结算、supplier、channel、capacity 或 routing policy。
3. 不引入 LLM prompt、外部 agent workflow 或后台定时任务。
4. 不把 insight 当成 SLA 承诺；它只是复盘和经营决策输入。

## 影响

- operator 可以在单一 API/dashboard 里看到“供给建议 + 定价建议 + profile 证据”的合成解释。
- P6 的“正和”判断有了可落库的复盘入口：cache 效率是否被用于降低成本、改善价格或推进自持评估。
- 后续真实 agent 可以替换 generate 的确定性规则，但不用改变 review / evidence schema。

## 验收

1. `OperatingInsight` 进入普通迁移和 fast migration。
2. `/api/operating_insights/generate` 能基于 `gb10-4t` 的 `TrafficProfile`、`SupplyDecision`、`PricingRecommendation` 生成 draft insight。
3. `/api/operating_insights` 能查询到该 insight，并校验 category、severity、linked decision / recommendation 和 evidence。
4. `/api/operating_insights/:id/acknowledge` 能把 insight 改为 acknowledged 并记录 operator。
5. 重复 generate 不覆盖已 review 状态。
6. `token-router-sim run` 能在真实进程链路中核验 operating insight API。
7. `aima2` 上 focused Go 测试和 `go test ./...` 通过，README 记录证据。
