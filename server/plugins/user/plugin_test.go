package user

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/crypto/bcrypt"

	"graft/server/internal/config"
	"graft/server/internal/container"
	authcontract "graft/server/internal/contract/auth"
	errorcodecontract "graft/server/internal/contract/errorcode"
	httpheadercontract "graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	useropenapi "graft/server/internal/contract/openapi/user"
	"graft/server/internal/cronx"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	authplugin "graft/server/plugins/auth"
	"graft/server/plugins/rbac"
	rbacstore "graft/server/plugins/rbac/store"
	usercontract "graft/server/plugins/user/contract"
	store "graft/server/plugins/user/store"
)

type successEnvelope[T any] struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"traceId"`
	Data    T      `json:"data"`
}

type loginResponse struct {
	AccessToken        string            `json:"access_token"`
	ExpiresAt          time.Time         `json:"expires_at"`
	MustChangePassword bool              `json:"must_change_password"`
	User               loginUserResponse `json:"user"`
}

const testAPIBasePath = "/api"

func decodeSuccessData[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	var payload successEnvelope[T]
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode success envelope: %v", err)
	}
	if !payload.Success || payload.Code != "OK" || payload.TraceID == "" {
		t.Fatalf("expected stable success envelope, got %#v", payload)
	}
	if recorder.Header().Get(httpx.RequestIDHeader) != payload.TraceID {
		t.Fatalf("expected response header trace id to match payload, got header=%q payload=%#v", recorder.Header().Get(httpx.RequestIDHeader), payload)
	}

	return payload.Data
}

func adaptTestAuthRepository(repo store.AuthRepository) userstoreAuthPair {
	pair := userstoreAuthPair{auth: repo}
	if passwordRepo, ok := repo.(store.PasswordChangeRepository); ok {
		pair.passwordChanges = passwordRepo
	}
	return pair
}

type userstoreAuthPair struct {
	auth            store.AuthRepository
	passwordChanges store.PasswordChangeRepository
}

// pluginTestAuthRepository 以内存状态模拟认证仓储的最小行为。
type pluginTestAuthRepository struct {
	getUserCredentialByUsername func(ctx context.Context, username string) (store.UserCredential, error)
	ensureUserCredential        func(ctx context.Context, input store.EnsureUserCredentialInput) (store.UserCredential, error)
	setPasswordHash             func(ctx context.Context, input store.SetPasswordHashInput) error
	resetPasswordAndRevoke      func(ctx context.Context, input store.ResetPasswordAndRevokeSessionsInput) error
	mu                          sync.Mutex
	refreshSessions             map[string]store.RefreshSession
}

// revokeByUserRaceAuthRepository 在测试中模拟“列出后、定向吊销前”目标 session
// 已被并发撤销的窗口，验证 revoke-others 的幂等语义。
type revokeByUserRaceAuthRepository struct {
	*pluginTestAuthRepository
	beforeFirstRevoke func(input store.RevokeRefreshSessionByUserIDInput)
	once              sync.Once
}

func (r *revokeByUserRaceAuthRepository) RevokeRefreshSessionByUserID(ctx context.Context, input store.RevokeRefreshSessionByUserIDInput) error {
	if r.beforeFirstRevoke != nil {
		r.once.Do(func() {
			r.beforeFirstRevoke(input)
		})
	}

	return r.pluginTestAuthRepository.RevokeRefreshSessionByUserID(ctx, input)
}

func (r *pluginTestAuthRepository) GetUserCredentialByUsername(ctx context.Context, username string) (store.UserCredential, error) {
	if r.getUserCredentialByUsername == nil {
		return store.UserCredential{}, store.ErrUserNotFound
	}

	return r.getUserCredentialByUsername(ctx, username)
}

func (r *pluginTestAuthRepository) SetPasswordHash(ctx context.Context, input store.SetPasswordHashInput) error {
	if r.setPasswordHash != nil {
		return r.setPasswordHash(ctx, input)
	}

	return nil
}

func (r *pluginTestAuthRepository) ResetPasswordAndRevokeRefreshSessions(
	ctx context.Context,
	input store.ResetPasswordAndRevokeSessionsInput,
) error {
	if r.resetPasswordAndRevoke != nil {
		return r.resetPasswordAndRevoke(ctx, input)
	}

	if err := r.SetPasswordHash(ctx, store.SetPasswordHashInput{
		UserID:             input.UserID,
		PasswordHash:       input.PasswordHash,
		MustChangePassword: input.MustChangePassword,
		ChangedAt:          &input.ChangedAt,
	}); err != nil {
		return err
	}

	return r.RevokeRefreshSessionsByUserID(ctx, store.RevokeRefreshSessionsByUserIDInput{
		UserID:    input.UserID,
		RevokedAt: input.ChangedAt,
	})
}

func (r *pluginTestAuthRepository) ChangePasswordAndRevokeOtherRefreshSessions(
	_ context.Context,
	input store.ChangePasswordAndRevokeOtherRefreshSessionsInput,
) error {
	if input.UserID == 0 {
		return store.ErrUserNotFound
	}

	return r.RevokeOtherRefreshSessionsByUserID(context.Background(), store.RevokeOtherRefreshSessionsInput{
		UserID:         input.UserID,
		CurrentTokenID: input.CurrentTokenID,
		RevokedAt:      input.ChangedAt,
	})
}

func (r *pluginTestAuthRepository) EnsureUserCredential(ctx context.Context, input store.EnsureUserCredentialInput) (store.UserCredential, error) {
	if r.ensureUserCredential != nil {
		return r.ensureUserCredential(ctx, input)
	}

	if r.getUserCredentialByUsername != nil {
		credential, err := r.getUserCredentialByUsername(ctx, input.Username)
		if err == nil {
			return credential, nil
		}
		if !errors.Is(err, store.ErrUserNotFound) {
			return store.UserCredential{}, err
		}
	}

	hash := input.PasswordHash
	return store.UserCredential{
		UserID:             1,
		Username:           input.Username,
		PasswordHash:       &hash,
		MustChangePassword: input.MustChangePassword,
	}, nil
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

func (r *pluginTestAuthRepository) RevokeOtherRefreshSessionsByUserID(_ context.Context, input store.RevokeOtherRefreshSessionsInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for tokenID, session := range r.refreshSessions {
		if session.UserID != input.UserID || session.RevokedAt != nil || tokenID == input.CurrentTokenID {
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
	list    func(ctx context.Context) ([]store.User, error)
	create  func(ctx context.Context, input store.CreateUserInput) (store.User, error)
	update  func(ctx context.Context, input store.UpdateUserInput) (store.User, error)
	status  func(ctx context.Context, input store.SetUserStatusInput) (store.User, error)
	delete  func(ctx context.Context, input store.DeleteUserInput) error
}

func (r pluginTestUserRepository) GetByID(ctx context.Context, id uint64) (store.User, error) {
	if r.getByID == nil {
		return store.User{}, store.ErrUserNotFound
	}

	return r.getByID(ctx, id)
}

func (r pluginTestUserRepository) List(ctx context.Context) ([]store.User, error) {
	if r.list == nil {
		return []store.User{}, nil
	}

	return r.list(ctx)
}

func (r pluginTestUserRepository) Create(ctx context.Context, input store.CreateUserInput) (store.User, error) {
	if r.create == nil {
		return store.User{
			ID:        99,
			Username:  input.Username,
			Display:   input.Display,
			Status:    input.Status,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}, nil
	}

	return r.create(ctx, input)
}

func (r pluginTestUserRepository) Update(ctx context.Context, input store.UpdateUserInput) (store.User, error) {
	if r.update == nil {
		return store.User{
			ID:        input.ID,
			Username:  input.Username,
			Display:   input.Display,
			Status:    usercontract.UserStatusEnabled,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}, nil
	}

	return r.update(ctx, input)
}

func (r pluginTestUserRepository) SetStatus(ctx context.Context, input store.SetUserStatusInput) (store.User, error) {
	if r.status == nil {
		return store.User{
			ID:        input.ID,
			Username:  "alice",
			Display:   "Alice",
			Status:    input.Status,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}, nil
	}

	return r.status(ctx, input)
}

func (r pluginTestUserRepository) Delete(ctx context.Context, input store.DeleteUserInput) error {
	if r.delete == nil {
		return nil
	}

	return r.delete(ctx, input)
}

type pluginTestRBACRepository struct {
	permissions             map[uint64][]rbacstore.Permission
	roles                   map[uint64][]rbacstore.Role
	ensureRole              func(ctx context.Context, input rbacstore.EnsureRoleInput) (rbacstore.Role, error)
	ensurePermission        func(ctx context.Context, input rbacstore.EnsurePermissionInput) (rbacstore.Permission, error)
	createRole              func(ctx context.Context, input rbacstore.CreateRoleInput) (rbacstore.Role, error)
	updateRole              func(ctx context.Context, input rbacstore.UpdateRoleInput) (rbacstore.Role, error)
	setRoleStatus           func(ctx context.Context, input rbacstore.SetRoleStatusInput) (rbacstore.Role, error)
	softDeleteRole          func(ctx context.Context, input rbacstore.SoftDeleteRoleInput) error
	assignPermissionsToRole func(ctx context.Context, input rbacstore.AssignPermissionsToRoleInput) error
	replacePermissions      func(ctx context.Context, input rbacstore.ReplacePermissionsForRoleInput) error
	addPermissionsToRole    func(ctx context.Context, input rbacstore.AddPermissionsToRoleInput) error
	removePermissions       func(ctx context.Context, input rbacstore.RemovePermissionsFromRoleInput) error
	assignRoleToUser        func(ctx context.Context, input rbacstore.AssignRoleToUserInput) error
	replaceRolesForUser     func(ctx context.Context, input rbacstore.ReplaceRolesForUserInput) error
	addRolesToUser          func(ctx context.Context, input rbacstore.AddRolesToUserInput) error
	removeRolesFromUser     func(ctx context.Context, input rbacstore.RemoveRolesFromUserInput) error
}

func (r pluginTestRBACRepository) EnsureRole(ctx context.Context, input rbacstore.EnsureRoleInput) (rbacstore.Role, error) {
	if r.ensureRole != nil {
		return r.ensureRole(ctx, input)
	}

	return rbacstore.Role{ID: 1, Name: input.Name, Display: input.Display}, nil
}

func (r pluginTestRBACRepository) EnsurePermission(ctx context.Context, input rbacstore.EnsurePermissionInput) (rbacstore.Permission, error) {
	if r.ensurePermission != nil {
		return r.ensurePermission(ctx, input)
	}

	return rbacstore.Permission{ID: 1, Code: input.Code, Display: input.Display}, nil
}

func (r pluginTestRBACRepository) CreateRole(ctx context.Context, input rbacstore.CreateRoleInput) (rbacstore.Role, error) {
	if r.createRole != nil {
		return r.createRole(ctx, input)
	}

	return rbacstore.Role{ID: 1, Name: input.Name, Display: input.Display, Description: input.Description, Builtin: input.Builtin}, nil
}

func (r pluginTestRBACRepository) UpdateRole(ctx context.Context, input rbacstore.UpdateRoleInput) (rbacstore.Role, error) {
	if r.updateRole != nil {
		return r.updateRole(ctx, input)
	}

	return rbacstore.Role{ID: input.ID, Name: input.Name, Display: input.Display, Description: input.Description}, nil
}

func (r pluginTestRBACRepository) SetRoleStatus(ctx context.Context, input rbacstore.SetRoleStatusInput) (rbacstore.Role, error) {
	if r.setRoleStatus != nil {
		return r.setRoleStatus(ctx, input)
	}

	return rbacstore.Role{ID: input.ID, Status: input.Status}, nil
}

func (r pluginTestRBACRepository) SoftDeleteRole(ctx context.Context, input rbacstore.SoftDeleteRoleInput) error {
	if r.softDeleteRole != nil {
		return r.softDeleteRole(ctx, input)
	}

	return nil
}

func (r pluginTestRBACRepository) AssignPermissionsToRole(ctx context.Context, input rbacstore.AssignPermissionsToRoleInput) error {
	if r.assignPermissionsToRole != nil {
		return r.assignPermissionsToRole(ctx, input)
	}

	return nil
}

func (r pluginTestRBACRepository) ReplacePermissionsForRole(ctx context.Context, input rbacstore.ReplacePermissionsForRoleInput) error {
	if r.replacePermissions != nil {
		return r.replacePermissions(ctx, input)
	}

	return nil
}

func (r pluginTestRBACRepository) AddPermissionsToRole(ctx context.Context, input rbacstore.AddPermissionsToRoleInput) error {
	if r.addPermissionsToRole != nil {
		return r.addPermissionsToRole(ctx, input)
	}

	return nil
}

func (r pluginTestRBACRepository) RemovePermissionsFromRole(ctx context.Context, input rbacstore.RemovePermissionsFromRoleInput) error {
	if r.removePermissions != nil {
		return r.removePermissions(ctx, input)
	}

	return nil
}

func (r pluginTestRBACRepository) AssignRoleToUser(ctx context.Context, input rbacstore.AssignRoleToUserInput) error {
	if r.assignRoleToUser != nil {
		return r.assignRoleToUser(ctx, input)
	}

	return nil
}

func (r pluginTestRBACRepository) ReplaceRolesForUser(ctx context.Context, input rbacstore.ReplaceRolesForUserInput) error {
	if r.replaceRolesForUser != nil {
		return r.replaceRolesForUser(ctx, input)
	}

	return nil
}

func (r pluginTestRBACRepository) AddRolesToUser(ctx context.Context, input rbacstore.AddRolesToUserInput) error {
	if r.addRolesToUser != nil {
		return r.addRolesToUser(ctx, input)
	}

	return nil
}

func (r pluginTestRBACRepository) RemoveRolesFromUser(ctx context.Context, input rbacstore.RemoveRolesFromUserInput) error {
	if r.removeRolesFromUser != nil {
		return r.removeRolesFromUser(ctx, input)
	}

	return nil
}

func (r pluginTestRBACRepository) GetRoleByID(_ context.Context, roleID uint64) (rbacstore.Role, error) {
	return rbacstore.Role{ID: roleID}, nil
}

func (r pluginTestRBACRepository) GetPermissionByID(_ context.Context, permissionID uint64) (rbacstore.Permission, error) {
	return rbacstore.Permission{ID: permissionID}, nil
}

func (r pluginTestRBACRepository) ListRolesByUserID(_ context.Context, userID uint64) ([]rbacstore.Role, error) {
	if r.roles == nil {
		return []rbacstore.Role{}, nil
	}

	return r.roles[userID], nil
}

func (r pluginTestRBACRepository) ListRolesByUserIDs(_ context.Context, userIDs []uint64) (map[uint64][]rbacstore.Role, error) {
	result := make(map[uint64][]rbacstore.Role, len(userIDs))
	for _, userID := range userIDs {
		if r.roles == nil {
			result[userID] = []rbacstore.Role{}
			continue
		}
		result[userID] = r.roles[userID]
	}
	return result, nil
}

func (r pluginTestRBACRepository) ListRoles(_ context.Context, _ rbacstore.RoleFilter) ([]rbacstore.Role, error) {
	return []rbacstore.Role{}, nil
}

func (r pluginTestRBACRepository) ListPermissionsByUserID(_ context.Context, userID uint64) ([]rbacstore.Permission, error) {
	if r.permissions == nil {
		return []rbacstore.Permission{}, nil
	}

	return r.permissions[userID], nil
}

func (r pluginTestRBACRepository) ListPermissions(_ context.Context, _ rbacstore.PermissionFilter) ([]rbacstore.Permission, error) {
	return []rbacstore.Permission{}, nil
}

func (r pluginTestRBACRepository) ListRolePermissionBindings(_ context.Context, _ uint64) ([]rbacstore.RolePermissionBinding, error) {
	return []rbacstore.RolePermissionBinding{}, nil
}

func newPluginTestContext(t *testing.T, userRepo store.UserRepository, authRepo store.AuthRepository) (*plugin.Context, *gin.Engine) {
	return newPluginTestContextWithLoggerAndPermissions(t, zap.NewNop(), userRepo, authRepo, map[uint64][]rbacstore.Permission{
		7: {{Code: usercontract.UserReadPermission.String()}},
	})
}

func newPluginTestContextWithPermissions(t *testing.T, userRepo store.UserRepository, authRepo store.AuthRepository, permissions map[uint64][]rbacstore.Permission) (*plugin.Context, *gin.Engine) {
	return newPluginTestContextWithLoggerAndPermissions(t, zap.NewNop(), userRepo, authRepo, permissions)
}

func newPluginTestContextWithLoggerAndPermissions(
	t *testing.T,
	logger *zap.Logger,
	userRepo store.UserRepository,
	authRepo store.AuthRepository,
	permissions map[uint64][]rbacstore.Permission,
) (*plugin.Context, *gin.Engine) {
	t.Helper()

	if authRepo == nil {
		authRepo = &pluginTestAuthRepository{}
	}
	if repo, ok := authRepo.(*pluginTestAuthRepository); ok && repo.getUserCredentialByUsername == nil {
		repo.getUserCredentialByUsername = func(_ context.Context, username string) (store.UserCredential, error) {
			userID := uint64(7)
			switch username {
			case "admin", "graft":
				userID = 9
			case "bob":
				userID = 8
			}
			return store.UserCredential{
				UserID:             userID,
				Username:           username,
				MustChangePassword: false,
			}, nil
		}
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctx := &plugin.Context{
		LifecycleContext: context.Background(),
		Logger:           logger,
		Config: &config.Config{Auth: config.AuthConfig{
			AccessTokenTTL:        15 * time.Minute,
			RefreshTokenTTL:       24 * time.Hour,
			SigningKey:            "test-signing-key",
			RefreshCookieName:     "graft_refresh_token",
			RefreshCookieSameSite: "lax",
			RefreshCookiePath:     "/",
		}, I18n: config.I18nConfig{
			DefaultLocale:    "zh-CN",
			FallbackLocale:   "zh-CN",
			SupportedLocales: []string{"zh-CN", "en-US"},
		}},
		I18n:               i18n.New(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}),
		Router:             engine.Group("/api"),
		Services:           container.New(),
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
	}

	pluginInstance := NewPlugin(userRepo, authRepo)
	if err := pluginInstance.Register(ctx); err != nil {
		t.Fatalf("register plugin: %v", err)
	}
	authPlugin := authplugin.NewPlugin()
	if err := authPlugin.Register(ctx); err != nil {
		t.Fatalf("register auth plugin: %v", err)
	}
	if err := rbac.NewPlugin(pluginTestRBACRepository{
		roles: map[uint64][]rbacstore.Role{
			7: {{ID: 1, Name: "admin", Display: "管理员"}},
			8: {{ID: 2, Name: "viewer", Display: "只读用户"}},
			9: {{ID: 1, Name: "admin", Display: "管理员"}},
		},
		permissions: permissions,
	}).Register(ctx); err != nil {
		t.Fatalf("register rbac plugin: %v", err)
	}
	if err := pluginInstance.Boot(ctx); err != nil {
		t.Fatalf("boot plugin: %v", err)
	}
	if err := authPlugin.Boot(ctx); err != nil {
		t.Fatalf("boot auth plugin: %v", err)
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
	setBearerAuthorizationHeader(request, token)
	return request
}

func newAuthorizedJSONRequestForSession(
	t *testing.T,
	method string,
	path string,
	userID uint64,
	sessionID string,
	body any,
) *http.Request {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal json request: %v", err)
		}
		reader = strings.NewReader(string(payload))
	}

	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
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
	setBearerAuthorizationHeader(request, token)
	return request
}

func replaceRouteTokens(path string, replacements ...string) string {
	if len(replacements) == 0 {
		return path
	}

	return strings.NewReplacer(replacements...).Replace(path)
}

func authRoutePath(fragment string, replacements ...string) string {
	return replaceRouteTokens(
		usercontract.JoinRoute(usercontract.JoinRoute(testAPIBasePath, usercontract.AuthGroup), fragment),
		replacements...,
	)
}

func usersRoutePath(fragment string, replacements ...string) string {
	return replaceRouteTokens(
		usercontract.JoinRoute(usercontract.JoinRoute(testAPIBasePath, usercontract.UsersGroup), fragment),
		replacements...,
	)
}

func setBearerAuthorizationHeader(request *http.Request, token string) {
	request.Header.Set(httpheadercontract.Authorization.String(), authcontract.Bearer.Prefix()+token)
}

func authorizationTokenFromRequest(request *http.Request) string {
	return strings.TrimSpace(strings.TrimPrefix(request.Header.Get(httpheadercontract.Authorization.String()), authcontract.Bearer.Prefix()))
}

func seedRefreshSession(t *testing.T, authRepo store.AuthRepository, userID uint64, expiresAt time.Time) string {
	t.Helper()

	if authRepo == nil {
		t.Fatal("auth repository is required to seed refresh session")
	}

	tokenID := fmt.Sprintf("session-%s", uuid.NewString())
	if _, err := authRepo.CreateRefreshSession(context.Background(), store.CreateRefreshSessionInput{
		UserID:    userID,
		TokenID:   tokenID,
		ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatalf("seed refresh session: %v", err)
	}

	return tokenID
}

func testUser(id uint64, username string, display string) store.User {
	now := time.Now()
	return store.User{
		ID:        id,
		Username:  username,
		Display:   display,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func fixedUserRepository(users ...store.User) pluginTestUserRepository {
	byID := make(map[uint64]store.User, len(users))
	for _, user := range users {
		byID[user.ID] = user
	}

	return pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			user, ok := byID[id]
			if !ok {
				return store.User{}, store.ErrUserNotFound
			}
			return user, nil
		},
	}
}

func newSessionAdminEngine(t *testing.T, authRepo *pluginTestAuthRepository, users ...store.User) *gin.Engine {
	t.Helper()

	_, engine := newPluginTestContextWithPermissions(t, fixedUserRepository(users...), authRepo, map[uint64][]rbacstore.Permission{
		9: {{Code: usercontract.UserSessionReadPermission.String()}, {Code: usercontract.UserSessionRevokePermission.String()}},
	})

	return engine
}

func newCredentialRepository(username string, userID uint64, passwordHash string) *pluginTestAuthRepository {
	return &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, candidate string) (store.UserCredential, error) {
			if candidate != username {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:       userID,
				Username:     username,
				PasswordHash: &passwordHash,
			}, nil
		},
	}
}

func assertStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()

	if recorder.Code != want {
		t.Fatalf("expected status %d, got %d", want, recorder.Code)
	}
}

func decodeErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder) httpx.ErrorResponse {
	t.Helper()

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return payload
}

func assertErrorPayload(t *testing.T, payload httpx.ErrorResponse, messageKey string, code string, locale string) {
	t.Helper()

	if payload.MessageKey != messageKey || payload.Code != code || payload.Locale != locale {
		t.Fatalf("expected error payload key=%s code=%s locale=%s, got %#v", messageKey, code, locale, payload)
	}
}

func assertContractErrorPayload(t *testing.T, payload httpx.ErrorResponse, messageKey messagecontract.Key, locale string) {
	t.Helper()

	assertErrorPayload(t, payload, messageKey.String(), errorcodecontract.FromMessageKey(messageKey).String(), locale)
}

func assertErrorFieldDetail(t *testing.T, payload httpx.ErrorResponse, field string) {
	t.Helper()

	if payload.Details["field"] != field {
		t.Fatalf("expected field detail %s, got %#v", field, payload)
	}
}

func assertSessionRevoked(t *testing.T, authRepo store.AuthRepository, tokenID string) {
	t.Helper()

	session, err := authRepo.GetRefreshSessionByTokenID(context.Background(), tokenID)
	if err != nil {
		t.Fatalf("load revoked session %q: %v", tokenID, err)
	}
	if session.RevokedAt == nil {
		t.Fatalf("expected session %q to be revoked, got %#v", tokenID, session)
	}
}

func assertSessionActive(t *testing.T, authRepo store.AuthRepository, tokenID string) {
	t.Helper()

	session, err := authRepo.GetRefreshSessionByTokenID(context.Background(), tokenID)
	if err != nil {
		t.Fatalf("load active session %q: %v", tokenID, err)
	}
	if session.RevokedAt != nil {
		t.Fatalf("expected session %q to remain active, got %#v", tokenID, session)
	}
}

func assertClearedCookie(t *testing.T, cookies []*http.Cookie, expectedName string) {
	t.Helper()

	if len(cookies) == 0 {
		t.Fatal("expected cleared refresh cookie")
	}
	if cookies[0].Name != expectedName || cookies[0].Value != "" || cookies[0].MaxAge >= 0 {
		t.Fatalf("expected cleared refresh cookie, got %#v", cookies[0])
	}
}

func loginUser(t *testing.T, engine *gin.Engine, username string, password string, locale string) (loginResponse, []*http.Cookie) {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthLogin), strings.NewReader(fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)))
	request.Header.Set("Content-Type", "application/json")
	if locale != "" {
		request.Header.Set(i18n.LocaleHeader, locale)
	}
	engine.ServeHTTP(recorder, request)
	assertStatus(t, recorder, http.StatusOK)

	return decodeSuccessData[loginResponse](t, recorder), recorder.Result().Cookies()
}

func parseRefreshCookieClaims(t *testing.T, cookie *http.Cookie) *refreshTokenSubject {
	t.Helper()

	manager, err := newRefreshTokenManager(config.AuthConfig{
		RefreshTokenTTL: 24 * time.Hour,
		SigningKey:      "test-signing-key",
	})
	if err != nil {
		t.Fatalf("new refresh token manager: %v", err)
	}

	claims, err := manager.Parse(cookie.Value)
	if err != nil {
		t.Fatalf("parse refresh cookie token: %v", err)
	}

	return claims
}

func assertSessionSummary(t *testing.T, item sessionSummary, sessionID string, current bool) {
	t.Helper()

	if item.SessionID != sessionID || item.Current != current {
		t.Fatalf("expected session %s current=%v, got %#v", sessionID, current, item)
	}
}

func assertSessionsAbsent(t *testing.T, payload []sessionSummary, sessionIDs ...string) {
	t.Helper()

	for _, item := range payload {
		for _, sessionID := range sessionIDs {
			if item.SessionID == sessionID {
				t.Fatalf("expected filtered sessions to be absent, got %#v", payload)
			}
		}
	}
}

func assertGeneratedSessionSummary(t *testing.T, item generated.SessionSummary, sessionID string, current bool) {
	t.Helper()

	if item.SessionId != sessionID || item.Current != current {
		t.Fatalf("expected generated session %s current=%v, got %#v", sessionID, current, item)
	}
}

func assertGeneratedSessionsAbsent(t *testing.T, payload []generated.SessionSummary, sessionIDs ...string) {
	t.Helper()

	for _, item := range payload {
		for _, sessionID := range sessionIDs {
			if item.SessionId == sessionID {
				t.Fatalf("expected filtered generated sessions to be absent, got %#v", payload)
			}
		}
	}
}

func firstCookie(t *testing.T, cookies []*http.Cookie) *http.Cookie {
	t.Helper()

	if len(cookies) == 0 {
		t.Fatal("expected refresh cookie to be present")
	}

	return cookies[0]
}

func assertRefreshCookieWritten(t *testing.T, cookie *http.Cookie, expectedName string) {
	t.Helper()

	if cookie.Name != expectedName || cookie.Value == "" {
		t.Fatalf("expected refresh cookie %q, got %#v", expectedName, cookie)
	}
}

func assertAccessClaims(t *testing.T, claims *pluginapi.AccessTokenClaims, userID uint64) {
	t.Helper()

	if claims.UserID != userID || claims.SessionID == "" {
		t.Fatalf("expected stable token claims, got %#v", claims)
	}
}

func assertUserSummary(t *testing.T, summary pluginapi.UserSummary, id uint64, username string, display string) {
	t.Helper()

	if summary.ID != id || summary.Username != username || summary.Display != display {
		t.Fatalf("expected stable user summary, got %#v", summary)
	}
}

func assertLoginPayload(t *testing.T, payload loginResponse, userID uint64, username string, displayName string) {
	t.Helper()

	if payload.AccessToken == "" {
		t.Fatal("expected access token in login response")
	}
	if payload.User.ID != userID || payload.User.Username != username || payload.User.DisplayName != displayName {
		t.Fatalf("expected current user summary, got %#v", payload.User)
	}
	if payload.ExpiresAt.IsZero() || payload.ExpiresAt.Before(time.Now().UTC()) {
		t.Fatalf("expected future access token expiry, got %#v", payload)
	}
}

func assertUserPluginRegistry(t *testing.T, ctx *plugin.Context) {
	t.Helper()

	items := ctx.PermissionRegistry.Items()
	if len(items) < 6 {
		t.Fatalf("expected at least six registered permissions, got %#v", items)
	}
	// 权限断言依赖 Registry.Items() 保持注册顺序，避免插件对外声明面静默漂移。
	if items[0].Code != usercontract.UserReadPermission.String() ||
		items[1].Code != usercontract.UserCreatePermission.String() ||
		items[2].Code != usercontract.UserUpdatePermission.String() ||
		items[3].Code != usercontract.UserDisablePermission.String() ||
		items[4].Code != usercontract.UserSessionRevokePermission.String() ||
		items[5].Code != usercontract.UserSessionReadPermission.String() {
		t.Fatalf("expected canonical leading user permissions, got %#v", items)
	}
	for _, item := range items[:6] {
		if item.Category != "api" {
			t.Fatalf("expected leading user permission %s to declare category api, got %#v", item.Code, item)
		}
	}

	menuItems := ctx.MenuRegistry.Items()
	if len(menuItems) == 0 {
		t.Fatalf("expected registered menu items, got %#v", menuItems)
	}
	if menuItems[0].Path != "/access-control/users" ||
		menuItems[0].Code != "user.list" ||
		menuItems[0].Permission != usercontract.UserReadPermission.String() {
		t.Fatalf("expected leading /users menu item to remain canonical, got %#v", menuItems)
	}
}

func assertDefaultAdminBootEffects(t *testing.T, ensuredDefaultAdmin bool, assignedRole bool) {
	t.Helper()

	if !ensuredDefaultAdmin {
		t.Fatal("expected boot to ensure default admin")
	}
	if !assignedRole {
		t.Fatal("expected boot to assign default admin role")
	}
}

func newExistingDefaultAdminAuthRepository(
	t *testing.T,
	defaultHash string,
	passwordChangedAt time.Time,
	ensuredDefaultAdmin *bool,
	updatedCredential *bool,
) *pluginTestAuthRepository {
	t.Helper()

	return &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username != defaultAdminUsername {
				t.Fatalf("expected default admin username, got %q", username)
			}

			return store.UserCredential{
				UserID:             9,
				Username:           username,
				PasswordHash:       &defaultHash,
				MustChangePassword: false,
				PasswordChangedAt:  &passwordChangedAt,
			}, nil
		},
		ensureUserCredential: func(context.Context, store.EnsureUserCredentialInput) (store.UserCredential, error) {
			*ensuredDefaultAdmin = true
			return store.UserCredential{}, nil
		},
		setPasswordHash: func(_ context.Context, input store.SetPasswordHashInput) error {
			*updatedCredential = true
			if input.UserID != 9 {
				t.Fatalf("expected default admin user id 9, got %d", input.UserID)
			}
			if input.PasswordHash != defaultHash {
				t.Fatal("expected default admin bootstrap reconciliation to preserve password hash")
			}
			if !input.MustChangePassword {
				t.Fatal("expected default admin bootstrap reconciliation to require password change")
			}
			if input.ChangedAt == nil || !input.ChangedAt.Equal(passwordChangedAt) {
				t.Fatalf("expected password changed timestamp %v, got %#v", passwordChangedAt, input.ChangedAt)
			}
			return nil
		},
	}
}

func newDefaultAdminBootRBACRepository(t *testing.T, assignedRole *bool) pluginTestRBACRepository {
	t.Helper()

	return pluginTestRBACRepository{
		ensureRole: func(_ context.Context, input rbacstore.EnsureRoleInput) (rbacstore.Role, error) {
			if !input.Builtin {
				t.Fatal("expected default admin role to be marked builtin")
			}
			return rbacstore.Role{ID: 1, Name: input.Name, Display: input.Display}, nil
		},
		ensurePermission: func(_ context.Context, input rbacstore.EnsurePermissionInput) (rbacstore.Permission, error) {
			if input.Category != "api" {
				t.Fatalf("expected ensured permission %s to carry category api, got %#v", input.Code, input)
			}
			return rbacstore.Permission{ID: 1, Code: input.Code, Display: input.Display}, nil
		},
		assignPermissionsToRole: func(_ context.Context, _ rbacstore.AssignPermissionsToRoleInput) error {
			return nil
		},
		assignRoleToUser: func(_ context.Context, input rbacstore.AssignRoleToUserInput) error {
			*assignedRole = true
			if input.UserID != 9 {
				t.Fatalf("expected default admin user id 9, got %d", input.UserID)
			}
			return nil
		},
	}
}

func newDefaultAdminBootAuthRepository(t *testing.T, ensuredDefaultAdmin *bool) *pluginTestAuthRepository {
	t.Helper()

	return &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, username string) (store.UserCredential, error) {
			if username == defaultAdminUsername {
				return store.UserCredential{}, store.ErrUserNotFound
			}

			return store.UserCredential{
				UserID:             7,
				Username:           username,
				MustChangePassword: false,
			}, nil
		},
		ensureUserCredential: func(_ context.Context, input store.EnsureUserCredentialInput) (store.UserCredential, error) {
			*ensuredDefaultAdmin = true
			if input.Username != defaultAdminUsername {
				t.Fatalf("expected default admin username, got %q", input.Username)
			}
			if !input.MustChangePassword {
				t.Fatal("expected default admin bootstrap to require password change")
			}
			if input.PasswordHash == "" {
				t.Fatal("expected default admin bootstrap password hash to be populated")
			}

			return store.UserCredential{
				UserID:             9,
				Username:           input.Username,
				MustChangePassword: input.MustChangePassword,
			}, nil
		},
	}
}

func newDefaultAdminBootPluginContext(_ store.AuthRepository, _ rbacstore.Repository) *plugin.Context {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	return &plugin.Context{
		LifecycleContext: context.Background(),
		Logger:           zap.NewNop(),
		Config: &config.Config{Auth: config.AuthConfig{
			AccessTokenTTL:        15 * time.Minute,
			RefreshTokenTTL:       24 * time.Hour,
			SigningKey:            "test-signing-key",
			RefreshCookieName:     "graft_refresh_token",
			RefreshCookieSameSite: "lax",
			RefreshCookiePath:     "/",
		}},
		I18n:               i18n.New(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}),
		Router:             engine.Group(testAPIBasePath),
		Services:           container.New(),
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
	}
}

func assertBootstrapPayload(t *testing.T, payload bootstrapResponse) {
	t.Helper()

	assertBootstrapIdentityAndPermissions(t, payload)
	assertBootstrapMenus(t, payload.Menus)
	assertBootstrapLocaleSnapshot(t, payload.Locale)
}

func assertBootstrapIdentityAndPermissions(t *testing.T, payload bootstrapResponse) {
	t.Helper()

	if payload.User.ID != 7 || payload.User.Username != "alice" || payload.User.DisplayName != "Alice" {
		t.Fatalf("expected current user summary, got %#v", payload.User)
	}
	if !slices.Equal(payload.Roles, []string{"admin"}) {
		t.Fatalf("expected sorted role snapshot, got %#v", payload.Roles)
	}
	if !slices.Equal(payload.Permissions, []string{usercontract.UserReadPermission.String()}) {
		t.Fatalf("expected sorted unique permissions, got %#v", payload.Permissions)
	}
}

func assertBootstrapMenus(t *testing.T, menus []bootstrapMenuResponse) {
	t.Helper()

	if len(menus) != 4 {
		t.Fatalf("expected filtered menus to keep access-control root, overview, user, and public entries, got %#v", menus)
	}

	expected := []bootstrapMenuResponse{
		{
			Code:     "access-control.root",
			Path:     "/access-control",
			Order:    0,
			TitleKey: "menu.access_control.title",
		},
		{
			Code:     "access-control.overview",
			Path:     "/access-control/overview",
			Order:    1,
			TitleKey: "menu.access_control.overview.title",
		},
		{
			Code:       "user.list",
			Path:       "/access-control/users",
			Order:      2,
			Permission: usercontract.UserReadPermission.String(),
			TitleKey:   "menu.access_control.users.title",
		},
		{
			Code:  "profile.self",
			Path:  "/profile",
			Order: 999,
		},
	}

	for i, want := range expected {
		got := menus[i]
		if got.Code != want.Code || got.Path != want.Path || got.Permission != want.Permission || got.TitleKey != want.TitleKey || got.Order != want.Order {
			t.Fatalf("expected bootstrap menu %d to be %#v, got %#v", i, want, got)
		}
	}
}

func assertBootstrapLocaleSnapshot(t *testing.T, locale bootstrapLocaleSnapshot) {
	t.Helper()

	if locale.CurrentLocale != "en-US" || locale.DefaultLocale != "zh-CN" || locale.FallbackLocale != "zh-CN" {
		t.Fatalf("expected locale snapshot en-US/zh-CN/zh-CN, got %#v", locale)
	}
	if !slices.Equal(locale.SupportedLocales, []string{"zh-CN", "en-US"}) {
		t.Fatalf("expected supported locales snapshot, got %#v", locale)
	}
}

func assertNilSuccessPayload(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	if payload := decodeSuccessData[any](t, recorder); payload != nil {
		t.Fatalf("expected success payload to be nil, got %#v", payload)
	}
}

func assertRefreshRotationPayload(t *testing.T, payload loginResponse) {
	t.Helper()

	if payload.AccessToken == "" || payload.ExpiresAt.IsZero() {
		t.Fatalf("expected rotated access token payload, got %#v", payload)
	}
}

func assertRotatedCookie(t *testing.T, oldCookie *http.Cookie, newCookies []*http.Cookie) {
	t.Helper()

	newCookie := firstCookie(t, newCookies)
	if newCookie.Value == oldCookie.Value {
		t.Fatalf("expected rotated refresh cookie, got old=%#v new=%#v", oldCookie, newCookie)
	}
}

func assertInvalidTokenResponse(t *testing.T, recorder *httptest.ResponseRecorder, locale string) {
	t.Helper()

	assertStatus(t, recorder, http.StatusUnauthorized)
	assertContractErrorPayload(t, decodeErrorResponse(t, recorder), messagecontract.AuthTokenInvalid, locale)
}

func assertMissingTokenResponse(t *testing.T, recorder *httptest.ResponseRecorder, locale string) {
	t.Helper()

	assertStatus(t, recorder, http.StatusUnauthorized)
	assertContractErrorPayload(t, decodeErrorResponse(t, recorder), messagecontract.AuthTokenMissing, locale)
}

func assertSessionNotFoundResponse(t *testing.T, recorder *httptest.ResponseRecorder, locale string) {
	t.Helper()

	assertStatus(t, recorder, http.StatusNotFound)
	assertContractErrorPayload(t, decodeErrorResponse(t, recorder), messagecontract.AuthSessionNotFound, locale)
}

func assertInvalidArgumentFieldResponse(t *testing.T, recorder *httptest.ResponseRecorder, locale string, field string) {
	t.Helper()

	assertStatus(t, recorder, http.StatusBadRequest)
	payload := decodeErrorResponse(t, recorder)
	assertContractErrorPayload(t, payload, messagecontract.CommonInvalidArgument, locale)
	assertErrorFieldDetail(t, payload, field)
}

func assertForbiddenResponse(t *testing.T, recorder *httptest.ResponseRecorder, locale string) {
	t.Helper()

	assertStatus(t, recorder, http.StatusForbidden)
	assertContractErrorPayload(t, decodeErrorResponse(t, recorder), messagecontract.AuthForbidden, locale)
}

func assertNoCookieMutation(t *testing.T, cookies []*http.Cookie) {
	t.Helper()

	if len(cookies) != 0 {
		t.Fatalf("expected no cookie mutation, got %#v", cookies)
	}
}

func assertGeneratedActiveSessions(t *testing.T, payload []generated.SessionSummary, expected ...generated.SessionSummary) {
	t.Helper()

	if len(payload) != len(expected) {
		t.Fatalf("expected %d generated active sessions, got %#v", len(expected), payload)
	}

	for i, item := range expected {
		assertGeneratedSessionSummary(t, payload[i], item.SessionId, item.Current)
	}
}

func assertGeneratedSessionsFiltered(t *testing.T, payload []generated.SessionSummary, sessionIDs ...string) {
	t.Helper()
	assertGeneratedSessionsAbsent(t, payload, sessionIDs...)
}

func loginAliceEngine(t *testing.T, passwordHash string) (*pluginTestAuthRepository, *gin.Engine) {
	t.Helper()

	authRepo := newCredentialRepository("alice", 7, passwordHash)
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice"), testUser(8, "bob", "Bob")), authRepo)
	return authRepo, engine
}

func loginAdminEngine(t *testing.T, passwordHash string) (*pluginTestAuthRepository, *gin.Engine) {
	t.Helper()

	authRepo := newCredentialRepository("admin", 9, passwordHash)
	_, engine := newPluginTestContextWithPermissions(t, fixedUserRepository(
		testUser(7, "alice", "Alice"),
		testUser(8, "bob", "Bob"),
		testUser(9, "admin", "Admin"),
	), authRepo, map[uint64][]rbacstore.Permission{
		9: {{Code: usercontract.UserSessionRevokePermission.String()}},
	})
	return authRepo, engine
}

func loginAliceAndParseSession(t *testing.T, engine *gin.Engine) (loginResponse, *http.Cookie, *refreshTokenSubject) {
	t.Helper()

	loginPayload, loginCookies := loginUser(t, engine, "alice", "secret", "")
	refreshCookie := firstCookie(t, loginCookies)

	return loginPayload, refreshCookie, parseRefreshCookieClaims(t, refreshCookie)
}

// TestRegisterPublishesContracts 验证用户插件注册时会暴露权限、菜单与稳定
// 的跨插件用户服务。
func TestRegisterPublishesContracts(t *testing.T) {
	ctx, _ := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice")), nil)

	assertUserPluginRegistry(t, ctx)

	svcAny, err := ctx.Services.Resolve((*pluginapi.UserService)(nil))
	if err != nil {
		t.Fatalf("resolve user service: %v", err)
	}

	summary, err := svcAny.(pluginapi.UserService).GetUserByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("get user by id: %v", err)
	}
	assertUserSummary(t, summary, 7, "alice", "Alice")
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
	engine.ServeHTTP(recorder, newAuthorizedRequestForUser(t, usersRoutePath(usercontract.UserByID, ":id", "not-a-number"), authRepo, 7))

	assertStatus(t, recorder, http.StatusBadRequest)

	payload := decodeErrorResponse(t, recorder)
	assertContractErrorPayload(t, payload, messagecontract.CommonInvalidArgument, "zh-CN")
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
	request := newAuthorizedRequestForUser(t, usersRoutePath(usercontract.UserByID, ":id", "8"), authRepo, 7)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assertContractErrorPayload(t, payload, messagecontract.UserNotFound, "en-US")
	if payload.Message != "User not found" || payload.Error != payload.Message {
		t.Fatalf("expected not found payload, got %#v", payload)
	}
}

// TestUserRouteReturnsSummary 验证用户查询成功时会返回跨插件稳定 DTO，而不
// 直接把生成后的 HTTP 契约模型作为边界输出，而不是复用跨插件摘要 DTO。
func TestUserRouteReturnsGeneratedUserListItem(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(context.Context, uint64) (store.User, error) {
			return store.User{
				ID:        7,
				Username:  "alice",
				Display:   "Alice",
				Status:    usercontract.UserStatusEnabled,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}, authRepo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedRequestForUser(t, usersRoutePath(usercontract.UserByID, ":id", "7"), authRepo, 7))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	payload := decodeSuccessData[userListItem](t, recorder)
	if payload.Id != 7 || payload.Username != "alice" || payload.Display != "Alice" {
		t.Fatalf("expected generated user detail payload, got %#v", payload)
	}
	if payload.Status != usercontract.UserStatusEnabled {
		t.Fatalf("expected generated user status, got %#v", payload)
	}
}

// TestBootEnsuresDefaultAdmin 验证默认管理员初始化只在 Boot 阶段执行，
// 避免 Register 阶段引入持久化副作用。
func TestBootEnsuresDefaultAdmin(t *testing.T) {
	ensuredDefaultAdmin := false
	assignedRole := false
	authRepo := newDefaultAdminBootAuthRepository(t, &ensuredDefaultAdmin)
	rbacRepo := newDefaultAdminBootRBACRepository(t, &assignedRole)

	ctx := newDefaultAdminBootPluginContext(authRepo, rbacRepo)

	pluginInstance := NewPlugin(
		pluginTestUserRepository{},
		authRepo,
	)
	if err := pluginInstance.Register(ctx); err != nil {
		t.Fatalf("register plugin: %v", err)
	}
	if ensuredDefaultAdmin {
		t.Fatal("expected register to stay side-effect free for default admin bootstrap")
	}
	if err := rbac.NewPlugin(rbacRepo).Register(ctx); err != nil {
		t.Fatalf("register rbac plugin: %v", err)
	}

	if err := pluginInstance.Boot(ctx); err != nil {
		t.Fatalf("boot plugin: %v", err)
	}

	assertDefaultAdminBootEffects(t, ensuredDefaultAdmin, assignedRole)
}

// TestBootMarksExistingDefaultAdminForPasswordChange 验证升级后仍使用初始化密码的默认管理员
// 会在 Boot 阶段被精确标记为强制改密，而不覆盖已存储的密码散列或最近改密时间。
func TestBootMarksExistingDefaultAdminForPasswordChange(t *testing.T) {
	defaultHashBytes, err := bcrypt.GenerateFromPassword([]byte(defaultAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate default admin password hash: %v", err)
	}
	defaultHash := string(defaultHashBytes)
	passwordChangedAt := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)

	var ensuredDefaultAdmin bool
	var updatedCredential bool
	authRepo := newExistingDefaultAdminAuthRepository(
		t,
		defaultHash,
		passwordChangedAt,
		&ensuredDefaultAdmin,
		&updatedCredential,
	)

	var assignedRole bool
	rbacRepo := newDefaultAdminBootRBACRepository(t, &assignedRole)

	ctx := newDefaultAdminBootPluginContext(authRepo, rbacRepo)
	pluginInstance := NewPlugin(
		pluginTestUserRepository{},
		authRepo,
	)
	if err := pluginInstance.Register(ctx); err != nil {
		t.Fatalf("register plugin: %v", err)
	}
	if err := rbac.NewPlugin(rbacRepo).Register(ctx); err != nil {
		t.Fatalf("register rbac plugin: %v", err)
	}
	if err := pluginInstance.Boot(ctx); err != nil {
		t.Fatalf("boot plugin: %v", err)
	}

	if ensuredDefaultAdmin {
		t.Fatal("expected existing default admin bootstrap not to recreate the credential")
	}
	if !updatedCredential {
		t.Fatal("expected boot to mark existing default admin for password change")
	}
	if !assignedRole {
		t.Fatal("expected boot to assign default admin role")
	}
}

// TestBootFailsWithoutSharedRouteAuthorizer 验证 Boot 会在共享 Authorizer 未注册时
// fail closed，而不是继续让用户路由带着未绑定的授权器启动。
func TestBootFailsWithoutSharedRouteAuthorizer(t *testing.T) {
	ctx := newDefaultAdminBootPluginContext(&pluginTestAuthRepository{}, pluginTestRBACRepository{})
	pluginInstance := NewPlugin(
		pluginTestUserRepository{},
		&pluginTestAuthRepository{},
	)
	if err := pluginInstance.Register(ctx); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	err := pluginInstance.Boot(ctx)
	if err == nil {
		t.Fatal("expected boot to fail without shared authorizer")
	}
	if !strings.Contains(err.Error(), "resolve route authorizer") {
		t.Fatalf("expected route authorizer resolution failure, got %v", err)
	}
}

// TestUserListRouteReturnsStableItems 验证用户列表路由会返回真实后端最小列表
// DTO，供 web `/users` 页面摆脱 demo 数据源。
func TestUserListRouteReturnsStableItems(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	createdAt := time.Date(2026, time.May, 15, 8, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(context.Context, uint64) (store.User, error) {
			return store.User{
				ID:        7,
				Username:  "alice",
				Display:   "Alice",
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}, nil
		},
		list: func(context.Context) ([]store.User, error) {
			return []store.User{
				{
					ID:        7,
					Username:  "alice",
					Display:   "Alice",
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				{
					ID:        8,
					Username:  "bob",
					Display:   "Bob",
					CreatedAt: createdAt.Add(time.Hour),
					UpdatedAt: updatedAt.Add(time.Hour),
				},
			}, nil
		},
	}, authRepo)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedRequestForUser(t, usersRoutePath(usercontract.UserCollection), authRepo, 7))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	payload := decodeSuccessData[userListResponse](t, recorder)
	if len(payload.Items) != 2 {
		t.Fatalf("expected two list items, got %#v", payload.Items)
	}
	if payload.Items[0].Id != 7 || payload.Items[0].Username != "alice" || payload.Items[0].Display != "Alice" {
		t.Fatalf("expected first stable user list item, got %#v", payload.Items[0])
	}
	if payload.Items[0].CreatedAt != createdAt.Format(time.RFC3339) || payload.Items[0].UpdatedAt != updatedAt.Format(time.RFC3339) {
		t.Fatalf("expected RFC3339 timestamps, got %#v", payload.Items[0])
	}
}

// TestUserListRouteReturnsInternalErrorContract 验证用户列表仓储失败时仍返回统一本地化错误契约。
func TestUserListRouteReturnsInternalErrorContract(t *testing.T) {
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
		list: func(context.Context) ([]store.User, error) {
			return nil, errors.New("boom")
		},
	}, authRepo)

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForUser(t, usersRoutePath(usercontract.UserCollection), authRepo, 7)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != messagecontract.CommonInternalError.String() || payload.Locale != "en-US" {
		t.Fatalf("expected localized internal error payload, got %#v", payload)
	}
}

func TestCreateUserRouteReturnsStableItem(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	createdAt := time.Date(2026, time.May, 22, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(5 * time.Minute)
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return testUser(7, "alice", "Alice"), nil
		},
		create: func(_ context.Context, input store.CreateUserInput) (store.User, error) {
			if input.Username != "carol" || input.Display != "Carol" {
				t.Fatalf("unexpected create input: %#v", input)
			}
			if input.Status != usercontract.UserStatusEnabled || !input.MustChangePassword {
				t.Fatalf("expected enabled user with required password change, got %#v", input)
			}
			if input.PasswordHash == "Password12345" {
				t.Fatalf("expected hashed password, got %#v", input)
			}

			return store.User{
				ID:        11,
				Username:  input.Username,
				Display:   input.Display,
				Status:    input.Status,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}, nil
		},
	}, authRepo, map[uint64][]rbacstore.Permission{
		7: {
			{Code: usercontract.UserReadPermission.String()},
			{Code: usercontract.UserCreatePermission.String()},
		},
	})

	sessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	request := newAuthorizedJSONRequestForSession(t, http.MethodPost, usersRoutePath(usercontract.UserCollection), 7, sessionID, map[string]any{
		"username": "carol",
		"display":  "Carol",
		"password": "Password12345",
	})
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	payload := decodeSuccessData[userListItem](t, recorder)
	if payload.Id != 11 || payload.Username != "carol" || payload.Display != "Carol" || payload.Status != usercontract.UserStatusEnabled {
		t.Fatalf("unexpected created payload: %#v", payload)
	}
}

func TestCreateUserMapperBuildsBusinessCommand(t *testing.T) {
	command := toCreateUserCommand(useropenapi.PostUsersJSONRequestBody{
		Username: "carol",
		Display:  "Carol",
		Password: "Password12345",
	}, 7)

	if command.Username != "carol" || command.Display != "Carol" || command.Password != "Password12345" || command.ActorID != 7 {
		t.Fatalf("unexpected create user command: %#v", command)
	}
}

func TestUpdateUserMapperBuildsBusinessCommand(t *testing.T) {
	command := toUpdateUserCommand(useropenapi.PostUserUpdateJSONRequestBody{
		Username: "carol.ops",
		Display:  "Carol Ops",
	}, 11, 7)

	if command.ID != 11 || command.Username != "carol.ops" || command.Display != "Carol Ops" || command.ActorID != 7 {
		t.Fatalf("unexpected update user command: %#v", command)
	}
}

func TestUpdateUserStatusMapperBuildsBusinessCommand(t *testing.T) {
	command, ok := toUpdateUserStatusCommand(useropenapi.PostUserStatusJSONRequestBody{
		Status: useropenapi.Disabled,
	}, 11, 7)
	if !ok {
		t.Fatal("expected status mapper to accept generated enum")
	}

	if command.ID != 11 || command.Status != usercontract.UserStatusDisabled || command.ActorID != 7 {
		t.Fatalf("unexpected update user status command: %#v", command)
	}
}

func TestUserServiceCreateUserDoesNotImportOpenAPIContract(t *testing.T) {
	content, err := os.ReadFile("plugin.go")
	if err != nil {
		t.Fatalf("read plugin.go: %v", err)
	}

	if strings.Contains(string(content), "internal/contract/openapi") {
		t.Fatalf("service implementation must not import openapi contract package")
	}
}

func TestCreateUserRouteReturnsPasswordPolicyViolationContract(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return testUser(7, "alice", "Alice"), nil
		},
	}, authRepo, map[uint64][]rbacstore.Permission{
		7: {
			{Code: usercontract.UserReadPermission.String()},
			{Code: usercontract.UserCreatePermission.String()},
		},
	})

	sessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	request := newAuthorizedJSONRequestForSession(t, http.MethodPost, usersRoutePath(usercontract.UserCollection), 7, sessionID, map[string]any{
		"username": "carol",
		"display":  "Carol",
		"password": "short",
	})
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusBadRequest)
	payload := decodeErrorResponse(t, recorder)
	assertContractErrorPayload(t, payload, messagecontract.AuthPasswordPolicyViolation, "zh-CN")
	assertErrorFieldDetail(t, payload, "password")
	if payload.Code != errorcodecontract.AuthPasswordPolicyViolation.String() {
		t.Fatalf("expected password policy code, got %#v", payload)
	}
}

func TestCreateUserRouteReturnsFieldLevelInvalidArgumentContract(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return testUser(7, "alice", "Alice"), nil
		},
	}, authRepo, map[uint64][]rbacstore.Permission{
		7: {
			{Code: usercontract.UserReadPermission.String()},
			{Code: usercontract.UserCreatePermission.String()},
		},
	})

	tests := []struct {
		name  string
		body  map[string]any
		field string
	}{
		{
			name: "missing username",
			body: map[string]any{
				"display":  "Carol",
				"password": "Password12345",
			},
			field: "username",
		},
		{
			name: "missing display",
			body: map[string]any{
				"username": "carol",
				"password": "Password12345",
			},
			field: "display",
		},
		{
			name: "missing password",
			body: map[string]any{
				"username": "carol",
				"display":  "Carol",
			},
			field: "password",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
			request := newAuthorizedJSONRequestForSession(t, http.MethodPost, usersRoutePath(usercontract.UserCollection), 7, sessionID, tc.body)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)

			assertStatus(t, recorder, http.StatusBadRequest)
			payload := decodeErrorResponse(t, recorder)
			assertContractErrorPayload(t, payload, messagecontract.CommonInvalidArgument, "zh-CN")
			assertErrorFieldDetail(t, payload, tc.field)
		})
	}
}

func TestCreateUserRouteLogsPasswordPolicyViolationWithoutPasswordLeak(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	core, logs := observer.New(zap.ErrorLevel)
	logger := zap.New(core)
	_, engine := newPluginTestContextWithLoggerAndPermissions(t, logger, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return testUser(7, "alice", "Alice"), nil
		},
	}, authRepo, map[uint64][]rbacstore.Permission{
		7: {
			{Code: usercontract.UserReadPermission.String()},
			{Code: usercontract.UserCreatePermission.String()},
		},
	})

	sessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	password := "short"
	request := newAuthorizedJSONRequestForSession(t, http.MethodPost, usersRoutePath(usercontract.UserCollection), 7, sessionID, map[string]any{
		"username": "carol",
		"display":  "Carol",
		"password": password,
	})
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusBadRequest)
	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected one error log, got %d", len(entries))
	}
	entry := entries[0]
	if entry.Message != "create user failed" {
		t.Fatalf("unexpected log message: %#v", entry)
	}
	fields := entry.ContextMap()
	if fields["operation"] != "create_user" || fields["route"] != "/api/users" || fields["method"] != http.MethodPost {
		t.Fatalf("expected stable route log fields, got %#v", fields)
	}
	if fields["response_code"] != errorcodecontract.AuthPasswordPolicyViolation.String() || fields["message_key"] != messagecontract.AuthPasswordPolicyViolation.String() {
		t.Fatalf("expected stable error metadata, got %#v", fields)
	}
	if fields["field"] != "password" {
		t.Fatalf("expected password field detail, got %#v", fields)
	}
	if rendered := fmt.Sprint(fields); strings.Contains(rendered, password) {
		t.Fatalf("expected password to stay out of logs, got %#v", fields)
	}
}

func TestUpdateUserRouteReturnsStableItem(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			if id != 7 {
				return store.User{}, store.ErrUserNotFound
			}
			return testUser(7, "alice", "Alice"), nil
		},
		update: func(_ context.Context, input store.UpdateUserInput) (store.User, error) {
			if input.ID != 7 || input.Username != "alice.ops" || input.Display != "Alice Ops" {
				t.Fatalf("unexpected update input: %#v", input)
			}
			return store.User{
				ID:        input.ID,
				Username:  input.Username,
				Display:   input.Display,
				Status:    usercontract.UserStatusEnabled,
				CreatedAt: time.Date(2026, time.May, 20, 9, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, time.May, 22, 9, 30, 0, 0, time.UTC),
			}, nil
		},
	}, authRepo, map[uint64][]rbacstore.Permission{
		7: {
			{Code: usercontract.UserReadPermission.String()},
			{Code: usercontract.UserUpdatePermission.String()},
		},
	})

	sessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	request := newAuthorizedJSONRequestForSession(t, http.MethodPost, usersRoutePath(usercontract.UserUpdateRoute, ":id", "7"), 7, sessionID, map[string]any{
		"username": "alice.ops",
		"display":  "Alice Ops",
	})
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	payload := decodeSuccessData[userListItem](t, recorder)
	if payload.Username != "alice.ops" || payload.Display != "Alice Ops" {
		t.Fatalf("unexpected updated payload: %#v", payload)
	}
}

func TestSetUserStatusRouteRevokesTargetSessions(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 8:
				return testUser(8, "bob", "Bob"), nil
			case 9:
				return testUser(9, "admin", "Admin"), nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
		status: func(_ context.Context, input store.SetUserStatusInput) (store.User, error) {
			return store.User{
				ID:        input.ID,
				Username:  "bob",
				Display:   "Bob",
				Status:    input.Status,
				CreatedAt: time.Now().UTC().Add(-time.Hour),
				UpdatedAt: time.Now().UTC(),
			}, nil
		},
	}, authRepo, map[uint64][]rbacstore.Permission{
		9: {
			{Code: usercontract.UserReadPermission.String()},
			{Code: usercontract.UserDisablePermission.String()},
		},
	})

	adminSession := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))
	targetSession := seedRefreshSession(t, authRepo, 8, time.Now().UTC().Add(time.Hour))
	request := newAuthorizedJSONRequestForSession(
		t,
		http.MethodPost,
		usersRoutePath(usercontract.UserStatusRoute, ":id", "8"),
		9,
		adminSession,
		map[string]any{"status": usercontract.UserStatusDisabled},
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	payload := decodeSuccessData[userListItem](t, recorder)
	if payload.Status != usercontract.UserStatusDisabled {
		t.Fatalf("expected disabled payload, got %#v", payload)
	}
	assertSessionRevoked(t, authRepo, targetSession)
	assertSessionActive(t, authRepo, adminSession)
}

func TestSetUserStatusRouteRejectsUnknownGeneratedEnumValue(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 8:
				return testUser(8, "bob", "Bob"), nil
			case 9:
				return testUser(9, "admin", "Admin"), nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
		status: func(context.Context, store.SetUserStatusInput) (store.User, error) {
			t.Fatal("expected repository status update not to be called")
			return store.User{}, nil
		},
	}, authRepo, map[uint64][]rbacstore.Permission{
		9: {
			{Code: usercontract.UserReadPermission.String()},
			{Code: usercontract.UserDisablePermission.String()},
		},
	})

	adminSession := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))
	request := newAuthorizedJSONRequestForSession(
		t,
		http.MethodPost,
		usersRoutePath(usercontract.UserStatusRoute, ":id", "8"),
		9,
		adminSession,
		map[string]any{"status": "paused"},
	)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	assertInvalidArgumentFieldResponse(t, recorder, "en-US", "status")
}

func TestResetUserPasswordRouteUsesAtomicResetContract(t *testing.T) {
	authRepo := &pluginTestAuthRepository{
		resetPasswordAndRevoke: func(_ context.Context, input store.ResetPasswordAndRevokeSessionsInput) error {
			if input.UserID != 8 || !input.MustChangePassword {
				t.Fatalf("unexpected reset input: %#v", input)
			}
			if input.PasswordHash == "Password12345" {
				t.Fatalf("expected hashed password, got %#v", input)
			}
			return nil
		},
	}
	_, engine := newPluginTestContextWithPermissions(t, fixedUserRepository(testUser(8, "bob", "Bob"), testUser(9, "admin", "Admin")), authRepo, map[uint64][]rbacstore.Permission{
		9: {
			{Code: usercontract.UserReadPermission.String()},
			{Code: usercontract.UserUpdatePermission.String()},
		},
	})

	adminSession := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))
	request := newAuthorizedJSONRequestForSession(
		t,
		http.MethodPost,
		usersRoutePath(usercontract.UserResetPasswordRoute, ":id", "8"),
		9,
		adminSession,
		map[string]any{"new_password": "Password12345"},
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	assertNilSuccessPayload(t, recorder)
}

func TestResetUserPasswordRouteReturnsPasswordPolicyViolationContract(t *testing.T) {
	authRepo := &pluginTestAuthRepository{
		resetPasswordAndRevoke: func(context.Context, store.ResetPasswordAndRevokeSessionsInput) error {
			t.Fatal("expected reset password repository operation not to be called")
			return nil
		},
	}
	_, engine := newPluginTestContextWithPermissions(t, fixedUserRepository(testUser(8, "bob", "Bob"), testUser(9, "admin", "Admin")), authRepo, map[uint64][]rbacstore.Permission{
		9: {
			{Code: usercontract.UserReadPermission.String()},
			{Code: usercontract.UserUpdatePermission.String()},
		},
	})

	adminSession := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))
	request := newAuthorizedJSONRequestForSession(
		t,
		http.MethodPost,
		usersRoutePath(usercontract.UserResetPasswordRoute, ":id", "8"),
		9,
		adminSession,
		map[string]any{"new_password": "short"},
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusBadRequest)
	payload := decodeErrorResponse(t, recorder)
	assertContractErrorPayload(t, payload, messagecontract.AuthPasswordPolicyViolation, "zh-CN")
	assertErrorFieldDetail(t, payload, "new_password")
	if payload.Code != errorcodecontract.AuthPasswordPolicyViolation.String() {
		t.Fatalf("expected password policy code, got %#v", payload)
	}
}

func TestResetUserPasswordRouteReturnsPasswordReuseForbiddenContract(t *testing.T) {
	authRepo := &pluginTestAuthRepository{
		resetPasswordAndRevoke: func(context.Context, store.ResetPasswordAndRevokeSessionsInput) error {
			t.Fatal("expected reset password repository operation not to be called")
			return nil
		},
	}
	_, engine := newPluginTestContextWithPermissions(t, fixedUserRepository(testUser(8, "bob", "Bob"), testUser(9, "admin", "Admin")), authRepo, map[uint64][]rbacstore.Permission{
		9: {
			{Code: usercontract.UserReadPermission.String()},
			{Code: usercontract.UserUpdatePermission.String()},
		},
	})

	adminSession := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))
	request := newAuthorizedJSONRequestForSession(
		t,
		http.MethodPost,
		usersRoutePath(usercontract.UserResetPasswordRoute, ":id", "8"),
		9,
		adminSession,
		map[string]any{"new_password": defaultAdminPassword},
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusBadRequest)
	payload := decodeErrorResponse(t, recorder)
	assertContractErrorPayload(t, payload, messagecontract.AuthPasswordReuseForbidden, "zh-CN")
	assertErrorFieldDetail(t, payload, "new_password")
	if payload.Code != errorcodecontract.AuthPasswordReuseForbidden.String() {
		t.Fatalf("expected password reuse code, got %#v", payload)
	}
}

func TestDeleteUserRouteRevokesSessionsAndReturnsNilPayload(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, pluginTestUserRepository{
		getByID: func(_ context.Context, id uint64) (store.User, error) {
			switch id {
			case 8:
				return testUser(8, "bob", "Bob"), nil
			case 9:
				return testUser(9, "admin", "Admin"), nil
			default:
				return store.User{}, store.ErrUserNotFound
			}
		},
		delete: func(_ context.Context, input store.DeleteUserInput) error {
			if input.ID != 8 || input.ActorID != 9 {
				t.Fatalf("unexpected delete input: %#v", input)
			}
			return nil
		},
	}, authRepo, map[uint64][]rbacstore.Permission{
		9: {
			{Code: usercontract.UserReadPermission.String()},
			{Code: usercontract.UserDisablePermission.String()},
		},
	})

	adminSession := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))
	targetSession := seedRefreshSession(t, authRepo, 8, time.Now().UTC().Add(time.Hour))
	request := newAuthorizedJSONRequestForSession(
		t,
		http.MethodPost,
		usersRoutePath(usercontract.UserDeleteRoute, ":id", "8"),
		9,
		adminSession,
		nil,
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	assertNilSuccessPayload(t, recorder)
	assertSessionRevoked(t, authRepo, targetSession)
}

// TestUserRouteRequiresPermissionMiddleware 验证插件路由仍复用统一的后端
// 权限守卫契约，而不是在插件内部发散独立鉴权格式。
func TestUserRouteRequiresPermissionMiddleware(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, usersRoutePath(usercontract.UserByID, ":id", "7"), nil)
	engine.ServeHTTP(recorder, request)
	assertMissingTokenResponse(t, recorder, "zh-CN")
}

// TestBootstrapRouteRequiresAuthenticatedActor 验证 bootstrap 契约仍复用统一
// 的请求鉴权中间件，而不是在插件内分叉另一套登录态判断。
func TestBootstrapRouteRequiresAuthenticatedActor(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, nil)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, authRoutePath(usercontract.AuthBootstrap), nil)
	engine.ServeHTTP(recorder, request)
	assertMissingTokenResponse(t, recorder, "zh-CN")
}

// TestBootstrapRouteReturnsFilteredContract 验证 bootstrap 路由会返回当前用户、
// 去重排序后的权限列表、按权限过滤的菜单以及 locale 配置快照。
func TestBootstrapRouteReturnsFilteredContract(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	ctx, engine := newPluginTestContextWithPermissions(t, fixedUserRepository(testUser(7, "alice", "Alice")), authRepo, map[uint64][]rbacstore.Permission{
		7: {
			{Code: " " + usercontract.UserReadPermission.String() + " "},
			{Code: usercontract.UserReadPermission.String()},
			{Code: ""},
		},
	})
	ctx.MenuRegistry.Register(menu.Item{
		Code:  "profile.self",
		Title: "个人中心",
		Path:  "/profile",
		Icon:  "user-circle",
		Order: 999,
	})
	ctx.MenuRegistry.Register(menu.Item{
		Code:       "audit.list",
		Title:      "审计日志",
		Path:       "/audit",
		Icon:       "secured",
		Order:      200,
		Permission: "audit.read",
	})

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForUser(t, authRoutePath(usercontract.AuthBootstrap), authRepo, 7)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)

	payload := decodeSuccessData[bootstrapResponse](t, recorder)
	assertBootstrapPayload(t, payload)
	if len(payload.Menus) != 4 {
		t.Fatalf("expected filtered menus to keep access-control root, overview, user, and public entries, got %#v", payload.Menus)
	}
	if payload.Menus[0].Path != "/access-control" || payload.Menus[0].TitleKey != "menu.access_control.title" {
		t.Fatalf("unexpected filtered root menu: %#v", payload.Menus[0])
	}
	if payload.Menus[0].Order != 0 {
		t.Fatalf("unexpected filtered root menu order: %#v", payload.Menus[0])
	}
	if payload.Menus[1].Path != "/access-control/overview" || payload.Menus[1].TitleKey != "menu.access_control.overview.title" {
		t.Fatalf("unexpected filtered overview menu: %#v", payload.Menus[1])
	}
	if payload.Menus[1].Order != 1 {
		t.Fatalf("unexpected filtered overview menu order: %#v", payload.Menus[1])
	}
	if payload.Menus[2].Path != "/access-control/users" || payload.Menus[2].TitleKey != "menu.access_control.users.title" {
		t.Fatalf("unexpected filtered user menu: %#v", payload.Menus[2])
	}
	if payload.Menus[2].Order != 2 {
		t.Fatalf("unexpected filtered user menu order: %#v", payload.Menus[2])
	}
	if payload.Menus[3].Path != "/profile" {
		t.Fatalf("unexpected filtered public menu: %#v", payload.Menus[3])
	}
	if payload.Menus[3].Order != 999 {
		t.Fatalf("unexpected filtered public menu order: %#v", payload.Menus[3])
	}
}

// TestBootstrapLocaleSnapshotDeduplicatesFallbackLocales 验证默认 locale 与回退 locale
// 相同时，bootstrap locale 快照不会返回重复语言项。
func TestBootstrapLocaleSnapshotDeduplicatesFallbackLocales(t *testing.T) {
	reader := newBootstrapReader(
		config.I18nConfig{
			DefaultLocale:    "zh-CN",
			FallbackLocale:   "zh-CN",
			SupportedLocales: nil,
		},
		i18n.New(config.I18nConfig{
			DefaultLocale:    "zh-CN",
			FallbackLocale:   "zh-CN",
			SupportedLocales: []string{"zh-CN"},
		}),
		nil,
		nil,
		nil,
	)

	snapshot := reader.localeSnapshot(httptest.NewRequest(http.MethodGet, authRoutePath(usercontract.AuthBootstrap), nil))
	if !slices.Equal(snapshot.SupportedLocales, []string{"zh-CN"}) {
		t.Fatalf("expected duplicate fallback locales to collapse, got %#v", snapshot.SupportedLocales)
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
			arrange: func(t *testing.T, _ *pluginTestAuthRepository) string {
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
			request := newAuthorizedRequestForSession(t, usersRoutePath(usercontract.UserByID, ":id", "7"), 7, sessionID)

			authAny, err := ctx.Services.Resolve((*pluginapi.AuthService)(nil))
			if err != nil {
				t.Fatalf("resolve auth service: %v", err)
			}

			token := authorizationTokenFromRequest(request)
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
			arrange: func(t *testing.T, _ *pluginTestAuthRepository) *http.Request {
				t.Helper()
				return newAuthorizedRequestForSession(t, usersRoutePath(usercontract.UserByID, ":id", "7"), 7, "missing-session")
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

				return newAuthorizedRequestForSession(t, usersRoutePath(usercontract.UserByID, ":id", "7"), 7, sessionID)
			},
		},
		{
			name: "expired session",
			arrange: func(t *testing.T, repo *pluginTestAuthRepository) *http.Request {
				t.Helper()
				sessionID := seedRefreshSession(t, repo, 7, time.Now().UTC().Add(-time.Minute))
				return newAuthorizedRequestForSession(t, usersRoutePath(usercontract.UserByID, ":id", "7"), 7, sessionID)
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

			assertInvalidTokenResponse(t, recorder, "en-US")
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

	authRepo := newCredentialRepository("alice", 7, passwordHash)
	ctx, engine := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice")), authRepo)

	payload, cookies := loginUser(t, engine, "alice", "secret", "en-US")
	assertLoginPayload(t, payload, 7, "alice", "Alice")
	refreshCookie := firstCookie(t, cookies)
	assertRefreshCookieWritten(t, refreshCookie, ctx.Config.Auth.RefreshCookieName)

	authAny, err := ctx.Services.Resolve((*pluginapi.AuthService)(nil))
	if err != nil {
		t.Fatalf("resolve auth service: %v", err)
	}
	claims, err := authAny.(pluginapi.AuthService).ParseAccessToken(context.Background(), payload.AccessToken)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	assertAccessClaims(t, claims, 7)
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

	authRepo := newCredentialRepository("alice", 7, passwordHash)
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice")), authRepo)

	_, cookies := loginUser(t, engine, "alice", "secret", "")
	refreshCookie := firstCookie(t, cookies)

	refreshRecorder := httptest.NewRecorder()
	refreshRequest := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthRefresh), nil)
	refreshRequest.AddCookie(refreshCookie)
	refreshRequest.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(refreshRecorder, refreshRequest)

	assertStatus(t, refreshRecorder, http.StatusOK)

	payload := decodeSuccessData[loginResponse](t, refreshRecorder)
	assertRefreshRotationPayload(t, payload)
	assertRotatedCookie(t, refreshCookie, refreshRecorder.Result().Cookies())
}

// TestRefreshRouteRejectsRestrictedSession 验证 must_change_password=true 的受限会话
// 不能继续通过 refresh 获取新 token。
func TestRefreshRouteRejectsRestrictedSession(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, candidate string) (store.UserCredential, error) {
			if candidate != defaultAdminUsername {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:             9,
				Username:           defaultAdminUsername,
				PasswordHash:       &passwordHash,
				MustChangePassword: true,
			}, nil
		},
	}
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(9, defaultAdminUsername, defaultAdminDisplay)), authRepo)

	_, cookies := loginUser(t, engine, defaultAdminUsername, "secret", "")
	refreshCookie := firstCookie(t, cookies)
	refreshSubject := parseRefreshCookieClaims(t, refreshCookie)

	refreshRecorder := httptest.NewRecorder()
	refreshRequest := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthRefresh), nil)
	refreshRequest.AddCookie(refreshCookie)
	refreshRequest.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(refreshRecorder, refreshRequest)

	assertForbiddenResponse(t, refreshRecorder, "en-US")
	assertNoCookieMutation(t, refreshRecorder.Result().Cookies())
	assertSessionActive(t, authRepo, refreshSubject.TokenID)
}

// TestRefreshRouteRejectsReusedRefreshCookie 验证 refresh 成功轮换后，旧 cookie
// 不能再次消费原 refresh session。
func TestRefreshRouteRejectsReusedRefreshCookie(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo := newCredentialRepository("alice", 7, passwordHash)
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice")), authRepo)

	_, cookies := loginUser(t, engine, "alice", "secret", "")
	refreshCookie := firstCookie(t, cookies)

	firstRefreshRecorder := httptest.NewRecorder()
	firstRefreshRequest := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthRefresh), nil)
	firstRefreshRequest.AddCookie(refreshCookie)
	firstRefreshRequest.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(firstRefreshRecorder, firstRefreshRequest)
	assertStatus(t, firstRefreshRecorder, http.StatusOK)

	reuseRecorder := httptest.NewRecorder()
	reuseRequest := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthRefresh), nil)
	reuseRequest.AddCookie(refreshCookie)
	reuseRequest.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(reuseRecorder, reuseRequest)
	assertInvalidTokenResponse(t, reuseRecorder, "en-US")
}

// TestRefreshRouteRejectsMissingCookie 验证缺少 refresh cookie 时仍返回统一的
// 本地化认证失败契约。
func TestRefreshRouteRejectsMissingCookie(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, &pluginTestAuthRepository{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthRefresh), nil)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertMissingTokenResponse(t, recorder, "en-US")
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

	authRepo := newCredentialRepository("alice", 7, passwordHash)
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice")), authRepo)

	_, cookies := loginUser(t, engine, "alice", "secret", "")
	refreshCookie := firstCookie(t, cookies)
	claims := parseRefreshCookieClaims(t, refreshCookie)

	logoutRecorder := httptest.NewRecorder()
	logoutRequest := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthLogout), nil)
	logoutRequest.AddCookie(refreshCookie)
	engine.ServeHTTP(logoutRecorder, logoutRequest)

	assertStatus(t, logoutRecorder, http.StatusOK)
	assertNilSuccessPayload(t, logoutRecorder)
	assertSessionRevoked(t, authRepo, claims.TokenID)
	assertClearedCookie(t, logoutRecorder.Result().Cookies(), refreshCookie.Name)
}

// TestLogoutRouteRejectsMissingCookie 验证缺少 refresh cookie 时，logout 继续复用
// 统一的本地化 refresh-session 错误契约。
func TestLogoutRouteRejectsMissingCookie(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, &pluginTestAuthRepository{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthLogout), nil)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertMissingTokenResponse(t, recorder, "en-US")
}

// TestRevokeAllSessionsRouteRevokesCurrentUserSessions 验证当前用户自助撤销会吊销
// 其全部 refresh sessions，并让当前受保护请求与后续 refresh 一并失效。
func TestRevokeAllSessionsRouteRevokesCurrentUserSessions(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo, engine := loginAliceEngine(t, passwordHash)
	loginPayload, refreshCookie, currentClaims := loginAliceAndParseSession(t, engine)

	secondarySessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	otherUserSessionID := seedRefreshSession(t, authRepo, 8, time.Now().UTC().Add(time.Hour))

	revokeRecorder := httptest.NewRecorder()
	revokeRequest := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthSessionsRevokeAll), nil)
	setBearerAuthorizationHeader(revokeRequest, loginPayload.AccessToken)
	revokeRequest.AddCookie(refreshCookie)
	engine.ServeHTTP(revokeRecorder, revokeRequest)

	assertStatus(t, revokeRecorder, http.StatusOK)
	assertNilSuccessPayload(t, revokeRecorder)

	for _, tokenID := range []string{currentClaims.TokenID, secondarySessionID} {
		assertSessionRevoked(t, authRepo, tokenID)
	}

	assertSessionActive(t, authRepo, otherUserSessionID)
	assertClearedCookie(t, revokeRecorder.Result().Cookies(), refreshCookie.Name)

	protectedRecorder := httptest.NewRecorder()
	protectedRequest := httptest.NewRequest(http.MethodGet, usersRoutePath(usercontract.UserByID, ":id", "7"), nil)
	setBearerAuthorizationHeader(protectedRequest, loginPayload.AccessToken)
	protectedRequest.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(protectedRecorder, protectedRequest)
	assertInvalidTokenResponse(t, protectedRecorder, "en-US")

	refreshRecorder := httptest.NewRecorder()
	refreshRequest := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthRefresh), nil)
	refreshRequest.AddCookie(refreshCookie)
	refreshRequest.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(refreshRecorder, refreshRequest)
	assertInvalidTokenResponse(t, refreshRecorder, "en-US")
}

// TestRevokeAllSessionsRouteRequiresAuthenticatedActor 验证当前用户自助撤销入口继续
// 复用统一 request-auth 守卫，而不是在插件内发散新的未登录响应格式。
func TestRevokeAllSessionsRouteRequiresAuthenticatedActor(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, &pluginTestAuthRepository{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthSessionsRevokeAll), nil)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertMissingTokenResponse(t, recorder, "en-US")
}

// TestRevokeOtherSessionsRouteRevokesNonCurrentSessions 验证当前用户保留当前会话时，
// 只会清退自己名下的其它有效 session，不会误伤当前会话或其他用户会话。
func TestRevokeOtherSessionsRouteRevokesNonCurrentSessions(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice"), testUser(8, "bob", "Bob")), authRepo)

	currentSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	otherSessionIDs := []string{
		seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(3*time.Hour)),
		seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(4*time.Hour)),
	}
	otherUserSessionID := seedRefreshSession(t, authRepo, 8, time.Now().UTC().Add(time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, authRoutePath(usercontract.AuthSessionsRevokeOthers), 7, currentSessionID)
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	assertNilSuccessPayload(t, recorder)
	assertSessionActive(t, authRepo, currentSessionID)
	for _, tokenID := range otherSessionIDs {
		assertSessionRevoked(t, authRepo, tokenID)
	}
	assertSessionActive(t, authRepo, otherUserSessionID)
}

// TestRevokeOtherSessionsRouteIgnoresAlreadyRevokedRaces 验证 revoke-others 在列出
// 后、定向吊销前遇到已被并发撤销的 session 时，仍继续清退剩余会话并返回成功。
func TestRevokeOtherSessionsRouteIgnoresAlreadyRevokedRaces(t *testing.T) {
	baseRepo := &pluginTestAuthRepository{}
	baseRepo.getUserCredentialByUsername = func(_ context.Context, username string) (store.UserCredential, error) {
		switch username {
		case "alice":
			return store.UserCredential{
				UserID:             7,
				Username:           "alice",
				MustChangePassword: false,
			}, nil
		case "bob":
			return store.UserCredential{
				UserID:             8,
				Username:           "bob",
				MustChangePassword: false,
			}, nil
		default:
			return store.UserCredential{}, store.ErrUserNotFound
		}
	}
	currentSessionID := seedRefreshSession(t, baseRepo, 7, time.Now().UTC().Add(2*time.Hour))
	time.Sleep(time.Microsecond)
	staleSessionID := seedRefreshSession(t, baseRepo, 7, time.Now().UTC().Add(3*time.Hour))
	time.Sleep(time.Microsecond)
	raceSessionID := seedRefreshSession(t, baseRepo, 7, time.Now().UTC().Add(4*time.Hour))
	otherUserSessionID := seedRefreshSession(t, baseRepo, 8, time.Now().UTC().Add(time.Hour))

	authRepo := &revokeByUserRaceAuthRepository{
		pluginTestAuthRepository: baseRepo,
		beforeFirstRevoke: func(input store.RevokeRefreshSessionByUserIDInput) {
			if input.TokenID != raceSessionID {
				t.Fatalf("expected first revoke target %q, got %q", raceSessionID, input.TokenID)
			}
			if err := baseRepo.RevokeRefreshSession(context.Background(), store.RevokeRefreshSessionInput{
				TokenID:   raceSessionID,
				RevokedAt: input.RevokedAt,
			}); err != nil {
				t.Fatalf("simulate concurrent revoke: %v", err)
			}
		},
	}

	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice"), testUser(8, "bob", "Bob")), authRepo)

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, authRoutePath(usercontract.AuthSessionsRevokeOthers), 7, currentSessionID)
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	assertNilSuccessPayload(t, recorder)
	assertSessionActive(t, baseRepo, currentSessionID)
	assertSessionRevoked(t, baseRepo, raceSessionID)
	assertSessionRevoked(t, baseRepo, staleSessionID)
	assertSessionActive(t, baseRepo, otherUserSessionID)
}

// TestRevokeOtherSessionsRouteRequiresAuthenticatedActor 验证保留当前会话的批量清退
// 入口继续复用统一 request-auth 守卫，而不是发散新的未登录响应契约。
func TestRevokeOtherSessionsRouteRequiresAuthenticatedActor(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, &pluginTestAuthRepository{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthSessionsRevokeOthers), nil)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertMissingTokenResponse(t, recorder, "en-US")
}

// TestRevokeOtherSessionsRouteAllowsOnlyCurrentSession 验证当前用户只剩当前会话时，
// revoke-others 仍幂等返回成功，且不会额外清理 refresh cookie。
func TestRevokeOtherSessionsRouteAllowsOnlyCurrentSession(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice")), authRepo)

	currentSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, authRoutePath(usercontract.AuthSessionsRevokeOthers), 7, currentSessionID)
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	assertNilSuccessPayload(t, recorder)
	assertSessionActive(t, authRepo, currentSessionID)
	assertNoCookieMutation(t, recorder.Result().Cookies())
}

// TestListCurrentUserSessionsRouteReturnsActiveSessions 验证当前用户自助会话列表只返回
// 其自身当前有效的 refresh sessions，并准确标记当前请求会话。
func TestListCurrentUserSessionsRouteReturnsActiveSessions(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice"), testUser(8, "bob", "Bob")), authRepo)

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
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, authRoutePath(usercontract.AuthSessions), 7, currentSessionID)
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)

	payload := decodeSuccessData[[]sessionSummary](t, recorder)
	if len(payload) != 2 {
		t.Fatalf("expected two active sessions, got %#v", payload)
	}
	assertSessionSummary(t, payload[0], newerSessionID, false)
	assertSessionSummary(t, payload[1], currentSessionID, true)
	assertSessionsAbsent(t, payload, expiredSessionID, otherUserSessionID)
}

// TestListCurrentUserSessionsRouteAppliesLimit 验证当前用户会话列表会在插件边界内
// 应用显式 limit，而不要求仓储提前暴露分页协议。
func TestListCurrentUserSessionsRouteAppliesLimit(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice")), authRepo)

	currentSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	time.Sleep(time.Microsecond)
	middleSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	time.Sleep(time.Microsecond)
	newestSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(3*time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, authRoutePath(usercontract.AuthSessions)+"?limit=2", 7, currentSessionID)
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)

	payload := decodeSuccessData[[]sessionSummary](t, recorder)
	if len(payload) != 2 {
		t.Fatalf("expected two sessions after limit, got %#v", payload)
	}
	assertSessionSummary(t, payload[0], newestSessionID, false)
	assertSessionSummary(t, payload[1], middleSessionID, false)
	assertSessionsAbsent(t, payload, currentSessionID)
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
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, authRoutePath(usercontract.AuthSessions)+"?limit=0", 7, seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertInvalidArgumentFieldResponse(t, recorder, "en-US", "limit")
}

// TestListCurrentUserSessionsRouteRequiresAuthenticatedActor 验证当前用户会话列表继续
// 复用统一 request-auth 守卫，而不是在插件内发散新的未登录契约。
func TestListCurrentUserSessionsRouteRequiresAuthenticatedActor(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{}, &pluginTestAuthRepository{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, authRoutePath(usercontract.AuthSessions), nil)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertMissingTokenResponse(t, recorder, "en-US")
}

// TestAdminListUserSessionsRouteReturnsActiveSessions 验证管理员读取入口只返回目标用户
// 的当前有效 session，并继续标记请求主体自己的当前会话。
func TestAdminListUserSessionsRouteReturnsActiveSessions(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	engine := newSessionAdminEngine(t, authRepo,
		testUser(7, "alice", "Alice"),
		testUser(8, "bob", "Bob"),
		testUser(9, "admin", "Admin"),
	)

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
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, usersRoutePath(usercontract.UserSessions, ":id", "7"), 9, adminSession)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	payload := decodeSuccessData[[]generated.SessionSummary](t, recorder)
	assertGeneratedActiveSessions(t, payload,
		generated.SessionSummary{SessionId: targetNewerSession, Current: false},
		generated.SessionSummary{SessionId: targetCurrentSession, Current: false},
	)
	assertGeneratedSessionsFiltered(t, payload, targetExpiredSession, adminSession, otherUserSession)
}

// TestAdminListUserSessionsRouteAppliesLimit 验证管理员读取入口同样支持最小显式
// limit，避免首次会话治理就扩散分页契约到仓储层。
func TestAdminListUserSessionsRouteAppliesLimit(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	engine := newSessionAdminEngine(t, authRepo,
		testUser(7, "alice", "Alice"),
		testUser(9, "admin", "Admin"),
	)

	oldestSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	time.Sleep(time.Microsecond)
	middleSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	time.Sleep(time.Microsecond)
	newestSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(3*time.Hour))
	adminSessionID := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, usersRoutePath(usercontract.UserSessions, ":id", "7")+"?limit=2", 9, adminSessionID)
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	payload := decodeSuccessData[[]generated.SessionSummary](t, recorder)
	assertGeneratedActiveSessions(t, payload,
		generated.SessionSummary{SessionId: newestSessionID, Current: false},
		generated.SessionSummary{SessionId: middleSessionID, Current: false},
	)
	assertGeneratedSessionsFiltered(t, payload, oldestSessionID, adminSessionID)
}

// TestAdminListUserSessionsRouteRejectsInvalidLimit 验证管理员会话读取入口会拒绝非法
// limit，并保持统一的参数错误契约。
func TestAdminListUserSessionsRouteRejectsInvalidLimit(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	engine := newSessionAdminEngine(t, authRepo,
		testUser(7, "alice", "Alice"),
		testUser(9, "admin", "Admin"),
	)

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, usersRoutePath(usercontract.UserSessions, ":id", "7")+"?limit=101", 9, seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertInvalidArgumentFieldResponse(t, recorder, "en-US", "limit")
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
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, usersRoutePath(usercontract.UserSessions, ":id", "7"), 7, seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)
	payload := decodeErrorResponse(t, recorder)
	assertStatus(t, recorder, http.StatusForbidden)
	assertContractErrorPayload(t, payload, messagecontract.AuthForbidden, "en-US")
	if payload.Details["permission"] != usercontract.UserSessionReadPermission.String() {
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
	}, authRepo, map[uint64][]rbacstore.Permission{
		9: {{Code: usercontract.UserSessionReadPermission.String()}},
	})

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, usersRoutePath(usercontract.UserSessions, ":id", "7"), 9, seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != messagecontract.UserNotFound.String() || payload.Locale != "en-US" {
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
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, authRoutePath(usercontract.AuthSessionRevoke, ":sessionID", targetSessionID), 7, currentSessionID)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if payload := decodeSuccessData[any](t, recorder); payload != nil {
		t.Fatalf("expected current-session revoke payload to be nil, got %#v", payload)
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

	authRepo, engine := loginAliceEngine(t, passwordHash)

	loginPayload, loginCookies := loginUser(t, engine, "alice", "secret", "")
	refreshCookie := firstCookie(t, loginCookies)
	currentClaims := parseRefreshCookieClaims(t, refreshCookie)

	revokeRecorder := httptest.NewRecorder()
	revokeRequest := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthSessionRevoke, ":sessionID", currentClaims.TokenID), nil)
	setBearerAuthorizationHeader(revokeRequest, loginPayload.AccessToken)
	revokeRequest.AddCookie(refreshCookie)
	engine.ServeHTTP(revokeRecorder, revokeRequest)

	assertStatus(t, revokeRecorder, http.StatusOK)
	assertNilSuccessPayload(t, revokeRecorder)
	assertSessionRevoked(t, authRepo, currentClaims.TokenID)
	assertClearedCookie(t, revokeRecorder.Result().Cookies(), refreshCookie.Name)
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
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, authRoutePath(usercontract.AuthSessionRevoke, ":sessionID", "missing-session"), 7, seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertSessionNotFoundResponse(t, recorder, "en-US")
}

// TestAdminRevokeUserSessionRouteRevokesOnlyTargetSession 验证管理员定向吊销只会影响
// 目标用户的指定 session，不会误伤其他用户或同用户其他会话。
func TestAdminRevokeUserSessionRouteRevokesOnlyTargetSession(t *testing.T) {
	authRepo := &pluginTestAuthRepository{}
	_, engine := newPluginTestContextWithPermissions(t, fixedUserRepository(
		testUser(7, "alice", "Alice"),
		testUser(8, "bob", "Bob"),
		testUser(9, "admin", "Admin"),
	), authRepo, map[uint64][]rbacstore.Permission{
		9: {{Code: usercontract.UserSessionRevokePermission.String()}},
	})

	targetSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	otherTargetSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(3*time.Hour))
	otherUserSessionID := seedRefreshSession(t, authRepo, 8, time.Now().UTC().Add(time.Hour))
	adminSessionID := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, usersRoutePath(usercontract.UserSessionByIDRevoke, ":id", "7", ":sessionID", targetSessionID), 9, adminSessionID)
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	assertNilSuccessPayload(t, recorder)
	assertSessionRevoked(t, authRepo, targetSessionID)
	assertSessionActive(t, authRepo, otherTargetSessionID)
	assertSessionActive(t, authRepo, otherUserSessionID)
	assertSessionActive(t, authRepo, adminSessionID)
}

// TestAdminRevokeUserSessionRouteClearsCurrentCookieWhenRevokingSelfCurrentSession 验证管理员
// 定向吊销自己当前请求绑定的 session 时，会同步清理 refresh cookie。
func TestAdminRevokeUserSessionRouteClearsCurrentCookieWhenRevokingSelfCurrentSession(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	_, engine := loginAdminEngine(t, passwordHash)

	loginPayload, loginCookies := loginUser(t, engine, "admin", "secret", "")
	refreshCookie := firstCookie(t, loginCookies)
	currentClaims := parseRefreshCookieClaims(t, refreshCookie)

	revokeRecorder := httptest.NewRecorder()
	revokeRequest := httptest.NewRequest(http.MethodPost, usersRoutePath(usercontract.UserSessionByIDRevoke, ":id", "9", ":sessionID", currentClaims.TokenID), nil)
	setBearerAuthorizationHeader(revokeRequest, loginPayload.AccessToken)
	revokeRequest.AddCookie(refreshCookie)
	engine.ServeHTTP(revokeRecorder, revokeRequest)

	assertStatus(t, revokeRecorder, http.StatusOK)
	assertNilSuccessPayload(t, revokeRecorder)
	assertClearedCookie(t, revokeRecorder.Result().Cookies(), refreshCookie.Name)
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
	}, authRepo, map[uint64][]rbacstore.Permission{
		9: {{Code: usercontract.UserSessionRevokePermission.String()}},
	})

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, usersRoutePath(usercontract.UserSessionByIDRevoke, ":id", "7", ":sessionID", "missing-session"), 9, seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertSessionNotFoundResponse(t, recorder, "en-US")
}

// TestAdminRevokeUserSessionsRouteRevokesTargetUserSessions 验证管理员入口会按用户 ID
// 吊销目标用户的全部 refresh sessions，并保持其他用户 session 不受影响。
func TestAdminRevokeUserSessionsRouteRevokesTargetUserSessions(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	authRepo, engine := loginAdminEngine(t, passwordHash)

	targetSessionA := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	targetSessionB := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(2*time.Hour))
	otherUserSession := seedRefreshSession(t, authRepo, 8, time.Now().UTC().Add(time.Hour))
	adminSession := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, usersRoutePath(usercontract.UserSessionsRevokeAll, ":id", "7"), 9, adminSession)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	assertNilSuccessPayload(t, recorder)
	assertSessionRevoked(t, authRepo, targetSessionA)
	assertSessionRevoked(t, authRepo, targetSessionB)
	assertSessionActive(t, authRepo, otherUserSession)
	assertSessionActive(t, authRepo, adminSession)
	assertNoCookieMutation(t, recorder.Result().Cookies())
}

// TestAdminRevokeUserSessionsRouteClearsCurrentCookieWhenRevokingSelf 验证管理员按自己的
// 用户 ID 执行批量吊销时，会同步清理当前 refresh cookie。
func TestAdminRevokeUserSessionsRouteClearsCurrentCookieWhenRevokingSelf(t *testing.T) {
	passwordHash, err := newPasswordHasher().Hash("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	_, engine := loginAdminEngine(t, passwordHash)

	loginPayload, loginCookies := loginUser(t, engine, "admin", "secret", "")
	refreshCookie := firstCookie(t, loginCookies)

	revokeRecorder := httptest.NewRecorder()
	revokeRequest := httptest.NewRequest(http.MethodPost, usersRoutePath(usercontract.UserSessionsRevokeAll, ":id", "9"), nil)
	setBearerAuthorizationHeader(revokeRequest, loginPayload.AccessToken)
	revokeRequest.AddCookie(refreshCookie)
	engine.ServeHTTP(revokeRecorder, revokeRequest)

	assertStatus(t, revokeRecorder, http.StatusOK)
	assertNilSuccessPayload(t, revokeRecorder)
	assertClearedCookie(t, revokeRecorder.Result().Cookies(), refreshCookie.Name)
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
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, usersRoutePath(usercontract.UserSessionsRevokeAll, ":id", "8"), 7, seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)
	payload := decodeErrorResponse(t, recorder)
	assertStatus(t, recorder, http.StatusForbidden)
	assertContractErrorPayload(t, payload, messagecontract.AuthForbidden, "en-US")
	if payload.Details["permission"] != usercontract.UserSessionRevokePermission.String() {
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
	}, authRepo, map[uint64][]rbacstore.Permission{
		9: {{Code: usercontract.UserSessionRevokePermission.String()}},
	})

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, usersRoutePath(usercontract.UserSessionsRevokeAll, ":id", "not-a-number"), 9, seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour)))
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)
	assertInvalidArgumentFieldResponse(t, recorder, "en-US", "id")
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
	request := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthLogin), strings.NewReader(`{"username":"alice","password":"wrong"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assertContractErrorPayload(t, payload, messagecontract.AuthInvalidCredentials, "en-US")
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
			request := httptest.NewRequest(http.MethodPost, authRoutePath(usercontract.AuthLogin), strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			engine.ServeHTTP(recorder, request)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, recorder.Code)
			}

			var payload httpx.ErrorResponse
			if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			assertContractErrorPayload(t, payload, messagecontract.CommonInvalidArgument, "zh-CN")
			if payload.Details["field"] != tc.field {
				t.Fatalf("expected %s field detail, got %#v", tc.field, payload.Details)
			}
		})
	}
}

// TestCompleteRequiredPasswordChangeRouteAllowsRestrictedSession 验证
// 首次强制改密接口只要求受限会话提供 new_password。
func TestCompleteRequiredPasswordChangeRouteAllowsRestrictedSession(t *testing.T) {
	currentHash, err := newPasswordHasher().Hash(defaultAdminPassword)
	if err != nil {
		t.Fatalf("hash default admin password: %v", err)
	}

	var called bool
	var received store.ChangePasswordAndRevokeOtherRefreshSessionsInput
	authRepo := &passwordChangeAtomicAuthRepository{
		pluginTestAuthRepository: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(_ context.Context, candidate string) (store.UserCredential, error) {
				if candidate != defaultAdminUsername {
					return store.UserCredential{}, store.ErrUserNotFound
				}
				return store.UserCredential{
					UserID:             9,
					Username:           defaultAdminUsername,
					PasswordHash:       &currentHash,
					MustChangePassword: true,
				}, nil
			},
		},
		changePasswordAndRevokeOtherRefreshSessions: func(_ context.Context, input store.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
			called = true
			received = input
			return nil
		},
	}
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(9, defaultAdminUsername, defaultAdminDisplay)), authRepo)

	currentSessionID := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))
	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, authRoutePath(usercontract.AuthCompleteRequiredPasswordChange), 9, currentSessionID)
	request.Body = io.NopCloser(strings.NewReader(`{"new_password":"next-password-123"}`))
	request.ContentLength = int64(len(`{"new_password":"next-password-123"}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusOK)
	assertNilSuccessPayload(t, recorder)
	if !called {
		t.Fatal("expected password change repository operation to be called")
	}
	if received.CurrentTokenID != currentSessionID {
		t.Fatalf("expected current session id %q, got %q", currentSessionID, received.CurrentTokenID)
	}
	if received.MustChangePassword {
		t.Fatal("expected must-change flag to be cleared")
	}
}

// TestChangePasswordRouteRejectsMissingCurrentPassword 验证
// 普通改密接口缺少原密码时返回稳定的参数错误契约。
func TestChangePasswordRouteRejectsMissingCurrentPassword(t *testing.T) {
	currentHash, err := newPasswordHasher().Hash("current-password")
	if err != nil {
		t.Fatalf("hash current password: %v", err)
	}

	authRepo := &passwordChangeAtomicAuthRepository{
		pluginTestAuthRepository: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(_ context.Context, candidate string) (store.UserCredential, error) {
				if candidate != "alice" {
					return store.UserCredential{}, store.ErrUserNotFound
				}
				return store.UserCredential{
					UserID:             7,
					Username:           "alice",
					PasswordHash:       &currentHash,
					MustChangePassword: false,
				}, nil
			},
		},
		changePasswordAndRevokeOtherRefreshSessions: func(context.Context, store.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
			t.Fatal("expected password change repository operation not to be called")
			return nil
		},
	}
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice")), authRepo)

	currentSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, authRoutePath(usercontract.AuthChangePassword), 7, currentSessionID)
	request.Body = io.NopCloser(strings.NewReader(`{"current_password":"","new_password":"next-password-123"}`))
	request.ContentLength = int64(len(`{"current_password":"","new_password":"next-password-123"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertInvalidArgumentFieldResponse(t, recorder, "en-US", "current_password")
}

// TestCompleteRequiredPasswordChangeRouteRejectsNonRestrictedSession 验证
// 非 must_change_password 会话不能调用首次强制改密接口。
func TestCompleteRequiredPasswordChangeRouteRejectsNonRestrictedSession(t *testing.T) {
	currentHash, err := newPasswordHasher().Hash("current-password")
	if err != nil {
		t.Fatalf("hash current password: %v", err)
	}

	authRepo := &passwordChangeAtomicAuthRepository{
		pluginTestAuthRepository: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(_ context.Context, candidate string) (store.UserCredential, error) {
				if candidate != "alice" {
					return store.UserCredential{}, store.ErrUserNotFound
				}
				return store.UserCredential{
					UserID:       7,
					Username:     "alice",
					PasswordHash: &currentHash,
				}, nil
			},
		},
		changePasswordAndRevokeOtherRefreshSessions: func(context.Context, store.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
			t.Fatal("expected password change repository operation not to be called")
			return nil
		},
	}
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(7, "alice", "Alice")), authRepo)

	currentSessionID := seedRefreshSession(t, authRepo, 7, time.Now().UTC().Add(time.Hour))
	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, authRoutePath(usercontract.AuthCompleteRequiredPasswordChange), 7, currentSessionID)
	request.Body = io.NopCloser(strings.NewReader(`{"new_password":"next-password-123"}`))
	request.ContentLength = int64(len(`{"new_password":"next-password-123"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertForbiddenResponse(t, recorder, "en-US")
}

// TestRestrictedSessionCannotAccessBusinessRoutes 验证
// must_change_password=true 的受限会话访问普通业务接口时返回 403。
func TestRestrictedSessionCannotAccessBusinessRoutes(t *testing.T) {
	currentHash, err := newPasswordHasher().Hash(defaultAdminPassword)
	if err != nil {
		t.Fatalf("hash default admin password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, candidate string) (store.UserCredential, error) {
			if candidate != defaultAdminUsername {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:             9,
				Username:           defaultAdminUsername,
				PasswordHash:       &currentHash,
				MustChangePassword: true,
			}, nil
		},
	}
	_, engine := newPluginTestContextWithPermissions(
		t,
		fixedUserRepository(testUser(9, defaultAdminUsername, defaultAdminDisplay)),
		authRepo,
		map[uint64][]rbacstore.Permission{
			9: {{Code: usercontract.UserReadPermission.String()}},
		},
	)

	currentSessionID := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))
	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, usersRoutePath(usercontract.UserCollection), 9, currentSessionID)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertForbiddenResponse(t, recorder, "en-US")
}

// TestRestrictedSessionCanReadBootstrap 验证 must_change_password=true 的受限会话
// 仍可访问 bootstrap 契约，以便 web 恢复当前用户、权限与受限态快照。
func TestRestrictedSessionCanReadBootstrap(t *testing.T) {
	currentHash, err := newPasswordHasher().Hash(defaultAdminPassword)
	if err != nil {
		t.Fatalf("hash default admin password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, candidate string) (store.UserCredential, error) {
			if candidate != defaultAdminUsername {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:             9,
				Username:           defaultAdminUsername,
				PasswordHash:       &currentHash,
				MustChangePassword: true,
			}, nil
		},
	}
	_, engine := newPluginTestContextWithPermissions(
		t,
		fixedUserRepository(testUser(9, defaultAdminUsername, defaultAdminDisplay)),
		authRepo,
		map[uint64][]rbacstore.Permission{
			9: {{Code: usercontract.UserReadPermission.String()}},
		},
	)

	currentSessionID := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))
	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodGet, authRoutePath(usercontract.AuthBootstrap), 9, currentSessionID)
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	payload := decodeSuccessData[bootstrapResponse](t, recorder)
	if !payload.MustChangePassword {
		t.Fatalf("expected restricted bootstrap snapshot, got %#v", payload)
	}
	if payload.User.Username != defaultAdminUsername || payload.User.DisplayName != defaultAdminDisplay {
		t.Fatalf("expected default admin identity, got %#v", payload.User)
	}
	if !slices.Contains(payload.Permissions, usercontract.UserReadPermission.String()) {
		t.Fatalf("expected bootstrap permissions to include %s, got %#v", usercontract.UserReadPermission.String(), payload.Permissions)
	}
}

// TestRestrictedSessionCannotUseNormalChangePasswordRoute 验证
// 受限会话不能再复用普通 change-password 接口。
func TestRestrictedSessionCannotUseNormalChangePasswordRoute(t *testing.T) {
	currentHash, err := newPasswordHasher().Hash(defaultAdminPassword)
	if err != nil {
		t.Fatalf("hash default admin password: %v", err)
	}

	authRepo := &pluginTestAuthRepository{
		getUserCredentialByUsername: func(_ context.Context, candidate string) (store.UserCredential, error) {
			if candidate != defaultAdminUsername {
				return store.UserCredential{}, store.ErrUserNotFound
			}
			return store.UserCredential{
				UserID:             9,
				Username:           defaultAdminUsername,
				PasswordHash:       &currentHash,
				MustChangePassword: true,
			}, nil
		},
	}
	_, engine := newPluginTestContext(t, fixedUserRepository(testUser(9, defaultAdminUsername, defaultAdminDisplay)), authRepo)

	currentSessionID := seedRefreshSession(t, authRepo, 9, time.Now().UTC().Add(time.Hour))
	recorder := httptest.NewRecorder()
	request := newAuthorizedRequestForSessionWithMethod(t, http.MethodPost, authRoutePath(usercontract.AuthChangePassword), 9, currentSessionID)
	request.Body = io.NopCloser(strings.NewReader(`{"current_password":"graft-admin","new_password":"next-password-123"}`))
	request.ContentLength = int64(len(`{"current_password":"graft-admin","new_password":"next-password-123"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	assertForbiddenResponse(t, recorder, "en-US")
}
