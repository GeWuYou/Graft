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
}

// SavedViewResponse is the shared wire representation returned by consumer-owned saved-view routes.
type SavedViewResponse struct {
	ID             uint64          `json:"id"`
	Name           string          `json:"name"`
	QueryState     json.RawMessage `json:"query_state"`
	PageSize       int             `json:"page_size"`
	VisibleColumns []string        `json:"visible_columns"`
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

// BindSavedViewRequest binds the owner-neutral saved-view request body.
func BindSavedViewRequest(ctx *gin.Context, localizer *i18n.Service) (SavedViewRequest, bool) {
	var request SavedViewRequest
	if ctx == nil || ctx.ShouldBindJSON(&request) != nil {
		AbortLocalizedError(ctx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "body"})
		return SavedViewRequest{}, false
	}
	return request, true
}

// SavedViewOwnerID returns the authenticated user ID that owns a private view.
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

// SavedViewID reads and validates the saved-view identifier from a route parameter.
func SavedViewID(ctx *gin.Context) (uint64, bool) {
	if ctx == nil {
		return 0, false
	}
	id, err := strconv.ParseUint(strings.TrimSpace(ctx.Param("viewId")), 10, 64)
	return id, err == nil && id > 0
}

// ToSavedViewResponse maps a stored saved view to its shared HTTP representation.
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
		CreatedAt:      view.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      view.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// ValidateSavedViewQueryState rejects unknown or wrongly shaped fields, then reuses the owner list binder for semantic validation.
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

func appendSavedViewString(values url.Values, key string, raw json.RawMessage) bool {
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	values.Add(key, value)
	return true
}

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

func appendSavedViewBool(values url.Values, key string, raw json.RawMessage) bool {
	var value bool
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	values.Add(key, strconv.FormatBool(value))
	return true
}

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

// WriteSavedViewError writes a normalized response for saved-view storage failures.
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
	AbortLocalizedError(ctx, localizer, status, messageKey, nil)
}
