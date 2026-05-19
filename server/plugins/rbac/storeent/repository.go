package storeent

import (
	"context"
	"fmt"
	"math"

	"graft/server/internal/ent"
	entpermission "graft/server/internal/ent/permission"
	entrole "graft/server/internal/ent/role"
	entrolepermission "graft/server/internal/ent/rolepermission"
	entuserrole "graft/server/internal/ent/userrole"
	rbacstore "graft/server/plugins/rbac/store"
)

type repository struct {
	client *ent.Client
}

// NewRepository builds the RBAC plugin's Ent-backed repository.
func NewRepository(client *ent.Client) (rbacstore.Repository, error) {
	if client == nil {
		return nil, fmt.Errorf("rbac storeent requires a non-nil ent client")
	}

	return &repository{client: client}, nil
}

func (r *repository) EnsureRole(ctx context.Context, input rbacstore.EnsureRoleInput) (rbacstore.Role, error) {
	record, err := r.findRoleByName(ctx, input.Name)
	if err == nil {
		record, err = r.upgradeRoleBuiltinIfNeeded(ctx, record, input.Builtin, "upgrade ensured role builtin state")
		if err != nil {
			return rbacstore.Role{}, err
		}
		return toStoreRole(record), nil
	}
	if !ent.IsNotFound(err) {
		return rbacstore.Role{}, fmt.Errorf("query ensured role by name: %w", err)
	}

	record, err = r.createRoleRecord(ctx, input)
	if err != nil {
		if ent.IsConstraintError(err) {
			record, err = r.findRoleAfterCreateConflict(ctx, input)
			if err != nil {
				return rbacstore.Role{}, err
			}
			return toStoreRole(record), nil
		}

		return rbacstore.Role{}, fmt.Errorf("create ensured role: %w", err)
	}

	return toStoreRole(record), nil
}

func (r *repository) EnsurePermission(ctx context.Context, input rbacstore.EnsurePermissionInput) (rbacstore.Permission, error) {
	return ensureUniqueEntity(
		func() (*ent.Permission, error) {
			return r.client.Permission.Query().
				Where(entpermission.CodeEQ(input.Code)).
				Only(ctx)
		},
		func() (*ent.Permission, error) {
			return r.client.Permission.Create().
				SetCode(input.Code).
				SetDisplay(input.Display).
				SetNillableDescription(input.Description).
				SetCategory(input.Category).
				Save(ctx)
		},
		toStorePermission,
		"query ensured permission by code",
		"create ensured permission",
		"re-query ensured permission after conflict",
	)
}

func (r *repository) CreateRole(ctx context.Context, input rbacstore.CreateRoleInput) (rbacstore.Role, error) {
	record, err := r.client.Role.Create().
		SetName(input.Name).
		SetDisplay(input.Display).
		SetNillableDescription(input.Description).
		SetBuiltin(input.Builtin).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return rbacstore.Role{}, rbacstore.ErrRoleNameConflict
		}

		return rbacstore.Role{}, fmt.Errorf("create role: %w", err)
	}

	return toStoreRole(record), nil
}

func (r *repository) UpdateRole(ctx context.Context, input rbacstore.UpdateRoleInput) (rbacstore.Role, error) {
	roleID, err := toEntID(input.ID)
	if err != nil {
		return rbacstore.Role{}, err
	}

	record, err := r.client.Role.UpdateOneID(roleID).
		SetName(input.Name).
		SetDisplay(input.Display).
		SetNillableDescription(input.Description).
		Save(ctx)
	if err != nil {
		switch {
		case ent.IsNotFound(err):
			return rbacstore.Role{}, rbacstore.ErrRoleNotFound
		case ent.IsConstraintError(err):
			return rbacstore.Role{}, rbacstore.ErrRoleNameConflict
		default:
			return rbacstore.Role{}, fmt.Errorf("update role %d: %w", input.ID, err)
		}
	}

	return toStoreRole(record), nil
}

func (r *repository) AssignPermissionsToRole(ctx context.Context, input rbacstore.AssignPermissionsToRoleInput) error {
	roleID, err := toEntID(input.RoleID)
	if err != nil {
		return err
	}

	for _, permissionID := range input.PermissionIDs {
		entPermissionID, err := toEntID(permissionID)
		if err != nil {
			return err
		}

		exists, err := r.client.RolePermission.Query().
			Where(
				entrolepermission.RoleIDEQ(roleID),
				entrolepermission.PermissionIDEQ(entPermissionID),
			).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("check role permission assignment: %w", err)
		}
		if exists {
			continue
		}

		if _, err := r.client.RolePermission.Create().
			SetRoleID(roleID).
			SetPermissionID(entPermissionID).
			Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				continue
			}

			return fmt.Errorf("assign permission %d to role %d: %w", permissionID, input.RoleID, err)
		}
	}

	return nil
}

func (r *repository) ReplacePermissionsForRole(ctx context.Context, input rbacstore.ReplacePermissionsForRoleInput) error {
	return replaceStableAssignmentWithConfig(
		ctx,
		r.client,
		input.RoleID,
		input.PermissionIDs,
		buildRolePermissionAssignmentConfig,
	)
}

func (r *repository) AssignRoleToUser(ctx context.Context, input rbacstore.AssignRoleToUserInput) error {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return err
	}
	roleID, err := toEntID(input.RoleID)
	if err != nil {
		return err
	}

	exists, err := r.client.UserRole.Query().
		Where(
			entuserrole.UserIDEQ(userID),
			entuserrole.RoleIDEQ(roleID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check user role assignment: %w", err)
	}
	if exists {
		return nil
	}

	if _, err := r.client.UserRole.Create().
		SetUserID(userID).
		SetRoleID(roleID).
		Save(ctx); err != nil {
		if ent.IsConstraintError(err) {
			duplicate, duplicateErr := r.client.UserRole.Query().
				Where(
					entuserrole.UserIDEQ(userID),
					entuserrole.RoleIDEQ(roleID),
				).
				Exist(ctx)
			if duplicateErr == nil && duplicate {
				return nil
			}
		}

		return fmt.Errorf("assign role %d to user %d: %w", input.RoleID, input.UserID, err)
	}

	return nil
}

func (r *repository) ReplaceRolesForUser(ctx context.Context, input rbacstore.ReplaceRolesForUserInput) error {
	return replaceStableAssignmentWithConfig(
		ctx,
		r.client,
		input.UserID,
		input.RoleIDs,
		buildUserRoleAssignmentConfig,
	)
}

func (r *repository) GetRoleByID(ctx context.Context, roleID uint64) (rbacstore.Role, error) {
	id, err := toEntID(roleID)
	if err != nil {
		return rbacstore.Role{}, err
	}

	record, err := r.client.Role.Query().
		Where(entrole.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return rbacstore.Role{}, rbacstore.ErrRoleNotFound
		}

		return rbacstore.Role{}, fmt.Errorf("get role by id %d: %w", roleID, err)
	}

	return toStoreRole(record), nil
}

func (r *repository) ListRolesByUserID(ctx context.Context, userID uint64) ([]rbacstore.Role, error) {
	id, err := toEntID(userID)
	if err != nil {
		return nil, err
	}

	records, err := r.client.UserRole.Query().
		Where(entuserrole.UserIDEQ(id)).
		QueryRole().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles by user id: %w", err)
	}

	roles := make([]rbacstore.Role, 0, len(records))
	for _, record := range records {
		roles = append(roles, toStoreRole(record))
	}

	return roles, nil
}

func (r *repository) ListRoles(ctx context.Context) ([]rbacstore.Role, error) {
	records, err := r.client.Role.Query().
		Order(ent.Asc(entrole.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	roles := make([]rbacstore.Role, 0, len(records))
	for _, record := range records {
		roles = append(roles, toStoreRole(record))
	}

	return roles, nil
}

func (r *repository) ListPermissionsByUserID(ctx context.Context, userID uint64) ([]rbacstore.Permission, error) {
	id, err := toEntID(userID)
	if err != nil {
		return nil, err
	}

	roleRecords, err := r.client.UserRole.Query().
		Where(entuserrole.UserIDEQ(id)).
		QueryRole().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list user roles for permissions: %w", err)
	}
	if len(roleRecords) == 0 {
		return []rbacstore.Permission{}, nil
	}

	roleIDs := make([]int, 0, len(roleRecords))
	for _, roleRecord := range roleRecords {
		roleIDs = append(roleIDs, roleRecord.ID)
	}

	records, err := r.client.Permission.Query().
		Where(entpermission.HasRolePermissionsWith(entrolepermission.RoleIDIn(roleIDs...))).
		Unique(true).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions by user id: %w", err)
	}

	permissions := make([]rbacstore.Permission, 0, len(records))
	for _, record := range records {
		permissions = append(permissions, toStorePermission(record))
	}

	return permissions, nil
}

func (r *repository) ListPermissions(ctx context.Context) ([]rbacstore.Permission, error) {
	records, err := r.client.Permission.Query().
		Order(ent.Asc(entpermission.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}

	permissions := make([]rbacstore.Permission, 0, len(records))
	for _, record := range records {
		permissions = append(permissions, toStorePermission(record))
	}

	return permissions, nil
}

func (r *repository) ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]rbacstore.RolePermissionBinding, error) {
	id, err := toEntID(roleID)
	if err != nil {
		return nil, err
	}

	if _, err := r.client.Role.Get(ctx, id); err != nil {
		if ent.IsNotFound(err) {
			return nil, rbacstore.ErrRoleNotFound
		}
		return nil, fmt.Errorf("get role for permission bindings: %w", err)
	}

	records, err := r.client.RolePermission.Query().
		Where(entrolepermission.RoleIDEQ(id)).
		Order(ent.Asc(entrolepermission.FieldPermissionID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list role permission bindings: %w", err)
	}

	bindings := make([]rbacstore.RolePermissionBinding, 0, len(records))
	for _, record := range records {
		bindings = append(bindings, rbacstore.RolePermissionBinding{
			RoleID:       roleID,
			PermissionID: toStoreID(record.PermissionID),
		})
	}

	return bindings, nil
}

func (r *repository) findRoleByName(ctx context.Context, name string) (*ent.Role, error) {
	return r.client.Role.Query().
		Where(entrole.NameEQ(name)).
		Only(ctx)
}

func (r *repository) createRoleRecord(ctx context.Context, input rbacstore.EnsureRoleInput) (*ent.Role, error) {
	return r.client.Role.Create().
		SetName(input.Name).
		SetDisplay(input.Display).
		SetNillableDescription(input.Description).
		SetBuiltin(input.Builtin).
		Save(ctx)
}

func (r *repository) findRoleAfterCreateConflict(
	ctx context.Context,
	input rbacstore.EnsureRoleInput,
) (*ent.Role, error) {
	record, err := r.findRoleByName(ctx, input.Name)
	if err != nil {
		return nil, fmt.Errorf("re-query ensured role after conflict: %w", err)
	}

	return r.upgradeRoleBuiltinIfNeeded(ctx, record, input.Builtin, "upgrade ensured role builtin state after conflict")
}

func (r *repository) upgradeRoleBuiltinIfNeeded(
	ctx context.Context,
	record *ent.Role,
	builtin bool,
	errorContext string,
) (*ent.Role, error) {
	if !builtin || record.Builtin {
		return record, nil
	}

	updated, err := r.client.Role.UpdateOneID(record.ID).
		SetBuiltin(true).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errorContext, err)
	}

	return updated, nil
}

func toEntID(id uint64) (int, error) {
	if id == 0 || id > math.MaxInt {
		return 0, rbacstore.ErrInvalidID
	}

	return int(id), nil
}

func toStoreID(id int) uint64 {
	//nolint:gosec // Ent IDs come from the controlled schema and remain positive.
	return uint64(id)
}

func toUniqueEntIDs(ids []uint64) ([]int, error) {
	if len(ids) == 0 {
		return []int{}, nil
	}

	converted := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		entID, err := toEntID(id)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[entID]; ok {
			continue
		}

		seen[entID] = struct{}{}
		converted = append(converted, entID)
	}

	return converted, nil
}

type stableAssignmentSetConfig struct {
	startContext         string
	commitContext        string
	checkTargetContext   string
	countRelationContext string
	deleteStaleContext   string
	checkBindingContext  string
	createBindingContext string
	targetID             uint64
	relationIDs          []int
	relationCount        int
	targetMissing        error
	relationMissing      error
	checkTargetExists    func(tx *ent.Tx) (bool, error)
	countRelationRecords func(tx *ent.Tx, ids []int) (int, error)
	deleteStale          func(tx *ent.Tx, ids []int) error
	bindingExists        func(tx *ent.Tx, relationID int) (bool, error)
	createBinding        func(tx *ent.Tx, relationID int) error
}

type stableAssignmentConfigTemplate struct {
	startContext         string
	commitFormat         string
	checkTargetContext   string
	countRelationContext string
	deleteStaleContext   string
	checkBindingContext  string
	createBindingContext string
	targetMissing        error
	relationMissing      error
	checkTargetExists    func(context.Context, *ent.Tx, int) (bool, error)
	countRelationRecords func(context.Context, *ent.Tx, []int) (int, error)
	deleteStale          func(context.Context, *ent.Tx, int, []int) error
	bindingExists        func(context.Context, *ent.Tx, int, int) (bool, error)
	createBinding        func(context.Context, *ent.Tx, int, int) error
}

func replaceStableAssignmentWithConfig(
	ctx context.Context,
	client *ent.Client,
	targetID uint64,
	relationIDs []uint64,
	build func(ctx context.Context, targetID uint64, entTargetID int, relationIDs []int) stableAssignmentSetConfig,
) error {
	entTargetID, err := toEntID(targetID)
	if err != nil {
		return err
	}

	entRelationIDs, err := toUniqueEntIDs(relationIDs)
	if err != nil {
		return err
	}

	return replaceStableAssignmentSet(ctx, client, build(ctx, targetID, entTargetID, entRelationIDs))
}

func replaceStableAssignmentSet(
	ctx context.Context,
	client *ent.Client,
	config stableAssignmentSetConfig,
) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", config.startContext, err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	if err := ensureStableAssignmentTarget(tx, config); err != nil {
		return err
	}
	if err := validateStableAssignmentRelations(tx, config); err != nil {
		return err
	}
	if err := deleteStableAssignments(tx, config); err != nil {
		return err
	}
	if err := insertStableAssignments(tx, config); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", config.commitContext, err)
	}
	tx = nil

	return nil
}

func buildRolePermissionAssignmentConfig(
	ctx context.Context,
	targetID uint64,
	entTargetID int,
	entRelationIDs []int,
) stableAssignmentSetConfig {
	return buildStableAssignmentConfig(ctx, targetID, entTargetID, entRelationIDs, stableAssignmentConfigTemplate{
		startContext:         "start replace role permissions tx",
		commitFormat:         "commit replace role permissions for role %d",
		checkTargetContext:   "check role %d before replacing permissions",
		countRelationContext: "count permissions for role %d replacement",
		deleteStaleContext:   "delete stale permissions for role %d",
		checkBindingContext:  "check role permission replacement",
		createBindingContext: "replace permission %d for role %d",
		targetMissing:        rbacstore.ErrRoleNotFound,
		relationMissing:      rbacstore.ErrPermissionNotFound,
		checkTargetExists:    roleTargetExists,
		countRelationRecords: countPermissionsByIDs,
		deleteStale:          deleteStaleRolePermissions,
		bindingExists:        rolePermissionBindingExists,
		createBinding:        createRolePermissionBinding,
	})
}

func buildUserRoleAssignmentConfig(
	ctx context.Context,
	targetID uint64,
	entTargetID int,
	entRelationIDs []int,
) stableAssignmentSetConfig {
	return buildStableAssignmentConfig(ctx, targetID, entTargetID, entRelationIDs, stableAssignmentConfigTemplate{
		startContext:         "start replace user roles tx",
		commitFormat:         "commit replace user roles for user %d",
		checkTargetContext:   "check user %d before replacing roles",
		countRelationContext: "count roles for user %d replacement",
		deleteStaleContext:   "delete stale roles for user %d",
		checkBindingContext:  "check user role replacement",
		createBindingContext: "replace role %d for user %d",
		checkTargetExists:    userRoleTargetExists,
		targetMissing:        nil,
		relationMissing:      rbacstore.ErrRoleNotFound,
		countRelationRecords: countRolesByIDs,
		deleteStale:          deleteStaleUserRoles,
		bindingExists:        userRoleBindingExists,
		createBinding:        createUserRoleBinding,
	})
}

func buildStableAssignmentConfig(
	ctx context.Context,
	targetID uint64,
	entTargetID int,
	entRelationIDs []int,
	template stableAssignmentConfigTemplate,
) stableAssignmentSetConfig {
	return stableAssignmentSetConfig{
		startContext:         template.startContext,
		commitContext:        fmt.Sprintf(template.commitFormat, targetID),
		checkTargetContext:   template.checkTargetContext,
		countRelationContext: template.countRelationContext,
		deleteStaleContext:   template.deleteStaleContext,
		checkBindingContext:  template.checkBindingContext,
		createBindingContext: template.createBindingContext,
		targetID:             targetID,
		relationIDs:          entRelationIDs,
		relationCount:        len(entRelationIDs),
		targetMissing:        template.targetMissing,
		relationMissing:      template.relationMissing,
		checkTargetExists: func(tx *ent.Tx) (bool, error) {
			return template.checkTargetExists(ctx, tx, entTargetID)
		},
		countRelationRecords: func(tx *ent.Tx, ids []int) (int, error) {
			return template.countRelationRecords(ctx, tx, ids)
		},
		deleteStale: func(tx *ent.Tx, ids []int) error {
			return template.deleteStale(ctx, tx, entTargetID, ids)
		},
		bindingExists: func(tx *ent.Tx, relationID int) (bool, error) {
			return template.bindingExists(ctx, tx, entTargetID, relationID)
		},
		createBinding: func(tx *ent.Tx, relationID int) error {
			return template.createBinding(ctx, tx, entTargetID, relationID)
		},
	}
}

func roleTargetExists(ctx context.Context, tx *ent.Tx, targetID int) (bool, error) {
	return tx.Role.Query().Where(entrole.IDEQ(targetID)).Exist(ctx)
}

func countPermissionsByIDs(ctx context.Context, tx *ent.Tx, ids []int) (int, error) {
	return tx.Permission.Query().Where(entpermission.IDIn(ids...)).Count(ctx)
}

func deleteStaleRolePermissions(ctx context.Context, tx *ent.Tx, targetID int, ids []int) error {
	deleteQuery := tx.RolePermission.Delete().Where(entrolepermission.RoleIDEQ(targetID))
	if len(ids) > 0 {
		deleteQuery = deleteQuery.Where(entrolepermission.Not(entrolepermission.PermissionIDIn(ids...)))
	}
	_, err := deleteQuery.Exec(ctx)
	return err
}

func rolePermissionBindingExists(ctx context.Context, tx *ent.Tx, targetID int, relationID int) (bool, error) {
	return tx.RolePermission.Query().
		Where(
			entrolepermission.RoleIDEQ(targetID),
			entrolepermission.PermissionIDEQ(relationID),
		).
		Exist(ctx)
}

func createRolePermissionBinding(ctx context.Context, tx *ent.Tx, targetID int, relationID int) error {
	_, err := tx.RolePermission.Create().SetRoleID(targetID).SetPermissionID(relationID).Save(ctx)
	return err
}

func userRoleTargetExists(context.Context, *ent.Tx, int) (bool, error) {
	return true, nil
}

func countRolesByIDs(ctx context.Context, tx *ent.Tx, ids []int) (int, error) {
	return tx.Role.Query().Where(entrole.IDIn(ids...)).Count(ctx)
}

func deleteStaleUserRoles(ctx context.Context, tx *ent.Tx, targetID int, ids []int) error {
	deleteQuery := tx.UserRole.Delete().Where(entuserrole.UserIDEQ(targetID))
	if len(ids) > 0 {
		deleteQuery = deleteQuery.Where(entuserrole.Not(entuserrole.RoleIDIn(ids...)))
	}
	_, err := deleteQuery.Exec(ctx)
	return err
}

func userRoleBindingExists(ctx context.Context, tx *ent.Tx, targetID int, relationID int) (bool, error) {
	return tx.UserRole.Query().
		Where(
			entuserrole.UserIDEQ(targetID),
			entuserrole.RoleIDEQ(relationID),
		).
		Exist(ctx)
}

func createUserRoleBinding(ctx context.Context, tx *ent.Tx, targetID int, relationID int) error {
	_, err := tx.UserRole.Create().SetUserID(targetID).SetRoleID(relationID).Save(ctx)
	return err
}

func ensureStableAssignmentTarget(tx *ent.Tx, config stableAssignmentSetConfig) error {
	exists, err := config.checkTargetExists(tx)
	if err != nil {
		return fmt.Errorf(config.checkTargetContext+": %w", config.targetID, err)
	}
	if !exists && config.targetMissing != nil {
		return config.targetMissing
	}

	return nil
}

func validateStableAssignmentRelations(tx *ent.Tx, config stableAssignmentSetConfig) error {
	if config.relationCount == 0 {
		return nil
	}

	count, err := config.countRelationRecords(tx, config.relationIDs)
	if err != nil {
		return fmt.Errorf(config.countRelationContext+": %w", config.targetID, err)
	}
	if count != config.relationCount {
		return config.relationMissing
	}

	return nil
}

func deleteStableAssignments(tx *ent.Tx, config stableAssignmentSetConfig) error {
	if err := config.deleteStale(tx, config.relationIDs); err != nil {
		return fmt.Errorf(config.deleteStaleContext+": %w", config.targetID, err)
	}

	return nil
}

func insertStableAssignments(tx *ent.Tx, config stableAssignmentSetConfig) error {
	for _, relationID := range config.relationIDs {
		exists, err := config.bindingExists(tx, relationID)
		if err != nil {
			return fmt.Errorf("%s: %w", config.checkBindingContext, err)
		}
		if exists {
			continue
		}

		if err := config.createBinding(tx, relationID); err != nil {
			if ent.IsConstraintError(err) {
				continue
			}

			return fmt.Errorf(config.createBindingContext+": %w", relationID, config.targetID, err)
		}
	}

	return nil
}

func ensureUniqueEntity[T any](
	query func() (T, error),
	create func() (T, error),
	mapResult func(T) rbacstore.Permission,
	queryContext string,
	createContext string,
	conflictQueryContext string,
) (rbacstore.Permission, error) {
	record, err := query()
	if err == nil {
		return mapResult(record), nil
	}
	if !ent.IsNotFound(err) {
		return rbacstore.Permission{}, fmt.Errorf("%s: %w", queryContext, err)
	}

	record, err = create()
	if err != nil {
		if ent.IsConstraintError(err) {
			record, err = query()
			if err != nil {
				return rbacstore.Permission{}, fmt.Errorf("%s: %w", conflictQueryContext, err)
			}

			return mapResult(record), nil
		}

		return rbacstore.Permission{}, fmt.Errorf("%s: %w", createContext, err)
	}

	return mapResult(record), nil
}

func toStoreRole(record *ent.Role) rbacstore.Role {
	return rbacstore.Role{
		ID:          toStoreID(record.ID),
		Name:        record.Name,
		Display:     record.Display,
		Description: record.Description,
		Builtin:     record.Builtin,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

func toStorePermission(record *ent.Permission) rbacstore.Permission {
	return rbacstore.Permission{
		ID:          toStoreID(record.ID),
		Code:        record.Code,
		Display:     record.Display,
		Description: record.Description,
		Category:    record.Category,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}
