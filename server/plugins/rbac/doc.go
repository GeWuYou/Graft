// Package rbac 提供 MVP 阶段最小可用的后端授权与只读管理插件。
//
// 当前实现把基于仓储的权限判断能力暴露为稳定的 `pluginapi.Authorizer`，
// 同时提供角色/权限的只读管理接口与菜单元数据；写操作与更复杂治理仍保留
// 在后续 RBAC 管理切片中收敛。
package rbac
