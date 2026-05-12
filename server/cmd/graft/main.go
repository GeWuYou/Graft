package main

import (
	"log"

	"graft/server/internal/app"
	"graft/server/plugins/user"
)

// main assembles the MVP runtime shell and starts the HTTP process.
func main() {
	runtime, err := app.NewRuntime(user.NewPlugin())
	if err != nil {
		log.Fatalf("create runtime: %v", err)
	}

	if err := runtime.Run(); err != nil {
		log.Fatalf("run runtime: %v", err)
	}
}
