# Graft

Graft 是一个基于 Go 和 Vue 3 的组合式后台平台，目标是通过插件机制快速接入新功能，而不是把所有业务硬编码进一个固定后台。

当前仓库优先完善设计与实施文档，核心决策已经收敛为：

* 后端：`Go + Gin + Ent + PostgreSQL`
* 前端：`Vue 3 + TypeScript + Vite`
* UI：`TDesign Vue Next`
* 架构：插件化平台
* 依赖管理：轻量 DI / 服务注册，不引入重量级 IoC

## 文档

* [项目设计](ai-plan/design/项目设计.md)
* [插件与依赖注入设计](ai-plan/design/插件与依赖注入设计.md)
* [前端架构设计](ai-plan/design/前端架构设计.md)
* [MVP 实施计划](ai-plan/roadmap/MVP实施计划.md)
* [AI 任务追踪与恢复设计](ai-plan/design/AI任务追踪与恢复设计.md)
* [AI Plan 恢复索引](ai-plan/public/README.md)
* [AI 环境清单说明](.ai/environment/README.md)

## 当前状态

项目目前仍处于架构与实施设计阶段。开始编码前，先以 `ai-plan/design/` 与 `ai-plan/roadmap/` 下文档固化边界与约束；复杂长期任务的恢复入口位于 `ai-plan/public/` 下主题跟踪和轨迹文件。

仓库同时维护 `.ai/environment/` 作为环境真值入口：

* `tools.raw.yaml` 记录当前机器与仓库相关的原始环境事实
* `tools.ai.yaml` 记录给 AI 和贡献者消费的精简环境摘要

## 本地启动 `server`

当前 `server` 使用 `.env` 作为运行时主配置源。推荐把 GoLand 或其他 IDE 的 working directory 设为 `server`；如果从仓库根启动，程序会回退读取 `server/.env`。

最小启动步骤：

1. 复制 `server/.env.example` 为 `server/.env`
2. 修改 `server/.env` 中的 auth 密钥
3. 进入 `server` 目录后执行 `go run ./cmd/graft dev`

如果缺少 auth 密钥，启动会直接报错：`GRAFT_AUTH_JWT_SECRET or GRAFT_AUTH_SIGNING_KEY is required`。

如果你需要生成新的本地 auth 密钥，可以在 `server` 目录下运行：

```bash
go run ./cmd/graft-jwt-secret
go run ./cmd/graft-signing-key
```

两个程序都会输出一行可直接粘贴到 `server/.env` 的配置文本。

推荐的本地开发入口已经统一为一个 Go CLI 命令：

```bash
cd server
go run ./cmd/graft dev
```

`graft dev` 会先执行显式迁移，再在迁移成功后启动服务；它是开发期编排命令，不会改变 `graft serve` 的纯运行时语义。

如果你需要在本地持续开发时自动 rebuild 并重启 `graft serve`，仓库现在提供固定的 Air 配置：

```bash
cd server
go get -tool github.com/air-verse/air@v1.65.2
go run ./cmd/graft dev air
```

约束：

* Air 只负责监听本地 Go 代码与 `.env` 变化后重新 build，并重启 `graft serve`。
* Air 不执行 `graft dev`，也不执行 `graft migrate up`。
* `server` 仍然按当前规则自行读取 `.env` / `server/.env`；Air 不通过 `env_files` 接管配置加载语义。
* `graft dev air` 只是 `go tool air -c .air.toml` 的 CLI 包装，不改变 Air 配置本身。

推荐流程：

```bash
cd server
go run ./cmd/graft migrate up
go run ./cmd/graft dev air
```

其中：

* 首次本地启动，或数据库 schema / migration 发生变化时，先手动执行 `graft migrate up`。
* 仅做日常 Go 代码调试时，直接运行 `go run ./cmd/graft dev air` 即可。
* 如果你只想一次性“迁移后启动”，仍然继续使用 `go run ./cmd/graft dev`。

如果你需要反复验证默认管理员首次登录强制改密流程，可以使用 dev-only 重置入口：

```bash
cd server
go run ./cmd/graft dev reset-admin
```

该命令只允许在 `GRAFT_APP_ENV=local` 或 `test` 下使用，并会执行三件事：

1. 确保默认管理员 `graft` 存在
2. 把密码重置为 `graft-admin`
3. 把 `must_change_password` 重新置为 `true`

典型调试流程：

```bash
cd server
go run ./cmd/graft dev reset-admin
```

然后在浏览器里清空 `localStorage` / `sessionStorage`，重新用 `graft / graft-admin` 登录，即可再次验证受限态和强制改密弹窗。

Windows PowerShell / CMD 可以直接使用同一条命令：

```powershell
cd server
go run ./cmd/graft dev
```

如果你已经先编译过 CLI，也可以直接运行：

```powershell
cd server
.\graft.exe dev
```

如果你需要把迁移和启动拆开，仍然可以继续使用显式两步命令：

```bash
cd server
go run ./cmd/graft migrate up
go run ./cmd/graft serve
```

注意：

* 根命令 `graft` 只显示帮助，不会启动服务。
* `graft dev` 与 `graft migrate up` 都依赖本机已安装 `atlas` CLI。
* 当你新增、重命名或调整 `server/internal/ent/migrate/migrations/` 下的 migration 文件后，必须先在 `server` 目录执行 `atlas migrate hash --dir file://internal/ent/migrate/migrations`，否则 `graft dev` / `graft migrate up` 会因为 `atlas.sum` 校验不匹配而失败。
* `graft serve` 启动前会连接 PostgreSQL 和 Redis；若地址不可达，启动会直接失败。
* 若本地库结构已经同步，也可以只运行 `graft serve`；否则请先执行迁移。
* 在 GoLand 或其他 IDE 中，推荐统一使用 working directory=`server`、程序入口 `./cmd/graft`、程序参数 `dev`。

## 后端验证 `server`

后端完成态统一通过一个 Go CLI 入口执行：

```bash
cd server
go run ./cmd/graft validate backend
```

仓库固定使用 `golangci-lint v2.12.2` 作为后端统一 lint 运行器，并要求 agent、本地开发与 CI 复用同一入口，
而不是各自维护第二套参数。

后端完成态质量链顺序固定为：

1. `graft validate backend --stage lint`
2. `go test` 最小直接覆盖范围
3. `go build ./cmd/graft`
4. 需要启动链路时再补 `graft validate smoke`

其中 backend blocking lint gate is changed-file scoped against the resolved base branch。
changed-file scoped means whole-file enforcement on files changed relative to the resolved base branch via
`--new-from-rev=<merge-base> --whole-files`。untouched files are not blocking gate failures。full-repo lint is
audit-only backlog scanning。touching a historical hotspot file means the touched file must satisfy the current lint
gate, unless a narrowly documented temporary `nolint` is justified。new code must not expand the lint backlog。

## 本地启动 `web`

前端开发环境配置不再直接提交真实 `web/.env.development`，而是提交模板文件 `web/.env.example`，本地实际配置保持忽略状态。

最小启动步骤：

1. 复制 `web/.env.example` 为 `web/.env.development`
2. 按本地后端地址调整 `VITE_API_TARGET`
3. 进入 `web` 目录后执行 `bun run dev`

当前默认开发链路为：

* 浏览器访问 `http://localhost:3002/api/...`
* Vite dev proxy 再把请求转发到 `VITE_API_TARGET`

说明：

* `web/.env.development`、`web/.env.local` 与其它 `web/.env.*` 本地文件都应保持未跟踪状态。
* `web/.env.example` 只作为共享模板，不应放入个人密钥或机器专属地址。

## 前端验证 `web`

前端完成态统一通过一个脚本入口执行：

```bash
cd web
bun run check
```

说明：

* `bun run check` 会按 `format:check -> typecheck -> lint -> stylelint -> test:run -> build` 顺序执行，是本地交付、
  交接和合并前的前端真值入口。
* 开发中间态可以执行更小的直接命令，但 README、skill 和 CI 只能复用这条入口或其显式执行切片，不应再定义第二套
  完成态规则。
* 本地 contract governance changed-scan 默认在 `pre-commit` 阶段阻断；`pre-push` 不再重复执行该扫描，推送后的正式阻断由 CI 承担。
