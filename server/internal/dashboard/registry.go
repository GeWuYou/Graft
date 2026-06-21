package dashboard

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// Registry stores dashboard contributions in registration order.
type Registry struct {
	mu                sync.RWMutex
	widgetDefinitions map[string]WidgetDefinition
	widgetOrder       []string
}

// NewRegistry creates an empty dashboard contribution registry.
func NewRegistry() *Registry {
	return &Registry{
		widgetDefinitions: make(map[string]WidgetDefinition),
		widgetOrder:       make([]string, 0),
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

	if err := r.storeWidgetDefinition(normalized); err != nil {
		return err
	}
	return nil
}

// Get returns one registered widget definition snapshot.
func (r *Registry) Get(id string) (WidgetDefinition, bool) {
	if r == nil {
		return WidgetDefinition{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	definition, ok := r.widgetDefinitions[strings.TrimSpace(id)]
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

	items := make([]WidgetDefinition, 0, len(r.widgetOrder))
	for _, id := range r.widgetOrder {
		items = append(items, cloneDefinition(r.widgetDefinitions[id]))
	}
	sortWidgetDefinitions(items)
	return items
}

func (r *Registry) storeWidgetDefinition(definition WidgetDefinition) error {
	if existing, exists := r.widgetDefinitions[definition.ID]; exists {
		return fmt.Errorf("dashboard widget %s already registered by module %s", definition.ID, existing.ModuleKey)
	}

	r.widgetDefinitions[definition.ID] = definition
	r.widgetOrder = append(r.widgetOrder, definition.ID)
	return nil
}

// sortWidgetDefinitions sorts widget definitions in-place by order, then by ID.
func sortWidgetDefinitions(items []WidgetDefinition) {
	slices.SortStableFunc(items, func(left, right WidgetDefinition) int {
		if left.Order != right.Order {
			return left.Order - right.Order
		}
		return strings.Compare(left.ID, right.ID)
	})
}

// normalizeDefinition normalizes fields and validates the widget definition. It returns the normalized definition and nil on success, or an error if validation fails.
func normalizeDefinition(definition WidgetDefinition) (WidgetDefinition, error) {
	normalized := normalizeDefinitionStrings(definition)
	normalized = normalizeDefinitionDefaults(normalized)
	if err := validateDefinition(normalized); err != nil {
		return WidgetDefinition{}, err
	}
	return normalized, nil
}

func normalizeDefinitionStrings(definition WidgetDefinition) WidgetDefinition {
	normalized := cloneDefinition(definition)
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.ModuleKey = strings.TrimSpace(normalized.ModuleKey)
	normalized.TitleKey = strings.TrimSpace(normalized.TitleKey)
	normalized.Title = strings.TrimSpace(normalized.Title)
	normalized.DescriptionKey = strings.TrimSpace(normalized.DescriptionKey)
	normalized.Description = strings.TrimSpace(normalized.Description)
	normalized.RouteLocation = strings.TrimSpace(normalized.RouteLocation)
	normalized.Action.LabelKey = strings.TrimSpace(normalized.Action.LabelKey)
	normalized.Action.Label = strings.TrimSpace(normalized.Action.Label)
	normalized.Action.Route = strings.TrimSpace(normalized.Action.Route)
	normalized.RequiredPermissions = trimNonEmptyStrings(normalized.RequiredPermissions)
	return normalized
}

func normalizeDefinitionDefaults(definition WidgetDefinition) WidgetDefinition {
	normalized := definition
	if normalized.Size == "" {
		normalized.Size = WidgetSizeMedium
	}
	if normalized.Category == "" {
		normalized.Category = WidgetCategorySystem
	}
	if normalized.Priority == "" {
		normalized.Priority = WidgetPriorityNormal
	}
	if normalized.Action.Route == "" {
		normalized.Action.Route = normalized.RouteLocation
	}
	return normalized
}

func validateDefinition(definition WidgetDefinition) error {
	if err := validateDefinitionIdentity(definition); err != nil {
		return err
	}
	if err := validateDefinitionFramework(definition); err != nil {
		return err
	}
	if definition.Loader == nil {
		return fmt.Errorf("dashboard widget %s loader is required", definition.ID)
	}
	if definition.LoaderTimeout < 0 {
		return fmt.Errorf("dashboard widget %s loader timeout must not be negative", definition.ID)
	}
	return nil
}

func validateDefinitionIdentity(definition WidgetDefinition) error {
	if definition.ID == "" {
		return errors.New("dashboard widget id is required")
	}
	if definition.ModuleKey == "" {
		return fmt.Errorf("dashboard widget %s module key is required", definition.ID)
	}
	if definition.Type == "" {
		return fmt.Errorf("dashboard widget %s type is required", definition.ID)
	}
	if !validWidgetType(definition.Type) {
		return fmt.Errorf("dashboard widget %s has unsupported type %q", definition.ID, definition.Type)
	}
	return nil
}

func validateDefinitionFramework(definition WidgetDefinition) error {
	if !validWidgetSize(definition.Size) {
		return fmt.Errorf("dashboard widget %s has unsupported size %q", definition.ID, definition.Size)
	}
	if !validWidgetCategory(definition.Category) {
		return fmt.Errorf("dashboard widget %s has unsupported category %q", definition.ID, definition.Category)
	}
	if !validWidgetPriority(definition.Priority) {
		return fmt.Errorf("dashboard widget %s has unsupported priority %q", definition.ID, definition.Priority)
	}
	return nil
}

// cloneDefinition returns a copy of the definition with a separately copied RequiredPermissions slice.
func cloneDefinition(definition WidgetDefinition) WidgetDefinition {
	cloned := definition
	cloned.RequiredPermissions = append([]string(nil), definition.RequiredPermissions...)
	return cloned
}

// validWidgetType reports whether widgetType is a valid widget type.
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
	case WidgetSizeSmall, WidgetSizeMedium, WidgetSizeLarge:
		return true
	default:
		return false
	}
}

func validWidgetCategory(category WidgetCategory) bool {
	switch category {
	case WidgetCategorySystem, WidgetCategorySecurity, WidgetCategoryOperation, WidgetCategoryBusiness:
		return true
	default:
		return false
	}
}

func validWidgetPriority(priority WidgetPriority) bool {
	switch priority {
	case WidgetPriorityCritical, WidgetPriorityWarning, WidgetPriorityNormal, WidgetPriorityInfo:
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
