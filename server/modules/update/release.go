package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultReleaseRepository = "GeWuYou/Graft"
	releaseHTTPTimeout       = 10 * time.Second
	releaseResponseMaxBytes  = 4 << 20
	manifestResponseMaxBytes = 1 << 20
)

// Release 描述经过 manifest 校验后可参与通道选择的不可变发行版本。
type Release struct {
	Version      string    `json:"version"`
	Channel      string    `json:"channel"`
	Notes        string    `json:"notes"`
	PublishedAt  time.Time `json:"published_at"`
	ManifestURL  string    `json:"manifest_url"`
	ServerDigest string    `json:"server_digest"`
	WebDigest    string    `json:"web_digest"`
	ChecksumsURL string    `json:"checksums_url"`
}

// ReleaseProvider 从一个可信发布源读取发布目录。
type ReleaseProvider interface {
	List(context.Context) ([]Release, error)
}

// GitHubReleaseProvider 只接受 GitHub Release 中与 tag 一致、包含两个 OCI digest 的 manifest。
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
	Version string `json:"version"`
	Channel string `json:"channel"`
	Images  struct {
		Server struct {
			Digest string `json:"digest"`
		} `json:"server"`
		Web struct {
			Digest string `json:"digest"`
		} `json:"web"`
	} `json:"images"`
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
	manifestURL, checksumsURL := releaseAssetURLs(source.Assets)
	if manifestURL == "" {
		return Release{}, false
	}
	manifest, ok := downloadManifest(ctx, client, manifestURL)
	if !ok || !validReleaseManifest(source.TagName, manifest) {
		return Release{}, false
	}
	return buildRelease(source, manifest, manifestURL, checksumsURL), true
}

func releaseAssetURLs(assets []githubAsset) (string, string) {
	var manifestURL, checksumsURL string
	for _, asset := range assets {
		switch asset.Name {
		case "release-manifest.json":
			manifestURL = asset.BrowserDownloadURL
		case "checksums.txt", "checksums.sha256":
			checksumsURL = asset.BrowserDownloadURL
		}
	}
	return manifestURL, checksumsURL
}

func validReleaseManifest(tagName string, manifest releaseManifest) bool {
	version, err := ParseVersion(manifest.Version)
	if err != nil || version.String() != strings.TrimPrefix(strings.TrimSpace(tagName), "v") {
		return false
	}
	if !validDigest(manifest.Images.Server.Digest) || !validDigest(manifest.Images.Web.Digest) {
		return false
	}
	channel := strings.ToLower(strings.TrimSpace(manifest.Channel))
	return channel == "stable" || channel == "beta"
}

func buildRelease(source githubRelease, manifest releaseManifest, manifestURL, checksumsURL string) Release {
	version, _ := ParseVersion(manifest.Version)
	return Release{Version: version.String(), Channel: strings.ToLower(strings.TrimSpace(manifest.Channel)), Notes: source.Body, PublishedAt: source.PublishedAt.UTC(), ManifestURL: manifestURL, ServerDigest: manifest.Images.Server.Digest, WebDigest: manifest.Images.Web.Digest, ChecksumsURL: checksumsURL}
}

func downloadManifest(ctx context.Context, client *http.Client, location string) (releaseManifest, bool) {
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
	var manifest releaseManifest
	if err := json.NewDecoder(io.LimitReader(response.Body, manifestResponseMaxBytes)).Decode(&manifest); err != nil {
		return releaseManifest{}, false
	}
	return manifest, true
}

func validDigest(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "sha256:") && len(value) == len("sha256:")+64
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
