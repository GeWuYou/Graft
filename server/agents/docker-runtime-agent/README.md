# Docker Runtime Agent

Docker Runtime Agent 是独立部署单元。Runtime Target 负责 enrollment、delivery、activation 和 ledger；Agent 只读取交付后的配置与一次性 bootstrap token，不能拥有 PostgreSQL 或 Vault 凭据。架构与信任边界以 [Credential Vault and Runtime Target Agent Protocol](../../../ai-plan/design/architecture/credential-vault-and-runtime-target-agent-protocol.md) 为准。

## Local debug

本地 Agent 调试有明确的独立单元：`server: Air + Agent Gateway` 负责后端 Gateway，`agent: Docker Runtime` 负责 Agent 进程，`web: dev` 负责前端；`fullstack: dev` 会并行组合这三个单元。单独调试 Agent 时，先执行 `agent: prepare Docker Runtime development`，再启动 Gateway Server，最后启动 Agent（它会调用 Backend-owned `deliver`）。Gateway Server 读取 Server 目录中的 `.env.docker-runtime-agent`，该文件以 `server/.env` 为基线，并覆写本地数据库、Redis、Vault 和 TLS 集成值；普通 `server: Air 热重载` 仍只读取 `server/.env`，不会准备或启动 Agent。Agent 只读取同一 Agent 根目录下由 delivery 生成且被忽略的 `agent.json`，并从隔离的 `.data/docker-runtime-agent-dev` 读取其 bootstrap、trust 与 state 材料。仓库在 [VS Code 本地任务说明](../../../.vscode/README.md) 中分别提供共享 PostgreSQL 与独立 PostgreSQL 模板；开发者将所选模板复制为被忽略的 `.vscode/tasks.json` 后，`fullstack: dev` 会显式使用对应模式。独立模式在 `127.0.0.1:15432` 启动本地 PostgreSQL；直接运行 CLI 时默认仍为 `shared`。宿主机 Agent 随后连接本地 Server 的 `127.0.0.1:8443` bootstrap listener 和 `127.0.0.1:8444` mTLS listener。

配置中的 `target_id`、`agent_id`、bootstrap token 与 CA 必须来自 Backend-owned delivery；不能为了调试手写或重用生产 token。VS Code 的 Agent 启动项会等待这些开发交付材料，因此日志直接输出到调试器且无需构建 Agent 镜像；正常 production/Compose 启动仍会在配置错误时立即失败。

开发 Vault 在 pending delivery 后被重启或重建时，显式运行 `graft dev docker-runtime-agent reset`。它归档本开发拓扑的忽略数据并重新准备依赖；Runtime Target binding 的状态边界仍以 [Agent Protocol](../../../ai-plan/design/architecture/credential-vault-and-runtime-target-agent-protocol.md) 为准。

`config/agent.json.example` 保留给 Compose 容器，使用 Compose DNS 名和容器内挂载路径；宿主机调试使用同一 Agent 根目录下由 Backend delivery 写入的 `agent.json`。`agent.json.example` 仅展示该宿主机交付文件的结构，不能替代 delivery。

## Official Compose

官方根 Compose 默认启动 `docker-runtime-agent`，使用与 server/web 相同的 `GRAFT_IMAGE_TAG`。服务只挂载配置、token、trust bundle、state 和 Docker socket；启动前必须完成 Backend 的 Vault PKI、bootstrap TLS、mTLS 与 delivery 准备，挂载契约以 [Agent Protocol](../../../ai-plan/design/architecture/credential-vault-and-runtime-target-agent-protocol.md) 为准。Agent 不开放入站端口，Compose healthcheck 只读取本地 readiness 状态。
