package build

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	buildstore "graft/server/modules/build/store"
)

const snapshotMaterializationClaimTTL = 10 * time.Minute

// CleanupExpiredSnapshotMaterializations 删除已到期的 Build-owned 快照物化内容，保留 Snapshot、
// Execution Plan 和 Artifact 审计事实。它只接受私有快照根目录下的路径，遇到非法路径会释放 claim 并失败。
func (s *Service) CleanupExpiredSnapshotMaterializations(ctx context.Context, now time.Time, limit int) (int, error) {
	if s == nil || now.IsZero() || limit < 1 {
		return 0, errors.New("invalid snapshot materialization cleanup request")
	}
	repository, ok := s.repository.(buildstore.SnapshotMaterializationRetentionRepository)
	if !ok {
		return 0, errors.New("snapshot materialization retention repository is unavailable")
	}
	items, err := repository.ClaimExpiredSnapshotMaterializations(ctx, now.UTC(), now.UTC().Add(-snapshotMaterializationClaimTTL), limit)
	if err != nil {
		return 0, err
	}
	purged := 0
	for _, item := range items {
		if !isManagedSnapshotMaterialization(item.MaterializationRef) {
			_ = repository.ReleaseSnapshotMaterializationClaim(ctx, item.SnapshotID)
			return purged, fmt.Errorf("snapshot materialization %q is outside Build-owned storage", item.SnapshotID)
		}
		if err := os.RemoveAll(item.MaterializationRef); err != nil {
			_ = repository.ReleaseSnapshotMaterializationClaim(ctx, item.SnapshotID)
			return purged, fmt.Errorf("purge snapshot materialization %q: %w", item.SnapshotID, err)
		}
		if err := repository.MarkSnapshotMaterializationPurged(ctx, item.SnapshotID); err != nil {
			return purged, err
		}
		purged++
	}
	return purged, nil
}

func isManagedSnapshotMaterialization(path string) bool {
	root := filepath.Clean(filepath.Join(os.TempDir(), "graft-build-snapshots"))
	path = filepath.Clean(strings.TrimSpace(path))
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
