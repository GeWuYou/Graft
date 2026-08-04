package moduleapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"
)

// ErrModuleConfigVersionConflict 表示模块配置的 If-Match 版本已过期。
var ErrModuleConfigVersionConflict = errors.New("module config version conflict")

// ModuleConfigValue 是模块管理配置的有效 JSON 值与覆盖状态；它不暴露 System Config 的存储实现。
type ModuleConfigValue struct {
	EffectiveValue json.RawMessage
	DefaultValue   json.RawMessage
	OverrideValue  json.RawMessage
	HasOverride    bool
	// Version 是 Module Config 的单调递增版本，用于构造强 ETag 和 HTTP If-Match 条件。
	Version       int64
	UpdatedAt     *time.Time
	UpdatedByName string
}

// ModuleConfigManager 仅向配置声明 owner 暴露模块管理配置的读取和写入能力。
//
// 调用方必须传入自己的稳定模块标识和配置 key。实现会拒绝非 module-managed 配置及 owner 不匹配的访问，
// 防止业务模块绕过 System Config 的通用 API 或互相修改配置。
type ModuleConfigManager interface {
	GetModuleConfig(ctx context.Context, moduleName string, key string) (ModuleConfigValue, error)
	UpdateModuleConfig(ctx context.Context, moduleName string, key string, value json.RawMessage, userID *uint64, expectedVersion int64) (ModuleConfigValue, error)
	ResetModuleConfig(ctx context.Context, moduleName string, key string, userID *uint64, expectedVersion int64) (ModuleConfigValue, error)
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
	CurrentOutboundNetworkPolicy(ctx context.Context) (OutboundNetworkPolicy, error)
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
	NewOutboundHTTPClient(ctx context.Context, options ...OutboundHTTPClientOption) (*http.Client, error)
}

// OutboundDiagnosticTarget 是固定、注册式出站连通性诊断目标。
//
// Execute 不接收管理员提供的 URL，避免诊断接口成为 SSRF 入口。
type OutboundDiagnosticTarget interface {
	Name() string
	DisplayName() string
	ExecuteOutboundDiagnostic(ctx context.Context) (OutboundDiagnosticResult, error)
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
	RegisterOutboundDiagnosticTarget(target OutboundDiagnosticTarget) error
	OutboundDiagnosticTarget(name string) (OutboundDiagnosticTarget, bool)
	OutboundDiagnosticTargets() []OutboundDiagnosticTarget
	ConnectivityTargetRegistry
}

// ConnectivityTargetID 是连通性目标的稳定身份，由 owning module 声明并供诊断页面、历史和报告关联使用。
type ConnectivityTargetID string

// ConnectivityProbeKind 标识可扩展的诊断阶段。新增协议阶段只需声明新的 kind，无需改变报告存储形状。
type ConnectivityProbeKind string

const (
	maxConnectivitySummaryRunes    = 512
	maxConnectivityErrorCodeLength = 128
	//nolint:revive // 同一组 Probe 常量共享紧邻的类型语义注释。
	ConnectivityProbeDNS           ConnectivityProbeKind = "dns"
	ConnectivityProbeTCP           ConnectivityProbeKind = "tcp"
	ConnectivityProbeTLS           ConnectivityProbeKind = "tls"
	ConnectivityProbeCertificate   ConnectivityProbeKind = "certificate"
	ConnectivityProbeHTTP          ConnectivityProbeKind = "http"
	ConnectivityProbeSMTPBanner    ConnectivityProbeKind = "smtp_banner"
	ConnectivityProbeSMTPEHLO      ConnectivityProbeKind = "smtp_ehlo"
	ConnectivityProbeLDAPBind      ConnectivityProbeKind = "ldap_bind"
	ConnectivityProbeOIDCDiscovery ConnectivityProbeKind = "oidc_discovery"
	ConnectivityProbeWebhookPost   ConnectivityProbeKind = "webhook_post"
	ConnectivityProbeOCIPing       ConnectivityProbeKind = "oci_ping"
)

// ConnectivityTargetFeature 标识目标可安全提供的报告能力，而不是某个固定协议的实现细节。
type ConnectivityTargetFeature string

const (
	//nolint:revive // 同一组 feature 常量共享紧邻的类型语义注释。
	ConnectivityFeatureHistory    ConnectivityTargetFeature = "history"
	ConnectivityFeatureExport     ConnectivityTargetFeature = "export"
	ConnectivityFeatureExitIP     ConnectivityTargetFeature = "exit_ip"
	ConnectivityFeatureProxyRoute ConnectivityTargetFeature = "proxy_route"
)

// ConnectivityTargetCapabilities 描述目标可执行的探测和可展示的报告能力。
type ConnectivityTargetCapabilities struct {
	ProbeKinds []ConnectivityProbeKind
	Features   []ConnectivityTargetFeature
}

// ConnectivityTargetDescriptor 是 module 注册的连通性目标声明，不携带可执行 URL、凭据或其他秘密。
type ConnectivityTargetDescriptor struct {
	ID           ConnectivityTargetID
	ModuleID     string
	Category     string
	TitleKey     string
	Capabilities ConnectivityTargetCapabilities
}

// Snapshot 返回不会与模块实现共享 capability slice 的目标描述副本。
func (d ConnectivityTargetDescriptor) Snapshot() ConnectivityTargetDescriptor {
	d.Capabilities.ProbeKinds = append([]ConnectivityProbeKind(nil), d.Capabilities.ProbeKinds...)
	d.Capabilities.Features = append([]ConnectivityTargetFeature(nil), d.Capabilities.Features...)
	return d
}

// ConnectivityTarget 是可扩展连通性诊断目标的 canonical module boundary。
//
// 实现必须只返回经净化的报告；它不能把原始请求、响应、凭据、代理地址或完整出口 IP 放入报告。
type ConnectivityTarget interface {
	ConnectivityTargetDescriptor() ConnectivityTargetDescriptor
	RunConnectivityProbes(ctx context.Context) (ConnectivityReport, error)
}

// ConnectivityRouteDestination 由能够安全说明其请求主机的目标实现，供 Network 生成策略解释。
// 它不携带 URL、端口、凭据或代理地址，也不进入持久化报告。
type ConnectivityRouteDestination interface {
	ConnectivityRouteHost() string
}

// ConnectivityTargetRegistry 注册 canonical 连通性目标，并向消费者提供稳定排序的描述快照。
type ConnectivityTargetRegistry interface {
	RegisterConnectivityTarget(target ConnectivityTarget) error
	ConnectivityTarget(id ConnectivityTargetID) (ConnectivityTarget, bool)
	ConnectivityTargets() []ConnectivityTarget
	ConnectivityTargetDescriptors() []ConnectivityTargetDescriptor
}

// ConnectivityReportStatus 是一份连通性报告的总体状态。
type ConnectivityReportStatus string

const (
	//nolint:revive // 同一组 report status 常量共享紧邻的类型语义注释。
	ConnectivityReportStatusHealthy  ConnectivityReportStatus = "healthy"
	ConnectivityReportStatusDegraded ConnectivityReportStatus = "degraded"
	ConnectivityReportStatusFailed   ConnectivityReportStatus = "failed"
)

// ProbeStatus 是单个 Probe 的完成状态。
type ProbeStatus string

const (
	//nolint:revive // 同一组 Probe status 常量共享紧邻的类型语义注释。
	ProbeStatusSucceeded ProbeStatus = "succeeded"
	ProbeStatusFailed    ProbeStatus = "failed"
	ProbeStatusSkipped   ProbeStatus = "skipped"
)

// ProbeResult 是无原始网络载荷的单个诊断阶段结果。
type ProbeResult struct {
	Kind       ConnectivityProbeKind
	Status     ProbeStatus
	Duration   time.Duration
	HTTPStatus *int
	Summary    string
	ErrorCode  string
	OccurredAt time.Time
}

// RouteExplanation 解释平台出站策略的决策，不暴露代理端点、凭据或规则树。
type RouteExplanation struct {
	MatchedStrategy string
	Decision        string
	Reason          string
}

// ExitIPDisclosure 只保存用于普通展示的掩码出口 IP；完整值仅能由未来受权限保护的实时查询提供。
type ExitIPDisclosure struct {
	Masked    string
	Available bool
}

// ConnectivityReport 是版本化、可扩展且经净化的诊断报告 envelope。
type ConnectivityReport struct {
	SchemaVersion int
	TargetID      ConnectivityTargetID
	Status        ConnectivityReportStatus
	CheckedAt     time.Time
	TotalLatency  time.Duration
	Probes        []ProbeResult
	Route         *RouteExplanation
	ExitIP        *ExitIPDisclosure
}

// NewConnectivityReport 构造净化后的报告快照。该边界没有原始报文或完整出口 IP 字段，避免未来持久化层误接收敏感网络数据。
//
//nolint:revive // 该构造函数完整保留固定的跨模块报告字段，拆分参数会弱化调用处审计性。
func NewConnectivityReport(targetID ConnectivityTargetID, status ConnectivityReportStatus, checkedAt time.Time, totalLatency time.Duration, probes []ProbeResult, route *RouteExplanation, exitIP *ExitIPDisclosure) ConnectivityReport {
	probeSnapshots := make([]ProbeResult, 0, len(probes))
	for _, probe := range probes {
		if probe.Duration < 0 {
			probe.Duration = 0
		}
		if probe.Kind != ConnectivityProbeHTTP || probe.HTTPStatus == nil || *probe.HTTPStatus < 100 || *probe.HTTPStatus > 599 {
			probe.HTTPStatus = nil
		} else {
			httpStatus := *probe.HTTPStatus
			probe.HTTPStatus = &httpStatus
		}
		probe.Summary = sanitizeConnectivityText(probe.Summary)
		probe.ErrorCode = sanitizeConnectivityErrorCode(probe.ErrorCode)
		probe.OccurredAt = probe.OccurredAt.UTC()
		probeSnapshots = append(probeSnapshots, probe)
	}
	report := ConnectivityReport{SchemaVersion: 1, TargetID: ConnectivityTargetID(strings.TrimSpace(string(targetID))), Status: status, CheckedAt: checkedAt.UTC(), TotalLatency: maxDuration(totalLatency, 0), Probes: probeSnapshots}
	if route != nil {
		routeSnapshot := RouteExplanation{MatchedStrategy: sanitizeConnectivityText(route.MatchedStrategy), Decision: sanitizeConnectivityText(route.Decision), Reason: sanitizeConnectivityText(route.Reason)}
		report.Route = &routeSnapshot
	}
	if exitIP != nil {
		if masked := sanitizeMaskedExitIP(exitIP.Masked); masked != "" {
			report.ExitIP = &ExitIPDisclosure{Masked: masked, Available: exitIP.Available}
		}
	}
	return report
}

// Snapshot 返回不会与调用方共享 slice 或指针字段的报告副本。
func (r ConnectivityReport) Snapshot() ConnectivityReport {
	return NewConnectivityReport(r.TargetID, r.Status, r.CheckedAt, r.TotalLatency, r.Probes, r.Route, r.ExitIP)
}

func maxDuration(value, minimum time.Duration) time.Duration {
	if value < minimum {
		return minimum
	}
	return value
}

func sanitizeConnectivityText(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(value))
	if containsSensitiveConnectivityContent(value) {
		return "sanitized diagnostic detail"
	}
	runes := []rune(value)
	if len(runes) > maxConnectivitySummaryRunes {
		return string(runes[:maxConnectivitySummaryRunes])
	}
	return value
}

//nolint:cyclop // 错误码仅允许稳定 ASCII 字符的白名单需保持直观可审计。
func sanitizeConnectivityErrorCode(value string) string {
	value = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			return r
		}
		return -1
	}, strings.TrimSpace(value))
	if len(value) > maxConnectivityErrorCodeLength {
		return value[:maxConnectivityErrorCodeLength]
	}
	return value
}

func sanitizeMaskedExitIP(value string) string {
	value = sanitizeConnectivityText(value)
	if !strings.ContainsAny(value, "*•") {
		return ""
	}
	return value
}

func containsSensitiveConnectivityContent(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"authorization:", "proxy-authorization:", "password=", "token=", "secret=", "://"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// OutboundNetworkConsumer 是显式声明使用平台出站网络策略的模块消费者。
// 它与诊断目标分离，避免没有诊断能力的消费者从页面运行态中消失。
type OutboundNetworkConsumer interface {
	Name() string
	DisplayName() string
}

// OutboundNetworkConsumerRegistry 注册使用平台策略的模块消费者。
type OutboundNetworkConsumerRegistry interface {
	RegisterOutboundNetworkConsumer(consumer OutboundNetworkConsumer) error
	OutboundNetworkConsumers() []OutboundNetworkConsumer
}
