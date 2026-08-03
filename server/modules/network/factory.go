package network

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http/httpproxy"

	"graft/server/internal/moduleapi"
)

// HTTPClientFactory 使用当前平台网络策略创建独立 HTTP client；它不会读取或修改进程环境变量。
type HTTPClientFactory struct {
	provider  moduleapi.OutboundNetworkProvider
	mu        sync.Mutex
	policy    string
	transport *http.Transport
}

const (
	defaultDialTimeout           = 30 * time.Second
	defaultDialKeepAlive         = 30 * time.Second
	defaultMaxIdleConnections    = 100
	defaultIdleConnTimeout       = 90 * time.Second
	defaultTLSHandshakeTimeout   = 10 * time.Second
	defaultExpectContinueTimeout = 1 * time.Second
)

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
	transport, err := f.transportForPolicy(policy)
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: transport, Timeout: configured.Timeout}, nil
}

func (f *HTTPClientFactory) transportForPolicy(policy moduleapi.OutboundNetworkPolicy) (*http.Transport, error) {
	// 按完整策略指纹复用连接池；策略热更新时只关闭旧 transport 的空闲连接，避免影响已在途请求。
	fingerprint := outboundPolicyFingerprint(policy)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.transport != nil && f.policy == fingerprint {
		return f.transport, nil
	}
	transport := newDefaultTransport()
	transport.Proxy = proxyFunc(policy)
	previous := f.transport
	f.transport, f.policy = transport, fingerprint
	if previous != nil {
		previous.CloseIdleConnections()
	}
	return transport, nil
}

func newDefaultTransport() *http.Transport {
	return &http.Transport{
		DialContext:           (&net.Dialer{Timeout: defaultDialTimeout, KeepAlive: defaultDialKeepAlive}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          defaultMaxIdleConnections,
		IdleConnTimeout:       defaultIdleConnTimeout,
		TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
		ExpectContinueTimeout: defaultExpectContinueTimeout,
	}
}

func outboundPolicyFingerprint(policy moduleapi.OutboundNetworkPolicy) string {
	return strings.Join([]string{strconv.FormatBool(policy.Enabled), policy.HTTPProxy, policy.HTTPSProxy, strings.Join(policy.NoProxy, "\x00")}, "\x00")
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
