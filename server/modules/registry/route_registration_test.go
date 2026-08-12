package registry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
	containerdi "graft/server/internal/container"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	buildcontract "graft/server/modules/build/contract"
	registrycontract "graft/server/modules/registry/contract"
	registrystore "graft/server/modules/registry/store"
)

func TestRegistryRoutesRequireExpectedPermissions(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		method     string
		path       string
		body       string
		permission string
	}{
		{name: "connections read", method: http.MethodGet, path: "/api/registries", permission: registrycontract.ReadPermission},
		{name: "connection create", method: http.MethodPost, path: "/api/registries", body: `{"connection_ref":"registry:new","display_name":"New","provider":"generic_oci","endpoint":"https://new.example"}`, permission: registrycontract.CreatePermission},
		{name: "connection update", method: http.MethodPut, path: "/api/registries/registry:primary", body: `{"display_name":"Primary","endpoint":"https://registry.example","enabled":true,"insecure":false}`, permission: registrycontract.UpdatePermission},
		{name: "connection delete", method: http.MethodDelete, path: "/api/registries/registry:primary", permission: registrycontract.DeletePermission},
		{name: "connection verify", method: http.MethodPost, path: "/api/registries/registry:primary/verify", permission: registrycontract.VerifyPermission},
		{name: "repository assignment", method: http.MethodGet, path: "/api/registries/registry:primary/repository-assignments?repository_ref=team/app", permission: registrycontract.AssignmentManagePermission},
		{name: "available build destinations", method: http.MethodGet, path: "/api/registries/available-destinations", permission: buildcontract.BuildCreatePermission},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			repository := newRegistryRouteRepository()
			authorizer := &registryRouteAuthorizer{}
			engine := newRegistryRouteTestEngine(t, repository, authorizer)
			authorizer.reset()
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, registryRouteRequest(testCase.method, testCase.path, testCase.body, "7"))
			if response.Code >= http.StatusInternalServerError {
				t.Fatalf("route returned %d: %s", response.Code, response.Body.String())
			}
			if !slices.Contains(authorizer.permissions, testCase.permission) {
				t.Fatalf("expected permission %q, got %#v", testCase.permission, authorizer.permissions)
			}
		})
	}
}

func TestRegistryConnectionResponseDoesNotExposeCredentialReference(t *testing.T) {
	repository := newRegistryRouteRepository()
	repository.connection.CredentialRef = "credential:production-secret"
	engine := newRegistryRouteTestEngine(t, repository, &registryRouteAuthorizer{})

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, registryRouteRequest(http.MethodGet, "/api/registries/registry:primary", "", "7"))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, "credential:production-secret") || strings.Contains(body, `"credential_ref"`) {
		t.Fatalf("connection response leaked credential material: %s", body)
	}
	if !strings.Contains(body, `"credential_configured":true`) {
		t.Fatalf("connection response did not expose credential state: %s", body)
	}
}

func TestRegistryRepositoryAndAssignmentRoutesMutateOwnedResources(t *testing.T) {
	repository := newRegistryRouteRepository()
	engine := newRegistryRouteTestEngine(t, repository, &registryRouteAuthorizer{})

	create := httptest.NewRecorder()
	engine.ServeHTTP(create, registryRouteRequest(http.MethodPost, "/api/registries/registry:primary/repositories", `{"repository_ref":"team/release","display_name":"New repository","allow_pull":true,"allow_push":true}`, "7"))
	if create.Code != http.StatusCreated || !strings.Contains(create.Body.String(), "team/release") {
		t.Fatalf("create repository = %d: %s", create.Code, create.Body.String())
	}

	update := httptest.NewRecorder()
	engine.ServeHTTP(update, registryRouteRequest(http.MethodPut, "/api/registries/registry:primary/repositories?repository_ref=team/release", `{"display_name":"Release repository","allow_pull":true,"allow_push":false}`, "7"))
	if update.Code != http.StatusOK || !strings.Contains(update.Body.String(), `"allow_push":false`) {
		t.Fatalf("update repository = %d: %s", update.Code, update.Body.String())
	}

	grant := httptest.NewRecorder()
	engine.ServeHTTP(grant, registryRouteRequest(http.MethodPost, "/api/registries/registry:primary/repository-assignments?repository_ref=team/release", `{"user_id":9}`, "7"))
	if grant.Code != http.StatusCreated || !strings.Contains(grant.Body.String(), `"user_id":9`) {
		t.Fatalf("grant assignment = %d: %s", grant.Code, grant.Body.String())
	}

	revoke := httptest.NewRecorder()
	engine.ServeHTTP(revoke, registryRouteRequest(http.MethodDelete, "/api/registries/registry:primary/repository-assignments/9?repository_ref=team/release", "", "7"))
	if revoke.Code != http.StatusOK {
		t.Fatalf("revoke assignment = %d: %s", revoke.Code, revoke.Body.String())
	}

	deleteResponse := httptest.NewRecorder()
	engine.ServeHTTP(deleteResponse, registryRouteRequest(http.MethodDelete, "/api/registries/registry:primary/repositories?repository_ref=team/release", "", "7"))
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete repository = %d: %s", deleteResponse.Code, deleteResponse.Body.String())
	}
}

func TestRegistryAvailableDestinationsAreBoundToRequestActor(t *testing.T) {
	repository := newRegistryRouteRepository()
	repository.destinationsByActor[7] = []registrystore.Destination{{ConnectionRef: "registry:primary", ConnectionName: "Primary", RepositoryRef: "team/app", RepositoryName: "Application", AllowPull: false, AllowPush: true}}
	engine := newRegistryRouteTestEngine(t, repository, &registryRouteAuthorizer{})

	allowed := httptest.NewRecorder()
	engine.ServeHTTP(allowed, registryRouteRequest(http.MethodGet, "/api/registries/available-destinations?limit=1&offset=0", "", "7"))
	if allowed.Code != http.StatusOK || !strings.Contains(allowed.Body.String(), "team/app") {
		t.Fatalf("authorized destinations = %d: %s", allowed.Code, allowed.Body.String())
	}
	if repository.availableActor != 7 {
		t.Fatalf("available destination actor = %d, want 7", repository.availableActor)
	}
	if !strings.Contains(allowed.Body.String(), `"allow_pull":false`) {
		t.Fatalf("available destination did not preserve pull policy: %s", allowed.Body.String())
	}
	if !strings.Contains(allowed.Body.String(), `"total":1`) || !strings.Contains(allowed.Body.String(), `"limit":1`) || !strings.Contains(allowed.Body.String(), `"offset":0`) {
		t.Fatalf("available destination pagination = %s", allowed.Body.String())
	}

	denied := httptest.NewRecorder()
	engine.ServeHTTP(denied, registryRouteRequest(http.MethodGet, "/api/registries/available-destinations", "", "8"))
	if denied.Code != http.StatusOK || strings.Contains(denied.Body.String(), "team/app") {
		t.Fatalf("unassigned destinations = %d: %s", denied.Code, denied.Body.String())
	}
}

func TestRegistrySystemManagedConnectionsCannotBeUpdatedOrDeleted(t *testing.T) {
	repository := newRegistryRouteRepository()
	repository.connection.SystemManaged = true
	engine := newRegistryRouteTestEngine(t, repository, &registryRouteAuthorizer{})

	for _, testCase := range []struct {
		name   string
		method string
		body   string
	}{
		{name: "update", method: http.MethodPut, body: `{"display_name":"Primary","endpoint":"https://registry.example","enabled":true,"insecure":false}`},
		{name: "delete", method: http.MethodDelete},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, registryRouteRequest(testCase.method, "/api/registries/registry:primary", testCase.body, "7"))
			if response.Code != http.StatusConflict {
				t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
			}
		})
	}
}

type registryRouteAuthService struct{}

func (registryRouteAuthService) CurrentUser(ctx context.Context) (*moduleapi.CurrentUser, error) {
	auth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || auth.Claims == nil || auth.Claims.UserID == 0 {
		return nil, errors.New("route test claims are unavailable")
	}
	return &moduleapi.CurrentUser{ID: auth.Claims.UserID, Username: "registry-user", DisplayName: "Registry User"}, nil
}

func (registryRouteAuthService) ParseAccessToken(_ context.Context, token string) (*moduleapi.AccessTokenClaims, error) {
	userID, err := strconv.ParseUint(strings.TrimPrefix(token, "user-"), 10, 64)
	if err != nil || userID == 0 || token != "user-"+strconv.FormatUint(userID, 10) {
		return nil, errors.New("invalid route test token")
	}
	return &moduleapi.AccessTokenClaims{UserID: userID, SessionID: "registry-route-session", ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil
}

type registryRouteAuthorizer struct{ permissions []string }

func (a *registryRouteAuthorizer) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, permission string) error {
	a.permissions = append(a.permissions, permission)
	return nil
}

func (a *registryRouteAuthorizer) reset() { a.permissions = nil }

func newRegistryRouteTestEngine(t *testing.T, repository *registryRouteRepository, authorizer *registryRouteAuthorizer) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	services := containerdi.New()
	if err := services.RegisterSingleton((*moduleapi.AuthService)(nil), func(containerdi.Resolver) (any, error) { return registryRouteAuthService{}, nil }); err != nil {
		t.Fatal(err)
	}
	if err := services.RegisterSingleton((*moduleapi.Authorizer)(nil), func(containerdi.Resolver) (any, error) { return authorizer, nil }); err != nil {
		t.Fatal(err)
	}
	ctx := &module.Context{Router: engine.Group("/api"), Services: services, EventBus: eventbus.New(zap.NewNop()), I18n: i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}})}
	if err := registerRegistryRoutes(ctx, NewService(repository)); err != nil {
		t.Fatal(err)
	}
	return engine
}

func registryRouteRequest(method, path, body, userID string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer user-"+userID)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

type registryRouteRepository struct {
	connection          registrystore.Connection
	repositories        []registrystore.Repository
	assignments         map[string][]registrystore.UserAssignment
	destinationsByActor map[uint64][]registrystore.Destination
	availableActor      uint64
}

func newRegistryRouteRepository() *registryRouteRepository {
	now := time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC)
	return &registryRouteRepository{
		connection:          registrystore.Connection{ConnectionRef: "registry:primary", DisplayName: "Primary", Provider: registryProviderGenericOCI, Endpoint: "https://registry.example", Enabled: true, Availability: true, VerificationStatus: verificationSucceeded, CreatedAt: now, UpdatedAt: now},
		repositories:        []registrystore.Repository{{ConnectionRef: "registry:primary", RepositoryRef: "app", DisplayName: "Application", AllowPull: true, AllowPush: true, CreatedAt: now, UpdatedAt: now}},
		assignments:         map[string][]registrystore.UserAssignment{},
		destinationsByActor: make(map[uint64][]registrystore.Destination),
	}
}

func (r *registryRouteRepository) ResolveAuthorizedRepository(context.Context, uint64, string, string) (registrystore.AuthorizedRepository, error) {
	return registrystore.AuthorizedRepository{}, registrystore.ErrNotFound
}

func (r *registryRouteRepository) ResolveAuthorizedCopySource(context.Context, uint64, string, string) (registrystore.AuthorizedRepository, error) {
	return registrystore.AuthorizedRepository{}, registrystore.ErrNotFound
}

func (r *registryRouteRepository) ResolveRepositoryBinding(context.Context, string, string) (registrystore.AuthorizedRepository, error) {
	return registrystore.AuthorizedRepository{}, registrystore.ErrNotFound
}

func (r *registryRouteRepository) ListConnections(context.Context, string, int, int) ([]registrystore.Connection, int, error) {
	return []registrystore.Connection{r.connection}, 1, nil
}

func (r *registryRouteRepository) GetConnection(_ context.Context, connectionRef string) (registrystore.Connection, error) {
	if connectionRef != r.connection.ConnectionRef {
		return registrystore.Connection{}, registrystore.ErrNotFound
	}
	return r.connection, nil
}

func (r *registryRouteRepository) CreateConnection(_ context.Context, input registrystore.ConnectionInput, _ uint64) (registrystore.Connection, error) {
	r.connection = registrystore.Connection{ConnectionRef: input.ConnectionRef, DisplayName: input.DisplayName, Provider: input.Provider, Endpoint: input.Endpoint, CredentialRef: input.CredentialRef, Enabled: input.Enabled, Insecure: input.Insecure, Description: input.Description, AuthMode: input.AuthMode, CreatedAt: r.connection.CreatedAt, UpdatedAt: r.connection.UpdatedAt}
	return r.connection, nil
}

func (r *registryRouteRepository) UpdateConnection(ctx context.Context, connectionRef string, input registrystore.ConnectionInput, _ uint64) (registrystore.Connection, error) {
	if connectionRef != r.connection.ConnectionRef {
		return registrystore.Connection{}, registrystore.ErrNotFound
	}
	return r.CreateConnection(ctx, input, 0)
}

func (r *registryRouteRepository) DeleteConnection(_ context.Context, connectionRef string, _ uint64) error {
	if connectionRef != r.connection.ConnectionRef {
		return registrystore.ErrNotFound
	}
	return nil
}

func (r *registryRouteRepository) SetVerification(_ context.Context, connectionRef string, available bool, status, errorCode string) (registrystore.Connection, error) {
	if connectionRef != r.connection.ConnectionRef {
		return registrystore.Connection{}, registrystore.ErrNotFound
	}
	r.connection.Availability, r.connection.VerificationStatus, r.connection.LastVerificationErrorCode = available, status, errorCode
	return r.connection, nil
}

func (r *registryRouteRepository) ListRepositories(_ context.Context, connectionRef string, limit, offset int) ([]registrystore.Repository, int, error) {
	if connectionRef != r.connection.ConnectionRef {
		return nil, 0, registrystore.ErrNotFound
	}
	return page(r.repositories, limit, offset), len(r.repositories), nil
}

func (r *registryRouteRepository) CreateRepository(_ context.Context, connectionRef string, input registrystore.RepositoryInput, _ uint64) (registrystore.Repository, error) {
	if connectionRef != r.connection.ConnectionRef {
		return registrystore.Repository{}, registrystore.ErrNotFound
	}
	item := registrystore.Repository{ConnectionRef: connectionRef, RepositoryRef: input.RepositoryRef, DisplayName: input.DisplayName, AllowPull: input.AllowPull, AllowPush: input.AllowPush, CreatedAt: r.connection.CreatedAt, UpdatedAt: r.connection.UpdatedAt}
	r.repositories = append(r.repositories, item)
	return item, nil
}

func (r *registryRouteRepository) UpdateRepository(_ context.Context, connectionRef, repositoryRef string, input registrystore.RepositoryInput, _ uint64) (registrystore.Repository, error) {
	for index := range r.repositories {
		if r.repositories[index].ConnectionRef == connectionRef && r.repositories[index].RepositoryRef == repositoryRef {
			r.repositories[index].DisplayName, r.repositories[index].AllowPull, r.repositories[index].AllowPush = input.DisplayName, input.AllowPull, input.AllowPush
			return r.repositories[index], nil
		}
	}
	return registrystore.Repository{}, registrystore.ErrNotFound
}

func (r *registryRouteRepository) DeleteRepository(_ context.Context, connectionRef, repositoryRef string, _ uint64) error {
	for index := range r.repositories {
		if r.repositories[index].ConnectionRef == connectionRef && r.repositories[index].RepositoryRef == repositoryRef {
			r.repositories = append(r.repositories[:index], r.repositories[index+1:]...)
			return nil
		}
	}
	return registrystore.ErrNotFound
}

func (r *registryRouteRepository) ListAssignments(_ context.Context, connectionRef, repositoryRef string, limit, offset int) ([]registrystore.UserAssignment, int, error) {
	if !r.repositoryExists(connectionRef, repositoryRef) {
		return nil, 0, registrystore.ErrNotFound
	}
	items := r.assignments[connectionRef+"/"+repositoryRef]
	return page(items, limit, offset), len(items), nil
}

func (r *registryRouteRepository) ReplaceAssignments(_ context.Context, connectionRef, repositoryRef string, userIDs []uint64, actorID uint64) ([]registrystore.UserAssignment, error) {
	if !r.repositoryExists(connectionRef, repositoryRef) {
		return nil, registrystore.ErrNotFound
	}
	items := make([]registrystore.UserAssignment, 0, len(userIDs))
	for _, userID := range userIDs {
		items = append(items, registrystore.UserAssignment{ConnectionRef: connectionRef, RepositoryRef: repositoryRef, UserID: userID, CreatedAt: r.connection.CreatedAt, CreatedBy: actorID})
	}
	r.assignments[connectionRef+"/"+repositoryRef] = items
	return append([]registrystore.UserAssignment(nil), items...), nil
}

func (r *registryRouteRepository) GrantAssignment(_ context.Context, connectionRef, repositoryRef string, userID, actorID uint64) (registrystore.UserAssignment, error) {
	if !r.repositoryExists(connectionRef, repositoryRef) || userID == 0 {
		return registrystore.UserAssignment{}, registrystore.ErrNotFound
	}
	key := connectionRef + "/" + repositoryRef
	for _, item := range r.assignments[key] {
		if item.UserID == userID {
			return item, nil
		}
	}
	item := registrystore.UserAssignment{ConnectionRef: connectionRef, RepositoryRef: repositoryRef, UserID: userID, CreatedAt: r.connection.CreatedAt, CreatedBy: actorID}
	r.assignments[key] = append(r.assignments[key], item)
	return item, nil
}

func (r *registryRouteRepository) RevokeAssignment(_ context.Context, connectionRef, repositoryRef string, userID, _ uint64) error {
	if !r.repositoryExists(connectionRef, repositoryRef) {
		return registrystore.ErrNotFound
	}
	key := connectionRef + "/" + repositoryRef
	for index, item := range r.assignments[key] {
		if item.UserID == userID {
			r.assignments[key] = append(r.assignments[key][:index], r.assignments[key][index+1:]...)
			return nil
		}
	}
	return registrystore.ErrNotFound
}

func (r *registryRouteRepository) ListAvailableDestinations(_ context.Context, actorID uint64, limit, offset int) ([]registrystore.Destination, int, error) {
	r.availableActor = actorID
	items := r.destinationsByActor[actorID]
	return page(items, limit, offset), len(items), nil
}

func page[T any](items []T, limit, offset int) []T {
	if offset >= len(items) {
		return []T{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]T(nil), items[offset:end]...)
}

func (r *registryRouteRepository) repositoryExists(connectionRef, repositoryRef string) bool {
	for _, item := range r.repositories {
		if item.ConnectionRef == connectionRef && item.RepositoryRef == repositoryRef {
			return true
		}
	}
	return false
}
