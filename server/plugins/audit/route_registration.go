package audit

import (
	auditopenapi "graft/server/internal/contract/openapi/audit"
	"graft/server/internal/httpx"
	"graft/server/internal/plugin"
	auditcontract "graft/server/plugins/audit/contract"
)

func registerAuditRoutes(ctx *plugin.Context, pluginName string, reader auditReader, guard auditGuard) {
	group := ctx.Router.Group(auditcontract.AuditGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(auditcontract.AuditCollection, guard.read, handleListAuditLogs(ctx, pluginName, reader))
}

var _ auditopenapi.ReadServerInterface = auditReadGeneratedHandler{}
