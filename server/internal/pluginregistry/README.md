# pluginregistry

## 用途

`pluginregistry` 暴露 compile-time 生成的历史 plugin 命名下的模块接线产物，作为 `serve` / `migrate` 等中心化入口唯一允许消费的模块清单。

## 职责边界

这个模块负责：

* 暴露生成后的 `plugin.Descriptor` 集合
* 按依赖关系构造运行时模块实例
* 汇总当前 owner-aligned 默认迁移目录集合
* 提供唯一允许的集中接线文件 `generated.go`

这个模块不负责：

* 运行时扫描模块目录
* 动态发现、热加载或外部分发
* 承载业务逻辑

## 主要入口

* `registry.go`：描述符快照、运行时模块构造与默认迁移目录汇总
* `generated.go`：由 `go generate ./internal/pluginregistry` 生成的唯一集中接线产物
* `cmd/pluginregistrygen/main.go`：生成器实现

## 迁移目录语义

* `DefaultMigrationDir`
  - CLI 默认值使用的选择器，不对应真实目录
  - `graft migrate up` / `graft dev` / `graft validate smoke` 在未显式传入 `--dir` 时，会通过它展开 live core-owned + module-owned 默认迁移链
* `HistoricalSharedMigrationDir`
  - 保留历史共享 Atlas 迁移目录的显式路径
  - 仅用于手动或诊断场景；它不再属于默认迁移链
* `CoreMigrationDirs()`
  - 暴露默认链中的 core-owned live 迁移目录
  - 当前 `internal/httpx/migrations` 持有 `access_logs` 的 canonical migration authority

## 维护提示

新增 backend module 时，先在历史目录 `server/plugins/<name>/descriptor.go` 暴露 `NewDescriptor()`，再运行
`go generate ./internal/pluginregistry` 更新 `generated.go`。除生成产物外，不要再手写中心化模块列表。
