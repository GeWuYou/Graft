package build

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

type retentionBuildRepository struct {
	*recordingBuildRepository
	claimed    []buildstore.ExpiredSnapshotMaterialization
	purged     []string
	released   []string
	claimErr   error
	purgeErr   error
	releaseErr error
}

func (r *retentionBuildRepository) ClaimExpiredSnapshotMaterializations(context.Context, time.Time, time.Time, int) ([]buildstore.ExpiredSnapshotMaterialization, error) {
	if r.claimErr != nil {
		return nil, r.claimErr
	}
	return append([]buildstore.ExpiredSnapshotMaterialization(nil), r.claimed...), nil
}
func (r *retentionBuildRepository) MarkSnapshotMaterializationPurged(_ context.Context, snapshotID string) error {
	r.purged = append(r.purged, snapshotID)
	return r.purgeErr
}
func (r *retentionBuildRepository) ReleaseSnapshotMaterializationClaim(_ context.Context, snapshotID string) error {
	r.released = append(r.released, snapshotID)
	return r.releaseErr
}

func TestAdoptSnapshotMaterializationCopiesIntoBuildOwnedRoot(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := adoptSnapshotMaterialization(moduleapi.WorkspaceSnapshot{ID: "snapshot_test", MaterializedRoot: source})
	if err != nil {
		t.Fatalf("adopt snapshot materialization: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(snapshot.MaterializedRoot) })
	if filepath.Dir(snapshot.MaterializedRoot) == filepath.Dir(source) {
		t.Fatalf("materialization remained under source root: %q", snapshot.MaterializedRoot)
	}
	content, err := os.ReadFile(filepath.Join(snapshot.MaterializedRoot, "Dockerfile"))
	if err != nil || string(content) != "FROM scratch\n" {
		t.Fatalf("copied source = %q, error = %v", content, err)
	}
}

func TestAdoptSnapshotMaterializationRejectsSymbolicLink(t *testing.T) {
	source := t.TempDir()
	if err := os.Symlink("elsewhere", filepath.Join(source, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := adoptSnapshotMaterialization(moduleapi.WorkspaceSnapshot{ID: "snapshot_test", MaterializedRoot: source}); err == nil {
		t.Fatal("expected symbolic link materialization to be rejected")
	}
}

func TestAdoptSnapshotMaterializationRejectsDigestMismatch(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := adoptSnapshotMaterialization(moduleapi.WorkspaceSnapshot{ID: "snapshot_test", ContentDigest: "sha256:" + strings.Repeat("0", 64), MaterializedRoot: source}); err == nil {
		t.Fatal("expected immutable snapshot digest mismatch to fail closed")
	}
}

func TestCleanupExpiredSnapshotMaterializationsPurgesOnlyManagedMaterialization(t *testing.T) {
	root := filepath.Join(os.TempDir(), "graft-build-snapshots")
	if err := os.MkdirAll(root, managedSnapshotDirectoryMode); err != nil {
		t.Fatal(err)
	}
	materialization, err := os.MkdirTemp(root, "snapshot-cleanup-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(materialization, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	repository := &retentionBuildRepository{recordingBuildRepository: &recordingBuildRepository{}, claimed: []buildstore.ExpiredSnapshotMaterialization{{SnapshotID: "snapshot_expired", MaterializationRef: materializationReference(materialization)}}}
	service, err := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildDocker{}, repository)
	if err != nil {
		t.Fatal(err)
	}
	purged, err := service.CleanupExpiredSnapshotMaterializations(context.Background(), time.Now().UTC(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if purged != 1 || len(repository.purged) != 1 || repository.purged[0] != "snapshot_expired" {
		t.Fatalf("cleanup result = %d purged=%v", purged, repository.purged)
	}
	if _, err := os.Stat(materialization); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("materialization was not removed: %v", err)
	}
}

func TestCleanupExpiredSnapshotMaterializationsRefusesOutsideBuildStorage(t *testing.T) {
	outside := t.TempDir()
	repository := &retentionBuildRepository{recordingBuildRepository: &recordingBuildRepository{}, claimed: []buildstore.ExpiredSnapshotMaterialization{{SnapshotID: "snapshot_invalid", MaterializationRef: outside}}}
	service, err := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildDocker{}, repository)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CleanupExpiredSnapshotMaterializations(context.Background(), time.Now().UTC(), 10)
	if err == nil || len(repository.released) != 1 || repository.released[0] != "snapshot_invalid" {
		t.Fatalf("cleanup error=%v released=%v", err, repository.released)
	}
	if _, statErr := os.Stat(outside); statErr != nil {
		t.Fatalf("outside path must not be deleted: %v", statErr)
	}
}
