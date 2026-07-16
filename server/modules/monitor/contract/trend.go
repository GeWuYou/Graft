package contract

import "time"

const (
	// TrendRangeQueryKey 标识规范趋势范围查询参数。
	TrendRangeQueryKey      = "trend_range"
	tenMinuteTrendWindow    = 10 * time.Minute
	thirtyMinuteTrendWindow = 30 * time.Minute
)

// TrendRange 标识 monitor 模块稳定的趋势范围契约。
type TrendRange string

// String 返回线路传输格式的趋势范围值。
func (r TrendRange) String() string {
	return string(r)
}

const (
	// TrendRange10Minutes 标识默认的 10 分钟趋势窗口。
	TrendRange10Minutes TrendRange = "10m"
	// TrendRange30Minutes 标识 30 分钟趋势窗口。
	TrendRange30Minutes TrendRange = "30m"
	// TrendRange1Hour 标识 1 小时趋势窗口。
	TrendRange1Hour TrendRange = "1h"
)

// Duration 返回趋势范围对应的规范时长。
func (r TrendRange) Duration() time.Duration {
	switch r {
	case TrendRange30Minutes:
		return thirtyMinuteTrendWindow
	case TrendRange1Hour:
		return time.Hour
	default:
		return tenMinuteTrendWindow
	}
}
