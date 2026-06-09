// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package dashboard owns the MVP dashboard contribution registry and aggregate
// routes. It must stay limited to runtime contributions; dashboard persistence
// and user preferences belong in a future module.
package dashboard

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
)

const (
	defaultLoaderTimeout = 2 * time.Second
	moduleKeyCore        = "core"
)

// WidgetType is the stable dashboard renderer discriminator.
type WidgetType string

const (
	// WidgetTypeStatGroup renders multiple compact statistics.
	WidgetTypeStatGroup WidgetType = "stat-group"
	// WidgetTypeAlertList renders actionable alert rows.
	WidgetTypeAlertList WidgetType = "alert-list"
	// WidgetTypeLinkList renders navigation links.
	WidgetTypeLinkList WidgetType = "link-list"
	// WidgetTypeTimeline renders chronological events.
	WidgetTypeTimeline WidgetType = "timeline"
	// WidgetTypeHealth renders health summary and health rows.
	WidgetTypeHealth WidgetType = "health"
)

// WidgetSize describes the dashboard grid span requested by one contribution.
type WidgetSize string

const (
	// WidgetSizeSmall requests one compact dashboard card.
	WidgetSizeSmall WidgetSize = "small"
	// WidgetSizeMedium requests one two-column dashboard card.
	WidgetSizeMedium WidgetSize = "medium"
	// WidgetSizeLarge requests one large list dashboard card.
	WidgetSizeLarge WidgetSize = "large"
)

// WidgetCategory groups dashboard contributions for framework-rendered sections.
type WidgetCategory string

const (
	// WidgetCategorySystem groups platform and module runtime widgets.
	WidgetCategorySystem WidgetCategory = "system"
	// WidgetCategorySecurity groups audit, authorization, and risk widgets.
	WidgetCategorySecurity WidgetCategory = "security"
	// WidgetCategoryOperation groups operational attention widgets.
	WidgetCategoryOperation WidgetCategory = "operation"
	// WidgetCategoryBusiness groups future business-domain widgets.
	WidgetCategoryBusiness WidgetCategory = "business"
)

// WidgetPriority describes framework ordering before stable order/id tie-breakers.
type WidgetPriority string

const (
	// WidgetPriorityCritical moves critical risk, failure, or outage widgets first.
	WidgetPriorityCritical WidgetPriority = "critical"
	// WidgetPriorityWarning moves degraded widgets after critical ones.
	WidgetPriorityWarning WidgetPriority = "warning"
	// WidgetPriorityNormal is the default contribution priority.
	WidgetPriorityNormal WidgetPriority = "normal"
	// WidgetPriorityInfo is for informational widgets that should trail attention items.
	WidgetPriorityInfo WidgetPriority = "info"
)

// WidgetStatus describes the aggregate load result for one widget.
type WidgetStatus string

const (
	// WidgetStatusNormal indicates the widget loaded successfully.
	WidgetStatusNormal WidgetStatus = "normal"
	// WidgetStatusWarning indicates the widget loaded with degraded state.
	WidgetStatusWarning WidgetStatus = "warning"
	// WidgetStatusError indicates the widget loader failed.
	WidgetStatusError WidgetStatus = "error"
	// WidgetStatusDisabled indicates the widget is intentionally disabled.
	WidgetStatusDisabled WidgetStatus = "disabled"
)

// WidgetState describes whether and how a loaded widget should be shown.
type WidgetState string

const (
	// WidgetStateHidden omits a widget from the dashboard summary response.
	WidgetStateHidden WidgetState = "hidden"
	// WidgetStateNormal shows a normal widget.
	WidgetStateNormal WidgetState = "normal"
	// WidgetStateWarning shows an attention widget.
	WidgetStateWarning WidgetState = "warning"
	// WidgetStateCritical shows a critical attention widget.
	WidgetStateCritical WidgetState = "critical"
)

// HealthStatus is the stable status vocabulary for health widget payloads.
type HealthStatus string

const (
	// HealthStatusHealthy indicates a healthy item.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusDegraded indicates a degraded item.
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusDisabled indicates a disabled item.
	HealthStatusDisabled HealthStatus = "disabled"
	// HealthStatusUnknown indicates missing health evidence.
	HealthStatusUnknown HealthStatus = "unknown"
)

// QuickLinkDefinition is the module-declared dashboard quick-entry contract.
type QuickLinkDefinition struct {
	ID                  string
	ModuleKey           string
	TitleKey            string
	Title               string
	DescriptionKey      string
	Description         string
	Icon                string
	RouteLocation       string
	RequiredPermissions []string
	Order               int
}

// WidgetDefinition is the module-declared dashboard insight contribution contract.
type WidgetDefinition struct {
	ID                  string
	ModuleKey           string
	TitleKey            string
	Title               string
	DescriptionKey      string
	Description         string
	Type                WidgetType
	Size                WidgetSize
	Category            WidgetCategory
	Priority            WidgetPriority
	Order               int
	RefreshInterval     time.Duration
	RouteLocation       string
	Action              WidgetAction
	RequiredPermissions []string
	LoaderTimeout       time.Duration
	Loader              WidgetLoader
}

// WidgetAction is the framework-rendered card action contract.
type WidgetAction struct {
	LabelKey string
	Label    string
	Route    string
}

// WidgetPayloadMetadata lets a loader describe framework state without custom card markup.
type WidgetPayloadMetadata struct {
	Visible          *bool
	State            WidgetState
	PriorityOverride WidgetPriority
	Summary          WidgetSummaryMetrics
}

// WidgetSummaryMetrics contributes to the fixed dashboard summary header.
type WidgetSummaryMetrics struct {
	FailedTasks      int
	HighRiskEvents   int
	AbnormalServices int
}

// WidgetRequest describes one request-scoped widget load invocation.
type WidgetRequest struct {
	WidgetID    string
	ModuleKey   string
	Type        WidgetType
	RequestAuth moduleapi.RequestAuthContext
}

// WidgetLoader loads one widget payload for the current request. Implementations
// must observe the context and return promptly when it is canceled or reaches
// its deadline so dashboard requests cannot retain loader goroutines.
type WidgetLoader interface {
	Load(context.Context, WidgetRequest) (WidgetPayload, error)
}

// WidgetLoaderFunc adapts a function into a WidgetLoader.
type WidgetLoaderFunc func(context.Context, WidgetRequest) (WidgetPayload, error)

// Load invokes f.
func (f WidgetLoaderFunc) Load(ctx context.Context, req WidgetRequest) (WidgetPayload, error) {
	if f == nil {
		return WidgetPayload{}, nil
	}
	return f(ctx, req)
}

// WidgetPayload is intentionally a plain object for OpenAPI generation
// stability. Concrete payload schemas are still documented in OpenAPI.
type WidgetPayload map[string]any

// Metadata returns framework control metadata declared by a widget loader.
func (p WidgetPayload) Metadata() WidgetPayloadMetadata {
	metadata := WidgetPayloadMetadata{}
	if p == nil {
		return metadata
	}
	if value, ok := boolValue(p["visible"]); ok {
		metadata.Visible = &value
	}
	if state, ok := stateValue(p["state"]); ok {
		metadata.State = state
	}
	if priority, ok := priorityValue(p["priority"]); ok {
		metadata.PriorityOverride = priority
	}
	metadata.Summary = WidgetSummaryMetrics{
		FailedTasks:      intMetricValue(p["failed_tasks"]),
		HighRiskEvents:   intMetricValue(p["high_risk_events"]),
		AbnormalServices: intMetricValue(p["abnormal_services"]),
	}
	return metadata
}

// PublicPayload removes framework-only metadata from the widget payload body.
func (p WidgetPayload) PublicPayload() WidgetPayload {
	if p == nil {
		return WidgetPayload{}
	}
	result := make(WidgetPayload, len(p))
	for key, value := range p {
		switch key {
		case "visible", "state", "priority":
			continue
		default:
			result[key] = value
		}
	}
	return result
}

func boolValue(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		normalized := strings.TrimSpace(strings.ToLower(typed))
		if normalized == "true" {
			return true, true
		}
		if normalized == "false" {
			return false, true
		}
	}
	return false, false
}

func stateValue(value any) (WidgetState, bool) {
	state, ok := stringValue(value)
	if !ok {
		return "", false
	}
	switch WidgetState(state) {
	case WidgetStateHidden, WidgetStateNormal, WidgetStateWarning, WidgetStateCritical:
		return WidgetState(state), true
	default:
		return "", false
	}
}

func priorityValue(value any) (WidgetPriority, bool) {
	priority, ok := stringValue(value)
	if !ok {
		return "", false
	}
	switch WidgetPriority(priority) {
	case WidgetPriorityCritical, WidgetPriorityWarning, WidgetPriorityNormal, WidgetPriorityInfo:
		return WidgetPriority(priority), true
	default:
		return "", false
	}
}

func stringValue(value any) (string, bool) {
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	text = strings.TrimSpace(strings.ToLower(text))
	return text, text != ""
}

func intMetricValue(value any) int {
	switch typed := value.(type) {
	case int:
		return nonNegativeInt(typed)
	case int64:
		return intFromInt64(typed)
	case float64:
		return nonNegativeInt(int(typed))
	case string:
		return intFromString(typed)
	}
	return 0
}

func intFromInt64(value int64) int {
	if value <= 0 {
		return 0
	}
	if value > int64(math.MaxInt) {
		return math.MaxInt
	}
	return int(value)
}

func intFromString(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return nonNegativeInt(parsed)
}

func nonNegativeInt(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

// WidgetError is the per-widget non-fatal error surfaced to the renderer.
type WidgetError struct {
	Code       string `json:"code"`
	MessageKey string `json:"message_key,omitempty"`
	Message    string `json:"message,omitempty"`
}

// HealthPayload is the MVP health widget payload shape.
type HealthPayload struct {
	Summary          HealthSummaryItem `json:"summary"`
	Items            []HealthItem      `json:"items"`
	HealthyModules   int               `json:"healthy_modules,omitempty"`
	AbnormalServices int               `json:"abnormal_services,omitempty"`
}

// HealthSummaryItem summarizes one health widget.
type HealthSummaryItem struct {
	Status   HealthStatus `json:"status"`
	LabelKey string       `json:"label_key,omitempty"`
	Label    string       `json:"label,omitempty"`
}

// HealthItem describes one health row.
type HealthItem struct {
	Key            string       `json:"key"`
	LabelKey       string       `json:"label_key"`
	Label          string       `json:"label"`
	Status         HealthStatus `json:"status"`
	DescriptionKey string       `json:"description_key,omitempty"`
	Description    string       `json:"description,omitempty"`
	RouteLocation  string       `json:"route_location,omitempty"`
}
