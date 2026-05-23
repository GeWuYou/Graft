package rbac

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/pluginapi"
	rbacstore "graft/server/plugins/rbac/store"
)

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
	case errors.Is(err, rbacstore.ErrRoleNotFound):
		status = http.StatusNotFound
		key = messagecontract.RoleNotFound
	case errors.Is(err, pluginapi.ErrUserNotFound):
		status = http.StatusNotFound
		key = messagecontract.UserNotFound
	case errors.Is(err, rbacstore.ErrRoleNameConflict):
		status = http.StatusBadRequest
		key = messagecontract.CommonInvalidArgument
		details = map[string]any{"field": "name"}
	case errors.Is(err, rbacstore.ErrPermissionNotFound):
		status = http.StatusBadRequest
		key = messagecontract.CommonInvalidArgument
		details = map[string]any{"field": "permission_ids"}
	case errors.Is(err, errBuiltinRoleNameImmutable):
		status = http.StatusBadRequest
		key = messagecontract.CommonInvalidArgument
		details = map[string]any{"field": "name"}
	case errors.Is(err, errCannotRemoveOwnAdminRole):
		status = http.StatusForbidden
		key = messagecontract.RbacCannotRemoveOwnAdminRole
	case errors.Is(err, errInvalidPermissionIDs), errors.Is(err, errInvalidRoleIDs), errors.Is(err, rbacstore.ErrInvalidID):
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
