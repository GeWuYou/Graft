package container

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/eventbus"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
	containercontract "graft/server/modules/container/contract"
)

const (
	containerResourceType        = "container"
	containerOperationTTL        = 30 * time.Second
	containerAuditPublishTimeout = 3 * time.Second
	maskedEnvironmentPlaceholder = "*****"
)

type environmentPlainAccessContextKey struct{}

type service struct {
	runtimeMu               sync.Mutex
	runtime                 Runtime
	runtimeOptions          containerRuntimeOptions
	runtimeFactory          func(containerRuntimeOptions) (Runtime, error)
	systemConfig            moduleapi.SystemConfigResolver
	auditBus                eventbus.Bus
	logger                  *zap.Logger
	moduleName              string
	mountUsageCache         *mountUsageCache
	enabled                 bool
	dangerousActionsEnabled bool
	shellEnabled            bool
	defaultTail             int
	maxTail                 int
	environmentPolicy       containercontract.EnvironmentPolicy
	orchestratorPolicies    orchestratorActionPolicies
	websocketAllowedOrigins []string
	realtimeTickets         realtimeauth.Service
	realtimeHub             realtime.Hub
	topicIssuers            realtime.TopicIssuerRegistry
	authorizer              moduleapi.Authorizer
	runtimeTargets          moduleapi.RuntimeTargetReader
	buildTargets            moduleapi.BuildRuntimeTargetReader
	tasks                   moduleapi.TaskService
	statsCollector          *statsCollector
	runtimeEventManagerMu   sync.RWMutex
	runtimeEventManager     *runtimeEventManager
	logTopicStreamerMu      sync.Mutex
	logTopicStreamer        *logTopicStreamer
	logTopicStreamerFactory func(realtime.Hub, *zap.Logger, func() (Runtime, error)) (*logTopicStreamer, error)
	lifecycleContext        context.Context
}

type containerServiceOptions struct {
	runtime                              Runtime
	runtimeOptions                       containerRuntimeOptions
	runtimeFactory                       func(containerRuntimeOptions) (Runtime, error)
	systemConfig                         moduleapi.SystemConfigResolver
	auditBus                             eventbus.Bus
	logger                               *zap.Logger
	moduleName                           string
	mountUsageCache                      *mountUsageCache
	enabled                              bool
	dangerousActionsEnabled              bool
	shellEnabled                         bool
	defaultTail                          int
	maxTail                              int
	resourceStatsCacheTTLSeconds         int
	resourceStatsCacheStaleWindowSeconds int
	environmentPolicy                    containercontract.EnvironmentPolicy
	orchestratorPolicies                 orchestratorActionPolicies
	websocketAllowedOrigins              []string
	realtimeTickets                      realtimeauth.Service
	realtimeHub                          realtime.Hub
	topicIssuers                         realtime.TopicIssuerRegistry
	authorizer                           moduleapi.Authorizer
	runtimeTargets                       moduleapi.RuntimeTargetReader
	buildTargets                         moduleapi.BuildRuntimeTargetReader
	tasks                                moduleapi.TaskService
	logTopicStreamerFactory              func(realtime.Hub, *zap.Logger, func() (Runtime, error)) (*logTopicStreamer, error)
}

// newContainerService 根据模块上下文解析配置、运行时、实时订阅和鉴权依赖；运行时目标读取器属于可选能力，解析失败不阻止服务创建。
func newContainerService(ctx *module.Context, moduleName string) (*service, error) {
	options := containerOptionsFromConfig(ctx)
	systemConfig, err := resolveSystemConfigResolver(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve container system config: %w", err)
	}
	runtime := Runtime(disabledRuntime{})
	allowedOrigins := []string{}
	if ctx != nil && ctx.Config != nil {
		allowedOrigins = append(allowedOrigins, ctx.Config.HTTPX.WebSocketAllowedOrigins...)
	}
	realtimeTickets, err := resolveRealtimeTicketService(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve container realtime ticket service: %w", err)
	}
	realtimeHub, err := resolveRealtimeHub(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve container realtime hub: %w", err)
	}
	topicIssuers, err := resolveRealtimeTopicIssuerRegistry(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve container topic issuer registry: %w", err)
	}
	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve container authorizer: %w", err)
	}
	runtimeTargets, _ := module.ResolveService[moduleapi.RuntimeTargetReader](ctx.Services, (*moduleapi.RuntimeTargetReader)(nil))
	buildTargets, _ := module.ResolveService[moduleapi.BuildRuntimeTargetReader](ctx.Services, (*moduleapi.BuildRuntimeTargetReader)(nil))
	tasks, err := module.ResolveService[moduleapi.TaskService](ctx.Services, (*moduleapi.TaskService)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve task service: %w", err)
	}
	return newService(containerServiceOptions{
		runtime:                 runtime,
		runtimeOptions:          options,
		systemConfig:            systemConfig,
		auditBus:                ctx.EventBus,
		logger:                  ctx.Logger,
		moduleName:              moduleName,
		enabled:                 options.enabled,
		dangerousActionsEnabled: options.dangerousActionsEnabled,
		shellEnabled:            defaultContainerShellEnabled,
		defaultTail:             options.defaultTail,
		maxTail:                 options.maxTail,
		environmentPolicy:       options.environmentPolicy,
		orchestratorPolicies:    options.orchestratorPolicies,
		websocketAllowedOrigins: allowedOrigins,
		realtimeTickets:         realtimeTickets,
		realtimeHub:             realtimeHub,
		topicIssuers:            topicIssuers,
		authorizer:              authorizer,
		runtimeTargets:          runtimeTargets,
		buildTargets:            buildTargets,
		tasks:                   tasks,
	})
}

// newService 创建容器服务并应用默认值与归一化配置；实时票据服务是实时订阅的必需依赖，缺失时直接返回错误。
func newService(options containerServiceOptions) (*service, error) {
	options.defaultTail, options.maxTail = normalizeContainerLogTailBounds(options.defaultTail, options.maxTail)
	if options.realtimeTickets == nil {
		return nil, errors.New("realtime ticket service is required")
	}
	runtimeOptions := options.runtimeOptions
	if strings.TrimSpace(runtimeOptions.runtime) == "" {
		runtimeOptions.runtime = defaultContainerRuntime
	}
	if strings.TrimSpace(runtimeOptions.endpoint) == "" {
		runtimeOptions.endpoint = defaultContainerDockerEndpoint
	}
	runtimeOptions.dangerousActionsEnabled = options.dangerousActionsEnabled
	runtimeOptions.defaultTail = options.defaultTail
	runtimeOptions.maxTail = options.maxTail
	runtimeOptions.resourceStatsCacheTTLSeconds = options.resourceStatsCacheTTLSeconds
	runtimeOptions.resourceStatsCacheStaleWindowSeconds = options.resourceStatsCacheStaleWindowSeconds
	runtimeOptions.logger = options.logger
	environmentPolicy := normalizeEnvironmentPolicy(options.environmentPolicy.String())
	runtimeFactory := options.runtimeFactory
	if runtimeFactory == nil {
		runtimeFactory = newContainerRuntime
	}
	mountUsageCache := options.mountUsageCache
	if mountUsageCache == nil {
		mountUsageCache = newMountUsageCache(containerMountUsageCacheTTL)
	}
	runtime := options.runtime
	if _, ok := runtime.(*DockerRuntime); ok {
		runtime = newRuntimeLease(runtime)
	}
	return &service{
		runtime:                 runtime,
		runtimeOptions:          runtimeOptions,
		runtimeFactory:          runtimeFactory,
		auditBus:                options.auditBus,
		logger:                  options.logger,
		moduleName:              firstNonEmpty(options.moduleName, moduleID),
		mountUsageCache:         mountUsageCache,
		enabled:                 options.enabled,
		systemConfig:            options.systemConfig,
		dangerousActionsEnabled: options.dangerousActionsEnabled,
		shellEnabled:            options.shellEnabled,
		defaultTail:             options.defaultTail,
		maxTail:                 options.maxTail,
		environmentPolicy:       environmentPolicy,
		orchestratorPolicies:    options.orchestratorPolicies.normalized(),
		websocketAllowedOrigins: append([]string(nil), options.websocketAllowedOrigins...),
		realtimeTickets:         options.realtimeTickets,
		realtimeHub:             options.realtimeHub,
		topicIssuers:            options.topicIssuers,
		authorizer:              options.authorizer,
		runtimeTargets:          options.runtimeTargets,
		buildTargets:            options.buildTargets,
		tasks:                   options.tasks,
		logTopicStreamerFactory: options.logTopicStreamerFactory,
	}, nil
}

// resolveRealtimeTicketService 从模块上下文解析实时认证服务；上下文或服务注册器不可用时返回错误。
func resolveRealtimeTicketService(ctx *module.Context) (realtimeauth.Service, error) {
	if ctx == nil || ctx.Services == nil {
		return nil, errors.New("realtime ticket service resolver is unavailable")
	}

	return module.ResolveService[realtimeauth.Service](ctx.Services, (*realtimeauth.Service)(nil))
}

// resolveRealtimeHub 优先使用模块上下文中的实时总线，否则从服务容器解析；两处都不可用时返回错误。
func resolveRealtimeHub(ctx *module.Context) (realtime.Hub, error) {
	if ctx != nil && ctx.Realtime != nil {
		return ctx.Realtime, nil
	}
	if ctx == nil || ctx.Services == nil {
		return nil, errors.New("realtime hub resolver is unavailable")
	}

	return module.ResolveService[realtime.Hub](ctx.Services, (*realtime.Hub)(nil))
}

// resolveRealtimeTopicIssuerRegistry 从模块服务容器解析主题签发器注册表；上下文或服务注册器不可用时返回错误。
func resolveRealtimeTopicIssuerRegistry(ctx *module.Context) (realtime.TopicIssuerRegistry, error) {
	if ctx == nil || ctx.Services == nil {
		return nil, errors.New("realtime topic issuer registry resolver is unavailable")
	}

	return module.ResolveService[realtime.TopicIssuerRegistry](ctx.Services, (*realtime.TopicIssuerRegistry)(nil))
}

func (s *service) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("container service close context is required")
	}
	var closeErr error
	if err := s.closeLogTopicStreamer(ctx); err != nil {
		closeErr = errors.Join(closeErr, err)
	}
	if err := s.closeStatsCollector(ctx); err != nil {
		closeErr = errors.Join(closeErr, err)
	}
	if err := s.closeRuntimeEventManager(ctx); err != nil {
		closeErr = errors.Join(closeErr, err)
	}
	if closeErr != nil {
		return closeErr
	}
	if err := s.closeRuntime(); err != nil {
		closeErr = errors.Join(closeErr, err)
	}
	return closeErr
}

func (s *service) closeLogTopicStreamer(ctx context.Context) error {
	s.logTopicStreamerMu.Lock()
	logTopicStreamer := s.logTopicStreamer
	s.logTopicStreamerMu.Unlock()
	if logTopicStreamer != nil {
		err := logTopicStreamer.Close(ctx)
		if err == nil {
			s.logTopicStreamerMu.Lock()
			if s.logTopicStreamer == logTopicStreamer {
				s.logTopicStreamer = nil
			}
			s.logTopicStreamerMu.Unlock()
		}
		return err
	}
	return nil
}

func (s *service) closeStatsCollector(ctx context.Context) error {
	if s.statsCollector != nil {
		err := s.statsCollector.Stop(ctx)
		if err == nil {
			s.statsCollector = nil
		}
		return err
	}
	return nil
}

func (s *service) closeRuntimeEventManager(ctx context.Context) error {
	s.runtimeEventManagerMu.Lock()
	runtimeEventManager := s.runtimeEventManager
	s.runtimeEventManagerMu.Unlock()
	if runtimeEventManager != nil {
		err := runtimeEventManager.Stop(ctx)
		if err == nil {
			s.runtimeEventManagerMu.Lock()
			if s.runtimeEventManager == runtimeEventManager {
				s.runtimeEventManager = nil
			}
			s.runtimeEventManagerMu.Unlock()
		}
		return err
	}
	return nil
}

func (s *service) closeRuntime() error {
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()
	runtime := s.runtime
	if runtime == nil {
		return nil
	}
	s.runtime = nil
	return runtime.Close()
}

func (s *service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return ListResult{}, err
	}
	normalized, err := normalizeListQuery(query)
	if err != nil {
		return ListResult{}, err
	}
	target := moduleapi.RuntimeTargetSummary{ID: 1, DisplayName: "Local Docker", Provider: runtimeNameDocker}
	if s.runtimeTargets != nil {
		target, err = s.runtimeTargets.ReadDockerTarget(ctx, normalized.RuntimeTargetID)
		if err != nil {
			return ListResult{}, err
		}
	} else if normalized.RuntimeTargetID != nil {
		return ListResult{}, errInvalidListQuery
	}
	runtime, err := s.runtimeForRequestContext(ctx)
	if err != nil {
		return ListResult{}, err
	}
	info, err := runtime.Info(ctx)
	if err != nil {
		return ListResult{}, err
	}
	items, err := runtime.List(ctx, normalized)
	if err != nil {
		return ListResult{}, err
	}
	filtered := filterContainerSummaries(items, normalized)
	paged := pageContainerSummaries(filtered, normalized)
	paged = applyActionAvailability(paged, s.effectiveActionPolicy(ctx))
	return ListResult{
		Runtime:       info,
		RuntimeTarget: target,
		Items:         paged,
		Total:         len(filtered),
		Limit:         normalized.Limit,
		Offset:        normalized.Offset,
		Summary:       summarizeContainers(filtered),
	}, nil
}

func (s *service) DashboardSummary(ctx context.Context, _ dashboardSummaryQuery) (dashboardSummaryResult, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return dashboardSummaryResult{}, err
	}
	runtime, err := s.runtimeForRequestContext(ctx)
	if err != nil {
		return dashboardSummaryResult{}, err
	}
	items, err := runtime.List(ctx, ListQuery{})
	if err != nil {
		return dashboardSummaryResult{}, err
	}
	items = applyActionAvailability(items, s.effectiveActionPolicy(ctx))
	return buildContainerDashboardSummary(items), nil
}

func (s *service) DockerSystem(ctx context.Context) (RuntimeInfo, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return RuntimeInfo{}, err
	}
	runtime, err := s.runtimeForRequestContext(ctx)
	if err != nil {
		return RuntimeInfo{}, err
	}
	return runtime.Info(ctx)
}

func (s *service) dockerResources(ctx context.Context) (DockerResourceReader, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return nil, err
	}
	runtime, err := s.runtimeForRequestContext(ctx)
	if err != nil {
		return nil, err
	}
	reader, ok := runtime.(DockerResourceReader)
	if !ok {
		return nil, errUnsupportedContainerRuntime
	}
	return reader, nil
}

// DockerImages 从一次 runtime 快照返回过滤后的镜像分页和完整 inventory 摘要。
func (s *service) DockerImages(ctx context.Context, query DockerImageListQuery) (DockerImageListResult, error) {
	reader, err := s.dockerResources(ctx)
	if err != nil {
		return DockerImageListResult{}, err
	}
	normalized, err := normalizeDockerImageListQuery(query)
	if err != nil {
		return DockerImageListResult{}, err
	}
	snapshot, err := reader.ListDockerImages(ctx)
	if err != nil {
		return DockerImageListResult{}, err
	}
	filtered := filterDockerImages(snapshot.Items, normalized.Keyword)
	if normalized.Unused {
		filtered = filterUnusedDockerImages(filtered)
	}
	return DockerImageListResult{
		Items:   pageDockerImages(filtered, normalized.Offset, normalized.Limit),
		Total:   len(filtered),
		Summary: snapshot.Summary,
	}, nil
}

func (s *service) DockerImage(ctx context.Context, id string) (DockerImage, error) {
	reader, err := s.dockerResources(ctx)
	if err != nil {
		return DockerImage{}, fmt.Errorf("load Docker image resources: %w", err)
	}
	image, err := reader.ReadDockerImage(ctx, id)
	if err != nil {
		return DockerImage{}, fmt.Errorf("read Docker image %q: %w", id, err)
	}
	return image, nil
}

func dockerImageHasRepositoryTag(image DockerImage, reference string) bool {
	for _, tag := range image.RepositoryTags {
		if strings.TrimSpace(tag) == reference {
			return true
		}
	}
	return false
}

func (s *service) DockerNetworks(ctx context.Context) ([]DockerNetwork, error) {
	reader, err := s.dockerResources(ctx)
	if err != nil {
		return nil, err
	}
	return reader.ListDockerNetworks(ctx)
}

// DockerNetworksPage 返回按查询条件筛选并分页的 Docker 网络列表。
func (s *service) DockerNetworksPage(ctx context.Context, query DockerNetworkListQuery) (DockerNetworkListResult, error) {
	reader, err := s.dockerResources(ctx)
	if err != nil {
		return DockerNetworkListResult{}, fmt.Errorf("resolve docker resources: %w", err)
	}
	items, err := reader.ListDockerNetworks(ctx)
	if err != nil {
		return DockerNetworkListResult{}, fmt.Errorf("list docker networks: %w", err)
	}
	return listDockerNetworks(items, query), nil
}

func (s *service) DockerNetwork(ctx context.Context, id string) (DockerNetwork, error) {
	reader, err := s.dockerResources(ctx)
	if err != nil {
		return DockerNetwork{}, err
	}
	return reader.ReadDockerNetwork(ctx, id)
}

func (s *service) DockerVolumes(ctx context.Context, query DockerVolumeListQuery) (DockerVolumeListResult, error) {
	reader, err := s.dockerResources(ctx)
	if err != nil {
		return DockerVolumeListResult{}, err
	}
	items, err := reader.ListDockerVolumes(ctx)
	if err != nil {
		return DockerVolumeListResult{}, err
	}
	return listDockerVolumes(items, query), nil
}

func (s *service) DockerVolume(ctx context.Context, id string) (DockerVolume, error) {
	reader, err := s.dockerResources(ctx)
	if err != nil {
		return DockerVolume{}, err
	}
	return reader.ReadDockerVolume(ctx, id)
}

func (s *service) Detail(ctx context.Context, ref Ref) (Detail, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return Detail{}, err
	}
	runtime, err := s.runtimeForRequestContext(ctx)
	if err != nil {
		return Detail{}, err
	}
	detail, err := runtime.Detail(ctx, ref)
	if err != nil {
		return Detail{}, err
	}
	adjusted := applyActionAvailability([]Summary{detail.Summary}, s.effectiveActionPolicy(ctx))
	if len(adjusted) == 1 {
		detail.Summary = adjusted[0]
	}
	detail = s.applyEnvironmentPolicy(ctx, detail)
	detail = s.attachCachedMountUsage(ref, detail)
	return detail, nil
}

func (s *service) attachCachedMountUsage(ref Ref, detail Detail) Detail {
	if s == nil || s.mountUsageCache == nil {
		return detail
	}
	for index := range detail.Mounts {
		mount := &detail.Mounts[index]
		if strings.TrimSpace(mount.ID) == "" {
			mount.ID = stableMountID(*mount)
		}
		if usage, ok := s.mountUsageCache.get(mountUsageCacheKey(ref, mount.ID)); ok {
			mount.Usage = &usage
		}
	}
	return detail
}

func (s *service) MountUsageList(ctx context.Context, ref Ref) ([]MountUsage, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return nil, err
	}
	runtime, err := s.runtimeForRequestContext(ctx)
	if err != nil {
		return nil, err
	}
	mounts, err := runtime.Mounts(ctx, ref)
	if err != nil {
		return nil, err
	}
	items := make([]MountUsage, 0, len(mounts))
	for _, mount := range mounts {
		if strings.TrimSpace(mount.ID) == "" {
			mount.ID = stableMountID(mount)
		}
		cacheKey := mountUsageCacheKey(ref, mount.ID)
		if usage, ok := s.mountUsageCache.get(cacheKey); ok {
			usage.ContainerID = ref.Value
			items = append(items, usage)
			continue
		}
		status := containerMountUsageStatusNotMeasured
		if !mountUsageSupported(mount) {
			status = containerMountUsageStatusUnsupported
		}
		items = append(items, mountUsageFromMount(ref.Value, mount, status, 0, ""))
	}
	return items, nil
}

func (s *service) RefreshMountUsage(ctx context.Context, ref Ref, mountID string) (MountUsage, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return MountUsage{}, err
	}
	mountID = strings.TrimSpace(mountID)
	if !isValidMountID(mountID) {
		return MountUsage{}, errInvalidRef
	}
	cacheKey := mountUsageCacheKey(ref, mountID)
	runtime, err := s.runtimeForRequestContext(ctx)
	if err != nil {
		return MountUsage{}, err
	}
	usageCtx, cancel := context.WithTimeout(ctx, containerMountUsageTimeout)
	defer cancel()
	usage, err := runtime.MountUsage(usageCtx, ref, mountID)
	if err != nil {
		return MountUsage{}, err
	}
	if usage.Status == containerMountUsageStatusMeasured {
		s.mountUsageCache.set(cacheKey, usage)
	}
	return usage, nil
}

func (s *service) Logs(ctx context.Context, ref Ref, query LogQuery) (Logs, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return Logs{}, err
	}
	normalized, err := s.normalizeLogQuery(ctx, query)
	if err != nil {
		return Logs{}, err
	}
	runtime, err := s.runtimeForRequestContext(ctx)
	if err != nil {
		return Logs{}, err
	}
	return runtime.Logs(ctx, ref, normalized)
}

// BatchLifecycleActionResult 表示容器生命周期批量提交的有序聚合结果；它只记录 Task 接收事实，不代表外部动作已完成。
type BatchLifecycleActionResult struct {
	Action        string
	Total         int
	AcceptedCount int
	FailedCount   int
	RequestID     string
	Items         []BatchLifecycleActionItem
}

// BatchLifecycleActionItem 保留单个容器的 Task receipt 或可展示的提交失败，避免批量部分失败被整体错误遮蔽。
type BatchLifecycleActionItem struct {
	ID         string
	Action     string
	Accepted   bool
	TaskID     uint64
	Status     moduleapi.TaskStatus
	ErrorCode  string
	MessageKey string
	Message    string
}

// BatchLifecycleAction 为一次批量请求提交一个含多个顺序 Stage 的 Task，并把预校验失败保留在对应 item 中。
func (s *service) BatchLifecycleAction(ctx context.Context, command BatchActionCommand, requestedBy uint64, idempotencyKey string) (BatchLifecycleActionResult, error) {
	normalized, err := normalizeBatchActionCommand(command)
	if err != nil || !isContainerLifecycleTaskAction(normalized.Action) {
		return BatchLifecycleActionResult{}, errInvalidBatchAction
	}
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return BatchLifecycleActionResult{}, err
	}
	if !s.dangerousActionsAllowed(ctx) {
		return BatchLifecycleActionResult{}, errDangerousActionsDisabled
	}
	result := BatchLifecycleActionResult{
		Action:    normalized.Action,
		Total:     len(normalized.IDs),
		RequestID: requestIDFromContext(ctx),
		Items:     make([]BatchLifecycleActionItem, 0, len(normalized.IDs)),
	}
	acceptedRefs := s.collectBatchLifecycleCandidates(ctx, normalized, &result)
	if len(acceptedRefs) == 0 {
		return result, nil
	}
	receipt, submitErr := s.SubmitContainerLifecycleBatchAction(ctx, acceptedRefs, normalized.Action, ActionOptions{Force: normalized.Force}, requestedBy, idempotencyKey)
	s.appendBatchLifecycleSubmissionResults(ctx, &result, acceptedRefs, ActionOptions{Force: normalized.Force}, receipt, submitErr)
	return result, nil
}

func (s *service) collectBatchLifecycleCandidates(ctx context.Context, command BatchActionCommand, result *BatchLifecycleActionResult) []Ref {
	acceptedRefs := make([]Ref, 0, len(command.IDs))
	options := ActionOptions{Force: command.Force}
	for _, rawID := range command.IDs {
		ref, err := parseRef(rawID)
		if err != nil {
			result.Items = append(result.Items, batchLifecycleActionFailure(rawID, command.Action, err))
			result.FailedCount++
			s.publishLifecycleTaskSubmissionAudit(ctx, Ref{Value: rawID}, command.Action, options, moduleapi.TaskReceipt{}, err)
			continue
		}
		if blockedItem, blocked := s.lifecycleActionPolicyFailure(ctx, ref, command.Action, options); blocked {
			result.Items = append(result.Items, blockedItem)
			result.FailedCount++
			continue
		}
		acceptedRefs = append(acceptedRefs, ref)
	}
	return acceptedRefs
}

func (s *service) appendBatchLifecycleSubmissionResults(ctx context.Context, result *BatchLifecycleActionResult, refs []Ref, options ActionOptions, receipt moduleapi.TaskReceipt, submitErr error) {
	for _, ref := range refs {
		s.publishLifecycleTaskSubmissionAudit(ctx, ref, result.Action, options, receipt, submitErr)
		if submitErr != nil {
			result.Items = append(result.Items, batchLifecycleActionFailure(ref.Value, result.Action, submitErr))
			result.FailedCount++
			continue
		}
		result.Items = append(result.Items, BatchLifecycleActionItem{ID: ref.Value, Action: result.Action, Accepted: true, TaskID: receipt.TaskID, Status: receipt.Status})
		result.AcceptedCount++
	}
}

func batchLifecycleActionFailure(id string, action string, err error) BatchLifecycleActionItem {
	messageKey := messageKeyForError(err).String()
	return BatchLifecycleActionItem{ID: id, Action: action, ErrorCode: messageKey, MessageKey: messageKey, Message: fallbackMessageForError(err)}
}
