package container

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

type dockerImageBuildRuntime struct {
	fakeRuntime
	image       DockerImage
	requestedID string
}

func (r *dockerImageBuildRuntime) ListDockerImages(context.Context) (DockerImageListResult, error) {
	return DockerImageListResult{}, nil
}

func (r *dockerImageBuildRuntime) ReadDockerImage(_ context.Context, id string) (DockerImage, error) {
	r.requestedID = id
	return r.image, nil
}

func (r *dockerImageBuildRuntime) ListDockerNetworks(context.Context) ([]DockerNetwork, error) {
	return nil, nil
}

func (r *dockerImageBuildRuntime) ReadDockerNetwork(context.Context, string) (DockerNetwork, error) {
	return DockerNetwork{}, nil
}

func (r *dockerImageBuildRuntime) ListDockerVolumes(context.Context) ([]DockerVolume, error) {
	return nil, nil
}

func (r *dockerImageBuildRuntime) ReadDockerVolume(context.Context, string) (DockerVolume, error) {
	return DockerVolume{}, nil
}

func TestBuildImageStreamsOutputAndPopulatesDockerFacts(t *testing.T) {
	workspace, dockerBinary := writeDockerBuildScript(t, `
printf 'build started\n'
while [ ! -f "$PWD/allow-finish" ]; do sleep 0.01; done
printf 'build completed\n' >&2
printf 'sha256:built-image' > "$iid_file"
`)
	runtime := &dockerImageBuildRuntime{image: DockerImage{
		ID:                "sha256:built-image",
		RepositoryDigests: []string{"example/other@sha256:other", "example/app@sha256:expected"},
		SizeBytes:         1234,
		OperatingSystem:   "linux",
		Architecture:      "arm64",
		Variant:           "v8",
	}}
	service, err := newTestService(containerServiceOptions{runtime: runtime, enabled: true})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	t.Setenv("PATH", filepath.Dir(dockerBinary)+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var entries []moduleapi.TaskLogEntry
	result, err := service.BuildImage(ctx, testDockerBuildInput(workspace), func(_ context.Context, entry moduleapi.TaskLogEntry) error {
		entries = append(entries, entry)
		if entry.Line == "build started" {
			return os.WriteFile(filepath.Join(workspace, "allow-finish"), []byte("ok"), 0o600)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("build image: %v", err)
	}
	if !slices.ContainsFunc(entries, func(entry moduleapi.TaskLogEntry) bool {
		return entry.Stream == "stdout" && entry.Line == "build started"
	}) || !slices.ContainsFunc(entries, func(entry moduleapi.TaskLogEntry) bool {
		return entry.Stream == "stderr" && entry.Line == "build completed"
	}) {
		t.Fatalf("expected streamed stdout and stderr entries, got %#v", entries)
	}
	if runtime.requestedID != "sha256:built-image" {
		t.Fatalf("inspected image id = %q", runtime.requestedID)
	}
	want := moduleapi.DockerImageBuildResult{ImageID: "sha256:built-image", Digest: "example/app@sha256:expected", Repository: "example/app", Tag: "v1", SizeBytes: 1234, OS: "linux", Architecture: "arm64", Variant: "v8"}
	if result != want {
		t.Fatalf("result = %#v, want %#v", result, want)
	}
}

func TestBuildImagePreservesDockerCommandError(t *testing.T) {
	workspace, dockerBinary := writeDockerBuildScript(t, `
printf 'build failed\n' >&2
exit 7
`)
	service, err := newTestService(containerServiceOptions{runtime: &dockerImageBuildRuntime{}, enabled: true})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	t.Setenv("PATH", filepath.Dir(dockerBinary)+string(os.PathListSeparator)+os.Getenv("PATH"))
	_, err = service.BuildImage(context.Background(), testDockerBuildInput(workspace), nil)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected wrapped exec.ExitError, got %v", err)
	}
	if exitErr.ExitCode() != 7 {
		t.Fatalf("exit code = %d, want 7", exitErr.ExitCode())
	}
}

func TestBuildImageReturnsSinkError(t *testing.T) {
	workspace, dockerBinary := writeDockerBuildScript(t, "printf 'build started\\n'\n")
	service, err := newTestService(containerServiceOptions{runtime: &dockerImageBuildRuntime{}, enabled: true})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	t.Setenv("PATH", filepath.Dir(dockerBinary)+string(os.PathListSeparator)+os.Getenv("PATH"))
	sinkErr := errors.New("append build log")
	_, err = service.BuildImage(context.Background(), testDockerBuildInput(workspace), func(context.Context, moduleapi.TaskLogEntry) error { return sinkErr })
	if !errors.Is(err, sinkErr) {
		t.Fatalf("expected sink error, got %v", err)
	}
}

func testDockerBuildInput(workspace string) moduleapi.DockerImageBuildInput {
	return moduleapi.DockerImageBuildInput{WorkspaceRoot: workspace, ContextPath: "context", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1"}
}

func writeDockerBuildScript(t *testing.T, body string) (string, string) {
	t.Helper()
	workspace := t.TempDir()
	binDir := t.TempDir()
	script := "#!/bin/sh\nset -eu\niid_file=''\nwhile [ $# -gt 0 ]; do\n  case \"$1\" in\n    --iidfile) iid_file=\"$2\"; shift 2 ;;\n    *) shift ;;\n  esac\ndone\n" + body
	dockerBinary := filepath.Join(binDir, "docker")
	if err := os.WriteFile(dockerBinary, []byte(script), 0o600); err != nil {
		t.Fatalf("write docker script: %v", err)
	}
	// #nosec G302 -- test fixture must be executable after its private initial write.
	if err := os.Chmod(dockerBinary, 0o700); err != nil {
		t.Fatalf("make docker script executable: %v", err)
	}
	return workspace, dockerBinary
}

var _ DockerResourceReader = (*dockerImageBuildRuntime)(nil)
