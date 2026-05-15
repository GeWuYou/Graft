package entstore

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

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

	if _, err := repo.ListRolesByUserID(context.Background(), 0); !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected ListRolesByUserID to return ErrInvalidID, got %v", err)
	}
	if _, err := repo.ListPermissionsByUserID(context.Background(), 0); !errors.Is(err, store.ErrInvalidID) {
		t.Fatalf("expected ListPermissionsByUserID to return ErrInvalidID, got %v", err)
	}
}
