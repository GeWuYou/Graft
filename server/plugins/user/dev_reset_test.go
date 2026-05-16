package user

import (
	"context"
	"strings"
	"testing"
	"time"

	"graft/server/internal/store"
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

	if err := ResetDefaultAdminForDevelopment(context.Background(), state.authRepo, state.rbacRepo); err != nil {
		t.Fatalf("reset default admin: %v", err)
	}

	assertDevResetState(t, state)
}

func TestResetDefaultAdminForDevelopmentRejectsNonDevelopmentEnv(t *testing.T) {
	t.Setenv("GRAFT_APP_ENV", "production")

	state := newDevResetState(t, "unused")
	err := ResetDefaultAdminForDevelopment(context.Background(), state.authRepo, state.rbacRepo)
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
	setPasswordInput       store.SetPasswordHashInput
	assignRoleInput        store.AssignRoleToUserInput
	assignPermissionsInput store.AssignPermissionsToRoleInput
	authRepo               *pluginTestAuthRepository
	rbacRepo               pluginTestRBACRepository
}

func newDevResetState(t *testing.T, currentHash string) *devResetState {
	t.Helper()

	state := &devResetState{}
	state.authRepo = &pluginTestAuthRepository{
		ensureUserCredential: func(_ context.Context, input store.EnsureUserCredentialInput) (store.UserCredential, error) {
			state.ensured = true
			return store.UserCredential{
				UserID:             9,
				Username:           input.Username,
				PasswordHash:       &currentHash,
				MustChangePassword: false,
			}, nil
		},
		setPasswordHash: func(_ context.Context, input store.SetPasswordHashInput) error {
			state.setPasswordInput = input
			return nil
		},
	}
	state.authRepo.refreshSessions = map[string]store.RefreshSession{
		"session-a": {
			UserID:    9,
			TokenID:   "session-a",
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		},
	}
	state.rbacRepo = pluginTestRBACRepository{
		ensureRole: func(_ context.Context, input store.EnsureRoleInput) (store.Role, error) {
			if !input.Builtin {
				t.Fatal("expected development reset to keep the default admin role builtin")
			}
			return store.Role{ID: 3, Name: input.Name, Display: input.Display}, nil
		},
		ensurePermission: func(_ context.Context, input store.EnsurePermissionInput) (store.Permission, error) {
			return store.Permission{ID: uint64(len(input.Code)), Code: input.Code, Display: input.Display}, nil
		},
		assignPermissionsToRole: func(_ context.Context, input store.AssignPermissionsToRoleInput) error {
			state.assignPermissionsInput = input
			return nil
		},
		assignRoleToUser: func(_ context.Context, input store.AssignRoleToUserInput) error {
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
}
