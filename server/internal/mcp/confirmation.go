package mcp

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"graft/server/internal/moduleapi"
)

const (
	confirmationTokenPrefix = "graft_mcp_confirm_" // #nosec G101 -- 固定公开前缀不包含凭据或熵。
	confirmationTokenBytes  = 32
)

var (
	// errConfirmationTokenInvalid 表示确认令牌不存在、已经消费或与当前请求不匹配。
	errConfirmationTokenInvalid = errors.New("mcp confirmation token is invalid")
	// errConfirmationTokenExpired 表示确认令牌已经超过服务端许可的确认窗口。
	errConfirmationTokenExpired = errors.New("mcp confirmation token is expired")
)

type confirmationRecord struct {
	tokenID            uint64
	action             string
	requestFingerprint string
	expiresAt          time.Time
}

// ConfirmationTokens 在单个 runtime 内保存短生命周期、一次性的二阶段确认令牌。
//
// 令牌不持久化也不跨节点复制；多节点部署在没有共享消费存储前会对跨节点消费
// fail closed，避免把客户端确认语义误当成服务端授权事实。
type ConfirmationTokens struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	records map[string]confirmationRecord
}

func newConfirmationTokens(ttl time.Duration) (*ConfirmationTokens, error) {
	if ttl <= 0 {
		return nil, errors.New("mcp confirmation token ttl must be greater than zero")
	}
	return &ConfirmationTokens{
		ttl:     ttl,
		now:     time.Now,
		records: make(map[string]confirmationRecord),
	}, nil
}

// Issue 为已认证调用者签发绑定 action 与请求摘要的一次性确认令牌。
func (s *ConfirmationTokens) Issue(ctx context.Context, action string, requestFingerprint string) (string, error) {
	caller, ok := callerFromContext(ctx)
	if !ok {
		return "", moduleapi.ErrUnauthenticated
	}
	action = strings.TrimSpace(action)
	requestFingerprint = strings.TrimSpace(requestFingerprint)
	if action == "" || requestFingerprint == "" {
		return "", errors.New("mcp confirmation action and request fingerprint are required")
	}

	randomBytes := make([]byte, confirmationTokenBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("read mcp confirmation token entropy: %w", err)
	}
	token := confirmationTokenPrefix + base64.RawURLEncoding.EncodeToString(randomBytes)
	now := s.currentTime()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeExpiredLocked(now)
	s.records[token] = confirmationRecord{
		tokenID:            caller.tokenID,
		action:             action,
		requestFingerprint: requestFingerprint,
		expiresAt:          now.Add(s.ttl),
	}
	return token, nil
}

// Consume 消费一次性确认令牌，并验证调用者、action、请求摘要和有效期。
func (s *ConfirmationTokens) Consume(ctx context.Context, token string, action string, requestFingerprint string) error {
	caller, ok := callerFromContext(ctx)
	if !ok {
		return moduleapi.ErrUnauthenticated
	}
	token = strings.TrimSpace(token)
	action = strings.TrimSpace(action)
	requestFingerprint = strings.TrimSpace(requestFingerprint)
	if token == "" || action == "" || requestFingerprint == "" {
		return errConfirmationTokenInvalid
	}
	now := s.currentTime()

	s.mu.Lock()
	defer s.mu.Unlock()
	record, exists := s.records[token]
	if !exists {
		return errConfirmationTokenInvalid
	}
	// 任何已命中的令牌都会在首次消费尝试后失效，阻断窃取令牌后的重放。
	delete(s.records, token)
	if !record.expiresAt.After(now) {
		return errConfirmationTokenExpired
	}
	if record.tokenID != caller.tokenID || record.action != action || record.requestFingerprint != requestFingerprint {
		return errConfirmationTokenInvalid
	}
	return nil
}

func (s *ConfirmationTokens) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func (s *ConfirmationTokens) removeExpiredLocked(now time.Time) {
	for token, record := range s.records {
		if !record.expiresAt.After(now) {
			delete(s.records, token)
		}
	}
}
