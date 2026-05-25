package auth

import (
	"math"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/pluginapi"
)

func toLoginResponse(result pluginapi.AuthRefreshResult) generated.LoginResponse {
	var response generated.LoginResponse
	response.AccessToken = result.AccessToken
	response.ExpiresAt = result.AccessExpiry
	response.MustChangePassword = result.MustChangePassword
	response.User.Id = mustConvertGeneratedUserID(result.User.ID)
	response.User.Username = result.User.Username
	response.User.DisplayName = result.User.DisplayName

	return response
}

func toBootstrapResponse(payload pluginapi.AuthBootstrapPayload) generated.BootstrapResponse {
	menus := make([]struct {
		Code       string  `json:"code"`
		Icon       string  `json:"icon"`
		Path       string  `json:"path"`
		Permission string  `json:"permission"`
		Title      string  `json:"title"`
		TitleKey   *string `json:"title_key,omitempty"`
	}, 0, len(payload.Menus))
	for _, item := range payload.Menus {
		menus = append(menus, struct {
			Code       string  `json:"code"`
			Icon       string  `json:"icon"`
			Path       string  `json:"path"`
			Permission string  `json:"permission"`
			Title      string  `json:"title"`
			TitleKey   *string `json:"title_key,omitempty"`
		}{
			Code:       item.Code,
			Title:      item.Title,
			TitleKey:   optionalStringPointer(item.TitleKey),
			Path:       item.Path,
			Icon:       item.Icon,
			Permission: item.Permission,
		})
	}

	var response generated.BootstrapResponse
	response.User.Id = mustConvertGeneratedUserID(payload.User.ID)
	response.User.Username = payload.User.Username
	response.User.DisplayName = payload.User.DisplayName
	response.MustChangePassword = payload.MustChangePassword
	response.Roles = append([]string(nil), payload.Roles...)
	response.Permissions = append([]string(nil), payload.Permissions...)
	response.Menus = menus
	response.Locale.CurrentLocale = payload.Locale.CurrentLocale
	response.Locale.DefaultLocale = payload.Locale.DefaultLocale
	response.Locale.FallbackLocale = payload.Locale.FallbackLocale
	response.Locale.SupportedLocales = append([]string(nil), payload.Locale.SupportedLocales...)

	return response
}

func mustConvertGeneratedUserID(id uint64) int64 {
	if id > math.MaxInt64 {
		panic("auth generated response user id exceeds int64")
	}
	return int64(id)
}

func optionalStringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func toSessionSummaries(items []pluginapi.AuthSessionSummary) []generated.SessionSummary {
	summaries := make([]generated.SessionSummary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, generated.SessionSummary{
			SessionId: item.SessionID,
			CreatedAt: item.CreatedAt,
			ExpiresAt: item.ExpiresAt,
			Current:   item.Current,
		})
	}

	return summaries
}
