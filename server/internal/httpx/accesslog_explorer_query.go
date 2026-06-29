package httpx

import (
	"net/http"
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

// bindAccessLogListQuery 解析接入日志列表查询参数并返回结果。
// 解析过程中按顺序处理分页、身份与文本过滤、数值过滤、时间过滤、排序规则，并在最后校验是否包含未允许的查询键。
// 返回解析后的查询条件和第一个无效字段名；当全部成功时，无效字段名为空字符串。
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

// bindAccessLogPagination 解析接入日志列表查询的分页参数，并在有效时写入查询条件。
// 当 page 或 page_size 解析失败时，返回对应的参数名；成功时返回空字符串。
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

// bindAccessLogIdentityFilters 提取接入日志列表的身份与文本筛选参数。
// 它会对 request_id、trace_id、keyword、username、method、path 和 route 进行空白字符裁剪后写入查询条件。
func bindAccessLogIdentityFilters(ctx *gin.Context, query *AccessLogListQuery) {
	query.RequestID = strings.TrimSpace(ctx.Query("request_id"))
	query.TraceID = strings.TrimSpace(ctx.Query("trace_id"))
	query.Keyword = strings.TrimSpace(ctx.Query("keyword"))
	query.Username = strings.TrimSpace(ctx.Query("username"))
	query.Method = strings.TrimSpace(ctx.Query("method"))
	query.Path = strings.TrimSpace(ctx.Query("path"))
	query.Route = strings.TrimSpace(ctx.Query("route"))
}

// bindAccessLogNumericFilters 依次绑定接入日志列表查询中的数值过滤条件。
// 任何一个过滤条件解析失败时，返回对应的无效参数键；全部成功时返回空字符串。
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

	if invalidKey := validateAccessLogNumericFilters(query); invalidKey != "" {
		return invalidKey
	}

	return ""
}

// bindAccessLogTimeFilters 依次绑定接入日志列表的时间筛选条件。
// 返回第一个解析失败的查询键名；全部成功时返回空字符串。
func bindAccessLogTimeFilters(ctx *gin.Context, query *AccessLogListQuery) string {
	if invalidKey := bindPrimaryAccessLogTimeFilters(ctx, query); invalidKey != "" {
		return invalidKey
	}

	if invalidKey := bindLegacyAccessLogTimeFilters(ctx, query); invalidKey != "" {
		return invalidKey
	}

	return ""
}

// bindAccessLogOrdering 解析接入日志列表的路由匹配、状态分组和排序参数。
//
// 返回第一个无效参数键；全部有效时返回空字符串。
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

// bindPrimaryAccessLogTimeFilters 解析接入日志列表的主要时间筛选条件并写入查询对象。
// 支持 started_from、started_to、occurred_from 和 occurred_to，值必须为 RFC3339 时间格式。
// @return 返回空字符串表示成功；若某个字段解析失败，则返回该字段名。
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

// bindLegacyAccessLogTimeFilters 保留旧版接入日志时间筛选参数的兼容入口。
//
// @returns 始终返回空字符串。
func bindLegacyAccessLogTimeFilters(_ *gin.Context, _ *AccessLogListQuery) string {
	return ""
}

// bindAccessLogPathMatch 解析接入日志路径匹配模式并写入查询条件。
// 支持空值或 exact 模式，以及 prefix 模式；当参数值无效时返回 "path_match"。
// @returns 解析失败时返回无效参数键名；成功时返回空字符串。
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

// bindAccessLogStatusGroups 解析并设置访问日志列表的状态组筛选条件。
// 仅接受 `4xx` 和 `5xx` 状态组。
//
// @returns 解析成功时返回空字符串；当任一值无效时返回 `"status_group"`。
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

// bindAccessLogSorts 解析接入日志列表的排序条件并写入查询对象。
//
// @param ctx Gin 上下文。
// @param query 要写入排序条件的查询对象。
// @returns 解析成功时返回空字符串；当排序参数格式或取值无效时返回 "sort"。
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

// queryArrayCompat 返回指定查询参数及其 `[]` 形式参数的合并结果。
// 当 `key[]` 没有值时，仅返回 `key` 对应的值；否则按 `key`、`key[]` 的顺序合并返回。
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

// parseOptionalRFC3339QueryValue 解析指定查询参数中的可选 RFC3339 时间。
// 参数为空时返回空值；解析失败时返回参数名。
// @returns 当参数为空时返回 nil 和空字符串；当解析成功时返回解析后的时间指针和空字符串；当解析失败时返回 nil 和参数名。
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

// rejectUnknownAccessLogListQueryKeys 检查请求中的查询参数键是否都在允许列表内，并返回第一个未知键。
// 如果所有键都允许，则返回空字符串。
func rejectUnknownAccessLogListQueryKeys(ctx *gin.Context) string {
	for key := range ctx.Request.URL.Query() {
		if _, ok := accessLogAllowedListQueryKeys[key]; !ok {
			return key
		}
	}

	return ""
}

// parseOptionalIntQueryValue 解析可选的整数查询值。
//
// @param raw 原始查询字符串。
// @returns 解析得到的整数值；当输入为空时，ok 为 false；解析失败时返回错误。
func parseOptionalIntQueryValue(raw string) (int, bool, error) {
	return parseOptionalQueryValue(raw, strconv.Atoi)
}

// parseOptionalInt64QueryValue 解析可选的 int64 查询参数值。
// 空字符串表示参数缺失；非空时按十进制解析为 int64。
//
// @returns 解析后的值、是否存在有效输入，以及解析错误。
func parseOptionalInt64QueryValue(raw string) (int64, bool, error) {
	return parseOptionalQueryValue(raw, func(value string) (int64, error) {
		return strconv.ParseInt(value, 10, 64)
	})
}

// bindAccessLogUserIDFilter 解析并设置查询中的用户 ID 条件。
//
// @return 解析失败时返回 "user_id"，成功或参数缺失时返回空字符串。
func bindAccessLogUserIDFilter(ctx *gin.Context, query *AccessLogListQuery) string {
	return bindOptionalQueryValue(ctx.Query("user_id"), "user_id", func(value uint64) {
		query.UserID = &value
	}, parseOptionalUint64QueryValue)
}

// bindAccessLogStatusCodeFilter 绑定接入日志列表查询中的状态码筛选条件。
// @return 成功时返回空字符串；当参数值无法解析时返回 "status_code"。
func bindAccessLogStatusCodeFilter(ctx *gin.Context, query *AccessLogListQuery) string {
	return bindOptionalQueryValue(ctx.Query("status_code"), "status_code", func(value int) {
		query.StatusCode = &value
	}, parseOptionalIntQueryValue)
}

// bindAccessLogDurationMinFilter 绑定接入日志列表查询中的最小时长筛选条件。
//
// @returns 若参数 `duration_min_ms` 解析失败则返回该键名；否则返回空字符串。
func bindAccessLogDurationMinFilter(ctx *gin.Context, query *AccessLogListQuery) string {
	return bindOptionalQueryValue(ctx.Query("duration_min_ms"), "duration_min_ms", func(value int64) {
		query.DurationMinMS = &value
	}, parseOptionalInt64QueryValue)
}

// bindAccessLogDurationMaxFilter 解析并设置接入日志查询的最大耗时过滤条件。
// 成功时将 `duration_max_ms` 写入 `query.DurationMaxMS`；当参数存在且无法解析时返回对应的无效键名。
// @returns 解析失败时返回 `"duration_max_ms"`，否则返回空字符串。
func bindAccessLogDurationMaxFilter(ctx *gin.Context, query *AccessLogListQuery) string {
	return bindOptionalQueryValue(ctx.Query("duration_max_ms"), "duration_max_ms", func(value int64) {
		query.DurationMaxMS = &value
	}, parseOptionalInt64QueryValue)
}

// validateAccessLogNumericFilters 校验数值筛选条件的业务边界。
// 返回第一个无效字段名；全部有效时返回空字符串。
func validateAccessLogNumericFilters(query *AccessLogListQuery) string {
	if query == nil {
		return ""
	}
	if invalidKey := validateAccessLogStatusCodeFilter(query.StatusCode); invalidKey != "" {
		return invalidKey
	}
	if invalidKey := validateAccessLogDurationFilter("duration_min_ms", query.DurationMinMS); invalidKey != "" {
		return invalidKey
	}
	if invalidKey := validateAccessLogDurationFilter("duration_max_ms", query.DurationMaxMS); invalidKey != "" {
		return invalidKey
	}
	if query.DurationMinMS != nil && query.DurationMaxMS != nil && *query.DurationMinMS > *query.DurationMaxMS {
		return "duration_min_ms"
	}
	return ""
}

func validateAccessLogStatusCodeFilter(statusCode *int) string {
	if statusCode != nil && (*statusCode < http.StatusContinue || *statusCode > http.StatusNetworkAuthenticationRequired) {
		return "status_code"
	}
	return ""
}

func validateAccessLogDurationFilter(key string, value *int64) string {
	if value != nil && *value < 0 {
		return key
	}
	return ""
}

// bindOptionalQueryValue 解析可选字符串并在成功时赋值。
//
// 当 parse 返回错误时，返回 invalidKey；当参数存在且解析成功时调用 assign 写入结果；
// 参数为空时直接成功返回。
//
// @returns 成功时返回空字符串，解析失败时返回 invalidKey。
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

// 当解析成功时，返回解析后的值、true 和 nil。
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

// parseOptionalUint64QueryValue 解析可选的 uint64 查询参数值。
// 当输入为空时，返回零值和 false；当输入有效时，返回解析后的值和 true。
func parseOptionalUint64QueryValue(raw string) (uint64, bool, error) {
	return parseOptionalQueryValue(raw, func(value string) (uint64, error) {
		return strconv.ParseUint(value, 10, 64)
	})
}
