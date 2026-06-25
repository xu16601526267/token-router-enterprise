# Git 工作树交接说明

此仓库由用户上传的源码 ZIP 重建为可继续开发的 Git 工作树。

## 分支与历史

- `main`：当前企业版改造进度，包含 B/C 双模式 UI、全中文侧边栏、企业控制塔、企业 API Key、渠道中心、用量分析、用户治理与计费中心等当前代码。
- `baseline-uploaded`：用户最初上传的仓库快照。
- 标签 `uploaded-baseline`：原始快照提交。
- 标签 `enterprise-progress-2026-06-25`：当前改造进度提交。

> 原始上传包不包含 `.git` 目录，因此无法恢复上游仓库原有提交历史、作者信息及远程地址。本仓库中的两次提交是为了清晰保留“原始快照 → 当前改造”的可审阅差异而重新构建的。

## 常用命令

```bash
# 查看当前状态
git status

# 查看改造提交
git log --oneline --decorate --graph --all

# 查看相对原始快照的全部变化
git diff baseline-uploaded..main

# 只列出改动文件
git diff --name-status baseline-uploaded..main

# 回到原始快照进行比较
git switch baseline-uploaded

# 返回当前企业版
git switch main
```

## 视觉与继续开发资料

- Codex 接手说明：`CODEX_HANDOFF.md`
- 8 张 Image 2 UI 基准：`docs/codex-handoff/ui-reference/`
- 剩余任务：`docs/codex-handoff/Token_Router_企业版_剩余待完成任务清单.docx`
- 已改文件清单：`docs/codex-handoff/CHANGED_FILES.md`

## 验证提示

前端采用 React 19 + TypeScript；后端采用 Go + Gin + GORM。具体环境、构建和后续验收要求请先阅读 `CODEX_HANDOFF.md`。
