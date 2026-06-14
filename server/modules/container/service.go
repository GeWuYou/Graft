// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

const (
	containerResourceType = "container"
	containerOperationTTL = 30 * time.Second
)

type service struct {
	runtime                 Runtime
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
	runtime, err := newContainerRuntime(options)
	if err != nil {
		return nil, err
	}
	return newService(containerServiceOptions{
		runtime:                 runtime,
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
	return &service{
		runtime:                 options.runtime,
		auditBus:                options.auditBus,
		logger:                  options.logger,
		moduleName:              firstNonEmpty(options.moduleName, moduleID),
		enabled:                 options.enabled,
		dangerousActionsEnabled: options.dangerousActionsEnabled,
		defaultTail:             options.defaultTail,
		maxTail:                 options.maxTail,
	}, nil
}

func (s *service) Close() error {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.Close()
}

func (s *service) List(ctx context.Context) (RuntimeInfo, []Summary, error) {
	if err := s.requireEnabled(); err != nil {
		return RuntimeInfo{}, nil, err
	}
	info, err := s.runtime.Info(ctx)
	if err != nil {
		return RuntimeInfo{}, nil, err
	}
	items, err := s.runtime.List(ctx, ListQuery{})
	if err != nil {
		return RuntimeInfo{}, nil, err
	}
	return info, items, nil
}

func (s *service) Detail(ctx context.Context, ref Ref) (Detail, error) {
	if err := s.requireEnabled(); err != nil {
		return Detail{}, err
	}
	return s.runtime.Detail(ctx, ref)
}

func (s *service) Logs(ctx context.Context, ref Ref, query LogQuery) (Logs, error) {
	if err := s.requireEnabled(); err != nil {
		return Logs{}, err
	}
	normalized, err := s.normalizeLogQuery(query)
	if err != nil {
		return Logs{}, err
	}
	return s.runtime.Logs(ctx, ref, normalized)
}

func (s *service) Start(ctx context.Context, ref Ref) (ActionResult, error) {
	return s.runAction(ctx, ref, containerActionStart, s.runtime.Start)
}

func (s *service) Stop(ctx context.Context, ref Ref) (ActionResult, error) {
	return s.runAction(ctx, ref, containerActionStop, s.runtime.Stop)
}

func (s *service) Restart(ctx context.Context, ref Ref) (ActionResult, error) {
	return s.runAction(ctx, ref, containerActionRestart, s.runtime.Restart)
}

func (s *service) runAction(
	ctx context.Context,
	ref Ref,
	action string,
	run func(context.Context, Ref) (ActionResult, error),
) (ActionResult, error) {
	if err := s.requireEnabled(); err != nil {
		return ActionResult{}, err
	}
	if !s.dangerousActionsEnabled {
		result := ActionResult{ID: ref.Value, Action: action, Runtime: runtimeNameDocker}
		s.publishActionAudit(ctx, result, errDangerousActionsDisabled)
		return ActionResult{}, errDangerousActionsDisabled
	}
	actionCtx, cancel := context.WithTimeout(ctx, containerOperationTTL)
	defer cancel()
	result, err := run(actionCtx, ref)
	if result.Action == "" {
		result.Action = action
	}
	s.publishActionAudit(ctx, result, err)
	if err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

func (s *service) requireEnabled() error {
	if s == nil || !s.enabled || s.runtime == nil {
		return errRuntimeDisabled
	}
	return nil
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
			return LogQuery{}, errInvalidRef
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
	if ctx == nil || ctx.Config == nil {
		return options
	}
	return options
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
