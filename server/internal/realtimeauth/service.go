package realtimeauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"graft/server/internal/kvx"
	kvkeys "graft/server/internal/kvx/keys"
)

const (
	defaultTicketTTL     = 30 * time.Second
	expiredTicketTTL     = 5 * time.Second
	minTicketSecretBytes = 24
	defaultResourceType  = "unknown"
	defaultConsumeLeeway = 0 * time.Second
	ticketPartCount      = 2
)

var (
	// ErrInvalidTicket 表示提供的实时票据格式错误或未知。
	ErrInvalidTicket = errors.New("realtime ticket invalid")
	// ErrExpiredTicket 表示提供的实时票据已过期。
	ErrExpiredTicket = errors.New("realtime ticket expired")
	// ErrUsedTicket 表示提供的实时票据已经消费过。
	ErrUsedTicket = errors.New("realtime ticket already used")
	// ErrScopeMismatch 表示票据 scope 与请求 scope 不匹配。
	ErrScopeMismatch = errors.New("realtime ticket scope mismatch")
	// ErrResourceMismatch 表示票据绑定的资源与请求不匹配。
	ErrResourceMismatch = errors.New("realtime ticket resource mismatch")
	// ErrTicketRequired 表示请求必须提供实时票据值。
	ErrTicketRequired = errors.New("realtime ticket required")
	// ErrInvalidTicketSpec 表示票据签发或消费请求未通过校验。
	ErrInvalidTicketSpec = errors.New("realtime ticket request invalid")
)

const (
	ticketStatusIssued = "issued"
	ticketStatusUsed   = "used"
)

// Clock 为票据签发和消费提供当前时间；测试可注入固定时钟以复现过期边界。
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// Service 负责签发和一次性消费短生命周期实时票据。
type Service interface {
	Issue(ctx context.Context, req IssueRequest) (IssuedTicket, error)
	Consume(ctx context.Context, req ConsumeRequest) (ConsumedTicket, error)
}

// IssueRequest 描述新签发实时票据要绑定的主体、资源、会话和客户端信息。
type IssueRequest struct {
	UserID       uint64
	ResourceType string
	ResourceID   string
	Scope        string
	SessionID    string
	ClientIP     string
	UserAgent    string
	Command      string
	Cols         int
	Rows         int
	TTL          time.Duration
}

// ConsumeRequest 标识消费时提供的票据及其预期资源绑定。
type ConsumeRequest struct {
	Ticket       string
	ResourceType string
	ResourceID   string
	Scope        string
}

// IssuedTicket 是签发成功后返回给调用方的票据载荷；Ticket 只应在建立实时连接时使用。
type IssuedTicket struct {
	TicketID     string
	Ticket       string
	SessionID    string
	ExpiresAt    time.Time
	UserID       uint64
	ResourceType string
	ResourceID   string
	Scope        string
	Command      string
	Cols         int
	Rows         int
}

// ConsumedTicket 是一次成功消费后返回给服务端连接建立流程的票据载荷。
type ConsumedTicket struct {
	TicketID     string
	SessionID    string
	UserID       uint64
	ResourceType string
	ResourceID   string
	Scope        string
	Command      string
	Cols         int
	Rows         int
	ClientIP     string
	UserAgent    string
	ExpiresAt    time.Time
}

type ticketRecord struct {
	TicketID     string     `json:"ticket_id"`
	SecretHash   string     `json:"secret_hash"`
	SessionID    string     `json:"session_id"`
	UserID       uint64     `json:"user_id"`
	ResourceType string     `json:"resource_type"`
	ResourceID   string     `json:"resource_id"`
	Scope        string     `json:"scope"`
	Command      string     `json:"command"`
	Cols         int        `json:"cols"`
	Rows         int        `json:"rows"`
	ClientIP     string     `json:"client_ip"`
	UserAgent    string     `json:"user_agent"`
	ExpiresAt    time.Time  `json:"expires_at"`
	Status       string     `json:"status"`
	UsedAt       *time.Time `json:"used_at,omitempty"`
}

// Options 配置一个基于 KV 的实时票据服务；Store 必须非空，其余字段可使用默认值。
type Options struct {
	Store      kvx.Store
	Clock      Clock
	KeyPrefix  string
	KeyBuilder func(ticketID string) string
}

type kvService struct {
	store      kvx.Store
	clock      Clock
	keyPrefix  string
	keyBuilder func(ticketID string) string
}

type consumeState struct {
	key          string
	secret       string
	scope        string
	resourceType string
	resourceID   string
	now          time.Time
}

type ticketSnapshot struct {
	item   kvx.Item
	record ticketRecord
}

// NewService 使用给定的 KV 存储创建一个实时票据服务，应用默认配置选项。
func NewService(store kvx.Store) (Service, error) {
	return NewServiceWithOptions(Options{Store: store})
}

// NewServiceWithOptions 使用给定选项创建基于 KV 的实时票据服务，并为未指定或空值应用默认值。
// Store 为空时返回错误；Clock 默认使用系统时钟，KeyPrefix 默认使用 realtimeauth/tickets，KeyBuilder 默认按前缀和票据 ID 拼接键。
func NewServiceWithOptions(options Options) (Service, error) {
	if options.Store == nil {
		return nil, errors.New("realtime ticket kv store is required")
	}

	clock := options.Clock
	if clock == nil {
		clock = systemClock{}
	}

	keyBuilder := options.KeyBuilder
	keyPrefix := strings.TrimSpace(options.KeyPrefix)
	if keyPrefix == "" {
		keyPrefix = kvkeys.Join("realtimeauth", "tickets")
	}
	if keyBuilder == nil {
		keyBuilder = func(ticketID string) string {
			return kvkeys.Join(keyPrefix, ticketID)
		}
	}

	return &kvService{
		store:      options.Store,
		clock:      clock,
		keyPrefix:  keyPrefix,
		keyBuilder: keyBuilder,
	}, nil
}

// NewMemoryService 创建由进程内 KV 存储支持的实时票据服务。
func NewMemoryService() Service {
	return NewMemoryServiceWithClock(nil)
}

// NewMemoryServiceWithClock 创建由进程内 KV 存储支持的实时票据服务，并使用指定时钟执行时间操作；clock 为空时使用系统时钟，初始化失败会 panic。
func NewMemoryServiceWithClock(clock Clock) Service {
	service, err := NewServiceWithOptions(Options{
		Store: kvx.NewMemory(kvx.MemoryOptions{Clock: clock}),
		Clock: clock,
	})
	if err != nil {
		panic(err)
	}
	return service
}

func (s *kvService) Issue(ctx context.Context, req IssueRequest) (IssuedTicket, error) {
	if req.UserID == 0 || strings.TrimSpace(req.ResourceID) == "" || strings.TrimSpace(req.Scope) == "" {
		return IssuedTicket{}, ErrInvalidTicketSpec
	}

	ttl := req.TTL
	if ttl <= 0 {
		ttl = defaultTicketTTL
	}

	resourceType := strings.TrimSpace(req.ResourceType)
	if resourceType == "" {
		resourceType = defaultResourceType
	}

	ticketID := uuid.NewString()
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = "shell_session_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	}

	secret, err := randomSecret(minTicketSecretBytes)
	if err != nil {
		return IssuedTicket{}, fmt.Errorf("generate realtime ticket secret: %w", err)
	}

	now := s.clock.Now()
	expiresAt := now.Add(ttl)
	record := ticketRecord{
		TicketID:     ticketID,
		SecretHash:   hashTicketSecret(secret),
		SessionID:    sessionID,
		UserID:       req.UserID,
		ResourceType: resourceType,
		ResourceID:   strings.TrimSpace(req.ResourceID),
		Scope:        strings.TrimSpace(req.Scope),
		Command:      strings.TrimSpace(req.Command),
		Cols:         req.Cols,
		Rows:         req.Rows,
		ClientIP:     strings.TrimSpace(req.ClientIP),
		UserAgent:    strings.TrimSpace(req.UserAgent),
		ExpiresAt:    expiresAt,
		Status:       ticketStatusIssued,
	}

	encoded, err := kvx.EncodeJSON(record)
	if err != nil {
		return IssuedTicket{}, fmt.Errorf("encode realtime ticket record: %w", err)
	}
	if err := s.store.Put(ctx, s.ticketKey(ticketID), encoded, ttl+expiredTicketTTL); err != nil {
		return IssuedTicket{}, fmt.Errorf("store realtime ticket record: %w", err)
	}

	return IssuedTicket{
		TicketID:     ticketID,
		Ticket:       ticketID + "." + secret,
		SessionID:    sessionID,
		ExpiresAt:    expiresAt,
		UserID:       req.UserID,
		ResourceType: resourceType,
		ResourceID:   record.ResourceID,
		Scope:        record.Scope,
		Command:      record.Command,
		Cols:         record.Cols,
		Rows:         record.Rows,
	}, nil
}

func (s *kvService) Consume(ctx context.Context, req ConsumeRequest) (ConsumedTicket, error) {
	state, err := s.newConsumeState(req)
	if err != nil {
		return ConsumedTicket{}, err
	}

	snapshot, ok, err := s.loadTicketSnapshot(ctx, state.key, "load")
	if err != nil {
		return ConsumedTicket{}, err
	}
	if !ok {
		return ConsumedTicket{}, ErrInvalidTicket
	}

	if err := s.validateTicketSnapshot(ctx, state, snapshot.record); err != nil {
		return ConsumedTicket{}, err
	}

	return s.consumeSnapshot(ctx, state, snapshot)
}

func (s *kvService) newConsumeState(req ConsumeRequest) (consumeState, error) {
	ticketID, secret, err := splitTicket(strings.TrimSpace(req.Ticket))
	if err != nil {
		return consumeState{}, err
	}

	scope := strings.TrimSpace(req.Scope)
	resourceID := strings.TrimSpace(req.ResourceID)
	resourceType := strings.TrimSpace(req.ResourceType)
	if scope == "" || resourceID == "" {
		return consumeState{}, ErrInvalidTicketSpec
	}
	if resourceType == "" {
		resourceType = defaultResourceType
	}

	return consumeState{
		key:          s.ticketKey(ticketID),
		secret:       secret,
		scope:        scope,
		resourceType: resourceType,
		resourceID:   resourceID,
		now:          s.clock.Now().Add(defaultConsumeLeeway),
	}, nil
}

func (s *kvService) loadTicketSnapshot(ctx context.Context, key string, operation string) (ticketSnapshot, bool, error) {
	item, ok, err := s.store.Get(ctx, key)
	if err != nil {
		return ticketSnapshot{}, false, fmt.Errorf("%s realtime ticket record: %w", operation, err)
	}
	if !ok {
		return ticketSnapshot{}, false, nil
	}

	record, err := decodeTicketRecord(item.Value)
	if err != nil {
		return ticketSnapshot{}, false, fmt.Errorf("decode realtime ticket record: %w", err)
	}

	return ticketSnapshot{item: item, record: record}, true, nil
}

func (s *kvService) validateTicketSnapshot(ctx context.Context, state consumeState, record ticketRecord) error {
	if record.ExpiresAt.Before(state.now) {
		_ = s.store.Delete(ctx, state.key)
		return ErrExpiredTicket
	}

	if err := validateStoredTicket(record, state.now, state.secret, state.scope, state.resourceType, state.resourceID); err != nil {
		if errors.Is(err, ErrExpiredTicket) {
			_ = s.store.Delete(ctx, state.key)
		}
		return err
	}

	return nil
}

func (s *kvService) consumeSnapshot(ctx context.Context, state consumeState, snapshot ticketSnapshot) (ConsumedTicket, error) {
	record := snapshot.record
	usedAt := state.now
	record.Status = ticketStatusUsed
	record.UsedAt = &usedAt

	replacement, err := kvx.EncodeJSON(record)
	if err != nil {
		return ConsumedTicket{}, fmt.Errorf("encode consumed realtime ticket record: %w", err)
	}

	remainingTTL := record.ExpiresAt.Sub(state.now)
	if remainingTTL <= 0 {
		_ = s.store.Delete(ctx, state.key)
		return ConsumedTicket{}, ErrExpiredTicket
	}

	swapped, err := s.store.CompareAndSwap(ctx, state.key, snapshot.item.Value, replacement, remainingTTL)
	if err != nil {
		return ConsumedTicket{}, fmt.Errorf("consume realtime ticket record: %w", err)
	}
	if !swapped {
		return s.resolveConcurrentConsume(ctx, state)
	}

	return consumedTicketFromRecord(record), nil
}

func (s *kvService) resolveConcurrentConsume(ctx context.Context, state consumeState) (ConsumedTicket, error) {
	snapshot, ok, err := s.loadTicketSnapshot(ctx, state.key, "reload")
	if err != nil {
		return ConsumedTicket{}, err
	}
	if !ok {
		return ConsumedTicket{}, ErrInvalidTicket
	}

	if err := s.validateTicketSnapshot(ctx, state, snapshot.record); err != nil {
		return ConsumedTicket{}, err
	}

	return ConsumedTicket{}, ErrUsedTicket
}

func (s *kvService) ticketKey(ticketID string) string {
	return s.keyBuilder(strings.TrimSpace(ticketID))
}

// decodeTicketRecord 将 JSON 字节解码为 ticketRecord。
func decodeTicketRecord(value []byte) (ticketRecord, error) {
	return kvx.DecodeJSON[ticketRecord](value)
}

// consumedTicketFromRecord 从票据记录字段构造消费成功后供连接流程使用的 ConsumedTicket。
func consumedTicketFromRecord(record ticketRecord) ConsumedTicket {
	return ConsumedTicket{
		TicketID:     record.TicketID,
		SessionID:    record.SessionID,
		UserID:       record.UserID,
		ResourceType: record.ResourceType,
		ResourceID:   record.ResourceID,
		Scope:        record.Scope,
		Command:      record.Command,
		Cols:         record.Cols,
		Rows:         record.Rows,
		ClientIP:     record.ClientIP,
		UserAgent:    record.UserAgent,
		ExpiresAt:    record.ExpiresAt,
	}
}

// randomSecret 生成指定大小的随机密钥字符串，并编码为十六进制。若 size 小于或等于零，则使用 minTicketSecretBytes。若随机字节生成失败，返回相应错误。
func randomSecret(size int) (string, error) {
	if size <= 0 {
		size = minTicketSecretBytes
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// hashTicketSecret 返回给定 secret 的 SHA-256 十六进制摘要；存储侧只保留摘要而不保留明文 secret。
func hashTicketSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// splitTicket 将票证字符串解析为 ticketID 和 secret 两个分量。票证必须采用 ticketID.secret 的格式（以点号分隔），返回 ErrTicketRequired 如果票证为空，返回 ErrInvalidTicket 如果格式无效或任一分量去空格后为空。
func splitTicket(raw string) (string, string, error) {
	if raw == "" {
		return "", "", ErrTicketRequired
	}
	parts := strings.SplitN(raw, ".", ticketPartCount)
	if len(parts) != ticketPartCount || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", ErrInvalidTicket
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

// validateStoredTicket 校验票据未过期、未消费、secret 正确，且 scope 与资源绑定均匹配；失败时返回对应的稳定票据错误。
func validateStoredTicket(
	record ticketRecord,
	now time.Time,
	secret string,
	scope string,
	resourceType string,
	resourceID string,
) error {
	if record.ExpiresAt.Before(now) {
		return ErrExpiredTicket
	}
	if record.Status == ticketStatusUsed || record.UsedAt != nil {
		return ErrUsedTicket
	}
	if record.SecretHash != hashTicketSecret(secret) {
		return ErrInvalidTicket
	}
	if record.Scope != scope {
		return ErrScopeMismatch
	}
	if record.ResourceID != resourceID || record.ResourceType != resourceType {
		return ErrResourceMismatch
	}
	return nil
}
