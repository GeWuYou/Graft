// Package locales exposes read-only embedded locale descriptors for the security module.
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources exposes read-only locale descriptors for the security module.
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("security"))
	if err != nil {
		return nil, fmt.Errorf("load security locale resources: %w", err)
	}
	return resources, nil
}
