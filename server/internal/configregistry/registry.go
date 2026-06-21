package configregistry

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// Registry stores module-declared config definitions in registration order.
type Registry struct {
	mu          sync.RWMutex
	definitions map[string]Definition
	order       []string
}

// NewRegistry creates an empty system configuration definition registry.
func NewRegistry() *Registry {
	return &Registry{
		definitions: make(map[string]Definition),
		order:       make([]string, 0),
	}
}

// Register validates and stores one module-declared config definition.
func (r *Registry) Register(definition Definition) error {
	if r == nil {
		return errors.New("config registry is unavailable")
	}
	if definition.RuntimeApplyMode == "" {
		definition.RuntimeApplyMode = RuntimeApplyModeUnknown
	}
	if err := validateDefinition(definition); err != nil {
		return err
	}

	normalized := definition.Snapshot()
	normalized.Key = strings.TrimSpace(normalized.Key)
	normalized.Module = strings.TrimSpace(normalized.Module)
	normalized.Domain = strings.TrimSpace(normalized.Domain)
	normalized.DomainKey = strings.TrimSpace(normalized.DomainKey)
	normalized.DomainLabel = strings.TrimSpace(normalized.DomainLabel)
	normalized.Group = strings.TrimSpace(normalized.Group)
	normalized.GroupKey = strings.TrimSpace(normalized.GroupKey)
	normalized.GroupLabel = strings.TrimSpace(normalized.GroupLabel)
	normalized.GroupDescription = strings.TrimSpace(normalized.GroupDescription)
	normalized.GroupDescriptionKey = strings.TrimSpace(normalized.GroupDescriptionKey)
	normalized.Title = strings.TrimSpace(normalized.Title)
	normalized.TitleKey = strings.TrimSpace(normalized.TitleKey)
	normalized.Description = strings.TrimSpace(normalized.Description)
	normalized.DescriptionKey = strings.TrimSpace(normalized.DescriptionKey)
	normalized.Permission = strings.TrimSpace(normalized.Permission)
	normalized.Tags = trimNonEmptyStrings(normalized.Tags)
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, exists := r.definitions[normalized.Key]; exists {
		return fmt.Errorf("config definition %s already registered by module %s", normalized.Key, existing.Module)
	}
	r.definitions[normalized.Key] = normalized
	r.order = append(r.order, normalized.Key)
	return nil
}

func trimNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		if item := strings.TrimSpace(value); item != "" {
			trimmed = append(trimmed, item)
		}
	}
	return trimmed
}

// Get returns one definition by key.
func (r *Registry) Get(key string) (Definition, bool) {
	if r == nil {
		return Definition{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	definition, ok := r.definitions[strings.TrimSpace(key)]
	if !ok {
		return Definition{}, false
	}
	return definition.Snapshot(), true
}

// Items returns definitions ordered by caller-facing order fields.
func (r *Registry) Items() []Definition {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Definition, 0, len(r.order))
	for _, key := range r.order {
		items = append(items, r.definitions[key].Snapshot())
	}
	slices.SortStableFunc(items, func(left, right Definition) int {
		if left.Order != right.Order {
			return left.Order - right.Order
		}
		return strings.Compare(left.Key, right.Key)
	})
	return items
}
