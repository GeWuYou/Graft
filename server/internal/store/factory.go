package store

// Factory exposes the minimal repository set available to plugins.
//
// The MVP keeps this interface intentionally small and will only add new
// repository accessors when a plugin needs them.
type Factory interface {
	// Users returns the repository used by the user capability plugin.
	Users() UserRepository
}
