package build

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
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
	buildstore "graft/server/modules/build/store"
)

func buildSnapshotMultipart(t *testing.T, archive []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("archive", "input.tar")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(archive); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body, writer.FormDataContentType()
}

func TestBuildInputSnapshotUploadRoute(t *testing.T) {
	archive := &bytes.Buffer{}
	tarWriter := tar.NewWriter(archive)
	if err := tarWriter.WriteHeader(&tar.Header{Name: "Dockerfile", Mode: 0o644, Size: int64(len("FROM alpine\n"))}); err != nil {
		t.Fatal(err)
	}
	_, _ = tarWriter.Write([]byte("FROM alpine\n"))
	_ = tarWriter.Close()
	repository := &recordingBuildRepository{}
	engine := newBuildRouteTestEngine(t, &recordingBuildTasks{}, repository)
	body, contentType := buildSnapshotMultipart(t, archive.Bytes())
	request := httptest.NewRequest(http.MethodPost, "/api/build/input-snapshots", body)
	request.Header.Set("Authorization", "Bearer route-test-token")
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected successful snapshot upload, got %d: %s", response.Code, response.Body.String())
	}
	invalidBody, invalidType := buildSnapshotMultipart(t, []byte("not an archive"))
	invalidRequest := httptest.NewRequest(http.MethodPost, "/api/build/input-snapshots", invalidBody)
	invalidRequest.Header.Set("Authorization", "Bearer route-test-token")
	invalidRequest.Header.Set("Content-Type", invalidType)
	invalidResponse := httptest.NewRecorder()
	engine.ServeHTTP(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid snapshot upload to return 400, got %d", invalidResponse.Code)
	}
}

//nolint:gocyclo,cyclop // 表驱动式请求绑定回归同时覆盖分页、快照、执行和时间筛选。
func TestBuildListQueryBindsBuildOwnedHistoryFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	request := httptest.NewRequest("GET", "/api/build/jobs?limit=50&offset=100&search=release&image_repository=example%2Fapp&image_tag=v1&build_status=running&builder_id=4&created_after=2026-08-01T00%3A00%3A00Z&created_before=2026-08-02T00%3A00%3A00Z", nil)
	context.Request = request

	query, ok := buildListQuery(context)
	if !ok {
		t.Fatal("expected valid Build history query")
	}
	if query.Limit != 50 || query.Offset != 100 || query.ApplicationID != nil {
		t.Fatalf("unexpected pagination or application filter: %#v", query)
	}
	if query.ImageRepository == nil || *query.ImageRepository != "example/app" || query.ImageTag == nil || *query.ImageTag != "v1" {
		t.Fatalf("unexpected image filters: %#v", query)
	}
	if query.Search == nil || *query.Search != "release" || query.BuildStatus == nil || *query.BuildStatus != buildstore.StatusFilterRunning || query.BuilderID == nil || *query.BuilderID != 4 {
		t.Fatalf("unexpected execution filters: %#v", query)
	}
	if query.CreatedAfter == nil || query.CreatedBefore == nil || !query.CreatedAfter.Before(*query.CreatedBefore) {
		t.Fatalf("unexpected creation range: %#v", query)
	}
}

func TestBuildListQueryRejectsInvalidHistoryRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/api/build/jobs?created_after=2026-08-02T00%3A00%3A00Z&created_before=2026-08-01T00%3A00%3A00Z", nil)

	if _, ok := buildListQuery(context); ok {
		t.Fatal("expected reverse creation range to be rejected")
	}
}

type buildRouteAuthService struct{}

func (buildRouteAuthService) CurrentUser(context.Context) (*moduleapi.CurrentUser, error) {
	return &moduleapi.CurrentUser{ID: 7, Username: "admin", DisplayName: "Admin"}, nil
}

func (buildRouteAuthService) ParseAccessToken(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
	return &moduleapi.AccessTokenClaims{UserID: 7, SessionID: "session-1", ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil
}

type buildRouteAuthorizer struct{}

func (buildRouteAuthorizer) Authorize(context.Context, moduleapi.RequestAuthContext, string) error {
	return nil
}

func newBuildRouteTestEngine(t *testing.T, tasks *recordingBuildTasks, repository *recordingBuildRepository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	services := containerdi.New()
	if err := services.RegisterSingleton((*moduleapi.AuthService)(nil), func(containerdi.Resolver) (any, error) { return buildRouteAuthService{}, nil }); err != nil {
		t.Fatal(err)
	}
	if err := services.RegisterSingleton((*moduleapi.Authorizer)(nil), func(containerdi.Resolver) (any, error) { return buildRouteAuthorizer{}, nil }); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(&recordingBuildContexts{}, tasks, tasks, repository)
	if err != nil {
		t.Fatal(err)
	}
	target := moduleapi.BuildRuntimeTargetSummary{ID: 4, DisplayName: "Local Docker", Available: true, SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}}
	service.ConfigureV2Submission(v2SnapshotResolver{snapshot: moduleapi.WorkspaceSnapshot{ID: "snapshot", SourceKind: "application_workspace", SourceReference: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContentDigest: "sha256:test", MaterializedRoot: "/workspace/app"}}, v2TargetReader{target: target}, v2TargetAssignments{allowed: true, targets: []moduleapi.BuildRuntimeTargetSummary{target}}, v2RegistryResolver{})
	service.ConfigureArtifactPromotion(tasks, &promotionRegistryStub{}, v2TargetAssignments{allowed: true})
	ctx := &module.Context{Router: engine.Group("/api"), Services: services, EventBus: eventbus.New(zap.NewNop()), I18n: i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}})}
	if err := registerRoutes(ctx, service); err != nil {
		t.Fatal(err)
	}
	return engine
}

func buildAuthorizedRequest(method, path, body string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer route-test-token")
	request.Header.Set("Idempotency-Key", "route-test-key")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func TestBuildRoutesUseModuleRouterPrefixExactlyOnce(t *testing.T) {
	engine := newBuildRouteTestEngine(t, &recordingBuildTasks{}, &recordingBuildRepository{})

	canonical := httptest.NewRecorder()
	engine.ServeHTTP(canonical, httptest.NewRequest(http.MethodGet, "/api/build/jobs", nil))
	if canonical.Code != http.StatusUnauthorized {
		t.Fatalf("canonical route status = %d, body=%s", canonical.Code, canonical.Body.String())
	}

	duplicated := httptest.NewRecorder()
	engine.ServeHTTP(duplicated, httptest.NewRequest(http.MethodGet, "/api/api/build/jobs", nil))
	if duplicated.Code != http.StatusNotFound {
		t.Fatalf("duplicated API prefix status = %d, body=%s", duplicated.Code, duplicated.Body.String())
	}
}

func TestBuildRoutesRejectInvalidListQuery(t *testing.T) {
	engine := newBuildRouteTestEngine(t, &recordingBuildTasks{}, &recordingBuildRepository{})
	for _, path := range []string{"/api/build/jobs?limit=0"} {
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, buildAuthorizedRequest(http.MethodGet, path, ""))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("path = %s, status = %d, body=%s", path, response.Code, response.Body.String())
		}
	}
}

func TestBuildRoutesExposeBuildSelectorReadModels(t *testing.T) {
	repository := &recordingBuildRepository{}
	engine := newBuildRouteTestEngine(t, &recordingBuildTasks{}, repository)

	targetResponse := httptest.NewRecorder()
	engine.ServeHTTP(targetResponse, buildAuthorizedRequest(http.MethodGet, "/api/build/runtime-targets", ""))
	if targetResponse.Code != http.StatusOK || !strings.Contains(targetResponse.Body.String(), "Local Docker") {
		t.Fatalf("unexpected runtime target selector response: status=%d body=%s", targetResponse.Code, targetResponse.Body.String())
	}
}

func TestBuildRoutesExposeImmutableArtifactReadModel(t *testing.T) {
	repository := &recordingBuildRepository{artifactResult: buildstore.V2ArtifactListResult{Items: []buildstore.V2ArtifactProjection{{ArtifactID: "artifact_test", Digest: "sha256:deadbeef", MediaType: "application/vnd.oci.image.manifest.v1+json", Platforms: []string{"linux/amd64"}, SizeBytes: 123, CreatedAt: time.Date(2026, time.August, 6, 0, 0, 0, 0, time.UTC)}}, Total: 1}}
	engine := newBuildRouteTestEngine(t, &recordingBuildTasks{}, repository)

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, buildAuthorizedRequest(http.MethodGet, "/api/build/artifacts?limit=10&offset=0", ""))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "sha256:deadbeef") || strings.Contains(response.Body.String(), "repository") {
		t.Fatalf("unexpected artifact response: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestBuildRoutesMapBadBuildIDAndInternalReadFailureSeparately(t *testing.T) {
	t.Run("bad build id", func(t *testing.T) {
		engine := newBuildRouteTestEngine(t, &recordingBuildTasks{}, &recordingBuildRepository{})
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, buildAuthorizedRequest(http.MethodGet, "/api/build/jobs/%20", ""))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
		}
	})
	t.Run("internal repository failure", func(t *testing.T) {
		engine := newBuildRouteTestEngine(t, &recordingBuildTasks{}, &recordingBuildRepository{getBuildIDErr: errors.New("database unavailable")})
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, buildAuthorizedRequest(http.MethodGet, "/api/build/jobs/build_test", ""))
		if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "common.internalError") {
			t.Fatalf("unexpected response: status=%d body=%s", response.Code, response.Body.String())
		}
	})
}

func TestBuildSubmitRouteUsesInternalErrorKeyForRuntimeFailure(t *testing.T) {
	engine := newBuildRouteTestEngine(t, &recordingBuildTasks{err: errors.New("task runtime unavailable")}, &recordingBuildRepository{})
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, buildAuthorizedRequest(http.MethodPost, "/api/build/jobs", `{"input_snapshot_id":"workspace_app","runtime_target_id":4,"template_ref":"oci-dockerfile/default@v1","driver":"docker-engine@v1","destination":{"kind":"oci_registry","connection_ref":"registry:default","repository_ref":"team/app","reference":"v1"}}`))
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "common.internalError") {
		t.Fatalf("unexpected response: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestBuildArtifactPromotionRouteSubmitsStableIdentifiersOnly(t *testing.T) {
	digest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	tasks := &recordingBuildTasks{}
	repository := &recordingBuildRepository{publicationSources: []moduleapi.ArtifactPublicationSource{{ArtifactID: "artifact_1", PublicationID: "publication_1", Digest: digest, MediaType: "application/vnd.oci.image.manifest.v1+json", DestinationKind: "oci_registry", ConnectionRef: "registry:source", RepositoryRef: "team/source"}}}
	engine := newBuildRouteTestEngine(t, tasks, repository)

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, buildAuthorizedRequest(http.MethodPost, "/api/build/artifact-promotions", `{"artifact_id":"artifact_1","publication_id":"publication_1","runtime_target_id":4,"destination":{"kind":"oci_registry","connection_ref":"registry:destination","repository_ref":"team/destination","reference":"stable"}}`))
	if response.Code != http.StatusAccepted || !strings.Contains(response.Body.String(), `"task_id":42`) {
		t.Fatalf("unexpected promotion response: status=%d body=%s", response.Code, response.Body.String())
	}
	if tasks.input.RequestedBy != 7 || tasks.input.IdempotencyKey != "route-test-key" || tasks.input.Owner.ID != "publication_1" {
		t.Fatalf("unexpected promotion task submission: %#v", tasks.input)
	}
	if strings.Contains(response.Body.String(), "endpoint") || strings.Contains(response.Body.String(), "credential") || strings.Contains(response.Body.String(), "binding") {
		t.Fatalf("promotion response leaked protected fields: %s", response.Body.String())
	}
}

func TestBuildArtifactPromotionRouteRejectsMalformedInput(t *testing.T) {
	for name, request := range map[string]*http.Request{
		"missing idempotency key": func() *http.Request {
			request := buildAuthorizedRequest(http.MethodPost, "/api/build/artifact-promotions", `{"artifact_id":"artifact_1","publication_id":"publication_1","runtime_target_id":4,"destination":{"kind":"oci_registry","connection_ref":"registry:destination","repository_ref":"team/destination","reference":"stable"}}`)
			request.Header.Del("Idempotency-Key")
			return request
		}(),
		"oversized idempotency key": func() *http.Request {
			request := buildAuthorizedRequest(http.MethodPost, "/api/build/artifact-promotions", `{"artifact_id":"artifact_1","publication_id":"publication_1","runtime_target_id":4,"destination":{"kind":"oci_registry","connection_ref":"registry:destination","repository_ref":"team/destination","reference":"stable"}}`)
			request.Header.Set("Idempotency-Key", strings.Repeat("a", moduleapi.TaskIdempotencyKeyMaxRunes+1))
			return request
		}(),
		"malformed body": buildAuthorizedRequest(http.MethodPost, "/api/build/artifact-promotions", `{`),
	} {
		t.Run(name, func(t *testing.T) {
			engine := newBuildRouteTestEngine(t, &recordingBuildTasks{}, &recordingBuildRepository{})
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestBuildArtifactPromotionRouteMapsTaskConflict(t *testing.T) {
	digest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	tasks := &recordingBuildTasks{err: moduleapi.ErrTaskSubmissionConflict}
	repository := &recordingBuildRepository{publicationSources: []moduleapi.ArtifactPublicationSource{{ArtifactID: "artifact_1", PublicationID: "publication_1", Digest: digest, MediaType: "application/vnd.oci.image.manifest.v1+json", DestinationKind: "oci_registry", ConnectionRef: "registry:source", RepositoryRef: "team/source"}}}
	engine := newBuildRouteTestEngine(t, tasks, repository)

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, buildAuthorizedRequest(http.MethodPost, "/api/build/artifact-promotions", `{"artifact_id":"artifact_1","publication_id":"publication_1","runtime_target_id":4,"destination":{"kind":"oci_registry","connection_ref":"registry:destination","repository_ref":"team/destination","reference":"stable"}}`))
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
}

func TestBuildPaginationUsesStoreBounds(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodGet, "/api/build/jobs", nil)
	query, ok := buildPaginationQuery(context)
	if !ok || query.Limit != buildstore.DefaultListLimit {
		t.Fatalf("default query = %#v, ok=%v", query, ok)
	}
}
