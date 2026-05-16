package rbac

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/plugin"
	"graft/server/internal/store"
	rbaccontract "graft/server/plugins/rbac/contract"
)

type createRoleRequest struct {
	Name        string  `json:"name"`
	Display     string  `json:"display"`
	Description *string `json:"description"`
}

type updateRoleRequest struct {
	Name        string  `json:"name"`
	Display     string  `json:"display"`
	Description *string `json:"description"`
}

type replaceRolePermissionsRequest struct {
	PermissionIDs *[]uint64 `json:"permission_ids"`
}

type replaceUserRolesRequest struct {
	RoleIDs *[]uint64 `json:"role_ids"`
}

func registerRoleWriteRoutes(
	group *gin.RouterGroup,
	ctx *plugin.Context,
	pluginName string,
	writer writeManagementService,
	guards managementGuards,
) {
	group.POST(rbaccontract.RoleCollection, guards.roleCreate, func(ginCtx *gin.Context) {
		var request createRoleRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "body",
			})
			return
		}

		roleInput, ok := normalizeCreateRoleInput(request)
		if !ok {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "name",
			})
			return
		}
		if strings.TrimSpace(roleInput.Display) == "" {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "display",
			})
			return
		}

		role, err := writer.CreateRole(ginCtx.Request.Context(), roleInput)
		if err != nil {
			writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, pluginName, err, "id")
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, toRoleListItem(role))
	})

	group.POST(rbaccontract.RoleUpdateRoute, guards.roleUpdate, func(ginCtx *gin.Context) {
		roleID, err := parseManagementID(ginCtx.Param("id"))
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "id",
			})
			return
		}

		var request updateRoleRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "body",
			})
			return
		}

		roleInput, ok := normalizeUpdateRoleInput(roleID, request)
		if !ok {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "name",
			})
			return
		}
		if strings.TrimSpace(roleInput.Display) == "" {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "display",
			})
			return
		}

		role, err := writer.UpdateRole(ginCtx.Request.Context(), roleInput)
		if err != nil {
			writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, pluginName, err, "id")
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, toRoleListItem(role))
	})

	group.POST(rbaccontract.RolePermissionAssignRoute, guards.rolePermissionAssign, func(ginCtx *gin.Context) {
		roleID, err := parseManagementID(ginCtx.Param("id"))
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "id",
			})
			return
		}

		var request replaceRolePermissionsRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "body",
			})
			return
		}
		if request.PermissionIDs == nil || hasInvalidStableIDs(*request.PermissionIDs) {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "permission_ids",
			})
			return
		}

		if err := writer.ReplacePermissionsForRole(ginCtx.Request.Context(), store.ReplacePermissionsForRoleInput{
			RoleID:        roleID,
			PermissionIDs: *request.PermissionIDs,
		}); err != nil {
			writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, pluginName, err, "permission_ids")
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func registerUserRoleRoutes(
	ctx *plugin.Context,
	pluginName string,
	writer writeManagementService,
	authenticated gin.HandlerFunc,
) {
	group := ctx.Router.Group(rbaccontract.UsersGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.POST(rbaccontract.UserRoleAssignRoute, authenticated, func(ginCtx *gin.Context) {
		userID, err := parseManagementID(ginCtx.Param("id"))
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "id",
			})
			return
		}

		var request replaceUserRolesRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "body",
			})
			return
		}
		if request.RoleIDs == nil || hasInvalidStableIDs(*request.RoleIDs) {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "role_ids",
			})
			return
		}

		if err := writer.ReplaceRolesForUser(ginCtx.Request.Context(), store.ReplaceRolesForUserInput{
			UserID:  userID,
			RoleIDs: *request.RoleIDs,
		}); err != nil {
			writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, pluginName, err, "role_ids")
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func normalizeCreateRoleInput(request createRoleRequest) (store.CreateRoleInput, bool) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return store.CreateRoleInput{}, false
	}

	return store.CreateRoleInput{
		Name:        name,
		Display:     strings.TrimSpace(request.Display),
		Description: normalizeOptionalString(request.Description),
		Builtin:     false,
	}, true
}

func normalizeUpdateRoleInput(roleID uint64, request updateRoleRequest) (store.UpdateRoleInput, bool) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return store.UpdateRoleInput{}, false
	}

	return store.UpdateRoleInput{
		ID:          roleID,
		Name:        name,
		Display:     strings.TrimSpace(request.Display),
		Description: normalizeOptionalString(request.Description),
	}, true
}

func normalizeOptionalString(input *string) *string {
	if input == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*input)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func hasInvalidStableIDs(ids []uint64) bool {
	for _, id := range ids {
		if id == 0 {
			return true
		}
	}

	return false
}

func parseManagementID(input string) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(input), 10, 64)
	if err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, errors.New("id must be greater than zero")
	}

	return id, nil
}

func toRoleListItem(role store.Role) roleListItem {
	return roleListItem{
		ID:          role.ID,
		Name:        role.Name,
		Display:     role.Display,
		Description: role.Description,
		Builtin:     role.Builtin,
	}
}

func writeRBACManagementError(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	logger *zap.Logger,
	pluginName string,
	err error,
	invalidField string,
) {
	status := http.StatusInternalServerError
	key := messagecontract.CommonInternalError
	details := map[string]any(nil)

	switch {
	case errors.Is(err, store.ErrRoleNotFound):
		status = http.StatusNotFound
		key = messagecontract.RoleNotFound
	case errors.Is(err, store.ErrUserNotFound):
		status = http.StatusNotFound
		key = messagecontract.UserNotFound
	case errors.Is(err, store.ErrRoleNameConflict):
		status = http.StatusBadRequest
		key = messagecontract.CommonInvalidArgument
		details = map[string]any{"field": "name"}
	case errors.Is(err, store.ErrPermissionNotFound):
		status = http.StatusBadRequest
		key = messagecontract.CommonInvalidArgument
		details = map[string]any{"field": "permission_ids"}
	case errors.Is(err, errBuiltinRoleNameImmutable):
		status = http.StatusBadRequest
		key = messagecontract.CommonInvalidArgument
		details = map[string]any{"field": "name"}
	case errors.Is(err, errInvalidPermissionIDs), errors.Is(err, errInvalidRoleIDs), errors.Is(err, store.ErrInvalidID):
		status = http.StatusBadRequest
		key = messagecontract.CommonInvalidArgument
		details = map[string]any{"field": invalidField}
	default:
		logger.Error("rbac management write failed",
			zap.String("plugin", pluginName),
			zap.Error(err),
		)
	}

	writeLocalizedContractError(ginCtx, localizer, status, key, details)
}

func writeLocalizedContractError(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	status int,
	key messagecontract.Key,
	data map[string]any,
) {
	httpx.WriteLocalizedError(ginCtx, localizer, status, key.String(), data)
}
