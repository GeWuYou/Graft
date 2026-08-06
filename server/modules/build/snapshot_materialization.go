package build

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"graft/server/internal/moduleapi"
)

const managedSnapshotDirectoryMode = 0o750

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
	failed = false
	snapshot.MaterializedRoot = destination
	return snapshot, nil
}
