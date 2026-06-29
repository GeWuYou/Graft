package config

import (
	"fmt"
	"net/url"
)

// 返回首个无效条目的错误信息，全部通过时返回 nil。
func validateWebSocketAllowedOrigins(origins []string) error {
	for _, origin := range origins {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("invalid GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS entry %q", origin)
		}
	}
	return nil
}
