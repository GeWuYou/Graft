package project

import "testing"

func TestDecodeAllowedImportRootsAcceptsJSONArrayPayload(t *testing.T) {
	t.Parallel()

	roots, err := decodeAllowedImportRoots(`[{"id":"team","label":"Team","path":"/srv/team"}]`)
	if err != nil {
		t.Fatalf("decode array payload: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if roots[0].ID != "team" || roots[0].Path != "/srv/team" {
		t.Fatalf("unexpected root: %#v", roots[0])
	}
}

func TestDecodeAllowedImportRootsAcceptsJSONStringPayload(t *testing.T) {
	t.Parallel()

	roots, err := decodeAllowedImportRoots(`"[{\"id\":\"team\",\"label\":\"Team\",\"path\":\"/srv/team\"}]"`)
	if err != nil {
		t.Fatalf("decode nested string payload: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if roots[0].Label != "Team" || roots[0].Path != "/srv/team" {
		t.Fatalf("unexpected root: %#v", roots[0])
	}
}

func TestDecodeAllowedImportRootsReturnsErrorForInvalidPayload(t *testing.T) {
	t.Parallel()

	if _, err := decodeAllowedImportRoots(`{"id":"broken"}`); err == nil {
		t.Fatalf("expected invalid payload error")
	}
}
