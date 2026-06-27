package rbac

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
	rbacstore "graft/server/modules/rbac/store"
)

type rbacManagementErrorMapping struct {
	status  int
	key     messagecontract.Key
	details map[string]any
	known   bool
}

func writeRBACManagementError(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	logger *zap.Logger,
	moduleName string,
	err error,
	invalidField string,
) {
	mapping := rbacManagementErrorResponse(err, invalidField)
	if !mapping.known {
		logger.Error("rbac management write failed",
			zap.String("module", moduleName),
			zap.Error(err),
		)
	}

	writeLocalizedContractError(ginCtx, localizer, mapping.status, mapping.key, mapping.details)
}

func rbacManagementErrorResponse(
	err error,
	invalidField string,
) rbacManagementErrorMapping {
	if mapping := rbacManagementLookupError(err); mapping.known {
		return mapping
	}
	if mapping := rbacManagementMutationError(err, invalidField); mapping.known {
		return mapping
	}
	return rbacManagementErrorMapping{
		status: http.StatusInternalServerError,
		key:    messagecontract.CommonInternalError,
	}
}

func rbacManagementLookupError(err error) rbacManagementErrorMapping {
	switch {
	case errors.Is(err, rbacstore.ErrRoleNotFound):
		return knownRBACManagementError(http.StatusNotFound, messagecontract.RoleNotFound, nil)
	case errors.Is(err, moduleapi.ErrUserNotFound):
		return knownRBACManagementError(http.StatusNotFound, messagecontract.UserNotFound, nil)
	case errors.Is(err, rbacstore.ErrPermissionNotFound):
		return knownRBACManagementError(
			http.StatusBadRequest,
			messagecontract.CommonInvalidArgument,
			map[string]any{"field": "permission_ids"},
		)
	default:
		return rbacManagementErrorMapping{}
	}
}

func rbacManagementMutationError(
	err error,
	invalidField string,
) rbacManagementErrorMapping {
	switch {
	case errors.Is(err, rbacstore.ErrRoleNameConflict), errors.Is(err, errBuiltinRoleNameImmutable):
		return knownRBACManagementError(
			http.StatusBadRequest,
			messagecontract.CommonInvalidArgument,
			map[string]any{"field": "name"},
		)
	case isRoleLifecycleMutationError(err):
		return knownRBACManagementError(
			http.StatusConflict,
			messagecontract.CommonInvalidArgument,
			map[string]any{"field": invalidField},
		)
	case errors.Is(err, errCannotRemoveOwnAdminRole):
		return knownRBACManagementError(http.StatusForbidden, messagecontract.RbacCannotRemoveOwnAdminRole, nil)
	case errors.Is(err, errProtectedUserRoleMutation):
		return knownRBACManagementError(http.StatusForbidden, messagecontract.UserProtectedDefaultAdminImmutable, nil)
	case errors.Is(err, rbacstore.ErrRolePermissionsImmutable):
		return knownRBACManagementError(http.StatusForbidden, messagecontract.RbacBuiltinAdminPermissionsImmutable, nil)
	case isInvalidIDMutationError(err):
		return knownRBACManagementError(
			http.StatusBadRequest,
			messagecontract.CommonInvalidArgument,
			map[string]any{"field": invalidField},
		)
	default:
		return rbacManagementErrorMapping{}
	}
}

func knownRBACManagementError(
	status int,
	key messagecontract.Key,
	details map[string]any,
) rbacManagementErrorMapping {
	return rbacManagementErrorMapping{
		status:  status,
		key:     key,
		details: details,
		known:   true,
	}
}

func isRoleLifecycleMutationError(err error) bool {
	return errors.Is(err, rbacstore.ErrRoleBuiltinImmutable) ||
		errors.Is(err, rbacstore.ErrRoleEnabledDeletionForbidden) ||
		errors.Is(err, rbacstore.ErrRoleBindingsExist) ||
		errors.Is(err, rbacstore.ErrRoleDisabledAssignmentForbidden)
}

func isInvalidIDMutationError(err error) bool {
	return errors.Is(err, errInvalidPermissionIDs) ||
		errors.Is(err, errInvalidRoleIDs) ||
		errors.Is(err, rbacstore.ErrInvalidID)
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
