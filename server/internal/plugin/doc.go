// Package plugin 定义历史命名下的模块生命周期契约与运行时管理逻辑。
//
// 当前仓库的 canonical 架构语义是 compile-time modules；这里只保留
// `plugin` 包名来兼容现有目录与 import path。这个包保持模块排序、注册、
// 启动与关闭规则可见，避免业务能力回流到 core 运行时，也避免通过隐式
// 框架行为隐藏模块边界。
package plugin
