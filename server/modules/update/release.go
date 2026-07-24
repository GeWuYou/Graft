package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultReleaseRepository = "GeWuYou/Graft"
	releaseHTTPTimeout       = 10 * time.Second
	releaseResponseMaxBytes  = 4 << 20
	manifestResponseMaxBytes = 1 << 20
	manifestChecksumMaxBytes = 1 << 10
)

// Release 描述经过 manifest 校验后可参与通道选择的不可变发行版本。
type Release struct {
	Version              string            `json:"version"`
	Channel              string            `json:"channel"`
	Notes                string            `json:"notes"`
	NotesURL             string            `json:"notes_url"`
	UpgradeNotes         string            `json:"upgrade_notes"`
	MinimumSourceVersion string            `json:"minimum_source_version"`
	PublishedAt          time.Time         `json:"published_at"`
	ManifestURL          string            `json:"manifest_url"`
	ServerDigest         string            `json:"server_digest"`
	WebDigest            string            `json:"web_digest"`
	ServerImage          string            `json:"server_image"`
	WebImage             string            `json:"web_image"`
	ServerRef            string            `json:"server_reference"`
	WebRef               string            `json:"web_reference"`
	RunnerImage          string            `json:"runner_image"`
	RunnerDigest         string            `json:"runner_digest"`
	RunnerRef            string            `json:"runner_reference"`
	ChecksumsURL         string            `json:"checksums_url"`
	AssetSHA256          map[string]string `json:"asset_sha256"`
}

// ReleaseProvider 从一个可信发布源读取发布目录。
type ReleaseProvider interface {
	List(context.Context) ([]Release, error)
}

// GitHubReleaseProvider 只接受 GitHub Release 中与 tag 一致、包含官方 OCI digest 与 runner 身份的 manifest。
type GitHubReleaseProvider struct {
	Repository string
	Client     *http.Client
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Body        string        `json:"body"`
	Prerelease  bool          `json:"prerelease"`
	Draft       bool          `json:"draft"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}
type releaseManifest struct {
	ReleaseTag           string                   `json:"release_tag"`
	Version              string                   `json:"version"`
	Channel              string                   `json:"channel"`
	ReleaseNotesURL      string                   `json:"release_notes_url"`
	MinimumSourceVersion string                   `json:"minimum_source_version"`
	UpgradeNotes         string                   `json:"upgrade_notes"`
	Artifacts            releaseManifestArtifacts `json:"artifacts"`
	Images               struct {
		Server struct {
			Image     string `json:"image"`
			Digest    string `json:"digest"`
			Reference string `json:"reference"`
		} `json:"server"`
		Web struct {
			Image     string `json:"image"`
			Digest    string `json:"digest"`
			Reference string `json:"reference"`
		} `json:"web"`
	} `json:"images"`
	Runners struct {
		Compose struct {
			Image     string `json:"image"`
			Digest    string `json:"digest"`
			Reference string `json:"reference"`
		} `json:"compose"`
	} `json:"runners"`
}

type releaseManifestArtifacts struct {
	Server    string            `json:"server"`
	Web       string            `json:"web"`
	Checksums string            `json:"checksums"`
	SHA256    map[string]string `json:"sha256"`
}

// List 获取并验证上游 Release；任一个 Release 的 manifest 无效时仅跳过该版本。
func (p GitHubReleaseProvider) List(ctx context.Context) ([]Release, error) {
	repository := strings.TrimSpace(p.Repository)
	if repository == "" {
		repository = defaultReleaseRepository
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: releaseHTTPTimeout}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repository+"/releases", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request GitHub releases: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request GitHub releases: unexpected status %d", response.StatusCode)
	}
	var payload []githubRelease
	if err := json.NewDecoder(io.LimitReader(response.Body, releaseResponseMaxBytes)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode GitHub releases: %w", err)
	}
	items := make([]Release, 0, len(payload))
	for _, item := range payload {
		if item.Draft {
			continue
		}
		release, ok := p.verifyRelease(ctx, client, item)
		if ok {
			items = append(items, release)
		}
	}
	return items, nil
}

func (p GitHubReleaseProvider) verifyRelease(ctx context.Context, client *http.Client, source githubRelease) (Release, bool) {
	manifestURL, manifestChecksumURL := releaseManifestURLs(source.Assets)
	if manifestURL == "" || manifestChecksumURL == "" {
		return Release{}, false
	}
	manifest, ok := downloadManifest(ctx, client, manifestURL, manifestChecksumURL)
	if !ok || !validReleaseManifest(source.TagName, manifest) || !releaseMatchesGitHubMetadata(source, manifest) {
		return Release{}, false
	}
	checksumsURL := releaseChecksumsURL(manifest, source.Assets)
	if checksumsURL == "" {
		return Release{}, false
	}
	return buildRelease(source, manifest, manifestURL, checksumsURL), true
}

func releaseManifestURLs(assets []githubAsset) (string, string) {
	var manifestURL, manifestChecksumURL string
	for _, asset := range assets {
		switch asset.Name {
		case "release-manifest.json":
			manifestURL = asset.BrowserDownloadURL
		case "release-manifest.json.sha256":
			manifestChecksumURL = asset.BrowserDownloadURL
		}
	}
	return manifestURL, manifestChecksumURL
}

func releaseChecksumsURL(manifest releaseManifest, assets []githubAsset) string {
	checksumsName := strings.TrimSpace(manifest.Artifacts.Checksums)
	if checksumsName == "" {
		return ""
	}
	for _, asset := range assets {
		if asset.Name == checksumsName {
			return asset.BrowserDownloadURL
		}
	}
	return ""
}

//nolint:cyclop // Manifest acceptance is intentionally one auditable sequence of independent authority checks.
func validReleaseManifest(tagName string, manifest releaseManifest) bool {
	version, err := ParseVersion(manifest.Version)
	if err != nil || version.String() != strings.TrimPrefix(strings.TrimSpace(tagName), "v") || strings.TrimSpace(manifest.ReleaseTag) != strings.TrimSpace(tagName) || !validAbsoluteURL(manifest.ReleaseNotesURL) || strings.TrimSpace(manifest.UpgradeNotes) == "" {
		return false
	}
	if minimum := strings.TrimSpace(manifest.MinimumSourceVersion); minimum != "" {
		if _, err := ParseVersion(minimum); err != nil {
			return false
		}
	}
	if !validAssetManifest(manifest.Artifacts) {
		return false
	}
	if !validImageIdentity(manifest.Images.Server.Image, manifest.Images.Server.Digest, manifest.Images.Server.Reference) ||
		!validImageIdentity(manifest.Images.Web.Image, manifest.Images.Web.Digest, manifest.Images.Web.Reference) ||
		!validImageIdentity(manifest.Runners.Compose.Image, manifest.Runners.Compose.Digest, manifest.Runners.Compose.Reference) {
		return false
	}
	if manifest.Runners.Compose.Image != composeRunnerImage(manifest.Images.Server.Image) {
		return false
	}
	if strings.TrimSpace(manifest.Artifacts.Checksums) == "" {
		return false
	}
	return releaseChannelMatchesVersion(strings.ToLower(strings.TrimSpace(manifest.Channel)), version)
}

func releaseMatchesGitHubMetadata(source githubRelease, manifest releaseManifest) bool {
	version, err := ParseVersion(manifest.Version)
	return err == nil && source.Prerelease == version.IsPrerelease()
}

func releaseChannelMatchesVersion(channel string, version Version) bool {
	switch channel {
	case "stable":
		return !version.IsPrerelease()
	case "beta":
		return validBetaPrerelease(version.Prerelease)
	default:
		return false
	}
}

func validBetaPrerelease(value string) bool {
	sequence, ok := strings.CutPrefix(value, "beta.")
	if !ok || sequence == "" || (len(sequence) > 1 && sequence[0] == '0') {
		return false
	}
	_, err := strconv.ParseUint(sequence, 10, 64)
	return err == nil
}

func buildRelease(source githubRelease, manifest releaseManifest, manifestURL, checksumsURL string) Release {
	version, _ := ParseVersion(manifest.Version)
	return Release{Version: version.String(), Channel: strings.ToLower(strings.TrimSpace(manifest.Channel)), Notes: source.Body, NotesURL: manifest.ReleaseNotesURL, UpgradeNotes: manifest.UpgradeNotes, MinimumSourceVersion: manifest.MinimumSourceVersion, PublishedAt: source.PublishedAt.UTC(), ManifestURL: manifestURL, ServerDigest: manifest.Images.Server.Digest, WebDigest: manifest.Images.Web.Digest, ServerImage: manifest.Images.Server.Image, WebImage: manifest.Images.Web.Image, ServerRef: manifest.Images.Server.Reference, WebRef: manifest.Images.Web.Reference, RunnerImage: manifest.Runners.Compose.Image, RunnerDigest: manifest.Runners.Compose.Digest, RunnerRef: manifest.Runners.Compose.Reference, ChecksumsURL: checksumsURL, AssetSHA256: manifest.Artifacts.SHA256}
}

func validAbsoluteURL(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "https://github.com/") && !strings.ContainsAny(value, " \t\r\n")
}

func validAssetManifest(value releaseManifestArtifacts) bool {
	if strings.TrimSpace(value.Server) == "" || strings.TrimSpace(value.Web) == "" || strings.TrimSpace(value.Checksums) == "" || len(value.SHA256) < 2 {
		return false
	}
	for _, name := range []string{value.Server, value.Web} {
		digest, ok := value.SHA256[name]
		if !ok || len(strings.TrimSpace(digest)) != sha256.Size*2 {
			return false
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return false
		}
	}
	return true
}

func downloadManifest(ctx context.Context, client *http.Client, location, checksumLocation string) (releaseManifest, bool) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return releaseManifest{}, false
	}
	response, err := client.Do(request)
	if err != nil {
		return releaseManifest{}, false
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return releaseManifest{}, false
	}
	contents, err := io.ReadAll(io.LimitReader(response.Body, manifestResponseMaxBytes))
	if err != nil {
		return releaseManifest{}, false
	}
	checksum, ok := downloadManifestChecksum(ctx, client, checksumLocation)
	if !ok || fmt.Sprintf("%x", sha256.Sum256(contents)) != checksum {
		return releaseManifest{}, false
	}
	var manifest releaseManifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		return releaseManifest{}, false
	}
	return manifest, true
}

func downloadManifestChecksum(ctx context.Context, client *http.Client, location string) (string, bool) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return "", false
	}
	response, err := client.Do(request)
	if err != nil {
		return "", false
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", false
	}
	contents, err := io.ReadAll(io.LimitReader(response.Body, manifestChecksumMaxBytes))
	if err != nil {
		return "", false
	}
	return parseManifestChecksum(contents)
}

func parseManifestChecksum(contents []byte) (string, bool) {
	parts := strings.Fields(string(contents))
	if len(parts) < 1 || len(parts[0]) != sha256.Size*2 {
		return "", false
	}
	if _, err := hex.DecodeString(parts[0]); err != nil {
		return "", false
	}
	return strings.ToLower(parts[0]), true
}

func validDigest(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func validImageIdentity(image, digest, reference string) bool {
	image = strings.TrimSpace(image)
	path := strings.TrimPrefix(image, "ghcr.io/")
	segments := strings.Split(path, "/")
	if !strings.HasPrefix(image, "ghcr.io/") || len(segments) != 2 || segments[0] == "" || segments[1] == "" || strings.ContainsAny(image, "@ \t\n") || strings.Contains(path, ":") || !validDigest(digest) {
		return false
	}
	return reference == image+"@"+strings.TrimSpace(digest)
}

func composeRunnerImage(serverImage string) string {
	return strings.TrimSuffix(serverImage, "/graft-server") + "/graft-updater"
}

// SelectLatest 返回与当前通道兼容且严格较新的发布。稳定用户只接收 stable；beta 用户同时接收 beta 与后续 stable。
func SelectLatest(current Version, releases []Release) (Release, bool) {
	var selected Release
	found := false
	for _, release := range releases {
		candidate, err := ParseVersion(release.Version)
		if err != nil || candidate.Compare(current) <= 0 {
			continue
		}
		if !eligibleChannel(current, release.Channel) {
			continue
		}
		if !found {
			selected, found = release, true
			continue
		}
		selectedVersion, _ := ParseVersion(selected.Version)
		if candidate.Compare(selectedVersion) > 0 {
			selected = release
		}
	}
	return selected, found
}

func eligibleChannel(current Version, channel string) bool {
	return channel == "stable" || current.IsPrerelease() && channel == "beta"
}
