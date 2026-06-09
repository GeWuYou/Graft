// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"context"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
	rbacstore "graft/server/modules/rbac/store"
	userstore "graft/server/modules/user/store"
)

// TestResetDefaultAdminForDevelopmentResetsCredentialAndRole 验证 dev-only 重置会把
// 默认管理员恢复到初始化密码 + 必须改密状态，并补齐角色绑定。
func TestResetDefaultAdminForDevelopmentResetsCredentialAndRole(t *testing.T) {
	t.Setenv("GRAFT_APP_ENV", "local")

	currentHash, err := newPasswordHasher().Hash("custom-password-123")
	if err != nil {
		t.Fatalf("hash existing password: %v", err)
	}

	state := newDevResetState(t, currentHash)

	if err := ResetDefaultAdminForDevelopment(
		context.Background(),
		state.authRepo,
		devResetRBACBootstrapStub{state: state},
	); err != nil {
		t.Fatalf("reset default admin: %v", err)
	}

	assertDevResetState(t, state)
}

func TestResetDefaultAdminForDevelopmentRejectsNonDevelopmentEnv(t *testing.T) {
	t.Setenv("GRAFT_APP_ENV", "production")

	state := newDevResetState(t, "unused")
	err := ResetDefaultAdminForDevelopment(
		context.Background(),
		state.authRepo,
		devResetRBACBootstrapStub{state: state},
	)
	if err == nil {
		t.Fatal("expected development env guard error")
	}
	if !strings.Contains(err.Error(), "only available in local/test environments") {
		t.Fatalf("expected development env guard, got %v", err)
	}
	if state.ensured {
		t.Fatal("did not expect reset flow to touch repositories outside local/test env")
	}
}

type devResetState struct {
	ensured                bool
	setPasswordInput       userstore.SetPasswordHashInput
	assignRoleInput        rbacstore.AssignRoleToUserInput
	assignPermissionsInput rbacstore.AssignPermissionsToRoleInput
	authRepo               *moduleTestAuthRepository
	rbacRepo               moduleTestRBACRepository
}

type devResetRBACBootstrapStub struct {
	state *devResetState
}

func (s devResetRBACBootstrapStub) EnsureDefaultAdminAccess(ctx context.Context, userID uint64, permissions []moduleapi.PermissionSeed) error {
	role, err := s.state.rbacRepo.EnsureRole(ctx, rbacstore.EnsureRoleInput{
		Name:    "admin",
		Display: "管理员",
		Builtin: true,
	})
	if err != nil {
		return err
	}

	permissionIDs := make([]uint64, 0, len(permissions))
	for _, item := range permissions {
		record, err := s.state.rbacRepo.EnsurePermission(ctx, rbacstore.EnsurePermissionInput{
			Code:        item.Code,
			Display:     item.Display,
			Description: devResetStringPtrOrNil(item.Description),
			Category:    item.Category,
		})
		if err != nil {
			return err
		}
		permissionIDs = append(permissionIDs, record.ID)
	}
	if len(permissionIDs) > 0 {
		if err := s.state.rbacRepo.AssignPermissionsToRole(ctx, rbacstore.AssignPermissionsToRoleInput{
			RoleID:        role.ID,
			PermissionIDs: permissionIDs,
		}); err != nil {
			return err
		}
	}

	return s.state.rbacRepo.AssignRoleToUser(ctx, rbacstore.AssignRoleToUserInput{
		UserID: userID,
		RoleID: role.ID,
	})
}

func devResetStringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	result := value
	return &result
}

func newDevResetState(t *testing.T, currentHash string) *devResetState {
	t.Helper()

	state := &devResetState{}
	state.authRepo = &moduleTestAuthRepository{
		ensureUserCredential: func(_ context.Context, input userstore.EnsureUserCredentialInput) (userstore.UserCredential, error) {
			state.ensured = true
			return userstore.UserCredential{
				UserID:             9,
				Username:           input.Username,
				PasswordHash:       &currentHash,
				MustChangePassword: false,
			}, nil
		},
		setPasswordHash: func(_ context.Context, input userstore.SetPasswordHashInput) error {
			state.setPasswordInput = input
			return nil
		},
	}
	state.authRepo.refreshSessions = map[string]userstore.RefreshSession{
		"session-a": {
			UserID:    9,
			TokenID:   "session-a",
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		},
	}
	state.rbacRepo = moduleTestRBACRepository{
		ensureRole: func(_ context.Context, input rbacstore.EnsureRoleInput) (rbacstore.Role, error) {
			if !input.Builtin {
				t.Fatal("expected development reset to keep the default admin role builtin")
			}
			return rbacstore.Role{ID: 3, Name: input.Name, Display: input.Display}, nil
		},
		ensurePermission: func(_ context.Context, input rbacstore.EnsurePermissionInput) (rbacstore.Permission, error) {
			return rbacstore.Permission{ID: uint64(len(input.Code)), Code: input.Code, Display: input.Display}, nil
		},
		assignPermissionsToRole: func(_ context.Context, input rbacstore.AssignPermissionsToRoleInput) error {
			state.assignPermissionsInput = input
			return nil
		},
		assignRoleToUser: func(_ context.Context, input rbacstore.AssignRoleToUserInput) error {
			state.assignRoleInput = input
			return nil
		},
	}

	return state
}

func assertDevResetState(t *testing.T, state *devResetState) {
	t.Helper()

	if !state.ensured {
		t.Fatal("expected default admin ensure credential to be called")
	}
	assertDevResetPasswordState(t, state)
	assertDevResetRoleState(t, state)
}

func assertDevResetPasswordState(t *testing.T, state *devResetState) {
	t.Helper()

	if state.setPasswordInput.UserID != 9 {
		t.Fatalf("expected reset user id 9, got %d", state.setPasswordInput.UserID)
	}
	if !state.setPasswordInput.MustChangePassword {
		t.Fatal("expected must_change_password to be restored to true")
	}
	if state.setPasswordInput.ChangedAt == nil || state.setPasswordInput.ChangedAt.IsZero() {
		t.Fatal("expected changed_at to be populated for reset flow")
	}
	if err := newPasswordHasher().Compare(state.setPasswordInput.PasswordHash, defaultAdminPassword); err != nil {
		t.Fatalf("expected reset password hash to match default admin password: %v", err)
	}
	if session, ok := state.authRepo.refreshSessions["session-a"]; !ok || session.RevokedAt == nil {
		t.Fatalf("expected existing refresh session to be revoked, got %#v", state.authRepo.refreshSessions["session-a"])
	}
}

func assertDevResetRoleState(t *testing.T, state *devResetState) {
	t.Helper()

	if state.assignRoleInput.UserID != 9 || state.assignRoleInput.RoleID != 3 {
		t.Fatalf("expected role binding to user 9 / role 3, got %#v", state.assignRoleInput)
	}
	if state.assignPermissionsInput.RoleID != 3 || len(state.assignPermissionsInput.PermissionIDs) == 0 {
		t.Fatalf("expected role permissions to be assigned, got %#v", state.assignPermissionsInput)
	}
	if len(state.assignPermissionsInput.PermissionIDs) != len(userPermissionItems("user")) {
		t.Fatalf("expected minimal admin access to match user module permissions, got %#v", state.assignPermissionsInput)
	}
}
