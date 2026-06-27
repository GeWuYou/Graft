package audit

import (
	auditopenapi "graft/server/internal/contract/openapi/audit"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	auditcontract "graft/server/modules/audit/contract"
)

// registerAuditRoutes 为审计相关接口注册路由，并挂载请求 ID 中间件。
// 它注册审计概览、审计日志、审计事件、可见性策略及可见性覆盖的读取、更新和删除路由。
func registerAuditRoutes(ctx *module.Context, moduleName string, reader auditReader, guard auditGuard) {
	group := ctx.Router.Group(auditcontract.AuditGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(auditcontract.AuditOverviewCollection, guard.read, handleReadAuditOverview(ctx, moduleName, reader))
	group.GET(auditcontract.AuditCollection, guard.read, handleListAuditLogs(ctx, moduleName, reader))
	group.GET(auditcontract.AuditItem, guard.read, handleReadAuditLog(ctx, moduleName, reader))
	group.GET(auditcontract.AuditIncidentItem, guard.read, handleReadAuditIncident(ctx, moduleName, reader))
	group.GET(auditcontract.AuditVisibilityPolicyCollection, guard.manage, handleReadAuditVisibilityPolicy(ctx, moduleName, reader))
	group.PUT(auditcontract.AuditVisibilityPolicyCollection, guard.manage, handleUpdateAuditVisibilityDefault(ctx, moduleName, reader))
	group.PUT(auditcontract.AuditVisibilityOverrideCollection, guard.manage, handleUpsertAuditVisibilityOverride(ctx, moduleName, reader))
	group.DELETE(auditcontract.AuditVisibilityOverrideCollection, guard.manage, handleDeleteAuditVisibilityOverride(ctx, moduleName, reader))
}

var _ auditopenapi.ReadServerInterface = auditReadGeneratedHandler{}
