package store

import authstore "graft/server/modules/auth/store"

var (
	// ErrRefreshSessionNotFound is kept as a temporary compatibility alias during auth extraction.
	ErrRefreshSessionNotFound = authstore.ErrRefreshSessionNotFound
)

// UserCredential keeps the temporary compatibility alias to the auth-owned credential DTO.
type UserCredential = authstore.UserCredential

// SetPasswordHashInput keeps the temporary compatibility alias to the auth-owned password update input.
type SetPasswordHashInput = authstore.SetPasswordHashInput

// ResetPasswordAndRevokeSessionsInput keeps the temporary compatibility alias to the auth-owned reset input.
type ResetPasswordAndRevokeSessionsInput = authstore.ResetPasswordAndRevokeSessionsInput

// ChangePasswordAndRevokeOtherRefreshSessionsInput keeps the temporary compatibility alias to the auth-owned self-service password-change input.
type ChangePasswordAndRevokeOtherRefreshSessionsInput = authstore.ChangePasswordAndRevokeOtherRefreshSessionsInput

// EnsureUserCredentialInput keeps the temporary compatibility alias to the auth-owned ensured-credential input.
type EnsureUserCredentialInput = authstore.EnsureUserCredentialInput

// RevokeOtherRefreshSessionsInput keeps the temporary compatibility alias to the auth-owned revoke-others input.
type RevokeOtherRefreshSessionsInput = authstore.RevokeOtherRefreshSessionsInput

// RefreshSession keeps the temporary compatibility alias to the auth-owned refresh-session DTO.
type RefreshSession = authstore.RefreshSession

// ListActiveRefreshSessionsByUserIDInput keeps the temporary compatibility alias to the auth-owned active-session query input.
type ListActiveRefreshSessionsByUserIDInput = authstore.ListActiveRefreshSessionsByUserIDInput

// CreateRefreshSessionInput keeps the temporary compatibility alias to the auth-owned session-creation input.
type CreateRefreshSessionInput = authstore.CreateRefreshSessionInput

// RevokeRefreshSessionInput keeps the temporary compatibility alias to the auth-owned single-session revoke input.
type RevokeRefreshSessionInput = authstore.RevokeRefreshSessionInput

// RevokeRefreshSessionsByUserIDInput keeps the temporary compatibility alias to the auth-owned bulk revoke input.
type RevokeRefreshSessionsByUserIDInput = authstore.RevokeRefreshSessionsByUserIDInput

// RevokeRefreshSessionByUserIDInput keeps the temporary compatibility alias to the auth-owned targeted revoke input.
type RevokeRefreshSessionByUserIDInput = authstore.RevokeRefreshSessionByUserIDInput

// RotateRefreshSessionInput keeps the temporary compatibility alias to the auth-owned rotation input.
type RotateRefreshSessionInput = authstore.RotateRefreshSessionInput

// PasswordChangeRepository keeps the temporary compatibility alias to the auth-owned atomic password-change contract.
type PasswordChangeRepository = authstore.PasswordChangeRepository

// AuthRepository keeps the temporary compatibility alias to the auth-owned repository contract.
type AuthRepository = authstore.AuthRepository
