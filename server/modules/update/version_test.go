package update

import "testing"

func TestSelectLatestHonorsStableAndBetaChannels(t *testing.T) {
	beta, _ := ParseVersion("0.9.1-beta.1")
	stable, _ := ParseVersion("0.9.1")
	releases := []Release{{Version: "0.9.1-beta.2", Channel: "beta"}, {Version: "0.10.0-beta.1", Channel: "beta"}, {Version: "0.10.0", Channel: "stable"}}
	selected, ok := SelectLatest(beta, releases)
	if !ok || selected.Version != "0.10.0" {
		t.Fatalf("beta expected later stable release, got %#v", selected)
	}
	selected, ok = SelectLatest(stable, releases)
	if !ok || selected.Version != "0.10.0" {
		t.Fatalf("stable must not select beta release, got %#v", selected)
	}
}

func TestDetectInstallationProfileRequiresDeclaredAndDetectedCompose(t *testing.T) {
	profile := DetectInstallationProfile(func(key string) string {
		if key == "GRAFT_UPDATE_COMPOSE_ROOT" {
			return "/opt/graft"
		}
		return "binary"
	}, func() (string, error) { return "/usr/local/bin/graft", nil })
	if profile.Capability != "manual_guidance" {
		t.Fatalf("expected conservative capability, got %s", profile.Capability)
	}
}
