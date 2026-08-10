package runtimetarget

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"errors"
	"math/big"
	"net/url"
	"strings"
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
	csrDER := createBootstrapValidationCSR(t)
	_, fingerprint, err := parseBootstrapCSR(csrDER)
	if err != nil {
		t.Fatalf("parse test CSR: %v", err)
	}
	authorization := store.AgentBootstrapAuthorization{Issuance: store.AgentCertificateIssuance{IssuanceKey: "issuance-1"}, Generation: store.AgentTrustGeneration{Generation: 1, Identity: store.AgentTrustIdentity{TargetID: 7, AgentID: "agent-7"}}}
	issued := newBootstrapValidationCertificate(t, authorization, csrDER, now)
	if err := validateIssuedBootstrapCertificate(issued, authorization, fingerprint, now); err != nil {
		t.Fatalf("validate issued certificate: %v", err)
	}
	issued.IssuanceKey = "other"
	if err := validateIssuedBootstrapCertificate(issued, authorization, fingerprint, now); err == nil || errors.Is(err, moduleapi.ErrAgentCertificateIssuanceNotFound) {
		t.Fatalf("mismatched issuance validation error = %v", err)
	}
	wrongAuthorization := authorization
	wrongAuthorization.Generation.Identity.AgentID = "other"
	if err := validateIssuedBootstrapCertificate(issued, wrongAuthorization, fingerprint, now); err == nil {
		t.Fatal("certificate with mismatched SPIFFE URI unexpectedly accepted")
	}
}

func TestValidateIssuedBootstrapCertificateRejectsLegacyGenerationURI(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	csrDER := createBootstrapValidationCSR(t)
	_, fingerprint, err := parseBootstrapCSR(csrDER)
	if err != nil {
		t.Fatalf("parse test CSR: %v", err)
	}
	authorization := store.AgentBootstrapAuthorization{Issuance: store.AgentCertificateIssuance{IssuanceKey: "issuance-1"}, Generation: store.AgentTrustGeneration{Generation: 1, Identity: store.AgentTrustIdentity{TargetID: 7, AgentID: "agent-7"}}}
	issued := newBootstrapValidationCertificateForURI(t, authorization, csrDER, now, agentSPIFFEURI(authorization.Generation)+"/generation/1")
	if err := validateIssuedBootstrapCertificate(issued, authorization, fingerprint, now); err == nil {
		t.Fatal("certificate with legacy generation URI unexpectedly accepted")
	}
}

func TestValidAgentSPIFFEPathSegment(t *testing.T) {
	for _, value := range []string{"agent-7", "agent_7", "agent.7", "A7"} {
		if !validAgentSPIFFEPathSegment(value) {
			t.Fatalf("valid SPIFFE path segment %q rejected", value)
		}
	}
	for _, value := range []string{"", "agent/7", "agent 7", "agent?7", "agent#7", "agent%2F7"} {
		if validAgentSPIFFEPathSegment(value) {
			t.Fatalf("invalid SPIFFE path segment %q accepted", value)
		}
	}
}

func TestValidateIssuedBootstrapCertificateRejectsSecurityFailures(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	csrDER := createBootstrapValidationCSR(t)
	_, fingerprint, err := parseBootstrapCSR(csrDER)
	if err != nil {
		t.Fatalf("parse test CSR: %v", err)
	}
	authorization := store.AgentBootstrapAuthorization{Issuance: store.AgentCertificateIssuance{IssuanceKey: "issuance-1"}, Generation: store.AgentTrustGeneration{Generation: 1, Identity: store.AgentTrustIdentity{TargetID: 7, AgentID: "agent-7"}}}
	valid := newBootstrapValidationCertificate(t, authorization, csrDER, now)
	cases := map[string]func(*moduleapi.IssuedAgentCertificate){
		"certificate expired":  func(issued *moduleapi.IssuedAgentCertificate) { issued.ExpiresAt = now.Add(-time.Second) },
		"trust bundle expired": func(issued *moduleapi.IssuedAgentCertificate) { issued.TrustBundle.ExpiresAt = now.Add(-time.Second) },
		"empty chain":          func(issued *moduleapi.IssuedAgentCertificate) { issued.CertificateChainDER = nil },
		"leaf key mismatch": func(issued *moduleapi.IssuedAgentCertificate) {
			issued.PublicKeyFingerprint = "sha256:" + strings.Repeat("0", 64)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			issued := valid
			mutate(&issued)
			if err := validateIssuedBootstrapCertificate(issued, authorization, fingerprint, now); err == nil {
				t.Fatal("invalid issued certificate unexpectedly accepted")
			}
		})
	}
}

func createBootstrapValidationCSR(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate validation key: %v", err)
	}
	encoded, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{}, key)
	if err != nil {
		t.Fatalf("create validation CSR: %v", err)
	}
	return encoded
}

func newBootstrapValidationCertificate(t *testing.T, authorization store.AgentBootstrapAuthorization, csrDER []byte, now time.Time) moduleapi.IssuedAgentCertificate {
	return newBootstrapValidationCertificateForURI(t, authorization, csrDER, now, agentSPIFFEURI(authorization.Generation))
}

func newBootstrapValidationCertificateForURI(t *testing.T, authorization store.AgentBootstrapAuthorization, csrDER []byte, now time.Time, uri string) moduleapi.IssuedAgentCertificate {
	t.Helper()
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		t.Fatalf("parse validation CSR: %v", err)
	}
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate validation CA key: %v", err)
	}
	caTemplate := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test-ca"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(2 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create validation CA: %v", err)
	}
	ca, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse validation CA: %v", err)
	}
	identityURI, err := url.Parse(uri)
	if err != nil {
		t.Fatalf("parse expected SPIFFE URI: %v", err)
	}
	leafTemplate := &x509.Certificate{SerialNumber: big.NewInt(7), Subject: pkix.Name{CommonName: "agent"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour), URIs: []*url.URL{identityURI}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, ca, csr.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create validation leaf: %v", err)
	}
	fingerprint := sha256.Sum256(csr.RawSubjectPublicKeyInfo)
	return moduleapi.IssuedAgentCertificate{IssuanceKey: authorization.Issuance.IssuanceKey, CertificateSerial: "7", CertificateChainDER: [][]byte{leafDER, caDER}, PublicKeyFingerprint: "sha256:" + hex.EncodeToString(fingerprint[:]), ExpiresAt: now.Add(time.Hour), TrustBundle: moduleapi.TrustBundleReference{Reference: "vault:bundle", Version: "1", ExpiresAt: now.Add(2 * time.Hour)}}
}
