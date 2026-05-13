package main

import (
	"log"

	"graft/server/internal/cli"
)

// main 执行 Graft 的显式 CLI 入口。
func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		log.Fatalf("execute graft command: %v", err)
	}
}
