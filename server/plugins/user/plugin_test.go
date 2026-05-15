package user

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
	"graft/server/plugins/rbac"
)

// pluginTestStoreFactory 为插件路由测试提供最小仓储装配。
type pluginTestStoreFactory struct {
	auth        store.AuthRepository
	users       store.UserRepository
	permissions map[uint64][]store.Permission
}

func (f pluginTestStoreFactory) Auth() store.AuthRepository {
	return f.auth
}

func (f pluginTestStoreFactory) Users() store.UserRepository {
	return f.users
}

func (f pluginTestStoreFactory) RBAC() store.RBACRepository {
	return pluginTestRBACRepository{permissions: f.permissions}
}

// pluginTestAuthRepository 以内存状态模拟认证仓储的最小行为。
type pluginTestAuthRepository struct {
	getUserCredentialByUsername func(ctx context.Context, username string) (store.UserCredential, error)
	mu                          sync.Mutex
	refreshSessions             map[string]store.RefreshSession
}

func (r pluginTestAuthRepository) GetUserCredentialByUsername(ctx context.Context, username string) (store.UserCredential, error) {
	if r.getUserCredentialByUsername == nil {
		return store.UserCredential{}, store.ErrUserNotFound
	}

	return r.getUserCredentialByUsername(ctx, username)
}

func (pluginTestAuthRepository) SetPasswordHash(context.Context, store.SetPasswordHashInput) error {
	return nil
}

func (r *pluginTestAuthRepository) CreateRefreshSession(_ context.Context, input store.CreateRefreshSessionInput) (store.RefreshSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.refreshSessions == nil {
		r.refreshSessions = make(map[string]store.RefreshSession)
	}

	session := store.RefreshSession{
		ID:        uint64(len(r.refreshSessions) + 1),
		UserID:    input.UserID,
		TokenID:   input.TokenID,
		ExpiresAt: input.ExpiresAt,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	r.refreshSessions[input.TokenID] = session
	return session, nil
}

func (r *pluginTestAuthRepository) GetRefreshSessionByTokenID(_ context.Context, tokenID string) (store.RefreshSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.refreshSessions[tokenID]
	if !ok {
		return store.RefreshSession{}, store.ErrRefreshSessionNotFound
	}
	return session, nil
}

func (r *pluginTestAuthRepository) RevokeRefreshSession(_ context.Context, input store.RevokeRefreshSessionInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.refreshSessions[input.TokenID]
	if !ok {
		return store.ErrRefreshSessionNotFound
	}
	session.RevokedAt = &input.RevokedAt
	session.ReplacedByTokenID = input.ReplacedByTokenID
	session.UpdatedAt = input.RevokedAt
	r.refreshSessions[input.TokenID] = session
	return nil
}

func (r *pluginTestAuthRepository) RevokeRefreshSessionsByUserID(_ context.Context, input store.RevokeRefreshSessionsByUserIDInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for tokenID, session := range r.refreshSessions {
		if session.UserID != input.UserID || session.RevokedAt != nil {
			continue
		}

		session.RevokedAt = &input.RevokedAt
		session.UpdatedAt = input.RevokedAt
		r.refreshSessions[tokenID] = session
	}

	return nil
}

func (r *pluginTestAuthRepository) RevokeRefreshSessionByUserID(_ context.Context, input store.RevokeRefreshSessionByUserIDInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if input.UserID == 0 {
		return store.ErrInvalidID
	}

	session, ok := r.refreshSessions[input.TokenID]
	if !ok || session.UserID != input.UserID || session.RevokedAt != nil || !session.ExpiresAt.After(input.RevokedAt) {
		return store.ErrRefreshSessionNotFound
	}

	session.RevokedAt = &input.RevokedAt
	session.UpdatedAt = input.RevokedAt
	r.refreshSessions[input.TokenID] = session
	return nil
}

func (r *pluginTestAuthRepository) ListActiveRefreshSessionsByUserID(_ context.Context, input store.ListActiveRefreshSessionsByUserIDInput) ([]store.RefreshSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if input.UserID == 0 {
		return nil, store.ErrInvalidID
	}

	sessions := make([]store.RefreshSession, 0, len(r.refreshSessions))
	for _, session := range r.refreshSessions {
		if session.UserID != input.UserID || session.RevokedAt != nil || !session.ExpiresAt.After(input.Now) {
			continue
		}

		sessions = append(sessions, session)
	}

	slices.SortFunc(sessions, func(left store.RefreshSession, right store.RefreshSession) int {
		if compare := right.CreatedAt.Compare(left.CreatedAt); compare != 0 {
			return compare
		}
		return cmp.Compare(right.TokenID, left.TokenID)
	})

	return sessions, nil
}

func (r *pluginTestAuthRepository) RotateRefreshSession(_ context.Context, input store.RotateRefreshSessionInput) (store.RefreshSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.refreshSessions[input.CurrentTokenID]
	if !ok {
		return store.RefreshSession{}, store.ErrRefreshSessionNotFound
	}
	if current.RevokedAt != nil || !current.ExpiresAt.After(input.Now) {
		return store.RefreshSession{}, store.ErrRefreshSessionNotFound
	}

	current.RevokedAt = &input.RevokedAt
	current.ReplacedByTokenID = &input.NewTokenID
	current.UpdatedAt = input.RevokedAt
	r.refreshSessions[input.CurrentTokenID] = current

	next := store.RefreshSession{
		ID:        uint64(len(r.refreshSessions) + 1),
		UserID:    current.UserID,
		TokenID:   input.NewTokenID,
		ExpiresAt: input.NewExpiresAt,
		CreatedAt: input.RevokedAt,
		UpdatedAt: input.RevokedAt,
	}
	r.refreshSessions[input.NewTokenID] = next
	return next, nil
}

// pluginTestUserRepository 为插件路由测试收敛最小用户读取能力。
type pluginTestUserRepository struct {
	getByID func(ctx context.Context, id uint64) (store.User, error)
}

func (r pluginTestUserRepository) GetByID(ctx context.Context, id uint64) (store.User, error) {
	if r.getByID == nil {
		return store.User{}, store.ErrUserNotFound
	}

	return r.getByID(ctx, id)
}

// pluginTestRBACRepository 为插件路由测试模拟最小权限读取结果。
type pluginTestRBACRepository struct {
	permissions map[uint64][]store.Permission
}

func (r pluginTestRBACRepository) ListRolesByUserID(ctx context.Context, userID uint64) ([]store.Role, error) {
	return nil, nil
}

func (r pluginTestRBACRepository) ListPermissionsByUserID(ctx context.Context, userID uint64) ([]store.Permission, error) {
	if r.permissions == nil {
		return []store.Permission{}, nil
	}

	return r.permissions[userID], nil
}

func newPluginTestContext(t *testing.T, userRepo store.UserRepository, authRepo store.AuthRepository) (*plugin.Context, *gin.Engine) {
	return newPluginTestContextWithPermissions(t, userRepo, authRepo, map[uint64][]store.Permission{
		7: {{Code: "user.read"}},
	})
}

func newPluginTestContextWithPermissions(t *testing.T, userRepo store.UserRepository, authRepo store.AuthRepository, permissions map[uint64][]store.Permission) (*plugin.Context, *gin.Engine) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctx := &plugin.Context{
		Logger: zap.NewNop(),
		Config: &config.Config{Auth: config.AuthConfig{
			AccessTokenTTL:        15 * time.Minute,
			RefreshTokenTTL:       24 * time.Hour,
			SigningKey:            "test-signing-key",
			RefreshCookieName:     "graft_refresh_token",
			RefreshCookieSameSite: "lax",
			RefreshCookiePath:     "/",
		}},
		I18n:     i18n.New(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}),
		Router:   engine.Group("/api"),
		Services: container.New(),
		Stores: pluginTestStoreFactory{
			auth:        authRepo,
			users:       userRepo,
			permissions: permissions,
		},
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
	}

	if err := NewPlugin().Register(ctx); err != nil {
		t.Fatalf("register plugin: %v", err)
	}
	if err := rbac.NewPlugin().Register(ctx); err != nil {
		t.Fatalf("register rbac plugin: %v", err)
	}

	return ctx, engine
}

func newAuthorizedRequestForUser(t *testing.T, path string, authRepo store.AuthRepository, userID uint64) *http.Request {
	t.Helper()

	sessionID := seedRefreshSession(t, authRepo, userID, time.Now().UTC().Add(time.Hour))
	return newAuthorizedRequestForSession(t, path, userID, sessionID)
}

func newAuthorizedRequestForSession(t *testing.T, path string, userID uint64, sessionID string) *http.Request {
	return newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, path, userID, sessionID)
}

func newAuthorizedRequestForSessionWithMethod(t *testing.T, method string, path string, userID uint64, sessionID string) *http.Request {
	t.Helper()

	manager, err := newAccessTokenManager(config.AuthConfig{
		AccessTokenTTL: 15 * time.Minute,
		SigningKey:     "test-signing-key",
	})
	if err != nil {
		t.Fatalf("new access token manager: %v", err)
	}
	token, _, err := manager.Issue(accessTokenSubject{
		UserID:       userID,
		SessionID:    sessionID,
		TokenVersion: 1,
	})
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}

	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	return request
}

func seedRefreshSession(t *testing.T, authRepo store.AuthRepository, userID uint64, expiresAt time.Time) string {
	t.Helper()

	if authRepo == nil {
		t.Fatal("auth repository is required to seed refresh session")
	}

	tokenID := fmt.Sprintf("session-%d", time.Now().UTC().UnixNano())
	if _, err := authRepo.CreateRefreshSession(context.Background(), store.CreateRefreshSessionInput{
		UserID:    userID,
		TokenID:   tokenID,
		ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatalf("seed refresh session: %v", err)
	}

	return tokenID
}

// TestRegisterPublishesContracts 验证用户插件注册时会暴露权限、菜单与稳定
// 的跨插件用户服务。
func TestRegisterPublishesContracts(t *testing.T) {
	ctx, _ := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(context.Context, uint64) (store.User, error) {
			return store.User{
				ID:        7,
				Username:  "alice",
				Display:   "Alice",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}, nil)

	if items := ctx.PermissionRegistry.Items(); len(items) != 3 {
		t.Fatalf("expected three user permissions, got %#v", items)
	}
	if items := ctx.PermissionRegistry.Items(); items[0].Code != "user.read" || items[1].Code != "user.session.revoke" || items[2].Code != "user.session.read" {
		t.Fatalf("expected user.read, user.session.revoke and user.session.read permissions, got %#v", items)
	}
	if items := ctx.MenuRegistry.Items(); len(items) != 1 || items[0].Path != "/users" {
		t.Fatalf("expected one /users menu item, got %#v", items)
	}

	svcAny, err := ctx.Services.Resolve((*pluginapi.UserService)(nil))
	if err != nil {
		t.Fatalf("resolve user service: %v", err)
	}

	summary, err := svcAny.(pluginapi.UserService).GetUserByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("get user by id: %v", err)
	}
	if summary.ID != 7 || summary.Username != "alice" || summary.Display != "Alice" {
		t.Fatalf("expected stable user summary, got %#v", summary)
	}
}

// TestUserRouteRejectsInvalidID 验证用户查询路由会把非法 ID 收敛为 400
// JSON 响应，而不是继续访问仓储。
func TestUserRouteRejectsInvalidID(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id == 7 {
				return store.User{
					ID:        7,
					Username:  "alice",
					Display:   "Alice",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			}
			t.Fatalf("user repository should not be called for invalid route id, got %d", id)
			return store.User{}, nil
		},
	}, authRepo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedRequestForUser(t, "/api/users/not-a-number", authRepo, 7))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "common.invalid_argument" || payload.Locale != "zh-CN" {
		t.Fatalf("expected localized invalid argument contract, got %#v", payload)
	}
	if payload.Message != "请求参数不合法" || payload.Error != payload.Message {
		t.Fatalf("expected parse error payload, got %#v", payload)
	}
	if payload.Details["field"] != "id" {
		t.Fatalf("expected field detail to be id, got %#v", payload)
	}
}

// TestUserRouteReturnsNotFoundContract 验证仓储未命中时，路由会返回 404
// 与稳定错误消息，便于前端后续接入统一空态分支。
func TestUserRouteReturnsNotFoundContract(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id == 7 {
				return store.User{
					ID:        7,
					Username:  "alice",
					Display:   "Alice",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			}
			return store.User{}, store.ErrUserNotFound
		},
	}, authRepo)

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForUser(t, "/api/users/8", authRepo, 7)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "user.not_found" || payload.Locale != "en-US" {
		t.Fatalf("expected localized not found contract, got %#v", payload)
	}
	if payload.Message != "User not found" || payload.Error != payload.Message {
		t.Fatalf("expected not found payload, got %#v", payload)
	}
}

// TestUserRouteReturnsSummary 验证用户查询成功时会返回跨插件稳定 DTO，而不
// 直接泄漏仓储层内部结构。
func TestUserRouteReturnsSummary(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(context.Context, uint64) (store.User, error) {
			return store.User{
				ID:        7,
				Username:  "alice",
				Display:   "Alice",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}, authRepo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedRequestForUser(t, "/api/users/7", authRepo, 7))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload pluginapi.UserSummary
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ID != 7 || payload.Username != "alice" || payload.Display != "Alice" {
		t.Fatalf("expected stable user summary payload, got %#v", payload)
	}
}

// TestUserRouteRequiresPermissionMiddleware 验证插件路由仍复用统一的后端
// 权限守卫契约，而不是在插件内部发散独立鉴权格式。
func TestUserRouteRequiresPermissionMiddleware(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/users/7", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.missing_actor" {
		t.Fatalf("expected permission middleware payload, got %#v", payload)
	}
}

// TestAuthServiceCurrentUserRequiresClaims 验证当前主体解析要求调用链先建立稳定 claims。
func TestAuthServiceCurrentUserRequiresClaims(t *testing.T) {
	service := authService{
		users: pluginTestUserRepository{
			getByID: func(context.Context, uint64) (store.User, error) {
				t.Fatal("user repository should not be called when claims are missing")
				return store.User{}, nil
			},
		},
	}

	_, err := service.CurrentUser(context.Background())
	if !errors.Is(err, pluginapi.ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}
}

// TestAuthServiceParseAccessTokenRequiresActiveSession 验证 access token 除了 JWT
// 自身合法，还要求对应 session 仍存在、未吊销且未过期。
func TestAuthServiceParseAccessTokenRequiresActiveSession(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(t *testing.T, repo *pluginTestAuthRepository) string
	}{
		{
			name: "missing session",
			arrange: func(t *testing.T, repo *pluginTestAuthRepository) string {
				t.Helper()
				return "missing-session"
			},
		},
		{
			name: "revoked session",
			arrange: func(t *testing.T, repo *pluginTestAuthRepository) string {
				t.Helper()

				sessionID := seedRefreshSession(t, repo, 7, time.Now().UTC().Add(time.Hour))
				revokedAt := time.Now().UTC()
				if err := repo.RevokeRefreshSession(context.Background(), store.RevokeRefreshSessionInput{
					TokenID:   sessionID,
					RevokedAt: revokedAt,
				}); err != nil {
					t.Fatalf("revoke refresh session: %v", err)
				}
				return sessionID
			},
		},
		{
			name: "expired session",
			arrange: func(t *testing.T, repo *pluginTestAuthRepository) string {
				t.Helper()
				return seedRefreshSession(t, repo, 7, time.Now().UTC().Add(-time.Minute))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			authRepo := &pluginTestAuthRepository{}
			ctx, _ := newPluginTestContext(t, pluginTestUserRepository{
				getByID: func(_ context.Context, id uint64) (store.User, error) {
					if id != 7 {
						return store.User{}, store.ErrUserNotFound
					}

					return store.User{
						ID:        7,
						Username:  "alice",
						Display:   "Alice",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil
				},
			}, authRepo)

			sessionID := tc.arrange(t, authRepo)
			request := newAuthorizedRequestForSession(t, "/api/users/7", 7, sessionID)

			authAny, err := ctx.Services.Resolve((*pluginapi.AuthService)(nil))
			if err != nil {
				t.Fatalf("resolve auth service: %v", err)
			}

			token := strings.TrimSpace(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
			_, err = authAny.(pluginapi.AuthService).ParseAccessToken(context.Background(), token)
			if !errors.Is(err, pluginapi.ErrInvalidAccessToken) {
				t.Fatalf("expected ErrInvalidAccessToken, got %v", err)
			}
		})
	}
}

// TestUserRouteRejectsInactiveSession 验证受保护请求会在 JWT 之外继续校验
// access token 对应 session 的服务端状态。
func TestUserRouteRejectsInactiveSession(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(t *testing.T, repo *pluginTestAuthRepository) *http.Request
	}{
		{
			name: "missing session",
			arrange: func(t *testing.T, repo *pluginTestAuthRepository) *http.Request {
				t.Helper()
				return newAuthorizedRequestForSession(t, "/api/users/7", 7, "missing-session")
			},
		},
		{
			name: "revoked session",
			arrange: func(t *testing.T, repo *pluginTestAuthRepository) *http.Request {
				t.Helper()

				sessionID := seedRefreshSession(t, repo, 7, time.Now().UTC().Add(time.Hour))
				if err := repo.RevokeRefreshSession(context.Background(), store.RevokeRefreshSessionInput{
					TokenID:   sessionID,
					RevokedAt: time.Now().UTC(),
				}); err != nil {
					t.Fatalf("revoke refresh session: %v", err)
				}

				return newAuthorizedRequestForSession(t, "/api/users/7", 7, sessionID)
			},
		},
		{
			name: "expired session",
			arrange: func(t *testing.T, repo *pluginTestAuthRepository) *http.Request {
				t.Helper()
				sessionID := seedRefreshSession(t, repo, 7, time.Now().UTC().Add(-time.Minute))
				return newAuthorizedRequestForSession(t, "/api/users/7", 7, sessionID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			authRepo := &pluginTestAuthRepository{}
			_, engine := newPluginTestContext(t, pluginTestUserRepository{
				getByID: func(context.Context, uint64) (store.User, error) {
					return store.User{
						ID:        7,
						Username:  "alice",
						Display:   "Alice",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil
				},
			}, authRepo)

			recorder := httptest.NewRecorder()
			request := tc.arrange(t, authRepo)
			request.Header.Set(i18n.LocaleHeader, "en-US")
			engine.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
			}

			var payload httpx.ErrorResponse
			if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload.MessageKey != "auth.missing_actor" || payload.Locale != "en-US" {
				t.Fatalf("expected missing actor payload, got %#v", payload)
			}
		})
	}
}

// TestLoginRouteReturnsTokenAndCurrentUserSummary 验证登录接口会校验口令并返回
// access token 与当前用户摘要，而不是泄漏仓储实现细节。
func TestLoginRouteReturnsTokenAndCurrentUserSummary(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username != "alice" {
				return store.UserCredential{}, store.ErrUserNotFound
			}

			return store.UserCredential{
				UserID:       7,
				Username:     "alice",
				PasswordHash: &passwordHash,
			}, nil
		},
	}
	ctx, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}

			return store.User{
				ID:        7,
				Username:  "alice",
				Display:   "Alice",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}, authRepo)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload loginResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.AccessToken == "" {
		t.Fatal("expected access token in login response")
	}
	if payload.User.ID != 7 || payload.User.Username != "alice" || payload.User.DisplayName != "Alice" {
		t.Fatalf("expected current user summary, got %#v", payload.User)
	}
	if payload.ExpiresAt.IsZero() {
		t.Fatal("expected access token expiry in response")
	}
	if payload.ExpiresAt.Before(time.Now().UTC()) {
		t.Fatalf("expected future expiry, got %v", payload.ExpiresAt)
	}
	if len(recorder.Result().Cookies()) == 0 {
		t.Fatal("expected refresh cookie to be written on login")
	}
	refreshCookie := recorder.Result().Cookies()[0]
	if refreshCookie.Name != ctx.Config.Auth.RefreshCookieName || refreshCookie.Value == "" {
		t.Fatalf("expected refresh cookie %q, got %#v", ctx.Config.Auth.RefreshCookieName, refreshCookie)
	}

	authAny, err := ctx.Services.Resolve((*pluginapi.AuthService)(nil))
	if err != nil {
		t.Fatalf("resolve auth service: %v", err)
	}
	claims, err := authAny.(pluginapi.AuthService).ParseAccessToken(context.Background(), payload.AccessToken)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if claims.UserID != 7 || claims.SessionID == "" {
		t.Fatalf("expected stable token claims, got %#v", claims)
	}
	if _, err := authRepo.GetRefreshSessionByTokenID(context.Background(), refreshCookie.Value); err == nil {
		t.Fatal("expected raw cookie token not to equal stored token id")
	}
}

// TestRefreshRouteRotatesSessionAndReturnsNewAccessToken 验证 refresh 路由会从
// cookie 读取 refresh token，校验会话后完成一次轮换并返回新的 access token。
func TestRefreshRouteRotatesSessionAndReturnsNewAccessToken(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username != "alice" {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:       7,
				Username:     "alice",
				PasswordHash: &passwordHash,
			}, nil
		},
	}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{
				ID:        7,
				Username:  "alice",
				Display:   "Alice",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}, authRepo)

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, loginRecorder.Code)
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected refresh cookie from login")
	}

	refreshRecorder := httptest.NewRecorder()
	refreshRequest := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshRequest.AddCookie(cookies[0])
	refreshRequest.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(refreshRecorder, refreshRequest)

	if refreshRecorder.Code != http.StatusOK {
		t.Fatalf("expected refresh status %d, got %d", http.StatusOK, refreshRecorder.Code)
	}

	var payload loginResponse
	if err := json.NewDecoder(refreshRecorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if payload.AccessToken == "" || payload.ExpiresAt.IsZero() {
		t.Fatalf("expected rotated access token payload, got %#v", payload)
	}
	newCookies := refreshRecorder.Result().Cookies()
	if len(newCookies) == 0 || newCookies[0].Value == cookies[0].Value {
		t.Fatalf("expected rotated refresh cookie, got old=%#v new=%#v", cookies, newCookies)
	}
}

// TestRefreshRouteRejectsReusedRefreshCookie 验证 refresh 成功轮换后，旧 cookie
// 不能再次消费原 refresh session。
func TestRefreshRouteRejectsReusedRefreshCookie(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username != "alice" {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:       7,
				Username:     "alice",
				PasswordHash: &passwordHash,
			}, nil
		},
	}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{
				ID:        7,
				Username:  "alice",
				Display:   "Alice",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}, authRepo)

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, loginRecorder.Code)
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected refresh cookie from login")
	}

	firstRefreshRecorder := httptest.NewRecorder()
	firstRefreshRequest := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	firstRefreshRequest.AddCookie(cookies[0])
	firstRefreshRequest.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(firstRefreshRecorder, firstRefreshRequest)
	if firstRefreshRecorder.Code != http.StatusOK {
		t.Fatalf("expected first refresh status %d, got %d", http.StatusOK, firstRefreshRecorder.Code)
	}

	reuseRecorder := httptest.NewRecorder()
	reuseRequest := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	reuseRequest.AddCookie(cookies[0])
	reuseRequest.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(reuseRecorder, reuseRequest)
	if reuseRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected reused refresh status %d, got %d", http.StatusUnauthorized, reuseRecorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(reuseRecorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode reused refresh response: %v", err)
	}
	if payload.MessageKey != "auth.invalid_refresh_session" || payload.Locale != "en-US" {
		t.Fatalf("expected invalid refresh payload for reused cookie, got %#v", payload)
	}
}

// TestRefreshRouteRejectsMissingCookie 验证缺少 refresh cookie 时仍返回统一的
// 本地化认证失败契约。
func TestRefreshRouteRejectsMissingCookie(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, &pluginTestAuthRepository{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.invalid_refresh_session" || payload.Locale != "en-US" {
		t.Fatalf("expected invalid refresh payload, got %#v", payload)
	}
}

// TestLoginDoesNotIssueOrphanedAccessToken 验证基础 Login 流程只做认证，不再
// 提前签发未绑定 refresh session 的 access token。
func TestLoginDoesNotIssueOrphanedAccessToken(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authSvc, err := newAuthService(config.AuthConfig{
		AccessTokenTTL:        time.Hour,
		SigningKey:            "secret-key",
		RefreshTokenTTL:       24 * time.Hour,
		RefreshCookieName:     "graft_refresh_token",
		RefreshCookiePath:     "/",
		RefreshCookieSameSite: "lax",
	}, &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username != "alice" {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:       7,
				Username:     "alice",
				PasswordHash: &passwordHash,
			}, nil
		},
	}, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{
				ID:       7,
				Username: "alice",
				Display:  "Alice",
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	result, err := authSvc.Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if result.User.ID != 7 || result.User.Username != "alice" {
		t.Fatalf("expected authenticated user summary, got %#v", result.User)
	}
}

// TestLogoutRouteRevokesCurrentRefreshSession 验证 logout 路由会读取当前 refresh
// cookie，吊销对应会话，并下发清除 cookie 的响应。
func TestLogoutRouteRevokesCurrentRefreshSession(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username != "alice" {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:       7,
				Username:     "alice",
				PasswordHash: &passwordHash,
			}, nil
		},
	}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{
				ID:        7,
				Username:  "alice",
				Display:   "Alice",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}, authRepo)

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, loginRecorder.Code)
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected refresh cookie from login")
	}

	manager, err := newRefreshTokenManager(config.AuthConfig{
		RefreshTokenTTL: 24 * time.Hour,
		SigningKey:      "test-signing-key",
	})
	if err != nil {
		t.Fatalf("new refresh token manager: %v", err)
	}
	claims, err := manager.Parse(cookies[0].Value)
	if err != nil {
		t.Fatalf("parse refresh cookie token: %v", err)
	}

	logoutRecorder := httptest.NewRecorder()
	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutRequest.AddCookie(cookies[0])
	engine.ServeHTTP(logoutRecorder, logoutRequest)

	if logoutRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected logout status %d, got %d", http.StatusNoContent, logoutRecorder.Code)
	}

	session, err := authRepo.GetRefreshSessionByTokenID(context.Background(), claims.TokenID)
	if err != nil {
		t.Fatalf("load revoked session: %v", err)
	}
	if session.RevokedAt == nil {
		t.Fatalf("expected refresh session to be revoked, got %#v", session)
	}

	responseCookies := logoutRecorder.Result().Cookies()
	if len(responseCookies) == 0 {
		t.Fatal("expected logout to clear refresh cookie")
	}
	if responseCookies[0].Name != cookies[0].Name || responseCookies[0].Value != "" || responseCookies[0].MaxAge >= 0 {
		t.Fatalf("expected cleared refresh cookie, got %#v", responseCookies[0])
	}
}

// TestLogoutRouteRejectsMissingCookie 验证缺少 refresh cookie 时，logout 继续复用
// 统一的本地化 refresh-session 错误契约。
func TestLogoutRouteRejectsMissingCookie(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, &pluginTestAuthRepository{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.invalid_refresh_session" || payload.Locale != "en-US" {
		t.Fatalf("expected invalid refresh payload, got %#v", payload)
	}
}

// TestRevokeAllSessionsRouteRevokesCurrentUserSessions 验证当前用户自助撤销会吊销
// 其全部 refresh sessions，并让当前受保护请求与后续 refresh 一并失效。
func TestRevokeAllSessionsRouteRevokesCurrentUserSessions(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username != "alice" {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:       7,
				Username:     "alice",
				PasswordHash: &passwordHash,
			}, nil
		},
	}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 7:
				return store.User{
					ID:        7,
					Username:  "alice",
					Display:   "Alice",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			case 8:
				return store.User{
					ID:        8,
					Username:  "bob",
					Display:   "Bob",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
	}, authRepo)

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, loginRecorder.Code)
	}

	var loginPayload loginResponse
	if err := json.NewDecoder(loginRecorder.Body).Decode(&loginPayload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	loginCookies := loginRecorder.Result().Cookies()
	if len(loginCookies) == 0 {
		t.Fatal("expected refresh cookie from login")
	}

	refreshManager, err := newRefreshTokenManager(config.AuthConfig{
		RefreshTokenTTL: 24 * time.Hour,
		SigningKey:      "test-signing-key",
	})
	if err != nil {
		t.Fatalf("new refresh token manager: %v", err)
	}
	currentClaims, err := refreshManager.Parse(loginCookies[0].Value)
	if err != nil {
		t.Fatalf("parse current refresh cookie: %v", err)
	}

	secondarySessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	otherUserSessionID := seedRefreshSession(t, authRepo, 8, time.Now().UTC().Add(time.Hour))

	revokeRecorder := httptest.NewRecorder()
	revokeRequest := httptest.NewRequest(http.MethodPost, "/api/auth/sessions/revoke-all", nil)
	revokeRequest.Header.Set("Authorization", "Bearer "+loginPayload.AccessToken)
	revokeRequest.AddCookie(loginCookies[0])
	engine.ServeHTTP(revokeRecorder, revokeRequest)

	if revokeRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected revoke-all status %d, got %d", http.StatusNoContent, revokeRecorder.Code)
	}

	for _, tokenID := range []string{currentClaims.TokenID, secondarySessionID} {
		session, err := authRepo.GetRefreshSessionByTokenID(context.Background(), tokenID)
		if err != nil {
			t.Fatalf("load revoked session %q: %v", tokenID, err)
		}
		if session.RevokedAt == nil {
			t.Fatalf("expected session %q to be revoked, got %#v", tokenID, session)
		}
	}

	otherUserSession, err := authRepo.GetRefreshSessionByTokenID(context.Background(), otherUserSessionID)
	if err != nil {
		t.Fatalf("load untouched session: %v", err)
	}
	if otherUserSession.RevokedAt != nil {
		t.Fatalf("expected other user session to remain active, got %#v", otherUserSession)
	}

	responseCookies := revokeRecorder.Result().Cookies()
	if len(responseCookies) == 0 {
		t.Fatal("expected revoke-all to clear refresh cookie")
	}
	if responseCookies[0].Name != loginCookies[0].Name || responseCookies[0].Value != "" || responseCookies[0].MaxAge >= 0 {
		t.Fatalf("expected cleared refresh cookie, got %#v", responseCookies[0])
	}

	protectedRecorder := httptest.NewRecorder()
	protectedRequest := httptest.NewRequest(http.MethodGet, "/api/users/7", nil)
	protectedRequest.Header.Set("Authorization", "Bearer "+loginPayload.AccessToken)
	protectedRequest.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(protectedRecorder, protectedRequest)

	if protectedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected protected request status %d, got %d", http.StatusUnauthorized, protectedRecorder.Code)
	}

	var protectedPayload httpx.ErrorResponse
	if err := json.NewDecoder(protectedRecorder.Body).Decode(&protectedPayload); err != nil {
		t.Fatalf("decode protected request response: %v", err)
	}
	if protectedPayload.MessageKey != "auth.missing_actor" || protectedPayload.Locale != "en-US" {
		t.Fatalf("expected missing actor payload after revoke-all, got %#v", protectedPayload)
	}

	refreshRecorder := httptest.NewRecorder()
	refreshRequest := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	refreshRequest.AddCookie(loginCookies[0])
	refreshRequest.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(refreshRecorder, refreshRequest)

	if refreshRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected refresh status %d after revoke-all, got %d", http.StatusUnauthorized, refreshRecorder.Code)
	}

	var refreshPayload httpx.ErrorResponse
	if err := json.NewDecoder(refreshRecorder.Body).Decode(&refreshPayload); err != nil {
		t.Fatalf("decode refresh response after revoke-all: %v", err)
	}
	if refreshPayload.MessageKey != "auth.invalid_refresh_session" || refreshPayload.Locale != "en-US" {
		t.Fatalf("expected invalid refresh payload after revoke-all, got %#v", refreshPayload)
	}
}

// TestRevokeAllSessionsRouteRequiresAuthenticatedActor 验证当前用户自助撤销入口继续
// 复用统一 request-auth 守卫，而不是在插件内发散新的未登录响应格式。
func TestRevokeAllSessionsRouteRequiresAuthenticatedActor(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, &pluginTestAuthRepository{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/sessions/revoke-all", nil)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.missing_actor" || payload.Locale != "en-US" {
		t.Fatalf("expected missing actor payload, got %#v", payload)
	}
}

// TestRevokeOtherSessionsRouteRevokesNonCurrentSessions 验证当前用户保留当前会话时，
// 只会清退自己名下的其它有效 session，不会误伤当前会话或其他用户会话。
func TestRevokeOtherSessionsRouteRevokesNonCurrentSessions(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 7:
				return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			case 8:
				return store.User{ID: 8, Username: "bob", Display: "Bob", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
	}, authRepo)

	currentSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	otherSessionIDs := []string{
		seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(3*time.Hour)),
		seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(4*time.Hour)),
	}
	otherUserSessionID := seedRefreshSession(t, authRepo, 8, time.Now().UTC().Add(time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, "/api/auth/sessions/revoke-others", 7, currentSessionID)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	currentSession, err := authRepo.GetRefreshSessionByTokenID(context.Background(), currentSessionID)
	if err != nil {
		t.Fatalf("load current session: %v", err)
	}
	if currentSession.RevokedAt != nil {
		t.Fatalf("expected current session to remain active, got %#v", currentSession)
	}

	for _, tokenID := range otherSessionIDs {
		session, err := authRepo.GetRefreshSessionByTokenID(context.Background(), tokenID)
		if err != nil {
			t.Fatalf("load other session %q: %v", tokenID, err)
		}
		if session.RevokedAt == nil {
			t.Fatalf("expected other session %q to be revoked, got %#v", tokenID, session)
		}
	}

	otherUserSession, err := authRepo.GetRefreshSessionByTokenID(context.Background(), otherUserSessionID)
	if err != nil {
		t.Fatalf("load other user session: %v", err)
	}
	if otherUserSession.RevokedAt != nil {
		t.Fatalf("expected other user session to remain active, got %#v", otherUserSession)
	}
}

// TestRevokeOtherSessionsRouteRequiresAuthenticatedActor 验证保留当前会话的批量清退
// 入口继续复用统一 request-auth 守卫，而不是发散新的未登录响应契约。
func TestRevokeOtherSessionsRouteRequiresAuthenticatedActor(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, &pluginTestAuthRepository{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/sessions/revoke-others", nil)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.missing_actor" || payload.Locale != "en-US" {
		t.Fatalf("expected missing actor payload, got %#v", payload)
	}
}

// TestRevokeOtherSessionsRouteAllowsOnlyCurrentSession 验证当前用户只剩当前会话时，
// revoke-others 仍幂等返回成功，且不会额外清理 refresh cookie。
func TestRevokeOtherSessionsRouteAllowsOnlyCurrentSession(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}, authRepo)

	currentSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, "/api/auth/sessions/revoke-others", 7, currentSessionID)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	currentSession, err := authRepo.GetRefreshSessionByTokenID(context.Background(), currentSessionID)
	if err != nil {
		t.Fatalf("load current session: %v", err)
	}
	if currentSession.RevokedAt != nil {
		t.Fatalf("expected current session to remain active, got %#v", currentSession)
	}

	if responseCookies := recorder.Result().Cookies(); len(responseCookies) != 0 {
		t.Fatalf("expected no cookie mutation, got %#v", responseCookies)
	}
}

// TestListCurrentUserSessionsRouteReturnsActiveSessions 验证当前用户自助会话列表只返回
// 其自身当前有效的 refresh sessions，并准确标记当前请求会话。
func TestListCurrentUserSessionsRouteReturnsActiveSessions(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 7:
				return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			case 8:
				return store.User{ID: 8, Username: "bob", Display: "Bob", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
	}, authRepo)

	currentSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	time.Sleep(time.Microsecond)
	newerSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	expiredSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(-time.Minute))
	otherUserSessionID := seedRefreshSession(t, authRepo, 8, time.Now().UTC().Add(time.Hour))
	if err := authRepo.RevokeRefreshSession(context.Background(), store.RevokeRefreshSessionInput{
		TokenID:   expiredSessionID,
		RevokedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("revoke expired test session: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, "/api/auth/sessions", 7, currentSessionID)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload []sessionSummary
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("expected two active sessions, got %#v", payload)
	}
	if payload[0].SessionID != newerSessionID || payload[0].Current {
		t.Fatalf("expected newer non-current session first, got %#v", payload[0])
	}
	if payload[1].SessionID != currentSessionID || !payload[1].Current {
		t.Fatalf("expected current session second and marked current, got %#v", payload[1])
	}
	for _, item := range payload {
		if item.SessionID == expiredSessionID || item.SessionID == otherUserSessionID {
			t.Fatalf("expected filtered sessions to be absent, got %#v", payload)
		}
	}
}

// TestListCurrentUserSessionsRouteAppliesLimit 验证当前用户会话列表会在插件边界内
// 应用显式 limit，而不要求仓储提前暴露分页协议。
func TestListCurrentUserSessionsRouteAppliesLimit(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}, authRepo)

	currentSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	time.Sleep(time.Microsecond)
	middleSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	time.Sleep(time.Microsecond)
	newestSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(3*time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, "/api/auth/sessions?limit=2", 7, currentSessionID)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload []sessionSummary
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("expected two sessions after limit, got %#v", payload)
	}
	if payload[0].SessionID != newestSessionID || payload[1].SessionID != middleSessionID {
		t.Fatalf("expected newest sessions after limit, got %#v", payload)
	}
	for _, item := range payload {
		if item.SessionID == currentSessionID {
			t.Fatalf("expected oldest current session to be trimmed by limit, got %#v", payload)
		}
	}
}

// TestListCurrentUserSessionsRouteRejectsInvalidLimit 验证当前用户会话列表会拒绝非法
// limit，保持稳定的 invalid_argument 契约。
func TestListCurrentUserSessionsRouteRejectsInvalidLimit(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}, authRepo)

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, "/api/auth/sessions?limit=0", 7, seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "common.invalid_argument" || payload.Locale != "en-US" {
		t.Fatalf("expected invalid argument payload, got %#v", payload)
	}
	if payload.Details["field"] != "limit" {
		t.Fatalf("expected denied field detail, got %#v", payload)
	}
}

// TestListCurrentUserSessionsRouteRequiresAuthenticatedActor 验证当前用户会话列表继续
// 复用统一 request-auth 守卫，而不是在插件内发散新的未登录契约。
func TestListCurrentUserSessionsRouteRequiresAuthenticatedActor(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, &pluginTestAuthRepository{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/auth/sessions", nil)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.missing_actor" || payload.Locale != "en-US" {
		t.Fatalf("expected missing actor payload, got %#v", payload)
	}
}

// TestAdminListUserSessionsRouteReturnsActiveSessions 验证管理员读取入口只返回目标用户
// 的当前有效 session，并继续标记请求主体自己的当前会话。
func TestAdminListUserSessionsRouteReturnsActiveSessions(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 7:
				return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			case 8:
				return store.User{ID: 8, Username: "bob", Display: "Bob", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			case 9:
				return store.User{ID: 9, Username: "admin", Display: "Admin", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
	}, authRepo, map[uint64][]store.Permission{
		9: {{Code: "user.session.read"}},
	})

	targetCurrentSession := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	time.Sleep(time.Microsecond)
	targetNewerSession := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(3*time.Hour))
	targetExpiredSession := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(-time.Minute))
	adminSession := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))
	otherUserSession := seedRefreshSession(t, authRepo, 8, time.Now().UTC().Add(time.Hour))
	if err := authRepo.RevokeRefreshSession(context.Background(), store.RevokeRefreshSessionInput{
		TokenID:   targetExpiredSession,
		RevokedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("revoke expired test session: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, "/api/users/7/sessions", 9, adminSession)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload []sessionSummary
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("expected two active target-user sessions, got %#v", payload)
	}
	if payload[0].SessionID != targetNewerSession || payload[0].Current {
		t.Fatalf("expected newer target-user session first, got %#v", payload[0])
	}
	if payload[1].SessionID != targetCurrentSession || payload[1].Current {
		t.Fatalf("expected target current list item not to be marked current for admin request, got %#v", payload[1])
	}
	for _, item := range payload {
		if item.SessionID == targetExpiredSession || item.SessionID == adminSession || item.SessionID == otherUserSession {
			t.Fatalf("expected filtered sessions to be absent, got %#v", payload)
		}
	}
}

// TestAdminListUserSessionsRouteAppliesLimit 验证管理员读取入口同样支持最小显式
// limit，避免首次会话治理就扩散分页契约到仓储层。
func TestAdminListUserSessionsRouteAppliesLimit(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 7:
				return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			case 9:
				return store.User{ID: 9, Username: "admin", Display: "Admin", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
	}, authRepo, map[uint64][]store.Permission{
		9: {{Code: "user.session.read"}},
	})

	oldestSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	time.Sleep(time.Microsecond)
	middleSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	time.Sleep(time.Microsecond)
	newestSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(3*time.Hour))
	adminSessionID := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, "/api/users/7/sessions?limit=2", 9, adminSessionID)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload []sessionSummary
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("expected two sessions after limit, got %#v", payload)
	}
	if payload[0].SessionID != newestSessionID || payload[1].SessionID != middleSessionID {
		t.Fatalf("expected newest target-user sessions after limit, got %#v", payload)
	}
	for _, item := range payload {
		if item.SessionID == oldestSessionID || item.SessionID == adminSessionID {
			t.Fatalf("expected oldest target or admin session to be absent, got %#v", payload)
		}
	}
}

// TestAdminListUserSessionsRouteRejectsInvalidLimit 验证管理员会话读取入口会拒绝非法
// limit，并保持统一的参数错误契约。
func TestAdminListUserSessionsRouteRejectsInvalidLimit(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 7:
				return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			case 9:
				return store.User{ID: 9, Username: "admin", Display: "Admin", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
	}, authRepo, map[uint64][]store.Permission{
		9: {{Code: "user.session.read"}},
	})

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, "/api/users/7/sessions?limit=101", 9, seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "common.invalid_argument" || payload.Locale != "en-US" {
		t.Fatalf("expected invalid argument payload, got %#v", payload)
	}
	if payload.Details["field"] != "limit" {
		t.Fatalf("expected denied field detail, got %#v", payload)
	}
}

// TestAdminListUserSessionsRouteRequiresDedicatedPermission 验证管理员读取入口不会误复用
// user.read，而是要求显式的 session 读取权限。
func TestAdminListUserSessionsRouteRequiresDedicatedPermission(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 7:
				return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			case 9:
				return store.User{ID: 9, Username: "admin", Display: "Admin", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
	}, authRepo)

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, "/api/users/7/sessions", 7, seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.missing_permission" || payload.Locale != "en-US" {
		t.Fatalf("expected missing permission payload, got %#v", payload)
	}
	if payload.Details["permission"] != "user.session.read" {
		t.Fatalf("expected denied permission detail, got %#v", payload)
	}
}

// TestAdminListUserSessionsRouteReturnsNotFoundContract 验证目标用户不存在时，会话读取入口
// 仍返回稳定的 user.not_found 契约，而不是把空结果伪装成成功。
func TestAdminListUserSessionsRouteReturnsNotFoundContract(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 9:
				return store.User{ID: 9, Username: "admin", Display: "Admin", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
	}, authRepo, map[uint64][]store.Permission{
		9: {{Code: "user.session.read"}},
	})

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, "/api/users/7/sessions", 9, seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "user.not_found" || payload.Locale != "en-US" {
		t.Fatalf("expected user.not_found payload, got %#v", payload)
	}
}

// TestRevokeCurrentUserSessionRouteRevokesOnlyTargetSession 验证当前用户定向吊销只会
// 影响指定 session，不会误伤同用户名下其他有效 session。
func TestRevokeCurrentUserSessionRouteRevokesOnlyTargetSession(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 7:
				return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
	}, authRepo)

	currentSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	targetSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(3*time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, "/api/auth/sessions/"+targetSessionID+"/revoke", 7, currentSessionID)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	targetSession, err := authRepo.GetRefreshSessionByTokenID(context.Background(), targetSessionID)
	if err != nil {
		t.Fatalf("load target session: %v", err)
	}
	if targetSession.RevokedAt == nil {
		t.Fatalf("expected target session to be revoked, got %#v", targetSession)
	}

	currentSession, err := authRepo.GetRefreshSessionByTokenID(context.Background(), currentSessionID)
	if err != nil {
		t.Fatalf("load current session: %v", err)
	}
	if currentSession.RevokedAt != nil {
		t.Fatalf("expected current session to remain active, got %#v", currentSession)
	}
}

// TestRevokeCurrentUserSessionRouteClearsCookieWhenRevokingCurrentSession 验证当前用户
// 吊销自己当前请求绑定的 session 时，会同步清理 refresh cookie。
func TestRevokeCurrentUserSessionRouteClearsCookieWhenRevokingCurrentSession(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username != "alice" {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:       7,
				Username:     "alice",
				PasswordHash: &passwordHash,
			}, nil
		},
	}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}, authRepo)

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, loginRecorder.Code)
	}

	var loginPayload loginResponse
	if err := json.NewDecoder(loginRecorder.Body).Decode(&loginPayload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	loginCookies := loginRecorder.Result().Cookies()
	if len(loginCookies) == 0 {
		t.Fatal("expected refresh cookie from login")
	}

	refreshManager, err := newRefreshTokenManager(config.AuthConfig{
		RefreshTokenTTL: 24 * time.Hour,
		SigningKey:      "test-signing-key",
	})
	if err != nil {
		t.Fatalf("new refresh token manager: %v", err)
	}
	currentClaims, err := refreshManager.Parse(loginCookies[0].Value)
	if err != nil {
		t.Fatalf("parse current refresh cookie: %v", err)
	}

	revokeRecorder := httptest.NewRecorder()
	revokeRequest := httptest.NewRequest(http.MethodPost, "/api/auth/sessions/"+currentClaims.TokenID+"/revoke", nil)
	revokeRequest.Header.Set("Authorization", "Bearer "+loginPayload.AccessToken)
	revokeRequest.AddCookie(loginCookies[0])
	engine.ServeHTTP(revokeRecorder, revokeRequest)

	if revokeRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, revokeRecorder.Code)
	}

	currentSession, err := authRepo.GetRefreshSessionByTokenID(context.Background(), currentClaims.TokenID)
	if err != nil {
		t.Fatalf("load current session: %v", err)
	}
	if currentSession.RevokedAt == nil {
		t.Fatalf("expected current session to be revoked, got %#v", currentSession)
	}

	responseCookies := revokeRecorder.Result().Cookies()
	if len(responseCookies) == 0 {
		t.Fatal("expected current-session revoke to clear refresh cookie")
	}
	if responseCookies[0].Name != loginCookies[0].Name || responseCookies[0].Value != "" || responseCookies[0].MaxAge >= 0 {
		t.Fatalf("expected cleared refresh cookie, got %#v", responseCookies[0])
	}
}

// TestRevokeCurrentUserSessionRouteReturnsNotFoundContract 验证当前用户定向吊销未命中时
// 返回稳定的 session-not-found 契约。
func TestRevokeCurrentUserSessionRouteReturnsNotFoundContract(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}, authRepo)

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, "/api/auth/sessions/missing-session/revoke", 7, seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.session_not_found" || payload.Locale != "en-US" {
		t.Fatalf("expected session not found payload, got %#v", payload)
	}
}

// TestAdminRevokeUserSessionRouteRevokesOnlyTargetSession 验证管理员定向吊销只会影响
// 目标用户的指定 session，不会误伤其他用户或同用户其他会话。
func TestAdminRevokeUserSessionRouteRevokesOnlyTargetSession(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 7:
				return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			case 8:
				return store.User{ID: 8, Username: "bob", Display: "Bob", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			case 9:
				return store.User{ID: 9, Username: "admin", Display: "Admin", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
	}, authRepo, map[uint64][]store.Permission{
		9: {{Code: "user.session.revoke"}},
	})

	targetSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	otherTargetSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(3*time.Hour))
	otherUserSessionID := seedRefreshSession(t, authRepo, 8, time.Now().UTC().Add(time.Hour))
	adminSessionID := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, "/api/users/7/sessions/"+targetSessionID+"/revoke", 9, adminSessionID)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	targetSession, err := authRepo.GetRefreshSessionByTokenID(context.Background(), targetSessionID)
	if err != nil {
		t.Fatalf("load target session: %v", err)
	}
	if targetSession.RevokedAt == nil {
		t.Fatalf("expected target session to be revoked, got %#v", targetSession)
	}

	for _, tokenID := range []string{otherTargetSessionID, otherUserSessionID, adminSessionID} {
		session, err := authRepo.GetRefreshSessionByTokenID(context.Background(), tokenID)
		if err != nil {
			t.Fatalf("load untouched session %q: %v", tokenID, err)
		}
		if session.RevokedAt != nil {
			t.Fatalf("expected session %q to remain active, got %#v", tokenID, session)
		}
	}
}

// TestAdminRevokeUserSessionRouteClearsCurrentCookieWhenRevokingSelfCurrentSession 验证管理员
// 定向吊销自己当前请求绑定的 session 时，会同步清理 refresh cookie。
func TestAdminRevokeUserSessionRouteClearsCurrentCookieWhenRevokingSelfCurrentSession(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username != "admin" {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:       9,
				Username:     "admin",
				PasswordHash: &passwordHash,
			}, nil
		},
	}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 9 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{ID: 9, Username: "admin", Display: "Admin", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}, authRepo, map[uint64][]store.Permission{
		9: {{Code: "user.session.revoke"}},
	})

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, loginRecorder.Code)
	}

	var loginPayload loginResponse
	if err := json.NewDecoder(loginRecorder.Body).Decode(&loginPayload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	loginCookies := loginRecorder.Result().Cookies()
	if len(loginCookies) == 0 {
		t.Fatal("expected refresh cookie from login")
	}

	refreshManager, err := newRefreshTokenManager(config.AuthConfig{
		RefreshTokenTTL: 24 * time.Hour,
		SigningKey:      "test-signing-key",
	})
	if err != nil {
		t.Fatalf("new refresh token manager: %v", err)
	}
	currentClaims, err := refreshManager.Parse(loginCookies[0].Value)
	if err != nil {
		t.Fatalf("parse current refresh cookie: %v", err)
	}

	revokeRecorder := httptest.NewRecorder()
	revokeRequest := httptest.NewRequest(http.MethodPost, "/api/users/9/sessions/"+currentClaims.TokenID+"/revoke", nil)
	revokeRequest.Header.Set("Authorization", "Bearer "+loginPayload.AccessToken)
	revokeRequest.AddCookie(loginCookies[0])
	engine.ServeHTTP(revokeRecorder, revokeRequest)

	if revokeRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, revokeRecorder.Code)
	}

	responseCookies := revokeRecorder.Result().Cookies()
	if len(responseCookies) == 0 {
		t.Fatal("expected self current-session revoke to clear refresh cookie")
	}
	if responseCookies[0].Name != loginCookies[0].Name || responseCookies[0].Value != "" || responseCookies[0].MaxAge >= 0 {
		t.Fatalf("expected cleared refresh cookie, got %#v", responseCookies[0])
	}
}

// TestAdminRevokeUserSessionRouteReturnsNotFoundContract 验证管理员定向吊销未命中时
// 返回稳定的 session-not-found 契约。
func TestAdminRevokeUserSessionRouteReturnsNotFoundContract(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 7:
				return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			case 9:
				return store.User{ID: 9, Username: "admin", Display: "Admin", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
	}, authRepo, map[uint64][]store.Permission{
		9: {{Code: "user.session.revoke"}},
	})

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, "/api/users/7/sessions/missing-session/revoke", 9, seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.session_not_found" || payload.Locale != "en-US" {
		t.Fatalf("expected session not found payload, got %#v", payload)
	}
}

// TestAdminRevokeUserSessionsRouteRevokesTargetUserSessions 验证管理员入口会按用户 ID
// 吊销目标用户的全部 refresh sessions，并保持其他用户 session 不受影响。
func TestAdminRevokeUserSessionsRouteRevokesTargetUserSessions(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username != "admin" {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:       9,
				Username:     "admin",
				PasswordHash: &passwordHash,
			}, nil
		},
	}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 7:
				return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			case 8:
				return store.User{ID: 8, Username: "bob", Display: "Bob", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			case 9:
				return store.User{ID: 9, Username: "admin", Display: "Admin", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
	}, authRepo, map[uint64][]store.Permission{
		9: {{Code: "user.session.revoke"}},
	})

	targetSessionA := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	targetSessionB := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	otherUserSession := seedRefreshSession(t, authRepo, 8, time.Now().UTC().Add(time.Hour))
	adminSession := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, "/api/users/7/sessions/revoke-all", 9, adminSession)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	for _, tokenID := range []string{targetSessionA, targetSessionB} {
		session, err := authRepo.GetRefreshSessionByTokenID(context.Background(), tokenID)
		if err != nil {
			t.Fatalf("load revoked session %q: %v", tokenID, err)
		}
		if session.RevokedAt == nil {
			t.Fatalf("expected target session %q to be revoked, got %#v", tokenID, session)
		}
	}

	for _, tokenID := range []string{otherUserSession, adminSession} {
		session, err := authRepo.GetRefreshSessionByTokenID(context.Background(), tokenID)
		if err != nil {
			t.Fatalf("load untouched session %q: %v", tokenID, err)
		}
		if session.RevokedAt != nil {
			t.Fatalf("expected session %q to remain active, got %#v", tokenID, session)
		}
	}

	if cookies := recorder.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("expected admin revoking another user not to clear cookies, got %#v", cookies)
	}
}

// TestAdminRevokeUserSessionsRouteClearsCurrentCookieWhenRevokingSelf 验证管理员按自己的
// 用户 ID 执行批量吊销时，会同步清理当前 refresh cookie。
func TestAdminRevokeUserSessionsRouteClearsCurrentCookieWhenRevokingSelf(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username != "admin" {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:       9,
				Username:     "admin",
				PasswordHash: &passwordHash,
			}, nil
		},
	}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 9 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{ID: 9, Username: "admin", Display: "Admin", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}, authRepo, map[uint64][]store.Permission{
		9: {{Code: "user.session.revoke"}},
	})

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, loginRecorder.Code)
	}

	var loginPayload loginResponse
	if err := json.NewDecoder(loginRecorder.Body).Decode(&loginPayload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	loginCookies := loginRecorder.Result().Cookies()
	if len(loginCookies) == 0 {
		t.Fatal("expected refresh cookie from login")
	}

	revokeRecorder := httptest.NewRecorder()
	revokeRequest := httptest.NewRequest(http.MethodPost, "/api/users/9/sessions/revoke-all", nil)
	revokeRequest.Header.Set("Authorization", "Bearer "+loginPayload.AccessToken)
	revokeRequest.AddCookie(loginCookies[0])
	engine.ServeHTTP(revokeRecorder, revokeRequest)

	if revokeRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, revokeRecorder.Code)
	}

	responseCookies := revokeRecorder.Result().Cookies()
	if len(responseCookies) == 0 {
		t.Fatal("expected self revoke to clear refresh cookie")
	}
	if responseCookies[0].Name != loginCookies[0].Name || responseCookies[0].Value != "" || responseCookies[0].MaxAge >= 0 {
		t.Fatalf("expected cleared refresh cookie, got %#v", responseCookies[0])
	}
}

// TestAdminRevokeUserSessionsRouteRequiresDedicatedPermission 验证管理员撤销入口不会
// 误复用 user.read，而是要求显式的 session 管理权限。
func TestAdminRevokeUserSessionsRouteRequiresDedicatedPermission(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{ID: 7, Username: "alice", Display: "Alice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}, authRepo)

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, "/api/users/8/sessions/revoke-all", 7, seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.missing_permission" || payload.Locale != "en-US" {
		t.Fatalf("expected missing permission payload, got %#v", payload)
	}
	if payload.Details["permission"] != "user.session.revoke" {
		t.Fatalf("expected denied permission detail, got %#v", payload)
	}
}

// TestAdminRevokeUserSessionsRouteRejectsInvalidID 验证管理员撤销入口会把非法用户 ID
// 收敛为稳定的参数错误响应。
func TestAdminRevokeUserSessionsRouteRejectsInvalidID(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 9 {
				return store.User{}, store.ErrUserNotFound
			}
			return store.User{ID: 9, Username: "admin", Display: "Admin", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}, authRepo, map[uint64][]store.Permission{
		9: {{Code: "user.session.revoke"}},
	})

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, "/api/users/not-a-number/sessions/revoke-all", 9, seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "common.invalid_argument" || payload.Locale != "en-US" {
		t.Fatalf("expected invalid argument payload, got %#v", payload)
	}
	if payload.Details["field"] != "id" {
		t.Fatalf("expected field detail to be id, got %#v", payload)
	}
}

// TestLoginRouteRejectsInvalidCredentials 验证用户名不存在或口令不匹配时，
// 登录接口会返回稳定的本地化认证失败响应。
func TestLoginRouteRejectsInvalidCredentials(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}

			return store.User{
				ID:        7,
				Username:  "alice",
				Display:   "Alice",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}, &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username != "alice" {
				return store.UserCredential{}, store.ErrUserNotFound
			}

			return store.UserCredential{
				UserID:       7,
				Username:     "alice",
				PasswordHash: &passwordHash,
			}, nil
		},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"wrong"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.invalid_credentials" || payload.Locale != "en-US" {
		t.Fatalf("expected invalid credentials payload, got %#v", payload)
	}
}

// TestLoginRouteRejectsMissingCredentials 验证缺失用户名或密码时，登录接口会
// 返回统一的参数校验错误而不是继续触发认证流程。
func TestLoginRouteRejectsMissingCredentials(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, &pluginTestAuthRepository{})

	tests := []struct {
		name       string
		body       string
		field      string
		wantStatus int
	}{
		{
			name:       "missing username",
			body:       `{"password":"secret"}`,
			field:      "username",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing password",
			body:       `{"username":"alice"}`,
			field:      "password",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			engine.ServeHTTP(recorder, request)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, recorder.Code)
			}

			var payload httpx.ErrorResponse
			if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload.MessageKey != "common.invalid_argument" {
				t.Fatalf("expected invalid argument payload, got %#v", payload)
			}
			if payload.Details["field"] != tc.field {
				t.Fatalf("expected %s field detail, got %#v", tc.field, payload.Details)
			}
		})
	}
}
