package container

import "testing"

func TestValidateDockerNetworkCreateCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command DockerNetworkCreateCommand
		valid   bool
	}{
		{name: "bridge without ipam", command: DockerNetworkCreateCommand{Name: "private", Driver: "bridge"}, valid: true},
		{name: "ipv4 subnet and gateway", command: DockerNetworkCreateCommand{Name: "private", Driver: "bridge", IPAM: &DockerNetworkIPAMConfig{Subnet: "172.30.0.0/16", Gateway: "172.30.0.1"}}, valid: true},
		{name: "empty name", command: DockerNetworkCreateCommand{Driver: "bridge"}},
		{name: "unsupported driver", command: DockerNetworkCreateCommand{Name: "private", Driver: "custom"}},
		{name: "ipv6 subnet", command: DockerNetworkCreateCommand{Name: "private", Driver: "bridge", IPAM: &DockerNetworkIPAMConfig{Subnet: "2001:db8::/64"}}},
		{name: "gateway outside subnet", command: DockerNetworkCreateCommand{Name: "private", Driver: "bridge", IPAM: &DockerNetworkIPAMConfig{Subnet: "172.30.0.0/16", Gateway: "172.31.0.1"}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateDockerNetworkCreateCommand(test.command)
			if test.valid && err != nil {
				t.Fatalf("expected valid command, got %v", err)
			}
			if !test.valid && err != errInvalidDockerNetworkRequest {
				t.Fatalf("expected invalid request, got %v", err)
			}
		})
	}
}

func TestDockerNetworkRemovalGuards(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"bridge", "host", "none"} {
		if !isDockerDefaultNetwork(name) {
			t.Fatalf("expected %q to be protected", name)
		}
	}
	if isDockerDefaultNetwork("application-network") {
		t.Fatal("expected user-defined network to remain removable when unused")
	}
}
