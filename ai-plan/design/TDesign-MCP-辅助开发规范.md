# TDesign MCP 辅助开发规范

## 1. 目标

`web` 使用 `TDesign Vue Next` 作为主 UI 框架。引入 TDesign MCP 的目标是让 AI 在生成、修改和评审前端代码时，
能直接查询 TDesign 的组件列表、组件文档、DOM 结构和变更日志，减少组件 API 误用、样式选择器猜测和升级风险。

TDesign MCP 是开发时知识源，不是 `web` 的运行时依赖。不要把 `tdesign-mcp-server` 加入 `web/package.json`，
也不要让业务代码依赖 MCP Server。

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

配置完成后，重启 Codex 会话，让新的 MCP Server 进入可用工具列表。之后处理 `web` 的 TDesign 代码时，
应先通过 TDesign MCP 查询 `vue-next` 组件资料，再生成或修改代码。

这个配置属于开发者本机 Codex 配置，不提交到仓库。仓库只记录规范和推荐命令。

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

## 4. 默认使用流程

AI 处理 `web` 的 TDesign 页面、模块或组件时，应按下面顺序使用 MCP：

1. 用 `get-component-list` 查找可用组件，并显式选择 `vue-next` 框架。
2. 用 `get-component-docs` 查询目标组件的 props、events、slots、示例和使用约束。
3. 涉及自定义样式、选择器、布局覆盖时，用 `get-component-dom` 查询组件 DOM 结构。
4. 升级 `tdesign-vue-next` 或修改受版本影响的组件时，用 `get-component-changelog` 查询变更日志。

向 AI 发起任务时，应明确说明本项目使用 `TDesign Vue Next` / `vue-next`，避免生成 React、Vue 2 或移动端组件写法。

## 5. 推荐场景

TDesign MCP 应作为这些场景的默认资料来源：

* 标准后台 CRUD 页面：表格、搜索表单、分页、弹窗、抽屉、批量操作。
* 后台壳和导航：布局、菜单、面包屑、头像、标签、按钮、图标。
* 表单密集页面：输入框、选择器、日期选择器、上传、表单校验和提交反馈。
* 样式调整：覆盖 TDesign 默认样式前，先确认 DOM 结构和可用 class。
* 组件库升级：升级前检查表格、表单、弹窗、菜单、图标和全局配置相关 changelog。
* 代码评审：反查组件 API，识别过期属性、错误事件名、错误插槽名和不符合 TDesign Vue Next 的写法。
* 模块脚手架：新增 `web/src/modules/<name>` 时，先用 MCP 确认页面所需组件组合。

## 6. 项目约束

MCP 查询结果必须服从本仓库的前端架构规则：

* 新模块仍按 `menu + route + page + api + permission` 接入。
* 页面仍放在 `web/src/pages` 或 `web/src/modules/<name>`。
* 共享状态仍放入 Pinia store，页面局部状态保留在页面或模块内。
* `UnoCSS` 只做辅助布局和少量原子样式，不用来重写整套 TDesign 视觉体系。
* 不因 MCP 示例引入额外 UI 库、React 写法、Vue 2 写法或移动端组件写法。

## 7. 验证方式

Codex MCP 接入通过下面命令验证：

```bash
codex mcp list
codex mcp get tdesign
```

通用 MCP 客户端接入通过客户端验证：

* 客户端能启动 `tdesign-mcp-server`。
* `get-component-list` 能返回 `vue-next` 组件列表。
* `get-component-docs` 能查询 `Button`、`Table`、`Form`、`Dialog`、`Menu`。
* `get-component-dom` 能返回至少一个常用组件的 DOM 结构。

实际修改 `web` 代码时，仍按仓库规则运行前端验证命令，例如 `bun run typecheck` 或 `bun run build`。

## 8. 参考来源

* TDesign Vue Next MCP 页面：`https://tdesign.tencent.com/vue-next/mcp`
* TDesign MCP Server 仓库说明：`https://cnb.cool/tencent/tdesign/tdesign-mcp-server`
* 腾讯云 MCP Server 页面：`https://cloud.tencent.com/developer/mcp/server/11721`
* Codex MCP 命令：`codex mcp --help`
* JetBrains AI Assistant MCP 文档：`https://www.jetbrains.com/help/ai-assistant/mcp.html`
