// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package user

// CreateUserCommand is the business-level input for creating a managed user.
type CreateUserCommand struct {
	Username string
	Display  string
	Password string
	ActorID  uint64
}

// UpdateUserCommand is the business-level input for updating a managed user profile.
type UpdateUserCommand struct {
	ID       uint64
	Username string
	Display  string
	ActorID  uint64
}

// UpdateUserStatusCommand is the business-level input for updating a managed user status.
type UpdateUserStatusCommand struct {
	ID      uint64
	Status  string
	ActorID uint64
}
