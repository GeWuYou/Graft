// Package plugin defines runtime plugin contracts and lifecycle management.
package plugin

import (
	"errors"
	"fmt"
	"sort"

	"github.com/gin-gonic/gin"

	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/menu"
	"graft/server/internal/migration"
	"graft/server/internal/permission"
)

// Plugin declares the stable lifecycle contract for all backend plugins.
type Plugin interface {
	// Name returns the stable plugin identifier used in dependencies and metadata.
	Name() string
	// Version returns the current plugin version string.
	Version() string
	// DependsOn returns required plugin names that must register first.
	DependsOn() []string
	// Register declares routes, permissions, menus, migrations, jobs, and services.
	Register(ctx *Context) error
	// Boot starts runtime behavior after all registrations and migrations complete.
	Boot(ctx *Context) error
	// Shutdown releases runtime resources in reverse startup order.
	Shutdown(ctx *Context) error
}

// Context exposes the explicit runtime handles that plugins may use.
type Context struct {
	Router             gin.IRouter
	Services           *container.Container
	MenuRegistry       *menu.Registry
	PermissionRegistry *permission.Registry
	MigrationRegistry  *migration.Registry
	CronRegistry       *cronx.Registry
}

// Manager orders plugins and drives lifecycle execution.
type Manager struct {
	plugins []Plugin
}

// NewManager creates an empty plugin manager.
func NewManager() *Manager {
	return &Manager{plugins: make([]Plugin, 0)}
}

// RegisterPlugin adds one plugin to the manager before runtime startup.
func (m *Manager) RegisterPlugin(p Plugin) error {
	if p == nil {
		return errors.New("plugin is required")
	}

	for _, existing := range m.plugins {
		if existing.Name() == p.Name() {
			return fmt.Errorf("plugin already registered: %s", p.Name())
		}
	}

	m.plugins = append(m.plugins, p)
	return nil
}

// Ordered returns plugins sorted by declared dependencies.
func (m *Manager) Ordered() ([]Plugin, error) {
	total := len(m.plugins)
	if total == 0 {
		return nil, nil
	}

	index := make(map[string]Plugin, total)
	inDegree := make(map[string]int, total)
	edges := make(map[string][]string, total)

	for _, p := range m.plugins {
		name := p.Name()
		index[name] = p
		inDegree[name] = 0
	}

	for _, p := range m.plugins {
		for _, dependency := range p.DependsOn() {
			if _, ok := index[dependency]; !ok {
				return nil, fmt.Errorf("plugin %s depends on missing plugin %s", p.Name(), dependency)
			}

			edges[dependency] = append(edges[dependency], p.Name())
			inDegree[p.Name()]++
		}
	}

	queue := make([]string, 0)
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	ordered := make([]Plugin, 0, total)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		ordered = append(ordered, index[name])

		for _, next := range edges[name] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
				sort.Strings(queue)
			}
		}
	}

	if len(ordered) != total {
		return nil, errors.New("plugin dependency cycle detected")
	}

	return ordered, nil
}
