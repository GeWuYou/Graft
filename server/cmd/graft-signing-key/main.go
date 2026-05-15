package main

import (
	"fmt"
	"log"

	"graft/server/internal/keygen"
)

const signingKeyEnvKey = "GRAFT_AUTH_SIGNING_KEY"

// main 输出一行可直接写入 `.env` 的 signing key 配置。
func main() {
	line, err := keygen.GenerateEnvLine(signingKeyEnvKey)
	if err != nil {
		log.Fatalf("generate %s: %v", signingKeyEnvKey, err)
	}

	fmt.Println(line)
}
