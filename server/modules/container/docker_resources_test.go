package container

import "testing"

func TestListDockerVolumesFiltersSortsAndPages(t *testing.T) {
	t.Parallel()

	used := int64(2)
	items := []DockerVolume{
		{Name: "zeta", Driver: "local", Scope: "local", ReferenceCount: &used},
		{Name: "alpha", Driver: "local", Scope: "local", ReferenceCount: &used},
		{Name: "beta", Driver: "nfs", Scope: "global"},
	}
	result := listDockerVolumes(items, DockerVolumeListQuery{Driver: "local", Usage: "used", Limit: 1, Offset: 1})

	if result.Total != 2 || result.Limit != 1 || result.Offset != 1 {
		t.Fatalf("unexpected page metadata: %#v", result)
	}
	if len(result.Items) != 1 || result.Items[0].Name != "zeta" {
		t.Fatalf("expected sorted second matching volume, got %#v", result.Items)
	}
}

func TestDockerVolumeUsageUnknownDoesNotMatchUnused(t *testing.T) {
	if dockerVolumeUsageMatches(nil, "unused") {
		t.Fatal("unknown reference count must not match unused")
	}
	zero := int64(0)
	if !dockerVolumeUsageMatches(&zero, "unused") {
		t.Fatal("zero reference count must match unused")
	}
}

func TestMapDockerVolumeErrorPreservesConflict(t *testing.T) {
	t.Parallel()

	if got := mapDockerVolumeError(assertError("volume is in use")); got != errDockerVolumeConflict {
		t.Fatalf("expected volume conflict, got %v", got)
	}
}

type volumeTestError string

func (e volumeTestError) Error() string { return string(e) }

func assertError(message string) error { return volumeTestError(message) }
