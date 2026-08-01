// Package locales 暴露 Build 模块内嵌的只读语言资源描述。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 返回 Build 模块的本地化资源，解析仍由 i18n 边界集中负责。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("build"))
	if err != nil {
		return nil, fmt.Errorf("load build locale resources: %w", err)
	}
	return resources, nil
}
