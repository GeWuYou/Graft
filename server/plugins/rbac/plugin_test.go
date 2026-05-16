package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/cronx"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
	rbaccontract "graft/server/plugins/rbac/contract"
)

type testRBACRepository struct {
	roles              []store.Role
	permissions        []store.Permission
	rolesByUserID      []store.Role
	permissionsByUser  []store.Permission
	listRolesErr       error
	listPermissionsErr error
	permissionsErr     error
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
	return r.rolesByUserID, nil
}

func (r testRBACRepository) ListRoles(_ context.Context) ([]store.Role, error) {
	if r.listRolesErr != nil {
		return nil, r.listRolesErr
	}
	return r.roles, nil
}

func (r testRBACRepository) ListPermissionsByUserID(_ context.Context, _ uint64) ([]store.Permission, error) {
	if r.permissionsErr != nil {
		return nil, r.permissionsErr
	}
	return r.permissionsByUser, nil
}

func (r testRBACRepository) ListPermissions(_ context.Context) ([]store.Permission, error) {
	if r.listPermissionsErr != nil {
		return nil, r.listPermissionsErr
	}
	return r.permissions, nil
}

type pluginTestStoreFactory struct {
	rbac store.RBACRepository
}

func (f pluginTestStoreFactory) Audit() store.AuditRepository { return nil }
func (f pluginTestStoreFactory) Users() store.UserRepository  { return nil }
func (f pluginTestStoreFactory) Auth() store.AuthRepository   { return nil }
func (f pluginTestStoreFactory) RBAC() store.RBACRepository   { return f.rbac }

type testAuthService struct {
	user pluginapi.CurrentUser
}

func (s testAuthService) CurrentUser(ctx context.Context) (*pluginapi.CurrentUser, error) {
	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return nil, pluginapi.ErrUnauthenticated
	}

	user := s.user
	return &user, nil
}

func (s testAuthService) ParseAccessToken(_ context.Context, token string) (*pluginapi.AccessTokenClaims, error) {
	if token == "" {
		return nil, pluginapi.ErrInvalidAccessToken
	}

	return &pluginapi.AccessTokenClaims{
		UserID:       s.user.ID,
		SessionID:    "session-1",
		TokenVersion: 1,
		IssuedAt:     time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(time.Minute),
	}, nil
}

func newPluginTestContext(t *testing.T, repo store.RBACRepository) (*plugin.Context, *gin.Engine) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctx := &plugin.Context{
		LifecycleContext:   context.Background(),
		Logger:             zap.NewNop(),
		Config:             &config.Config{},
		I18n:               i18n.New(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}),
		Router:             engine.Group("/api"),
		Services:           container.New(),
		Stores:             pluginTestStoreFactory{rbac: repo},
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
	}

	if err := ctx.Services.RegisterSingleton((*pluginapi.AuthService)(nil), func(container.Resolver) (any, error) {
		return testAuthService{
			user: pluginapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"},
		}, nil
	}); err != nil {
		t.Fatalf("register auth service: %v", err)
	}

	if err := NewPlugin().Register(ctx); err != nil {
		t.Fatalf("register rbac plugin: %v", err)
	}

	return ctx, engine
}

func newAuthorizedRequest(path string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer token")
	return request
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
			permissionsByUser: []store.Permission{{Code: "user.read"}},
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
			permissionsByUser: []store.Permission{{Code: "dashboard.view"}},
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
			permissionsErr: repositoryErr,
		},
	}

	err := service.Authorize(context.Background(), pluginapi.RequestAuthContext{
		User: &pluginapi.CurrentUser{ID: 7},
	}, "user.read")
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

// TestRegisterRegistersReadManagementContracts 验证 RBAC 插件会注册稳定的权限、菜单和共享授权服务。
func TestRegisterRegistersReadManagementContracts(t *testing.T) {
	ctx, _ := newPluginTestContext(t, testRBACRepository{})

	items := ctx.PermissionRegistry.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 registered permissions, got %d", len(items))
	}
	if items[0].Code != rbaccontract.RoleReadPermission.String() || items[1].Code != rbaccontract.PermissionReadPermission.String() {
		t.Fatalf("unexpected registered permissions: %#v", items)
	}

	menus := ctx.MenuRegistry.Items()
	if len(menus) != 1 {
		t.Fatalf("expected 1 registered menu, got %d", len(menus))
	}
	if menus[0].Path != rbaccontract.RolesGroup || menus[0].Permission != rbaccontract.RoleReadPermission.String() {
		t.Fatalf("unexpected registered menu: %#v", menus[0])
	}

	resolved, err := ctx.Services.Resolve((*pluginapi.Authorizer)(nil))
	if err != nil {
		t.Fatalf("resolve authorizer: %v", err)
	}
	if _, ok := resolved.(pluginapi.Authorizer); !ok {
		t.Fatalf("expected pluginapi.Authorizer, got %T", resolved)
	}
}

// TestRoleRoutesListRoles 验证角色只读接口会复用统一鉴权与成功 envelope。
func TestRoleRoutesListRoles(t *testing.T) {
	description := "Platform administrators"
	repo := testRBACRepository{
		roles: []store.Role{
			{
				ID:          1,
				Name:        "admin",
				Display:     "管理员",
				Description: &description,
				Builtin:     true,
			},
		},
		permissionsByUser: []store.Permission{{Code: rbaccontract.RoleReadPermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedRequest("/api/roles"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload httpx.SuccessResponse[roleListResponse]
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Success || payload.Code != "OK" {
		t.Fatalf("expected success envelope, got %#v", payload)
	}
	if len(payload.Data.Items) != 1 {
		t.Fatalf("expected one role item, got %#v", payload.Data.Items)
	}
	if payload.Data.Items[0].Builtin != true || payload.Data.Items[0].Name != "admin" {
		t.Fatalf("unexpected role item: %#v", payload.Data.Items[0])
	}
}

// TestPermissionRoutesRejectMissingPermission 验证只读权限接口仍以后端授权结果作为最终边界。
func TestPermissionRoutesRejectMissingPermission(t *testing.T) {
	repo := testRBACRepository{
		permissionsByUser: []store.Permission{{Code: rbaccontract.RoleReadPermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequest("/api/permissions")
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != messagecontract.AuthForbidden.String() || payload.Code != "AUTH_FORBIDDEN" {
		t.Fatalf("unexpected forbidden payload: %#v", payload)
	}
	if payload.Locale != "en-US" {
		t.Fatalf("expected locale en-US, got %#v", payload)
	}
	if payload.Details["permission"] != rbaccontract.PermissionReadPermission.String() {
		t.Fatalf("expected denied permission detail, got %#v", payload)
	}
}

// TestPermissionRoutesPropagateReadFailure 验证仓储读取失败会走统一本地化内部错误响应。
func TestPermissionRoutesPropagateReadFailure(t *testing.T) {
	repo := testRBACRepository{
		permissionsByUser:  []store.Permission{{Code: rbaccontract.PermissionReadPermission.String()}},
		listPermissionsErr: errors.New("list permissions failed"),
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequest("/api/permissions")
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != messagecontract.CommonInternalError.String() || payload.Code != "COMMON_INTERNAL_ERROR" {
		t.Fatalf("unexpected internal-error payload: %#v", payload)
	}
	if payload.Locale != "en-US" {
		t.Fatalf("expected locale en-US, got %#v", payload)
	}
}
