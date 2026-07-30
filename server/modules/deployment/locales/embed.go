// Package locales 暴露 Deployment Runtime 内嵌的只读语言资源描述；解析和注册仍由 i18n 边界集中负责。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 暴露 Deployment Runtime 内嵌的只读语言资源描述。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("deployment"))
	if err != nil {
		return nil, fmt.Errorf("load deployment locale resources: %w", err)
	}
	return resources, nil
}
