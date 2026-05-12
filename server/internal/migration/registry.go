// Package migration stores explicit migration registrations for plugins.
package migration

// Item describes one migration that belongs to a plugin.
type Item struct {
	Name   string
	Plugin string
}

// Registry stores declared migrations in registration order.
type Registry struct {
	items []Item
}

// NewRegistry creates an empty migration registry.
func NewRegistry() *Registry {
	return &Registry{items: make([]Item, 0)}
}

// Register appends one migration item.
func (r *Registry) Register(item Item) {
	r.items = append(r.items, item)
}

// Items returns a copy of the registered migrations.
func (r *Registry) Items() []Item {
	items := make([]Item, len(r.items))
	copy(items, r.items)
	return items
}
