package openapi

import "graft/server/internal/contract/openapi/generated"

// APIEnvelope aliases the generated top-level HTTP success or error envelope shape.
type APIEnvelope = generated.ApiEnvelope

// PostUsersJSONRequestBody aliases the generated JSON request body for POST /api/users.
type PostUsersJSONRequestBody = generated.PostUsersJSONRequestBody

// PostAuthChangePasswordJSONRequestBody aliases the generated JSON request body for POST /api/auth/change-password.
type PostAuthChangePasswordJSONRequestBody = generated.PostAuthChangePasswordJSONRequestBody

// PostAuthCompleteRequiredPasswordChangeJSONRequestBody aliases the generated JSON request body for POST /api/auth/complete-required-password-change.
type PostAuthCompleteRequiredPasswordChangeJSONRequestBody = generated.PostAuthCompleteRequiredPasswordChangeJSONRequestBody

// PostUserUpdateJSONRequestBody aliases the generated JSON request body for POST /api/users/{id}/update.
type PostUserUpdateJSONRequestBody = generated.PostUserUpdateJSONRequestBody

// PostUserStatusJSONRequestBody aliases the generated JSON request body for POST /api/users/{id}/status.
type PostUserStatusJSONRequestBody = generated.PostUserStatusJSONRequestBody

// PostUserResetPasswordJSONRequestBody aliases the generated JSON request body for POST /api/users/{id}/reset-password.
type PostUserResetPasswordJSONRequestBody = generated.PostUserResetPasswordJSONRequestBody

// PostRolesJSONRequestBody aliases the generated JSON request body for POST /api/roles.
type PostRolesJSONRequestBody = generated.PostRolesJSONRequestBody

// PostRoleUpdateJSONRequestBody aliases the generated JSON request body for POST /api/roles/{id}/update.
type PostRoleUpdateJSONRequestBody = generated.PostRoleUpdateJSONRequestBody

// PostRolePermissionAssignJSONRequestBody aliases the generated JSON request body for POST /api/roles/{id}/permissions/assign.
type PostRolePermissionAssignJSONRequestBody = generated.PostRolePermissionAssignJSONRequestBody

// PostUserRolesAssignJSONRequestBody aliases the generated JSON request body for POST /api/users/{id}/roles/assign.
type PostUserRolesAssignJSONRequestBody = generated.PostUserRolesAssignJSONRequestBody

// PostUserStatusJSONBodyStatus aliases the generated route-local status enum for POST /api/users/{id}/status.
type PostUserStatusJSONBodyStatus = generated.PostUserStatusJSONBodyStatus

const (
	// PostUserStatusJSONBodyStatusEnabled is the generated status enum member for an enabled managed user.
	PostUserStatusJSONBodyStatusEnabled = generated.PostUserStatusJSONBodyStatusEnabled
	// PostUserStatusJSONBodyStatusDisabled is the generated status enum member for a disabled managed user.
	PostUserStatusJSONBodyStatusDisabled = generated.PostUserStatusJSONBodyStatusDisabled
)
