package build

import "testing"

func TestBuiltinBuildIntentRegistryResolvesVersionedIntent(t *testing.T) {
	registry := newBuiltinBuildIntentRegistry()
	template, driver, err := registry.ResolveBuildIntent("oci-dockerfile/default", "docker-engine")
	if err != nil {
		t.Fatal(err)
	}
	if template.Ref != "oci-dockerfile/default@v1" || template.Version != "v1" {
		t.Fatalf("template = %#v", template)
	}
	if driver.Ref != "docker-engine@v1" || driver.Version != "v1" {
		t.Fatalf("driver = %#v", driver)
	}
}

func TestBuiltinBuildIntentRegistryRejectsUnknownOrIncompatibleIntent(t *testing.T) {
	registry := newBuiltinBuildIntentRegistry()
	for _, test := range []struct {
		name   string
		tmpl   string
		driver string
	}{
		{name: "unknown template", tmpl: "oci-bake/default@v1", driver: "docker-engine@v1"},
		{name: "unknown driver", tmpl: "oci-dockerfile/default@v1", driver: "kaniko@v1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := registry.ResolveBuildIntent(test.tmpl, test.driver); err == nil {
				t.Fatal("expected intent to be rejected")
			}
		})
	}
}
