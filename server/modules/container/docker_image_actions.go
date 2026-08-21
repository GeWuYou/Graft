package container

import (
	"errors"
	"strings"
	"unicode"
)

const maxDockerImageReferenceLength = 512

var (
	errInvalidDockerImageReference = errors.New("invalid docker image reference")
	errDockerImageTagNotAssociated = errors.New("docker image tag does not reference image")
)

func validateDockerImageReference(value string) error {
	if value == "" || len(value) > maxDockerImageReferenceLength || strings.Contains(value, "..") || strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) >= 0 {
		return errInvalidDockerImageReference
	}
	return nil
}
