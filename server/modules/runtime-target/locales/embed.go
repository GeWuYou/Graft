// Package locales owns embedded runtime-target server locale resources.
package locales

import (
	"embed"
	"fmt"
	"graft/server/internal/i18n"
)

//go:embed *.yaml
var files embed.FS

// EmbeddedLocaleResources exposes runtime-target locale resources to the central i18n facade.
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(files, i18n.Namespace("runtime-target"))
	if err != nil {
		return nil, fmt.Errorf("load runtime target locale resources: %w", err)
	}
	return resources, nil
}
