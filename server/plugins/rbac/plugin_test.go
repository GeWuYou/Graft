package rbac

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
)

type testRBACRepository struct {
	permissions []store.Permission
	err         error
}

func (r testRBACRepository) ListRolesByUserID(ctx context.Context, userID uint64) ([]store.Role, error) {
	return nil, nil
}

func (r testRBACRepository) ListPermissionsByUserID(ctx context.Context, userID uint64) ([]store.Permission, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.permissions, nil
}

// TestAuthorizerRejectsUnauthenticatedRequest 验证缺少主体时会返回稳定未登录错误。
func TestAuthorizerRejectsUnauthenticatedRequest(t *testing.T) {
	service := authorizer{rbac: testRBACRepository{}}

	err := service.Authorize(context.Background(), pluginapi.RequestAuthContext{}, "user.read")
	if !errors.Is(err, pluginapi.ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}
}

// TestAuthorizerAllowsGrantedPermission 验证命中的权限码会被授权通过。
func TestAuthorizerAllowsGrantedPermission(t *testing.T) {
	service := authorizer{
		rbac: testRBACRepository{
			permissions: []store.Permission{{Code: "user.read"}},
		},
	}

	err := service.Authorize(context.Background(), pluginapi.RequestAuthContext{
		User: &pluginapi.CurrentUser{ID: 7},
	}, "user.read")
	if err != nil {
		t.Fatalf("expected authorization success, got %v", err)
	}
}

// TestAuthorizerRejectsMissingPermission 验证未命中权限码时会返回稳定拒绝错误。
func TestAuthorizerRejectsMissingPermission(t *testing.T) {
	service := authorizer{
		rbac: testRBACRepository{
			permissions: []store.Permission{{Code: "dashboard.view"}},
		},
	}

	err := service.Authorize(context.Background(), pluginapi.RequestAuthContext{
		User: &pluginapi.CurrentUser{ID: 7},
	}, "user.read")
	if !errors.Is(err, pluginapi.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}
