package agent

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const testBuildDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestBuildMaterialOperationAllowlistAndStrictShape(t *testing.T) {
	root := t.TempDir()
	contextRoot := filepath.Join(root, "context")
	if err := os.MkdirAll(contextRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contextRoot, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	contextMaterial := &buildContextMaterial{Root: root, ContextPath: "context", DockerfilePath: "context/Dockerfile", Repository: "team/app", Reference: "latest", Platform: "linux/amd64", BuildArgs: []buildArgument{{Name: "VERSION", Value: "1"}}}
	destination := &buildRegistryMaterial{Endpoint: "https://registry.example", Repository: "team/app", Reference: "stable", Username: "agent", Password: "secret"}
	source := &buildRegistrySourceMaterial{Endpoint: "https://source.example", Repository: "team/app", Digest: testBuildDigest, MediaType: ocispec.MediaTypeImageManifest, Username: "agent", Password: "secret"}
	artifacts := []buildPlatformArtifactMaterial{{Platform: "linux/amd64", Digest: testBuildDigest, MediaType: ocispec.MediaTypeImageManifest, SizeBytes: 42}}
	tests := []struct {
		name      string
		operation string
		material  buildExecutionMaterial
		valid     bool
	}{
		{name: "local image", operation: buildImageLocalOperation, material: buildExecutionMaterial{Context: contextMaterial}, valid: true},
		{name: "published image", operation: buildImagePublishOperation, material: buildExecutionMaterial{Context: contextMaterial, Destination: destination}, valid: true},
		{name: "manifest", operation: buildManifestOperation, material: buildExecutionMaterial{Destination: destination, PlatformArtifacts: artifacts}, valid: true},
		{name: "copy", operation: buildArtifactCopyOperation, material: buildExecutionMaterial{Destination: destination, Source: source}, valid: true},
		{name: "local rejects destination", operation: buildImageLocalOperation, material: buildExecutionMaterial{Context: contextMaterial, Destination: destination}},
		{name: "manifest rejects context", operation: buildManifestOperation, material: buildExecutionMaterial{Context: contextMaterial, Destination: destination, PlatformArtifacts: artifacts}},
		{name: "copy rejects platform artifacts", operation: buildArtifactCopyOperation, material: buildExecutionMaterial{Destination: destination, Source: source, PlatformArtifacts: artifacts}},
		{name: "unknown operation", operation: "build.shell.v1", material: buildExecutionMaterial{Context: contextMaterial}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validBuildMaterial(test.operation, test.material); got != test.valid {
				t.Fatalf("validBuildMaterial()=%v want=%v", got, test.valid)
			}
		})
	}
	if !validBuildLeaseInput(json.RawMessage(`{"build_id":"build-1"}`)) || validBuildLeaseInput(json.RawMessage(`{}`)) || validBuildLeaseInput(json.RawMessage(`[]`)) {
		t.Fatal("provider-neutral lease input object validation drifted")
	}
	var strict buildExecutionMaterial
	if err := strictDecode([]byte(`{"context":null,"command":"docker build ."}`), &strict); err == nil {
		t.Fatal("command-bearing material passed strict decoding")
	}
}

func TestBuildContextArchiveHonorsIgnoreAndKeepsDockerfile(t *testing.T) {
	root := t.TempDir()
	contextRoot := filepath.Join(root, "context")
	if err := os.MkdirAll(filepath.Join(contextRoot, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"Dockerfile":    "FROM scratch\n",
		".dockerignore": "secret.txt\nDockerfile\n",
		"secret.txt":    "must-not-leave-agent",
		"nested/app":    "payload",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(contextRoot, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := normalizeBuildPaths(buildContextMaterial{Root: root, ContextPath: "context", DockerfilePath: "context/Dockerfile"})
	if err != nil {
		t.Fatal(err)
	}
	archive, err := buildContextArchive(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = archive.Close() }()
	reader := tar.NewReader(archive)
	names := make([]string, 0, 4)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, strings.TrimPrefix(filepath.ToSlash(header.Name), "./"))
	}
	if slices.Contains(names, "secret.txt") || !slices.Contains(names, "Dockerfile") || !slices.Contains(names, ".dockerignore") || !slices.Contains(names, "nested/app") {
		t.Fatalf("archive entries=%v", names)
	}
}

func TestBuildPathsRejectDockerfileOutsideContext(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "context"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := normalizeBuildPaths(buildContextMaterial{Root: root, ContextPath: "context", DockerfilePath: "Dockerfile"})
	if err == nil {
		t.Fatal("Dockerfile outside context unexpectedly accepted")
	}
}

func TestBuildManifestDocumentUsesImmutablePlatformDigests(t *testing.T) {
	artifacts := []buildPlatformArtifactMaterial{{Platform: "linux/amd64", Digest: testBuildDigest, MediaType: ocispec.MediaTypeImageManifest, SizeBytes: 42}}
	payload, descriptor, err := buildManifestDocument(artifacts)
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.MediaType != ocispec.MediaTypeImageIndex || descriptor.Size != int64(len(payload)) || descriptor.Digest.String() == "" {
		t.Fatalf("descriptor=%#v payload=%s", descriptor, payload)
	}
	var index ocispec.Index
	if err := json.Unmarshal(payload, &index); err != nil {
		t.Fatal(err)
	}
	if len(index.Manifests) != 1 || index.Manifests[0].Digest.String() != testBuildDigest || index.Manifests[0].MediaType != ocispec.MediaTypeImageManifest || index.Manifests[0].Size != 42 || index.Manifests[0].Platform == nil || index.Manifests[0].Platform.OS != "linux" || index.Manifests[0].Platform.Architecture != "amd64" {
		t.Fatalf("index=%#v", index)
	}
}

func TestBuildRegistryMaterialRejectsEmbeddedCredentialsAndCommands(t *testing.T) {
	credentialEndpoint := (&url.URL{Scheme: "https", Host: "registry.example", User: url.UserPassword("user", "secret")}).String()
	if _, _, err := registryBase(credentialEndpoint); err == nil {
		t.Fatal("credential-bearing endpoint unexpectedly accepted")
	}
	if _, _, err := registryReference(buildRegistryMaterial{Endpoint: "https://registry.example", Repository: "team/app", Reference: "stable;docker push evil"}); err == nil {
		t.Fatal("command-like reference unexpectedly accepted")
	}
	result := buildExecutionResultPayload{Digest: testBuildDigest, Repository: "team/app", Reference: "stable", MediaType: ocispec.MediaTypeImageManifest, SizeBytes: 42}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"endpoint", "username", "password", "command", "root", "dockerfile"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("result leaked %q: %s", forbidden, payload)
		}
	}
}

//nolint:gocognit,gocyclo,cyclop // The conformance test deliberately exercises the full SDK request and normalized result seam.
func TestBuildImageLocalUsesMobySDKAndReturnsProviderNeutralResult(t *testing.T) {
	root := t.TempDir()
	contextRoot := filepath.Join(root, "context")
	if err := os.MkdirAll(contextRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contextRoot, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	buildObserved := false
	inspectObserved := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead && r.URL.Path == "/_ping":
			w.Header().Set("API-Version", "1.55")
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/build"):
			buildObserved = true
			if r.URL.Query().Get("dockerfile") != "Dockerfile" || !strings.Contains(r.URL.Query().Get("t"), "team/app:latest") || r.URL.Query().Get("buildargs") == "" {
				t.Fatalf("build query=%s", r.URL.RawQuery)
			}
			reader := tar.NewReader(r.Body)
			foundDockerfile := false
			for {
				header, err := reader.Next()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				foundDockerfile = foundDockerfile || strings.TrimPrefix(filepath.ToSlash(header.Name), "./") == "Dockerfile"
			}
			if !foundDockerfile {
				t.Fatal("Moby build context omitted Dockerfile")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"stream":"build complete"}`+"\n")
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/images/") && strings.HasSuffix(r.URL.Path, "/json"):
			inspectObserved = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"Id":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","Size":64,"Os":"linux","Architecture":"amd64"}`)
		default:
			t.Fatalf("unexpected Moby request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	material := buildExecutionMaterial{Context: &buildContextMaterial{
		Root: root, ContextPath: "context", DockerfilePath: "context/Dockerfile", Repository: "team/app", Reference: "latest", BuildArgs: []buildArgument{{Name: "VERSION", Value: "1"}},
	}}
	result, failureCode := executeBuildImage(t.Context(), config{DockerSocket: server.URL}, buildImageLocalOperation, material)
	if failureCode != "" {
		t.Fatalf("failureCode=%q", failureCode)
	}
	if !buildObserved || !inspectObserved || result.Repository != "team/app" || result.Reference != "latest" || result.ImageID == "" || result.SizeBytes != 64 || result.OS != "linux" || result.Architecture != "amd64" {
		t.Fatalf("buildObserved=%v inspectObserved=%v result=%#v", buildObserved, inspectObserved, result)
	}
}
