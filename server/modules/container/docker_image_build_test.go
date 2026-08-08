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

type snapshotDeliveryBuildTargetReader struct {
	target moduleapi.BuildRuntimeTargetSummary
}

func (r snapshotDeliveryBuildTargetReader) ReadBuildTarget(context.Context, int64) (moduleapi.BuildRuntimeTargetSummary, error) {
	return r.target, nil
}

func TestDeliverWorkspaceSnapshotProvesManagedTargetLocalMaterialization(t *testing.T) {
	if err := os.MkdirAll(filepath.Join(os.TempDir(), "graft-build-snapshots"), 0o750); err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(filepath.Join(os.TempDir(), "graft-build-snapshots"), "snapshot-delivery-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	service, err := newTestService(containerServiceOptions{
		runtime:      &dockerImageBuildRuntime{},
		enabled:      true,
		buildTargets: snapshotDeliveryBuildTargetReader{target: moduleapi.BuildRuntimeTargetSummary{ID: 4, Provider: runtimeNameDocker, Available: true, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	reference, err := moduleapi.NewWorkspaceSnapshotMaterializationReference("snapshot-1", "sha256:source", root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := (containerImageBuilder{service: service}).DeliverWorkspaceSnapshot(context.Background(), moduleapi.WorkspaceSnapshotDeliveryRequest{TargetID: 4, SnapshotID: "snapshot-1", ContentDigest: "sha256:source", MaterializationRef: reference, DeliveryMode: moduleapi.SnapshotDeliveryModeTargetLocal})
	if err != nil {
		t.Fatal(err)
	}
	if result.TargetID != 4 || result.SnapshotID != "snapshot-1" || result.ContentDigest != "sha256:source" || result.DeliveryMode != moduleapi.SnapshotDeliveryModeTargetLocal {
		t.Fatalf("delivery proof = %#v", result)
	}
}

func TestDeliverWorkspaceSnapshotRejectsUnsupportedTransferAndUnmanagedRoot(t *testing.T) {
	service, err := newTestService(containerServiceOptions{
		runtime:      &dockerImageBuildRuntime{},
		enabled:      true,
		buildTargets: snapshotDeliveryBuildTargetReader{target: moduleapi.BuildRuntimeTargetSummary{ID: 4, Provider: runtimeNameDocker, Available: true, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	builder := containerImageBuilder{service: service}
	request := moduleapi.WorkspaceSnapshotDeliveryRequest{TargetID: 4, SnapshotID: "snapshot-1", ContentDigest: "sha256:source", MaterializationRef: "build-snapshot:outside", DeliveryMode: moduleapi.SnapshotDeliveryModeProviderTransfer}
	if _, err := builder.DeliverWorkspaceSnapshot(context.Background(), request); err == nil {
		t.Fatal("provider-transfer unexpectedly succeeded without a provider adapter")
	}
	request.DeliveryMode = moduleapi.SnapshotDeliveryModeTargetLocal
	if _, err := builder.DeliverWorkspaceSnapshot(context.Background(), request); err == nil {
		t.Fatal("unmanaged snapshot root unexpectedly succeeded")
	}
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

func TestManifestSourceReferencesUsesDigestAndRejectsDuplicatePlatform(t *testing.T) {
	digestA := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestB := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	sources, err := manifestSourceReferences("registry.example:5000/project/app:v1", []moduleapi.PlatformArtifact{{Platform: "linux/amd64", Digest: digestA}, {Platform: "linux/arm64", Digest: digestB}})
	if err != nil {
		t.Fatalf("manifestSourceReferences: %v", err)
	}
	if !slices.Equal(sources, []string{"registry.example:5000/project/app@" + digestA, "registry.example:5000/project/app@" + digestB}) {
		t.Fatalf("sources = %#v", sources)
	}
	_, err = manifestSourceReferences("registry.example/project/app:v1", []moduleapi.PlatformArtifact{{Platform: "linux/amd64", Digest: digestA}, {Platform: "linux/amd64", Digest: digestB}})
	if err == nil {
		t.Fatal("duplicate platform manifest sources succeeded")
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

func TestDockerBuildLogWriterDropsNilSinkAndBoundsLongLines(t *testing.T) {
	writer := newDockerBuildLogSink(context.Background(), nil).writer("stdout")
	if _, err := writer.Write([]byte("quiet output")); err != nil {
		t.Fatal(err)
	}
	owner := newDockerBuildLogSink(context.Background(), func(context.Context, moduleapi.TaskLogEntry) error { return nil })
	if _, err := owner.writer("stdout").Write(make([]byte, maxDockerBuildLogBuffer+1)); err == nil {
		t.Fatal("expected oversized log line error")
	}
}

func TestPublicationReferenceUsesProviderEndpointAndRejectsInvalidEndpoint(t *testing.T) {
	reference, err := publicationReference(moduleapi.RegistryPublicationBinding{Endpoint: "https://registry.example/v2", Destination: moduleapi.AuthorizedArtifactDestination{RepositoryRef: "team/api", Reference: "v1"}})
	if err != nil || reference != "registry.example/v2/team/api:v1" {
		t.Fatalf("reference = %q, err = %v", reference, err)
	}
	if _, err := publicationReference(moduleapi.RegistryPublicationBinding{Endpoint: "unix:///var/run/docker.sock", Destination: moduleapi.AuthorizedArtifactDestination{RepositoryRef: "team/api", Reference: "v1"}}); err == nil {
		t.Fatal("expected non-HTTP registry endpoint to be rejected")
	}
}

func TestPublishImageOnTargetRejectsUnsupportedCredentialExecutionMode(t *testing.T) {
	builder := containerImageBuilder{service: &service{}}
	_, err := builder.PublishImageOnTarget(context.Background(), 1, moduleapi.DockerImageBuildResult{ImageID: "sha256:image"}, moduleapi.RegistryPublicationBinding{
		AuthExecution: moduleapi.RegistryAuthExecution{Mode: "unsupported"},
	}, nil)
	if err == nil {
		t.Fatal("expected unsupported credential execution mode to be rejected")
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
