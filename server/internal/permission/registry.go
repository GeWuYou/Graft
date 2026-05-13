// Package permission stores backend permission declarations for plugins.
package permission

// Item represents one permission point declared by a plugin.
type Item struct {
	Code        string
	Name        string
	Description string
	Plugin      string
}

// Registry stores declared permission items in registration order.
type Registry struct {
	items []Item
}

// NewRegistry creates an empty permission registry.
func NewRegistry() *Registry {
	return &Registry{items: make([]Item, 0)}
}

// Register appends one permission item to the registry.
func (r *Registry) Register(item Item) {
	r.items = append(r.items, item)
}

// Items returns a copy of the registered permission set.
func (r *Registry) Items() []Item {
	items := make([]Item, len(r.items))
	copy(items, r.items)
	return items
}
