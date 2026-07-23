package update

import (
	"strings"
	"testing"
)

func TestReleaseAssetURLsRecognizesPublishedChecksumName(t *testing.T) {
	_, checksumsURL := releaseAssetURLs("v1.2.3", []githubAsset{{Name: "graft-sha256sums-v1.2.3.txt", BrowserDownloadURL: "https://example.test/checksums"}})
	if checksumsURL != "https://example.test/checksums" {
		t.Fatalf("expected canonical checksum asset URL, got %q", checksumsURL)
	}
}

func TestValidReleaseManifestRequiresVersionChannelPair(t *testing.T) {
	manifest := releaseManifest{}
	manifest.Images.Server.Digest = validTestDigest('a')
	manifest.Images.Web.Digest = validTestDigest('b')

	manifest.Version = "1.2.3"
	manifest.Channel = "beta"
	if validReleaseManifest("v1.2.3", manifest) {
		t.Fatal("expected stable version in beta channel to reject manifest")
	}

	manifest.Version = "1.2.3-beta.1"
	manifest.Channel = "stable"
	if validReleaseManifest("v1.2.3-beta.1", manifest) {
		t.Fatal("expected beta version in stable channel to reject manifest")
	}

	manifest.Channel = "beta"
	if !validReleaseManifest("v1.2.3-beta.1", manifest) {
		t.Fatal("expected beta version in beta channel to accept manifest")
	}
}

func TestReleaseMatchesGitHubPrereleaseMetadata(t *testing.T) {
	manifest := releaseManifest{Version: "1.2.3-beta.1"}
	if releaseMatchesGitHubMetadata(githubRelease{Prerelease: false}, manifest) {
		t.Fatal("expected prerelease metadata mismatch to reject release")
	}
	if !releaseMatchesGitHubMetadata(githubRelease{Prerelease: true}, manifest) {
		t.Fatal("expected matching prerelease metadata to accept release")
	}
}

func TestValidDigestRequiresStrictHex(t *testing.T) {
	if validDigest("sha256:" + strings.Repeat("z", 64)) {
		t.Fatal("expected non-hex digest to reject")
	}
	if !validDigest(validTestDigest('a')) {
		t.Fatal("expected lowercase hexadecimal digest to accept")
	}
}

func validTestDigest(character byte) string {
	return "sha256:" + strings.Repeat(string(character), 64)
}
