package project

import "testing"

func TestToRuntimeImportInspectResponseMapsStructuredResources(t *testing.T) {
	t.Parallel()

	overlay := "overlay"
	internal := true
	local := "local"
	result := RuntimeImportInspectResult{
		InspectionID:               "inspect-1",
		CandidateKey:               "runtime:demo",
		ResolvedWorkingDirectory:   "/srv/demo",
		CanonicalProjectName:       "demo",
		CanonicalProjectNameSource: "computed",
		DisplayNameSuggested:       "Demo",
		ComposeFiles: []FileView{
			{AbsolutePath: "/srv/demo/compose.yaml", DisplayPath: "compose.yaml", Kind: "compose", Role: "primary"},
		},
		EnvFiles: []FileView{
			{AbsolutePath: "/srv/demo/.env", DisplayPath: ".env", Kind: "env", Role: "primary"},
		},
		ServiceNames: []string{"web", "worker"},
		NetworkResources: []RuntimeImportNetworkResource{
			{
				Name:           "backend",
				Driver:         &overlay,
				Internal:       &internal,
				Containers:     []string{"demo-web-1", "demo-worker-1"},
				ContainerCount: 2,
				Services:       []string{"web", "worker"},
				ServiceCount:   2,
			},
		},
		VolumeResources: []RuntimeImportVolumeResource{
			{
				Name:           "data",
				Driver:         &local,
				Anonymous:      false,
				MountTarget:    "/data",
				MountedBy:      []string{"web", "worker"},
				Containers:     []string{"demo-web-1", "demo-worker-1"},
				ContainerCount: 2,
			},
		},
		RuntimeMembers: []RuntimeImportMember{
			{ContainerID: "c1", ContainerName: "demo-web-1", ServiceName: "web", State: "running"},
		},
		ConfigHash:       "abc123",
		Warnings:         []string{"working_directory_derived_from_config_files"},
		Conflicts:        []string{},
		ValidationStatus: "ready",
	}

	response := toRuntimeImportInspectResponse(result)
	if response.LifecycleConfiguration.Profiles == nil {
		t.Fatal("expected lifecycle profiles to serialize as an empty array")
	}
	if len(response.Networks) != 1 {
		t.Fatalf("expected one mapped network resource, got %#v", response.Networks)
	}
	if response.Networks[0].Driver == nil || *response.Networks[0].Driver != "overlay" {
		t.Fatalf("expected mapped network driver, got %#v", response.Networks[0].Driver)
	}
	if response.Networks[0].Internal == nil || !*response.Networks[0].Internal {
		t.Fatalf("expected mapped network internal, got %#v", response.Networks[0].Internal)
	}
	if response.Networks[0].ContainerCount != 2 || response.Networks[0].ServiceCount != 2 {
		t.Fatalf("unexpected mapped network counts %#v", response.Networks[0])
	}
	if len(response.Volumes) != 1 {
		t.Fatalf("expected one mapped volume resource, got %#v", response.Volumes)
	}
	if response.Volumes[0].Driver == nil || *response.Volumes[0].Driver != "local" {
		t.Fatalf("expected mapped volume driver, got %#v", response.Volumes[0].Driver)
	}
	if response.Volumes[0].MountTarget != "/data" || response.Volumes[0].Anonymous {
		t.Fatalf("unexpected mapped volume resource %#v", response.Volumes[0])
	}
}
