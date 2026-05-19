# pluginregistry

## 用途

`pluginregistry` 暴露 compile-time 生成的插件接线产物，作为 `serve` / `migrate` 等中心化入口唯一允许消费的插件清单。

## 职责边界

这个模块负责：

* 暴露生成后的 `plugin.Descriptor` 集合
* 按依赖关系构造运行时插件实例
* 汇总当前默认迁移目录集合
* 提供唯一允许的集中接线文件 `generated.go`

这个模块不负责：

* 运行时扫描插件目录
* 动态发现、热加载或外部插件分发
* 承载业务逻辑

## 主要入口

* `registry.go`：描述符快照、运行时插件构造与默认迁移目录汇总
* `generated.go`：由 `go generate ./internal/pluginregistry` 生成的唯一集中接线产物
* `cmd/pluginregistrygen/main.go`：生成器实现

## 维护提示

新增插件时，先在 `server/plugins/<name>/descriptor.go` 暴露 `NewDescriptor()`，再运行 `go generate ./internal/pluginregistry` 更新
`generated.go`。除生成产物外，不要再手写中心化插件列表。
