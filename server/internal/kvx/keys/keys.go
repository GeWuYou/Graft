package keys

import "strings"

const separator = ":"

// Join 从修剪后的非空部分构建单个稳定的 KV 键。
func Join(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		cleaned = append(cleaned, part)
	}
	return strings.Join(cleaned, separator)
}
