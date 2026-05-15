package main

import (
	"fmt"
	"log"

	"graft/server/internal/keygen"
)

const jwtSecretEnvKey = "GRAFT_AUTH_JWT_SECRET"

// main 输出一行可直接写入 `.env` 的 JWT secret 配置。
func main() {
	line, err := keygen.GenerateEnvLine(jwtSecretEnvKey)
	if err != nil {
		log.Fatalf("generate %s: %v", jwtSecretEnvKey, err)
	}

	fmt.Println(line)
}
