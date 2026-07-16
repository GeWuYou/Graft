package project

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

const (
	applicationNameAvailabilityAvailable  = "available"
	applicationNameAvailabilityRegistered = "registered"
	applicationNameAvailabilityReusable   = "reusable_workspace"
)

// ApplicationNameAvailabilityRequest 描述受管创建的第一步应用名称预检请求。
type ApplicationNameAvailabilityRequest struct {
	ApplicationName string
}

// ApplicationNameAvailabilityResult 区分注册表已拥有名称与尚未登记的受管目录。
type ApplicationNameAvailabilityResult struct {
	Status            string
	WorkspacePath     string
	WorkspaceEntries  []ManagedWorkspaceEntry
	ComposeFilePath   *string
	WorkspaceNonEmpty bool
}

type reusableWorkspaceResult struct {
	entries         []ManagedWorkspaceEntry
	composeFilePath *string
	nonEmpty        bool
	exists          bool
}

// CheckApplicationNameAvailability 在开始编辑工作区前解析受管根目录下的应用名称。
func (s *Service) CheckApplicationNameAvailability(ctx context.Context, request ApplicationNameAvailabilityRequest) (ApplicationNameAvailabilityResult, error) {
	name, err := normalizeApplicationName(stringPointer(request.ApplicationName))
	if err != nil {
		return ApplicationNameAvailabilityResult{}, err
	}
	rootInfo, err := s.ManagedRoot(ctx)
	if err != nil {
		return ApplicationNameAvailabilityResult{}, err
	}
	if rootInfo.Status != projectcontract.ManagedRootStatusReady.String() || rootInfo.ConfiguredRootDirectory == nil {
		return ApplicationNameAvailabilityResult{}, fmt.Errorf("%w: managed root is unavailable", errProjectInvalidArgument)
	}
	if err := s.ensureApplicationNameUnregistered(ctx, *name); err != nil {
		if errors.Is(err, errProjectApplicationNameOccupied) {
			return ApplicationNameAvailabilityResult{Status: applicationNameAvailabilityRegistered}, nil
		}
		return ApplicationNameAvailabilityResult{}, err
	}
	workspacePath := filepath.Join(*rootInfo.ConfiguredRootDirectory, *name)
	workspace, err := readReusableWorkspace(workspacePath)
	if err != nil {
		return ApplicationNameAvailabilityResult{}, err
	}
	if !workspace.exists {
		return ApplicationNameAvailabilityResult{Status: applicationNameAvailabilityAvailable, WorkspacePath: workspacePath}, nil
	}
	return ApplicationNameAvailabilityResult{Status: applicationNameAvailabilityReusable, WorkspacePath: workspacePath, WorkspaceEntries: workspace.entries, ComposeFilePath: workspace.composeFilePath, WorkspaceNonEmpty: workspace.nonEmpty}, nil
}

func (s *Service) ensureApplicationNameUnregistered(ctx context.Context, name string) error {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return err
	}
	lookup, ok := repository.(projectstore.ApplicationNameLookupRepository)
	if !ok {
		return nil
	}
	_, err = lookup.GetByApplicationName(ctx, name)
	if err == nil {
		return errProjectApplicationNameOccupied
	}
	if errors.Is(err, projectstore.ErrProjectNotFound) {
		return nil
	}
	return mapStoreError(err)
}

func readReusableWorkspace(root string) (reusableWorkspaceResult, error) {
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return reusableWorkspaceResult{}, nil
	}
	if err != nil {
		return reusableWorkspaceResult{}, managedWorkspaceUnsafeError("workspace path unavailable")
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return reusableWorkspaceResult{}, managedWorkspaceUnsafeError("workspace path is not a safe directory")
	}
	rootFS, err := os.OpenRoot(root)
	if err != nil {
		return reusableWorkspaceResult{}, managedWorkspaceUnsafeError("open managed workspace")
	}
	defer func() { _ = rootFS.Close() }()
	collector := reusableWorkspaceCollector{root: rootFS, rootPath: root, entries: make([]ManagedWorkspaceEntry, 0), composeCandidates: make([]string, 0, 1)}
	err = filepath.WalkDir(root, collector.visit)
	if err != nil {
		return reusableWorkspaceResult{}, fmt.Errorf("%w: %v", errors.Join(errProjectInvalidArgument, errProjectWorkspaceUnsafe), err)
	}
	sort.Slice(collector.entries, func(i, j int) bool { return collector.entries[i].Path < collector.entries[j].Path })
	if len(collector.composeCandidates) == 1 {
		return reusableWorkspaceResult{entries: collector.entries, composeFilePath: stringPointer(collector.composeCandidates[0]), nonEmpty: len(collector.entries) > 0, exists: true}, nil
	}
	return reusableWorkspaceResult{entries: collector.entries, nonEmpty: len(collector.entries) > 0, exists: true}, nil
}

func managedWorkspaceUnsafeError(reason string) error {
	return fmt.Errorf("%w: %s", errors.Join(errProjectInvalidArgument, errProjectWorkspaceUnsafe), reason)
}

type reusableWorkspaceCollector struct {
	root              *os.Root
	rootPath          string
	entries           []ManagedWorkspaceEntry
	composeCandidates []string
	totalBytes        int
}

func (c *reusableWorkspaceCollector) visit(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if path == c.rootPath {
		return nil
	}
	relative, err := filepath.Rel(c.rootPath, path)
	if err != nil {
		return err
	}
	relative = filepath.ToSlash(relative)
	if entry.Type()&os.ModeSymlink != 0 {
		return fmt.Errorf("workspace contains a symbolic link")
	}
	if entry.IsDir() {
		c.entries = append(c.entries, ManagedWorkspaceEntry{Path: relative, NodeType: "directory"})
		return nil
	}
	if !entry.Type().IsRegular() {
		return fmt.Errorf("workspace contains an unsupported entry")
	}
	return c.addFile(relative, entry)
}

func (c *reusableWorkspaceCollector) addFile(relative string, entry fs.DirEntry) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}
	if info.Size() > maxWorkspaceFileBytes || c.totalBytes+int(info.Size()) > maxWorkspaceTotalBytes {
		return fmt.Errorf("workspace exceeds managed-create limits")
	}
	if len(c.entries) >= maxWorkspaceEntryCount {
		return fmt.Errorf("workspace exceeds managed-create limits")
	}
	content, err := c.root.ReadFile(relative)
	if err != nil || !utf8.Valid(content) || bytes.Contains(content, []byte{0}) {
		return fmt.Errorf("workspace contains a non-text file")
	}
	c.totalBytes += len(content)
	text := string(content)
	c.entries = append(c.entries, ManagedWorkspaceEntry{Path: relative, NodeType: "file", Content: &text})
	if isReusableComposeFile(relative) {
		c.composeCandidates = append(c.composeCandidates, relative)
	}
	return nil
}

func isReusableComposeFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return base == "compose.yaml" || base == "compose.yml" || base == "docker-compose.yaml" || base == "docker-compose.yml"
}
