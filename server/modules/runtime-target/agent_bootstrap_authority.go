package runtimetarget

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

// runtimeTargetAgentBootstrapAuthority 协调 Runtime Target 授权与 Credential Vault PKI 外部副作用。
type runtimeTargetAgentBootstrapAuthority struct {
	repository *store.SQLRepository
	pepper     *config.EnrollmentPepperProvider
	issuer     moduleapi.AgentCertificateIssuer
	now        func() time.Time
	random     io.Reader
}

func newRuntimeTargetAgentBootstrapAuthority(repository *store.SQLRepository, pepper *config.EnrollmentPepperProvider, issuer moduleapi.AgentCertificateIssuer) moduleapi.AgentBootstrapAuthority {
	return runtimeTargetAgentBootstrapAuthority{repository: repository, pepper: pepper, issuer: issuer, now: time.Now, random: rand.Reader}
}

// BootstrapAgent 验证一次性材料、协调 Vault 签发并原子激活绑定世代。
//
//nolint:gocognit,cyclop // token、CSR、签发协调与激活的失败关闭分支必须位于同一服务边界。
func (a runtimeTargetAgentBootstrapAuthority) BootstrapAgent(ctx context.Context, request moduleapi.AgentBootstrapRequest) (moduleapi.AgentBootstrapResult, error) {
	if a.repository == nil || a.issuer == nil || len(a.enrollmentPepper()) == 0 {
		return moduleapi.AgentBootstrapResult{}, moduleapi.ErrAgentBootstrapRejected
	}
	csr, fingerprint, err := parseBootstrapCSR(request.CSRDER)
	if err != nil || strings.TrimSpace(request.BootstrapToken) == "" {
		return moduleapi.AgentBootstrapResult{}, moduleapi.ErrAgentBootstrapRejected
	}
	issuanceKey, err := a.randomValue()
	if err != nil {
		return moduleapi.AgentBootstrapResult{}, moduleapi.ErrAgentBootstrapRejected
	}
	pepper := a.enrollmentPepper()
	authorization, _, err := a.repository.AuthorizeAgentCertificateIssuance(ctx, tokenVerifier(request.BootstrapToken, pepper), fingerprint, issuanceKey, a.currentTime())
	if err != nil {
		return moduleapi.AgentBootstrapResult{}, normalizeAgentBootstrapError(err)
	}
	issued, err := a.resolveIssuedCertificate(ctx, authorization, csr.Raw)
	if err != nil {
		return moduleapi.AgentBootstrapResult{}, normalizeAgentBootstrapError(err)
	}
	if err := validateIssuedBootstrapCertificate(issued, authorization, fingerprint, a.currentTime()); err != nil {
		return moduleapi.AgentBootstrapResult{}, moduleapi.ErrAgentBootstrapRejected
	}
	if _, _, err := a.repository.RecordIssuedAgentCertificate(ctx, issuanceFromCertificate(authorization.Issuance.IssuanceKey, issued), a.currentTime()); err != nil {
		return moduleapi.AgentBootstrapResult{}, normalizeAgentBootstrapError(err)
	}
	if _, _, err := a.repository.CompleteAgentCertificateIssuance(ctx, authorization.Issuance.IssuanceKey, a.currentTime()); err != nil {
		return moduleapi.AgentBootstrapResult{}, normalizeAgentBootstrapError(err)
	}
	return moduleapi.AgentBootstrapResult{CertificateChainDER: cloneCertificateChain(issued.CertificateChainDER), TrustBundle: issued.TrustBundle, ExpiresAt: issued.ExpiresAt}, nil
}

func (a runtimeTargetAgentBootstrapAuthority) resolveIssuedCertificate(ctx context.Context, authorization store.AgentBootstrapAuthorization, csrDER []byte) (moduleapi.IssuedAgentCertificate, error) {
	issued, err := a.issuer.ReconcileCSR(ctx, authorization.Issuance.IssuanceKey)
	if err == nil {
		return issued, nil
	}
	if !errors.Is(err, moduleapi.ErrAgentCertificateIssuanceNotFound) {
		return moduleapi.IssuedAgentCertificate{}, err
	}
	return a.issuer.IssueCSR(ctx, moduleapi.AgentCertificateIssuanceRequest{IdentityID: authorization.Generation.Identity.IdentityID, TargetID: authorization.Generation.Identity.TargetID, AgentID: authorization.Generation.Identity.AgentID, Generation: authorization.Generation.Generation, IssuanceKey: authorization.Issuance.IssuanceKey, SPIFFEURI: agentSPIFFEURI(authorization.Generation), CSRDER: append([]byte(nil), csrDER...)})
}

func (a runtimeTargetAgentBootstrapAuthority) enrollmentPepper() []byte {
	if a.pepper == nil {
		return nil
	}
	return a.pepper.Pepper()
}

func (a runtimeTargetAgentBootstrapAuthority) currentTime() time.Time {
	if a.now == nil {
		return time.Now().UTC()
	}
	return a.now().UTC()
}

func (a runtimeTargetAgentBootstrapAuthority) randomValue() (string, error) {
	reader := a.random
	if reader == nil {
		reader = rand.Reader
	}
	value := make([]byte, agentDeliveryTokenBytes)
	if _, err := io.ReadFull(reader, value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func parseBootstrapCSR(encoded []byte) (*x509.CertificateRequest, string, error) {
	csr, err := x509.ParseCertificateRequest(encoded)
	if err != nil || csr.CheckSignature() != nil {
		return nil, "", errors.New("bootstrap CSR is invalid")
	}
	fingerprint := sha256.Sum256(csr.RawSubjectPublicKeyInfo)
	return csr, hex.EncodeToString(fingerprint[:]), nil
}

func agentSPIFFEURI(generation store.AgentTrustGeneration) string {
	return fmt.Sprintf("spiffe://graft/runtime-target/%d/builder-agent/%s", generation.Identity.TargetID, generation.Identity.AgentID)
}

//nolint:cyclop // 证书 DTO、DER 叶证书、CSR 公钥和精确 URI 必须在同一激活门禁内联立校验。
func validateIssuedBootstrapCertificate(issued moduleapi.IssuedAgentCertificate, authorization store.AgentBootstrapAuthorization, csrFingerprint string, now time.Time) error {
	if issued.IssuanceKey != authorization.Issuance.IssuanceKey || strings.TrimSpace(issued.CertificateSerial) == "" || issued.PublicKeyFingerprint != "sha256:"+csrFingerprint || !issued.ExpiresAt.After(now) || strings.TrimSpace(issued.TrustBundle.Reference) == "" || strings.TrimSpace(issued.TrustBundle.Version) == "" || !issued.TrustBundle.ExpiresAt.After(now) || len(issued.CertificateChainDER) == 0 {
		return errors.New("issued bootstrap certificate is invalid")
	}
	leaf, err := x509.ParseCertificate(issued.CertificateChainDER[0])
	if err != nil {
		return errors.New("issued bootstrap certificate leaf is invalid")
	}
	leafFingerprint := sha256.Sum256(leaf.RawSubjectPublicKeyInfo)
	if issued.PublicKeyFingerprint != "sha256:"+hex.EncodeToString(leafFingerprint[:]) {
		return errors.New("issued bootstrap certificate public key does not match evidence")
	}
	expectedURI := agentSPIFFEURI(authorization.Generation)
	if len(leaf.URIs) != 1 || leaf.URIs[0].String() != expectedURI {
		return errors.New("issued bootstrap certificate identity does not match authorization")
	}
	return nil
}

func issuanceFromCertificate(issuanceKey string, issued moduleapi.IssuedAgentCertificate) store.AgentCertificateIssuance {
	return store.AgentCertificateIssuance{IssuanceKey: issuanceKey, CertificateIssuer: "vault-pki", CertificateSerial: issued.CertificateSerial, CertificatePublicKeyFingerprint: issued.PublicKeyFingerprint, CertificateExpiresAt: &issued.ExpiresAt, TrustBundleRef: issued.TrustBundle.Reference, TrustBundleVersion: issued.TrustBundle.Version, TrustBundleExpiresAt: &issued.TrustBundle.ExpiresAt}
}

func cloneCertificateChain(chain [][]byte) [][]byte {
	result := make([][]byte, len(chain))
	for index := range chain {
		result[index] = append([]byte(nil), chain[index]...)
	}
	return result
}

func normalizeAgentBootstrapError(err error) error {
	if errors.Is(err, store.ErrAgentDeliveryRejected) || errors.Is(err, store.ErrAgentTrustNotFound) || errors.Is(err, moduleapi.ErrAgentCertificateIssuanceNotFound) {
		return moduleapi.ErrAgentBootstrapRejected
	}
	return err
}
