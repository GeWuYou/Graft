package user

import "testing"

func TestUserHeaderPointerReturnsTrimmedValue(t *testing.T) {
	value := userHeaderPointer(" zh-CN ")
	if value == nil {
		t.Fatal("expected non-nil header pointer")
	}
	if *value != "zh-CN" {
		t.Fatalf("expected trimmed header value, got %q", *value)
	}
}
