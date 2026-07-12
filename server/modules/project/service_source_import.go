package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	generated "graft/server/internal/contract/openapi/generated"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

// CreationMethodCatalog returns creation methods without executing a method-specific workflow.
func (s *Service) CreationMethodCatalog(ctx context.Context) (CreationMethodCatalogResult, error) {
	managedRoot, err := s.ManagedRoot(ctx)
	if err != nil {
		return CreationMethodCatalogResult{}, err
	}
	items := []generated.ProjectCreationMethod{
		{
			Method:        generated.ProjectCreationMethodTypeBlank,
			Availability:  generated.ProjectCreationMethodAvailability(mapManagedCreationMethodAvailability(managedRoot.Status)),
			BlockedReason: managedRootCreationBlockedReason(managedRoot.Status),
		},
		{
			Method:       generated.ProjectCreationMethodTypeTemplate,
			Availability: generated.ProjectCreationMethodAvailabilityReady,
		},
		{
			Method:       generated.ProjectCreationMethodTypeImport,
			Availability: generated.ProjectCreationMethodAvailabilityReady,
		},
	}
	return CreationMethodCatalogResult{Items: items}, nil
}

// DiscoveryCandidates returns bounded local discovery candidates without auto-registering projects.
func (s *Service) DiscoveryCandidates(ctx context.Context) (DiscoveryCandidatesResult, error) {
	managedRoot, err := s.ManagedRoot(ctx)
	if err != nil {
		return DiscoveryCandidatesResult{}, err
	}
	result := DiscoveryCandidatesResult{
		SourceType:            projectcontract.SourceKindManaged.String(),
		SupportsScan:          false,
		SupportsAutoDiscovery: false,
		StatusReason:          managedRoot.StatusReason,
	}
	if managedRoot.ConfiguredRootDirectory != nil {
		root := *managedRoot.ConfiguredRootDirectory
		result.AuthorityRoot = &root
	}
	if managedRoot.Status != projectcontract.ManagedRootStatusReady.String() || managedRoot.ConfiguredRootDirectory == nil {
		return result, nil
	}
	result.SupportsScan = true
	result.SupportsAutoDiscovery = true

	repository, err := s.repositoryOrErr()
	if err != nil {
		return DiscoveryCandidatesResult{}, err
	}
	items, err := s.scanDiscoveryCandidates(ctx, repository, *managedRoot.ConfiguredRootDirectory, managedRoot.ConfigKey)
	if err != nil {
		return DiscoveryCandidatesResult{}, err
	}
	result.Items = items
	return result, nil
}

// ImportDirectorySources returns operator-allowlisted import roots plus managed-root injection.
func (s *Service) ImportDirectorySources(ctx context.Context) (ImportDirectorySourceResult, error) {
	roots, err := s.importRootDefinitions(ctx)
	if err != nil {
		return ImportDirectorySourceResult{}, err
	}
	result := ImportDirectorySourceResult{Items: make([]ImportDirectorySource, 0, len(roots))}
	for _, root := range roots {
		result.Items = append(result.Items, ImportDirectorySource{
			Provider:    importProviderLocal,
			RootID:      root.id,
			Label:       root.label,
			Path:        root.path,
			InitialPath: normalizeBrowsePath(root.initialPath),
			Managed:     root.managed,
		})
	}
	return result, nil
}

// ListRuntimeImportCandidates returns runtime-driven Compose import candidates while keeping project as the inspect owner.
func (s *Service) ListRuntimeImportCandidates(
	ctx context.Context,
	query RuntimeImportCandidateListQuery,
) (RuntimeImportCandidatesResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return RuntimeImportCandidatesResult{}, err
	}
	if s.runtimeReader == nil {
		return RuntimeImportCandidatesResult{}, errProjectServiceUnavailable
	}
	rawCandidates, err := s.runtimeReader.ListImportCandidates(ctx, projectcontract.HostScopeLocal.String())
	if err != nil {
		return RuntimeImportCandidatesResult{}, err
	}
	existingItems, err := listProjectConflictScanItems(ctx, repository)
	if err != nil {
		return RuntimeImportCandidatesResult{}, mapStoreError(err)
	}
	items := make([]RuntimeImportCandidate, 0, len(rawCandidates))
	for _, rawCandidate := range rawCandidates {
		candidate, validateErr := s.validatedRuntimeImportCandidate(rawCandidate)
		if validateErr != nil {
			return RuntimeImportCandidatesResult{}, validateErr
		}
		if conflictReason := runtimeImportCandidateExistingConflict(candidate, existingItems); conflictReason != "" {
			candidate = markAlreadyImportedRuntimeImportCandidate(candidate, conflictReason)
		}
		items = append(items, candidate)
	}
	items = dedupeRuntimeImportCandidates(items)
	sortRuntimeImportCandidates(items)
	return buildRuntimeImportCandidatesResult(items, query), nil
}

// InspectRuntimeCandidate resolves a runtime candidate and reuses the inspect/import pipeline for preview generation.
func (s *Service) InspectRuntimeCandidate(ctx context.Context, request RuntimeImportInspectRequest) (RuntimeImportInspectResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return RuntimeImportInspectResult{}, err
	}
	candidate, err := s.runtimeImportCandidateByKey(ctx, request.CandidateKey)
	if err != nil {
		return RuntimeImportInspectResult{}, err
	}
	validatedCandidate, err := s.validatedRuntimeImportCandidate(candidate)
	if err != nil {
		return RuntimeImportInspectResult{}, err
	}
	existingItems, err := listProjectConflictScanItems(ctx, repository)
	if err != nil {
		return RuntimeImportInspectResult{}, mapStoreError(err)
	}
	if conflictReason := runtimeImportCandidateExistingConflict(validatedCandidate, existingItems); conflictReason != "" {
		return RuntimeImportInspectResult{}, errProjectConflict
	}
	if !validatedCandidate.Importable || validatedCandidate.Status != importRuntimeCandidateStatusReady {
		return RuntimeImportInspectResult{}, errProjectInvalidArgument
	}
	candidate = candidateFromValidatedRuntimeImportCandidate(candidate, validatedCandidate)
	session, err := s.inspectRuntimeCandidateSession(ctx, repository, candidate, request)
	if err != nil {
		return RuntimeImportInspectResult{}, err
	}
	runtimeMembers, err := s.runtimeImportCandidateMembers(ctx, candidate)
	if err != nil {
		return RuntimeImportInspectResult{}, err
	}
	return runtimeImportInspectResultFromSession(candidate.CandidateKey, session, runtimeMembers), nil
}

func listProjectConflictScanItems(
	ctx context.Context,
	repository projectstore.Repository,
) ([]projectstore.ProjectAggregate, error) {
	items := make([]projectstore.ProjectAggregate, 0, projectConflictScanSize)
	offset := 0
	for {
		page, err := repository.List(ctx, projectstore.ListQuery{Limit: projectConflictScanSize, Offset: offset})
		if err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		offset += len(page.Items)
		if len(page.Items) == 0 || offset >= page.Total {
			return items, nil
		}
	}
}

// BrowseImportDirectories returns a bounded root-relative directory listing for import flows.
func (s *Service) BrowseImportDirectories(ctx context.Context, query ImportDirectoryBrowseQuery) (ImportDirectoryBrowseResult, error) {
	query = normalizeDirectoryBrowseQuery(query)
	root, err := s.resolveImportRoot(ctx, query.Provider, query.RootID)
	if err != nil {
		return ImportDirectoryBrowseResult{}, err
	}
	absolute, err := resolveRootPath(root, query.Path)
	if err != nil {
		return ImportDirectoryBrowseResult{}, fmt.Errorf("%w: invalid relative path", errProjectDirectoryForbidden)
	}
	entries, err := os.ReadDir(absolute)
	if err != nil {
		return ImportDirectoryBrowseResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	items := buildImportDirectoryItems(query.Path, entries)
	sortImportDirectoryItems(items, query.SortBy, query.Order)
	start := minInt(query.Offset, len(items))
	end := minInt(start+query.Limit, len(items))
	resultItems := append([]ImportDirectoryItem(nil), items[start:end]...)
	return ImportDirectoryBrowseResult{
		Provider:    query.Provider,
		RootID:      root.id,
		CurrentPath: query.Path,
		ParentPath:  parentBrowsePath(query.Path),
		Limit:       query.Limit,
		Offset:      query.Offset,
		HasMore:     end < len(items),
		SortBy:      query.SortBy,
		Order:       query.Order,
		Items:       resultItems,
	}, nil
}

// InspectImportDirectory discovers files, parses compose once, and stores a short-lived inspection session.
func (s *Service) InspectImportDirectory(ctx context.Context, request ImportInspectRequest) (ImportInspectResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ImportInspectResult{}, err
	}
	root, err := s.resolveImportRoot(ctx, request.DirectoryRef.Provider, request.DirectoryRef.RootID)
	if err != nil {
		return ImportInspectResult{}, err
	}
	absolute, err := resolveRootPath(root, request.DirectoryRef.Path)
	if err != nil {
		return ImportInspectResult{}, fmt.Errorf("%w: invalid relative path", errProjectDirectoryForbidden)
	}
	discovered, err := discoverImportFiles(absolute)
	if err != nil {
		return ImportInspectResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	session, err := s.inspectImportRequest(ctx, repository, ImportRequest{
		WorkingDirectory:             absolute,
		ComposeFiles:                 discovered.composeFiles,
		EnvFiles:                     discovered.envFiles,
		DisplayName:                  request.DisplayName,
		CanonicalProjectNameOverride: request.CanonicalProjectNameOverride,
	})
	if err != nil {
		return ImportInspectResult{}, err
	}
	if len(discovered.warnings) > 0 {
		session.Warnings = append(session.Warnings, discovered.warnings...)
		if s.inspectCache != nil {
			s.inspectCache.storeSession(session)
		}
	}
	return importInspectResultFromSession(request.DirectoryRef, session), nil
}

// ImportByInspection validates inspection freshness and persists the inspected project.
func (s *Service) ImportByInspection(ctx context.Context, request ImportExecuteRequest) (generated.ProjectImportResponse, error) {
	if s.inspectCache == nil {
		return generated.ProjectImportResponse{}, errProjectInspectionExpired
	}
	session, ok := s.inspectCache.lookupSession(strings.TrimSpace(request.InspectionID))
	if !ok {
		return generated.ProjectImportResponse{}, errProjectInspectionExpired
	}
	response, importErr := s.importInspectionSession(ctx, session, importInspectionCommitInput{
		DisplayName:       request.DisplayName,
		CanonicalOverride: request.CanonicalProjectNameOverride,
		LifecycleConfig:   request.LifecycleConfiguration,
		ActorID:           request.ActorID,
	})
	if importErr != nil {
		if errors.Is(importErr, errProjectConflict) && errors.Is(importErr, errProjectFileHashMismatch) {
			return generated.ProjectImportResponse{}, errProjectInspectionStale
		}
		return generated.ProjectImportResponse{}, importErr
	}
	return response, nil
}
