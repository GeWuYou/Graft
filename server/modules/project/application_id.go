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
	ulidTimestampBits          = 48
	byteMask                   = 0xff
)

// newApplicationID 生成不暴露 SQL 密钥的公开应用标识，格式为 `app_<ULID>`。
// 返回生成的应用标识字符串。
func newApplicationID() string {
	var value [16]byte
	// ULID stores a 48-bit millisecond timestamp followed by 80 random bits.
	timestamp := uint64(time.Now().UTC().UnixMilli()) & ((uint64(1) << ulidTimestampBits) - 1)
	for index := 5; index >= 0; index-- {
		value[index] = byte(timestamp & byteMask) // #nosec G115 -- the mask bounds the conversion to one byte.
		timestamp >>= byteBits
	}
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

// isApplicationID 校验字符串是否为格式正确的应用标识。
// 首尾空白会被忽略，标识必须包含 app_ 前缀及 26 个 Crockford Base32 字符。
// 返回 true 表示格式有效，否则返回 false。
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
