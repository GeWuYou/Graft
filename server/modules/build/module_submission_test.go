package build

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	containerdi "graft/server/internal/container"
	openapigen "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/cronx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
)

func TestRegisteredBuildModuleSubmitsUploadedSnapshot(t *testing.T) {
	repository, tasks := &recordingBuildRepository{}, &recordingBuildTasks{}
	engine := newRegisteredBuildSubmissionEngine(t, repository, tasks)
	snapshotID := uploadRegisteredBuildSnapshot(t, engine)
	t.Cleanup(func() {
		_ = releaseMaterialization(context.Background(), repository.inputSnapshots.Items[0].MaterializationRef)
	})
	payload, err := json.Marshal(map[string]any{
		"input_snapshot_id": snapshotID, "runtime_target_id": 4, "template_ref": v2DockerfileTemplate, "driver": v2DockerEngineDriver,
		"destination": map[string]string{"kind": v2OCIDestination, "connection_ref": "registry:primary", "repository_ref": "team/app", "reference": "v1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, buildAuthorizedRequest(http.MethodPost, "/api/build/jobs", string(payload)))
	if response.Code != http.StatusAccepted {
		t.Fatalf("submit uploaded snapshot status = %d, body=%s", response.Code, response.Body.String())
	}
	if tasks.beginCalls != 1 || tasks.materializeCalls != 1 || repository.v2Plan.Workspace.ID != snapshotID {
		t.Fatalf("uploaded snapshot was not materialized through Task submission: begin=%d materialize=%d snapshot=%s", tasks.beginCalls, tasks.materializeCalls, repository.v2Plan.Workspace.ID)
	}
}

func newRegisteredBuildSubmissionEngine(t *testing.T, repository *recordingBuildRepository, tasks *recordingBuildTasks) *gin.Engine {
	t.Helper()
	services := containerdi.New()
	target := moduleapi.BuildRuntimeTargetSummary{ID: 4, Available: true, ProviderCapabilityProfile: "oci-build", ProviderCapabilityVersion: "docker/v1", SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}, BuildFeatures: []string{"registry-login"}}
	for key, value := range map[any]any{
		(*moduleapi.AuthService)(nil):                          buildRouteAuthService{},
		(*moduleapi.Authorizer)(nil):                           buildRouteAuthorizer{},
		(*moduleapi.TaskSubmissionService)(nil):                tasks,
		(*moduleapi.TaskBatchQueryService)(nil):                tasks,
		(*moduleapi.TaskRuntimeRegistrar)(nil):                 testBuildRegistrar{},
		(*moduleapi.ApplicationWorkspaceSnapshotResolver)(nil): v2SnapshotResolver{},
		(*moduleapi.BuildRuntimeTargetReader)(nil):             v2TargetReader{target: target},
		(*moduleapi.RuntimeTargetBuildAssignmentReader)(nil):   v2TargetAssignments{allowed: true},
		(*moduleapi.RegistryDestinationResolver)(nil):          v2RegistryResolver{},
	} {
		if err := services.RegisterSingleton(key, func(containerdi.Resolver) (any, error) { return value, nil }); err != nil {
			t.Fatal(err)
		}
	}
	engine := gin.New()
	ctx := &module.Context{Services: services, Router: engine.Group("/api"), MenuRegistry: menu.NewRegistry(), PermissionRegistry: permission.NewRegistry(), CronRegistry: cronx.NewRegistry(), I18n: i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}})}
	if err := NewModule(repository).Register(ctx); err != nil {
		t.Fatalf("register Build module: %v", err)
	}
	return engine
}

func uploadRegisteredBuildSnapshot(t *testing.T, engine *gin.Engine) string {
	t.Helper()
	archive := &bytes.Buffer{}
	writer := tar.NewWriter(archive)
	if err := writer.WriteHeader(&tar.Header{Name: "Dockerfile", Mode: 0o644, Size: int64(len("FROM scratch\n"))}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("FROM scratch\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	body, contentType := buildSnapshotMultipart(t, archive.Bytes())
	request := httptest.NewRequest(http.MethodPost, "/api/build/input-snapshots", body)
	request.Header.Set("Authorization", "Bearer route-test-token")
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload snapshot status = %d, body=%s", response.Code, response.Body.String())
	}
	var envelope openapigen.EnvelopedBuildInputSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.SnapshotId == "" {
		t.Fatalf("upload snapshot identity is unavailable: %s", response.Body.String())
	}
	return envelope.Data.SnapshotId
}
