package dashboard

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// Registry stores dashboard widget contributions in registration order.
type Registry struct {
	mu          sync.RWMutex
	definitions map[string]WidgetDefinition
	order       []string
}

// NewRegistry creates an empty dashboard widget registry.
func NewRegistry() *Registry {
	return &Registry{
		definitions: make(map[string]WidgetDefinition),
		order:       make([]string, 0),
	}
}

// Register validates and stores one widget contribution.
func (r *Registry) Register(definition WidgetDefinition) error {
	if r == nil {
		return errors.New("dashboard registry is unavailable")
	}

	normalized, err := normalizeDefinition(definition)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, exists := r.definitions[normalized.ID]; exists {
		return fmt.Errorf("dashboard widget %s already registered by module %s", normalized.ID, existing.ModuleKey)
	}

	r.definitions[normalized.ID] = normalized
	r.order = append(r.order, normalized.ID)
	return nil
}

// Get returns one registered widget definition snapshot.
func (r *Registry) Get(id string) (WidgetDefinition, bool) {
	if r == nil {
		return WidgetDefinition{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	definition, ok := r.definitions[strings.TrimSpace(id)]
	if !ok {
		return WidgetDefinition{}, false
	}
	return cloneDefinition(definition), true
}

// Items returns registered widget definitions ordered by order then id.
func (r *Registry) Items() []WidgetDefinition {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]WidgetDefinition, 0, len(r.order))
	for _, id := range r.order {
		items = append(items, cloneDefinition(r.definitions[id]))
	}
	slices.SortStableFunc(items, func(left, right WidgetDefinition) int {
		if left.Order != right.Order {
			return left.Order - right.Order
		}
		return strings.Compare(left.ID, right.ID)
	})
	return items
}

func normalizeDefinition(definition WidgetDefinition) (WidgetDefinition, error) {
	normalized := cloneDefinition(definition)
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.ModuleKey = strings.TrimSpace(normalized.ModuleKey)
	normalized.TitleKey = strings.TrimSpace(normalized.TitleKey)
	normalized.Title = strings.TrimSpace(normalized.Title)
	normalized.DescriptionKey = strings.TrimSpace(normalized.DescriptionKey)
	normalized.Description = strings.TrimSpace(normalized.Description)
	normalized.RouteLocation = strings.TrimSpace(normalized.RouteLocation)
	normalized.RequiredPermissions = trimNonEmptyStrings(normalized.RequiredPermissions)

	if normalized.ID == "" {
		return WidgetDefinition{}, errors.New("dashboard widget id is required")
	}
	if normalized.ModuleKey == "" {
		return WidgetDefinition{}, fmt.Errorf("dashboard widget %s module key is required", normalized.ID)
	}
	if normalized.Type == "" {
		return WidgetDefinition{}, fmt.Errorf("dashboard widget %s type is required", normalized.ID)
	}
	if !validWidgetType(normalized.Type) {
		return WidgetDefinition{}, fmt.Errorf("dashboard widget %s has unsupported type %q", normalized.ID, normalized.Type)
	}
	if normalized.Size == "" {
		normalized.Size = WidgetSizeMedium
	}
	if !validWidgetSize(normalized.Size) {
		return WidgetDefinition{}, fmt.Errorf("dashboard widget %s has unsupported size %q", normalized.ID, normalized.Size)
	}
	if normalized.Loader == nil {
		return WidgetDefinition{}, fmt.Errorf("dashboard widget %s loader is required", normalized.ID)
	}
	if normalized.LoaderTimeout < 0 {
		return WidgetDefinition{}, fmt.Errorf("dashboard widget %s loader timeout must not be negative", normalized.ID)
	}

	return normalized, nil
}

func cloneDefinition(definition WidgetDefinition) WidgetDefinition {
	cloned := definition
	cloned.RequiredPermissions = append([]string(nil), definition.RequiredPermissions...)
	return cloned
}

func validWidgetType(widgetType WidgetType) bool {
	switch widgetType {
	case WidgetTypeStatGroup, WidgetTypeAlertList, WidgetTypeLinkList, WidgetTypeTimeline, WidgetTypeHealth:
		return true
	default:
		return false
	}
}

func validWidgetSize(size WidgetSize) bool {
	switch size {
	case WidgetSizeSmall, WidgetSizeMedium, WidgetSizeLarge, WidgetSizeFull:
		return true
	default:
		return false
	}
}

func trimNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	items := make([]string, 0, len(values))
	for _, value := range values {
		if item := strings.TrimSpace(value); item != "" {
			items = append(items, item)
		}
	}
	return items
}
