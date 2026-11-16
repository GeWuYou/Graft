package main

import (
	"context"
	"testing"
	"time"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	composeapi "github.com/docker/compose/v5/pkg/api"
)

type captureRecreateService struct {
	composeapi.Compose
	project *composetypes.Project
	options composeapi.UpOptions
}

func (s *captureRecreateService) Up(_ context.Context, project *composetypes.Project, options composeapi.UpOptions) error {
	s.project, s.options = project, options
	return nil
}

func TestControllerRecreateWaitUsesTheIntendedMinuteDuration(t *testing.T) {
	service := &captureRecreateService{}
	runtime := &composeRuntime{service: service, project: &composetypes.Project{Name: "frozen-project",
		Services: composetypes.Services{"server": {Name: "server"}, "web": {Name: "web"}}}}
	if err := runtime.up(t.Context(), []string{"server", "web"}); err != nil {
		t.Fatal(err)
	}
	if !service.options.Start.Wait || service.options.Start.WaitTimeout != time.Minute {
		t.Fatalf("recreate wait used %v instead of its one-minute budget", service.options.Start.WaitTimeout)
	}
	if service.project.Name != "frozen-project" || len(service.project.Services) != 2 || service.options.Start.Project != service.project {
		t.Fatalf("recreate lost its frozen project: %#v", service.project)
	}
}
