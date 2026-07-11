package contract

import "strings"

const (
	// ProjectLifecycleMaxAdditionalArgs bounds the number of user-supplied Compose arguments.
	ProjectLifecycleMaxAdditionalArgs = 32
	// ProjectLifecycleMaxAdditionalArgLength bounds one user-supplied Compose argument.
	ProjectLifecycleMaxAdditionalArgLength = 256
)

// NormalizeLifecycleAdditionalArgs trims and validates the storage-safe form of
// user-supplied Compose arguments. Command-authority overrides remain owned by
// the project service, where Compose commands are assembled.
func NormalizeLifecycleAdditionalArgs(values []string) ([]string, bool) {
	if len(values) > ProjectLifecycleMaxAdditionalArgs {
		return nil, false
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		argument := strings.TrimSpace(value)
		if argument == "" || len(argument) > ProjectLifecycleMaxAdditionalArgLength || strings.ContainsAny(argument, "\r\n\x00") {
			return nil, false
		}
		normalized = append(normalized, argument)
	}
	return normalized, true
}
