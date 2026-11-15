# 开发验证阶段

根 AGENTS.md 持有验证阶段，server/web 领域命令持有检查实现。

## 日常迭代与提交准备

迭代运行必要编译和能够诊断当前问题的最小检查。提交准备审阅完整任务 diff 与消费者，选择有意义的回归测试。

`just check-local-plan` 显示当前 worktree 变更的检查计划。`just check-local` 执行同一计划。
可以重复传入 `--file <仓库相对路径>` 与 `--test <消费者测试路径>`。
显式消费者必须存在且属于对应语言测试族；删除整个 Go 包或最后一份测试时选择现存消费者，未覆盖删除项会报告缺口。
默认选择 HEAD 与 worktree 的差异及未跟踪文件，删除项保留在影响分析中。
检查目标是当前 worktree；所选文件存在 staged/worktree 内容分歧时执行入口返回缺口，提交前须使验证内容与暂存内容一致。

Go 按直接及显式消费者包测试，Web 按文件格式、lint、样式和选定测试检查。OpenAPI 变更另执行
`just openapi-check` 并选择两端消费者测试。治理变更执行对应结构校验。
空范围、未知影响、缺少测试和缺少工具均明确报告，执行过程收集独立失败，缺口返回非零状态。
依赖使用 `just setup` 显式准备，定向检查仅消费已安装工具。

`just check-changed` 是 `check-local` 的兼容入口。完整领域检查由 `just check-server`、
`just check-web` 和 `just check` 显式调用。检查成功、完成提交准备、通过 CI 和完成运行验收分别记录。

## Bun workspace 与 Turbo

根 `package.json`、`bun.lock` 统一管理 JavaScript 依赖，`bun install --frozen-lockfile` 在根目录执行。
server 的 `package.json` 仅暴露 Go 命令，Go module 持有 Go 依赖。
Just 包装开发、生成和运维命令，`bun run build`、`bun run test`、`bun run check:code` 通过
`scripts/turbo.mjs` 编排代码任务，领域命令可独立使用。

只有前端生产构建使用当前检出的 .turbo/cache 本地结果缓存。缓存输入包括源码、生成客户端、配置、OpenAPI 源、锁文件、
前端环境文件、构建变量以及 OS/架构/Bun/Node 身份。构建命中恢复完整 dist。
测试、检查、生成新鲜度与迁移历史每次执行。共享包或构建输入改变时同步修改任务输入并验证失效。
`--force` 强制执行并更新本地缓存。数据库、Docker、浏览器和发布验收分别提供真实环境证据。

## CI 与独立工具

CI 分别执行治理、契约、迁移、server 和 web 检查，并在汇总 job 收集独立结果。
后端 buildtest 的 `--test-report <path>` 将同一次 Go 测试流交给报告生成器，再继续构建。
测试失败和报告生成失败分别传播到门禁。报告呈现消费已有结果。

项目命令、CI 和领域规范可独立于个人 harness 使用。个人启动、任务池和交付编排可以通过外部 adapter
映射已有命令，仓库技能引用产品权威，避免重复定义阶段与权限。


## 首轮测量记录

2026-11-15 在原生产源码 346854ff5daae3f804f8fad706202c9e983d2067 的任务检出上，
使用 Linux x64、Bun 1.3.14、Node 24.19.0 与原前端锁定依赖进行测量。
直接前端构建为 81.28 秒，首次 Turbo 构建为 83.39 秒，输入稳定后的缓存命中为 0.16 秒。
命中恢复 dist 的行为另由临时仓库实测。该结果衡量本机重复构建，完整 CI 耗时需要独立比较。

旧池中的 node_modules 与锁文件版本存在分歧，先通过任务目录中的干净冻结安装修复依赖条件。
workspace 合并时，相同包版本节点的原传递依赖发生去重，fast-uri 从原根工具链的 3.1.2 统一到前端已有
3.1.3，brace-expansion 从原根工具链的 5.0.6 统一到前端已有 5.0.7，其余既有版本集合保留。
OpenAPI bundler 使用的 @redocly/openapi-core 显式声明为既有锁定版本 1.34.17。

## 本机运行时 CI

独立的 Local runtime acceptance job 消费 server-build-test 的已验证 Linux/amd64 二进制及 web-check 的生产 dist。
产物携带实际 checkout commit 与文件 SHA-256，消费端先核对来源和字节，再构建带同一 revision 标签的运行时镜像。
后端与 Web 测试各执行一次，运行验收复用其构建产物。

验收在任务私有 Docker daemon 中执行核心 API 与实际引擎核对，再用冻结的官方根 Compose 从空目录安装并执行同组操作。
两组证据都必须覆盖 canonical 必需场景集合且状态为 passed，未验证和缺失场景使 job 失败。
Required checks passed 将该 job 纳入结果汇总，跳过结果保持失败。

收尾始终按已注册 project 身份销毁任务资源。上传范围采用精确文件白名单，包含 API 场景、
源码指纹、官方安装、清理与构建来源证明。instance.json、环境文件和原始错误日志保留在 runner 的本地目录。
镜像构建与平台自更新的完整运行场景由各自验收驱动提供证据，核心操作 CI 与它们分别记录。
