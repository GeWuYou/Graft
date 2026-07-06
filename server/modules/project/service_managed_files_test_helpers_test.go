package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type managedDraftProposal struct {
	ComposePath    string
	ComposeContent string
	EnvPath        string
	EnvContent     *string
}

type managedDraftRestore struct {
	Path    string
	Content []byte
	Exists  bool
}

// writeManagedDraft 在受管工作目录中写入草稿内容，并记录原始状态以便恢复。
func writeManagedDraft(
	workingDirectory string,
	proposal managedDraftProposal,
) (restoreItems []managedDraftRestore, err error) {
	targets := []struct {
		path    string
		content string
	}{
		{path: proposal.ComposePath, content: proposal.ComposeContent},
	}
	if proposal.EnvPath != "" && proposal.EnvContent != nil {
		targets = append(targets, struct {
			path    string
			content string
		}{path: proposal.EnvPath, content: *proposal.EnvContent})
	}
	fsRoot, err := openManagedRootFS(filepath.Clean(workingDirectory))
	if err != nil {
		return nil, fmt.Errorf("open managed draft root: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeManagedRootFS(fsRoot))
	}()
	restoreItems = make([]managedDraftRestore, 0, len(targets))
	for _, target := range targets {
		relative, err := fsRoot.relative(target.path)
		if err != nil {
			return nil, fmt.Errorf("resolve managed draft path %s: %w", target.path, err)
		}
		original, err := fsRoot.root.ReadFile(relative)
		exists := err == nil
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read managed draft source %s: %w", target.path, err)
		}
		restoreItems = append(restoreItems, managedDraftRestore{
			Path:    target.path,
			Content: append([]byte(nil), original...),
			Exists:  exists,
		})
		if err := fsRoot.root.WriteFile(relative, []byte(target.content), managedCreateFileMode); err != nil {
			return nil, fmt.Errorf("write managed draft target %s: %w", target.path, err)
		}
	}
	return restoreItems, nil
}

// restoreManagedDraft 按记录的恢复项还原受管草稿文件的原始状态。
func restoreManagedDraft(workingDirectory string, items []managedDraftRestore) (err error) {
	fsRoot, err := openManagedRootFS(filepath.Clean(workingDirectory))
	if err != nil {
		return fmt.Errorf("open managed draft restore root: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeManagedRootFS(fsRoot))
	}()
	for index := len(items) - 1; index >= 0; index-- {
		item := items[index]
		relative, relErr := fsRoot.relative(item.Path)
		if relErr != nil {
			err = errors.Join(err, fmt.Errorf("resolve managed draft restore path %s: %w", item.Path, relErr))
			continue
		}
		if item.Exists {
			if writeErr := fsRoot.root.WriteFile(relative, item.Content, managedCreateFileMode); writeErr != nil {
				err = errors.Join(err, fmt.Errorf("restore managed draft file %s: %w", item.Path, writeErr))
			}
			continue
		}
		if removeErr := fsRoot.root.Remove(relative); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, fmt.Errorf("remove managed draft file %s: %w", item.Path, removeErr))
		}
	}
	return err
}

func restoreManagedDraftOnFailure(workingDirectory string, restoreItems []managedDraftRestore, resultErr *error) {
	if resultErr == nil || *resultErr == nil {
		return
	}
	*resultErr = errors.Join(*resultErr, restoreManagedDraft(workingDirectory, restoreItems))
}
