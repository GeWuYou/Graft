package user

import (
	"context"
	"net/http"

	"graft/server/internal/moduleapi"
)

type authFlowBridge struct {
	auth      *authService
	bootstrap bootstrapReader
}

func (b authFlowBridge) StartLogin(ctx context.Context, username string, password string) (moduleapi.AuthRefreshResult, error) {
	result, err := b.auth.LoginWithRefresh(ctx, username, password)
	if err != nil {
		return moduleapi.AuthRefreshResult{}, err
	}

	return moduleapi.AuthRefreshResult{
		AccessToken:        result.AccessToken,
		AccessExpiry:       result.AccessExpiry,
		RefreshToken:       result.RefreshToken,
		RefreshExpiry:      result.RefreshExpiry,
		MustChangePassword: result.MustChangePassword,
		User: moduleapi.CurrentUser{
			ID:          result.User.ID,
			Username:    result.User.Username,
			DisplayName: result.User.DisplayName,
		},
	}, nil
}

func (b authFlowBridge) RefreshSession(ctx context.Context, refreshToken string) (moduleapi.AuthRefreshResult, error) {
	result, err := b.auth.RefreshWithRotation(ctx, refreshToken)
	if err != nil {
		return moduleapi.AuthRefreshResult{}, err
	}

	return moduleapi.AuthRefreshResult{
		AccessToken:        result.AccessToken,
		AccessExpiry:       result.AccessExpiry,
		RefreshToken:       result.RefreshToken,
		RefreshExpiry:      result.RefreshExpiry,
		MustChangePassword: result.MustChangePassword,
		User: moduleapi.CurrentUser{
			ID:          result.User.ID,
			Username:    result.User.Username,
			DisplayName: result.User.DisplayName,
		},
	}, nil
}

func (b authFlowBridge) LogoutCurrentSession(ctx context.Context, refreshToken string) error {
	return b.auth.LogoutCurrentSession(ctx, refreshToken)
}

func (b authFlowBridge) RevokeAllCurrentUserSessions(ctx context.Context) error {
	return b.auth.RevokeAllCurrentUserSessions(ctx)
}

func (b authFlowBridge) RevokeOtherCurrentUserSessions(ctx context.Context) error {
	return b.auth.RevokeOtherCurrentUserSessions(ctx)
}

func (b authFlowBridge) ListCurrentUserSessions(ctx context.Context, limit int) ([]moduleapi.AuthSessionSummary, error) {
	sessions, err := b.auth.ListCurrentUserSessions(ctx, sessionListOptions{Limit: limit})
	if err != nil {
		return nil, err
	}

	summaries := make([]moduleapi.AuthSessionSummary, 0, len(sessions))
	for _, session := range sessions {
		summaries = append(summaries, moduleapi.AuthSessionSummary{
			SessionID: session.SessionID,
			CreatedAt: session.CreatedAt,
			ExpiresAt: session.ExpiresAt,
			Current:   session.Current,
		})
	}

	return summaries, nil
}

func (b authFlowBridge) RevokeCurrentUserSession(ctx context.Context, sessionID string) error {
	return b.auth.RevokeCurrentUserSession(ctx, sessionID)
}

func (b authFlowBridge) ReadBootstrapPayload(ctx context.Context, request *http.Request) (moduleapi.AuthBootstrapPayload, error) {
	payload, err := b.bootstrap.Read(ctx, request)
	if err != nil {
		return moduleapi.AuthBootstrapPayload{}, err
	}

	menus := make([]moduleapi.AuthBootstrapMenuItem, 0, len(payload.Menus))
	for _, item := range payload.Menus {
		menus = append(menus, moduleapi.AuthBootstrapMenuItem{
			Code:       item.Code,
			Title:      item.Title,
			TitleKey:   item.TitleKey,
			Path:       item.Path,
			Icon:       item.Icon,
			Order:      item.Order,
			Permission: item.Permission,
		})
	}

	return moduleapi.AuthBootstrapPayload{
		User: moduleapi.CurrentUser{
			ID:          payload.User.ID,
			Username:    payload.User.Username,
			DisplayName: payload.User.DisplayName,
		},
		MustChangePassword: payload.MustChangePassword,
		Roles:              append([]string(nil), payload.Roles...),
		Permissions:        append([]string(nil), payload.Permissions...),
		Menus:              menus,
		Locale: moduleapi.AuthBootstrapLocaleSnapshot{
			CurrentLocale:    payload.Locale.CurrentLocale,
			DefaultLocale:    payload.Locale.DefaultLocale,
			FallbackLocale:   payload.Locale.FallbackLocale,
			SupportedLocales: append([]string(nil), payload.Locale.SupportedLocales...),
		},
	}, nil
}

func (b authFlowBridge) ChangeCurrentUserPassword(ctx context.Context, currentPassword string, newPassword string) error {
	return b.auth.ChangeCurrentUserPassword(ctx, currentPassword, newPassword)
}

func (b authFlowBridge) CompleteRequiredPasswordChange(ctx context.Context, newPassword string) error {
	return b.auth.CompleteRequiredPasswordChange(ctx, newPassword)
}

func (b authFlowBridge) IsRestrictedPasswordChangeSession(ctx context.Context) (bool, error) {
	return b.auth.isRestrictedPasswordChangeSession(ctx)
}

func (b authFlowBridge) RouteError(err error) moduleapi.AuthRouteError {
	status, key := mapAuthError(err)
	return moduleapi.AuthRouteError{
		Status:     status,
		MessageKey: key.String(),
		Data:       authErrorDetails(err),
	}
}

var _ moduleapi.AuthFlowService = authFlowBridge{}
