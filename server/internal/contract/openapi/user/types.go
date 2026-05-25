package useropenapi

// ReadServerInterface is the minimal generated handler contract for guarded user read migration.
type ReadServerInterface interface {
	GetUsers(params GetUsersParams)
	GetUserById(id uint64, params GetUserByIdParams)
}

// WriteServerInterface is the minimal generated handler contract for guarded user write migration.
type WriteServerInterface interface {
	PostUsers(params PostUsersParams, body PostUsersJSONRequestBody)
	PostUserUpdate(id uint64, params PostUserUpdateParams, body PostUserUpdateJSONRequestBody)
	PostUserStatus(id uint64, params PostUserStatusParams, body PostUserStatusJSONRequestBody)
	PostUserResetPassword(id uint64, params PostUserResetPasswordParams, body PostUserResetPasswordJSONRequestBody)
	PostUserDelete(id uint64, params PostUserDeleteParams)
}
