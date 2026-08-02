# ADR-020: Configuration Governance Schema

- Status: accepted
- Date: 2026-08-02
- Scope: deployment environment configuration, official Compose topology, startup preflight, configuration upgrade diagnostics, and CI validation

## Context

部署配置随着版本演进会新增、重命名、废弃或删除。仅靠 Viper 默认值、`.env` 示例和运行日志无法让已升级的部署在
数据库迁移或服务初始化前发现缺失和遗留配置；Compose 对环境、挂载、Secret 与 runtime 参数也没有统一的应用级
契约。配置生命周期因此分散在 runtime、模板、Compose 和 CI 中。

## Decision

采用嵌入 `server` 二进制的版本化 YAML Configuration Schema 作为部署配置生命周期的唯一 authority。每个
`vN.yaml` 是完整不可变快照，声明环境项的类型、必填性、默认值、说明、引入版本、废弃/删除状态、替代项和
约束，并声明官方 Compose 所需服务拓扑。最新被二进制支持的 Schema 是校验目标；`GRAFT_CONFIG_SCHEMA_VERSION`
声明 operator 的 `.env` 模板版本。

所有启动命令通过同一 source-aware resolver 按 `CLI > process environment > .env > Schema default` 得到最终值。
每个值携带不暴露 secret 的 provenance。`serve` 与 `migrate up` 都在数据库、migration executor、runtime、业务
模块和 HTTP server 之前执行校验；失败时禁止后续初始化并以非零状态退出。

`graft config validate` 是公共验证入口，提供文本、JSON 和仅生成建议的 patch 输出。配置 finding 的退出码为
`2`，内部/I/O 故障为 `1`。Compose 使用 Compose Specification loader 在无 Docker daemon 的条件下解析、插值和
比较契约；官方生产部署增加 one-shot gate，使 bootstrap 与 server 都依赖其成功完成。CI 在 build 后、migration
test 前运行同一官方模板校验。

首期 migration 只诊断并生成 patch；不写入 `.env`、Compose、Secret 或外部配置。旧键的重命名通过
deprecated/removed 生命周期和 replacement 建议治理，不引入 alias、双读、兼容 DTO 或隐式 fallback。

## Consequences

新增 required 配置成为可验证的 release contract，已部署实例能在不可逆操作前收到完整、来源明确且秘密安全的错误。
官方 Compose 和 CLI 共享一个判断面，CI 能在 release 前发现模板漂移。Schema 版本快照和变更记录为未来受控的
配置迁移提供基础，但初期仍要求 operator 审阅并应用建议。

实现需要将当前默认值和校验从分散的 Go 代码、模板及 Compose 收敛到 Schema consumer，并为动态默认值维持受限
命名 resolver。每个新增配置都要同步 Schema、模板、Compose、CI 和回归测试，增加了发布前审查责任。

## Rejected Alternatives

- 仅扩展 Viper 校验：无法表达版本、废弃状态、Compose 拓扑和统一 provenance。
- 以 `.env.example` 作为 Schema：不能安全表达类型、跨字段语义、Compose contract 或已删除项，且不随二进制发布。
- 在 Compose 启动后由应用警告：已可能执行目录初始化、数据库迁移或服务启动，不能满足最早阻断要求。
- 自动原地修改 `.env`：可能损坏 operator 自定义、权限与 Secret 管理边界；在显式迁移命令获得独立安全设计前不实现。
