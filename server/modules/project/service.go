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

	"graft/server/internal/config"

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
	errProjectServiceUnavailable      = errors.New("project service is unavailable")
	errProjectInvalidArgument         = errors.New("project invalid argument")
	errProjectApplicationNameRequired = errors.New("project application name is required")
	errProjectInvalidApplicationName  = errors.New("project application name is invalid")
	errProjectApplicationNameOccupied = errors.New("project application name is occupied")
	errProjectInvalidCanonicalName    = errors.New("project invalid canonical name")
	errProjectNotFound                = errors.New("project not found")
	errProjectConflict                = errors.New("project conflict")
	errProjectImportValidation        = errors.New("project import validation failed")
	errProjectManagedRootUnconfigured = errors.New("project managed root is unconfigured")
	errProjectManagedRootInvalid      = errors.New("project managed root is invalid")
	errProjectInvalidCompose          = errors.New("project compose configuration is invalid")
	errProjectWorkspaceUnsafe         = errors.New("project managed workspace is unsafe")
	errProjectWorkspaceWriteFailed    = errors.New("project managed workspace write failed")
	errProjectUnsupportedLifecycle    = errors.New("project lifecycle is unsupported")
	errProjectLifecycleReview         = errors.New("project lifecycle configuration review required")
	errProjectFileNotFound            = errors.New("project file not found")
	errProjectDestroyBlocked          = errors.New("project destroy blocked by ownership guard")
	errProjectManagedFlow             = errors.New("project managed flow is unsupported")
	errProjectDirectoryForbidden      = errors.New("project directory browse forbidden")
	errProjectInspectionExpired       = errors.New("project inspection expired")
	errProjectInspectionStale         = errors.New("project inspection stale")
	errProjectFileHashMismatch        = errors.New("project file hash mismatch")
	errProjectRuntimeUnavailable      = errors.New("project runtime is unavailable")
	errProjectComposeNameOccupied     = errors.New("compose project name is already occupied on runtime target")
	errProjectActorAttribution        = errors.New("project actor attribution required")
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

// ListQuery 描述项目列表筛选条件。
type ListQuery struct {
	Limit   int
	Offset  int
	Keyword string
	// Sort accepts the restricted project-list sort expressions; an empty value uses created_at:desc.
	Sort            string
	ApplicationType string
	RuntimeTargetID *int64
	Provider        string
	SourceKind      string
	RuntimeStatus   string
	DriftStatus     string
}

// ImportRequest 描述当前阶段导入校验和导入请求载荷。
type ImportRequest struct {
	WorkingDirectory             string
	DisplayName                  *string
	ComposeFiles                 []string
	EnvFiles                     []string
	CanonicalProjectNameOverride *string
	ActorID                      *uint64
}

// ListResult 返回分页项目列表。
type ListResult struct {
	Items  []generated.ProjectListItem
	Total  int
	Limit  int
	Offset int
}

// CreationMethodCatalogResult 返回可用的 Compose 项目创建方式。
type CreationMethodCatalogResult struct {
	Items []generated.ProjectCreationMethod
}

// ComposeRuntimeTargets 返回实现 Compose 能力契约的已注册运行时目标。
func (s *Service) ComposeRuntimeTargets(ctx context.Context) ([]moduleapi.ComposeRuntimeTargetSummary, error) {
	return s.listComposeTargets(ctx)
}

// ActivityAuthority 标识稳定的项目活动权威契约。
type ActivityAuthority string

const (
	// ProjectActivityAuthorityFrontendFanout keeps project activity in frontend fan-out over container authority.
	ProjectActivityAuthorityFrontendFanout ActivityAuthority = "frontend-fanout"
	// ProjectActivityAuthorityBackendPlanned reserves a future backend aggregation owner without implementing it yet.
	ProjectActivityAuthorityBackendPlanned ActivityAuthority = "backend-planned"
)

// DiscoveryCandidateResult 返回一个有界目录扫描或自动发现预览候选。
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

// DiscoveryCandidatesResult 返回有界扫描/发现候选权威表面。
type DiscoveryCandidatesResult struct {
	SourceType            string
	AuthorityRoot         *string
	SupportsScan          bool
	SupportsAutoDiscovery bool
	StatusReason          *string
	Items                 []DiscoveryCandidateResult
}

// ImportValidationResult 返回静态导入校验结果。
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

// ConfigurationMetadataResult 返回只读配置元数据。
type ConfigurationMetadataResult struct {
	ProjectID          uint64
	ComposeFiles       []generated.ProjectFileItem
	EnvFiles           []generated.ProjectFileItem
	OwnershipMode      string
	DriftStatus        string
	DiagnosticsSummary []string
}

// ConfigurationPreviewResult 返回只读规范化 Compose 配置预览。
type ConfigurationPreviewResult struct {
	ProjectID             uint64
	CanonicalProjectName  string
	ConfigHash            string
	NormalizedComposeYAML string
	RefreshedAt           *time.Time
}

// ConfigurationFileResult 返回只读原始文件内容。
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

// LifecycleStrategyKind 标识内部生命周期策略 owner。
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

// LifecycleReviewStatus 标识生命周期配置是否可以执行。
type LifecycleReviewStatus string

const (
	// LifecycleReviewStatusReviewRequired blocks lifecycle execution until the user reviews imported defaults.
	LifecycleReviewStatusReviewRequired LifecycleReviewStatus = "review_required"
	// LifecycleReviewStatusConfirmed allows lifecycle execution with the persisted configuration.
	LifecycleReviewStatusConfirmed LifecycleReviewStatus = "confirmed"
)

// LifecycleStandardConfig 保存可编辑的标准 Compose 执行选项。
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

// LifecycleConfiguration 保存项目拥有的生命周期执行配置。
type LifecycleConfiguration struct {
	StrategyKind LifecycleStrategyKind
	ReviewStatus LifecycleReviewStatus
	WorkingDir   string
	ComposeFiles []string
	ProjectName  string
	Standard     LifecycleStandardConfig
}

// ActionResult 返回第一阶段有界动作状态。
type ActionResult struct {
	ProjectID    uint64
	Action       generated.ProjectActionResponseAction
	Result       generated.ProjectActionResponseResult
	MessageKey   *string
	Message      *string
	GuardResults []GuardResult
}

// BatchActionRequest 描述一次项目批量动作执行。
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

// BatchActionItemResult 返回一个项目的批量动作结果。
type BatchActionItemResult struct {
	ActionResult
	Skipped bool
}

// BatchActionResult 返回带逐项目结果的批量动作汇总结果。
type BatchActionResult struct {
	TotalCount     int
	CompletedCount int
	BlockedCount   int
	SkippedCount   int
	Items          []BatchActionItemResult
}

// GuardResult 是项目动作被阻断或受保护时使用的稳定结构化契约。
type GuardResult struct {
	Code       string
	MessageKey *string
	Detail     *string
}

// DestroyRequest 描述受保护销毁选项。
type DestroyRequest struct {
	RemoveNamedVolumes          bool
	AutoUnregister              bool
	ImagePrune                  bool
	DeleteWorkingDirectory      bool
	ConfirmCanonicalProjectName string
	ActorID                     *uint64
}

// ManagedRootInfo 返回有界的受管根目录契约元数据。
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

// ManagedProjectCreateRequest 描述受管创建契约载荷。
type ManagedProjectCreateRequest struct {
	DisplayName            string
	RuntimeTargetID        uint64
	ApplicationName        *string
	ReuseExistingWorkspace bool
	ComposeFileName        string
	ComposeFileContent     string
	EnvFileName            *string
	EnvFileContent         *string
	WorkspaceEntries       []ManagedWorkspaceEntry
	ComposeFilePath        string
	EnvFilePaths           []string
	LifecycleConfig        *LifecycleStandardConfig
}

// ManagedWorkspaceEntry 表示任意 UTF-8 文本文件或空/非空目录。
type ManagedWorkspaceEntry struct {
	Path     string
	NodeType string
	Content  *string
}

// ManagedProjectCreateValidationResult 返回创建契约校验元数据，不写入文件。
type ManagedProjectCreateValidationResult struct {
	ManagedRoot             ManagedRootInfo
	SourceType              string
	DisplayName             string
	ComposeProjectName      string
	ApplicationName         *string
	OwnershipMode           string
	WorkspacePath           string
	WorkingDirectory        string
	CanonicalProjectName    string
	ComposeFileName         string
	EnvFileName             *string
	ComposeFileAbsolutePath string
	EnvFileAbsolutePath     *string
	SourceMetadata          map[string]string
	Warnings                []string
	ReusedExistingWorkspace bool
}

// ManagedProjectCreateResult 返回文件写入并持久化后的受管项目启动信息。
type ManagedProjectCreateResult struct {
	Validation           ManagedProjectCreateValidationResult
	SourceType           string
	ProjectID            uint64
	ApplicationID        string
	ConfigHash           string
	DeclaredServiceCount int
	RefreshedAt          time.Time
}

// Service 拥有项目注册、导入及只读刷新/配置用例。
type Service struct {
	repository                   projectstore.Repository
	runtimeReader                moduleapi.ContainerProjectRuntimeReader
	resourceReader               moduleapi.ContainerProjectResourceReader
	logReader                    moduleapi.ContainerProjectLogReader
	configResolver               moduleapi.SystemConfigResolver
	savedViews                   moduleapi.SavedViewService
	runtimeTargets               moduleapi.ComposeRuntimeTargetReader
	authorizer                   moduleapi.Authorizer
	realtimeTickets              realtimeauth.Service
	realtimeHub                  realtime.Hub
	topicIssuers                 realtime.TopicIssuerRegistry
	streamersMu                  sync.Mutex
	workspaceMutationMu          sync.Mutex
	applicationNameMu            sync.Mutex
	listTopicStreamer            *projectListTopicStreamer
	runtimeTopicStreamer         *projectRuntimeTopicStreamer
	lifecycleConfigTopicStreamer *projectLifecycleConfigTopicStreamer
	logTopicStreamer             *projectLogTopicStreamer
	inspectCache                 *importInspectionCache
	auditBus                     eventbus.Bus
	logger                       *zap.Logger
	moduleName                   string
	taskService                  moduleapi.TaskService
	debugConfig                  config.ProjectConfig
}

// SetTaskService 配置平台拥有的 Task Runtime 提交边界。
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

// ServiceOption 定制项目服务依赖。
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

// SetSavedViewService injects the generic saved-view persistence boundary.
func (s *Service) SetSavedViewService(service moduleapi.SavedViewService) {
	if s != nil {
		s.savedViews = service
	}
}

// SetRuntimeTargetReader injects the narrow Runtime Target identity authority.
func (s *Service) SetRuntimeTargetReader(reader moduleapi.ComposeRuntimeTargetReader) {
	if s != nil {
		s.runtimeTargets = reader
	}
}

// WithRuntimeTargetReader configures the Compose runtime target reader used by the service.
func WithRuntimeTargetReader(reader moduleapi.ComposeRuntimeTargetReader) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.runtimeTargets = reader
	})
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
	if query.ApplicationType != "" && query.ApplicationType != "compose" {
		return ListResult{}, errProjectInvalidArgument
	}
	if query.Provider != "" && query.Provider != "docker" {
		return ListResult{Items: []generated.ProjectListItem{}, Limit: normalizeListLimit(query.Limit), Offset: maxInt(query.Offset, 0)}, nil
	}
	targets, err := s.listComposeTargets(ctx)
	if err != nil {
		return ListResult{}, err
	}
	targetByID := runtimeTargetLookup(targets)
	if !validRuntimeTargetID(query.RuntimeTargetID, targetByID) {
		return ListResult{}, errProjectInvalidArgument
	}
	storeQuery := projectstore.ListQuery{
		Limit:           query.Limit,
		Offset:          query.Offset,
		Keyword:         strings.TrimSpace(query.Keyword),
		Sort:            strings.TrimSpace(query.Sort),
		RuntimeTargetID: query.RuntimeTargetID,
		SourceKind:      strings.TrimSpace(query.SourceKind),
		DriftStatus:     strings.TrimSpace(query.DriftStatus),
	}
	if query.RuntimeStatus != "" {
		return s.listRuntimeStatusPage(ctx, repository, storeQuery, query, targetByID)
	}
	storeResult, err := repository.List(ctx, storeQuery)
	if err != nil {
		return ListResult{}, mapStoreError(err)
	}
	items := s.mapProjectListItems(ctx, storeResult.Items, targetByID, "")
	return ListResult{Items: items, Total: storeResult.Total, Limit: normalizeListLimit(query.Limit), Offset: maxInt(query.Offset, 0)}, nil
}

// 当 ID 为空时返回 true；当 ID 小于 1 或不对应已知目标时返回 false。
func validRuntimeTargetID(id *int64, targets map[uint64]moduleapi.ComposeRuntimeTargetSummary) bool {
	// nil 表示未指定筛选；非 nil 值必须命中已发现的运行时目标，避免把未知 ID 当作有效过滤条件。
	if id == nil {
		return true
	}
	if *id < 1 {
		return false
	}
	_, ok := targets[uint64(*id)] // #nosec G115 -- positivity is checked immediately above.
	return ok
}

// listRuntimeStatusPage applies the runtime-owned status filter before pagination.
func (s *Service) listRuntimeStatusPage(
	ctx context.Context,
	repository projectstore.Repository,
	storeQuery projectstore.ListQuery,
	query ListQuery,
	targetByID map[uint64]moduleapi.ComposeRuntimeTargetSummary,
) (ListResult, error) {
	// 运行时状态是容器 authority 的派生筛选，必须先筛选完整结果再分页，避免页内数量漂移。
	matched := make([]generated.ProjectListItem, 0)
	offset := 0
	for {
		storeQuery.Limit = maxProjectListLimit
		storeQuery.Offset = offset
		page, err := repository.List(ctx, storeQuery)
		if err != nil {
			return ListResult{}, mapStoreError(err)
		}
		matched = append(matched, s.mapProjectListItems(ctx, page.Items, targetByID, query.RuntimeStatus)...)
		offset += len(page.Items)
		if len(page.Items) == 0 || offset >= page.Total {
			break
		}
	}
	pageOffset := maxInt(query.Offset, 0)
	if pageOffset > len(matched) {
		pageOffset = len(matched)
	}
	pageEnd := minInt(pageOffset+normalizeListLimit(query.Limit), len(matched))
	return ListResult{Items: matched[pageOffset:pageEnd], Total: len(matched), Limit: normalizeListLimit(query.Limit), Offset: maxInt(query.Offset, 0)}, nil
}

func (s *Service) mapProjectListItems(
	ctx context.Context,
	items []projectstore.ProjectAggregate,
	targetByID map[uint64]moduleapi.ComposeRuntimeTargetSummary,
	runtimeStatus string,
) []generated.ProjectListItem {
	managedRootDirectory := s.readyManagedRootDirectory(ctx)
	mappedItems := make([]generated.ProjectListItem, 0, len(items))
	for _, item := range items {
		runtimeSummary, runtimeErr := s.runtimeSummary(ctx, item)
		mapped := toProjectListItemWithManagedRoot(item, managedRootDirectory, &runtimeSummary, runtimeErr)
		mapped.ApplicationType = generated.ProjectListItemApplicationTypeCompose
		if item.Project.RuntimeTargetID != nil {
			if target, ok := targetByID[*item.Project.RuntimeTargetID]; ok {
				mapped.RuntimeTarget = &generated.ProjectRuntimeTargetSummary{Id: target.ID, DisplayName: target.DisplayName, Provider: generated.ProjectRuntimeTargetSummaryProvider(target.Provider)}
			}
		}
		if runtimeStatus != "" && (mapped.RuntimeStatus == nil || string(*mapped.RuntimeStatus) != runtimeStatus) {
			continue
		}
		mappedItems = append(mappedItems, mapped)
	}
	return mappedItems
}

func (s *Service) listComposeTargets(ctx context.Context) ([]moduleapi.ComposeRuntimeTargetSummary, error) {
	if s == nil || s.runtimeTargets == nil {
		return []moduleapi.ComposeRuntimeTargetSummary{}, nil
	}
	return s.runtimeTargets.ListComposeTargets(ctx)
}

// runtimeTargetLookup indexes valid runtime-target summaries by ID.
func runtimeTargetLookup(targets []moduleapi.ComposeRuntimeTargetSummary) map[uint64]moduleapi.ComposeRuntimeTargetSummary {
	// 仅索引合法的非零目标 ID；重复 ID 由后一次发现结果覆盖，保持最新的启动期摘要。
	byID := make(map[uint64]moduleapi.ComposeRuntimeTargetSummary, len(targets))
	for _, target := range targets {
		if target.ID > 0 {
			byID[uint64(target.ID)] = target
		}
	}
	return byID
}

// BackfillRuntimeTargets binds historical local Compose records after runtime-target Boot discovery.
func (s *Service) BackfillRuntimeTargets(ctx context.Context) error {
	if s == nil || s.repository == nil || s.runtimeTargets == nil {
		return nil
	}
	target, err := s.runtimeTargets.ReadComposeTarget(ctx, nil)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("backfill runtime targets: read compose target failed", zap.String("module", s.moduleName), zap.Error(err))
		}
		return nil
	}
	if target.ID < 1 {
		return nil
	}
	return s.repository.BackfillRuntimeTarget(ctx, uint64(target.ID))
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
	return s.importInspectionSession(ctx, session, importInspectionCommitInput{
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
		ApplicationId:      aggregate.Project.ApplicationID,
		ComposeProjectName: aggregate.Project.ComposeProjectName,
		CollectedAt:        optionalRFC3339Time(resourceSummary.CollectedAt),
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
	definitionKey := projectcontract.ApplicationRootDirectoryConfig.String()
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
		if rootInfo.Status == projectcontract.ManagedRootStatusInvalid.String() {
			return ManagedProjectCreateValidationResult{}, errProjectManagedRootInvalid
		}
		return ManagedProjectCreateValidationResult{}, errProjectManagedRootUnconfigured
	}

	normalized, err := normalizeManagedCreateRequest(request)
	if err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	workspace, err := s.resolveManagedCreateWorkspace(
		ctx,
		*rootInfo.ConfiguredRootDirectory,
		normalized.ApplicationName,
		request.ReuseExistingWorkspace,
	)
	if err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	composeName, composeContent, err := ensureComposeProjectName(normalized.ComposeFileContent, *normalized.ApplicationName)
	if err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	if err := s.ensureManagedCreateRuntimeNameAvailable(ctx, normalized.RuntimeTargetID, composeName); err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	normalized.ComposeFileContent = composeContent
	composeFileAbsolutePath := filepath.Join(workspace.workingDirectory, normalized.ComposeFileName)
	envFileAbsolutePath := managedCreateEnvAbsolutePath(workspace.workingDirectory, normalized.EnvFileName)
	warnings := make([]string, 0, managedCreateWarningsCap)
	warnings = append(warnings, "Managed create validation checks authority, normalized names, and target paths before any file-write execution.")
	if normalized.EnvFileName == nil {
		warnings = append(warnings, "No env file is declared; create execution will only materialize the compose file.")
	}

	sourceMetadata := managedCreateSourceMetadata(rootInfo.ConfigKey, *workspace.applicationName, normalized.ComposeFileName, normalized.EnvFileName)

	return ManagedProjectCreateValidationResult{
		ManagedRoot:             rootInfo,
		SourceType:              "managed",
		DisplayName:             normalized.DisplayName,
		ComposeProjectName:      composeName,
		ApplicationName:         workspace.applicationName,
		OwnershipMode:           projectcontract.OwnershipModeManagedRootDedicated.String(),
		WorkspacePath:           workspace.workingDirectory,
		WorkingDirectory:        workspace.workingDirectory,
		CanonicalProjectName:    composeName,
		ComposeFileName:         normalized.ComposeFileName,
		EnvFileName:             normalized.EnvFileName,
		ComposeFileAbsolutePath: composeFileAbsolutePath,
		EnvFileAbsolutePath:     envFileAbsolutePath,
		SourceMetadata:          sourceMetadata,
		Warnings:                warnings,
		ReusedExistingWorkspace: workspace.exists,
	}, nil
}

func managedCreateSourceMetadata(rootKey, applicationName, composeFileName string, envFileName *string) map[string]string {
	metadata := map[string]string{
		"managed_root_key":          rootKey,
		"application_name":          applicationName,
		"managed_compose_file_name": composeFileName,
	}
	if envFileName != nil {
		metadata["managed_env_file_name"] = *envFileName
	}
	return metadata
}

type managedCreateWorkspace struct {
	workingDirectory string
	applicationName  *string
	exists           bool
}

func (s *Service) resolveManagedCreateWorkspace(ctx context.Context, root string, applicationName *string, reuse bool) (managedCreateWorkspace, error) {
	if applicationName == nil {
		return managedCreateWorkspace{}, errProjectInvalidArgument
	}
	if err := s.ensureApplicationNameUnregistered(ctx, *applicationName); err != nil {
		return managedCreateWorkspace{}, err
	}
	workingDirectory := filepath.Join(root, *applicationName)
	reusable, err := readReusableWorkspace(workingDirectory)
	if err != nil {
		return managedCreateWorkspace{}, err
	}
	if reusable.exists && !reuse {
		return managedCreateWorkspace{}, errProjectApplicationNameOccupied
	}
	return managedCreateWorkspace{workingDirectory: workingDirectory, applicationName: applicationName, exists: reusable.exists}, nil
}

func (s *Service) ensureManagedCreateRuntimeNameAvailable(ctx context.Context, runtimeTargetID uint64, composeName string) error {
	if s.runtimeTargets == nil {
		return nil
	}
	targetID, err := s.resolveComposeRuntimeTarget(ctx, runtimeTargetID)
	if err != nil {
		return err
	}
	return s.ensureComposeProjectNameAvailableForCreate(ctx, targetID, composeName)
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
	// repository 是服务的持久化 authority；缺失时返回稳定错误而不是让调用方触发 nil pointer。
	if s == nil || s.repository == nil {
		return nil, errProjectServiceUnavailable
	}
	return s.repository, nil
}

// ResolveApplicationID resolves the only public Project HTTP identifier to the
// module-private key used by project-owned tables and task records.
func (s *Service) ResolveApplicationID(ctx context.Context, applicationID string) (uint64, error) {
	if !isApplicationID(applicationID) {
		return 0, errProjectInvalidArgument
	}
	repository, err := s.repositoryOrErr()
	if err != nil {
		return 0, err
	}
	lookup, ok := repository.(projectstore.ApplicationLookupRepository)
	if !ok {
		return 0, errProjectServiceUnavailable
	}
	aggregate, err := lookup.GetByApplicationID(ctx, applicationID)
	if err != nil {
		return 0, mapStoreError(err)
	}
	return aggregate.Project.ID, nil
}

func (s *Service) getAggregate(ctx context.Context, projectID uint64) (projectstore.ProjectAggregate, error) {
	// 所有详情、刷新和生命周期操作共享聚合读取入口，确保项目、文件与快照使用同一份存储视图。
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
		return fmt.Errorf("%w: %w", errProjectInvalidArgument, err)
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
