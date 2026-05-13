// Package store defines the neutral persistence contracts exposed to plugins.
//
// Core owns these contracts so plugin code can depend on explicit repository
// capabilities without importing or leaking a concrete ORM client.
package store
