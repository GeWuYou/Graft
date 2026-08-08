package runtimetarget

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	containerdi "graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

type moduleCredentialProvider struct{}

func (moduleCredentialProvider) Prepare(context.Context, moduleapi.CredentialRequest) (moduleapi.EphemeralCredentialSession, error) {
	return moduleapi.EphemeralCredentialSession{}, nil
}

func (moduleCredentialProvider) Inject(context.Context, moduleapi.EphemeralCredentialSession, moduleapi.CredentialInjectionTarget) error {
	return nil
}

func (moduleCredentialProvider) Revoke(context.Context, moduleapi.EphemeralCredentialSession) error {
	return nil
}

func TestRegisterReadersRegistersRuntimeExecutionAdapterWhenCredentialProviderExists(t *testing.T) {
	services := containerdi.New()
	if err := services.RegisterSingleton((*moduleapi.CredentialProvider)(nil), func(containerdi.Resolver) (any, error) {
		return moduleCredentialProvider{}, nil
	}); err != nil {
		t.Fatalf("register credential provider: %v", err)
	}
	if err := NewModule(nil).registerReaders(&module.Context{Services: services}); err != nil {
		t.Fatalf("register runtime target readers: %v", err)
	}
	if _, err := services.Resolve((*moduleapi.RuntimeExecutionAdapter)(nil)); err != nil {
		t.Fatalf("resolve runtime execution adapter: %v", err)
	}
	if _, err := services.Resolve((*moduleapi.RuntimeTargetBuilderTelemetryReader)(nil)); err != nil {
		t.Fatalf("resolve builder telemetry facade: %v", err)
	}
	if _, err := services.Resolve((*moduleapi.RuntimeTargetBuilderTelemetryControlPlane)(nil)); err != nil {
		t.Fatalf("resolve builder telemetry control plane: %v", err)
	}
	if _, err := services.Resolve((*moduleapi.BuilderTelemetryProvider)(nil)); !errors.Is(err, containerdi.ErrServiceNotRegistered) {
		t.Fatalf("provider must not bypass the runtime target facade: %v", err)
	}
}

func TestIsolatedDockerEnvironmentRejectsInheritedAuthentication(t *testing.T) {
	t.Setenv("DOCKER_CONFIG", "/ambient/docker")
	t.Setenv("DOCKER_AUTH_CONFIG", "ambient-auth")
	environment := isolatedDockerEnvironment("/isolated/docker")
	for _, entry := range environment {
		if strings.HasPrefix(entry, "DOCKER_AUTH_CONFIG=") || entry == "DOCKER_CONFIG=/ambient/docker" {
			t.Fatalf("inherited Docker authentication remained in child environment: %q", entry)
		}
	}
	if !slices.Contains(environment, "DOCKER_CONFIG=/isolated/docker") {
		t.Fatalf("isolated Docker config missing from child environment: %#v", environment)
	}
}

func TestBuildTargetReaderOnlyReturnsAssignedHealthyBuildTargets(t *testing.T) {
	db := openBuildTargetTestDB(t)
	seedBuildTarget(t, db, 1, "Docker builder", `["image_build","workspace_access"]`, true)
	seedBuildTarget(t, db, 2, "Unavailable builder", `["image_build"]`, false)
	seedBuildTarget(t, db, 3, "No build capability", `["workspace_access"]`, true)
	if _, err := db.Exec(`INSERT INTO runtime_target_user_assignments (runtime_target_id, user_id, created_by, updated_by, deleted_at, deleted_by) VALUES (1, 9, 0, 0, 0, 0), (2, 9, 0, 0, 0, 0), (3, 9, 0, 0, 0, 0)`); err != nil {
		t.Fatalf("seed assignments: %v", err)
	}

	reader := runtimeTargetReader{repository: store.NewSQLRepository(db)}
	targets, err := reader.ListAssignedBuildTargets(context.Background(), 9)
	if err != nil {
		t.Fatalf("list assigned build targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("target count = %d, want 1", len(targets))
	}
	if target := targets[0]; target.ID != 1 || target.DisplayName != "Docker builder" || target.Provider != "docker" || !target.Available || !reflect.DeepEqual(target.SupportedDrivers, []string{"docker-engine"}) || !reflect.DeepEqual(target.SupportedPlatforms, []string{"linux/amd64"}) || !reflect.DeepEqual(target.WorkspaceLocalities, []string{"target-local", "build-snapshot"}) || !reflect.DeepEqual(target.SnapshotDeliveryModes, []string{moduleapi.SnapshotDeliveryModeTargetLocal}) {
		t.Fatalf("build target = %#v", target)
	}
	assertBuildTargetSummaryDoesNotExposeConnection(t)
}

func TestCanUseBuildTargetRechecksAssignmentAndCurrentEligibility(t *testing.T) {
	db := openBuildTargetTestDB(t)
	seedBuildTarget(t, db, 1, "Assigned builder", `["image_build"]`, true)
	seedBuildTarget(t, db, 2, "Unavailable builder", `["image_build"]`, false)
	seedBuildTarget(t, db, 3, "Unassigned builder", `["image_build"]`, true)
	if _, err := db.Exec(`INSERT INTO runtime_target_user_assignments (runtime_target_id, user_id, created_by, updated_by, deleted_at, deleted_by) VALUES (1, 9, 0, 0, 0, 0), (2, 9, 0, 0, 0, 0)`); err != nil {
		t.Fatalf("seed assignments: %v", err)
	}

	reader := runtimeTargetReader{repository: store.NewSQLRepository(db)}
	for _, tc := range []struct {
		name     string
		targetID int64
		want     bool
	}{
		{name: "assigned healthy builder", targetID: 1, want: true},
		{name: "assigned unavailable builder", targetID: 2, want: false},
		{name: "unassigned remote builder", targetID: 3, want: false},
		{name: "invalid target", targetID: 0, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			allowed, err := reader.CanUseBuildTarget(context.Background(), 9, tc.targetID)
			if err != nil {
				t.Fatalf("can use build target: %v", err)
			}
			if allowed != tc.want {
				t.Fatalf("allowed = %v, want %v", allowed, tc.want)
			}
		})
	}
}

func TestDockerTargetConnectionStaysPrivateToProviderBoundary(t *testing.T) {
	db := openBuildTargetTestDB(t)
	seedBuildTarget(t, db, 1, "Local Docker", `["image_build"]`, true)
	repository := store.NewSQLRepository(db)
	connection, err := repository.GetDockerTargetConnection(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if connection.TargetID != 1 || connection.Endpoint != "unix:///var/run/docker.sock" || connection.ConnectionKind != "unix_socket" {
		t.Fatalf("connection = %#v", connection)
	}
	if _, found := reflect.TypeOf(moduleapi.BuildRuntimeTargetSummary{}).FieldByName("Endpoint"); found {
		t.Fatal("BuildRuntimeTargetSummary must not expose endpoint")
	}
	seedBuildTarget(t, db, 2, "Unavailable Docker", `["image_build"]`, false)
	if _, err := repository.GetDockerTargetConnection(context.Background(), 2); !errors.Is(err, store.ErrUnavailable) {
		t.Fatalf("unavailable target error = %v, want %v", err, store.ErrUnavailable)
	}
	if _, err := db.Exec(`INSERT INTO runtime_targets (id, provider, endpoint, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, system_managed, deleted_at) VALUES (?, 'docker', ?, 'Credential URL', 'redacted', 'tcp', ?, true, '', false, 0)`, 4, "tcp://user:password@remote.example:2376", `["image_build"]`); err != nil {
		t.Fatalf("seed credential endpoint target: %v", err)
	}
	if _, err := repository.GetDockerTargetConnection(context.Background(), 4); err == nil {
		t.Fatal("Docker endpoint with embedded credentials unexpectedly accepted")
	}
}

func TestRemoteDockerProviderRequiresProviderTransferAndPreservesSnapshotIdentity(t *testing.T) {
	db := openBuildTargetTestDB(t)
	if _, err := db.Exec(`INSERT INTO runtime_targets (id, provider, endpoint, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, system_managed, deleted_at) VALUES (?, 'docker', ?, ?, 'redacted', 'tcp', ?, true, '', false, 0)`, 2, "tcp://remote.example:2376", "Remote Docker", `["image_build"]`); err != nil {
		t.Fatalf("seed remote target: %v", err)
	}
	root, err := os.MkdirTemp(filepath.Join(os.TempDir(), "graft-build-snapshots"), "provider-test-")
	if err != nil {
		t.Fatalf("create managed snapshot root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	provider := dockerTargetProvider{repository: store.NewSQLRepository(db)}
	if provider.ProviderID() != "docker-target" {
		t.Fatalf("provider id = %q", provider.ProviderID())
	}
	reader := runtimeTargetReader{repository: store.NewSQLRepository(db)}
	target, err := reader.ReadBuildTarget(context.Background(), 2)
	if err != nil {
		t.Fatalf("read remote build target: %v", err)
	}
	if target.SnapshotDeliveryModes[0] != moduleapi.SnapshotDeliveryModeProviderTransfer || len(target.WorkspaceLocalities) != 1 || target.WorkspaceLocalities[0] != "build-snapshot" {
		t.Fatalf("remote build target delivery = %#v, localities = %#v", target.SnapshotDeliveryModes, target.WorkspaceLocalities)
	}
	request := moduleapi.WorkspaceSnapshotDeliveryRequest{TargetID: 2, SnapshotID: "snapshot-remote", ContentDigest: "sha256:source", MaterializedRoot: root, DeliveryMode: moduleapi.SnapshotDeliveryModeProviderTransfer}
	result, err := provider.DeliverWorkspaceSnapshot(context.Background(), request)
	if err != nil {
		t.Fatalf("deliver remote snapshot: %v", err)
	}
	if result.TargetID != request.TargetID || result.SnapshotID != request.SnapshotID || result.ContentDigest != request.ContentDigest {
		t.Fatalf("delivery proof = %#v, want identity-preserving proof", result)
	}
	request.DeliveryMode = moduleapi.SnapshotDeliveryModeTargetLocal
	if _, err := provider.DeliverWorkspaceSnapshot(context.Background(), request); err == nil {
		t.Fatal("remote target accepted target-local snapshot delivery")
	}
}

func TestDockerProviderConformanceRequiresExecutableProviderFacts(t *testing.T) {
	db := openBuildTargetTestDB(t)
	seedBuildTarget(t, db, 1, "Local Docker", `["image_build"]`, true)
	provider := dockerTargetProvider{repository: store.NewSQLRepository(db)}
	request := moduleapi.ProviderExecutionConformanceRequest{
		TargetID: 1, DriverRef: "docker-engine", Platform: "linux/amd64", SnapshotID: "snapshot-1",
		ContentDigest: "sha256:source", DeliveryMode: moduleapi.SnapshotDeliveryModeTargetLocal,
	}
	result, err := provider.ConformProviderExecution(context.Background(), request)
	if err != nil {
		t.Fatalf("provider conformance: %v", err)
	}
	if !result.Executable || result.ProviderID != "docker-target" || result.ConformanceVersion == "" || !result.SnapshotDeliveryProof || !result.DriverExecutionProof || !result.PublicationProof || !result.CancellationProof || !result.CleanupProof {
		t.Fatalf("provider conformance result = %#v", result)
	}
	request.DriverRef = "kaniko"
	if _, err := provider.ConformProviderExecution(context.Background(), request); err == nil {
		t.Fatal("unsupported driver unexpectedly passed provider conformance")
	}
}

func TestProviderBuildInputAcceptsWorkspaceRootContext(t *testing.T) {
	root, err := os.MkdirTemp(filepath.Join(os.TempDir(), "graft-build-snapshots"), "input-test-")
	if err != nil {
		t.Fatalf("create managed snapshot root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	paths, err := normalizeProviderBuildInput(moduleapi.DockerImageBuildInput{
		WorkspaceRoot: root, ContextPath: ".", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1",
	})
	if err != nil {
		t.Fatalf("normalize provider build input: %v", err)
	}
	if paths.root != root || paths.contextPath != root || paths.dockerfilePath != filepath.Join(root, "Dockerfile") {
		t.Fatalf("normalized paths = %#v", paths)
	}
}

func TestDockerProviderCopiesOCIArtifactByDigestAndVerifiesDestination(t *testing.T) {
	db := openBuildTargetTestDB(t)
	seedBuildTarget(t, db, 1, "Local Docker", `["image_build"]`, true)
	provider := dockerTargetProvider{repository: store.NewSQLRepository(db)}
	digest := "sha256:" + strings.Repeat("a", 64)
	input := moduleapi.OCIArtifactCopyInput{
		Source: moduleapi.ArtifactPublicationSource{
			ArtifactID: "artifact-1", PublicationID: "publication-1", Digest: digest,
			MediaType: "application/vnd.oci.image.manifest.v1+json", DestinationKind: "oci_registry",
			ConnectionRef: "registry:source", RepositoryRef: "team/api",
		},
		Destination: moduleapi.AuthorizedArtifactDestination{
			Kind: "oci_registry", ConnectionRef: "registry:destination", RepositoryRef: "team/api", Reference: "promoted",
		},
	}
	binding := moduleapi.RegistryArtifactCopyBinding{
		SourceEndpoint: "https://source.example", SourceCredentialRef: "ref:source",
		SourceAuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral},
		Destination: moduleapi.RegistryPublicationBinding{
			Destination: input.Destination, Endpoint: "https://destination.example", CredentialRef: "ref:destination",
			AuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral},
		},
	}
	raw := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.image.config.v1+json","size":1,"digest":"sha256:` + strings.Repeat("b", 64) + `"},"layers":[]}`)
	oldCommand, oldOutput := providerCommandRunner, providerOutputRunner
	t.Cleanup(func() { providerCommandRunner, providerOutputRunner = oldCommand, oldOutput })
	var commandArgs []string
	providerCommandRunner = func(_ context.Context, _ moduleapi.DockerImageBuildLogSink, args ...string) error {
		commandArgs = append([]string(nil), args...)
		return nil
	}
	providerOutputRunner = func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) > 0 && args[len(args)-2] == "--raw" {
			return raw, nil
		}
		return []byte(digest + "\n"), nil
	}
	result, err := provider.CopyOCIArtifactOnTarget(context.WithValue(context.Background(), dockerCredentialConfigContextKey{}, "/tmp/isolated"), 1, input, binding, nil)
	if err != nil {
		t.Fatalf("copy OCI artifact: %v", err)
	}
	if result.Digest != digest || result.MediaType != input.Source.MediaType || result.SizeBytes != int64(len(raw)) {
		t.Fatalf("copy result = %#v", result)
	}
	wantArgs := []string{"--host", "unix:///var/run/docker.sock", "buildx", "imagetools", "create", "--tag", "destination.example/team/api:promoted", "source.example/team/api@" + digest}
	if !reflect.DeepEqual(commandArgs, wantArgs) {
		t.Fatalf("docker copy args = %#v, want %#v", commandArgs, wantArgs)
	}
}

func TestDockerProviderRejectsOCIArtifactCopyDigestMismatch(t *testing.T) {
	db := openBuildTargetTestDB(t)
	seedBuildTarget(t, db, 1, "Local Docker", `["image_build"]`, true)
	provider := dockerTargetProvider{repository: store.NewSQLRepository(db)}
	digest := "sha256:" + strings.Repeat("a", 64)
	input := moduleapi.OCIArtifactCopyInput{Source: moduleapi.ArtifactPublicationSource{
		ArtifactID: "artifact-1", PublicationID: "publication-1", Digest: digest, MediaType: "application/vnd.oci.image.manifest.v1+json", DestinationKind: "oci_registry", ConnectionRef: "registry:source", RepositoryRef: "team/api",
	}, Destination: moduleapi.AuthorizedArtifactDestination{Kind: "oci_registry", ConnectionRef: "registry:destination", RepositoryRef: "team/api", Reference: "promoted"}}
	binding := moduleapi.RegistryArtifactCopyBinding{SourceEndpoint: "https://source.example", SourceCredentialRef: "ref:source", SourceAuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}, Destination: moduleapi.RegistryPublicationBinding{Destination: input.Destination, Endpoint: "https://destination.example", CredentialRef: "ref:destination", AuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}}}
	oldCommand, oldOutput := providerCommandRunner, providerOutputRunner
	t.Cleanup(func() { providerCommandRunner, providerOutputRunner = oldCommand, oldOutput })
	providerCommandRunner = func(_ context.Context, _ moduleapi.DockerImageBuildLogSink, _ ...string) error { return nil }
	providerOutputRunner = func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) > 0 && args[len(args)-2] == "--raw" {
			return []byte(`{"mediaType":"application/vnd.oci.image.manifest.v1+json"}`), nil
		}
		return []byte("sha256:" + strings.Repeat("c", 64)), nil
	}
	if _, err := provider.CopyOCIArtifactOnTarget(context.WithValue(context.Background(), dockerCredentialConfigContextKey{}, "/tmp/isolated"), 1, input, binding, nil); err == nil {
		t.Fatal("digest mismatch unexpectedly succeeded")
	}
}

func TestNonDockerBuildProvidersRemainFailClosedUntilProviderAuthorityExists(t *testing.T) {
	db := openBuildTargetTestDB(t)
	if _, err := db.Exec(`INSERT INTO runtime_targets (id, provider, endpoint, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, system_managed, deleted_at) VALUES (?, 'kubernetes', 'https://cluster.example', 'Kubernetes Build Farm', 'redacted', 'tcp', ?, true, '', false, 0)`, 7, `["image_build"]`); err != nil {
		t.Fatalf("seed Kubernetes build target: %v", err)
	}
	reader := runtimeTargetReader{repository: store.NewSQLRepository(db)}
	if _, err := reader.ReadBuildTarget(context.Background(), 7); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Kubernetes build target error = %v, want %v", err, store.ErrNotFound)
	}
	connection, err := store.NewSQLRepository(db).GetProviderConnection(context.Background(), 7)
	if err != nil {
		t.Fatalf("private provider connection = %v", err)
	}
	if connection.Provider != "kubernetes" || connection.Endpoint != "https://cluster.example" {
		t.Fatalf("private provider connection = %#v", connection)
	}
	if _, found := reflect.TypeOf(moduleapi.BuildRuntimeTargetSummary{}).FieldByName("Endpoint"); found {
		t.Fatal("Build target summary must not expose provider endpoint")
	}
	if _, err := db.Exec(`INSERT INTO runtime_targets (id, provider, endpoint, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, system_managed, deleted_at) VALUES (?, 'kubernetes', ?, 'Credential URL', 'redacted', 'tcp', ?, true, '', false, 0)`, 8, "https://user:password@cluster.example", `["image_build"]`); err != nil {
		t.Fatalf("seed credential endpoint target: %v", err)
	}
	if _, err := store.NewSQLRepository(db).GetProviderConnection(context.Background(), 8); !errors.Is(err, store.ErrUnavailable) {
		t.Fatalf("credential endpoint error = %v, want %v", err, store.ErrUnavailable)
	}
}

func assertBuildTargetSummaryDoesNotExposeConnection(t *testing.T) {
	t.Helper()
	typeOfSummary := reflect.TypeOf(moduleapi.BuildRuntimeTargetSummary{})
	for _, forbidden := range []string{"Endpoint", "EndpointLabel", "Credential", "Credentials", "Secret"} {
		if _, found := typeOfSummary.FieldByName(forbidden); found {
			t.Fatalf("BuildRuntimeTargetSummary must not expose %s", forbidden)
		}
	}
}

func seedBuildTarget(t *testing.T, db *sql.DB, id int, displayName, capabilities string, available bool) {
	t.Helper()
	endpoint, systemManaged := "tcp://remote.example:2376", false
	if id == 1 {
		endpoint, systemManaged = "unix:///var/run/docker.sock", true
	}
	if _, err := db.Exec(`INSERT INTO runtime_targets (id, provider, endpoint, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, system_managed, deleted_at) VALUES (?, 'docker', ?, ?, 'redacted', 'unix_socket', ?, ?, '', ?, 0)`, id, endpoint, displayName, capabilities, available, systemManaged); err != nil {
		t.Fatalf("seed runtime target %d: %v", id, err)
	}
}

func openBuildTargetTestDB(t *testing.T) *sql.DB {
	db := openRuntimeTargetTestDB(t)
	if _, err := db.Exec(`CREATE TABLE runtime_targets (
		id INTEGER PRIMARY KEY,
		provider TEXT NOT NULL,
		endpoint TEXT NOT NULL,
		display_name TEXT NOT NULL,
		endpoint_label TEXT NOT NULL,
		connection_kind TEXT NOT NULL,
		capabilities_json BLOB NOT NULL,
		availability BOOLEAN NOT NULL,
		last_error TEXT NOT NULL,
		system_managed BOOLEAN NOT NULL DEFAULT false,
		checked_at DATETIME NULL,
		deleted_at INTEGER NOT NULL
	)`); err != nil {
		t.Fatalf("create runtime_targets: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE runtime_target_user_assignments (
		runtime_target_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		created_by INTEGER NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_by INTEGER NOT NULL,
		deleted_at INTEGER NOT NULL DEFAULT 0,
		deleted_by INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		t.Fatalf("create runtime_target_user_assignments: %v", err)
	}
	return db
}

func openRuntimeTargetTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
