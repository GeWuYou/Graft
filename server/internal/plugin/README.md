# plugin

## 用途

`plugin` 定义 `server` 在历史命名下的模块生命周期契约与排序管理逻辑。

当前 canonical 语义是：

* backend business capability = module
* `plugin` 只是保留现有包路径与目录名的历史称呼

## 职责边界

这个模块负责：

* 定义模块生命周期接口
* 定义 `plugin.Descriptor` 与 `plugin.Builder` 这条历史命名下的 compile-time module 接线边界
* 暴露模块可见的运行时上下文
* 按依赖关系排序模块

这个模块不负责：

* 承载具体业务逻辑
* 替业务模块隐藏启动顺序
* 代替跨模块公共接口设计

## 主要入口

* `doc.go`：包职责与边界说明
* `plugin.go`：插件接口、描述符、Builder、上下文与管理器实现

## 关键依赖

* 由 `server/internal/app` 在运行时装配阶段调用
* 依赖 `container`、`menu`、`permission`、`cronx`、`eventbus`、`store`、`logger`、`i18n` 等核心能力
* 供 `server/plugins/*` 中的业务模块实现和消费
* 由 `server/internal/pluginregistry` 消费，用于生成 compile-time module registry

## 维护提示

如果新增能力会让模块直接依赖其它模块内部实现，或让核心开始承载业务判断，应先回看 `ai-plan/design/项目设计.md` 与 `ai-plan/design/插件与依赖注入设计.md`，再调整边界。
