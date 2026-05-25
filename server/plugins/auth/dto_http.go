package auth

import "time"

type loginUserResponse struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type loginResponse struct {
	AccessToken        string            `json:"access_token"`
	ExpiresAt          time.Time         `json:"expires_at"`
	MustChangePassword bool              `json:"must_change_password"`
	User               loginUserResponse `json:"user"`
}

type bootstrapResponse struct {
	User               loginUserResponse       `json:"user"`
	MustChangePassword bool                    `json:"must_change_password"`
	Roles              []string                `json:"roles"`
	Permissions        []string                `json:"permissions"`
	Menus              []bootstrapMenuResponse `json:"menus"`
	Locale             bootstrapLocaleSnapshot `json:"locale"`
}

type bootstrapMenuResponse struct {
	Code       string `json:"code"`
	Title      string `json:"title"`
	TitleKey   string `json:"title_key,omitempty"`
	Path       string `json:"path"`
	Icon       string `json:"icon"`
	Permission string `json:"permission"`
}

type bootstrapLocaleSnapshot struct {
	CurrentLocale    string   `json:"current_locale"`
	DefaultLocale    string   `json:"default_locale"`
	FallbackLocale   string   `json:"fallback_locale"`
	SupportedLocales []string `json:"supported_locales"`
}

type sessionSummary struct {
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Current   bool      `json:"current"`
}
