# Platform Self Update Design

本文件是 Graft 平台自更新的仓库级设计真相。它把 Graft 定义为管理自身生命周期的 self-hosted platform，而不是在运行中替换容器内二进制的安装器。

## Product Boundary

- 入口为 `Platform -> System Maintenance -> Updates`；左侧 Graft 标识下的当前版本是进入此页的快捷入口。
- 更新是管理员确认后的受治理操作：自动检查可以启用，自动安装不在当前承诺范围内。
- `server`、`web`、数据库迁移和配置快照必须对应同一目标 release；不得混用版本。官方 Compose 用共享 `GRAFT_IMAGE_TAG` 选择 server 和 web 镜像，初次部署可手工选择 `latest`、`beta` 或固定版本；runner 只能写入从已验证 manifest 得到的明确版本 Tag，并且必须验证 pull 后 digest 与同一 manifest 一致；Tag 不能单独成为升级事实。
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
- stable 安装只选择比当前版本新的 stable release，不将 beta 作为候选。
- beta 安装选择更高的 beta 或其后的 stable release。因此 `0.9.1-beta.1` 可升级到 `0.9.1-beta.2`、`0.10.0-beta.1`，以及后来发布的稳定版本。
- catalog 条目必须通过 manifest 校验后才可展示为可升级；release notes 只作为展示内容，不能替代 manifest。
- Update 持久化最近成功验证的 catalog projection，缓存窗口为 24 小时；网络失败必须显示检查失败和缓存陈旧，不能把缓存解释为“已是最新”。

## Compose Update Policy

官方 Compose `.env` 是更新策略与镜像声明的唯一 owner：固定的官方 server/web 仓库通过一个 `GRAFT_IMAGE_TAG` 共同选择镜像版本；模板默认 `latest`，也可手工使用 `beta` 或固定发行版本。模板默认 `GRAFT_UPDATE_POLICY=stable`，其余有效值为 `beta`、`fixed` 或 `manual`。之后服务端每次都重新读取 `.env`，Web 不维护策略副本。

- `stable` 只选择已验证 stable release；`beta` 只选择已验证 beta channel 允许的 release；`fixed` 仅允许管理员从已验证 catalog 选择具体版本。
- `manual` 只保存策略，不改写镜像引用、不拉取、不迁移、不重建服务。
- stable、beta 与 fixed 在实际 runner 执行前均解析到一个 verified manifest；runner 从 server/web 的完整目标引用提取同一明确版本 Tag，写入 `GRAFT_IMAGE_TAG` 后比对 server/web digest，成功才可跨越 migration/recreate 边界。
- `latest` 与 `beta` 是可变的手工部署 tag，不能替代上述 manifest/digest 验证。`nightly` 没有发布和 manifest 链，明确不支持；不存在替代镜像配置键的兼容路径。

## Installation Profile And Capability Matrix

环境变量 `GRAFT_UPDATE_DEPLOYMENT_MODE` 仅为 declared mode，绝不是授权证据。Update 模块构造以下只读事实：

```go
InstallationProfile {
    DeclaredMode
    DetectedMode
    Capability
}
```

检测需验证 official Compose 文件、同一 host absolute compose root、Docker socket 可用性、镜像坐标和运行中服务；声明与检测矛盾时 capability 降级为不可执行并显示原因。已设置的 `GRAFT_UPDATE_COMPOSE_ROOT` 是唯一 Compose root authority，显式空值或相对路径必须 fail closed，不得自动回退到 binary 或 Docker 自动发现；只有环境变量未设置且 Docker API 可用时才检查当前 server 容器的 Compose labels、config files 和 bind mounts，生成一个或多个 host root 候选。候选必须显示给管理员确认，且每次升级启动前重新发现并验证；候选 key 和选择结果不持久化，前端不得提交原始 host path。

| Capability | Official Compose | Binary + systemd | Binary manual |
| --- | --- | --- | --- |
| 检查版本、Release Notes、checksum | supported | supported | supported |
| 自动执行（管理员确认） | supported | manual only | manual only |
| 备份 | supported | environment-dependent | environment-dependent |
| migration | runner 执行 | 指引显式执行 | 指引显式执行 |
| 恢复操作 | supported scope | operator-controlled | operator-controlled |

二进制部署是一等安装类型：同 tag 下载 server binary 和 web distribution，验证 manifest SHA256，并生成 systemd 或手工安装步骤。`GRAFT_UPDATE_BINARY_PATH`、`GRAFT_UPDATE_WEB_ROOT` 与 `GRAFT_UPDATE_SERVICE_MANAGER=systemd|manual` 是完整指引的必要输入；systemd 还要求 `GRAFT_UPDATE_SERVICE_NAME`。缺少任何必要输入时，UI 必须显示阻断原因，不能给出不可靠的完整升级承诺。MVP 不替换正在运行的 binary，也不尝试接管 systemd。

## Update Lifecycle

更新目录的候选状态是 `AVAILABLE`；持久 UpdateOperation 的标准生命周期为 `PLANNING -> BACKING_UP -> PULLING -> MIGRATING -> RECREATING -> VERIFYING -> SUCCESS`，任一阶段可到 `FAILED`。runner 交接后由 receipt 结算最终阶段；迁移后失败固定为 `NEEDS_ATTENTION`，迁移前完成配置和镜像恢复则为 `RECOVERED`。不能把 forward-only schema migration 伪装成自动数据库 rollback。

所有阶段写入 Task Runtime、审计事件、不可变 target manifest digest、backup reference、migration 和 health receipt。权限分为 `platform-update.read`、`platform-update.check` 和 `platform-update.manage`；Phase 1 不暴露一个看似可执行但没有执行能力的 `execute` 权限。

### 启动失败诊断

更新启动在预检、操作持久化或 runner 交接失败时，Update 模块以 HTTP `request_id` 为唯一键保存一条不可变的受控诊断记录。记录包含稳定失败码、失败阶段、目标版本、已创建时的 operation/task 标识，以及最长 32 KiB 的错误链文本；文本必须在写入前过滤 password、token、secret、Authorization、Cookie 和 DSN 密码。诊断归 Update 所有，不属于 access log、App Log 或 audit 的替代品。

标准 `POST /api/platform/updates/operations` 仍只返回稳定错误码、可本地化安全文案和 `traceId`，不得把原始错误、Docker stderr、命令、宿主机路径或凭证写入响应。持有 `platform-update.manage` 的管理员可通过 `GET /api/platform/updates/diagnostics/{requestId}` 读取相同请求的已脱敏详情；该读取本身写入审计事件。Update Center 只在启动失败后使用 `traceId` 请求该受保护端点，并保留按 request ID 跳转 App Log 的入口。

失败处理必须同步尝试保存诊断，并通过注入的 `AppLogger` 写入带请求关联的 ERROR 应用日志；诊断保存或审计投递失败不得改变原始安全错误响应。一次启动失败仅发布一条失败审计事实，审计元数据不得携带诊断文本或 Compose 候选 key。

## Compose Execution Boundary

官方 Compose 安装的已确认升级由短生命周期 runner 执行。runner 没有业务状态、HTTP API 或常驻生命周期；它只接收由 server 预检并固定的目标完整镜像引用、预期 manifest digest、host compose root 和受限 Compose 命令。输入通过 Docker API inline 传入，不要求 server 直接访问自动发现的宿主路径；runner 挂载 Docker socket 后执行备份、原子写 `.env`、`docker compose pull`、验证实际 digest、bootstrap migration、受控 recreate、health check，并将 marker-bounded receipt 写入带 operation/protocol labels 的保留容器日志。server 通过 Docker API 读取、校验并结算 receipt 后才清理 runner。

`GRAFT_UPDATE_COMPOSE_ROOT` 非空时必须是宿主机绝对路径，并在 runner 中以相同绝对路径挂载；Docker daemon 返回的 Linux host path 是执行权威。为空时，Docker API 发现结果只作为待确认候选，不能由 server 容器路径、WSL 映射路径或前端输入推导和替代。server 不直接在自身容器中执行 Compose：它会在 recreate 中被停止，且容器内 CLI 和路径不能可靠代表 host daemon。详细信任边界由 `ADR-006` 固定。

## Scope

当前主题实现：release manifest、read-only discovery、顶部轻量更新提醒、`Platform -> System Maintenance -> Updates` 管理页、独立 backup capability、管理员确认的 Compose 执行、历史和恢复证据。顶部提醒只消费认证壳生命周期内的单一 discovery snapshot；它不承担独立轮询或第二套发现请求。

明确延后：无确认自动安装、多节点编排、Kubernetes executor、持久 Update Agent、容器内 binary replacement、host/systemd binary replacement，以及承诺自动 schema rollback。
