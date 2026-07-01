package project

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
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
	errProjectUnsupportedLifecycle = errors.New("project lifecycle is unsupported")
	errProjectFileNotFound         = errors.New("project file not found")
	errProjectDestroyBlocked       = errors.New("project destroy blocked by ownership guard")
	errProjectManagedFlow          = errors.New("project managed flow is unsupported")
	errProjectDirectoryForbidden   = errors.New("project directory browse forbidden")
	errProjectInspectionExpired    = errors.New("project inspection expired")
	errProjectInspectionStale      = errors.New("project inspection stale")
)

const (
	defaultProjectListLimit  = 20
	maxProjectListLimit      = 100
	projectConflictScanSize  = 100
	projectDiscoveryScanSize = 8
	minLifecycleArgCount     = 2
	maxCommandOutputSummary  = 120
	managedCreateWarningsCap = 2
	draftWarningsCap         = 2
	managedCreateDirMode     = 0o750
	managedCreateFileMode    = 0o600
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

// SourceCatalogResult returns the bounded project source entrypoints owned by project authority.
type SourceCatalogResult struct {
	Items []generated.ProjectSourceEntry
}

// ActivityAuthority identifies the stable project activity authority contract.
type ActivityAuthority string

const (
	// ProjectActivityAuthorityFrontendFanout keeps project activity in frontend fan-out over container authority.
	ProjectActivityAuthorityFrontendFanout ActivityAuthority = "frontend-fanout"
	// ProjectActivityAuthorityBackendPlanned reserves a future backend aggregation owner without implementing it yet.
	ProjectActivityAuthorityBackendPlanned ActivityAuthority = "backend-planned"
)

// DiscoveryCandidateResult returns one bounded directory-scan or auto-discovery preview candidate.
type DiscoveryCandidateResult struct {
	CandidateKey               string
	CandidateKind              string
	SourceKind                 string
	SourceType                 string
	SourceMetadata             map[string]string
	DisplayName                string
	CanonicalProjectName       string
	CanonicalProjectNameSource string
	WorkingDirectory           string
	OwnershipMode              string
	HostScope                  string
	Status                     string
	RecommendedAction          string
	StatusReason               *string
	ComposeFiles               []generated.ProjectFileItem
	EnvFiles                   []generated.ProjectFileItem
	DeclaredServiceNames       []string
	ServiceCount               int
	ConfigHash                 string
	Warnings                   []string
	Conflicts                  []string
}

// DiscoveryCandidatesResult returns the bounded scan/discovery candidate authority surface.
type DiscoveryCandidatesResult struct {
	SourceType            string
	AuthorityRoot         *string
	SupportsScan          bool
	SupportsAutoDiscovery bool
	StatusReason          *string
	Items                 []DiscoveryCandidateResult
}

// ImportValidationResult returns the static import validation result.
type ImportValidationResult struct {
	CanonicalProjectName       string
	CanonicalProjectNameSource string
	WorkingDirectory           string
	ComposeFiles               []generated.ProjectFileItem
	EnvFiles                   []generated.ProjectFileItem
	ServiceCount               int
	NetworkNames               []string
	VolumeNames                []string
	Warnings                   []string
	Conflicts                  []string
	ConfigHash                 string
	DeclaredServiceNames       []string
	InspectionID               *string
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

// ConfigurationDraft describes one Phase 2.4 managed configuration draft.
type ConfigurationDraft struct {
	ComposeFileContent string
	EnvFileContent     *string
}

// ConfigurationDiffFile describes one file-level diff projection.
type ConfigurationDiffFile struct {
	Kind            string
	Path            string
	Changed         bool
	CurrentHash     string
	ProposedHash    string
	CurrentContent  string
	ProposedContent string
}

// ConfigurationDiffResult returns bounded managed draft diff output.
type ConfigurationDiffResult struct {
	ProjectID            uint64
	CanonicalProjectName string
	OwnershipMode        string
	CurrentConfigHash    string
	ProposedConfigHash   string
	HasChanges           bool
	Files                []ConfigurationDiffFile
	Warnings             []string
}

// ConfigurationValidateResult returns bounded managed draft validation output.
type ConfigurationValidateResult struct {
	ProjectID             uint64
	CanonicalProjectName  string
	OwnershipMode         string
	ProposedConfigHash    string
	NormalizedComposeYAML string
	DeclaredServiceNames  []string
	Warnings              []string
}

// DeployResult returns bounded managed deploy output.
type DeployResult struct {
	ProjectID            uint64
	Action               string
	Result               string
	CanonicalProjectName string
	OwnershipMode        string
	ConfigHash           string
	RefreshedAt          time.Time
	DeclaredServiceCount int
	MessageKey           *string
	Message              *string
	GuardResults         []GuardResult
}

type preparedConfigurationDraft struct {
	Proposal    managedDraftProposal
	ParseResult projectcompose.Result
	Warnings    []string
}

// ActionResult returns bounded phase-1 action status.
type ActionResult struct {
	ProjectID    uint64
	Action       generated.ProjectActionResponseAction
	Result       generated.ProjectActionResponseResult
	MessageKey   *string
	Message      *string
	GuardResults []GuardResult
}

// GuardResult is the stable structured contract for blocked/guarded project actions.
type GuardResult struct {
	Code       string
	MessageKey *string
	Detail     *string
}

// DestroyRequest describes guarded destroy options.
type DestroyRequest struct {
	RemoveNamedVolumes          bool
	DeleteWorkingDirectory      bool
	ConfirmCanonicalProjectName string
	ActorID                     *uint64
}

// ManagedRootInfo returns bounded managed-root contract metadata.
type ManagedRootInfo struct {
	SourceType              string
	Status                  string
	ConfigKey               string
	ConfiguredRootDirectory *string
	OwnershipMode           string
	CreatePermission        string
	SupportsManagedCreate   bool
	StatusReason            *string
}

// ManagedProjectCreateRequest describes Phase 2 managed-create contract payloads.
type ManagedProjectCreateRequest struct {
	DisplayName              string
	CanonicalProjectName     string
	RelativeProjectDirectory string
	ComposeFileName          string
	ComposeFileContent       string
	EnvFileName              *string
	EnvFileContent           *string
}

// ManagedProjectCreateValidationResult returns create-contract validation metadata without writing files.
type ManagedProjectCreateValidationResult struct {
	ManagedRoot             ManagedRootInfo
	SourceType              string
	DisplayName             string
	CanonicalProjectName    string
	OwnershipMode           string
	WorkingDirectory        string
	ComposeFileName         string
	EnvFileName             *string
	ComposeFileAbsolutePath string
	EnvFileAbsolutePath     *string
	SourceMetadata          map[string]string
	Warnings                []string
}

// ManagedProjectCreateResult returns the created managed project bootstrap after write + persist.
type ManagedProjectCreateResult struct {
	Validation           ManagedProjectCreateValidationResult
	SourceType           string
	ProjectID            uint64
	ConfigHash           string
	DeclaredServiceCount int
	RefreshedAt          time.Time
}

// Service owns project registry, import, and readonly refresh/configuration use cases.
type Service struct {
	repository     projectstore.Repository
	runtimeReader  moduleapi.ContainerProjectRuntimeReader
	configResolver moduleapi.SystemConfigResolver
	inspectCache   *importInspectionCache
}

// NewService 创建项目服务边界并应用可选配置。
// 当 repository 为空时返回错误。
func NewService(repository projectstore.Repository, options ...ServiceOption) (*Service, error) {
	if repository == nil {
		return nil, errors.New("project repository is unavailable")
	}
	service := &Service{
		repository:   repository,
		inspectCache: newImportInspectionCache(),
	}
	for _, option := range options {
		if option != nil {
			option.apply(service)
		}
	}
	return service, nil
}

// ServiceOption customizes project service dependencies.
type ServiceOption interface{ apply(*Service) }

type serviceOptionFunc func(*Service)

func (f serviceOptionFunc) apply(s *Service) { f(s) }

// WithRuntimeReader 设置容器运行时聚合读取器。
// 用于提供项目成员运行态汇总所需的运行时边界。
func WithRuntimeReader(reader moduleapi.ContainerProjectRuntimeReader) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.runtimeReader = reader
	})
}

// WithSystemConfigResolver 注入用于 managed-create 权限校验的系统配置读取边界。
func WithSystemConfigResolver(resolver moduleapi.SystemConfigResolver) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.configResolver = resolver
	})
}

// SetRuntimeReader injects the runtime reader after module registration resolves cross-module services.
func (s *Service) SetRuntimeReader(reader moduleapi.ContainerProjectRuntimeReader) {
	if s == nil {
		return
	}
	s.runtimeReader = reader
}

// SetSystemConfigResolver injects the system-config resolver after module registration.
func (s *Service) SetSystemConfigResolver(resolver moduleapi.SystemConfigResolver) {
	if s == nil {
		return
	}
	s.configResolver = resolver
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
	managedRootDirectory := s.readyManagedRootDirectory(ctx)
	items := make([]generated.ProjectListItem, 0, len(storeResult.Items))
	for _, item := range storeResult.Items {
		runtimeSummary, _ := s.runtimeSummary(ctx, item)
		items = append(items, toProjectListItemWithManagedRoot(item, managedRootDirectory, runtimeSummary))
	}
	return ListResult{Items: items, Total: storeResult.Total, Limit: normalizeListLimit(query.Limit), Offset: maxInt(query.Offset, 0)}, nil
}

// SourceCatalog returns the bounded Phase 3 source entrypoints without executing source-specific provisioning.
func (s *Service) SourceCatalog(ctx context.Context) (SourceCatalogResult, error) {
	managedRoot, err := s.ManagedRoot(ctx)
	if err != nil {
		return SourceCatalogResult{}, err
	}
	items := []generated.ProjectSourceEntry{
		{
			Type:            generated.ProjectSourceEntryType("managed"),
			Status:          generated.ProjectSourceEntryStatus(mapManagedSourceCatalogStatus(managedRoot.Status)),
			DisplayName:     "Managed Project",
			TitleKey:        "project.list.sourceKinds.managed",
			HostScope:       generated.ProjectHostScope(projectcontract.HostScopeLocal),
			RoutePath:       projectcontract.ProjectManagedCreateMenuPath,
			RouteName:       "ProjectManagedCreate",
			Permission:      projectcontract.ProjectCreatePermission.String(),
			MenuGroup:       projectcontract.ProjectMenuPath,
			Description:     "Create a managed Compose project under the canonical managed root owned by project authority.",
			DescriptionKey:  "project.createSource.descriptions.managed",
			MetadataFields:  []string{"managed_root_key", "managed_relative_directory", "managed_compose_file_name", "managed_env_file_name"},
			StatusReason:    managedRoot.StatusReason,
			StatusReasonKey: managedRootStatusReasonKey(managedRoot.Status),
		},
		{
			Type:            generated.ProjectSourceEntryType("git"),
			Status:          generated.ProjectSourceEntryStatus("planned"),
			DisplayName:     "Git Project",
			TitleKey:        "project.list.sourceKinds.git",
			HostScope:       generated.ProjectHostScope(projectcontract.HostScopeLocal),
			RoutePath:       projectcontract.ProjectGitCreateMenuPath,
			RouteName:       "ProjectGitCreate",
			Permission:      projectcontract.ProjectCreatePermission.String(),
			MenuGroup:       projectcontract.ProjectMenuPath,
			Description:     "Reserve the canonical git-backed project source boundary without introducing clone, scan, or remote-host execution in this batch.",
			DescriptionKey:  "project.createSource.descriptions.git",
			MetadataFields:  []string{"git_repository_url", "git_reference", "git_compose_subpath"},
			StatusReason:    stringPointer("This source remains planned. Materialization will land in a later bounded batch."),
			StatusReasonKey: stringPointer("project.createSource.statusReason.planned"),
		},
		{
			Type:            generated.ProjectSourceEntryType("template"),
			Status:          generated.ProjectSourceEntryStatus("planned"),
			DisplayName:     "Template Project",
			TitleKey:        "project.list.sourceKinds.template",
			HostScope:       generated.ProjectHostScope(projectcontract.HostScopeLocal),
			RoutePath:       projectcontract.ProjectTemplateCreateMenuPath,
			RouteName:       "ProjectTemplateCreate",
			Permission:      projectcontract.ProjectCreatePermission.String(),
			MenuGroup:       projectcontract.ProjectMenuPath,
			Description:     "Reserve the canonical template-backed project source boundary without introducing template instantiation, discovery, or remote-host execution in this batch.",
			DescriptionKey:  "project.createSource.descriptions.template",
			MetadataFields:  []string{"template_key", "template_version", "template_instance_name"},
			StatusReason:    stringPointer("This source remains planned. Materialization will land in a later bounded batch."),
			StatusReasonKey: stringPointer("project.createSource.statusReason.planned"),
		},
		{
			Type:            generated.ProjectSourceEntryType("remote-host"),
			Status:          generated.ProjectSourceEntryStatus("planned"),
			DisplayName:     "Remote Host Project",
			TitleKey:        "project.list.sourceKinds.remote-host",
			HostScope:       generated.ProjectHostScope(projectcontract.HostScopeRemote),
			RoutePath:       projectcontract.ProjectRemoteHostCreateMenuPath,
			RouteName:       "ProjectRemoteHostCreate",
			Permission:      projectcontract.ProjectCreatePermission.String(),
			MenuGroup:       projectcontract.ProjectMenuPath,
			Description:     "Reserve the canonical remote-host project boundary without introducing remote execution, secret persistence, or runtime ownership in this batch.",
			DescriptionKey:  "project.createSource.descriptions.remoteHost",
			MetadataFields:  []string{"remote_host_key", "remote_compose_path", "activity_authority", "activity_rollup_scope"},
			StatusReason:    stringPointer("This source remains planned. Remote execution and backend activity aggregation are still out of scope."),
			StatusReasonKey: stringPointer("project.createSource.statusReason.planned"),
		},
	}
	return SourceCatalogResult{Items: items}, nil
}

// DiscoveryCandidates returns bounded local discovery candidates without auto-registering projects.
func (s *Service) DiscoveryCandidates(ctx context.Context) (DiscoveryCandidatesResult, error) {
	managedRoot, err := s.ManagedRoot(ctx)
	if err != nil {
		return DiscoveryCandidatesResult{}, err
	}
	result := DiscoveryCandidatesResult{
		SourceType:            string(generated.ProjectSourceEntryTypeManaged),
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
	status := "ready"
	if len(session.Conflicts) > 0 {
		status = "conflict"
	}
	return ImportInspectResult{
		InspectionID:               session.ID,
		DirectoryRef:               request.DirectoryRef,
		ResolvedWorkingDirectory:   session.WorkingDir,
		CanonicalProjectName:       session.CanonicalName,
		CanonicalProjectNameSource: session.CanonicalSource,
		DisplayNameSuggested:       session.DisplayName,
		ComposeFiles:               toFileViews(session.ParseResult.ComposeFiles),
		EnvFiles:                   toFileViews(session.ParseResult.EnvFiles),
		ServiceNames:               append([]string(nil), session.ParseResult.ServiceNames...),
		NetworkNames:               append([]string(nil), session.ParseResult.NetworkNames...),
		VolumeNames:                append([]string(nil), session.ParseResult.VolumeNames...),
		ConfigHash:                 session.ParseResult.ConfigHash,
		Warnings:                   append([]string(nil), session.Warnings...),
		Conflicts:                  append([]string(nil), session.Conflicts...),
		ValidationStatus:           status,
	}, nil
}

// ImportByInspection validates inspection freshness and persists the inspected project.
func (s *Service) ImportByInspection(ctx context.Context, request ImportExecuteRequest) (generated.ProjectImportResponse, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}
	if s.inspectCache == nil {
		return generated.ProjectImportResponse{}, errProjectInspectionExpired
	}
	session, ok := s.inspectCache.lookupSession(strings.TrimSpace(request.InspectionID))
	if !ok {
		return generated.ProjectImportResponse{}, errProjectInspectionExpired
	}
	response, importErr := s.importInspectionSession(ctx, repository, session, request.DisplayName, request.CanonicalProjectNameOverride, request.ActorID)
	if importErr != nil {
		if errors.Is(importErr, errProjectConflict) && strings.Contains(importErr.Error(), "file hash mismatch") {
			return generated.ProjectImportResponse{}, errProjectInspectionStale
		}
		return generated.ProjectImportResponse{}, importErr
	}
	return response, nil
}

// Get returns one project detail payload.
func (s *Service) Get(ctx context.Context, projectID uint64) (generated.ProjectDetailResponse, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return generated.ProjectDetailResponse{}, err
	}
	runtimeSummary, _ := s.runtimeSummary(ctx, aggregate)
	return toProjectDetailResponseWithManagedRoot(aggregate, s.readyManagedRootDirectory(ctx), runtimeSummary), nil
}

// ValidateImport resolves static compose inputs and reports bounded import validation results.
func (s *Service) ValidateImport(ctx context.Context, request ImportRequest) (ImportValidationResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ImportValidationResult{}, err
	}
	session, err := s.inspectImportRequest(ctx, repository, request)
	if err != nil {
		return ImportValidationResult{}, err
	}
	return s.validationResultFromSession(session), nil
}

// Import validates and registers one project.
func (s *Service) Import(ctx context.Context, request ImportRequest) (generated.ProjectImportResponse, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}
	session, err := s.inspectImportRequest(ctx, repository, request)
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}
	return s.importInspectionSession(ctx, repository, session, request.DisplayName, request.CanonicalProjectNameOverride, request.ActorID)
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
	updated, err := repository.RefreshProject(ctx, buildRefreshProjectInput(projectID, parseResult, now, actorID))
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
	runtimeSummary, _ := s.runtimeSummary(ctx, aggregate)
	serviceMembers := membersByService(runtimeSummary.Members)
	items := make([]generated.ProjectServiceItem, 0, len(parseResult.Services))
	for _, item := range parseResult.Services {
		members := serviceMembers[item.ServiceName]
		generatedItem := generated.ProjectServiceItem{
			ServiceName: item.ServiceName,
		}
		applyGeneratedServiceMembers(&generatedItem, members)
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

// ManagedRoot reports the canonical managed-root authority for future managed-create flows.
func (s *Service) ManagedRoot(ctx context.Context) (ManagedRootInfo, error) {
	definitionKey := projectcontract.ProjectManagedRootConfig.String()
	info := ManagedRootInfo{
		SourceType:            "managed",
		Status:                projectcontract.ManagedRootStatusUnconfigured.String(),
		ConfigKey:             definitionKey,
		OwnershipMode:         projectcontract.OwnershipModeManagedRootDedicated.String(),
		CreatePermission:      projectcontract.ProjectCreatePermission.String(),
		SupportsManagedCreate: false,
	}
	if s.configResolver == nil {
		reason := "system config resolver unavailable"
		info.Status = projectcontract.ManagedRootStatusInvalid.String()
		info.StatusReason = &reason
		return info, nil
	}

	raw, err := s.configResolver.ResolveDefaultConfig(ctx, definitionKey)
	if err != nil {
		reason := "managed root config definition is unavailable"
		info.Status = projectcontract.ManagedRootStatusInvalid.String()
		info.StatusReason = &reason
		return info, nil
	}

	var root string
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		reason := "managed root config default value is not a string"
		info.Status = projectcontract.ManagedRootStatusInvalid.String()
		info.StatusReason = &reason
		return info, nil
	}
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." {
		reason := "managed root directory is not configured"
		info.StatusReason = &reason
		return info, nil
	}
	if !filepath.IsAbs(root) {
		reason := "managed root directory must be absolute"
		info.Status = projectcontract.ManagedRootStatusInvalid.String()
		info.ConfiguredRootDirectory = stringPointer(root)
		info.StatusReason = &reason
		return info, nil
	}
	info.Status = projectcontract.ManagedRootStatusReady.String()
	info.SupportsManagedCreate = true
	info.ConfiguredRootDirectory = stringPointer(root)
	return info, nil
}

// ValidateManagedCreate resolves bounded managed-create paths and naming rules without writing files.
func (s *Service) ValidateManagedCreate(ctx context.Context, request ManagedProjectCreateRequest) (ManagedProjectCreateValidationResult, error) {
	rootInfo, err := s.ManagedRoot(ctx)
	if err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	if rootInfo.Status != projectcontract.ManagedRootStatusReady.String() || rootInfo.ConfiguredRootDirectory == nil {
		return ManagedProjectCreateValidationResult{}, fmt.Errorf("%w: %s", errProjectInvalidArgument, projectcontract.ProjectManagedRootUnconfigured.String())
	}

	normalized, err := normalizeManagedCreateRequest(request)
	if err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	workingDirectory := filepath.Join(*rootInfo.ConfiguredRootDirectory, normalized.RelativeProjectDirectory)
	composeFileAbsolutePath := filepath.Join(workingDirectory, normalized.ComposeFileName)
	envFileAbsolutePath := managedCreateEnvAbsolutePath(workingDirectory, normalized.EnvFileName)
	warnings := make([]string, 0, managedCreateWarningsCap)
	warnings = append(warnings, "Managed create validation checks authority, normalized names, and target paths before any file-write execution.")
	if normalized.EnvFileName == nil {
		warnings = append(warnings, "No env file is declared; create execution will only materialize the compose file.")
	}

	return ManagedProjectCreateValidationResult{
		ManagedRoot:             rootInfo,
		SourceType:              "managed",
		DisplayName:             normalized.DisplayName,
		CanonicalProjectName:    normalized.CanonicalProjectName,
		OwnershipMode:           projectcontract.OwnershipModeManagedRootDedicated.String(),
		WorkingDirectory:        workingDirectory,
		ComposeFileName:         normalized.ComposeFileName,
		EnvFileName:             normalized.EnvFileName,
		ComposeFileAbsolutePath: composeFileAbsolutePath,
		EnvFileAbsolutePath:     envFileAbsolutePath,
		SourceMetadata: map[string]string{
			"managed_root_key":           rootInfo.ConfigKey,
			"managed_relative_directory": normalized.RelativeProjectDirectory,
			"managed_compose_file_name":  normalized.ComposeFileName,
			"managed_env_file_name":      stringValue(normalized.EnvFileName),
		},
		Warnings: warnings,
	}, nil
}

// CreateManagedProject writes managed project files under the configured managed root and persists the registry bootstrap.
func (s *Service) CreateManagedProject(
	ctx context.Context,
	request ManagedProjectCreateRequest,
	actorID *uint64,
) (result ManagedProjectCreateResult, err error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	validation, err := s.ValidateManagedCreate(ctx, request)
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
	shouldCleanup := true
	defer func() {
		if shouldCleanup {
			err = errors.Join(err, cleanupManagedCreate(createdDir, createdFiles))
		}
	}()

	parseResult, err := projectcompose.Load(projectcompose.Input{
		WorkingDirectory: validation.WorkingDirectory,
		ComposeFiles:     []string{validation.ComposeFileAbsolutePath},
		EnvFiles:         managedCreateEnvFileList(validation.EnvFileAbsolutePath),
	})
	if err != nil {
		return ManagedProjectCreateResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}

	now := time.Now().UTC()
	aggregate, err := repository.ImportProject(ctx, projectstore.ImportProjectInput{
		DisplayName:                normalized.DisplayName,
		CanonicalProjectName:       normalized.CanonicalProjectName,
		CanonicalProjectNameSource: projectcontract.CanonicalProjectNameSourceOverride.String(),
		SourceKind:                 projectcontract.SourceKindManaged.String(),
		HostScope:                  projectcontract.HostScopeLocal.String(),
		WorkingDirectory:           validation.WorkingDirectory,
		OwnershipMode:              projectcontract.OwnershipModeManagedRootDedicated.String(),
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
		ActorID: actorID,
	})
	if err != nil {
		return ManagedProjectCreateResult{}, mapStoreError(err)
	}
	shouldCleanup = false
	result = ManagedProjectCreateResult{
		Validation:           validation,
		SourceType:           "managed",
		ProjectID:            aggregate.Project.ID,
		ConfigHash:           parseResult.ConfigHash,
		DeclaredServiceCount: len(parseResult.ServiceNames),
		RefreshedAt:          now,
	}
	return result, nil
}

type normalizedManagedCreateRequest struct {
	DisplayName              string
	CanonicalProjectName     string
	RelativeProjectDirectory string
	ComposeFileName          string
	ComposeFileContent       string
	EnvFileName              *string
	EnvFileContent           *string
}

// normalizeManagedCreateRequest 规范化受控创建请求并校验必填字段。
// 它会修剪显示名、规范名、compose 内容和文件名，校验相对目录、compose 文件名以及可选 env 文件信息，并在必填项缺失时返回错误。
func normalizeManagedCreateRequest(request ManagedProjectCreateRequest) (normalizedManagedCreateRequest, error) {
	displayName := strings.TrimSpace(request.DisplayName)
	canonicalName := strings.TrimSpace(request.CanonicalProjectName)
	composeFileContent := strings.TrimSpace(request.ComposeFileContent)
	relativeDir, err := normalizeManagedRelativeDirectory(request.RelativeProjectDirectory)
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	composeFileName, err := normalizeManagedFileName(request.ComposeFileName, "compose")
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	if displayName == "" || canonicalName == "" || composeFileContent == "" {
		return normalizedManagedCreateRequest{}, fmt.Errorf("%w: missing required managed-create fields", errProjectInvalidArgument)
	}
	envFileName, err := normalizeManagedOptionalFileName(request.EnvFileName, "env")
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	envFileContent := normalizeManagedOptionalContent(request.EnvFileContent)
	if envFileName == nil {
		envFileContent = nil
	}
	return normalizedManagedCreateRequest{
		DisplayName:              displayName,
		CanonicalProjectName:     canonicalName,
		RelativeProjectDirectory: relativeDir,
		ComposeFileName:          composeFileName,
		ComposeFileContent:       composeFileContent,
		EnvFileName:              envFileName,
		EnvFileContent:           envFileContent,
	}, nil
}

// normalizeManagedRelativeDirectory 规范化并校验 managed-create 的相对项目目录。
// 它会去除首尾空白、统一路径分隔符并清理路径，同时确保目录保持在 managed root 下。
// @param value 待规范化的相对目录。
// @returns 规范化后的相对目录；当目录为空、为绝对路径或会逃逸 managed root 时返回错误。
func normalizeManagedRelativeDirectory(value string) (string, error) {
	relativeDir := strings.TrimSpace(value)
	if relativeDir == "" {
		return "", fmt.Errorf("%w: missing required managed-create fields", errProjectInvalidArgument)
	}
	if filepath.IsAbs(relativeDir) || relativeDir == "." || relativeDir == ".." {
		return "", fmt.Errorf("%w: relative project directory must stay under managed root", errProjectInvalidArgument)
	}
	if strings.Contains(relativeDir, `\`) {
		relativeDir = strings.ReplaceAll(relativeDir, `\`, "/")
	}
	relativeDir = filepath.Clean(relativeDir)
	if relativeDir == "." || strings.HasPrefix(relativeDir, "..") {
		return "", fmt.Errorf("%w: relative project directory escapes managed root", errProjectInvalidArgument)
	}
	return relativeDir, nil
}

// 它会去除首尾空白并提取最后一个路径段；当结果为空、`.` 或路径分隔符时返回错误。
func normalizeManagedFileName(value string, label string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "." || trimmed == string(filepath.Separator) {
		return "", fmt.Errorf("%w: invalid %s file name", errProjectInvalidArgument, label)
	}
	if filepath.IsAbs(trimmed) || strings.Contains(trimmed, "/") || strings.Contains(trimmed, `\`) {
		return "", fmt.Errorf("%w: invalid %s file name", errProjectInvalidArgument, label)
	}
	fileName := filepath.Base(trimmed)
	if fileName == "" || fileName == "." || fileName != trimmed {
		return "", fmt.Errorf("%w: invalid %s file name", errProjectInvalidArgument, label)
	}
	return fileName, nil
}

// normalizeManagedOptionalFileName 规范化可选文件名，空值或纯空白时返回 nil。
//
// 当输入为空或仅包含空白字符时，返回 nil。
func normalizeManagedOptionalFileName(value *string, label string) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	fileName, err := normalizeManagedFileName(*value, label)
	if err != nil {
		return nil, fmt.Errorf("normalize %s file name: %w", label, err)
	}
	return &fileName, nil
}

// managedCreateEnvAbsolutePath 生成受控项目环境文件的绝对路径。
//
// 当未提供环境文件名时返回 nil；否则返回工作目录下对应环境文件的绝对路径。
func managedCreateEnvAbsolutePath(workingDirectory string, envFileName *string) *string {
	if envFileName == nil {
		return nil
	}
	envAbs := filepath.Join(workingDirectory, *envFileName)
	return &envAbs
}

// normalizeManagedOptionalContent 去除可选内容两端的空白字符并返回结果。
//
// @param value 待规范化的内容指针。
// @returns 规范化后的内容指针；当输入为 nil 时返回 nil。
func normalizeManagedOptionalContent(value *string) *string {
	if value == nil {
		return nil
	}
	content := strings.TrimSpace(*value)
	return &content
}

// ensureManagedCreatePathsUnderRoot 验证托管项目工作目录位于已配置的 managed root 下。
// ensureManagedCreatePathsUnderRoot 验证托管项目工作目录位于已配置的 managed root 内。
//
// 当未配置根目录，或工作目录与根目录之间不存在有效的相对关系，或工作目录超出 managed root 时，返回 errProjectInvalidArgument。
func ensureManagedCreatePathsUnderRoot(validation ManagedProjectCreateValidationResult) error {
	if validation.ManagedRoot.ConfiguredRootDirectory == nil {
		return errProjectInvalidArgument
	}
	root := filepath.Clean(*validation.ManagedRoot.ConfiguredRootDirectory)
	workingDirectory := filepath.Clean(validation.WorkingDirectory)
	relative, err := filepath.Rel(root, workingDirectory)
	if err != nil {
		return fmt.Errorf("%w: invalid managed root relationship", errProjectInvalidArgument)
	}
	if relative == "." || relative == "" || strings.HasPrefix(relative, "..") {
		return fmt.Errorf("%w: managed project directory must stay under managed root", errProjectInvalidArgument)
	}
	return nil
}

// guardCode 返回仅包含指定代码的守卫结果。
func guardCode(code string) GuardResult {
	return GuardResult{Code: code}
}

func guardDetail(code string, detail string) GuardResult {
	detail = strings.TrimSpace(detail)
	return GuardResult{Code: code, Detail: &detail}
}

// Up executes docker compose up -d within the project's registered working directory.
func (s *Service) Up(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.runLifecycleAction(ctx, projectID, actorID, generated.ProjectActionUp, []string{"compose", "up", "-d"})
}

// Down executes docker compose down for the registered project without removing volumes by default.
func (s *Service) Down(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.runLifecycleAction(ctx, projectID, actorID, generated.ProjectActionDown, []string{"compose", "down"})
}

// Restart executes docker compose restart for the registered project.
func (s *Service) Restart(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.runLifecycleAction(ctx, projectID, actorID, generated.ProjectActionRestart, []string{"compose", "restart"})
}

// Unregister removes the project registry record without touching host files.
func (s *Service) Unregister(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	if _, err := s.getAggregate(ctx, projectID); err != nil {
		return ActionResult{}, err
	}
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ActionResult{}, err
	}
	if err := repository.UnregisterProject(ctx, projectstore.UnregisterProjectInput{
		ProjectID: projectID,
		ActorID:   actorID,
	}); err != nil {
		return ActionResult{}, mapStoreError(err)
	}
	messageKey := projectcontract.ProjectUnregisterCompleted.String()
	return ActionResult{
		ProjectID:  projectID,
		Action:     generated.ProjectActionUnregister,
		Result:     generated.ProjectActionResultCompleted,
		MessageKey: &messageKey,
		Message:    &messageKey,
		GuardResults: []GuardResult{
			guardCode("registry_deleted"),
			guardCode("working_directory_preserved"),
			guardCode("runtime_state_not_persisted"),
		},
	}, nil
}

// Destroy executes guarded teardown steps and then unregisters the project record.
func (s *Service) Destroy(ctx context.Context, projectID uint64, request DestroyRequest) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	if result, blockErr := validateDestroyRequest(projectID, aggregate, request); blockErr != nil {
		return result, blockErr
	}
	return s.destroyAfterGuard(ctx, aggregate, request)
}

// validateDestroyRequest 校验销毁请求是否允许继续执行，并在违反保护条件时返回阻断结果。
// 当确认名称不匹配、请求删除命名卷，或请求删除工作目录但项目并非受控根专属所有权时，
// 返回带有相应守卫结果的阻断动作结果和销毁阻断错误。
// @param projectID 项目标识。
// @param aggregate 项目聚合数据。
// @param request 销毁请求。
// @returns 允许继续销毁时返回空动作结果和 nil；否则返回阻断动作结果和销毁阻断错误。
func validateDestroyRequest(
	projectID uint64,
	aggregate projectstore.ProjectAggregate,
	request DestroyRequest,
) (ActionResult, error) {
	guardResults := []GuardResult{}
	if strings.TrimSpace(request.ConfirmCanonicalProjectName) != aggregate.Project.CanonicalProjectName {
		return blockedActionResult(projectID, generated.ProjectActionDestroy, append(guardResults, guardCode("confirm_canonical_project_name_mismatch"))), errProjectDestroyBlocked
	}
	guardResults = append(guardResults, guardCode("confirm_canonical_project_name_matched"))

	if request.RemoveNamedVolumes {
		guardResults = append(guardResults, guardDetail("remove_named_volumes_blocked", "phase1_volume_exclusivity_not_proven"))
		return blockedActionResult(projectID, generated.ProjectActionDestroy, guardResults), errProjectDestroyBlocked
	}

	if request.DeleteWorkingDirectory && aggregate.Project.OwnershipMode != projectcontract.OwnershipModeManagedRootDedicated.String() {
		guardResults = append(guardResults, guardDetail("delete_working_directory_blocked", "ownership_mode_external"))
		return blockedActionResult(projectID, generated.ProjectActionDestroy, guardResults), errProjectDestroyBlocked
	}
	return ActionResult{}, nil
}

func (s *Service) destroyAfterGuard(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	request DestroyRequest,
) (ActionResult, error) {
	projectID := aggregate.Project.ID
	guardResults := []GuardResult{guardCode("confirm_canonical_project_name_matched")}
	if _, err := s.runLifecycleActionWithAggregate(ctx, aggregate, request.ActorID, generated.ProjectActionDown, []string{"compose", "down"}); err != nil {
		return ActionResult{}, err
	}
	guardResults = append(guardResults, guardCode("compose_down_completed"))

	if request.DeleteWorkingDirectory {
		if err := deleteManagedWorkingDirectory(aggregate.Project.WorkingDirectory); err != nil {
			return ActionResult{}, fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
		}
		guardResults = append(guardResults, guardCode("working_directory_deleted"))
	} else {
		guardResults = append(guardResults, guardCode("working_directory_preserved"))
	}

	repository, err := s.repositoryOrErr()
	if err != nil {
		return ActionResult{}, err
	}
	if err := repository.UnregisterProject(ctx, projectstore.UnregisterProjectInput{
		ProjectID: projectID,
		ActorID:   request.ActorID,
	}); err != nil {
		return ActionResult{}, mapStoreError(err)
	}
	guardResults = append(guardResults, guardCode("registry_deleted"))
	messageKey := projectcontract.ProjectDestroyCompleted.String()
	return ActionResult{
		ProjectID:    projectID,
		Action:       generated.ProjectActionDestroy,
		Result:       generated.ProjectActionResultCompleted,
		MessageKey:   &messageKey,
		Message:      &messageKey,
		GuardResults: guardResults,
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
	defer func() {
		err = errors.Join(err, restoreManagedDraft(aggregate.Project.WorkingDirectory, restoreItems))
	}()

	now := time.Now().UTC()
	if _, err := s.runLifecycleActionWithAggregate(ctx, aggregate, actorID, generated.ProjectActionDeploy, []string{"compose", "up", "-d"}); err != nil {
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
		guardDetail("command", "docker compose up -d"),
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

// UnsupportedLifecycleAction returns an explicit batch-2 blocked action result.
func (s *Service) UnsupportedLifecycleAction(projectID uint64, action generated.ProjectActionResponseAction) (ActionResult, error) {
	return ActionResult{
		ProjectID:    projectID,
		Action:       action,
		Result:       generated.ProjectActionResultBlocked,
		MessageKey:   stringPointer(projectcontract.ProjectLifecycleAccepted.String()),
		Message:      stringPointer(projectcontract.ProjectLifecycleAccepted.String()),
		GuardResults: []GuardResult{guardDetail("batch-2-scope", "lifecycle execution is deferred to phase-1-batch-3")},
	}, errProjectUnsupportedLifecycle
}

func (s *Service) runLifecycleAction(
	ctx context.Context,
	projectID uint64,
	actorID *uint64,
	action generated.ProjectActionResponseAction,
	args []string,
) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	return s.runLifecycleActionWithAggregate(ctx, aggregate, actorID, action, args)
}

func (s *Service) runLifecycleActionWithAggregate(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	_ *uint64,
	action generated.ProjectActionResponseAction,
	args []string,
) (ActionResult, error) {
	if err := ensureProjectLifecycleReady(aggregate); err != nil {
		return blockedActionResult(aggregate.Project.ID, action, []GuardResult{guardDetail("lifecycle_blocked", "refresh_required")}), err
	}
	if err := ensureLifecycleCommandArgs(args); err != nil {
		return blockedActionResult(aggregate.Project.ID, action, []GuardResult{guardDetail("lifecycle_blocked", "invalid_command")}), err
	}
	commandOutput, err := s.runComposeCommand(ctx, aggregate, args)
	if err != nil {
		result := blockedActionResult(aggregate.Project.ID, action, []GuardResult{guardDetail("lifecycle_failed", summarizeCommandOutput(commandOutput))})
		return result, fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
	}
	messageKey := lifecycleMessageKey(action).String()
	return ActionResult{
		ProjectID:  aggregate.Project.ID,
		Action:     action,
		Result:     generated.ProjectActionResultCompleted,
		MessageKey: &messageKey,
		Message:    &messageKey,
		GuardResults: []GuardResult{
			guardDetail("command", strings.Join(args, " ")),
			guardDetail("host_scope", aggregate.Project.HostScope),
		},
	}, nil
}

func (s *Service) runComposeCommand(ctx context.Context, aggregate projectstore.ProjectAggregate, args []string) (string, error) {
	// #nosec G204 -- binary is fixed to docker and args are validated command fragments, not shell-expanded input.
	command := exec.CommandContext(ctx, "docker", args...)
	command.Dir = aggregate.Project.WorkingDirectory
	command.Env = os.Environ()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return strings.TrimSpace(stdout.String() + "\n" + stderr.String()), err
}

// ensureLifecycleCommandArgs 校验生命周期命令参数。
// 只有当参数数量满足要求且每个参数都包含非空白内容时才通过。
func ensureLifecycleCommandArgs(args []string) error {
	if len(args) < minLifecycleArgCount {
		return errProjectInvalidArgument
	}
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			return errProjectInvalidArgument
		}
	}
	return nil
}

// ensureProjectLifecycleReady 检查项目是否满足执行生命周期操作的条件。
func ensureProjectLifecycleReady(aggregate projectstore.ProjectAggregate) error {
	if strings.TrimSpace(aggregate.Project.HostScope) != projectcontract.HostScopeLocal.String() {
		return errProjectUnsupportedLifecycle
	}
	if aggregate.Project.LastRefreshStatus != projectcontract.RefreshStatusSuccess.String() {
		return errProjectUnsupportedLifecycle
	}
	return nil
}

// blockedActionResult 返回一个标记为阻止的项目操作结果，并保留给定的守卫结果。
//
// @param projectID 项目 ID。
// @param action 操作类型。
// @param guardResults 守卫结果列表。
// @returns 标记为 blocked 的 ActionResult，包含项目 ID、操作类型、阻止消息以及守卫结果副本。
func blockedActionResult(projectID uint64, action generated.ProjectActionResponseAction, guardResults []GuardResult) ActionResult {
	messageKey := projectcontract.ProjectLifecycleBlocked.String()
	return ActionResult{
		ProjectID:    projectID,
		Action:       action,
		Result:       generated.ProjectActionResultBlocked,
		MessageKey:   &messageKey,
		Message:      &messageKey,
		GuardResults: append([]GuardResult(nil), guardResults...),
	}
}

// lifecycleMessageKey 返回指定生命周期动作对应的完成消息键。
func lifecycleMessageKey(action generated.ProjectActionResponseAction) projectcontract.MessageKey {
	switch action {
	case generated.ProjectActionUp:
		return projectcontract.ProjectUpCompleted
	case generated.ProjectActionDown:
		return projectcontract.ProjectDownCompleted
	case generated.ProjectActionRestart:
		return projectcontract.ProjectRestartCompleted
	case generated.ProjectActionDestroy:
		return projectcontract.ProjectDestroyCompleted
	case generated.ProjectActionDeploy:
		return projectcontract.ProjectDeployCompleted
	case generated.ProjectActionUnregister:
		return projectcontract.ProjectUnregisterCompleted
	default:
		return projectcontract.ProjectLifecycleAccepted
	}
}

// summarizeCommandOutput 归一化并截断命令输出摘要。
// 它会去除首尾空白，空输出返回 "command_failed"，并将过长内容截断到最大摘要长度。
// @param output 原始命令输出。
// @returns 处理后的输出摘要。
func summarizeCommandOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "command_failed"
	}
	if len(trimmed) > maxCommandOutputSummary {
		return trimmed[:maxCommandOutputSummary]
	}
	return trimmed
}

func (s *Service) runtimeSummary(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
) (moduleapi.ContainerProjectRuntimeSummary, error) {
	if s == nil || s.runtimeReader == nil {
		return moduleapi.ContainerProjectRuntimeSummary{
			CanonicalProjectName: aggregate.Project.CanonicalProjectName,
			Members:              []moduleapi.ContainerProjectMember{},
		}, nil
	}
	return s.runtimeReader.ListProjectMembers(ctx, aggregate.Project.HostScope, aggregate.Project.CanonicalProjectName)
}

// membersByService 按服务名称对容器运行时成员进行分组。
// @return 按服务名称索引的成员列表。
func membersByService(items []moduleapi.ContainerProjectMember) map[string][]moduleapi.ContainerProjectMember {
	result := make(map[string][]moduleapi.ContainerProjectMember)
	for _, item := range items {
		result[item.ServiceName] = append(result[item.ServiceName], item)
	}
	return result
}

// applyGeneratedServiceMembers 将运行时成员填充到服务项中，并统计运行与停止数量。
// 当 target 为空时不执行任何操作。
func applyGeneratedServiceMembers(target *generated.ProjectServiceItem, items []moduleapi.ContainerProjectMember) {
	if target == nil {
		return
	}
	//nolint:revive // OpenAPI generated anonymous member field is ContainerId.
	type generatedProjectServiceMember = struct {
		ContainerId   string `json:"container_id"`
		ContainerName string `json:"container_name"`
		State         string `json:"state"`
	}
	members := make([]generatedProjectServiceMember, 0, len(items))
	for _, item := range items {
		members = append(members, generatedProjectServiceMember{
			ContainerId:   item.ContainerID,
			ContainerName: item.ContainerName,
			State:         item.CanonicalState,
		})
		if item.CanonicalState == "running" {
			target.RunningCount++
		} else {
			target.StoppedCount++
		}
	}
	target.ContainerMembers = members
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

func (s *Service) inspectImportRequest(
	ctx context.Context,
	repository projectstore.Repository,
	request ImportRequest,
) (importInspectionSession, error) {
	parseResult, validation, err := s.parseImportRequest(request)
	if err != nil {
		return importInspectionSession{}, err
	}
	conflicts, err := s.computeConflicts(ctx, repository, request, validation)
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
		return generated.ProjectImportResponse{}, fmt.Errorf("%w: file hash mismatch", errProjectConflict)
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
		Project: toProjectDetailResponse(aggregate),
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

func (s *Service) scanDiscoveryCandidates(
	ctx context.Context,
	repository projectstore.Repository,
	rootDirectory string,
	configKey string,
) ([]DiscoveryCandidateResult, error) {
	entries, err := os.ReadDir(rootDirectory)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	candidates := make([]DiscoveryCandidateResult, 0, projectDiscoveryScanSize)
	for _, entry := range entries {
		if len(candidates) >= projectDiscoveryScanSize {
			break
		}
		name, ok := visibleDirectoryEntryName(entry)
		if !ok {
			continue
		}
		candidate, err := s.buildDiscoveryCandidate(ctx, repository, rootDirectory, name, configKey)
		if err != nil {
			continue
		}
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return strings.Compare(candidates[i].WorkingDirectory, candidates[j].WorkingDirectory) < 0
	})
	return candidates, nil
}

// buildImportDirectoryItems 构建可浏览的子目录条目列表。
// 仅包含目录条目，并为每个条目填充规范化路径；如果可获取修改时间，则同时记录其 UTC 时间。
func buildImportDirectoryItems(currentPath string, entries []os.DirEntry) []ImportDirectoryItem {
	items := make([]ImportDirectoryItem, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		item := ImportDirectoryItem{
			Name: name,
			Path: normalizeBrowsePath(filepath.Join(currentPath, name)),
		}
		if info, infoErr := entry.Info(); infoErr == nil {
			modifiedAt := info.ModTime().UTC()
			item.ModifiedAt = &modifiedAt
		}
		items = append(items, item)
	}
	return items
}

// sortImportDirectoryItems 按指定字段和顺序排列导入目录项。
// 当按修改时间排序时，会优先比较修改时间；无法区分时按名称排序。
func sortImportDirectoryItems(items []ImportDirectoryItem, sortBy string, order string) {
	sort.Slice(items, func(i, j int) bool {
		if sortBy == importDirectorySortByModified {
			if decided, ok := compareModifiedTime(items[i].ModifiedAt, items[j].ModifiedAt, order); ok {
				return decided
			}
		}
		return compareDirectoryNames(items[i].Name, items[j].Name, order)
	})
}

// compareModifiedTime 根据修改时间和排序方向比较两个时间。
// 左右时间为空时按零值时间处理；当时间相同时返回 false, false。
// 返回的第一个值表示左侧是否应排在右侧之前，第二个值表示两者是否可比较。
func compareModifiedTime(leftAt *time.Time, rightAt *time.Time, order string) (bool, bool) {
	left := time.Time{}
	right := time.Time{}
	if leftAt != nil {
		left = *leftAt
	}
	if rightAt != nil {
		right = *rightAt
	}
	if left.Equal(right) {
		return false, false
	}
	if order == importDirectoryOrderDesc {
		return left.After(right), true
	}
	return left.Before(right), true
}

// compareDirectoryNames 按指定顺序比较两个目录名的先后。
//
// 当 order 为降序时，左值大于右值返回 true；否则左值小于右值返回 true。
func compareDirectoryNames(left string, right string, order string) bool {
	if order == importDirectoryOrderDesc {
		return strings.Compare(left, right) > 0
	}
	return strings.Compare(left, right) < 0
}

// visibleDirectoryEntryName 返回可见目录项的名称。
//
// 它仅接受目录项，并过滤掉空名称、以 `.` 开头的名称以及非目录项。
// 当目录项可用时返回其修剪后的名称。
func visibleDirectoryEntryName(entry os.DirEntry) (string, bool) {
	if !entry.IsDir() {
		return "", false
	}
	name := strings.TrimSpace(entry.Name())
	if name == "" || strings.HasPrefix(name, ".") {
		return "", false
	}
	return name, true
}

func (s *Service) buildDiscoveryCandidate(
	ctx context.Context,
	repository projectstore.Repository,
	rootDirectory string,
	name string,
	configKey string,
) (DiscoveryCandidateResult, error) {
	workingDirectory := filepath.Join(rootDirectory, name)
	discovered, err := discoverImportFiles(workingDirectory)
	if err != nil {
		return DiscoveryCandidateResult{}, err
	}
	session, err := s.inspectImportRequest(ctx, repository, ImportRequest{
		WorkingDirectory: workingDirectory,
		ComposeFiles:     discovered.composeFiles,
		EnvFiles:         discovered.envFiles,
	})
	if err != nil {
		return DiscoveryCandidateResult{}, err
	}
	if len(discovered.warnings) > 0 {
		session.Warnings = append(session.Warnings, discovered.warnings...)
	}
	status, recommendedAction, statusReason := discoveryCandidateStatus(session.Conflicts)
	return DiscoveryCandidateResult{
		CandidateKey:  candidateKeyForWorkingDirectory(workingDirectory),
		CandidateKind: "directory-scan",
		SourceKind:    projectcontract.SourceKindManaged.String(),
		SourceType:    string(generated.ProjectSourceEntryTypeManaged),
		SourceMetadata: map[string]string{
			"managed_root_key":           configKey,
			"managed_relative_directory": name,
			"managed_compose_file_name":  firstProjectFileDisplayName(toGeneratedFilesFromCompose(session.ParseResult.ComposeFiles)),
			"managed_env_file_name":      firstProjectFileDisplayName(toGeneratedFilesFromCompose(session.ParseResult.EnvFiles)),
		},
		DisplayName:                session.CanonicalName,
		CanonicalProjectName:       session.CanonicalName,
		CanonicalProjectNameSource: session.CanonicalSource,
		WorkingDirectory:           session.WorkingDir,
		OwnershipMode:              projectcontract.OwnershipModeManagedRootDedicated.String(),
		HostScope:                  projectcontract.HostScopeLocal.String(),
		Status:                     status,
		RecommendedAction:          recommendedAction,
		StatusReason:               statusReason,
		ComposeFiles:               toGeneratedFilesFromCompose(session.ParseResult.ComposeFiles),
		EnvFiles:                   toGeneratedFilesFromCompose(session.ParseResult.EnvFiles),
		DeclaredServiceNames:       append([]string(nil), session.ParseResult.ServiceNames...),
		ServiceCount:               len(session.ParseResult.ServiceNames),
		ConfigHash:                 session.ParseResult.ConfigHash,
		Warnings:                   append([]string(nil), session.Warnings...),
		Conflicts:                  append([]string(nil), session.Conflicts...),
	}, nil
}

// discoveryCandidateStatus 根据冲突列表返回发现候选的状态、建议操作和状态原因。
// 当不存在冲突时返回“ready”和“import”；当存在冲突时返回“conflict”和“review”，并提供原因说明。
// @returns 状态字符串、建议操作字符串，以及在存在冲突时指向状态原因的指针。
func discoveryCandidateStatus(conflicts []string) (string, string, *string) {
	if len(conflicts) == 0 {
		return "ready", "import", nil
	}
	reason := "Existing registry ownership or canonical-name conflicts require review before import."
	return "conflict", "review", &reason
}

// candidateKeyForWorkingDirectory 生成工作目录的扫描候选键。
// @returns 基于修剪后的工作目录生成的键，格式为 `scan:` 加上 SHA-256 摘要前 8 字节的十六进制值。
func candidateKeyForWorkingDirectory(workingDirectory string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(workingDirectory)))
	return "scan:" + hex.EncodeToString(sum[:8])
}

// firstProjectFileDisplayName 返回首个项目文件展示路径的基名；当列表为空时返回空字符串。
func firstProjectFileDisplayName(items []generated.ProjectFileItem) string {
	if len(items) == 0 {
		return ""
	}
	return filepath.Base(items[0].DisplayPath)
}

// displayPathsFromCompose 返回一组 compose 文件投影的显示路径列表。
//
// @returns 按输入顺序提取的显示路径；当输入为空时返回 nil。
func displayPathsFromCompose(files []projectcompose.FileProjection) []string {
	if len(files) == 0 {
		return nil
	}
	result := make([]string, 0, len(files))
	for _, file := range files {
		result = append(result, file.DisplayPath)
	}
	return result
}

// sameDisplayName 判断给定名称与已有名称在去除首尾空白后是否一致。
//
// @param value 待比较的名称。
// @param existing 已存在的名称。
// sameDisplayName 判断两个显示名称去除首尾空白后是否相等。
// @returns 两者去除首尾空白后相等时返回 true，否则返回 false。
func sameDisplayName(value *string, existing string) bool {
	if value == nil {
		return false
	}
	return strings.TrimSpace(*value) == strings.TrimSpace(existing)
}

func (s *Service) importRootDefinitions(ctx context.Context) ([]importRootDefinition, error) {
	managedRootInfo, err := s.ManagedRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve managed root info: %w", err)
	}
	var managedRoot *string
	if managedRootInfo.Status == projectcontract.ManagedRootStatusReady.String() {
		managedRoot = managedRootInfo.ConfiguredRootDirectory
	}
	if s.configResolver == nil {
		return fallbackImportRoots(normalizeImportRootDefinitions(nil, managedRoot)), nil
	}
	raw, err := s.configResolver.ResolveDefaultConfig(ctx, projectcontract.ProjectImportAllowedRootsConfig.String())
	if err != nil {
		return fallbackImportRoots(normalizeImportRootDefinitions(nil, managedRoot)), nil
	}
	decoded, decodeErr := decodeAllowedImportRoots(raw)
	if decodeErr != nil {
		return nil, fmt.Errorf("%w: invalid import root config", errProjectInvalidArgument)
	}
	return fallbackImportRoots(normalizeImportRootDefinitions(decoded, managedRoot)), nil
}

// fallbackImportRoots 尝试为导入根列表补充当前工作目录作为回退根。
// 如果无法获取当前工作目录，则直接返回原列表。
func fallbackImportRoots(roots []importRootDefinition) []importRootDefinition {
	return injectFallbackImportRoot(roots, "")
}

func (s *Service) resolveImportRoot(ctx context.Context, provider string, rootID string) (importRootDefinition, error) {
	if strings.TrimSpace(provider) != "" && strings.TrimSpace(provider) != importProviderLocal {
		return importRootDefinition{}, fmt.Errorf("%w: unsupported provider", errProjectDirectoryForbidden)
	}
	roots, err := s.importRootDefinitions(ctx)
	if err != nil {
		return importRootDefinition{}, err
	}
	for _, root := range roots {
		if root.id == strings.TrimSpace(rootID) {
			return root, nil
		}
	}
	return importRootDefinition{}, fmt.Errorf("%w: unknown root", errProjectDirectoryForbidden)
}

// sameWorkingDirectory 判断两个工作目录在去除首尾空白后是否相同。
// sameWorkingDirectory 判断两个工作目录路径是否相同。
// @returns 去除首尾空白并忽略大小写后路径相同则为 `true`，否则为 `false`。
func sameWorkingDirectory(left string, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

// toProjectListItemWithManagedRoot 将聚合信息映射为项目列表项，并在提供运行时摘要时补充容器数量。
// 结果包含项目标识、名称、来源、工作目录、声明服务数，以及最近刷新和漂移状态。
func toProjectListItemWithManagedRoot(
	aggregate projectstore.ProjectAggregate,
	managedRootDirectory string,
	runtimeSummary ...moduleapi.ContainerProjectRuntimeSummary,
) generated.ProjectListItem {
	serviceCount := 0
	if aggregate.Snapshot != nil {
		serviceCount = aggregate.Snapshot.DeclaredServiceCount
	}
	counts := generated.ProjectContainerCounts{}
	if len(runtimeSummary) > 0 {
		counts.Running = runtimeSummary[0].RunningCount
		counts.Stopped = runtimeSummary[0].StoppedCount
		counts.Total = runtimeSummary[0].RunningCount + runtimeSummary[0].StoppedCount
	}
	return generated.ProjectListItem{
		Id:                         mustGeneratedID(aggregate.Project.ID),
		DisplayName:                aggregate.Project.DisplayName,
		CanonicalProjectName:       aggregate.Project.CanonicalProjectName,
		CanonicalProjectNameSource: generated.ProjectCanonicalNameSource(aggregate.Project.CanonicalProjectNameSource),
		SourceKind:                 generated.ProjectSourceKind(aggregate.Project.SourceKind),
		SourceMetadata:             buildListSourceMetadataWithManagedRoot(aggregate, managedRootDirectory),
		ActivityAuthority:          generated.ProjectActivityAuthority(resolveActivityAuthority(aggregate)),
		HostScope:                  generated.ProjectHostScope(aggregate.Project.HostScope),
		OwnershipMode:              generated.ProjectOwnershipMode(aggregate.Project.OwnershipMode),
		WorkingDirectory:           aggregate.Project.WorkingDirectory,
		ServiceCount:               serviceCount,
		ContainerCounts:            counts,
		LastRefreshStatus:          generated.ProjectRefreshStatus(aggregate.Project.LastRefreshStatus),
		LastRefreshAt:              aggregate.Project.LastRefreshAt,
		DriftStatus:                generated.ProjectDriftStatus(aggregate.Project.DriftStatus),
	}
}

// toProjectDetailResponse 将项目聚合数据转换为详情响应。
//
// toProjectDetailResponse 将项目聚合转换为详情响应，并在提供运行时汇总时填充容器运行与停止数量。
// 当聚合包含快照时，会写入服务数；刷新错误信息和配置哈希仅在存在时写入响应。
func toProjectDetailResponse(
	aggregate projectstore.ProjectAggregate,
	runtimeSummary ...moduleapi.ContainerProjectRuntimeSummary,
) generated.ProjectDetailResponse {
	return toProjectDetailResponseWithManagedRoot(aggregate, "", runtimeSummary...)
}

func toProjectDetailResponseWithManagedRoot(
	aggregate projectstore.ProjectAggregate,
	managedRootDirectory string,
	runtimeSummary ...moduleapi.ContainerProjectRuntimeSummary,
) generated.ProjectDetailResponse {
	counts := generated.ProjectContainerCounts{}
	if len(runtimeSummary) > 0 {
		counts.Running = runtimeSummary[0].RunningCount
		counts.Stopped = runtimeSummary[0].StoppedCount
		counts.Total = runtimeSummary[0].RunningCount + runtimeSummary[0].StoppedCount
	}
	item := generated.ProjectDetailResponse{
		CanonicalProjectName:       aggregate.Project.CanonicalProjectName,
		CanonicalProjectNameSource: generated.ProjectCanonicalNameSource(aggregate.Project.CanonicalProjectNameSource),
		ComposeFiles:               toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())),
		ContainerCounts:            counts,
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
		SourceMetadata:             buildDetailSourceMetadataWithManagedRoot(aggregate, managedRootDirectory),
		ActivityAuthority:          generated.ProjectActivityAuthority(resolveActivityAuthority(aggregate)),
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

// toGeneratedFiles 将存储的文件记录转换为生成的文件项列表。
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

// 将 compose 投影文件转换为生成的项目文件项列表。
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

// toStoreFiles 将 compose 和 env 文件投影转换为存储层文件记录。
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

// 返回的文件先按 OrderIndex 升序排列，OrderIndex 相同时按 ID 升序排列。
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

// collectFilesByKind 返回指定类型的文件绝对路径列表。
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

// normalizeSnapshotJSON 将输入内容规范化为 JSON 表示。
// 输入为空时返回 "{}"；当解析或重新编码失败时，返回原始内容。
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

// yamlJSONRoundTrip 将 JSON 数据解析到目标值。
//
// @param raw 要解析的数据。
// @param target 接收解析结果的目标值。
// @returns 解析过程中返回的错误。
func yamlJSONRoundTrip(raw []byte, target any) error {
	return json.Unmarshal(raw, target)
}

// digestServiceNames 计算服务名称集合的稳定摘要。
// digestServiceNames 对服务名按字典序排序后计算摘要。
// 返回排序后的名称序列对应的 SHA-256 十六进制字符串。
func digestServiceNames(names []string) string {
	normalized := append([]string(nil), names...)
	sort.Strings(normalized)
	hasher := sha256.New()
	for _, item := range normalized {
		mustWriteDigestFragment(hasher, []byte(item))
		mustWriteDigestFragment(hasher, []byte{0})
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// mustWriteDigestFragment 将摘要片段写入给定写入器。
// 写入失败时会 panic。
func mustWriteDigestFragment(writer io.Writer, value []byte) {
	if _, err := writer.Write(value); err != nil {
		panic(fmt.Sprintf("project digest writer failed: %v", err))
	}
}

// buildRefreshProjectInput 组装项目刷新持久化输入，包含刷新状态、快照、文件与操作者信息。
// buildRefreshProjectInput 构建用于刷新项目存储记录的输入。
// 它写入刷新状态、刷新时间、配置哈希、归一化后的 compose 快照和声明服务摘要，并保留操作者信息。
func buildRefreshProjectInput(
	projectID uint64,
	parseResult projectcompose.Result,
	now time.Time,
	actorID *uint64,
) projectstore.RefreshProjectInput {
	return projectstore.RefreshProjectInput{
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
	}
}

// displayNameOrCanonical 返回修剪后的显示名称或规范名称。
//
// 当显示名称存在且非空时，返回其去除首尾空白后的值；否则返回规范名称。
func displayNameOrCanonical(displayName *string, canonical string) string {
	if displayName != nil && strings.TrimSpace(*displayName) != "" {
		return strings.TrimSpace(*displayName)
	}
	return canonical
}

// fileName 返回路径的最后一个段。
// 它按正斜杠分割路径并取最后一项。
func fileName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// ensureManagedProjectAggregate 仅允许受控根目录专用归属模式的项目进入受控草案流程。
// ensureManagedProjectAggregate 检查聚合是否处于 managed-root-dedicated 归属模式，并在不满足时返回 errProjectManagedFlow。
func ensureManagedProjectAggregate(aggregate projectstore.ProjectAggregate) error {
	if aggregate.Project.OwnershipMode != projectcontract.OwnershipModeManagedRootDedicated.String() {
		return errProjectManagedFlow
	}
	return nil
}

func (s *Service) prepareConfigurationDraft(
	aggregate projectstore.ProjectAggregate,
	draft ConfigurationDraft,
) (preparedConfigurationDraft, error) {
	current, err := loadManagedDraftContent(aggregate)
	if err != nil {
		return preparedConfigurationDraft{}, err
	}
	composeContent := normalizeTextBlock(draft.ComposeFileContent)
	if strings.TrimSpace(composeContent) == "" {
		return preparedConfigurationDraft{}, errProjectInvalidArgument
	}
	proposal := managedDraftProposal{
		ComposePath:    current.ComposePath,
		ComposeContent: composeContent,
		EnvPath:        current.EnvPath,
	}
	if draft.EnvFileContent != nil {
		content := normalizeTextBlock(*draft.EnvFileContent)
		proposal.EnvContent = &content
	} else if current.EnvPath != "" {
		content := current.EnvContent
		proposal.EnvContent = &content
	}
	draftInput, err := buildManagedDraftInput(aggregate.Project.WorkingDirectory, proposal)
	if err != nil {
		return preparedConfigurationDraft{}, err
	}
	parseResult, err := projectcompose.Load(draftInput)
	if err != nil {
		return preparedConfigurationDraft{}, err
	}
	warnings := make([]string, 0, draftWarningsCap)
	if strings.TrimSpace(current.ComposeContent) == strings.TrimSpace(proposal.ComposeContent) &&
		strings.TrimSpace(current.EnvContent) == strings.TrimSpace(derefString(proposal.EnvContent)) {
		warnings = append(warnings, "Draft matches the current tracked managed project files.")
	}
	return preparedConfigurationDraft{
		Proposal:    proposal,
		ParseResult: parseResult,
		Warnings:    warnings,
	}, nil
}

// buildConfigurationDiffFile 构建配置文件的差异结果，包含内容变更、哈希和统一 diff。
//
// 返回的结果会反映当前内容与提议内容的归一化比较，并保留对应的文件类型和路径。
func buildConfigurationDiffFile(kind string, path string, current string, proposed string) ConfigurationDiffFile {
	return ConfigurationDiffFile{
		Kind:            kind,
		Path:            path,
		Changed:         normalizeTextBlock(current) != normalizeTextBlock(proposed),
		CurrentHash:     hashString(current),
		ProposedHash:    hashString(proposed),
		CurrentContent:  normalizeTextBlock(current),
		ProposedContent: buildUnifiedDiff(current, proposed),
	}
}

// buildUnifiedDiff 生成当前内容与提议内容之间的统一差异文本。
// 当差异文本为空或仅包含空白字符时，返回规范化后的提议内容。
func buildUnifiedDiff(current string, proposed string) string {
	differ := diffmatchpatch.New()
	patches := differ.PatchMake(current, proposed)
	text := differ.PatchToText(patches)
	if strings.TrimSpace(text) == "" {
		return normalizeTextBlock(proposed)
	}
	return text
}

// hashString 返回归一化文本块的 SHA-256 十六进制摘要。
func hashString(value string) string {
	sum := sha256.Sum256([]byte(normalizeTextBlock(value)))
	return hex.EncodeToString(sum[:])
}

// normalizeTextBlock 规范化文本块的换行、行尾空白和整体边界，并在非空时补充结尾换行符。
func normalizeTextBlock(value string) string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " \t")
	}
	joined := strings.TrimSpace(strings.Join(lines, "\n"))
	if joined == "" {
		return ""
	}
	return joined + "\n"
}

// derefString 返回指针指向的字符串值。
// 如果指针为空，则返回空字符串。
func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// nonEmptyString 在 primary 为空白时返回 fallback。
//
// @return primary 经修剪后非空时返回其原值；否则返回 fallback。
func nonEmptyString(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

// stringPointer 返回一个指向非空白字符串的指针。
//
// 如果输入经修剪后为空，则返回 nil。
func stringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// optionalString 将字符串包装为可选字符串指针。
func optionalString(value string) *string {
	return stringPointer(value)
}

// stringValue 返回指针指向的字符串；当指针为 nil 时返回空字符串。
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// mapManagedSourceCatalogStatus 将托管根状态映射为来源目录状态。
//
// @param status 托管根状态字符串。
// @returns 目录状态值；当状态为 ready 时返回 "ready"，其余情况返回 "blocked"。
func mapManagedSourceCatalogStatus(status string) string {
	switch strings.TrimSpace(status) {
	case projectcontract.ManagedRootStatusReady.String():
		return "ready"
	case projectcontract.ManagedRootStatusUnconfigured.String(), projectcontract.ManagedRootStatusInvalid.String():
		return "blocked"
	default:
		return "blocked"
	}
}

// managedRootStatusReasonKey 将托管根状态映射为状态原因键。
// 返回与给定托管根状态对应的原因键；当状态为就绪时返回 nil。
func managedRootStatusReasonKey(status string) *string {
	switch strings.TrimSpace(status) {
	case projectcontract.ManagedRootStatusReady.String():
		return nil
	case projectcontract.ManagedRootStatusUnconfigured.String():
		return stringPointer("project.createSource.statusReason.managedUnconfigured")
	case projectcontract.ManagedRootStatusInvalid.String():
		return stringPointer("project.createSource.statusReason.managedInvalid")
	default:
		return stringPointer("project.createSource.statusReason.managedUnknown")
	}
}

// toGeneratedSourceMetadata 将源元数据映射为生成的项目来源元数据。
//
// 仅在至少有一个已知字段可映射时返回结果；否则返回 nil。
func toGeneratedSourceMetadata(metadata map[string]string) *generated.ProjectSourceMetadata {
	if len(metadata) == 0 {
		return nil
	}
	result := generated.ProjectSourceMetadata{}
	assignSourceMetadataField(metadata, "managed_root_key", &result.ManagedRootKey)
	assignSourceMetadataField(metadata, "managed_relative_directory", &result.ManagedRelativeDirectory)
	assignSourceMetadataField(metadata, "managed_compose_file_name", &result.ManagedComposeFileName)
	assignSourceMetadataField(metadata, "managed_env_file_name", &result.ManagedEnvFileName)
	assignSourceMetadataField(metadata, "git_repository_url", &result.GitRepositoryUrl)
	assignSourceMetadataField(metadata, "git_reference", &result.GitReference)
	assignSourceMetadataField(metadata, "git_compose_subpath", &result.GitComposeSubpath)
	assignSourceMetadataField(metadata, "template_key", &result.TemplateKey)
	assignSourceMetadataField(metadata, "template_version", &result.TemplateVersion)
	assignSourceMetadataField(metadata, "template_instance_name", &result.TemplateInstanceName)
	if result == (generated.ProjectSourceMetadata{}) {
		return nil
	}
	return &result
}

// assignSourceMetadataField 将来源元数据中的指定值去除首尾空白后写入目标指针。
// 当对应值为空时保持目标不变。
func assignSourceMetadataField(metadata map[string]string, key string, target **string) {
	value := strings.TrimSpace(metadata[key])
	if value == "" {
		return
	}
	*target = &value
}

// buildListSourceMetadataWithManagedRoot 为项目列表构建来源元数据。
// 当来源类型为受托管根或远程主机时返回对应的来源元数据；其他来源类型返回 nil。
func buildListSourceMetadataWithManagedRoot(aggregate projectstore.ProjectAggregate, managedRootDirectory string) *generated.ProjectSourceMetadata {
	switch strings.TrimSpace(aggregate.Project.SourceKind) {
	case projectcontract.SourceKindManaged.String():
		return buildManagedSourceMetadata(aggregate, managedRootDirectory)
	case projectcontract.SourceKindRemoteHost.String():
		return buildRemoteHostSourceMetadata(aggregate)
	default:
		return nil
	}
}

// buildDetailSourceMetadataWithManagedRoot 返回项目详情来源元数据。
// 如果没有可映射的来源信息，则返回 nil。
func buildDetailSourceMetadataWithManagedRoot(aggregate projectstore.ProjectAggregate, managedRootDirectory string) *generated.ProjectSourceMetadata {
	switch strings.TrimSpace(aggregate.Project.SourceKind) {
	case projectcontract.SourceKindManaged.String():
		return buildManagedSourceMetadata(aggregate, managedRootDirectory)
	case projectcontract.SourceKindRemoteHost.String():
		return buildRemoteHostSourceMetadata(aggregate)
	default:
		return nil
	}
}

// buildManagedSourceMetadata 生成托管项目的来源元数据。
// 结果包含托管根标识、相对目录，以及已登记的 Compose 和环境文件名。
func buildManagedSourceMetadata(aggregate projectstore.ProjectAggregate, managedRootDirectory string) *generated.ProjectSourceMetadata {
	composeFiles := filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())
	envFiles := filterFiles(aggregate.Files, projectcontract.FileKindEnv.String())
	metadata := map[string]string{
		"managed_root_key": projectcontract.ProjectManagedRootConfig.String(),
	}
	if relativePath := deriveManagedRelativeDirectory(managedRootDirectory, aggregate.Project.WorkingDirectory); relativePath != "" {
		metadata["managed_relative_directory"] = relativePath
	}
	if len(composeFiles) > 0 {
		metadata["managed_compose_file_name"] = filepath.Base(composeFiles[0].AbsolutePath)
	}
	if len(envFiles) > 0 {
		metadata["managed_env_file_name"] = filepath.Base(envFiles[0].AbsolutePath)
	}
	return toGeneratedSourceMetadata(metadata)
}

// buildRemoteHostSourceMetadata 构建远程主机来源元数据。
// 元数据包含活动权威和汇总范围。
func buildRemoteHostSourceMetadata(aggregate projectstore.ProjectAggregate) *generated.ProjectSourceMetadata {
	activityAuthority := string(resolveActivityAuthority(aggregate))
	rollupScope := "planned-remote-summary"
	return &generated.ProjectSourceMetadata{
		ActivityAuthority:   &activityAuthority,
		ActivityRollupScope: &rollupScope,
	}
}

// resolveActivityAuthority 根据项目主机范围确定活动执行方式。
// 当项目的 HostScope 为 remote 时返回 `ProjectActivityAuthorityBackendPlanned`，否则返回 `ProjectActivityAuthorityFrontendFanout`。
func resolveActivityAuthority(aggregate projectstore.ProjectAggregate) ActivityAuthority {
	if strings.TrimSpace(aggregate.Project.HostScope) == projectcontract.HostScopeRemote.String() {
		return ProjectActivityAuthorityBackendPlanned
	}
	return ProjectActivityAuthorityFrontendFanout
}

// deriveManagedRelativeDirectory 从工作目录推导托管相对目录。
// 当存在可用 managed root 时返回其相对路径；否则回退到清理后的路径基名。
func deriveManagedRelativeDirectory(managedRootDirectory string, workingDirectory string) string {
	cleaned := filepath.Clean(strings.TrimSpace(workingDirectory))
	if cleaned == "" || cleaned == "." || cleaned == string(filepath.Separator) {
		return ""
	}
	root := filepath.Clean(strings.TrimSpace(managedRootDirectory))
	if !hasUsableManagedRoot(root) {
		return filepath.Base(cleaned)
	}
	relative, err := filepath.Rel(root, cleaned)
	if err == nil && isUsableManagedRelativePath(relative) {
		return filepath.ToSlash(relative)
	}
	return filepath.Base(cleaned)
}

func hasUsableManagedRoot(root string) bool {
	return root != "" && root != "." && root != string(filepath.Separator)
}

func isUsableManagedRelativePath(relative string) bool {
	if relative == "" || relative == "." || relative == ".." {
		return false
	}
	return !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func (s *Service) readyManagedRootDirectory(ctx context.Context) string {
	if s == nil {
		return ""
	}
	managedRoot, err := s.ManagedRoot(ctx)
	if err != nil || managedRoot.Status != projectcontract.ManagedRootStatusReady.String() || managedRoot.ConfiguredRootDirectory == nil {
		return ""
	}
	return filepath.Clean(strings.TrimSpace(*managedRoot.ConfiguredRootDirectory))
}

// uniqueStrings 返回去重后的字符串切片，保留首次出现的顺序。
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

// normalizeListLimit 将列表限制值规范到默认值或最大值范围内。
//
// @returns 规范后的列表限制值：当输入小于等于 0 时返回默认值，超过最大值时返回最大值，否则返回原值。
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

// mustGeneratedID 将生成的无符号 ID 转换为 int64。
// 当值为 0 或超出 int64 可表示范围时会 panic。
func mustGeneratedID(value uint64) int64 {
	if value == 0 || value > math.MaxInt64 {
		panic("project generated id out of range")
	}
	return int64(value)
}

// maxInt 返回 value 与 minimum 中较大的值。
func maxInt(value int, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

// mapStoreError 将存储层错误映射为本包使用的错误。
// 已知的无效输入、未找到、冲突和文件不存在错误会转换为对应的本地哨兵错误；其他错误原样返回。
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
