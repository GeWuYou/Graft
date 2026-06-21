// Package locales exposes read-only embedded locale descriptors for the
// system-config module.
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources exposes read-only locale descriptors for the
// system-config module. Parsing and registration stay centralized in i18n.
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("system-config"))
	if err != nil {
		return nil, fmt.Errorf("load system-config locale resources: %w", err)
	}
	return resources, nil
}
