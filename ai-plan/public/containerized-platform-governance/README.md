# Containerized Platform Governance

目标为执行已批准的工程治理与自包含本机 Docker/Compose 运维方案，覆盖构建、平台更新、迁移与诊断。

恢复先遵守根 AGENTS.md。治理来源为 AGENTS.md，task class 为 cross-boundary，recovery source 为 parent topic。
权威为根规则、server/web 领域规则、ADR-029 及模块契约。
范围包括工程入口、本机执行与部署验收；实施进度和 Work Contract 由
[todos](todos/containerized-platform-governance-tracking.md) 持有，
[trace](traces/containerized-platform-governance-trace.md) 记录证据。

平台代码基线为 346854ff5daae3f804f8fad706202c9e983d2067。完整方案与阶段依据见
[实施路线](../../roadmap/MVP实施计划.md)。运行复验和 CI 结果须绑定实际交付 revision。
