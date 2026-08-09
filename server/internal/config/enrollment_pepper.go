package config

import (
	"bytes"
	"errors"
	"os"
	"strings"
)

// EnrollmentPepperProvider 保存启动时从安装级 secret file 读取的 enrollment pepper。
// 未配置文件表示当前安装尚未启用 Agent enrollment；调用方必须在创建 delivery grant 前确认 provider 可用。
type EnrollmentPepperProvider struct {
	pepper []byte
}

// NewEnrollmentPepperProvider 从已校验的部署配置构造 EnrollmentPepperProvider。
// 该函数只在构造期读取一次文件，且不在错误中泄露文件内容或路径。
func NewEnrollmentPepperProvider(security EnrollmentSecurityConfig) (*EnrollmentPepperProvider, error) {
	path := strings.TrimSpace(security.PepperFile)
	if path == "" {
		return nil, nil
	}

	contents, err := os.ReadFile(path) // #nosec G304 -- 路径来自已校验的安装级部署配置。
	if err != nil {
		return nil, errors.New("enrollment pepper source is unavailable")
	}
	if len(bytes.TrimSpace(contents)) == 0 {
		return nil, errors.New("enrollment pepper source is invalid")
	}

	return &EnrollmentPepperProvider{pepper: bytes.Clone(contents)}, nil
}

// Pepper 返回 enrollment pepper 的防御性副本。
// 调用方不得记录、持久化或向不受信任边界传播返回值。
func (p *EnrollmentPepperProvider) Pepper() []byte {
	if p == nil {
		return nil
	}
	return bytes.Clone(p.pepper)
}
