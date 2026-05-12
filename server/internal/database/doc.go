// Package database opens the PostgreSQL-backed Ent client for the core runtime.
//
// Database ownership stays in core so plugins can depend on explicit
// repository contracts instead of constructing their own storage connections.
package database
