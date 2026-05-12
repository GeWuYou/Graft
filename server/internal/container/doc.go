// Package container provides explicit singleton registration and resolution.
//
// The implementation deliberately stays narrow: plugin and core code must still
// construct dependencies visibly and use string service keys instead of
// reflection-based auto wiring.
package container
