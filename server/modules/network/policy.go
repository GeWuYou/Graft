package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/http/httpproxy"

	"graft/server/internal/moduleapi"
)

const outboundConfigKey = "network.outbound"

var errInvalidOutboundPolicy = errors.New("invalid outbound network policy")

type outboundConfig struct {
	Enabled        bool      `json:"enabled"`
	HTTPProxy      string    `json:"http_proxy"`
	HTTPSProxy     string    `json:"https_proxy"`
	NoProxy        []string  `json:"no_proxy"`
	Authentication *struct{} `json:"authentication,omitempty"`
}

// PolicyProvider 从 module-managed System Config 读取和验证当前平台出站策略。
type PolicyProvider struct {
	configs moduleapi.ModuleConfigManager
}

// NewPolicyProvider 创建基于 module-managed System Config 的策略读取器。
func NewPolicyProvider(configs moduleapi.ModuleConfigManager) (*PolicyProvider, error) {
	if configs == nil {
		return nil, errors.New("module config manager is unavailable")
	}
	return &PolicyProvider{configs: configs}, nil
}

// CurrentOutboundNetworkPolicy 返回当前有效策略。失效的已持久化值会返回错误，而不是静默退回环境变量或默认代理。
func (p *PolicyProvider) CurrentOutboundNetworkPolicy(ctx context.Context) (moduleapi.OutboundNetworkPolicy, error) {
	if p == nil || p.configs == nil {
		return moduleapi.OutboundNetworkPolicy{}, errors.New("outbound network policy provider is unavailable")
	}
	value, err := p.configs.GetModuleConfig(ctx, moduleID, outboundConfigKey)
	if err != nil {
		return moduleapi.OutboundNetworkPolicy{}, fmt.Errorf("read outbound network policy: %w", err)
	}
	return decodeOutboundPolicy(value.EffectiveValue)
}

func decodeOutboundPolicy(raw json.RawMessage) (moduleapi.OutboundNetworkPolicy, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return moduleapi.OutboundNetworkPolicy{}, fmt.Errorf("%w: decode JSON: %v", errInvalidOutboundPolicy, err)
	}
	for _, key := range []string{"enabled", "http_proxy", "https_proxy", "no_proxy"} {
		if _, ok := fields[key]; !ok {
			return moduleapi.OutboundNetworkPolicy{}, fmt.Errorf("%w: %s is required", errInvalidOutboundPolicy, key)
		}
	}
	for key := range fields {
		if key != "enabled" && key != "http_proxy" && key != "https_proxy" && key != "no_proxy" && key != "authentication" {
			return moduleapi.OutboundNetworkPolicy{}, fmt.Errorf("%w: %s is not allowed", errInvalidOutboundPolicy, key)
		}
	}
	var config outboundConfig
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return moduleapi.OutboundNetworkPolicy{}, fmt.Errorf("%w: decode JSON: %v", errInvalidOutboundPolicy, err)
	}
	if config.Authentication != nil {
		return moduleapi.OutboundNetworkPolicy{}, fmt.Errorf("%w: authentication is reserved for Secret Management", errInvalidOutboundPolicy)
	}
	httpProxy, err := validateProxyURL(config.HTTPProxy)
	if err != nil {
		return moduleapi.OutboundNetworkPolicy{}, err
	}
	httpsProxy, err := validateProxyURL(config.HTTPSProxy)
	if err != nil {
		return moduleapi.OutboundNetworkPolicy{}, err
	}
	noProxy, err := normalizeNoProxy(config.NoProxy)
	if err != nil {
		return moduleapi.OutboundNetworkPolicy{}, err
	}
	return moduleapi.OutboundNetworkPolicy{Enabled: config.Enabled, HTTPProxy: httpProxy, HTTPSProxy: httpsProxy, NoProxy: noProxy}, nil
}

func validateProxyURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil {
		return "", fmt.Errorf("%w: proxy URL is invalid", errInvalidOutboundPolicy)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%w: proxy URL scheme must be http or https", errInvalidOutboundPolicy)
	}
	if parsed.User != nil || parsed.Hostname() == "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" || parsed.Opaque != "" {
		return "", fmt.Errorf("%w: proxy URL must contain only scheme, host, and optional port", errInvalidOutboundPolicy)
	}
	if port := parsed.Port(); port != "" {
		value, err := strconv.ParseUint(port, 10, 16)
		if err != nil || value == 0 {
			return "", fmt.Errorf("%w: proxy URL port is invalid", errInvalidOutboundPolicy)
		}
	}
	return parsed.String(), nil
}

func normalizeNoProxy(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			return nil, fmt.Errorf("%w: no_proxy entries must not be empty", errInvalidOutboundPolicy)
		}
		if strings.ContainsAny(value, "\r\n, /@") {
			return nil, fmt.Errorf("%w: no_proxy entries must be individual values", errInvalidOutboundPolicy)
		}
		if err := validateNoProxyEntry(value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	// httpproxy parses the same matching grammar used by Go HTTP proxy resolution. Parsing once also rejects malformed CIDRs.
	config := httpproxy.Config{NoProxy: strings.Join(result, ",")}
	if config.ProxyFunc() == nil {
		return nil, fmt.Errorf("%w: no_proxy is invalid", errInvalidOutboundPolicy)
	}
	return result, nil
}

func validateNoProxyEntry(value string) error {
	if value == "*" {
		return nil
	}
	if strings.Contains(value, "/") {
		if _, _, err := net.ParseCIDR(value); err != nil {
			return fmt.Errorf("%w: no_proxy CIDR is invalid", errInvalidOutboundPolicy)
		}
		return nil
	}
	host := value
	if strings.HasPrefix(host, "*.") {
		host = strings.TrimPrefix(host, "*")
	} else if strings.Contains(host, "*") {
		return fmt.Errorf("%w: no_proxy wildcard is invalid", errInvalidOutboundPolicy)
	}
	if strings.HasPrefix(host, ".") {
		host = strings.TrimPrefix(host, ".")
	}
	if host == "" {
		return fmt.Errorf("%w: no_proxy host is invalid", errInvalidOutboundPolicy)
	}
	if strings.HasPrefix(host, "[") {
		parsedHost, port, err := net.SplitHostPort(host)
		if err != nil || net.ParseIP(parsedHost) == nil || !validPort(port) {
			return fmt.Errorf("%w: no_proxy host-port is invalid", errInvalidOutboundPolicy)
		}
		return nil
	}
	if ip := net.ParseIP(host); ip != nil {
		return nil
	}
	if candidate, port, ok := strings.Cut(host, ":"); ok {
		if strings.Contains(port, ":") || !validPort(port) {
			return fmt.Errorf("%w: no_proxy host-port is invalid", errInvalidOutboundPolicy)
		}
		host = candidate
	}
	if !validHostname(host) {
		return fmt.Errorf("%w: no_proxy host is invalid", errInvalidOutboundPolicy)
	}
	return nil
}

func validPort(value string) bool {
	port, err := strconv.ParseUint(value, 10, 16)
	return err == nil && port > 0
}

func validHostname(value string) bool {
	if len(value) > 253 || value == "" {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '-' {
				return false
			}
		}
	}
	return true
}
