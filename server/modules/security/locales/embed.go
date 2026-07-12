// Package locales exposes read-only embedded locale descriptors for the security module.
package locales

import (
	"embed"
	"fmt"

	"graft/server/internal/i18n"
)

//go:embed *.yaml
var embeddedLocaleFiles embed.FS

// EmbeddedLocaleResources 提供安全模块嵌入的语言资源描述符。
// 成功时返回资源描述符及 nil；加载失败时返回 nil 及带上下文的错误。
func EmbeddedLocaleResources() ([]i18n.EmbeddedLocaleResource, error) {
	resources, err := i18n.EmbeddedLocaleResourcesFromFS(embeddedLocaleFiles, i18n.Namespace("security"))
	if err != nil {
		return nil, fmt.Errorf("load security locale resources: %w", err)
	}
	return resources, nil
}
