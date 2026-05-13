# Graft

Graft 是一个基于 Go 和 Vue 3 的组合式后台平台，目标是通过插件机制快速接入新功能，而不是把所有业务硬编码进一个固定后台。

当前仓库优先完善设计与实施文档，核心决策已经收敛为：

* 后端：`Go + Gin + Ent + PostgreSQL`
* 前端：`Vue 3 + TypeScript + Vite`
* UI：`TDesign Vue Next`
* 架构：插件化平台
* 依赖管理：轻量 DI / 服务注册，不引入重量级 IoC

## 文档

* [项目设计](ai-plan/design/项目设计.md)
* [插件与依赖注入设计](ai-plan/design/插件与依赖注入设计.md)
* [前端架构设计](ai-plan/design/前端架构设计.md)
* [MVP 实施计划](ai-plan/roadmap/MVP实施计划.md)
* [AI 任务追踪与恢复设计](ai-plan/design/AI任务追踪与恢复设计.md)
* [AI Plan 启动索引](ai-plan/public/README.md)
* [AI 环境清单说明](.ai/environment/README.md)

## 当前状态

项目目前仍处于架构与实施设计阶段。开始编码前，先以 `ai-plan/design/` 与 `ai-plan/roadmap/` 下文档固化边界与约束；复杂长期任务再以 `ai-plan/public/` 下主题跟踪和轨迹文件作为恢复入口。

仓库同时维护 `.ai/environment/` 作为环境真值入口：

* `tools.raw.yaml` 记录当前机器与仓库相关的原始环境事实
* `tools.ai.yaml` 记录给 AI 和贡献者消费的精简环境摘要
