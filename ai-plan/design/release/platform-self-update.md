# Platform Self Update Design

本文件是 Graft 平台自更新的仓库级设计真相。它把 Graft 定义为管理自身生命周期的 self-hosted platform，而不是在运行中替换容器内二进制的安装器。

## Product Boundary

- 入口为 `Platform -> System Maintenance -> Updates`；左侧 Graft 标识下的当前版本是进入此页的快捷入口。
- 更新是管理员确认后的受治理操作：自动检查可以启用，自动安装不在当前承诺范围内。
- `server`、`web`、数据库迁移和配置快照必须对应同一目标 release；不得混用版本。官方 Compose 用共享 `GRAFT_IMAGE_TAG` 同时声明 server/web 镜像选择和更新策略：`latest` 跟随 stable、`beta` 跟随 Beta、SemVer tag 固定到一个 stable 或 Beta release。runner 从已验证 manifest 解析明确目标并验证 digest；解析结果是运行时状态，不能改写跟随 tag 或单独成为升级事实。
- `server/modules/update` 和 `server/modules/backup` 是两个独立模块。Update 消费 Backup capability；Atlas migration 仍由 core CLI 拥有，不创建 migration 业务模块。
- Backup 是可审计资产，Task 是其生成过程。Backup Detail 负责说明资产覆盖范围、工件大小、完整性摘要、保留状态和恢复证据；关联 Task 只负责展示阶段与执行日志。安全读取面可以公开配置快照和 PostgreSQL 转储的大小、SHA-256 与恢复证据，但绝不公开存储位置、配置或转储内容、执行命令和密钥。
- `AVAILABLE` 仅表示备份工件已记录且仍在保留期，不表示恢复已验证或承诺可自动回滚。恢复验证必须有受控流程写入的证据；数据库 migration 仍遵循 forward-only 策略。

## Release Authority And Manifest

GitHub Release 是 release catalog 和 release notes 的权威来源；GHCR digest 是 Compose 运行时镜像身份。`.github/workflows/publish.yml` 先完成 release artifact、binary smoke 和本地加载镜像的 Compose smoke，再发布同一 release tag 的 server/web/runner 镜像；所有正式镜像 tag 推送成功后才生成并附带 `release-manifest.json`：

```json
{
  "schema_version": 1,
  "version": "0.10.0-beta.1",
  "channel": "beta",
  "release_tag": "v0.10.0-beta.1",
  "release_notes_url": "https://github.com/<owner>/Graft/releases/tag/v0.10.0-beta.1",
  "minimum_source_version": "0.9.0",
  "upgrade_notes": "Read release notes and run the declared migration before runtime startup.",
  "images": {
    "server": { "image": "ghcr.io/<owner>/graft-server", "digest": "sha256:...", "reference": "ghcr.io/<owner>/graft-server@sha256:..." },
    "web": { "image": "ghcr.io/<owner>/graft-web", "digest": "sha256:...", "reference": "ghcr.io/<owner>/graft-web@sha256:..." }
  },
  "runners": {
    "compose": { "image": "ghcr.io/<owner>/graft-compose-runner", "digest": "sha256:...", "reference": "ghcr.io/<owner>/graft-compose-runner@sha256:..." }
  },
  "artifacts": {
    "server": "graft-server-linux-amd64-v0.10.0-beta.1.tar.gz",
    "web": "graft-web-dist-v0.10.0-beta.1.tar.gz",
    "checksums": "graft-sha256sums-v0.10.0-beta.1.txt",
    "sha256": { "graft-server-linux-amd64-v0.10.0-beta.1.tar.gz": "...", "graft-web-dist-v0.10.0-beta.1.tar.gz": "..." }
  },
  "migration": { "required_before_runtime": true, "command": "graft migrate up", "mode": "forward-only" }
}
```

`release-manifest.json.sha256` 与 manifest 同时作为 GitHub Release asset 发布；读取端先验证 manifest checksum，后验证 JSON 内容。Compose runner 从 `server/runner/compose/Dockerfile` 由同一 release workflow 构建和推送；manifest 只接受当前 Buildx 输出的 immutable digest，绝不接受仓库变量、外部断言 digest 或 mutable tag。`required_before_runtime` 表示每个受支持升级都必须执行显式且幂等的 migration command；它不声称每个 release 都包含 SQL 变更。目标 manifest 的 tag、SemVer channel、Release Notes URL、最低来源版本、升级说明、artifact 名称与逐项 SHA256、server/web/runner digest/reference 必须交叉校验，任何不一致、缺失 runner 或 mutable runner identity 都使目标不可执行。

正式 GitHub Release 创建前的任一构建、smoke 或镜像发布失败都会进入失败清理路径。清理只在同 tag 的 GitHub Release 不存在、且远端 tag 仍指向本次 workflow 的触发 commit 时删除该 tag；GitHub Release 已创建后不自动删除 tag，也不承诺 GHCR 多镜像推送、运行时镜像或数据库 schema 的事务性回滚。

## Version And Channel Selection

- 版本遵循 `MAJOR.MINOR.PATCH` 和 `MAJOR.MINOR.PATCH-beta.N`。
- `latest`（stable tracking）只选择比当前版本新的 stable release，不将 Beta 作为候选。
- `beta`（Beta tracking）只选择比当前版本新的 Beta release，不将 stable 作为候选。
- 固定 stable tag 只可选择比当前版本严格更高的 stable release；固定 Beta tag 只可选择比当前版本严格更高的 Beta release。固定版本选择界面不得出现同版本、较低版本或跨频道版本。
- 频道切换是独立的受治理产品操作，不是普通升级；当前切片不提供或隐式执行频道切换。
- catalog 条目必须通过 manifest 校验后才可展示为可升级；release notes 只作为展示内容，不能替代 manifest。
- Update 持久化最近成功验证的 catalog projection，缓存窗口为 24 小时；网络失败必须显示检查失败和缓存陈旧，不能把缓存解释为“已是最新”。

## Compose Image Tag Strategy

官方 Compose `.env` 的 `GRAFT_IMAGE_TAG` 是唯一的镜像声明与更新策略 owner：固定的官方 server/web 仓库通过它共同选择镜像版本，模板默认 `latest`。

| `GRAFT_IMAGE_TAG` | Deployment mode | Upgrade candidates |
| --- | --- | --- |
| `latest` | stable tracking | strictly newer verified stable releases |
| `beta` | Beta tracking | strictly newer verified Beta releases |
| `vX.Y.Z` | fixed stable | strictly newer verified stable releases |
| `vX.Y.Z-beta.N` | fixed Beta | strictly newer verified Beta releases |

- 服务端只读取注入到 server 容器的 `GRAFT_IMAGE_TAG`，不得读取宿主机 `.env`；Web 不维护第二个可写策略值。
- 每次升级在 runner 执行前都解析到一个 verified manifest，且该 manifest 的 release、官方 server/web 仓库和 immutable digest 必须一致。解析出的 release tag 与 digest 是运行时状态，不能成为第二份配置。
- 对 `latest` 或 `beta`，runner 仅以本次运行的 Compose override 使用 manifest-derived explicit target 拉取、迁移、重建并验证 digest；成功后 `.env` 必须仍保留原 tracking tag。对 fixed tag 的管理员确认升级，runner 原子写入新选择的、更高且同频道的 fixed tag。
- 服务器必须在执行前重新验证严格升序、同频道、官方 Release membership 与 manifest digest；不得依赖前端候选列表阻止降级或跨频道。
- 切换 `latest`/`beta` 或 tracking/fixed 是独立的受治理操作，不是升级操作；当前范围不支持该切换。`nightly` 没有发布和 manifest 链，明确不支持。不存在 `GRAFT_UPDATE_POLICY` 或替代镜像配置键的兼容、alias、双读或 fallback。

## Installation Profile And Capability Matrix

### Platform Readiness Diagnostics

`Readiness Evaluator` is the platform's common read-only diagnostic model, not an Update Center view concern. A domain owns the facts and rules for its own prerequisite checks, while the shared model owns ordered checks, state, severity, blocking semantics, structured evidence, safe actions, and the overall next action. Update is the first consumer; Backup, Restore, Compose management, deployment, registry, and future controlled operations must publish prerequisites through this model instead of creating another status-list protocol.

The evaluator returns stable IDs and locale keys rather than final display text. Each check has an extensible `order`, `state`, `severity`, and `blocking` flag, so clients sort and render without a hard-coded checklist. Evidence is structured as code, localized label key, current value, expected value, pass state, and sensitivity. Actions are typed (`documentation`, `navigate`, `copy`, or `recheck`) and server-authorized; values that would reveal host paths or other deployment details remain restricted to callers with update management permission. The UI must show actionable failure or warning guidance inline and reserve a diagnostic Drawer for the full evidence set.

环境变量 `GRAFT_DEPLOYMENT_RUNTIME` 仅为 declared runtime，绝不是授权证据。Deployment Runtime 构造并拥有以下只读事实；Update 只消费它：

```go
DeploymentContext {
    Runtime
    ComposeRoot
    ConfigFiles
    ProjectName
    Fingerprint
}
```

检测需验证 official Compose 文件、同一 host absolute compose root、Docker socket 可用性、镜像坐标和运行中服务；声明与检测矛盾时 capability 降级为不可执行并显示原因。Deployment Runtime 是唯一允许解释环境变量、Docker inspect facts 与宿主机路径并构造 `DeploymentContext` 的组件；Container 只返回原始 inspect facts。已设置的 `GRAFT_DEPLOYMENT_COMPOSE_ROOT` 是唯一 Compose root authority，显式空值或相对路径必须 fail closed，不得自动回退到 binary 或 Docker 自动发现；只有变量未设置且 Docker API 可用时才通过 Container 检查当前 server 的 Compose labels、config files 和 bind mounts，生成一个或多个 host root 候选。候选必须显示给管理员确认，且每次升级启动前重新发现、验证并 freeze immutable operation snapshot；候选 key 和选择结果不持久化，前端不得提交原始 host path。

| Capability | Official Compose | Binary + systemd | Binary manual |
| --- | --- | --- | --- |
| 检查版本、Release Notes、checksum | supported | supported | supported |
| 自动执行（管理员确认） | supported | manual only | manual only |
| 备份 | supported | environment-dependent | environment-dependent |
| migration | runner 执行 | 指引显式执行 | 指引显式执行 |
| 恢复操作 | supported scope | operator-controlled | operator-controlled |

二进制部署是一等安装类型：同 tag 下载 server binary 和 web distribution，验证 manifest SHA256，并生成 systemd 或手工安装步骤。`GRAFT_DEPLOYMENT_RUNTIME=binary` 声明 runtime；`GRAFT_DEPLOYMENT_SERVICE_MANAGER=systemd|manual` 和 `GRAFT_DEPLOYMENT_SERVICE_NAME` 只描述受支持的服务控制面。Deployment Runtime 负责从受控 runtime facts 解释 service action，不用 `GRAFT_DEPLOYMENT_BINARY_PATH`、`GRAFT_DEPLOYMENT_WEB_ROOT` 或 Update 专属变量猜测 filesystem layout 或 `ExecStart`。缺少受支持 runtime facts 时，UI 必须显示阻断原因，不能给出不可靠的完整升级承诺。MVP 不替换正在运行的 binary，也不尝试接管 systemd。

## Update Lifecycle

更新目录的候选状态是 `AVAILABLE`。用户确认后，server 持久化受授权的 update request，但该记录不是执行状态。`graft-compose-runner` 随后取得一个 operation lease，成为该 operation 的唯一 lifecycle controller，并在官方 Compose 的 named update-state volume 中持久化版本化原子 `current.json` snapshot 和 append-only event records。runner 是唯一 writer，server 只读挂载并在 schema、operation binding、revision 与完整性校验成功后消费状态。状态包含 `operation`、`phase`、`progress`、safe `message`、`started_at`、`finished_at`、safe structured `error`、`runner_id` 和绑定目标/冻结 deployment snapshot 的版本化 identity；不包含凭证、host path、backup location/content、命令输出或 Docker stderr。事件记录按 operation 隔离，只有单调 revision、时间、phase、allowlisted action code 与安全文案输入；它们提供可回放的节点时间线，不是 runner stdout/stderr 的代理。

执行 phase 固定为 `READY -> PREFLIGHT -> BACKUP -> PULL_IMAGES -> STOP_SERVICES -> APPLY_UPDATE -> MIGRATION -> START_SERVICES -> HEALTH_CHECK -> SUCCESS`；任一阶段可至 `FAILED`，而迁移前的受控配置/镜像恢复以 `ROLLBACK` 终止。runner 在每个不可逆动作前先持久化 transition。迁移开始后不执行自动数据库 rollback/restore，失败只能记录 `FAILED` 与人工恢复证据。runner crash 或 host interruption 留下 state volume 中的最后 snapshot/events；stale non-terminal work 只能由已授权 manual recovery runner 接管，server 不得改写 phase 模拟恢复。

当 Docker 证实绑定 runner 已退出、但最后一个已验证 snapshot 仍非 terminal 时，server 必须返回 `state_source=runner_terminated`、`state_available=false`、`error=PLATFORM_UPDATE_RUNNER_TERMINATED`，并只保留最后可信的 `phase`、`progress` 与 safe `message` 作为诊断上下文。该投影不是运行中的 `READY` 或其他 phase；浏览器必须停止轮询并为持有 `platform-update.manage` 的管理员显示受控诊断入口。server 只持久化一条 allowlisted 诊断；绝不将 Docker stderr、命令、host path、凭证或 state-store 原始 I/O 错误返回给 API/UI。

管理员可调用 `POST /api/platform/updates/operations/{operationID}/recovery` 启动一次性恢复 runner。该操作要求 `platform-update.manage`，且只接受 runner identity 匹配且已经退出、snapshot 已验证为 non-terminal、并且 migration 尚未开始的操作；运行中、已终态、身份不匹配、已恢复、state 不可读或 migration 已开始均 fail closed。服务端在 Docker 拉取镜像前原子写入不透明恢复认领，作为授权和重复启动协调证据，而不是 runner phase；只有已证明恢复容器尚未创建时才会撤销认领，创建尝试后的不确定结果必须保留认领并返回冲突。恢复容器使用部署时显式配置的 `GRAFT_UPDATE_RECOVERY_RUNNER_IMAGE`，该值必须是官方 `graft-compose-runner` 的 immutable SHA-256 reference；不得复用可能尚未支持恢复协议的故障 runner 镜像，缺失或无效配置必须明确拒绝恢复。恢复 runner 只写入安全 terminal failure/rollback 结果，绝不继续被中断的升级；terminal projection 完成后才允许新升级重新进行资格检查。

`platform.update.operations.<operationID>` 是 Update 进度的 canonical realtime topic。Update 只发布已授权读取、已验证的 runner snapshot 和 allowlisted 节点事件；它不单独维护 SSE 或 WebSocket endpoint。`server/internal/realtime` 统一签发受 topic 和调用者绑定的一次性 ticket；同一 ticket 可由 WebSocket 或 SSE gateway 任一方单次消费。升级重建中断连接或用户打开新标签页时，浏览器先通过 API 发现 active operation、读取当前状态与指定 revision 之后的受限事件，再重新申请 ticket 并订阅。SSE/WebSocket 是 transport，不是事实源；server 不可用期间 runner 继续写 state volume，server 恢复后重新 reconcile 最新 snapshot 或 terminal result。活动请求尚未产生可验证 runner state 时，server 必须显式返回状态源不可用，前端不得将其伪装成持续 `READY` 进度；`runner_terminated` 同样不是 live progress，必须终止当前浏览器会话的进度轮询。

runner 没有 PostgreSQL credentials、HTTP API、SSE 或 WebSocket service。server 记录请求与审计启动事实，并将已验证 terminal runner result 幂等投影为 PostgreSQL update history、backup/audit references 和安全诊断；数据库是 terminal business history authority，不是 active phase/progress owner。权限分为 `platform-update.read`、`platform-update.check` 和 `platform-update.manage`；Phase 1 不暴露一个看似可执行但没有执行能力的 `execute` 权限。

### Update Operation Deployment Strategy Snapshot

`update_operations.deployment_strategy` 是创建操作时冻结的部署升级策略，值为 `stable_tracking`、`beta_tracking`、`pinned_stable`、`pinned_beta` 或 `unknown`。它必须与 API、审计元数据和持久化快照使用同一个名字；`update_mode` 不保留 API、存储或兼容 alias。

已发布的 `202607300001_update_operation_mode.sql` 先创建历史列 `update_mode`，因此不可重写。后续
`202607300002_rename_update_operation_deployment_strategy.sql` 前向迁移在该列存在且目标列尚不存在时重命名为
`deployment_strategy`，重命名检查约束并刷新列注释。官方链对新库和既有库均按 `300001 -> 300002` 执行，最终 schema
只保留 `deployment_strategy`。受 `0.11.0-beta.22` 影响的实例在 bootstrap 前必须停止旧 `server`/`web`；已发布的
`v0.11.0-beta.23` 只包含 `300001`，因此只能修复缺列故障。只有后续官方 release 的已验证 manifest 明确包含
`300002` 时，恢复流程才可完成该重命名；不得把未发布 tag 当作恢复目标。

官方 Compose 的正常启动顺序不允许 server 在 migration 前接收流量：`bootstrap` 执行 `graft migrate up`，server 依赖其成功完成，web 再依赖 server 健康。`0.11.0-beta.22` 的故障不是该顺序被绕过，而是该 release 的 server 已写入 `update_mode`，对应 `300001` 却未随 release 发布；该缺失使首次创建更新操作先于可用 schema 失败。任何受影响实例必须先通过 bootstrap 补齐迁移，不能依赖运行中的旧 server 自行修复。

### 启动失败诊断

更新启动在预检、操作持久化或 runner 交接失败时，Update 模块以 HTTP `request_id` 为唯一键保存一条不可变的受控诊断记录。记录包含稳定失败码、失败阶段、目标版本、已创建时的 operation/task 标识，以及最长 32 KiB 的错误链文本；文本必须在写入前过滤 password、token、secret、Authorization、Cookie 和 DSN 密码。诊断归 Update 所有，不属于 access log、App Log 或 audit 的替代品。

标准 `POST /api/platform/updates/operations` 仍只返回稳定错误码、可本地化安全文案和 `traceId`，不得把原始错误、Docker stderr、命令、宿主机路径或凭证写入响应。持有 `platform-update.manage` 的管理员可通过 `GET /api/platform/updates/diagnostics/{requestId}` 读取相同请求的已脱敏详情；该读取本身写入审计事件。Update Center 只在启动失败后使用 `traceId` 请求该受保护端点，并保留按 request ID 跳转 App Log 的入口。

失败处理必须同步尝试保存诊断，并通过注入的 `AppLogger` 写入带请求关联的 ERROR 应用日志；诊断保存或审计投递失败不得改变原始安全错误响应。一次启动失败仅发布一条失败审计事实，审计元数据不得携带诊断文本或 Compose 候选 key。

## Compose Execution Boundary

官方 Compose 安装的已确认升级由短生命周期 `graft-compose-runner` 执行。runner 接收由 server 预检并冻结的 target immutable references、manifest identity、operation identity、host Compose root 与受限 Compose command input，并成为该 operation 的 controller。输入通过 Docker API inline 传入，不要求 server 直接访问自动发现的宿主路径；runner 挂载 Docker socket 和其专用 named update-state volume 后执行备份、runner-scoped Compose override、`docker compose pull`、实际 digest 验证、bootstrap migration、受控 recreate 与 health check。它不得将已声明的 `latest` 或 `beta` 改写为解析出的 release tag，也不得以 retained container logs 作为状态存储或向数据库直写。

state volume 是 active execution/recovery authority：runner 使用原子 snapshot/event persistence、lease 和 heartbeat 保证只有一个 non-terminal operation。server 仅用 read-only mount 重新 reconcile、校验并投影 terminal history；state corruption、operation mismatch 或 stale lease 均 fail closed 并要求受保护的 recovery-runner 操作。runner logs 可以保留诊断证据，但不结算 update operation。详细的 lifecycle ownership 由 ADR-009 固定。

冻结的 `DeploymentContext.ComposeRoot` 必须是宿主机绝对路径，并在 runner 中以相同绝对路径挂载；Docker daemon 返回的 Linux host path 是执行权威。变量未设置时，Docker API 发现结果只作为待确认候选，不能由 server 容器路径、WSL 映射路径或前端输入推导和替代。server 不直接在自身容器中执行 Compose：它会在 recreate 中被停止，且容器内 CLI 和路径不能可靠代表 host daemon。Compose-root 与 Deployment Runtime 信任边界由 ADR-006（未被 ADR-009 取代的部分）与 ADR-008 固定。

## Scope

当前主题实现：release manifest、read-only discovery、顶部轻量更新提醒、`Platform -> System Maintenance -> Updates` 管理页、独立 backup capability、管理员确认的 Compose 执行、历史和恢复证据。顶部提醒只消费认证壳生命周期内的单一 discovery snapshot；它不承担独立轮询或第二套发现请求。

明确延后：无确认自动安装、多节点编排、Kubernetes executor、持久 Update Agent、容器内 binary replacement、host/systemd binary replacement，以及承诺自动 schema rollback。runner 是短生命周期 controller，不是持久 Update Agent。
