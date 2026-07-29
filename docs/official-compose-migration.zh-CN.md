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
git clone https://github.com/GeWuYou/Graft.git /opt/graft
cd /opt/graft
cp compose.env.example .env
```

不要把旧的自定义 Compose 文件直接合并到官方文件。应从已版本化的 `compose.yml` 开始，只迁移受支持的部署值，例如凭据、端口、挂载目录和允许来源。官方拓扑包含 `server`、`web`、`bootstrap`、`postgres` 和 `redis`；server 还必须挂载 `/var/run/docker.sock`，以发现项目并启动短生命周期升级 runner。

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
# latest 跟随稳定频道，beta 跟随 Beta 频道。
# 固定版本（例如 v0.11.0-beta.21）会锁定在对应频道。
GRAFT_IMAGE_TAG=beta
GRAFT_UPDATE_DEPLOYMENT_MODE=compose
```

`GRAFT_IMAGE_TAG` 是唯一的镜像版本与升级策略配置。不要增加第二个更新策略变量。跟随标签在成功受控升级后仍保持 `latest` 或 `beta`；已验证的发行版 digest 只在本次升级期间使用。

通常应让 `GRAFT_UPDATE_COMPOSE_ROOT` 保持未设置。server 会通过 Docker 发现自身 Compose 项目，结果存在歧义时管理员必须确认候选。只有自动发现无法识别项目时，才将该值设置为第 1 步中的绝对目录：

```dotenv
GRAFT_UPDATE_COMPOSE_ROOT=/opt/graft
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

## 故障排查

| 检查结果 | 应修复的内容 |
| --- | --- |
| 未检测到官方 Compose 部署 | 使用仓库 `compose.yml`，设置 `GRAFT_UPDATE_DEPLOYMENT_MODE=compose`，再从该根目录重新创建实例。 |
| 无法检测 Compose 项目 | 检查 server 是否挂载 `/var/run/docker.sock`。若自动发现仍不可用，将 `GRAFT_UPDATE_COMPOSE_ROOT` 设为 Docker daemon 宿主机上的绝对 Compose 根目录。 |
| 镜像策略无效 | 将 `GRAFT_IMAGE_TAG` 设为 `latest`、`beta` 或兼容的固定发行版 Tag。 |
| 无法获取发行信息 | 检查主机访问发行服务的网络，然后重新检查。此状态不代表一定有可升级版本。 |
| 缺少升级权限 | 使用已授予 `platform-update.manage` 的账户登录。 |

不要为了让非官方部署通过检查而添加 alias、回退镜像变量或自定义 runner。当 Graft 无法证明 Compose 根目录、拓扑、镜像策略和发行身份符合要求时，受控升级会保持阻断状态。

## 回退边界

数据库迁移开始前，如果预迁移步骤失败，runner 可以恢复之前的配置快照和镜像引用。迁移开始后验证失败时，Graft 会记录 `NEEDS_ATTENTION`，不会自动回滚或恢复数据库。请使用迁移前的备份，并按故障处理流程进行迁移后恢复。
