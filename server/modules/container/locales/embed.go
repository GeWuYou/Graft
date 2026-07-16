// Package locales 暴露容器模块内嵌的只读语言资源描述；解析和注册仍由 i18n 边界集中负责。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 暴露容器模块的只读内嵌语言资源描述；解析和注册仍由 i18n 边界集中负责。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("container"))
	if err != nil {
		return nil, fmt.Errorf("load container locale resources: %w", err)
	}
	return resources, nil
}
