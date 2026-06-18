// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package locales exposes read-only embedded locale descriptors for the
// scheduler module.
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources exposes read-only locale descriptors for the
// scheduler module. Parsing and registration stay centralized in i18n.
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("scheduler"))
	if err != nil {
		return nil, fmt.Errorf("load scheduler locale resources: %w", err)
	}
	return resources, nil
}
