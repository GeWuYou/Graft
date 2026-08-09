package credentialvault

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	credentialvaultcontract "graft/server/modules/credential-vault/contract"
)

func TestNewModuleSpecUsesCredentialVaultModuleID(t *testing.T) {
	if got := NewModuleSpec().Name(); got != credentialvaultcontract.ModuleID {
		t.Fatalf("module id = %q, want %q", got, credentialvaultcontract.ModuleID)
	}
}

func TestModuleDoesNotRegisterMachineIdentityAuthorityWhenDisabled(t *testing.T) {
	services := container.New()
	if err := NewModule(config.CredentialVaultConfig{}, nil).Register(&module.Context{Services: services}); err != nil {
		t.Fatalf("register disabled credential vault module: %v", err)
	}
	_, err := services.Resolve((*moduleapi.MachineIdentityAuthority)(nil))
	if !errors.Is(err, container.ErrServiceNotRegistered) {
		t.Fatalf("resolve disabled authority error = %v, want service not registered", err)
	}
}

func TestModuleRegistersUnavailableAuthorityWhenEnabledWithoutAdapter(t *testing.T) {
	services := container.New()
	if err := NewModule(config.CredentialVaultConfig{Enabled: true}, nil).Register(&module.Context{Services: services}); err != nil {
		t.Fatalf("register enabled credential vault module: %v", err)
	}
	authority, err := module.ResolveService[moduleapi.MachineIdentityAuthority](services, (*moduleapi.MachineIdentityAuthority)(nil))
	if err != nil {
		t.Fatalf("resolve authority: %v", err)
	}
	_, err = authority.CreateEnrollment(context.Background(), moduleapi.MachineEnrollmentRequest{})
	if !errors.Is(err, ErrMachineIdentityAuthorityUnavailable) {
		t.Fatalf("create enrollment error = %v, want unavailable", err)
	}
}

func TestModuleRegistersProvidedVaultPKIAdapter(t *testing.T) {
	services := container.New()
	adapter := testVaultPKIAdapter{}
	if err := NewModule(config.CredentialVaultConfig{Enabled: true}, adapter).Register(&module.Context{Services: services}); err != nil {
		t.Fatalf("register credential vault module: %v", err)
	}
	authority, err := module.ResolveService[moduleapi.MachineIdentityAuthority](services, (*moduleapi.MachineIdentityAuthority)(nil))
	if err != nil {
		t.Fatalf("resolve authority: %v", err)
	}
	enrollment, err := authority.CreateEnrollment(context.Background(), moduleapi.MachineEnrollmentRequest{})
	if err != nil {
		t.Fatalf("create enrollment: %v", err)
	}
	if enrollment.IdentityID != "vault-identity" {
		t.Fatalf("identity id = %q", enrollment.IdentityID)
	}
}

type testVaultPKIAdapter struct{}

func (testVaultPKIAdapter) CreateEnrollment(context.Context, moduleapi.MachineEnrollmentRequest) (moduleapi.MachineEnrollment, error) {
	return moduleapi.MachineEnrollment{IdentityID: "vault-identity"}, nil
}

func (testVaultPKIAdapter) ActivateGeneration(context.Context, moduleapi.MachineIdentityActivation) error {
	return nil
}

func (testVaultPKIAdapter) RotateGeneration(context.Context, moduleapi.MachineIdentityRotationRequest) (moduleapi.MachineEnrollment, error) {
	return moduleapi.MachineEnrollment{}, nil
}

func (testVaultPKIAdapter) RevokeGeneration(context.Context, moduleapi.MachineIdentityRevocation) error {
	return nil
}

func (testVaultPKIAdapter) ReadTrustBundle(context.Context, moduleapi.TrustBundleRequest) (moduleapi.TrustBundleReference, error) {
	return moduleapi.TrustBundleReference{}, nil
}
