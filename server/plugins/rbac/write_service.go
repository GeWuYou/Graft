package rbac

import (
	"context"
	"errors"
	"strings"

	"graft/server/internal/pluginapi"
	rbacstore "graft/server/plugins/rbac/store"
)

var (
	errBuiltinRoleNameImmutable = errors.New("builtin role name is immutable")
	errCannotRemoveOwnAdminRole = errors.New("cannot remove own admin role")
	errInvalidPermissionIDs     = errors.New("invalid permission ids")
	errInvalidRoleIDs           = errors.New("invalid role ids")
)

const builtinAdminRoleName = "admin"

type writeManagementService interface {
	CreateRole(ctx context.Context, input rbacstore.CreateRoleInput) (rbacstore.Role, error)
	UpdateRole(ctx context.Context, input rbacstore.UpdateRoleInput) (rbacstore.Role, error)
	SetRoleStatus(ctx context.Context, input rbacstore.SetRoleStatusInput) (rbacstore.Role, error)
	SoftDeleteRole(ctx context.Context, input rbacstore.SoftDeleteRoleInput) error
	ReplacePermissionsForRole(ctx context.Context, input rbacstore.ReplacePermissionsForRoleInput) error
	AddPermissionsToRole(ctx context.Context, input rbacstore.AddPermissionsToRoleInput) error
	RemovePermissionsFromRole(ctx context.Context, input rbacstore.RemovePermissionsFromRoleInput) error
	ReplaceRolesForUser(ctx context.Context, input rbacstore.ReplaceRolesForUserInput) error
	AddRolesToUser(ctx context.Context, input rbacstore.AddRolesToUserInput) error
	RemoveRolesFromUser(ctx context.Context, input rbacstore.RemoveRolesFromUserInput) error
	ReplaceRolesForUsers(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error
	AddRolesToUsers(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error
	RemoveRolesFromUsers(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error
}

type managementWriter struct {
	users pluginapi.UserService
	rbac  rbacstore.Repository
}

func (w managementWriter) CreateRole(ctx context.Context, input rbacstore.CreateRoleInput) (rbacstore.Role, error) {
	if w.rbac == nil {
		return rbacstore.Role{}, errors.New("rbac repository is unavailable")
	}

	return w.rbac.CreateRole(ctx, input)
}

func (w managementWriter) UpdateRole(ctx context.Context, input rbacstore.UpdateRoleInput) (rbacstore.Role, error) {
	if w.rbac == nil {
		return rbacstore.Role{}, errors.New("rbac repository is unavailable")
	}

	current, err := w.rbac.GetRoleByID(ctx, input.ID)
	if err != nil {
		return rbacstore.Role{}, err
	}
	if current.Builtin && strings.TrimSpace(current.Name) != strings.TrimSpace(input.Name) {
		return rbacstore.Role{}, errBuiltinRoleNameImmutable
	}

	return w.rbac.UpdateRole(ctx, input)
}

func (w managementWriter) SetRoleStatus(ctx context.Context, input rbacstore.SetRoleStatusInput) (rbacstore.Role, error) {
	if w.rbac == nil {
		return rbacstore.Role{}, errors.New("rbac repository is unavailable")
	}

	return w.rbac.SetRoleStatus(ctx, input)
}

func (w managementWriter) SoftDeleteRole(ctx context.Context, input rbacstore.SoftDeleteRoleInput) error {
	if w.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}

	return w.rbac.SoftDeleteRole(ctx, input)
}

func (w managementWriter) ReplacePermissionsForRole(ctx context.Context, input rbacstore.ReplacePermissionsForRoleInput) error {
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
		if errors.Is(err, rbacstore.ErrPermissionNotFound) {
			if validationErr := ensurePermissionIDsExist(ctx, w.rbac, input.PermissionIDs); validationErr != nil {
				return validationErr
			}

			return errInvalidPermissionIDs
		}

		return err
	}

	return nil
}

func (w managementWriter) AddPermissionsToRole(ctx context.Context, input rbacstore.AddPermissionsToRoleInput) error {
	if w.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}
	if _, err := w.rbac.GetRoleByID(ctx, input.RoleID); err != nil {
		return err
	}
	if err := ensurePermissionIDsExist(ctx, w.rbac, input.PermissionIDs); err != nil {
		return err
	}
	return w.rbac.AddPermissionsToRole(ctx, input)
}

func (w managementWriter) RemovePermissionsFromRole(ctx context.Context, input rbacstore.RemovePermissionsFromRoleInput) error {
	if w.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}
	if _, err := w.rbac.GetRoleByID(ctx, input.RoleID); err != nil {
		return err
	}
	if err := ensurePermissionIDsExist(ctx, w.rbac, input.PermissionIDs); err != nil {
		return err
	}
	return w.rbac.RemovePermissionsFromRole(ctx, input)
}

func (w managementWriter) ReplaceRolesForUser(ctx context.Context, input rbacstore.ReplaceRolesForUserInput) error {
	if w.users == nil {
		return errors.New("user service is unavailable")
	}
	if w.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}
	if _, err := w.users.GetUserByID(ctx, input.UserID); err != nil {
		return err
	}
	if err := w.ensureActorKeepsBuiltinAdminRole(ctx, input); err != nil {
		return err
	}

	if err := w.rbac.ReplaceRolesForUser(ctx, input); err != nil {
		if errors.Is(err, rbacstore.ErrRoleNotFound) {
			if validationErr := ensureRoleIDsExist(ctx, w.rbac, input.RoleIDs); validationErr != nil {
				return validationErr
			}

			return errInvalidRoleIDs
		}

		return err
	}

	return nil
}

func (w managementWriter) AddRolesToUser(ctx context.Context, input rbacstore.AddRolesToUserInput) error {
	if err := w.ensureRoleMutationPreconditions(ctx, []uint64{input.UserID}, input.RoleIDs); err != nil {
		return err
	}
	return w.rbac.AddRolesToUser(ctx, input)
}

func (w managementWriter) RemoveRolesFromUser(ctx context.Context, input rbacstore.RemoveRolesFromUserInput) error {
	if err := w.ensureRoleMutationPreconditions(ctx, []uint64{input.UserID}, input.RoleIDs); err != nil {
		return err
	}
	if err := w.ensureActorCanRemoveRoles(ctx, input.UserID, input.RoleIDs); err != nil {
		return err
	}
	return w.rbac.RemoveRolesFromUser(ctx, input)
}

func (w managementWriter) ReplaceRolesForUsers(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
	if err := w.ensureRoleMutationPreconditions(ctx, input.UserIDs, input.RoleIDs); err != nil {
		return err
	}
	for _, userID := range input.UserIDs {
		if err := w.ensureActorCanReplaceRoles(ctx, userID, input.RoleIDs); err != nil {
			return err
		}
		if err := w.rbac.ReplaceRolesForUser(ctx, rbacstore.ReplaceRolesForUserInput{
			UserID:  userID,
			RoleIDs: input.RoleIDs,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (w managementWriter) AddRolesToUsers(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
	if err := w.ensureRoleMutationPreconditions(ctx, input.UserIDs, input.RoleIDs); err != nil {
		return err
	}
	for _, userID := range input.UserIDs {
		if err := w.rbac.AddRolesToUser(ctx, rbacstore.AddRolesToUserInput{
			UserID:  userID,
			RoleIDs: input.RoleIDs,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (w managementWriter) RemoveRolesFromUsers(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
	if err := w.ensureRoleMutationPreconditions(ctx, input.UserIDs, input.RoleIDs); err != nil {
		return err
	}
	for _, userID := range input.UserIDs {
		if err := w.ensureActorCanRemoveRoles(ctx, userID, input.RoleIDs); err != nil {
			return err
		}
		if err := w.rbac.RemoveRolesFromUser(ctx, rbacstore.RemoveRolesFromUserInput{
			UserID:  userID,
			RoleIDs: input.RoleIDs,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (w managementWriter) ensureActorKeepsBuiltinAdminRole(ctx context.Context, input rbacstore.ReplaceRolesForUserInput) error {
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

func (w managementWriter) ensureActorCanReplaceRoles(ctx context.Context, userID uint64, roleIDs []uint64) error {
	return w.ensureActorKeepsBuiltinAdminRole(ctx, rbacstore.ReplaceRolesForUserInput{
		UserID:  userID,
		RoleIDs: roleIDs,
	})
}

func (w managementWriter) ensureActorCanRemoveRoles(ctx context.Context, userID uint64, roleIDs []uint64) error {
	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil || requestAuth.User.ID == 0 || requestAuth.User.ID != userID {
		return nil
	}

	currentRoles, err := w.rbac.ListRolesByUserID(ctx, userID)
	if err != nil {
		return err
	}

	builtinAdmin, hasBuiltinAdmin := findBuiltinAdminRole(currentRoles)
	if !hasBuiltinAdmin {
		return nil
	}

	for _, roleID := range roleIDs {
		if roleID == builtinAdmin.ID {
			return errCannotRemoveOwnAdminRole
		}
	}

	return nil
}

func (w managementWriter) ensureRoleMutationPreconditions(ctx context.Context, userIDs []uint64, roleIDs []uint64) error {
	if w.users == nil {
		return errors.New("user service is unavailable")
	}
	if w.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}
	for _, userID := range userIDs {
		if _, err := w.users.GetUserByID(ctx, userID); err != nil {
			return err
		}
	}
	if err := ensureRoleIDsExist(ctx, w.rbac, roleIDs); err != nil {
		return err
	}
	return nil
}

func findBuiltinAdminRole(roles []rbacstore.Role) (rbacstore.Role, bool) {
	for _, role := range roles {
		if role.Builtin && strings.TrimSpace(role.Name) == builtinAdminRoleName {
			return role, true
		}
	}

	return rbacstore.Role{}, false
}

func ensurePermissionIDsExist(ctx context.Context, repository rbacstore.Repository, permissionIDs []uint64) error {
	if len(permissionIDs) == 0 {
		return nil
	}

	permissions, err := repository.ListPermissions(ctx, rbacstore.PermissionFilter{})
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

func ensureRoleIDsExist(ctx context.Context, repository rbacstore.Repository, roleIDs []uint64) error {
	if len(roleIDs) == 0 {
		return nil
	}

	roles, err := repository.ListRoles(ctx, rbacstore.RoleFilter{})
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
