package dashboard

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/config"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/logger"
	"graft/server/internal/moduleapi"
)

const (
	operationWidgetLoad      = "dashboard_widget_load"
	errorCodeLoadFailed      = "DASHBOARD_WIDGET_LOAD_FAILED"
	errorCodePanic           = "DASHBOARD_WIDGET_PANIC"
	errorCodeTimeout         = "DASHBOARD_WIDGET_TIMEOUT"
	defaultWidgetActionKey   = "dashboard.actions.details"
	defaultWidgetActionLabel = "View details"
)

// ModuleRuntimeSummaryProvider returns the current module runtime summary.
type ModuleRuntimeSummaryProvider func() generated.ModuleRuntimeSummary

// Service aggregates fixed system summary and visible widget payloads.
type Service struct {
	config               *config.Config
	registry             *Registry
	authorizer           moduleapi.Authorizer
	logger               logger.AppLogger
	moduleRuntimeSummary ModuleRuntimeSummaryProvider
}

// ServiceOptions contains dependencies for dashboard aggregation.
type ServiceOptions struct {
	Config               *config.Config
	Registry             *Registry
	Authorizer           moduleapi.Authorizer
	Logger               logger.AppLogger
	ModuleRuntimeSummary ModuleRuntimeSummaryProvider
}

// NewService creates a dashboard aggregation service.
func NewService(options ServiceOptions) *Service {
	appLogger := options.Logger
	if appLogger == nil {
		appLogger = logger.NewAppLogger(zap.NewNop())
	}
	registry := options.Registry
	if registry == nil {
		registry = NewRegistry()
	}

	return &Service{
		config:               options.Config,
		registry:             registry,
		authorizer:           options.Authorizer,
		logger:               appLogger.Named("internal.dashboard"),
		moduleRuntimeSummary: options.ModuleRuntimeSummary,
	}
}

// Summary returns the dashboard system summary and all visible contributions.
func (s *Service) Summary(ctx context.Context, requestAuth moduleapi.RequestAuthContext) generated.DashboardSummaryResponse {
	quickLinks := s.visibleQuickLinks(ctx, requestAuth, s.registry.QuickLinks())
	widgets := s.visibleWidgets(ctx, requestAuth, s.registry.Items())
	return generated.DashboardSummaryResponse{
		QuickLinks:    quickLinks,
		SystemSummary: s.systemSummary(requestAuth, widgets),
		Widgets:       widgets,
	}
}

// Widget returns one visible widget by id.
func (s *Service) Widget(ctx context.Context, requestAuth moduleapi.RequestAuthContext, id string) (generated.DashboardWidget, bool) {
	definition, ok := s.registry.Get(id)
	if !ok || !s.canReadWidget(ctx, requestAuth, definition) {
		return generated.DashboardWidget{}, false
	}
	widget := s.loadWidget(ctx, requestAuth, definition)
	if !widget.Visible {
		return generated.DashboardWidget{}, false
	}
	return widget, true
}

func (s *Service) visibleQuickLinks(
	ctx context.Context,
	requestAuth moduleapi.RequestAuthContext,
	definitions []QuickLinkDefinition,
) []generated.DashboardQuickLink {
	quickLinks := make([]generated.DashboardQuickLink, 0, len(definitions))
	for _, definition := range definitions {
		if !s.canReadPermissions(ctx, requestAuth, definition.RequiredPermissions) {
			continue
		}
		quickLinks = append(quickLinks, quickLinkFromDefinition(definition))
	}
	return quickLinks
}

func (s *Service) visibleWidgets(
	ctx context.Context,
	requestAuth moduleapi.RequestAuthContext,
	definitions []WidgetDefinition,
) []generated.DashboardWidget {
	widgets := make([]generated.DashboardWidget, 0, len(definitions))
	for _, definition := range definitions {
		if !s.canReadWidget(ctx, requestAuth, definition) {
			continue
		}
		widget := s.loadWidget(ctx, requestAuth, definition)
		if !widget.Visible {
			continue
		}
		widgets = append(widgets, widget)
	}
	sortLoadedWidgets(widgets)
	return widgets
}

func (s *Service) canReadWidget(
	ctx context.Context,
	requestAuth moduleapi.RequestAuthContext,
	definition WidgetDefinition,
) bool {
	return s.canReadPermissions(ctx, requestAuth, definition.RequiredPermissions)
}

func (s *Service) canReadPermissions(
	ctx context.Context,
	requestAuth moduleapi.RequestAuthContext,
	requiredPermissions []string,
) bool {
	if len(requiredPermissions) == 0 {
		return true
	}
	if s.authorizer == nil {
		return false
	}
	for _, permission := range requiredPermissions {
		if err := s.authorizer.Authorize(ctx, requestAuth, permission); err != nil {
			return false
		}
	}
	return true
}

func (s *Service) loadWidget(
	ctx context.Context,
	requestAuth moduleapi.RequestAuthContext,
	definition WidgetDefinition,
) generated.DashboardWidget {
	started := time.Now()
	timeout := definition.LoaderTimeout
	if timeout == 0 {
		timeout = defaultLoaderTimeout
	}

	loadCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payload, loadError := s.safeLoad(loadCtx, requestAuth, definition)
	duration := time.Since(started)
	if loadError != nil {
		s.logLoadError(ctx, definition, duration, loadError)
		return widgetFromDefinition(definition, nil, WidgetStatusError, widgetErrorFromError(loadError))
	}

	s.logLoadSuccess(ctx, definition, duration)
	return widgetFromDefinition(definition, payload, WidgetStatusNormal, nil)
}

func (s *Service) safeLoad(
	ctx context.Context,
	requestAuth moduleapi.RequestAuthContext,
	definition WidgetDefinition,
) (WidgetPayload, error) {
	resultCh := make(chan loadResult, 1)
	go func() {
		payload, err := invokeLoader(ctx, requestAuth, definition)
		select {
		case resultCh <- loadResult{payload: payload, err: err}:
		case <-ctx.Done():
		}
	}()

	select {
	case result := <-resultCh:
		return result.payload, result.err
	case <-ctx.Done():
		return nil, widgetLoadContextError(ctx.Err())
	}
}

func invokeLoader(
	ctx context.Context,
	requestAuth moduleapi.RequestAuthContext,
	definition WidgetDefinition,
) (payload WidgetPayload, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = widgetLoadError{
				code:    errorCodePanic,
				message: fmt.Sprintf("dashboard widget loader panic: %v", recovered),
				panic:   true,
			}
		}
	}()

	payload, err = definition.Loader.Load(ctx, WidgetRequest{
		WidgetID:    definition.ID,
		ModuleKey:   definition.ModuleKey,
		Type:        definition.Type,
		RequestAuth: requestAuth,
	})
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, widgetLoadError{code: errorCodeTimeout, message: err.Error(), timeout: true}
		}
		return nil, widgetLoadError{code: errorCodeLoadFailed, message: err.Error()}
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, widgetLoadError{code: errorCodeTimeout, message: context.DeadlineExceeded.Error(), timeout: true}
	}
	if payload == nil {
		payload = WidgetPayload{}
	}
	return payload, nil
}

type loadResult struct {
	payload WidgetPayload
	err     error
}

const (
	priorityWeightCritical = 0
	priorityWeightWarning  = 1
	priorityWeightNormal   = 2
	priorityWeightInfo     = 3
)

func (s *Service) systemSummary(
	requestAuth moduleapi.RequestAuthContext,
	widgets []generated.DashboardWidget,
) generated.DashboardSystemSummary {
	var user generated.DashboardCurrentUserSummary
	if requestAuth.User != nil {
		user = generated.DashboardCurrentUserSummary{
			DisplayName: requestAuth.User.DisplayName,
			Username:    requestAuth.User.Username,
		}
	}

	appEnv := ""
	defaultLocale := ""
	fallbackLocale := ""
	if s.config != nil {
		appEnv = strings.TrimSpace(s.config.App.Env)
		defaultLocale = strings.TrimSpace(s.config.I18n.DefaultLocale)
		fallbackLocale = strings.TrimSpace(s.config.I18n.FallbackLocale)
	}

	moduleSummary := generated.DashboardModuleSummary{}
	if s.moduleRuntimeSummary != nil {
		summary := s.moduleRuntimeSummary()
		moduleSummary = generated.DashboardModuleSummary{
			DegradedModules: summary.DegradedModules,
			EnabledModules:  summary.EnabledModules,
			TotalModules:    summary.TotalModules,
		}
	}

	return generated.DashboardSystemSummary{
		AbnormalServices: summaryAbnormalServices(widgets),
		AppEnv:           appEnv,
		CurrentUser:      user,
		FailedTasks: summaryMetric(widgets, func(widget generated.DashboardWidget) int {
			return intMetricValue(widget.Payload["failed_tasks"])
		}),
		HighRiskEvents: summaryMetric(widgets, func(widget generated.DashboardWidget) int {
			return intMetricValue(widget.Payload["high_risk_events"])
		}),
		Locale: generated.DashboardLocaleSummary{
			DefaultLocale:  defaultLocale,
			FallbackLocale: fallbackLocale,
		},
		Modules:        moduleSummary,
		VisibleWidgets: len(widgets),
	}
}

func widgetFromDefinition(
	definition WidgetDefinition,
	payload WidgetPayload,
	status WidgetStatus,
	widgetError *generated.DashboardWidgetError,
) generated.DashboardWidget {
	metadata := payload.Metadata()
	publicPayload := payload.PublicPayload()
	visible := widgetVisible(status, metadata)
	state := widgetState(status, metadata)
	priority := widgetPriority(definition.Priority, state, metadata)
	widget := baseGeneratedWidget(definition, publicPayload, status, visible, state, priority)
	applyWidgetText(&widget, definition)
	applyWidgetRuntimeFields(&widget, definition)
	if widgetError != nil {
		widget.Error = widgetError
	}
	return widget
}

func baseGeneratedWidget(
	definition WidgetDefinition,
	payload WidgetPayload,
	status WidgetStatus,
	visible bool,
	state WidgetState,
	priority WidgetPriority,
) generated.DashboardWidget {
	return generated.DashboardWidget{
		Category:  generated.DashboardWidgetCategory(definition.Category),
		Id:        definition.ID,
		ModuleKey: definition.ModuleKey,
		Order:     definition.Order,
		Payload:   payloadMap(payload),
		Priority:  generated.DashboardWidgetPriority(priority),
		Size:      generated.DashboardWidgetSize(definition.Size),
		State:     generated.DashboardWidgetState(state),
		Status:    ptr(generated.DashboardWidgetStatus(status)),
		Type:      generated.DashboardWidgetType(definition.Type),
		Visible:   visible,
	}
}

func applyWidgetText(widget *generated.DashboardWidget, definition WidgetDefinition) {
	if widget == nil {
		return
	}
	if len(definition.RequiredPermissions) > 0 {
		widget.RequiredPermissions = ptr(append([]string(nil), definition.RequiredPermissions...))
	}
	if definition.TitleKey != "" {
		widget.TitleKey = &definition.TitleKey
	}
	if definition.Title != "" {
		widget.Title = &definition.Title
	}
	if definition.DescriptionKey != "" {
		widget.DescriptionKey = &definition.DescriptionKey
	}
	if definition.Description != "" {
		widget.Description = &definition.Description
	}
}

func applyWidgetRuntimeFields(widget *generated.DashboardWidget, definition WidgetDefinition) {
	if widget == nil {
		return
	}
	if definition.RefreshInterval > 0 {
		seconds := int(definition.RefreshInterval / time.Second)
		widget.RefreshIntervalSeconds = &seconds
	}
	if definition.RouteLocation != "" {
		widget.RouteLocation = &definition.RouteLocation
	}
	if definition.Action.Route != "" {
		labelKey := definition.Action.LabelKey
		if labelKey == "" {
			labelKey = defaultWidgetActionKey
		}
		label := definition.Action.Label
		if label == "" {
			label = defaultWidgetActionLabel
		}
		widget.Action = &generated.DashboardWidgetAction{
			LabelKey: labelKey,
			Label:    label,
			Route:    definition.Action.Route,
		}
	}
}

func widgetVisible(status WidgetStatus, metadata WidgetPayloadMetadata) bool {
	if metadata.State == WidgetStateHidden {
		return false
	}
	if metadata.Visible != nil {
		return *metadata.Visible
	}
	return status != WidgetStatusDisabled
}

func widgetState(status WidgetStatus, metadata WidgetPayloadMetadata) WidgetState {
	if metadata.State != "" {
		return metadata.State
	}
	switch status {
	case WidgetStatusError:
		return WidgetStateCritical
	case WidgetStatusWarning:
		return WidgetStateWarning
	default:
		return WidgetStateNormal
	}
}

func widgetPriority(base WidgetPriority, state WidgetState, metadata WidgetPayloadMetadata) WidgetPriority {
	if metadata.PriorityOverride != "" {
		return metadata.PriorityOverride
	}
	switch state {
	case WidgetStateCritical:
		return WidgetPriorityCritical
	case WidgetStateWarning:
		if priorityWeight(base) > priorityWeight(WidgetPriorityWarning) {
			return WidgetPriorityWarning
		}
	}
	return base
}

func sortLoadedWidgets(widgets []generated.DashboardWidget) {
	slices.SortStableFunc(widgets, func(left, right generated.DashboardWidget) int {
		if left.Priority != right.Priority {
			return priorityWeight(WidgetPriority(left.Priority)) - priorityWeight(WidgetPriority(right.Priority))
		}
		if left.Order != right.Order {
			return left.Order - right.Order
		}
		return strings.Compare(left.Id, right.Id)
	})
}

func priorityWeight(priority WidgetPriority) int {
	switch priority {
	case WidgetPriorityCritical:
		return priorityWeightCritical
	case WidgetPriorityWarning:
		return priorityWeightWarning
	case WidgetPriorityNormal:
		return priorityWeightNormal
	case WidgetPriorityInfo:
		return priorityWeightInfo
	default:
		return priorityWeightNormal
	}
}

func summaryMetric(widgets []generated.DashboardWidget, metric func(generated.DashboardWidget) int) int {
	total := 0
	for _, widget := range widgets {
		total += metric(widget)
	}
	return total
}

func summaryAbnormalServices(widgets []generated.DashboardWidget) int {
	total := 0
	for _, widget := range widgets {
		total += intMetricValue(widget.Payload["abnormal_services"])
	}
	return total
}

func quickLinkFromDefinition(definition QuickLinkDefinition) generated.DashboardQuickLink {
	quickLink := generated.DashboardQuickLink{
		Id:            definition.ID,
		ModuleKey:     definition.ModuleKey,
		Order:         definition.Order,
		RouteLocation: definition.RouteLocation,
	}
	if definition.TitleKey != "" {
		quickLink.TitleKey = &definition.TitleKey
	}
	if definition.Title != "" {
		quickLink.Title = &definition.Title
	}
	if definition.DescriptionKey != "" {
		quickLink.DescriptionKey = &definition.DescriptionKey
	}
	if definition.Description != "" {
		quickLink.Description = &definition.Description
	}
	if definition.Icon != "" {
		quickLink.Icon = &definition.Icon
	}
	if len(definition.RequiredPermissions) > 0 {
		quickLink.RequiredPermissions = ptr(append([]string(nil), definition.RequiredPermissions...))
	}
	return quickLink
}

func payloadMap(payload WidgetPayload) map[string]interface{} {
	if payload == nil {
		return map[string]interface{}{}
	}
	result := make(map[string]interface{}, len(payload))
	for key, value := range payload {
		result[key] = value
	}
	return result
}

func widgetErrorFromError(err error) *generated.DashboardWidgetError {
	loadErr := widgetLoadError{}
	if errors.As(err, &loadErr) {
		return &generated.DashboardWidgetError{
			Code:    loadErr.code,
			Message: ptr(loadErr.message),
		}
	}
	return &generated.DashboardWidgetError{
		Code:    errorCodeLoadFailed,
		Message: ptr(err.Error()),
	}
}

func widgetLoadContextError(err error) widgetLoadError {
	if errors.Is(err, context.DeadlineExceeded) {
		return widgetLoadError{code: errorCodeTimeout, message: context.DeadlineExceeded.Error(), timeout: true}
	}
	if err != nil {
		return widgetLoadError{code: errorCodeLoadFailed, message: err.Error()}
	}
	return widgetLoadError{code: errorCodeLoadFailed, message: context.Canceled.Error()}
}

func (s *Service) logLoadSuccess(ctx context.Context, definition WidgetDefinition, duration time.Duration) {
	s.logger.Debug(ctx, "dashboard widget loaded",
		logger.StringField(logger.FieldOperation, operationWidgetLoad),
		logger.StringField("widget_id", definition.ID),
		logger.StringField("module_key", definition.ModuleKey),
		logger.StringField("widget_type", string(definition.Type)),
		logger.Int64Field("duration_ms", duration.Milliseconds()),
	)
}

func (s *Service) logLoadError(
	ctx context.Context,
	definition WidgetDefinition,
	duration time.Duration,
	err error,
) {
	loadErr := widgetLoadError{}
	_ = errors.As(err, &loadErr)

	fields := []logger.Field{
		logger.StringField(logger.FieldOperation, operationWidgetLoad),
		logger.StringField("widget_id", definition.ID),
		logger.StringField("module_key", definition.ModuleKey),
		logger.StringField("widget_type", string(definition.Type)),
		logger.Int64Field("duration_ms", duration.Milliseconds()),
		logger.BoolField("timeout", loadErr.timeout),
		logger.BoolField("panic", loadErr.panic),
		logger.ErrorField(err),
	}
	if requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx); ok && requestAuth.User != nil {
		fields = append(fields, logger.Uint64Field("user_id", requestAuth.User.ID))
	}
	if loadErr.panic {
		fields = append(fields, logger.StringField("stack", string(debug.Stack())))
	}

	s.logger.Error(ctx, "dashboard widget load failed", fields...)
}

type widgetLoadError struct {
	code    string
	message string
	timeout bool
	panic   bool
}

func (e widgetLoadError) Error() string {
	if e.message != "" {
		return e.message
	}
	return e.code
}

func ptr[T any](value T) *T {
	return &value
}

// RequestAuthFromContext returns the current request auth context or an empty value.
func RequestAuthFromContext(ctx context.Context) moduleapi.RequestAuthContext {
	requestAuth, _ := moduleapi.RequestAuthContextFromContext(ctx)
	return requestAuth
}
