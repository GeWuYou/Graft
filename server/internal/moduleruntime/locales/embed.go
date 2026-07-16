// Package locales 为 module-runtime owner 暴露只读的嵌入式本地化资源描述。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 为 module-runtime owner 暴露只读的本地化资源描述；解析和注册统一由 i18n 集中处理。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("module-runtime"))
	if err != nil {
		return nil, fmt.Errorf("load module-runtime locale resources: %w", err)
	}
	return resources, nil
}
