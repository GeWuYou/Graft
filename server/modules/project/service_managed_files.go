package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

type managedRootFS struct {
	root    *os.Root
	rootDir string
}

func writeManagedProjectFiles(
	validation ManagedProjectCreateValidationResult,
	normalized normalizedManagedCreateRequest,
) (workingDirectory string, createdFiles []string, err error) {
	workingDirectory = filepath.Clean(validation.WorkingDirectory)
	parentRoot, relativeWorkingDirectory, err := openManagedProjectParentRoot(workingDirectory)
	if err != nil {
		return "", nil, fmt.Errorf("open managed project parent root: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeManagedRootFS(parentRoot))
	}()
	if err := parentRoot.root.MkdirAll(relativeWorkingDirectory, managedCreateDirMode); err != nil {
		return "", nil, fmt.Errorf("create working directory: %w", err)
	}
	workingRoot, err := parentRoot.root.OpenRoot(relativeWorkingDirectory)
	if err != nil {
		return workingDirectory, nil, fmt.Errorf("open working directory: %w", err)
	}
	defer func() {
		err = errors.Join(err, workingRoot.Close())
	}()
	createdFiles = []string{}
	composeRelativePath, err := relativePathWithinRoot(workingDirectory, validation.ComposeFileAbsolutePath)
	if err != nil {
		return workingDirectory, createdFiles, fmt.Errorf("resolve compose file path: %w", err)
	}
	if err := workingRoot.WriteFile(composeRelativePath, []byte(normalized.ComposeFileContent), managedCreateFileMode); err != nil {
		return workingDirectory, createdFiles, fmt.Errorf("write compose file: %w", err)
	}
	createdFiles = append(createdFiles, validation.ComposeFileAbsolutePath)
	if validation.EnvFileAbsolutePath != nil && normalized.EnvFileContent != nil {
		envRelativePath, relErr := relativePathWithinRoot(workingDirectory, *validation.EnvFileAbsolutePath)
		if relErr != nil {
			return workingDirectory, createdFiles, fmt.Errorf("resolve env file path: %w", relErr)
		}
		if err := workingRoot.WriteFile(envRelativePath, []byte(*normalized.EnvFileContent), managedCreateFileMode); err != nil {
			return workingDirectory, createdFiles, fmt.Errorf("write env file: %w", err)
		}
		createdFiles = append(createdFiles, *validation.EnvFileAbsolutePath)
	}
	return workingDirectory, createdFiles, nil
}

func cleanupManagedCreate(createdDir string, createdFiles []string) (err error) {
	if len(createdFiles) == 0 && createdDir == "" {
		return nil
	}
	fsRoot, err := openManagedRootFSForPaths(createdDir, createdFiles...)
	if err != nil {
		return fmt.Errorf("open cleanup root: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeManagedRootFS(fsRoot))
	}()
	err = errors.Join(err, cleanupManagedCreateFiles(fsRoot, createdFiles))
	if createdDir != "" {
		err = errors.Join(err, removeManagedCreatePath(fsRoot, createdDir, "directory"))
	}
	return err
}

func cleanupManagedCreateFiles(fsRoot *managedRootFS, createdFiles []string) error {
	var err error
	for i := len(createdFiles) - 1; i >= 0; i-- {
		err = errors.Join(err, removeManagedCreatePath(fsRoot, createdFiles[i], "file"))
	}
	return err
}

func removeManagedCreatePath(fsRoot *managedRootFS, absolutePath string, kind string) error {
	relative, err := fsRoot.relative(absolutePath)
	if err != nil {
		return fmt.Errorf("resolve cleanup %s %s: %w", kind, absolutePath, err)
	}
	if removeErr := fsRoot.root.Remove(relative); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return fmt.Errorf("remove cleanup %s %s: %w", kind, absolutePath, removeErr)
	}
	return nil
}

func managedCreateEnvFileList(envFileAbsolutePath *string) []string {
	if envFileAbsolutePath == nil {
		return nil
	}
	return []string{*envFileAbsolutePath}
}

type managedDraftContent struct {
	ComposePath       string
	ComposeContent    string
	EnvPath           string
	EnvContent        string
	CurrentConfigHash string
}

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

func loadManagedDraftContent(aggregate projectstore.ProjectAggregate) (result managedDraftContent, err error) {
	composeFiles := filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())
	if len(composeFiles) == 0 {
		return managedDraftContent{}, fmt.Errorf("%w: missing compose file authority", errProjectImportValidation)
	}
	fsRoot, err := openManagedRootFS(filepath.Clean(aggregate.Project.WorkingDirectory))
	if err != nil {
		return managedDraftContent{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	defer func() {
		err = errors.Join(err, closeManagedRootFS(fsRoot))
	}()
	composeRelativePath, err := fsRoot.relative(composeFiles[0].AbsolutePath)
	if err != nil {
		return managedDraftContent{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	composeContent, err := fsRoot.root.ReadFile(composeRelativePath)
	if err != nil {
		return managedDraftContent{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	result = managedDraftContent{
		ComposePath:       composeFiles[0].AbsolutePath,
		ComposeContent:    normalizeTextBlock(string(composeContent)),
		CurrentConfigHash: aggregate.Project.LastRefreshConfigHash,
	}
	envFiles := filterFiles(aggregate.Files, projectcontract.FileKindEnv.String())
	if len(envFiles) > 0 {
		envRelativePath, relErr := fsRoot.relative(envFiles[0].AbsolutePath)
		if relErr != nil {
			return managedDraftContent{}, fmt.Errorf("%w: %v", errProjectImportValidation, relErr)
		}
		envContent, readErr := fsRoot.root.ReadFile(envRelativePath)
		if readErr != nil {
			return managedDraftContent{}, fmt.Errorf("%w: %v", errProjectImportValidation, readErr)
		}
		result.EnvPath = envFiles[0].AbsolutePath
		result.EnvContent = normalizeTextBlock(string(envContent))
	}
	return result, nil
}

func buildManagedDraftInput(workingDirectory string, proposal managedDraftProposal) (projectcompose.Input, error) {
	input := projectcompose.Input{
		WorkingDirectory: workingDirectory,
		ComposeFiles:     []string{proposal.ComposePath},
	}
	if proposal.EnvPath != "" && proposal.EnvContent != nil {
		input.EnvFiles = []string{proposal.EnvPath}
	}
	return input.WithContentOverrides(map[string][]byte{
		proposal.ComposePath: []byte(proposal.ComposeContent),
		proposal.EnvPath:     optionalDraftBytes(proposal.EnvPath, proposal.EnvContent),
	}), nil
}

func optionalDraftBytes(path string, content *string) []byte {
	if path == "" || content == nil {
		return nil
	}
	return []byte(*content)
}

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

func openManagedRootFS(rootDir string) (*managedRootFS, error) {
	absolute := filepath.Clean(strings.TrimSpace(rootDir))
	if absolute == "" {
		return nil, fmt.Errorf("managed root directory is required")
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, fmt.Errorf("open managed root %s: %w", absolute, err)
	}
	return &managedRootFS{root: root, rootDir: absolute}, nil
}

func openManagedProjectParentRoot(workingDirectory string) (*managedRootFS, string, error) {
	absolute := filepath.Clean(strings.TrimSpace(workingDirectory))
	if absolute == "" {
		return nil, "", fmt.Errorf("working directory is required")
	}
	parentDir := filepath.Dir(absolute)
	relativeWorkingDirectory := filepath.Base(absolute)
	if relativeWorkingDirectory == "." || relativeWorkingDirectory == string(filepath.Separator) || relativeWorkingDirectory == "" {
		return nil, "", fmt.Errorf("working directory is invalid")
	}
	fsRoot, err := openManagedRootFS(parentDir)
	if err != nil {
		return nil, "", fmt.Errorf("open managed project parent %s: %w", parentDir, err)
	}
	return fsRoot, relativeWorkingDirectory, nil
}

func openManagedRootFSForPaths(rootDir string, paths ...string) (*managedRootFS, error) {
	if strings.TrimSpace(rootDir) != "" {
		return openManagedRootFS(rootDir)
	}
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		return openManagedRootFS(filepath.Dir(filepath.Clean(trimmed)))
	}
	return nil, fmt.Errorf("managed root directory is required")
}

func (fsRoot *managedRootFS) relative(path string) (string, error) {
	if fsRoot == nil || fsRoot.root == nil {
		return "", fmt.Errorf("managed root is unavailable")
	}
	relative, err := relativePathWithinRoot(fsRoot.rootDir, path)
	if err != nil {
		return "", err
	}
	if relative == "." {
		return ".", nil
	}
	return relative, nil
}

func relativePathWithinRoot(rootDir string, path string) (string, error) {
	relative, err := filepath.Rel(filepath.Clean(strings.TrimSpace(rootDir)), filepath.Clean(strings.TrimSpace(path)))
	if err != nil {
		return "", err
	}
	if relative == "." || relative == "" {
		return ".", nil
	}
	if strings.HasPrefix(relative, "..") {
		return "", fmt.Errorf("path escapes managed root")
	}
	return relative, nil
}

func (fsRoot *managedRootFS) release() error {
	if fsRoot == nil || fsRoot.root == nil {
		return nil
	}
	return fsRoot.root.Close()
}

func closeManagedRootFS(fsRoot *managedRootFS) error {
	if fsRoot == nil {
		return nil
	}
	if err := fsRoot.release(); err != nil {
		return fmt.Errorf("close managed root %s: %w", fsRoot.rootDir, err)
	}
	return nil
}

func deleteManagedWorkingDirectory(workingDirectory string) (err error) {
	fsRoot, err := openManagedRootFS(filepath.Clean(workingDirectory))
	if err != nil {
		return fmt.Errorf("open managed working directory %s: %w", workingDirectory, err)
	}
	defer func() {
		err = errors.Join(err, closeManagedRootFS(fsRoot))
	}()
	if err := fsRoot.root.RemoveAll("."); err != nil {
		return fmt.Errorf("remove managed working directory %s: %w", workingDirectory, err)
	}
	return nil
}
