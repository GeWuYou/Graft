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

func TestSelectLatestForChannelUsesTheConfiguredChannel(t *testing.T) {
	testCases := []struct {
		name     string
		current  string
		channel  string
		releases []Release
		want     string
		wantOK   bool
	}{
		{name: "beta selects latest beta", current: "0.9.1", channel: "beta", releases: []Release{{Version: "0.9.2-beta.1", Channel: "beta"}, {Version: "0.9.2", Channel: "stable"}, {Version: "0.10.0-beta.1", Channel: "beta"}}, want: "0.10.0-beta.1", wantOK: true},
		{name: "stable ignores beta", current: "0.9.1", channel: "stable", releases: []Release{{Version: "0.9.2-beta.1", Channel: "beta"}, {Version: "0.9.2", Channel: "stable"}, {Version: "0.10.0-beta.1", Channel: "beta"}}, want: "0.9.2", wantOK: true},
		{name: "invalid channel blocks selection", current: "0.9.1", channel: "fixed", releases: []Release{{Version: "1.0.0", Channel: "stable"}}},
		{name: "same and older releases are ignored", current: "1.0.0", channel: "stable", releases: []Release{{Version: "1.0.0", Channel: "stable"}, {Version: "0.9.9", Channel: "stable"}}},
		{name: "invalid and mixed channels are filtered", current: "1.0.0", channel: "stable", releases: []Release{{Version: "invalid", Channel: "stable"}, {Version: "1.1.0-beta.1", Channel: "beta"}, {Version: "1.0.1", Channel: "stable"}}, want: "1.0.1", wantOK: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			current, err := ParseVersion(testCase.current)
			if err != nil {
				t.Fatalf("parse current version: %v", err)
			}
			selected, ok := SelectLatestForChannel(current, testCase.channel, testCase.releases)
			if ok != testCase.wantOK || selected.Version != testCase.want {
				t.Fatalf("selected = %#v, ok = %t; want version %q, ok = %t", selected, ok, testCase.want, testCase.wantOK)
			}
		})
	}
}
