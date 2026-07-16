package realtime

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtimeauth"
)

const defaultSubscriptionTicketTTL = 30 * time.Second
const initialTopicIssuerCapacity = 4

var (
	// ErrTopicRequired 表示缺少实时主题。
	ErrTopicRequired   = errors.New("realtime topic required")
	// ErrTopicNotFound 表示没有签发器拥有请求的主题。
	ErrTopicNotFound   = errors.New("realtime topic not found")
	// ErrTopicForbidden 表示调用方无权订阅请求的主题。
	ErrTopicForbidden  = errors.New("realtime topic forbidden")
	// ErrTopicConflict 表示准备主题订阅时发生暂时性冲突。
	ErrTopicConflict   = errors.New("realtime topic unavailable")
	// ErrIssuerRequired 表示缺少主题签发器依赖。
	ErrIssuerRequired  = errors.New("realtime subscription issuer is required")
	// ErrDuplicateIssuer 表示同一主题前缀被重复注册。
	ErrDuplicateIssuer = errors.New("realtime subscription issuer already registered")
)

// SubscriptionRequest 携带签发主题订阅信息所需的规范化请求上下文。
type SubscriptionRequest struct {
	Topic       string
	RequestAuth moduleapi.RequestAuthContext
	ClientIP    string
	UserAgent   string
}

// SubscriptionResponse 返回实时主题的 WebSocket 引导数据。
type SubscriptionResponse struct {
	Topic        string    `json:"topic"`
	Ticket       string    `json:"ticket"`
	WebSocketURL string    `json:"websocket_url"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// SubscriptionIssuer 为一个受限主题族签发 WebSocket 引导数据。
type SubscriptionIssuer interface {
	IssueSubscription(ctx context.Context, request SubscriptionRequest) (SubscriptionResponse, error)
}

// TopicIssuerRegistry 将主题前缀解析到其拥有者签发器。
type TopicIssuerRegistry interface {
	Register(prefix string, issuer SubscriptionIssuer) error
	Resolve(topic string) (SubscriptionIssuer, bool)
}

type topicIssuerRegistry struct {
	mu      sync.RWMutex
	entries []topicIssuerEntry
}

type topicIssuerEntry struct {
	prefix string
	issuer SubscriptionIssuer
}

// NewTopicIssuerRegistry 创建用于统一实时网关的内存主题签发器注册表。
// 它返回一个已初始化的注册表，初始容量为 `initialTopicIssuerCapacity`。
func NewTopicIssuerRegistry() TopicIssuerRegistry {
	return &topicIssuerRegistry{
		entries: make([]topicIssuerEntry, 0, initialTopicIssuerCapacity),
	}
}

func (r *topicIssuerRegistry) Register(prefix string, issuer SubscriptionIssuer) error {
	normalizedPrefix := NormalizeTopic(prefix)
	if normalizedPrefix == "" {
		return ErrTopicRequired
	}
	if issuer == nil {
		return ErrIssuerRequired
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, entry := range r.entries {
		if entry.prefix == normalizedPrefix {
			return fmt.Errorf("%w: %s", ErrDuplicateIssuer, normalizedPrefix)
		}
	}

	r.entries = append(r.entries, topicIssuerEntry{
		prefix: normalizedPrefix,
		issuer: issuer,
	})
	return nil
}

func (r *topicIssuerRegistry) Resolve(topic string) (SubscriptionIssuer, bool) {
	normalizedTopic := NormalizeTopic(topic)
	if normalizedTopic == "" {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched SubscriptionIssuer
	longestPrefix := 0
	for _, entry := range r.entries {
		if !strings.HasPrefix(normalizedTopic, entry.prefix) {
			continue
		}
		if len(entry.prefix) <= longestPrefix {
			continue
		}
		longestPrefix = len(entry.prefix)
		matched = entry.issuer
	}

	return matched, matched != nil
}

// TicketIssuer 将受限 WebSocket 票据签发委托给实时认证服务。
type TicketIssuer struct {
	Tickets realtimeauth.Service
}

// IssueTopicTicket 为请求的主题签发受限 WebSocket 票据。
func (i TicketIssuer) IssueTopicTicket(
	ctx context.Context,
	request SubscriptionRequest,
) (realtimeauth.IssuedTicket, error) {
	if i.Tickets == nil {
		return realtimeauth.IssuedTicket{}, ErrIssuerRequired
	}
	if request.RequestAuth.User == nil {
		return realtimeauth.IssuedTicket{}, ErrTopicForbidden
	}

	issued, err := i.Tickets.Issue(ctx, realtimeauth.IssueRequest{
		UserID:       request.RequestAuth.User.ID,
		ResourceType: WebSocketTopicResourceType,
		ResourceID:   request.Topic,
		Scope:        WebSocketTopicScope,
		ClientIP:     request.ClientIP,
		UserAgent:    request.UserAgent,
		TTL:          defaultSubscriptionTicketTTL,
	})
	if err != nil {
		return realtimeauth.IssuedTicket{}, err
	}
	return issued, nil
}

// BuildTopicWebSocketURL 生成指定 topic 与 ticket 对应的标准 WebSocket 地址。
// 地址固定为 "/ws"，并将 topic 和 ticket 编码为查询参数。
func BuildTopicWebSocketURL(topic string, ticket string) string {
	values := url.Values{}
	values.Set("topic", topic)
	values.Set("ticket", ticket)
	return "/ws?" + values.Encode()
}

// BuildSubscriptionRequest 构建实时订阅请求并填充规范化的上下文信息。
// 它会规范化 topic，提取请求认证信息，并收集客户端 IP 和 User-Agent。
func BuildSubscriptionRequest(ctx context.Context, topic string) SubscriptionRequest {
	request := SubscriptionRequest{
		Topic: NormalizeTopic(topic),
	}
	if requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx); ok {
		request.RequestAuth = requestAuth
	}
	request.ClientIP = strings.TrimSpace(currentRequestClientIP(ctx))
	request.UserAgent = strings.TrimSpace(currentRequestUserAgent(ctx))
	return request
}

// currentRequestClientIP 从上下文中提取客户端 IP 并返回其去除首尾空白后的值。
func currentRequestClientIP(ctx context.Context) string {
	requestAudit, ok := httpx.RequestAuditContextFromContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(requestAudit.ClientIP)
}

// currentRequestUserAgent 返回请求审计上下文中的 User-Agent，并去除首尾空白。
func currentRequestUserAgent(ctx context.Context) string {
	requestAudit, ok := httpx.RequestAuditContextFromContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(requestAudit.UserAgent)
}
