# Platform Governance

This directory is for cross-cutting platform governance that does not fit one single product domain.

## Use This Directory For

- cache and configuration governance
- localization and shared-asset governance
- logging or other platform-level operating rules
- cross-cutting authority notes that are neither purely backend nor purely frontend

Configuration Schema、`.env`/Compose 校验、启动 preflight 和配置升级诊断由
[配置治理与迁移规范](配置治理与迁移规范.md) 定义；配置值属于部署配置还是 System Config 仍由
[部署配置与运行时策略治理规范](部署配置与运行时策略治理规范.md) 决定。

## Boundaries

- Use `../backend/` or `../frontend/` when authority is clearly concentrated on one side.
- Use `../../domains/` when the document is mainly about one bounded capability area.
