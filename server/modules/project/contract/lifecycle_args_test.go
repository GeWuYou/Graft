package contract

import "testing"

func TestNormalizeLifecycleAdditionalArgs(t *testing.T) {
	values, valid := NormalizeLifecycleAdditionalArgs([]string{" --progress ", "plain"})
	if !valid || len(values) != 2 || values[0] != "--progress" || values[1] != "plain" {
		t.Fatalf("unexpected normalized lifecycle args: %#v, valid=%t", values, valid)
	}
	if _, valid := NormalizeLifecycleAdditionalArgs([]string{"line\nbreak"}); valid {
		t.Fatal("expected lifecycle argument with a newline to be rejected")
	}
}
