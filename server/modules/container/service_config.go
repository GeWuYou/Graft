package container

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
)

type containerRuntimeOptions struct {
	enabled                              bool
	runtime                              string
	endpoint                             string
	dangerousActionsEnabled              bool
	defaultTail                          int
	maxTail                              int
	resourceStatsCacheTTLSeconds         int
	resourceStatsCacheStaleWindowSeconds int
	resourceStatsCollectIntervalSeconds  int
	environmentPolicy                    containercontract.EnvironmentPolicy
	orchestratorPolicies                 orchestratorActionPolicies
	logger                               *zap.Logger
}

// containerOptionsFromConfig 从模块上下文构建容器运行时选项，并按默认值、配置注册表和显式模块配置的顺序应用覆盖。
func containerOptionsFromConfig(ctx *module.Context) containerRuntimeOptions {
	options := containerRuntimeOptions{
		enabled:                              defaultContainerEnabled,
		runtime:                              defaultContainerRuntime,
		endpoint:                             defaultContainerDockerEndpoint,
		dangerousActionsEnabled:              defaultContainerDangerousActionsEnabled,
		defaultTail:                          defaultContainerLogsDefaultTail,
		maxTail:                              defaultContainerLogsMaxTail,
		resourceStatsCacheTTLSeconds:         defaultContainerResourceStatsCacheTTL,
		resourceStatsCacheStaleWindowSeconds: defaultContainerResourceStatsStaleWindow,
		resourceStatsCollectIntervalSeconds:  defaultContainerResourceStatsCollectInterval,
		environmentPolicy:                    defaultContainerEnvironmentPolicy,
		orchestratorPolicies: orchestratorActionPolicies{
			Compose:    defaultContainerComposeActionLevel,
			Swarm:      defaultContainerSwarmActionLevel,
			Kubernetes: defaultContainerKubernetesActionLevel,
			Unknown:    defaultContainerUnknownActionLevel,
		},
	}
	if ctx == nil {
		return options
	}
	applyContainerBoolDefault(ctx, containercontract.ContainerRuntimeEnabledConfig.String(), &options.enabled)
	applyContainerStringDefault(ctx, containercontract.ContainerRuntimeConfig.String(), &options.runtime)
	applyContainerStringDefault(ctx, containercontract.ContainerDockerEndpointConfig.String(), &options.endpoint)
	applyContainerIntDefault(ctx, containercontract.ContainerLogsDefaultTailConfig.String(), &options.defaultTail)
	applyContainerIntDefault(ctx, containercontract.ContainerLogsMaxTailConfig.String(), &options.maxTail)
	applyContainerIntDefault(ctx, containercontract.ContainerResourceStatsCacheTTLConfig.String(), &options.resourceStatsCacheTTLSeconds)
	applyContainerIntDefault(
		ctx,
		containercontract.ContainerResourceStatsCacheStaleWindowConfig.String(),
		&options.resourceStatsCacheStaleWindowSeconds,
	)
	applyContainerIntDefault(
		ctx,
		containercontract.ContainerResourceStatsCollectIntervalConfig.String(),
		&options.resourceStatsCollectIntervalSeconds,
	)
	applyContainerBoolDefault(ctx, containercontract.ContainerDangerousActionsEnabledConfig.String(), &options.dangerousActionsEnabled)
	applyContainerEnvironmentPolicyDefault(ctx, containercontract.ContainerEnvironmentPolicyConfig.String(), &options.environmentPolicy)
	applyContainerOrchestratorActionLevelDefault(ctx, containercontract.ContainerComposeActionLevelConfig.String(), &options.orchestratorPolicies.Compose)
	applyContainerOrchestratorActionLevelDefault(ctx, containercontract.ContainerSwarmActionLevelConfig.String(), &options.orchestratorPolicies.Swarm)
	applyContainerOrchestratorActionLevelDefault(ctx, containercontract.ContainerKubernetesActionLevelConfig.String(), &options.orchestratorPolicies.Kubernetes)
	applyContainerOrchestratorActionLevelDefault(ctx, containercontract.ContainerUnknownActionLevelConfig.String(), &options.orchestratorPolicies.Unknown)
	if ctx.Config != nil {
		options.enabled = ctx.Config.Container.RuntimeEnabled
		options.runtime = ctx.Config.Container.Runtime
		options.endpoint = ctx.Config.Container.DockerEndpoint
		options.defaultTail = ctx.Config.Container.LogsDefaultTail
		options.maxTail = ctx.Config.Container.LogsMaxTail
		options.dangerousActionsEnabled = ctx.Config.Container.DangerousActionsEnabled
	}
	return options
}

// applyContainerOrchestratorActionLevelDefault applies a default orchestrator action level from the configuration registry to the target, normalizing the value and silently ignoring missing or invalid values.
func applyContainerOrchestratorActionLevelDefault(
	ctx *module.Context,
	key string,
	target *containercontract.OrchestratorActionLevel,
) {
	if target == nil {
		return
	}
	raw, ok := containerDefaultValue(ctx, key)
	if !ok {
		return
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		*target = normalizeOrchestratorActionLevel(value)
	}
}

// applyContainerEnvironmentPolicyDefault 从配置注册表中读取默认容器环境策略，并应用到 target，对缺失或无效值无声忽略。
func applyContainerEnvironmentPolicyDefault(ctx *module.Context, key string, target *containercontract.EnvironmentPolicy) {
	if target == nil {
		return
	}
	raw, ok := containerDefaultValue(ctx, key)
	if !ok {
		return
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		*target = normalizeEnvironmentPolicy(value)
	}
}

func applyContainerBoolDefault(ctx *module.Context, key string, target *bool) {
	if target == nil {
		return
	}
	raw, ok := containerDefaultValue(ctx, key)
	if !ok {
		return
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err == nil {
		*target = value
	}
}

// applyContainerStringDefault 从容器配置注册表为目标指针应用字符串默认值。
func applyContainerStringDefault(ctx *module.Context, key string, target *string) {
	if target == nil {
		return
	}
	raw, ok := containerDefaultValue(ctx, key)
	if !ok {
		return
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil && strings.TrimSpace(value) != "" {
		*target = strings.TrimSpace(value)
	}
}

// applyContainerIntDefault 从配置注册表应用正整数默认值至目标。
func applyContainerIntDefault(ctx *module.Context, key string, target *int) {
	if target == nil {
		return
	}
	raw, ok := containerDefaultValue(ctx, key)
	if !ok {
		return
	}
	var value int
	if err := json.Unmarshal(raw, &value); err == nil && value > 0 {
		*target = value
	}
}

// systemConfigReadContext selects an appropriate context for system configuration operations.
// systemConfigReadContext returns the module's lifecycle context if available,
// otherwise a background context.
func systemConfigReadContext(ctx *module.Context) context.Context {
	if ctx != nil && ctx.LifecycleContext != nil {
		return ctx.LifecycleContext
	}
	return context.Background()
}

// resolveStartupRuntimeOptions updates the provided container runtime options by resolving runtime and endpoint configuration from system config, using the provided values as fallbacks.
func resolveStartupRuntimeOptions(
	ctx context.Context,
	resolver moduleapi.SystemConfigResolver,
	options containerRuntimeOptions,
) containerRuntimeOptions {
	options.runtime = resolveStringConfigValue(ctx, resolver, containercontract.ContainerRuntimeConfig.String(), options.runtime)
	options.endpoint = resolveStringConfigValue(ctx, resolver, containercontract.ContainerDockerEndpointConfig.String(), options.endpoint)
	return options
}

// containerDefaultValue 从模块上下文的配置注册表中检索指定配置项的默认值，返回对应的 JSON 消息及该值是否存在的标志。
func containerDefaultValue(ctx *module.Context, key string) (json.RawMessage, bool) {
	if ctx == nil || ctx.ConfigRegistry == nil {
		return nil, false
	}
	definition, ok := ctx.ConfigRegistry.Get(key)
	if !ok || len(definition.DefaultValue) == 0 {
		return nil, false
	}
	return definition.DefaultValue, true
}

// resolveSystemConfigResolver resolves the system config resolver from the module context's services, returning nil if unavailable or unresolved.
func resolveSystemConfigResolver(ctx *module.Context) moduleapi.SystemConfigResolver {
	if ctx == nil || ctx.Services == nil {
		return nil
	}
	resolved, err := ctx.Services.Resolve((*moduleapi.SystemConfigResolver)(nil))
	if err != nil {
		return nil
	}
	resolver, ok := resolved.(moduleapi.SystemConfigResolver)
	if !ok {
		return nil
	}
	return resolver
}

// resolveStringConfigValue resolves a string configuration value by key, trimmed of whitespace. If resolution fails or the resolved value is blank, the trimmed fallback is returned.
func resolveStringConfigValue(
	ctx context.Context,
	resolver moduleapi.SystemConfigResolver,
	key string,
	fallback string,
) string {
	if resolver == nil {
		return strings.TrimSpace(fallback)
	}
	raw, err := resolver.ResolveDefaultConfig(ctx, key)
	if err != nil {
		return strings.TrimSpace(fallback)
	}
	var value string
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return strings.TrimSpace(fallback)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return strings.TrimSpace(fallback)
	}
	return value
}

func (s *service) resolveIntegerConfig(ctx context.Context, key string, fallback int) int {
	if s == nil || s.systemConfig == nil {
		return fallback
	}
	raw, err := s.systemConfig.ResolveDefaultConfig(ctx, key)
	if err != nil {
		return fallback
	}
	var value int
	if err := json.Unmarshal([]byte(raw), &value); err != nil || value <= 0 {
		return fallback
	}
	return value
}

// NormalizeContainerLogTailBounds normalizes log tail bounds, applying package defaults
// for non-positive values and capping maxTail to a maximum limit.
// normalizeContainerLogTailBounds ensures default and maximum log tail bounds are positive, capped to system limits, and properly ordered.
func normalizeContainerLogTailBounds(defaultTail int, maxTail int) (int, int) {
	if defaultTail <= 0 {
		defaultTail = defaultContainerLogsDefaultTail
	}
	if maxTail <= 0 || maxTail > defaultContainerLogsMaxTail {
		maxTail = defaultContainerLogsMaxTail
	}
	if defaultTail > maxTail {
		defaultTail = maxTail
	}
	return defaultTail, maxTail
}

func (s *service) effectiveLogTailBounds(ctx context.Context) (int, int) {
	defaultTail := defaultContainerLogsDefaultTail
	maxTail := defaultContainerLogsMaxTail
	if s != nil {
		defaultTail = s.defaultTail
		maxTail = s.maxTail
	}
	defaultTail = s.resolveIntegerConfig(ctx, containercontract.ContainerLogsDefaultTailConfig.String(), defaultTail)
	maxTail = s.resolveIntegerConfig(ctx, containercontract.ContainerLogsMaxTailConfig.String(), maxTail)
	return normalizeContainerLogTailBounds(defaultTail, maxTail)
}

// normalizeContainerResourceStatsCacheBounds 归一化资源统计缓存的 TTL 和过期窗口。
// 当任一值小于等于 0 时，使用默认配置值。
func normalizeContainerResourceStatsCacheBounds(ttlSeconds int, staleWindowSeconds int) (int, int) {
	if ttlSeconds <= 0 {
		ttlSeconds = defaultContainerResourceStatsCacheTTL
	}
	if staleWindowSeconds <= 0 {
		staleWindowSeconds = defaultContainerResourceStatsStaleWindow
	}
	return ttlSeconds, staleWindowSeconds
}

// normalizeContainerResourceStatsCollectInterval 将资源统计采集间隔归一为有效值。
// 当 intervalSeconds 小于等于 0 时，返回默认采集间隔。
func normalizeContainerResourceStatsCollectInterval(intervalSeconds int) int {
	if intervalSeconds <= 0 {
		return defaultContainerResourceStatsCollectInterval
	}
	return intervalSeconds
}

func (s *service) effectiveResourceStatsCollectInterval(ctx context.Context) time.Duration {
	intervalSeconds := defaultContainerResourceStatsCollectInterval
	if s != nil {
		intervalSeconds = s.runtimeOptions.resourceStatsCollectIntervalSeconds
	}
	intervalSeconds = s.resolveIntegerConfig(
		ctx,
		containercontract.ContainerResourceStatsCollectIntervalConfig.String(),
		intervalSeconds,
	)
	return time.Duration(normalizeContainerResourceStatsCollectInterval(intervalSeconds)) * time.Second
}

func (s *service) effectiveResourceStatsCacheBounds(ctx context.Context) (time.Duration, time.Duration) {
	ttlSeconds := defaultContainerResourceStatsCacheTTL
	staleWindowSeconds := defaultContainerResourceStatsStaleWindow
	if s != nil {
		ttlSeconds = s.runtimeOptions.resourceStatsCacheTTLSeconds
		staleWindowSeconds = s.runtimeOptions.resourceStatsCacheStaleWindowSeconds
	}
	ttlSeconds = s.resolveIntegerConfig(ctx, containercontract.ContainerResourceStatsCacheTTLConfig.String(), ttlSeconds)
	staleWindowSeconds = s.resolveIntegerConfig(
		ctx,
		containercontract.ContainerResourceStatsCacheStaleWindowConfig.String(),
		staleWindowSeconds,
	)
	ttlSeconds, staleWindowSeconds = normalizeContainerResourceStatsCacheBounds(ttlSeconds, staleWindowSeconds)
	return time.Duration(ttlSeconds) * time.Second, time.Duration(staleWindowSeconds) * time.Second
}

func (s *service) runtimeAccessEnabled(ctx context.Context) bool {
	if s == nil {
		return false
	}
	if s.systemConfig == nil {
		return s.enabled
	}
	return s.systemConfig.IsBooleanConfigEnabled(ctx, containercontract.ContainerRuntimeEnabledConfig.String(), s.enabled)
}

func (s *service) dangerousActionsAllowed(ctx context.Context) bool {
	if s == nil {
		return false
	}
	if s.systemConfig == nil {
		return s.dangerousActionsEnabled
	}
	return s.systemConfig.IsBooleanConfigEnabled(
		ctx,
		containercontract.ContainerDangerousActionsEnabledConfig.String(),
		s.dangerousActionsEnabled,
	)
}

type orchestratorActionPolicies struct {
	Compose    containercontract.OrchestratorActionLevel
	Swarm      containercontract.OrchestratorActionLevel
	Kubernetes containercontract.OrchestratorActionLevel
	Unknown    containercontract.OrchestratorActionLevel
}

func (p orchestratorActionPolicies) normalized() orchestratorActionPolicies {
	p.Compose = normalizeOrchestratorActionLevel(p.Compose.String())
	p.Swarm = normalizeOrchestratorActionLevel(p.Swarm.String())
	p.Kubernetes = normalizeOrchestratorActionLevel(p.Kubernetes.String())
	p.Unknown = normalizeOrchestratorActionLevel(p.Unknown.String())
	return p
}

func (p orchestratorActionPolicies) levelFor(orchestratorType string) containercontract.OrchestratorActionLevel {
	switch strings.TrimSpace(strings.ToLower(orchestratorType)) {
	case containerOrchestratorCompose:
		return p.Compose
	case containerOrchestratorSwarm:
		return p.Swarm
	case containerOrchestratorKubernetes:
		return p.Kubernetes
	case containerOrchestratorUnknown:
		return p.Unknown
	default:
		return containercontract.ContainerOrchestratorActionLevelAllow
	}
}

type effectiveActionPolicy struct {
	dangerousAllowed bool
	orchestrators    orchestratorActionPolicies
}

func (p effectiveActionPolicy) decorate(info OrchestratorInfo) OrchestratorInfo {
	info = normalizedOrchestratorInfo(info)
	level := p.orchestrators.levelFor(info.Type)
	if !p.dangerousAllowed {
		level = containercontract.ContainerOrchestratorActionLevelReadonly
	}
	info.ActionLevel = level.String()
	info.BatchActionAllowed = p.dangerousAllowed && level == containercontract.ContainerOrchestratorActionLevelAllow
	info.Warnings = orchestratorWarningsFor(info, level)
	if info.Managed && strings.TrimSpace(info.RecommendedAction) == "" {
		info.RecommendedAction = recommendedActionOpenController
	}
	return info
}

func (p effectiveActionPolicy) singleBlockedFor(orchestratorType string) bool {
	if !p.dangerousAllowed {
		return true
	}
	return p.orchestrators.levelFor(orchestratorType) == containercontract.ContainerOrchestratorActionLevelReadonly
}

func (p effectiveActionPolicy) batchBlockedFor(orchestratorType string) bool {
	if !p.dangerousAllowed {
		return true
	}
	return p.orchestrators.levelFor(orchestratorType) != containercontract.ContainerOrchestratorActionLevelAllow
}

func (s *service) effectiveActionPolicy(ctx context.Context) effectiveActionPolicy {
	return effectiveActionPolicy{
		dangerousAllowed: s.dangerousActionsAllowed(ctx),
		orchestrators:    s.effectiveOrchestratorPolicies(ctx),
	}
}

func (s *service) effectiveOrchestratorPolicies(ctx context.Context) orchestratorActionPolicies {
	if s == nil || s.systemConfig == nil {
		if s == nil {
			return orchestratorActionPolicies{}.normalized()
		}
		return s.orchestratorPolicies.normalized()
	}
	fallback := s.orchestratorPolicies.normalized()
	return orchestratorActionPolicies{
		Compose:    s.resolveOrchestratorActionLevel(ctx, containercontract.ContainerComposeActionLevelConfig.String(), fallback.Compose),
		Swarm:      s.resolveOrchestratorActionLevel(ctx, containercontract.ContainerSwarmActionLevelConfig.String(), fallback.Swarm),
		Kubernetes: s.resolveOrchestratorActionLevel(ctx, containercontract.ContainerKubernetesActionLevelConfig.String(), fallback.Kubernetes),
		Unknown:    s.resolveOrchestratorActionLevel(ctx, containercontract.ContainerUnknownActionLevelConfig.String(), fallback.Unknown),
	}.normalized()
}

func (s *service) resolveOrchestratorActionLevel(
	ctx context.Context,
	key string,
	fallback containercontract.OrchestratorActionLevel,
) containercontract.OrchestratorActionLevel {
	if s == nil || s.systemConfig == nil {
		return fallback
	}
	raw, err := s.systemConfig.ResolveDefaultConfig(ctx, key)
	if err != nil {
		return fallback
	}
	var value string
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fallback
	}
	return normalizeOrchestratorActionLevel(value)
}

func (s *service) shellAllowed(ctx context.Context) bool {
	if s == nil {
		return false
	}
	if s.systemConfig == nil {
		return s.shellEnabled
	}
	return s.systemConfig.IsBooleanConfigEnabled(
		ctx,
		containercontract.ContainerShellEnabledConfig.String(),
		s.shellEnabled,
	)
}

func (s *service) maskedEnvironmentCopyEnabled(ctx context.Context) bool {
	if s == nil || s.systemConfig == nil {
		return defaultContainerEnvironmentMaskedCopy
	}
	return s.systemConfig.IsBooleanConfigEnabled(
		ctx,
		containercontract.ContainerEnvironmentMaskedCopyEnabledConfig.String(),
		defaultContainerEnvironmentMaskedCopy,
	)
}

// 当容器运行被禁用时返回禁用运行时；当运行时类型为 Docker 或默认值时返回 Docker 运行时；其他类型返回错误。
func newContainerRuntime(options containerRuntimeOptions) (Runtime, error) {
	if !options.enabled {
		return disabledRuntime{}, nil
	}
	if strings.TrimSpace(options.runtime) != defaultContainerRuntime && strings.TrimSpace(options.runtime) != runtimeNameDocker {
		return nil, errUnsupportedContainerRuntime
	}
	return NewDockerRuntime(
		options.endpoint,
		options.logger,
		time.Duration(options.resourceStatsCacheTTLSeconds)*time.Second,
		time.Duration(options.resourceStatsCacheStaleWindowSeconds)*time.Second,
	)
}

func (s *service) runtimeForRequest() (Runtime, error) {
	if s == nil {
		return nil, errRuntimeDisabled
	}
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()
	if s.runtime != nil {
		if _, disabled := s.runtime.(disabledRuntime); !disabled {
			return s.runtime, nil
		}
	}
	options := s.runtimeOptions
	options.enabled = true
	options.dangerousActionsEnabled = s.dangerousActionsEnabled
	options.defaultTail = s.defaultTail
	options.maxTail = s.maxTail
	options.resourceStatsCacheTTLSeconds = s.runtimeOptions.resourceStatsCacheTTLSeconds
	options.resourceStatsCacheStaleWindowSeconds = s.runtimeOptions.resourceStatsCacheStaleWindowSeconds
	options.logger = s.logger
	runtime, err := s.runtimeFactory(options)
	if err != nil {
		return nil, err
	}
	if dockerRuntime, ok := runtime.(*DockerRuntime); ok {
		ttl, staleWindow := s.effectiveResourceStatsCacheBounds(context.Background())
		dockerRuntime.updateResourceStatsCachePolicy(ttl, staleWindow)
	}
	s.runtime = runtime
	return runtime, nil
}

func (s *service) startStatsCollector(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.realtimeHub == nil {
		return nil
	}
	if s.statsCollector == nil {
		s.statsCollector = newStatsCollector(
			s.collectStatsSnapshots,
			s.realtimeHub,
			s.logger,
			s.moduleName,
		)
	}
	s.statsCollector.interval = s.effectiveResourceStatsCollectInterval(ctx)
	return s.statsCollector.Start(ctx)
}

func (s *service) stopStatsCollector(ctx context.Context) error {
	if s == nil || s.statsCollector == nil {
		return nil
	}
	return s.statsCollector.Stop(ctx)
}

func (s *service) startRuntimeEventManager(ctx context.Context) error {
	if s == nil || s.realtimeHub == nil {
		return nil
	}
	s.runtimeEventManagerMu.Lock()
	if s.runtimeEventManager == nil {
		s.runtimeEventManager = newRuntimeEventManager(
			s.realtimeHub,
			s.logger,
			s.runtimeEventSourceRegistrations(),
			RuntimeEventStreamContext{Runtime: s.defaultRuntimeEventStreamRuntime()},
		)
	}
	manager := s.runtimeEventManager
	s.runtimeEventManagerMu.Unlock()
	return manager.Start(ctx)
}

func (s *service) runtimeEventManagerForRead() *runtimeEventManager {
	if s == nil {
		return nil
	}
	s.runtimeEventManagerMu.RLock()
	defer s.runtimeEventManagerMu.RUnlock()
	return s.runtimeEventManager
}

func (s *service) runtimeEventSourceRegistrations() []runtimeEventSourceRegistration {
	if s == nil {
		return nil
	}
	return []runtimeEventSourceRegistration{
		{
			name:          s.defaultRuntimeEventSourceName(),
			streamContext: RuntimeEventStreamContext{Runtime: s.defaultRuntimeEventStreamRuntime()},
			load: func() (RuntimeEventSource, error) {
				runtime, err := s.runtimeForRequest()
				if err != nil {
					if errors.Is(err, errRuntimeDisabled) {
						return nil, nil
					}
					return nil, err
				}
				source, ok := runtime.(RuntimeEventSource)
				if !ok {
					return nil, nil
				}
				return source, nil
			},
		},
	}
}

func (s *service) defaultRuntimeEventSourceName() string {
	return firstNonEmpty(strings.TrimSpace(s.runtimeOptions.runtime), runtimeNameDocker)
}

func (s *service) defaultRuntimeEventStreamRuntime() string {
	return s.defaultRuntimeEventSourceName()
}

func (s *service) collectStatsSnapshots(ctx context.Context) ([]StatsSnapshot, error) {
	if s == nil || !s.runtimeAccessEnabled(ctx) {
		return nil, nil
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		return nil, err
	}
	collectorRuntime, ok := runtime.(StatsCollectorRuntime)
	if !ok {
		return nil, nil
	}
	return collectorRuntime.CollectStatsSnapshots(ctx)
}

func (s *service) registerRealtimeTopics() error {
	if s == nil {
		return nil
	}
	if s.topicIssuers == nil {
		return errors.New("realtime topic issuer registry is unavailable")
	}
	if err := s.topicIssuers.Register(containercontract.ContainerListStatsTopic, s); err != nil {
		return err
	}
	if err := s.topicIssuers.Register(containercontract.ContainerDashboardSummaryTopic, s); err != nil {
		return err
	}
	if err := s.topicIssuers.Register(containercontract.ContainerStatsTopicPrefix, s); err != nil {
		return err
	}
	if err := s.topicIssuers.Register(containercontract.ContainerEventsTopicPrefix, s); err != nil {
		return err
	}
	return s.topicIssuers.Register(containercontract.ContainerLogsTopicPrefix, s)
}
