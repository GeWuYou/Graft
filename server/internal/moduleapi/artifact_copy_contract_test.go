package moduleapi_test

import (
	"context"
	"reflect"
	"testing"

	"graft/server/internal/moduleapi"
)

func TestArtifactCopyContractsKeepSourceIdentityDigestAddressedAndBindingsPrivate(t *testing.T) {
	assertMissingFields(t, reflect.TypeOf(moduleapi.ArtifactPublicationSource{}), "Reference", "Tag", "Endpoint", "CredentialRef", "Secret")
	assertMissingFields(t, reflect.TypeOf(moduleapi.AuthorizedArtifactCopy{}), "Endpoint", "CredentialRef", "Secret")
	if _, ok := any(copyCapabilityStub{}).(moduleapi.TargetBoundOCIArtifactCopyCapability); !ok {
		t.Fatal("copy capability contract is not satisfiable")
	}
}

func assertMissingFields(t *testing.T, typ reflect.Type, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, found := typ.FieldByName(name); found {
			t.Fatalf("%s must not expose %s", typ.Name(), name)
		}
	}
}

type copyCapabilityStub struct{}

func (copyCapabilityStub) CopyOCIArtifactOnTarget(context.Context, int64, moduleapi.OCIArtifactCopyInput, moduleapi.RegistryArtifactCopyBinding, moduleapi.DockerImageBuildLogSink) (moduleapi.OCIArtifactCopyResult, error) {
	return moduleapi.OCIArtifactCopyResult{}, nil
}
