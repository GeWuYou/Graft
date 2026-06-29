package container

import (
	"context"
	"errors"
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
	logTopicStreamerFactory              func(realtime.Hub, *zap.Logger, func() (Runtime, error)) (*logTopicStreamer, error)
}

// newContainerService 根据模块上下文初始化容器服务，并解析运行时、实时订阅和鉴权依赖。
// 解析任一必需依赖失败时返回错误。
func newContainerService(ctx *module.Context, moduleName string) (*service, error) {
	options := containerOptionsFromConfig(ctx)
	systemConfig := resolveSystemConfigResolver(ctx)
	options = resolveStartupRuntimeOptions(systemConfigReadContext(ctx), systemConfig, options)
	runtime := Runtime(disabledRuntime{})
	allowedOrigins := []string{}
	if ctx != nil && ctx.Config != nil {
		allowedOrigins = append(allowedOrigins, ctx.Config.HTTPX.WebSocketAllowedOrigins...)
	}
	realtimeTickets, err := resolveRealtimeTicketService(ctx)
	if err != nil {
		return nil, err
	}
	realtimeHub, err := resolveRealtimeHub(ctx)
	if err != nil {
		return nil, err
	}
	topicIssuers, err := resolveRealtimeTopicIssuerRegistry(ctx)
	if err != nil {
		return nil, err
	}
	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return nil, err
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
	})
}

// newService 初始化容器服务实例，并应用默认值与归一化配置。
// realtimeTickets 不能为空，否则返回错误。
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
	return &service{
		runtime:                 options.runtime,
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
		logTopicStreamerFactory: options.logTopicStreamerFactory,
	}, nil
}

// resolveRealtimeTicketService 从模块上下文中解析实时认证服务。
//
// 当 ctx 或 ctx.Services 为空时返回错误。
//
// @returns 解析得到的 realtimeauth.Service，或在上下文不可用时返回错误。
func resolveRealtimeTicketService(ctx *module.Context) (realtimeauth.Service, error) {
	if ctx == nil || ctx.Services == nil {
		return nil, errors.New("realtime ticket service resolver is unavailable")
	}

	return module.ResolveService[realtimeauth.Service](ctx.Services, (*realtimeauth.Service)(nil))
}

// resolveRealtimeHub 从模块上下文中解析实时消息总线。
// 优先返回 ctx.Realtime；当 ctx.Services 可用时，再从服务容器中解析 realtime.Hub。
//
// @returns 解析到的实时消息总线；当上下文或服务解析器不可用时返回错误。
func resolveRealtimeHub(ctx *module.Context) (realtime.Hub, error) {
	if ctx != nil && ctx.Realtime != nil {
		return ctx.Realtime, nil
	}
	if ctx == nil || ctx.Services == nil {
		return nil, errors.New("realtime hub resolver is unavailable")
	}

	return module.ResolveService[realtime.Hub](ctx.Services, (*realtime.Hub)(nil))
}

// 当 ctx 或其 Services 为空时返回错误。
func resolveRealtimeTopicIssuerRegistry(ctx *module.Context) (realtime.TopicIssuerRegistry, error) {
	if ctx == nil || ctx.Services == nil {
		return nil, errors.New("realtime topic issuer registry resolver is unavailable")
	}

	return module.ResolveService[realtime.TopicIssuerRegistry](ctx.Services, (*realtime.TopicIssuerRegistry)(nil))
}

func (s *service) Close() error {
	if s == nil {
		return nil
	}
	var closeErr error
	s.logTopicStreamerMu.Lock()
	logTopicStreamer := s.logTopicStreamer
	s.logTopicStreamer = nil
	s.logTopicStreamerMu.Unlock()
	if logTopicStreamer != nil {
		if err := logTopicStreamer.Close(context.Background()); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
	}
	if s.statsCollector != nil {
		if err := s.statsCollector.Stop(context.Background()); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
		s.statsCollector = nil
	}
	s.runtimeEventManagerMu.Lock()
	runtimeEventManager := s.runtimeEventManager
	s.runtimeEventManager = nil
	s.runtimeEventManagerMu.Unlock()
	if runtimeEventManager != nil {
		if err := runtimeEventManager.Stop(context.Background()); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
	}
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()
	runtime := s.runtime
	if runtime == nil {
		return closeErr
	}
	s.runtime = nil
	if err := runtime.Close(); err != nil {
		closeErr = errors.Join(closeErr, err)
	}
	return closeErr
}

func (s *service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return ListResult{}, err
	}
	normalized, err := normalizeListQuery(query)
	if err != nil {
		return ListResult{}, err
	}
	runtime, err := s.runtimeForRequest()
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
		Runtime: info,
		Items:   paged,
		Total:   len(filtered),
		Limit:   normalized.Limit,
		Offset:  normalized.Offset,
		Summary: summarizeContainers(filtered),
	}, nil
}

func (s *service) DashboardSummary(ctx context.Context, _ dashboardSummaryQuery) (dashboardSummaryResult, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return dashboardSummaryResult{}, err
	}
	runtime, err := s.runtimeForRequest()
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

func (s *service) Detail(ctx context.Context, ref Ref) (Detail, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return Detail{}, err
	}
	runtime, err := s.runtimeForRequest()
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
	runtime, err := s.runtimeForRequest()
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
	runtime, err := s.runtimeForRequest()
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
	runtime, err := s.runtimeForRequest()
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

func (s *service) BatchAction(ctx context.Context, command BatchActionCommand) (BatchActionResult, error) {
	normalized, err := normalizeBatchActionCommand(command)
	if err != nil {
		return BatchActionResult{}, err
	}
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return BatchActionResult{}, err
	}
	policy := s.effectiveActionPolicy(ctx)
	if !policy.dangerousAllowed {
		blocked := BatchActionResult{
			Action:    normalized.Action,
			Total:     len(normalized.IDs),
			RequestID: requestIDFromContext(ctx),
			Items:     make([]BatchActionItem, 0, len(normalized.IDs)),
		}
		for _, ref := range normalized.IDs {
			result := ActionResult{ID: ref, Action: normalized.Action, Runtime: runtimeNameDocker}
			s.publishActionAudit(ctx, result, ActionOptions{Force: normalized.Force}, errDangerousActionsDisabled)
			blocked.Items = append(blocked.Items, batchActionFailure(ref, normalized.Action, errDangerousActionsDisabled))
		}
		blocked.FailedCount = len(blocked.Items)
		blocked = withBatchActionMessage(blocked)
		s.publishBatchActionAudit(ctx, blocked, ActionOptions{Force: normalized.Force})
		return BatchActionResult{}, errDangerousActionsDisabled
	}
	result := BatchActionResult{
		Action:    normalized.Action,
		Total:     len(normalized.IDs),
		RequestID: requestIDFromContext(ctx),
		Items:     make([]BatchActionItem, 0, len(normalized.IDs)),
	}
	for _, rawID := range normalized.IDs {
		ref, parseErr := parseRef(rawID)
		if parseErr != nil {
			item := batchActionFailure(rawID, normalized.Action, parseErr)
			result.Items = append(result.Items, item)
			result.FailedCount++
			s.publishActionAudit(ctx, item.Result, ActionOptions{Force: normalized.Force}, parseErr)
			continue
		}
		if blockedItem, blocked := s.batchActionPolicyFailure(
			ctx,
			ref,
			normalized.Action,
			ActionOptions{Force: normalized.Force},
		); blocked {
			result.Items = append(result.Items, blockedItem)
			result.FailedCount++
			continue
		}
		actionResult, actionErr := s.runAction(ctx, ref, normalized.Action, ActionOptions{Force: normalized.Force})
		item := batchActionItem(ref.Value, normalized.Action, actionResult, actionErr)
		result.Items = append(result.Items, item)
		if actionErr != nil {
			result.FailedCount++
			continue
		}
		result.SuccessCount++
	}
	result = withBatchActionMessage(result)
	s.publishBatchActionAudit(ctx, result, ActionOptions{Force: normalized.Force})
	return result, nil
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
	runtime, err := s.runtimeForRequest()
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
