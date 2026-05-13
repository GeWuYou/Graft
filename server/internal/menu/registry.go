// Package menu stores backend-declared navigation metadata for the web shell.
package menu

// Item represents one backend-declared menu entry.
type Item struct {
	Code       string
	Title      string
	Path       string
	Icon       string
	Permission string
	Plugin     string
}

// Registry stores declared menu items in registration order.
type Registry struct {
	items []Item
}

// NewRegistry creates an empty menu registry.
func NewRegistry() *Registry {
	return &Registry{items: make([]Item, 0)}
}

// Register appends one menu item to the registry.
func (r *Registry) Register(item Item) {
	r.items = append(r.items, item)
}

// Items returns a copy of the registered menu set.
func (r *Registry) Items() []Item {
	items := make([]Item, len(r.items))
	copy(items, r.items)
	return items
}
