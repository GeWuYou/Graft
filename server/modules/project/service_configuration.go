package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	projectcontract "graft/server/modules/project/contract"
)

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

// ConfigurationFile returns one readonly configuration file content payload.
func (s *Service) ConfigurationFile(ctx context.Context, projectID uint64, fileID uint64) (ConfigurationFileResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ConfigurationFileResult{}, err
	}
	file, err := repository.GetFile(ctx, projectID, fileID)
	if err != nil {
		return ConfigurationFileResult{}, mapStoreError(err)
	}
	content, err := os.ReadFile(file.AbsolutePath)
	if err != nil {
		return ConfigurationFileResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	return ConfigurationFileResult{
		FileID:       file.ID,
		Kind:         file.Kind,
		Path:         file.AbsolutePath,
		Content:      string(content),
		DownloadName: fileName(file.AbsolutePath),
	}, nil
}

// DiffConfiguration compares a managed draft against current tracked project files without writing persistent changes.
func (s *Service) DiffConfiguration(ctx context.Context, projectID uint64, draft ConfigurationDraft) (ConfigurationDiffResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	if err := ensureManagedProjectAggregate(aggregate); err != nil {
		return ConfigurationDiffResult{}, err
	}
	current, err := loadManagedDraftContent(aggregate)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	prepared, err := s.prepareConfigurationDraft(aggregate, draft)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	files := []ConfigurationDiffFile{
		buildConfigurationDiffFile(projectcontract.FileKindCompose.String(), current.ComposePath, current.ComposeContent, prepared.Proposal.ComposeContent),
	}
	if current.EnvPath != "" || prepared.Proposal.EnvPath != "" {
		files = append(files, buildConfigurationDiffFile(projectcontract.FileKindEnv.String(), nonEmptyString(current.EnvPath, prepared.Proposal.EnvPath), current.EnvContent, derefString(prepared.Proposal.EnvContent)))
	}
	hasChanges := false
	for _, item := range files {
		if item.Changed {
			hasChanges = true
			break
		}
	}
	return ConfigurationDiffResult{
		ProjectID:            projectID,
		CanonicalProjectName: aggregate.Project.CanonicalProjectName,
		OwnershipMode:        aggregate.Project.OwnershipMode,
		CurrentConfigHash:    nonEmptyString(aggregate.Project.LastRefreshConfigHash, current.CurrentConfigHash),
		ProposedConfigHash:   prepared.ParseResult.ConfigHash,
		HasChanges:           hasChanges,
		Files:                files,
		Warnings:             prepared.Warnings,
	}, nil
}

// ValidateConfiguration validates a managed draft without persisting any file changes.
func (s *Service) ValidateConfiguration(ctx context.Context, projectID uint64, draft ConfigurationDraft) (ConfigurationValidateResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationValidateResult{}, err
	}
	if err := ensureManagedProjectAggregate(aggregate); err != nil {
		return ConfigurationValidateResult{}, err
	}
	prepared, err := s.prepareConfigurationDraft(aggregate, draft)
	if err != nil {
		return ConfigurationValidateResult{}, err
	}
	return ConfigurationValidateResult{
		ProjectID:             projectID,
		CanonicalProjectName:  aggregate.Project.CanonicalProjectName,
		OwnershipMode:         aggregate.Project.OwnershipMode,
		ProposedConfigHash:    prepared.ParseResult.ConfigHash,
		NormalizedComposeYAML: prepared.ParseResult.NormalizedComposeYAML,
		DeclaredServiceNames:  append([]string(nil), prepared.ParseResult.ServiceNames...),
		Warnings:              prepared.Warnings,
	}, nil
}

// DeployConfiguration writes one managed draft, refreshes the snapshot, and runs docker compose up -d.
func (s *Service) DeployConfiguration(
	ctx context.Context,
	projectID uint64,
	draft ConfigurationDraft,
	actorID *uint64,
) (result DeployResult, err error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return DeployResult{}, err
	}
	if err := ensureManagedProjectAggregate(aggregate); err != nil {
		return DeployResult{}, err
	}
	prepared, err := s.prepareConfigurationDraft(aggregate, draft)
	if err != nil {
		return DeployResult{}, err
	}
	repository, err := s.repositoryOrErr()
	if err != nil {
		return DeployResult{}, err
	}
	restoreItems, err := writeManagedDraft(aggregate.Project.WorkingDirectory, prepared.Proposal)
	if err != nil {
		return DeployResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	defer restoreManagedDraftOnFailure(aggregate.Project.WorkingDirectory, restoreItems, &err)

	now := time.Now().UTC()
	upArgs, err := lifecycleUpArgs(aggregate, lifecycleConfigurationFromAggregate(aggregate))
	if err != nil {
		return DeployResult{}, err
	}
	if _, err := s.executeLifecycleActionWithAggregate(ctx, aggregate, generated.ProjectActionResponseActionProjectActionDeploy, upArgs); err != nil {
		return DeployResult{}, err
	}
	updated, err := repository.RefreshProject(ctx, buildRefreshProjectInput(projectID, prepared.ParseResult, now, actorID))
	if err != nil {
		return DeployResult{}, mapStoreError(err)
	}
	messageKey := projectcontract.ProjectDeployCompleted.String()
	guardResults := []GuardResult{
		guardCode("managed_project"),
		guardCode("draft_written"),
		guardDetail("command", strings.Join(upArgs, " ")),
		guardCode("snapshot_refreshed"),
	}
	if len(prepared.Warnings) > 0 {
		guardResults = append(guardResults, guardDetail("warnings", strings.Join(prepared.Warnings, "|")))
	}
	result = DeployResult{
		ProjectID:            projectID,
		Action:               "deploy",
		Result:               "completed",
		CanonicalProjectName: updated.Project.CanonicalProjectName,
		OwnershipMode:        updated.Project.OwnershipMode,
		ConfigHash:           prepared.ParseResult.ConfigHash,
		RefreshedAt:          now,
		DeclaredServiceCount: len(prepared.ParseResult.ServiceNames),
		MessageKey:           &messageKey,
		Message:              &messageKey,
		GuardResults:         guardResults,
	}
	return result, nil
}

func restoreManagedDraftOnFailure(workingDirectory string, restoreItems []managedDraftRestore, resultErr *error) {
	if resultErr == nil || *resultErr == nil {
		return
	}
	*resultErr = errors.Join(*resultErr, restoreManagedDraft(workingDirectory, restoreItems))
}
