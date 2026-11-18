package audit

import (
	"context"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	auditopenapi "graft/server/internal/contract/openapi/audit"
	"graft/server/internal/contract/openapi/strictsupport"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	auditcontract "graft/server/modules/audit/contract"
)

// registerAuditRoutes 为审计相关接口注册路由，挂载请求 ID 中间件，并注册审计日志、审计事件、可见性策略、可见性覆盖及可选已保存视图的相关路由。
// registerAuditRoutes 为审计相关接口注册路由，挂载请求 ID 中间件，并注册审计日志、审计事件、可见性策略、可见性覆盖及可选已保存视图的相关路由。
// 审计日志批量删除路由当前与 canonical OpenAPI 的 deletions 路径存在契约漂移；
// strict adapter 覆盖该操作的绑定契约，但线上路由保持既有 batch-delete 片段，等待契约修复后切换。
func registerAuditRoutes(ctx *module.Context, moduleName string, reader auditReader, guard auditGuard, savedViews moduleapi.SavedViewService) {
	group := ctx.Router.Group(auditcontract.AuditGroup)
	group.Use(httpx.RequestIDMiddleware())
	wrapper := auditopenapi.ServerInterfaceWrapper{
		Handler: auditopenapi.NewStrictHandlerWithOptions(
			auditStrictServer{
				savedViews:          auditSavedViewHandlersOf(ctx, savedViews),
				listLogs:            handleListAuditLogs(ctx, moduleName, reader),
				readLog:             handleReadAuditLog(ctx, moduleName, reader),
				readIncident:        handleReadAuditIncident(ctx, moduleName, reader),
				readPolicy:          handleReadAuditVisibilityPolicy(ctx, moduleName, reader),
				updatePolicy:        handleUpdateAuditVisibilityDefault(ctx, moduleName, reader),
				upsertOverride:      handleUpsertAuditVisibilityOverride(ctx, moduleName, reader),
				upsertOverrideBatch: handleUpsertAuditVisibilityOverrides(ctx, moduleName, reader),
				deleteOverride:      handleDeleteAuditVisibilityOverride(ctx, moduleName, reader),
				batchDeleteLogs:     handleBatchDeleteAuditLogs(ctx, moduleName, reader),
			},
			nil,
			auditopenapi.StrictGinServerOptions{
				HandlerErrorFunc: strictsupport.HandlerErrorHandler(ctx.I18n, ctx.Logger),
			},
		),
		ErrorHandler: strictsupport.BindingErrorHandler(ctx.I18n, messagecontract.CommonInvalidArgument.String()),
	}
	if savedViews != nil {
		registerAuditSavedViewRoutes(group, ctx, guard.read, wrapper)
	}
	group.GET(auditcontract.AuditCollection, guard.read, wrapper.GetAuditLogs)
	// 手写保留：与 canonical OpenAPI 的 /logs/deletions 存在契约漂移，等待契约修复后切换挂载。
	group.POST(auditcontract.AuditBatchDeleteCollection, guard.delete, handleBatchDeleteAuditLogs(ctx, moduleName, reader))
	group.GET(auditcontract.AuditItem, guard.read, wrapper.GetAuditLogDetail)
	group.GET(auditcontract.AuditIncidentItem, guard.read, wrapper.GetAuditIncident)
	group.GET(auditcontract.AuditVisibilityPolicyCollection, guard.manage, wrapper.GetAuditVisibilityPolicy)
	group.PUT(auditcontract.AuditVisibilityPolicyCollection, guard.manage, wrapper.PutAuditVisibilityPolicy)
	group.PUT(auditcontract.AuditVisibilityOverrideCollection, guard.manage, wrapper.PutAuditVisibilityOverride)
	group.PUT(auditcontract.AuditVisibilityOverrideBatchCollection, guard.manage, wrapper.PutAuditVisibilityOverridesBatch)
	group.DELETE(auditcontract.AuditVisibilityOverrideCollection, guard.manage, wrapper.DeleteAuditVisibilityOverride)
}

// auditStrictServer 将审计路由主体适配到生成的 strict server 契约；
// 编译期断言保证 OpenAPI 审计操作变化时必须在适配层补齐。
type auditStrictServer struct {
	savedViews          auditSavedViewHandlers
	listLogs            gin.HandlerFunc
	readLog             gin.HandlerFunc
	readIncident        gin.HandlerFunc
	readPolicy          gin.HandlerFunc
	updatePolicy        func(*gin.Context, auditopenapi.PutAuditVisibilityPolicyJSONRequestBody)
	upsertOverride      func(*gin.Context, auditopenapi.PutAuditVisibilityOverrideJSONRequestBody)
	upsertOverrideBatch func(*gin.Context, auditopenapi.PutAuditVisibilityOverridesBatchJSONRequestBody)
	deleteOverride      gin.HandlerFunc
	batchDeleteLogs     gin.HandlerFunc
}

var _ auditopenapi.StrictServerInterface = auditStrictServer{}

func (s auditStrictServer) GetAuditLogs(ctx context.Context, _ auditopenapi.GetAuditLogsRequestObject) (auditopenapi.GetAuditLogsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.listLogs)
}

func (s auditStrictServer) PostAuditLogDeletion(ctx context.Context, _ auditopenapi.PostAuditLogDeletionRequestObject) (auditopenapi.PostAuditLogDeletionResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.batchDeleteLogs)
}

func (s auditStrictServer) GetAuditLogDetail(ctx context.Context, _ auditopenapi.GetAuditLogDetailRequestObject) (auditopenapi.GetAuditLogDetailResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.readLog)
}

func (s auditStrictServer) GetAuditIncident(ctx context.Context, _ auditopenapi.GetAuditIncidentRequestObject) (auditopenapi.GetAuditIncidentResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.readIncident)
}

func (s auditStrictServer) GetAuditVisibilityPolicy(ctx context.Context, _ auditopenapi.GetAuditVisibilityPolicyRequestObject) (auditopenapi.GetAuditVisibilityPolicyResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.readPolicy)
}

func (s auditStrictServer) PutAuditVisibilityPolicy(ctx context.Context, request auditopenapi.PutAuditVisibilityPolicyRequestObject) (auditopenapi.PutAuditVisibilityPolicyResponseObject, error) {
	if request.Body == nil {
		return nil, strictsupport.ErrMissingRequestBody
	}
	return nil, strictsupport.Dispatch(ctx, func(ginCtx *gin.Context) { s.updatePolicy(ginCtx, *request.Body) })
}

func (s auditStrictServer) PutAuditVisibilityOverride(ctx context.Context, request auditopenapi.PutAuditVisibilityOverrideRequestObject) (auditopenapi.PutAuditVisibilityOverrideResponseObject, error) {
	if request.Body == nil {
		return nil, strictsupport.ErrMissingRequestBody
	}
	return nil, strictsupport.Dispatch(ctx, func(ginCtx *gin.Context) { s.upsertOverride(ginCtx, *request.Body) })
}

func (s auditStrictServer) PutAuditVisibilityOverridesBatch(ctx context.Context, request auditopenapi.PutAuditVisibilityOverridesBatchRequestObject) (auditopenapi.PutAuditVisibilityOverridesBatchResponseObject, error) {
	if request.Body == nil {
		return nil, strictsupport.ErrMissingRequestBody
	}
	return nil, strictsupport.Dispatch(ctx, func(ginCtx *gin.Context) { s.upsertOverrideBatch(ginCtx, *request.Body) })
}

func (s auditStrictServer) DeleteAuditVisibilityOverride(ctx context.Context, _ auditopenapi.DeleteAuditVisibilityOverrideRequestObject) (auditopenapi.DeleteAuditVisibilityOverrideResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.deleteOverride)
}

func (s auditStrictServer) GetAuditLogSavedViews(ctx context.Context, _ auditopenapi.GetAuditLogSavedViewsRequestObject) (auditopenapi.GetAuditLogSavedViewsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.savedViews.list)
}

func (s auditStrictServer) PostAuditLogSavedView(ctx context.Context, _ auditopenapi.PostAuditLogSavedViewRequestObject) (auditopenapi.PostAuditLogSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.savedViews.create)
}

func (s auditStrictServer) PutAuditLogSavedView(ctx context.Context, _ auditopenapi.PutAuditLogSavedViewRequestObject) (auditopenapi.PutAuditLogSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.savedViews.update)
}

func (s auditStrictServer) DeleteAuditLogSavedView(ctx context.Context, _ auditopenapi.DeleteAuditLogSavedViewRequestObject) (auditopenapi.DeleteAuditLogSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.savedViews.delete)
}
