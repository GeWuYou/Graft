package dockerexecution

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestResourceResolutionFreezesCanonicalContainerAndComposeProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_ping") {
			w.Header().Set("API-Version", "1.55")
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/containers/alias/json") {
			t.Errorf("request=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Id":"canonical","Config":{"Labels":{"com.docker.compose.project":"managed"}}}`))
	}))
	defer server.Close()
	request := Request{Capability: "container_execution", Protocol: containerExecutionProtocol, OperationID: "container.lifecycle.stop.v1", Input: json.RawMessage(`{"container_ref":"alias"}`)}
	resolved, keys, err := ResolveResources(context.Background(), Environment{DockerSocket: server.URL}, request)
	if err != nil {
		t.Fatal(err)
	}
	var intent containerExecutionIntent
	if err := json.Unmarshal(resolved.Input, &intent); err != nil {
		t.Fatal(err)
	}
	if intent.ContainerRef != "canonical" || !slices.Contains(keys, "container:canonical") || !slices.Contains(keys, "compose:managed") {
		t.Fatalf("intent=%#v keys=%v", intent, keys)
	}
}

func TestImageResourceKeysNormalizeTagAliases(t *testing.T) {
	if imageResourceKey("nginx") != imageResourceKey("docker.io/library/nginx:latest") {
		t.Fatal("equivalent tag aliases escaped coordination")
	}
}

func TestImageUntagAcceptsSourceAliasAndRejectsReboundTag(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		targetID string
		wantCode string
	}{
		{"same-image", "sha256:source", ""},
		{"changed-tag", "sha256:changed", failureResourceConflict},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			removed := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/_ping") {
					w.Header().Set("API-Version", "1.55")
					return
				}
				switch {
				case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/images/source-alias/json"):
					_, _ = w.Write([]byte(`{"Id":"sha256:source"}`))
				case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/json"):
					_ = json.NewEncoder(w).Encode(map[string]string{"Id": testCase.targetID})
				case r.Method == http.MethodDelete:
					removed = true
					_, _ = w.Write([]byte(`[]`))
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()
			result := Execute(context.Background(), Environment{DockerSocket: server.URL}, Request{Capability: "container_execution", Protocol: containerExecutionProtocol, OperationID: "container.image.untag.v1", Input: json.RawMessage(`{"image_ref":"source-alias","tag_ref":"target:latest"}`)})
			if result.FailureCode != testCase.wantCode || removed != (testCase.wantCode == "") {
				t.Fatalf("result=%#v removed=%v", result, removed)
			}
		})
	}
}
