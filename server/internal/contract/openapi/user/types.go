package useropenapi

// WriteServerInterface is the minimal generated handler contract for guarded user write migration.
type WriteServerInterface interface {
	PostUsers(params PostUsersParams, body PostUsersJSONRequestBody)
	PostUserUpdate(id uint64, params PostUserUpdateParams, body PostUserUpdateJSONRequestBody)
}
