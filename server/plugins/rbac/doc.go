// Package rbac 提供 MVP 阶段最小可用的后端授权插件。
//
// 当前实现只负责把基于仓储的权限判断能力暴露为稳定的
// `pluginapi.Authorizer`，不在这一阶段承载角色、权限管理接口或菜单。
package rbac
