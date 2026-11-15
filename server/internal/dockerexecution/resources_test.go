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
