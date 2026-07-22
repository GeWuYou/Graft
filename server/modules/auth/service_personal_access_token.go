package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"graft/server/internal/moduleapi"
	"graft/server/modules/auth/store"
)

const (
	personalAccessTokenSecretBytes  = 32
	personalAccessTokenPrefix       = "gpat_"
	personalAccessTokenDisplaySize  = 16
	defaultPersonalAccessTokenLimit = 50
	maxPersonalAccessTokenLimit     = 100
	maxPersonalAccessTokenNameBytes = 128
	maxPersonalAccessTokenScopes    = 64
	maxPersonalAccessTokenScopeSize = 128
)

var (
	errPersonalAccessTokenStoreUnavailable = errors.New("personal access token store is unavailable")
	errInvalidPersonalAccessTokenInput     = errors.New("personal access token input is invalid")
)

// CreateCurrentUserPersonalAccessToken 为当前登录用户签发一个只显示一次明文的个人 API Token。
// Token scope 仅作为后续 MCP 调用的收窄条件，实际业务权限仍由 RBAC 在执行点重新判断。
func (s authService) CreateCurrentUserPersonalAccessToken(
	ctx context.Context,
	input moduleapi.PersonalAccessTokenCreateInput,
) (moduleapi.PersonalAccessTokenIssued, error) {
	if s.personalTokens == nil {
		return moduleapi.PersonalAccessTokenIssued{}, errPersonalAccessTokenStoreUnavailable
	}
	user, err := currentPersonalAccessTokenUser(ctx)
	if err != nil {
		return moduleapi.PersonalAccessTokenIssued{}, err
	}
	now := s.currentTime()
	name, scopes, err := normalizePersonalAccessTokenInput(input, now)
	if err != nil {
		return moduleapi.PersonalAccessTokenIssued{}, err
	}

	token, err := newPersonalAccessTokenSecret()
	if err != nil {
		return moduleapi.PersonalAccessTokenIssued{}, err
	}
	record, err := s.personalTokens.CreatePersonalAccessToken(ctx, store.CreatePersonalAccessTokenInput{
		UserID:      user.ID,
		Name:        name,
		TokenPrefix: personalAccessTokenDisplayPrefix(token),
		SecretHash:  hashPersonalAccessToken(token),
		Scopes:      scopes,
		ExpiresAt:   input.ExpiresAt.UTC(),
	})
	if err != nil {
		return moduleapi.PersonalAccessTokenIssued{}, err
	}
	return moduleapi.PersonalAccessTokenIssued{Summary: toPersonalAccessTokenSummary(record), Token: token}, nil
}

// ListCurrentUserPersonalAccessTokens 返回当前用户可管理的个人 API Token 生命周期摘要。
func (s authService) ListCurrentUserPersonalAccessTokens(ctx context.Context, limit int) ([]moduleapi.PersonalAccessTokenSummary, error) {
	if s.personalTokens == nil {
		return nil, errPersonalAccessTokenStoreUnavailable
	}
	user, err := currentPersonalAccessTokenUser(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultPersonalAccessTokenLimit
	}
	if limit > maxPersonalAccessTokenLimit {
		return nil, fmt.Errorf("personal access token limit must not exceed %d", maxPersonalAccessTokenLimit)
	}
	records, err := s.personalTokens.ListPersonalAccessTokensByUserID(ctx, store.ListPersonalAccessTokensByUserIDInput{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}

	items := make([]moduleapi.PersonalAccessTokenSummary, 0, len(records))
	for _, record := range records {
		items = append(items, toPersonalAccessTokenSummary(record))
	}
	return items, nil
}

// RevokeCurrentUserPersonalAccessToken 撤销当前用户拥有的一个个人 API Token。
func (s authService) RevokeCurrentUserPersonalAccessToken(ctx context.Context, tokenID uint64) error {
	if s.personalTokens == nil {
		return errPersonalAccessTokenStoreUnavailable
	}
	if tokenID == 0 {
		return errors.New("personal access token id is required")
	}
	user, err := currentPersonalAccessTokenUser(ctx)
	if err != nil {
		return err
	}
	return s.personalTokens.RevokePersonalAccessTokenByUserID(ctx, store.RevokePersonalAccessTokenByUserIDInput{
		UserID:    user.ID,
		TokenID:   tokenID,
		RevokedAt: s.currentTime(),
	})
}

// AuthenticatePersonalAccessToken 验证个人 API Token 并建立 MCP 所需的最小身份与 scope 事实。
// 它不会读取或推导任何 RBAC 权限；MCP scope gate 必须在每个 operation 上继续调用 Authorizer。
func (s authService) AuthenticatePersonalAccessToken(ctx context.Context, token string) (moduleapi.PersonalAccessTokenCaller, error) {
	if s.personalTokens == nil {
		return moduleapi.PersonalAccessTokenCaller{}, errPersonalAccessTokenStoreUnavailable
	}
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, personalAccessTokenPrefix) {
		return moduleapi.PersonalAccessTokenCaller{}, moduleapi.ErrInvalidPersonalAccessToken
	}
	record, err := s.personalTokens.GetPersonalAccessTokenBySecretHash(ctx, hashPersonalAccessToken(token))
	if err != nil {
		if errors.Is(err, store.ErrPersonalAccessTokenNotFound) {
			return moduleapi.PersonalAccessTokenCaller{}, moduleapi.ErrInvalidPersonalAccessToken
		}
		return moduleapi.PersonalAccessTokenCaller{}, err
	}
	now := s.currentTime()
	if record.RevokedAt != nil {
		return moduleapi.PersonalAccessTokenCaller{}, moduleapi.ErrInvalidPersonalAccessToken
	}
	if !record.ExpiresAt.After(now) {
		return moduleapi.PersonalAccessTokenCaller{}, moduleapi.ErrExpiredPersonalAccessToken
	}
	user, err := s.identity.GetCurrentUserByID(ctx, record.UserID)
	if err != nil {
		return moduleapi.PersonalAccessTokenCaller{}, moduleapi.ErrInvalidPersonalAccessToken
	}
	if err := s.personalTokens.MarkPersonalAccessTokenUsed(ctx, record.ID, now); err != nil {
		if errors.Is(err, store.ErrPersonalAccessTokenNotFound) {
			return moduleapi.PersonalAccessTokenCaller{}, moduleapi.ErrInvalidPersonalAccessToken
		}
		return moduleapi.PersonalAccessTokenCaller{}, err
	}
	return moduleapi.PersonalAccessTokenCaller{
		TokenID:   record.ID,
		User:      user,
		Scopes:    append([]string(nil), record.Scopes...),
		ExpiresAt: record.ExpiresAt,
	}, nil
}

func (s authService) currentTime() time.Time {
	if s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func currentPersonalAccessTokenUser(ctx context.Context) (moduleapi.CurrentUser, error) {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil || requestAuth.User.ID == 0 {
		return moduleapi.CurrentUser{}, moduleapi.ErrUnauthenticated
	}
	return *requestAuth.User, nil
}

func normalizePersonalAccessTokenInput(input moduleapi.PersonalAccessTokenCreateInput, now time.Time) (string, []string, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > maxPersonalAccessTokenNameBytes {
		return "", nil, fmt.Errorf("%w: name must contain at most %d bytes", errInvalidPersonalAccessTokenInput, maxPersonalAccessTokenNameBytes)
	}
	if !input.ExpiresAt.After(now) {
		return "", nil, fmt.Errorf("%w: expiry must be in the future", errInvalidPersonalAccessTokenInput)
	}
	if len(input.Scopes) == 0 || len(input.Scopes) > maxPersonalAccessTokenScopes {
		return "", nil, fmt.Errorf("%w: scopes must contain between 1 and %d items", errInvalidPersonalAccessTokenInput, maxPersonalAccessTokenScopes)
	}

	scopes := make([]string, 0, len(input.Scopes))
	seen := make(map[string]struct{}, len(input.Scopes))
	for _, rawScope := range input.Scopes {
		scope := strings.TrimSpace(rawScope)
		if !validPersonalAccessTokenScope(scope) {
			return "", nil, fmt.Errorf("%w: scope %q is invalid", errInvalidPersonalAccessTokenInput, rawScope)
		}
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}
	if len(scopes) == 0 {
		return "", nil, fmt.Errorf("%w: scopes are required", errInvalidPersonalAccessTokenInput)
	}
	return name, scopes, nil
}

func validPersonalAccessTokenScope(scope string) bool {
	if scope == "" || scope == "*" || len(scope) > maxPersonalAccessTokenScopeSize {
		return false
	}
	for _, character := range scope {
		if !validPersonalAccessTokenScopeCharacter(character) {
			return false
		}
	}
	return true
}

func validPersonalAccessTokenScopeCharacter(character rune) bool {
	return unicode.IsLower(character) || unicode.IsDigit(character) || strings.ContainsRune(".:-_", character)
}

func newPersonalAccessTokenSecret() (string, error) {
	secret := make([]byte, personalAccessTokenSecretBytes)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("read personal access token secret: %w", err)
	}
	return personalAccessTokenPrefix + base64.RawURLEncoding.EncodeToString(secret), nil
}

func hashPersonalAccessToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func personalAccessTokenDisplayPrefix(token string) string {
	if len(token) <= personalAccessTokenDisplaySize {
		return token
	}
	return token[:personalAccessTokenDisplaySize]
}

func toPersonalAccessTokenSummary(record store.PersonalAccessToken) moduleapi.PersonalAccessTokenSummary {
	return moduleapi.PersonalAccessTokenSummary{
		ID:          record.ID,
		Name:        record.Name,
		TokenPrefix: record.TokenPrefix,
		Scopes:      append([]string(nil), record.Scopes...),
		ExpiresAt:   record.ExpiresAt,
		RevokedAt:   record.RevokedAt,
		LastUsedAt:  record.LastUsedAt,
		CreatedAt:   record.CreatedAt,
	}
}
