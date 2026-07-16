package contract

// Category 标识通知稳定的分类契约。
type Category string

// String 返回规范化的通知分类值。
func (c Category) String() string {
	return string(c)
}

const (
	// CategorySecurity 覆盖安全和审计通知。
	CategorySecurity Category = "SECURITY"
	// CategoryTask 覆盖定时任务和运行历史通知。
	CategoryTask Category = "TASK"
	// CategoryConfig 预留给配置变更通知。
	CategoryConfig Category = "CONFIG"
	// CategoryOperations 预留给运行时运维通知。
	CategoryOperations Category = "OPERATIONS"
	// CategorySystem 预留给平台系统通知。
	CategorySystem Category = "SYSTEM"
)

// ValidCategory 判断 value 是否为已知的通知分类契约。
func ValidCategory(value Category) bool {
	switch value {
	case CategorySecurity, CategoryTask, CategoryConfig, CategoryOperations, CategorySystem:
		return true
	default:
		return false
	}
}
