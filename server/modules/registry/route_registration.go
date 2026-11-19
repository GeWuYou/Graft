package registry

import (
	"errors"
	"fmt"
	registryopenapi "graft/server/internal/contract/openapi/registry"
	"graft/server/internal/contract/openapi/strictsupport"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	openapigen "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/httpx"
	"graft/server/internal/logger"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	buildcontract "graft/server/modules/build/contract"
	registrycontract "graft/server/modules/registry/contract"
	registrystore "graft/server/modules/registry/store"
)

func registerRegistryRoutes(ctx *module.Context, service *Service) error {
	auth, err := module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
	if err != nil {
		return err
	}
	authorizer, err := module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
	if err != nil {
		return err
	}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	group := ctx.Router.Group("/registries")
	group.Use(httpx.RequestIDMiddleware())
	routes := registryStrictServer{module: ctx, service: service}
	wrapper := registryopenapi.ServerInterfaceWrapper{
		Handler: registryopenapi.NewStrictHandlerWithOptions(routes, nil, registryopenapi.StrictGinServerOptions{
			RequestErrorHandlerFunc: strictsupport.RequestErrorHandler(ctx.I18n, "common.invalid_argument", nil),
			HandlerErrorFunc:        strictsupport.HandlerErrorHandler(ctx.I18n, ctx.Logger),
		}),
		ErrorHandler: strictsupport.BindingErrorHandler(ctx.I18n, "common.invalid_argument"),
	}
	group.GET("/available-destinations", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildCreatePermission, publisher), strictsupport.Route(wrapper.GetRegistryAvailableDestinations, routes.handleAvailableDestinations))
	group.GET("", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.ReadPermission, publisher), strictsupport.Route(wrapper.GetRegistries, routes.handleListConnections))
	group.POST("", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.CreatePermission, publisher), strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PostRegistry, routes.handleCreateConnection))
	group.GET("/:connectionRef", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.ReadPermission, publisher), strictsupport.Route(wrapper.GetRegistry, routes.handleGetConnection))
	group.PUT("/:connectionRef", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.UpdatePermission, publisher), strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PutRegistry, func(c *gin.Context) { routes.handleUpdateConnection(c, nil) }))
	group.DELETE("/:connectionRef", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.DeletePermission, publisher), strictsupport.Route(wrapper.DeleteRegistry, routes.handleDeleteConnection))
	group.POST("/:connectionRef/verify", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.VerifyPermission, publisher), strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PostRegistryVerify, func(c *gin.Context) { routes.handleVerifyConnection(c, nil) }))
	group.GET("/:connectionRef/repositories", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.ReadPermission, publisher), strictsupport.Route(wrapper.GetRegistryArtifactRepositories, routes.handleListRepositories))
	group.POST("/:connectionRef/repositories", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.CreatePermission, publisher), strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PostRegistryArtifactRepository, func(c *gin.Context) { routes.handleCreateRepository(c, nil) }))
	group.PUT("/:connectionRef/repositories", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.UpdatePermission, publisher), strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PutRegistryArtifactRepository, func(c *gin.Context) { routes.handleUpdateRepository(c, nil) }))
	group.DELETE("/:connectionRef/repositories", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.DeletePermission, publisher), strictsupport.Route(wrapper.DeleteRegistryArtifactRepository, routes.handleDeleteRepository))
	group.GET("/:connectionRef/repository-assignment-candidates", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.AssignmentManagePermission, publisher), strictsupport.Route(wrapper.GetRegistryRepositoryAssignmentCandidates, routes.handleListAssignmentCandidates))
	group.GET("/:connectionRef/repository-assignments", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.AssignmentManagePermission, publisher), strictsupport.Route(wrapper.GetRegistryArtifactRepositoryAssignments, routes.handleListAssignments))
	group.POST("/:connectionRef/repository-assignments", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.AssignmentManagePermission, publisher), strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PostRegistryArtifactRepositoryAssignment, func(c *gin.Context) { routes.handleGrantAssignment(c, nil) }))
	group.PUT("/:connectionRef/repository-assignments", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.AssignmentManagePermission, publisher), strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PutRegistryArtifactRepositoryAssignments, func(c *gin.Context) { routes.handleReplaceAssignments(c, nil) }))
	group.POST("/:connectionRef/repository-assignments/batch", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.AssignmentManagePermission, publisher), strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PostRegistryArtifactRepositoryAssignmentsBatchAdd, func(c *gin.Context) { routes.handleAddAssignments(c, nil) }))
	group.POST("/:connectionRef/repository-assignments/batch-revoke", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.AssignmentManagePermission, publisher), strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PostRegistryArtifactRepositoryAssignmentsBatchRevoke, func(c *gin.Context) { routes.handleRevokeAssignments(c, nil) }))
	group.DELETE("/:connectionRef/repository-assignments/:userId", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.AssignmentManagePermission, publisher), strictsupport.Route(wrapper.DeleteRegistryArtifactRepositoryAssignment, routes.handleRevokeAssignment))
	return nil
}

func (s registryStrictServer) handleListConnections(c *gin.Context) {
	ctx, service := s.module, s.service
	limit, offset := parsePage(c)
	items, total, err := service.ListConnections(c, c.Query("search"), limit, offset)
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, openapigen.RegistryConnectionListResponse{Items: mapConnections(items), Limit: limit, Offset: offset, Total: int64(total)})
}
func (s registryStrictServer) handleGetConnection(c *gin.Context) {
	ctx, service := s.module, s.service
	item, err := service.GetConnection(c, c.Param("connectionRef"))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, mapConnection(item))
}
func (s registryStrictServer) handleCreateConnection(c *gin.Context) {
	ctx, service := s.module, s.service
	var request openapigen.PostRegistryJSONRequestBody
	if c.ShouldBindJSON(&request) != nil {
		invalidRegistryRequest(c, ctx)
		return
	}
	item, err := service.CreateConnection(c, connectionInputFromCreate(request), registryActorID(c))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusCreated, mapConnection(item))
}
func (s registryStrictServer) handleUpdateConnection(c *gin.Context, bound *openapigen.PutRegistryJSONRequestBody) {
	ctx, service := s.module, s.service
	existing, err := service.GetConnection(c, c.Param("connectionRef"))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	var request openapigen.PutRegistryJSONRequestBody
	if bound != nil {
		request = *bound
	}
	if bound == nil && c.ShouldBindJSON(&request) != nil {
		invalidRegistryRequest(c, ctx)
		return
	}
	input := connectionInputFromUpdate(request, existing)
	item, err := service.UpdateConnection(c, existing.ConnectionRef, input, registryActorID(c))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, mapConnection(item))
}
func (s registryStrictServer) handleDeleteConnection(c *gin.Context) {
	ctx, service := s.module, s.service
	if err := service.DeleteConnection(c, c.Param("connectionRef"), registryActorID(c)); err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess[any](c, http.StatusOK, nil)
}
func (s registryStrictServer) handleVerifyConnection(c *gin.Context, bound *openapigen.PostRegistryVerifyJSONRequestBody) {
	ctx, service := s.module, s.service
	var request openapigen.PostRegistryVerifyJSONRequestBody
	if bound != nil {
		request = *bound
	}
	if bound == nil && c.ShouldBindJSON(&request) != nil {
		invalidRegistryRequest(c, ctx)
		return
	}
	item, err := service.VerifyConnection(c, c.Param("connectionRef"), registryActorID(c), VerificationInput{RuntimeTargetID: request.RuntimeTargetId, RepositoryRef: request.RepositoryRef})
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	verifiedAt := item.UpdatedAt
	if item.LastVerifiedAt != nil {
		verifiedAt = *item.LastVerifiedAt
	}
	response := openapigen.RegistryConnectionVerification{ConnectionRef: item.ConnectionRef, Status: openapigen.RegistryVerificationResult(item.VerificationStatus), VerifiedAt: verifiedAt}
	if item.LastVerificationErrorCode != "" {
		value := item.LastVerificationErrorCode
		response.ErrorCode = &value
	}
	httpx.WriteSuccess(c, http.StatusOK, response)
}
func (s registryStrictServer) handleListRepositories(c *gin.Context) {
	ctx, service := s.module, s.service
	limit, offset := parsePage(c)
	items, total, err := service.ListRepositories(c, c.Param("connectionRef"), limit, offset)
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, openapigen.RegistryArtifactRepositoryListResponse{Items: mapRepositories(items), Total: int64(total), Limit: limit, Offset: offset})
}
func (s registryStrictServer) handleCreateRepository(c *gin.Context, bound *openapigen.PostRegistryArtifactRepositoryJSONRequestBody) {
	ctx, service := s.module, s.service
	var request openapigen.PostRegistryArtifactRepositoryJSONRequestBody
	if bound != nil {
		request = *bound
	}
	if bound == nil && c.ShouldBindJSON(&request) != nil {
		invalidRegistryRequest(c, ctx)
		return
	}
	allowPull, allowPush := true, true
	if request.AllowPull != nil {
		allowPull = *request.AllowPull
	}
	if request.AllowPush != nil {
		allowPush = *request.AllowPush
	}
	grantCreatorUse := true
	if request.GrantCreatorUse != nil {
		grantCreatorUse = *request.GrantCreatorUse
	}
	item, err := service.CreateRepository(c, c.Param("connectionRef"), registrystore.RepositoryInput{RepositoryRef: request.RepositoryRef, DisplayName: request.DisplayName, AllowPull: allowPull, AllowPush: allowPush, GrantCreatorUse: grantCreatorUse}, registryActorID(c))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusCreated, mapRepository(item))
}
func (s registryStrictServer) handleReplaceAssignments(c *gin.Context, bound *openapigen.PutRegistryArtifactRepositoryAssignmentsJSONRequestBody) {
	ctx, service := s.module, s.service
	var request openapigen.PutRegistryArtifactRepositoryAssignmentsJSONRequestBody
	if bound != nil {
		request = *bound
	}
	if bound == nil && c.ShouldBindJSON(&request) != nil {
		invalidRegistryRequest(c, ctx)
		return
	}
	repositoryRef := registryRepositoryRef(c)
	if repositoryRef == "" {
		invalidRegistryRequest(c, ctx)
		return
	}
	userIDs := make([]uint64, len(request.UserIds))
	seen := make(map[uint64]struct{}, len(request.UserIds))
	for index, userID := range request.UserIds {
		if userID < 1 {
			invalidRegistryRequest(c, ctx)
			return
		}
		value := uint64(userID)
		if _, exists := seen[value]; exists {
			invalidRegistryRequest(c, ctx)
			return
		}
		seen[value] = struct{}{}
		userIDs[index] = value
	}
	items, err := service.ReplaceAssignments(c, c.Param("connectionRef"), repositoryRef, userIDs, registryActorID(c))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	limit := max(1, len(items))
	httpx.WriteSuccess(c, http.StatusOK, openapigen.RegistryArtifactRepositoryUserAssignmentListResponse{Items: mapAssignments(items), Total: int64(len(items)), Limit: limit, Offset: 0})
}

//nolint:dupl // 批量追加与撤销共享请求校验边界，但分别绑定不同的 OpenAPI DTO 和持久化结果。
func (s registryStrictServer) handleAddAssignments(c *gin.Context, bound *openapigen.PostRegistryArtifactRepositoryAssignmentsBatchAddJSONRequestBody) {
	ctx, service := s.module, s.service
	var request openapigen.PostRegistryArtifactRepositoryAssignmentsBatchAddJSONRequestBody
	if bound != nil {
		request = *bound
	}
	if bound == nil && c.ShouldBindJSON(&request) != nil {
		invalidRegistryRequest(c, ctx)
		return
	}
	repositoryRefs, ok := uniqueRepositoryRefs(request.RepositoryRefs)
	userIDs, usersOK := uniqueAssignmentUserIDs(request.UserIds)
	if !ok || !usersOK {
		invalidRegistryRequest(c, ctx)
		return
	}
	result, err := service.AddAssignments(c, c.Param("connectionRef"), registrystore.AssignmentBatchAddInput{RepositoryRefs: repositoryRefs, UserIDs: userIDs}, registryActorID(c))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, openapigen.RegistryArtifactRepositoryAssignmentBatchAddResult{Total: result.Total, AddedCount: result.AddedCount, AlreadyAssignedCount: result.AlreadyAssignedCount})
}

//nolint:dupl // 批量追加与撤销共享请求校验边界，但分别绑定不同的 OpenAPI DTO 和持久化结果。
func (s registryStrictServer) handleRevokeAssignments(c *gin.Context, bound *openapigen.PostRegistryArtifactRepositoryAssignmentsBatchRevokeJSONRequestBody) {
	ctx, service := s.module, s.service
	var request openapigen.PostRegistryArtifactRepositoryAssignmentsBatchRevokeJSONRequestBody
	if bound != nil {
		request = *bound
	}
	if bound == nil && c.ShouldBindJSON(&request) != nil {
		invalidRegistryRequest(c, ctx)
		return
	}
	repositoryRefs, ok := uniqueRepositoryRefs(request.RepositoryRefs)
	userIDs, usersOK := uniqueAssignmentUserIDs(request.UserIds)
	if !ok || !usersOK {
		invalidRegistryRequest(c, ctx)
		return
	}
	result, err := service.RevokeAssignments(c, c.Param("connectionRef"), registrystore.AssignmentBatchRevokeInput{RepositoryRefs: repositoryRefs, UserIDs: userIDs}, registryActorID(c))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, openapigen.RegistryArtifactRepositoryAssignmentBatchRevokeResult{Total: result.Total, RevokedCount: result.RevokedCount, NotAssignedCount: result.NotAssignedCount})
}

func uniqueRepositoryRefs(values []string) ([]string, bool) {
	if len(values) == 0 {
		return nil, false
	}
	refs := make([]string, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		ref := strings.TrimSpace(value)
		if ref == "" {
			return nil, false
		}
		if _, exists := seen[ref]; exists {
			return nil, false
		}
		seen[ref] = struct{}{}
		refs[index] = ref
	}
	return refs, true
}

func uniqueAssignmentUserIDs(values []int64) ([]uint64, bool) {
	if len(values) == 0 {
		return nil, false
	}
	userIDs := make([]uint64, len(values))
	seen := make(map[uint64]struct{}, len(values))
	for index, value := range values {
		if value < 1 {
			return nil, false
		}
		userID := uint64(value)
		if _, exists := seen[userID]; exists {
			return nil, false
		}
		seen[userID] = struct{}{}
		userIDs[index] = userID
	}
	return userIDs, true
}
func (s registryStrictServer) handleUpdateRepository(c *gin.Context, bound *openapigen.PutRegistryArtifactRepositoryJSONRequestBody) {
	ctx, service := s.module, s.service
	var request openapigen.PutRegistryArtifactRepositoryJSONRequestBody
	if bound != nil {
		request = *bound
	}
	if bound == nil && c.ShouldBindJSON(&request) != nil {
		invalidRegistryRequest(c, ctx)
		return
	}
	repositoryRef := registryRepositoryRef(c)
	if repositoryRef == "" {
		invalidRegistryRequest(c, ctx)
		return
	}
	item, err := service.UpdateRepository(c, c.Param("connectionRef"), repositoryRef, registrystore.RepositoryInput{RepositoryRef: repositoryRef, DisplayName: request.DisplayName, AllowPull: request.AllowPull, AllowPush: request.AllowPush}, registryActorID(c))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, mapRepository(item))
}
func (s registryStrictServer) handleDeleteRepository(c *gin.Context) {
	ctx, service := s.module, s.service
	repositoryRef := registryRepositoryRef(c)
	if repositoryRef == "" {
		invalidRegistryRequest(c, ctx)
		return
	}
	if err := service.DeleteRepository(c, c.Param("connectionRef"), repositoryRef, registryActorID(c)); err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess[any](c, http.StatusOK, nil)
}
func (s registryStrictServer) handleListAssignments(c *gin.Context) {
	ctx, service := s.module, s.service
	repositoryRef := registryRepositoryRef(c)
	if repositoryRef == "" {
		invalidRegistryRequest(c, ctx)
		return
	}
	limit, offset := parsePage(c)
	items, total, err := service.ListAssignments(c, c.Param("connectionRef"), repositoryRef, limit, offset)
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, openapigen.RegistryArtifactRepositoryUserAssignmentListResponse{Items: mapAssignments(items), Total: int64(total), Limit: limit, Offset: offset})
}

func (s registryStrictServer) handleListAssignmentCandidates(c *gin.Context) {
	ctx, service := s.module, s.service
	rawRepositoryRefs := c.QueryArray("repository_ref")
	if len(rawRepositoryRefs) == 0 || len(rawRepositoryRefs) > 100 {
		invalidRegistryRequest(c, ctx)
		return
	}
	repositoryRefs := normalizeRepositoryRefs(rawRepositoryRefs)
	if len(repositoryRefs) == 0 {
		invalidRegistryRequest(c, ctx)
		return
	}
	limit, offset := parsePage(c)
	items, total, err := service.ListAssignmentCandidates(c, c.Param("connectionRef"), repositoryRefs, c.Query("search"), limit, offset)
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	response := make([]openapigen.RegistryRepositoryAssignmentCandidate, 0, len(items))
	for _, item := range items {
		userID, err := toOpenAPIUserID(item.UserID)
		if err != nil {
			writeRegistryError(c, ctx, err)
			return
		}
		response = append(response, openapigen.RegistryRepositoryAssignmentCandidate{
			Id:                      userID,
			Username:                item.Username,
			Display:                 item.Display,
			Status:                  item.Status,
			AssignedRepositoryCount: item.AssignedRepositoryCount,
			SelectedRepositoryCount: item.SelectedRepositoryCount,
			AuthorizationState:      openapigen.RegistryRepositoryAssignmentCandidateAuthorizationState(item.AuthorizationState),
		})
	}
	httpx.WriteSuccess(c, http.StatusOK, openapigen.RegistryRepositoryAssignmentCandidateListResponse{Items: response, Total: int64(total), Limit: limit, Offset: offset})
}

func toOpenAPIUserID(userID uint64) (int64, error) {
	if userID > math.MaxInt64 {
		return 0, fmt.Errorf("registry assignment candidate user ID exceeds OpenAPI integer range")
	}
	return int64(userID), nil
}
func (s registryStrictServer) handleGrantAssignment(c *gin.Context, bound *openapigen.PostRegistryArtifactRepositoryAssignmentJSONRequestBody) {
	ctx, service := s.module, s.service
	var request openapigen.PostRegistryArtifactRepositoryAssignmentJSONRequestBody
	if bound != nil {
		request = *bound
	}
	if (bound == nil && c.ShouldBindJSON(&request) != nil) || request.UserId < 1 {
		invalidRegistryRequest(c, ctx)
		return
	}
	repositoryRef := registryRepositoryRef(c)
	if repositoryRef == "" {
		invalidRegistryRequest(c, ctx)
		return
	}
	item, err := service.GrantAssignment(c, c.Param("connectionRef"), repositoryRef, uint64(request.UserId), registryActorID(c))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusCreated, mapAssignment(item))
}
func (s registryStrictServer) handleRevokeAssignment(c *gin.Context) {
	ctx, service := s.module, s.service
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || userID == 0 {
		invalidRegistryRequest(c, ctx)
		return
	}
	repositoryRef := registryRepositoryRef(c)
	if repositoryRef == "" {
		invalidRegistryRequest(c, ctx)
		return
	}
	if err := service.RevokeAssignment(c, c.Param("connectionRef"), repositoryRef, userID, registryActorID(c)); err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess[any](c, http.StatusOK, nil)
}
func (s registryStrictServer) handleAvailableDestinations(c *gin.Context) {
	ctx, service := s.module, s.service
	actorID := registryActorID(c)
	limit, offset := parsePage(c)
	items, total, err := service.ListAvailableDestinations(c, actorID, limit, offset)
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	result := make([]openapigen.RegistryAvailableDestination, 0, len(items))
	for _, item := range items {
		result = append(result, openapigen.RegistryAvailableDestination{Kind: "oci_registry", ConnectionRef: item.ConnectionRef, ConnectionDisplayName: item.ConnectionName, RepositoryRef: item.RepositoryRef, RepositoryDisplayName: item.RepositoryName, AllowPull: item.AllowPull, AllowPush: item.AllowPush})
	}
	httpx.WriteSuccess(c, http.StatusOK, openapigen.RegistryAvailableDestinationListResponse{Items: result, Total: int64(total), Limit: limit, Offset: offset})
}

func registryActorID(c *gin.Context) uint64 {
	if auth, ok := moduleapi.RequestAuthContextFromContext(c.Request.Context()); ok && auth.User != nil {
		return auth.User.ID
	}
	return 0
}

func registryRepositoryRef(c *gin.Context) string {
	return strings.TrimSpace(c.Query("repository_ref"))
}

func parsePage(c *gin.Context) (int, int) {
	limit, offset := 20, 0
	if raw := c.Query("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value >= 1 && value <= 100 {
			limit = value
		}
	}
	if raw := c.Query("offset"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value >= 0 {
			offset = value
		}
	}
	return limit, offset
}
func invalidRegistryRequest(c *gin.Context, ctx *module.Context) {
	httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
}
func writeRegistryError(c *gin.Context, ctx *module.Context, err error) {
	if errors.Is(err, registrystore.ErrNotFound) {
		httpx.WriteLocalizedError(c, ctx.I18n, http.StatusNotFound, "common.not_found", nil)
		return
	}
	if errors.Is(err, registrystore.ErrConflict) || errors.Is(err, registrystore.ErrSystemManaged) {
		httpx.WriteLocalizedError(c, ctx.I18n, http.StatusConflict, "common.invalidArgument", nil)
		return
	}
	if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "required") {
		invalidRegistryRequest(c, ctx)
		return
	}
	_ = logger.ReportError(c, ctx.AppLogger, "registry operation failed", err)
	httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
}
func connectionInputFromCreate(request openapigen.PostRegistryJSONRequestBody) registrystore.ConnectionInput {
	input := registrystore.ConnectionInput{ConnectionRef: request.ConnectionRef, DisplayName: request.DisplayName, Provider: string(request.Provider), Endpoint: request.Endpoint, Enabled: true}
	if request.CredentialRef != nil {
		input.CredentialRef = *request.CredentialRef
	}
	if request.Description != nil {
		input.Description = *request.Description
	}
	if request.Enabled != nil {
		input.Enabled = *request.Enabled
	}
	if request.Insecure != nil {
		input.Insecure = *request.Insecure
	}
	return input
}
func connectionInputFromUpdate(request openapigen.PutRegistryJSONRequestBody, existing registrystore.Connection) registrystore.ConnectionInput {
	input := registrystore.ConnectionInput{ConnectionRef: existing.ConnectionRef, DisplayName: request.DisplayName, Provider: existing.Provider, Endpoint: request.Endpoint, Enabled: request.Enabled, Insecure: request.Insecure, Description: existing.Description, CredentialRef: existing.CredentialRef}
	if request.CredentialRef != nil {
		input.CredentialRef = *request.CredentialRef
	}
	if request.Description != nil {
		input.Description = *request.Description
	}
	return input
}
func mapConnection(item registrystore.Connection) openapigen.RegistryConnection {
	description := item.Description
	managed := item.SystemManaged
	return openapigen.RegistryConnection{ConnectionRef: item.ConnectionRef, DisplayName: item.DisplayName, Provider: openapigen.RegistryConnectionProvider(item.Provider), Endpoint: item.Endpoint, Enabled: item.Enabled, Insecure: item.Insecure, CredentialConfigured: item.CredentialRef != "", Availability: item.Availability, ConnectionVerificationStatus: openapigen.RegistryConnectionVerificationStatus(item.VerificationStatus), LastVerifiedAt: item.LastVerifiedAt, Description: &description, SystemManaged: &managed, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
func mapConnections(items []registrystore.Connection) []openapigen.RegistryConnection {
	result := make([]openapigen.RegistryConnection, 0, len(items))
	for _, item := range items {
		result = append(result, mapConnection(item))
	}
	return result
}
func mapRepository(item registrystore.Repository) openapigen.RegistryArtifactRepository {
	return openapigen.RegistryArtifactRepository{ConnectionRef: item.ConnectionRef, RepositoryRef: item.RepositoryRef, DisplayName: item.DisplayName, AllowPull: item.AllowPull, AllowPush: item.AllowPush, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
func mapRepositories(items []registrystore.Repository) []openapigen.RegistryArtifactRepository {
	result := make([]openapigen.RegistryArtifactRepository, 0, len(items))
	for _, item := range items {
		result = append(result, mapRepository(item))
	}
	return result
}
func mapAssignment(item registrystore.UserAssignment) openapigen.RegistryArtifactRepositoryUserAssignment {
	createdBy := int64(item.CreatedBy) // #nosec G115 -- 用户主键受数据库 BIGINT 正值约束。
	userID := int64(item.UserID)       // #nosec G115 -- 用户主键受数据库 BIGINT 正值约束。
	return openapigen.RegistryArtifactRepositoryUserAssignment{ConnectionRef: item.ConnectionRef, RepositoryRef: item.RepositoryRef, UserId: userID, CreatedAt: item.CreatedAt, CreatedBy: &createdBy}
}
func mapAssignments(items []registrystore.UserAssignment) []openapigen.RegistryArtifactRepositoryUserAssignment {
	result := make([]openapigen.RegistryArtifactRepositoryUserAssignment, 0, len(items))
	for _, item := range items {
		result = append(result, mapAssignment(item))
	}
	return result
}
