package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"graft/server/internal/moduleapi"
)

var (
	// errMCPCallerUnavailable 表示请求没有通过个人 API Token 建立 MCP 调用者。
	errMCPCallerUnavailable = errors.New("mcp caller is unavailable")
	// errMCPAuthorizationConfiguration 表示 operation 缺少可同时映射 RBAC 和 scope 的授权定义。
	errMCPAuthorizationConfiguration = errors.New("mcp operation authorization configuration is invalid")
)

type callerContextKey struct{}

// caller 保留 MCP transport 需要的已验证调用者事实。
//
// 它不复制 auth 的 Token 解析或用户身份查询；创建只发生在 HTTP adapter 边界，
// 后续 Tool 处理器从 context 读取同一个主体。
type caller struct {
	tokenID   uint64
	user      moduleapi.CurrentUser
	scopes    []string
	expiresAt int64
}

func newCaller(source moduleapi.PersonalAccessTokenCaller) (caller, error) {
	if source.TokenID == 0 || source.User.ID == 0 || source.ExpiresAt.IsZero() {
		return caller{}, errMCPCallerUnavailable
	}
	return caller{
		tokenID:   source.TokenID,
		user:      source.User,
		scopes:    append([]string(nil), source.Scopes...),
		expiresAt: source.ExpiresAt.UTC().Unix(),
	}, nil
}

func withCaller(ctx context.Context, value caller) context.Context {
	return context.WithValue(ctx, callerContextKey{}, value)
}

func callerFromContext(ctx context.Context) (caller, bool) {
	if ctx == nil {
		return caller{}, false
	}
	value, ok := ctx.Value(callerContextKey{}).(caller)
	return value, ok
}

func (c caller) hasScope(scope string) bool {
	for _, granted := range c.scopes {
		if granted == scope {
			return true
		}
	}
	return false
}

// ScopeGate 始终先调用现有 RBAC authorizer，再检查个人 API Token 的精确 scope。
//
// 此顺序确保 Token 只能减少既有权限，不能借由 scope 值独立获得业务授权。
type ScopeGate struct {
	authorizer moduleapi.Authorizer
}

// Authorize 验证当前 MCP 调用者同时具备既有 RBAC permission 与 Token scope。
func (g ScopeGate) Authorize(ctx context.Context, permission string, scope string) error {
	caller, ok := callerFromContext(ctx)
	if !ok {
		return moduleapi.ErrUnauthenticated
	}
	permission = strings.TrimSpace(permission)
	scope = strings.TrimSpace(scope)
	if permission == "" || scope == "" {
		return errMCPAuthorizationConfiguration
	}
	if g.authorizer == nil {
		return errors.New("mcp authorizer is unavailable")
	}

	request := moduleapi.RequestAuthContext{User: &caller.user}
	if err := g.authorizer.Authorize(ctx, request, permission); err != nil {
		return err
	}
	if !caller.hasScope(scope) {
		return fmt.Errorf("%w: personal API token lacks scope %q", moduleapi.ErrPermissionDenied, scope)
	}
	return nil
}
