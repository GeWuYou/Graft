package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"graft/server/internal/moduleapi"
)

const managedSnapshotDirectoryMode = 0o750

type buildWorkspaceMaterializer struct{}

var _ moduleapi.WorkspaceMaterializer = buildWorkspaceMaterializer{}

func (buildWorkspaceMaterializer) MaterializeSnapshot(_ context.Context, snapshot moduleapi.WorkspaceSnapshot, _ moduleapi.WorkspaceMaterializationRequest) (moduleapi.WorkspaceMaterialization, error) {
	materialized, err := adoptSnapshotMaterialization(snapshot)
	if err != nil {
		return moduleapi.WorkspaceMaterialization{}, err
	}
	return moduleapi.WorkspaceMaterialization{SnapshotID: materialized.ID, ContentDigest: materialized.ContentDigest, MaterializedRoot: materialized.MaterializedRoot}, nil
}

func (buildWorkspaceMaterializer) ReleaseMaterialization(_ context.Context, materialization moduleapi.WorkspaceMaterialization) error {
	if strings.TrimSpace(materialization.MaterializedRoot) == "" {
		return errors.New("invalid workspace materialization")
	}
	managedRoot := filepath.Join(os.TempDir(), "graft-build-snapshots")
	relative, err := filepath.Rel(managedRoot, materialization.MaterializedRoot)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("workspace materialization is outside Build-owned root")
	}
	return os.RemoveAll(materialization.MaterializedRoot)
}

// adoptSnapshotMaterialization 将来源 adapter 交付的临时物化副本转交给 Build。
// 该目录仅供执行器按冻结计划读取，不进入 HTTP、Task metadata 或审阅投影。
//
//nolint:gocognit,gocyclo,cyclop // 复制边界必须逐项拒绝链接和非常规文件。
func adoptSnapshotMaterialization(snapshot moduleapi.WorkspaceSnapshot) (moduleapi.WorkspaceSnapshot, error) {
	if strings.TrimSpace(snapshot.ID) == "" || !filepath.IsAbs(snapshot.MaterializedRoot) {
		return moduleapi.WorkspaceSnapshot{}, errors.New("invalid source snapshot materialization")
	}
	root := filepath.Join(os.TempDir(), "graft-build-snapshots")
	if err := os.MkdirAll(root, managedSnapshotDirectoryMode); err != nil {
		return moduleapi.WorkspaceSnapshot{}, err
	}
	destination, err := os.MkdirTemp(root, "snapshot-")
	if err != nil {
		return moduleapi.WorkspaceSnapshot{}, err
	}
	failed := true
	defer func() {
		if failed {
			_ = os.RemoveAll(destination)
		}
	}()
	if err := filepath.WalkDir(snapshot.MaterializedRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(snapshot.MaterializedRoot, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("snapshot materialization contains symbolic link")
		}
		output := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(output, managedSnapshotDirectoryMode)
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			if err != nil {
				return err
			}
			return errors.New("snapshot materialization contains unsupported entry")
		}
		source, err := os.Open(path) // #nosec G122,G304 -- path is enumerated from the adapter-owned temporary snapshot root and links are rejected before opening.
		if err != nil {
			return err
		}
		defer func() { _ = source.Close() }()
		outputFile, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm()) // #nosec G304 -- output is derived below Build-owned materialization root.
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(outputFile, source)
		closeErr := outputFile.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}); err != nil {
		return moduleapi.WorkspaceSnapshot{}, err
	}
	snapshot.MaterializedRoot = destination
	if len(snapshot.ContentDigest) == len("sha256:")+64 && strings.HasPrefix(snapshot.ContentDigest, "sha256:") {
		actual, err := materializationDigest(destination)
		if err != nil {
			return moduleapi.WorkspaceSnapshot{}, err
		}
		if actual != snapshot.ContentDigest {
			return moduleapi.WorkspaceSnapshot{}, errors.New("snapshot materialization digest does not match immutable snapshot")
		}
	}
	failed = false
	return snapshot, nil
}

//nolint:gocognit,cyclop // 目录摘要必须在同一遍历中拒绝链接、非常规文件并收集稳定条目。
func materializationDigest(root string) (string, error) {
	entries := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("snapshot materialization contains symbolic link")
		}
		if entry.IsDir() {
			entries = append(entries, "d:"+filepath.ToSlash(rel))
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			if err != nil {
				return err
			}
			return errors.New("snapshot materialization contains unsupported entry")
		}
		content, err := os.ReadFile(path) // #nosec G122,G304 -- path is enumerated beneath the adapter-owned root after link rejection.
		if err != nil {
			return err
		}
		sum := sha256.Sum256(content)
		entries = append(entries, "f:"+filepath.ToSlash(rel)+":"+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(entries)
	hash := sha256.New()
	for _, entry := range entries {
		_, _ = io.WriteString(hash, entry+"\n")
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
