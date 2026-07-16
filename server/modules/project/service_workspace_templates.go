package project

import (
	"context"
	"encoding/json"
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
	maxWorkspaceEntryCount = 1024
	maxWorkspaceFileBytes  = 1 << 20
	maxWorkspaceTotalBytes = 10 << 20
)

// WorkspaceEntry 表示项目工作区中的通用文本文件或目录条目。
// 文件名不按扩展名限制，仅执行路径安全和文本内容安全校验。
type WorkspaceEntry struct {
	Path     string
	NodeType string
	Content  *string
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

// templateRoot 返回应用根目录下的模板目录路径。
func templateRoot(applicationRoot string) string { return filepath.Join(applicationRoot, "templates") }

// loadWorkspaceTemplate 加载指定工作区模板中的文件和目录条目，并按路径排序。
// key 必须是有效的模板标识。
// 返回模板条目；加载失败时返回错误。
func loadWorkspaceTemplate(applicationRoot, key string) ([]WorkspaceEntry, error) {
	key = strings.TrimSpace(key)
	if !applicationNamePattern.MatchString(key) {
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

// workspaceTemplateEntry 将模板文件系统条目转换为工作区条目。
// 它跳过模板根目录，拒绝符号链接、非 UTF-8 文本和包含空字节的文件，并返回是否跳过及处理错误。
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
	// #nosec G304 -- WalkDir 路径锚定在已校验的 Application Root 模板目录，并拒绝符号链接。
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
