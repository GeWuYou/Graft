// Package container provides the explicit singleton registry used by core and plugins.
package container

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// Provider builds a singleton service using the current resolver.
type Provider func(resolver Resolver) (any, error)

// Registry exposes explicit singleton registration.
type Registry interface {
	// RegisterSingleton registers a singleton provider under the given key.
	RegisterSingleton(key any, provider Provider) error
}

// Resolver exposes explicit singleton resolution.
type Resolver interface {
	// Resolve returns a singleton for the provided key.
	Resolve(key any) (any, error)
}

// Container stores singleton providers and instances for the runtime shell.
type Container struct {
	mu        sync.RWMutex
	providers map[string]Provider
	instances map[string]any
	inflight  map[string]*inflightCall
}

type inflightCall struct {
	done chan struct{}
	val  any
	err  error
}

// New creates an empty service container.
func New() *Container {
	return &Container{
		providers: make(map[string]Provider),
		instances: make(map[string]any),
		inflight:  make(map[string]*inflightCall),
	}
}

// RegisterSingleton stores one provider for one service key.
func (c *Container) RegisterSingleton(key any, provider Provider) error {
	if provider == nil {
		return errors.New("provider is required")
	}

	name := keyName(key)

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.providers[name]; exists {
		return fmt.Errorf("service already registered: %s", name)
	}

	c.providers[name] = provider
	return nil
}

// Resolve builds the singleton once and caches the result.
func (c *Container) Resolve(key any) (any, error) {
	name := keyName(key)

	c.mu.Lock()
	if instance, ok := c.instances[name]; ok {
		c.mu.Unlock()
		return instance, nil
	}

	if call, ok := c.inflight[name]; ok {
		c.mu.Unlock()
		<-call.done
		if call.err != nil {
			return nil, fmt.Errorf("build service %s: %w", name, call.err)
		}
		return call.val, nil
	}

	provider, ok := c.providers[name]
	if !ok {
		c.mu.Unlock()
		return nil, fmt.Errorf("service not registered: %s", name)
	}

	call := &inflightCall{done: make(chan struct{})}
	c.inflight[name] = call
	c.mu.Unlock()

	instance, err := provider(c)

	c.mu.Lock()
	if err == nil {
		c.instances[name] = instance
	}
	call.val = instance
	call.err = err
	close(call.done)
	delete(c.inflight, name)
	c.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("build service %s: %w", name, err)
	}
	return instance, nil
}

func keyName(key any) string {
	if key == nil {
		return "<nil>"
	}

	return reflect.TypeOf(key).String()
}
