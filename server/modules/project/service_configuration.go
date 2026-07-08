package project

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

const configurationDiffWarningsCapacity = 2

// ConfigurationMetadata returns readonly configuration metadata.
func (s *Service) ConfigurationMetadata(ctx context.Context, projectID uint64) (ConfigurationMetadataResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationMetadataResult{}, err
	}
	return ConfigurationMetadataResult{
		ProjectID:          projectID,
		ComposeFiles:       toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())),
		EnvFiles:           toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindEnv.String())),
		OwnershipMode:      aggregate.Project.OwnershipMode,
		DriftStatus:        aggregate.Project.DriftStatus,
		DiagnosticsSummary: nil,
	}, nil
}

// ConfigurationPreview returns the latest static normalized compose preview.
func (s *Service) ConfigurationPreview(ctx context.Context, projectID uint64) (ConfigurationPreviewResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationPreviewResult{}, err
	}
	parseResult, err := s.loadFromAggregate(aggregate)
	if err != nil {
		return ConfigurationPreviewResult{}, err
	}
	return ConfigurationPreviewResult{
		ProjectID:             projectID,
		CanonicalProjectName:  aggregate.Project.CanonicalProjectName,
		ConfigHash:            parseResult.ConfigHash,
		NormalizedComposeYAML: parseResult.NormalizedComposeYAML,
		RefreshedAt:           snapshotRefreshedAt(aggregate.Snapshot),
	}, nil
}

// DiffConfiguration compares the current saved project files or draft overrides with the latest refreshed snapshot baseline.
func (s *Service) DiffConfiguration(
	ctx context.Context,
	projectID uint64,
	request ConfigurationDiffRequest,
) (ConfigurationDiffResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	overrides, err := buildConfigurationDiffOverrides(aggregate.Project.WorkingDirectory, request)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	files, hasFileChanges, err := buildConfigurationDiffFiles(aggregate, request, overrides)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	parseResult, err := s.loadFromAggregateWithOverrides(aggregate, overrides)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	warnings := configurationDiffWarnings(len(aggregate.Files))
	return ConfigurationDiffResult{
		ProjectID:            projectID,
		CanonicalProjectName: aggregate.Project.CanonicalProjectName,
		OwnershipMode:        aggregate.Project.OwnershipMode,
		CurrentConfigHash:    snapshotConfigHash(aggregate.Snapshot),
		ProposedConfigHash:   parseResult.ConfigHash,
		HasChanges:           hasFileChanges || snapshotConfigHash(aggregate.Snapshot) != parseResult.ConfigHash,
		Files:                files,
		Warnings:             warnings,
	}, nil
}

// ValidateConfiguration validates the current saved project files without mutating them.
func (s *Service) ValidateConfiguration(ctx context.Context, projectID uint64) (ConfigurationValidateResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationValidateResult{}, err
	}
	parseResult, err := s.loadFromAggregate(aggregate)
	if err != nil {
		return ConfigurationValidateResult{}, err
	}
	return ConfigurationValidateResult{
		ProjectID:             projectID,
		CanonicalProjectName:  aggregate.Project.CanonicalProjectName,
		OwnershipMode:         aggregate.Project.OwnershipMode,
		ProposedConfigHash:    parseResult.ConfigHash,
		NormalizedComposeYAML: parseResult.NormalizedComposeYAML,
		DeclaredServiceNames:  append([]string(nil), parseResult.ServiceNames...),
		Warnings:              nil,
	}, nil
}

// DeployConfiguration refreshes the saved project state and executes docker compose up -d using the current disk files.
func (s *Service) DeployConfiguration(
	ctx context.Context,
	projectID uint64,
	actorID *uint64,
) (result DeployResult, err error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return DeployResult{}, err
	}
	parseResult, err := s.loadFromAggregate(aggregate)
	if err != nil {
		return DeployResult{}, err
	}
	repository, err := s.repositoryOrErr()
	if err != nil {
		return DeployResult{}, err
	}
	now := time.Now().UTC()
	updated, err := repository.RefreshProject(ctx, buildRefreshProjectInput(projectID, parseResult, now, actorID))
	if err != nil {
		return DeployResult{}, mapStoreError(err)
	}
	upArgs, err := lifecycleUpArgs(updated, lifecycleConfigurationFromAggregate(updated))
	if err != nil {
		return DeployResult{}, err
	}
	if _, err := s.executeLifecycleActionWithAggregate(ctx, updated, generated.ProjectActionResponseActionProjectActionDeploy, upArgs); err != nil {
		return DeployResult{}, err
	}
	messageKey := projectcontract.ProjectDeployCompleted.String()
	result = DeployResult{
		ProjectID:            projectID,
		Action:               "deploy",
		Result:               "completed",
		CanonicalProjectName: updated.Project.CanonicalProjectName,
		OwnershipMode:        updated.Project.OwnershipMode,
		ConfigHash:           parseResult.ConfigHash,
		RefreshedAt:          now,
		DeclaredServiceCount: len(parseResult.ServiceNames),
		MessageKey:           &messageKey,
		Message:              &messageKey,
		GuardResults: []GuardResult{
			guardDetail("command", strings.Join(upArgs, " ")),
			guardCode("saved_disk_state_used"),
			guardCode("snapshot_refreshed"),
		},
	}
	return result, nil
}

func buildConfigurationDiffFiles(
	aggregate projectstore.ProjectAggregate,
	request ConfigurationDiffRequest,
	overrides map[string]string,
) ([]ConfigurationDiffFile, bool, error) {
	if len(request.Files) > 0 {
		return buildWorkspaceDraftDiffFiles(aggregate, request)
	}

	currentFiles, err := loadTrackedFileContents(aggregate, overrides)
	if err != nil {
		return nil, false, err
	}
	files := make([]ConfigurationDiffFile, 0, len(currentFiles))
	hasChanges := false
	for _, item := range currentFiles {
		fileDiff := configurationDiffFileFromTracked(item)
		if !fileDiff.Changed {
			continue
		}
		hasChanges = true
		files = append(files, fileDiff)
	}
	return files, hasChanges, nil
}

func buildWorkspaceDraftDiffFiles(
	aggregate projectstore.ProjectAggregate,
	request ConfigurationDiffRequest,
) ([]ConfigurationDiffFile, bool, error) {
	rootDir, _, err := resolveProjectWorkspaceDirectory(aggregate.Project.WorkingDirectory, "")
	if err != nil {
		return nil, false, err
	}
	trackedKinds := trackedProjectFileKinds(rootDir, aggregate.Files)
	files := make([]ConfigurationDiffFile, 0, len(request.Files))
	hasChanges := false
	seenPaths := make(map[string]struct{}, len(request.Files))

	for _, item := range request.Files {
		_, relativePath, err := resolveProjectWorkspaceFilePath(aggregate.Project.WorkingDirectory, item.Path)
		if err != nil {
			return nil, false, errProjectInvalidArgument
		}
		if _, exists := seenPaths[relativePath]; exists {
			continue
		}
		seenPaths[relativePath] = struct{}{}

		fileKind, _, editable := classifyWorkspaceFile(relativePath, trackedKinds)
		if !editable || fileKind == "unsupported" {
			return nil, false, errProjectInvalidArgument
		}

		absolutePath := filepath.Join(rootDir, relativePath)
		// #nosec G304 -- absolutePath is derived from a path already constrained to the validated project workspace root.
		currentContent, err := os.ReadFile(absolutePath)
		if err != nil {
			return nil, false, mapWorkspacePathError(err)
		}
		proposedContent := normalizeTextBlock(item.Content)
		if string(currentContent) == proposedContent {
			continue
		}

		hasChanges = true
		files = append(files, ConfigurationDiffFile{
			Kind:            fileKind,
			Path:            relativePath,
			DisplayPath:     relativePath,
			Changed:         true,
			CurrentHash:     hashString(string(currentContent)),
			ProposedHash:    hashString(proposedContent),
			CurrentContent:  string(currentContent),
			ProposedContent: proposedContent,
		})
	}

	return files, hasChanges, nil
}

func configurationDiffFileFromTracked(item trackedWorkspaceFile) ConfigurationDiffFile {
	baselineContent := normalizeTextBlock(item.BaselineContent)
	currentContent := normalizeTextBlock(item.Content)
	baselineHash := hashString(baselineContent)
	currentHash := hashString(currentContent)
	changed := baselineContent != currentContent
	if baselineHash == "" && currentHash == "" {
		changed = false
	}
	return ConfigurationDiffFile{
		Kind:            item.Kind,
		Path:            item.Path,
		DisplayPath:     item.DisplayPath,
		Changed:         changed,
		CurrentHash:     baselineHash,
		ProposedHash:    currentHash,
		CurrentContent:  baselineContent,
		ProposedContent: currentContent,
	}
}

func buildConfigurationDiffOverrides(
	workingDirectory string,
	request ConfigurationDiffRequest,
) (map[string]string, error) {
	if len(request.Files) == 0 {
		return nil, nil
	}
	overrides := make(map[string]string, len(request.Files))
	for _, item := range request.Files {
		_, relativePath, err := resolveProjectWorkspaceFilePath(workingDirectory, item.Path)
		if err != nil {
			return nil, errProjectInvalidArgument
		}
		overrides[relativePath] = normalizeTextBlock(item.Content)
	}
	return overrides, nil
}

func (s *Service) loadFromAggregateWithOverrides(
	aggregate projectstore.ProjectAggregate,
	overrides map[string]string,
) (projectcompose.Result, error) {
	if len(overrides) == 0 {
		return s.loadFromAggregate(aggregate)
	}
	absoluteOverrides := make(map[string][]byte, len(overrides))
	for relativePath, content := range overrides {
		if strings.TrimSpace(relativePath) == "" {
			continue
		}
		absoluteOverrides[filepath.Join(aggregate.Project.WorkingDirectory, relativePath)] = []byte(content)
	}
	return projectcompose.Load(projectcompose.Input{
		WorkingDirectory: aggregate.Project.WorkingDirectory,
		ComposeFiles:     collectFilesByKind(aggregate.Files, projectcontract.FileKindCompose.String()),
		EnvFiles:         collectFilesByKind(aggregate.Files, projectcontract.FileKindEnv.String()),
	}.WithContentOverrides(absoluteOverrides))
}

func configurationDiffWarnings(fileCount int) []string {
	warnings := make([]string, 0, configurationDiffWarningsCapacity)
	if fileCount == 0 {
		warnings = append(warnings, "No tracked compose or env files are registered for the project.")
	}
	return warnings
}

func snapshotConfigHash(snapshot *projectstore.Snapshot) string {
	if snapshot == nil {
		return ""
	}
	return snapshot.ConfigHash
}

func snapshotRefreshedAt(snapshot *projectstore.Snapshot) *time.Time {
	if snapshot == nil {
		return nil
	}
	refreshedAt := snapshot.RefreshedAt
	return &refreshedAt
}
