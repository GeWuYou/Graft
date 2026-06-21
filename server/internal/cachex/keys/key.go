// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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

// New creates one validated cache key.
func New(namespace string, name string, parts ...string) (Key, error) {
	trimmedNamespace := strings.TrimSpace(namespace)
	if trimmedNamespace == "" {
		return Key{}, fmt.Errorf("cache key namespace is required")
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Key{}, fmt.Errorf("cache key name is required")
	}

	normalizedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart == "" {
			return Key{}, fmt.Errorf("cache key part is required")
		}
		normalizedParts = append(normalizedParts, trimmedPart)
	}

	return Key{
		namespace: trimmedNamespace,
		name:      trimmedName,
		parts:     normalizedParts,
	}, nil
}

// MustNew creates one cache key and panics when validation fails.
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
