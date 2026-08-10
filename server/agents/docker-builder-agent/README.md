# Docker Builder Agent

Docker Builder Agent 是独立部署单元。Runtime Target 负责 enrollment、delivery、activation 和 ledger；Agent 只读取交付后的配置与一次性 bootstrap token，不能拥有 PostgreSQL 或 Vault 凭据。

## Local debug

将 `.env.example` 复制为 `.env`，再将 `config/agent.json.example` 复制为被 `.env` 引用的本地配置。配置中的 `target_id`、`agent_id`、bootstrap token 与 CA 必须来自 Backend-owned delivery；不能为了调试手写或重用生产 token。

VS Code 的 `agent: Docker Builder` 与 `agent: Docker Builder（单次）` 启动项会读取该 `.env`。命令行也可用 `--config <path>` 显式覆盖环境变量。

## Compose profile

官方根 Compose 的 `docker-builder-agent` profile 使用与 server/web 相同的 `GRAFT_IMAGE_TAG`。profile 只挂载配置、token、trust bundle、state 和 Docker socket；启动前必须完成 Backend 的 Vault PKI、bootstrap TLS、mTLS 与 delivery 准备。
