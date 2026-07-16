// Package locales 暴露 system-config 模块只读的内嵌语言资源描述。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 返回 system-config 模块的只读语言资源描述；解析与注册仍由 i18n 服务集中负责。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("system-config"))
	if err != nil {
		return nil, fmt.Errorf("load system-config locale resources: %w", err)
	}
	return resources, nil
}
