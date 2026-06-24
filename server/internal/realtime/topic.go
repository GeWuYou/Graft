package realtime

import "strings"

// NormalizeTopic 去除主题字符串首尾的空白字符。
// 返回去除首尾空白字符后的主题；如果输入全为空白，则返回空字符串。
func NormalizeTopic(topic string) string {
	return strings.TrimSpace(topic)
}
