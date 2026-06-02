# module

## 用途

`module` 定义 `server` 的模块生命周期契约与排序管理逻辑。

当前 canonical 语义是：

* backend business capability = module
* `plugin` 只在少量导出接口命名中保留为历史称呼

## 职责边界

这个模块负责：

* 定义模块生命周期接口
* 定义 `module.Spec` 与 `module.Builder` 这条 compile-time module 接线边界
* 暴露模块可见的运行时上下文
* 按依赖关系排序模块

这个模块不负责：

* 承载具体业务逻辑
* 替业务模块隐藏启动顺序
* 代替跨模块公共接口设计

## 主要入口

* `doc.go`：包职责与边界说明
* `module.go`：模块生命周期接口、`Spec`、`Builder`、上下文与管理器实现

## 关键依赖

* 由 `server/internal/app` 在运行时装配阶段调用
* 依赖 `container`、`menu`、`permission`、`cronx`、`eventbus`、`store`、`logger`、`i18n` 等核心能力
* 供 `server/modules/*` 中的业务模块实现和消费
* 由 `server/internal/moduleregistry` 消费，用于生成 compile-time module registry

## 维护提示

如果新增能力会让模块直接依赖其它模块内部实现，或让核心开始承载业务判断，应先回看 `ai-plan/design/项目设计.md` 与 `ai-plan/design/模块与依赖注入设计.md`，再调整边界。
