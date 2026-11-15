# TDesign MCP 辅助开发规范

## 1. 目标

`web` 使用 `TDesign Vue Next` 作为主 UI 框架。引入 TDesign MCP 的目标是让 AI 在生成、修改和评审前端代码时，
能直接查询 TDesign 的组件列表、组件文档、DOM 结构和变更日志，减少组件 API 误用、样式选择器猜测和升级风险。

TDesign MCP 是开发时知识源，不是 `web` 的运行时依赖。不要把 `tdesign-mcp-server` 加入 `web/package.json`、
仓库脚本、CI、hooks，也不要让业务代码依赖 MCP Server。

`web` 的 TDesign Vue Next 运行时组件引入由 Vite resolver 负责按需解析。应用入口不得使用
`app.use(TDesign)` 全量注册，也不得引入 `tdesign-vue-next/es/style/index.css` 全量样式；程序式插件和渲染函数组件
只在使用点显式 import，并在 resolver 未覆盖时补对应组件级 style import。

## 2. Codex 接入方式

Codex 可以使用 MCP，但 MCP Server 必须先注册到 Codex 客户端配置中。当前项目优先把 TDesign MCP 配置给 Codex，
因为 Codex 是主要 AI 编码入口，Rider 主要用于打开、运行和检查项目。

在本机执行：

```bash
codex mcp add tdesign -- npx -y tdesign-mcp-server@latest
```

验证配置：

```bash
codex mcp list
codex mcp get tdesign
```

配置完成后，重启 Codex 会话，让新的 MCP Server 进入可用工具列表。之后处理新增或不确定的 TDesign API/DOM/版本行为时，
按第 4 节查询 `vue-next` 资料；已验证用法可以复用证据。

这个配置属于开发者本机 Codex 用户级配置，不提交到仓库。多个本机 `Graft` worktree 可以共享同一份 Codex MCP 配置，
仓库只记录规范和推荐命令，不记录个人私有安装状态。

## 3. 通用 MCP 客户端配置

在支持 MCP 的 AI 客户端中增加 TDesign MCP Server。不同客户端的根字段可能是 `mcpServers` 或 `servers`，
按客户端要求选择其中一种。

```json
{
  "mcpServers": {
    "tdesign-mcp-server": {
      "command": "npx",
      "args": ["-y", "tdesign-mcp-server@latest"]
    }
  }
}
```

如果客户端使用 `servers` 字段：

```json
{
  "servers": {
    "tdesign-mcp-server": {
      "command": "npx",
      "args": ["-y", "tdesign-mcp-server@latest"]
    }
  }
}
```

配置不提交到个人 IDE 私有目录。需要共享配置时，应优先把示例写入 `ai-plan/` 文档，避免绑定某一个编辑器。

### 3.1 Rider 接入方式

Rider 在当前工作流中主要用于打开、运行和诊断项目，不是必须的 TDesign MCP 入口。只有在使用 Rider AI Assistant
生成或修改 `web` 代码时，才需要把 Rider AI Assistant 作为 MCP Client 连接 TDesign MCP Server：

1. 打开 `Settings | Tools | AI Assistant | Model Context Protocol (MCP)`。
2. 点击 `Add`，选择 `STDIO` 类型。
3. 填入上面的 `mcpServers` JSON 配置。
4. `Working directory` 选择仓库根目录。
5. `Server level` 选择当前项目，而不是全局，避免把 Graft 的 TDesign 约束污染到其他项目。
6. 点击 `Apply`，确认状态列显示连接成功。
7. 打开可用工具列表，确认至少能看到 `get-component-list`、`get-component-docs`、`get-component-dom`。

如果连接失败，先在终端手动运行下面命令确认本机 `node` / `npm` / 网络链路是否可用：

```bash
npx -y tdesign-mcp-server@latest
```

Rider 也内置了 MCP Server，可以让外部 AI 客户端访问 IDE 工具。那是另一个方向，不是 TDesign 接入的必需步骤。
本项目需要的是让 Rider AI Assistant 连接 TDesign MCP Server。

## 4. 按风险查询与证据复用

`web/AGENTS.md` 决定前端规则，根 `AGENTS.md` 决定启动与授权。本节是组件知识查询指导，不是每次修改都执行的
固定调用序列。新页面/重构/复杂布局先读 DESIGN 和相关视觉规则；简单文案、主题 token 或已验证局部修复不全读。

| 实际未知或风险 | 查询 |
| --- | --- |
| 不确定组件名称/存在性/框架 | get_component_list，指定 vue-next |
| 新增或不确定 props/events/slots/用法 | get_component_docs |
| 实际依赖组件内部 DOM/选择器/插槽布局 | get_component_dom |
| 升级、受版本影响行为或兼容疑问 | get_component_changelog |

只查询本轮受影响组件；不为文案修改重新查询完整组件列表。当前任务已有有效证据且版本/用法没有变化时直接复用，
不得凭经验猜未验证的 API 或 DOM。优先 TDesign MCP；不可用时采用官方文档并记录原因/受影响组件。MCP 不成为
CI、hook、package 或运行时依赖，查询成功本身也不等于验证通过。

closeout 用简短证据说明 queried/reused/not-needed/fallback、组件、查询种类及采用判断；可引用已有证据，不要求
重复空模板。仅复用时写清对应证据与版本未变，MCP 不可用时说明 official docs fallback。没有组件风险时无需查询。

## 5. 推荐场景

TDesign MCP 应作为这些场景的默认资料来源：

- 标准后台 CRUD 页面：表格、搜索表单、分页、弹窗、抽屉、批量操作。
- 后台壳和导航：布局、菜单、面包屑、头像、标签、按钮、图标。
- 表单密集页面：输入框、选择器、日期选择器、上传、表单校验和提交反馈。
- 样式调整：覆盖 TDesign 默认样式前，先确认 DOM 结构和可用 class。
- 组件库升级：升级前检查表格、表单、弹窗、菜单、图标和全局配置相关 changelog。
- 代码评审：反查组件 API，识别过期属性、错误事件名、错误插槽名和不符合 TDesign Vue Next 的写法。
- 模块脚手架：新增 `web/src/modules/<name>` 时，先用 MCP 确认页面所需组件组合。

## 6. 项目约束

MCP 查询结果必须服从本仓库的前端架构规则：

- 新模块仍按 `menu + route + page + api + permission` 接入。
- 壳层页面放在 `web/src/app/**`，业务页面放在 `web/src/modules/<name>/pages/**`。
- 共享状态仍放入 Pinia store，页面局部状态保留在页面或模块内。
- `UnoCSS` 只做辅助布局和少量原子样式，不用来重写整套 TDesign 视觉体系。
- 不因 MCP 示例引入额外 UI 库、React 写法、Vue 2 写法或移动端组件写法。
- 不把 `tdesign-mcp-server` 写入 `web/package.json`、仓库脚本、CI、hooks 或任何运行时依赖。

外部 UI 文章、repo-local plugin skill 中引用的外部 UI 示例 / 设计资产，或第三方示例进入前端实现前，必须先经过 TDesign / Graft 翻译：

- 示例结构先映射到 `DESIGN.md` 和 `前端视觉设计规范.md` 的页面类型，而不是默认生成营销 landing page。
- 组件实现必须落到 Vue SFC + TypeScript 和 `TDesign Vue Next` 组件；不得复制 React / shadcn / Tailwind 运行写法。
- 只允许参考交互意图、信息层级或动效节奏；不得复制 raw DOM widget、全局样式重置或新增 package。
- 样式覆盖必须以 TDesign MCP DOM / docs 为依据，并保持 token、theme、i18n 和可访问 label 边界。
- 验证继续回到 web 完成入口；仅获授权且需要浏览器时补对应证据。不得把 MCP 查询、外部示例截图或隐藏脚本当作第二套 validation truth。

## 7. 验证方式

Codex MCP 接入通过下面命令验证：

```bash
codex mcp list
codex mcp get tdesign
```

通用 MCP 客户端接入通过客户端验证：

- 客户端能启动 `tdesign-mcp-server`。
- `get-component-list` 能返回 `vue-next` 组件列表。
- `get-component-docs` 能查询 `Button`、`Table`、`Form`、`Dialog`、`Menu`。
- `get-component-dom` 能返回至少一个常用组件的 DOM 结构。

实际修改 web 时，中间迭代使用最小直接检查，完成入口仍是 web/AGENTS.md 的 `bun run check`，可复用未变输入的有效证据。

### 7.1 closeout 记录示例

```text
TDesign MCP preflight: used
- ui_component_change: yes
- mcp_queried: yes
- framework: vue-next
- components: Dialog, Button, Form
- queries: get_component_list, get_component_docs, get_component_dom
- adoption: adopted
- reason: adopted queried API and DOM guidance directly
```

如果本轮不涉及 TDesign 组件：

```text
TDesign MCP preflight: not applicable
- ui_component_change: no
- mcp_queried: no
- framework: not-applicable
- components: none
- queries: none
- adoption: not_adopted
- reason: no TDesign Vue Next component usage changed in this slice
```

如果 MCP 不可用：

```text
TDesign MCP preflight: fallback to official docs
- ui_component_change: yes
- mcp_queried: fallback_to_official_docs
- framework: vue-next
- components: Table
- queries: get_component_docs
- adoption: partially_adopted
- reason: MCP unavailable in current Codex session; used official docs for the affected component
```

## 8. 参考来源

- TDesign Vue Next MCP 页面：`https://tdesign.tencent.com/vue-next/mcp`
- TDesign MCP Server 仓库说明：`https://cnb.cool/tencent/tdesign/tdesign-mcp-server`
- 腾讯云 MCP Server 页面：`https://cloud.tencent.com/developer/mcp/server/11721`
- Codex MCP 命令：`codex mcp --help`
- JetBrains AI Assistant MCP 文档：`https://www.jetbrains.com/help/ai-assistant/mcp.html`
