# token-router enterprise backend

> `develop` 分支是后端专用基线，用于承接后续独立界面重构。

本仓库基于 `new-api` / QuantumNous 生态改造，当前分支只保留 Go 后端能力：API 中转、租户/客户/API Key 管理、供应商与模型路由、用量计费、对账结算、运营治理和后台任务。`develop` 分支不包含旧界面代码，后续界面重构应通过后端 API 重新接入。

## 当前范围

- OpenAI 兼容中转接口与渠道转发。
- A/B/C 多层级租户、客户、成员和 API Key 管理。
- 供应商、模型、渠道、路由策略和可用性治理。
- 请求日志、用量归集、成本/收入核算、缓存命中计费和对账单。
- 钱包、充值、兑换码、优惠券和结算相关后端接口。
- SLA、资源风险、运营事件、经营洞察和审批类后端能力。
- 后台 worker：用量出站、遥测采集、异步归集和定时任务。
- Docker、Compose、Makefile 和 Go 构建流程均面向后端服务。

## 分支说明

- `main`：当前稳定工作分支，包含旧版本线上演进记录。
- `develop`：后端专用开发分支，供后续独立界面重构使用。

## 开发环境

推荐 Go 1.22+。数据库可使用 SQLite、MySQL 或 PostgreSQL；Redis 根据部署配置启用。

常用命令：

```bash
go test ./...
go build -o bin/token-router .
docker compose -f docker-compose.dev.yml up -d
```

## 配置

复制 `.env.example` 后按环境调整：

```bash
cp .env.example .env
```

常见配置项包括数据库、Redis、日志、请求中转、计费、遥测和后台任务开关。`develop` 分支只保留后端服务配置。

## 后端接口分区

- `/api/*`：控制台、租户、账号、计费、结算、运营治理等管理接口。
- `/v1/*`：OpenAI 兼容中转接口。
- `/dashboard/*`：后端看板数据接口。
- `/video/*`：视频相关中转接口。

## 验证

当前 `develop` 分支已按后端服务验证：

```bash
go test ./...
go build -o /tmp/token-router-develop-api .
```

本地环境如未安装 Docker，则 Docker 镜像构建需要在具备 Docker 的机器上执行。

## 许可证

沿用上游项目许可和本仓库补充声明。第三方依赖说明见 `NOTICE` 和 `THIRD-PARTY-LICENSES.md`。
