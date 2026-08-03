# 本地三版本验证环境

本地验证环境在 `.local/graft-validation/` 下维护三套相互隔离的官方 Compose 部署，用于快速复现和排查版本问题。它们均使用生产环境配置（`GRAFT_APP_ENV=production`），并保留完整功能配置。

| 实例 | 镜像 Tag | 默认地址 |
| --- | --- | --- |
| `beta` | `beta` | http://127.0.0.1:3101 |
| `latest` | `latest` | http://127.0.0.1:3102 |
| `fixed` | `v0.11.0-beta.39` | http://127.0.0.1:3103 |

每套实例使用独立的 Compose project name、数据目录和更新状态 volume，不共享 PostgreSQL、Redis、应用、备份或导入数据。三套实例仍共用本机 Docker daemon；server 挂载 Docker socket 的能力意味着实例中的有权限操作者可影响该 daemon 管理的容器，不能将它视为安全隔离边界。

## 使用工具

统一通过以下命令维护环境：

```bash
python3 scripts/local_graft_validation.py <command>
```

支持的命令为：`init`、`up`、`down`、`pull`、`status`、`logs`、`doctor`、`sync-compose`、`set-config`、`record-access` 和 `reset`。

`init` 创建本地环境；先执行 `pull <instance>`，它会按已拉取 server 镜像的 OCI revision 同步相同发行版的官方 `compose.yml` 与配置模板，再执行 `up <instance>`。`down` 用于停止服务；`status`、`logs`、`doctor` 用于状态与诊断；`sync-compose`、`set-config`、`record-access` 用于维护本地配置和访问记录；`reset` 用于显式重置指定实例。端口变更或启动前应先处理工具报告的端口冲突。

固定版本镜像不可用时，保留 `fixed` 实例的配置和端口并报告失败；不得自动回退到 `beta` 或 `latest`。

## 本地访问登记

工具生成 `.local/README.md` 作为本机访问登记页，记录实例地址、当前镜像 Tag、状态和常用操作。该文件及其中的本地凭据均不纳入 Git。

首次初始化时可使用默认管理员账号 `graft` 和密码 `graft-admin` 登录；系统强制修改密码后，后续密码仅由操作者在 `.local/README.md` 中自行维护。不要将变更后的密码、令牌或机器绝对路径写入受跟踪文档。
