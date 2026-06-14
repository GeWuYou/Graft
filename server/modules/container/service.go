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

type service struct {
	runtimeMu               sync.Mutex
	runtime                 Runtime
	runtimeOptions          containerRuntimeOptions
	runtimeFactory          func(containerRuntimeOptions) (Runtime, error)
	systemConfig            moduleapi.SystemConfigResolver
	auditBus                eventbus.Bus
	logger                  *zap.Logger
	moduleName              string
	enabled                 bool
	dangerousActionsEnabled bool
	defaultTail             int
	maxTail                 int
}

type containerServiceOptions struct {
	runtime                 Runtime
	runtimeOptions          containerRuntimeOptions
	runtimeFactory          func(containerRuntimeOptions) (Runtime, error)
	systemConfig            moduleapi.SystemConfigResolver
	auditBus                eventbus.Bus
	logger                  *zap.Logger
	moduleName              string
	enabled                 bool
	dangerousActionsEnabled bool
	defaultTail             int
	maxTail                 int
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
	})
}

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
	runtimeFactory := options.runtimeFactory
	if runtimeFactory == nil {
		runtimeFactory = newContainerRuntime
	}
	return &service{
		runtime:                 options.runtime,
		runtimeOptions:          runtimeOptions,
		runtimeFactory:          runtimeFactory,
		auditBus:                options.auditBus,
		logger:                  options.logger,
		moduleName:              firstNonEmpty(options.moduleName, moduleID),
		enabled:                 options.enabled,
		systemConfig:            options.systemConfig,
		dangerousActionsEnabled: options.dangerousActionsEnabled,
		defaultTail:             options.defaultTail,
		maxTail:                 options.maxTail,
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

func (s *service) List(ctx context.Context) (RuntimeInfo, []Summary, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return RuntimeInfo{}, nil, err
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		return RuntimeInfo{}, nil, err
	}
	info, err := runtime.Info(ctx)
	if err != nil {
		return RuntimeInfo{}, nil, err
	}
	items, err := runtime.List(ctx, ListQuery{})
	if err != nil {
		return RuntimeInfo{}, nil, err
	}
	return info, items, nil
}

func (s *service) Detail(ctx context.Context, ref Ref) (Detail, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return Detail{}, err
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		return Detail{}, err
	}
	return runtime.Detail(ctx, ref)
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
	return s.runAction(ctx, ref, containerActionStart)
}

func (s *service) Stop(ctx context.Context, ref Ref) (ActionResult, error) {
	return s.runAction(ctx, ref, containerActionStop)
}

func (s *service) Restart(ctx context.Context, ref Ref) (ActionResult, error) {
	return s.runAction(ctx, ref, containerActionRestart)
}

func (s *service) runAction(
	ctx context.Context,
	ref Ref,
	action string,
) (ActionResult, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return ActionResult{}, err
	}
	if !s.dangerousActionsAllowed(ctx) {
		result := ActionResult{ID: ref.Value, Action: action, Runtime: runtimeNameDocker}
		s.publishActionAudit(ctx, result, errDangerousActionsDisabled)
		return ActionResult{}, errDangerousActionsDisabled
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		return ActionResult{}, err
	}
	actionCtx, cancel := context.WithTimeout(ctx, containerOperationTTL)
	defer cancel()
	result, err := runWithRuntime(actionCtx, ref, action, runtime)
	if result.Action == "" {
		result.Action = action
	}
	s.publishActionAudit(ctx, result, err)
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

func runWithRuntime(ctx context.Context, ref Ref, action string, runtime Runtime) (ActionResult, error) {
	switch action {
	case containerActionStart:
		return runtime.Start(ctx, ref)
	case containerActionStop:
		return runtime.Stop(ctx, ref)
	default:
		return runtime.Restart(ctx, ref)
	}
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

func (s *service) publishActionAudit(ctx context.Context, result ActionResult, err error) {
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
}

func containerOptionsFromConfig(ctx *module.Context) containerRuntimeOptions {
	options := containerRuntimeOptions{
		enabled:                 defaultContainerEnabled,
		runtime:                 defaultContainerRuntime,
		endpoint:                defaultContainerDockerEndpoint,
		dangerousActionsEnabled: defaultContainerDangerousActionsEnabled,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
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
func (disabledRuntime) Close() error { return nil }

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
