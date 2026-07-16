// Package locales 暴露审计模块只读的内嵌语言资源描述。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 暴露审计模块只读的内嵌语言资源描述；解析和注册仍由 i18n 集中负责。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("audit"))
	if err != nil {
		return nil, fmt.Errorf("load audit locale resources: %w", err)
	}
	return resources, nil
}
