// Package testassert 提供后端测试断言辅助函数。
package testassert

// SameStringSet 判断 actual 与 expected 是否包含同一组唯一字符串。
func SameStringSet(actual []string, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	seen := make(map[string]bool, len(actual))
	for _, value := range actual {
		seen[value] = true
	}
	for _, value := range expected {
		if !seen[value] {
			return false
		}
	}
	return true
}
