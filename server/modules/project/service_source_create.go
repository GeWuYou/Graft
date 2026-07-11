package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
)

const (
	defaultTemplateKey     = "empty-compose"
	defaultTemplateVersion = "v1"
	maxGitWorkspaceFiles   = 128
	maxGitWorkspaceBytes   = 1 << 20
)

// GitProjectCreateRequest contains the safe, credential-free inputs for a Git-backed project.
type GitProjectCreateRequest struct {
	DisplayName              string
	CanonicalProjectName     string
	RelativeProjectDirectory string
	RepositoryURL            string
	Reference                string
	ComposeSubpath           string
	LifecycleConfig          *LifecycleStandardConfig
}

// TemplateProjectCreateRequest selects one bundled, module-owned template.
type TemplateProjectCreateRequest struct {
	DisplayName              string
	CanonicalProjectName     string
	RelativeProjectDirectory string
	TemplateKey              string
	TemplateVersion          string
	TemplateInstanceName     string
	LifecycleConfig          *LifecycleStandardConfig
}

// CreateGitProject resolves a public or locally reachable repository in an isolated staging directory.
// Credentials are intentionally neither accepted nor persisted by this source adapter.
func (s *Service) CreateGitProject(ctx context.Context, request GitProjectCreateRequest, actorID *uint64) (ManagedProjectCreateResult, error) {
	workspace, metadata, err := resolveGitWorkspace(ctx, request)
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	return s.createMaterializedSourceProject(ctx, workspace, projectcontract.SourceKindGit.String(), metadata, actorID)
}

// ValidateGitProject resolves and validates a Git workspace without writing to the managed root.
func (s *Service) ValidateGitProject(ctx context.Context, request GitProjectCreateRequest) (ManagedProjectCreateValidationResult, error) {
	workspace, metadata, err := resolveGitWorkspace(ctx, request)
	if err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	return s.validateMaterializedSource(ctx, workspace, projectcontract.SourceKindGit.String(), metadata)
}

// CreateTemplateProject materializes a bundled template and registers it through the shared pipeline.
func (s *Service) CreateTemplateProject(ctx context.Context, request TemplateProjectCreateRequest, actorID *uint64) (ManagedProjectCreateResult, error) {
	workspace, metadata, err := resolveTemplateWorkspace(request)
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	return s.createMaterializedSourceProject(ctx, workspace, projectcontract.SourceKindTemplate.String(), metadata, actorID)
}

// ValidateTemplateProject checks a bundled template and its eventual managed-root target without writing it.
func (s *Service) ValidateTemplateProject(ctx context.Context, request TemplateProjectCreateRequest) (ManagedProjectCreateValidationResult, error) {
	workspace, metadata, err := resolveTemplateWorkspace(request)
	if err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	return s.validateMaterializedSource(ctx, workspace, projectcontract.SourceKindTemplate.String(), metadata)
}

func (s *Service) validateMaterializedSource(ctx context.Context, request ManagedProjectCreateRequest, sourceType string, metadata map[string]string) (ManagedProjectCreateValidationResult, error) {
	result, err := s.ValidateManagedCreate(ctx, request)
	if err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	result.SourceType = sourceType
	result.SourceMetadata = metadata
	result.Warnings = append(result.Warnings, "Source resolution only prepares a workspace; creation does not deploy or start containers.")
	return result, nil
}

func (s *Service) createMaterializedSourceProject(ctx context.Context, request ManagedProjectCreateRequest, sourceType string, metadata map[string]string, actorID *uint64) (result ManagedProjectCreateResult, err error) {
	validation, err := s.validateMaterializedSource(ctx, request, sourceType, metadata)
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	normalized, err := normalizeManagedCreateRequest(request)
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	if err := ensureManagedCreatePathsUnderRoot(validation); err != nil {
		return ManagedProjectCreateResult{}, err
	}
	createdDir, createdFiles, err := writeManagedProjectFiles(validation, normalized)
	if err != nil {
		return ManagedProjectCreateResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			err = errors.Join(err, cleanupManagedCreate(createdDir, createdFiles))
		}
	}()
	parseResult, err := projectcompose.Load(projectcompose.Input{WorkingDirectory: validation.WorkingDirectory, ComposeFiles: []string{validation.ComposeFileAbsolutePath}, EnvFiles: managedCreateEnvFileList(validation.EnvFileAbsolutePath)})
	if err != nil {
		return ManagedProjectCreateResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	aggregate, now, err := s.createProjectFromWorkspace(ctx, CreationCommand{DisplayName: normalized.DisplayName, CanonicalProjectName: normalized.CanonicalProjectName, CanonicalProjectNameSource: projectcontract.CanonicalProjectNameSourceOverride.String(), SourceKind: sourceType, HostScope: projectcontract.HostScopeLocal.String(), WorkingDirectory: validation.WorkingDirectory, OwnershipMode: projectcontract.OwnershipModeManagedRootDedicated.String(), SourceMetadata: metadata, LifecycleConfig: defaultManagedLifecycleConfig(normalized.LifecycleConfig), ParseResult: parseResult, ActorID: actorID})
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	cleanup = false
	return ManagedProjectCreateResult{Validation: validation, SourceType: sourceType, ProjectID: aggregate.Project.ID, ConfigHash: parseResult.ConfigHash, DeclaredServiceCount: len(parseResult.ServiceNames), RefreshedAt: now}, nil
}

func resolveTemplateWorkspace(request TemplateProjectCreateRequest) (ManagedProjectCreateRequest, map[string]string, error) {
	key := strings.TrimSpace(request.TemplateKey)
	if key == "" {
		key = defaultTemplateKey
	}
	version := strings.TrimSpace(request.TemplateVersion)
	if version == "" {
		version = defaultTemplateVersion
	}
	if key != defaultTemplateKey || version != defaultTemplateVersion {
		return ManagedProjectCreateRequest{}, nil, fmt.Errorf("%w: unknown template key or version", errProjectInvalidArgument)
	}
	instance := strings.TrimSpace(request.TemplateInstanceName)
	if instance == "" {
		instance = request.CanonicalProjectName
	}
	return ManagedProjectCreateRequest{DisplayName: request.DisplayName, CanonicalProjectName: request.CanonicalProjectName, RelativeProjectDirectory: request.RelativeProjectDirectory, ComposeFileName: "compose.yaml", ComposeFileContent: "services: {}\n", EnvFileName: stringPointer(".env"), EnvFileContent: stringPointer("\n"), WorkspaceFiles: []ManagedWorkspaceFile{{Path: "compose.yaml", Content: "services: {}\n"}, {Path: ".env", Content: "\n"}}, LifecycleConfig: request.LifecycleConfig}, map[string]string{"template_key": key, "template_version": version, "template_instance_name": instance}, nil
}

func resolveGitWorkspace(ctx context.Context, request GitProjectCreateRequest) (ManagedProjectCreateRequest, map[string]string, error) {
	url := strings.TrimSpace(request.RepositoryURL)
	if url == "" || strings.ContainsAny(url, "\r\n") {
		return ManagedProjectCreateRequest{}, nil, fmt.Errorf("%w: git repository URL is required", errProjectInvalidArgument)
	}
	stage, err := os.MkdirTemp("", "graft-project-git-")
	if err != nil {
		return ManagedProjectCreateRequest{}, nil, fmt.Errorf("create git staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stage) }()
	if output, err := runGit(ctx, "clone", "--no-checkout", "--", url, stage); err != nil {
		return ManagedProjectCreateRequest{}, nil, fmt.Errorf("%w: git clone failed: %s", errProjectImportValidation, strings.TrimSpace(output))
	}
	reference := strings.TrimSpace(request.Reference)
	if reference == "" {
		reference = "HEAD"
	}
	if output, err := runGit(ctx, "-C", stage, "checkout", "--detach", reference); err != nil {
		return ManagedProjectCreateRequest{}, nil, fmt.Errorf("%w: git reference could not be resolved: %s", errProjectImportValidation, strings.TrimSpace(output))
	}
	subpath, err := normalizeGitComposeSubpath(request.ComposeSubpath)
	if err != nil {
		return ManagedProjectCreateRequest{}, nil, err
	}
	workspaceRoot := filepath.Join(stage, subpath)
	resolved, err := readGitWorkspace(workspaceRoot)
	if err != nil {
		return ManagedProjectCreateRequest{}, nil, err
	}
	composeContent := workspaceFileContent(resolved.files, resolved.composePath)
	requestWorkspace := ManagedProjectCreateRequest{DisplayName: request.DisplayName, CanonicalProjectName: request.CanonicalProjectName, RelativeProjectDirectory: request.RelativeProjectDirectory, ComposeFileName: filepath.Base(resolved.composePath), ComposeFileContent: composeContent, ComposeFilePath: resolved.composePath, WorkspaceFiles: resolved.files, LifecycleConfig: request.LifecycleConfig}
	if resolved.envPath != "" {
		envContent := workspaceFileContent(resolved.files, resolved.envPath)
		requestWorkspace.EnvFileName = stringPointer(filepath.Base(resolved.envPath))
		requestWorkspace.EnvFileContent = stringPointer(envContent)
		requestWorkspace.EnvFilePaths = []string{resolved.envPath}
	}
	return requestWorkspace, map[string]string{"git_repository_url": url, "git_reference": reference, "git_compose_subpath": subpath}, nil
}

func runGit(ctx context.Context, args ...string) (string, error) {
	// Arguments are assembled from fixed Git subcommands plus validated source fields; no shell is invoked.
	//nolint:gosec // G204 is intentional for the narrow project-owned Git runner.
	command := exec.CommandContext(ctx, "git", args...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.CombinedOutput()
	return string(output), err
}

func normalizeGitComposeSubpath(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return ".", nil
	}
	return normalizeManagedWorkspacePath(value)
}

type gitWorkspace struct {
	files       []ManagedWorkspaceFile
	composePath string
	envPath     string
}

//nolint:gocognit,gocyclo,cyclop // Filesystem entry validation and source selection stay together to preserve the staging boundary.
func readGitWorkspace(root string) (gitWorkspace, error) {
	files := make([]ManagedWorkspaceFile, 0)
	composePath, envPath := "", ""
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: git workspace may not contain symlinks", errProjectInvalidArgument)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("%w: git workspace contains unsupported file", errProjectInvalidArgument)
		}
		// `path` is emitted by WalkDir beneath the isolated staging root after symlink rejection.
		//nolint:gosec // G304 is constrained to the validated staging tree.
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(content) > maxGitWorkspaceBytes || !utf8.Valid(content) || strings.IndexByte(string(content), 0) >= 0 {
			return fmt.Errorf("%w: git workspace files must be UTF-8 text up to 1 MiB", errProjectInvalidArgument)
		}
		relative = filepath.ToSlash(relative)
		files = append(files, ManagedWorkspaceFile{Path: relative, Content: string(content)})
		if composePath == "" && isComposeFile(filepath.Base(relative)) {
			composePath = relative
		}
		if envPath == "" && filepath.Base(relative) == ".env" {
			envPath = relative
		}
		if len(files) > maxGitWorkspaceFiles {
			return fmt.Errorf("%w: git workspace exceeds %d text files", errProjectInvalidArgument, maxGitWorkspaceFiles)
		}
		return nil
	})
	if err != nil {
		return gitWorkspace{}, fmt.Errorf("read git workspace: %w", err)
	}
	if composePath == "" {
		return gitWorkspace{}, fmt.Errorf("%w: no compose.yaml, compose.yml, docker-compose.yaml, or docker-compose.yml found", errProjectInvalidArgument)
	}
	return gitWorkspace{files: files, composePath: composePath, envPath: envPath}, nil
}

func isComposeFile(name string) bool {
	return name == "compose.yaml" || name == "compose.yml" || name == "docker-compose.yaml" || name == "docker-compose.yml"
}

func workspaceFileContent(files []ManagedWorkspaceFile, path string) string {
	for _, file := range files {
		if file.Path == path {
			return file.Content
		}
	}
	return ""
}
