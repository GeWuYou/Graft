package main

import (
	"log"

	"graft/server/internal/cli"
)

// main 执行 Graft 的显式 CLI 入口。
//
// 根命令返回错误时直接以非零状态退出，保证脚本与运维入口可以可靠感知失败。
func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		log.Fatalf("execute graft command: %v", err)
	}
}
