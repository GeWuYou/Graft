// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package keys provides stable state-store key sanitization helpers.
package keys

import "strings"

var segmentReplacer = strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-", ".", "-")

// Segment normalizes a key segment for state storage, using the normalized fallback if the value becomes empty after normalization.
func Segment(value string, fallback string) string {
	sanitized := normalizeSegment(value)
	if sanitized == "" {
		return normalizeSegment(fallback)
	}
	return sanitized
}

// normalizeSegment 规范化段字符串，将其转换为小写、去除空白，并将特殊字符（如空格、斜杠、冒号、句点等）替换为连字符。若规范化后结果为空字符串，则返回空字符串。
func normalizeSegment(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}

	return segmentReplacer.Replace(trimmed)
}
