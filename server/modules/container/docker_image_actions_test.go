package container

import (
	"errors"
	"testing"
)

func TestValidateDockerImageReference(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "alpine latest", "registry.example.com/app@sha256:abc", "bad\nimage", "../image"} {
		if err := validateDockerImageReference(value); !errors.Is(err, errInvalidDockerImageReference) {
			t.Fatalf("expected invalid reference for %q, got %v", value, err)
		}
	}
	for _, value := range []string{"alpine:3.20", "registry.example.com/team/app:stable", "sha256:abc123"} {
		if err := validateDockerImageReference(value); err != nil {
			t.Fatalf("expected valid reference %q, got %v", value, err)
		}
	}
}

func TestMapDockerImageRemoveErrorClassifiesImageInUse(t *testing.T) {
	t.Parallel()

	err := mapDockerImageRemoveError(errors.New("conflict: unable to delete image because it is being used by stopped container"))
	if !errors.Is(err, errDockerImageInUse) {
		t.Fatalf("expected image-in-use error, got %v", err)
	}
}
