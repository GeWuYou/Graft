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

func (r testRBACRepository) EnsureRole(_ context.Context, _ store.EnsureRoleInput) (store.Role, error) {
	return store.Role{}, nil
}

func (r testRBACRepository) EnsurePermission(_ context.Context, _ store.EnsurePermissionInput) (store.Permission, error) {
	return store.Permission{}, nil
}

func (r testRBACRepository) AssignPermissionsToRole(_ context.Context, _ store.AssignPermissionsToRoleInput) error {
	return nil
}

func (r testRBACRepository) AssignRoleToUser(_ context.Context, _ store.AssignRoleToUserInput) error {
	return nil
}

func (r testRBACRepository) ListRolesByUserID(_ context.Context, _ uint64) ([]store.Role, error) {
	return nil, nil
}

func (r testRBACRepository) ListPermissionsByUserID(_ context.Context, _ uint64) ([]store.Permission, error) {
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

// TestAuthorizerPropagatesRepositoryFailure 验证权限仓储失败会直接向调用方传播。
func TestAuthorizerPropagatesRepositoryFailure(t *testing.T) {
	repositoryErr := errors.New("repository failed")
	service := authorizer{
		rbac: testRBACRepository{
			err: repositoryErr,
		},
	}

	err := service.Authorize(context.Background(), pluginapi.RequestAuthContext{
		User: &pluginapi.CurrentUser{ID: 7},
	}, "user.read")
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}
