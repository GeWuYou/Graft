# Docker Builder Agent

Docker Builder Agent 是独立部署单元。Runtime Target 负责 enrollment、delivery、activation 和 ledger；Agent 只读取交付后的配置与一次性 bootstrap token，不能拥有 PostgreSQL 或 Vault 凭据。

## Local debug

VS Code 的启动任务会先调用 Backend-owned `graft dev docker-builder-agent prepare` 与 `deliver`，将被忽略的 `config/agent.local.json` 和交付材料写入隔离的 `.data/docker-builder-agent-dev`。宿主机 Agent 随后连接本地 Server 的 `127.0.0.1:8443` bootstrap listener 和 `127.0.0.1:8444` mTLS listener。

配置中的 `target_id`、`agent_id`、bootstrap token 与 CA 必须来自 Backend-owned delivery；不能为了调试手写或重用生产 token。VS Code 的 Agent 启动项会等待这些开发交付材料，因此日志直接输出到调试器且无需构建 Agent 镜像；正常 production/Compose 启动仍会在配置错误时立即失败。

开发 Vault 在 pending delivery 后被重启或重建时，显式运行 `graft dev docker-builder-agent reset`。它仅归档本开发拓扑的忽略数据并重新准备依赖，避免绕过 Runtime Target 的单一 live grant 约束。

`config/agent.json.example` 保留给 Compose 容器，使用 Compose DNS 名和容器内挂载路径；不要把它用于宿主机调试。

## Compose profile

官方根 Compose 的 `docker-builder-agent` profile 使用与 server/web 相同的 `GRAFT_IMAGE_TAG`。profile 只挂载配置、token、trust bundle、state 和 Docker socket；启动前必须完成 Backend 的 Vault PKI、bootstrap TLS、mTLS 与 delivery 准备。
