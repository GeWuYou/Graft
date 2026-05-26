package auth

import (
	"fmt"
	"math"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/pluginapi"
)

func toLoginResponse(result pluginapi.AuthRefreshResult) (generated.LoginResponse, error) {
	var response generated.LoginResponse
	response.AccessToken = result.AccessToken
	response.ExpiresAt = result.AccessExpiry
	response.MustChangePassword = result.MustChangePassword
	convertedID, err := mustConvertGeneratedUserID(result.User.ID)
	if err != nil {
		return generated.LoginResponse{}, err
	}
	response.User.Id = convertedID
	response.User.Username = result.User.Username
	response.User.DisplayName = result.User.DisplayName

	return response, nil
}

func toBootstrapResponse(payload pluginapi.AuthBootstrapPayload) (generated.BootstrapResponse, error) {
	menus := make([]generated.BootstrapMenu, 0, len(payload.Menus))
	for _, item := range payload.Menus {
		menus = append(menus, generated.BootstrapMenu{
			Code:       item.Code,
			Title:      item.Title,
			TitleKey:   optionalStringPointer(item.TitleKey),
			Path:       item.Path,
			Icon:       item.Icon,
			Permission: item.Permission,
		})
	}

	var response generated.BootstrapResponse
	convertedID, err := mustConvertGeneratedUserID(payload.User.ID)
	if err != nil {
		return generated.BootstrapResponse{}, err
	}
	response.User.Id = convertedID
	response.User.Username = payload.User.Username
	response.User.DisplayName = payload.User.DisplayName
	response.MustChangePassword = payload.MustChangePassword
	response.Roles = append([]string(nil), payload.Roles...)
	response.Permissions = append([]string(nil), payload.Permissions...)
	response.Menus = menus
	response.Locale = generated.BootstrapLocale{
		CurrentLocale:    payload.Locale.CurrentLocale,
		DefaultLocale:    payload.Locale.DefaultLocale,
		FallbackLocale:   payload.Locale.FallbackLocale,
		SupportedLocales: append([]string(nil), payload.Locale.SupportedLocales...),
	}

	return response, nil
}

func mustConvertGeneratedUserID(id uint64) (int64, error) {
	if id > math.MaxInt64 {
		return 0, fmt.Errorf("auth generated response user id exceeds int64: %d", id)
	}
	return int64(id), nil
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
