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
		WorkingDirectory: request.WorkingDirectory,
		ComposeFiles:     request.ComposeFiles,
		EnvFiles:         request.EnvFiles,
	})
	if err != nil {
		return projectcompose.Result{}, ImportValidationResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	canonicalName := parseResult.CanonicalProjectName
	canonicalNameSource := parseResult.CanonicalNameSource
	if request.CanonicalProjectNameOverride != nil {
		override, err := validateExplicitCanonicalProjectName(*request.CanonicalProjectNameOverride)
		if err != nil {
			return projectcompose.Result{}, ImportValidationResult{}, err
		}
		canonicalName = override
		canonicalNameSource = projectcontract.CanonicalProjectNameSourceOverride.String()
	}
	validation := ImportValidationResult{
		CanonicalProjectName:       canonicalName,
		CanonicalProjectNameSource: canonicalNameSource,
		WorkingDirectory:           parseResult.WorkingDirectory,
		ComposeFiles:               toGeneratedFilesFromCompose(parseResult.ComposeFiles),
		EnvFiles:                   toGeneratedFilesFromCompose(parseResult.EnvFiles),
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
			Path:     parseResult.WorkingDirectory,
		},
		WorkingDir:      parseResult.WorkingDirectory,
		CanonicalName:   validation.CanonicalProjectName,
		CanonicalSource: validation.CanonicalProjectNameSource,
		DisplayName:     defaultImportedDisplayName(request.DisplayName, parseResult.WorkingDirectory, validation.CanonicalProjectName),
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
		CanonicalProjectName:       session.CanonicalName,
		CanonicalProjectNameSource: session.CanonicalSource,
		WorkingDirectory:           session.WorkingDir,
		ComposeFiles:               toGeneratedFilesFromCompose(session.ParseResult.ComposeFiles),
		EnvFiles:                   toGeneratedFilesFromCompose(session.ParseResult.EnvFiles),
		ServiceCount:               len(session.ParseResult.ServiceNames),
		NetworkNames:               append([]string(nil), session.ParseResult.NetworkNames...),
		VolumeNames:                append([]string(nil), session.ParseResult.VolumeNames...),
		Warnings:                   append([]string(nil), session.Warnings...),
		Conflicts:                  append([]string(nil), session.Conflicts...),
		ConfigHash:                 session.ParseResult.ConfigHash,
		DeclaredServiceNames:       append([]string(nil), session.ParseResult.ServiceNames...),
		InspectionID:               &inspectionID,
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
) (generated.ProjectImportResponse, error) {
	if len(session.Conflicts) > 0 {
		return generated.ProjectImportResponse{}, fmt.Errorf("%w: %s", errProjectConflict, strings.Join(session.Conflicts, ", "))
	}
	currentRequest := ImportRequest{
		WorkingDirectory:             session.WorkingDir,
		ComposeFiles:                 displayPathsFromCompose(session.ParseResult.ComposeFiles),
		EnvFiles:                     displayPathsFromCompose(session.ParseResult.EnvFiles),
		DisplayName:                  input.DisplayName,
		CanonicalProjectNameOverride: input.CanonicalOverride,
		ActorID:                      input.ActorID,
	}
	freshParse, freshValidation, err := s.parseImportRequest(currentRequest)
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}
	if !sameFileHashes(session.FileHashes, freshParse) {
		return generated.ProjectImportResponse{}, errors.Join(errProjectConflict, errProjectFileHashMismatch)
	}
	if input.LifecycleConfig == nil {
		return generated.ProjectImportResponse{}, errProjectInvalidArgument
	}
	normalizedLifecycleConfig, err := normalizeLifecycleStandardConfig(*input.LifecycleConfig)
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}
	aggregate, now, err := s.createProjectFromWorkspace(ctx, CreationCommand{DisplayName: defaultImportedDisplayName(input.DisplayName, freshParse.WorkingDirectory, freshValidation.CanonicalProjectName), CanonicalProjectName: freshValidation.CanonicalProjectName, CanonicalProjectNameSource: freshValidation.CanonicalProjectNameSource, SourceKind: projectcontract.SourceKindImported.String(), HostScope: projectcontract.HostScopeLocal.String(), WorkingDirectory: freshParse.WorkingDirectory, OwnershipMode: projectcontract.OwnershipModeExternal.String(), LifecycleConfig: normalizedLifecycleConfig, ParseResult: freshParse, ActorID: input.ActorID})
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}

	response := generated.ProjectImportResponse{
		Project: toProjectDetailResponse(aggregate, nil, errProjectRuntimeUnavailable),
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
	targetWD := strings.TrimSpace(validation.WorkingDirectory)
	targetCanonical := strings.TrimSpace(validation.CanonicalProjectName)
	for _, item := range existing.Items {
		if sameWorkingDirectory(targetWD, item.Project.WorkingDirectory) {
			conflicts = append(conflicts, "working_directory")
		}
		if strings.EqualFold(item.Project.CanonicalProjectName, targetCanonical) {
			conflicts = append(conflicts, "canonical_project_name")
		}
	}
	sort.Strings(conflicts)
	return uniqueStrings(conflicts), nil
}
