package moduleapi_test

import (
	"reflect"
	"testing"

	"graft/server/internal/moduleapi"
)

func TestArtifactCopyContractsKeepSourceIdentityDigestAddressedAndBindingsPrivate(t *testing.T) {
	assertMissingFields(t, reflect.TypeOf(moduleapi.ArtifactPublicationSource{}), "Reference", "Tag", "Endpoint", "CredentialRef", "Secret")
	assertMissingFields(t, reflect.TypeOf(moduleapi.AuthorizedArtifactCopy{}), "Endpoint", "CredentialRef", "Secret")
}

func assertMissingFields(t *testing.T, typ reflect.Type, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, found := typ.FieldByName(name); found {
			t.Fatalf("%s must not expose %s", typ.Name(), name)
		}
	}
}
