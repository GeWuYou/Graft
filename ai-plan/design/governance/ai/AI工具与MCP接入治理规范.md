# AI 工具与 MCP 接入治理规范

## 1. 目标

这份文档定义 `Graft` 引入 AI 开发工具、MCP Server、仓库 skill 和 Python helper 的统一治理边界。

目标：

- 提升 AI 探索、评审、浏览器检查和外部文档查询效率。
- 保持根 `AGENTS.md`、`ai-plan/design/`、`web/AGENTS.md`、`server/AGENTS.md` 的 source of truth 地位。
- 防止 MCP、skill 或脚本变成隐藏运行时依赖、第二套验证入口或隐式恢复状态。

非目标：

- 不把 MCP Server 加入 `server/go.mod`、`web/package.json`、CI、hooks 或业务运行时。
- 不要求所有贡献者安装相同的个人 MCP 客户端配置。
- 不用第三方工具查询结果替代真实源码、设计文档、OpenAPI authority 或仓库验证命令。

## 2. 分层原则

AI 工具按职责分层：

- `AGENTS.md`
  - 仓库启动、authority-first、验证、提交、closeout、subagent 继承规则。
- `ai-plan/design/**`
  - 长期设计真值、工具接入规范、MCP 风险边界。
- `.agents/skills/**`
  - 可复用工作流、具体命令顺序、可选 helper 脚本。
- `.ai/environment/**`
  - 生成的本机工具能力事实。
- MCP Server
  - 开发时知识源或交互辅助源。
- 仓库 / 团队级 Codex plugin 与外部 skill pack
  - AI guidance asset，用于沉淀可复用提示、前端生成规则和浏览器 QA 流程。
  - 不属于 `server` / `web` runtime、CI、hooks、package manager 依赖或仓库 authority。

MCP 和 skill 不得定义第二套启动 receipt、第二套 validation truth、第二套 commit 规则或第二套恢复入口。
workflow 级 thin skill 也不得定义第二套 intake truth；如果存在 `graft-work-intake`，它只能执行文档已经批准的
`Work Intake` 规则。

仓库根 `Justfile` 属于 optional developer entrypoint / convenience layer：

- 可以推荐给贡献者作为 first-run 或 common workflow shortcut。
- 不得被表述为新的 startup authority、validation truth、runtime contract 或 CI contract。
- `just check`、`just lint`、`just generate` 等聚合 recipe 只能包装既有 authoritative commands，例如 `graft validate backend`、`bun run check`、`web/package.json` scripts、`scripts/**`、`docker compose` 与 GitHub Actions 已持有的真实 authority。
- 复杂逻辑仍应保留在现有 Go CLI、`web/package.json`、`scripts/**`、`docker compose` 和 GitHub Actions 中；Just recipe 负责可选的人机入口整合，而不是重定义规则。

当仓库把某类治理沉淀为 AI guardrail 时，落地顺序应优先遵循：

- 规范
- `AGENTS.md`
- CI / validation script
- review checklist

不要只补设计文档却不更新 `AGENTS.md`、校验脚本或 review 口径；否则多人协作和多 agent 场景下这些规则很快会失效。

## 3. MCP 风险等级

接入 MCP 前必须先按能力分级：

| 等级 | 能力 | 默认策略 |
| --- | --- | --- |
| `L0` | 本地只读知识源，如组件文档、代码图查询 | 可采用；记录 closeout evidence |
| `L1` | 远程只读知识源，如第三方库文档、PR/check 读取 | 可试点；限制 token 权限和用途 |
| `L2` | 写能力工具，如 GitHub 评论、PR 修改、issue 修改 | 仅在仓库 skill 明确包裹后使用 |
| `L3` | 凭证、数据库、生产数据、shell 写能力 | 默认拒绝；需要单独设计和最小权限证明 |

所有 MCP 接入都必须遵守：

- 最小权限。
- 只在任务需要时启用。
- 不提交个人客户端配置、token、数据库 DSN 或私有 IDE 配置。
- 不把 MCP 可用性作为任务完成硬门禁。
- MCP 输出只是证据线索；最终结论必须回到仓库 authority 和真实文件。

## 4. 当前采纳清单

### 4.1 Adopted

- `codegraph`
  - 等级：`L0`
  - 用途：符号定位、调用链、影响面、模块入口探索。
  - 规范：`ai-plan/design/governance/ai/CodeGraph-MCP-辅助开发规范.md`
  - 约束：不进入项目依赖、CI、hooks 或 runtime。
- `tdesign`
  - 等级：`L0`
  - 用途：TDesign Vue Next 组件列表、文档、DOM、changelog 查询。
  - 规范：`ai-plan/design/governance/frontend/TDesign-MCP-辅助开发规范.md`
  - 约束：只作为前端组件知识源。

### 4.2 Pilot

- `context7`
  - 等级：`L1`
  - 用途：查询 Go、Vue、Vite、Pinia、Gin、Ent、Casbin 等第三方库的当前文档。
  - 接入方式：用户级 Codex MCP 配置；当前推荐 stdio 入口：

    ```bash
    codex mcp add context7 -- npx -y @upstash/context7-mcp@latest
    ```

  - 采用条件：需要外部库当前文档，且本仓库源码或设计文档不能回答。
  - closeout：记录查询的库、采用程度和 fallback 原因。
- `github`
  - 等级：`L1` 默认；开启评论或 PR 修改时视为 `L2`
  - 用途：读取当前 PR、review thread、Actions check 和失败日志，补强 `graft-pr-review`。
  - 接入方式：用户级 Codex MCP 配置，默认只读和最小 toolsets：

    ```bash
    codex mcp add github -- bash -lc 'GITHUB_PERSONAL_ACCESS_TOKEN="$(gh auth token)" exec docker run -i --rm -e GITHUB_PERSONAL_ACCESS_TOKEN -e GITHUB_TOOLSETS="context,repos,pull_requests,actions" -e GITHUB_READ_ONLY=1 ghcr.io/github/github-mcp-server'
    ```

  - 采用条件：PR 审查或 CI 排障需要 GitHub 实时状态。
  - 约束：写操作必须通过 `graft-pr-review`、`graft-pr-create` 或后续专门 skill 的显式流程。
- `playwright`
  - 等级：`L1`
  - 用途：作为 `graft-web-browser-agent` 的探索层，快速识别页面结构、role、label、TDesign 弹窗/抽屉交互和稳定 selector。
  - 接入方式：用户级 Codex MCP 配置：

    ```bash
    codex mcp add playwright -- npx -y @playwright/mcp@latest --headless --browser=chromium
    ```

  - 采用条件：页面结构未知、交互复杂、需要先探索再固化为可复现 browser artifact。
  - 约束：MCP 只负责探索；可审计截图、文本、登录状态和 summary 仍由 `graft-web-browser-agent` 脚本产出。
- `headroom`
  - 等级：`L1` 默认；启用 memory / learn 的受控本地试点时仍不得越过本节目录隔离和人工确认边界。
  - 定位：optional / local / user-level / MCP-based AI context compression tool。
  - 用途：通过 Codex MCP 按需压缩、检索和统计上下文，降低 AI 辅助开发时的大型 tool output / log / file context token 压力；MCP 工具默认包括 `headroom_compress`、`headroom_retrieve` 和 `headroom_stats`。
  - 推荐接入方式：用户本地 Python 工具安装，并以 Codex MCP server 试点接入；当前已验证入口：

    ```bash
    .ai/venv/bin/python -m pip install "headroom-ai[proxy]"
    codex mcp add headroom -- .ai/venv/bin/headroom mcp serve
    ```

  - 说明：该安装方式只记录本地 MCP 可用事实，不把 Headroom 变成项目必装依赖、运行时依赖或 CI 依赖。
  - 采用条件：任务需要处理大体量 `rg` 搜索输出、长日志、大文件、OpenAPI bundle、测试 / 构建 / lint 长输出、历史设计文档或归档材料摘要，且普通真实文件读取和现有 MCP 不能有效控制上下文成本。
  - 约束：
    - 默认关闭匿名 telemetry：使用 `HEADROOM_TELEMETRY=off` 或等价本地配置。
    - MCP 默认只用于压缩、检索和统计上下文。
    - 精确验证、调试、失败排查、迁移、发布检查时必须保留原始命令输出，或明确说明已使用压缩输出并在必要时回退 raw output。
    - 不得在项目治理文件中要求 Agent `always prefix with rtk`；RTK / Headroom 自动注入块会把本地可选压缩工具升级成项目级强制 Agent 行为，影响团队、CI 和其它 Agent。
    - Headroom / RTK 不得自动写入根 `AGENTS.md`、`server/AGENTS.md`、`web/AGENTS.md`、仓库外的 Codex 用户级指令文件、`CLAUDE.md`、`GEMINI.md` 或 Codex `instructions.md`。
    - Headroom memory 仅允许作为受控本地试点写入 `.ai/headroom/memory/**`，该目录必须由 `.gitignore` 排除，且不得替代 `ai-plan/public/**` 的 topic recovery 真值。
    - `headroom learn` 仅允许作为候选 lesson 生成器试点写入 `.ai/headroom/learn/**`，该目录必须由 `.gitignore` 排除；learn 输出不能直接写入 `AGENTS.md`、设计文档或 `ai-plan/lessons/**`，必须人工 review 后再走 `graft-lessons-learned` 或对应治理路径。

### 4.2.1 Optional Local Developer Tools

- `eff-u-code` / `fuck-u-code`
  - 等级：`L1`
  - 定位：developer-local optional code quality helper
  - 用途：对 `server` 与 `web/src` 做手动热点扫描，辅助发现大文件、重复、复杂度和结构问题
  - 接入方式：仓库根级项目依赖安装；当前项目级入口：

    ```bash
    bun install
    bun run quality:eff-u-code -- all
    ```

  - 仓库入口：通过 `bun run quality:eff-u-code -- <scope>` 调用本地安装的 `fuck-u-code analyze ...`
  - 仓库策略入口：通过 `bun run quality:eff-u-code:gate -- ...` 调用仓库自定义评估器；`eff-u-code` 只提供原始 JSON 质量信号，真正的 Gate owner 是 `Graft Quality Policy`
  - 本地集中治理入口：通过 `bun run quality:eff-u-code:gate:server`、`bun run quality:eff-u-code:gate:web`、`bun run quality:eff-u-code:gate:all` 运行可选的全项目扫描
  - hook / CI 评分入口：通过 `bun run quality:eff-u-code:score:changed`、`bun run quality:eff-u-code:score:server`、`bun run quality:eff-u-code:score:web`、`bun run quality:eff-u-code:score:all` 运行仓库派生的 Scope / Project Score gate
  - 约束：
    - not part of the formal validation flow
    - 不是正式 validation flow 的一部分。
    - 不得把 upstream `eff-u-code` 原始总分直接接入 `graft validate backend`、`bun run check`、完成态或 blocking gate。
    - 如需进入 PR 级门禁，只允许“基于 eff-u-code 的仓库自定义评估器”进入独立 CI job；不得把工具原始分数当成 acceptance contract。
    - 不得加入 `server/go.mod`、`web/package.json`、runtime 脚本或部署流程。
    - 仓库只采纳本地 `analyze` 能力；`mcp-install`、`ai-review`、`update`、`uninstall` 不属于仓库工作流。
    - 输出只作为本地热点线索，不得替代真实 lint/test/authority 结论。
    - `Curated Score` 仅用于展示，不参与阻断；阻断必须完全由仓库规则集决定。
    - 仓库可以基于文件级 `Curated Score` 再派生 Scope / Project Score 作为 hook / CI gate，但 acceptance contract 仍然属于仓库自定义评估器，而不是 upstream `eff-u-code` 原始总分。
    - 文档 / 治理 Gate 必须独立于 `eff-u-code`：README、ADR、Contract、OpenAPI、Public API Comment 等结构化治理规则属于 `Graft`，不是 `eff-u-code` 的一部分。
    - PR 门禁默认使用 changed / incremental 语义；本地全项目扫描是可选的集中治理入口，不得把二者混成同一套阻断语义。
    - PR 门禁应优先拦截可操作的高价值问题；对仓库已接受的声明式镜像、扫描器/映射器对称实现、受治理允许的结构性重复，应通过仓库规则集显式降噪，而不是让其长期作为阻断噪声存在。
    - 全项目扫描应至少支持 `server`、`web` 和 `all` 三个本地入口，便于按前后端分治或集中治理。
    - Project Score Gate 的终端输出应采用 layered diagnostics：默认先给 Scope / Project Score、Threshold、Coverage、Top Contributors、Rule Category Summary、Severity Summary 和 Potential Score Gain，再按 Top N 展开详情；避免直接输出“最差代码排行榜”或全量规则刷屏。

### 4.3 Rejected By Default

- `filesystem` / `git` MCP
  - 原因：Codex 已有本地文件和 Git CLI 能力；重复能力会增加权限面。
- `memory` MCP
  - 原因：会制造 `ai-plan/public/**` 之外的隐藏恢复真值。
- `postgres` / 数据库 MCP
  - 原因：涉及凭证和数据读取风险；仅在明确运行时数据排查主题中单独设计。
- 通用 shell 写能力 MCP
  - 原因：会绕过仓库已有命令、验证和提交治理。

## 5. Skill 设计规则

新增或修改仓库 skill 时：

- 先确认是否已有 skill 可以扩展，避免重复工作流。
- `SKILL.md` 只放触发条件、关键流程、拒绝条件和必要命令。
- 复杂、可重复、易错的步骤优先放到 `scripts/**`。
- 若新增 `agents/openai.yaml`，默认只写 `display_name`、`short_description`、`default_prompt`。
- skill 不得要求读取无关大文件，不得把 task tracking 写到 `.agents/skills/**`。
- skill 的脚本必须能在无项目运行时服务的情况下做最小结构验证。
- `graft-ai-plan-governance` 是 `ai-plan/**` router、active topic recovery、template、catalog 与 bounded validator
  的窄治理 skill；它必须回到 root `AGENTS.md`、`ai-plan/AGENTS.md`、`ai-plan/README.md` 与
  `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`，不得定义第二套 startup receipt、validation truth 或 recovery
  path。
- `graft-work-intake` 是 workflow orchestrator，而不是内容生产 skill；它只负责 `Work Intake`、`Work Contract`、
  contract-driven minimal bootstrap 和 dispatch，所有领域正文仍由对应专业 skill 负责。
- `graft-ai-governance-audit` 保持 broader AI tooling / MCP / skill / inventory drift audit；当同一批次同时修改
  `.agents/skills/**`、`scripts/**` 或本规范时，可与 `graft-ai-plan-governance` 组合使用。
- `graft-table-design` 是数据库表设计、Ent schema、migration、审计字段、软删除、索引和数据库注释的治理
  skill；它必须回到 `ai-plan/design/governance/backend/数据库表设计与迁移规范.md`，不得定义第二套 schema 或 migration 真相。
- `graft-sql-migration` 是创建或修改 live Atlas/PostgreSQL migration SQL 的专用治理 skill；SQL 注释硬规则、
  live / legacy 边界和 `scripts/validate_sql_migrations.py` 校验要求以该 skill 和
  `ai-plan/design/governance/backend/数据库表设计与迁移规范.md` 为准。
- `graft-shared-asset-reuse` 是共享资产发现与 registry 维护评估的治理 skill；registry 位于
  `.ai/registries/**`，它是人工策展索引，不是全量源码 inventory，也不得替代 `web`、`server`、OpenAPI 或
  module contract authority。
- repo-local plugin `plugins/graft-frontend-vibe-toolchain` 是项目级 Codex guidance asset，用于把外部文章、
  UI 示例和 vibe-coding 技法翻译成 Graft 的 Vue 3、TDesign Vue Next、页面类型、i18n、token/theme 和浏览器
  QA 规则；它不得修改 `web/package.json`、引入 React / shadcn / Tailwind 运行基线，或替代 `DESIGN.md`、
  `web/AGENTS.md`、TDesign MCP 规范和 `bun run check`。
- 外部 frontend skill pack 只能作为输入素材；采纳前必须收口成仓库 skill / plugin 或设计文档规则，并明确
  fallback、禁用项、验证命令和 authority owner。

## 6. Python Helper 规则

Python helper 用于仓库自动化和 AI 辅助验证时：

- 默认使用标准库；只有重复复杂解析或强 schema 校验需要时才引入依赖。
- 当前脚本测试默认使用 `unittest`，不要为了少量脚本提前引入 `pytest`。
- 新脚本应支持从仓库根运行，并避免读取 `.ai/venv/**`、`.codegraph/**`、`node_modules/**` 等派生产物。
- 脚本失败信息必须指向可修复的文件和规则。
- 如果脚本变成完成态验证入口，必须在根 `AGENTS.md` 或相应 skill 中明确定位为 docs/automation 结构检查，而不是 `server` 或 `web` 的第二套完成标准。
- 当 MCP 能稳定提供实时上下文时，skill 应优先使用 MCP 简化人工探索，把 Python helper 保留为可复现结构化输出、
  离线 fallback 或批量固化步骤。
- `scripts/validate_shared_asset_registries.py` 只做轻量 registry 结构检查：解析 YAML、校验必填字段、允许类型、
  已登记路径和 examples 存在、拒绝 generated-only authority 与本地 / 构建产物路径；它不得自动生成 registry，
  新发现的未登记文件最多作为 warning，不作为阻断错误。

## 7. Closeout Evidence

使用 AI 工具或 MCP 的任务，在 closeout 中记录：

```text
AI tooling evidence:
- tools: codegraph / tdesign / context7 / github / playwright / headroom / eff-u-code / none
- mcp_queried: yes | no | fallback
- risk_level: L0 | L1 | L2 | L3 | not-applicable
- adoption: adopted | partially_adopted | not_adopted
- authority_verified_by: files/docs/validation command
- reason: <why used or skipped>
```

没有使用 MCP 时也应记录 `tools: none` 或在对应 skill 的 closeout 字段中说明不适用。

## 8. 评审清单

评审 AI 工具治理变更时至少检查：

- 是否引入运行时依赖、CI 硬依赖、hook 硬依赖或个人配置提交。
- 是否把 MCP 输出当成 source of truth。
- 是否绕过根 `AGENTS.md` 的 startup、validation、commit、closeout 规则。
- 是否与 `.ai/environment/tools.ai.yaml` 的工具事实冲突。
- 是否能在 MCP 不可用时退回 `rg`、真实文件读取、官方文档或现有 CLI。
- 是否有结构性验证覆盖新增文档或 skill。
- 对于 `eff-u-code` 这类本地 helper，是否明确写明它不属于正式 validation flow。

评审 AI guardrail 治理变更时至少追加检查：

- 是否同时更新了对应 `AGENTS.md` 入口，而不是只新增孤立设计文档。
- 是否区分了必须写成 agent guardrail 的规则与只适合保留在设计文档中的规则。
- 是否把可机检规则收口进现有 validation script、lint 或测试，而不是只靠人工记忆。
- 是否避免使用注释覆盖率、文档覆盖率这类 AI 易作弊指标。
- 是否为多 agent closeout 规定最小证据字段，例如修改范围、影响模块、风险点、验证步骤和回滚方案。
