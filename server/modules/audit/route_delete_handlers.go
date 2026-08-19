package audit

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	auditopenapi "graft/server/internal/contract/openapi/audit"
	"graft/server/internal/httpx"
	"graft/server/internal/logger"
	"graft/server/internal/module"
	auditstore "graft/server/modules/audit/store"
)

type auditBatchDeleter interface {
	DeleteByIDs(ctx context.Context, ids []uint64, input auditstore.AuditLogDeletionInput) (int64, error)
}

// handleBatchDeleteAuditLogs 处理受专用删除权限保护的审计日志批量删除。
func handleBatchDeleteAuditLogs(ctx *module.Context, moduleName string, reader auditReader) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		deleter, ok := reader.(auditBatchDeleter)
		if !ok {
			reported := reportAuditRouteError(ginCtx, ctx, "audit deletion service unavailable", errors.New("audit batch deleter is unavailable"),
				logger.StringField("module", moduleName), logger.StringField(logger.FieldOperation, "delete_audit_logs"))
			httpx.AbortAppError(ginCtx, ctx.I18n, ctx.Logger, reported)
			return
		}
		var request auditopenapi.PostAuditLogDeletionJSONRequestBody
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "ids"})
			return
		}
		ids := make([]uint64, 0, len(request.Ids))
		for _, id := range request.Ids {
			if id <= 0 {
				httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "ids"})
				return
			}
			ids = append(ids, uint64(id))
		}
		actorID, actorName := currentAuditActor(ginCtx)
		deleted, err := deleter.DeleteByIDs(withAuditRequestLocale(ginCtx, ctx), ids, auditstore.AuditLogDeletionInput{
			IdempotencyKey: ginCtx.GetHeader("Idempotency-Key"),
			ActorUserID:    actorID,
			ActorUsername:  actorName,
			RequestID:      ginCtx.GetHeader(httpheader.RequestID.String()),
		})
		if err != nil {
			switch {
			case errors.Is(err, auditstore.ErrAuditValidation):
				httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
			case errors.Is(err, auditstore.ErrAuditLogNotFound):
				httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusNotFound, messagecontract.CommonNotFound.String(), nil)
			case errors.Is(err, auditstore.ErrAuditLogProtected):
				httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusConflict, messagecontract.CommonInvalidArgument.String(), nil)
			default:
				reported := reportAuditRouteError(ginCtx, ctx, "delete audit logs failed", err,
					logger.StringField("module", moduleName), logger.StringField(logger.FieldOperation, "delete_audit_logs"))
				httpx.AbortAppError(ginCtx, ctx.I18n, ctx.Logger, reported)
			}
			return
		}
		ginCtx.Set("audit.batch.deleted", deleted)
		results := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			results = append(results, map[string]any{"id": fmt.Sprint(id), "status": "deleted"})
		}
		httpx.WriteSuccess(ginCtx, http.StatusOK, map[string]any{
			"operation_id": ginCtx.GetHeader("Idempotency-Key"),
			"summary":      map[string]int{"requested": len(ids), "succeeded": len(ids), "failed": 0},
			"results":      results,
		})
	}
}
