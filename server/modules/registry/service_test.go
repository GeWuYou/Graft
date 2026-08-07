package registry

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"graft/server/internal/moduleapi"
	registrystore "graft/server/modules/registry/store"
)

func TestResolveArtifactDestinationRequiresLiveAssignedPublishRepository(t *testing.T) {
	db := openRegistryTestDB(t)
	seedRegistryDestination(t, db, true, true, false, true)
	repository, err := registrystore.NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	service := NewService(repository)

	destination, err := service.ResolveArtifactDestination(context.Background(), 9, moduleapi.BuildDestination{
		Kind: "oci_registry", ConnectionRef: "registry:primary", RepositoryRef: "team/api", Reference: "v1",
	})
	if err != nil {
		t.Fatalf("resolve destination: %v", err)
	}
	want := moduleapi.AuthorizedArtifactDestination{Kind: "oci_registry", ConnectionRef: "registry:primary", RepositoryRef: "team/api", Reference: "v1"}
	if !reflect.DeepEqual(destination, want) {
		t.Fatalf("destination = %#v, want %#v", destination, want)
	}
	assertAuthorizedDestinationIsNonSecret(t)
}

func TestResolveArtifactDestinationRejectsUnauthorizedAndUnavailableRepositories(t *testing.T) {
	for _, tc := range []struct {
		name                string
		connectionAvailable bool
		allowPush           bool
		unassigned          bool
	}{
		{name: "unassigned", connectionAvailable: true, allowPush: true, unassigned: true},
		{name: "connection unavailable", connectionAvailable: false, allowPush: true},
		{name: "push disabled", connectionAvailable: true, allowPush: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := openRegistryTestDB(t)
			seedRegistryDestination(t, db, tc.connectionAvailable, tc.allowPush, tc.unassigned, true)
			repository, err := registrystore.NewSQLRepository(db)
			if err != nil {
				t.Fatalf("new repository: %v", err)
			}
			_, err = NewService(repository).ResolveArtifactDestination(context.Background(), 9, moduleapi.BuildDestination{
				Kind: "oci_registry", ConnectionRef: "registry:primary", RepositoryRef: "team/api", Reference: "v1",
			})
			if err == nil {
				t.Fatal("resolve destination unexpectedly succeeded")
			}
		})
	}
}

func TestResolvePublicationBindingUsesEphemeralCredentialMode(t *testing.T) {
	db := openRegistryTestDB(t)
	seedRegistryDestination(t, db, true, true, false, true)
	repository, err := registrystore.NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	binding, err := NewService(repository).ResolvePublicationBinding(context.Background(), moduleapi.AuthorizedArtifactDestination{
		Kind: "oci_registry", ConnectionRef: "registry:primary", RepositoryRef: "team/api", Reference: "v1",
	})
	if err != nil {
		t.Fatalf("resolve publication binding: %v", err)
	}
	if binding.AuthExecution.Mode != moduleapi.RegistryAuthExecutionEphemeral {
		t.Fatalf("credential execution mode = %q", binding.AuthExecution.Mode)
	}
	if binding.Endpoint != "https://registry.example" || binding.CredentialRef != "credential:registry-primary" {
		t.Fatalf("publication binding = %#v", binding)
	}
}

func TestAuthorizeArtifactCopyRequiresSourcePullAndDestinationPush(t *testing.T) {
	for _, tc := range []struct {
		name      string
		allowPull bool
		allowPush bool
		wantError bool
	}{
		{name: "authorized", allowPull: true, allowPush: true},
		{name: "source pull disabled", allowPull: false, allowPush: true, wantError: true},
		{name: "destination push disabled", allowPull: true, allowPush: false, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := openRegistryTestDB(t)
			seedRegistryDestination(t, db, true, tc.allowPush, false, tc.allowPull)
			repository, err := registrystore.NewSQLRepository(db)
			if err != nil {
				t.Fatal(err)
			}
			_, err = NewService(repository).AuthorizeArtifactCopy(context.Background(), 9, artifactCopySource(), moduleapi.BuildDestination{Kind: "oci_registry", ConnectionRef: "registry:primary", RepositoryRef: "team/api", Reference: "promoted"})
			if (err != nil) != tc.wantError {
				t.Fatalf("AuthorizeArtifactCopy() error = %v, wantError %v", err, tc.wantError)
			}
		})
	}
}

func TestResolveArtifactCopyBindingKeepsCredentialsInExecutionOnlyBinding(t *testing.T) {
	db := openRegistryTestDB(t)
	seedRegistryDestination(t, db, true, true, false, true)
	repository, err := registrystore.NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repository)
	copy, err := service.AuthorizeArtifactCopy(context.Background(), 9, artifactCopySource(), moduleapi.BuildDestination{Kind: "oci_registry", ConnectionRef: "registry:primary", RepositoryRef: "team/api", Reference: "promoted"})
	if err != nil {
		t.Fatalf("AuthorizeArtifactCopy() error = %v", err)
	}
	binding, err := service.ResolveArtifactCopyBinding(context.Background(), copy)
	if err != nil {
		t.Fatalf("ResolveArtifactCopyBinding() error = %v", err)
	}
	if binding.SourceEndpoint != "https://registry.example" || binding.SourceCredentialRef != "credential:registry-primary" || binding.Destination.CredentialRef != "credential:registry-primary" {
		t.Fatalf("copy binding = %#v", binding)
	}
}

func artifactCopySource() moduleapi.ArtifactPublicationSource {
	return moduleapi.ArtifactPublicationSource{ArtifactID: "artifact_1", PublicationID: "publication_1", Digest: "sha256:" + strings.Repeat("a", 64), MediaType: "application/vnd.oci.image.manifest.v1+json", DestinationKind: "oci_registry", ConnectionRef: "registry:primary", RepositoryRef: "team/api"}
}

func TestNewModuleSpecOwnsRegistryMigration(t *testing.T) {
	spec := NewModuleSpec()
	if spec.Name() != moduleID {
		t.Fatalf("module id = %q, want %q", spec.Name(), moduleID)
	}
	if !reflect.DeepEqual(spec.MigrationPath, []string{"modules/registry/migrations"}) {
		t.Fatalf("migration path = %#v", spec.MigrationPath)
	}
}

func assertAuthorizedDestinationIsNonSecret(t *testing.T) {
	t.Helper()
	typeOfDestination := reflect.TypeOf(moduleapi.AuthorizedArtifactDestination{})
	for _, forbidden := range []string{"Endpoint", "Credential", "Credentials", "Secret", "Token", "Password"} {
		if _, found := typeOfDestination.FieldByName(forbidden); found {
			t.Fatalf("AuthorizedArtifactDestination must not expose %s", forbidden)
		}
	}
}

func seedRegistryDestination(t *testing.T, db *sql.DB, connectionAvailable, allowPush, unassigned, allowPull bool) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO registry_connections (id, connection_ref, endpoint, credential_ref, availability, deleted_at) VALUES (1, 'registry:primary', 'https://registry.example', 'credential:registry-primary', ?, 0)`, connectionAvailable); err != nil {
		t.Fatalf("seed connection: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO artifact_repositories (id, connection_id, repository_ref, allow_pull, allow_push, deleted_at) VALUES (1, 1, 'team/api', ?, ?, 0)`, allowPull, allowPush); err != nil {
		t.Fatalf("seed repository: %v", err)
	}
	if !unassigned {
		if _, err := db.Exec(`INSERT INTO artifact_repository_user_assignments (repository_id, user_id, deleted_at) VALUES (1, 9, 0)`); err != nil {
			t.Fatalf("seed assignment: %v", err)
		}
	}
}

func openRegistryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE registry_connections (id INTEGER PRIMARY KEY, connection_ref TEXT NOT NULL, endpoint TEXT NOT NULL, credential_ref TEXT NULL, availability BOOLEAN NOT NULL, deleted_at INTEGER NOT NULL)`,
		`CREATE TABLE artifact_repositories (id INTEGER PRIMARY KEY, connection_id INTEGER NOT NULL, repository_ref TEXT NOT NULL, allow_pull BOOLEAN NOT NULL, allow_push BOOLEAN NOT NULL, deleted_at INTEGER NOT NULL)`,
		`CREATE TABLE artifact_repository_user_assignments (repository_id INTEGER NOT NULL, user_id INTEGER NOT NULL, deleted_at INTEGER NOT NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create registry test table: %v", err)
		}
	}
	return db
}
