// Package store 定义暴露给插件的中立持久化契约。
//
// 这些契约由 core 持有，插件只依赖显式仓储能力，不直接导入或泄漏具体 ORM 客户端。
package store
