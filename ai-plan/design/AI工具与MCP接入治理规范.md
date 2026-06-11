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

MCP 和 skill 不得定义第二套启动 receipt、第二套 validation truth、第二套 commit 规则或第二套恢复入口。

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
  - 规范：`ai-plan/design/CodeGraph-MCP-辅助开发规范.md`
  - 约束：不进入项目依赖、CI、hooks 或 runtime。
- `tdesign`
  - 等级：`L0`
  - 用途：TDesign Vue Next 组件列表、文档、DOM、changelog 查询。
  - 规范：`ai-plan/design/TDesign-MCP-辅助开发规范.md`
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
    codex mcp add headroom -- /home/gewuyou/project/go/Graft-wt/feat/wt-audit-plugin-mvp/.ai/venv/bin/headroom mcp serve
    ```

  - 说明：该安装方式只记录本地 MCP 可用事实，不把 Headroom 变成项目必装依赖、运行时依赖或 CI 依赖。
  - 采用条件：任务需要处理大体量 `rg` 搜索输出、长日志、大文件、OpenAPI bundle、测试 / 构建 / lint 长输出、历史设计文档或归档材料摘要，且普通真实文件读取和现有 MCP 不能有效控制上下文成本。
  - 约束：
    - 默认关闭匿名 telemetry：使用 `HEADROOM_TELEMETRY=off` 或等价本地配置。
    - MCP 默认只用于压缩、检索和统计上下文。
    - 精确验证、调试、失败排查、迁移、发布检查时必须保留原始命令输出，或明确说明已使用压缩输出并在必要时回退 raw output。
    - 不得在项目治理文件中要求 Agent `always prefix with rtk`；RTK / Headroom 自动注入块会把本地可选压缩工具升级成项目级强制 Agent 行为，影响团队、CI 和其它 Agent。
    - Headroom / RTK 不得自动写入根 `AGENTS.md`、`server/AGENTS.md`、`web/AGENTS.md`、`/root/.codex/AGENTS.md`、`CLAUDE.md`、`GEMINI.md` 或 Codex `instructions.md`。
    - Headroom memory 仅允许作为受控本地试点写入 `.ai/headroom/memory/**`，该目录必须由 `.gitignore` 排除，且不得替代 `ai-plan/public/**` 的 topic recovery 真值。
    - `headroom learn` 仅允许作为候选 lesson 生成器试点写入 `.ai/headroom/learn/**`，该目录必须由 `.gitignore` 排除；learn 输出不能直接写入 `AGENTS.md`、设计文档或 `ai-plan/lessons/**`，必须人工 review 后再走 `graft-lessons-learned` 或对应治理路径。

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

## 6. Python Helper 规则

Python helper 用于仓库自动化和 AI 辅助验证时：

- 默认使用标准库；只有重复复杂解析或强 schema 校验需要时才引入依赖。
- 当前脚本测试默认使用 `unittest`，不要为了少量脚本提前引入 `pytest`。
- 新脚本应支持从仓库根运行，并避免读取 `.ai/venv/**`、`.codegraph/**`、`node_modules/**` 等派生产物。
- 脚本失败信息必须指向可修复的文件和规则。
- 如果脚本变成完成态验证入口，必须在根 `AGENTS.md` 或相应 skill 中明确定位为 docs/automation 结构检查，而不是 `server` 或 `web` 的第二套完成标准。
- 当 MCP 能稳定提供实时上下文时，skill 应优先使用 MCP 简化人工探索，把 Python helper 保留为可复现结构化输出、
  离线 fallback 或批量固化步骤。

## 7. Closeout Evidence

使用 AI 工具或 MCP 的任务，在 closeout 中记录：

```text
AI tooling evidence:
- tools: codegraph / tdesign / context7 / github / playwright / headroom / none
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
