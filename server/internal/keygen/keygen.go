// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package keygen

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

const randomSecretBytes = 32

// GenerateEnvLine 生成一行可直接写入 `.env` 的密钥配置。
//
// 生成语义：
//   - 使用 `crypto/rand` 生成固定 32 字节随机值。
//   - 使用 URL-safe、无 padding 的 base64 文本输出，方便直接粘贴到 `.env`。
//   - 调用方负责决定具体环境变量名；该函数不关心运行时使用场景差异。
func GenerateEnvLine(envKey string) (string, error) {
	normalizedKey := strings.TrimSpace(envKey)
	if normalizedKey == "" {
		return "", fmt.Errorf("env key is required")
	}

	secret, err := generateSecret(randomSecretBytes)
	if err != nil {
		return "", err
	}

	return normalizedKey + "=" + secret, nil
}

func generateSecret(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("secret size must be greater than zero")
	}

	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("read random secret: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
