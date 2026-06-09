// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	auditopenapi "graft/server/internal/contract/openapi/audit"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	auditcontract "graft/server/modules/audit/contract"
)

func registerAuditRoutes(ctx *module.Context, moduleName string, reader auditReader, guard auditGuard) {
	group := ctx.Router.Group(auditcontract.AuditGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(auditcontract.AuditOverviewCollection, guard.read, handleReadAuditOverview(ctx, moduleName, reader))
	group.GET(auditcontract.AuditCollection, guard.read, handleListAuditLogs(ctx, moduleName, reader))
	group.GET(auditcontract.AuditIncidentItem, guard.read, handleReadAuditIncident(ctx, moduleName, reader))
}

var _ auditopenapi.ReadServerInterface = auditReadGeneratedHandler{}
