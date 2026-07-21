package container

import (
	"testing"
	"time"

	"github.com/moby/moby/api/types/network"
)

func TestClassifyDockerNetworkMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		networkName string
		ingress     bool
		labels      map[string]string
		wantSource  DockerNetworkSource
		wantSystem  map[string]string
		wantUser    map[string]string
	}{
		{
			name:        "compose network",
			networkName: "arcane_default",
			labels: map[string]string{
				"com.docker.compose.project": "arcane",
				"com.docker.compose.network": "default",
				"com.docker.compose.version": "5.1.0",
				"environment":                "production",
			},
			wantSource: DockerNetworkSource{Kind: dockerNetworkSourceCompose, ComposeProject: "arcane", ComposeNetwork: "default", ComposeVersion: "5.1.0"},
			wantSystem: map[string]string{
				"com.docker.compose.project": "arcane",
				"com.docker.compose.network": "default",
				"com.docker.compose.version": "5.1.0",
			},
			wantUser: map[string]string{"environment": "production"},
		},
		{
			name:        "swarm network",
			networkName: "frontend",
			labels: map[string]string{
				"com.docker.stack.namespace": "portal",
				"com.docker.swarm.scope":     "swarm",
				"io.docker.managed":          "true",
				"team":                       "platform",
			},
			wantSource: DockerNetworkSource{Kind: dockerNetworkSourceSwarm, SwarmStack: "portal"},
			wantSystem: map[string]string{
				"com.docker.stack.namespace": "portal",
				"com.docker.swarm.scope":     "swarm",
				"io.docker.managed":          "true",
			},
			wantUser: map[string]string{"team": "platform"},
		},
		{
			name:        "ingress network",
			networkName: "ingress",
			ingress:     true,
			wantSource:  DockerNetworkSource{Kind: dockerNetworkSourceSwarm},
			wantSystem:  map[string]string{},
			wantUser:    map[string]string{},
		},
		{
			name:        "docker default network",
			networkName: "bridge",
			wantSource:  DockerNetworkSource{Kind: dockerNetworkSourceDocker},
			wantSystem:  map[string]string{},
			wantUser:    map[string]string{},
		},
		{
			name:        "custom network",
			networkName: "production",
			labels:      map[string]string{"traefik.enable": "true"},
			wantSource:  DockerNetworkSource{Kind: dockerNetworkSourceCustom},
			wantSystem:  map[string]string{},
			wantUser:    map[string]string{"traefik.enable": "true"},
		},
		{
			name:        "conflicting compose and swarm labels",
			networkName: "ambiguous",
			labels: map[string]string{
				"com.docker.compose.project": "app",
				"com.docker.stack.namespace": "stack",
			},
			wantSource: DockerNetworkSource{Kind: dockerNetworkSourceUnknown},
			wantSystem: map[string]string{
				"com.docker.compose.project": "app",
				"com.docker.stack.namespace": "stack",
			},
			wantUser: map[string]string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source, groups := classifyDockerNetworkMetadata(test.networkName, test.ingress, test.labels)
			if source != test.wantSource {
				t.Fatalf("unexpected source: got %#v, want %#v", source, test.wantSource)
			}
			assertStringMapEqual(t, groups.System, test.wantSystem, "system labels")
			assertStringMapEqual(t, groups.User, test.wantUser, "user labels")
		})
	}
}

func TestDockerNetworkBuildsMetadataProjection(t *testing.T) {
	t.Parallel()

	labels := map[string]string{"com.docker.compose.project": "graft", "owner": "platform"}
	projected := dockerNetwork(network.Network{Name: "graft_default", Labels: labels, Created: time.Now()}, 2)

	if projected.Source != (DockerNetworkSource{Kind: dockerNetworkSourceCompose, ComposeProject: "graft"}) {
		t.Fatalf("unexpected source projection: %#v", projected.Source)
	}
	assertStringMapEqual(t, projected.LabelGroups.System, map[string]string{"com.docker.compose.project": "graft"}, "projected system labels")
	assertStringMapEqual(t, projected.LabelGroups.User, map[string]string{"owner": "platform"}, "projected user labels")
	assertStringMapEqual(t, projected.Labels, labels, "raw labels")
}

func assertStringMapEqual(t *testing.T, got, want map[string]string, name string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected %s length: got %#v, want %#v", name, got, want)
	}
	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Fatalf("unexpected %s: got %#v, want %#v", name, got, want)
		}
	}
}
