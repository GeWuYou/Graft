// Package app owns explicit runtime assembly for the Graft server process.
//
// The package keeps startup order visible in code so future core services and
// plugins can be wired without reflection, package scanning, or hidden
// lifecycle hooks.
package app
