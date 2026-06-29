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
	if id >= 0 {
		record.ID = uint64(id)
	}
	record.TraceID = traceID.String
	record.Route = route.String
	record.ClientIP = clientIP.String
	record.UserAgent = userAgent.String
	record.Username = username.String
	record.UserID = nullableScannedUint64(userID)
	record.RequestSize = nullableScannedInt64(requestSize)
	record.ResponseSize = nullableScannedInt64(responseSize)
	record.StartedAt = startedAt.UTC()
	record.OccurredAt = record.OccurredAt.UTC()

	return record, nil
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableUint64(value *uint64) (any, error) {
	if value == nil {
		return nil, nil
	}
	if *value > math.MaxInt64 {
		return nil, fmt.Errorf("user id %d exceeds bigint range", *value)
	}

	return int64(*value), nil
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func cloneUint64Pointer(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func nullableScannedUint64(value sql.NullInt64) *uint64 {
	if !value.Valid || value.Int64 < 0 {
		return nil
	}
	converted := uint64(value.Int64)
	return &converted
}

func nullableScannedInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	converted := value.Int64
	return &converted
}
