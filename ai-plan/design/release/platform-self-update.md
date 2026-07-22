# Platform Self Update Design

本文件是 Graft 平台自更新的仓库级设计真相。它把 Graft 定义为管理自身生命周期的 self-hosted platform，而不是在运行中替换容器内二进制的安装器。

## Product Boundary

- 入口为 `Platform -> Updates`；左侧 Graft 标识下的当前版本是进入此页的快捷入口。
- 更新是管理员确认后的受治理操作：自动检查可以启用，自动安装不在当前承诺范围内。
- `server`、`web`、数据库迁移和配置快照必须对应同一目标 release；不得混用 tag 或 mutable tag 作为升级事实。
- `server/modules/update` 和 `server/modules/backup` 是两个独立模块。Update 消费 Backup capability；Atlas migration 仍由 core CLI 拥有，不创建 migration 业务模块。

## Release Authority And Manifest

GitHub Release 是 release catalog 和 release notes 的权威来源；GHCR digest 是 Compose 运行时镜像身份。`.github/workflows/publish.yml` 在同一 release tag 的 server/web 镜像均 push 成功后生成并附带 `release-manifest.json`：

```json
{
  "schema_version": 1,
  "version": "0.10.0-beta.1",
  "channel": "beta",
  "release_tag": "v0.10.0-beta.1",
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
    "checksums": "graft-sha256sums-v0.10.0-beta.1.txt"
  },
  "migration": { "required_before_runtime": true, "command": "graft migrate up", "mode": "forward-only" }
}
```

`release-manifest.json.sha256` 与 manifest 同时作为 GitHub Release asset 发布；读取端先验证 manifest checksum，后验证 JSON 内容。Compose runner 从 `server/runner/compose/Dockerfile` 由同一 release workflow 构建和推送；manifest 只接受当前 Buildx 输出的 immutable digest，绝不接受仓库变量、外部断言 digest 或 mutable tag。`required_before_runtime` 表示每个受支持升级都必须执行显式且幂等的 migration command；它不声称每个 release 都包含 SQL 变更。目标 manifest 的 tag、SemVer channel、artifact 名称和 server/web/runner digest/reference 必须交叉校验，任何不一致、缺失 runner 或 mutable runner identity 都使目标不可执行。

## Version And Channel Selection

- 版本遵循 `MAJOR.MINOR.PATCH` 和 `MAJOR.MINOR.PATCH-beta.N`。
- stable 安装只选择比当前版本新的 stable release，不将 beta 作为候选。
- beta 安装选择更高的 beta 或其后的 stable release。因此 `0.9.1-beta.1` 可升级到 `0.9.1-beta.2`、`0.10.0-beta.1`，以及后来发布的稳定版本。
- catalog 条目必须通过 manifest 校验后才可展示为可升级；release notes 只作为展示内容，不能替代 manifest。
- Update 持久化最近成功验证的 catalog projection，缓存窗口为 24 小时；网络失败必须显示检查失败和缓存陈旧，不能把缓存解释为“已是最新”。

## Installation Profile And Capability Matrix

环境变量 `GRAFT_UPDATE_DEPLOYMENT_MODE` 仅为 declared mode，绝不是授权证据。Update 模块构造以下只读事实：

```go
InstallationProfile {
    DeclaredMode
    DetectedMode
    Capability
}
```

检测需验证 official Compose 文件、同一 host absolute compose root、Docker socket 可用性、镜像坐标和运行中服务；声明与检测矛盾时 capability 降级为不可执行并显示原因。

| Capability | Official Compose | Binary + systemd | Binary manual |
| --- | --- | --- | --- |
| 检查版本、Release Notes、checksum | supported | supported | supported |
| 自动执行（管理员确认） | supported | manual only | manual only |
| 备份 | supported | environment-dependent | environment-dependent |
| migration | runner 执行 | 指引显式执行 | 指引显式执行 |
| 恢复操作 | supported scope | operator-controlled | operator-controlled |

二进制部署是一等安装类型：同 tag 下载 server binary 和 web distribution，验证 checksum，并生成 systemd 或手工安装步骤。MVP 不替换正在运行的 binary，也不尝试接管 systemd。

## Update Lifecycle

更新任务的标准状态为 `AVAILABLE -> PREFLIGHT -> BACKING_UP -> PULLING -> MIGRATING -> RECREATING -> VERIFYING -> SUCCESS`；任一阶段可到 `FAILED`。`ROLLBACK_PENDING` 和 `RESTORED` 记录恢复决策，不能把 forward-only schema migration 伪装成自动数据库 rollback。

所有阶段写入 Task Runtime、审计事件、不可变 target manifest digest、backup reference、migration 和 health receipt。权限分为 `platform-update.read`、`platform-update.check` 和 `platform-update.manage`；Phase 1 不暴露一个看似可执行但没有执行能力的 `execute` 权限。

## Compose Execution Boundary

官方 Compose 安装的已确认升级由短生命周期 runner 执行。runner 没有业务状态、HTTP API 或常驻生命周期；它只接收由 server 预检并固定的目标 digest、host compose root、受限 Compose 命令和 receipt 位置，挂载 Docker socket 后执行：备份、`docker compose pull`、bootstrap migration、受控 recreate、health check、写 receipt。

`GRAFT_UPDATE_COMPOSE_ROOT` 必须是宿主机绝对路径，并在 runner 中以相同绝对路径挂载。server 不直接在自身容器中执行 Compose：它会在 recreate 中被停止，且容器内 CLI 和路径不能可靠代表 host daemon。详细信任边界由 `ADR-006` 固定。

## Scope

当前主题实现：release manifest、read-only discovery、Update Center、独立 backup capability、管理员确认的 Compose 执行、历史和恢复证据。

明确延后：无确认自动安装、多节点编排、Kubernetes executor、持久 Update Agent、容器内 binary replacement、host/systemd binary replacement，以及承诺自动 schema rollback。
