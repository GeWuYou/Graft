package contract

// MessageKey identifies a stable Security module message key.
type MessageKey string

// String returns the canonical message key value.
func (k MessageKey) String() string { return string(k) }

const (
	// OverviewMenuTitle identifies the Security overview menu label.
	OverviewMenuTitle MessageKey = "menu.security.overview.title"
	// OverviewReadDisplay identifies the display label for overview read permission.
	OverviewReadDisplay MessageKey = "security.permission.overview.read.display"
	// OverviewReadDescription identifies the overview read permission description.
	OverviewReadDescription MessageKey = "security.permission.overview.read.description"
)
