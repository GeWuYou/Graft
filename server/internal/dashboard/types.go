// Package dashboard owns the MVP dashboard contribution registry and aggregate
// routes. It must stay limited to runtime contributions; dashboard persistence
// and user preferences belong in a future module.
package dashboard

import (
	"context"
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
	// WidgetSizeSmall requests a small dashboard grid span.
	WidgetSizeSmall WidgetSize = "small"
	// WidgetSizeMedium requests a medium dashboard grid span.
	WidgetSizeMedium WidgetSize = "medium"
	// WidgetSizeLarge requests a large dashboard grid span.
	WidgetSizeLarge WidgetSize = "large"
	// WidgetSizeFull requests the full dashboard grid width.
	WidgetSizeFull WidgetSize = "full"
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
	Order               int
	RefreshInterval     time.Duration
	RouteLocation       string
	RequiredPermissions []string
	LoaderTimeout       time.Duration
	Loader              WidgetLoader
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

// WidgetError is the per-widget non-fatal error surfaced to the renderer.
type WidgetError struct {
	Code       string `json:"code"`
	MessageKey string `json:"message_key,omitempty"`
	Message    string `json:"message,omitempty"`
}

// HealthPayload is the MVP health widget payload shape.
type HealthPayload struct {
	Summary HealthSummaryItem `json:"summary"`
	Items   []HealthItem      `json:"items"`
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
