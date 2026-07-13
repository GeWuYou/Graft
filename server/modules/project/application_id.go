package project

import (
	"crypto/rand"
	"encoding/binary"
	"strings"
	"time"
)

const crockfordBase32 = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

const (
	applicationIDEncodedLength = 26
	base32GroupBits            = 5
	ulidLeadingPaddingBits     = 2
	byteBits                   = 8
	mostSignificantBit         = 7
	base32MostSignificantBit   = 4
)

// newApplicationID produces the public app_<ULID> identity without exposing the SQL key.
func newApplicationID() string {
	var value [16]byte
	binary.BigEndian.PutUint64(value[:8], uint64(time.Now().UTC().UnixMilli()))
	if _, err := rand.Read(value[6:]); err != nil {
		// The timestamp prefix still keeps the value well formed; collisions remain protected by DB uniqueness.
		binary.BigEndian.PutUint64(value[8:], uint64(time.Now().UTC().UnixNano()))
	}
	encoded := make([]byte, applicationIDEncodedLength)
	for group := range encoded {
		var digit byte
		for offset := 0; offset < base32GroupBits; offset++ {
			streamBit := group*base32GroupBits + offset
			if streamBit < ulidLeadingPaddingBits {
				continue // ULID is 128 bits encoded into a 130-bit base32 stream.
			}
			dataBit := streamBit - ulidLeadingPaddingBits
			if value[dataBit/byteBits]&(1<<uint(mostSignificantBit-dataBit%byteBits)) != 0 {
				digit |= 1 << uint(base32MostSignificantBit-offset)
			}
		}
		encoded[group] = crockfordBase32[digit]
	}
	return "app_" + string(encoded)
}

func isApplicationID(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != len("app_")+applicationIDEncodedLength || !strings.HasPrefix(value, "app_") {
		return false
	}
	for _, character := range value[len("app_"):] {
		if !strings.ContainsRune(crockfordBase32, character) {
			return false
		}
	}
	return true
}
