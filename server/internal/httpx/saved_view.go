package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
)

// SavedViewRequest is the owner-neutral transport shape used by consumer-owned saved-view routes.
type SavedViewRequest struct {
	Name           string          `json:"name"`
	QueryState     json.RawMessage `json:"query_state"`
	PageSize       int             `json:"page_size"`
	VisibleColumns []string        `json:"visible_columns"`
	IsDefault      bool            `json:"is_default"`
}

// SavedViewResponse is the shared wire representation returned by consumer-owned saved-view routes.
type SavedViewResponse struct {
	ID             uint64          `json:"id"`
	Name           string          `json:"name"`
	QueryState     json.RawMessage `json:"query_state"`
	PageSize       int             `json:"page_size"`
	VisibleColumns []string        `json:"visible_columns"`
	IsDefault      bool            `json:"is_default"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

// SavedViewQueryValueKind constrains a query-state field's JSON representation before an owner reuses its list binder.
type SavedViewQueryValueKind uint8

const (
	// SavedViewQueryString accepts a JSON string query-state value.
	SavedViewQueryString SavedViewQueryValueKind = iota
	// SavedViewQueryNumber accepts a JSON number query-state value.
	SavedViewQueryNumber
	// SavedViewQueryBool accepts a JSON boolean query-state value.
	SavedViewQueryBool
	// SavedViewQueryStringSlice accepts a JSON string array query-state value.
	SavedViewQueryStringSlice
)

// BindSavedViewRequest 将请求体绑定为已保存视图请求；绑定失败时返回本地化的 400 错误响应。
func BindSavedViewRequest(ctx *gin.Context, localizer *i18n.Service) (SavedViewRequest, bool) {
	var request SavedViewRequest
	if ctx == nil || ctx.ShouldBindJSON(&request) != nil {
		AbortLocalizedError(ctx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "body"})
		return SavedViewRequest{}, false
	}
	return request, true
}

// SavedViewOwnerID returns the authenticated user ID that owns the saved view.
// It returns false when the request context does not contain a valid user ID.
func SavedViewOwnerID(ctx *gin.Context) (uint64, bool) {
	if ctx == nil || ctx.Request == nil {
		return 0, false
	}
	auth, ok := moduleapi.RequestAuthContextFromContext(ctx.Request.Context())
	if !ok || auth.User == nil || auth.User.ID == 0 {
		return 0, false
	}
	return auth.User.ID, true
}

// SavedViewID parses the positive saved-view identifier from the `viewId` route parameter.
// It returns the identifier and true when parsing succeeds; otherwise, it returns 0 and false.
func SavedViewID(ctx *gin.Context) (uint64, bool) {
	if ctx == nil {
		return 0, false
	}
	id, err := strconv.ParseUint(strings.TrimSpace(ctx.Param("viewId")), 10, 64)
	return id, err == nil && id > 0
}

// ToSavedViewResponse 将存储的已保存视图转换为共享的 HTTP 响应表示。
// 当视图 ID 为零或查询状态不是有效 JSON 时返回错误；有效视图的时间字段格式化为 UTC RFC3339 字符串。
func ToSavedViewResponse(view moduleapi.SavedView) (SavedViewResponse, error) {
	if view.ID == 0 || !json.Valid(view.QueryState) {
		return SavedViewResponse{}, errors.New("invalid saved view")
	}
	return SavedViewResponse{
		ID:             view.ID,
		Name:           view.Name,
		QueryState:     append(json.RawMessage(nil), view.QueryState...),
		PageSize:       view.PageSize,
		VisibleColumns: append([]string(nil), view.VisibleColumns...),
		IsDefault:      view.IsDefault,
		CreatedAt:      view.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      view.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// ValidateSavedViewQueryState 验证保存视图查询状态的字段结构和语义。
// 返回 nil 表示查询状态有效，否则返回错误。
func ValidateSavedViewQueryState(
	raw json.RawMessage,
	fields map[string]SavedViewQueryValueKind,
	bind func(*gin.Context) string,
) error {
	var state map[string]json.RawMessage
	if !json.Valid(raw) || json.Unmarshal(raw, &state) != nil || state == nil {
		return errors.New("invalid saved-view query state")
	}
	values := make(url.Values, len(state))
	for key, value := range state {
		kind, ok := fields[key]
		if !ok || !appendSavedViewQueryValue(values, key, value, kind) {
			return errors.New("invalid saved-view query state")
		}
	}

	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	request := httptest.NewRequest(http.MethodGet, "/?"+values.Encode(), nil)
	ginCtx.Request = request
	if bind(ginCtx) != "" {
		return errors.New("invalid saved-view query state")
	}
	return nil
}

// appendSavedViewQueryValue 将 JSON 值按指定类型转换为查询参数并追加到 values。
// 如果类型未知或值转换失败，则返回 false；否则返回 true。
func appendSavedViewQueryValue(values url.Values, key string, raw json.RawMessage, kind SavedViewQueryValueKind) bool {
	switch kind {
	case SavedViewQueryString:
		return appendSavedViewString(values, key, raw)
	case SavedViewQueryNumber:
		return appendSavedViewNumber(values, key, raw)
	case SavedViewQueryBool:
		return appendSavedViewBool(values, key, raw)
	case SavedViewQueryStringSlice:
		return appendSavedViewStringSlice(values, key, raw)
	default:
		return false
	}
}

// appendSavedViewString 将 JSON 字符串值添加到指定的查询参数中。
// 如果 raw 无法解析为字符串，则返回 false；否则返回 true。
func appendSavedViewString(values url.Values, key string, raw json.RawMessage) bool {
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	values.Add(key, value)
	return true
}

// appendSavedViewNumber 将 JSON 数字值添加到查询参数中。
// 如果 raw 不是有效的 JSON 数字，则返回 false；否则返回 true。
func appendSavedViewNumber(values url.Values, key string, raw json.RawMessage) bool {
	var value json.Number
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return false
	}
	values.Add(key, value.String())
	return true
}

// appendSavedViewBool 将 JSON 布尔值添加为查询参数值。
// 如果 raw 不是有效的布尔值，则返回 false；否则返回 true。
func appendSavedViewBool(values url.Values, key string, raw json.RawMessage) bool {
	var value bool
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	values.Add(key, strconv.FormatBool(value))
	return true
}

// appendSavedViewStringSlice 将 JSON 字符串数组追加到指定的查询参数中。
// 如果 raw 无法解析为字符串数组，则返回 false；否则返回 true。
func appendSavedViewStringSlice(values url.Values, key string, raw json.RawMessage) bool {
	var valuesSlice []string
	if json.Unmarshal(raw, &valuesSlice) != nil {
		return false
	}
	for _, value := range valuesSlice {
		values.Add(key, value)
	}
	return true
}

// WriteSavedViewError writes an HTTP error response for a saved-view storage failure.
// It maps invalid input, conflicts, and missing views to their corresponding status
// codes and uses an internal-server-error response for other failures.
func WriteSavedViewError(ctx *gin.Context, localizer *i18n.Service, err error) {
	status := http.StatusInternalServerError
	messageKey := messagecontract.CommonInternalError.String()
	switch {
	case errors.Is(err, moduleapi.ErrSavedViewInvalidInput):
		status = http.StatusBadRequest
		messageKey = messagecontract.CommonInvalidArgument.String()
	case errors.Is(err, moduleapi.ErrSavedViewConflict):
		status = http.StatusConflict
		messageKey = messagecontract.CommonInvalidArgument.String()
	case errors.Is(err, moduleapi.ErrSavedViewNotFound):
		status = http.StatusNotFound
		messageKey = "common.not_found"
	}
	if status == http.StatusInternalServerError && err != nil {
		logUnreportedInternalError(ctx, zap.L(), err)
	}
	AbortLocalizedError(ctx, localizer, status, messageKey, nil)
}
