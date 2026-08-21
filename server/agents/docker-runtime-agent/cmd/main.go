// Package main 提供独立 Docker Runtime Agent 的进程入口。
package main

import agent "graft/server/agents/docker-runtime-agent/internal"

func main() {
	agent.RunCLI()
}
