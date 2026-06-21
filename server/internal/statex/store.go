package statex

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// TimeSeriesStore stores timestamp-scored serialized state entries.
type TimeSeriesStore interface {
	Append(context.Context, string, TimeSeriesSample, RetentionPolicy) error
	Range(context.Context, string, TimeSeriesQuery) ([]TimeSeriesSample, error)
}

// TimeSeriesSample is one serialized authority-state sample.
type TimeSeriesSample struct {
	ObservedAt time.Time
	Payload    []byte
}

// TimeSeriesQuery describes one score/time window query.
type TimeSeriesQuery struct {
	StartAt time.Time
	EndAt   time.Time
}

// RetentionPolicy trims old samples and sets TTL for the whole series key.
type RetentionPolicy struct {
	TrimBefore   time.Time
	ExpiresAfter time.Duration
}

type redisTimeSeriesStore struct {
	client redis.Cmdable
}

// NewRedisTimeSeriesStore 使用 Redis 客户端创建一个 TimeSeriesStore。若客户端为 nil，则返回错误。
func NewRedisTimeSeriesStore(client redis.Cmdable) (TimeSeriesStore, error) {
	if client == nil {
		return nil, errors.New("statex redis client is required")
	}

	return &redisTimeSeriesStore{client: client}, nil
}

// Append stores one serialized sample under the given key using the sample time as the score.
func (s *redisTimeSeriesStore) Append(
	ctx context.Context,
	key string,
	sample TimeSeriesSample,
	policy RetentionPolicy,
) error {
	if err := validateKey(key); err != nil {
		return err
	}

	observedAt := sample.ObservedAt.UTC()
	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(observedAt.UnixMilli()),
		Member: encodeMember(observedAt, sample.Payload),
	})
	if !policy.TrimBefore.IsZero() {
		pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(policy.TrimBefore.UTC().UnixMilli(), 10))
	}
	if policy.ExpiresAfter > 0 {
		pipe.Expire(ctx, key, policy.ExpiresAfter)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("append time-series sample: %w", err)
	}

	return nil
}

// Range returns serialized samples ordered by score within the requested time window.
func (s *redisTimeSeriesStore) Range(ctx context.Context, key string, query TimeSeriesQuery) ([]TimeSeriesSample, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	members, err := s.client.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:     key,
		Start:   strconv.FormatInt(query.StartAt.UTC().UnixMilli(), 10),
		Stop:    strconv.FormatInt(query.EndAt.UTC().UnixMilli(), 10),
		ByScore: true,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("range time-series samples: %w", err)
	}

	samples := make([]TimeSeriesSample, 0, len(members))
	for _, member := range members {
		payload, err := decodeMember(member.Member)
		if err != nil {
			return nil, fmt.Errorf("decode time-series sample member: %w", err)
		}
		samples = append(samples, TimeSeriesSample{
			ObservedAt: time.UnixMilli(int64(member.Score)).UTC(),
			Payload:    payload,
		})
	}

	return samples, nil
}

// encodeMember formats the given timestamp and payload bytes into a string member for Redis ZSET storage.
func encodeMember(observedAt time.Time, payload []byte) string {
	return strconv.FormatInt(observedAt.UnixNano(), 10) + "|" + base64.RawStdEncoding.EncodeToString(payload)
}

// decodeMember 从编码的成员字符串中提取负载。
//
// 成员应采用格式 "<timestamp>|<base64编码的负载>"。
// 如果成员中不存在 "|" 分隔符，将整个原始值作为负载返回。
// 否则将 "|" 后的部分从 base64 解码。base64 解码失败时返回错误。
func decodeMember(member any) ([]byte, error) {
	raw, err := memberString(member)
	if err != nil {
		return nil, err
	}

	timestamp, encodedPayload, ok := strings.Cut(raw, "|")
	if !ok {
		return []byte(raw), nil
	}
	if _, err := strconv.ParseInt(timestamp, 10, 64); err != nil {
		return []byte(raw), nil
	}

	payload, err := base64.RawStdEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	return payload, nil
}

// memberString converts member to a string.
// It accepts string and []byte; other types return an error.
func memberString(member any) (string, error) {
	switch typed := member.(type) {
	case string:
		return typed, nil
	case []byte:
		return string(typed), nil
	default:
		return "", fmt.Errorf("unsupported member type %T", member)
	}
}

// Validate ensures the time query is bounded and ordered.
func (q TimeSeriesQuery) Validate() error {
	if q.StartAt.IsZero() {
		return errors.New("statex time-series query start time is required")
	}
	if q.EndAt.IsZero() {
		return errors.New("statex time-series query end time is required")
	}
	if q.EndAt.Before(q.StartAt) {
		return errors.New("statex time-series query end time must not be before start time")
	}
	return nil
}

// validateKey 验证 key 在去除空白后不为空。
func validateKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("statex key is required")
	}
	return nil
}
