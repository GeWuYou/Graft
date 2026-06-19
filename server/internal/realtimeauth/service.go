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
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultTicketTTL     = 30 * time.Second
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

type storedTicket struct {
	ticketID     string
	secretHash   string
	sessionID    string
	userID       uint64
	resourceType string
	resourceID   string
	scope        string
	command      string
	cols         int
	rows         int
	clientIP     string
	userAgent    string
	expiresAt    time.Time
	usedAt       *time.Time
}

type memoryService struct {
	clock Clock
	mu    sync.Mutex
	store map[string]storedTicket
}

// NewMemoryService creates an in-memory realtime ticket service backed by the system clock.
func NewMemoryService() Service {
	return &memoryService{
		clock: systemClock{},
		store: make(map[string]storedTicket),
	}
}

// NewMemoryServiceWithClock creates an in-memory realtime ticket service using the provided clock, or the system clock if clock is nil.
func NewMemoryServiceWithClock(clock Clock) Service {
	if clock == nil {
		clock = systemClock{}
	}
	return &memoryService{
		clock: clock,
		store: make(map[string]storedTicket),
	}
}

func (s *memoryService) Issue(_ context.Context, req IssueRequest) (IssuedTicket, error) {
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
	record := storedTicket{
		ticketID:     ticketID,
		secretHash:   hashTicketSecret(secret),
		sessionID:    sessionID,
		userID:       req.UserID,
		resourceType: resourceType,
		resourceID:   strings.TrimSpace(req.ResourceID),
		scope:        strings.TrimSpace(req.Scope),
		command:      strings.TrimSpace(req.Command),
		cols:         req.Cols,
		rows:         req.Rows,
		clientIP:     strings.TrimSpace(req.ClientIP),
		userAgent:    strings.TrimSpace(req.UserAgent),
		expiresAt:    expiresAt,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneExpiredLocked(now)
	s.store[ticketID] = record

	return IssuedTicket{
		TicketID:     ticketID,
		Ticket:       ticketID + "." + secret,
		SessionID:    sessionID,
		ExpiresAt:    expiresAt,
		UserID:       req.UserID,
		ResourceType: resourceType,
		ResourceID:   record.resourceID,
		Scope:        record.scope,
		Command:      record.command,
		Cols:         record.cols,
		Rows:         record.rows,
	}, nil
}

func (s *memoryService) Consume(_ context.Context, req ConsumeRequest) (ConsumedTicket, error) {
	ticketID, secret, err := splitTicket(strings.TrimSpace(req.Ticket))
	if err != nil {
		return ConsumedTicket{}, err
	}
	scope := strings.TrimSpace(req.Scope)
	resourceID := strings.TrimSpace(req.ResourceID)
	resourceType := strings.TrimSpace(req.ResourceType)
	if scope == "" || resourceID == "" {
		return ConsumedTicket{}, ErrInvalidTicketSpec
	}
	if resourceType == "" {
		resourceType = defaultResourceType
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock.Now().Add(defaultConsumeLeeway)
	s.pruneExpiredLocked(now)
	record, ok := s.store[ticketID]
	if !ok {
		return ConsumedTicket{}, ErrInvalidTicket
	}
	if err := validateStoredTicket(record, now, secret, scope, resourceType, resourceID); err != nil {
		if errors.Is(err, ErrExpiredTicket) {
			delete(s.store, ticketID)
		}
		return ConsumedTicket{}, err
	}
	usedAt := now
	record.usedAt = &usedAt
	s.store[ticketID] = record

	return ConsumedTicket{
		TicketID:     record.ticketID,
		SessionID:    record.sessionID,
		UserID:       record.userID,
		ResourceType: record.resourceType,
		ResourceID:   record.resourceID,
		Scope:        record.scope,
		Command:      record.command,
		Cols:         record.cols,
		Rows:         record.rows,
		ClientIP:     record.clientIP,
		UserAgent:    record.userAgent,
		ExpiresAt:    record.expiresAt,
	}, nil
}

func (s *memoryService) pruneExpiredLocked(now time.Time) {
	for key, item := range s.store {
		if item.expiresAt.Before(now) {
			delete(s.store, key)
		}
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

// validateStoredTicket validates a stored ticket against the current time, secret, scope, and resource binding.
func validateStoredTicket(
	record storedTicket,
	now time.Time,
	secret string,
	scope string,
	resourceType string,
	resourceID string,
) error {
	if record.expiresAt.Before(now) {
		return ErrExpiredTicket
	}
	if record.usedAt != nil {
		return ErrUsedTicket
	}
	if record.secretHash != hashTicketSecret(secret) {
		return ErrInvalidTicket
	}
	if record.scope != scope {
		return ErrScopeMismatch
	}
	if record.resourceID != resourceID || record.resourceType != resourceType {
		return ErrResourceMismatch
	}
	return nil
}
