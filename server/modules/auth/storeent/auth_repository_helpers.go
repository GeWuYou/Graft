package storeent

import (
	"context"
	"errors"
	"fmt"
	"time"

	authstore "graft/server/modules/auth/store"
	ent "graft/server/modules/user/ent"
	refreshsessionent "graft/server/modules/user/ent/refreshsession"
	userent "graft/server/modules/user/ent/user"
	userstore "graft/server/modules/user/store"
)

// authUserID 将输入的用户 ID 转换为 ent 用户 ID。
// 当输入 ID 无效时，返回 userstore.ErrUserNotFound。
//
// @returns 转换后的用户 ID；如果输入 ID 无效则返回 userstore.ErrUserNotFound，否则返回转换错误。
func authUserID(inputUserID uint64) (int, error) {
	userID, err := toEntID(inputUserID)
	if err != nil {
		if errors.Is(err, userstore.ErrInvalidID) {
			return 0, userstore.ErrUserNotFound
		}
		return 0, err
	}
	return userID, nil
}

func (r *authRepository) queryUserCredentialByUsername(ctx context.Context, username string) (*ent.User, error) {
	record, err := r.client.User.Query().
		Where(
			userent.UsernameEQ(username),
			userent.DeletedAtEQ(0),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, userstore.ErrUserNotFound
		}
		return nil, fmt.Errorf("query user credential by username: %w", err)
	}
	return record, nil
}

func (r *authRepository) updatePasswordHash(ctx context.Context, userID int, input authstore.SetPasswordHashInput) error {
	updater := r.client.User.UpdateOneID(userID).
		Where(userent.DeletedAtEQ(0)).
		SetPasswordHash(input.PasswordHash).
		SetMustChangePassword(input.MustChangePassword)
	if input.ChangedAt != nil {
		updater = updater.SetPasswordChangedAt(*input.ChangedAt)
	}

	if err := updater.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return userstore.ErrUserNotFound
		}
		return fmt.Errorf("set user password hash: %w", err)
	}

	return nil
}

// beginAuthTx 开启一个鉴权事务，并返回回滚清理函数。
// 成功时返回事务、用于回滚的清理函数，以及空错误；失败时返回带有操作名称的错误。
func beginAuthTx(ctx context.Context, client *ent.Client, action string) (*ent.Tx, func(), error) {
	tx, err := client.Tx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin %s transaction: %w", action, err)
	}

	cleanup := func() {
		_ = tx.Rollback()
	}
	return tx, cleanup, nil
}

// 若提交失败且错误为 `context.Canceled` 或 `context.DeadlineExceeded`，则原样返回该错误；其他错误会包装为带操作名的错误。
func commitAuthTx(tx *ent.Tx, action string) error {
	if commitErr := tx.Commit(); commitErr != nil {
		if errors.Is(commitErr, context.Canceled) || errors.Is(commitErr, context.DeadlineExceeded) {
			return commitErr
		}
		return fmt.Errorf("commit %s transaction: %w", action, commitErr)
	}

	return nil
}

type passwordUpdateTxInput struct {
	userID             int
	passwordHash       string
	mustChangePassword bool
	changedAt          time.Time
	contextMessage     string
}

// setUserPasswordInTx 在事务中更新用户的密码哈希、强制修改密码标记和密码更新时间。
// 如果指定用户不存在，返回 userstore.ErrUserNotFound；其他错误会带上上下文信息返回。
func setUserPasswordInTx(
	ctx context.Context,
	tx *ent.Tx,
	input passwordUpdateTxInput,
) error {
	updater := tx.User.UpdateOneID(input.userID).
		Where(userent.DeletedAtEQ(0)).
		SetPasswordHash(input.passwordHash).
		SetMustChangePassword(input.mustChangePassword).
		SetPasswordChangedAt(input.changedAt)

	if err := updater.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return userstore.ErrUserNotFound
		}
		return fmt.Errorf("%s: %w", input.contextMessage, err)
	}

	return nil
}

// 失败时返回带有上下文信息的错误。
func revokeOtherRefreshSessionsInTx(
	ctx context.Context,
	tx *ent.Tx,
	userID int,
	currentTokenID string,
	revokedAt time.Time,
	contextMessage string,
) error {
	updater := tx.RefreshSession.Update().
		Where(
			refreshsessionent.UserIDEQ(userID),
			refreshsessionent.RevokedAtIsNil(),
		)
	if currentTokenID != "" {
		updater = updater.Where(refreshsessionent.TokenIDNEQ(currentTokenID))
	}

	if _, err := updater.SetRevokedAt(revokedAt).Save(ctx); err != nil {
		return fmt.Errorf("%s: %w", contextMessage, err)
	}

	return nil
}

// loadActiveRefreshSessionForRotation 加载可用于刷新令牌轮换的当前刷新会话。
// 会话必须与给定令牌 ID 匹配、未撤销且尚未过期。
//
// @returns 找到且处于有效状态的刷新会话；如果不存在或已失效则返回 `authstore.ErrRefreshSessionNotFound`；查询失败时返回对应错误。
func loadActiveRefreshSessionForRotation(
	ctx context.Context,
	tx *ent.Tx,
	currentTokenID string,
	now time.Time,
) (*ent.RefreshSession, error) {
	current, err := tx.RefreshSession.Query().
		Where(refreshsessionent.TokenIDEQ(currentTokenID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, authstore.ErrRefreshSessionNotFound
		}
		return nil, fmt.Errorf("query current refresh session for rotation: %w", err)
	}
	if current.RevokedAt != nil || !current.ExpiresAt.After(now) {
		return nil, authstore.ErrRefreshSessionNotFound
	}

	return current, nil
}

// revokeRefreshSessionForRotation 撤销指定的刷新会话并记录其替换令牌。
//
// @param sessionID 要撤销的刷新会话 ID。
// @param input 包含撤销时间、新令牌 ID 和当前时间的旋转输入。
// @returns 更新成功时返回 nil；如果会话不存在、已撤销或已过期，则返回 authstore.ErrRefreshSessionNotFound。
func revokeRefreshSessionForRotation(
	ctx context.Context,
	tx *ent.Tx,
	sessionID int,
	input authstore.RotateRefreshSessionInput,
) error {
	affected, err := tx.RefreshSession.Update().
		Where(
			refreshsessionent.IDEQ(sessionID),
			refreshsessionent.RevokedAtIsNil(),
			refreshsessionent.ExpiresAtGT(input.Now),
		).
		SetRevokedAt(input.RevokedAt).
		SetReplacedByTokenID(input.NewTokenID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("revoke current refresh session during rotation: %w", err)
	}
	if affected == 0 {
		return authstore.ErrRefreshSessionNotFound
	}

	return nil
}

// createRotatedRefreshSession 在事务中创建新的刷新会话。
// 该会话使用输入中的新令牌和过期时间，并归属于指定用户。
func createRotatedRefreshSession(
	ctx context.Context,
	tx *ent.Tx,
	userID int,
	input authstore.RotateRefreshSessionInput,
) (*ent.RefreshSession, error) {
	next, err := tx.RefreshSession.Create().
		SetUserID(userID).
		SetTokenID(input.NewTokenID).
		SetExpiresAt(input.NewExpiresAt).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create rotated refresh session: %w", err)
	}

	return next, nil
}

// commitRefreshRotation 提交刷新会话旋转事务。
// 成功时返回 nil。
func commitRefreshRotation(tx *ent.Tx) error {
	return commitAuthTx(tx, "refresh session rotation")
}

// runAuthTx 按事务方式执行给定回调，并在成功后提交事务。
// 回调返回错误时直接返回该错误；提交失败时返回提交错误。
func runAuthTx(ctx context.Context, client *ent.Client, action string, fn func(tx *ent.Tx) error) error {
	tx, cleanup, err := beginAuthTx(ctx, client, action)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := fn(tx); err != nil {
		return err
	}

	return commitAuthTx(tx, action)
}
