package project

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

func (s *Service) parseImportRequest(
	request ImportRequest,
) (projectcompose.Result, ImportValidationResult, error) {
	parseResult, err := projectcompose.Load(projectcompose.Input{
		WorkspacePath: request.WorkspacePath,
		ComposeFiles:  request.ComposeFiles,
		EnvFiles:      request.EnvFiles,
	})
	if err != nil {
		return projectcompose.Result{}, ImportValidationResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	canonicalName := parseResult.ComposeProjectName
	canonicalNameSource := parseResult.CanonicalNameSource
	if request.ComposeProjectNameOverride != nil {
		override, err := validateExplicitComposeProjectName(*request.ComposeProjectNameOverride)
		if err != nil {
			return projectcompose.Result{}, ImportValidationResult{}, err
		}
		canonicalName = override
		canonicalNameSource = projectcontract.ComposeProjectNameSourceOverride.String()
	}
	validation := ImportValidationResult{
		ComposeProjectName:       canonicalName,
		ComposeProjectNameSource: canonicalNameSource,
		WorkspacePath:            parseResult.WorkspacePath,
		ComposeFiles:             toGeneratedFilesFromCompose(parseResult.ComposeFiles),
		EnvFiles:                 toGeneratedFilesFromCompose(parseResult.EnvFiles),
	}
	return parseResult, validation, nil
}

func (s *Service) inspectImportRequest(
	ctx context.Context,
	repository projectstore.Repository,
	request ImportRequest,
) (importInspectionSession, error) {
	parseResult, validation, err := s.parseImportRequest(request)
	if err != nil {
		return importInspectionSession{}, err
	}
	conflicts, err := s.computeConflicts(ctx, repository, validation)
	if err != nil {
		return importInspectionSession{}, err
	}
	createdAt := time.Now().UTC()
	session := importInspectionSession{
		DirectoryRef: ImportDirectoryReference{
			Provider: importProviderLocal,
			Path:     parseResult.WorkspacePath,
		},
		WorkingDir:      parseResult.WorkspacePath,
		CanonicalName:   validation.ComposeProjectName,
		CanonicalSource: validation.ComposeProjectNameSource,
		DisplayName:     defaultImportedDisplayName(request.DisplayName, parseResult.WorkspacePath, validation.ComposeProjectName),
		ParseResult:     parseResult,
		Conflicts:       append([]string(nil), conflicts...),
		Warnings:        append([]string(nil), parseResult.Warnings...),
		CreatedAt:       createdAt,
		ExpiresAt:       createdAt.Add(importInspectionSessionTTL),
		FileHashes:      snapshotFileHashes(parseResult),
	}
	session.ID = inspectionSessionID(session.DirectoryRef, parseResult.ConfigHash, createdAt)
	if s.inspectCache != nil {
		s.inspectCache.storeSession(session)
	}
	return session, nil
}

func (s *Service) validationResultFromSession(session importInspectionSession) ImportValidationResult {
	inspectionID := session.ID
	return ImportValidationResult{
		ComposeProjectName:       session.CanonicalName,
		ComposeProjectNameSource: session.CanonicalSource,
		WorkspacePath:            session.WorkingDir,
		ComposeFiles:             toGeneratedFilesFromCompose(session.ParseResult.ComposeFiles),
		EnvFiles:                 toGeneratedFilesFromCompose(session.ParseResult.EnvFiles),
		ServiceCount:             len(session.ParseResult.ServiceNames),
		NetworkNames:             append([]string(nil), session.ParseResult.NetworkNames...),
		VolumeNames:              append([]string(nil), session.ParseResult.VolumeNames...),
		Warnings:                 append([]string(nil), session.Warnings...),
		Conflicts:                append([]string(nil), session.Conflicts...),
		ConfigHash:               session.ParseResult.ConfigHash,
		DeclaredServiceNames:     append([]string(nil), session.ParseResult.ServiceNames...),
		InspectionID:             &inspectionID,
	}
}

type importInspectionCommitInput struct {
	DisplayName       *string
	CanonicalOverride *string
	LifecycleConfig   *LifecycleStandardConfig
	ActorID           *uint64
}

func (s *Service) importInspectionSession(
	ctx context.Context,
	session importInspectionSession,
	input importInspectionCommitInput,
) (generated.ApplicationImportResponse, error) {
	if len(session.Conflicts) > 0 {
		return generated.ApplicationImportResponse{}, fmt.Errorf("%w: %s", errProjectConflict, strings.Join(session.Conflicts, ", "))
	}
	currentRequest := ImportRequest{
		WorkspacePath:              session.WorkingDir,
		ComposeFiles:               displayPathsFromCompose(session.ParseResult.ComposeFiles),
		EnvFiles:                   displayPathsFromCompose(session.ParseResult.EnvFiles),
		DisplayName:                input.DisplayName,
		ComposeProjectNameOverride: input.CanonicalOverride,
		ActorID:                    input.ActorID,
	}
	freshParse, freshValidation, err := s.parseImportRequest(currentRequest)
	if err != nil {
		return generated.ApplicationImportResponse{}, err
	}
	if !sameFileHashes(session.FileHashes, freshParse) {
		return generated.ApplicationImportResponse{}, errors.Join(errProjectConflict, errProjectFileHashMismatch)
	}
	if input.LifecycleConfig == nil {
		return generated.ApplicationImportResponse{}, errProjectInvalidArgument
	}
	normalizedLifecycleConfig, err := normalizeLifecycleStandardConfig(*input.LifecycleConfig)
	if err != nil {
		return generated.ApplicationImportResponse{}, err
	}
	aggregate, now, err := s.createProjectFromWorkspace(ctx, CreationCommand{DisplayName: defaultImportedDisplayName(input.DisplayName, freshParse.WorkspacePath, freshValidation.ComposeProjectName), ComposeProjectName: freshValidation.ComposeProjectName, ComposeProjectNameSource: freshValidation.ComposeProjectNameSource, SourceType: projectcontract.SourceTypeImported.String(), WorkspacePath: freshParse.WorkspacePath, OwnershipMode: projectcontract.OwnershipModeExternal.String(), LifecycleConfig: normalizedLifecycleConfig, ParseResult: freshParse, ActorID: input.ActorID})
	if err != nil {
		return generated.ApplicationImportResponse{}, err
	}

	response := generated.ApplicationImportResponse{
		Application: toProjectDetailResponse(aggregate, nil, errProjectRuntimeUnavailable),
	}
	response.SnapshotSummary.ConfigHash = freshParse.ConfigHash
	response.SnapshotSummary.RefreshedAt = now
	serviceCount := len(freshParse.ServiceNames)
	response.SnapshotSummary.DeclaredServiceCount = &serviceCount
	return response, nil
}

// defaultImportedDisplayName 按“显式名称、工作目录基名、规范项目名”的顺序选择导入项目展示名；
// 每个候选值都会先去除首尾空白，保证持久化名称与导入检查结果一致。
func defaultImportedDisplayName(displayName *string, workingDirectory string, canonical string) string {
	if displayName != nil && strings.TrimSpace(*displayName) != "" {
		return strings.TrimSpace(*displayName)
	}
	base := strings.TrimSpace(filepath.Base(workingDirectory))
	if base != "" && base != "." {
		return base
	}
	return canonical
}

func (s *Service) computeConflicts(
	ctx context.Context,
	repository projectstore.Repository,
	validation ImportValidationResult,
) ([]string, error) {
	existing, err := repository.List(ctx, projectstore.ListQuery{Limit: projectConflictScanSize, Offset: 0})
	if err != nil {
		return nil, mapStoreError(err)
	}
	conflicts := make([]string, 0)
	targetWD := strings.TrimSpace(validation.WorkspacePath)
	targetCanonical := strings.TrimSpace(validation.ComposeProjectName)
	for _, item := range existing.Items {
		if sameWorkspacePath(targetWD, item.Application.WorkspacePath) {
			conflicts = append(conflicts, "workspace_path")
		}
		if strings.EqualFold(item.Application.ComposeProjectName, targetCanonical) {
			conflicts = append(conflicts, "compose_project_name")
		}
	}
	sort.Strings(conflicts)
	return uniqueStrings(conflicts), nil
}
