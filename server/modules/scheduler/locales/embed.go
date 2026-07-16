// Package locales 为 scheduler 模块提供只读的内嵌语言资源描述。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 暴露 scheduler 模块的只读语言资源描述；解析和注册仍由 i18n 服务集中负责。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("scheduler"))
	if err != nil {
		return nil, fmt.Errorf("load scheduler locale resources: %w", err)
	}
	return resources, nil
}
