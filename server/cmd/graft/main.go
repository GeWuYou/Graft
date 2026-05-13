package main

import (
	"log"

	"graft/server/internal/cli"
)

// main executes the explicit Graft CLI entrypoint.
func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		log.Fatalf("execute graft command: %v", err)
	}
}
