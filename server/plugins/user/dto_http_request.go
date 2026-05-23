package user

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type completeRequiredPasswordChangeRequest struct {
	NewPassword string `json:"new_password"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Display  string `json:"display"`
	Password string `json:"password"`
}

type updateUserRequest struct {
	Username string `json:"username"`
	Display  string `json:"display"`
}

type updateUserStatusRequest struct {
	Status string `json:"status"`
}

type resetUserPasswordRequest struct {
	NewPassword string `json:"new_password"`
}
