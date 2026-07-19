package container

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
	"testing"

	cerrdefs "github.com/containerd/errdefs"

	containercontract "graft/server/modules/container/contract"
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

func TestMapDockerImageRemoveErrorUsesStableCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want error
		code containercontract.DockerImageRemoveErrorCode
	}{
		{name: "multiple tags", err: errors.New("conflict: unable to delete sha256:abc (must be forced) - image is referenced in multiple repositories"), want: errDockerImageMultipleTags, code: containercontract.DockerImageMultipleTagsError},
		{name: "image in use", err: errors.New("conflict: unable to delete image because it is being used by stopped container"), want: errDockerImageInUse, code: containercontract.DockerImageInUseError},
		{name: "not found", err: cerrdefs.ErrNotFound, want: errDockerImageNotFound, code: containercontract.DockerImageNotFoundError},
		{name: "not found daemon message", err: errors.New("No such image: sha256:abc"), want: errDockerImageNotFound, code: containercontract.DockerImageNotFoundError},
		{name: "timeout", err: context.DeadlineExceeded, want: errDockerImageTimeout, code: containercontract.DockerTimeout},
		{name: "runtime unavailable", err: fmt.Errorf("connect Docker socket: %w", syscall.ECONNREFUSED), want: errDockerImageRuntimeUnavailable, code: containercontract.DockerRuntimeUnavailable},
		{name: "communication", err: &net.OpError{Op: "read", Net: "unix", Err: syscall.ECONNRESET}, want: errDockerImageCommunication, code: containercontract.DockerCommunicationError},
		{name: "unknown", err: errors.New("daemon returned an unclassified error"), want: errDockerImageRemoveFailed, code: containercontract.DockerImageRemoveUnknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapped := mapDockerImageRemoveError(test.err)
			if !errors.Is(mapped, test.want) {
				t.Fatalf("expected %v, got %v", test.want, mapped)
			}
			if code := dockerImageRemoveErrorCodeFor(mapped); code != test.code {
				t.Fatalf("expected code %s, got %s", test.code, code)
			}
		})
	}
}
