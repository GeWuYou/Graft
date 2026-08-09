package runtimetarget

import (
	"testing"

	credentialvaultcontract "graft/server/modules/credential-vault/contract"
)

func TestNewModuleSpecDependsOnSavedView(t *testing.T) {
	for _, dependency := range NewModuleSpec().Dependencies {
		if dependency == "saved-view" {
			return
		}
	}
	t.Fatal("runtime-target must depend on saved-view before resolving SavedViewService")
}

func TestNewModuleSpecDependsOnCredentialVault(t *testing.T) {
	for _, dependency := range NewModuleSpec().Dependencies {
		if dependency == credentialvaultcontract.ModuleID {
			return
		}
	}
	t.Fatal("runtime-target must depend on credential-vault before resolving AgentCertificateIssuer")
}
