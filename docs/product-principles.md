# token-router 产品设计理念与原则

> 本文定义 token-router 的"**为什么**"和"**以什么标准做**"。技术上"怎么做"见 [`architecture.md`](architecture.md)。
> 两份文档是同一件事的两层：architecture 是数据与路由的实现，本文是它要服务的经营理念。两者必须互相对得上。

---

## 一句话定位

**做 token 领域的"京东严选"——不做最便宜的 token 集市，做企业级品质与稳定性最值得信赖的 token 严选中转站。**

B 端 API 流转里，供给侧（企业富余的 token 流量）和需求侧（对质量、稳定性有刚性要求的企业）之间，缺一个"可信的品质与数据中枢"。token-router 要成为这一环：上游的 token 想进入高质量需求侧，最可信的路径是先过我们这一关；下游想买到稳、真、可对账的 token，第一选择是我们。

这个"**必经一环**"的地位不是预设的，是靠品控、SLA 和可信数据**挣来的**。

---

## 为什么是"京东"，不是"淘宝 / 拼多多"

这三种模式对应三种完全不同的产品哲学。我们明确选京东这一支：

| 维度 | 集市模式（淘宝 / 拼多多） | 严选模式（京东 / 我们） |
|---|---|---|
| 第一卖点 | 全、便宜、比价 | 品质、稳定、确定性 |
| 供给侧 | 谁都能上，海量长尾 | 严入选，有门槛，持续评级 |
| 品控 | 靠买家评价事后博弈 | 平台主动品控 + 数据背书 |
| 履约 | 不确定，看运气 | SLA 确定性承诺（对应京东"211 限时达"） |
| 信任来源 | 七天无理由 / 平台仲裁 | 正品 + 自营品控 + 全程可追溯 |

把京东的招牌能力翻译到 token：

- **正品保证** → **token 真实性与质量可信**：不降智、不偷工减料、不掺假路由，模型与质量所见即所得。
- **211 限时达（履约确定性）** → **SLA**：可用性、延迟、稳定性是被承诺、被监控、被兑现的，不是"尽力而为"。
- **自营品控** → **数据化品控**：用可信数据对品质做"自营级"把控。
- **供应商体系** → **上游入选 + 评级 + 优胜劣汰**。
- **自营 + POP 第三方双轨** → 见下。

### 供给三轨：第三方代销 + 自营垫资自采 + 自持算力

京东的结构是**自营 + POP 第三方 + 自建基础设施（物流）**。我们的供给侧同样分三轨，由轻到重、由代销到垂直一体化：

| 供给模式 | 对应京东 | 怎么运作 | 库存 / 资金风险 | 平台怎么记 |
|---|---|---|---|---|
| **第三方供给**（当前 / 起步） | POP 第三方商家 | 企业把自有 token 流量挂上来，平台路由 + 计量 + 线下结算，赚价差 | 无（轻资产） | `Supplier` + 协议价（按量后付） |
| **自营供给**（垫资自采，后续） | 京东自营采销 | 公司线下垫资采一批 token 容量，作自营库存持有再出售，吃完整价差 | 有（库存 + 垫资） | "自营 supplier" + 预付成本基准 + 库存核销 |
| **自持算力**（自建推理，最深） | 京东自建物流 / 垂直一体化 | 平台接入自有推理算力（org 的 Mooncake / SGLang / KTransformers KVCache 后端）直接承接选定流量，自己掌握 cache 与硬件 | 重（算力 capex / opex），但 cache 命中近乎免费、SLA 完全可控 | "自持 supplier" + 算力成本摊销（卡时 / 机时）→ cache-aware 单位成本 |

三轨对下游**完全一致**：同样的 cache-aware 计量、同样的质量 / SLA 标准、同样的对账数据。差别只在成本侧（按量协议 / 预付批量 / 算力摊销）和谁担风险。

**往哪一轨扩，由流量数据说话**：当流量画像显示某切片 **cache 局部性高 + 量稳 + SLA 愿付溢价**，自持算力吃下它的边际收益最高（cache 全归自己、硬件可控、SLA 可保）；量大但不必自建的，走自营采购锁成本；长尾、试探性的，留给第三方。**记录供需两端流量，就是为了能做这个判断**——设计见 [`traffic-and-supply.md`](traffic-and-supply.md)。

> **边界不变**：垫资自采与自持算力都是**线下公司经营 / capex 行为**——平台只把它们当成"有成本基准的供应商（自营 / 自持类型）"来**记账与核销**。平台软件本身仍不做支付 / 钱包 / 打款（architecture 定性 #1 不变）。扩的是**经营模式与基础设施**，不是把平台变成支付系统。

---

## 三个支点：为什么这套理念能自洽

"既要品质优先又要低成本""既要平台赚钱又要两端都更满意"——这些诉求听起来互相矛盾。三个支点让它们同时成立：

### 1. 经济支点：KV cache 让"共赢"成立（正和，不是零和）

普通的 token 二道贩子是零和的——平台毛利来自从某一端榨取。我们不是。会话亲和路由复用 KV cache，创造的是**真实的成本下降**（命中的 token 成本极低），这是技术新创造的剩余，不是从谁兜里掏的。这块剩余三方分：上游拿到 cache 友好、可预测的稳定流量，下游在同等质量下拿到更低价格，平台拿到健康毛利。

**因为有真实效率增量，"共赢"才不是口号。**

### 2. 信任支点：可信数据是平台唯一的真资产

不管走第三方代销还是自营垫资自采，平台软件**唯一的真资产都是数据**——比任何人都更精准、更诚实地度量质量与成本，这是被信任、能成为"必经一环"的根本。cache-aware 计量、每笔调用一行可对账台账、幂等不重复计——这套数据能力不是记账后台，它**就是产品本身**。质量评级、SLA 兑现、定价、对账，全部长在这层数据上。

**数据一旦不可信，整个生意就塌了。**

### 3. 运营支点：AI agent 让"品质优先"与"低成本"同时成立

京东式品控很贵——选品、质检、运营、客服都是人力。如果用人肉做 token 严选，"低运营成本"就破产了。解法是把经营交给 AI agent：入选测试、品控监控、供应商评级、定价与分流优化、策略迭代，都由 agent 执行与驱动。

**AI 不是一个功能，它是"高品质 + 低成本"能并存的前提。**

---

## 核心原则

> 每条都落到平台的真实机制上，不是口号。`→ 落点` 标出它在系统里的对应物。

**P1　严选，不做集市。**
品质与确定性优先于最低价。宁可上游少而精，不要多而杂。下游为"稳和真"付费，不是为"最便宜"而来。
`→ 落点`：上游有入选门槛；不追求渠道数量，追求每个渠道的质量分。

**P2　无数据，不承诺。**
每一条质量主张、每一个成本数字、每一次对账，都必须有可追溯、可复算的数据支撑。没有数据支撑的承诺一律不做。
`→ 落点`：`UsageLedger` 每调用一行、RequestId 幂等、cache 拆分、双价台账；对账数据可导出复核。

**P3　先度量，再承诺 SLA。**
质量是被测出来的，不是被声明的。SLA 分层（延迟 / 可用性 / 稳定性等级）必须对应可验证的度量，分层定价对应分层质量。
`→ 落点`：命中率、延迟、错误率、稳定性按 supplier / channel / model 持续统计，成为 SLA 等级与定价依据。

**P4　供应商优胜劣汰。**
严入选 + 持续评级 + 动态分流。好上游拿到更多流量与更优结算，差上游被挤出。评级是动态的，不是一次性准入。
`→ 落点`：入选测试评分 + 运行期质量分；亲和路由 / 权重向高分上游倾斜；低分上游降权直至淘汰。

**P5　经营即算法，但人握方向盘。**
入选测试、运营分析、品控、定价、策略优化由 AI agent 驱动——这是低运营成本的根本，也是持续优化的引擎。**当前模式：agent 出洞察与建议，人在 dashboard 上决策批准后才生效，定价尤其必须人审。** 放权给 agent 自动执行（调参 / 分流）是更后面、需单独评估的事。
`→ 落点`：agent 跑入选测试套件并打分、持续读台账与质量数据，产出定价 / 分流权重 / SLA 阈值的调整建议；经人在 dashboard 批准后落库生效。

**P6　做大正和，不做两端博弈。**
平台毛利来自技术与运营创造的真实效率（cache 复用、AI 低成本运营），不来自对某一端的信息差或榨取。每个决策先问一句："这让三方的总盘子变大了吗？"
`→ 落点`：定价与分流以"总剩余最大化 + 三方都不更差"为目标，而非单纯抬卖价或压成本价。

**P7　理念是假设，不是教条。**
这套经营理念本身就是待验证的假设，由 AI 持续用数据校验。指标证伪了，就敢推翻、敢重写——包括这份文档。
`→ 落点`：经营策略以数据反馈闭环迭代；定期复盘哪些假设被数据支持 / 证伪。

**P8　守住边界：平台软件只做数据与信任中枢。**
平台软件不做支付 / 钱包 / 打款——这条不变。公司可以线下垫资自采（自营供给），但那是线下财务行为，平台只把它当一个"有成本基准的自营供应商"记账核销，软件本身不碰资金托管。信任来自透明可对账的数据。
`→ 落点`：与 architecture 定性 #1 一致——真实资金流（含自营垫资）走线下，平台只产出对账数据 + 自营库存核销。

**P9　流量即情报。**
需求端与供给端的每一笔流量都要被记录——不只为对账，更是供给战略的情报。哪里需求密集、cache 局部性高、SLA 愿付溢价，数据会告诉你；据此决定招募哪些第三方上游、哪些切片值得自营采购、哪些切片值得自持算力直接吃下。**流量数据是平台扩张方向的方向盘。**
`→ 落点`：在 `UsageLedger`（事实台账）之上加供给容量遥测 + 流量画像聚合层 + 流量预测层 + 供给决策层；设计见 [`traffic-and-supply.md`](traffic-and-supply.md)。

---

## 这套理念对路线图的约束

理念要能管住取舍，否则只是装饰。几条硬约束：

- **度量先于承诺**：没有质量度量能力之前，不对外承诺任何 SLA 等级。→ 质量统计报表优先于 SLA 商业化。
- **入选先于扩张**：入选 + 评级体系（哪怕最简版）要早于"多接上游"。宁可慢，不可滥。
- **数据可信先于一切花活**：台账的精准、幂等、cache 拆分是地基，任何上层（评级 / SLA / 定价 / AI 经营）都不能绕过它另起炉灶。
- **AI 经营：洞察归 agent，决策归人（dashboard 模式）**：当前 agent 只出洞察与建议，人在 dashboard 决策批准（定价尤其必须人审）；opportunity 进入 action plan 时只复制证据，不绕过 approved decision；自动接管执行（调参 / 分流）是更后面、需单独评估的事。
- **先记录，后扩张**：供给三轨往外扩之前，需求 + 供给两端的流量遥测要先到位——没有流量画像就盲目加供应商 / 自建算力，等于拍脑袋下注。
- **自营 / 自持晚于第三方、更晚于数据地基**：垫资自采与自持算力要等第三方供给跑通、台账与成本核算稳了再上；它们新增"成本基准 + 库存 / 算力核销"，必须复用同一套 cache-aware 计量，不另起炉灶。
- **边界不漂移**：把**平台软件**推向"支付 / 钱包 / 打款执行"的需求，默认拒绝——自营垫资是线下财务行为，不等于让软件碰资金。

---

## 附：与 architecture 的对应关系

本文定"标准与理念"，architecture 定"实现"。两者对得上的地方与还需补的地方：

| 原则 | 当前 architecture 状态 |
|---|---|
| P2 / P3 / P8 | **已对齐**——直接对应 cache-aware 计量、`UsageLedger`、两条定性 |
| P1 / P4 | **已部分对齐**——`Supplier` 有三轨类型和 runtime posture，`SupplierScorecard` / `SupplierEvaluation` / SLA admission evidence 已形成准入与复评控制面；`SupplierPostureRecommendation` 已把 scorecard 与 open quality/capacity insight 物化为 observe / boost / downgrade / disable 人审建议，`Posture` dashboard 已提供 generate / approve / reject / apply 人审入口，approved `disable` apply 会进入既有 runtime supplier gate，approved `downgrade` apply 会创建 `SupplierRoutePreference(weight_percent=25)` 并在 normal channel selection 中降低候选权重，approved `boost` apply 会创建 `SupplierRoutePreference(weight_percent=150)` 让优秀 supplier 获得更多 normal-routing 候选权重；active route preference 的来源、权重、effective window 与 operator evidence 已在 `Posture` dashboard 展示，operator 也可手工 activate/disable `1..200` bounded route preference；自动淘汰与 agent 自动调权仍需后续 ADR |
| 自营 / 自持供给 | **已部分对齐**——`Supplier.Type`、`SupplyDecision`、`SupplyActionPlan`、`SupplyActionExecution`、`SupplyRoutingPolicy` 已打通 self-hosted 人审路由闭环，policy 激活已要求 passed runtime SLA evidence，并可用 `traffic_percent=1..100` 做 deterministic session canary；`Routing Policies` dashboard 已可在 activation dialog 中显式选择并回看 traffic share；policy miss 会进入 `OperatingInsight` 复盘面，canary bucket 外流量则正常回到普通 channel selection 且不写 miss insight；`SupplyCapacity.used_tokens` 已可从业务 ledger 回填，`SupplyCapacityTelemetry` 已可记录 GPU utilization 与容量来源证据，也可从已配置 channel upstream 主动 collect、按已有 capacity snapshot sweep，或由 `token-router-supply telemetry sweep` runner / `token-router-supply telemetry agent` / 显式启用的 API server telemetry worker 在部署侧周期调用；`SupplyTelemetryAgent` 已能记录常驻 agent / worker heartbeat 与最近 sweep 摘要；capacity telemetry 缺失 / stale / hot GPU / low headroom 已可进入 `OperatingInsight(category=capacity_risk)` 复盘面；`SupplyCostProfile` 已可记录 self-hosted 固定成本、可变单位成本和摊销单位成本，并把 savings 接入 `SupplyExpansionOpportunity.rank_score`，dashboard 已可记录/展示 cost profile 与 opportunity savings evidence；`SupplyPrepaidLot` 已可记录 self-operated 线下预付批次、从业务 ledger refresh drawdown / remaining / usage source，并已在 dashboard 查询/记录/刷新；`SupplyActionExecution` 已可从业务 ledger refresh execution 级 drawdown / remaining / usage source，用于库存 / 算力核销 read model |
| P9（流量情报） | **已部分对齐**——`UsageLedger` 已补 session/cache/latency/status/SLA/supply node，`SupplyCapacity` 可显式 refresh 用量并链接 record、upstream-collected、sweep-collected、runner-collected、agent-collected 或 API-worker-collected telemetry evidence，`SupplyTelemetryAgent` 可证明部署侧采集进程 / API worker 存活与最近 sweep 结果，capacity telemetry risk 与 active routing policy fallback 已能生成 operator insight，`TrafficProfile` 已能沉淀画像，`TrafficForecast` 已能物化下一周期 recency-weighted forecast，并可显式生成 seasonal/anomaly-aware forecast evidence，在 dashboard 展示 source/target window、confidence 与 gap，`SupplyDecision` 已能优先使用匹配 forecast evidence 驱动三轨建议，`SupplyExpansionOpportunity` 已能把供给缺口 / self-hosted cache 机会物化为可排序 read model，并可接入和展示 `SupplyCostProfile` 的 self-hosted savings evidence；`SupplyPrepaidLot` 已能把 self-operated 预付批次和 ledger-backed drawdown 变成 dashboard 可查询/记录/刷新的 read model；后续仍需更复杂 ML / 外部数据 forecast、自动复盘与自动执行边界 |
| P5 / P7 | **已部分对齐**——agent-readable recommendations / insights 已落库，`token-router-supply review once` 可由 cron / systemd timer 一次性刷新 scorecard、posture、traffic、forecast、pricing、decision、opportunity 与 operating insight 这些 review read models，`token-router-supply review agent` 可用常驻 interval loop 周期刷新同一 review set，`deploy/systemd/` 已把 API / telemetry / review 进程固化为可迁移部署模板，`deploy/smoke/token-router-gb10-process-smoke.sh` 已把 strict gb10 mock、API-only server、demand simulator 与 review agent 单轮 proof 变成可复现 smoke；dashboard 负责 approve / reject / acknowledge / apply / activate。supplier posture 建议也已进入同一人审 dashboard 模式，approved downgrade / boost 的 route preference overlay 仍是人审 apply 后才生效，manual route preference 也要求 operator 显式提交 reason / note / effective window，并可在 dashboard 回看 active overlay 证据；self-hosted routing policy activation 也要求 operator 显式提交 `traffic_percent` canary share，不再由 UI 隐式按 `100%` 生效；自动执行策略仍需后续单独评估 |
