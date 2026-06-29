package audit

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	auditstore "graft/server/modules/audit/store"
)

type auditReader interface {
	List(ctx context.Context, query ListQuery) (ListResult, error)
	Detail(ctx context.Context, id uint64) (DetailResult, error)
	Overview(ctx context.Context, preset auditstore.AuditTimePreset) (OverviewResult, error)
	Incident(ctx context.Context, eventID uint64) (IncidentResult, error)
	VisibilityPolicy(ctx context.Context) (VisibilityPolicyResult, error)
	UpdateVisibilityDefault(
		ctx context.Context,
		strategy auditstore.AuditVisibilityStrategy,
		userID *uint64,
		username string,
	) (auditstore.AuditVisibilityDefault, error)
	UpdateVisibilityOverride(ctx context.Context, input auditstore.UpsertAuditVisibilityOverrideInput) (auditstore.AuditVisibilityOverride, error)
	DeleteVisibilityOverride(ctx context.Context, source auditstore.AuditSource, actionKey string) error
}

type auditListResult = ListResult
type auditDetailResult = DetailResult
type auditOverviewResult = OverviewResult
type auditIncidentResult = IncidentResult

type auditGuard struct {
	read   gin.HandlerFunc
	manage gin.HandlerFunc
}

// handleListAuditLogs 创建审计日志列表查询的处理器。
// handleListAuditLogs 返回一个用于查询审计日志列表的 Gin 处理器。
// 处理器会校验列表查询参数和可见性范围访问权限，读取审计日志列表，并将结果写回响应。
func handleListAuditLogs(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	logger := auditRouteLogger(ctx)

	return func(ginCtx *gin.Context) {
		_, query, invalidField := bindGeneratedAuditListParams(ginCtx)
		if !ensureAuditListParamsBound(ginCtx, ctx, invalidField) {
			return
		}

		if !ensureAuditVisibilityScopeAccess(ginCtx, ctx, query.VisibilityScope) {
			return
		}

		result, err := reader.List(withAuditRequestLocale(ginCtx, ctx), query)
		if err != nil {
			if handleAuditListReadError(ginCtx, ctx, logger, moduleName, err) {
				return
			}
			return
		}

		payload, mapErr := toAuditLogListResponse(result)
		if mapErr != nil {
			logger.Error("map audit logs response failed",
				zap.String("module", moduleName),
				zap.Error(mapErr),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}
