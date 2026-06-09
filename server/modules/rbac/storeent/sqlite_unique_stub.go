// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

//go:build !cgo

package storeent

func isSQLiteUniqueViolation(error) bool {
	return false
}
