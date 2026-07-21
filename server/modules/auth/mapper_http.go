package auth

import (
	"fmt"
	"math"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
)

func toLoginResponse(result moduleapi.AuthRefreshResult) (generated.LoginResponse, error) {
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

// toBootstrapResponse 将认证引导载荷转换为 HTTP 响应，并复制角色、权限、菜单和语言环境数据；用户 ID 超出 int64 范围时返回错误。
func toBootstrapResponse(payload moduleapi.AuthBootstrapPayload) (generated.BootstrapResponse, error) {
	menus := make([]generated.BootstrapMenu, 0, len(payload.Menus))
	for _, item := range payload.Menus {
		menus = append(menus, generated.BootstrapMenu{
			Code:            item.Code,
			ParentCode:      optionalStringPointer(item.ParentCode),
			Kind:            generated.BootstrapMenuKind(item.Kind),
			Title:           item.Title,
			TitleKey:        optionalStringPointer(item.TitleKey),
			SectionKey:      optionalStringPointer(item.SectionKey),
			SectionTitleKey: optionalStringPointer(item.SectionTitleKey),
			Path:            optionalStringPointer(item.Path),
			Icon:            item.Icon,
			Order:           optionalIntPointer(item.Order),
			Permission:      item.Permission,
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

func optionalIntPointer(value int) *int {
	return &value
}

func toSessionSummaries(items []moduleapi.AuthSessionSummary) []generated.SessionSummary {
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

func toPersonalAccessTokenSummaries(items []moduleapi.PersonalAccessTokenSummary) ([]generated.PersonalAccessTokenSummary, error) {
	summaries := make([]generated.PersonalAccessTokenSummary, 0, len(items))
	for _, item := range items {
		summary, err := toGeneratedPersonalAccessTokenSummary(item)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func toPersonalAccessTokenIssued(item moduleapi.PersonalAccessTokenIssued) (generated.PersonalAccessTokenIssued, error) {
	summary, err := toGeneratedPersonalAccessTokenSummary(item.Summary)
	if err != nil {
		return generated.PersonalAccessTokenIssued{}, err
	}
	return generated.PersonalAccessTokenIssued{
		CreatedAt:   summary.CreatedAt,
		ExpiresAt:   summary.ExpiresAt,
		Id:          summary.Id,
		LastUsedAt:  summary.LastUsedAt,
		Name:        summary.Name,
		RevokedAt:   summary.RevokedAt,
		Scopes:      summary.Scopes,
		Token:       item.Token,
		TokenPrefix: summary.TokenPrefix,
	}, nil
}

func toGeneratedPersonalAccessTokenSummary(item moduleapi.PersonalAccessTokenSummary) (generated.PersonalAccessTokenSummary, error) {
	id, err := mustConvertGeneratedUserID(item.ID)
	if err != nil {
		return generated.PersonalAccessTokenSummary{}, err
	}
	return generated.PersonalAccessTokenSummary{
		CreatedAt:   item.CreatedAt,
		ExpiresAt:   item.ExpiresAt,
		Id:          id,
		LastUsedAt:  item.LastUsedAt,
		Name:        item.Name,
		RevokedAt:   item.RevokedAt,
		Scopes:      append([]string(nil), item.Scopes...),
		TokenPrefix: item.TokenPrefix,
	}, nil
}
