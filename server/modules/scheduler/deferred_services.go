// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"errors"
	"sync"

	"graft/server/internal/moduleapi"
)

type deferredAuthService struct {
	mu     sync.RWMutex
	target moduleapi.AuthService
}

func newDeferredAuthService() *deferredAuthService {
	return &deferredAuthService{}
}

func (s *deferredAuthService) SetTarget(target moduleapi.AuthService) error {
	if target == nil {
		return errors.New("auth service is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.target = target
	return nil
}

func (s *deferredAuthService) CurrentUser(ctx context.Context) (*moduleapi.CurrentUser, error) {
	target := s.currentTarget()
	if target == nil {
		return nil, errors.New("auth service is unavailable")
	}
	return target.CurrentUser(ctx)
}

func (s *deferredAuthService) ParseAccessToken(ctx context.Context, token string) (*moduleapi.AccessTokenClaims, error) {
	target := s.currentTarget()
	if target == nil {
		return nil, errors.New("auth service is unavailable")
	}
	return target.ParseAccessToken(ctx, token)
}

func (s *deferredAuthService) currentTarget() moduleapi.AuthService {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.target
}

var _ moduleapi.AuthService = (*deferredAuthService)(nil)

type deferredAuthorizer struct {
	mu     sync.RWMutex
	target moduleapi.Authorizer
}

func newDeferredAuthorizer() *deferredAuthorizer {
	return &deferredAuthorizer{}
}

func (a *deferredAuthorizer) SetTarget(target moduleapi.Authorizer) error {
	if target == nil {
		return errors.New("authorizer is required")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.target = target
	return nil
}

func (a *deferredAuthorizer) Authorize(
	ctx context.Context,
	request moduleapi.RequestAuthContext,
	permission string,
) error {
	target := a.currentTarget()
	if target == nil {
		return errors.New("authorizer is unavailable")
	}

	return target.Authorize(ctx, request, permission)
}

func (a *deferredAuthorizer) currentTarget() moduleapi.Authorizer {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.target
}

var _ moduleapi.Authorizer = (*deferredAuthorizer)(nil)
