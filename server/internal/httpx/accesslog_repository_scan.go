package httpx

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
)

type accessLogScanner interface {
	Scan(dest ...any) error
}

// scanAccessLog 将一行访问日志扫描并组装为 AccessLog。
// 如果扫描失败，返回空的 AccessLog 和错误；如果日志 ID 小于 0 或用户 ID 不能转换为有效的非负值，也返回错误。
// @returns 扫描并填充后的 AccessLog，以及对应的错误。
func scanAccessLog(scanner accessLogScanner) (AccessLog, error) {
	var (
		id           int64
		traceID      sql.NullString
		route        sql.NullString
		clientIP     sql.NullString
		userAgent    sql.NullString
		userID       sql.NullInt64
		username     sql.NullString
		requestSize  sql.NullInt64
		responseSize sql.NullInt64
		startedAt    time.Time
		record       AccessLog
	)

	if err := scanner.Scan(
		&id,
		&record.RequestID,
		&traceID,
		&record.Method,
		&record.Path,
		&route,
		&record.StatusCode,
		&record.DurationMS,
		&clientIP,
		&userAgent,
		&userID,
		&username,
		&requestSize,
		&responseSize,
		&startedAt,
		&record.OccurredAt,
	); err != nil {
		return AccessLog{}, err
	}
	if id < 0 {
		return AccessLog{}, fmt.Errorf("access log id must be non-negative: %d", id)
	}
	record.ID = uint64(id)
	userIDValue, err := nullableScannedUint64(userID)
	if err != nil {
		return AccessLog{}, err
	}
	record.TraceID = traceID.String
	record.Route = route.String
	record.ClientIP = clientIP.String
	record.UserAgent = userAgent.String
	record.Username = username.String
	record.UserID = userIDValue
	record.RequestSize = nullableScannedInt64(requestSize)
	record.ResponseSize = nullableScannedInt64(responseSize)
	record.StartedAt = startedAt.UTC()
	record.OccurredAt = record.OccurredAt.UTC()

	return record, nil
}

// nullableString 将字符串去除首尾空白后返回可空值。
// 如果结果为空字符串，则返回 nil；否则返回裁剪后的字符串。
func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

// nullableUint64 将可空的 uint64 转换为数据库可存储的整数值。
//
// @param value 待转换的用户 ID 指针。
// @returns 空值、转换后的 int64，或范围超出时的错误。
func nullableUint64(value *uint64) (any, error) {
	if value == nil {
		return nil, nil
	}
	if *value > math.MaxInt64 {
		return nil, fmt.Errorf("user id %d exceeds bigint range", *value)
	}

	return int64(*value), nil
}

// nullableInt64 将整数指针转换为可空值。
// @return value 为 nil 时返回 nil；否则返回其指向的 int64 值。
func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

// cloneUint64Pointer 复制一个 uint64 指针所指向的值并返回其新地址。
// 当输入为 nil 时，返回 nil。
func cloneUint64Pointer(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

// cloneInt64Pointer 返回 value 的独立副本指针。
func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

// nullableScannedUint64 将可空的 int64 扫描值转换为 uint64 指针。
// 当值无效时返回 nil；当值小于 0 时返回错误。
//
// @param value 扫描得到的可空整数值。
// @returns 转换后的 uint64 指针，或在值无效时返回 nil；当值为负数时返回错误。
func nullableScannedUint64(value sql.NullInt64) (*uint64, error) {
	if !value.Valid {
		return nil, nil
	}
	if value.Int64 < 0 {
		return nil, fmt.Errorf("access log user id must be non-negative: %d", value.Int64)
	}
	converted := uint64(value.Int64)
	return &converted, nil
}

// nullableScannedInt64 将有效的 sql.NullInt64 转为 *int64。
// @returns value 有效时返回其整数值的指针，value 无效时返回 nil。
func nullableScannedInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	converted := value.Int64
	return &converted
}
