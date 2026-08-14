package runtimetarget

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	httpheader "graft/server/internal/contract/httpheader"
	"graft/server/internal/moduleapi"
)

const ociRegistryVerificationTimeout = 10 * time.Second

var ociRegistryVerificationHTTPClient = &http.Client{
	Timeout:       ociRegistryVerificationTimeout,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
}

// VerifyOCIRegistryOnTarget probes only the OCI V2 root after Runtime Target resolves the explicit target.
// The credential is read solely from the adapter-created temporary config and never enters the returned evidence.
func (p dockerTargetProvider) VerifyOCIRegistryOnTarget(ctx context.Context, request moduleapi.OCIRegistryVerificationRequest) (moduleapi.OCIRegistryVerificationResult, error) {
	if !hasIsolatedCredentialConfig(ctx) || !validOCIRegistryVerificationRequest(request) {
		return moduleapi.OCIRegistryVerificationResult{}, errors.New("OCI registry verification input is invalid")
	}
	if _, err := p.connection(ctx, request.RuntimeTargetID); err != nil {
		return moduleapi.OCIRegistryVerificationResult{}, err
	}
	endpoint, err := ociRegistryV2Endpoint(request.Endpoint)
	if err != nil {
		return moduleapi.OCIRegistryVerificationResult{}, err
	}
	authorization, err := isolatedRegistryAuthorization(ctx, request.Endpoint)
	if err != nil {
		return moduleapi.OCIRegistryVerificationResult{}, err
	}
	return probeOCIRegistryV2(ctx, endpoint, authorization), nil
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

func isolatedRegistryAuthorization(ctx context.Context, endpoint string) (string, error) {
	configDir, ok := ctx.Value(dockerCredentialConfigContextKey{}).(string)
	if !ok || strings.TrimSpace(configDir) == "" {
		return "", errors.New("isolated Docker credential context is required")
	}
	contents, err := os.ReadFile(filepath.Join(configDir, "config.json")) // #nosec G304 -- configDir is created by the credential adapter for this call.
	if err != nil {
		return "", errors.New("read isolated registry credential config")
	}
	var config struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	if json.Unmarshal(contents, &config) != nil {
		return "", errors.New("read isolated registry credential config")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", errors.New("read isolated registry credential config")
	}
	for _, key := range []string{parsed.Host, strings.TrimRight(endpoint, "/")} {
		if encoded := strings.TrimSpace(config.Auths[key].Auth); encoded != "" {
			if _, err := base64.StdEncoding.DecodeString(encoded); err == nil {
				return "Basic " + encoded, nil
			}
		}
	}
	return "", errors.New("isolated registry credential is unavailable")
}
