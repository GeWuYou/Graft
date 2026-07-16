// Package locales 提供 rbac 模块只读的内嵌本地化资源描述。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 返回 rbac 模块只读的内嵌本地化资源描述；解析和注册仍由 i18n 集中负责。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("rbac"))
	if err != nil {
		return nil, fmt.Errorf("load rbac locale resources: %w", err)
	}
	return resources, nil
}
