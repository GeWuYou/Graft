package contract

// Category identifies a stable notification category contract.
type Category string

// String returns the canonical category value.
func (c Category) String() string {
	return string(c)
}

const (
	// CategorySecurity covers security and audit notifications.
	CategorySecurity Category = "SECURITY"
	// CategoryTask covers scheduled task and run-history notifications.
	CategoryTask Category = "TASK"
	// CategoryConfig is reserved for configuration notifications.
	CategoryConfig Category = "CONFIG"
	// CategoryOperations is reserved for runtime operations notifications.
	CategoryOperations Category = "OPERATIONS"
	// CategorySystem is reserved for platform system notifications.
	CategorySystem Category = "SYSTEM"
)

// ValidCategory reports whether value is a known category contract.
func ValidCategory(value Category) bool {
	switch value {
	case CategorySecurity, CategoryTask, CategoryConfig, CategoryOperations, CategorySystem:
		return true
	default:
		return false
	}
}
