// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package announcement

import (
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	announcementopenapi "graft/server/internal/contract/openapi/announcement"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	announcementcontract "graft/server/modules/announcement/contract"
	announcementstore "graft/server/modules/announcement/store"
)

type announcementGuards struct {
	authenticated gin.HandlerFunc
	read          gin.HandlerFunc
	create        gin.HandlerFunc
	update        gin.HandlerFunc
	publish       gin.HandlerFunc
	delete        gin.HandlerFunc
}

type announcementRouteRuntime struct {
	ctx     *module.Context
	service *Service
}

func registerAnnouncementRoutes(ctx *module.Context, service *Service, guards announcementGuards) error {
	if ctx == nil || ctx.Router == nil {
		return nil
	}
	if service == nil {
		return errors.New("announcement service is unavailable")
	}
	routes := announcementRouteRuntime{ctx: ctx, service: service}
	admin := ctx.Router.Group(announcementcontract.AnnouncementGroup)
	admin.Use(httpx.RequestIDMiddleware())
	admin.GET(announcementcontract.AnnouncementCollectionRoute, guards.read, routes.handleAdminList)
	admin.POST(announcementcontract.AnnouncementCollectionRoute, guards.create, routes.handleAdminCreate)
	admin.GET(announcementcontract.AnnouncementDetailRoute, guards.read, routes.handleAdminGet)
	admin.PUT(announcementcontract.AnnouncementDetailRoute, guards.update, routes.handleAdminUpdate)
	admin.POST(announcementcontract.AnnouncementPublishRoute, guards.publish, routes.handlePublish)
	admin.POST(announcementcontract.AnnouncementArchiveRoute, guards.publish, routes.handleArchive)
	admin.DELETE(announcementcontract.AnnouncementDetailRoute, guards.delete, routes.handleDelete)

	user := ctx.Router.Group(announcementcontract.MyAnnouncementGroup)
	user.Use(httpx.RequestIDMiddleware())
	user.GET(announcementcontract.MyAnnouncementCollectionRoute, guards.authenticated, routes.handleUserList)
	user.POST(announcementcontract.MyAnnouncementReadRoute, guards.authenticated, routes.handleMarkRead)
	user.POST(announcementcontract.MyAnnouncementReadAllRoute, guards.authenticated, routes.handleMarkAllRead)
	user.GET(announcementcontract.MyAnnouncementUnreadCountRoute, guards.authenticated, routes.handleUnreadCount)
	return nil
}

func (r announcementRouteRuntime) handleAdminList(ginCtx *gin.Context) {
	params, ok := bindAdminListParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	announcementGeneratedHandler{}.GetAnnouncements(params)
	result, err := r.service.ListAdmin(ginCtx.Request.Context(), AdminListQuery{
		Status:   stringFromPointer(params.Status),
		Level:    stringFromPointer(params.Level),
		Pinned:   params.Pinned,
		Keyword:  stringFromPointer(params.Keyword),
		Page:     intFromPointer(params.Page),
		PageSize: intFromPointer(params.PageSize),
		Sort:     stringFromPointer(params.Sort),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	response, err := toAnnouncementListResponse(result)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, response)
}

func (r announcementRouteRuntime) handleAdminCreate(ginCtx *gin.Context) {
	var request announcementopenapi.PostAnnouncementsJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	announcementGeneratedHandler{}.PostAnnouncements(bindCommonParams(ginCtx), request)
	item, err := r.service.Create(ginCtx.Request.Context(), announcementstore.CreateInput{
		Title:        request.Title,
		Content:      request.Content,
		Level:        string(request.Level),
		Status:       announcementcontract.AnnouncementStatusDraft.String(),
		DeliveryMode: string(request.DeliveryMode),
		Pinned:       boolFromPointer(request.Pinned),
		PublishAt:    request.PublishAt,
		ExpireAt:     request.ExpireAt,
		ActorID:      currentUserIDPointer(ginCtx),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	response, err := toAnnouncementItem(item)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusCreated, response)
}

func (r announcementRouteRuntime) handleAdminGet(ginCtx *gin.Context) {
	id, ok := bindAnnouncementID(ginCtx, r.ctx)
	if !ok {
		return
	}
	generatedID, ok := bindGeneratedAnnouncementID(ginCtx, r.ctx, id)
	if !ok {
		return
	}
	announcementGeneratedHandler{}.GetAnnouncement(generatedID, bindGetAnnouncementParams(ginCtx))
	item, err := r.service.GetAdmin(ginCtx.Request.Context(), id)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	response, err := toAnnouncementItem(item)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, response)
}

func (r announcementRouteRuntime) handleAdminUpdate(ginCtx *gin.Context) {
	id, ok := bindAnnouncementID(ginCtx, r.ctx)
	if !ok {
		return
	}
	var request announcementopenapi.PutAnnouncementJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	generatedID, ok := bindGeneratedAnnouncementID(ginCtx, r.ctx, id)
	if !ok {
		return
	}
	announcementGeneratedHandler{}.PutAnnouncement(generatedID, bindPutAnnouncementParams(ginCtx), request)
	item, err := r.service.Update(ginCtx.Request.Context(), id, announcementstore.UpdateInput{
		Title:        request.Title,
		Content:      request.Content,
		Level:        string(request.Level),
		DeliveryMode: string(request.DeliveryMode),
		Pinned:       boolFromPointer(request.Pinned),
		PublishAt:    request.PublishAt,
		ExpireAt:     request.ExpireAt,
		ActorID:      currentUserIDPointer(ginCtx),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	response, err := toAnnouncementItem(item)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, response)
}

func (r announcementRouteRuntime) handlePublish(ginCtx *gin.Context) {
	id, ok := bindAnnouncementID(ginCtx, r.ctx)
	if !ok {
		return
	}
	var request announcementopenapi.PostAnnouncementPublishJSONRequestBody
	if !bindOptionalJSON(ginCtx, r.ctx, &request) {
		return
	}
	generatedID, ok := bindGeneratedAnnouncementID(ginCtx, r.ctx, id)
	if !ok {
		return
	}
	announcementGeneratedHandler{}.PostAnnouncementPublish(generatedID, bindPostAnnouncementPublishParams(ginCtx), request)
	item, err := r.service.Publish(ginCtx.Request.Context(), id, request.PublishAt, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	response, err := toAnnouncementItem(item)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, response)
}

func (r announcementRouteRuntime) handleArchive(ginCtx *gin.Context) {
	id, ok := bindAnnouncementID(ginCtx, r.ctx)
	if !ok {
		return
	}
	generatedID, ok := bindGeneratedAnnouncementID(ginCtx, r.ctx, id)
	if !ok {
		return
	}
	announcementGeneratedHandler{}.PostAnnouncementArchive(generatedID, bindPostAnnouncementArchiveParams(ginCtx))
	item, err := r.service.Archive(ginCtx.Request.Context(), id, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	response, err := toAnnouncementItem(item)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, response)
}

func (r announcementRouteRuntime) handleDelete(ginCtx *gin.Context) {
	id, ok := bindAnnouncementID(ginCtx, r.ctx)
	if !ok {
		return
	}
	generatedID, ok := bindGeneratedAnnouncementID(ginCtx, r.ctx, id)
	if !ok {
		return
	}
	announcementGeneratedHandler{}.DeleteAnnouncement(generatedID, bindDeleteAnnouncementParams(ginCtx))
	actorID := uint64(0)
	if current := currentUserIDPointer(ginCtx); current != nil {
		actorID = *current
	}
	if err := r.service.Delete(ginCtx.Request.Context(), id, actorID); err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, emptyResponse{})
}

func (r announcementRouteRuntime) handleUserList(ginCtx *gin.Context) {
	userID, ok := currentUserID(ginCtx, r.ctx)
	if !ok {
		return
	}
	params, ok := bindUserListParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	announcementGeneratedHandler{}.GetMyAnnouncements(params)
	result, err := r.service.ListCurrentUser(ginCtx.Request.Context(), UserListQuery{
		UserID:     userID,
		UnreadOnly: boolFromPointer(params.UnreadOnly),
		Page:       intFromPointer(params.Page),
		PageSize:   intFromPointer(params.PageSize),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	response, err := toMyAnnouncementListResponse(result)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, response)
}

func (r announcementRouteRuntime) handleMarkRead(ginCtx *gin.Context) {
	userID, ok := currentUserID(ginCtx, r.ctx)
	if !ok {
		return
	}
	id, ok := bindAnnouncementID(ginCtx, r.ctx)
	if !ok {
		return
	}
	generatedID, ok := bindGeneratedAnnouncementID(ginCtx, r.ctx, id)
	if !ok {
		return
	}
	announcementGeneratedHandler{}.PostMyAnnouncementRead(generatedID, bindPostMyAnnouncementReadParams(ginCtx))
	item, err := r.service.MarkRead(ginCtx.Request.Context(), userID, id)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	response, err := toMyAnnouncementItem(item)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, response)
}

func (r announcementRouteRuntime) handleMarkAllRead(ginCtx *gin.Context) {
	userID, ok := currentUserID(ginCtx, r.ctx)
	if !ok {
		return
	}
	announcementGeneratedHandler{}.PostMyAnnouncementsReadAll(bindPostMyAnnouncementsReadAllParams(ginCtx))
	count, err := r.service.MarkAllRead(ginCtx.Request.Context(), userID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, announcementReadAllResponse{UpdatedCount: count})
}

func (r announcementRouteRuntime) handleUnreadCount(ginCtx *gin.Context) {
	userID, ok := currentUserID(ginCtx, r.ctx)
	if !ok {
		return
	}
	announcementGeneratedHandler{}.GetMyAnnouncementsUnreadCount(bindGetMyAnnouncementsUnreadCountParams(ginCtx))
	count, err := r.service.UnreadCount(ginCtx.Request.Context(), userID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, announcementUnreadCountResponse{Count: count})
}

func (r announcementRouteRuntime) writeRouteError(ginCtx *gin.Context, err error) {
	if err == nil {
		httpx.WriteSuccess(ginCtx, http.StatusOK, emptyResponse{})
		return
	}
	if errors.Is(err, errAnnouncementInvalidInput) {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	if errors.Is(err, errAnnouncementNotFound) {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusNotFound, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	if errors.Is(err, errAnnouncementPublishedDelete) {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusConflict, announcementcontract.AnnouncementPublishedDeleteForbidden.String(), nil)
		return
	}
	if errors.Is(err, errAnnouncementInvalidTransition) {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusConflict, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	if r.ctx.Logger != nil {
		r.ctx.Logger.Error("announcement route failed", zap.String("module", moduleID), zap.Error(err))
	}
	httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
}

type announcementGeneratedHandler struct{}

func (announcementGeneratedHandler) GetAnnouncements(announcementopenapi.GetAnnouncementsParams) {}

func (announcementGeneratedHandler) PostAnnouncements(announcementopenapi.PostAnnouncementsParams, announcementopenapi.PostAnnouncementsJSONRequestBody) {
}

func (announcementGeneratedHandler) GetAnnouncement(int64, announcementopenapi.GetAnnouncementParams) {
}

func (announcementGeneratedHandler) PutAnnouncement(int64, announcementopenapi.PutAnnouncementParams, announcementopenapi.PutAnnouncementJSONRequestBody) {
}

func (announcementGeneratedHandler) PostAnnouncementPublish(int64, announcementopenapi.PostAnnouncementPublishParams, announcementopenapi.PostAnnouncementPublishJSONRequestBody) {
}

func (announcementGeneratedHandler) PostAnnouncementArchive(int64, announcementopenapi.PostAnnouncementArchiveParams) {
}

func (announcementGeneratedHandler) DeleteAnnouncement(int64, announcementopenapi.DeleteAnnouncementParams) {
}

func (announcementGeneratedHandler) GetMyAnnouncements(announcementopenapi.GetMyAnnouncementsParams) {
}

func (announcementGeneratedHandler) PostMyAnnouncementRead(int64, announcementopenapi.PostMyAnnouncementReadParams) {
}

func (announcementGeneratedHandler) PostMyAnnouncementsReadAll(announcementopenapi.PostMyAnnouncementsReadAllParams) {
}

func (announcementGeneratedHandler) GetMyAnnouncementsUnreadCount(announcementopenapi.GetMyAnnouncementsUnreadCountParams) {
}

func bindAdminListParams(ginCtx *gin.Context, ctx *module.Context) (announcementopenapi.GetAnnouncementsParams, bool) {
	locale, requestID := commonHeaders(ginCtx)
	query := ginCtx.Request.URL.Query()
	params := announcementopenapi.GetAnnouncementsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
		Status:       optionalTypedQuery[announcementopenapi.GetAnnouncementsParamsStatus](query.Get("status")),
		Level:        optionalTypedQuery[announcementopenapi.GetAnnouncementsParamsLevel](query.Get("level")),
		Keyword:      optionalQuery(query.Get("keyword")),
		Sort:         optionalTypedQuery[announcementopenapi.GetAnnouncementsParamsSort](query.Get("sort")),
	}
	var ok bool
	if params.Pinned, ok = optionalBoolQuery(query.Get("pinned")); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return announcementopenapi.GetAnnouncementsParams{}, false
	}
	if params.Page, ok = optionalIntQuery(query.Get("page"), 1, 0); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return announcementopenapi.GetAnnouncementsParams{}, false
	}
	if params.PageSize, ok = optionalIntQuery(query.Get("page_size"), 1, maxPageSize); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return announcementopenapi.GetAnnouncementsParams{}, false
	}
	return params, true
}

func bindUserListParams(ginCtx *gin.Context, ctx *module.Context) (announcementopenapi.GetMyAnnouncementsParams, bool) {
	locale, requestID := commonHeaders(ginCtx)
	query := ginCtx.Request.URL.Query()
	params := announcementopenapi.GetMyAnnouncementsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
	var ok bool
	if params.UnreadOnly, ok = optionalBoolQuery(query.Get("unread_only")); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return announcementopenapi.GetMyAnnouncementsParams{}, false
	}
	if params.Page, ok = optionalIntQuery(query.Get("page"), 1, 0); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return announcementopenapi.GetMyAnnouncementsParams{}, false
	}
	if params.PageSize, ok = optionalIntQuery(query.Get("page_size"), 1, maxPageSize); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return announcementopenapi.GetMyAnnouncementsParams{}, false
	}
	return params, true
}

func bindCommonParams(ginCtx *gin.Context) announcementopenapi.PostAnnouncementsParams {
	locale, requestID := commonHeaders(ginCtx)
	return announcementopenapi.PostAnnouncementsParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGetAnnouncementParams(ginCtx *gin.Context) announcementopenapi.GetAnnouncementParams {
	locale, requestID := commonHeaders(ginCtx)
	return announcementopenapi.GetAnnouncementParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPutAnnouncementParams(ginCtx *gin.Context) announcementopenapi.PutAnnouncementParams {
	locale, requestID := commonHeaders(ginCtx)
	return announcementopenapi.PutAnnouncementParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostAnnouncementPublishParams(ginCtx *gin.Context) announcementopenapi.PostAnnouncementPublishParams {
	locale, requestID := commonHeaders(ginCtx)
	return announcementopenapi.PostAnnouncementPublishParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostAnnouncementArchiveParams(ginCtx *gin.Context) announcementopenapi.PostAnnouncementArchiveParams {
	locale, requestID := commonHeaders(ginCtx)
	return announcementopenapi.PostAnnouncementArchiveParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindDeleteAnnouncementParams(ginCtx *gin.Context) announcementopenapi.DeleteAnnouncementParams {
	locale, requestID := commonHeaders(ginCtx)
	return announcementopenapi.DeleteAnnouncementParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostMyAnnouncementReadParams(ginCtx *gin.Context) announcementopenapi.PostMyAnnouncementReadParams {
	locale, requestID := commonHeaders(ginCtx)
	return announcementopenapi.PostMyAnnouncementReadParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostMyAnnouncementsReadAllParams(ginCtx *gin.Context) announcementopenapi.PostMyAnnouncementsReadAllParams {
	locale, requestID := commonHeaders(ginCtx)
	return announcementopenapi.PostMyAnnouncementsReadAllParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGetMyAnnouncementsUnreadCountParams(ginCtx *gin.Context) announcementopenapi.GetMyAnnouncementsUnreadCountParams {
	locale, requestID := commonHeaders(ginCtx)
	return announcementopenapi.GetMyAnnouncementsUnreadCountParams{XGraftLocale: locale, XRequestId: requestID}
}

func commonHeaders(ginCtx *gin.Context) (*string, *string) {
	locale := ginCtx.GetHeader(string(httpheader.Locale))
	requestID := httpx.EnsureRequestID(ginCtx)
	return &locale, &requestID
}

func bindAnnouncementID(ginCtx *gin.Context, ctx *module.Context) (uint64, bool) {
	raw := strings.TrimSpace(ginCtx.Param("id"))
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return 0, false
	}
	return value, true
}

func bindGeneratedAnnouncementID(ginCtx *gin.Context, ctx *module.Context, value uint64) (int64, bool) {
	if value > math.MaxInt64 {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return 0, false
	}
	return int64(value), true
}

func currentUserID(ginCtx *gin.Context, ctx *module.Context) (uint64, bool) {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok || requestAuth.User == nil || requestAuth.User.ID == 0 {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return 0, false
	}
	return requestAuth.User.ID, true
}

func currentUserIDPointer(ginCtx *gin.Context) *uint64 {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok || requestAuth.User == nil || requestAuth.User.ID == 0 {
		return nil
	}
	userID := requestAuth.User.ID
	return &userID
}

func bindJSON[T any](ginCtx *gin.Context, ctx *module.Context, target *T) bool {
	if err := ginCtx.ShouldBindJSON(target); err != nil {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return false
	}
	return true
}

func bindOptionalJSON[T any](ginCtx *gin.Context, ctx *module.Context, target *T) bool {
	if ginCtx.Request == nil || ginCtx.Request.Body == nil {
		return true
	}
	if err := ginCtx.ShouldBindJSON(target); err != nil && !errors.Is(err, io.EOF) {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return false
	}
	return true
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

func stringFromPointer[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func optionalIntQuery(raw string, min int, max int) (*int, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, true
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < min || (max > 0 && parsed > max) {
		return nil, false
	}
	return &parsed, true
}

func optionalBoolQuery(raw string) (*bool, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, true
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func abortInvalidQuery(ginCtx *gin.Context, ctx *module.Context) {
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
}

func intFromPointer(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func boolFromPointer(value *bool) bool {
	return value != nil && *value
}
