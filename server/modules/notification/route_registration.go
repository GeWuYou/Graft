package notification

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	notificationopenapi "graft/server/internal/contract/openapi/notification"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

type notificationGuards struct {
	view gin.HandlerFunc
	read gin.HandlerFunc
}

type notificationRouteRuntime struct {
	ctx     *module.Context
	service *Service
}

func registerNotificationRoutes(ctx *module.Context, service *Service, guards notificationGuards) {
	runtime := notificationRouteRuntime{ctx: ctx, service: service}
	group := ctx.Router.Group("/notifications")
	group.Use(httpx.RequestIDMiddleware())
	group.GET("", guards.view, runtime.handleList)
	group.GET("/unread-count", guards.view, runtime.handleUnreadCount)
	group.POST("/:delivery_id/read", guards.read, runtime.handleRead)
	group.POST("/read-all", guards.read, runtime.handleReadAll)
	group.DELETE("/:delivery_id", guards.read, runtime.handleDelete)
}

func (r notificationRouteRuntime) handleList(ginCtx *gin.Context) {
	userID, ok := currentUserID(ginCtx, r.ctx)
	if !ok {
		return
	}
	params, ok := bindNotificationListParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	result, err := r.service.List(ginCtx.Request.Context(), ListQuery{
		RecipientUserID: userID,
		Status:          stringFromPointer(params.Status),
		Severity:        stringFromPointer(params.Severity),
		Category:        stringFromPointer(params.Category),
		SourceModule:    stringFromPointer(params.SourceModule),
		OccurredFrom:    params.OccurredFrom,
		OccurredTo:      params.OccurredTo,
		Page:            intFromPointer(params.Page),
		PageSize:        intFromPointer(params.PageSize),
	})
	if err != nil {
		r.writeServiceError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toNotificationListResponse(result))
}

func (r notificationRouteRuntime) handleUnreadCount(ginCtx *gin.Context) {
	userID, ok := currentUserID(ginCtx, r.ctx)
	if !ok {
		return
	}
	count, err := r.service.UnreadCount(ginCtx.Request.Context(), userID)
	if err != nil {
		r.writeServiceError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toNotificationUnreadCountResponse(count))
}

func (r notificationRouteRuntime) handleRead(ginCtx *gin.Context) {
	userID, ok := currentUserID(ginCtx, r.ctx)
	if !ok {
		return
	}
	deliveryID, ok := bindDeliveryID(ginCtx, r.ctx)
	if !ok {
		return
	}
	delivery, err := r.service.MarkRead(ginCtx.Request.Context(), userID, deliveryID, time.Now().UTC())
	if err != nil {
		r.writeServiceError(ginCtx, err)
		return
	}
	item, err := r.service.Get(ginCtx.Request.Context(), userID, delivery.ID)
	if err != nil {
		r.writeServiceError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toNotificationItem(item))
}

func (r notificationRouteRuntime) handleReadAll(ginCtx *gin.Context) {
	userID, ok := currentUserID(ginCtx, r.ctx)
	if !ok {
		return
	}
	var body notificationopenapi.PostNotificationsReadAllJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&body); err != nil && !errors.Is(err, io.EOF) {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	query := readAllQueryFromBody(body)
	query.RecipientUserID = userID
	count, err := r.service.MarkAllReadMatching(ginCtx.Request.Context(), query, time.Now().UTC())
	if err != nil {
		r.writeServiceError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, map[string]int{"updated_count": count})
}

func (r notificationRouteRuntime) handleDelete(ginCtx *gin.Context) {
	userID, ok := currentUserID(ginCtx, r.ctx)
	if !ok {
		return
	}
	deliveryID, ok := bindDeliveryID(ginCtx, r.ctx)
	if !ok {
		return
	}
	if err := r.service.DeleteDelivery(ginCtx.Request.Context(), userID, deliveryID, time.Now().UTC()); err != nil {
		r.writeServiceError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, map[string]any{})
}

func bindNotificationListParams(ginCtx *gin.Context, ctx *module.Context) (notificationopenapi.GetNotificationsParams, bool) {
	query := ginCtx.Request.URL.Query()
	params := notificationopenapi.GetNotificationsParams{
		Status:       optionalTypedQuery[notificationopenapi.GetNotificationsParamsStatus](query.Get("status")),
		Severity:     optionalTypedQuery[notificationopenapi.GetNotificationsParamsSeverity](query.Get("severity")),
		Category:     optionalTypedQuery[notificationopenapi.GetNotificationsParamsCategory](query.Get("category")),
		SourceModule: optionalQuery(query.Get("source_module")),
	}
	var err error
	if params.Page, err = optionalIntQuery(query.Get("page")); err != nil {
		abortInvalidQuery(ginCtx, ctx, "page", err)
		return notificationopenapi.GetNotificationsParams{}, false
	}
	if params.PageSize, err = optionalIntQuery(query.Get("page_size")); err != nil {
		abortInvalidQuery(ginCtx, ctx, "page_size", err)
		return notificationopenapi.GetNotificationsParams{}, false
	}
	if params.OccurredFrom, err = optionalTimeQuery(query.Get("occurred_from")); err != nil {
		abortInvalidQuery(ginCtx, ctx, "occurred_from", err)
		return notificationopenapi.GetNotificationsParams{}, false
	}
	if params.OccurredTo, err = optionalTimeQuery(query.Get("occurred_to")); err != nil {
		abortInvalidQuery(ginCtx, ctx, "occurred_to", err)
		return notificationopenapi.GetNotificationsParams{}, false
	}
	return params, true
}

func currentUserID(ginCtx *gin.Context, ctx *module.Context) (uint64, bool) {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok || requestAuth.User == nil || requestAuth.User.ID == 0 {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return 0, false
	}
	return requestAuth.User.ID, true
}

func bindDeliveryID(ginCtx *gin.Context, ctx *module.Context) (uint64, bool) {
	raw := strings.TrimSpace(ginCtx.Param("delivery_id"))
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return 0, false
	}
	return value, true
}

func (r notificationRouteRuntime) writeServiceError(ginCtx *gin.Context, err error) {
	switch {
	case errors.Is(err, moduleapi.ErrNotificationInvalidInput):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
	case errors.Is(err, moduleapi.ErrNotificationDeliveryNotFound):
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusNotFound, messagecontract.CommonInvalidArgument.String(), nil)
	default:
		if r.ctx.Logger != nil {
			r.ctx.Logger.Error("notification route failed", zap.String("module", moduleID), zap.Error(err))
		}
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
	}
}

func optionalQuery(raw string) *string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	return &value
}

func optionalTypedQuery[T ~string](raw string) *T {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	typed := T(value)
	return &typed
}

func optionalIntQuery(raw string) (*int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func optionalTimeQuery(raw string) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func abortInvalidQuery(ginCtx *gin.Context, ctx *module.Context, name string, err error) {
	if ctx.Logger != nil {
		ctx.Logger.Warn("invalid notification query parameter",
			zap.String("param", name),
			zap.Error(err),
		)
	}
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
}

func intFromPointer(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
