// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package kvx

import "encoding/json"

// EncodeJSON marshals one value for KV persistence.
func EncodeJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}

// DecodeJSON unmarshals one stored JSON value.
func DecodeJSON[T any](value []byte) (T, error) {
	var decoded T
	err := json.Unmarshal(value, &decoded)
	return decoded, err
}
