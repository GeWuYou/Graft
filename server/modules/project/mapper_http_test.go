package project

import (
	"errors"
	"slices"
	"testing"
	"time"

	"graft/server/internal/contract/openapi/generated"
	projectcompose "graft/server/modules/project/compose"
)

func TestManagedCreateRequestMappersRejectUnsupportedLifecycleStrategy(t *testing.T) {
	t.Parallel()

	invalidLifecycle := &generated.ApplicationLifecycleConfigurationRequest{StrategyKind: generated.ApplicationLifecycleStrategyKind("unsupported")}
	if _, err := toManagedCreateRequest(generated.PostApplicationCreateValidateJSONRequestBody{LifecycleConfiguration: invalidLifecycle}); !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("expected invalid lifecycle strategy error from validate mapper, got %v", err)
	}
	if _, err := toManagedCreateExecuteRequest(generated.PostApplicationCreateJSONRequestBody{LifecycleConfiguration: invalidLifecycle}); !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("expected invalid lifecycle strategy error from create mapper, got %v", err)
	}
}

func TestManagedCreateRequestMappersUseCanonicalWorkspaceEntries(t *testing.T) {
	t.Parallel()
	compose := "services: {}\n"
	readme := "workspace notes\n"
	entries := []generated.ApplicationWorkspaceEntry{
		{Path: "config", NodeType: generated.ApplicationWorkspaceEntryNodeTypeDirectory},
		{Path: "config/readme", NodeType: generated.ApplicationWorkspaceEntryNodeTypeFile, Content: &readme},
		{Path: "compose/compose.yaml", NodeType: generated.ApplicationWorkspaceEntryNodeTypeFile, Content: &compose},
	}
	request, err := toManagedCreateExecuteRequest(generated.PostApplicationCreateJSONRequestBody{DisplayName: "Demo", RuntimeTargetId: 1, ComposeFilePath: "compose/compose.yaml", WorkspaceEntries: entries})
	if err != nil {
		t.Fatalf("map canonical workspace entries: %v", err)
	}
	if request.ComposeFilePath != "compose/compose.yaml" || request.ComposeFileName != "compose.yaml" || request.ComposeFileContent != compose {
		t.Fatalf("unexpected primary compose mapping: %#v", request)
	}
	if len(request.WorkspaceEntries) != len(entries) || request.WorkspaceEntries[0].NodeType != "directory" || request.WorkspaceEntries[1].Content == nil || *request.WorkspaceEntries[1].Content != readme {
		t.Fatalf("unexpected workspace entry mapping: %#v", request.WorkspaceEntries)
	}
}

func TestManagedCreateRequestMapperRejectsMissingPrimaryComposeEntry(t *testing.T) {
	t.Parallel()
	content := "text"
	_, err := toManagedCreateRequest(generated.PostApplicationCreateValidateJSONRequestBody{DisplayName: "Demo", RuntimeTargetId: 1, ComposeFilePath: "compose.yaml", WorkspaceEntries: []generated.ApplicationWorkspaceEntry{{Path: "README", NodeType: generated.ApplicationWorkspaceEntryNodeTypeFile, Content: &content}}})
	if !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("expected missing compose entry error, got %v", err)
	}
}

func TestToRuntimeImportInspectResponseMapsStructuredResources(t *testing.T) {
	t.Parallel()

	overlay := "overlay"
	internal := true
	local := "local"
	result := RuntimeImportInspectResult{
		InspectionID: "inspect-1", ExpiresAt: time.Date(2026, time.July, 11, 8, 5, 0, 0, time.UTC), CandidateKey: "runtime:demo",
		ResolvedWorkspacePath: "/srv/demo", ComposeProjectName: "demo", ComposeProjectNameSource: "computed", DisplayNameSuggested: "Demo",
		ComposeFiles:     []FileView{{AbsolutePath: "/srv/demo/compose.yaml", DisplayPath: "compose.yaml", Kind: "compose", Role: "primary"}},
		EnvFiles:         []FileView{{AbsolutePath: "/srv/demo/.env", DisplayPath: ".env", Kind: "env", Role: "primary"}},
		ServiceNames:     []string{"web", "worker"},
		ServiceOptions:   []projectcompose.ServiceProjection{{ServiceName: "web", DependsOn: []string{"db"}}, {ServiceName: "worker"}},
		NetworkResources: []RuntimeImportNetworkResource{{Name: "backend", Driver: &overlay, Internal: &internal, Containers: []string{"demo-web-1", "demo-worker-1"}, ContainerCount: 2, Services: []string{"web", "worker"}, ServiceCount: 2}},
		VolumeResources:  []RuntimeImportVolumeResource{{Name: "data", Driver: &local, Anonymous: false, MountTarget: "/data", MountedBy: []string{"web", "worker"}, Containers: []string{"demo-web-1", "demo-worker-1"}, ContainerCount: 2}},
		RuntimeMembers:   []RuntimeImportMember{{ContainerID: "c1", ContainerName: "demo-web-1", ServiceName: "web", State: "running"}},
		ConfigHash:       "abc123", Warnings: []string{"workspace_path_derived_from_config_files"}, Conflicts: []string{}, ValidationStatus: "ready",
	}

	response := toRuntimeImportInspectResponse(result)
	if !response.ExpiresAt.Equal(result.ExpiresAt) || response.LifecycleConfiguration.Profiles == nil {
		t.Fatalf("unexpected inspection response: %#v", response)
	}
	assertRuntimeInspectNetworkMapping(t, response)
	assertRuntimeInspectVolumeMapping(t, response)
	assertRuntimeInspectServiceOptions(t, response)
}

func assertRuntimeInspectNetworkMapping(t *testing.T, response generated.ApplicationImportRuntimeInspectResponse) {
	t.Helper()
	if len(response.Networks) != 1 || response.Networks[0].Driver == nil || *response.Networks[0].Driver != "overlay" || response.Networks[0].Internal == nil || !*response.Networks[0].Internal || response.Networks[0].ContainerCount != 2 || response.Networks[0].ServiceCount != 2 {
		t.Fatalf("unexpected network mapping: %#v", response.Networks)
	}
}

func assertRuntimeInspectVolumeMapping(t *testing.T, response generated.ApplicationImportRuntimeInspectResponse) {
	t.Helper()
	if len(response.Volumes) != 1 || response.Volumes[0].Driver == nil || *response.Volumes[0].Driver != "local" || response.Volumes[0].MountTarget != "/data" || response.Volumes[0].Anonymous {
		t.Fatalf("unexpected volume mapping: %#v", response.Volumes)
	}
}

func assertRuntimeInspectServiceOptions(t *testing.T, response generated.ApplicationImportRuntimeInspectResponse) {
	t.Helper()
	if len(response.ServiceOptions) != 2 || !slices.Equal(response.ServiceOptions[0].DependsOn, []string{"db"}) {
		t.Fatalf("unexpected service options: %#v", response.ServiceOptions)
	}
}
