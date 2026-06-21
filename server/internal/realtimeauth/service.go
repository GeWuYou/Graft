// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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
	// ErrInvalidTicket indicates the supplied realtime ticket is malformed or unknown.
	ErrInvalidTicket = errors.New("realtime ticket invalid")
	// ErrExpiredTicket indicates the supplied realtime ticket has expired.
	ErrExpiredTicket = errors.New("realtime ticket expired")
	// ErrUsedTicket indicates the supplied realtime ticket was already consumed.
	ErrUsedTicket = errors.New("realtime ticket already used")
	// ErrScopeMismatch indicates the ticket scope does not match the requested scope.
	ErrScopeMismatch = errors.New("realtime ticket scope mismatch")
	// ErrResourceMismatch indicates the ticket resource binding does not match the request.
	ErrResourceMismatch = errors.New("realtime ticket resource mismatch")
	// ErrTicketRequired indicates a realtime ticket value is required.
	ErrTicketRequired = errors.New("realtime ticket required")
	// ErrInvalidTicketSpec indicates a ticket issue or consume request failed validation.
	ErrInvalidTicketSpec = errors.New("realtime ticket request invalid")
)

const (
	ticketStatusIssued = "issued"
	ticketStatusUsed   = "used"
)

// Clock provides the current time for ticket issuance and consumption.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// Service issues and consumes short-lived realtime tickets.
type Service interface {
	Issue(ctx context.Context, req IssueRequest) (IssuedTicket, error)
	Consume(ctx context.Context, req ConsumeRequest) (ConsumedTicket, error)
}

// IssueRequest describes the fields captured in a newly issued realtime ticket.
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

// ConsumeRequest identifies the ticket and resource binding expected during consumption.
type ConsumeRequest struct {
	Ticket       string
	ResourceType string
	ResourceID   string
	Scope        string
}

// IssuedTicket is the caller-visible ticket payload returned after issue succeeds.
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

// ConsumedTicket is the server-side ticket payload returned after one successful consume.
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

// Options configures one KV-backed realtime ticket service.
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

// NewServiceWithOptions creates one KV-backed realtime ticket service with explicit options.
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

// NewMemoryService creates a realtime ticket service backed by an in-process KV store.
func NewMemoryService() Service {
	return NewMemoryServiceWithClock(nil)
}

// NewMemoryServiceWithClock creates a realtime ticket service backed by an in-process KV store, using the provided clock for time operations. If clock is nil, the system clock is used. It panics if service initialization fails.
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

// decodeTicketRecord decodes JSON bytes into a ticketRecord.
func decodeTicketRecord(value []byte) (ticketRecord, error) {
	return kvx.DecodeJSON[ticketRecord](value)
}

// ConsumedTicketFromRecord constructs a ConsumedTicket from the fields of a ticket record.
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

// hashTicketSecret returns the SHA-256 digest of the given secret as a hex-encoded string.
func hashTicketSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// SplitTicket 将 ticket 字符串解析为 ticketID 和密钥两个组件。Ticket 必须采用 ticketID.secret 的格式（以点号分隔）。若 ticket 为空返回 ErrTicketRequired，若格式无效或任一组件为空返回 ErrInvalidTicket。
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

// validateStoredTicket checks that a ticket record has not expired, has not been consumed,
// has the correct secret, matches the given scope, and matches the given resource type and ID.
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
