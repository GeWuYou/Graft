package project

import (
	"context"
	"errors"
	"fmt"
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
		override := strings.TrimSpace(*request.CanonicalProjectNameOverride)
		if override == "" {
			return projectcompose.Result{}, ImportValidationResult{}, errProjectInvalidArgument
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
		DisplayName:     displayNameOrCanonical(request.DisplayName, validation.CanonicalProjectName),
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

func (s *Service) importInspectionSession(
	ctx context.Context,
	repository projectstore.Repository,
	session importInspectionSession,
	displayName *string,
	canonicalOverride *string,
	actorID *uint64,
) (generated.ProjectImportResponse, error) {
	if len(session.Conflicts) > 0 {
		return generated.ProjectImportResponse{}, fmt.Errorf("%w: %s", errProjectConflict, strings.Join(session.Conflicts, ", "))
	}
	currentRequest := ImportRequest{
		WorkingDirectory:             session.WorkingDir,
		ComposeFiles:                 displayPathsFromCompose(session.ParseResult.ComposeFiles),
		EnvFiles:                     displayPathsFromCompose(session.ParseResult.EnvFiles),
		DisplayName:                  displayName,
		CanonicalProjectNameOverride: canonicalOverride,
		ActorID:                      actorID,
	}
	freshParse, freshValidation, err := s.parseImportRequest(currentRequest)
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}
	if !sameFileHashes(session.FileHashes, freshParse) {
		return generated.ProjectImportResponse{}, errors.Join(errProjectConflict, errProjectFileHashMismatch)
	}
	now := time.Now().UTC()
	aggregate, err := repository.ImportProject(ctx, projectstore.ImportProjectInput{
		DisplayName:                displayNameOrCanonical(displayName, freshValidation.CanonicalProjectName),
		CanonicalProjectName:       freshValidation.CanonicalProjectName,
		CanonicalProjectNameSource: freshValidation.CanonicalProjectNameSource,
		SourceKind:                 projectcontract.SourceKindImported.String(),
		HostScope:                  projectcontract.HostScopeLocal.String(),
		WorkingDirectory:           freshParse.WorkingDirectory,
		OwnershipMode:              projectcontract.OwnershipModeExternal.String(),
		LastRefreshStatus:          projectcontract.RefreshStatusSuccess.String(),
		LastRefreshAt:              &now,
		LastRefreshConfigHash:      freshParse.ConfigHash,
		LastObservedConfigHash:     freshParse.ConfigHash,
		LastDriftCheckedAt:         &now,
		DriftStatus:                projectcontract.DriftStatusClean.String(),
		Files:                      toStoreFiles(freshParse.ComposeFiles, freshParse.EnvFiles),
		Snapshot: &projectstore.Snapshot{
			ConfigHash:             freshParse.ConfigHash,
			NormalizedComposeJSON:  normalizeSnapshotJSON(freshParse.NormalizedComposeJSON),
			DeclaredServiceCount:   len(freshParse.ServiceNames),
			DeclaredServicesDigest: digestServiceNames(freshParse.ServiceNames),
			RefreshedAt:            now,
		},
		ActorID: actorID,
	})
	if err != nil {
		return generated.ProjectImportResponse{}, mapStoreError(err)
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
