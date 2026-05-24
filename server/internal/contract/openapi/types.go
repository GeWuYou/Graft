package openapi

import "graft/server/internal/contract/openapi/generated"

// APIEnvelope aliases the generated top-level HTTP success or error envelope shape.
type APIEnvelope = generated.ApiEnvelope

// PostUsersJSONRequestBody aliases the generated JSON request body for POST /api/users.
type PostUsersJSONRequestBody = generated.PostUsersJSONRequestBody

// PostUserUpdateJSONRequestBody aliases the generated JSON request body for POST /api/users/{id}/update.
type PostUserUpdateJSONRequestBody = generated.PostUserUpdateJSONRequestBody

// PostUserStatusJSONRequestBody aliases the generated JSON request body for POST /api/users/{id}/status.
type PostUserStatusJSONRequestBody = generated.PostUserStatusJSONRequestBody

// PostUserStatusJSONBodyStatus aliases the generated route-local status enum for POST /api/users/{id}/status.
type PostUserStatusJSONBodyStatus = generated.PostUserStatusJSONBodyStatus

const (
	// PostUserStatusJSONBodyStatusEnabled is the generated status enum member for an enabled managed user.
	PostUserStatusJSONBodyStatusEnabled = generated.PostUserStatusJSONBodyStatusEnabled
	// PostUserStatusJSONBodyStatusDisabled is the generated status enum member for a disabled managed user.
	PostUserStatusJSONBodyStatusDisabled = generated.PostUserStatusJSONBodyStatusDisabled
)
