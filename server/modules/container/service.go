package container

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"

	"graft/server/internal/eventbus"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
	containercontract "graft/server/modules/container/contract"
)

const (
	containerResourceType        = "container"
	containerBatchResourceType   = "container_batch"
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
	tasks                   moduleapi.TaskService
	statsCollector          *statsCollector
	runtimeEventManagerMu   sync.RWMutex
	runtimeEventManager     *runtimeEventManager
	logTopicStreamerMu      sync.Mutex
	logTopicStreamer        *logTopicStreamer
	logTopicStreamerFactory func(realtime.Hub, *zap.Logger, func() (Runtime, error)) (*logTopicStreamer, error)
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
	if err := s.closeRuntime(); err != nil {
		closeErr = errors.Join(closeErr, err)
	}
	return closeErr
}

func (s *service) closeLogTopicStreamer(ctx context.Context) error {
	s.logTopicStreamerMu.Lock()
	logTopicStreamer := s.logTopicStreamer
	s.logTopicStreamer = nil
	s.logTopicStreamerMu.Unlock()
	if logTopicStreamer != nil {
		return logTopicStreamer.Close(ctx)
	}
	return nil
}

func (s *service) closeStatsCollector(ctx context.Context) error {
	if s.statsCollector != nil {
		err := s.statsCollector.Stop(ctx)
		s.statsCollector = nil
		return err
	}
	return nil
}

func (s *service) closeRuntimeEventManager(ctx context.Context) error {
	s.runtimeEventManagerMu.Lock()
	runtimeEventManager := s.runtimeEventManager
	s.runtimeEventManager = nil
	s.runtimeEventManagerMu.Unlock()
	if runtimeEventManager != nil {
		return runtimeEventManager.Stop(ctx)
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

// DockerImageBatchRemoveResult 汇总镜像批量删除结果，并保留请求顺序和逐项失败原因。
type DockerImageBatchRemoveResult struct {
	Total        int
	SuccessCount int
	FailedCount  int
	RequestID    string
	Items        []DockerImageBatchRemoveItem
}

// DockerImageBatchRemoveItem 表示一个镜像删除请求的脱敏结果。
type DockerImageBatchRemoveItem struct {
	ID         string
	Success    bool
	ErrorCode  string
	MessageKey string
	Message    string
}

// DockerVolumeBatchRemoveResult 汇总数据卷批量删除结果，并保留请求顺序和逐项失败原因。
type DockerVolumeBatchRemoveResult struct {
	Total, SuccessCount, FailedCount int
	RequestID                        string
	Items                            []DockerVolumeBatchRemoveItem
}

// DockerVolumeBatchRemoveItem 表示一个数据卷删除请求的脱敏结果。
type DockerVolumeBatchRemoveItem struct {
	Name                           string
	Success                        bool
	ErrorCode, MessageKey, Message string
}

// DockerVolumeBatchRemove 按请求顺序逐项删除数据卷，允许运行时返回部分成功。
func (s *service) DockerVolumeBatchRemove(ctx context.Context, names []string, force bool) (DockerVolumeBatchRemoveResult, error) {
	if len(names) == 0 || len(names) > maxDockerVolumeBatchRemoveIDs {
		return DockerVolumeBatchRemoveResult{}, errInvalidListQuery
	}
	normalizedNames := make([]string, 0, len(names))
	seenNames := make(map[string]struct{}, len(names))
	for _, rawName := range names {
		name := strings.TrimSpace(rawName)
		if name == "" {
			return DockerVolumeBatchRemoveResult{}, errInvalidListQuery
		}
		if _, exists := seenNames[name]; exists {
			return DockerVolumeBatchRemoveResult{}, errInvalidListQuery
		}
		seenNames[name] = struct{}{}
		normalizedNames = append(normalizedNames, name)
	}
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return DockerVolumeBatchRemoveResult{}, err
	}
	if !s.dangerousActionsAllowed(ctx) {
		return DockerVolumeBatchRemoveResult{}, errDangerousActionsDisabled
	}
	result := DockerVolumeBatchRemoveResult{Total: len(normalizedNames), RequestID: requestIDFromContext(ctx), Items: make([]DockerVolumeBatchRemoveItem, 0, len(normalizedNames))}
	for _, name := range normalizedNames {
		item := DockerVolumeBatchRemoveItem{Name: name}
		err := s.RemoveDockerVolume(ctx, name, force)
		if err != nil {
			item.ErrorCode = messageKeyForError(err).String()
			item.MessageKey, item.Message = item.ErrorCode, fallbackMessageForError(err)
			result.FailedCount++
		} else {
			item.Success = true
			result.SuccessCount++
		}
		result.Items = append(result.Items, item)
	}
	s.publishDockerVolumeBatchAudit(ctx, result, force)
	return result, nil
}

// DockerImageBatchRemove 按请求顺序逐项删除镜像，允许 Docker daemon 返回部分成功。
func (s *service) DockerImageBatchRemove(ctx context.Context, ids []string, force bool) (DockerImageBatchRemoveResult, error) {
	if len(ids) == 0 || len(ids) > maxContainerBatchActionIDs {
		return DockerImageBatchRemoveResult{}, errInvalidListQuery
	}
	if err := s.requireRuntimeAccess(ctx); err != nil {
		result := dockerImageBatchRemoveRejectedResult(ctx, ids, err)
		s.publishDockerImageBatchAuditWithStatus(ctx, result, force, statusForError(err))
		return DockerImageBatchRemoveResult{}, err
	}
	if !s.dangerousActionsAllowed(ctx) {
		result := dockerImageBatchRemoveRejectedResult(ctx, ids, errDangerousActionsDisabled)
		s.publishDockerImageBatchAuditWithStatus(ctx, result, force, statusForError(errDangerousActionsDisabled))
		return DockerImageBatchRemoveResult{}, errDangerousActionsDisabled
	}
	result := DockerImageBatchRemoveResult{Total: len(ids), RequestID: requestIDFromContext(ctx), Items: make([]DockerImageBatchRemoveItem, 0, len(ids))}
	for _, rawID := range ids {
		id := strings.TrimSpace(rawID)
		item := DockerImageBatchRemoveItem{ID: id}
		if err := validateDockerImageReference(id); err != nil {
			item = dockerImageBatchRemoveFailure(item, err)
		} else if _, err := s.RemoveDockerImage(ctx, id, force); err != nil {
			logsafe.Error(s.logger, "docker image batch removal failed", zap.String("image_id", id), zap.Error(err))
			item = dockerImageBatchRemoveFailure(item, err)
		} else {
			item.Success = true
		}
		if item.Success {
			result.SuccessCount++
		} else {
			result.FailedCount++
		}
		result.Items = append(result.Items, item)
	}
	s.publishDockerImageBatchAudit(ctx, result, force)
	return result, nil
}

func dockerImageBatchRemoveFailure(item DockerImageBatchRemoveItem, err error) DockerImageBatchRemoveItem {
	key := messageKeyForError(err).String()
	item.ErrorCode, item.MessageKey, item.Message = dockerImageRemoveErrorCodeFor(err).String(), key, key
	return item
}

func dockerImageRemoveErrorCodeFor(err error) containercontract.DockerImageRemoveErrorCode {
	switch {
	case errors.Is(err, errDockerImageMultipleTags):
		return containercontract.DockerImageMultipleTagsError
	case errors.Is(err, errDockerImageInUse):
		return containercontract.DockerImageInUseError
	case errors.Is(err, errDockerImageNotFound):
		return containercontract.DockerImageNotFoundError
	case errors.Is(err, errDockerImageRuntimeUnavailable), errors.Is(err, errRuntimeDaemonUnavailable), errors.Is(err, errRuntimeSocketMissing), errors.Is(err, errUnsupportedContainerRuntime):
		return containercontract.DockerRuntimeUnavailable
	case errors.Is(err, errDockerImageTimeout), errors.Is(err, errContainerRuntimeTimeout):
		return containercontract.DockerTimeout
	case errors.Is(err, errDockerImageCommunication):
		return containercontract.DockerCommunicationError
	default:
		return containercontract.DockerImageRemoveUnknown
	}
}

func dockerImageBatchRemoveRejectedResult(ctx context.Context, ids []string, err error) DockerImageBatchRemoveResult {
	items := make([]DockerImageBatchRemoveItem, 0, len(ids))
	for _, rawID := range ids {
		item := dockerImageBatchRemoveFailure(DockerImageBatchRemoveItem{ID: strings.TrimSpace(rawID)}, err)
		items = append(items, item)
	}
	return DockerImageBatchRemoveResult{
		Total:       len(items),
		FailedCount: len(items),
		RequestID:   requestIDFromContext(ctx),
		Items:       items,
	}
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

func (s *service) dockerImageWriter(ctx context.Context) (DockerImageWriter, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return nil, fmt.Errorf("require Docker image writer runtime access: %w", err)
	}
	runtime, err := s.runtimeForRequestContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve Docker image writer runtime: %w", err)
	}
	writer, ok := runtime.(DockerImageWriter)
	if !ok {
		return nil, errUnsupportedContainerRuntime
	}
	return writer, nil
}

func (s *service) PullDockerImage(ctx context.Context, reference string, emit func(DockerImagePullEvent) error) error {
	writer, err := s.dockerImageWriter(ctx)
	if err != nil {
		return err
	}
	return writer.PullDockerImage(ctx, reference, emit)
}

func (s *service) TagDockerImage(ctx context.Context, id, target string) (DockerImageActionResult, error) {
	writer, err := s.dockerImageWriter(ctx)
	if err != nil {
		return DockerImageActionResult{ID: id, Action: "tag"}, err
	}
	if err := writer.TagDockerImage(ctx, id, target); err != nil {
		return DockerImageActionResult{ID: id, Action: "tag"}, err
	}
	return DockerImageActionResult{ID: id, Action: "tag", MessageKey: containercontract.DockerImageTagCompleted.String()}, nil
}

// UntagDockerImage 从镜像移除经归属校验的 Repository:Tag 引用，不会强制 daemon 清理镜像。
func (s *service) UntagDockerImage(ctx context.Context, id, reference string) (DockerImageActionResult, error) {
	if !s.dangerousActionsAllowed(ctx) {
		return DockerImageActionResult{ID: id, Action: "untag"}, errDangerousActionsDisabled
	}
	if err := validateDockerImageReference(reference); err != nil {
		return DockerImageActionResult{ID: id, Action: "untag"}, err
	}
	image, err := s.DockerImage(ctx, id)
	if err != nil {
		return DockerImageActionResult{ID: id, Action: "untag"}, err
	}
	if !dockerImageHasRepositoryTag(image, reference) {
		return DockerImageActionResult{ID: id, Action: "untag"}, errDockerImageTagNotAssociated
	}
	writer, err := s.dockerImageWriter(ctx)
	if err != nil {
		return DockerImageActionResult{ID: id, Action: "untag"}, err
	}
	if err := writer.UntagDockerImage(ctx, reference); err != nil {
		return DockerImageActionResult{ID: id, Action: "untag"}, fmt.Errorf("untag Docker image %q: %w", reference, err)
	}
	return DockerImageActionResult{ID: id, Action: "untag", MessageKey: containercontract.DockerImageUntagCompleted.String()}, nil
}

func dockerImageHasRepositoryTag(image DockerImage, reference string) bool {
	for _, tag := range image.RepositoryTags {
		if strings.TrimSpace(tag) == reference {
			return true
		}
	}
	return false
}

func (s *service) RemoveDockerImage(ctx context.Context, id string, force bool) (DockerImageActionResult, error) {
	if !s.dangerousActionsAllowed(ctx) {
		return DockerImageActionResult{ID: id, Action: "remove"}, errDangerousActionsDisabled
	}
	writer, err := s.dockerImageWriter(ctx)
	if err != nil {
		return DockerImageActionResult{ID: id, Action: "remove"}, err
	}
	if err := writer.RemoveDockerImage(ctx, id, force); err != nil {
		return DockerImageActionResult{ID: id, Action: "remove"}, err
	}
	return DockerImageActionResult{ID: id, Action: "remove", MessageKey: containercontract.DockerImageRemoveCompleted.String()}, nil
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

// RemoveDockerVolume 在 Docker 删除前读取数据卷，以保留安全审计所需元数据。
func (s *service) RemoveDockerVolume(ctx context.Context, id string, force bool) error {
	if !s.dangerousActionsAllowed(ctx) {
		return errDangerousActionsDisabled
	}
	reader, err := s.dockerResources(ctx)
	if err != nil {
		return err
	}
	volume, err := reader.ReadDockerVolume(ctx, id)
	if err != nil {
		return err
	}
	remover, ok := reader.(interface {
		RemoveDockerVolume(context.Context, string, bool) error
	})
	if !ok {
		return errUnsupportedContainerRuntime
	}
	actionCtx, cancel := context.WithTimeout(ctx, containerOperationTTL)
	defer cancel()
	err = remover.RemoveDockerVolume(actionCtx, id, force)
	s.publishDockerVolumeAudit(ctx, volume, force, err)
	return err
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

func (s *service) Start(ctx context.Context, ref Ref) (ActionResult, error) {
	return s.runAction(ctx, ref, containerActionStart, ActionOptions{})
}

func (s *service) Stop(ctx context.Context, ref Ref) (ActionResult, error) {
	return s.runAction(ctx, ref, containerActionStop, ActionOptions{})
}

func (s *service) Restart(ctx context.Context, ref Ref) (ActionResult, error) {
	return s.runAction(ctx, ref, containerActionRestart, ActionOptions{})
}

func (s *service) Remove(ctx context.Context, ref Ref, options RemoveOptions) (ActionResult, error) {
	return s.runAction(ctx, ref, containerActionRemove, ActionOptions(options))
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

// BatchLifecycleAction 为 start、stop、restart 逐项提交独立 Task，并把单项解析、策略或提交失败保留在对应 item 中。
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
	for _, rawID := range normalized.IDs {
		ref, parseErr := parseRef(rawID)
		if parseErr != nil {
			item := batchLifecycleActionFailure(rawID, normalized.Action, parseErr)
			result.Items = append(result.Items, item)
			result.FailedCount++
			s.publishLifecycleTaskSubmissionAudit(ctx, Ref{Value: rawID}, normalized.Action, ActionOptions{Force: normalized.Force}, moduleapi.TaskReceipt{}, parseErr)
			continue
		}
		if blockedItem, blocked := s.lifecycleActionPolicyFailure(ctx, ref, normalized.Action, ActionOptions{Force: normalized.Force}); blocked {
			result.Items = append(result.Items, blockedItem)
			result.FailedCount++
			continue
		}
		receipt, submitErr := s.SubmitContainerLifecycleAction(ctx, ref, normalized.Action, ActionOptions{Force: normalized.Force}, requestedBy, batchTaskIdempotencyKey(idempotencyKey, normalized.Action, ref.Value))
		if submitErr != nil {
			result.Items = append(result.Items, batchLifecycleActionFailure(ref.Value, normalized.Action, submitErr))
			result.FailedCount++
			continue
		}
		result.Items = append(result.Items, BatchLifecycleActionItem{ID: ref.Value, Action: normalized.Action, Accepted: true, TaskID: receipt.TaskID, Status: receipt.Status})
		result.AcceptedCount++
	}
	return result, nil
}

func batchLifecycleActionFailure(id string, action string, err error) BatchLifecycleActionItem {
	messageKey := messageKeyForError(err).String()
	return BatchLifecycleActionItem{ID: id, Action: action, ErrorCode: messageKey, MessageKey: messageKey, Message: fallbackMessageForError(err)}
}

func batchTaskIdempotencyKey(base string, action string, ref string) string {
	if strings.TrimSpace(base) == "" {
		return ""
	}
	key := fmt.Sprintf("%s:%s:%s", base, action, ref)
	if utf8.RuneCountInString(key) <= moduleapi.TaskIdempotencyKeyMaxRunes {
		return key
	}
	return fmt.Sprintf("container-batch:%x", sha256.Sum256([]byte(key)))
}

func (s *service) runAction(
	ctx context.Context,
	ref Ref,
	action string,
	options ActionOptions,
) (ActionResult, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return ActionResult{}, err
	}
	runtime, err := s.runtimeForRequestContext(ctx)
	if err != nil {
		return ActionResult{}, err
	}
	if !s.dangerousActionsAllowed(ctx) {
		result := ActionResult{ID: ref.Value, Action: action, Runtime: runtimeNameDocker}
		s.publishActionAudit(ctx, result, options, errDangerousActionsDisabled)
		return ActionResult{}, errDangerousActionsDisabled
	}
	policy := s.effectiveActionPolicy(ctx)
	detail, detailErr := runtime.Detail(ctx, ref)
	orchestrator := actionAuditOrchestrator(detail, detailErr)
	orchestratorType := effectiveActionAuditOrchestratorType(orchestrator, detailErr)
	if policy.singleBlockedFor(orchestratorType) {
		result := blockedActionAuditResult(ref, detail, action, orchestrator)
		s.publishActionAudit(ctx, result, options, errDangerousActionsDisabled)
		return ActionResult{}, errDangerousActionsDisabled
	}
	actionCtx, cancel := context.WithTimeout(ctx, containerOperationTTL)
	defer cancel()
	result, err := runWithRuntime(actionCtx, ref, action, options, runtime)
	if result.Action == "" {
		result.Action = action
	}
	if shouldBackfillActionAuditOrchestrator(result.Orchestrator, detailErr) {
		result.Orchestrator = orchestrator
	}
	if err == nil {
		result = withActionMessage(result)
	}
	s.publishActionAudit(ctx, result, options, err)
	if err != nil {
		return ActionResult{}, err
	}
	return result, nil
}
