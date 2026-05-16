// Package main 提供 JWT secret 辅助生成命令的进程入口。
package main

import (
	"fmt"
	"log"

	"graft/server/internal/keygen"
)

//nolint:gosec // 这里是配置键名而不是凭据本身，固定字符串用于输出 `.env` 模板行。
const jwtSecretEnvKey = "GRAFT_AUTH_JWT_SECRET"

// main 输出一行可直接写入 `.env` 的 JWT secret 配置。
func main() {
	line, err := keygen.GenerateEnvLine(jwtSecretEnvKey)
	if err != nil {
		log.Fatalf("generate %s: %v", jwtSecretEnvKey, err)
	}

	fmt.Println(line)
}
