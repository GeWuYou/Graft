// Package keys defines stable cache key composition for cachex.
package keys

import (
	"fmt"
	"strings"
)

// Key represents one composed cache key with explicit namespace and segments.
type Key struct {
	namespace string
	name      string
	parts     []string
}

// New creates a validated and normalized cache Key from the provided namespace, name, and optional parts.
// Whitespace is trimmed from all inputs. An error is returned if any input is empty or contains a colon.
func New(namespace string, name string, parts ...string) (Key, error) {
	trimmedNamespace := strings.TrimSpace(namespace)
	if trimmedNamespace == "" {
		return Key{}, fmt.Errorf("cache key namespace is required")
	}
	if strings.Contains(trimmedNamespace, ":") {
		return Key{}, fmt.Errorf("cache key namespace must not contain ':'")
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Key{}, fmt.Errorf("cache key name is required")
	}
	if strings.Contains(trimmedName, ":") {
		return Key{}, fmt.Errorf("cache key name must not contain ':'")
	}

	normalizedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart == "" {
			return Key{}, fmt.Errorf("cache key part is required")
		}
		if strings.Contains(trimmedPart, ":") {
			return Key{}, fmt.Errorf("cache key part must not contain ':'")
		}
		normalizedParts = append(normalizedParts, trimmedPart)
	}

	return Key{
		namespace: trimmedNamespace,
		name:      trimmedName,
		parts:     normalizedParts,
	}, nil
}

// MustNew creates a validated cache Key, panicking if validation fails.
func MustNew(namespace string, name string, parts ...string) Key {
	key, err := New(namespace, name, parts...)
	if err != nil {
		panic(err)
	}

	return key
}

// Namespace returns the stable key namespace.
func (k Key) Namespace() string {
	return k.namespace
}

// Name returns the stable key name.
func (k Key) Name() string {
	return k.name
}

// Parts returns defensive copies of the key path segments.
func (k Key) Parts() []string {
	cloned := make([]string, len(k.parts))
	copy(cloned, k.parts)
	return cloned
}

// String renders the key to a stable colon-separated form.
func (k Key) String() string {
	segments := []string{k.namespace, k.name}
	segments = append(segments, k.parts...)
	return strings.Join(segments, ":")
}
