package network

import (
	"database/sql"
	"errors"
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
	group.GET(networkcontract.OutboundNetworkRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), routes.handleGet)
	group.PUT(networkcontract.OutboundNetworkRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkWritePermission.String(), publisher), routes.handlePut)
	group.POST(networkcontract.OutboundNetworkResetRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkWritePermission.String(), publisher), routes.handleReset)
	group.POST(networkcontract.LegacyDiagnosticRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkDiagnosePermission.String(), publisher), routes.handleLegacyDiagnostic)
	group.GET(networkcontract.LegacyDiagnosticHistoryRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), routes.handleLegacyDiagnosticHistory)
	group.GET(networkcontract.ConnectivityTargetsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), routes.handleConnectivityTargets)
	group.GET(networkcontract.ConnectivityCustomTargetsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkManageTargetsPermission.String(), publisher), routes.handleConnectivityCustomTargets)
	group.POST(networkcontract.ConnectivityCustomTargetsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkManageTargetsPermission.String(), publisher), routes.handleCreateConnectivityCustomTarget)
	group.DELETE(networkcontract.ConnectivityCustomTargetRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkManageTargetsPermission.String(), publisher), routes.handleDeleteConnectivityCustomTarget)
	group.GET(networkcontract.ConnectivityLatestRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), routes.handleConnectivityLatest)
	group.GET(networkcontract.ConnectivityAggregateRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), routes.handleConnectivityAggregate)
	group.POST(networkcontract.ConnectivityRunRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkDiagnosePermission.String(), publisher), routes.handleConnectivityRun)
	group.POST(networkcontract.ConnectivityBatchRunRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkDiagnosePermission.String(), publisher), routes.handleConnectivityBatchRun)
	group.GET(networkcontract.ConnectivityHistoryRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), routes.handleConnectivityHistory)
	group.GET(networkcontract.ConnectivityReportRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), routes.handleConnectivityReport)
	group.GET(networkcontract.ConnectivityTraceRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), routes.handleConnectivityTrace)
	group.GET(networkcontract.ConnectivityExportRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), routes.handleConnectivityExport)
	return nil
}

func (r routeRuntime) handleGet(ginCtx *gin.Context) {
	networkGeneratedHandler{}.GetPlatformNetworkOutbound(bindGetParams(ginCtx))
	overview, err := r.service.Overview(ginCtx.Request.Context())
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	writeModuleConfigETag(ginCtx, overview.Version)
	httpx.WriteSuccess(ginCtx, http.StatusOK, toOverview(overview))
}

func (r routeRuntime) handlePut(ginCtx *gin.Context) {
	expectedVersion, ok := r.bindIfMatch(ginCtx)
	if !ok {
		return
	}
	var request generated.PutPlatformNetworkOutboundJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		r.badRequest(ginCtx)
		return
	}
	networkGeneratedHandler{}.PutPlatformNetworkOutbound(bindPutParams(ginCtx), request)
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
	networkGeneratedHandler{}.ResetPlatformNetworkOutbound(bindResetParams(ginCtx))
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

type networkGeneratedHandler struct{}

func (networkGeneratedHandler) GetPlatformNetworkOutbound(generated.GetPlatformNetworkOutboundParams) {
}
func (networkGeneratedHandler) PutPlatformNetworkOutbound(generated.PutPlatformNetworkOutboundParams, generated.PutPlatformNetworkOutboundJSONRequestBody) {
}
func (networkGeneratedHandler) ResetPlatformNetworkOutbound(generated.ResetPlatformNetworkOutboundParams) {
}

func bindGetParams(ginCtx *gin.Context) generated.GetPlatformNetworkOutboundParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetPlatformNetworkOutboundParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPutParams(ginCtx *gin.Context) generated.PutPlatformNetworkOutboundParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PutPlatformNetworkOutboundParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindResetParams(ginCtx *gin.Context) generated.ResetPlatformNetworkOutboundParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.ResetPlatformNetworkOutboundParams{XGraftLocale: locale, XRequestId: requestID}
}

func commonHeaders(ginCtx *gin.Context) (*string, *string) {
	locale := ginCtx.GetHeader(string(httpheader.Locale))
	requestID := httpx.EnsureRequestID(ginCtx)
	return &locale, &requestID
}
