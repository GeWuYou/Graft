package httpx

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var accessLogAllowedListQueryKeys = map[string]struct{}{
	"page":            {},
	"page_size":       {},
	"request_id":      {},
	"trace_id":        {},
	"keyword":         {},
	"user_id":         {},
	"username":        {},
	"method":          {},
	"path":            {},
	"path_match":      {},
	"route":           {},
	"status_code":     {},
	"status_group":    {},
	"duration_min_ms": {},
	"duration_max_ms": {},
	"started_from":    {},
	"started_to":      {},
	"occurred_from":   {},
	"occurred_to":     {},
	"sort":            {},
	"sort[]":          {},
}

func bindAccessLogListQuery(ctx *gin.Context) (AccessLogListQuery, string) {
	query := AccessLogListQuery{}

	if invalidField := bindAccessLogPagination(ctx, &query); invalidField != "" {
		return query, invalidField
	}
	bindAccessLogIdentityFilters(ctx, &query)
	if invalidField := bindAccessLogNumericFilters(ctx, &query); invalidField != "" {
		return query, invalidField
	}
	if invalidField := bindAccessLogTimeFilters(ctx, &query); invalidField != "" {
		return query, invalidField
	}
	if invalidField := bindAccessLogOrdering(ctx, &query); invalidField != "" {
		return query, invalidField
	}
	if invalidField := rejectUnknownAccessLogListQueryKeys(ctx); invalidField != "" {
		return query, invalidField
	}

	return query, ""
}

func bindAccessLogPagination(ctx *gin.Context, query *AccessLogListQuery) string {
	page, ok, err := parseOptionalIntQueryValue(ctx.Query("page"))
	if err != nil {
		return "page"
	}
	if ok {
		query.Page = page
	}

	pageSize, ok, err := parseOptionalIntQueryValue(ctx.Query("page_size"))
	if err != nil {
		return "page_size"
	}
	if ok {
		query.PageSize = pageSize
	}

	return ""
}

func bindAccessLogIdentityFilters(ctx *gin.Context, query *AccessLogListQuery) {
	query.RequestID = strings.TrimSpace(ctx.Query("request_id"))
	query.TraceID = strings.TrimSpace(ctx.Query("trace_id"))
	query.Keyword = strings.TrimSpace(ctx.Query("keyword"))
	query.Username = strings.TrimSpace(ctx.Query("username"))
	query.Method = strings.TrimSpace(ctx.Query("method"))
	query.Path = strings.TrimSpace(ctx.Query("path"))
	query.Route = strings.TrimSpace(ctx.Query("route"))
}

func bindAccessLogNumericFilters(ctx *gin.Context, query *AccessLogListQuery) string {
	for _, bind := range []func(*gin.Context, *AccessLogListQuery) string{
		bindAccessLogUserIDFilter,
		bindAccessLogStatusCodeFilter,
		bindAccessLogDurationMinFilter,
		bindAccessLogDurationMaxFilter,
	} {
		if invalidKey := bind(ctx, query); invalidKey != "" {
			return invalidKey
		}
	}

	return ""
}

func bindAccessLogTimeFilters(ctx *gin.Context, query *AccessLogListQuery) string {
	if invalidKey := bindPrimaryAccessLogTimeFilters(ctx, query); invalidKey != "" {
		return invalidKey
	}

	if invalidKey := bindLegacyAccessLogTimeFilters(ctx, query); invalidKey != "" {
		return invalidKey
	}

	return ""
}

func bindAccessLogOrdering(ctx *gin.Context, query *AccessLogListQuery) string {
	if invalidKey := bindAccessLogPathMatch(ctx, query); invalidKey != "" {
		return invalidKey
	}

	if invalidKey := bindAccessLogStatusGroups(ctx, query); invalidKey != "" {
		return invalidKey
	}
	if invalidKey := bindAccessLogSorts(ctx, query); invalidKey != "" {
		return invalidKey
	}

	return ""
}

func bindPrimaryAccessLogTimeFilters(ctx *gin.Context, query *AccessLogListQuery) string {
	startedFrom, invalidKey := parseOptionalRFC3339QueryValue(ctx, "started_from")
	if invalidKey != "" {
		return invalidKey
	}
	query.StartedFrom = startedFrom

	startedTo, invalidKey := parseOptionalRFC3339QueryValue(ctx, "started_to")
	if invalidKey != "" {
		return invalidKey
	}
	query.StartedTo = startedTo

	occurredFrom, invalidKey := parseOptionalRFC3339QueryValue(ctx, "occurred_from")
	if invalidKey != "" {
		return invalidKey
	}
	query.OccurredFrom = occurredFrom

	occurredTo, invalidKey := parseOptionalRFC3339QueryValue(ctx, "occurred_to")
	if invalidKey != "" {
		return invalidKey
	}
	query.OccurredTo = occurredTo

	return ""
}

func bindLegacyAccessLogTimeFilters(_ *gin.Context, _ *AccessLogListQuery) string {
	return ""
}

func bindAccessLogPathMatch(ctx *gin.Context, query *AccessLogListQuery) string {
	pathMatch := strings.TrimSpace(ctx.Query("path_match"))
	switch pathMatch {
	case "", string(AccessLogPathMatchExact):
		query.PathMatchMode = AccessLogPathMatchExact
	case string(AccessLogPathMatchPrefix):
		query.PathMatchMode = AccessLogPathMatchPrefix
	default:
		return "path_match"
	}

	return ""
}

func bindAccessLogStatusGroups(ctx *gin.Context, query *AccessLogListQuery) string {
	rawValues := ctx.QueryArray("status_group")
	if len(rawValues) == 0 {
		return ""
	}

	groups := make([]AccessLogStatusGroup, 0, len(rawValues))
	for _, raw := range rawValues {
		switch AccessLogStatusGroup(strings.TrimSpace(raw)) {
		case AccessLogStatusGroup4xx:
			groups = append(groups, AccessLogStatusGroup4xx)
		case AccessLogStatusGroup5xx:
			groups = append(groups, AccessLogStatusGroup5xx)
		default:
			return "status_group"
		}
	}
	query.StatusGroups = groups
	return ""
}

func bindAccessLogSorts(ctx *gin.Context, query *AccessLogListQuery) string {
	rawSorts := queryArrayCompat(ctx, "sort")
	if len(rawSorts) == 0 {
		return ""
	}

	sorts := make([]AccessLogSort, 0, len(rawSorts))
	for _, raw := range rawSorts {
		parts := strings.Split(strings.TrimSpace(raw), ":")
		if len(parts) != accessLogSortPartCount {
			return "sort"
		}

		field := normalizeAccessLogSortField(AccessLogSortField(strings.TrimSpace(parts[0])))
		if field == "" {
			return "sort"
		}

		order := strings.ToLower(strings.TrimSpace(parts[1]))
		if order != string(AccessLogSortOrderAsc) && order != string(AccessLogSortOrderDesc) {
			return "sort"
		}

		sorts = append(sorts, AccessLogSort{
			Field: field,
			Order: AccessLogSortOrder(order),
		})
	}
	query.Sorts = sorts
	return ""
}

func queryArrayCompat(ctx *gin.Context, key string) []string {
	values := ctx.QueryArray(key)
	bracketValues := ctx.QueryArray(key + "[]")
	if len(bracketValues) == 0 {
		return values
	}

	combined := make([]string, 0, len(values)+len(bracketValues))
	combined = append(combined, values...)
	combined = append(combined, bracketValues...)
	return combined
}

func parseOptionalRFC3339QueryValue(ctx *gin.Context, key string) (*time.Time, string) {
	queryValue := strings.TrimSpace(ctx.Query(key))
	if queryValue == "" {
		return nil, ""
	}

	value, err := time.Parse(time.RFC3339, queryValue)
	if err != nil {
		return nil, key
	}

	return &value, ""
}

func rejectUnknownAccessLogListQueryKeys(ctx *gin.Context) string {
	for key := range ctx.Request.URL.Query() {
		if _, ok := accessLogAllowedListQueryKeys[key]; !ok {
			return key
		}
	}

	return ""
}

func parseOptionalIntQueryValue(raw string) (int, bool, error) {
	return parseOptionalQueryValue(raw, strconv.Atoi)
}

func parseOptionalInt64QueryValue(raw string) (int64, bool, error) {
	return parseOptionalQueryValue(raw, func(value string) (int64, error) {
		return strconv.ParseInt(value, 10, 64)
	})
}

func bindAccessLogUserIDFilter(ctx *gin.Context, query *AccessLogListQuery) string {
	return bindOptionalQueryValue(ctx.Query("user_id"), "user_id", func(value uint64) {
		query.UserID = &value
	}, parseOptionalUint64QueryValue)
}

func bindAccessLogStatusCodeFilter(ctx *gin.Context, query *AccessLogListQuery) string {
	return bindOptionalQueryValue(ctx.Query("status_code"), "status_code", func(value int) {
		query.StatusCode = &value
	}, parseOptionalIntQueryValue)
}

func bindAccessLogDurationMinFilter(ctx *gin.Context, query *AccessLogListQuery) string {
	return bindOptionalQueryValue(ctx.Query("duration_min_ms"), "duration_min_ms", func(value int64) {
		query.DurationMinMS = &value
	}, parseOptionalInt64QueryValue)
}

func bindAccessLogDurationMaxFilter(ctx *gin.Context, query *AccessLogListQuery) string {
	return bindOptionalQueryValue(ctx.Query("duration_max_ms"), "duration_max_ms", func(value int64) {
		query.DurationMaxMS = &value
	}, parseOptionalInt64QueryValue)
}

func bindOptionalQueryValue[T any](
	raw string,
	invalidKey string,
	assign func(T),
	parse func(string) (T, bool, error),
) string {
	value, ok, err := parse(raw)
	if err != nil {
		return invalidKey
	}
	if ok {
		assign(value)
	}
	return ""
}

func parseOptionalQueryValue[T any](
	raw string,
	parse func(string) (T, error),
) (T, bool, error) {
	var zero T

	value := strings.TrimSpace(raw)
	if value == "" {
		return zero, false, nil
	}

	parsed, err := parse(value)
	if err != nil {
		return zero, false, err
	}

	return parsed, true, nil
}

func parseOptionalUint64QueryValue(raw string) (uint64, bool, error) {
	return parseOptionalQueryValue(raw, func(value string) (uint64, error) {
		return strconv.ParseUint(value, 10, 64)
	})
}
