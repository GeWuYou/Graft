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

// writeManagedProjectFiles 在受管根目录中创建工作目录并写入项目文件。
// 它会写入 compose 文件，并在提供环境文件路径和内容时写入 env 文件。
// @param validation 包含工作目录以及各文件绝对路径的校验结果。
// @param normalized 包含要写入的规范化文件内容。
// @returns 返回清理后的工作目录、已创建文件的绝对路径列表，以及错误。
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

// cleanupManagedCreate 清理受管创建过程中生成的文件和目录。
// 它会按逆序删除已创建的文件，并在提供目录时移除该目录；当没有任何待清理路径时返回 nil。
func cleanupManagedCreate(createdDir string, createdFiles []string) (err error) {
	if len(createdFiles) == 0 && createdDir == "" {
		return nil
	}
	fsRoot := (*managedRootFS)(nil)
	if strings.TrimSpace(createdDir) != "" {
		fsRoot, _, err = openManagedProjectParentRoot(createdDir)
	} else {
		fsRoot, err = openManagedRootFSForPaths("", createdFiles...)
	}
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

// cleanupManagedCreateFiles 按逆序删除受管创建过程中生成的文件。
// 
// 删除过程中产生的错误会被合并后返回。
func cleanupManagedCreateFiles(fsRoot *managedRootFS, createdFiles []string) error {
	var err error
	for i := len(createdFiles) - 1; i >= 0; i-- {
		err = errors.Join(err, removeManagedCreatePath(fsRoot, createdFiles[i], "file"))
	}
	return err
}

// kind 用于描述要移除的路径类型，通常为 "file" 或 "directory"。
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

// managedCreateEnvFileList 返回环境文件绝对路径列表。
//
// 当 envFileAbsolutePath 为 nil 时返回 nil；否则返回仅包含该路径的切片。
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

// loadManagedDraftContent 从受管工作目录加载 draft 的 compose 和可选 env 内容，并保留当前配置哈希。
// 当缺少 compose 文件授权或无法读取受管根目录中的文件时返回错误；env 文件存在时也会一并加载并归一化内容。
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

// buildManagedDraftInput 构造带有草稿内容覆盖的 projectcompose 输入。
// 它会使用给定的工作目录和草稿路径初始化输入，并在需要时附加 env 文件。
// @param workingDirectory 项目工作目录。
// @param proposal 草稿路径与内容。
// @returns 配置了内容覆盖的 projectcompose.Input，以及 WithContentOverrides 返回的错误。
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

// optionalDraftBytes 在路径和内容都存在时返回内容的字节切片。
// 当路径为空或内容为 nil 时，返回 nil。
func optionalDraftBytes(path string, content *string) []byte {
	if path == "" || content == nil {
		return nil
	}
	return []byte(*content)
}

// writeManagedDraft 在受管工作目录中写入草稿内容，并记录原始状态以便恢复。
// 它会覆盖 compose 文件，并在提供 env 路径和内容时一并覆盖 env 文件。
// @return restoreItems 用于恢复各目标文件原始内容的记录。
// @return err 写入或解析路径失败时返回的错误。
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
// 它会根据每个恢复项中的原始内容写回文件，或在原文件不存在时删除对应路径。
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

// openManagedRootFS 打开指定根目录的受管文件系统根。
// @param rootDir 受管根目录路径。
// @returns 指向受管根文件系统的句柄及错误；当目录为空或无法打开时返回错误。
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

// openManagedProjectParentRoot 打开工作目录的受管父目录并返回其相对工作目录。
// 它会清理并校验工作目录路径，拒绝空路径和无效路径。
// 返回受管根、工作目录在父目录中的相对名称，以及错误。
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

// openManagedRootFSForPaths 根据根目录或候选路径打开受管根目录。
// 优先使用 rootDir；当 rootDir 为空时，取第一个非空路径的父目录作为受管根目录。
// 如果没有可用路径，则返回错误。
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

// relativePathWithinRoot 将路径转换为相对于受管根目录的路径。
// 当路径与根目录相同或归一化后为空时，返回 "."；当路径越过根目录时返回错误。
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

// closeManagedRootFS 关闭受管根目录并返回关闭过程中发生的错误。
//
// @return 关闭受管根目录时产生的错误；若接收者为空或关闭成功，则返回 nil。
func closeManagedRootFS(fsRoot *managedRootFS) error {
	if fsRoot == nil {
		return nil
	}
	if err := fsRoot.release(); err != nil {
		return fmt.Errorf("close managed root %s: %w", fsRoot.rootDir, err)
	}
	return nil
}

// deleteManagedWorkingDirectory 删除受管工作目录中的全部内容。
// 它会打开该工作目录对应的受管根，并移除根下的所有文件和子目录。若打开、删除或关闭过程中发生错误，则返回相应错误。
func deleteManagedWorkingDirectory(workingDirectory string) (err error) {
	fsRoot, relativeWorkingDirectory, err := openManagedProjectParentRoot(workingDirectory)
	if err != nil {
		return fmt.Errorf("open managed working directory %s: %w", workingDirectory, err)
	}
	defer func() {
		err = errors.Join(err, closeManagedRootFS(fsRoot))
	}()
	if err := fsRoot.root.RemoveAll(relativeWorkingDirectory); err != nil {
		return fmt.Errorf("remove managed working directory %s: %w", workingDirectory, err)
	}
	return nil
}
