// Package mcp 提供产品 MCP 的受控 HTTP transport 基础，不承载业务 Tool 定义。
//
// 它只接收 auth 模块已经验证的个人 API Token 调用者，并为后续 OpenAPI compiler
// 提供 RBAC 优先的 scope 收窄与一次性确认令牌能力。
package mcp
