package runtimetarget

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	httpheader "graft/server/internal/contract/httpheader"
	"graft/server/internal/moduleapi"
)

const (
	ociRegistryVerificationTimeout = 10 * time.Second
	credentialSessionTTL           = 5 * time.Minute
)

var ociRegistryVerificationHTTPClient = &http.Client{
	Timeout:       ociRegistryVerificationTimeout,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
}

type runtimeOCIRegistryVerifier struct {
	targets     moduleapi.RuntimeTargetProviderConnectionReader
	credentials moduleapi.CredentialProvider
	materials   moduleapi.EphemeralCredentialMaterialProvider
}

// VerifyOCIRegistry probes only the OCI V2 root after Runtime Target resolves the explicit target.
// Credential material exists only in this call and is revoked before the method returns.
func (v runtimeOCIRegistryVerifier) VerifyOCIRegistry(ctx context.Context, request moduleapi.OCIRegistryVerificationRequest) (result moduleapi.OCIRegistryVerificationResult, err error) {
	if v.targets == nil || v.credentials == nil || v.materials == nil || !validOCIRegistryVerificationRequest(request) {
		return moduleapi.OCIRegistryVerificationResult{}, errors.New("OCI registry verification input is invalid")
	}
	if _, err := v.targets.GetProviderConnection(ctx, request.RuntimeTargetID); err != nil {
		return moduleapi.OCIRegistryVerificationResult{}, err
	}
	endpoint, err := ociRegistryV2Endpoint(request.Endpoint)
	if err != nil {
		return moduleapi.OCIRegistryVerificationResult{}, err
	}
	session, err := v.credentials.Prepare(ctx, moduleapi.CredentialRequest{CredentialRef: request.CredentialRef, Endpoint: request.Endpoint, RepositoryRef: request.RepositoryRef, Operation: request.Operation, ExpiresAt: time.Now().UTC().Add(credentialSessionTTL)})
	if err != nil {
		return moduleapi.OCIRegistryVerificationResult{}, err
	}
	defer func() {
		if revokeErr := v.credentials.Revoke(context.WithoutCancel(ctx), session); revokeErr != nil {
			err = errors.New("registry credential cleanup could not be verified")
		}
	}()
	material, err := v.materials.ResolveCredentialMaterial(ctx, session, moduleapi.CredentialInjectionTarget{Endpoint: request.Endpoint, RepositoryRef: request.RepositoryRef})
	if err != nil {
		return moduleapi.OCIRegistryVerificationResult{}, err
	}
	authorization := "Basic " + base64.StdEncoding.EncodeToString([]byte(material.Username+":"+material.Secret))
	result = probeOCIRegistryV2(ctx, endpoint, authorization)
	result.ProviderScopeConforms = true
	return result, nil
}

func probeOCIRegistryV2(ctx context.Context, endpoint *url.URL, authorization string) moduleapi.OCIRegistryVerificationResult {
	result := moduleapi.OCIRegistryVerificationResult{}
	unauthenticated, err := ociRegistryVerificationRequest(ctx, endpoint, "")
	if err != nil {
		return result
	}
	result.Reachable = true
	if unauthenticated.StatusCode != http.StatusOK && unauthenticated.StatusCode != http.StatusUnauthorized {
		return result
	}
	result.ProtocolCompatible = true
	result.AuthenticationChallenged = unauthenticated.StatusCode == http.StatusUnauthorized

	authenticated, err := ociRegistryVerificationRequest(ctx, endpoint, authorization)
	if err != nil {
		return result
	}
	if authenticated.StatusCode == http.StatusOK {
		result.AuthenticationSucceeded = true
	}
	return result
}

func ociRegistryV2Endpoint(raw string) (*url.URL, error) {
	endpoint, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || endpoint == nil || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || (endpoint.Scheme != "https" && endpoint.Scheme != "http") {
		return nil, errors.New("OCI registry verification endpoint is invalid")
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/v2/"
	return endpoint, nil
}

func ociRegistryVerificationRequest(ctx context.Context, endpoint *url.URL, authorization string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	if authorization != "" {
		request.Header.Set(string(httpheader.Authorization), authorization)
	}
	response, err := ociRegistryVerificationHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	return response, nil
}

func validOCIRegistryVerificationRequest(request moduleapi.OCIRegistryVerificationRequest) bool {
	return request.RuntimeTargetID > 0 && strings.TrimSpace(request.CredentialRef) != "" && strings.TrimSpace(request.Endpoint) != "" && strings.TrimSpace(request.RepositoryRef) != "" && strings.TrimSpace(request.Operation) != "" &&
		!strings.ContainsAny(request.CredentialRef+request.Endpoint+request.RepositoryRef+request.Operation, "\x00\r\n")
}
