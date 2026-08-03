package network

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/contract/httpheader"
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
	group.POST(networkcontract.OutboundNetworkDiagnosticRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkDiagnosePermission.String(), publisher), routes.handleDiagnostic)
	group.GET(networkcontract.OutboundNetworkDiagnosticHistoryRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, networkcontract.NetworkReadPermission.String(), publisher), routes.handleDiagnosticHistory)
	return nil
}

func (r routeRuntime) handleDiagnosticHistory(ginCtx *gin.Context) {
	targetName := strings.TrimSpace(ginCtx.Param("targetId"))
	params, limit, ok := bindDiagnosticHistoryParams(ginCtx)
	if !ok {
		r.badRequest(ginCtx)
		return
	}
	networkGeneratedHandler{}.GetPlatformNetworkDiagnosticHistory(targetName, params)
	items, err := r.service.DiagnosticHistory(ginCtx.Request.Context(), targetName, limit)
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	results := make([]generated.PlatformNetworkDiagnosticResult, 0, len(items))
	for _, item := range items {
		results = append(results, toDiagnosticResult(targetName, item))
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, generated.PlatformNetworkDiagnosticHistory{TargetId: targetName, Items: results})
}

func (r routeRuntime) handleGet(ginCtx *gin.Context) {
	networkGeneratedHandler{}.GetPlatformNetworkOutbound(bindGetParams(ginCtx))
	overview, err := r.service.Overview(ginCtx.Request.Context())
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toOverview(overview))
}

func (r routeRuntime) handlePut(ginCtx *gin.Context) {
	var request generated.PutPlatformNetworkOutboundJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		r.badRequest(ginCtx)
		return
	}
	networkGeneratedHandler{}.PutPlatformNetworkOutbound(bindPutParams(ginCtx), request)
	overview, err := r.service.Update(ginCtx.Request.Context(), moduleapi.OutboundNetworkPolicy{Enabled: request.Enabled, HTTPProxy: request.HttpProxy, HTTPSProxy: request.HttpsProxy, NoProxy: request.NoProxy}, currentUserID(ginCtx))
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toOverview(overview))
}

func (r routeRuntime) handleReset(ginCtx *gin.Context) {
	networkGeneratedHandler{}.ResetPlatformNetworkOutbound(bindResetParams(ginCtx))
	overview, err := r.service.Reset(ginCtx.Request.Context())
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toOverview(overview))
}

func (r routeRuntime) handleDiagnostic(ginCtx *gin.Context) {
	targetName := strings.TrimSpace(ginCtx.Param("targetId"))
	networkGeneratedHandler{}.PostPlatformNetworkDiagnostic(targetName, bindDiagnosticParams(ginCtx))
	result, err := r.service.Diagnose(ginCtx.Request.Context(), targetName)
	if err != nil {
		r.writeError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toDiagnosticResult(targetName, result))
}

func (r routeRuntime) badRequest(ginCtx *gin.Context) {
	httpx.AbortAppError(ginCtx, r.ctx.I18n, r.ctx.Logger, errInvalidOutboundPolicy)
}

func (r routeRuntime) writeError(ginCtx *gin.Context, err error) {
	if errors.Is(err, errDiagnosticTargetNotFound) {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusNotFound, "common.not_found", map[string]any{"resource": "outbound diagnostic target"})
		return
	}
	if errors.Is(err, errInvalidOutboundPolicy) {
		r.badRequest(ginCtx)
		return
	}
	httpx.AbortAppError(ginCtx, r.ctx.I18n, r.ctx.Logger, err)
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

func toDiagnosticResult(targetName string, value moduleapi.OutboundDiagnosticResult) generated.PlatformNetworkDiagnosticResult {
	status := generated.PlatformNetworkDiagnosticResultStatusFailed
	if value.Connected {
		status = generated.PlatformNetworkDiagnosticResultStatusConnected
	}
	result := generated.PlatformNetworkDiagnosticResult{TargetId: targetName, Status: status, TestedAt: value.TestedAt.UTC()}
	if value.Latency >= 0 {
		milliseconds := value.Latency.Milliseconds()
		result.LatencyMs = &milliseconds
	}
	if value.HTTPStatus >= http.StatusContinue && value.HTTPStatus <= 599 {
		result.HttpStatus = &value.HTTPStatus
	}
	if message := strings.TrimSpace(value.Message); message != "" {
		result.Error = &message
	}
	return result
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
func (networkGeneratedHandler) PostPlatformNetworkDiagnostic(string, generated.PostPlatformNetworkDiagnosticParams) {
}
func (networkGeneratedHandler) GetPlatformNetworkDiagnosticHistory(string, generated.GetPlatformNetworkDiagnosticHistoryParams) {
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

func bindDiagnosticParams(ginCtx *gin.Context) generated.PostPlatformNetworkDiagnosticParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostPlatformNetworkDiagnosticParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindDiagnosticHistoryParams(ginCtx *gin.Context) (generated.GetPlatformNetworkDiagnosticHistoryParams, int, bool) {
	locale, requestID := commonHeaders(ginCtx)
	params := generated.GetPlatformNetworkDiagnosticHistoryParams{XGraftLocale: locale, XRequestId: requestID}
	limit := 20
	if raw := strings.TrimSpace(ginCtx.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxDiagnosticHistoryLimit {
			return generated.GetPlatformNetworkDiagnosticHistoryParams{}, 0, false
		}
		limit = parsed
		params.Limit = &limit
	}
	return params, limit, true
}

func commonHeaders(ginCtx *gin.Context) (*string, *string) {
	locale := ginCtx.GetHeader(string(httpheader.Locale))
	requestID := httpx.EnsureRequestID(ginCtx)
	return &locale, &requestID
}
