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
	// ErrTopicRequired reports that the realtime topic is missing.
	ErrTopicRequired   = errors.New("realtime topic required")
	// ErrTopicNotFound reports that no issuer owns the requested topic.
	ErrTopicNotFound   = errors.New("realtime topic not found")
	// ErrTopicForbidden reports that the caller cannot subscribe to the requested topic.
	ErrTopicForbidden  = errors.New("realtime topic forbidden")
	// ErrTopicConflict reports a transient failure while preparing the topic subscription.
	ErrTopicConflict   = errors.New("realtime topic unavailable")
	// ErrIssuerRequired reports that a topic issuer dependency is missing.
	ErrIssuerRequired  = errors.New("realtime subscription issuer is required")
	// ErrDuplicateIssuer reports that the same topic prefix was registered more than once.
	ErrDuplicateIssuer = errors.New("realtime subscription issuer already registered")
)

// SubscriptionRequest carries normalized request context for topic subscription issuance.
type SubscriptionRequest struct {
	Topic       string
	RequestAuth moduleapi.RequestAuthContext
	ClientIP    string
	UserAgent   string
}

// SubscriptionResponse returns the websocket bootstrap data for a realtime topic.
type SubscriptionResponse struct {
	Topic        string    `json:"topic"`
	Ticket       string    `json:"ticket"`
	WebSocketURL string    `json:"websocket_url"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// SubscriptionIssuer issues websocket bootstrap data for one bounded topic family.
type SubscriptionIssuer interface {
	IssueSubscription(ctx context.Context, request SubscriptionRequest) (SubscriptionResponse, error)
}

// TopicIssuerRegistry resolves topic prefixes to their owning issuer.
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

// TicketIssuer delegates bounded websocket ticket issuance to the realtime auth service.
type TicketIssuer struct {
	Tickets realtimeauth.Service
}

// IssueTopicTicket issues a bounded websocket ticket for the requested topic.
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
