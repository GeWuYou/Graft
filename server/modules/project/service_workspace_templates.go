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
	"unicode/utf8"

	projectcontract "graft/server/modules/project/contract"
)

const (
	defaultTemplateKey     = "default"
	defaultTemplateVersion = "runtime"
)

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
		return true, nil
	}
	var enabled bool
	if err := json.Unmarshal([]byte(raw), &enabled); err != nil {
		return true, nil
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
	targetRoot := filepath.Join(templateRoot(applicationRoot), defaultTemplateKey)
	for _, bundledPath := range []string{"templates/default/compose.yaml", "templates/default/.env"} {
		content, err := bundledWorkspaceTemplates.ReadFile(bundledPath)
		if err != nil {
			return fmt.Errorf("read bundled workspace template: %w", err)
		}
		relative := strings.TrimPrefix(bundledPath, "templates/default/")
		target := filepath.Join(targetRoot, relative)
		if err := os.MkdirAll(filepath.Dir(target), managedCreateDirMode); err != nil {
			return fmt.Errorf("create default template directory: %w", err)
		}
		_, err = os.Lstat(target)
		if err == nil {
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect default template file: %w", err)
		}
		if err := os.WriteFile(target, content, managedCreateFileMode); err != nil {
			return fmt.Errorf("seed default template file: %w", err)
		}
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
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		item, skip, itemErr := workspaceTemplateEntry(root, path, entry, walkErr)
		if itemErr != nil || skip {
			return itemErr
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
	if path == root {
		return WorkspaceEntry{}, true, nil
	}
	if entry.Type()&os.ModeSymlink != 0 {
		return WorkspaceEntry{}, false, fmt.Errorf("template symlinks are not allowed")
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return WorkspaceEntry{}, false, err
	}
	normalized, err := normalizeManagedWorkspacePath(relative)
	if err != nil {
		return WorkspaceEntry{}, false, err
	}
	if entry.IsDir() {
		return WorkspaceEntry{Path: normalized, NodeType: "directory"}, false, nil
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
