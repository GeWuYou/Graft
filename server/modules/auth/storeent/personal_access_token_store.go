package storeent

import (
	"context"
	"fmt"
	"strconv"
	"time"

	authent "graft/server/modules/auth/ent"
	authpersonalaccesstokenent "graft/server/modules/auth/ent/authpersonalaccesstoken"
	"graft/server/modules/auth/store"
)

// NewPersonalAccessTokenStore 创建 auth 所拥有的个人 API Token Ent 存储。
func NewPersonalAccessTokenStore(client *authent.Client) (store.PersonalAccessTokenStore, error) {
	if client == nil {
		return nil, fmt.Errorf("auth personal access token store requires a non-nil ent client")
	}
	return &personalAccessTokenStore{client: client}, nil
}

type personalAccessTokenStore struct {
	client *authent.Client
}

func (r *personalAccessTokenStore) CreatePersonalAccessToken(
	ctx context.Context,
	input store.CreatePersonalAccessTokenInput,
) (store.PersonalAccessToken, error) {
	record, err := r.client.AuthPersonalAccessToken.Create().
		SetUserID(input.UserID).
		SetName(input.Name).
		SetTokenPrefix(input.TokenPrefix).
		SetSecretHash(input.SecretHash).
		SetScopes(append([]string(nil), input.Scopes...)).
		SetExpiresAt(input.ExpiresAt).
		Save(ctx)
	if err != nil {
		return store.PersonalAccessToken{}, fmt.Errorf("create personal access token: %w", err)
	}
	return toStorePersonalAccessToken(record), nil
}

func (r *personalAccessTokenStore) GetPersonalAccessTokenBySecretHash(ctx context.Context, secretHash string) (store.PersonalAccessToken, error) {
	record, err := r.client.AuthPersonalAccessToken.Query().Where(
		authpersonalaccesstokenent.SecretHashEQ(secretHash),
		authpersonalaccesstokenent.DeletedAtEQ(0),
	).Only(ctx)
	if err != nil {
		if authent.IsNotFound(err) {
			return store.PersonalAccessToken{}, store.ErrPersonalAccessTokenNotFound
		}
		return store.PersonalAccessToken{}, fmt.Errorf("query personal access token by secret hash: %w", err)
	}
	return toStorePersonalAccessToken(record), nil
}

func (r *personalAccessTokenStore) ListPersonalAccessTokensByUserID(
	ctx context.Context,
	input store.ListPersonalAccessTokensByUserIDInput,
) ([]store.PersonalAccessToken, error) {
	query := r.client.AuthPersonalAccessToken.Query().Where(
		authpersonalaccesstokenent.UserIDEQ(input.UserID),
		authpersonalaccesstokenent.DeletedAtEQ(0),
	).Order(authent.Desc(authpersonalaccesstokenent.FieldCreatedAt), authent.Desc(authpersonalaccesstokenent.FieldID))
	if input.Limit > 0 {
		query = query.Limit(input.Limit)
	}
	records, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list personal access tokens by user id: %w", err)
	}

	items := make([]store.PersonalAccessToken, 0, len(records))
	for _, record := range records {
		items = append(items, toStorePersonalAccessToken(record))
	}
	return items, nil
}

func (r *personalAccessTokenStore) RevokePersonalAccessTokenByUserID(
	ctx context.Context,
	input store.RevokePersonalAccessTokenByUserIDInput,
) error {
	tokenID, err := personalAccessTokenEntID(input.TokenID)
	if err != nil {
		return err
	}
	affected, err := r.client.AuthPersonalAccessToken.Update().Where(
		authpersonalaccesstokenent.IDEQ(tokenID),
		authpersonalaccesstokenent.UserIDEQ(input.UserID),
		authpersonalaccesstokenent.DeletedAtEQ(0),
		authpersonalaccesstokenent.RevokedAtIsNil(),
	).SetRevokedAt(input.RevokedAt).Save(ctx)
	if err != nil {
		return fmt.Errorf("revoke personal access token by user id: %w", err)
	}
	// 撤销管理接口对当前用户是幂等的，避免用状态差异泄漏凭据生命周期细节。
	_ = affected
	return nil
}

func (r *personalAccessTokenStore) MarkPersonalAccessTokenUsed(ctx context.Context, tokenID uint64, usedAt time.Time) error {
	entID, err := personalAccessTokenEntID(tokenID)
	if err != nil {
		return err
	}
	affected, err := r.client.AuthPersonalAccessToken.Update().Where(
		authpersonalaccesstokenent.IDEQ(entID),
		authpersonalaccesstokenent.DeletedAtEQ(0),
		authpersonalaccesstokenent.RevokedAtIsNil(),
	).SetLastUsedAt(usedAt).Save(ctx)
	if err != nil {
		return fmt.Errorf("mark personal access token used: %w", err)
	}
	if affected == 0 {
		return store.ErrPersonalAccessTokenNotFound
	}
	return nil
}

func toStorePersonalAccessToken(record *authent.AuthPersonalAccessToken) store.PersonalAccessToken {
	return store.PersonalAccessToken{
		ID:          uint64(record.ID), // #nosec G115 -- Ent ID 来自受控 schema，且保持为正数。
		UserID:      record.UserID,
		Name:        record.Name,
		TokenPrefix: record.TokenPrefix,
		SecretHash:  record.SecretHash,
		Scopes:      append([]string(nil), record.Scopes...),
		ExpiresAt:   record.ExpiresAt,
		RevokedAt:   record.RevokedAt,
		LastUsedAt:  record.LastUsedAt,
		CreatedAt:   record.CreatedAt,
	}
}

func personalAccessTokenEntID(tokenID uint64) (int, error) {
	maxID := uint64(1<<(strconv.IntSize-1) - 1)
	if tokenID > maxID {
		return 0, fmt.Errorf("personal access token id exceeds ent integer range: %d", tokenID)
	}
	return int(tokenID), nil
}

var _ store.PersonalAccessTokenStore = (*personalAccessTokenStore)(nil)
