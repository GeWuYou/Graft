package network

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/http/httpproxy"

	"graft/server/internal/moduleapi"
)

// HTTPClientFactory 使用当前平台网络策略创建独立 HTTP client；它不会读取或修改进程环境变量。
type HTTPClientFactory struct {
	provider moduleapi.OutboundNetworkProvider
}

// NewHTTPClientFactory 创建出站 HTTP client factory。
func NewHTTPClientFactory(provider moduleapi.OutboundNetworkProvider) (*HTTPClientFactory, error) {
	if provider == nil {
		return nil, errors.New("outbound network provider is unavailable")
	}
	return &HTTPClientFactory{provider: provider}, nil
}

// NewOutboundHTTPClient 创建带有调用方行为选项的 HTTP client。每次创建都会读取当前策略，且绝不使用 ProxyFromEnvironment。
func (f *HTTPClientFactory) NewOutboundHTTPClient(ctx context.Context, options ...moduleapi.OutboundHTTPClientOption) (*http.Client, error) {
	if f == nil || f.provider == nil {
		return nil, errors.New("outbound HTTP client factory is unavailable")
	}
	policy, err := f.provider.CurrentOutboundNetworkPolicy(ctx)
	if err != nil {
		return nil, err
	}
	configured := moduleapi.OutboundHTTPClientOptions{}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&configured); err != nil {
			return nil, err
		}
	}
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("default HTTP transport has unexpected type")
	}
	cloned := transport.Clone()
	cloned.Proxy = proxyFunc(policy)
	return &http.Client{Transport: cloned, Timeout: configured.Timeout}, nil
}

func proxyFunc(policy moduleapi.OutboundNetworkPolicy) func(*http.Request) (*url.URL, error) {
	if !policy.Enabled {
		return nil
	}
	config := httpproxy.Config{HTTPProxy: policy.HTTPProxy, HTTPSProxy: policy.HTTPSProxy, NoProxy: joinNoProxy(policy.NoProxy)}
	resolver := config.ProxyFunc()
	return func(request *http.Request) (*url.URL, error) {
		if request == nil || request.URL == nil {
			return nil, nil
		}
		return resolver(request.URL)
	}
}

func joinNoProxy(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, ",")
}
