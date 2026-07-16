// Package locales 负责 runtime-target 服务端内嵌的语言资源。
package locales

import (
	"embed"
	"fmt"
	"graft/server/internal/i18n"
)

//go:embed *.yaml
var files embed.FS

// EmbeddedLocaleResources 加载 runtime-target 命名空间的嵌入式本地化资源。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(files, i18n.Namespace("runtime-target"))
	if err != nil {
		return nil, fmt.Errorf("load runtime target locale resources: %w", err)
	}
	return resources, nil
}
