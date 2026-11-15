package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	composeapi "github.com/docker/compose/v5/pkg/api"
)

func TestComposeDefinitionPreparesCanonicalOwnershipLabels(t *testing.T) {
	for _, withEnv := range []bool{false, true} {
		t.Run(map[bool]string{false: "without-env", true: "explicit-env"}[withEnv], func(t *testing.T) {
			material := composeLabelTestMaterial(t, withEnv)
			project, err := loadComposeDefinition(context.Background(), material)
			if err != nil {
				t.Fatal(err)
			}
			if len(project.Services) != 2 {
				t.Fatalf("active services=%#v", project.ServiceNames())
			}
			if _, enabled := project.Services["unmanaged"]; enabled {
				t.Fatal("environment overrode explicit profile boundary")
			}
			for name, service := range project.Services {
				assertComposeOwnershipLabels(t, project, name, service.CustomLabels, material.EnvFiles)
			}
			if project.Services["app"].Labels["io.example.user"] != "preserved" {
				t.Fatal("application labels were lost")
			}
		})
	}
}

func TestComposeDispatcherPreservesLoadedOwnership(t *testing.T) {
	material := composeLabelTestMaterial(t, true)
	project, err := loadComposeDefinition(context.Background(), material)
	if err != nil {
		t.Fatal(err)
	}
	service := &recordingComposeService{}
	runComposeOperation(t, service, project, nil, "up")
	assertDispatchedComposeOwnership(t, project, service.upProject, material.EnvFiles)
	runComposeOperation(t, service, project, nil, "stop")
	assertDispatchedComposeOwnership(t, project, service.stopOptions.Project, material.EnvFiles)
	runComposeOperation(t, service, project, nil, "restart")
	assertDispatchedComposeOwnership(t, project, service.restartOptions.Project, material.EnvFiles)
	runComposeOperation(t, service, project, []string{"app"}, "pull")
	assertComposeOwnershipLabels(t, service.pullProject, "app", service.pullProject.Services["app"].CustomLabels, material.EnvFiles)
	if len(service.pullProject.Services) != 1 {
		t.Fatalf("pull scope=%v", service.pullProject.ServiceNames())
	}
}

func assertDispatchedComposeOwnership(t *testing.T, expected, actual *composetypes.Project, envFiles []string) {
	t.Helper()
	if actual != expected {
		t.Fatal("dispatcher replaced the prepared project")
	}
	for name, service := range actual.Services {
		assertComposeOwnershipLabels(t, actual, name, service.CustomLabels, envFiles)
	}
}

func assertComposeOwnershipLabels(t *testing.T, project *composetypes.Project, name string, labels composetypes.Labels, envFiles []string) {
	t.Helper()
	expected := composetypes.Labels{
		composeapi.ProjectLabel:     project.Name,
		composeapi.ServiceLabel:     name,
		composeapi.VersionLabel:     composeapi.ComposeVersion,
		composeapi.WorkingDirLabel:  project.WorkingDir,
		composeapi.ConfigFilesLabel: strings.Join(project.ComposeFiles, ","),
		composeapi.OneoffLabel:      "False",
	}
	if len(envFiles) > 0 {
		expected[composeapi.EnvironmentFileLabel] = strings.Join(envFiles, ",")
	}
	if len(labels) != len(expected) {
		t.Fatalf("service %s labels=%#v want=%#v", name, labels, expected)
	}
	for key, value := range expected {
		if labels[key] != value {
			t.Errorf("service %s label %s=%q want=%q", name, key, labels[key], value)
		}
	}
}

func composeLabelTestMaterial(t *testing.T, withEnv bool) applicationComposeMaterial {
	t.Helper()
	workspace := t.TempDir()
	composeFile := filepath.Join(workspace, "compose.yaml")
	definition := `services:
  app:
    image: busybox:latest
    labels:
      io.example.user: preserved
      com.docker.compose.project: forged-project
      com.docker.compose.service: forged-service
      com.docker.compose.oneoff: "True"
  maintenance:
    image: busybox:latest
    profiles: [maintenance]
  unmanaged:
    image: busybox:latest
    profiles: [unmanaged]
`
	if err := os.WriteFile(composeFile, []byte(definition), 0o600); err != nil {
		t.Fatal(err)
	}
	material := applicationComposeMaterial{WorkspacePath: workspace, ProjectName: "authoritative", ComposeFiles: []string{composeFile}, Profiles: []string{"maintenance"}}
	if withEnv {
		envFile := filepath.Join(workspace, "fixture.env")
		if err := os.WriteFile(envFile, []byte("COMPOSE_PROJECT_NAME=forged-project\nCOMPOSE_PROFILES=unmanaged\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		material.EnvFiles = []string{envFile}
	}
	return material
}
