package moduleapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// ModuleConfigValue 是模块管理配置的有效 JSON 值与覆盖状态；它不暴露 System Config 的存储实现。
type ModuleConfigValue struct {
	EffectiveValue json.RawMessage
	DefaultValue   json.RawMessage
	OverrideValue  json.RawMessage
	HasOverride    bool
	UpdatedAt      *time.Time
	UpdatedByName  string
}

// ModuleConfigManager 仅向配置声明 owner 暴露模块管理配置的读取和写入能力。
//
// 调用方必须传入自己的稳定模块标识和配置 key。实现会拒绝非 module-managed 配置及 owner 不匹配的访问，
// 防止业务模块绕过 System Config 的通用 API 或互相修改配置。
type ModuleConfigManager interface {
	GetModuleConfig(context.Context, string, string) (ModuleConfigValue, error)
	UpdateModuleConfig(context.Context, string, string, json.RawMessage, *uint64) (ModuleConfigValue, error)
	ResetModuleConfig(context.Context, string, string) (ModuleConfigValue, error)
}

// OutboundNetworkPolicy 是平台 HTTP(S) 出站策略，不包含超时、重试或其他 HTTP client 行为。
type OutboundNetworkPolicy struct {
	Enabled    bool
	HTTPProxy  string
	HTTPSProxy string
	NoProxy    []string
}

// OutboundNetworkProvider 读取当前有效的出站网络策略。
//
// 该能力只负责平台代理和绕过策略；Docker daemon、SMTP 和浏览器流量不属于此边界。
type OutboundNetworkProvider interface {
	CurrentOutboundNetworkPolicy(context.Context) (OutboundNetworkPolicy, error)
}

// OutboundHTTPClientOptions 是 HTTP client 行为选项。网络策略本身不承载这些字段。
type OutboundHTTPClientOptions struct {
	Timeout time.Duration
}

// OutboundHTTPClientOption 调整单个新建 client 的调用行为。
type OutboundHTTPClientOption func(*OutboundHTTPClientOptions) error

// WithTimeout 为新建 client 指定请求总超时；非正值会被 factory 拒绝。
func WithTimeout(timeout time.Duration) OutboundHTTPClientOption {
	return func(options *OutboundHTTPClientOptions) error {
		if timeout <= 0 {
			return ErrInvalidOutboundHTTPClientOption
		}
		options.Timeout = timeout
		return nil
	}
}

// ErrInvalidOutboundHTTPClientOption 表示调用方提供了不合法的 HTTP client 行为选项。
var ErrInvalidOutboundHTTPClientOption = outboundHTTPClientOptionError("invalid outbound HTTP client option")

type outboundHTTPClientOptionError string

func (e outboundHTTPClientOptionError) Error() string { return string(e) }

// OutboundHTTPClientFactory 为平台主动 HTTP(S) 请求创建应用当前网络策略的 client。
type OutboundHTTPClientFactory interface {
	NewOutboundHTTPClient(context.Context, ...OutboundHTTPClientOption) (*http.Client, error)
}

// OutboundDiagnosticTarget 是固定、注册式出站连通性诊断目标。
//
// Execute 不接收管理员提供的 URL，避免诊断接口成为 SSRF 入口。
type OutboundDiagnosticTarget interface {
	Name() string
	DisplayName() string
	ExecuteOutboundDiagnostic(context.Context) (OutboundDiagnosticResult, error)
}

// OutboundDiagnosticResult 是不含请求 URL、代理地址或凭据的诊断结果。
type OutboundDiagnosticResult struct {
	Connected  bool
	Latency    time.Duration
	HTTPStatus int
	TestedAt   time.Time
	Message    string
}

// OutboundDiagnosticRegistry 注册固定的网络诊断目标。
type OutboundDiagnosticRegistry interface {
	RegisterOutboundDiagnosticTarget(OutboundDiagnosticTarget) error
	OutboundDiagnosticTarget(string) (OutboundDiagnosticTarget, bool)
	OutboundDiagnosticTargets() []OutboundDiagnosticTarget
}

// OutboundNetworkConsumer 是显式声明使用平台出站网络策略的模块消费者。
// 它与诊断目标分离，避免没有诊断能力的消费者从页面运行态中消失。
type OutboundNetworkConsumer interface {
	Name() string
	DisplayName() string
}

// OutboundNetworkConsumerRegistry 注册使用平台策略的模块消费者。
type OutboundNetworkConsumerRegistry interface {
	RegisterOutboundNetworkConsumer(OutboundNetworkConsumer) error
	OutboundNetworkConsumers() []OutboundNetworkConsumer
}
