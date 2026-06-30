package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

var (
	errProjectServiceUnavailable   = errors.New("project service is unavailable")
	errProjectInvalidArgument      = errors.New("project invalid argument")
	errProjectNotFound             = errors.New("project not found")
	errProjectConflict             = errors.New("project conflict")
	errProjectImportValidation     = errors.New("project import validation failed")
	errProjectUnsupportedLifecycle = errors.New("project lifecycle is not implemented in batch 2")
	errProjectFileNotFound         = errors.New("project file not found")
)

const (
	defaultProjectListLimit = 20
	maxProjectListLimit     = 100
	projectConflictScanSize = 100
)

// ListQuery describes project list filters.
type ListQuery struct {
	Limit             int
	Offset            int
	SourceKind        string
	DriftStatus       string
	LastRefreshStatus string
}

// ImportRequest describes batch-2 import validate and import payloads.
type ImportRequest struct {
	WorkingDirectory             string
	DisplayName                  *string
	ComposeFiles                 []string
	EnvFiles                     []string
	CanonicalProjectNameOverride *string
	ActorID                      *uint64
}

// ListResult returns a paginated project list.
type ListResult struct {
	Items  []generated.ProjectListItem
	Total  int
	Limit  int
	Offset int
}

// ImportValidationResult returns the static import validation result.
type ImportValidationResult struct {
	CanonicalProjectName       string
	CanonicalProjectNameSource string
	WorkingDirectory           string
	ComposeFiles               []generated.ProjectFileItem
	EnvFiles                   []generated.ProjectFileItem
	ServiceCount               int
	Warnings                   []string
	Conflicts                  []string
	ConfigHash                 string
	DeclaredServiceNames       []string
}

// ConfigurationMetadataResult returns readonly configuration metadata.
type ConfigurationMetadataResult struct {
	ProjectID          uint64
	ComposeFiles       []generated.ProjectFileItem
	EnvFiles           []generated.ProjectFileItem
	OwnershipMode      string
	DriftStatus        string
	LastRefreshStatus  string
	LastRefreshAt      *time.Time
	DiagnosticsSummary []string
}

// ConfigurationPreviewResult returns readonly normalized compose preview.
type ConfigurationPreviewResult struct {
	ProjectID             uint64
	CanonicalProjectName  string
	ConfigHash            string
	NormalizedComposeYAML string
	RefreshedAt           *time.Time
}

// ConfigurationFileResult returns readonly raw file content.
type ConfigurationFileResult struct {
	FileID       uint64
	Kind         string
	Path         string
	Content      string
	DownloadName string
}

// ActionResult returns bounded phase-1 action status.
type ActionResult struct {
	ProjectID    uint64
	Action       generated.ProjectActionResponseAction
	Result       generated.ProjectActionResponseResult
	MessageKey   *string
	Message      *string
	GuardResults []string
}

// Service owns project registry, import, and readonly refresh/configuration use cases.
type Service struct {
	repository projectstore.Repository
}

// NewService creates the project service boundary.
func NewService(repository projectstore.Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("project repository is unavailable")
	}
	return &Service{repository: repository}, nil
}

// List returns one page of registered projects.
func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ListResult{}, err
	}
	storeResult, err := repository.List(ctx, projectstore.ListQuery{
		Limit:             query.Limit,
		Offset:            query.Offset,
		SourceKind:        strings.TrimSpace(query.SourceKind),
		DriftStatus:       strings.TrimSpace(query.DriftStatus),
		LastRefreshStatus: strings.TrimSpace(query.LastRefreshStatus),
	})
	if err != nil {
		return ListResult{}, mapStoreError(err)
	}
	items := make([]generated.ProjectListItem, 0, len(storeResult.Items))
	for _, item := range storeResult.Items {
		items = append(items, toProjectListItem(item))
	}
	return ListResult{Items: items, Total: storeResult.Total, Limit: normalizeListLimit(query.Limit), Offset: maxInt(query.Offset, 0)}, nil
}

// Get returns one project detail payload.
func (s *Service) Get(ctx context.Context, projectID uint64) (generated.ProjectDetailResponse, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return generated.ProjectDetailResponse{}, err
	}
	return toProjectDetailResponse(aggregate), nil
}

// ValidateImport resolves static compose inputs and reports bounded import validation results.
func (s *Service) ValidateImport(ctx context.Context, request ImportRequest) (ImportValidationResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ImportValidationResult{}, err
	}
	parseResult, validation, err := s.parseImportRequest(request)
	if err != nil {
		return ImportValidationResult{}, err
	}
	conflicts, err := s.computeConflicts(ctx, repository, request, validation)
	if err != nil {
		return ImportValidationResult{}, err
	}
	return ImportValidationResult{
		CanonicalProjectName:       validation.CanonicalProjectName,
		CanonicalProjectNameSource: validation.CanonicalProjectNameSource,
		WorkingDirectory:           parseResult.WorkingDirectory,
		ComposeFiles:               validation.ComposeFiles,
		EnvFiles:                   validation.EnvFiles,
		ServiceCount:               len(parseResult.ServiceNames),
		Warnings:                   append([]string(nil), parseResult.Warnings...),
		Conflicts:                  conflicts,
		ConfigHash:                 parseResult.ConfigHash,
		DeclaredServiceNames:       append([]string(nil), parseResult.ServiceNames...),
	}, nil
}

// Import validates and registers one project.
func (s *Service) Import(ctx context.Context, request ImportRequest) (generated.ProjectImportResponse, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}
	parseResult, validation, err := s.parseImportRequest(request)
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}
	conflicts, err := s.computeConflicts(ctx, repository, request, validation)
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}
	if len(conflicts) > 0 {
		return generated.ProjectImportResponse{}, fmt.Errorf("%w: %s", errProjectConflict, strings.Join(conflicts, ", "))
	}

	now := time.Now().UTC()
	aggregate, err := repository.ImportProject(ctx, projectstore.ImportProjectInput{
		DisplayName:                displayNameOrCanonical(request.DisplayName, validation.CanonicalProjectName),
		CanonicalProjectName:       validation.CanonicalProjectName,
		CanonicalProjectNameSource: validation.CanonicalProjectNameSource,
		SourceKind:                 projectcontract.SourceKindImported.String(),
		HostScope:                  projectcontract.HostScopeLocal.String(),
		WorkingDirectory:           parseResult.WorkingDirectory,
		OwnershipMode:              projectcontract.OwnershipModeExternal.String(),
		LastRefreshStatus:          projectcontract.RefreshStatusSuccess.String(),
		LastRefreshAt:              &now,
		LastRefreshConfigHash:      parseResult.ConfigHash,
		LastObservedConfigHash:     parseResult.ConfigHash,
		LastDriftCheckedAt:         &now,
		DriftStatus:                projectcontract.DriftStatusClean.String(),
		Files:                      toStoreFiles(parseResult.ComposeFiles, parseResult.EnvFiles),
		Snapshot: &projectstore.Snapshot{
			ConfigHash:             parseResult.ConfigHash,
			NormalizedComposeJSON:  normalizeSnapshotJSON(parseResult.NormalizedComposeJSON),
			DeclaredServiceCount:   len(parseResult.ServiceNames),
			DeclaredServicesDigest: digestServiceNames(parseResult.ServiceNames),
			RefreshedAt:            now,
		},
		ActorID: request.ActorID,
	})
	if err != nil {
		return generated.ProjectImportResponse{}, mapStoreError(err)
	}

	response := generated.ProjectImportResponse{
		Project: toProjectDetailResponse(aggregate),
	}
	response.SnapshotSummary.ConfigHash = parseResult.ConfigHash
	response.SnapshotSummary.RefreshedAt = now
	serviceCount := len(parseResult.ServiceNames)
	response.SnapshotSummary.DeclaredServiceCount = &serviceCount
	return response, nil
}

// Refresh reparses and persists the latest static compose snapshot.
func (s *Service) Refresh(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ActionResult{}, err
	}
	aggregate, err := repository.Get(ctx, projectID)
	if err != nil {
		return ActionResult{}, mapStoreError(err)
	}
	request := ImportRequest{
		WorkingDirectory: aggregate.Project.WorkingDirectory,
		DisplayName:      stringPointer(aggregate.Project.DisplayName),
		ComposeFiles:     collectFilesByKind(aggregate.Files, projectcontract.FileKindCompose.String()),
		EnvFiles:         collectFilesByKind(aggregate.Files, projectcontract.FileKindEnv.String()),
		ActorID:          actorID,
	}
	parseResult, _, err := s.parseImportRequest(request)
	if err != nil {
		return ActionResult{}, err
	}
	now := time.Now().UTC()
	updated, err := repository.RefreshProject(ctx, projectstore.RefreshProjectInput{
		ProjectID:              projectID,
		LastRefreshStatus:      projectcontract.RefreshStatusSuccess.String(),
		LastRefreshAt:          &now,
		LastRefreshConfigHash:  parseResult.ConfigHash,
		LastObservedConfigHash: parseResult.ConfigHash,
		LastDriftCheckedAt:     &now,
		DriftStatus:            projectcontract.DriftStatusClean.String(),
		Files:                  toStoreFiles(parseResult.ComposeFiles, parseResult.EnvFiles),
		Snapshot: &projectstore.Snapshot{
			ProjectID:              projectID,
			ConfigHash:             parseResult.ConfigHash,
			NormalizedComposeJSON:  normalizeSnapshotJSON(parseResult.NormalizedComposeJSON),
			DeclaredServiceCount:   len(parseResult.ServiceNames),
			DeclaredServicesDigest: digestServiceNames(parseResult.ServiceNames),
			RefreshedAt:            now,
		},
		ActorID: actorID,
	})
	if err != nil {
		return ActionResult{}, mapStoreError(err)
	}
	_ = updated
	return ActionResult{
		ProjectID:  projectID,
		Action:     generated.ProjectActionRefresh,
		Result:     generated.ProjectActionResultCompleted,
		MessageKey: stringPointer(projectcontract.ProjectRefreshCompleted.String()),
		Message:    stringPointer(projectcontract.ProjectRefreshCompleted.String()),
	}, nil
}

// Services returns static service projections plus empty runtime members for batch 2.
func (s *Service) Services(ctx context.Context, projectID uint64) (generated.ProjectServicesResponse, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return generated.ProjectServicesResponse{}, err
	}
	parseResult, err := s.loadFromAggregate(aggregate)
	if err != nil {
		return generated.ProjectServicesResponse{}, err
	}
	items := make([]generated.ProjectServiceItem, 0, len(parseResult.Services))
	for _, item := range parseResult.Services {
		generatedItem := generated.ProjectServiceItem{
			ServiceName:  item.ServiceName,
			RunningCount: 0,
			StoppedCount: 0,
		}
		generatedItem.ContainerMembers = generatedItem.ContainerMembers[:0]
		if item.Image != nil {
			generatedItem.Image = item.Image
		}
		if item.BuildContext != nil {
			generatedItem.BuildContext = item.BuildContext
		}
		if len(item.DeclaredPorts) > 0 {
			ports := append([]string(nil), item.DeclaredPorts...)
			generatedItem.DeclaredPorts = &ports
		}
		if len(item.DeclaredVolumes) > 0 {
			volumes := append([]string(nil), item.DeclaredVolumes...)
			generatedItem.DeclaredVolumes = &volumes
		}
		if len(item.DeclaredNetworks) > 0 {
			networks := append([]string(nil), item.DeclaredNetworks...)
			generatedItem.DeclaredNetworks = &networks
		}
		items = append(items, generatedItem)
	}
	return generated.ProjectServicesResponse{
		CanonicalProjectName: aggregate.Project.CanonicalProjectName,
		Items:                items,
		ProjectId:            mustGeneratedID(projectID),
	}, nil
}

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

// UnsupportedLifecycleAction returns an explicit batch-2 blocked action result.
func (s *Service) UnsupportedLifecycleAction(projectID uint64, action generated.ProjectActionResponseAction) (ActionResult, error) {
	return ActionResult{
		ProjectID:    projectID,
		Action:       action,
		Result:       generated.ProjectActionResultBlocked,
		MessageKey:   stringPointer(projectcontract.ProjectLifecycleAccepted.String()),
		Message:      stringPointer(projectcontract.ProjectLifecycleAccepted.String()),
		GuardResults: []string{"batch-2-scope: lifecycle execution is deferred to phase-1-batch-3"},
	}, errProjectUnsupportedLifecycle
}

func (s *Service) repositoryOrErr() (projectstore.Repository, error) {
	if s == nil || s.repository == nil {
		return nil, errProjectServiceUnavailable
	}
	return s.repository, nil
}

func (s *Service) getAggregate(ctx context.Context, projectID uint64) (projectstore.ProjectAggregate, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return projectstore.ProjectAggregate{}, err
	}
	if projectID == 0 {
		return projectstore.ProjectAggregate{}, errProjectInvalidArgument
	}
	aggregate, err := repository.Get(ctx, projectID)
	if err != nil {
		return projectstore.ProjectAggregate{}, mapStoreError(err)
	}
	return aggregate, nil
}

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

func (s *Service) computeConflicts(
	ctx context.Context,
	repository projectstore.Repository,
	request ImportRequest,
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
		if strings.EqualFold(item.Project.WorkingDirectory, targetWD) && !sameDisplayName(request.DisplayName, item.Project.DisplayName) {
			conflicts = append(conflicts, "working_directory")
		}
		if strings.EqualFold(item.Project.CanonicalProjectName, targetCanonical) && !sameWorkingDirectory(targetWD, item.Project.WorkingDirectory) {
			conflicts = append(conflicts, "canonical_project_name")
		}
	}
	sort.Strings(conflicts)
	return uniqueStrings(conflicts), nil
}

func sameDisplayName(value *string, existing string) bool {
	if value == nil {
		return false
	}
	return strings.TrimSpace(*value) == strings.TrimSpace(existing)
}

func sameWorkingDirectory(left string, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func toProjectListItem(aggregate projectstore.ProjectAggregate) generated.ProjectListItem {
	serviceCount := 0
	if aggregate.Snapshot != nil {
		serviceCount = aggregate.Snapshot.DeclaredServiceCount
	}
	return generated.ProjectListItem{
		Id:                         mustGeneratedID(aggregate.Project.ID),
		DisplayName:                aggregate.Project.DisplayName,
		CanonicalProjectName:       aggregate.Project.CanonicalProjectName,
		CanonicalProjectNameSource: generated.ProjectCanonicalNameSource(aggregate.Project.CanonicalProjectNameSource),
		SourceKind:                 generated.ProjectSourceKind(aggregate.Project.SourceKind),
		HostScope:                  generated.ProjectHostScope(aggregate.Project.HostScope),
		OwnershipMode:              generated.ProjectOwnershipMode(aggregate.Project.OwnershipMode),
		WorkingDirectory:           aggregate.Project.WorkingDirectory,
		ServiceCount:               serviceCount,
		ContainerCounts:            generated.ProjectContainerCounts{},
		LastRefreshStatus:          generated.ProjectRefreshStatus(aggregate.Project.LastRefreshStatus),
		LastRefreshAt:              aggregate.Project.LastRefreshAt,
		DriftStatus:                generated.ProjectDriftStatus(aggregate.Project.DriftStatus),
	}
}

func toProjectDetailResponse(aggregate projectstore.ProjectAggregate) generated.ProjectDetailResponse {
	item := generated.ProjectDetailResponse{
		CanonicalProjectName:       aggregate.Project.CanonicalProjectName,
		CanonicalProjectNameSource: generated.ProjectCanonicalNameSource(aggregate.Project.CanonicalProjectNameSource),
		ComposeFiles:               toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())),
		ContainerCounts:            generated.ProjectContainerCounts{},
		DisplayName:                aggregate.Project.DisplayName,
		DriftStatus:                generated.ProjectDriftStatus(aggregate.Project.DriftStatus),
		EnvFiles:                   toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindEnv.String())),
		HostScope:                  generated.ProjectHostScope(aggregate.Project.HostScope),
		Id:                         mustGeneratedID(aggregate.Project.ID),
		LastDriftCheckedAt:         aggregate.Project.LastDriftCheckedAt,
		LastRefreshAt:              aggregate.Project.LastRefreshAt,
		LastRefreshStatus:          generated.ProjectRefreshStatus(aggregate.Project.LastRefreshStatus),
		OwnershipMode:              generated.ProjectOwnershipMode(aggregate.Project.OwnershipMode),
		SourceKind:                 generated.ProjectSourceKind(aggregate.Project.SourceKind),
		WorkingDirectory:           aggregate.Project.WorkingDirectory,
	}
	if aggregate.Project.LastRefreshErrorCode != "" {
		item.LastRefreshErrorCode = stringPointer(aggregate.Project.LastRefreshErrorCode)
	}
	if aggregate.Project.LastRefreshErrorMessage != "" {
		item.LastRefreshErrorMessage = stringPointer(aggregate.Project.LastRefreshErrorMessage)
	}
	if aggregate.Project.LastRefreshConfigHash != "" {
		item.LastRefreshConfigHash = stringPointer(aggregate.Project.LastRefreshConfigHash)
	}
	if aggregate.Project.LastObservedConfigHash != "" {
		item.LastObservedConfigHash = stringPointer(aggregate.Project.LastObservedConfigHash)
	}
	if aggregate.Snapshot != nil {
		item.ServiceCount = aggregate.Snapshot.DeclaredServiceCount
	}
	return item
}

func toGeneratedFiles(files []projectstore.ProjectFile) []generated.ProjectFileItem {
	items := make([]generated.ProjectFileItem, 0, len(files))
	for _, item := range files {
		items = append(items, generated.ProjectFileItem{
			Id:                  mustGeneratedID(item.ID),
			Kind:                generated.ProjectFileKind(item.Kind),
			Role:                generated.ProjectFileRole(item.Role),
			AbsolutePath:        item.AbsolutePath,
			DisplayPath:         item.DisplayPath,
			OrderIndex:          item.OrderIndex,
			ExistsOnLastRefresh: item.ExistsOnLastRefresh,
			LastObservedHash:    optionalString(item.LastObservedHash),
		})
	}
	return items
}

func toGeneratedFilesFromCompose(files []projectcompose.FileProjection) []generated.ProjectFileItem {
	items := make([]generated.ProjectFileItem, 0, len(files))
	for index, item := range files {
		hash := item.Hash
		items = append(items, generated.ProjectFileItem{
			Id:                  int64(index + 1),
			Kind:                generated.ProjectFileKind(item.Kind),
			Role:                generated.ProjectFileRole(item.Role),
			AbsolutePath:        item.AbsolutePath,
			DisplayPath:         item.DisplayPath,
			OrderIndex:          item.OrderIndex,
			ExistsOnLastRefresh: item.Exists,
			LastObservedHash:    &hash,
		})
	}
	return items
}

func toStoreFiles(composeFiles []projectcompose.FileProjection, envFiles []projectcompose.FileProjection) []projectstore.ProjectFile {
	items := make([]projectstore.ProjectFile, 0, len(composeFiles)+len(envFiles))
	for _, item := range append(append([]projectcompose.FileProjection(nil), composeFiles...), envFiles...) {
		items = append(items, projectstore.ProjectFile{
			Kind:                item.Kind,
			Role:                item.Role,
			AbsolutePath:        item.AbsolutePath,
			DisplayPath:         item.DisplayPath,
			OrderIndex:          item.OrderIndex,
			ExistsOnLastRefresh: item.Exists,
			LastObservedHash:    item.Hash,
		})
	}
	return items
}

func filterFiles(files []projectstore.ProjectFile, kind string) []projectstore.ProjectFile {
	items := make([]projectstore.ProjectFile, 0)
	for _, item := range files {
		if item.Kind == kind {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(left, right int) bool {
		if items[left].OrderIndex == items[right].OrderIndex {
			return items[left].ID < items[right].ID
		}
		return items[left].OrderIndex < items[right].OrderIndex
	})
	return items
}

func collectFilesByKind(files []projectstore.ProjectFile, kind string) []string {
	filtered := filterFiles(files, kind)
	paths := make([]string, 0, len(filtered))
	for _, item := range filtered {
		paths = append(paths, item.AbsolutePath)
	}
	return paths
}

func (s *Service) loadFromAggregate(aggregate projectstore.ProjectAggregate) (projectcompose.Result, error) {
	return projectcompose.Load(projectcompose.Input{
		WorkingDirectory: aggregate.Project.WorkingDirectory,
		ComposeFiles:     collectFilesByKind(aggregate.Files, projectcontract.FileKindCompose.String()),
		EnvFiles:         collectFilesByKind(aggregate.Files, projectcontract.FileKindEnv.String()),
	})
}

func normalizeSnapshotJSON(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	var generic any
	if err := yamlJSONRoundTrip(raw, &generic); err != nil {
		return raw
	}
	encoded, err := json.Marshal(generic)
	if err != nil {
		return raw
	}
	return encoded
}

func yamlJSONRoundTrip(raw []byte, target any) error {
	return json.Unmarshal(raw, target)
}

func digestServiceNames(names []string) string {
	normalized := append([]string(nil), names...)
	sort.Strings(normalized)
	hasher := sha256.New()
	for _, item := range normalized {
		hasher.Write([]byte(item))
		hasher.Write([]byte{0})
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func displayNameOrCanonical(displayName *string, canonical string) string {
	if displayName != nil && strings.TrimSpace(*displayName) != "" {
		return strings.TrimSpace(*displayName)
	}
	return canonical
}

func fileName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func stringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalString(value string) *string {
	return stringPointer(value)
}

func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func normalizeListLimit(value int) int {
	switch {
	case value <= 0:
		return defaultProjectListLimit
	case value > maxProjectListLimit:
		return maxProjectListLimit
	default:
		return value
	}
}

func mustGeneratedID(value uint64) int64 {
	if value == 0 || value > math.MaxInt64 {
		panic("project generated id out of range")
	}
	return int64(value)
}

func maxInt(value int, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

func mapStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, projectstore.ErrInvalidInput):
		return errProjectInvalidArgument
	case errors.Is(err, projectstore.ErrProjectNotFound):
		return errProjectNotFound
	case errors.Is(err, projectstore.ErrProjectConflict):
		return errProjectConflict
	case errors.Is(err, projectstore.ErrFileNotFound):
		return errProjectFileNotFound
	default:
		return err
	}
}
