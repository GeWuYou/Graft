package project

import (
	"errors"
	"testing"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
)

func TestDeriveProjectRuntimeStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		summary *moduleapi.ContainerProjectRuntimeSummary
		err     error
		want    generated.ProjectRuntimeStatus
	}{
		{
			name: "unknown on runtime error",
			summary: &moduleapi.ContainerProjectRuntimeSummary{
				Members: []moduleapi.ContainerProjectMember{{CanonicalState: "running"}},
			},
			err:  errors.New("runtime offline"),
			want: generated.ProjectRuntimeStatusUnknown,
		},
		{
			name:    "stopped without runtime members",
			summary: &moduleapi.ContainerProjectRuntimeSummary{},
			want:    generated.ProjectRuntimeStatusStopped,
		},
		{
			name: "running when all members run",
			summary: &moduleapi.ContainerProjectRuntimeSummary{
				Members: []moduleapi.ContainerProjectMember{
					{CanonicalState: "running"},
					{CanonicalState: "running"},
				},
			},
			want: generated.ProjectRuntimeStatusRunning,
		},
		{
			name: "stopped when all members exited",
			summary: &moduleapi.ContainerProjectRuntimeSummary{
				Members: []moduleapi.ContainerProjectMember{
					{CanonicalState: "exited"},
					{CanonicalState: "exited"},
				},
			},
			want: generated.ProjectRuntimeStatusStopped,
		},
		{
			name: "transitioning when any member is transitioning",
			summary: &moduleapi.ContainerProjectRuntimeSummary{
				Members: []moduleapi.ContainerProjectMember{
					{CanonicalState: "running"},
					{CanonicalState: "restarting"},
				},
			},
			want: generated.ProjectRuntimeStatusTransitioning,
		},
		{
			name: "degraded for mixed stopped and running members",
			summary: &moduleapi.ContainerProjectRuntimeSummary{
				Members: []moduleapi.ContainerProjectMember{
					{CanonicalState: "running"},
					{CanonicalState: "exited"},
				},
			},
			want: generated.ProjectRuntimeStatusDegraded,
		},
		{
			name: "degraded for issue members",
			summary: &moduleapi.ContainerProjectRuntimeSummary{
				Members: []moduleapi.ContainerProjectMember{
					{CanonicalState: "dead"},
				},
			},
			want: generated.ProjectRuntimeStatusDegraded,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := deriveProjectRuntimeStatus(tc.summary, tc.err)
			if got == nil {
				t.Fatalf("expected status %q, got nil", tc.want)
			}
			if *got != tc.want {
				t.Fatalf("expected status %q, got %q", tc.want, *got)
			}
		})
	}
}

func TestBuildProjectContainerCounts(t *testing.T) {
	t.Parallel()

	counts := buildProjectContainerCounts(&moduleapi.ContainerProjectRuntimeSummary{
		Members: []moduleapi.ContainerProjectMember{
			{CanonicalState: "running"},
			{CanonicalState: "exited"},
			{CanonicalState: "restarting"},
			{CanonicalState: "dead"},
			{CanonicalState: "unknown"},
		},
	})

	if counts.Running != 1 || counts.Stopped != 1 || counts.Transitioning != 1 || counts.Issue != 2 || counts.Total != 5 {
		t.Fatalf("unexpected counts: %#v", counts)
	}
}
