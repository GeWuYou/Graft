package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/eventbus"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

var (
	errProjectServiceUnavailable   = errors.New("project service is unavailable")
	errProjectInvalidArgument      = errors.New("project invalid argument")
	errProjectInvalidCanonicalName = errors.New("project invalid canonical name")
	errProjectNotFound             = errors.New("project not found")
	errProjectConflict             = errors.New("project conflict")
	errProjectImportValidation     = errors.New("project import validation failed")
	errProjectUnsupportedLifecycle = errors.New("project lifecycle is unsupported")
	errProjectLifecycleReview      = errors.New("project lifecycle configuration review required")
	errProjectFileNotFound         = errors.New("project file not found")
	errProjectDestroyBlocked       = errors.New("project destroy blocked by ownership guard")
	errProjectManagedFlow          = errors.New("project managed flow is unsupported")
	errProjectDirectoryForbidden   = errors.New("project directory browse forbidden")
	errProjectInspectionExpired    = errors.New("project inspection expired")
	errProjectInspectionStale      = errors.New("project inspection stale")
	errProjectFileHashMismatch     = errors.New("project file hash mismatch")
	errProjectRuntimeUnavailable   = errors.New("project runtime is unavailable")
	errProjectActorAttribution     = errors.New("project actor attribution required")
)

const (
	defaultProjectListLimit      = 20
	maxProjectListLimit          = 100
	projectConflictScanSize      = 100
	projectDiscoveryScanSize     = 8
	maxWorkspaceAnnotationLength = projectcontract.ProjectWorkspaceAnnotationMaxLength
	minLifecycleArgCount         = 2
	maxCommandOutputSummary      = 120
	managedCreateWarningsCap     = 2
	draftWarningsCap             = 2
	managedCreateDirMode         = 0o750
	managedCreateFileMode        = 0o600
	projectComposeTimeout        = 5 * time.Minute
	lifecycleRedeployStepCap     = 4

	importRuntimeCandidateStatusReady           = "ready"
	importRuntimeCandidateStatusAlreadyImported = "already_imported"
	importRuntimeCandidateStatusBrokenCompose   = "broken_compose"

	importRuntimeReasonAlreadyImported          = "already_imported"
	importRuntimeReasonComposeParseFailed       = "compose_parse_failed"
	importRuntimeReasonConfigFilesNotAccessible = "config_files_not_accessible"
)

// ListQuery describes project list filters.
type ListQuery struct {
	Limit       int
	Offset      int
	SourceKind  string
	DriftStatus string
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

// workspaceFileBrowseQuery describes one lazy-loaded project-root directory browse request.
type workspaceFileBrowseQuery struct {
	Path       string
	ShowHidden bool
}

// workspaceFileItem describes one file-tree row returned by the project workspace authority.
type workspaceFileItem struct {
	Name            string
	RelativePath    string
	NodeType        string
	FileKind        string
	Readable        bool
	Editable        bool
	LanguageHint    string
	SizeBytes       int64
	HiddenByDefault bool
	HasChildren     bool
	Tooltip         string
	TooltipSource   string
	ProjectNote     string
}

// workspaceFilesResult returns one bounded project-root directory page.
type workspaceFilesResult struct {
	ProjectID     uint64
	RootPath      string
	CurrentPath   string
	ParentPath    *string
	HasMoreHidden bool
	Items         []workspaceFileItem
}

// workspaceFileContentResult returns one path-based project file payload.
type workspaceFileContentResult struct {
	ProjectID    uint64
	RelativePath string
	FileKind     string
	LanguageHint string
	Readable     bool
	Editable     bool
	Encoding     string
	Content      string
	SizeBytes    int64
}

// workspaceFileSaveRequest describes one writable project file update request.
type workspaceFileSaveRequest struct {
	Content string
}

// workspaceFileSaveResult returns the saved file projection after one write.
type workspaceFileSaveResult struct {
	ProjectID    uint64
	RelativePath string
	SavedAt      time.Time
	ContentHash  string
	SizeBytes    int64
}

// LifecycleStrategyKind identifies the internal lifecycle strategy owner.
type LifecycleStrategyKind string

const (
	// LifecycleStrategyKindStandard runs bounded docker compose commands from project authority.
	LifecycleStrategyKindStandard LifecycleStrategyKind = "standard"
)

const (
	defaultLifecycleWaitTimeoutSeconds = 120
	minLifecycleWaitTimeoutSeconds     = 1
	maxLifecycleWaitTimeoutSeconds     = 3600
)

// LifecycleReviewStatus identifies whether a lifecycle config can execute.
type LifecycleReviewStatus string

const (
	// LifecycleReviewStatusReviewRequired blocks lifecycle execution until the user reviews imported defaults.
	LifecycleReviewStatusReviewRequired LifecycleReviewStatus = "review_required"
	// LifecycleReviewStatusConfirmed allows lifecycle execution with the persisted configuration.
	LifecycleReviewStatusConfirmed LifecycleReviewStatus = "confirmed"
)

// LifecycleStandardConfig stores editable standard compose execution options.
type LifecycleStandardConfig struct {
	Profiles                 []string
	DownBeforeRedeploy       bool
	PullBeforeRedeploy       bool
	BuildBeforeUp            bool
	ForceRecreate            bool
	RemoveOrphans            bool
	WaitAfterUp              bool
	WaitTimeoutSeconds       int
	RenewAnonVolumes         bool
	PruneImagesAfterRedeploy bool
	AdditionalArgs           []string
}

// LifecycleConfiguration stores the project-owned lifecycle execution configuration.
type LifecycleConfiguration struct {
	StrategyKind LifecycleStrategyKind
	ReviewStatus LifecycleReviewStatus
	WorkingDir   string
	ComposeFiles []string
	ProjectName  string
	Standard     LifecycleStandardConfig
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

// ActionResult returns bounded phase-1 action status.
type ActionResult struct {
	ProjectID    uint64
	Action       generated.ProjectActionResponseAction
	Result       generated.ProjectActionResponseResult
	MessageKey   *string
	Message      *string
	GuardResults []GuardResult
}

// BatchActionRequest describes one project batch-action execution.
type BatchActionRequest struct {
	Action                      generated.ProjectBatchActionRequestAction
	ProjectIDs                  []uint64
	RemoveNamedVolumes          bool
	AutoUnregister              bool
	ImagePrune                  bool
	DeleteWorkingDirectory      bool
	ConfirmCanonicalProjectName *string
	ActorID                     *uint64
}

// BatchActionItemResult returns one per-project batch-action result.
type BatchActionItemResult struct {
	ActionResult
	Skipped bool
}

// BatchActionResult returns the aggregate batch-action outcome with per-item results.
type BatchActionResult struct {
	TotalCount     int
	CompletedCount int
	BlockedCount   int
	SkippedCount   int
	Items          []BatchActionItemResult
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
	AutoUnregister              bool
	ImagePrune                  bool
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
	repository                   projectstore.Repository
	runtimeReader                moduleapi.ContainerProjectRuntimeReader
	resourceReader               moduleapi.ContainerProjectResourceReader
	logReader                    moduleapi.ContainerProjectLogReader
	configResolver               moduleapi.SystemConfigResolver
	authorizer                   moduleapi.Authorizer
	realtimeTickets              realtimeauth.Service
	realtimeHub                  realtime.Hub
	topicIssuers                 realtime.TopicIssuerRegistry
	streamersMu                  sync.Mutex
	listTopicStreamer            *projectListTopicStreamer
	runtimeTopicStreamer         *projectRuntimeTopicStreamer
	lifecycleConfigTopicStreamer *projectLifecycleConfigTopicStreamer
	logTopicStreamer             *projectLogTopicStreamer
	inspectCache                 *importInspectionCache
	auditBus                     eventbus.Bus
	logger                       *zap.Logger
	moduleName                   string
	taskService                  moduleapi.TaskService
}

// SetTaskService configures the platform-owned Task Runtime submission boundary.
func (s *Service) SetTaskService(service moduleapi.TaskService) {
	if s != nil {
		s.taskService = service
	}
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

// WithResourceReader 设置项目概览资源读取器。
func WithResourceReader(reader moduleapi.ContainerProjectResourceReader) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.resourceReader = reader
	})
}

// WithLogReader sets the project log reader.
func WithLogReader(reader moduleapi.ContainerProjectLogReader) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.logReader = reader
	})
}

// WithSystemConfigResolver 注入用于 managed-create 权限校验的系统配置读取边界。
func WithSystemConfigResolver(resolver moduleapi.SystemConfigResolver) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.configResolver = resolver
	})
}

// WithAuthorizer injects the authorization boundary required by realtime topic issuance.
func WithAuthorizer(authorizer moduleapi.Authorizer) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.authorizer = authorizer
	})
}

// WithRealtime injects the unified realtime topic issuance dependencies.
func WithRealtime(
	tickets realtimeauth.Service,
	hub realtime.Hub,
	issuers realtime.TopicIssuerRegistry,
) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.realtimeTickets = tickets
		s.realtimeHub = hub
		s.topicIssuers = issuers
	})
}

// SetRuntimeReader injects the runtime reader after module registration resolves cross-module services.
func (s *Service) SetRuntimeReader(reader moduleapi.ContainerProjectRuntimeReader) {
	if s == nil {
		return
	}
	s.runtimeReader = reader
}

// SetResourceReader injects the resource reader after module registration resolves cross-module services.
func (s *Service) SetResourceReader(reader moduleapi.ContainerProjectResourceReader) {
	if s == nil {
		return
	}
	s.resourceReader = reader
}

// SetLogReader injects the log reader after module registration resolves cross-module services.
func (s *Service) SetLogReader(reader moduleapi.ContainerProjectLogReader) {
	if s == nil {
		return
	}
	s.logReader = reader
}

// SetSystemConfigResolver injects the system-config resolver after module registration.
func (s *Service) SetSystemConfigResolver(resolver moduleapi.SystemConfigResolver) {
	if s == nil {
		return
	}
	s.configResolver = resolver
}

// SetAuthorizer injects the authorizer after module registration.
func (s *Service) SetAuthorizer(authorizer moduleapi.Authorizer) {
	if s == nil {
		return
	}
	s.authorizer = authorizer
}

// SetRealtime injects the unified realtime dependencies after module registration.
func (s *Service) SetRealtime(
	tickets realtimeauth.Service,
	hub realtime.Hub,
	issuers realtime.TopicIssuerRegistry,
) {
	if s == nil {
		return
	}
	s.realtimeTickets = tickets
	s.realtimeHub = hub
	s.topicIssuers = issuers
}

// SetAuditPublisher injects the audit event publication dependencies after module registration.
func (s *Service) SetAuditPublisher(bus eventbus.Bus, logger *zap.Logger, moduleName string) {
	if s == nil {
		return
	}
	s.auditBus = bus
	s.logger = logger
	s.moduleName = strings.TrimSpace(moduleName)
}

// List returns one page of registered projects.
func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ListResult{}, err
	}
	storeResult, err := repository.List(ctx, projectstore.ListQuery{
		Limit:       query.Limit,
		Offset:      query.Offset,
		SourceKind:  strings.TrimSpace(query.SourceKind),
		DriftStatus: strings.TrimSpace(query.DriftStatus),
	})
	if err != nil {
		return ListResult{}, mapStoreError(err)
	}
	managedRootDirectory := s.readyManagedRootDirectory(ctx)
	items := make([]generated.ProjectListItem, 0, len(storeResult.Items))
	for _, item := range storeResult.Items {
		runtimeSummary, runtimeErr := s.runtimeSummary(ctx, item)
		items = append(items, toProjectListItemWithManagedRoot(item, managedRootDirectory, &runtimeSummary, runtimeErr))
	}
	return ListResult{Items: items, Total: storeResult.Total, Limit: normalizeListLimit(query.Limit), Offset: maxInt(query.Offset, 0)}, nil
}

// Get returns one project detail payload.
func (s *Service) Get(ctx context.Context, projectID uint64) (generated.ProjectDetailResponse, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return generated.ProjectDetailResponse{}, err
	}
	runtimeSummary, runtimeErr := s.runtimeSummary(ctx, aggregate)
	return toProjectDetailResponseWithManagedRoot(aggregate, s.readyManagedRootDirectory(ctx), &runtimeSummary, runtimeErr), nil
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
	lifecycleConfig := defaultLifecycleStandardConfig()
	return s.importInspectionSession(ctx, repository, session, importInspectionCommitInput{
		DisplayName:       request.DisplayName,
		CanonicalOverride: request.CanonicalProjectNameOverride,
		LifecycleConfig:   &lifecycleConfig,
		ActorID:           request.ActorID,
	})
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
		Action:     generated.ProjectActionResponseActionProjectActionRefresh,
		Result:     generated.ProjectActionResponseResultProjectActionResultCompleted,
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

// Overview returns the project-owned dashboard overview backed by the container module resource aggregate.
func (s *Service) Overview(ctx context.Context, projectID uint64) (generated.ProjectOverviewResponse, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return generated.ProjectOverviewResponse{}, err
	}
	parseResult, err := s.loadFromAggregate(aggregate)
	if err != nil {
		return generated.ProjectOverviewResponse{}, err
	}
	resourceSummary, _ := s.projectResourceSummary(ctx, aggregate)
	serviceResources := make(map[string]moduleapi.ContainerProjectServiceResourceSummary, len(resourceSummary.Services))
	for _, item := range resourceSummary.Services {
		serviceResources[item.ServiceName] = item
	}
	items := make([]generated.ProjectOverviewServiceItem, 0, len(parseResult.Services))
	healthyServiceCount := 0
	for _, item := range parseResult.Services {
		runtimeItem, ok := serviceResources[item.ServiceName]
		if !ok {
			runtimeItem = moduleapi.ContainerProjectServiceResourceSummary{ServiceName: item.ServiceName}
		}
		generatedItem, isHealthy := toProjectOverviewServiceItem(item, runtimeItem)
		if isHealthy {
			healthyServiceCount++
		}
		items = append(items, generatedItem)
	}
	return generated.ProjectOverviewResponse{
		ProjectId:            mustGeneratedID(projectID),
		CanonicalProjectName: aggregate.Project.CanonicalProjectName,
		CollectedAt:          optionalRFC3339Time(resourceSummary.CollectedAt),
		Health: generated.ProjectOverviewHealthSummary{
			HealthyServiceCount:     healthyServiceCount,
			HealthyContainerCount:   resourceSummary.HealthyContainerCount,
			UnhealthyContainerCount: resourceSummary.UnhealthyContainerCount,
			StartingContainerCount:  resourceSummary.StartingContainerCount,
			RestartCount:            resourceSummary.RestartCount,
			NetworksCount:           countDeclaredNetworks(parseResult.Services),
			VolumesCount:            countDeclaredVolumes(parseResult.Services),
		},
		Resources: generated.ProjectOverviewResourceSummary{
			StatsAvailable:               resourceSummary.StatsAvailable,
			StatsAvailableContainerCount: resourceSummary.StatsAvailableContainerCount,
			CpuPercent:                   resourceSummary.CPUPercent,
			MemoryUsageBytes:             resourceSummary.MemoryUsageBytes,
			MemoryLimitBytes:             resourceSummary.MemoryLimitBytes,
			RxBytes:                      resourceSummary.RxBytes,
			TxBytes:                      resourceSummary.TxBytes,
		},
		Services: items,
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
// Up executes docker compose up -d within the project's registered working directory.
func (s *Service) runtimeSummary(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
) (moduleapi.ContainerProjectRuntimeSummary, error) {
	if s == nil || s.runtimeReader == nil {
		return moduleapi.ContainerProjectRuntimeSummary{
			CanonicalProjectName: aggregate.Project.CanonicalProjectName,
			Members:              []moduleapi.ContainerProjectMember{},
		}, errProjectRuntimeUnavailable
	}
	return s.runtimeReader.ListProjectMembers(ctx, aggregate.Project.HostScope, aggregate.Project.CanonicalProjectName)
}

func (s *Service) projectResourceSummary(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
) (moduleapi.ContainerProjectResourceSummary, error) {
	if s == nil || s.resourceReader == nil {
		return moduleapi.ContainerProjectResourceSummary{
			CanonicalProjectName: aggregate.Project.CanonicalProjectName,
			Services:             []moduleapi.ContainerProjectServiceResourceSummary{},
		}, errProjectRuntimeUnavailable
	}
	return s.resourceReader.ReadProjectResourceSummary(ctx, aggregate.Project.HostScope, aggregate.Project.CanonicalProjectName)
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

// sameWorkingDirectory 判断两个工作目录在去除首尾空白后是否相同。
// sameWorkingDirectory 判断两个工作目录路径是否相同。
// @returns 去除首尾空白并忽略大小写后路径相同则为 `true`，否则为 `false`。
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
