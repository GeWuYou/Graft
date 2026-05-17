package entstore

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	localent "graft/server/internal/ent"
	"graft/server/internal/ent/enttest"
	"graft/server/internal/ent/hook"
	"graft/server/internal/ent/refreshsession"
	"graft/server/internal/store"
)

// TestToEntIDRejectsInvalidValues 验证无效稳定标识会返回统一的参数错误。
func TestToEntIDRejectsInvalidValues(t *testing.T) {
	testCases := []struct {
		name string
		id   uint64
	}{
		{name: "zero", id: 0},
		{name: "overflow", id: uint64(math.MaxInt) + 1},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := toEntID(testCase.id)
			if !errors.Is(err, store.ErrInvalidID) {
				t.Fatalf("expected ErrInvalidID, got %v", err)
			}
		})
	}
}

// TestUserRepositoryGetByIDMapsInvalidIDToNotFound 验证用户读取仍对上层维持稳定未命中语义。
func TestUserRepositoryGetByIDMapsInvalidIDToNotFound(t *testing.T) {
	repo := &userRepository{}

	_, err := repo.GetByID(context.Background(), 0)
	if !errors.Is(err, store.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

// TestAuthRepositorySetPasswordHashMapsInvalidIDToNotFound 验证口令更新会把无效用户标识映射为稳定领域错误。
func TestAuthRepositorySetPasswordHashMapsInvalidIDToNotFound(t *testing.T) {
	repo := &authRepository{}

	err := repo.SetPasswordHash(context.Background(), store.SetPasswordHashInput{
		UserID:       0,
		PasswordHash: "hash",
	})
	if !errors.Is(err, store.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

// TestAuthRepositorySetPasswordHashPreservesPasswordChangedAtWhenChangedAtNil 验证
// 可选 ChangedAt 为空时不会清空既有 password_changed_at。
func TestAuthRepositorySetPasswordHashPreservesPasswordChangedAtWhenChangedAtNil(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:set-password-preserve-changed-at?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	changedAt := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	passwordHash := "old-hash"
	record, err := client.User.Create().
		SetUsername("alice").
		SetDisplay("Alice").
		SetPasswordHash(passwordHash).
		SetMustChangePassword(false).
		SetPasswordChangedAt(changedAt).
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	repo := &authRepository{client: client}
	if err := repo.SetPasswordHash(context.Background(), store.SetPasswordHashInput{
		UserID:             toStoreID(record.ID),
		PasswordHash:       "new-hash",
		MustChangePassword: true,
		ChangedAt:          nil,
	}); err != nil {
		t.Fatalf("set password hash: %v", err)
	}

	reloaded, err := client.User.Get(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.PasswordHash == nil || *reloaded.PasswordHash != "new-hash" {
		t.Fatalf("expected password hash to update, got %#v", reloaded.PasswordHash)
	}
	if !reloaded.MustChangePassword {
		t.Fatalf("expected must_change_password to update, got %#v", reloaded)
	}
	if reloaded.PasswordChangedAt == nil || !reloaded.PasswordChangedAt.Equal(changedAt) {
		t.Fatalf("expected password_changed_at to stay %v, got %#v", changedAt, reloaded.PasswordChangedAt)
	}
}

// TestAuthRepositorySetPasswordHashOverwritesPasswordChangedAtWhenChangedAtProvided 验证
// 显式 ChangedAt 仍会覆盖既有 password_changed_at。
func TestAuthRepositorySetPasswordHashOverwritesPasswordChangedAtWhenChangedAtProvided(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:set-password-overwrite-changed-at?mode=memory&cache=shared&_fk=1")
	defer func() { _ = client.Close() }()

	originalChangedAt := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	nextChangedAt := time.Date(2026, 5, 16, 10, 30, 0, 0, time.UTC)
	record, err := client.User.Create().
		SetUsername("alice").
		SetDisplay("Alice").
		SetPasswordHash("old-hash").
		SetMustChangePassword(true).
		SetPasswordChangedAt(originalChangedAt).
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	repo := &authRepository{client: client}
	if err := repo.SetPasswordHash(context.Background(), store.SetPasswordHashInput{
		UserID:             toStoreID(record.ID),
		PasswordHash:       "new-hash",
		MustChangePassword: false,
		ChangedAt:          &nextChangedAt,
	}); err != nil {
		t.Fatalf("set password hash: %v", err)
	}

	reloaded, err := client.User.Get(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.PasswordChangedAt == nil || !reloaded.PasswordChangedAt.Equal(nextChangedAt) {
		t.Fatalf("expected password_changed_at to update to %v, got %#v", nextChangedAt, reloaded.PasswordChangedAt)
	}
}

// TestAuthRepositoryChangePasswordAndRevokeOtherRefreshSessionsMapsInvalidIDToNotFound
// 验证原子改密写路径会把无效用户标识映射为稳定领域错误。
func TestAuthRepositoryChangePasswordAndRevokeOtherRefreshSessionsMapsInvalidIDToNotFound(t *testing.T) {
	repo := &authRepository{}

	err := repo.ChangePasswordAndRevokeOtherRefreshSessions(
		context.Background(),
		store.ChangePasswordAndRevokeOtherRefreshSessionsInput{
			UserID:         0,
			PasswordHash:   "hash",
			ChangedAt:      time.Now().UTC(),
			CurrentTokenID: "current-session",
		},
	)
	if !errors.Is(err, store.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

type passwordChangeRollbackFixture struct {
	client         *localent.Client
	repo           *authRepository
	userID         int
	passwordHash   string
	currentTokenID string
	otherSessionID int
}

func newPasswordChangeRollbackFixture(t *testing.T) passwordChangeRollbackFixture {
	t.Helper()

	client := enttest.Open(t, "sqlite3", "file:password-change-rollback?mode=memory&cache=shared&_fk=1")
	passwordHash := "old-hash"
	userRecord, err := client.User.Create().
		SetUsername("alice").
		SetDisplay("Alice").
		SetPasswordHash(passwordHash).
		SetMustChangePassword(true).
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	currentSession, err := client.RefreshSession.Create().
		SetUserID(userRecord.ID).
		SetTokenID("keep-current-session").
		SetExpiresAt(time.Now().UTC().Add(2 * time.Hour)).
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed current session: %v", err)
	}
	otherSession, err := client.RefreshSession.Create().
		SetUserID(userRecord.ID).
		SetTokenID("revoke-me").
		SetExpiresAt(time.Now().UTC().Add(3 * time.Hour)).
		Save(context.Background())
	if err != nil {
		t.Fatalf("seed sibling session: %v", err)
	}

	return passwordChangeRollbackFixture{
		client:         client,
		repo:           &authRepository{client: client},
		userID:         userRecord.ID,
		passwordHash:   passwordHash,
		currentTokenID: currentSession.TokenID,
		otherSessionID: otherSession.ID,
	}
}

func installRefreshSessionFailureHook(client *localent.Client) {
	client.RefreshSession.Use(func(next localent.Mutator) localent.Mutator {
		return hook.RefreshSessionFunc(func(ctx context.Context, mutation *localent.RefreshSessionMutation) (localent.Value, error) {
			if mutation.Op().Is(localent.OpUpdate | localent.OpUpdateOne) {
				return nil, errors.New("forced refresh-session revoke failure")
			}
			return next.Mutate(ctx, mutation)
		})
	})
}

func assertPasswordChangeRollbackState(t *testing.T, fixture passwordChangeRollbackFixture) {
	t.Helper()

	reloadedUser, err := fixture.client.User.Get(context.Background(), fixture.userID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloadedUser.PasswordHash == nil || *reloadedUser.PasswordHash != fixture.passwordHash {
		t.Fatalf("expected password hash rollback to preserve old hash, got %#v", reloadedUser.PasswordHash)
	}
	if !reloadedUser.MustChangePassword {
		t.Fatalf("expected must_change_password rollback to preserve true, got %#v", reloadedUser)
	}
	if reloadedUser.PasswordChangedAt != nil {
		t.Fatalf("expected password_changed_at rollback to stay nil, got %#v", reloadedUser.PasswordChangedAt)
	}

	reloadedCurrent, err := fixture.client.RefreshSession.Query().
		Where(refreshsession.TokenIDEQ(fixture.currentTokenID)).
		Only(context.Background())
	if err != nil {
		t.Fatalf("reload current session: %v", err)
	}
	if reloadedCurrent.RevokedAt != nil {
		t.Fatalf("expected current session to remain active, got %#v", reloadedCurrent)
	}

	reloadedOther, err := fixture.client.RefreshSession.Get(context.Background(), fixture.otherSessionID)
	if err != nil {
		t.Fatalf("reload sibling session: %v", err)
	}
	if reloadedOther.RevokedAt != nil {
		t.Fatalf("expected sibling session revoke to roll back, got %#v", reloadedOther)
	}
}

// TestAuthRepositoryChangePasswordAndRevokeOtherRefreshSessionsRollsBackOnRevokeFailure 验证
// 第二步吊销 sibling sessions 失败时，密码更新也会一起回滚。
func TestAuthRepositoryChangePasswordAndRevokeOtherRefreshSessionsRollsBackOnRevokeFailure(t *testing.T) {
	fixture := newPasswordChangeRollbackFixture(t)
	defer func() { _ = fixture.client.Close() }()
	installRefreshSessionFailureHook(fixture.client)

	changedAt := time.Date(2026, 5, 16, 8, 30, 0, 0, time.UTC)
	err := fixture.repo.ChangePasswordAndRevokeOtherRefreshSessions(context.Background(), store.ChangePasswordAndRevokeOtherRefreshSessionsInput{
		UserID:             toStoreID(fixture.userID),
		PasswordHash:       "new-hash",
		MustChangePassword: false,
		ChangedAt:          changedAt,
		CurrentTokenID:     fixture.currentTokenID,
	})
	if err == nil || !strings.Contains(err.Error(), "revoke other refresh sessions during password change") {
		t.Fatalf("expected revoke failure, got %v", err)
	}

	assertPasswordChangeRollbackState(t, fixture)
}

// TestAuthRepositoryCreateRefreshSessionRejectsInvalidID 验证刷新会话创建保留无效标识错误语义。
func TestAuthRepositoryCreateRefreshSessionRejectsInvalidID(t *testing.T) {
	repo := &authRepository{}

	_, err := repo.CreateRefreshSession(context.Background(), store.CreateRefreshSessionInput{
		UserID:  0,
		TokenID: "token-id",
	})
	if !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

// TestAuthRepositoryRevokeRefreshSessionsByUserIDRejectsInvalidID 验证按用户批量吊销刷新会话时仍保留稳定参数错误语义。
func TestAuthRepositoryRevokeRefreshSessionsByUserIDRejectsInvalidID(t *testing.T) {
	repo := &authRepository{}

	err := repo.RevokeRefreshSessionsByUserID(context.Background(), store.RevokeRefreshSessionsByUserIDInput{
		UserID:    0,
		RevokedAt: time.Now().UTC(),
	})
	if !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

// TestAuthRepositoryRevokeRefreshSessionByUserIDRejectsInvalidID 验证按用户定向吊销单个刷新会话时仍保留稳定参数错误语义。
func TestAuthRepositoryRevokeRefreshSessionByUserIDRejectsInvalidID(t *testing.T) {
	repo := &authRepository{}

	err := repo.RevokeRefreshSessionByUserID(context.Background(), store.RevokeRefreshSessionByUserIDInput{
		UserID:    0,
		TokenID:   "token-id",
		RevokedAt: time.Now().UTC(),
	})
	if !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

// TestAuthRepositoryListActiveRefreshSessionsByUserIDRejectsInvalidID 验证按用户读取当前有效刷新会话列表时仍保留稳定参数错误语义。
func TestAuthRepositoryListActiveRefreshSessionsByUserIDRejectsInvalidID(t *testing.T) {
	repo := &authRepository{}

	_, err := repo.ListActiveRefreshSessionsByUserID(context.Background(), store.ListActiveRefreshSessionsByUserIDInput{
		UserID: 0,
		Now:    time.Now().UTC(),
	})
	if !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
}

// TestRBACRepositoryRejectsInvalidID 验证 RBAC 查询不会再把无效标识静默吞掉。
func TestRBACRepositoryRejectsInvalidID(t *testing.T) {
	repo := &rbacRepository{}

	if _, err := repo.GetRoleByID(context.Background(), 0); !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected GetRoleByID to return ErrInvalidID, got %v", err)
	}
	if _, err := repo.UpdateRole(context.Background(), store.UpdateRoleInput{ID: 0}); !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected UpdateRole to return ErrInvalidID, got %v", err)
	}
	if _, err := repo.ListRolesByUserID(context.Background(), 0); !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected ListRolesByUserID to return ErrInvalidID, got %v", err)
	}
	if _, err := repo.ListRolePermissionBindings(context.Background(), 0); !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected ListRolePermissionBindings to return ErrInvalidID, got %v", err)
	}
	if _, err := repo.ListPermissionsByUserID(context.Background(), 0); !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected ListPermissionsByUserID to return ErrInvalidID, got %v", err)
	}
	if err := repo.ReplacePermissionsForRole(context.Background(), store.ReplacePermissionsForRoleInput{RoleID: 0}); !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected ReplacePermissionsForRole to return ErrInvalidID, got %v", err)
	}
	if err := repo.ReplaceRolesForUser(context.Background(), store.ReplaceRolesForUserInput{UserID: 0}); !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected ReplaceRolesForUser to return ErrInvalidID, got %v", err)
	}
}
