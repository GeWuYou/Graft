package storeent

import (
	"context"
	"errors"
	"fmt"
	"time"

	ent "graft/server/modules/user/ent"
	refreshsessionent "graft/server/modules/user/ent/refreshsession"
	userent "graft/server/modules/user/ent/user"
	userstore "graft/server/modules/user/store"
)

// userAuthUserID 将外部用户 ID 转换为内部 Ent 用户 ID。
//
// @returns 转换后的用户 ID；当输入 ID 无效时返回 userstore.ErrUserNotFound，其他转换错误原样返回。
func userAuthUserID(inputUserID uint64) (int, error) {
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

func (r *authRepository) updatePasswordHash(ctx context.Context, userID int, input userstore.SetPasswordHashInput) error {
	updater := r.client.User.UpdateOneID(userID).
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

// beginUserAuthTx 启动用户认证事务并返回回滚清理函数。
// 清理函数会尝试回滚事务，用于在后续步骤失败时释放资源。
func beginUserAuthTx(ctx context.Context, client *ent.Client, action string) (*ent.Tx, func(), error) {
	tx, err := client.Tx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin %s transaction: %w", action, err)
	}

	cleanup := func() {
		_ = tx.Rollback()
	}
	return tx, cleanup, nil
}

// commitUserAuthTx 提交用户认证事务。
// 提交失败时，`context.Canceled` 和 `context.DeadlineExceeded` 会直接返回，其余错误会附带操作名包装。
func commitUserAuthTx(tx *ent.Tx, action string) error {
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
	requireActiveUser  bool
	contextMessage     string
}

// setUserPasswordInTx 在事务中更新用户密码信息，必要时仅作用于未删除用户。
// 找不到用户时返回 userstore.ErrUserNotFound，其余错误按 input.contextMessage 包装后返回。
func setUserPasswordInTx(
	ctx context.Context,
	tx *ent.Tx,
	input passwordUpdateTxInput,
) error {
	updater := tx.User.UpdateOneID(input.userID)
	if input.requireActiveUser {
		updater = updater.Where(userent.DeletedAtEQ(0))
	}
	updater = updater.
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

// revokeRefreshSessionsInTx 撤销指定用户的未撤销刷新会话。
// 当 currentTokenID 不为 nil 时，会保留该 token 对应的会话不作撤销。
// 返回保存更新时的错误。
func revokeRefreshSessionsInTx(
	ctx context.Context,
	tx *ent.Tx,
	userID int,
	revokedAt time.Time,
	currentTokenID *string,
	contextMessage string,
) error {
	updater := tx.RefreshSession.Update().
		Where(
			refreshsessionent.UserIDEQ(userID),
			refreshsessionent.RevokedAtIsNil(),
		)
	if currentTokenID != nil {
		updater = updater.Where(refreshsessionent.TokenIDNEQ(*currentTokenID))
	}

	if _, err := updater.SetRevokedAt(revokedAt).Save(ctx); err != nil {
		return fmt.Errorf("%s: %w", contextMessage, err)
	}

	return nil
}

// loadActiveRefreshSessionForRotation 加载可用于刷新令牌轮换的当前刷新会话，要求会话未撤销且未过期。
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
			return nil, userstore.ErrRefreshSessionNotFound
		}
		return nil, fmt.Errorf("query current refresh session for rotation: %w", err)
	}
	if current.RevokedAt != nil || !current.ExpiresAt.After(now) {
		return nil, userstore.ErrRefreshSessionNotFound
	}

	return current, nil
}

// input 提供撤销时间、替换 token ID 和判定是否仍有效的当前时间。
func revokeRefreshSessionForRotation(
	ctx context.Context,
	tx *ent.Tx,
	sessionID int,
	input userstore.RotateRefreshSessionInput,
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
		return userstore.ErrRefreshSessionNotFound
	}

	return nil
}

// createRotatedRefreshSession 在事务中创建轮换后的刷新会话。
// 它会设置用户 ID、新的令牌 ID 和过期时间；创建失败时返回包装后的错误。
func createRotatedRefreshSession(
	ctx context.Context,
	tx *ent.Tx,
	userID int,
	input userstore.RotateRefreshSessionInput,
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

// commitRefreshRotation 提交刷新会话轮换事务。
//
// @returns 提交成功时为 nil；提交失败时返回错误。
func commitRefreshRotation(tx *ent.Tx) error {
	return commitUserAuthTx(tx, "refresh session rotation")
}

// commitPasswordChange 提交密码修改事务。
func commitPasswordChange(tx *ent.Tx) error {
	return commitUserAuthTx(tx, "password change")
}

// commitResetPassword 提交重置密码事务，并返回提交结果。
func commitResetPassword(tx *ent.Tx) error {
	return commitUserAuthTx(tx, "reset password")
}
