package container

import (
	"context"
	"testing"
)

type dockerFactsDetailRuntime struct {
	stubProjectReaderRuntime
	detail Detail
}

func (r dockerFactsDetailRuntime) Detail(context.Context, Ref) (Detail, error) {
	return r.detail, nil
}

func TestCurrentContainerReturnsCopiedRawDockerFacts(t *testing.T) {
	runtime := dockerFactsDetailRuntime{detail: Detail{Summary: Summary{Labels: map[string]string{
		"example.label": "original",
	}}, Mounts: []Mount{{Type: "bind", Source: "/srv/graft", Destination: "/app"}}}}
	reader := containerProjectRuntimeReader{service: &service{runtime: runtime, enabled: true}}

	facts, err := reader.CurrentContainer(context.Background())
	if err != nil {
		t.Fatalf("read raw docker facts: %v", err)
	}
	if facts.Labels["example.label"] != "original" || len(facts.Mounts) != 1 || facts.Mounts[0].Source != "/srv/graft" {
		t.Fatalf("unexpected raw docker facts: %#v", facts)
	}
	facts.Labels["example.label"] = "mutated"
	facts.Mounts[0].Source = "/mutated"
	fresh, err := reader.CurrentContainer(context.Background())
	if err != nil {
		t.Fatalf("read copied raw docker facts: %v", err)
	}
	if fresh.Labels["example.label"] != "original" || fresh.Mounts[0].Source != "/srv/graft" {
		t.Fatalf("raw Docker facts leaked caller mutation: %#v", fresh)
	}
}
