# Docker Builder Agent

Docker Builder Agent 是独立部署单元。Runtime Target 负责 enrollment、delivery、activation 和 ledger；Agent 只读取交付后的配置与一次性 bootstrap token，不能拥有 PostgreSQL 或 Vault 凭据。

## Local debug

将 `config/agent.local.json.example` 复制为被忽略的 `config/agent.local.json`。该模板供宿主机 Agent 使用，连接本地 Server 的 `127.0.0.1:8443` bootstrap listener 和 `127.0.0.1:8444` mTLS listener，并复用根 Compose 的 `.data/docker-builder-agent/{bootstrap,trust,state}` 交付目录。

配置中的 `target_id`、`agent_id`、bootstrap token 与 CA 必须来自 Backend-owned delivery；不能为了调试手写或重用生产 token。VS Code 的 `agent: Docker Builder` 与 `agent: Docker Builder（单次）` 启动项会显式使用该本地配置，因此日志直接输出到调试器且无需构建 Agent 镜像。

`config/agent.json.example` 保留给 Compose 容器，使用 Compose DNS 名和容器内挂载路径；不要把它用于宿主机调试。

## Compose profile

官方根 Compose 的 `docker-builder-agent` profile 使用与 server/web 相同的 `GRAFT_IMAGE_TAG`。profile 只挂载配置、token、trust bundle、state 和 Docker socket；启动前必须完成 Backend 的 Vault PKI、bootstrap TLS、mTLS 与 delivery 准备。
