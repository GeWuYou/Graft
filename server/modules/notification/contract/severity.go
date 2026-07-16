package contract

// Severity 标识通知稳定的严重程度契约。
type Severity string

// String 返回规范化的通知严重程度值。
func (s Severity) String() string {
	return string(s)
}

const (
	// SeverityInfo 表示提示性通知。
	SeverityInfo Severity = "info"
	// SeverityWarning 表示需要关注的通知。
	SeverityWarning Severity = "warning"
	// SeverityError 表示明确的失败事件。
	SeverityError Severity = "error"
	// SeverityCritical 表示高风险或高影响事件。
	SeverityCritical Severity = "critical"
)

// ValidSeverity 判断 value 是否为已知的通知严重程度契约。
func ValidSeverity(value Severity) bool {
	switch value {
	case SeverityInfo, SeverityWarning, SeverityError, SeverityCritical:
		return true
	default:
		return false
	}
}
