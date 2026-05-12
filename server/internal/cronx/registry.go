// Package cronx stores plugin-declared cron jobs for later scheduler wiring.
package cronx

// Job describes one scheduled task declaration.
type Job struct {
	Name     string
	Schedule string
	Plugin   string
}

// Registry stores scheduled jobs in registration order.
type Registry struct {
	items []Job
}

// NewRegistry creates an empty cron job registry.
func NewRegistry() *Registry {
	return &Registry{items: make([]Job, 0)}
}

// Register appends one cron job declaration.
func (r *Registry) Register(item Job) {
	r.items = append(r.items, item)
}

// Items returns a copy of the registered job set.
func (r *Registry) Items() []Job {
	items := make([]Job, len(r.items))
	copy(items, r.items)
	return items
}
