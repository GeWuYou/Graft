// Package locales 提供公告模块只读嵌入式语言资源描述符。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 返回公告模块的只读语言资源描述符；解析和注册仍由 i18n 统一负责。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("announcement"))
	if err != nil {
		return nil, fmt.Errorf("load announcement locale resources: %w", err)
	}
	return resources, nil
}
