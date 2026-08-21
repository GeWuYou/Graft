package container

import (
	"errors"
	"testing"
)

func TestValidateDockerImageReference(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "alpine latest", "registry.example.com/app @sha256:abc", "bad\nimage", "../image", "registry.example.com/app\u000bimage"} {
		if err := validateDockerImageReference(value); !errors.Is(err, errInvalidDockerImageReference) {
			t.Fatalf("expected invalid reference for %q, got %v", value, err)
		}
	}
	for _, value := range []string{"alpine:3.20", "registry.example.com/team/app:stable", "registry.example.com/app@sha256:abc", "sha256:abc123"} {
		if err := validateDockerImageReference(value); err != nil {
			t.Fatalf("expected valid reference %q, got %v", value, err)
		}
	}
}
