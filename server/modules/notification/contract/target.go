package contract

// TargetType identifies a stable notification delivery target contract.
type TargetType string

// String returns the canonical target type value.
func (t TargetType) String() string {
	return string(t)
}

const (
	// TargetUser delivers to one user ID.
	TargetUser TargetType = "USER"
	// TargetRole is reserved for role-based fan-out.
	TargetRole TargetType = "ROLE"
	// TargetPermission is reserved for permission-based fan-out.
	TargetPermission TargetType = "PERMISSION"
	// TargetSystem is reserved for system-wide fan-out.
	TargetSystem TargetType = "SYSTEM"
)

// ValidTargetType reports whether value is a known target contract.
func ValidTargetType(value TargetType) bool {
	switch value {
	case TargetUser, TargetRole, TargetPermission, TargetSystem:
		return true
	default:
		return false
	}
}
