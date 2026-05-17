package rbac

import (
	"context"
	"errors"
	"strings"

	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
)

var (
	errBuiltinRoleNameImmutable = errors.New("builtin role name is immutable")
	errCannotRemoveOwnAdminRole = errors.New("cannot remove own admin role")
	errInvalidPermissionIDs     = errors.New("invalid permission ids")
	errInvalidRoleIDs           = errors.New("invalid role ids")
)

const builtinAdminRoleName = "admin"

type writeManagementService interface {
	CreateRole(ctx context.Context, input store.CreateRoleInput) (store.Role, error)
	UpdateRole(ctx context.Context, input store.UpdateRoleInput) (store.Role, error)
	ReplacePermissionsForRole(ctx context.Context, input store.ReplacePermissionsForRoleInput) error
	ReplaceRolesForUser(ctx context.Context, input store.ReplaceRolesForUserInput) error
}

type managementWriter struct {
	rbac store.RBACRepository
}

func (w managementWriter) CreateRole(ctx context.Context, input store.CreateRoleInput) (store.Role, error) {
	if w.rbac == nil {
		return store.Role{}, errors.New("rbac repository is unavailable")
	}

	return w.rbac.CreateRole(ctx, input)
}

func (w managementWriter) UpdateRole(ctx context.Context, input store.UpdateRoleInput) (store.Role, error) {
	if w.rbac == nil {
		return store.Role{}, errors.New("rbac repository is unavailable")
	}

	current, err := w.rbac.GetRoleByID(ctx, input.ID)
	if err != nil {
		return store.Role{}, err
	}
	if current.Builtin && strings.TrimSpace(current.Name) != strings.TrimSpace(input.Name) {
		return store.Role{}, errBuiltinRoleNameImmutable
	}

	return w.rbac.UpdateRole(ctx, input)
}

func (w managementWriter) ReplacePermissionsForRole(ctx context.Context, input store.ReplacePermissionsForRoleInput) error {
	if w.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}

	if _, err := w.rbac.GetRoleByID(ctx, input.RoleID); err != nil {
		return err
	}
	if err := ensurePermissionIDsExist(ctx, w.rbac, input.PermissionIDs); err != nil {
		return err
	}

	if err := w.rbac.ReplacePermissionsForRole(ctx, input); err != nil {
		if errors.Is(err, store.ErrPermissionNotFound) {
			if validationErr := ensurePermissionIDsExist(ctx, w.rbac, input.PermissionIDs); validationErr != nil {
				return validationErr
			}

			return errInvalidPermissionIDs
		}

		return err
	}

	return nil
}

func (w managementWriter) ReplaceRolesForUser(ctx context.Context, input store.ReplaceRolesForUserInput) error {
	if w.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}
	if err := w.ensureActorKeepsBuiltinAdminRole(ctx, input); err != nil {
		return err
	}

	if err := w.rbac.ReplaceRolesForUser(ctx, input); err != nil {
		if errors.Is(err, store.ErrRoleNotFound) {
			if validationErr := ensureRoleIDsExist(ctx, w.rbac, input.RoleIDs); validationErr != nil {
				return validationErr
			}

			return errInvalidRoleIDs
		}

		return err
	}

	return nil
}

func (w managementWriter) ensureActorKeepsBuiltinAdminRole(ctx context.Context, input store.ReplaceRolesForUserInput) error {
	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil || requestAuth.User.ID == 0 {
		return nil
	}
	if requestAuth.User.ID != input.UserID {
		return nil
	}

	currentRoles, err := w.rbac.ListRolesByUserID(ctx, input.UserID)
	if err != nil {
		return err
	}

	builtinAdmin, hasBuiltinAdmin := findBuiltinAdminRole(currentRoles)
	if !hasBuiltinAdmin {
		return nil
	}

	for _, roleID := range input.RoleIDs {
		if roleID == builtinAdmin.ID {
			return nil
		}
	}

	return errCannotRemoveOwnAdminRole
}

func findBuiltinAdminRole(roles []store.Role) (store.Role, bool) {
	for _, role := range roles {
		if role.Builtin && strings.TrimSpace(role.Name) == builtinAdminRoleName {
			return role, true
		}
	}

	return store.Role{}, false
}

func ensurePermissionIDsExist(ctx context.Context, repository store.RBACRepository, permissionIDs []uint64) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	permissions, err := repository.ListPermissions(ctx)
	if err != nil {
		return err
	}

	allowed := make(map[uint64]struct{}, len(permissions))
	for _, item := range permissions {
		allowed[item.ID] = struct{}{}
	}

	for _, permissionID := range permissionIDs {
		if _, ok := allowed[permissionID]; !ok {
			return errInvalidPermissionIDs
		}
	}

	return nil
}

func ensureRoleIDsExist(ctx context.Context, repository store.RBACRepository, roleIDs []uint64) error {
	if len(roleIDs) == 0 {
		return nil
	}

	roles, err := repository.ListRoles(ctx)
	if err != nil {
		return err
	}

	allowed := make(map[uint64]struct{}, len(roles))
	for _, item := range roles {
		allowed[item.ID] = struct{}{}
	}

	for _, roleID := range roleIDs {
		if _, ok := allowed[roleID]; !ok {
			return errInvalidRoleIDs
		}
	}

	return nil
}
