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
	"graft/server/internal/logger"

	"go.uber.org/zap"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/event"
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
	maxWorkspaceAnnotationLength = projectcontract.ApplicationWorkspaceAnnotationMaxLength
	minLifecycleArgCount         = 2
	maxCommandOutputSummary      = 120
	managedCreateWarningsCap     = 2
	draftWarningsCap             = 2
	managedCreateDirMode         = 0o750
	managedCreateFileMode        = 0o600
	projectComposeTimeout        = 5 * time.Minute
	lifecycleRedeployStepCap     = 4
	// localContainerRuntimeScope 只适配 container 模块当前的本地运行时查询边界，不属于 Application 持久化或公开契约。
	localContainerRuntimeScope = "local"

	importRuntimeCandidateStatusReady           = "ready"
	importRuntimeCandidateStatusAlreadyImported = "already_imported"
	importRuntimeCandidateStatusBrokenCompose   = "broken_compose"

	importRuntimeReasonAlreadyImported          = "already_imported"
	importRuntimeReasonComposeParseFailed       = "compose_parse_failed"
	importRuntimeReasonConfigFilesNotAccessible = "config_files_not_accessible"
)

// ListQuery 描述应用列表筛选条件。
type ListQuery struct {
	Limit   int
	Offset  int
	Keyword string
	// Sort 只接受应用列表白名单排序表达式；空值使用 created_at:desc。
	Sort                  string
	DeploymentAdapterKind string
	RuntimeTargetID       *int64
	Provider              string
	SourceType            string
	RuntimeStatus         string
	DriftStatus           string
}

// ImportRequest 描述当前阶段导入校验和导入请求载荷。
type ImportRequest struct {
	WorkspacePath              string
	DisplayName                *string
	ComposeFiles               []string
	EnvFiles                   []string
	ComposeProjectNameOverride *string
	ActorID                    *uint64
}

// ListResult 返回分页应用列表。
type ListResult struct {
	Items  []generated.ApplicationListItem
	Total  int
	Limit  int
	Offset int
}

// CreationMethodCatalogResult 返回可用的 Compose 应用创建方式。
type CreationMethodCatalogResult struct {
	Items []generated.ApplicationCreationMethod
}

// ComposeRuntimeTargets 返回实现 Compose 能力契约的已注册运行时目标。
func (s *Service) ComposeRuntimeTargets(ctx context.Context) ([]moduleapi.ComposeRuntimeTargetSummary, error) {
	return s.listComposeTargets(ctx)
}

// ActivityAuthority 标识稳定的应用活动聚合权威契约。
type ActivityAuthority string

const (
	// ApplicationActivityAuthorityFrontendFanout 表示前端基于 Container 权威数据聚合应用活动。
	ApplicationActivityAuthorityFrontendFanout ActivityAuthority = "frontend-fanout"
	// ApplicationActivityAuthorityBackendPlanned 仅保留未来后端聚合权威值，当前不实现第二套活动持久化。
	ApplicationActivityAuthorityBackendPlanned ActivityAuthority = "backend-planned"
)

// DiscoveryCandidateResult 返回一个有界目录扫描或自动发现预览候选。
type DiscoveryCandidateResult struct {
	CandidateKey             string
	CandidateKind            string
	SourceType               string
	SourceMetadata           map[string]string
	DisplayName              string
	ComposeProjectName       string
	ComposeProjectNameSource string
	WorkspacePath            string
	OwnershipMode            string
	Status                   string
	RecommendedAction        string
	StatusReason             *string
	ComposeFiles             []generated.ApplicationFileItem
	EnvFiles                 []generated.ApplicationFileItem
	DeclaredServiceNames     []string
	ServiceCount             int
	ConfigHash               string
	Warnings                 []string
	Conflicts                []string
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
	ComposeProjectName       string
	ComposeProjectNameSource string
	WorkspacePath            string
	ComposeFiles             []generated.ApplicationFileItem
	EnvFiles                 []generated.ApplicationFileItem
	ServiceCount             int
	NetworkNames             []string
	VolumeNames              []string
	Warnings                 []string
	Conflicts                []string
	ConfigHash               string
	DeclaredServiceNames     []string
	InspectionID             *string
}

// ConfigurationMetadataResult 返回只读配置元数据。
type ConfigurationMetadataResult struct {
	ApplicationRecordID uint64
	ApplicationID       string
	ComposeFiles        []generated.ApplicationFileItem
	EnvFiles            []generated.ApplicationFileItem
	OwnershipMode       string
	DriftStatus         string
	DiagnosticsSummary  []string
}

// ConfigurationPreviewResult 返回只读规范化 Compose 配置预览。
type ConfigurationPreviewResult struct {
	ApplicationRecordID   uint64
	ApplicationID         string
	ComposeProjectName    string
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

type workspaceFileBrowseQuery struct {
	Path       string
	ShowHidden bool
}

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
	ApplicationNote string
}

type workspaceFilesResult struct {
	ApplicationRecordID uint64
	ApplicationID       string
	RootPath            string
	CurrentPath         string
	ParentPath          *string
	HasMoreHidden       bool
	Items               []workspaceFileItem
}

type workspaceFileContentResult struct {
	ApplicationRecordID uint64
	ApplicationID       string
	RelativePath        string
	FileKind            string
	LanguageHint        string
	Readable            bool
	Editable            bool
	Encoding            string
	Content             string
	SizeBytes           int64
}

type workspaceFileSaveRequest struct {
	Content string
}

type workspaceFileSaveResult struct {
	ApplicationRecordID uint64
	ApplicationID       string
	RelativePath        string
	SavedAt             time.Time
	ContentHash         string
	SizeBytes           int64
}

// LifecycleStrategyKind 标识内部生命周期策略 owner。
type LifecycleStrategyKind string

const (
	// LifecycleStrategyKindStandard 表示从应用契约生成有界 Docker Compose 命令。
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
	// LifecycleReviewStatusReviewRequired 表示导入默认值尚未确认，禁止执行生命周期动作。
	LifecycleReviewStatusReviewRequired LifecycleReviewStatus = "review_required"
	// LifecycleReviewStatusConfirmed 表示持久化配置已确认，可以执行生命周期动作。
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
	StrategyKind    LifecycleStrategyKind
	ReviewStatus    LifecycleReviewStatus
	WorkingDir      string
	ComposeFiles    []string
	ApplicationName string
	Standard        LifecycleStandardConfig
}

// ActionResult 返回第一阶段有界动作状态。
type ActionResult struct {
	ApplicationRecordID uint64
	ApplicationID       string
	Action              generated.ApplicationActionResponseAction
	Result              generated.ApplicationActionResponseResult
	MessageKey          *string
	Message             *string
	GuardResults        []GuardResult
}

// BatchActionRequest 描述一次项目批量动作执行。
type BatchActionRequest struct {
	Action                    generated.ApplicationBatchActionRequestAction
	ApplicationRecordIDs      []uint64
	RemoveNamedVolumes        bool
	AutoUnregister            bool
	ImagePrune                bool
	DeleteWorkspacePath       bool
	ConfirmComposeProjectName *string
	ActorID                   *uint64
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
	RemoveNamedVolumes        bool
	AutoUnregister            bool
	ImagePrune                bool
	DeleteWorkspacePath       bool
	ConfirmComposeProjectName string
	ActorID                   *uint64
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

// ManagedApplicationCreateRequest 描述受管创建契约载荷。
type ManagedApplicationCreateRequest struct {
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
	TemplateVersionID      string
}

// ManagedWorkspaceEntry 表示任意 UTF-8 文本文件或空/非空目录。
type ManagedWorkspaceEntry struct {
	Path     string  `json:"path"`
	NodeType string  `json:"node_type"`
	Content  *string `json:"content,omitempty"`
}

// ManagedApplicationCreateValidationResult 返回创建契约校验元数据，不写入文件。
type ManagedApplicationCreateValidationResult struct {
	ManagedRoot             ManagedRootInfo
	SourceType              string
	DisplayName             string
	ComposeProjectName      string
	ApplicationName         *string
	OwnershipMode           string
	WorkspacePath           string
	ComposeFileName         string
	EnvFileName             *string
	ComposeFileAbsolutePath string
	EnvFileAbsolutePath     *string
	SourceMetadata          map[string]string
	Warnings                []string
	ReusedExistingWorkspace bool
}

// ManagedApplicationCreateResult 返回文件写入并持久化后的受管项目启动信息。
type ManagedApplicationCreateResult struct {
	Validation           ManagedApplicationCreateValidationResult
	SourceType           string
	ApplicationRecordID  uint64
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
	auditPublisher               event.Publisher
	logger                       *zap.Logger
	appLogger                    logger.AppLogger
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

// SetAppLogger 注入项目业务错误的统一记录边界。
func (s *Service) SetAppLogger(appLogger logger.AppLogger) {
	if s != nil {
		s.appLogger = appLogger
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

// WithRuntimeReader 设置容器运行时聚合读取器，为应用成员汇总提供 Container 权威边界。
func WithRuntimeReader(reader moduleapi.ContainerProjectRuntimeReader) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.runtimeReader = reader
	})
}

// WithResourceReader 设置应用概览使用的 Container 资源读取边界。
func WithResourceReader(reader moduleapi.ContainerProjectResourceReader) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.resourceReader = reader
	})
}

// WithLogReader 设置应用聚合日志使用的 Container 日志读取边界。
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

// WithAuthorizer 注入实时 topic 签发所需的鉴权边界。
func WithAuthorizer(authorizer moduleapi.Authorizer) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.authorizer = authorizer
	})
}

// WithRealtime 注入统一实时 topic 签发依赖。
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

// SetRuntimeReader 在模块注册解析跨模块服务后注入 Container 运行时读取器。
func (s *Service) SetRuntimeReader(reader moduleapi.ContainerProjectRuntimeReader) {
	if s == nil {
		return
	}
	s.runtimeReader = reader
}

// SetResourceReader 在模块注册解析跨模块服务后注入 Container 资源读取器。
func (s *Service) SetResourceReader(reader moduleapi.ContainerProjectResourceReader) {
	if s == nil {
		return
	}
	s.resourceReader = reader
}

// SetLogReader 在模块注册解析跨模块服务后注入 Container 日志读取器。
func (s *Service) SetLogReader(reader moduleapi.ContainerProjectLogReader) {
	if s == nil {
		return
	}
	s.logReader = reader
}

// SetSystemConfigResolver 在模块注册后注入系统配置解析器。
func (s *Service) SetSystemConfigResolver(resolver moduleapi.SystemConfigResolver) {
	if s == nil {
		return
	}
	s.configResolver = resolver
}

// SetSavedViewService 注入通用保存视图持久化边界。
func (s *Service) SetSavedViewService(service moduleapi.SavedViewService) {
	if s != nil {
		s.savedViews = service
	}
}

// SetRuntimeTargetReader 注入窄化的 Runtime Target 身份权威边界。
func (s *Service) SetRuntimeTargetReader(reader moduleapi.ComposeRuntimeTargetReader) {
	if s != nil {
		s.runtimeTargets = reader
	}
}

// WithRuntimeTargetReader 配置服务使用的 Compose Runtime Target 读取器。
func WithRuntimeTargetReader(reader moduleapi.ComposeRuntimeTargetReader) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.runtimeTargets = reader
	})
}

// SetAuthorizer 在模块注册后注入鉴权边界。
func (s *Service) SetAuthorizer(authorizer moduleapi.Authorizer) {
	if s == nil {
		return
	}
	s.authorizer = authorizer
}

// SetRealtime 在模块注册后注入统一实时通信依赖。
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

// SetAuditPublisher 在模块注册后注入审计事件发布依赖。
func (s *Service) SetAuditPublisher(publisher event.Publisher, logger *zap.Logger, moduleName string) {
	if s == nil {
		return
	}
	s.auditPublisher = publisher
	s.logger = logger
	s.moduleName = strings.TrimSpace(moduleName)
}

// List 返回一页存活应用注册记录。
func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ListResult{}, err
	}
	if query.DeploymentAdapterKind != "" && query.DeploymentAdapterKind != projectcontract.DeploymentAdapterKindCompose.String() {
		return ListResult{}, errProjectInvalidArgument
	}
	if query.Provider != "" && query.Provider != "docker" {
		return ListResult{Items: []generated.ApplicationListItem{}, Limit: normalizeListLimit(query.Limit), Offset: maxInt(query.Offset, 0)}, nil
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
		SourceType:      strings.TrimSpace(query.SourceType),
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

// listRuntimeStatusPage 先应用 Runtime Target 权威状态筛选，再执行分页。
func (s *Service) listRuntimeStatusPage(
	ctx context.Context,
	repository projectstore.Repository,
	storeQuery projectstore.ListQuery,
	query ListQuery,
	targetByID map[uint64]moduleapi.ComposeRuntimeTargetSummary,
) (ListResult, error) {
	// 运行时状态是容器 authority 的派生筛选，必须先筛选完整结果再分页，避免页内数量漂移。
	matched := make([]generated.ApplicationListItem, 0)
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
	items []projectstore.ApplicationAggregate,
	targetByID map[uint64]moduleapi.ComposeRuntimeTargetSummary,
	runtimeStatus string,
) []generated.ApplicationListItem {
	managedRootDirectory := s.readyManagedRootDirectory(ctx)
	mappedItems := make([]generated.ApplicationListItem, 0, len(items))
	for _, item := range items {
		runtimeSummary, runtimeErr := s.runtimeSummary(ctx, item)
		mapped := toProjectListItemWithManagedRoot(item, managedRootDirectory, &runtimeSummary, runtimeErr)
		mapped.DeploymentAdapterKind = generated.DeploymentAdapterKindCompose
		if item.Application.RuntimeTargetID != nil {
			if target, ok := targetByID[*item.Application.RuntimeTargetID]; ok {
				mapped.RuntimeTarget = &generated.ApplicationRuntimeTargetSummary{Id: target.ID, DisplayName: target.DisplayName, Provider: generated.ApplicationRuntimeTargetSummaryProvider(target.Provider)}
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

// BackfillRuntimeTargets 在 Runtime Target 启动发现完成后，为历史本地 Compose 应用补齐运行目标关联。
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

// Get 返回单个应用详情载荷。
func (s *Service) Get(ctx context.Context, projectID uint64) (generated.ApplicationDetailResponse, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return generated.ApplicationDetailResponse{}, err
	}
	runtimeSummary, runtimeErr := s.runtimeSummary(ctx, aggregate)
	return toProjectDetailResponseWithManagedRoot(aggregate, s.readyManagedRootDirectory(ctx), &runtimeSummary, runtimeErr), nil
}

// ValidateImport 解析静态 Compose 输入并返回有界的导入校验结果，不写入注册表。
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

// Import 校验并注册一个应用。
func (s *Service) Import(ctx context.Context, request ImportRequest) (generated.ApplicationImportResponse, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return generated.ApplicationImportResponse{}, err
	}
	session, err := s.inspectImportRequest(ctx, repository, request)
	if err != nil {
		return generated.ApplicationImportResponse{}, err
	}
	lifecycleConfig := defaultLifecycleStandardConfig()
	return s.importInspectionSession(ctx, session, importInspectionCommitInput{
		DisplayName:       request.DisplayName,
		CanonicalOverride: request.ComposeProjectNameOverride,
		LifecycleConfig:   &lifecycleConfig,
		ActorID:           request.ActorID,
	})
}

// Refresh 重新解析并持久化应用最新的静态 Compose 快照。
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
		WorkspacePath: aggregate.Application.WorkspacePath,
		DisplayName:   stringPointer(aggregate.Application.DisplayName),
		ComposeFiles:  collectFilesByKind(aggregate.Files, projectcontract.FileKindCompose.String()),
		EnvFiles:      collectFilesByKind(aggregate.Files, projectcontract.FileKindEnv.String()),
		ActorID:       actorID,
	}
	parseResult, _, err := s.parseImportRequest(request)
	if err != nil {
		return ActionResult{}, err
	}
	now := time.Now().UTC()
	updated, err := repository.RefreshApplication(ctx, buildRefreshApplicationInput(projectID, parseResult, now, actorID))
	if err != nil {
		return ActionResult{}, mapStoreError(err)
	}
	_ = updated
	return ActionResult{
		ApplicationRecordID: projectID,
		ApplicationID:       updated.Application.ApplicationID,
		Action:              generated.ApplicationActionResponseActionApplicationActionRefresh,
		Result:              generated.ApplicationActionResponseResultApplicationActionResultCompleted,
		MessageKey:          stringPointer(projectcontract.ApplicationRefreshCompleted.String()),
		Message:             stringPointer(projectcontract.ApplicationRefreshCompleted.String()),
	}, nil
}

// Services 返回应用的静态服务投影，并合并 Container 权威运行时成员。
func (s *Service) Services(ctx context.Context, projectID uint64) (generated.ApplicationServicesResponse, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return generated.ApplicationServicesResponse{}, err
	}
	parseResult, err := s.loadFromAggregate(aggregate)
	if err != nil {
		return generated.ApplicationServicesResponse{}, err
	}
	runtimeSummary, _ := s.runtimeSummary(ctx, aggregate)
	serviceMembers := membersByService(runtimeSummary.Members)
	items := make([]generated.ApplicationServiceItem, 0, len(parseResult.Services))
	for _, item := range parseResult.Services {
		members := serviceMembers[item.ServiceName]
		generatedItem := generated.ApplicationServiceItem{
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
	return generated.ApplicationServicesResponse{
		ComposeProjectName: aggregate.Application.ComposeProjectName,
		Items:              items,
		ApplicationId:      aggregate.Application.ApplicationID,
	}, nil
}

// Overview 返回应用概览；运行时资源数据始终来自 Container 模块聚合。
func (s *Service) Overview(ctx context.Context, projectID uint64) (generated.ApplicationOverviewResponse, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return generated.ApplicationOverviewResponse{}, err
	}
	parseResult, err := s.loadFromAggregate(aggregate)
	if err != nil {
		return generated.ApplicationOverviewResponse{}, err
	}
	resourceSummary, _ := s.projectResourceSummary(ctx, aggregate)
	serviceResources := make(map[string]moduleapi.ContainerProjectServiceResourceSummary, len(resourceSummary.Services))
	for _, item := range resourceSummary.Services {
		serviceResources[item.ServiceName] = item
	}
	items := make([]generated.ApplicationOverviewServiceItem, 0, len(parseResult.Services))
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
	return generated.ApplicationOverviewResponse{
		ApplicationId:      aggregate.Application.ApplicationID,
		ComposeProjectName: aggregate.Application.ComposeProjectName,
		CollectedAt:        optionalRFC3339Time(resourceSummary.CollectedAt),
		Health: generated.ApplicationOverviewHealthSummary{
			HealthyServiceCount:     healthyServiceCount,
			HealthyContainerCount:   resourceSummary.HealthyContainerCount,
			UnhealthyContainerCount: resourceSummary.UnhealthyContainerCount,
			StartingContainerCount:  resourceSummary.StartingContainerCount,
			RestartCount:            resourceSummary.RestartCount,
			NetworksCount:           countDeclaredNetworks(parseResult.Services),
			VolumesCount:            countDeclaredVolumes(parseResult.Services),
		},
		Resources: generated.ApplicationOverviewResourceSummary{
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

// ManagedRoot 返回受管应用创建流程使用的规范根目录权威信息。
func (s *Service) ManagedRoot(ctx context.Context) (ManagedRootInfo, error) {
	definitionKey := projectcontract.ApplicationRootDirectoryConfig.String()
	info := ManagedRootInfo{
		SourceType:            "managed",
		Status:                projectcontract.ManagedRootStatusUnconfigured.String(),
		ConfigKey:             definitionKey,
		OwnershipMode:         projectcontract.OwnershipModeManagedRootDedicated.String(),
		CreatePermission:      projectcontract.ApplicationCreatePermission.String(),
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

// ValidateManagedCreate 校验有界受管创建路径和命名规则，不写入文件。
func (s *Service) ValidateManagedCreate(ctx context.Context, request ManagedApplicationCreateRequest) (ManagedApplicationCreateValidationResult, error) {
	rootInfo, err := s.ManagedRoot(ctx)
	if err != nil {
		return ManagedApplicationCreateValidationResult{}, err
	}
	if err = requireManagedRootReady(rootInfo); err != nil {
		return ManagedApplicationCreateValidationResult{}, err
	}

	normalized, err := normalizeManagedCreateRequest(request)
	if err != nil {
		return ManagedApplicationCreateValidationResult{}, err
	}
	workspace, err := s.resolveManagedCreateWorkspace(
		ctx,
		*rootInfo.ConfiguredRootDirectory,
		normalized.ApplicationName,
		request.ReuseExistingWorkspace,
	)
	if err != nil {
		return ManagedApplicationCreateValidationResult{}, err
	}
	composeName, composeContent, err := ensureComposeProjectName(normalized.ComposeFileContent, *normalized.ApplicationName)
	if err != nil {
		return ManagedApplicationCreateValidationResult{}, err
	}
	if err := s.ensureManagedCreateRuntimeNameAvailable(ctx, normalized.RuntimeTargetID, composeName); err != nil {
		return ManagedApplicationCreateValidationResult{}, err
	}
	normalized.ComposeFileContent = composeContent
	composeFileAbsolutePath := filepath.Join(workspace.workingDirectory, normalized.ComposeFileName)
	envFileAbsolutePath := managedCreateEnvAbsolutePath(workspace.workingDirectory, normalized.EnvFileName)
	warnings := make([]string, 0, managedCreateWarningsCap)
	warnings = append(warnings, "Managed create validation checks authority, normalized names, and target paths before any file-write execution.")
	if normalized.EnvFileName == nil {
		warnings = append(warnings, "No env file is declared; create execution will only materialize the compose file.")
	}

	sourceType, sourceMetadata, err := s.resolveManagedCreateSource(ctx, request.TemplateVersionID, rootInfo.ConfigKey, *workspace.applicationName, normalized.ComposeFileName, normalized.EnvFileName)
	if err != nil {
		return ManagedApplicationCreateValidationResult{}, err
	}

	return ManagedApplicationCreateValidationResult{
		ManagedRoot:             rootInfo,
		SourceType:              sourceType,
		DisplayName:             normalized.DisplayName,
		ComposeProjectName:      composeName,
		ApplicationName:         workspace.applicationName,
		OwnershipMode:           projectcontract.OwnershipModeManagedRootDedicated.String(),
		WorkspacePath:           workspace.workingDirectory,
		ComposeFileName:         normalized.ComposeFileName,
		EnvFileName:             normalized.EnvFileName,
		ComposeFileAbsolutePath: composeFileAbsolutePath,
		EnvFileAbsolutePath:     envFileAbsolutePath,
		SourceMetadata:          sourceMetadata,
		Warnings:                warnings,
		ReusedExistingWorkspace: workspace.exists,
	}, nil
}

// requireManagedRootReady 将受管根目录状态映射为创建流程的稳定错误语义。
func requireManagedRootReady(root ManagedRootInfo) error {
	if root.Status == projectcontract.ManagedRootStatusReady.String() && root.ConfiguredRootDirectory != nil {
		return nil
	}
	if root.Status == projectcontract.ManagedRootStatusInvalid.String() {
		return errProjectManagedRootInvalid
	}
	return errProjectManagedRootUnconfigured
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

func (s *Service) runtimeSummary(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
) (moduleapi.ContainerProjectRuntimeSummary, error) {
	if s == nil || s.runtimeReader == nil {
		return moduleapi.ContainerProjectRuntimeSummary{
			CanonicalProjectName: aggregate.Application.ComposeProjectName,
			Members:              []moduleapi.ContainerProjectMember{},
		}, errProjectRuntimeUnavailable
	}
	return s.runtimeReader.ListProjectMembers(ctx, localContainerRuntimeScope, aggregate.Application.ComposeProjectName)
}

func (s *Service) projectResourceSummary(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
) (moduleapi.ContainerProjectResourceSummary, error) {
	if s == nil || s.resourceReader == nil {
		return moduleapi.ContainerProjectResourceSummary{
			CanonicalProjectName: aggregate.Application.ComposeProjectName,
			Services:             []moduleapi.ContainerProjectServiceResourceSummary{},
		}, errProjectRuntimeUnavailable
	}
	return s.resourceReader.ReadProjectResourceSummary(ctx, localContainerRuntimeScope, aggregate.Application.ComposeProjectName)
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
func applyGeneratedServiceMembers(target *generated.ApplicationServiceItem, items []moduleapi.ContainerProjectMember) {
	if target == nil {
		return
	}
	//nolint:revive // OpenAPI 生成匿名结构字段名固定为 ContainerId，手写映射需保持线格式兼容。
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

// ResolveApplicationID 将唯一公开的 Application HTTP 标识解析为模块表和任务内部使用的私有数值键。
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
	return aggregate.Application.ApplicationRecordID, nil
}

func (s *Service) getAggregate(ctx context.Context, projectID uint64) (projectstore.ApplicationAggregate, error) {
	// 所有详情、刷新和生命周期操作共享聚合读取入口，确保项目、文件与快照使用同一份存储视图。
	repository, err := s.repositoryOrErr()
	if err != nil {
		return projectstore.ApplicationAggregate{}, err
	}
	if projectID == 0 {
		return projectstore.ApplicationAggregate{}, errProjectInvalidArgument
	}
	aggregate, err := repository.Get(ctx, projectID)
	if err != nil {
		return projectstore.ApplicationAggregate{}, mapStoreError(err)
	}
	return aggregate, nil
}

// sameWorkspacePath 按当前本地文件系统语义比较清理后的工作目录路径。
func mapStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, projectstore.ErrInvalidInput):
		return fmt.Errorf("%w: %w", errProjectInvalidArgument, err)
	case errors.Is(err, projectstore.ErrApplicationNotFound):
		return errProjectNotFound
	case errors.Is(err, projectstore.ErrApplicationConflict):
		return errProjectConflict
	case errors.Is(err, projectstore.ErrFileNotFound):
		return errProjectFileNotFound
	default:
		return err
	}
}
