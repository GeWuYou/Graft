// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"graft/server/internal/eventbus"
	"graft/server/internal/moduleapi"
	rbacstore "graft/server/modules/rbac/store"
)

var (
	errBuiltinRoleNameImmutable = errors.New("builtin role name is immutable")
	errCannotRemoveOwnAdminRole = errors.New("cannot remove own admin role")
	errInvalidPermissionIDs     = errors.New("invalid permission ids")
	errInvalidRoleIDs           = errors.New("invalid role ids")
	errAtomicBatchWriterMissing = errors.New("rbac atomic batch writer is unavailable")
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

type batchUserRoleAtomicWriter interface {
	ReplaceRolesForUsersAtomically(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error
	AddRolesToUsersAtomically(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error
	RemoveRolesFromUsersAtomically(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error
}

type managementWriter struct {
	users    moduleapi.UserService
	rbac     rbacstore.Repository
	auditBus eventbus.Bus
	logger   *zap.Logger
}

func (w managementWriter) CreateRole(ctx context.Context, input rbacstore.CreateRoleInput) (rbacstore.Role, error) {
	if w.rbac == nil {
		return rbacstore.Role{}, errors.New("rbac repository is unavailable")
	}

	role, err := w.rbac.CreateRole(ctx, input)
	if err != nil {
		return rbacstore.Role{}, err
	}

	w.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "rbac.role.create",
		ResourceType: "role",
		ResourceID:   formatRBACAuditID(role.ID),
		ResourceName: role.Name,
		Success:      true,
		Message:      "role created",
		Metadata: map[string]any{
			"display_name": role.Display,
			"builtin":      role.Builtin,
			"status":       role.Status,
		},
	})

	return role, nil
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

	role, err := w.rbac.UpdateRole(ctx, input)
	if err != nil {
		return rbacstore.Role{}, err
	}

	w.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "rbac.role.update",
		ResourceType: "role",
		ResourceID:   formatRBACAuditID(role.ID),
		ResourceName: role.Name,
		Success:      true,
		Message:      "role updated",
		Metadata: map[string]any{
			"display_name": role.Display,
			"builtin":      role.Builtin,
			"status":       role.Status,
		},
	})

	return role, nil
}

func (w managementWriter) SetRoleStatus(ctx context.Context, input rbacstore.SetRoleStatusInput) (rbacstore.Role, error) {
	if w.rbac == nil {
		return rbacstore.Role{}, errors.New("rbac repository is unavailable")
	}

	role, err := w.rbac.SetRoleStatus(ctx, input)
	if err != nil {
		return rbacstore.Role{}, err
	}

	w.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "rbac.role.status.update",
		ResourceType: "role",
		ResourceID:   formatRBACAuditID(role.ID),
		ResourceName: role.Name,
		Success:      true,
		Message:      "role status updated",
		Metadata: map[string]any{
			"status": role.Status,
		},
	})

	return role, nil
}

func (w managementWriter) SoftDeleteRole(ctx context.Context, input rbacstore.SoftDeleteRoleInput) error {
	if w.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}

	role, err := w.rbac.GetRoleByID(ctx, input.ID)
	if err != nil {
		return err
	}
	if err := w.rbac.SoftDeleteRole(ctx, input); err != nil {
		return err
	}

	w.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "rbac.role.delete",
		ResourceType: "role",
		ResourceID:   formatRBACAuditID(role.ID),
		ResourceName: role.Name,
		Success:      true,
		Message:      "role deleted",
	})

	return nil
}

func (w managementWriter) ReplacePermissionsForRole(ctx context.Context, input rbacstore.ReplacePermissionsForRoleInput) error {
	if w.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}

	role, err := w.rbac.GetRoleByID(ctx, input.RoleID)
	if err != nil {
		return err
	}
	if isBuiltinAdminRole(role) {
		return rbacstore.ErrRolePermissionsImmutable
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

	w.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "rbac.role.permissions.replace",
		ResourceType: "role",
		ResourceID:   formatRBACAuditID(input.RoleID),
		ResourceName: role.Name,
		Success:      true,
		Message:      "role permissions replaced",
		Metadata: map[string]any{
			"permission_ids": append([]uint64(nil), input.PermissionIDs...),
		},
	})

	return nil
}

func (w managementWriter) AddPermissionsToRole(ctx context.Context, input rbacstore.AddPermissionsToRoleInput) error {
	return w.mutateRolePermissions(
		ctx,
		input.RoleID,
		input.PermissionIDs,
		"rbac.role.permissions.add",
		"role permissions added",
		func(ctx context.Context) error {
			return w.rbac.AddPermissionsToRole(ctx, input)
		},
	)
}

func (w managementWriter) RemovePermissionsFromRole(ctx context.Context, input rbacstore.RemovePermissionsFromRoleInput) error {
	return w.mutateRolePermissions(
		ctx,
		input.RoleID,
		input.PermissionIDs,
		"rbac.role.permissions.remove",
		"role permissions removed",
		func(ctx context.Context) error {
			return w.rbac.RemovePermissionsFromRole(ctx, input)
		},
	)
}

func (w managementWriter) mutateRolePermissions(
	ctx context.Context,
	roleID uint64,
	permissionIDs []uint64,
	action string,
	message string,
	run func(context.Context) error,
) error {
	if w.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}
	role, err := w.rbac.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if isBuiltinAdminRole(role) {
		return rbacstore.ErrRolePermissionsImmutable
	}
	if err := ensurePermissionIDsExist(ctx, w.rbac, permissionIDs); err != nil {
		return err
	}
	if err := run(ctx); err != nil {
		return err
	}

	w.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       action,
		ResourceType: "role",
		ResourceID:   formatRBACAuditID(roleID),
		ResourceName: role.Name,
		Success:      true,
		Message:      message,
		Metadata: map[string]any{
			"permission_ids": append([]uint64(nil), permissionIDs...),
		},
	})

	return nil
}

func isBuiltinAdminRole(role rbacstore.Role) bool {
	return role.Builtin && role.Name == builtinAdminRoleName
}

func (w managementWriter) ReplaceRolesForUser(ctx context.Context, input rbacstore.ReplaceRolesForUserInput) error {
	if w.users == nil {
		return errors.New("user service is unavailable")
	}
	if w.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}
	user, err := w.users.GetUserByID(ctx, input.UserID)
	if err != nil {
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

	w.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "rbac.user.roles.replace",
		ResourceType: "user",
		ResourceID:   formatRBACAuditID(input.UserID),
		ResourceName: user.Username,
		Success:      true,
		Message:      "user roles replaced",
		Metadata: map[string]any{
			"role_ids": append([]uint64(nil), input.RoleIDs...),
		},
	})

	return nil
}

func (w managementWriter) AddRolesToUser(ctx context.Context, input rbacstore.AddRolesToUserInput) error {
	if err := w.ensureRoleMutationPreconditions(ctx, []uint64{input.UserID}, input.RoleIDs); err != nil {
		return err
	}
	user, err := w.users.GetUserByID(ctx, input.UserID)
	if err != nil {
		return err
	}
	if err := w.rbac.AddRolesToUser(ctx, input); err != nil {
		return err
	}

	w.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "rbac.user.roles.add",
		ResourceType: "user",
		ResourceID:   formatRBACAuditID(input.UserID),
		ResourceName: user.Username,
		Success:      true,
		Message:      "user roles added",
		Metadata: map[string]any{
			"role_ids": append([]uint64(nil), input.RoleIDs...),
		},
	})

	return nil
}

func (w managementWriter) RemoveRolesFromUser(ctx context.Context, input rbacstore.RemoveRolesFromUserInput) error {
	if err := w.ensureRoleMutationPreconditions(ctx, []uint64{input.UserID}, input.RoleIDs); err != nil {
		return err
	}
	user, err := w.users.GetUserByID(ctx, input.UserID)
	if err != nil {
		return err
	}
	if err := w.ensureActorCanRemoveRoles(ctx, input.UserID, input.RoleIDs); err != nil {
		return err
	}
	if err := w.rbac.RemoveRolesFromUser(ctx, input); err != nil {
		return err
	}

	w.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "rbac.user.roles.remove",
		ResourceType: "user",
		ResourceID:   formatRBACAuditID(input.UserID),
		ResourceName: user.Username,
		Success:      true,
		Message:      "user roles removed",
		Metadata: map[string]any{
			"role_ids": append([]uint64(nil), input.RoleIDs...),
		},
	})

	return nil
}

func (w managementWriter) ReplaceRolesForUsers(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
	return w.runBatchRoleMutation(
		ctx,
		input,
		w.ensureActorCanReplaceRoles,
		func(batchWriter batchUserRoleAtomicWriter, ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
			return batchWriter.ReplaceRolesForUsersAtomically(ctx, input)
		},
	)
}

func (w managementWriter) AddRolesToUsers(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
	return w.runBatchRoleMutation(
		ctx,
		input,
		func(context.Context, uint64, []uint64) error { return nil },
		func(batchWriter batchUserRoleAtomicWriter, ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
			return batchWriter.AddRolesToUsersAtomically(ctx, input)
		},
	)
}

func (w managementWriter) RemoveRolesFromUsers(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
	return w.runBatchRoleMutation(
		ctx,
		input,
		w.ensureActorCanRemoveRoles,
		func(batchWriter batchUserRoleAtomicWriter, ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
			return batchWriter.RemoveRolesFromUsersAtomically(ctx, input)
		},
	)
}

func (w managementWriter) ensureActorKeepsBuiltinAdminRole(ctx context.Context, input rbacstore.ReplaceRolesForUserInput) error {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
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
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
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

func (w managementWriter) ensureBatchRoleMutationAllowed(
	ctx context.Context,
	userIDs []uint64,
	roleIDs []uint64,
	check func(context.Context, uint64, []uint64) error,
) error {
	for _, userID := range userIDs {
		if err := check(ctx, userID, roleIDs); err != nil {
			return err
		}
	}
	return nil
}

func (w managementWriter) runBatchRoleMutation(
	ctx context.Context,
	input rbacstore.BatchUserRoleMutationInput,
	check func(context.Context, uint64, []uint64) error,
	runAtomic func(batchUserRoleAtomicWriter, context.Context, rbacstore.BatchUserRoleMutationInput) error,
) error {
	if err := w.ensureRoleMutationPreconditions(ctx, input.UserIDs, input.RoleIDs); err != nil {
		return err
	}
	if err := w.ensureBatchRoleMutationAllowed(ctx, input.UserIDs, input.RoleIDs, check); err != nil {
		return err
	}
	if batchWriter, ok := w.rbac.(batchUserRoleAtomicWriter); ok {
		return runAtomic(batchWriter, ctx, input)
	}
	return errAtomicBatchWriterMissing
}

func findBuiltinAdminRole(roles []rbacstore.Role) (rbacstore.Role, bool) {
	for _, role := range roles {
		if role.Builtin && strings.TrimSpace(role.Name) == builtinAdminRoleName {
			return role, true
		}
	}

	return rbacstore.Role{}, false
}

func (w managementWriter) publishAudit(ctx context.Context, event moduleapi.AuditEvent) {
	if w.auditBus == nil {
		return
	}

	event.Operator = currentRBACAuditOperator(ctx)
	if err := w.auditBus.Publish(ctx, eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  moduleID,
		Payload: event,
	}); err != nil && w.logger != nil {
		w.logger.Warn("publish rbac audit event failed",
			zap.String("module", moduleID),
			zap.String("action", strings.TrimSpace(event.Action)),
			zap.Error(err),
		)
	}
}

func currentRBACAuditOperator(ctx context.Context) *moduleapi.CurrentUser {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil {
		return nil
	}

	user := *requestAuth.User
	return &user
}

func formatRBACAuditID(id uint64) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatUint(id, 10)
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
