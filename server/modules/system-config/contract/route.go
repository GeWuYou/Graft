package contract

const (
	// SystemConfigGroup is the API route group for system configuration management.
	SystemConfigGroup = "/system-configs"
	// SystemConfigCollectionRoute is the collection route fragment.
	SystemConfigCollectionRoute = ""
	// SystemConfigDetailRoute is the detail route fragment.
	SystemConfigDetailRoute = "/:key"
	// SystemConfigResetRoute is the reset route fragment.
	SystemConfigResetRoute = "/:key/reset"
	// SystemConfigMenuPath is the web menu path for the system configuration page.
	SystemConfigMenuPath = "/server/system-config"
)
