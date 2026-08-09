package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewEnrollmentPepperProviderReadsOnceAndReturnsDefensiveCopies(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "enrollment-pepper")
	if err := os.WriteFile(path, []byte("original-pepper"), 0o600); err != nil {
		t.Fatalf("write pepper fixture: %v", err)
	}

	provider, err := NewEnrollmentPepperProvider(EnrollmentSecurityConfig{PepperFile: path})
	if err != nil {
		t.Fatalf("construct pepper provider: %v", err)
	}
	if provider == nil {
		t.Fatal("expected configured pepper provider")
	}

	if err := os.WriteFile(path, []byte("changed-pepper"), 0o600); err != nil {
		t.Fatalf("replace pepper fixture: %v", err)
	}
	first := provider.Pepper()
	if got := string(first); got != "original-pepper" {
		t.Fatalf("pepper = %q, want startup value", got)
	}
	first[0] = 'x'
	if got := string(provider.Pepper()); got != "original-pepper" {
		t.Fatalf("pepper after caller mutation = %q, want defensive copy", got)
	}
}

func TestNewEnrollmentPepperProviderRejectsInvalidConfiguredSource(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	emptyPath := filepath.Join(directory, "empty")
	whitespacePath := filepath.Join(directory, "whitespace")
	if err := os.WriteFile(emptyPath, nil, 0o600); err != nil {
		t.Fatalf("write empty fixture: %v", err)
	}
	if err := os.WriteFile(whitespacePath, []byte(" \n\t "), 0o600); err != nil {
		t.Fatalf("write whitespace fixture: %v", err)
	}

	for _, testCase := range []struct {
		name string
		path string
		want string
	}{
		{name: "missing", path: filepath.Join(directory, "missing"), want: "unavailable"},
		{name: "empty", path: emptyPath, want: "invalid"},
		{name: "whitespace", path: whitespacePath, want: "invalid"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			provider, err := NewEnrollmentPepperProvider(EnrollmentSecurityConfig{PepperFile: testCase.path})
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("construct pepper provider error = %v, want %q", err, testCase.want)
			}
			if provider != nil {
				t.Fatalf("invalid source returned provider: %#v", provider)
			}
		})
	}
}

func TestNewEnrollmentPepperProviderAllowsUnconfiguredInstallation(t *testing.T) {
	t.Parallel()

	provider, err := NewEnrollmentPepperProvider(EnrollmentSecurityConfig{})
	if err != nil {
		t.Fatalf("construct unconfigured pepper provider: %v", err)
	}
	if provider != nil {
		t.Fatalf("unconfigured installation returned provider: %#v", provider)
	}
}
