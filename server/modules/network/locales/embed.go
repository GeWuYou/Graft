// Package locales 暴露 platform-network 模块只读的内嵌语言资源描述。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 返回 platform-network 模块的只读语言资源描述。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("network"))
	if err != nil {
		return nil, fmt.Errorf("load platform-network locale resources: %w", err)
	}
	return resources, nil
}
