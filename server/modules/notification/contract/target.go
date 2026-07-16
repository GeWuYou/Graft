package contract

// TargetType 标识通知稳定的投递目标契约。
type TargetType string

// String 返回规范化的投递目标类型值。
func (t TargetType) String() string {
	return string(t)
}

const (
	// TargetUser 投递给一个用户 ID。
	TargetUser TargetType = "USER"
	// TargetRole 预留给按角色扇出投递。
	TargetRole TargetType = "ROLE"
	// TargetPermission 预留给按权限扇出投递。
	TargetPermission TargetType = "PERMISSION"
	// TargetSystem 预留给全系统扇出投递。
	TargetSystem TargetType = "SYSTEM"
)

// ValidTargetType 判断 value 是否为已知的投递目标契约。
func ValidTargetType(value TargetType) bool {
	switch value {
	case TargetUser, TargetRole, TargetPermission, TargetSystem:
		return true
	default:
		return false
	}
}
