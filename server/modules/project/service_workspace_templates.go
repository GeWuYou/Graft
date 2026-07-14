package project

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	projectcontract "graft/server/modules/project/contract"
)

const (
	defaultTemplateKey     = "default"
	defaultTemplateVersion = "runtime"
	maxWorkspaceEntryCount = 1024
	maxWorkspaceFileBytes  = 1 << 20
	maxWorkspaceTotalBytes = 10 << 20
)

var defaultWorkspaceTemplateSeedMu sync.Mutex

//go:embed all:templates/default
var bundledWorkspaceTemplates embed.FS

// WorkspaceEntry is a generic text-file or directory entry in a project workspace.
// File names are intentionally not restricted by extension; only path and text safety rules apply.
type WorkspaceEntry struct {
	Path     string
	NodeType string
	Content  *string
}

// WorkspaceTemplate describes one operator-managed template directory.
type WorkspaceTemplate struct {
	Key         string
	DisplayName string
}

// WorkspaceDefaultsResult is the server-owned initial workspace model for blank creation.
type WorkspaceDefaultsResult struct {
	Templates          []WorkspaceTemplate
	DefaultTemplateKey string
	WorkspaceEntries   []WorkspaceEntry
	ComposeFilePath    string
}

// WorkspaceDefaults returns the server-owned initial workspace and available runtime templates for blank creation.
func (s *Service) WorkspaceDefaults(ctx context.Context) (WorkspaceDefaultsResult, error) {
	root, err := s.applicationRootDirectory(ctx)
	if err != nil {
		return WorkspaceDefaultsResult{}, err
	}
	if err := seedDefaultWorkspaceTemplate(root); err != nil {
		return WorkspaceDefaultsResult{}, err
	}
	templates, err := listWorkspaceTemplates(root)
	if err != nil {
		return WorkspaceDefaultsResult{}, err
	}
	prefill, err := s.blankCreatePrefillDefaultTemplate(ctx)
	if err != nil {
		return WorkspaceDefaultsResult{}, err
	}
	entries := blankWorkspaceEntries()
	if prefill {
		entries, err = loadWorkspaceTemplate(root, defaultTemplateKey)
		if err != nil {
			return WorkspaceDefaultsResult{}, err
		}
	}
	return WorkspaceDefaultsResult{Templates: templates, DefaultTemplateKey: defaultTemplateKey, WorkspaceEntries: entries, ComposeFilePath: "compose.yaml"}, nil
}

func (s *Service) applicationRootDirectory(ctx context.Context) (string, error) {
	if s == nil || s.configResolver == nil {
		return defaultApplicationRootDirectory, nil
	}
	raw, err := s.configResolver.ResolveDefaultConfig(ctx, projectcontract.ApplicationRootDirectoryConfig.String())
	if err != nil {
		return "", fmt.Errorf("%w: application root is unavailable", errProjectInvalidArgument)
	}
	var root string
	if json.Unmarshal([]byte(raw), &root) != nil {
		root = raw
	}
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." || !filepath.IsAbs(root) {
		return "", errProjectInvalidArgument
	}
	return root, nil
}

func (s *Service) blankCreatePrefillDefaultTemplate(ctx context.Context) (bool, error) {
	if s == nil || s.configResolver == nil {
		return true, nil
	}
	raw, err := s.configResolver.ResolveDefaultConfig(ctx, projectcontract.ProjectBlankCreatePrefillDefaultTemplateConfig.String())
	if err != nil {
		return false, fmt.Errorf("%w: resolve blank-create template prefill: %v", errProjectInvalidArgument, err)
	}
	var enabled bool
	if err := json.Unmarshal([]byte(raw), &enabled); err != nil {
		return false, fmt.Errorf("%w: decode blank-create template prefill: %v", errProjectInvalidArgument, err)
	}
	return enabled, nil
}

// blankWorkspaceEntries 返回包含空内容的初始 .env 和 compose.yaml 文件条目。
func blankWorkspaceEntries() []WorkspaceEntry {
	empty := ""
	return []WorkspaceEntry{{Path: ".env", NodeType: "file", Content: &empty}, {Path: "compose.yaml", NodeType: "file", Content: &empty}}
}

// templateRoot 返回应用根目录下的模板目录路径。
func templateRoot(applicationRoot string) string { return filepath.Join(applicationRoot, "templates") }

// seedDefaultWorkspaceTemplate adds missing bundled files to the default workspace template without overwriting existing files. It returns an error if the bundled files cannot be read or written, or if the target files cannot be inspected.
func seedDefaultWorkspaceTemplate(applicationRoot string) error {
	defaultWorkspaceTemplateSeedMu.Lock()
	defer defaultWorkspaceTemplateSeedMu.Unlock()
	targetRoot := filepath.Join(templateRoot(applicationRoot), defaultTemplateKey)
	for _, bundledPath := range []string{"templates/default/compose.yaml", "templates/default/.env"} {
		if err := seedBundledWorkspaceTemplateFile(targetRoot, bundledPath); err != nil {
			return err
		}
	}
	return nil
}

func seedBundledWorkspaceTemplateFile(targetRoot, bundledPath string) error {
	content, err := bundledWorkspaceTemplates.ReadFile(bundledPath)
	if err != nil {
		return fmt.Errorf("read bundled workspace template: %w", err)
	}
	relative := strings.TrimPrefix(bundledPath, "templates/default/")
	target := filepath.Join(targetRoot, relative)
	if err := os.MkdirAll(filepath.Dir(target), managedCreateDirMode); err != nil {
		return fmt.Errorf("create default template directory: %w", err)
	}
	if exists, err := regularTemplateFileExists(target, relative); err != nil || exists {
		return err
	}
	return publishNewWorkspaceTemplateFile(target, content)
}

func regularTemplateFileExists(target, relative string) (bool, error) {
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect default template file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("inspect default template file: %s is not a regular file", relative)
	}
	return true, nil
}

func publishNewWorkspaceTemplateFile(target string, content []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(target), ".graft-template-*")
	if err != nil {
		return fmt.Errorf("create default template temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := writeWorkspaceTemplateTemporaryFile(temporary, content); err != nil {
		return err
	}
	if err := os.Link(temporaryPath, target); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("publish default template file: %w", err)
	}
	_, err = regularTemplateFileExists(target, filepath.Base(target))
	return err
}

func writeWorkspaceTemplateTemporaryFile(temporary *os.File, content []byte) error {
	if err := temporary.Chmod(managedCreateFileMode); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set default template file mode: %w", err)
	}
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write default template temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync default template temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close default template temporary file: %w", err)
	}
	return nil
}

// listWorkspaceTemplates lists valid workspace templates under the application root,
// sorted by template key. It returns an error if the template directory cannot be read.
func listWorkspaceTemplates(applicationRoot string) ([]WorkspaceTemplate, error) {
	entries, err := os.ReadDir(templateRoot(applicationRoot))
	if err != nil {
		return nil, fmt.Errorf("read template directory: %w", err)
	}
	result := make([]WorkspaceTemplate, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !validTemplateKey(entry.Name()) {
			continue
		}
		result = append(result, WorkspaceTemplate{Key: entry.Name(), DisplayName: entry.Name()})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result, nil
}

// validTemplateKey reports whether the trimmed key matches the workspace template key format.
func validTemplateKey(key string) bool {
	key = strings.TrimSpace(key)
	return workspaceKeyPattern.MatchString(key)
}

// loadWorkspaceTemplate 加载指定工作区模板中的文件和目录条目，并按路径排序。
// key 必须是有效的模板标识。
// 返回模板条目；加载失败时返回错误。
func loadWorkspaceTemplate(applicationRoot, key string) ([]WorkspaceEntry, error) {
	if !validTemplateKey(key) {
		return nil, errProjectInvalidArgument
	}
	root := filepath.Join(templateRoot(applicationRoot), key)
	entries := make([]WorkspaceEntry, 0)
	totalBytes := int64(0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		item, skip, itemErr := workspaceTemplateEntry(root, path, entry, walkErr)
		if itemErr != nil || skip {
			return itemErr
		}
		if len(entries) >= maxWorkspaceEntryCount {
			return fmt.Errorf("template exceeds workspace entry limit")
		}
		if item.NodeType == "file" {
			totalBytes += int64(len(stringValue(item.Content)))
			if totalBytes > maxWorkspaceTotalBytes {
				return fmt.Errorf("template exceeds workspace total size limit")
			}
		}
		entries = append(entries, item)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load workspace template: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

// workspaceTemplateEntry converts a template filesystem entry into a workspace entry.
// It skips the template root and rejects symlinks and files that are not valid UTF-8
// text or contain NUL bytes. It returns whether the entry should be skipped and any
// error encountered while processing it.
func workspaceTemplateEntry(root, path string, entry fs.DirEntry, walkErr error) (WorkspaceEntry, bool, error) {
	if walkErr != nil {
		return WorkspaceEntry{}, false, walkErr
	}
	normalized, skip, err := normalizedWorkspaceTemplatePath(root, path, entry)
	if err != nil {
		return WorkspaceEntry{}, false, err
	}
	if skip {
		return WorkspaceEntry{}, true, nil
	}
	if entry.IsDir() {
		return WorkspaceEntry{Path: normalized, NodeType: "directory"}, false, nil
	}
	if err := validateWorkspaceTemplateFile(entry, normalized); err != nil {
		return WorkspaceEntry{}, false, err
	}
	// #nosec G304 -- WalkDir path is anchored to the validated Application Root template directory and rejects symlinks.
	content, err := os.ReadFile(path)
	if err != nil {
		return WorkspaceEntry{}, false, err
	}
	if !utf8.Valid(content) || strings.IndexByte(string(content), 0) >= 0 {
		return WorkspaceEntry{}, false, fmt.Errorf("template file %s is not UTF-8 text", normalized)
	}
	value := string(content)
	return WorkspaceEntry{Path: normalized, NodeType: "file", Content: &value}, false, nil
}

func normalizedWorkspaceTemplatePath(root, path string, entry fs.DirEntry) (string, bool, error) {
	if path == root {
		return "", true, nil
	}
	if entry.Type()&os.ModeSymlink != 0 {
		return "", false, fmt.Errorf("template symlinks are not allowed")
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return "", false, err
	}
	normalized, err := normalizeManagedWorkspacePath(relative)
	if err != nil {
		return "", false, err
	}
	return normalized, false, nil
}

func validateWorkspaceTemplateFile(entry fs.DirEntry, normalized string) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("template file %s is not regular", normalized)
	}
	if info.Size() > maxWorkspaceFileBytes {
		return fmt.Errorf("template file %s exceeds size limit", normalized)
	}
	return nil
}
