package httpx

import (
	"context"
	"fmt"
)

func (r *accessLogRepository) createAccessLog(
	ctx context.Context,
	queryer accessLogQueryer,
	input CreateAccessLogInput,
) (AccessLog, error) {
	record := normalizeCreateAccessLogInput(input)
	userIDValue, err := nullableUint64(record.UserID)
	if err != nil {
		return AccessLog{}, fmt.Errorf("create access log: %w", err)
	}

	args := []any{
		record.RequestID,
		record.TraceID,
		record.Method,
		record.Path,
		nullableString(record.Route),
		record.StatusCode,
		record.DurationMS,
		nullableString(record.ClientIP),
		nullableString(record.UserAgent),
		userIDValue,
		nullableString(record.Username),
		nullableInt64(record.RequestSize),
		nullableInt64(record.ResponseSize),
		record.StartedAt,
		record.OccurredAt,
	}

	query := fmt.Sprintf(`INSERT INTO access_logs (
		request_id,
		trace_id,
		method,
		path,
		route,
		status_code,
		duration_ms,
		client_ip,
		user_agent,
		user_id,
		username,
		request_size,
		response_size,
		started_at,
		occurred_at
	) VALUES (%s) RETURNING id`, r.placeholders(len(args)))

	var id int64
	if err := queryer.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return AccessLog{}, fmt.Errorf("create access log: %w", err)
	}
	if id < 0 {
		return AccessLog{}, fmt.Errorf("create access log: negative id %d", id)
	}

	//nolint:gosec // Negative values are rejected above, so this database-generated identifier stays non-negative.
	record.ID = uint64(id)
	return record, nil
}
