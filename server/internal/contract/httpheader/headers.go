// Package httpheader defines stable HTTP header contracts shared by the server runtime.
package httpheader

// Name identifies a stable HTTP header contract name.
type Name string

// String returns the wire-format header name.
func (n Name) String() string {
	return string(n)
}

const (
	// AcceptLanguage carries the standard request locale preferences.
	AcceptLanguage Name = "Accept-Language"

	// Authorization carries the caller authentication scheme and token.
	Authorization Name = "Authorization"

	// Locale carries the platform-specific explicit locale override.
	Locale Name = "X-Graft-Locale"

	// RequestID carries the stable request identifier echoed across the response envelope.
	RequestID Name = "X-Request-Id"

	// TraceID carries a legacy-compatible upstream trace identifier fallback.
	TraceID Name = "X-Trace-Id"
)
