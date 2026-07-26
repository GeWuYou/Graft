// Package locales 为 platform-backup 模块提供内嵌本地化资源。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 读取 platform-backup 的只读本地化资源。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("backup"))
	if err != nil {
		return nil, fmt.Errorf("load platform-backup locale resources: %w", err)
	}
	return resources, nil
}
