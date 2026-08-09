package credentialvault

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/event"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	credentialvaultcontract "graft/server/modules/credential-vault/contract"
)

func TestNewModuleSpecUsesCredentialVaultModuleID(t *testing.T) {
	if got := NewModuleSpec().Name(); got != credentialvaultcontract.ModuleID {
		t.Fatalf("module id = %q, want %q", got, credentialvaultcontract.ModuleID)
	}
}

func TestModuleDoesNotRegisterAgentCertificateIssuerWhenDisabled(t *testing.T) {
	services := container.New()
	if err := NewModule(config.CredentialVaultConfig{}, nil).Register(&module.Context{Services: services}); err != nil {
		t.Fatalf("register disabled credential vault module: %v", err)
	}
	_, err := services.Resolve((*moduleapi.AgentCertificateIssuer)(nil))
	if !errors.Is(err, container.ErrServiceNotRegistered) {
		t.Fatalf("resolve disabled authority error = %v, want service not registered", err)
	}
}

func TestModuleRegistersUnavailableIssuerWhenEnabledWithoutAdapter(t *testing.T) {
	services := container.New()
	if err := NewModule(config.CredentialVaultConfig{Enabled: true}, nil).Register(&module.Context{Services: services, EventRegistry: credentialVaultTestEventRegistry{}}); err != nil {
		t.Fatalf("register enabled credential vault module: %v", err)
	}
	issuer, err := module.ResolveService[moduleapi.AgentCertificateIssuer](services, (*moduleapi.AgentCertificateIssuer)(nil))
	if err != nil {
		t.Fatalf("resolve issuer: %v", err)
	}
	_, err = issuer.IssueCSR(context.Background(), moduleapi.AgentCertificateIssuanceRequest{})
	if !errors.Is(err, ErrAgentCertificateIssuerUnavailable) {
		t.Fatalf("issue CSR error = %v, want unavailable", err)
	}
	if _, err := issuer.ReconcileCSR(context.Background(), "issuance-1"); !errors.Is(err, ErrAgentCertificateIssuerUnavailable) {
		t.Fatalf("reconcile CSR error = %v, want unavailable", err)
	}
}

func TestModuleRegistersProvidedVaultPKIAdapter(t *testing.T) {
	services := container.New()
	adapter := testVaultPKIAdapter{}
	if err := NewModule(config.CredentialVaultConfig{Enabled: true}, adapter).Register(&module.Context{Services: services, EventRegistry: credentialVaultTestEventRegistry{}}); err != nil {
		t.Fatalf("register credential vault module: %v", err)
	}
	issuer, err := module.ResolveService[moduleapi.AgentCertificateIssuer](services, (*moduleapi.AgentCertificateIssuer)(nil))
	if err != nil {
		t.Fatalf("resolve issuer: %v", err)
	}
	certificate, err := issuer.IssueCSR(context.Background(), moduleapi.AgentCertificateIssuanceRequest{})
	if err != nil {
		t.Fatalf("issue CSR: %v", err)
	}
	if certificate.CertificateSerial != "vault-certificate" {
		t.Fatalf("certificate serial = %q", certificate.CertificateSerial)
	}
}

func TestModuleRejectsEnabledConfigurationWithoutEventRegistry(t *testing.T) {
	services := container.New()
	err := NewModule(config.CredentialVaultConfig{Enabled: true}, nil).Register(&module.Context{Services: services})
	if err == nil {
		t.Fatal("enabled credential vault module registered without an event registry")
	}
	_, err = services.Resolve((*moduleapi.AgentCertificateIssuer)(nil))
	if !errors.Is(err, container.ErrServiceNotRegistered) {
		t.Fatalf("issuer registered despite missing event registry: %v", err)
	}
}

type credentialVaultTestEventRegistry struct{}

func (credentialVaultTestEventRegistry) Register(event.Handler) error { return nil }

type testVaultPKIAdapter struct{}

func (testVaultPKIAdapter) IssueCSR(context.Context, moduleapi.AgentCertificateIssuanceRequest) (moduleapi.IssuedAgentCertificate, error) {
	return moduleapi.IssuedAgentCertificate{CertificateSerial: "vault-certificate"}, nil
}

func (testVaultPKIAdapter) ReconcileCSR(context.Context, string) (moduleapi.IssuedAgentCertificate, error) {
	return moduleapi.IssuedAgentCertificate{CertificateSerial: "vault-certificate"}, nil
}

func (testVaultPKIAdapter) ReadTrustBundle(context.Context, moduleapi.TrustBundleRequest) (moduleapi.TrustBundleReference, error) {
	return moduleapi.TrustBundleReference{}, nil
}

func (testVaultPKIAdapter) RevokeCertificate(context.Context, moduleapi.AgentCertificateRevocation) error {
	return nil
}
