package audit

import (
	"reflect"
	"testing"
)

func TestDescriptorDependenciesMatchRuntimePlugin(t *testing.T) {
	t.Parallel()

	repo := stubAuditRepository{}
	instance, err := NewPlugin(repo)
	if err != nil {
		t.Fatalf("NewPlugin() error = %v", err)
	}

	descriptor := NewDescriptor()
	if !reflect.DeepEqual(descriptor.DependsOn(), instance.DependsOn()) {
		t.Fatalf("descriptor dependencies = %v, runtime dependencies = %v", descriptor.DependsOn(), instance.DependsOn())
	}
}
