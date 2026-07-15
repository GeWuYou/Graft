package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type managedRootFS struct {
	root    *os.Root
	rootDir string
}

// writeManagedProjectFiles 在受管根目录中创建工作目录并写入项目文件。
// 它会写入 compose 文件，并在提供环境文件路径和内容时写入 env 文件。
// @param validation 包含工作目录以及各文件绝对路径的校验结果。
// @param normalized 包含要写入的规范化文件内容。
// writeManagedProjectFiles 创建工作目录并将工作区条目写入其中。
// 返回清理后的工作目录、已创建文件的绝对路径以及可能发生的错误。
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
	if err := materializeWorkspaceEntries(workingRoot, normalized.WorkspaceEntries, workingDirectory, &createdFiles); err != nil {
		return workingDirectory, createdFiles, err
	}
	return workingDirectory, createdFiles, nil
}

// materializeWorkspaceEntries creates workspace entries and records the absolute paths of created files. It stops at the first materialization error.
func materializeWorkspaceEntries(root *os.Root, entries []ManagedWorkspaceEntry, workingDirectory string, createdFiles *[]string) error {
	for _, entry := range entries {
		if err := materializeWorkspaceEntry(root, entry); err != nil {
			return err
		}
		if entry.NodeType == "file" {
			*createdFiles = append(*createdFiles, filepath.Join(workingDirectory, entry.Path))
		}
	}
	return nil
}

// materializeWorkspaceEntry creates a workspace directory or writes a workspace file
// under root according to the entry definition. Directory entries are created
// recursively; file entries require content.
func materializeWorkspaceEntry(root *os.Root, entry ManagedWorkspaceEntry) error {
	parent := filepath.Dir(entry.Path)
	if parent != "." {
		if err := root.MkdirAll(parent, managedCreateDirMode); err != nil {
			return fmt.Errorf("create workspace parent: %w", err)
		}
	}
	if entry.NodeType == "directory" {
		if err := root.MkdirAll(entry.Path, managedCreateDirMode); err != nil {
			return fmt.Errorf("create workspace directory: %w", err)
		}
		return nil
	}
	if entry.Content == nil {
		return fmt.Errorf("write workspace file: content is required")
	}
	if err := root.WriteFile(entry.Path, []byte(*entry.Content), managedCreateFileMode); err != nil {
		return fmt.Errorf("write workspace file: %w", err)
	}
	return nil
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
