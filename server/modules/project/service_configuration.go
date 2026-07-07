package project

import (
	"context"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
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
		LastRefreshStatus:  aggregate.Project.LastRefreshStatus,
		LastRefreshAt:      aggregate.Project.LastRefreshAt,
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
		RefreshedAt:           aggregate.Project.LastRefreshAt,
	}, nil
}

// DiffConfiguration compares the current saved project files with the latest refreshed snapshot baseline.
func (s *Service) DiffConfiguration(ctx context.Context, projectID uint64) (ConfigurationDiffResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	files, hasFileChanges, err := buildConfigurationDiffFiles(aggregate)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	parseResult, err := s.loadFromAggregate(aggregate)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	warnings := configurationDiffWarnings(aggregate, len(files))
	return ConfigurationDiffResult{
		ProjectID:            projectID,
		CanonicalProjectName: aggregate.Project.CanonicalProjectName,
		OwnershipMode:        aggregate.Project.OwnershipMode,
		CurrentConfigHash:    aggregate.Project.LastRefreshConfigHash,
		ProposedConfigHash:   parseResult.ConfigHash,
		HasChanges:           hasFileChanges || aggregate.Project.LastRefreshConfigHash != parseResult.ConfigHash,
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

func buildConfigurationDiffFiles(aggregate projectstore.ProjectAggregate) ([]ConfigurationDiffFile, bool, error) {
	currentFiles, err := loadTrackedFileContents(aggregate)
	if err != nil {
		return nil, false, err
	}
	files := make([]ConfigurationDiffFile, 0, len(currentFiles))
	hasChanges := false
	for _, item := range currentFiles {
		fileDiff := configurationDiffFileFromTracked(item)
		if fileDiff.Changed {
			hasChanges = true
		}
		files = append(files, fileDiff)
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

func configurationDiffWarnings(_ projectstore.ProjectAggregate, fileCount int) []string {
	warnings := make([]string, 0, configurationDiffWarningsCapacity)
	if fileCount == 0 {
		warnings = append(warnings, "No tracked compose or env files are registered for the project.")
	}
	return warnings
}
