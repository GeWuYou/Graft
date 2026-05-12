// Package cli owns the explicit Cobra command tree for the server process.
//
// Runtime startup and database migration stay as separate subcommands so schema
// changes never happen implicitly during normal application boot.
package cli
