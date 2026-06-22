// Package main generates the compile-time module registry artifact.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const (
	modulePath              = "graft/server"
	modulesDirName          = "modules"
	registryPkgName         = "moduleregistry"
	descriptorFile          = "descriptor.go"
	generatedFileName       = "generated.go"
	generatedFilePerm       = 0o600
	migrationsDirName       = "migrations"
	hashFileName            = "atlas.sum"
	internalDirName         = "internal"
	httpxMigrationsPath     = "internal/httpx/migrations"
	loggerMigrationsPath    = "internal/logger/migrations"
	drilldownMigrationsPath = "internal/drilldown/migrations"
)

type modulePackage struct {
	importAlias string
	importPath  string
}

type generatedMigrationDir struct {
	path  string
	files []generatedMigrationFile
}

type generatedMigrationFile struct {
	name    string
	content []byte
}

// Main generates a compile-time registry artifact by discovering module packages, collecting SQL migration files, and rendering the result to generated.go.
func main() {
	workdir, err := os.Getwd()
	if err != nil {
		failf("resolve working directory: %v", err)
	}

	modulesRoot := filepath.Clean(filepath.Join(workdir, "..", "..", modulesDirName))
	packages, err := collectModulePackages(modulesRoot)
	if err != nil {
		failf("collect module packages: %v", err)
	}

	migrationDirs, err := collectMigrationDirs(workdir, packages)
	if err != nil {
		failf("collect embedded migration dirs: %v", err)
	}

	content, err := renderGeneratedFile(packages, migrationDirs)
	if err != nil {
		failf("render generated file: %v", err)
	}

	outputPath := filepath.Join(workdir, generatedFileName)
	if err := os.WriteFile(outputPath, content, generatedFilePerm); err != nil {
		failf("write generated file: %v", err)
	}
}

// collectModulePackages 发现指定根目录下的模块包。
// 验证每个目录都包含 descriptor.go，按导入路径升序返回包列表。
func collectModulePackages(modulesRoot string) ([]modulePackage, error) {
	entries, err := os.ReadDir(modulesRoot)
	if err != nil {
		return nil, err
	}

	packages := make([]modulePackage, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}

		moduleDir := filepath.Join(modulesRoot, name)
		if !fileExists(filepath.Join(moduleDir, descriptorFile)) {
			return nil, fmt.Errorf("module package %s is missing %s", name, descriptorFile)
		}

		packages = append(packages, modulePackage{
			importAlias: sanitizeImportAlias(name) + "module",
			importPath:  modulePath + "/" + filepath.ToSlash(filepath.Join(modulesDirName, name)),
		})
	}

	sort.Slice(packages, func(left int, right int) bool {
		return packages[left].importPath < packages[right].importPath
	})
	return packages, nil
}

// collectMigrationDirs 收集来自内置迁移位置和各模块 migrations 目录中的迁移文件信息。缺失的目录会被跳过，返回的目录列表按路径升序排列。若读取过程中出错，返回该错误。
func collectMigrationDirs(workdir string, packages []modulePackage) ([]generatedMigrationDir, error) {
	serverRoot := filepath.Clean(filepath.Join(workdir, "..", ".."))

	paths := []string{
		httpxMigrationsPath,
		loggerMigrationsPath,
		drilldownMigrationsPath,
	}
	for _, pkg := range packages {
		moduleName := filepath.Base(pkg.importPath)
		paths = append(paths, filepath.ToSlash(filepath.Join(modulesDirName, moduleName, migrationsDirName)))
	}

	dirs := make([]generatedMigrationDir, 0, len(paths))
	for _, current := range paths {
		dir, ok, err := readMigrationDir(serverRoot, current)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		dirs = append(dirs, dir)
	}

	sort.Slice(dirs, func(left int, right int) bool {
		return dirs[left].path < dirs[right].path
	})
	return dirs, nil
}

// readMigrationDir 从迁移目录加载 SQL 文件和哈希文件。
// 目录不存在时返回 false，不作为错误。
// 路径存在但不是目录时返回错误。
// 仅收集扩展名为 .sql 的文件和 atlas.sum，按文件名升序排序后返回。
func readMigrationDir(serverRoot string, relativePath string) (generatedMigrationDir, bool, error) {
	absDir := filepath.Join(serverRoot, filepath.FromSlash(relativePath))
	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return generatedMigrationDir{}, false, nil
		}
		return generatedMigrationDir{}, false, err
	}
	if !info.IsDir() {
		return generatedMigrationDir{}, false, fmt.Errorf("migration path %s is not a directory", relativePath)
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return generatedMigrationDir{}, false, err
	}

	files := make([]generatedMigrationFile, 0, len(entries))
	for _, entry := range entries {
		file, ok, err := loadGeneratedMigrationFile(absDir, relativePath, entry)
		if err != nil {
			return generatedMigrationDir{}, false, err
		}
		if !ok {
			continue
		}
		files = append(files, file)
	}

	sort.Slice(files, func(left int, right int) bool {
		return files[left].name < files[right].name
	})

	return generatedMigrationDir{
		path:  filepath.ToSlash(relativePath),
		files: files,
	}, true, nil
}

func validateRegularMigrationFile(contentPath string, relativePath string, name string) error {
	fileInfo, err := os.Lstat(contentPath)
	if err != nil {
		return err
	}
	if fileInfo.Mode().IsRegular() {
		return nil
	}
	return fmt.Errorf(
		"migration file %s is not a regular file",
		filepath.ToSlash(filepath.Join(relativePath, name)),
	)
}

func loadGeneratedMigrationFile(absDir string, relativePath string, entry os.DirEntry) (generatedMigrationFile, bool, error) {
	if entry.IsDir() {
		return generatedMigrationFile{}, false, nil
	}

	name := entry.Name()
	if filepath.Ext(name) != ".sql" && name != hashFileName {
		return generatedMigrationFile{}, false, nil
	}

	contentPath := filepath.Join(absDir, name)
	if err := validateRegularMigrationFile(contentPath, relativePath, name); err != nil {
		return generatedMigrationFile{}, false, err
	}

	// Only reads files discovered from a repository-owned migration directory listing.
	// #nosec G304 -- contentPath is derived from a repository-owned migration directory listing under absDir.
	content, err := os.ReadFile(contentPath)
	if err != nil {
		return generatedMigrationFile{}, false, err
	}

	return generatedMigrationFile{
		name:    name,
		content: content,
	}, true, nil
}

// RenderGeneratedFile generates Go source code containing module specifications and embedded migration directories.
func renderGeneratedFile(packages []modulePackage, migrationDirs []generatedMigrationDir) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("// Code generated by go generate; DO NOT EDIT.\n")
	buffer.WriteString("package " + registryPkgName + "\n\n")
	buffer.WriteString("import (\n")
	buffer.WriteString("\t\"graft/server/internal/module\"\n")
	for _, current := range packages {
		_, _ = fmt.Fprintf(&buffer, "\t%s %q\n", current.importAlias, current.importPath)
	}
	buffer.WriteString(")\n\n")
	buffer.WriteString("var generatedModuleSpecs = []module.Spec{\n")
	for _, current := range packages {
		_, _ = fmt.Fprintf(&buffer, "\t%s.NewModuleSpec(),\n", current.importAlias)
	}
	buffer.WriteString("}\n")
	buffer.WriteString("\n")
	buffer.WriteString("var generatedEmbeddedMigrationDirs = []EmbeddedMigrationDir{\n")
	for _, dir := range migrationDirs {
		_, _ = fmt.Fprintf(&buffer, "\t{\n\t\tPath: %q,\n\t\tFiles: []EmbeddedMigrationFile{\n", dir.path)
		for _, file := range dir.files {
			_, _ = fmt.Fprintf(
				&buffer,
				"\t\t\t{Name: %q, Contents: []byte(%q)},\n",
				file.name,
				string(file.content),
			)
		}
		buffer.WriteString("\t\t},\n\t},\n")
	}
	buffer.WriteString("}\n")

	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, err
	}

	return formatted, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func sanitizeImportAlias(name string) string {
	var builder strings.Builder
	for _, current := range name {
		current = unicode.ToLower(current)
		if (current >= 'a' && current <= 'z') || (current >= '0' && current <= '9') {
			builder.WriteRune(current)
		} else {
			builder.WriteRune('_')
		}
	}

	alias := strings.Trim(builder.String(), "_")
	if alias == "" {
		return "modulepkg"
	}
	if alias[0] >= '0' && alias[0] <= '9' {
		return "modulepkg_" + alias
	}

	return alias
}

func failf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
