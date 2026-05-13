// Package config loads the environment-first runtime settings for the Graft server.
//
// The package keeps Docker and local development on the same path: real
// environment variables have priority, while an optional .env file can seed
// local defaults without being committed to the repository.
package config
