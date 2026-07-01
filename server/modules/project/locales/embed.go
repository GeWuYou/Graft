// Package locales exposes read-only embedded locale descriptors for the project
// module.
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources exposes read-only locale descriptors for the project
// EmbeddedLocaleResources 从内嵌文件系统加载项目命名空间的语言资源。
// 加载失败时返回带上下文的错误。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("project"))
	if err != nil {
		return nil, fmt.Errorf("load project locale resources: %w", err)
	}
	return resources, nil
}
