// Package user 提供 MVP 路径中的用户与认证示例模块。
//
// 这个包用于演示单个业务能力如何在不回流 core 的前提下，声明登录、
// refresh、当前用户 active-session 列表、当前 refresh session logout、
// 当前用户 all-sessions revoke、保留当前会话的 revoke-others、
// 当前/管理员定向 session revoke、
// 管理员按用户读取与批量 revoke、
// 受保护请求最小 session hardening、
// 菜单、权限、路由和公开服务，
// 并通过稳定的模块边界接入平台。
package user
