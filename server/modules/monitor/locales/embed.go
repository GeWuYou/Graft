// Package locales 为 monitor 模块提供只读的内嵌本地化资源描述。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 暴露 monitor 模块的只读内嵌本地化资源描述。
// 解析和注册仍由 i18n 服务集中负责。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("monitor"))
	if err != nil {
		return nil, fmt.Errorf("load monitor locale resources: %w", err)
	}
	return resources, nil
}
