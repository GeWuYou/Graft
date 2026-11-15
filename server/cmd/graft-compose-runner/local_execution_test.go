package main

import (
	"testing"

	composetypes "github.com/compose-spec/compose-go/v2/types"

	"graft/server/modules/update"
)

func TestControllerRejectsExecutionModeDrift(t *testing.T) {
	for _, scenario := range []struct {
		name, frozen, configured string
		allowed                  bool
	}{
		{name: "legacy missing mode", allowed: true},
		{name: "explicit Agent", frozen: "agent", configured: "agent", allowed: true},
		{name: "local", frozen: "local", configured: "local", allowed: true},
		{name: "local changed to Agent", frozen: "local", configured: "agent"},
		{name: "legacy changed to local", configured: "local"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			project := &composetypes.Project{Environment: composetypes.Mapping{"GRAFT_EXECUTION_MODE": scenario.configured}}
			err := validateProjectExecutionMode(project, update.RunnerInput{ExecutionMode: scenario.frozen})
			if (err == nil) != scenario.allowed {
				t.Fatalf("mode drift result = %v, allowed %v", err, scenario.allowed)
			}
		})
	}
}
