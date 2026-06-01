package audit

import (
	auditopenapi "graft/server/internal/contract/openapi/audit"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	auditcontract "graft/server/modules/audit/contract"
)

func registerAuditRoutes(ctx *module.Context, pluginName string, reader auditReader, guard auditGuard) {
	group := ctx.Router.Group(auditcontract.AuditGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(auditcontract.AuditOverviewCollection, guard.read, handleReadAuditOverview(ctx, pluginName, reader))
	group.GET(auditcontract.AuditCollection, guard.read, handleListAuditLogs(ctx, pluginName, reader))
	group.GET(auditcontract.AuditIncidentItem, guard.read, handleReadAuditIncident(ctx, pluginName, reader))
}

var _ auditopenapi.ReadServerInterface = auditReadGeneratedHandler{}
