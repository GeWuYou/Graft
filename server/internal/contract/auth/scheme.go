// Package auth 定义服务端运行时共享的稳定认证契约值。
package auth

// Scheme 标识稳定的 HTTP 认证方案令牌。
type Scheme string

// String 返回线上协议使用的认证方案名称。
func (s Scheme) String() string {
	return string(s)
}

// Prefix 返回 Authorization 请求头使用的规范方案前缀，并包含方案后的空格。
func (s Scheme) Prefix() string {
	return s.String() + " "
}

const (
	// Bearer 标识 HTTP Bearer 令牌认证方案。
	Bearer Scheme = "Bearer"
)
