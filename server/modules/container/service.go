// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
)

const (
	containerResourceType = "container"
	containerOperationTTL = 30 * time.Second
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
	defaultTail             int
	maxTail                 int
	environmentPolicy       containercontract.EnvironmentPolicy
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
	defaultTail             int
	maxTail                 int
	environmentPolicy       containercontract.EnvironmentPolicy
}

func newContainerService(ctx *module.Context, moduleName string) (*service, error) {
	options := containerOptionsFromConfig(ctx)
	runtime := Runtime(disabledRuntime{})
	return newService(containerServiceOptions{
		runtime:                 runtime,
		runtimeOptions:          options,
		systemConfig:            resolveSystemConfigResolver(ctx),
		auditBus:                ctx.EventBus,
		logger:                  ctx.Logger,
		moduleName:              moduleName,
		enabled:                 options.enabled,
		dangerousActionsEnabled: options.dangerousActionsEnabled,
		defaultTail:             options.defaultTail,
		maxTail:                 options.maxTail,
		environmentPolicy:       options.environmentPolicy,
	})
}

// newService creates a container service from the provided options, normalizing configuration values and applying defaults for optional components.
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
		defaultTail:             options.defaultTail,
		maxTail:                 options.maxTail,
		environmentPolicy:       environmentPolicy,
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
	paged = applyActionAvailability(paged, s.dangerousActionsAllowed(ctx))
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
	adjusted := applyActionAvailability([]Summary{detail.Summary}, s.dangerousActionsAllowed(ctx))
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
	if !s.dangerousActionsAllowed(ctx) {
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
	if !s.dangerousActionsAllowed(ctx) {
		result := ActionResult{ID: ref.Value, Action: action, Runtime: runtimeNameDocker}
		s.publishActionAudit(ctx, result, options, errDangerousActionsDisabled)
		return ActionResult{}, errDangerousActionsDisabled
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		return ActionResult{}, err
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
// policy determines visibility: Hidden clears values, Plain keeps them visible, and the
// default policy hides sensitive variables while optionally preserving the original
// value in CopyValue. Returns nil if the input is empty.
func applyEnvironmentPolicy(environment []EnvironmentVariable, options environmentPolicyOptions) []EnvironmentVariable {
	if len(environment) == 0 {
		return nil
	}
	mapped := make([]EnvironmentVariable, 0, len(environment))
	for _, item := range environment {
		item.Sensitive = item.Sensitive || isSensitiveEnvironmentKey(item.Key)
		item.CopyValue = nil
		switch options.policy {
		case containercontract.ContainerEnvironmentPolicyHidden:
			item.Value = ""
			item.Masked = true
		case containercontract.ContainerEnvironmentPolicyPlain:
			item.Masked = false
		default:
			if item.Sensitive {
				if options.maskedCopyEnabled {
					copyValue := item.Value
					item.CopyValue = &copyValue
				}
				item.Value = ""
				item.Masked = true
			} else {
				item.Masked = false
			}
		}
		mapped = append(mapped, item)
	}
	return mapped
}

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

func filterContainerSummaries(items []Summary, query ListQuery) []Summary {
	filtered := make([]Summary, 0, len(items))
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	for _, item := range items {
		if query.State != "" && item.State != query.State {
			continue
		}
		if query.Health != "" && effectiveHealth(item) != query.Health {
			continue
		}
		if keyword != "" && !summaryMatchesKeyword(item, keyword) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

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

func applyActionAvailability(items []Summary, dangerousAllowed bool) []Summary {
	adjusted := make([]Summary, 0, len(items))
	for _, item := range items {
		item.CanRemove = canRemoveState(item.State)
		if !dangerousAllowed {
			item.CanStart = false
			item.CanStop = false
			item.CanRestart = false
			item.CanRemove = false
		}
		adjusted = append(adjusted, item)
	}
	return adjusted
}

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
}

func containerOptionsFromConfig(ctx *module.Context) containerRuntimeOptions {
	options := containerRuntimeOptions{
		enabled:                 defaultContainerEnabled,
		runtime:                 defaultContainerRuntime,
		endpoint:                defaultContainerDockerEndpoint,
		dangerousActionsEnabled: defaultContainerDangerousActionsEnabled,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
		environmentPolicy:       defaultContainerEnvironmentPolicy,
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
