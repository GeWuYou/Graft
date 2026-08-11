package registry

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	openapigen "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/httpx"
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
	group.GET("/available-destinations", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildCreatePermission, publisher), func(c *gin.Context) { handleAvailableDestinations(c, ctx, service) })
	group.GET("", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.ReadPermission, publisher), func(c *gin.Context) { handleListConnections(c, ctx, service) })
	group.POST("", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.CreatePermission, publisher), func(c *gin.Context) { handleCreateConnection(c, ctx, service) })
	group.GET("/:connectionRef", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.ReadPermission, publisher), func(c *gin.Context) { handleGetConnection(c, ctx, service) })
	group.PUT("/:connectionRef", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.UpdatePermission, publisher), func(c *gin.Context) { handleUpdateConnection(c, ctx, service) })
	group.DELETE("/:connectionRef", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.DeletePermission, publisher), func(c *gin.Context) { handleDeleteConnection(c, ctx, service) })
	group.POST("/:connectionRef/verify", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.VerifyPermission, publisher), func(c *gin.Context) { handleVerifyConnection(c, ctx, service) })
	group.GET("/:connectionRef/repositories", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.ReadPermission, publisher), func(c *gin.Context) { handleListRepositories(c, ctx, service) })
	group.POST("/:connectionRef/repositories", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.CreatePermission, publisher), func(c *gin.Context) { handleCreateRepository(c, ctx, service) })
	group.PUT("/:connectionRef/repositories", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.UpdatePermission, publisher), func(c *gin.Context) { handleUpdateRepository(c, ctx, service) })
	group.DELETE("/:connectionRef/repositories", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.DeletePermission, publisher), func(c *gin.Context) { handleDeleteRepository(c, ctx, service) })
	group.GET("/:connectionRef/repository-assignments", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.AssignmentManagePermission, publisher), func(c *gin.Context) { handleListAssignments(c, ctx, service) })
	group.POST("/:connectionRef/repository-assignments", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.AssignmentManagePermission, publisher), func(c *gin.Context) { handleGrantAssignment(c, ctx, service) })
	group.DELETE("/:connectionRef/repository-assignments/:userId", httpx.RequirePermission(ctx.I18n, auth, authorizer, registrycontract.AssignmentManagePermission, publisher), func(c *gin.Context) { handleRevokeAssignment(c, ctx, service) })
	return nil
}

func handleListConnections(c *gin.Context, ctx *module.Context, service *Service) {
	limit, offset := parsePage(c)
	items, total, err := service.ListConnections(c, c.Query("search"), limit, offset)
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, openapigen.RegistryConnectionListResponse{Items: mapConnections(items), Limit: limit, Offset: offset, Total: int64(total)})
}
func handleGetConnection(c *gin.Context, ctx *module.Context, service *Service) {
	item, err := service.GetConnection(c, c.Param("connectionRef"))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, mapConnection(item))
}
func handleCreateConnection(c *gin.Context, ctx *module.Context, service *Service) {
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
func handleUpdateConnection(c *gin.Context, ctx *module.Context, service *Service) {
	existing, err := service.GetConnection(c, c.Param("connectionRef"))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	var request openapigen.PutRegistryJSONRequestBody
	if c.ShouldBindJSON(&request) != nil {
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
func handleDeleteConnection(c *gin.Context, ctx *module.Context, service *Service) {
	if err := service.DeleteConnection(c, c.Param("connectionRef"), registryActorID(c)); err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess[any](c, http.StatusOK, nil)
}
func handleVerifyConnection(c *gin.Context, ctx *module.Context, service *Service) {
	item, err := service.VerifyConnection(c, c.Param("connectionRef"), NewHTTPConnectionVerifier())
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	status := "verified"
	if item.VerificationStatus != verificationSucceeded {
		status = "failed"
	}
	verifiedAt := item.UpdatedAt
	if item.LastVerifiedAt != nil {
		verifiedAt = *item.LastVerifiedAt
	}
	response := openapigen.RegistryConnectionVerification{ConnectionRef: item.ConnectionRef, Status: status, VerifiedAt: verifiedAt}
	if item.LastVerificationErrorCode != "" {
		value := item.LastVerificationErrorCode
		response.ErrorCode = &value
	}
	httpx.WriteSuccess(c, http.StatusOK, response)
}
func handleListRepositories(c *gin.Context, ctx *module.Context, service *Service) {
	items, err := service.ListRepositories(c, c.Param("connectionRef"))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, openapigen.RegistryArtifactRepositoryListResponse{Items: mapRepositories(items)})
}
func handleCreateRepository(c *gin.Context, ctx *module.Context, service *Service) {
	var request openapigen.PostRegistryArtifactRepositoryJSONRequestBody
	if c.ShouldBindJSON(&request) != nil {
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
	item, err := service.CreateRepository(c, c.Param("connectionRef"), registrystore.RepositoryInput{RepositoryRef: request.RepositoryRef, DisplayName: request.DisplayName, AllowPull: allowPull, AllowPush: allowPush}, registryActorID(c))
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusCreated, mapRepository(item))
}
func handleUpdateRepository(c *gin.Context, ctx *module.Context, service *Service) {
	var request openapigen.PutRegistryArtifactRepositoryJSONRequestBody
	if c.ShouldBindJSON(&request) != nil {
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
func handleDeleteRepository(c *gin.Context, ctx *module.Context, service *Service) {
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
func handleListAssignments(c *gin.Context, ctx *module.Context, service *Service) {
	repositoryRef := registryRepositoryRef(c)
	if repositoryRef == "" {
		invalidRegistryRequest(c, ctx)
		return
	}
	items, err := service.ListAssignments(c, c.Param("connectionRef"), repositoryRef)
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, openapigen.RegistryArtifactRepositoryUserAssignmentListResponse{Items: mapAssignments(items)})
}
func handleGrantAssignment(c *gin.Context, ctx *module.Context, service *Service) {
	var request openapigen.PostRegistryArtifactRepositoryAssignmentJSONRequestBody
	if c.ShouldBindJSON(&request) != nil || request.UserId < 1 {
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
func handleRevokeAssignment(c *gin.Context, ctx *module.Context, service *Service) {
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
func handleAvailableDestinations(c *gin.Context, ctx *module.Context, service *Service) {
	actorID := registryActorID(c)
	items, err := service.ListAvailableDestinations(c, actorID)
	if err != nil {
		writeRegistryError(c, ctx, err)
		return
	}
	result := make([]openapigen.RegistryAvailableDestination, 0, len(items))
	for _, item := range items {
		result = append(result, openapigen.RegistryAvailableDestination{Kind: "oci_registry", ConnectionRef: item.ConnectionRef, ConnectionDisplayName: item.ConnectionName, RepositoryRef: item.RepositoryRef, RepositoryDisplayName: item.RepositoryName, AllowPull: true, AllowPush: true})
	}
	httpx.WriteSuccess(c, http.StatusOK, openapigen.RegistryAvailableDestinationListResponse{Items: result})
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
	if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "required") {
		invalidRegistryRequest(c, ctx)
		return
	}
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
	return openapigen.RegistryConnection{ConnectionRef: item.ConnectionRef, DisplayName: item.DisplayName, Provider: openapigen.RegistryConnectionProvider(item.Provider), Endpoint: item.Endpoint, Enabled: item.Enabled, Insecure: item.Insecure, CredentialConfigured: item.CredentialRef != "", Availability: item.Availability, VerificationStatus: openapigen.RegistryConnectionVerificationStatus(item.VerificationStatus), LastVerifiedAt: item.LastVerifiedAt, Description: &description, SystemManaged: &managed, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
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
