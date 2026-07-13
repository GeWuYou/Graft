package audit

import (
	auditopenapi "graft/server/internal/contract/openapi/audit"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	auditcontract "graft/server/modules/audit/contract"
)

// registerAuditRoutes 为审计相关接口注册路由，挂载请求 ID 中间件，并注册审计日志、审计事件、可见性策略、可见性覆盖及可选已保存视图的相关路由。
func registerAuditRoutes(ctx *module.Context, moduleName string, reader auditReader, guard auditGuard, savedViews moduleapi.SavedViewService) {
	group := ctx.Router.Group(auditcontract.AuditGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(auditcontract.AuditCollection, guard.read, handleListAuditLogs(ctx, moduleName, reader))
	if savedViews != nil {
		registerAuditSavedViewRoutes(group, ctx, guard.read, savedViews)
	}
	group.GET(auditcontract.AuditItem, guard.read, handleReadAuditLog(ctx, moduleName, reader))
	group.GET(auditcontract.AuditIncidentItem, guard.read, handleReadAuditIncident(ctx, moduleName, reader))
	group.GET(auditcontract.AuditVisibilityPolicyCollection, guard.manage, handleReadAuditVisibilityPolicy(ctx, moduleName, reader))
	group.PUT(auditcontract.AuditVisibilityPolicyCollection, guard.manage, handleUpdateAuditVisibilityDefault(ctx, moduleName, reader))
	group.PUT(auditcontract.AuditVisibilityOverrideCollection, guard.manage, handleUpsertAuditVisibilityOverride(ctx, moduleName, reader))
	group.DELETE(auditcontract.AuditVisibilityOverrideCollection, guard.manage, handleDeleteAuditVisibilityOverride(ctx, moduleName, reader))
}

var _ auditopenapi.ReadServerInterface = auditReadGeneratedHandler{}

func (auditReadGeneratedHandler) GetAuditLogSavedViews(auditopenapi.GetAuditLogSavedViewsParams) {}

func (auditReadGeneratedHandler) PostAuditLogSavedView(auditopenapi.PostAuditLogSavedViewParams, auditopenapi.PostAuditLogSavedViewJSONRequestBody) {
}

func (auditReadGeneratedHandler) PutAuditLogSavedView(int64, auditopenapi.PutAuditLogSavedViewParams, auditopenapi.PutAuditLogSavedViewJSONRequestBody) {
}

func (auditReadGeneratedHandler) DeleteAuditLogSavedView(int64, auditopenapi.DeleteAuditLogSavedViewParams) {
}
