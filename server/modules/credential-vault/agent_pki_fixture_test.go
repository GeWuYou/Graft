package credentialvault

import (
	"context"
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
	"sync"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

var errAgentPKIFixtureRejected = errors.New("agent PKI fixture rejected")

// ephemeralAgentPKIFixture 只服务 Vault owner 的协议测试，绝不由模块 descriptor 或生产运行时接线。
type ephemeralAgentPKIFixture struct {
	mu       sync.Mutex
	ca       *x509.Certificate
	caKey    *ecdsa.PrivateKey
	caDER    []byte
	issued   map[string]fixtureIssuedCertificate
	revoked  map[string]bool
	now      func() time.Time
	serialID int64
}

type fixtureIssuedCertificate struct {
	request moduleapi.AgentCertificateIssuanceRequest
	result  moduleapi.IssuedAgentCertificate
}

func newEphemeralAgentPKIFixture(t *testing.T, now func() time.Time) *ephemeralAgentPKIFixture {
	t.Helper()
	if now == nil {
		now = time.Now
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate fixture CA key: %v", err)
	}
	createdAt := now().UTC()
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "graft-test-agent-ca"}, NotBefore: createdAt.Add(-time.Minute), NotAfter: createdAt.Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create fixture CA certificate: %v", err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse fixture CA certificate: %v", err)
	}
	return &ephemeralAgentPKIFixture{ca: certificate, caKey: key, caDER: der, issued: map[string]fixtureIssuedCertificate{}, revoked: map[string]bool{}, now: now, serialID: 1}
}

func (f *ephemeralAgentPKIFixture) IssueCSR(_ context.Context, request moduleapi.AgentCertificateIssuanceRequest) (moduleapi.IssuedAgentCertificate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if existing, ok := f.issued[request.IssuanceKey]; ok {
		if !sameFixtureIssuanceRequest(existing.request, request) {
			return moduleapi.IssuedAgentCertificate{}, errAgentPKIFixtureRejected
		}
		return cloneFixtureIssuedCertificate(existing.result), nil
	}
	csr, uri, fingerprint, err := validateFixtureCSRRequest(request)
	if err != nil {
		return moduleapi.IssuedAgentCertificate{}, errAgentPKIFixtureRejected
	}
	f.serialID++
	now := f.now().UTC()
	expiresAt := now.Add(time.Hour)
	template := &x509.Certificate{SerialNumber: big.NewInt(f.serialID), Subject: pkix.Name{CommonName: "graft-agent"}, NotBefore: now.Add(-time.Minute), NotAfter: expiresAt, URIs: []*url.URL{uri}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, f.ca, csr.PublicKey, f.caKey)
	if err != nil {
		return moduleapi.IssuedAgentCertificate{}, errAgentPKIFixtureRejected
	}
	result := moduleapi.IssuedAgentCertificate{IssuanceKey: request.IssuanceKey, CertificateSerial: template.SerialNumber.String(), CertificateChainDER: [][]byte{der, append([]byte(nil), f.caDER...)}, PublicKeyFingerprint: "sha256:" + fingerprint, ExpiresAt: expiresAt, TrustBundle: moduleapi.TrustBundleReference{Reference: "fixture://graft-agent-ca", Version: "fixture-v1", ExpiresAt: f.ca.NotAfter}}
	f.issued[request.IssuanceKey] = fixtureIssuedCertificate{request: cloneFixtureIssuanceRequest(request), result: cloneFixtureIssuedCertificate(result)}
	return result, nil
}

func (f *ephemeralAgentPKIFixture) ReconcileCSR(_ context.Context, issuanceKey string) (moduleapi.IssuedAgentCertificate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	issued, ok := f.issued[issuanceKey]
	if !ok {
		return moduleapi.IssuedAgentCertificate{}, moduleapi.ErrAgentCertificateIssuanceNotFound
	}
	return cloneFixtureIssuedCertificate(issued.result), nil
}

func (f *ephemeralAgentPKIFixture) ReadTrustBundle(_ context.Context, _ moduleapi.TrustBundleRequest) (moduleapi.TrustBundleReference, error) {
	return moduleapi.TrustBundleReference{Reference: "fixture://graft-agent-ca", Version: "fixture-v1", ExpiresAt: f.ca.NotAfter}, nil
}
func (f *ephemeralAgentPKIFixture) RevokeCertificate(_ context.Context, revocation moduleapi.AgentCertificateRevocation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.revoked[revocation.CertificateSerial] = true
	return nil
}

func validateFixtureCSRRequest(request moduleapi.AgentCertificateIssuanceRequest) (*x509.CertificateRequest, *url.URL, string, error) {
	if request.IssuanceKey == "" || request.TargetID < 1 || request.AgentID == "" || request.Generation < 1 || len(request.CSRDER) == 0 {
		return nil, nil, "", errAgentPKIFixtureRejected
	}
	csr, err := x509.ParseCertificateRequest(request.CSRDER)
	if err != nil || csr.CheckSignature() != nil {
		return nil, nil, "", errAgentPKIFixtureRejected
	}
	uri, err := url.Parse(request.SPIFFEURI)
	if err != nil || uri.Scheme != "spiffe" || uri.Host != "graft" {
		return nil, nil, "", errAgentPKIFixtureRejected
	}
	fingerprint := sha256.Sum256(csr.RawSubjectPublicKeyInfo)
	return csr, uri, hex.EncodeToString(fingerprint[:]), nil
}

func sameFixtureIssuanceRequest(left, right moduleapi.AgentCertificateIssuanceRequest) bool {
	return left.IdentityID == right.IdentityID && left.TargetID == right.TargetID && left.AgentID == right.AgentID && left.Generation == right.Generation && left.IssuanceKey == right.IssuanceKey && left.SPIFFEURI == right.SPIFFEURI && string(left.CSRDER) == string(right.CSRDER)
}
func cloneFixtureIssuanceRequest(request moduleapi.AgentCertificateIssuanceRequest) moduleapi.AgentCertificateIssuanceRequest {
	request.CSRDER = append([]byte(nil), request.CSRDER...)
	return request
}
func cloneFixtureIssuedCertificate(issued moduleapi.IssuedAgentCertificate) moduleapi.IssuedAgentCertificate {
	issued.CertificateChainDER = cloneFixtureCertificateChain(issued.CertificateChainDER)
	return issued
}
func cloneFixtureCertificateChain(chain [][]byte) [][]byte {
	result := make([][]byte, len(chain))
	for i := range chain {
		result[i] = append([]byte(nil), chain[i]...)
	}
	return result
}

func TestEphemeralAgentPKIFixtureSignsAndReconcilesExactCSR(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	fixture := newEphemeralAgentPKIFixture(t, func() time.Time { return now })
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CSR key: %v", err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{}, key)
	if err != nil {
		t.Fatalf("create CSR: %v", err)
	}
	request := moduleapi.AgentCertificateIssuanceRequest{IdentityID: "identity-7", TargetID: 7, AgentID: "agent-7", Generation: 3, IssuanceKey: "issuance-7", SPIFFEURI: "spiffe://graft/runtime-target/7/builder-agent/agent-7/generation/3", CSRDER: csrDER}
	if _, err := fixture.ReconcileCSR(context.Background(), request.IssuanceKey); !errors.Is(err, moduleapi.ErrAgentCertificateIssuanceNotFound) {
		t.Fatalf("first reconcile = %v", err)
	}
	issued, err := fixture.IssueCSR(context.Background(), request)
	if err != nil {
		t.Fatalf("issue CSR: %v", err)
	}
	leaf, err := x509.ParseCertificate(issued.CertificateChainDER[0])
	if err != nil {
		t.Fatalf("parse issued leaf: %v", err)
	}
	if len(leaf.URIs) != 1 || leaf.URIs[0].String() != request.SPIFFEURI || leaf.SerialNumber.String() != issued.CertificateSerial {
		t.Fatalf("issued leaf identity = %#v", leaf.URIs)
	}
	if reconciled, err := fixture.ReconcileCSR(context.Background(), request.IssuanceKey); err != nil || string(reconciled.CertificateChainDER[0]) != string(issued.CertificateChainDER[0]) {
		t.Fatalf("reconcile = %#v, %v", reconciled, err)
	}
	changed := request
	changed.Generation = 4
	if _, err := fixture.IssueCSR(context.Background(), changed); !errors.Is(err, errAgentPKIFixtureRejected) {
		t.Fatalf("changed issuance = %v", err)
	}
}

var _ VaultPKIAdapter = (*ephemeralAgentPKIFixture)(nil)
