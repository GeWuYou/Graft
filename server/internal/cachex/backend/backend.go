// Package backend 提供 cachex 使用的后端适配器契约与实现。
package backend

import (
	"context"
	"time"
)

// Entry 是与具体后端无关的缓存存储载荷；Value 由实现负责防止外部修改。
type Entry struct {
	Value     []byte
	ExpiresAt time.Time
}

// Backend 定义 cachex 所需的最小机械存储操作，并以 bool 区分未命中与后端错误。
type Backend interface {
	Name() string
	Get(context.Context, string) (Entry, bool, error)
	Set(context.Context, string, Entry) error
	Delete(context.Context, string) error
}
