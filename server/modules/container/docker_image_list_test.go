package container

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/moby/moby/api/types/container"
)

type dockerImageListTestRuntime struct {
	fakeRuntime
	result DockerImageListResult
}

func (r dockerImageListTestRuntime) ListDockerImages(context.Context) (DockerImageListResult, error) {
	return r.result, nil
}

func (dockerImageListTestRuntime) ReadDockerImage(context.Context, string) (DockerImage, error) {
	return DockerImage{}, nil
}

func (dockerImageListTestRuntime) ListDockerNetworks(context.Context) ([]DockerNetwork, error) {
	return nil, nil
}

func (dockerImageListTestRuntime) ReadDockerNetwork(context.Context, string) (DockerNetwork, error) {
	return DockerNetwork{}, nil
}

func (dockerImageListTestRuntime) ListDockerVolumes(context.Context) ([]DockerVolume, error) {
	return nil, nil
}

func (dockerImageListTestRuntime) ReadDockerVolume(context.Context, string) (DockerVolume, error) {
	return DockerVolume{}, nil
}

func TestServiceDockerImagesFiltersPagesAndPreservesRuntimeSummary(t *testing.T) {
	runtime := dockerImageListTestRuntime{result: DockerImageListResult{
		Items: []DockerImage{
			{ID: "sha256:alpha", RepositoryTags: []string{"Example/App:Latest"}},
			{ID: "sha256:beta", RepositoryDigests: []string{"example/app@sha256:beta"}},
			{ID: "sha256:gamma", RepositoryTags: []string{"other/app:latest"}},
		},
		Summary: DockerImageListSummary{Total: 3, SizeBytes: 1234, InUse: 2, Dangling: 1},
	}}
	service, err := newRouteTestService(containerServiceOptions{runtime: runtime, enabled: true})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.DockerImages(context.Background(), DockerImageListQuery{Limit: 1, Offset: 1, Keyword: "EXAMPLE/APP"})
	if err != nil {
		t.Fatalf("list images: %v", err)
	}
	if result.Total != 2 || len(result.Items) != 1 || result.Items[0].ID != "sha256:beta" {
		t.Fatalf("unexpected filtered page: %#v", result)
	}
	if result.Summary != (DockerImageListSummary{Total: 3, SizeBytes: 1234, InUse: 2, Dangling: 1}) {
		t.Fatalf("expected complete runtime summary, got %#v", result.Summary)
	}
}

func TestFilterDockerImagesReturnsInputForEmptyKeyword(t *testing.T) {
	items := []DockerImage{{ID: "sha256:alpha"}}
	filtered := filterDockerImages(items, "  ")
	if len(filtered) != 1 || &filtered[0] != &items[0] {
		t.Fatalf("expected empty keyword to reuse input slice, got %#v", filtered)
	}
}

func TestDockerImageListRouteReturnsPaginationAndSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, engine := newRouteTestContext(&recordingAuthorizer{})
	service, err := newRouteTestService(containerServiceOptions{
		runtime: dockerImageListTestRuntime{result: DockerImageListResult{
			Items:   []DockerImage{{ID: "sha256:alpha", RepositoryTags: []string{"example/app:latest"}}},
			Summary: DockerImageListSummary{Total: 1, SizeBytes: 99, InUse: 1, Dangling: 0},
		}},
		enabled: true,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if err := registerRoutes(ctx, moduleID, service); err != nil {
		t.Fatalf("register routes: %v", err)
	}

	request := authorizedRequest(http.MethodGet, "/api/ops/docker/images?keyword=EXAMPLE&limit=20&offset=0")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{`"total":1`, `"limit":20`, `"offset":0`, `"size_bytes":99`, `"in_use":1`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %s in response: %s", expected, body)
		}
	}
}

func TestDockerImageListRouteBindsUnusedFilter(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantTotal string
	}{
		{name: "absent", wantTotal: `"total":2`},
		{name: "true", query: "?unused=true", wantTotal: `"total":1`},
		{name: "false", query: "?unused=false", wantTotal: `"total":2`},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			ctx, engine := newRouteTestContext(&recordingAuthorizer{})
			service, err := newRouteTestService(containerServiceOptions{
				runtime: dockerImageListTestRuntime{result: DockerImageListResult{Items: []DockerImage{
					{ID: "used", ContainerReferences: []DockerImageContainerReference{{ID: "container-1"}}},
					{ID: "unused"},
				}}},
				enabled: true,
			})
			if err != nil {
				t.Fatalf("new service: %v", err)
			}
			if err := registerRoutes(ctx, moduleID, service); err != nil {
				t.Fatalf("register routes: %v", err)
			}

			response := httptest.NewRecorder()
			engine.ServeHTTP(response, authorizedRequest(http.MethodGet, "/api/ops/docker/images"+testCase.query))
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), testCase.wantTotal) {
				t.Fatalf("expected 200 with %s, got %d: %s", testCase.wantTotal, response.Code, response.Body.String())
			}
		})
	}
}

func TestDockerImageListRouteRejectsInvalidUnusedFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, engine := newRouteTestContext(&recordingAuthorizer{})
	service, err := newRouteTestService(containerServiceOptions{runtime: dockerImageListTestRuntime{}, enabled: true})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if err := registerRoutes(ctx, moduleID, service); err != nil {
		t.Fatalf("register routes: %v", err)
	}

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedRequest(http.MethodGet, "/api/ops/docker/images?unused=invalid"))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "unused") {
		t.Fatalf("expected invalid unused query response, got %d: %s", response.Code, response.Body.String())
	}
}

func TestNormalizeDockerImageListQueryBounds(t *testing.T) {
	tests := []struct {
		name  string
		query DockerImageListQuery
		valid bool
		limit int
	}{
		{name: "defaults", valid: true, query: DockerImageListQuery{}, limit: defaultContainerListLimit},
		{name: "negative offset", query: DockerImageListQuery{Offset: -1}},
		{name: "over max", query: DockerImageListQuery{Limit: maxContainerListLimit + 1}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			query, err := normalizeDockerImageListQuery(testCase.query)
			if testCase.valid {
				if err != nil || query.Limit != testCase.limit {
					t.Fatalf("expected normalized query, got %#v, %v", query, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected invalid query, got %#v", query)
			}
		})
	}
}

func TestDockerImageReferencesAggregateContainerIDAndName(t *testing.T) {
	refs := dockerImageReferences([]container.Summary{{ID: "c1", ImageID: "sha256:image", Names: []string{"/web"}}, {ID: "c2", ImageID: "sha256:image", Names: []string{"/worker"}}})
	if len(refs["sha256:image"]) != 2 || refs["sha256:image"][0].Name != "web" || refs["sha256:image"][1].ID != "c2" {
		t.Fatalf("unexpected references: %#v", refs)
	}
}

func TestServiceDockerImagesFiltersUnusedByReferences(t *testing.T) {
	runtime := dockerImageListTestRuntime{result: DockerImageListResult{Items: []DockerImage{{ID: "used", ContainerReferences: []DockerImageContainerReference{{ID: "c1", Name: "web"}}}, {ID: "unused"}}}}
	service, err := newRouteTestService(containerServiceOptions{runtime: runtime, enabled: true})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	result, err := service.DockerImages(context.Background(), DockerImageListQuery{Limit: 20, Unused: true})
	if err != nil {
		t.Fatalf("list images: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != "unused" {
		t.Fatalf("unexpected unused images: %#v", result)
	}
}

func TestSortDockerImagesUsesCreatedTimeThenID(t *testing.T) {
	items := []DockerImage{
		{ID: "z", CreatedAt: "2026-07-18T10:00:00Z"},
		{ID: "a", CreatedAt: "2026-07-18T10:00:00Z"},
		{ID: "old", CreatedAt: "2026-07-17T10:00:00Z"},
	}
	sortDockerImages(items)
	if got := []string{items[0].ID, items[1].ID, items[2].ID}; strings.Join(got, ",") != "a,z,old" {
		t.Fatalf("unexpected stable image order: %v", got)
	}
	if imageCreatedAt(items[0]).Equal(time.Time{}) {
		t.Fatal("expected created timestamp to be parseable")
	}
}

func TestDockerImageIsDanglingTreatsPlaceholderTagsAsDangling(t *testing.T) {
	if !dockerImageIsDangling([]string{"<none>:<none>"}) {
		t.Fatal("expected placeholder tag to be dangling")
	}
	if dockerImageIsDangling([]string{"example/app:latest"}) {
		t.Fatal("expected tagged image not to be dangling")
	}
}
