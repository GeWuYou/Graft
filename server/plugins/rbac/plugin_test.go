package rbac

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
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
	"graft/server/plugins/rbac/storeadapter"
)

type testRBACRepository struct {
	roles              []store.Role
	permissions        []store.Permission
	rolesByUserID      []store.Role
	permissionsByUser  []store.Permission
	rolePermissionIDs  map[uint64][]uint64
	roleByID           map[uint64]store.Role
	listRolesFn        func(ctx context.Context) ([]store.Role, error)
	listPermissionsFn  func(ctx context.Context) ([]store.Permission, error)
	createRole         func(ctx context.Context, input store.CreateRoleInput) (store.Role, error)
	updateRole         func(ctx context.Context, input store.UpdateRoleInput) (store.Role, error)
	replacePermission  func(ctx context.Context, input store.ReplacePermissionsForRoleInput) error
	replaceUserRoles   func(ctx context.Context, input store.ReplaceRolesForUserInput) error
	listRolesErr       error
	listPermissionsErr error
	permissionsErr     error
}

type testUserRepository struct {
	users map[uint64]store.User
}

func (r testUserRepository) GetByID(_ context.Context, id uint64) (store.User, error) {
	if user, ok := r.users[id]; ok {
		return user, nil
	}

	return store.User{}, store.ErrUserNotFound
}

func (r testUserRepository) List(_ context.Context) ([]store.User, error) {
	items := make([]store.User, 0, len(r.users))
	for _, user := range r.users {
		items = append(items, user)
	}

	return items, nil
}

type testUserService struct {
	users map[uint64]store.User
}

func (s testUserService) GetUserByID(_ context.Context, id uint64) (pluginapi.UserSummary, error) {
	user, ok := s.users[id]
	if !ok {
		return pluginapi.UserSummary{}, pluginapi.ErrUserNotFound
	}

	return pluginapi.UserSummary{
		ID:       user.ID,
		Username: user.Username,
		Display:  user.Display,
	}, nil
}

func (r testRBACRepository) EnsureRole(_ context.Context, _ store.EnsureRoleInput) (store.Role, error) {
	return store.Role{}, nil
}

func (r testRBACRepository) EnsurePermission(_ context.Context, _ store.EnsurePermissionInput) (store.Permission, error) {
	return store.Permission{}, nil
}

func (r testRBACRepository) CreateRole(ctx context.Context, input store.CreateRoleInput) (store.Role, error) {
	if r.createRole != nil {
		return r.createRole(ctx, input)
	}

	return store.Role{ID: 1, Name: input.Name, Display: input.Display, Description: input.Description, Builtin: input.Builtin}, nil
}

func (r testRBACRepository) UpdateRole(ctx context.Context, input store.UpdateRoleInput) (store.Role, error) {
	if r.updateRole != nil {
		return r.updateRole(ctx, input)
	}

	return store.Role{ID: input.ID, Name: input.Name, Display: input.Display, Description: input.Description}, nil
}

func (r testRBACRepository) AssignPermissionsToRole(_ context.Context, _ store.AssignPermissionsToRoleInput) error {
	return nil
}

func (r testRBACRepository) ReplacePermissionsForRole(ctx context.Context, input store.ReplacePermissionsForRoleInput) error {
	if r.replacePermission != nil {
		return r.replacePermission(ctx, input)
	}

	return nil
}

func (r testRBACRepository) AssignRoleToUser(_ context.Context, _ store.AssignRoleToUserInput) error {
	return nil
}

func (r testRBACRepository) ReplaceRolesForUser(ctx context.Context, input store.ReplaceRolesForUserInput) error {
	if r.replaceUserRoles != nil {
		return r.replaceUserRoles(ctx, input)
	}

	return nil
}

func (r testRBACRepository) GetRoleByID(ctx context.Context, roleID uint64) (store.Role, error) {
	_ = ctx
	if r.roleByID != nil {
		if role, ok := r.roleByID[roleID]; ok {
			return role, nil
		}
	}

	return store.Role{}, store.ErrRoleNotFound
}

func (r testRBACRepository) ListRolesByUserID(_ context.Context, _ uint64) ([]store.Role, error) {
	return r.rolesByUserID, nil
}

func (r testRBACRepository) ListRoles(ctx context.Context) ([]store.Role, error) {
	if r.listRolesFn != nil {
		return r.listRolesFn(ctx)
	}
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

func (r testRBACRepository) ListPermissions(ctx context.Context) ([]store.Permission, error) {
	if r.listPermissionsFn != nil {
		return r.listPermissionsFn(ctx)
	}
	if r.listPermissionsErr != nil {
		return nil, r.listPermissionsErr
	}
	return r.permissions, nil
}

func (r testRBACRepository) ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]store.RolePermissionBinding, error) {
	if _, err := r.GetRoleByID(ctx, roleID); err != nil {
		return nil, err
	}

	permissionIDs := r.rolePermissionIDs[roleID]
	bindings := make([]store.RolePermissionBinding, 0, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		bindings = append(bindings, store.RolePermissionBinding{
			RoleID:       roleID,
			PermissionID: permissionID,
		})
	}

	return bindings, nil
}

type pluginTestStoreFactory struct {
	users store.UserRepository
	rbac  store.RBACRepository
}

func (f pluginTestStoreFactory) Audit() store.AuditRepository { return nil }
func (f pluginTestStoreFactory) Users() store.UserRepository  { return f.users }
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
		LifecycleContext: context.Background(),
		Logger:           zap.NewNop(),
		Config:           &config.Config{},
		I18n:             i18n.New(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}),
		Router:           engine.Group("/api"),
		Services:         container.New(),
		Stores: pluginTestStoreFactory{
			users: testUserRepository{
				users: map[uint64]store.User{
					7: {ID: 7, Username: "alice", Display: "Alice"},
					8: {ID: 8, Username: "bob", Display: "Bob"},
				},
			},
			rbac: repo,
		},
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
	if err := ctx.Services.RegisterSingleton((*pluginapi.UserService)(nil), func(container.Resolver) (any, error) {
		return testUserService{
			users: map[uint64]store.User{
				7: {ID: 7, Username: "alice", Display: "Alice"},
				8: {ID: 8, Username: "bob", Display: "Bob"},
			},
		}, nil
	}); err != nil {
		t.Fatalf("register user service: %v", err)
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

func newAuthorizedJSONRequest(method string, path string, body any) *http.Request {
	payload, _ := json.Marshal(body)
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Content-Type", "application/json")
	return request
}

// TestAuthorizerRejectsUnauthenticatedRequest 验证缺少主体时会返回稳定未登录错误。
func TestAuthorizerRejectsUnauthenticatedRequest(t *testing.T) {
	service := authorizer{rbac: storeadapter.NewInternalRepositoryAdapter(testRBACRepository{})}

	err := service.Authorize(context.Background(), pluginapi.RequestAuthContext{}, "user.read")
	if !errors.Is(err, pluginapi.ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated, got %v", err)
	}
}

// TestAuthorizerAllowsGrantedPermission 验证命中的权限码会被授权通过。
func TestAuthorizerAllowsGrantedPermission(t *testing.T) {
	service := authorizer{
		rbac: storeadapter.NewInternalRepositoryAdapter(testRBACRepository{
			permissionsByUser: []store.Permission{{Code: "user.read"}},
		}),
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
		rbac: storeadapter.NewInternalRepositoryAdapter(testRBACRepository{
			permissionsByUser: []store.Permission{{Code: "dashboard.view"}},
		}),
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
		rbac: storeadapter.NewInternalRepositoryAdapter(testRBACRepository{
			permissionsErr: repositoryErr,
		}),
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
	if len(items) != 7 {
		t.Fatalf("expected 7 registered permissions, got %d", len(items))
	}
	assertRegisteredPermissionCodes(t, items)
	for _, item := range items {
		if item.Category != "api" {
			t.Fatalf("expected registered permission %s to declare category api, got %#v", item.Code, item)
		}
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

func assertRegisteredPermissionCodes(t *testing.T, items []permission.Item) {
	t.Helper()

	actual := make([]string, 0, len(items))
	for _, item := range items {
		actual = append(actual, item.Code)
	}

	expected := []string{
		rbaccontract.RoleReadPermission.String(),
		rbaccontract.RoleCreatePermission.String(),
		rbaccontract.RoleUpdatePermission.String(),
		rbaccontract.RolePermissionAssignPermission.String(),
		rbaccontract.PermissionReadPermission.String(),
		rbaccontract.UserRoleReadPermission.String(),
		rbaccontract.UserRoleAssignPermission.String(),
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected registered permissions: %#v", items)
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

// TestRoleRoutesListRolePermissionBindings 验证角色权限绑定读取接口会返回稳定权限 ID 快照。
func TestRoleRoutesListRolePermissionBindings(t *testing.T) {
	repo := testRBACRepository{
		roleByID: map[uint64]store.Role{
			1: {ID: 1, Name: "admin", Display: "管理员", Builtin: true},
		},
		rolePermissionIDs: map[uint64][]uint64{
			1: {2, 5},
		},
		permissionsByUser: []store.Permission{{Code: rbaccontract.RolePermissionAssignPermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedRequest("/api/roles/1/permissions"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload httpx.SuccessResponse[rolePermissionBindingResponse]
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.PermissionIDs) != 2 || payload.Data.PermissionIDs[0] != 2 || payload.Data.PermissionIDs[1] != 5 {
		t.Fatalf("unexpected role permission bindings payload: %#v", payload)
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

// TestRoleCreateRouteCreatesRole 验证最小角色创建接口会复用统一鉴权与成功 envelope。
func TestRoleCreateRouteCreatesRole(t *testing.T) {
	description := "Operators"
	repo := testRBACRepository{
		createRole: func(_ context.Context, input store.CreateRoleInput) (store.Role, error) {
			if input.Name != "operator" || input.Display != "运维" || input.Description == nil || *input.Description != description {
				t.Fatalf("unexpected create role input: %#v", input)
			}
			if input.Builtin {
				t.Fatal("expected created role to remain non-builtin")
			}

			return store.Role{
				ID:          3,
				Name:        input.Name,
				Display:     input.Display,
				Description: input.Description,
				Builtin:     false,
			}, nil
		},
		permissionsByUser: []store.Permission{{Code: rbaccontract.RoleCreatePermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedJSONRequest(http.MethodPost, "/api/roles", map[string]any{
		"name":        "operator",
		"display":     "运维",
		"description": description,
	}))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload httpx.SuccessResponse[roleListItem]
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Success || payload.Data.ID != 3 || payload.Data.Name != "operator" {
		t.Fatalf("unexpected create-role payload: %#v", payload)
	}
}

// TestRoleUpdateRouteRejectsBuiltinRoleRename 验证 builtin 角色的稳定名称不会被写接口改掉。
func TestRoleUpdateRouteRejectsBuiltinRoleRename(t *testing.T) {
	repo := testRBACRepository{
		roleByID: map[uint64]store.Role{
			1: {ID: 1, Name: "admin", Display: "管理员", Builtin: true},
		},
		permissionsByUser: []store.Permission{{Code: rbaccontract.RoleUpdatePermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedJSONRequest(http.MethodPost, "/api/roles/1/update", map[string]any{
		"name":    "admin-renamed",
		"display": "管理员",
	}))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != messagecontract.CommonInvalidArgument.String() || payload.Details["field"] != "name" {
		t.Fatalf("unexpected builtin-role update payload: %#v", payload)
	}
}

// TestRolePermissionAssignRouteReplacesRolePermissions 验证最小角色权限分配接口走覆盖式仓储写入。
func TestRolePermissionAssignRouteReplacesRolePermissions(t *testing.T) {
	repo := testRBACRepository{
		roleByID: map[uint64]store.Role{
			1: {ID: 1, Name: "editor", Display: "编辑"},
		},
		permissions: []store.Permission{
			{ID: 2, Code: "user.read"},
			{ID: 3, Code: "role.read"},
		},
		replacePermission: func(_ context.Context, input store.ReplacePermissionsForRoleInput) error {
			if input.RoleID != 1 || len(input.PermissionIDs) != 2 || input.PermissionIDs[0] != 2 || input.PermissionIDs[1] != 3 {
				t.Fatalf("unexpected replace role permissions input: %#v", input)
			}
			return nil
		},
		permissionsByUser: []store.Permission{{Code: rbaccontract.RolePermissionAssignPermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedJSONRequest(http.MethodPost, "/api/roles/1/permissions/assign", map[string]any{
		"permission_ids": []uint64{2, 3},
	}))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

// TestRolePermissionAssignRouteMapsMissingPermissionToInvalidArgument 验证 replace 语义中的权限未命中仍稳定映射为参数错误。
func TestRolePermissionAssignRouteMapsMissingPermissionToInvalidArgument(t *testing.T) {
	repo := testRBACRepository{
		roleByID: map[uint64]store.Role{
			1: {ID: 1, Name: "editor", Display: "编辑"},
		},
		permissions: []store.Permission{
			{ID: 2, Code: "user.read"},
		},
		replacePermission: func(_ context.Context, _ store.ReplacePermissionsForRoleInput) error {
			return store.ErrPermissionNotFound
		},
		permissionsByUser: []store.Permission{
			{Code: rbaccontract.RolePermissionAssignPermission.String()},
			{Code: rbaccontract.PermissionReadPermission.String()},
		},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedJSONRequest(http.MethodPost, "/api/roles/1/permissions/assign", map[string]any{
		"permission_ids": []uint64{99},
	}))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != messagecontract.CommonInvalidArgument.String() || payload.Details["field"] != "permission_ids" {
		t.Fatalf("unexpected invalid-permission payload: %#v", payload)
	}
}

// TestRolePermissionAssignRouteMapsDeletedPermissionIDsToInvalidArgument 验证 TOCTOU 后消失的权限 ID 会稳定映射为参数错误。
func TestRolePermissionAssignRouteMapsDeletedPermissionIDsToInvalidArgument(t *testing.T) {
	listPermissionsCalls := 0
	repo := testRBACRepository{
		roleByID: map[uint64]store.Role{
			1: {ID: 1, Name: "editor", Display: "编辑"},
		},
		listPermissionsFn: func(_ context.Context) ([]store.Permission, error) {
			listPermissionsCalls++
			if listPermissionsCalls == 1 {
				return []store.Permission{
					{ID: 2, Code: "user.read"},
					{ID: 3, Code: "role.read"},
				}, nil
			}

			return []store.Permission{
				{ID: 2, Code: "user.read"},
			}, nil
		},
		replacePermission: func(_ context.Context, _ store.ReplacePermissionsForRoleInput) error {
			return store.ErrPermissionNotFound
		},
		permissionsByUser: []store.Permission{
			{Code: rbaccontract.RolePermissionAssignPermission.String()},
			{Code: rbaccontract.PermissionReadPermission.String()},
		},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedJSONRequest(http.MethodPost, "/api/roles/1/permissions/assign", map[string]any{
		"permission_ids": []uint64{2, 3},
	}))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != messagecontract.CommonInvalidArgument.String() || payload.Details["field"] != "permission_ids" {
		t.Fatalf("unexpected deleted-permission payload: %#v", payload)
	}
}

// TestUserRoleBindingRouteReturnsStableRoleIDs 验证目标用户角色读取接口会返回稳定排序的角色 ID 快照。
func TestUserRoleBindingRouteReturnsStableRoleIDs(t *testing.T) {
	repo := testRBACRepository{
		rolesByUserID: []store.Role{
			{ID: 5, Name: "operator", Display: "运维"},
			{ID: 2, Name: "editor", Display: "编辑"},
		},
		permissionsByUser: []store.Permission{{Code: rbaccontract.UserRoleReadPermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedRequest("/api/users/7/roles"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var payload httpx.SuccessResponse[userRoleBindingResponse]
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.RoleIDs) != 2 || payload.Data.RoleIDs[0] != 2 || payload.Data.RoleIDs[1] != 5 {
		t.Fatalf("unexpected user-role bindings payload: %#v", payload)
	}
}

// TestUserRoleBindingRouteReturnsUserNotFound 验证目标用户未命中时会返回稳定的用户未命中语义。
func TestUserRoleBindingRouteReturnsUserNotFound(t *testing.T) {
	repo := testRBACRepository{
		permissionsByUser: []store.Permission{{Code: rbaccontract.UserRoleReadPermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequest("/api/users/99/roles")
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != messagecontract.UserNotFound.String() || payload.Code != "USER_NOT_FOUND" || payload.Locale != "en-US" {
		t.Fatalf("unexpected user-role-binding payload: %#v", payload)
	}
}

// TestUserRoleAssignRouteReturnsUserNotFound 验证用户角色分配接口会保留稳定的用户未命中语义。
func TestUserRoleAssignRouteReturnsUserNotFound(t *testing.T) {
	repo := testRBACRepository{
		replaceUserRoles: func(_ context.Context, _ store.ReplaceRolesForUserInput) error {
			return store.ErrUserNotFound
		},
		permissionsByUser: []store.Permission{{Code: rbaccontract.UserRoleAssignPermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	request := newAuthorizedJSONRequest(http.MethodPost, "/api/users/7/roles/assign", map[string]any{
		"role_ids": []uint64{1},
	})
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != messagecontract.UserNotFound.String() || payload.Code != "USER_NOT_FOUND" || payload.Locale != "en-US" {
		t.Fatalf("unexpected user-role-assign payload: %#v", payload)
	}
}

// TestUserRoleAssignRouteMapsMissingRoleToInvalidArgument 验证 replace 语义中的角色未命中仍稳定映射为参数错误。
func TestUserRoleAssignRouteMapsMissingRoleToInvalidArgument(t *testing.T) {
	repo := testRBACRepository{
		roles: []store.Role{
			{ID: 1, Name: "editor", Display: "编辑"},
		},
		replaceUserRoles: func(_ context.Context, _ store.ReplaceRolesForUserInput) error {
			return store.ErrRoleNotFound
		},
		permissionsByUser: []store.Permission{{Code: rbaccontract.UserRoleAssignPermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedJSONRequest(http.MethodPost, "/api/users/7/roles/assign", map[string]any{
		"role_ids": []uint64{99},
	}))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != messagecontract.CommonInvalidArgument.String() || payload.Details["field"] != "role_ids" {
		t.Fatalf("unexpected invalid-role payload: %#v", payload)
	}
}

// TestUserRoleAssignRouteMapsDeletedRoleIDsToInvalidArgument 验证 TOCTOU 后消失的角色 ID 会稳定映射为参数错误。
func TestUserRoleAssignRouteMapsDeletedRoleIDsToInvalidArgument(t *testing.T) {
	listRolesCalls := 0
	repo := testRBACRepository{
		listRolesFn: func(_ context.Context) ([]store.Role, error) {
			listRolesCalls++
			if listRolesCalls == 1 {
				return []store.Role{
					{ID: 1, Name: "admin", Display: "管理员"},
					{ID: 2, Name: "editor", Display: "编辑"},
				}, nil
			}

			return []store.Role{
				{ID: 1, Name: "admin", Display: "管理员"},
			}, nil
		},
		replaceUserRoles: func(_ context.Context, _ store.ReplaceRolesForUserInput) error {
			return store.ErrRoleNotFound
		},
		permissionsByUser: []store.Permission{{Code: rbaccontract.UserRoleAssignPermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedJSONRequest(http.MethodPost, "/api/users/7/roles/assign", map[string]any{
		"role_ids": []uint64{1, 2},
	}))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != messagecontract.CommonInvalidArgument.String() || payload.Details["field"] != "role_ids" {
		t.Fatalf("unexpected deleted-role payload: %#v", payload)
	}
}

// TestUserRoleAssignRouteRejectsRemovingOwnBuiltinAdmin 验证当前管理员不能移除自己最后的 builtin admin 角色。
func TestUserRoleAssignRouteRejectsRemovingOwnBuiltinAdmin(t *testing.T) {
	repo := testRBACRepository{
		roles: []store.Role{
			{ID: 1, Name: "admin", Display: "管理员", Builtin: true},
			{ID: 2, Name: "editor", Display: "编辑"},
		},
		rolesByUserID: []store.Role{
			{ID: 1, Name: "admin", Display: "管理员", Builtin: true},
			{ID: 2, Name: "editor", Display: "编辑"},
		},
		permissionsByUser: []store.Permission{{Code: rbaccontract.UserRoleAssignPermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	request := newAuthorizedJSONRequest(http.MethodPost, "/api/users/7/roles/assign", map[string]any{
		"role_ids": []uint64{2},
	})
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != messagecontract.RbacCannotRemoveOwnAdminRole.String() ||
		payload.Code != "RBAC_CANNOT_REMOVE_OWN_ADMIN_ROLE" ||
		payload.Locale != "en-US" {
		t.Fatalf("unexpected self-lockout payload: %#v", payload)
	}
}

// TestUserRoleAssignRouteAllowsRetainingOwnBuiltinAdmin 验证当前管理员保留 builtin admin 时仍可更新自己的角色集合。
func TestUserRoleAssignRouteAllowsRetainingOwnBuiltinAdmin(t *testing.T) {
	var received store.ReplaceRolesForUserInput
	repo := testRBACRepository{
		roles: []store.Role{
			{ID: 1, Name: "admin", Display: "管理员", Builtin: true},
			{ID: 2, Name: "editor", Display: "编辑"},
		},
		rolesByUserID: []store.Role{
			{ID: 1, Name: "admin", Display: "管理员", Builtin: true},
		},
		replaceUserRoles: func(_ context.Context, input store.ReplaceRolesForUserInput) error {
			received = input
			return nil
		},
		permissionsByUser: []store.Permission{{Code: rbaccontract.UserRoleAssignPermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedJSONRequest(http.MethodPost, "/api/users/7/roles/assign", map[string]any{
		"role_ids": []uint64{1, 2},
	}))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !reflect.DeepEqual(received, store.ReplaceRolesForUserInput{UserID: 7, RoleIDs: []uint64{1, 2}}) {
		t.Fatalf("unexpected replace input: %#v", received)
	}
}

// TestUserRoleAssignRouteAllowsRemovingBuiltinAdminFromOtherUser 验证管理员给其他用户改角色时仍保持原有 replace 语义。
func TestUserRoleAssignRouteAllowsRemovingBuiltinAdminFromOtherUser(t *testing.T) {
	var received store.ReplaceRolesForUserInput
	repo := testRBACRepository{
		roles: []store.Role{
			{ID: 1, Name: "admin", Display: "管理员", Builtin: true},
			{ID: 2, Name: "editor", Display: "编辑"},
		},
		rolesByUserID: []store.Role{
			{ID: 1, Name: "admin", Display: "管理员", Builtin: true},
		},
		replaceUserRoles: func(_ context.Context, input store.ReplaceRolesForUserInput) error {
			received = input
			return nil
		},
		permissionsByUser: []store.Permission{{Code: rbaccontract.UserRoleAssignPermission.String()}},
	}
	_, engine := newPluginTestContext(t, repo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedJSONRequest(http.MethodPost, "/api/users/9/roles/assign", map[string]any{
		"role_ids": []uint64{2},
	}))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !reflect.DeepEqual(received, store.ReplaceRolesForUserInput{UserID: 9, RoleIDs: []uint64{2}}) {
		t.Fatalf("unexpected replace input for other user: %#v", received)
	}
}
