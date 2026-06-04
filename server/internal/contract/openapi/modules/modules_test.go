package modulesopenapi

import "testing"

func TestModuleRuntimeGeneratedAliasesExposeRuntimeEnums(t *testing.T) {
	t.Parallel()

	locale := "zh-CN"
	params := GetModulesRuntimeParams{XGraftLocale: &locale}
	if params.XGraftLocale == nil || *params.XGraftLocale != "zh-CN" {
		t.Fatalf("unexpected generated header params: %#v", params)
	}

	cases := []GetModulesRuntime200JSONResponseBodyDataItemsRuntimeStatus{
		GetModulesRuntime200JSONResponseBodyDataItemsRuntimeStatusRegistered,
		GetModulesRuntime200JSONResponseBodyDataItemsRuntimeStatusDisabled,
		GetModulesRuntime200JSONResponseBodyDataItemsRuntimeStatusDegraded,
		GetModulesRuntime200JSONResponseBodyDataItemsRuntimeStatusUnknown,
	}
	for _, value := range cases {
		if !value.Valid() {
			t.Fatalf("expected generated module runtime status to be valid: %q", value)
		}
	}
}
