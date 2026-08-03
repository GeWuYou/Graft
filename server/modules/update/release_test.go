package update

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestGitHubReleaseProviderRequiresVerifiedRunnerIdentity(t *testing.T) {
	manifest := []byte(`{"release_tag":"v1.2.3","version":"1.2.3","channel":"stable","release_notes_url":"https://github.com/owner/repo/releases/tag/v1.2.3","upgrade_notes":"Read the release notes.","minimum_source_version":"1.0.0","artifacts":{"server":"server.tar.gz","web":"web.tar.gz","checksums":"checksums.txt","sha256":{"server.tar.gz":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","web.tar.gz":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},"images":{"server":{"image":"ghcr.io/gewuyou/graft-server","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","reference":"ghcr.io/gewuyou/graft-server@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"web":{"image":"ghcr.io/gewuyou/graft-web","digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","reference":"ghcr.io/gewuyou/graft-web@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},"runners":{"compose":{"image":"ghcr.io/gewuyou/graft-compose-runner","digest":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","reference":"ghcr.io/gewuyou/graft-compose-runner@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}}}`)
	checksum := fmt.Sprintf("%x  release-manifest.json\n", sha256.Sum256(manifest))
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/repos/owner/repo/releases":
			_, _ = fmt.Fprintf(writer, `[{"tag_name":"v1.2.3","published_at":"2026-07-22T00:00:00Z","assets":[{"name":"release-manifest.json","browser_download_url":%q},{"name":"release-manifest.json.sha256","browser_download_url":%q},{"name":"checksums.txt","browser_download_url":%q}]}]`, server.URL+"/manifest", server.URL+"/manifest.sha256", server.URL+"/checksums")
		case "/manifest":
			_, _ = writer.Write(manifest)
		case "/manifest.sha256":
			_, _ = writer.Write([]byte(checksum))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	provider := GitHubReleaseProvider{Repository: "owner/repo", ClientFactory: &outboundClientFactoryStub{client: &http.Client{Transport: rewriteTransport{base: http.DefaultTransport, target: server.URL}}}}
	releases, err := provider.List(t.Context())
	if err != nil {
		t.Fatalf("list releases: %v", err)
	}
	if len(releases) != 1 || releases[0].RunnerRef != "ghcr.io/gewuyou/graft-compose-runner@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" || releases[0].NotesURL == "" || releases[0].AssetSHA256["server.tar.gz"] == "" {
		t.Fatalf("unexpected releases: %#v", releases)
	}
}

func TestGitHubReleaseProviderUsesOutboundClientFactory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/repos/owner/repo/releases" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte(`[]`))
	}))
	defer server.Close()
	factory := &outboundClientFactoryStub{client: &http.Client{Transport: rewriteTransport{base: http.DefaultTransport, target: server.URL}}}
	provider := GitHubReleaseProvider{Repository: "owner/repo", ClientFactory: factory}
	if _, err := provider.List(context.Background()); err != nil {
		t.Fatalf("list releases through factory: %v", err)
	}
	if factory.calls != 1 || factory.timeout != releaseHTTPTimeout {
		t.Fatalf("expected one factory client with release timeout, got calls=%d timeout=%s", factory.calls, factory.timeout)
	}
}

type outboundClientFactoryStub struct {
	client  *http.Client
	calls   int
	timeout time.Duration
}

func (s *outboundClientFactoryStub) NewOutboundHTTPClient(_ context.Context, options ...moduleapi.OutboundHTTPClientOption) (*http.Client, error) {
	s.calls++
	configured := moduleapi.OutboundHTTPClientOptions{}
	for _, option := range options {
		if err := option(&configured); err != nil {
			return nil, err
		}
	}
	s.timeout = configured.Timeout
	return s.client, nil
}

func TestValidReleaseManifestRejectsMissingRunner(t *testing.T) {
	if validReleaseManifest("v1.2.3", releaseManifest{Version: "1.2.3", Channel: "stable"}) {
		t.Fatal("expected missing runner identity to reject manifest")
	}
}

func TestValidImageIdentityRejectsMutableTag(t *testing.T) {
	if validImageIdentity("ghcr.io/gewuyou/graft-compose-runner:latest", "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "ghcr.io/gewuyou/graft-compose-runner:latest@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc") {
		t.Fatal("expected mutable image tag rejection")
	}
}

type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (transport rewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.URL.Scheme = "http"
	clone.URL.Host = transport.target[len("http://"):]
	return transport.base.RoundTrip(clone)
}

func TestReleaseChecksumsURLBindsManifestAsset(t *testing.T) {
	manifest := releaseManifest{}
	manifest.Artifacts.Checksums = "published-checksums.txt"
	if got := releaseChecksumsURL(manifest, []githubAsset{{Name: "published-checksums.txt", BrowserDownloadURL: "https://example.test/checksums"}}); got != "https://example.test/checksums" {
		t.Fatalf("expected manifest-bound checksum asset URL, got %q", got)
	}
}

func TestValidDigestRequiresStrictHex(t *testing.T) {
	if validDigest("sha256:" + strings.Repeat("z", 64)) {
		t.Fatal("expected non-hex digest to reject")
	}
}
