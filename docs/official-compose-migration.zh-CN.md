# 迁移到官方 Compose 部署

更新中心只会为官方 Compose 部署执行受控升级。本指南将已有 Graft 实例迁移到该部署模型；更新中心不是通用的宿主机命令执行器。

## 迁移前准备

- 安排维护窗口，并保留已独立验证的数据库备份。
- 记录当前镜像、Compose 文件、数据目录挂载、环境变量和对外端口。
- 以下命令必须在运行 Graft 的 Docker daemon 宿主机上执行。不要使用容器内路径、Windows/WSL 映射路径，或另一台 Docker 主机的路径。
- 当前受控 Compose runner 只支持 Linux `amd64` 主机。

## 1. 准备官方 Compose 根目录

选择 Docker daemon 宿主机上的绝对目录，例如 `/opt/graft`。目录顶层必须包含官方 `compose.yml`（或 `compose.yaml`）及该 Compose 项目使用的 `.env` 文件。

```bash
# 将 v0.11.0-beta.21 替换为当前已部署的确切固定官方 Tag。
# 在下文 GRAFT_IMAGE_TAG 中使用同一个 Tag；不要变更版本或频道。
git clone --branch v0.11.0-beta.21 --depth 1 https://github.com/GeWuYou/Graft.git /opt/graft
cd /opt/graft
cp compose.env.example .env
```

不要把旧的自定义 Compose 文件直接合并到官方文件。应克隆当前已部署的确切发行版本，只迁移受支持的部署值，例如凭据、端口、挂载目录和允许来源。迁移期间不要变更版本或发行频道。官方拓扑包含 `server`、`web`、`bootstrap`、`postgres` 和 `redis`；server 还必须挂载 `/var/run/docker.sock`，以发现项目并启动短生命周期升级 runner。

## 2. 保留已有数据

首次执行 `docker compose up` 前，必须在 `.env` 中设置已有持久目录。替换它们会创建新的空实例，而不是迁移当前实例。检查 `compose.yml` 中的 PostgreSQL 挂载，并保留已有的可选目录：

```dotenv
GRAFT_APPLICATION_ROOT_HOST_PATH=/opt/graft/apps
GRAFT_BACKUP_ARTIFACT_HOST_PATH=/opt/graft/backups
GRAFT_PROJECT_IMPORT_HOST_PATH=/opt/graft/imports
```

迁移已有实例时，请保持数据库凭据和 `GRAFT_AUTH_JWT_SECRET` 不变。在同一次迁移中改变它们，可能使新服务无法使用旧数据库，或使已有会话失效。

## 3. 声明升级策略

在 `.env` 中配置官方值：

```dotenv
# 使用第 1 步克隆且当前已部署的确切固定官方 Tag。
# 迁移期间不要变更此版本或其发行频道。
GRAFT_IMAGE_TAG=v0.11.0-beta.21
GRAFT_DEPLOYMENT_RUNTIME=compose
```

`GRAFT_IMAGE_TAG` 是唯一的镜像版本与升级策略配置，必须与第 1 步克隆 Compose 发行版所用的确切固定 Tag 相同；不要增加第二个更新策略变量。在后续受控升级中，`latest`、`beta` 等跟随标签仍保留在 `.env` 中，runner 只在本次升级期间使用由 manifest 推导出的发行目标。固定 Tag 升级会以原子方式写入同一频道中较新的已验证固定 Tag。

通常应让 `GRAFT_DEPLOYMENT_COMPOSE_ROOT` 保持未设置。Deployment Runtime 会通过 Docker 发现 server 自身的 Compose 项目，结果存在歧义时管理员必须确认候选。只有自动发现无法识别项目时，才将该值设置为第 1 步中的绝对目录：

```dotenv
GRAFT_DEPLOYMENT_COMPOSE_ROOT=/opt/graft
```

显式空值、相对路径、无效路径或过期路径都会阻断受控升级。Graft 不会回退到容器内路径、binary updater 或无关的 Compose 项目。

## 4. 启动并验证

```bash
cd /opt/graft
docker compose pull
docker compose up -d
docker compose ps
```

确认 `bootstrap` 已成功完成，且 `server`、`web`、`postgres` 与 `redis` 已健康或按预期运行。随后以管理员身份登录，在“平台 > 更新”中选择“检查更新”。唯一且高置信度的候选可以直接使用；多个候选或低置信度候选必须在升级流程中选择。页面展示的证据仅用于诊断，不应据此手工修改 Docker labels。

在应用包含 L2 或更高风险 migration sidecar 的发行版之前，使用将实际执行 migration 的同一官方 `bootstrap` 镜像和生产配置运行目标数据预检。官方 Compose 会将发行 sidecar 只读挂载到 `/opt/graft/migrations`：

```bash
docker compose run --rm --no-deps --entrypoint /app/graft bootstrap \
  migrate preflight --manifest /opt/graft/migrations/<module>/migrations/<version>_<name>.preflight.yaml
docker compose run --rm bootstrap
```

`graft migrate preflight` 只读执行，不会应用、生成、改写或重排 migration，也不会更新 Atlas revision。重复、引用或不变量检查失败时，必须先协调数据或遵循发行说明，再运行正常的 migration 命令。

必须执行 `docker compose pull` 和 `docker compose up -d`，以启动上文选择的同一固定发行版本。克隆该发行版到执行这些命令之间，不要变更 `GRAFT_IMAGE_TAG`、版本或频道。

## 恢复受影响的 `0.11.0-beta.22` 更新中心

`0.11.0-beta.22` 在启动更新操作时可能失败：该版本的 server 会写入历史列 `update_mode`，但相应 migration 未随该 release 的 bootstrap 镜像发布。这不是正常 Compose 启动顺序失效：官方 Compose 会先运行 `bootstrap`，server 仅在它成功后启动，web 仅在 server 健康后对外提供服务。不要在受影响镜像上重复尝试页面内升级。

先创建并验证数据库备份。本故障已发布的修复版本是 `v0.11.0-beta.23`；其 release manifest 声明了以下不可变镜像身份：

```text
ghcr.io/gewuyou/graft-server@sha256:b791e3d46d956ef9b0026cc64f90cb93e495633949506b0fae2457b63abedcaa
ghcr.io/gewuyou/graft-web@sha256:b30ef5b0647b1449051646b1d53fcf2dde54556bf221d65f6b45b07607356a0c
```

在 `.env` 中设置 `GRAFT_IMAGE_TAG=v0.11.0-beta.23`，并将上述 digest 与该 GitHub Release 的 `release-manifest.json` 资产核对，
然后在 Docker daemon 宿主机的 Compose 根目录执行。若 manifest 或镜像 digest 不一致，不要继续：

```bash
docker compose pull --policy always bootstrap server web
docker compose stop server web
docker compose run --rm bootstrap
docker compose up -d --no-deps --force-recreate server web
docker compose ps
docker compose exec -T server curl --fail --silent http://127.0.0.1:8080/healthz
```

已发布的 `v0.11.0-beta.23` bootstrap 会应用不可变的 `202607300001_update_operation_mode.sql`，修复 `0.11.0-beta.22` 缺少列导致的故障。
`202607300002_rename_update_operation_deployment_strategy.sql` 目前尚未包含在任何已发布版本中；不要虚构或选择未发布的 tag。
完成本次恢复后，只有在后续官方 release 的已验证 manifest 明确包含 `300002` 时，才能期待规范的 `deployment_strategy` schema。
应用该后续版本后，契约不会保留 `update_mode` 的 API 或存储 alias。

## 故障排查

| 检查结果 | 应修复的内容 |
| --- | --- |
| 未检测到官方 Compose 部署 | 使用仓库 `compose.yml`，设置 `GRAFT_DEPLOYMENT_RUNTIME=compose`，再从该根目录重新创建实例。 |
| 无法检测 Compose 项目 | 检查 server 是否挂载 `/var/run/docker.sock`。若自动发现仍不可用，将 `GRAFT_DEPLOYMENT_COMPOSE_ROOT` 设为 Docker daemon 宿主机上的绝对 Compose 根目录。 |
| 镜像策略无效 | 将 `GRAFT_IMAGE_TAG` 设为 `latest`、`beta` 或兼容的固定发行版 Tag。 |
| 无法获取发行信息 | 检查主机访问发行服务的网络，然后重新检查。此状态不代表一定有可升级版本。 |
| 缺少升级权限 | 使用已授予 `platform-update.manage` 的账户登录。 |

不要为了让非官方部署通过检查而添加 alias、回退镜像变量或自定义 runner。当 Graft 无法证明 Compose 根目录、拓扑、镜像策略和发行身份符合要求时，受控升级会保持阻断状态。

## 回退边界

数据库迁移开始前，如果预迁移步骤失败，runner 可以恢复之前的配置快照和镜像引用。迁移开始后验证失败时，Graft 会记录 `NEEDS_ATTENTION`，不会自动回滚或恢复数据库。请使用迁移前的备份，并按故障处理流程进行迁移后恢复。
