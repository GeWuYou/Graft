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
	defaultTicketTTL      = 30 * time.Second
	minTicketSecretBytes  = 24
	defaultResourceType   = "unknown"
	defaultConsumeLeeway  = 0 * time.Second
)

var (
	ErrInvalidTicket     = errors.New("realtime ticket invalid")
	ErrExpiredTicket     = errors.New("realtime ticket expired")
	ErrUsedTicket        = errors.New("realtime ticket already used")
	ErrScopeMismatch     = errors.New("realtime ticket scope mismatch")
	ErrResourceMismatch  = errors.New("realtime ticket resource mismatch")
	ErrTicketRequired    = errors.New("realtime ticket required")
	ErrInvalidTicketSpec = errors.New("realtime ticket request invalid")
)

type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

type Service interface {
	Issue(ctx context.Context, req IssueRequest) (IssuedTicket, error)
	Consume(ctx context.Context, req ConsumeRequest) (ConsumedTicket, error)
}

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

type ConsumeRequest struct {
	Ticket       string
	ResourceType string
	ResourceID   string
	Scope        string
}

type IssuedTicket struct {
	TicketID    string
	Ticket      string
	SessionID   string
	ExpiresAt   time.Time
	UserID      uint64
	ResourceType string
	ResourceID  string
	Scope       string
	Command     string
	Cols        int
	Rows        int
}

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

func NewMemoryService() Service {
	return &memoryService{
		clock: systemClock{},
		store: make(map[string]storedTicket),
	}
}

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
	if record.expiresAt.Before(now) {
		delete(s.store, ticketID)
		return ConsumedTicket{}, ErrExpiredTicket
	}
	if record.usedAt != nil {
		return ConsumedTicket{}, ErrUsedTicket
	}
	if record.secretHash != hashTicketSecret(secret) {
		return ConsumedTicket{}, ErrInvalidTicket
	}
	if record.scope != scope {
		return ConsumedTicket{}, ErrScopeMismatch
	}
	if record.resourceID != resourceID || record.resourceType != resourceType {
		return ConsumedTicket{}, ErrResourceMismatch
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

func hashTicketSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func splitTicket(raw string) (string, string, error) {
	if raw == "" {
		return "", "", ErrTicketRequired
	}
	parts := strings.SplitN(raw, ".", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", ErrInvalidTicket
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}
