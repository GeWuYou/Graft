package user

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"graft/server/internal/moduleapi"
)

// deferredAuthCapabilities lets user declare guarded routes before auth has
// registered its capability. Boot binds the auth-owned implementations once.
type deferredAuthCapabilities struct {
	mu       sync.RWMutex
	auth     moduleapi.AuthService
	sessions moduleapi.AuthSessionService
	flow     moduleapi.AuthFlowService
}

func newDeferredAuthCapabilities() *deferredAuthCapabilities { return &deferredAuthCapabilities{} }

func (d *deferredAuthCapabilities) SetTargets(auth moduleapi.AuthService, sessions moduleapi.AuthSessionService, flow moduleapi.AuthFlowService) error {
	if auth == nil || sessions == nil || flow == nil {
		return errors.New("auth capability targets are required")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.auth, d.sessions, d.flow = auth, sessions, flow
	return nil
}

type authCapabilityTargets struct {
	auth     moduleapi.AuthService
	sessions moduleapi.AuthSessionService
	flow     moduleapi.AuthFlowService
}

func (d *deferredAuthCapabilities) targets() (authCapabilityTargets, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.auth == nil || d.sessions == nil || d.flow == nil {
		return authCapabilityTargets{}, errors.New("auth capabilities are unavailable")
	}
	return authCapabilityTargets{auth: d.auth, sessions: d.sessions, flow: d.flow}, nil
}

func (d *deferredAuthCapabilities) CurrentUser(ctx context.Context) (*moduleapi.CurrentUser, error) {
	targets, err := d.targets()
	if err != nil {
		return nil, err
	}
	return targets.auth.CurrentUser(ctx)
}
func (d *deferredAuthCapabilities) ParseAccessToken(ctx context.Context, token string) (*moduleapi.AccessTokenClaims, error) {
	targets, err := d.targets()
	if err != nil {
		return nil, err
	}
	return targets.auth.ParseAccessToken(ctx, token)
}
func (d *deferredAuthCapabilities) ListSessionsByUserID(ctx context.Context, userID uint64) ([]moduleapi.AuthSessionSummary, error) {
	targets, err := d.targets()
	if err != nil {
		return nil, err
	}
	return targets.sessions.ListSessionsByUserID(ctx, userID)
}
func (d *deferredAuthCapabilities) RevokeSessionByUserID(ctx context.Context, userID uint64, sessionID string) (moduleapi.AuthSessionRevokeResult, error) {
	targets, err := d.targets()
	if err != nil {
		return moduleapi.AuthSessionRevokeResult{}, err
	}
	return targets.sessions.RevokeSessionByUserID(ctx, userID, sessionID)
}
func (d *deferredAuthCapabilities) RevokeSessionsByUserID(ctx context.Context, userID uint64) (moduleapi.AuthSessionRevokeResult, error) {
	targets, err := d.targets()
	if err != nil {
		return moduleapi.AuthSessionRevokeResult{}, err
	}
	return targets.sessions.RevokeSessionsByUserID(ctx, userID)
}
func (d *deferredAuthCapabilities) RevokeOtherSessionsByUserID(ctx context.Context, userID uint64, sessionID string) (moduleapi.AuthSessionRevokeResult, error) {
	targets, err := d.targets()
	if err != nil {
		return moduleapi.AuthSessionRevokeResult{}, err
	}
	return targets.sessions.RevokeOtherSessionsByUserID(ctx, userID, sessionID)
}
func (d *deferredAuthCapabilities) StartLogin(ctx context.Context, username, password string) (moduleapi.AuthRefreshResult, error) {
	targets, err := d.targets()
	if err != nil {
		return moduleapi.AuthRefreshResult{}, err
	}
	return targets.flow.StartLogin(ctx, username, password)
}
func (d *deferredAuthCapabilities) RefreshSession(ctx context.Context, token string) (moduleapi.AuthRefreshResult, error) {
	targets, err := d.targets()
	if err != nil {
		return moduleapi.AuthRefreshResult{}, err
	}
	return targets.flow.RefreshSession(ctx, token)
}
func (d *deferredAuthCapabilities) LogoutCurrentSession(ctx context.Context, token string) error {
	targets, err := d.targets()
	if err != nil {
		return err
	}
	return targets.flow.LogoutCurrentSession(ctx, token)
}
func (d *deferredAuthCapabilities) RevokeAllCurrentUserSessions(ctx context.Context) error {
	targets, err := d.targets()
	if err != nil {
		return err
	}
	return targets.flow.RevokeAllCurrentUserSessions(ctx)
}
func (d *deferredAuthCapabilities) RevokeOtherCurrentUserSessions(ctx context.Context) error {
	targets, err := d.targets()
	if err != nil {
		return err
	}
	return targets.flow.RevokeOtherCurrentUserSessions(ctx)
}
func (d *deferredAuthCapabilities) ListCurrentUserSessions(ctx context.Context, limit int) ([]moduleapi.AuthSessionSummary, error) {
	targets, err := d.targets()
	if err != nil {
		return nil, err
	}
	return targets.flow.ListCurrentUserSessions(ctx, limit)
}
func (d *deferredAuthCapabilities) RevokeCurrentUserSession(ctx context.Context, sessionID string) error {
	targets, err := d.targets()
	if err != nil {
		return err
	}
	return targets.flow.RevokeCurrentUserSession(ctx, sessionID)
}
func (d *deferredAuthCapabilities) ReadBootstrapPayload(ctx context.Context, request *http.Request) (moduleapi.AuthBootstrapPayload, error) {
	targets, err := d.targets()
	if err != nil {
		return moduleapi.AuthBootstrapPayload{}, err
	}
	return targets.flow.ReadBootstrapPayload(ctx, request)
}
func (d *deferredAuthCapabilities) ChangeCurrentUserPassword(ctx context.Context, current, next string) error {
	targets, err := d.targets()
	if err != nil {
		return err
	}
	return targets.flow.ChangeCurrentUserPassword(ctx, current, next)
}
func (d *deferredAuthCapabilities) CompleteRequiredPasswordChange(ctx context.Context, next string) error {
	targets, err := d.targets()
	if err != nil {
		return err
	}
	return targets.flow.CompleteRequiredPasswordChange(ctx, next)
}
func (d *deferredAuthCapabilities) IsRestrictedPasswordChangeSession(ctx context.Context) (bool, error) {
	targets, err := d.targets()
	if err != nil {
		return false, err
	}
	return targets.flow.IsRestrictedPasswordChangeSession(ctx)
}
func (d *deferredAuthCapabilities) RouteError(err error) moduleapi.AuthRouteError {
	targets, targetErr := d.targets()
	if targetErr != nil {
		return moduleapi.AuthRouteError{Status: http.StatusInternalServerError}
	}
	return targets.flow.RouteError(err)
}

var _ moduleapi.AuthService = (*deferredAuthCapabilities)(nil)
var _ moduleapi.AuthSessionService = (*deferredAuthCapabilities)(nil)
var _ moduleapi.AuthFlowService = (*deferredAuthCapabilities)(nil)
