// Package locales exposes read-only embedded locale descriptors for the
// module runtime owner.
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources exposes read-only locale descriptors for the
// module runtime owner. Parsing and registration stay centralized in i18n.
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("module-runtime"))
	if err != nil {
		return nil, fmt.Errorf("load module-runtime locale resources: %w", err)
	}
	return resources, nil
}
