package rbac

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"graft/server/internal/moduleapi"
	rbacstore "graft/server/modules/rbac/store"
)

type bootstrapService struct {
	rbac rbacstore.Repository
}

// NewBootstrapService 基于模块自有仓储构建稳定的 RBAC bootstrap 能力；调用方只能读取授权快照，不能绕过仓储修改绑定。
func NewBootstrapService(rbac rbacstore.Repository) moduleapi.RBACBootstrapService {
	if rbac == nil {
		return nil
	}

	return bootstrapService{rbac: rbac}
}

func (s bootstrapService) EnsureDefaultAdminAccess(
	ctx context.Context,
	userID uint64,
	permissions []moduleapi.PermissionSeed,
) error {
	if s.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}

	role, err := s.rbac.EnsureRole(ctx, rbacstore.EnsureRoleInput{
		Name:       builtinAdminRoleName,
		Display:    "管理员",
		DisplayKey: stringPtrOrNil("rbac.roles.admin.display"),
		Builtin:    true,
	})
	if err != nil {
		return fmt.Errorf("ensure default admin role: %w", err)
	}

	// 系统角色策略由版本化迁移同步。启动阶段只保证首位平台管理员拥有 Admin 角色，
	// 绝不能因为新注册权限而把它隐式授予历史角色。
	_ = permissions
	if err := s.rbac.AssignRoleToUser(ctx, rbacstore.AssignRoleToUserInput{
		UserID: userID,
		RoleID: role.ID,
	}); err != nil {
		return fmt.Errorf("assign default admin role to user: %w", err)
	}

	return nil
}

func stringPtrOrNil(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	result := value
	return &result
}

var _ moduleapi.RBACBootstrapService = bootstrapService{}
