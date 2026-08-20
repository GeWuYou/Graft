package user

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	userstore "graft/server/modules/user/store"
)

type idempotentDeleteRepository struct {
	createUserRepository
	targetID    uint64
	deleted     bool
	deleteCalls int
}

func (r *idempotentDeleteRepository) GetByID(_ context.Context, id uint64) (userstore.User, error) {
	if id != r.targetID || r.deleted {
		return userstore.User{}, userstore.ErrUserNotFound
	}
	return userstore.User{ID: id, Username: "delete-target", Display: "Delete Target", Status: "enabled"}, nil
}

func (r *idempotentDeleteRepository) GetDeletionState(_ context.Context, id uint64) (userstore.UserDeletionState, error) {
	if id != r.targetID {
		return userstore.UserDeletionState{}, userstore.ErrUserNotFound
	}
	return userstore.UserDeletionState{ID: id, Username: "delete-target", Deleted: r.deleted}, nil
}

func (r *idempotentDeleteRepository) Delete(context.Context, userstore.DeleteUserInput) error {
	r.deleteCalls++
	r.deleted = true
	return nil
}

func (r *idempotentDeleteRepository) RunInCompositeTransaction(
	ctx context.Context,
	callback func(context.Context, userstore.UserRepository, *sql.Tx) error,
) error {
	return callback(ctx, r, nil)
}

type idempotentDeleteAuthTransactions struct {
	revokeCalls int
}

func (a *idempotentDeleteAuthTransactions) BindAuthTransaction(*sql.Tx) (moduleapi.AuthTransactionAdapter, error) {
	return idempotentDeleteAuthTransaction{owner: a}, nil
}

type idempotentDeleteAuthTransaction struct {
	owner *idempotentDeleteAuthTransactions
}

func (idempotentDeleteAuthTransaction) ProvisionPasswordCredential(context.Context, moduleapi.AuthCredentialProvisionInput) error {
	return nil
}

func (a idempotentDeleteAuthTransaction) RevokeSessions(context.Context, uint64) error {
	a.owner.revokeCalls++
	return nil
}

func TestDeleteUserRouteReturnsNoContentForInitialAndTombstoneDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repository := &idempotentDeleteRepository{targetID: 9}
	authTransactions := &idempotentDeleteAuthTransactions{}
	auditBus := eventbus.New(zap.NewNop())
	auditCalls := 0
	if err := auditBus.Subscribe(string(moduleapi.AuditRecordEventName), func(context.Context, eventbus.Event) error {
		auditCalls++
		return nil
	}); err != nil {
		t.Fatalf("subscribe user audit event: %v", err)
	}
	service := userService{
		users:      repository,
		composites: repository,
		authTx:     authTransactions,
		auditBus:   auditBus,
		logger:     zap.NewNop(),
	}
	pass := func(ginCtx *gin.Context) { ginCtx.Next() }
	router := gin.New()
	registrar := userRouteRegistrar{
		ctx: &module.Context{
			Logger: zap.NewNop(),
			I18n: i18n.MustNew(config.I18nConfig{
				DefaultLocale:    "zh-CN",
				FallbackLocale:   "zh-CN",
				SupportedLocales: []string{"zh-CN", "en-US"},
			}),
		},
		moduleName: moduleID,
		userSvc:    service,
		guards: routeGuards{
			userRead:          pass,
			userDisable:       pass,
			restrictedSession: pass,
		},
	}
	group := router.Group("/api/users")
	registrar.registerUserReadRoutes(group)
	registrar.registerDeleteUserRoute(group)

	for attempt := 1; attempt <= 2; attempt++ {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodDelete, "/api/users/9", nil)
		router.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
			t.Fatalf("DELETE attempt %d response = %d %q, want 204 with no body", attempt, response.Code, response.Body.String())
		}
	}
	if repository.deleteCalls != 1 {
		t.Fatalf("repository delete calls = %d, want 1", repository.deleteCalls)
	}
	if authTransactions.revokeCalls != 1 {
		t.Fatalf("session revoke calls = %d, want 1", authTransactions.revokeCalls)
	}
	if auditCalls != 1 {
		t.Fatalf("user delete audit calls = %d, want 1", auditCalls)
	}

	getResponse := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/users/9", nil)
	router.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusNotFound {
		t.Fatalf("GET tombstone response = %d %q, want 404", getResponse.Code, getResponse.Body.String())
	}

	missingDeleteResponse := httptest.NewRecorder()
	missingDeleteRequest := httptest.NewRequest(http.MethodDelete, "/api/users/404", nil)
	router.ServeHTTP(missingDeleteResponse, missingDeleteRequest)
	if missingDeleteResponse.Code != http.StatusNotFound {
		t.Fatalf("DELETE missing user response = %d %q, want 404", missingDeleteResponse.Code, missingDeleteResponse.Body.String())
	}
}

func TestReadUserListQueryAppliesDefaultsAndLimitCap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = httptest.NewRequest("GET", "/api/users?keyword=admin&status=enabled&role_id=7&limit=999&offset=4", nil)

	query, ok := readUserListQuery(ginCtx)
	if !ok {
		t.Fatal("readUserListQuery() returned invalid")
	}
	if query.Keyword != "admin" || query.Status != "enabled" || query.Limit != maximumUserListLimit || query.Offset != 4 || query.RoleID == nil || *query.RoleID != 7 {
		t.Fatalf("readUserListQuery() = %#v", query)
	}
}

func TestReadUserListQueryRejectsInvalidValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, rawQuery := range []string{"limit=0", "offset=-1", "status=unknown", "role_id=0"} {
		t.Run(rawQuery, func(t *testing.T) {
			ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ginCtx.Request = httptest.NewRequest("GET", "/api/users?"+rawQuery, nil)
			if _, ok := readUserListQuery(ginCtx); ok {
				t.Fatalf("readUserListQuery(%q) accepted invalid query", rawQuery)
			}
		})
	}
}
