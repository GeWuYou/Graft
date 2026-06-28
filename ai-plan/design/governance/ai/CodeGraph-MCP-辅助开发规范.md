# CodeGraph MCP 辅助开发规范

## 1. 目标

`Graft` 是一个同时包含 `server`、`web`、OpenAPI、脚本和治理文档的组合式后台平台。引入 CodeGraph MCP 的目标是让
AI 编程助手在探索大型代码结构时，可以先查询本地代码图，再进入真实源码精读，从而减少重复 `rg` / read 文件轮次。

CodeGraph MCP 是开发时导航知识源，不是 `server` 或 `web` 的运行时依赖。不要把 `@colbymchenry/codegraph` 加入
`web/package.json`、`web/bun.lock`、`server/go.mod`、仓库脚本、CI 或 hooks，也不要让业务代码依赖 CodeGraph。

## 2. 安装方式

当前项目的前端包管理器是 Bun。若使用包管理器运行 CodeGraph 安装器，优先使用 Bun 的一次性执行入口：

```bash
bunx @colbymchenry/codegraph
```

若只需要把 `codegraph` CLI 安装到本机 PATH，可使用 Bun 的全局安装入口：

```bash
bun add -g @colbymchenry/codegraph
```

也可以使用 CodeGraph 官方安装脚本。Linux / macOS 使用：

```bash
curl -fsSL https://raw.githubusercontent.com/colbymchenry/codegraph/main/install.sh | sh
```

安装后验证：

```bash
codegraph --version
```

不要在 `web` 目录执行 `bun add @colbymchenry/codegraph`。CodeGraph 是本机 AI 工具，不属于前端项目依赖。

## 3. Codex 接入方式

Codex 可以使用 MCP，但 MCP Server 必须先注册到 Codex 客户端配置中。当前项目优先把 CodeGraph MCP 配置给 Codex，
因为 Codex 是主要 AI 编码入口。

在本机执行：

```bash
codex mcp add codegraph -- codegraph serve --mcp
```

验证配置：

```bash
codex mcp list
codex mcp get codegraph
```

配置完成后，重启 Codex 会话，让新的 MCP Server 进入可用工具列表。这个配置属于开发者本机 Codex 用户级配置，不提交
到仓库。多个本机 `Graft` worktree 可以共享同一份 Codex MCP 配置。

## 4. 通用 MCP 客户端配置

在支持 MCP 的 AI 客户端中增加 CodeGraph MCP Server。不同客户端的根字段可能是 `mcpServers` 或 `servers`，按客户端
要求选择其中一种。

```json
{
  "mcpServers": {
    "codegraph": {
      "type": "stdio",
      "command": "codegraph",
      "args": ["serve", "--mcp"]
    }
  }
}
```

如果客户端使用 `servers` 字段：

```json
{
  "servers": {
    "codegraph": {
      "type": "stdio",
      "command": "codegraph",
      "args": ["serve", "--mcp"]
    }
  }
}
```

配置不提交到个人 IDE 私有目录。需要共享配置时，应优先把示例写入 `ai-plan/` 文档，避免绑定某一个编辑器。

## 5. 项目初始化

每个本地 worktree 需要在仓库根目录初始化一次 CodeGraph 索引：

```bash
codegraph init -i
```

初始化会生成 `.codegraph/` 索引目录。`.codegraph/` 是本地派生产物，必须被 Git 忽略，不得提交。

常用检查命令：

```bash
codegraph status
git check-ignore .codegraph/codegraph.db
```

如需清理本地项目索引：

```bash
codegraph uninit
```

如需移除本机工具配置，按 CodeGraph 当前 CLI 支持执行：

```bash
codegraph uninstall
```

## 6. 默认使用流程

AI 处理跨目录或大范围结构问题时，可以按下面顺序使用 CodeGraph：

1. 先完成根 `AGENTS.md` 要求的 startup preflight。
2. 判断问题是否适合代码图查询：
   - 适合：符号定位、调用链、依赖关系、影响面、模块入口、路由或 handler 发现。
   - 不适合：仓库治理规则、设计真值、提交范围归属、验证入口、最新外部文档。
3. 当 CodeGraph MCP 可用且当前 worktree 已初始化时，先用 CodeGraph 查询候选符号、调用者、被调用者或影响范围。
4. 对所有准备修改或引用为结论的代码，继续读取真实文件并按仓库权威文档确认。
5. 如果 CodeGraph MCP 不可用，退回 `rg`、文件读取和现有仓库探索流程；不要因为 MCP 不可用而阻塞任务。

CodeGraph 查询结果是探索线索，不是 canonical authority。模块契约、菜单、权限、路由、OpenAPI、前端 bootstrap 和文档治理
仍以本仓库 `AGENTS.md` 与 `ai-plan/design/` 规则为准。

## 7. 项目约束

- 不把 CodeGraph 加入 `server`、`web`、CI、hooks 或仓库运行脚本。
- 不要求所有贡献者安装 CodeGraph；没有 CodeGraph 时仍使用现有 `rg` 与源码读取流程。
- 不用 CodeGraph 查询结果替代真实源码、设计文档或验证命令。
- 不提交 `.codegraph/`、个人 MCP 客户端配置或 IDE 私有配置。
- 不把 CodeGraph MCP 作为任务完成态硬门禁；它只提升探索效率。
- 当任务触及 `web` 的 TDesign Vue Next 组件用法时，仍按 `TDesign-MCP-辅助开发规范.md` 查询 TDesign MCP；CodeGraph 不替代组件库文档。

## 8. 验证方式

CodeGraph 本机接入通过下面命令验证：

```bash
codegraph --version
codegraph init -i
codegraph status
git check-ignore .codegraph/codegraph.db
```

Codex MCP 接入通过下面命令验证：

```bash
codex mcp list
codex mcp get codegraph
```

实际修改 `server`、`web`、OpenAPI 或脚本时，仍按根 `AGENTS.md` 与子域 `AGENTS.md` 运行对应验证入口。

## 9. closeout 记录示例

如果本轮使用了 CodeGraph：

```text
CodeGraph MCP preflight: used
- codegraph_queried: yes
- purpose: symbol lookup / call graph / impact analysis
- adoption: adopted
- reason: used as navigation evidence, then verified touched files directly
```

如果本轮不适用：

```text
CodeGraph MCP preflight: not applicable
- codegraph_queried: no
- purpose: not-applicable
- adoption: not_adopted
- reason: no broad code navigation or graph query needed in this slice
```

如果 CodeGraph 不可用：

```text
CodeGraph MCP preflight: fallback to rg
- codegraph_queried: no
- purpose: symbol lookup
- adoption: not_adopted
- reason: CodeGraph unavailable in current Codex session; used rg and direct file reads
```

## 10. 参考来源

- CodeGraph 项目：`https://github.com/colbymchenry/codegraph`
- CodeGraph 安装文档：`https://colbymchenry.github.io/codegraph/getting-started/installation/`
- CodeGraph MCP Server 文档：`https://colbymchenry.github.io/codegraph/reference/mcp-server/`
- CodeGraph CLI 文档：`https://colbymchenry.github.io/codegraph/reference/cli/`
- CodeGraph NPM 包：`https://www.npmjs.com/package/@colbymchenry/codegraph`
- Codex MCP 命令：`codex mcp --help`
