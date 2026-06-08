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
	quickLinks        map[string]QuickLinkDefinition
	quickLinkOrder    []string
}

// NewRegistry creates an empty dashboard contribution registry.
func NewRegistry() *Registry {
	return &Registry{
		widgetDefinitions: make(map[string]WidgetDefinition),
		widgetOrder:       make([]string, 0),
		quickLinks:        make(map[string]QuickLinkDefinition),
		quickLinkOrder:    make([]string, 0),
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

// RegisterQuickLink validates and stores one dashboard quick-entry contribution.
func (r *Registry) RegisterQuickLink(definition QuickLinkDefinition) error {
	if r == nil {
		return errors.New("dashboard registry is unavailable")
	}

	normalized, err := normalizeQuickLink(definition)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.storeQuickLink(normalized); err != nil {
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

// GetQuickLink returns one registered quick link definition snapshot.
func (r *Registry) GetQuickLink(id string) (QuickLinkDefinition, bool) {
	if r == nil {
		return QuickLinkDefinition{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	definition, ok := r.quickLinks[strings.TrimSpace(id)]
	if !ok {
		return QuickLinkDefinition{}, false
	}
	return cloneQuickLink(definition), true
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

// QuickLinks returns registered quick links ordered by order then id.
func (r *Registry) QuickLinks() []QuickLinkDefinition {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]QuickLinkDefinition, 0, len(r.quickLinkOrder))
	for _, id := range r.quickLinkOrder {
		items = append(items, cloneQuickLink(r.quickLinks[id]))
	}
	sortQuickLinks(items)
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

func (r *Registry) storeQuickLink(definition QuickLinkDefinition) error {
	if existing, exists := r.quickLinks[definition.ID]; exists {
		return fmt.Errorf("dashboard quick link %s already registered by module %s", definition.ID, existing.ModuleKey)
	}

	r.quickLinks[definition.ID] = definition
	r.quickLinkOrder = append(r.quickLinkOrder, definition.ID)
	return nil
}

func sortWidgetDefinitions(items []WidgetDefinition) {
	slices.SortStableFunc(items, func(left, right WidgetDefinition) int {
		if left.Order != right.Order {
			return left.Order - right.Order
		}
		return strings.Compare(left.ID, right.ID)
	})
}

func sortQuickLinks(items []QuickLinkDefinition) {
	slices.SortStableFunc(items, func(left, right QuickLinkDefinition) int {
		if left.Order != right.Order {
			return left.Order - right.Order
		}
		return strings.Compare(left.ID, right.ID)
	})
}

func normalizeQuickLink(definition QuickLinkDefinition) (QuickLinkDefinition, error) {
	normalized := cloneQuickLink(definition)
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.ModuleKey = strings.TrimSpace(normalized.ModuleKey)
	normalized.TitleKey = strings.TrimSpace(normalized.TitleKey)
	normalized.Title = strings.TrimSpace(normalized.Title)
	normalized.DescriptionKey = strings.TrimSpace(normalized.DescriptionKey)
	normalized.Description = strings.TrimSpace(normalized.Description)
	normalized.Icon = strings.TrimSpace(normalized.Icon)
	normalized.RouteLocation = strings.TrimSpace(normalized.RouteLocation)
	normalized.RequiredPermissions = trimNonEmptyStrings(normalized.RequiredPermissions)

	if normalized.ID == "" {
		return QuickLinkDefinition{}, errors.New("dashboard quick link id is required")
	}
	if normalized.ModuleKey == "" {
		return QuickLinkDefinition{}, fmt.Errorf("dashboard quick link %s module key is required", normalized.ID)
	}
	if normalized.TitleKey == "" && normalized.Title == "" {
		return QuickLinkDefinition{}, fmt.Errorf("dashboard quick link %s title key or title is required", normalized.ID)
	}
	if normalized.RouteLocation == "" {
		return QuickLinkDefinition{}, fmt.Errorf("dashboard quick link %s route location is required", normalized.ID)
	}

	return normalized, nil
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

func cloneQuickLink(definition QuickLinkDefinition) QuickLinkDefinition {
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
