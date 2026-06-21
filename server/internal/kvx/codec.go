package kvx

import "encoding/json"

// EncodeJSON marshals a value to JSON bytes for KV storage.
func EncodeJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}

// DecodeJSON unmarshals JSON bytes into a value of type T, returning the decoded value and any error from unmarshaling.
func DecodeJSON[T any](value []byte) (T, error) {
	var decoded T
	err := json.Unmarshal(value, &decoded)
	return decoded, err
}
