// Package keys 定义 cachex 使用的稳定缓存键组成规则。
package keys

import (
	"fmt"
	"strings"
)

// Key 表示由命名空间、名称和路径片段组成的缓存键。
type Key struct {
	namespace string
	name      string
	parts     []string
}

// New 创建经过校验和规范化的缓存键；所有输入会去除首尾空白，空值或包含冒号时返回错误。
func New(namespace string, name string, parts ...string) (Key, error) {
	trimmedNamespace := strings.TrimSpace(namespace)
	if trimmedNamespace == "" {
		return Key{}, fmt.Errorf("cache key namespace is required")
	}
	if strings.Contains(trimmedNamespace, ":") {
		return Key{}, fmt.Errorf("cache key namespace must not contain ':'")
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return Key{}, fmt.Errorf("cache key name is required")
	}
	if strings.Contains(trimmedName, ":") {
		return Key{}, fmt.Errorf("cache key name must not contain ':'")
	}

	normalizedParts := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart == "" {
			return Key{}, fmt.Errorf("cache key part is required")
		}
		if strings.Contains(trimmedPart, ":") {
			return Key{}, fmt.Errorf("cache key part must not contain ':'")
		}
		normalizedParts = append(normalizedParts, trimmedPart)
	}

	return Key{
		namespace: trimmedNamespace,
		name:      trimmedName,
		parts:     normalizedParts,
	}, nil
}

// MustNew 创建经过校验的缓存键；输入不合法时 panic，适用于静态且应始终有效的键定义。
func MustNew(namespace string, name string, parts ...string) Key {
	key, err := New(namespace, name, parts...)
	if err != nil {
		panic(err)
	}

	return key
}

// Namespace 返回缓存键的稳定命名空间。
func (k Key) Namespace() string {
	return k.namespace
}

// Name 返回缓存键的稳定名称。
func (k Key) Name() string {
	return k.name
}

// Parts 返回路径片段的防御性副本，调用方修改结果不会影响原键。
func (k Key) Parts() []string {
	cloned := make([]string, len(k.parts))
	copy(cloned, k.parts)
	return cloned
}

// String 将缓存键渲染为稳定的冒号分隔形式。
func (k Key) String() string {
	segments := []string{k.namespace, k.name}
	segments = append(segments, k.parts...)
	return strings.Join(segments, ":")
}
