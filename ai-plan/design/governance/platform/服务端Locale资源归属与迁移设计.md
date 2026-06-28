# 服务端 Locale 资源归属与迁移设计

## 1. 目标

本设计定义 `server` 侧 locale resource 的长期 ownership 与迁移路径。

目标：

- 保持 `server/internal/i18n.Service` 为唯一 server i18n facade。
- 将 locale resource 的物理 ownership 按功能边界下沉到各自 owner package。
- 保持 YAML 解析、locale/key 校验、duplicate 检查、registry construction、`Lookup`、`Message`、`Freeze` 与 diagnostics 继续集中在 `server/internal/i18n`。
- 不让 `server/internal/i18n` 反向 import `server/modules/*`。
- 不让业务模块各自实现 YAML loader、registry、freeze 或平行 lookup 系统。
- 不改变 stable key、现有 HTTP wire shape、`LookupRequest`、`Freeze`、`RegisteredMessageResources` 与 `RegisteredMessageKeyIDs`。

非目标：

- 不引入 `go-i18n`。
- 不保留 `server/internal/i18n/locales/modules/*.yaml` 和 owner-local locale 的长期双来源兼容。
- 不通过 runtime filesystem scan 任意目录发现 locale。

## 2. 推荐架构

### 2.1 Ownership 边界

```text
server/internal/i18n/locales/
  core.zh-CN.yaml
  core.en-US.yaml
  display.zh-CN.yaml
  display.en-US.yaml

server/internal/moduleruntime/locales/
  zh-CN.yaml
  en-US.yaml

server/modules/<module>/locales/
  zh-CN.yaml
  en-US.yaml
```

规则：

- `server/internal/i18n/locales/*.yaml`
  - 只承载 `core`、`display` 等 i18n infrastructure 自有 namespace。
- `server/modules/<module>/locales/*.yaml`
  - 承载 module-owned namespace。
- `server/internal/moduleruntime/locales/*.yaml`
  - 承载 internal runtime/platform component 自有 namespace。

### 2.2 Runtime 注册方向

- owner package 自己 `go:embed locales/*.yaml`。
- owner package 只暴露只读 embedded resource descriptor，不暴露 loader。
- runtime 在 `Freeze` 前、模块 `Register` 前，统一将这些 raw embedded resources 交给 `server/internal/i18n` 注册。
- `registerMessages()` 保持为“当前模块依赖的 key 是否已存在”校验函数，不承担加载职责。

推荐 resource descriptor 形态：

```go
type EmbeddedLocaleResource struct {
	Namespace i18n.Namespace
	Locale    i18n.LocaleTag
	Source    string
	Data      []byte
}
```

推荐 i18n 内部入口：

```go
func (s *Service) RegisterEmbeddedLocaleResources(resources []EmbeddedLocaleResource) error
```

### 2.3 Compile-time descriptor 承载

- `moduleregistry` 当前已经收集全部 compile-time modules 的 `module.Spec`。
- 推荐在 compile-time descriptor 层承载 locale resource provider 或 locale resource list。
- runtime 通过 compile-time registry 收集所有 compiled modules 的 locale resources，再统一预注册。

默认决策：

- 未启用模块的 locale resources 仍注册。
- 原因是 compile-time module registry、module runtime snapshot 和若干菜单/权限/历史显示路径都可能依赖 disabled module 的 stable key 诊断，不适合把 locale 注册绑定到 allowlist。

## 3. 事实约束

### 3.1 当前 loader 与 Freeze 顺序

- 当前 `i18n.New()` 在 runtime 构造阶段完成。
- 当前模块生命周期顺序为 `Register -> Freeze -> Boot`。
- `Freeze` 发生在所有模块 `Register` 完成之后、所有模块 `Boot` 之前。

这意味着：

- runtime 具备新增“模块 `Register` 前统一预注册 locale resources”的天然插槽。
- 现有 `registerMessages()` 可以保持只读校验语义，不需要改成 loader。

### 3.2 Go embed 约束

- `server/internal/i18n` 不能通过一个 `go:embed` 直接跨目录抓取 `server/modules/*/locales/*.yaml`。
- 因此“i18n package 自己统一 embed 所有模块目录”不可行。
- 正确方向是 owner package 自 embed，runtime 统一注册。

## 4. 迁移表

| Current path | Target path | Owner |
| --- | --- | --- |
| `server/internal/i18n/locales/core.*.yaml` | same | `server/internal/i18n` |
| `server/internal/i18n/locales/display.*.yaml` | same | `server/internal/i18n` |
| `server/internal/i18n/locales/system-config.*.yaml` | `server/modules/system-config/locales/*.yaml` | `system-config` |
| `server/internal/i18n/locales/modules/announcement.*.yaml` | `server/modules/announcement/locales/*.yaml` | `announcement` |
| `server/internal/i18n/locales/modules/audit.*.yaml` | `server/modules/audit/locales/*.yaml` | `audit` |
| `server/internal/i18n/locales/modules/container.*.yaml` | `server/modules/container/locales/*.yaml` | `container` |
| `server/internal/i18n/locales/modules/monitor.*.yaml` | `server/modules/monitor/locales/*.yaml` | `monitor` |
| `server/internal/i18n/locales/modules/rbac.*.yaml` | `server/modules/rbac/locales/*.yaml` | `rbac` |
| `server/internal/i18n/locales/modules/scheduler.*.yaml` | `server/modules/scheduler/locales/*.yaml` | `scheduler` |
| `server/internal/i18n/locales/modules/user.*.yaml` | `server/modules/user/locales/*.yaml` | `user` |
| `server/internal/i18n/locales/modules/module-runtime.*.yaml` | `server/internal/moduleruntime/locales/*.yaml` | `module-runtime` |

## 5. 最小实现切片

### Slice 1：补 i18n raw embedded registration 入口

- 只增加 `server/internal/i18n` 对 raw embedded resources 的统一注册入口。
- 不移动现有文件。
- 补 runtime 预注册插槽与测试。

### Slice 2：迁移一个低风险模块试点

- 首选 `announcement`，次选 `container`。
- owner package 自 embed `locales/*.yaml`。
- runtime 统一注册。
- 删除该模块原集中目录 locale 文件。

### Slice 3：迁移剩余业务模块

- `audit`
- `container` / `announcement` 中尚未迁移的那个
- `monitor`
- `rbac`
- `scheduler`
- `system-config`
- `user`

### Slice 4：迁移 `module-runtime`

- 将 `module-runtime` locale 迁到 `server/internal/moduleruntime/locales/*.yaml`。
- 保持 `core` / `display` 继续留在 `server/internal/i18n/locales/*.yaml`。

### Slice 5：治理收尾与回归阻断

- 更新 `ai-plan/design/governance/platform/本地化与i18n治理规范.md`
- 更新 `.agents/skills/graft-localization-governance/SKILL.md`
- 更新 public recovery topic
- 增加脚本或 CI 规则阻止新增 `server/internal/i18n/locales/modules/*.yaml`

## 6. 验证

```bash
git diff --check
cd server && go test ./internal/i18n/...
cd server && go test ./internal/app/... ./internal/moduleruntime/...
cd server && go test ./modules/announcement/... ./modules/container/... ./modules/audit/... ./modules/monitor/... ./modules/rbac/... ./modules/scheduler/... ./modules/system-config/... ./modules/user/...
cd server && go run ./cmd/graft validate backend --stage lint
cd server && go build ./cmd/graft
python3 /root/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/graft-localization-governance
```
