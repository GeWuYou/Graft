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
