// Package main 提供独立 Docker Builder Agent 的进程入口。
package main

import agent "graft/server/agents/docker-builder-agent/internal"

func main() {
	agent.RunCLI()
}
