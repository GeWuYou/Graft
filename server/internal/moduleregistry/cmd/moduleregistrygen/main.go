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
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".sql" && name != hashFileName {
			continue
		}

		contentPath := filepath.Join(absDir, name)
		// Only reads files discovered from a repository-owned migration directory listing.
		// #nosec G304 -- contentPath is derived from a repository-owned migration directory listing under absDir.
		content, err := os.ReadFile(contentPath)
		if err != nil {
			return generatedMigrationDir{}, false, err
		}

		files = append(files, generatedMigrationFile{
			name:    name,
			content: content,
		})
	}

	sort.Slice(files, func(left int, right int) bool {
		return files[left].name < files[right].name
	})

	return generatedMigrationDir{
		path:  filepath.ToSlash(relativePath),
		files: files,
	}, true, nil
}

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
