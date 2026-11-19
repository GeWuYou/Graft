package network

import (
	"context"
	"database/sql"
	"errors"
	networkopenapi "graft/server/internal/contract/openapi/network"
	"graft/server/internal/contract/openapi/strictsupport"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/contract/errorcode"
	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	networkcontract "graft/server/modules/network/contract"
)

type routeRuntime struct {
	ctx     *module.Context
	service *Service
}

func registerNetworkRoutes(ctx *module.Context, service *Service) error {
	if ctx == nil || ctx.Router == nil {
		return nil
	}
	if service == nil {
		return errors.New("platform network service is unavailable")
	}
	auth, err := module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
	if err != nil {
		return err
	}
	authorizer, err := module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
	if err != nil {
		return err
	}
	routes := routeRuntime{ctx: ctx, service: service}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	group := ctx.Router.Group(networkcontract.NetworkGroup)
	group.Use(httpx.RequestIDMiddleware())
	wrapper := networkopenapi.ServerInterfaceWrapper{
		Handler: networkopenapi.NewStrictHandlerWithOptions(networkStrictServer{routes: routes}, nil, networkopenapi.StrictGinServerOptions{
			RequestErrorHandlerFunc: strictsupport.RequestErrorHandler(ctx.I18n, messagecontract.CommonInvalidArgument.String(), nil),
			HandlerErrorFunc:        strictsupport.HandlerErrorHandler(ctx.I18n, ctx.Logger),
		}),
		ErrorHandler: strictsupport.BindingErrorHandler(ctx.I18n, messagecontract.CommonInvalidArgument.String()),
	}
	group.GET(networkcontract.OutboundNetworkRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), strictsupport.Route(wrapper.GetPlatformNetworkOutbound, nil))
	group.PUT(networkcontract.OutboundNetworkRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkWritePermission.String(), publisher), strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PutPlatformNetworkOutbound, func(c *gin.Context) { routes.handlePut(c, nil) }))
	group.POST(networkcontract.OutboundNetworkResetRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkWritePermission.String(), publisher), strictsupport.Route(wrapper.ResetPlatformNetworkOutbound, routes.handleReset))
	group.POST(networkcontract.LegacyDiagnosticRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkDiagnosePermission.String(), publisher), strictsupport.Route(wrapper.PostPlatformNetworkDiagnostic, routes.handleLegacyDiagnostic))
	group.GET(networkcontract.LegacyDiagnosticHistoryRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), strictsupport.Route(wrapper.GetPlatformNetworkDiagnosticHistory, routes.handleLegacyDiagnosticHistory))
	group.GET(networkcontract.ConnectivityTargetsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), strictsupport.Route(wrapper.GetPlatformConnectivityTargets, nil))
	group.GET(networkcontract.ConnectivityCustomTargetsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkManageTargetsPermission.String(), publisher), strictsupport.Route(wrapper.GetPlatformConnectivityCustomTargets, nil))
	group.POST(networkcontract.ConnectivityCustomTargetsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkManageTargetsPermission.String(), publisher), strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PostPlatformConnectivityCustomTarget, func(c *gin.Context) { routes.handleCreateConnectivityCustomTarget(c, nil) }))
	group.DELETE(networkcontract.ConnectivityCustomTargetRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkManageTargetsPermission.String(), publisher), strictsupport.Route(wrapper.DeletePlatformConnectivityCustomTarget, routes.handleDeleteConnectivityCustomTarget))
	group.GET(networkcontract.ConnectivityLatestRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), strictsupport.Route(wrapper.GetPlatformConnectivityLatest, nil))
	group.GET(networkcontract.ConnectivityAggregateRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), strictsupport.Route(wrapper.GetPlatformConnectivityAggregate, nil))
	group.POST(networkcontract.ConnectivityRunRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkDiagnosePermission.String(), publisher), strictsupport.Route(wrapper.PostPlatformConnectivityRun, routes.handleConnectivityRun))
	group.POST(networkcontract.ConnectivityBatchRunRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkDiagnosePermission.String(), publisher), strictsupport.Route(wrapper.PostPlatformConnectivityBatchRun, nil))
	group.GET(networkcontract.ConnectivityHistoryRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), strictsupport.Route(wrapper.GetPlatformConnectivityHistory, routes.handleConnectivityHistory))
	group.GET(networkcontract.ConnectivityReportRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), strictsupport.Route(wrapper.GetPlatformConnectivityReport, routes.handleConnectivityReport))
	group.GET(networkcontract.ConnectivityTraceRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), strictsupport.Route(wrapper.GetPlatformConnectivityTrace, routes.handleConnectivityTrace))
	group.GET(networkcontract.ConnectivityExportRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), strictsupport.Route(wrapper.GetPlatformConnectivityExport, routes.handleConnectivityExport))
	return nil
}

func (r routeRuntime) handleGet(ginCtx *gin.Context) {
	overview, err := r.service.Overview(ginCtx.Request.Context())
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	writeModuleConfigETag(ginCtx, overview.Version)
	httpx.WriteSuccess(ginCtx, http.StatusOK, toOverview(overview))
}

func (r routeRuntime) handlePut(ginCtx *gin.Context, bound *generated.PutPlatformNetworkOutboundJSONRequestBody) {
	expectedVersion, ok := r.bindIfMatch(ginCtx)
	if !ok {
		return
	}
	var request generated.PutPlatformNetworkOutboundJSONRequestBody
	if bound != nil {
		request = *bound
	} else {
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			r.badRequest(ginCtx)
			return
		}
	}
	overview, err := r.service.Update(ginCtx.Request.Context(), moduleapi.OutboundNetworkPolicy{Enabled: request.Enabled, HTTPProxy: request.HttpProxy, HTTPSProxy: request.HttpsProxy, NoProxy: request.NoProxy}, currentUserID(ginCtx), expectedVersion)
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	writeModuleConfigETag(ginCtx, overview.Version)
	httpx.WriteSuccess(ginCtx, http.StatusOK, toOverview(overview))
}

func (r routeRuntime) handleReset(ginCtx *gin.Context) {
	expectedVersion, ok := r.bindIfMatch(ginCtx)
	if !ok {
		return
	}
	overview, err := r.service.Reset(ginCtx.Request.Context(), currentUserID(ginCtx), expectedVersion)
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	writeModuleConfigETag(ginCtx, overview.Version)
	httpx.WriteSuccess(ginCtx, http.StatusOK, toOverview(overview))
}

func (r routeRuntime) badRequest(ginCtx *gin.Context) {
	httpx.AbortAppError(ginCtx, r.ctx.I18n, r.ctx.Logger, errInvalidOutboundPolicy)
}

func (r routeRuntime) writeError(ginCtx *gin.Context, err error) {
	if errors.Is(err, moduleapi.ErrModuleConfigVersionConflict) {
		if overview, overviewErr := r.service.Overview(ginCtx.Request.Context()); overviewErr == nil {
			writeModuleConfigETag(ginCtx, overview.Version)
		}
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusPreconditionFailed, messagecontract.ModuleConfigPreconditionFailed.String(), nil)
		return
	}
	if errors.Is(err, errCustomConnectivityTargetNotFound) {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusNotFound, "common.not_found", map[string]any{"resource": "custom connectivity target"})
		return
	}
	if errors.Is(err, errDiagnosticTargetNotFound) {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusNotFound, "common.not_found", map[string]any{"resource": "outbound diagnostic target"})
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusNotFound, "common.not_found", map[string]any{"resource": "connectivity report"})
		return
	}
	if errors.Is(err, errInvalidOutboundPolicy) {
		r.badRequest(ginCtx)
		return
	}
	httpx.AbortAppError(ginCtx, r.ctx.I18n, r.ctx.Logger, err)
}

func (r routeRuntime) bindIfMatch(ginCtx *gin.Context) (int64, bool) {
	raw := ginCtx.GetHeader(httpheader.IfMatch.String())
	if raw == "" {
		httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, http.StatusPreconditionRequired, errorcode.ModuleConfigPreconditionRequired.String(), messagecontract.ModuleConfigPreconditionRequired.String(), nil)
		ginCtx.Abort()
		return 0, false
	}
	if len(raw) < 3 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		r.badRequest(ginCtx)
		return 0, false
	}
	version, err := strconv.ParseInt(raw[1:len(raw)-1], 10, 64)
	if err != nil || version < 0 {
		r.badRequest(ginCtx)
		return 0, false
	}
	return version, true
}

func writeModuleConfigETag(ginCtx *gin.Context, version int64) {
	ginCtx.Header(httpheader.ETag.String(), strconv.Quote(strconv.FormatInt(version, 10)))
}

func toOverview(value Overview) generated.PlatformNetworkOverview {
	targets := make([]generated.PlatformNetworkDiagnosticTarget, 0, len(value.DiagnosticTargets))
	for _, target := range value.DiagnosticTargets {
		targets = append(targets, generated.PlatformNetworkDiagnosticTarget{Id: target.Name(), TitleKey: target.DisplayName()})
	}
	source := generated.PlatformNetworkOutboundPolicySourceDefault
	if value.HasOverride {
		source = generated.PlatformNetworkOutboundPolicySourceOverride
	}
	consumers := make([]generated.PlatformNetworkConsumer, 0, len(value.Consumers))
	for _, consumer := range value.Consumers {
		consumers = append(consumers, generated.PlatformNetworkConsumer{Id: consumer.Name(), TitleKey: consumer.DisplayName()})
	}
	policy := generated.PlatformNetworkOutboundPolicy{Config: generated.PlatformNetworkOutboundConfig{Enabled: value.Policy.Enabled, HttpProxy: value.Policy.HTTPProxy, HttpsProxy: value.Policy.HTTPSProxy, NoProxy: append([]string(nil), value.Policy.NoProxy...)}, Source: source}
	if value.UpdatedAt != nil {
		updatedAt := value.UpdatedAt.UTC()
		policy.UpdatedAt = &updatedAt
	}
	if name := strings.TrimSpace(value.UpdatedByName); name != "" {
		policy.UpdatedByName = &name
	}
	return generated.PlatformNetworkOverview{Policy: policy, DiagnosticTargets: targets, Consumers: consumers}
}

func currentUserID(ginCtx *gin.Context) *uint64 {
	if ginCtx == nil || ginCtx.Request == nil {
		return nil
	}
	auth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok || auth.User == nil {
		return nil
	}
	userID := auth.User.ID
	return &userID
}

// networkStrictServer 将现有 HTTP 响应写入接入生成的请求契约。
type networkStrictServer struct {
	routes routeRuntime
}

var _ networkopenapi.StrictServerInterface = networkStrictServer{}

func (s networkStrictServer) GetPlatformNetworkOutbound(ctx context.Context, _ networkopenapi.GetPlatformNetworkOutboundRequestObject) (networkopenapi.GetPlatformNetworkOutboundResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleGet)
}

func (s networkStrictServer) PutPlatformNetworkOutbound(ctx context.Context, req networkopenapi.PutPlatformNetworkOutboundRequestObject) (networkopenapi.PutPlatformNetworkOutboundResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.routes.handlePut(c, (*generated.PutPlatformNetworkOutboundJSONRequestBody)(req.Body))
	})
}

func (s networkStrictServer) ResetPlatformNetworkOutbound(ctx context.Context, _ networkopenapi.ResetPlatformNetworkOutboundRequestObject) (networkopenapi.ResetPlatformNetworkOutboundResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleReset)
}

func (s networkStrictServer) PostPlatformNetworkDiagnostic(ctx context.Context, _ networkopenapi.PostPlatformNetworkDiagnosticRequestObject) (networkopenapi.PostPlatformNetworkDiagnosticResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleLegacyDiagnostic)
}

func (s networkStrictServer) GetPlatformNetworkDiagnosticHistory(ctx context.Context, _ networkopenapi.GetPlatformNetworkDiagnosticHistoryRequestObject) (networkopenapi.GetPlatformNetworkDiagnosticHistoryResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleLegacyDiagnosticHistory)
}

func (s networkStrictServer) GetPlatformConnectivityTargets(ctx context.Context, _ networkopenapi.GetPlatformConnectivityTargetsRequestObject) (networkopenapi.GetPlatformConnectivityTargetsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleConnectivityTargets)
}

func (s networkStrictServer) GetPlatformConnectivityCustomTargets(ctx context.Context, _ networkopenapi.GetPlatformConnectivityCustomTargetsRequestObject) (networkopenapi.GetPlatformConnectivityCustomTargetsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleConnectivityCustomTargets)
}

func (s networkStrictServer) PostPlatformConnectivityCustomTarget(ctx context.Context, req networkopenapi.PostPlatformConnectivityCustomTargetRequestObject) (networkopenapi.PostPlatformConnectivityCustomTargetResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.routes.handleCreateConnectivityCustomTarget(c, req.Body) })
}

func (s networkStrictServer) DeletePlatformConnectivityCustomTarget(ctx context.Context, _ networkopenapi.DeletePlatformConnectivityCustomTargetRequestObject) (networkopenapi.DeletePlatformConnectivityCustomTargetResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDeleteConnectivityCustomTarget)
}

func (s networkStrictServer) GetPlatformConnectivityLatest(ctx context.Context, _ networkopenapi.GetPlatformConnectivityLatestRequestObject) (networkopenapi.GetPlatformConnectivityLatestResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleConnectivityLatest)
}

func (s networkStrictServer) GetPlatformConnectivityAggregate(ctx context.Context, _ networkopenapi.GetPlatformConnectivityAggregateRequestObject) (networkopenapi.GetPlatformConnectivityAggregateResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleConnectivityAggregate)
}

func (s networkStrictServer) PostPlatformConnectivityRun(ctx context.Context, _ networkopenapi.PostPlatformConnectivityRunRequestObject) (networkopenapi.PostPlatformConnectivityRunResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleConnectivityRun)
}

func (s networkStrictServer) PostPlatformConnectivityBatchRun(ctx context.Context, _ networkopenapi.PostPlatformConnectivityBatchRunRequestObject) (networkopenapi.PostPlatformConnectivityBatchRunResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleConnectivityBatchRun)
}

func (s networkStrictServer) GetPlatformConnectivityHistory(ctx context.Context, _ networkopenapi.GetPlatformConnectivityHistoryRequestObject) (networkopenapi.GetPlatformConnectivityHistoryResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleConnectivityHistory)
}

func (s networkStrictServer) GetPlatformConnectivityReport(ctx context.Context, _ networkopenapi.GetPlatformConnectivityReportRequestObject) (networkopenapi.GetPlatformConnectivityReportResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleConnectivityReport)
}

func (s networkStrictServer) GetPlatformConnectivityTrace(ctx context.Context, _ networkopenapi.GetPlatformConnectivityTraceRequestObject) (networkopenapi.GetPlatformConnectivityTraceResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleConnectivityTrace)
}

func (s networkStrictServer) GetPlatformConnectivityExport(ctx context.Context, _ networkopenapi.GetPlatformConnectivityExportRequestObject) (networkopenapi.GetPlatformConnectivityExportResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleConnectivityExport)
}
