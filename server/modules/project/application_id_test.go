package project

import (
	"regexp"
	"testing"
)

func TestNewApplicationIDUsesPublicULIDFormat(t *testing.T) {
	t.Parallel()
	value := newApplicationID()
	if !regexp.MustCompile(`^app_[0-9A-HJKMNP-TV-Z]{26}$`).MatchString(value) {
		t.Fatalf("application id %q does not use app_<ULID> format", value)
	}
}
