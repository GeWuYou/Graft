// Package locales 提供 Registry 模块自有的嵌入式本地化资源。
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 读取 Registry namespace 的所有 locale 文件。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("registry"))
	if err != nil {
		return nil, fmt.Errorf("load registry locale resources: %w", err)
	}
	return resources, nil
}
