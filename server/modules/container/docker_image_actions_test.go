package container

import (
	"errors"
	"strings"
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

func TestMapDockerImageRemoveErrorClassifiesImageInUse(t *testing.T) {
	t.Parallel()

	raw := "conflict: unable to delete image because it is being used by stopped container"
	err := mapDockerImageRemoveError(errors.New(raw))
	if !errors.Is(err, errDockerImageInUse) {
		t.Fatalf("expected image-in-use error, got %v", err)
	}
	if !strings.Contains(err.Error(), raw) {
		t.Fatalf("expected mapped error to retain daemon cause, got %v", err)
	}
}
