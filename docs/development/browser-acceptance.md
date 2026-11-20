# 浏览器验收（Browser Acceptance）

`scripts/browser_acceptance.py` 是风险分层的真实 API 浏览器验收入口：构建 web 生产包，
为每个套件提供隔离的 PostgreSQL/Redis（`tests/browser/compose` 覆盖层），通过测试专用
Go fixture 写入确定性数据，再以 Playwright 驱动 `tests/browser/suites` 下针对真实后端
的验收套件。设计为生产优先：页面始终是 `bun run build` 产物，后端始终是真实
`graft serve` 实例。

## 边界

- 只在开发者主检出（primary checkout）运行。编号工作树禁止服务、浏览器 QA 和
  Playwright 执行（根 AGENTS.md 与 web/AGENTS.md 同一约束）；工作树内允许的验证是
  单元测试（`python3 -B -m unittest scripts.test_browser_acceptance`）、`--dry-run`
  计划输出、`--list-suites`/`--list-engines` 与 fixture 的 `go vet`/无数据库单测。
- 不进入本地检查门禁：`just check`、`just check-local*`、`just check-suite` 与 CI
  workflow 均不运行浏览器套件；浏览器验收是显式的开发者运行时验收，证据不替代
  `just check-server` / `just check-web`。
- 重试恒为零：一次尝试要么通过要么产出一份脱敏 JSON 报告，不掩盖不稳定行为。

## 前置条件

1. Docker 守护进程（隔离的 PostgreSQL 16 / Redis 7 容器）。
2. Go、Bun（构建生产包并运行迁移、fixture 与后端）。
3. Playwright 引擎：`cd tests/browser && bun install && bun run browsers:install`。

## 套件与引擎矩阵

`just browser-acceptance-list` 列出全部套件与引擎。

| 套件 | 默认 | 标签 | 说明 |
| --- | --- | --- | --- |
| `workbench` | 是 | `@chromium-only` | 工作台摘要与真实后端聚合 |
| `task` | 是 | `@chromium-only` | fixture 应用的任务历史、阶段与日志 |
| `runtime-target` | 是 | `@cross-browser` | 运行目标列表渲染与真实探测事实 |
| `docker` | 否 | `@chromium-only` | 需要 Docker 守护进程 |
| `compose` | 否 | `@chromium-only` | 需要 Docker 守护进程 |

- 引擎矩阵：`chromium` 是基线，运行全部套件；`firefox` / `webkit` 只运行带
  `@cross-browser` 标记的套件。标记由 spec 文件首行注释（`// @cross-browser` /
  `// @chromium-only`）与 `playwright.config.ts` 的 `grepInvert` 双重持有，
  `scripts/test_browser_acceptance.py` 校验注册表与文件标记一致。
- daemon-required 套件必须显式 `--suite <name> --allow-daemon`，永远不会出现在
  默认选择里。
- 显式为 `@chromium-only` 套件指定跨浏览器引擎是配置错误（退出码 2），不静默跳过。

## 运行与分层

```sh
just browser-acceptance                          # 默认：local 层级、chromium、全部默认套件
just browser-acceptance --suite task --engine chromium
just browser-acceptance --suite runtime-target --engine firefox,webkit
just browser-acceptance --tier ci                # 固定全新环境，禁止 keep/reuse 诊断
just browser-acceptance --suite docker --allow-daemon   # daemon-required 套件
python3 -B scripts/browser_acceptance.py --dry-run --suite task
```

- `--tier local`（默认）允许 `--keep` / `--reuse-environment` / `--teardown-only`
  诊断；`--tier ci` 是固定流程：全新环境、无诊断残留、报告归档到
  `.artifacts/browser-acceptance/ci`。
- `--keep`：尝试结束后保留 compose 环境、后端与静态代理进程并打印清理命令；
  `--reuse-environment`：附着到上次 `--keep` 保留的环境（跳过 provisioning、迁移、
  种子与构建；数据是上次运行的状态，不再重置）；`--teardown-only`：按状态文件清理
  保留的环境。
- `--skip-build`：复用现有 `web/dist`（前提是此前构建过生产包）。
- 环境生命周期：每次尝试从 `tests/browser/compose/docker-compose.browser.yml` 启动
  独立的 tmpfs PostgreSQL/Redis（动态宿主端口），随后 `graft migrate up`、确定性种子、
  `graft dev reset-admin`（恢复确定性登录态）、`bun run build`、`graft serve` 与
  `scripts/browser/serve_dist.py`（生产包静态服务 + 镜像生产代理面：`/api`、`/ws`、
  `/healthz`、`/docs`、`/openapi.*`、`/mcp/docs*`、`/sse`）。尝试结束即 `down -v`
  完全清理，除非 `--keep`。

## 确定性种子

种子是各模块的测试专用代码，遵循现有 `GRAFT_AUDIT_POSTGRES_TEST_URL` 的环境变量门控
模式，由 harness 以 `GRAFT_BROWSER_SEED_DATABASE_URL=<隔离库 DSN> go test -run
TestSeedBrowserFixture -count=1` 调用：

- `server/modules/project/browser_seed_test.go`：通过模块自有
  `ImportApplication` 写路径创建固定 `app_APPBR0WSERFXTR000000000000` fixture 应用
  （schema 正确性由模块权威代码负责）。
- `server/modules/task/browser_seed_test.go`：向该应用写入确定性的
  `application.compose.*` 终态任务与阶段行。
- `server/modules/runtime-target/browser_seed_test.go`：写入两个不与本地 Docker
  探测冲突的确定性 runtime target。

所有种子在缺少环境变量时 `t.Skip`，常规 `go test` 与 CI 不依赖真实数据库。若某个
fixture 需要改非测试代码才能成立，按治理停止该 fixture 并上报，不在 harness 内
复刻模块内部实现。

## 报告与脱敏

每次尝试产出 `.artifacts/browser-acceptance/<run-id>/<suite>-<engine>/attempt.json`
与 Playwright JSON：结果、退出码、引擎、层级、环境端口与失败列表。失败元数据经过
净化：仓库外绝对路径与用户 home 目录被压缩为相对形式，DSN 密码、`Bearer` 凭证与
`password/token/secret/credential/api_key=` 形式的取值一律替换为 `[redacted]`，
环境变量值不进入报告。`--dry-run` 输出完整计划（步骤命令、Playwright 调用与
`retries: 0`），不执行任何副作用。

## 主检出完整运行序列

```sh
just setup                                        # 一次性：root/web 依赖与 Go modules
cd tests/browser && bun install && bun run browsers:install   # 一次性：Playwright 引擎
just browser-acceptance --tier local              # 默认套件 x chromium
just browser-acceptance --suite runtime-target --engine firefox,webkit
just browser-acceptance --tier ci                 # CI 层级（全新环境、无诊断）
just browser-acceptance-test                      # harness 单元测试（任何检出可用）
```

运行后检查 `.artifacts/browser-acceptance/<run-id>/summary.json`；失败的尝试在对应
`suite-engine` 目录下保留 attempt.json、Playwright JSON、stderr 日志与 trace
（`retain-on-failure`）。跨浏览器结果只在对应引擎真实执行后记录；单次运行通过不是
全面验收，最终验收仍由仓库 CI 与人工判断承担。
