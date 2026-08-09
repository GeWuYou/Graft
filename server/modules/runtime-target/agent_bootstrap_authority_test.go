package runtimetarget

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

func TestParseBootstrapCSRVerifiesSignatureAndReturnsPublicKeyFingerprint(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	encoded, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{}, privateKey)
	if err != nil {
		t.Fatalf("create test CSR: %v", err)
	}
	csr, fingerprint, err := parseBootstrapCSR(encoded)
	if err != nil || csr == nil || len(fingerprint) != 64 {
		t.Fatalf("parse bootstrap CSR = %#v, %q, %v", csr, fingerprint, err)
	}
	encoded[len(encoded)-1] ^= 1
	if _, _, err := parseBootstrapCSR(encoded); err == nil {
		t.Fatal("tampered CSR unexpectedly accepted")
	}
}

func TestValidateIssuedBootstrapCertificateBindsIssuanceAndCSR(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	authorization := store.AgentBootstrapAuthorization{Issuance: store.AgentCertificateIssuance{IssuanceKey: "issuance-1"}}
	issued := moduleapi.IssuedAgentCertificate{IssuanceKey: "issuance-1", CertificateSerial: "serial-1", PublicKeyFingerprint: "sha256:abc", ExpiresAt: now.Add(time.Hour), TrustBundle: moduleapi.TrustBundleReference{Reference: "vault:bundle", Version: "1", ExpiresAt: now.Add(time.Hour)}, CertificateChainDER: [][]byte{{1}}}
	if err := validateIssuedBootstrapCertificate(issued, authorization, "abc", now); err != nil {
		t.Fatalf("validate issued certificate: %v", err)
	}
	issued.IssuanceKey = "other"
	if err := validateIssuedBootstrapCertificate(issued, authorization, "abc", now); err == nil || errors.Is(err, moduleapi.ErrAgentCertificateIssuanceNotFound) {
		t.Fatalf("mismatched issuance validation error = %v", err)
	}
}
