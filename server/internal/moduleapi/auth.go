package moduleapi

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var (
	// ErrUnauthenticated 表示当前请求未建立有效登录态。
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrInvalidAccessToken 表示访问令牌格式、签名或主体信息无效。
	ErrInvalidAccessToken = errors.New("invalid access token")
	// ErrExpiredAccessToken 表示访问令牌已经超过有效期。
	ErrExpiredAccessToken = errors.New("expired access token")
	// ErrInvalidPersonalAccessToken 表示个人 API Token 不存在、已撤销或与已知格式不匹配。
	ErrInvalidPersonalAccessToken = errors.New("invalid personal access token")
	// ErrExpiredPersonalAccessToken 表示个人 API Token 的有效期已经结束。
	ErrExpiredPersonalAccessToken = errors.New("expired personal access token")
	// ErrPermissionDenied 表示认证成功但缺少访问所需权限。
	ErrPermissionDenied = errors.New("permission denied")
	// ErrPasswordPolicyViolation 表示密码不满足认证模块当前的密码策略。
	ErrPasswordPolicyViolation = errors.New("password policy violation")
	// ErrPasswordReuseForbidden 表示认证策略禁止重复使用该密码。
	ErrPasswordReuseForbidden = errors.New("password reuse forbidden")
)

type requestAuthContextKey struct{}

// CurrentUser 描述跨模块可依赖的当前登录主体摘要。
//
// 该 DTO 只承载认证与授权链路需要的稳定身份信息，不暴露任何存储实现或会话细节。
type CurrentUser struct {
	ID          uint64
	Username    string
	DisplayName string
}

// AccessTokenClaims 描述访问令牌中可被其它模块稳定消费的最小声明集。
//
// 这里仅保留身份与时效信息，不把权限列表、刷新令牌细节或额外身份系统塞进跨模块边界。
type AccessTokenClaims struct {
	UserID       uint64
	SessionID    string
	TokenVersion int
	ExpiresAt    time.Time
	IssuedAt     time.Time
}

// RequestAuthContext 描述一次请求在认证链路中的稳定上下文视图。
//
// 该 DTO 只用于跨模块传递认证结果与请求主体摘要，不负责解析、签发或刷新令牌。
type RequestAuthContext struct {
	User   *CurrentUser
	Claims *AccessTokenClaims
}

type personalAccessTokenCallerContextKey struct{}

// PersonalAccessTokenCaller 描述由 auth 模块验证后的个人 API Token 主体。
//
// Scopes 仅是授权收窄条件；调用方仍必须先通过当前用户的 RBAC 授权，不能把
// 这里的 scope 当作独立授权来源。
type PersonalAccessTokenCaller struct {
	TokenID   uint64
	User      CurrentUser
	Scopes    []string
	ExpiresAt time.Time
}

// PersonalAccessTokenCreateInput 描述当前用户创建个人 API Token 时可声明的最小策略。
//
// ExpiresAt 必须由调用方显式指定且晚于当前时间，避免产生无期限的机器凭据。
type PersonalAccessTokenCreateInput struct {
	Name      string
	Scopes    []string
	ExpiresAt time.Time
}

// PersonalAccessTokenSummary 描述可以安全返回给 Token 所有者的生命周期快照。
//
// Secret 永不出现在该结构中；只有创建成功的 PersonalAccessTokenIssued 会携带一次性明文值。
type PersonalAccessTokenSummary struct {
	ID          uint64
	Name        string
	TokenPrefix string
	Scopes      []string
	ExpiresAt   time.Time
	RevokedAt   *time.Time
	LastUsedAt  *time.Time
	CreatedAt   time.Time
}

// PersonalAccessTokenIssued 描述新建 Token 的一次性返回值。
//
// Token 只在创建响应中出现，调用方必须自行保存；auth 持久层仅保存其哈希。
type PersonalAccessTokenIssued struct {
	Summary PersonalAccessTokenSummary
	Token   string
}

// AuthSessionSummary 描述认证模块对外暴露的稳定会话摘要。
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
// 固化进跨模块 capability。
type AuthSessionRevokeResult struct {
	Revoked bool
}

// AuthRefreshResult 描述 auth 路由返回的稳定登录/刷新结果。
type AuthRefreshResult struct {
	AccessToken        string
	AccessExpiry       time.Time
	RefreshToken       string
	RefreshExpiry      time.Time
	MustChangePassword bool
	User               CurrentUser
}

// AuthBootstrapMenuItem 描述 bootstrap 响应中的单个菜单快照。
type AuthBootstrapMenuItem struct {
	Code            string
	ParentCode      string
	Kind            string
	Title           string
	TitleKey        string
	SectionKey      string
	SectionTitleKey string
	Path            string
	Icon            string
	Order           int
	Permission      string
}

// AuthBootstrapLocaleSnapshot 描述 bootstrap 响应中的 locale 快照。
type AuthBootstrapLocaleSnapshot struct {
	CurrentLocale    string
	DefaultLocale    string
	FallbackLocale   string
	SupportedLocales []string
}

// AuthBootstrapPayload 描述 `/auth/bootstrap` 返回的稳定载荷。
type AuthBootstrapPayload struct {
	User               CurrentUser
	MustChangePassword bool
	Roles              []string
	Permissions        []string
	Menus              []AuthBootstrapMenuItem
	Locale             AuthBootstrapLocaleSnapshot
}

// AuthRouteError 描述 auth 路由需要返回的稳定错误契约。
type AuthRouteError struct {
	Status     int
	MessageKey string
	Data       map[string]any
}

// WithRequestAuthContext 返回带有稳定请求鉴权上下文的派生 context。
//
// 该辅助函数让 core 中间件、认证服务和业务模块可以沿 `context.Context`
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

// WithPersonalAccessTokenCaller 返回带有已验证个人 API Token 主体的派生 context。
//
// core HTTP 中间件负责写入该值，MCP adapter 只消费它并转换为 transport-local caller context，
// 这样 MCP 不需要重新解析或旁路 auth 模块的 Token 生命周期规则。
func WithPersonalAccessTokenCaller(ctx context.Context, caller PersonalAccessTokenCaller) context.Context {
	return context.WithValue(ctx, personalAccessTokenCallerContextKey{}, caller)
}

// PersonalAccessTokenCallerFromContext 返回当前请求由个人 API Token 建立的调用者。
func PersonalAccessTokenCallerFromContext(ctx context.Context) (caller PersonalAccessTokenCaller, ok bool) {
	if ctx == nil {
		return PersonalAccessTokenCaller{}, false
	}

	caller, ok = ctx.Value(personalAccessTokenCallerContextKey{}).(PersonalAccessTokenCaller)
	return caller, ok
}

// AuthService 暴露认证链路的最小稳定能力接口。
//
// 调用方只能依赖这里声明的签名和错误语义，不应依赖具体 token 生成算法或 cookie 处理实现。
type AuthService interface {
	CurrentUser(ctx context.Context) (*CurrentUser, error)
	ParseAccessToken(ctx context.Context, token string) (*AccessTokenClaims, error)
}

// PersonalAccessTokenService 暴露 auth 模块拥有的个人 API Token 生命周期与验证能力。
//
// 管理操作绑定当前请求用户；AuthenticatePersonalAccessToken 只建立身份和 scope 收窄事实，
// 不替代 RBAC 对具体 permission 的判断。
type PersonalAccessTokenService interface {
	CreateCurrentUserPersonalAccessToken(ctx context.Context, input PersonalAccessTokenCreateInput) (PersonalAccessTokenIssued, error)
	ListCurrentUserPersonalAccessTokens(ctx context.Context, limit int) ([]PersonalAccessTokenSummary, error)
	RevokeCurrentUserPersonalAccessToken(ctx context.Context, tokenID uint64) error
	AuthenticatePersonalAccessToken(ctx context.Context, token string) (PersonalAccessTokenCaller, error)
}

// AuthSessionService 暴露认证模块拥有的稳定会话治理能力。
//
// user 模块若继续保留 `/users/:id/sessions` 管理入口，应只依赖该 capability，
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

// AuthCredentialManagementService 向用户资料管理暴露凭据生命周期操作，但不泄漏认证持久化细节。
// 违反密码策略时返回 ErrPasswordPolicyViolation；违反认证模块复用规则时返回 ErrPasswordReuseForbidden。
type AuthCredentialManagementService interface {
	ProvisionPasswordCredential(ctx context.Context, userID uint64, password string, mustChangePassword bool) error
	ResetPassword(ctx context.Context, userID uint64, password string) error
	RevokeSessions(ctx context.Context, userID uint64) error
}

// AuthFlowService 暴露 `/auth/*` 路由需要的稳定认证闭环能力。
type AuthFlowService interface {
	StartLogin(ctx context.Context, username string, password string) (AuthRefreshResult, error)
	RefreshSession(ctx context.Context, refreshToken string) (AuthRefreshResult, error)
	LogoutCurrentSession(ctx context.Context, refreshToken string) error
	RevokeAllCurrentUserSessions(ctx context.Context) error
	RevokeOtherCurrentUserSessions(ctx context.Context) error
	ListCurrentUserSessions(ctx context.Context, limit int) ([]AuthSessionSummary, error)
	RevokeCurrentUserSession(ctx context.Context, sessionID string) error
	ReadBootstrapPayload(ctx context.Context, request *http.Request) (AuthBootstrapPayload, error)
	ChangeCurrentUserPassword(ctx context.Context, currentPassword string, newPassword string) error
	CompleteRequiredPasswordChange(ctx context.Context, newPassword string) error
	IsRestrictedPasswordChangeSession(ctx context.Context) (bool, error)
	RouteError(err error) AuthRouteError
}

// UserIdentityProvider 仅暴露认证模块需要的用户资料身份事实。
// 凭据、密码状态和会话信息由认证模块拥有，不在此边界中暴露。
type UserIdentityProvider interface {
	LookupUserByUsername(ctx context.Context, username string) (CurrentUser, error)
	GetCurrentUserByID(ctx context.Context, userID uint64) (CurrentUser, error)
	EnsureDefaultAdminProfile(ctx context.Context) (CurrentUser, error)
}

// UserBootstrapProvider 暴露认证 bootstrap 路由需要的用户资料、RBAC、菜单和 locale 快照。
// 凭据状态仍由认证模块拥有。
type UserBootstrapProvider interface {
	ReadBootstrap(ctx context.Context, request *http.Request) (AuthBootstrapPayload, error)
}

// Authorizer 暴露请求级授权判断能力。
//
// 该接口只定义能力边界，不绑定具体权限引擎实现，便于后续由 rbac 或其它模块提供实现。
type Authorizer interface {
	Authorize(ctx context.Context, request RequestAuthContext, permission string) error
}
