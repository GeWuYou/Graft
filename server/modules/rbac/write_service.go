package rbac

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"graft/server/internal/event"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	rbaccontract "graft/server/modules/rbac/contract"
	rbacstore "graft/server/modules/rbac/store"
)

var (
	errBuiltinRoleNameImmutable    = errors.New("builtin role name is immutable")
	errCannotRemoveOwnAdminRole    = errors.New("cannot remove own admin role")
	errInvalidPermissionIDs        = errors.New("invalid permission ids")
	errInvalidRoleIDs              = errors.New("invalid role ids")
	errAtomicBatchWriterMissing    = errors.New("rbac atomic batch writer is unavailable")
	errAtomicAuditWriterMissing    = errors.New("rbac atomic audit writer is unavailable")
	errAtomicAuditPublisherMissing = errors.New("rbac transactional audit publisher is unavailable")
	errProtectedUserRoleMutation   = errors.New("protected user role mutation is forbidden")
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
	users  moduleapi.UserService
	rbac   rbacstore.Repository
	events event.TransactionalPublisher
}

type rolePermissionAuditLabels struct {
	action     string
	messageKey string
	message    string
}

type roleAuditLabels struct {
	action     string
	messageKey string
	message    string
	metadata   func(rbacstore.Role) map[string]any
}

type bindingReplacement struct {
	run        func(context.Context) error
	validate   func(context.Context) error
	isMissing  func(error) bool
	fallback   error
	auditEvent moduleapi.AuditEvent
}

type batchRoleAuditLabels struct {
	action     string
	messageKey string
	message    string
}

func (w managementWriter) CreateRole(ctx context.Context, input rbacstore.CreateRoleInput) (rbacstore.Role, error) {
	if w.rbac == nil {
		return rbacstore.Role{}, errors.New("rbac repository is unavailable")
	}

	return w.runRoleMutation(ctx, func(txCtx context.Context) (rbacstore.Role, error) {
		return w.rbac.CreateRole(txCtx, input)
	}, roleAuditLabels{
		action: "rbac.role.create", messageKey: "rbac.audit.roleCreated", message: "role created", metadata: roleAuditMetadata,
	})
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

	return w.runRoleMutation(ctx, func(txCtx context.Context) (rbacstore.Role, error) {
		return w.rbac.UpdateRole(txCtx, input)
	}, roleAuditLabels{
		action: "rbac.role.update", messageKey: "rbac.audit.roleUpdated", message: "role updated", metadata: roleAuditMetadata,
	})
}

func (w managementWriter) SetRoleStatus(ctx context.Context, input rbacstore.SetRoleStatusInput) (rbacstore.Role, error) {
	if w.rbac == nil {
		return rbacstore.Role{}, errors.New("rbac repository is unavailable")
	}

	return w.runRoleMutation(ctx, func(txCtx context.Context) (rbacstore.Role, error) {
		return w.rbac.SetRoleStatus(txCtx, input)
	}, roleAuditLabels{
		action: "rbac.role.status.update", messageKey: "rbac.audit.roleStatusUpdated", message: "role status updated",
		metadata: func(role rbacstore.Role) map[string]any { return map[string]any{"status": role.Status} },
	})
}

func (w managementWriter) SoftDeleteRole(ctx context.Context, input rbacstore.SoftDeleteRoleInput) error {
	if w.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}

	role, err := w.rbac.GetRoleByID(ctx, input.ID)
	if err != nil {
		return err
	}
	return w.runAtomicAudit(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		if err := w.rbac.SoftDeleteRole(txCtx, input); err != nil {
			return err
		}
		return w.publishAuditTx(txCtx, tx, moduleapi.AuditEvent{
			Action: "rbac.role.delete", ResourceType: "role", ResourceID: formatRBACAuditID(role.ID), ResourceName: role.Name,
			Success: true, MessageKey: "rbac.audit.roleDeleted", Message: "role deleted",
		})
	})
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

	return w.runBindingReplacement(ctx, newBindingReplacement(
		func(txCtx context.Context) error { return w.rbac.ReplacePermissionsForRole(txCtx, input) },
		func(txCtx context.Context) error { return ensurePermissionIDsExist(txCtx, w.rbac, input.PermissionIDs) },
		func(err error) bool { return errors.Is(err, rbacstore.ErrPermissionNotFound) },
		errInvalidPermissionIDs,
		permissionReplacementAuditEvent(input, role),
	))
}

func (w managementWriter) AddPermissionsToRole(ctx context.Context, input rbacstore.AddPermissionsToRoleInput) error {
	return w.mutateRolePermissions(
		ctx,
		input.RoleID,
		input.PermissionIDs,
		rolePermissionAuditLabels{
			action:     "rbac.role.permissions.add",
			messageKey: rbaccontract.AuditRolePermissionsAdded.String(),
			message:    "role permissions added",
		},
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
		rolePermissionAuditLabels{
			action:     "rbac.role.permissions.remove",
			messageKey: rbaccontract.AuditRolePermissionsRemoved.String(),
			message:    "role permissions removed",
		},
		func(ctx context.Context) error {
			return w.rbac.RemovePermissionsFromRole(ctx, input)
		},
	)
}

func (w managementWriter) mutateRolePermissions(
	ctx context.Context,
	roleID uint64,
	permissionIDs []uint64,
	auditLabels rolePermissionAuditLabels,
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
	return w.runAtomicAudit(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		if err := run(txCtx); err != nil {
			return err
		}
		return w.publishAuditTx(txCtx, tx, moduleapi.AuditEvent{
			Action: auditLabels.action, ResourceType: "role", ResourceID: formatRBACAuditID(roleID), ResourceName: role.Name,
			Success: true, MessageKey: auditLabels.messageKey, Message: auditLabels.message,
			Metadata: map[string]any{"permission_ids": append([]uint64(nil), permissionIDs...)},
		})
	})
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
	if user.ProtectedDefaultAdmin {
		return errProtectedUserRoleMutation
	}
	if err := w.ensureActorKeepsBuiltinAdminRole(ctx, input); err != nil {
		return err
	}

	return w.runBindingReplacement(ctx, newBindingReplacement(
		func(txCtx context.Context) error { return w.rbac.ReplaceRolesForUser(txCtx, input) },
		func(txCtx context.Context) error { return ensureRoleIDsExist(txCtx, w.rbac, input.RoleIDs) },
		func(err error) bool { return errors.Is(err, rbacstore.ErrRoleNotFound) },
		errInvalidRoleIDs,
		userRoleReplacementAuditEvent(input, user),
	))
}

func (w managementWriter) AddRolesToUser(ctx context.Context, input rbacstore.AddRolesToUserInput) error {
	user, err := w.ensureSingleRoleMutationPreconditions(ctx, input.UserID, input.RoleIDs)
	if err != nil {
		return err
	}
	return w.runAtomicAudit(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		if err := w.rbac.AddRolesToUser(txCtx, input); err != nil {
			return err
		}
		return w.publishAuditTx(txCtx, tx, moduleapi.AuditEvent{
			Action: "rbac.user.roles.add", ResourceType: "user", ResourceID: formatRBACAuditID(input.UserID), ResourceName: user.Username,
			Success: true, MessageKey: "rbac.audit.userRolesAdded", Message: "user roles added",
			Metadata: map[string]any{"role_ids": append([]uint64(nil), input.RoleIDs...)},
		})
	})
}

func (w managementWriter) RemoveRolesFromUser(ctx context.Context, input rbacstore.RemoveRolesFromUserInput) error {
	user, err := w.ensureSingleRoleMutationPreconditions(ctx, input.UserID, input.RoleIDs)
	if err != nil {
		return err
	}
	if err := w.ensureActorCanRemoveRoles(ctx, input.UserID, input.RoleIDs); err != nil {
		return err
	}
	return w.runAtomicAudit(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		if err := w.rbac.RemoveRolesFromUser(txCtx, input); err != nil {
			return err
		}
		return w.publishAuditTx(txCtx, tx, moduleapi.AuditEvent{
			Action: "rbac.user.roles.remove", ResourceType: "user", ResourceID: formatRBACAuditID(input.UserID), ResourceName: user.Username,
			Success: true, MessageKey: "rbac.audit.userRolesRemoved", Message: "user roles removed",
			Metadata: map[string]any{"role_ids": append([]uint64(nil), input.RoleIDs...)},
		})
	})
}

func (w managementWriter) ReplaceRolesForUsers(ctx context.Context, input rbacstore.BatchUserRoleMutationInput) error {
	return w.runBatchRoleMutation(
		ctx,
		input,
		batchRoleAuditLabels{action: "rbac.user.roles.replace.batch", messageKey: "rbac.audit.userRolesReplaced", message: "user roles replaced in batch"},
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
		batchRoleAuditLabels{action: "rbac.user.roles.add.batch", messageKey: "rbac.audit.userRolesAdded", message: "user roles added in batch"},
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
		batchRoleAuditLabels{action: "rbac.user.roles.remove.batch", messageKey: "rbac.audit.userRolesRemoved", message: "user roles removed in batch"},
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
		if _, err := w.requireRoleMutationUser(ctx, userID); err != nil {
			return err
		}
	}
	if err := ensureRoleIDsExist(ctx, w.rbac, roleIDs); err != nil {
		return err
	}
	return nil
}

func (w managementWriter) ensureSingleRoleMutationPreconditions(
	ctx context.Context,
	userID uint64,
	roleIDs []uint64,
) (moduleapi.UserSummary, error) {
	if w.users == nil {
		return moduleapi.UserSummary{}, errors.New("user service is unavailable")
	}
	if w.rbac == nil {
		return moduleapi.UserSummary{}, errors.New("rbac repository is unavailable")
	}
	user, err := w.requireRoleMutationUser(ctx, userID)
	if err != nil {
		return moduleapi.UserSummary{}, err
	}
	if err := ensureRoleIDsExist(ctx, w.rbac, roleIDs); err != nil {
		return moduleapi.UserSummary{}, err
	}
	return user, nil
}

func (w managementWriter) requireRoleMutationUser(ctx context.Context, userID uint64) (moduleapi.UserSummary, error) {
	user, err := w.users.GetUserByID(ctx, userID)
	if err != nil {
		return moduleapi.UserSummary{}, err
	}
	if user.ProtectedDefaultAdmin {
		return moduleapi.UserSummary{}, errProtectedUserRoleMutation
	}
	return user, nil
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
	auditLabels batchRoleAuditLabels,
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
		return w.runAtomicAudit(ctx, func(txCtx context.Context, tx *sql.Tx) error {
			if err := runAtomic(batchWriter, txCtx, input); err != nil {
				return err
			}
			return w.publishAuditTx(txCtx, tx, moduleapi.AuditEvent{
				Action: auditLabels.action, ResourceType: "user_role_batch", Success: true, MessageKey: auditLabels.messageKey, Message: auditLabels.message,
				Metadata: map[string]any{"user_ids": append([]uint64(nil), input.UserIDs...), "role_ids": append([]uint64(nil), input.RoleIDs...)},
			})
		})
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

func (w managementWriter) runRoleMutation(
	ctx context.Context,
	mutate func(context.Context) (rbacstore.Role, error),
	auditLabels roleAuditLabels,
) (rbacstore.Role, error) {
	var role rbacstore.Role
	err := w.runAtomicAudit(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		updated, err := mutate(txCtx)
		if err != nil {
			return err
		}
		role = updated
		metadata := map[string]any(nil)
		if auditLabels.metadata != nil {
			metadata = auditLabels.metadata(role)
		}
		return w.publishAuditTx(txCtx, tx, moduleapi.AuditEvent{
			Action: auditLabels.action, ResourceType: "role", ResourceID: formatRBACAuditID(role.ID), ResourceName: role.Name,
			Success: true, MessageKey: auditLabels.messageKey, Message: auditLabels.message, Metadata: metadata,
		})
	})
	if err != nil {
		return rbacstore.Role{}, err
	}
	return role, nil
}

func roleAuditMetadata(role rbacstore.Role) map[string]any {
	return map[string]any{"display_name": role.Display, "builtin": role.Builtin, "status": role.Status}
}

func newBindingReplacement(
	run func(context.Context) error,
	validate func(context.Context) error,
	isMissing func(error) bool,
	fallback error,
	auditEvent moduleapi.AuditEvent,
) bindingReplacement {
	return bindingReplacement{
		run: run, validate: validate, isMissing: isMissing, fallback: fallback, auditEvent: auditEvent,
	}
}

func permissionReplacementAuditEvent(input rbacstore.ReplacePermissionsForRoleInput, role rbacstore.Role) moduleapi.AuditEvent {
	return moduleapi.AuditEvent{
		Action: "rbac.role.permissions.replace", ResourceType: "role", ResourceID: formatRBACAuditID(input.RoleID), ResourceName: role.Name,
		Success: true, MessageKey: "rbac.audit.rolePermissionsReplaced", Message: "role permissions replaced",
		Metadata: map[string]any{"permission_ids": append([]uint64(nil), input.PermissionIDs...)},
	}
}

func userRoleReplacementAuditEvent(input rbacstore.ReplaceRolesForUserInput, user moduleapi.UserSummary) moduleapi.AuditEvent {
	return moduleapi.AuditEvent{
		Action: "rbac.user.roles.replace", ResourceType: "user", ResourceID: formatRBACAuditID(input.UserID), ResourceName: user.Username,
		Success: true, MessageKey: "rbac.audit.userRolesReplaced", Message: "user roles replaced",
		Metadata: map[string]any{"role_ids": append([]uint64(nil), input.RoleIDs...)},
	}
}

func (w managementWriter) runBindingReplacement(ctx context.Context, replacement bindingReplacement) error {
	return w.runAtomicAudit(ctx, func(txCtx context.Context, tx *sql.Tx) error {
		if err := replacement.run(txCtx); err != nil {
			if !replacement.isMissing(err) {
				return err
			}
			if validationErr := replacement.validate(txCtx); validationErr != nil {
				return validationErr
			}
			return replacement.fallback
		}
		return w.publishAuditTx(txCtx, tx, replacement.auditEvent)
	})
}

// runAtomicAudit 将 RBAC 写入与 durable audit event 固定在同一事务中，发布失败必须触发回滚。
func (w managementWriter) runAtomicAudit(ctx context.Context, mutation func(context.Context, *sql.Tx) error) error {
	runner, ok := w.rbac.(rbacstore.TransactionRunner)
	if !ok {
		return errAtomicAuditWriterMissing
	}
	return runner.RunInTransaction(ctx, mutation)
}

// publishAuditTx 只在已经由 RBAC repository 创建的事务内写入 durable audit event。
// 缺少发布器同样返回错误，确保业务写入不会脱离审计事实单独提交。
func (w managementWriter) publishAuditTx(ctx context.Context, tx *sql.Tx, payload moduleapi.AuditEvent) error {
	if w.events == nil {
		return errAtomicAuditPublisherMissing
	}
	payload.Operator = currentRBACAuditOperator(ctx)
	envelope, err := httpx.NewAuditEvent(moduleID, payload)
	if err != nil {
		return err
	}
	_, err = w.events.PublishTx(ctx, tx, envelope, event.PublishOptions{Delivery: event.DeliveryDurable})
	return err
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
