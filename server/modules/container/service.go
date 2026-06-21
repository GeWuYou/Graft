// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtimeauth"
	containercontract "graft/server/modules/container/contract"
	"graft/server/modules/container/terminal"
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
}

type containerServiceOptions struct {
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
}

// newContainerService constructs a container service with configuration from the module context.
func newContainerService(ctx *module.Context, moduleName string) (*service, error) {
	options := containerOptionsFromConfig(ctx)
	runtime := Runtime(disabledRuntime{})
	allowedOrigins := []string{}
	if ctx != nil && ctx.Config != nil {
		allowedOrigins = append(allowedOrigins, ctx.Config.HTTPX.WebSocketAllowedOrigins...)
	}
	return newService(containerServiceOptions{
		runtime:                 runtime,
		runtimeOptions:          options,
		systemConfig:            resolveSystemConfigResolver(ctx),
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
	})
}

// newService 创建容器服务，规范化配置参数并为缺失的组件应用默认实现。
func newService(options containerServiceOptions) (*service, error) {
	if options.defaultTail <= 0 {
		options.defaultTail = defaultContainerLogsDefaultTail
	}
	if options.maxTail <= 0 || options.maxTail > defaultContainerLogsMaxTail {
		options.maxTail = defaultContainerLogsMaxTail
	}
	if options.defaultTail > options.maxTail {
		options.defaultTail = options.maxTail
	}
	if options.realtimeTickets == nil {
		options.realtimeTickets = realtimeauth.NewMemoryService()
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
	}, nil
}

func (s *service) Close() error {
	if s == nil {
		return nil
	}
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()
	if s.runtime == nil {
		return nil
	}
	return s.runtime.Close()
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
	normalized, err := s.normalizeLogQuery(query)
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
		for _, ref := range normalized.IDs {
			result := ActionResult{ID: ref, Action: normalized.Action, Runtime: runtimeNameDocker}
			s.publishActionAudit(ctx, result, ActionOptions{Force: normalized.Force}, errDangerousActionsDisabled)
		}
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
	orchestratorType := containerOrchestratorUnknown
	if detailErr == nil {
		orchestratorType = effectiveOrchestratorType(detail.Summary)
	}
	if policy.singleBlockedFor(orchestratorType) {
		result := ActionResult{
			ID:      firstNonEmpty(ref.Value, detail.ID),
			Name:    detail.Name,
			Image:   detail.Image,
			Action:  action,
			Runtime: runtimeNameDocker,
		}
		s.publishActionAudit(ctx, result, options, errDangerousActionsDisabled)
		return ActionResult{}, errDangerousActionsDisabled
	}
	actionCtx, cancel := context.WithTimeout(ctx, containerOperationTTL)
	defer cancel()
	result, err := runWithRuntime(actionCtx, ref, action, options, runtime)
	if result.Action == "" {
		result.Action = action
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

func (s *service) batchActionPolicyFailure(
	ctx context.Context,
	ref Ref,
	action string,
	options ActionOptions,
) (BatchActionItem, bool) {
	if !isSupportedAction(action) {
		return BatchActionItem{}, false
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		return batchActionFailure(ref.Value, action, err), true
	}
	policy := s.effectiveActionPolicy(ctx)
	detail, detailErr := runtime.Detail(ctx, ref)
	orchestratorType := containerOrchestratorUnknown
	if detailErr == nil {
		orchestratorType = effectiveOrchestratorType(detail.Summary)
	}
	if policy.singleBlockedFor(orchestratorType) || policy.batchBlockedFor(orchestratorType) {
		result := ActionResult{
			ID:      firstNonEmpty(ref.Value, detail.ID),
			Name:    detail.Name,
			Image:   detail.Image,
			Action:  action,
			Runtime: runtimeNameDocker,
		}
		s.publishActionAudit(ctx, result, options, errDangerousActionsDisabled)
		return batchActionItem(ref.Value, action, result, errDangerousActionsDisabled), true
	}
	return BatchActionItem{}, false
}

func (s *service) requireRuntimeAccess(ctx context.Context) error {
	if s == nil || !s.runtimeAccessEnabled(ctx) {
		return errRuntimeDisabled
	}
	return nil
}

func runWithRuntime(ctx context.Context, ref Ref, action string, options ActionOptions, runtime Runtime) (ActionResult, error) {
	switch action {
	case containerActionStart:
		return runtime.Start(ctx, ref)
	case containerActionStop:
		return runtime.Stop(ctx, ref)
	case containerActionRemove:
		return runtime.Remove(ctx, ref, RemoveOptions(options))
	case containerActionRestart:
		return runtime.Restart(ctx, ref)
	default:
		return ActionResult{ID: ref.Value, Action: action, Runtime: runtimeNameDocker}, errInvalidBatchAction
	}
}

func withActionMessage(result ActionResult) ActionResult {
	if result.MessageKey != "" {
		return result
	}
	key := actionSuccessMessageKey(result.Action)
	result.MessageKey = key.String()
	result.Message = key.String()
	return result
}

func actionSuccessMessageKey(action string) containercontract.MessageKey {
	switch action {
	case containerActionStart:
		return containercontract.ContainerActionStartCompleted
	case containerActionStop:
		return containercontract.ContainerActionStopCompleted
	case containerActionRemove:
		return containercontract.ContainerActionRemoveCompleted
	default:
		return containercontract.ContainerActionRestartCompleted
	}
}

func normalizeBatchActionCommand(command BatchActionCommand) (BatchActionCommand, error) {
	action := strings.TrimSpace(command.Action)
	if !isSupportedAction(action) {
		return BatchActionCommand{}, errInvalidBatchAction
	}
	if len(command.IDs) == 0 || len(command.IDs) > maxContainerBatchActionIDs {
		return BatchActionCommand{}, errInvalidBatchAction
	}
	normalizedIDs := make([]string, 0, len(command.IDs))
	for _, id := range command.IDs {
		if strings.TrimSpace(id) == "" {
			return BatchActionCommand{}, errInvalidBatchAction
		}
		normalizedIDs = append(normalizedIDs, strings.TrimSpace(id))
	}
	return BatchActionCommand{Action: action, IDs: normalizedIDs, Force: command.Force}, nil
}

func isSupportedAction(action string) bool {
	switch action {
	case containerActionStart, containerActionStop, containerActionRestart, containerActionRemove:
		return true
	default:
		return false
	}
}

func batchActionFailure(id string, action string, err error) BatchActionItem {
	messageKey := messageKeyForError(err).String()
	return BatchActionItem{
		ID:         id,
		Action:     action,
		Success:    false,
		ErrorCode:  messageKey,
		MessageKey: messageKey,
		Message:    fallbackMessageForError(err),
		Result: ActionResult{
			ID:      id,
			Action:  action,
			Runtime: runtimeNameDocker,
		},
	}
}

func batchActionItem(id string, action string, result ActionResult, err error) BatchActionItem {
	if err != nil {
		if result.ID == "" {
			result.ID = id
		}
		if result.Action == "" {
			result.Action = action
		}
		if result.Runtime == "" {
			result.Runtime = runtimeNameDocker
		}
		item := batchActionFailure(firstNonEmpty(result.ID, id), result.Action, err)
		item.Name = result.Name
		item.Result = result
		return item
	}
	return BatchActionItem{
		ID:         firstNonEmpty(result.ID, id),
		Name:       result.Name,
		Action:     result.Action,
		Success:    true,
		MessageKey: result.MessageKey,
		Message:    result.Message,
		Result:     result,
	}
}

func withBatchActionMessage(result BatchActionResult) BatchActionResult {
	key := containercontract.ContainerBatchActionCompleted
	switch {
	case result.SuccessCount == 0:
		key = containercontract.ContainerBatchActionFailed
	case result.FailedCount > 0:
		key = containercontract.ContainerBatchActionPartial
	}
	result.MessageKey = key.String()
	result.Message = key.String()
	return result
}

func requestIDFromContext(ctx context.Context) string {
	if requestAudit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		return requestAudit.RequestID
	}
	return ""
}

func (s *service) applyEnvironmentPolicy(ctx context.Context, detail Detail) Detail {
	policy := s.environmentDisplayPolicy(ctx)
	if policy == containercontract.ContainerEnvironmentPolicyPlain && !environmentPlainAccessAllowed(ctx) {
		policy = containercontract.ContainerEnvironmentPolicyMasked
	}
	detail.EnvironmentPolicy = policy.String()
	detail.EnvironmentMaskedCopyEnabled = s.maskedEnvironmentCopyEnabled(ctx)
	detail.Environment = applyEnvironmentPolicy(detail.Environment, environmentPolicyOptions{
		maskedCopyEnabled: policy == containercontract.ContainerEnvironmentPolicyMasked &&
			environmentPlainAccessAllowed(ctx) &&
			s.maskedEnvironmentCopyEnabled(ctx),
		policy: policy,
	})
	return detail
}

func withEnvironmentPlainAccess(ctx context.Context) context.Context {
	return context.WithValue(ctx, environmentPlainAccessContextKey{}, true)
}

func environmentPlainAccessAllowed(ctx context.Context) bool {
	allowed, _ := ctx.Value(environmentPlainAccessContextKey{}).(bool)
	return allowed
}

type stringSystemConfigResolver interface {
	ResolveDefaultConfig(ctx context.Context, key string) (string, error)
}

func (s *service) environmentDisplayPolicy(ctx context.Context) containercontract.EnvironmentPolicy {
	fallback := defaultContainerEnvironmentPolicy
	if s != nil && s.environmentPolicy != "" {
		fallback = s.environmentPolicy
	}
	if s == nil || s.systemConfig == nil {
		return fallback
	}
	resolver, ok := s.systemConfig.(stringSystemConfigResolver)
	if !ok {
		return fallback
	}
	raw, err := resolver.ResolveDefaultConfig(
		ctx,
		containercontract.ContainerEnvironmentPolicyConfig.String(),
	)
	if err != nil {
		return fallback
	}
	var value string
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fallback
	}
	return normalizeEnvironmentPolicy(value)
}

type environmentPolicyOptions struct {
	policy            containercontract.EnvironmentPolicy
	maskedCopyEnabled bool
}

// applyEnvironmentPolicy applies environment display and masking policy to variables.
// Each variable is marked sensitive if its key matches known sensitive patterns. The
// returned payload always carries explicit display-state fields so downstream consumers
// applyEnvironmentPolicy modifies environment variables to enforce the specified display policy, controlling value visibility through masking, hiding, or plaintext modes.
func applyEnvironmentPolicy(environment []EnvironmentVariable, options environmentPolicyOptions) []EnvironmentVariable {
	if len(environment) == 0 {
		return nil
	}
	mapped := make([]EnvironmentVariable, 0, len(environment))
	for _, item := range environment {
		item.Sensitive = item.Sensitive || isSensitiveEnvironmentKey(item.Key)
		item.CopyValue = ""
		item.DisplayValue = item.Value
		item.ValueMasked = false
		item.ValueHidden = false
		switch options.policy {
		case containercontract.ContainerEnvironmentPolicyHidden:
			item.Value = ""
			item.DisplayValue = "[HIDDEN]"
			item.ValueHidden = true
			item.Masked = true
		case containercontract.ContainerEnvironmentPolicyPlain:
			item.Masked = false
		default:
			if item.Sensitive {
				if options.maskedCopyEnabled && strings.TrimSpace(item.Value) != "" {
					item.CopyValue = item.Value
				}
				item.Value = ""
				item.DisplayValue = maskedEnvironmentPlaceholder
				item.ValueMasked = true
				item.Masked = true
			} else {
				item.Masked = false
			}
		}
		mapped = append(mapped, item)
	}
	return mapped
}

// normalizeEnvironmentPolicy 将字符串规范化为环境策略类型。
// 识别 Hidden 和 Plain 策略；若输入不匹配任何已知策略，则默认返回 Masked。
func normalizeEnvironmentPolicy(value string) containercontract.EnvironmentPolicy {
	switch containercontract.EnvironmentPolicy(strings.ToLower(strings.TrimSpace(value))) {
	case containercontract.ContainerEnvironmentPolicyHidden:
		return containercontract.ContainerEnvironmentPolicyHidden
	case containercontract.ContainerEnvironmentPolicyPlain:
		return containercontract.ContainerEnvironmentPolicyPlain
	default:
		return containercontract.ContainerEnvironmentPolicyMasked
	}
}

// normalizeOrchestratorActionLevel normalizes a string to an orchestrator action level,
// returning Readonly or Allow if matched, or Warn as the default.
func normalizeOrchestratorActionLevel(value string) containercontract.OrchestratorActionLevel {
	switch containercontract.OrchestratorActionLevel(strings.ToLower(strings.TrimSpace(value))) {
	case containercontract.ContainerOrchestratorActionLevelReadonly:
		return containercontract.ContainerOrchestratorActionLevelReadonly
	case containercontract.ContainerOrchestratorActionLevelAllow:
		return containercontract.ContainerOrchestratorActionLevelAllow
	default:
		return containercontract.ContainerOrchestratorActionLevelWarn
	}
}

// NormalizedOrchestratorInfo normalizes the provided orchestrator information by validating the type, deriving managed status from type, normalizing scope kinds, trimming whitespace from string fields, applying default confidence based on managed status, and ensuring the warnings slice is initialized.
func normalizedOrchestratorInfo(info OrchestratorInfo) OrchestratorInfo {
	info.Type = effectiveOrchestratorTypeFromValue(info.Type)
	info.Managed = info.Type != containerOrchestratorStandalone
	info.GroupScopeKind = normalizeContainerSourceScopeKind(info.GroupScopeKind)
	info.MemberScopeKind = normalizeContainerSourceScopeKind(info.MemberScopeKind)
	info.GroupValue = strings.TrimSpace(info.GroupValue)
	info.MemberValue = strings.TrimSpace(info.MemberValue)
	info.GroupDisplayName = strings.TrimSpace(info.GroupDisplayName)
	info.MemberDisplayName = strings.TrimSpace(info.MemberDisplayName)
	if strings.TrimSpace(info.Confidence) == "" {
		if info.Managed {
			info.Confidence = orchestratorConfidenceMedium
		} else {
			info.Confidence = orchestratorConfidenceHigh
		}
	}
	if info.Warnings == nil {
		info.Warnings = []string{}
	}
	return info
}

// EffectiveOrchestratorType returns the normalized orchestrator type from the container summary.
func effectiveOrchestratorType(item Summary) string {
	return effectiveOrchestratorTypeFromValue(item.Orchestrator.Type)
}

// effectiveOrchestratorTypeFromValue returns the normalized orchestrator type for the given value, defaulting to standalone if the value is invalid.
func effectiveOrchestratorTypeFromValue(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if isValidContainerOrchestrator(value) {
		return value
	}
	return containerOrchestratorStandalone
}

// orchestratorWarningsFor returns a deduplicated slice of warnings for an orchestrator, combining base warnings with those derived from managed status and action level constraints.
func orchestratorWarningsFor(
	info OrchestratorInfo,
	level containercontract.OrchestratorActionLevel,
) []string {
	const extraOrchestratorWarnings = 2

	seen := map[string]struct{}{}
	warnings := make([]string, 0, len(info.Warnings)+extraOrchestratorWarnings)
	appendWarning := func(code string) {
		code = strings.TrimSpace(code)
		if code == "" {
			return
		}
		if _, ok := seen[code]; ok {
			return
		}
		seen[code] = struct{}{}
		warnings = append(warnings, code)
	}
	for _, code := range info.Warnings {
		appendWarning(code)
	}
	if info.Managed {
		appendWarning(orchestratorWarningManagedActionRisk)
	}
	switch level {
	case containercontract.ContainerOrchestratorActionLevelReadonly:
		appendWarning(orchestratorWarningReadonly)
		appendWarning(orchestratorWarningBatchBlocked)
	case containercontract.ContainerOrchestratorActionLevelWarn:
		appendWarning(orchestratorWarningBatchBlocked)
	}
	return warnings
}

// isSensitiveEnvironmentKey 判断环境变量键是否表示敏感值。
func isSensitiveEnvironmentKey(key string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(key))
	for _, marker := range sensitiveEnvironmentKeyMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

var sensitiveEnvironmentKeyMarkers = []string{
	"PASSWORD",
	"PASSWD",
	"TOKEN",
	"SECRET",
	"KEY",
	"AUTH",
	"CREDENTIAL",
	"PRIVATE",
	"CERT",
	"COOKIE",
	"SESSION",
}

func (s *service) normalizeLogQuery(query LogQuery) (LogQuery, error) {
	if query.Tail == 0 {
		query.Tail = s.defaultTail
	}
	if query.Tail < 0 || query.Tail > s.maxTail || query.Tail > defaultContainerLogsMaxTail {
		return LogQuery{}, errLogsTooLarge
	}
	if !query.Stdout && !query.Stderr {
		query.Stdout = true
		query.Stderr = true
	}
	if query.Since != "" {
		if _, err := parseLogSince(query.Since); err != nil {
			return LogQuery{}, fmt.Errorf("%w: %v", errInvalidLogQuery, err)
		}
	}
	return query, nil
}

// filterContainerSummaries returns the summaries that match the query criteria.
func filterContainerSummaries(items []Summary, query ListQuery) []Summary {
	filtered := make([]Summary, 0, len(items))
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	for _, item := range items {
		if !summaryMatchesListQuery(item, query, keyword) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// summaryMatchesListQuery 确定容器摘要是否与列表查询的所有过滤条件相匹配。
func summaryMatchesListQuery(item Summary, query ListQuery, keyword string) bool {
	return summaryMatchesState(item, query.State) &&
		summaryMatchesHealth(item, query.Health) &&
		summaryMatchesOrchestrator(item, query.Orchestrator) &&
		summaryMatchesSourceScopeFilter(item, query.SourceScopeKind, query.SourceScope) &&
		summaryMatchesKeywordFilter(item, keyword)
}

// summaryMatchesState 检查容器摘要的状态是否与给定的状态匹配，空字符串表示接受任何状态。
func summaryMatchesState(item Summary, state string) bool {
	return state == "" || item.State == state
}

// summaryMatchesHealth reports whether a container summary matches the given health filter.
func summaryMatchesHealth(item Summary, health string) bool {
	return health == "" || effectiveHealth(item) == health
}

// summaryMatchesOrchestrator reports whether a container summary matches the
// given orchestrator filter.
func summaryMatchesOrchestrator(item Summary, orchestrator string) bool {
	return orchestrator == "" || effectiveOrchestratorType(item) == orchestrator
}

// summaryMatchesSourceScopeFilter 检查容器摘要是否与源作用域过滤条件匹配。
// 当 scopeKind 为空时返回 true，表示不应用该过滤；否则检查摘要是否与指定作用域相匹配。
func summaryMatchesSourceScopeFilter(item Summary, scopeKind string, scope string) bool {
	return scopeKind == "" || summaryMatchesSourceScope(item, scopeKind, scope)
}

// SummaryMatchesKeywordFilter reports whether a Summary matches the keyword filter, where an empty keyword matches all summaries.
func summaryMatchesKeywordFilter(item Summary, keyword string) bool {
	return keyword == "" || summaryMatchesKeyword(item, keyword)
}

// pageContainerSummaries 根据查询条件对容器摘要进行分页。
// 返回从指定偏移开始、不超过指定限制数量的摘要切片，若偏移超过总项数则返回空切片。
func pageContainerSummaries(items []Summary, query ListQuery) []Summary {
	if query.Offset >= len(items) {
		return []Summary{}
	}
	end := query.Offset + query.Limit
	if end > len(items) {
		end = len(items)
	}
	return items[query.Offset:end]
}

// summarizeContainers computes aggregate counts of containers grouped by state and health status.
func summarizeContainers(items []Summary) ListSummary {
	summary := ListSummary{Total: len(items)}
	for _, item := range items {
		switch item.State {
		case "running":
			summary.Running++
		case "created", "exited", "paused", "restarting":
			summary.Stopped++
		case "dead", "unknown", "removing":
			summary.Error++
		}
		switch effectiveHealth(item) {
		case containerHealthHealthy:
			summary.Healthy++
		case containerHealthUnhealthy:
			summary.Unhealthy++
		default:
			summary.HealthUnavailable++
		}
	}
	return summary
}

// applyActionAvailability 根据编排器策略和容器状态对容器摘要应用动作可用性限制，禁用危险操作被禁用或编排器操作级别为只读时的所有可变动作。
func applyActionAvailability(items []Summary, policy effectiveActionPolicy) []Summary {
	adjusted := make([]Summary, 0, len(items))
	for _, item := range items {
		item.CanRemove = canRemoveState(item.State)
		item.Orchestrator = policy.decorate(item.Orchestrator)
		if !policy.dangerousAllowed || item.Orchestrator.ActionLevel == containercontract.ContainerOrchestratorActionLevelReadonly.String() {
			item.CanStart = false
			item.CanStop = false
			item.CanRestart = false
			item.CanRemove = false
		}
		adjusted = append(adjusted, item)
	}
	return adjusted
}

// summaryMatchesKeyword reports whether the keyword matches any of the container summary's searchable fields.
func summaryMatchesKeyword(item Summary, keyword string) bool {
	values := []string{
		item.ID,
		item.ShortID,
		item.Name,
		item.Image,
		item.ImageID,
		item.Status,
		item.State,
		item.Runtime,
		item.RestartPolicy,
		item.PrimaryIP,
		item.NetworkSummary,
		item.ComposeProject,
		item.ComposeService,
	}
	values = append(values, item.Names...)
	for _, port := range item.Ports {
		values = append(values, port.IP, strconv.Itoa(port.PrivatePort), port.Type)
		if port.PublicPort != nil {
			values = append(values, strconv.Itoa(*port.PublicPort))
		}
	}
	for _, network := range item.Networks {
		values = append(values, network.Name, network.NetworkID, network.EndpointID, network.Gateway, network.IPAddress, network.MacAddress)
	}
	for key, value := range item.Labels {
		values = append(values, key, value)
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
}

// NormalizeContainerSourceScopeKind 规范化容器源作用域类型值，转换为小写并去除空白。返回规范化后的值（如果为支持的作用域类型）或空字符串。
func normalizeContainerSourceScopeKind(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if !isValidContainerSourceScopeKind(value) {
		return ""
	}
	return value
}

// sourceScopeKindCompatibleWithOrchestrator reports whether a source scope kind is compatible with an orchestrator type.
func sourceScopeKindCompatibleWithOrchestrator(orchestrator string, scopeKind string) bool {
	scopeKind = normalizeContainerSourceScopeKind(scopeKind)
	if scopeKind == "" {
		return false
	}
	switch scopeKind {
	case composeProjectScopeKind, composeServiceScopeKind:
		return orchestrator == "" || orchestrator == containerOrchestratorCompose
	case swarmStackScopeKind, swarmTaskScopeKind:
		return orchestrator == "" || orchestrator == containerOrchestratorSwarm
	case kubernetesNamespaceScopeKind, kubernetesPodScopeKind:
		return orchestrator == "" || orchestrator == containerOrchestratorKubernetes
	default:
		return false
	}
}

// summaryMatchesSourceScope 判断容器摘要是否与指定的源作用域类型和值相匹配。
// 比较采用不区分大小写的方式，源作用域类型必须与容器的编排器类型兼容。
func summaryMatchesSourceScope(item Summary, scopeKind string, scope string) bool {
	scopeKind = normalizeContainerSourceScopeKind(scopeKind)
	scope = strings.TrimSpace(scope)
	if scopeKind == "" || scope == "" {
		return false
	}
	info := normalizedOrchestratorInfo(item.Orchestrator)
	if info.Type != "" && !sourceScopeKindCompatibleWithOrchestrator(info.Type, scopeKind) {
		return false
	}
	for _, candidate := range sourceScopeCandidates(item, info, scopeKind) {
		if strings.EqualFold(candidate, scope) {
			return true
		}
	}
	return false
}

// SourceScopeCandidates returns candidate values from the container summary and orchestrator information for matching against the given scope kind.
func sourceScopeCandidates(item Summary, info OrchestratorInfo, scopeKind string) []string {
	switch scopeKind {
	case composeProjectScopeKind:
		return []string{item.ComposeProject, info.GroupValue}
	case composeServiceScopeKind:
		return []string{item.ComposeService, info.MemberValue}
	case swarmStackScopeKind, kubernetesNamespaceScopeKind:
		return []string{info.GroupValue}
	case swarmTaskScopeKind, kubernetesPodScopeKind:
		return []string{info.MemberValue}
	default:
		return nil
	}
}

// effectiveHealth 返回项目的有效健康状态，若未设定则默认为不可用。
func effectiveHealth(item Summary) string {
	if item.Health == "" {
		return containerHealthUnavailable
	}
	return item.Health
}

func parseLogSince(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if timestamp, err := time.Parse(time.RFC3339, value); err == nil {
		return timestamp.UTC().Format(time.RFC3339), nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration < 0 {
		return "", fmt.Errorf("invalid since value")
	}
	return strconv.FormatInt(time.Now().UTC().Add(-duration).Unix(), 10), nil
}

func (s *service) publishActionAudit(ctx context.Context, result ActionResult, options ActionOptions, err error) {
	if s == nil || s.auditBus == nil {
		return
	}
	action := "ops.container." + strings.TrimSpace(result.Action)
	messageKey := ""
	message := ""
	if err != nil {
		messageKey = messageKeyForError(err).String()
		message = fallbackMessageForError(err)
	}
	metadata := map[string]any{
		"container_id":   result.ID,
		"container_name": result.Name,
		"image":          result.Image,
		"action":         action,
		"runtime":        firstNonEmpty(result.Runtime, runtimeNameDocker),
		"endpoint":       safeEndpointLabel(s.runtimeOptions.endpoint),
		"force":          options.Force,
		"result":         auditResult(err),
		"error":          messageKey,
		"status_before":  result.StatusBefore,
		"status_after":   result.StatusAfter,
	}
	if requestAudit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		metadata["requestId"] = requestAudit.RequestID
		metadata["traceId"] = requestAudit.TraceID
	}
	event := moduleapi.AuditEvent{
		Kind:          moduleapi.AuditEventKindDomain,
		Operator:      currentAuditOperator(ctx),
		Action:        action,
		ResourceType:  containerResourceType,
		ResourceID:    firstNonEmpty(result.ID, result.Name),
		ResourceName:  result.Name,
		StatusCode:    auditStatusCode(err),
		Success:       err == nil,
		MessageKey:    messageKey,
		Message:       message,
		Metadata:      metadata,
		RequestMethod: "",
		RequestPath:   "",
	}
	if publishErr := s.auditBus.Publish(ctx, eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  s.moduleName,
		Payload: event,
	}); publishErr != nil && s.logger != nil {
		s.logger.Warn("publish container audit event failed",
			zap.String("module", s.moduleName),
			zap.String("action", action),
			zap.Error(publishErr),
		)
	}
}

func currentAuditOperator(ctx context.Context) *moduleapi.CurrentUser {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil {
		return nil
	}
	user := *requestAuth.User
	return &user
}

func auditResult(err error) string {
	if err != nil {
		return "failed"
	}
	return "success"
}

func auditStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	return statusForError(err)
}

type containerRuntimeOptions struct {
	enabled                 bool
	runtime                 string
	endpoint                string
	dangerousActionsEnabled bool
	defaultTail             int
	maxTail                 int
	environmentPolicy       containercontract.EnvironmentPolicy
	orchestratorPolicies    orchestratorActionPolicies
}

// ContainerOptionsFromConfig 从模块上下文中提取容器运行时配置选项，应用默认值、配置注册表中的设置，最后使用显式模块配置覆盖。
func containerOptionsFromConfig(ctx *module.Context) containerRuntimeOptions {
	options := containerRuntimeOptions{
		enabled:                 defaultContainerEnabled,
		runtime:                 defaultContainerRuntime,
		endpoint:                defaultContainerDockerEndpoint,
		dangerousActionsEnabled: defaultContainerDangerousActionsEnabled,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
		environmentPolicy:       defaultContainerEnvironmentPolicy,
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
	resolver, ok := s.systemConfig.(stringSystemConfigResolver)
	if !ok {
		return fallback
	}
	raw, err := resolver.ResolveDefaultConfig(ctx, key)
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

// It returns a disabled runtime if the container is disabled, a Docker runtime if enabled, or an error if the runtime type is unsupported.
func newContainerRuntime(options containerRuntimeOptions) (Runtime, error) {
	if !options.enabled {
		return disabledRuntime{}, nil
	}
	if strings.TrimSpace(options.runtime) != defaultContainerRuntime && strings.TrimSpace(options.runtime) != runtimeNameDocker {
		return nil, errUnsupportedContainerRuntime
	}
	return NewDockerRuntime(options.endpoint)
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
	runtime, err := s.runtimeFactory(options)
	if err != nil {
		return nil, err
	}
	s.runtime = runtime
	return runtime, nil
}

// ShellSessionRequest describes one requested interactive container shell session.
type ShellSessionRequest struct {
	Command string
	Cols    int
	Rows    int
}

// ShellSession contains the issued shell ticket and websocket bootstrap data.
type ShellSession struct {
	SessionID    string
	Command      string
	Cols         int
	Rows         int
	ExpiresAt    time.Time
	WebSocketURL string
}

// ShellHandshake contains the validated ticket payload used to open a terminal session.
type ShellHandshake struct {
	SessionID    string
	Command      string
	Cols         int
	Rows         int
	ResourceID   string
	ResourceName string
	UserID       uint64
}

// ShellSessionCloseSummary carries audit-safe shell session close details.
type ShellSessionCloseSummary struct {
	SessionID    string
	ResourceID   string
	ResourceName string
	Command      string
	UserID       uint64
}

type shellAuditPayload struct {
	action  string
	detail  Detail
	issued  *realtimeauth.IssuedTicket
	command string
	reason  string
	err     error
}

const (
	containerShellScope        = "container.shell"
	containerShellResourceType = "container"
)

func (s *service) IssueShellSession(ctx context.Context, ref Ref, request ShellSessionRequest) (ShellSession, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return ShellSession{}, err
	}
	if !s.shellAllowed(ctx) {
		return ShellSession{}, errShellDisabled
	}
	normalized, err := normalizeShellSessionRequest(request)
	if err != nil {
		return ShellSession{}, err
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		return ShellSession{}, err
	}
	detail, err := runtime.Detail(ctx, ref)
	if err != nil {
		return ShellSession{}, err
	}
	if strings.TrimSpace(strings.ToLower(detail.State)) != "running" {
		return ShellSession{}, errContainerNotRunning
	}
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil {
		return ShellSession{}, errShellForbidden
	}
	s.publishShellAudit(ctx, shellAuditPayload{
		action:  containercontract.ContainerAuditActionShellSessionRequested.String(),
		detail:  detail,
		command: normalized.Command,
	})
	issued, err := s.realtimeTickets.Issue(ctx, realtimeauth.IssueRequest{
		UserID:       requestAuth.User.ID,
		ResourceType: containerShellResourceType,
		ResourceID:   ref.Value,
		Scope:        containerShellScope,
		ClientIP:     currentRequestClientIP(ctx),
		UserAgent:    currentRequestUserAgent(ctx),
		Command:      normalized.Command,
		Cols:         normalized.Cols,
		Rows:         normalized.Rows,
		TTL:          containerOperationTTL,
	})
	if err != nil {
		s.publishShellAudit(ctx, shellAuditPayload{
			action:  containercontract.ContainerAuditActionShellTicketRejected.String(),
			detail:  detail,
			command: normalized.Command,
			reason:  "ticket_issue_failed",
			err:     errShellSessionFailed,
		})
		return ShellSession{}, errShellSessionFailed
	}
	s.publishShellAudit(ctx, shellAuditPayload{
		action:  containercontract.ContainerAuditActionShellTicketIssued.String(),
		detail:  detail,
		issued:  &issued,
		command: normalized.Command,
	})
	return ShellSession{
		SessionID:    issued.SessionID,
		Command:      issued.Command,
		Cols:         issued.Cols,
		Rows:         issued.Rows,
		ExpiresAt:    issued.ExpiresAt,
		WebSocketURL: buildShellWebSocketURL(ref, issued.Ticket),
	}, nil
}

func (s *service) ConsumeShellSessionTicket(ctx context.Context, ref Ref, ticket string, origin string) (ShellHandshake, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return ShellHandshake{}, err
	}
	if !s.shellAllowed(ctx) {
		return ShellHandshake{}, errShellDisabled
	}
	if err := realtimeauth.ValidateOrigin(origin, s.websocketAllowedOrigins); err != nil {
		return ShellHandshake{}, errShellOriginDenied
	}
	consumed, err := s.realtimeTickets.Consume(ctx, realtimeauth.ConsumeRequest{
		Ticket:       ticket,
		ResourceType: containerShellResourceType,
		ResourceID:   ref.Value,
		Scope:        containerShellScope,
	})
	if err != nil {
		return ShellHandshake{}, mapRealtimeTicketError(err)
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		return ShellHandshake{}, err
	}
	detail, err := runtime.Detail(ctx, ref)
	if err != nil {
		return ShellHandshake{}, err
	}
	if strings.TrimSpace(strings.ToLower(detail.State)) != "running" {
		s.publishShellAudit(ctx, shellAuditPayload{
			action:  containercontract.ContainerAuditActionShellTicketRejected.String(),
			detail:  detail,
			command: consumed.Command,
			reason:  "container_not_running",
			err:     errContainerNotRunning,
		})
		return ShellHandshake{}, errContainerNotRunning
	}
	s.publishShellAudit(ctx, shellAuditPayload{
		action:  containercontract.ContainerAuditActionShellSessionStarted.String(),
		detail:  detail,
		command: consumed.Command,
	})
	return ShellHandshake{
		SessionID:    consumed.SessionID,
		Command:      consumed.Command,
		Cols:         consumed.Cols,
		Rows:         consumed.Rows,
		ResourceID:   detail.ID,
		ResourceName: detail.Name,
		UserID:       consumed.UserID,
	}, nil
}

func (s *service) OpenShellTerminalSession(ctx context.Context, ref Ref, handshake ShellHandshake) (terminal.Session, error) {
	if s == nil {
		return nil, errShellSessionFailed
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		s.publishShellSessionFailed(ctx, handshake, "runtime_unavailable", err)
		return nil, err
	}
	session, err := runtime.Shell(ctx, ref, handshake.Command)
	if err != nil {
		s.publishShellSessionFailed(ctx, handshake, "session_open_failed", err)
		return nil, err
	}
	return session, nil
}

// normalizeShellSessionRequest 验证并规范化 Shell 会话请求。
// 确保命令为 sh、bash 或 ash 之一，且终端行列尺寸均为正数。
// 返回规范化后的请求或错误。
func normalizeShellSessionRequest(request ShellSessionRequest) (ShellSessionRequest, error) {
	command := strings.TrimSpace(strings.ToLower(request.Command))
	switch command {
	case "sh", "bash", "ash":
	default:
		return ShellSessionRequest{}, errShellCommandNotFound
	}
	if request.Cols <= 0 || request.Rows <= 0 {
		return ShellSessionRequest{}, errShellInvalidSize
	}
	return ShellSessionRequest{
		Command: command,
		Cols:    request.Cols,
		Rows:    request.Rows,
	}, nil
}

// buildShellWebSocketURL constructs a WebSocket URL for accessing the shell of a specified container.
func buildShellWebSocketURL(ref Ref, ticket string) string {
	values := url.Values{}
	values.Set("ticket", ticket)
	return "/api" + containercontract.ContainerAPIGroup + "/" + url.PathEscape(ref.Value) + "/shell/ws?" + values.Encode()
}

// mapRealtimeTicketError 将实时票证错误映射为对应的 Shell 特定错误。
// 若错误为未知类型，则返回会话失败错误。
func mapRealtimeTicketError(err error) error {
	switch {
	case errors.Is(err, realtimeauth.ErrExpiredTicket):
		return errShellTicketExpired
	case errors.Is(err, realtimeauth.ErrUsedTicket):
		return errShellTicketUsed
	case errors.Is(err, realtimeauth.ErrResourceMismatch), errors.Is(err, realtimeauth.ErrScopeMismatch), errors.Is(err, realtimeauth.ErrInvalidTicket), errors.Is(err, realtimeauth.ErrTicketRequired):
		return errShellTicketInvalid
	default:
		return errShellSessionFailed
	}
}

// CurrentRequestClientIP 从请求审计上下文中提取客户端 IP 地址。如果审计上下文不存在，返回空字符串。
func currentRequestClientIP(ctx context.Context) string {
	requestAudit, ok := httpx.RequestAuditContextFromContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(requestAudit.ClientIP)
}

// currentRequestUserAgent returns the User-Agent from the current request's audit context, or an empty string if unavailable.
func currentRequestUserAgent(ctx context.Context) string {
	requestAudit, ok := httpx.RequestAuditContextFromContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(requestAudit.UserAgent)
}

func detachedAuditContext(ctx context.Context) (context.Context, context.CancelFunc) {
	auditCtx, cancel := context.WithTimeout(context.Background(), containerAuditPublishTimeout)
	if requestAudit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		auditCtx = httpx.WithRequestAuditContext(auditCtx, requestAudit)
	}
	if requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx); ok {
		auditCtx = moduleapi.WithRequestAuthContext(auditCtx, requestAuth)
	}
	return auditCtx, cancel
}

func (s *service) publishShellSessionClosed(
	ctx context.Context,
	handshake ShellHandshake,
	startedAt time.Time,
	reason string,
	err error,
) {
	if s == nil || s.auditBus == nil {
		return
	}
	auditCtx, cancel := detachedAuditContext(ctx)
	defer cancel()
	duration := time.Since(startedAt)
	metadata := map[string]any{
		"container_id":   handshake.ResourceID,
		"container_name": handshake.ResourceName,
		"command":        handshake.Command,
		"result":         auditResult(err),
		"session_id":     handshake.SessionID,
		"duration_ms":    duration.Milliseconds(),
		"close_reason":   strings.TrimSpace(reason),
	}
	if requestAudit, ok := httpx.RequestAuditContextFromContext(auditCtx); ok {
		metadata["requestId"] = requestAudit.RequestID
		metadata["traceId"] = requestAudit.TraceID
		metadata["route"] = requestAudit.Route
		metadata["client_ip"] = requestAudit.ClientIP
		metadata["user_agent"] = requestAudit.UserAgent
	}
	user := currentAuditOperator(auditCtx)
	if user == nil && handshake.UserID != 0 {
		user = &moduleapi.CurrentUser{ID: handshake.UserID}
	}
	event := moduleapi.AuditEvent{
		Kind:         moduleapi.AuditEventKindDomain,
		Operator:     user,
		Action:       containercontract.ContainerAuditActionShellSessionClosed.String(),
		ResourceType: containerResourceType,
		ResourceID:   firstNonEmpty(handshake.ResourceID, handshake.ResourceName),
		ResourceName: handshake.ResourceName,
		StatusCode:   auditStatusCode(err),
		Success:      err == nil,
		Metadata:     metadata,
	}
	if err != nil {
		event.MessageKey = messageKeyForError(err).String()
		event.Message = fallbackMessageForError(err)
	}
	if publishErr := s.auditBus.Publish(auditCtx, eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  s.moduleName,
		Payload: event,
	}); publishErr != nil && s.logger != nil {
		s.logger.Warn("publish container shell close audit event failed",
			zap.String("module", s.moduleName),
			zap.String("action", containercontract.ContainerAuditActionShellSessionClosed.String()),
			zap.Error(publishErr),
		)
	}
}

func (s *service) publishShellSessionFailed(ctx context.Context, handshake ShellHandshake, reason string, err error) {
	if s == nil || s.auditBus == nil {
		return
	}
	auditCtx, cancel := detachedAuditContext(ctx)
	defer cancel()
	metadata := map[string]any{
		"container_id":   handshake.ResourceID,
		"container_name": handshake.ResourceName,
		"command":        handshake.Command,
		"result":         auditResult(err),
		"session_id":     handshake.SessionID,
		"reason":         strings.TrimSpace(reason),
	}
	if requestAudit, ok := httpx.RequestAuditContextFromContext(auditCtx); ok {
		metadata["requestId"] = requestAudit.RequestID
		metadata["traceId"] = requestAudit.TraceID
		metadata["route"] = requestAudit.Route
		metadata["client_ip"] = requestAudit.ClientIP
		metadata["user_agent"] = requestAudit.UserAgent
	}
	user := currentAuditOperator(auditCtx)
	if user == nil && handshake.UserID != 0 {
		user = &moduleapi.CurrentUser{ID: handshake.UserID}
	}
	event := moduleapi.AuditEvent{
		Kind:         moduleapi.AuditEventKindDomain,
		Operator:     user,
		Action:       containercontract.ContainerAuditActionShellSessionFailed.String(),
		ResourceType: containerResourceType,
		ResourceID:   firstNonEmpty(handshake.ResourceID, handshake.ResourceName),
		ResourceName: handshake.ResourceName,
		StatusCode:   auditStatusCode(err),
		Success:      false,
		Metadata:     metadata,
	}
	if err != nil {
		event.MessageKey = messageKeyForError(err).String()
		event.Message = fallbackMessageForError(err)
	}
	if publishErr := s.auditBus.Publish(auditCtx, eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  s.moduleName,
		Payload: event,
	}); publishErr != nil && s.logger != nil {
		s.logger.Warn("publish container shell failure audit event failed",
			zap.String("module", s.moduleName),
			zap.String("action", containercontract.ContainerAuditActionShellSessionFailed.String()),
			zap.Error(publishErr),
		)
	}
}

func (s *service) publishShellAudit(ctx context.Context, payload shellAuditPayload) {
	if s == nil || s.auditBus == nil {
		return
	}
	metadata := map[string]any{
		"container_id":   payload.detail.ID,
		"container_name": payload.detail.Name,
		"command":        strings.TrimSpace(payload.command),
		"result":         auditResult(payload.err),
	}
	if payload.reason != "" {
		metadata["reason"] = payload.reason
	}
	if payload.issued != nil {
		metadata["session_id"] = payload.issued.SessionID
		metadata["ticket_id"] = payload.issued.TicketID
		metadata["expires_at"] = payload.issued.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if requestAudit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		metadata["requestId"] = requestAudit.RequestID
		metadata["traceId"] = requestAudit.TraceID
		metadata["route"] = requestAudit.Route
		metadata["client_ip"] = requestAudit.ClientIP
		metadata["user_agent"] = requestAudit.UserAgent
	}
	event := moduleapi.AuditEvent{
		Kind:         moduleapi.AuditEventKindDomain,
		Operator:     currentAuditOperator(ctx),
		Action:       payload.action,
		ResourceType: containerResourceType,
		ResourceID:   firstNonEmpty(payload.detail.ID, payload.detail.Name),
		ResourceName: payload.detail.Name,
		StatusCode:   auditStatusCode(payload.err),
		Success:      payload.err == nil,
		Metadata:     metadata,
	}
	if payload.err != nil {
		event.MessageKey = messageKeyForError(payload.err).String()
		event.Message = fallbackMessageForError(payload.err)
	}
	if publishErr := s.auditBus.Publish(ctx, eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  s.moduleName,
		Payload: event,
	}); publishErr != nil && s.logger != nil {
		s.logger.Warn("publish container shell audit event failed",
			zap.String("module", s.moduleName),
			zap.String("action", payload.action),
			zap.Error(publishErr),
		)
	}
}

type disabledRuntime struct{}

func (disabledRuntime) Info(context.Context) (RuntimeInfo, error) {
	return RuntimeInfo{Runtime: runtimeNameDocker, Status: "disabled"}, errRuntimeDisabled
}
func (disabledRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	return nil, errRuntimeDisabled
}
func (disabledRuntime) Detail(context.Context, Ref) (Detail, error) {
	return Detail{}, errRuntimeDisabled
}
func (disabledRuntime) Mounts(context.Context, Ref) ([]Mount, error) {
	return nil, errRuntimeDisabled
}
func (disabledRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	return MountUsage{}, errRuntimeDisabled
}
func (disabledRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) {
	return Logs{}, errRuntimeDisabled
}
func (disabledRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) {
	return nil, errRuntimeDisabled
}
func (disabledRuntime) Start(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, errRuntimeDisabled
}
func (disabledRuntime) Stop(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, errRuntimeDisabled
}
func (disabledRuntime) Restart(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, errRuntimeDisabled
}
func (disabledRuntime) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	return ActionResult{}, errRuntimeDisabled
}
func (disabledRuntime) Close() error { return nil }

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
