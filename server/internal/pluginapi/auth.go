package pluginapi

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrUnauthenticated 表示当前请求未建立有效登录态。
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrInvalidAccessToken 表示访问令牌格式、签名或主体信息无效。
	ErrInvalidAccessToken = errors.New("invalid access token")
	// ErrExpiredAccessToken 表示访问令牌已经超过有效期。
	ErrExpiredAccessToken = errors.New("expired access token")
	// ErrPermissionDenied 表示认证成功但缺少访问所需权限。
	ErrPermissionDenied = errors.New("permission denied")
)

type requestAuthContextKey struct{}

// CurrentUser 描述跨插件可依赖的当前登录主体摘要。
//
// 该 DTO 只承载认证与授权链路需要的稳定身份信息，不暴露任何存储实现或会话细节。
type CurrentUser struct {
	ID          uint64
	Username    string
	DisplayName string
}

// UserAuthCredential 描述认证链路依赖的最小用户口令与受限态摘要。
//
// 该 DTO 只暴露登录、refresh、bootstrap 与受限会话判断真正需要的稳定字段，
// 不泄漏 user 插件内部实体、仓储或 ORM 细节。
type UserAuthCredential struct {
	UserID             uint64
	Username           string
	PasswordHash       *string
	MustChangePassword bool
	PasswordChangedAt  *time.Time
}

// AccessTokenClaims 描述访问令牌中可被其它插件稳定消费的最小声明集。
//
// 这里仅保留身份与时效信息，不把权限列表、刷新令牌细节或额外身份系统塞进跨插件边界。
type AccessTokenClaims struct {
	UserID       uint64
	SessionID    string
	TokenVersion int
	ExpiresAt    time.Time
	IssuedAt     time.Time
}

// RequestAuthContext 描述一次请求在认证链路中的稳定上下文视图。
//
// 该 DTO 只用于跨插件传递认证结果与请求主体摘要，不负责解析、签发或刷新令牌。
type RequestAuthContext struct {
	User   *CurrentUser
	Claims *AccessTokenClaims
}

// AuthSessionSummary 描述认证插件对外暴露的稳定会话摘要。
//
// 这里保留当前会话治理与列表展示所需的最小字段，不暴露 refresh token、
// rotation 历史或底层持久化主键。
type AuthSessionSummary struct {
	SessionID string
	UserID    uint64
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
	Current   bool
}

// AuthSessionRevokeResult 描述一次会话撤销请求的稳定结果。
//
// 当前阶段只暴露“是否命中并撤销成功”的最小语义，避免把底层写路径细节
// 固化进跨插件 capability。
type AuthSessionRevokeResult struct {
	Revoked bool
}

// WithRequestAuthContext 返回带有稳定请求鉴权上下文的派生 context。
//
// 该辅助函数让 core 中间件、认证服务和业务插件可以沿 `context.Context`
// 显式传递一次请求的认证结果，而不必依赖框架私有全局状态。
func WithRequestAuthContext(ctx context.Context, auth RequestAuthContext) context.Context {
	return context.WithValue(ctx, requestAuthContextKey{}, auth)
}

// RequestAuthContextFromContext 读取一次请求当前已解析的鉴权上下文。
//
// 当调用链尚未建立认证结果时，返回值中的 `ok` 为 false，调用方应按未登录
// 路径处理，而不是假设这里一定存在有效主体。
func RequestAuthContextFromContext(ctx context.Context) (auth RequestAuthContext, ok bool) {
	if ctx == nil {
		return RequestAuthContext{}, false
	}

	auth, ok = ctx.Value(requestAuthContextKey{}).(RequestAuthContext)
	return auth, ok
}

// AuthService 暴露认证链路的最小稳定能力接口。
//
// 调用方只能依赖这里声明的签名和错误语义，不应依赖具体 token 生成算法或 cookie 处理实现。
type AuthService interface {
	CurrentUser(ctx context.Context) (*CurrentUser, error)
	ParseAccessToken(ctx context.Context, token string) (*AccessTokenClaims, error)
}

// AuthSessionService 暴露认证插件拥有的稳定会话治理能力。
//
// user 插件若继续保留 `/users/:id/sessions` 管理入口，应只依赖该 capability，
// 而不是直接访问 refresh session store 或 ORM 实现。
type AuthSessionService interface {
	ListSessionsByUserID(ctx context.Context, userID uint64) ([]AuthSessionSummary, error)
	RevokeSessionByUserID(ctx context.Context, userID uint64, sessionID string) (AuthSessionRevokeResult, error)
	RevokeSessionsByUserID(ctx context.Context, userID uint64) (AuthSessionRevokeResult, error)
	RevokeOtherSessionsByUserID(
		ctx context.Context,
		userID uint64,
		currentSessionID string,
	) (AuthSessionRevokeResult, error)
}

// UserAuthIdentityService 暴露 auth 插件可依赖的稳定用户身份能力。
//
// auth 通过它读取登录凭据、当前主体摘要与改密写路径；该接口故意不暴露
// user 插件的仓储实现、Ent client 或管理员资源管理语义。
type UserAuthIdentityService interface {
	GetCredentialByUsername(ctx context.Context, username string) (UserAuthCredential, error)
	GetCurrentUserByID(ctx context.Context, userID uint64) (CurrentUser, error)
	SetPasswordByUserID(
		ctx context.Context,
		userID uint64,
		passwordHash string,
		mustChangePassword bool,
		changedAt *time.Time,
	) error
}

// Authorizer 暴露请求级授权判断能力。
//
// 该接口只定义能力边界，不绑定具体权限引擎实现，便于后续由 rbac 或其它插件提供实现。
type Authorizer interface {
	Authorize(ctx context.Context, request RequestAuthContext, permission string) error
}
