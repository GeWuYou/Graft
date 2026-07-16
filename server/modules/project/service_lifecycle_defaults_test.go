package project

import "testing"

func TestDefaultLifecycleStandardConfig(t *testing.T) {
	config := defaultLifecycleStandardConfig()

	if !config.DownBeforeRedeploy || !config.RemoveOrphans {
		t.Fatalf("expected clean redeploy and orphan removal enabled by default: %#v", config)
	}
	if config.WaitTimeoutSeconds != defaultLifecycleWaitTimeoutSeconds {
		t.Fatalf("expected default wait timeout %d, got %d", defaultLifecycleWaitTimeoutSeconds, config.WaitTimeoutSeconds)
	}
	if len(config.Profiles) != 0 || len(config.AdditionalArgs) != 0 {
		t.Fatalf("expected empty profile and additional argument defaults: %#v", config)
	}
}
