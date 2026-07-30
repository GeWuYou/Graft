package project

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

type restartExecutorRuntimeReader struct {
	summary moduleapi.ContainerProjectRuntimeSummary
	err     error
}

type recordingLifecycleTaskService struct {
	submitted atomic.Bool
	input     moduleapi.SubmitTaskInput
}

func (s *recordingLifecycleTaskService) Submit(_ context.Context, input moduleapi.SubmitTaskInput) (moduleapi.TaskReceipt, error) {
	s.input = input
	s.submitted.Store(true)
	return moduleapi.TaskReceipt{TaskID: 1}, nil
}

func (*recordingLifecycleTaskService) SettleExternalReceipt(context.Context, moduleapi.ExternalTaskReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	return moduleapi.ExternalReceiptSettlement{}, nil
}

func (*recordingLifecycleTaskService) Cancel(context.Context, uint64) error { return nil }

func (*recordingLifecycleTaskService) RetryStage(context.Context, uint64, uint64) error { return nil }

type countingFailingRuntimeReader struct{ projectReads atomic.Int32 }

func (r *countingFailingRuntimeReader) ListProjectMembers(context.Context, string, string) (moduleapi.ContainerProjectRuntimeSummary, error) {
	r.projectReads.Add(1)
	return moduleapi.ContainerProjectRuntimeSummary{}, errors.New("runtime unavailable")
}

func (*countingFailingRuntimeReader) ListImportCandidates(context.Context, string) ([]moduleapi.ContainerProjectRuntimeCandidate, error) {
	return nil, nil
}

func (*countingFailingRuntimeReader) ListImportCandidateMembers(context.Context, string, moduleapi.ContainerProjectRuntimeCandidate) ([]moduleapi.ContainerProjectMember, error) {
	return nil, nil
}

func TestRestartSubmissionDoesNotReadRuntimeStatus(t *testing.T) {
	t.Parallel()

	reader := &countingFailingRuntimeReader{}
	taskService := &recordingLifecycleTaskService{}
	service, err := NewService(&stubProjectRepository{aggregate: projectstore.ApplicationAggregate{
		Application: projectstore.Application{
			ApplicationRecordID:   17,
			ApplicationID:         "app_01ARZ3NDEKTSV4RRFFQ69G5FAV",
			ComposeProjectName:    "compose-demo",
			WorkspacePath:         "/srv/compose-demo",
			LifecycleReviewStatus: projectcontract.LifecycleReviewStatusConfirmed.String(),
			LifecycleStrategyKind: projectcontract.LifecycleStrategyKindStandard.String(),
		},
		Snapshot: &projectstore.Snapshot{ApplicationRecordID: 17, ConfigHash: "cfg-1", RefreshedAt: time.Now().UTC()},
	}}, WithRuntimeReader(reader))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	service.SetTaskService(taskService)

	result, err := service.Restart(authenticatedApplicationActionContext(), 17, nil)
	if err != nil {
		t.Fatalf("submit restart: %v", err)
	}
	if !taskService.submitted.Load() || result.Result != generated.ApplicationActionResponseResultApplicationActionResultAccepted {
		t.Fatalf("restart result = %#v, task submitted = %t", result, taskService.submitted.Load())
	}
	if reads := reader.projectReads.Load(); reads != 0 {
		t.Fatalf("restart submission read runtime status %d times", reads)
	}
}

func (r restartExecutorRuntimeReader) ListProjectMembers(context.Context, string, string) (moduleapi.ContainerProjectRuntimeSummary, error) {
	return r.summary, r.err
}

func (restartExecutorRuntimeReader) ListImportCandidates(context.Context, string) ([]moduleapi.ContainerProjectRuntimeCandidate, error) {
	return nil, nil
}

func (restartExecutorRuntimeReader) ListImportCandidateMembers(context.Context, string, moduleapi.ContainerProjectRuntimeCandidate) ([]moduleapi.ContainerProjectMember, error) {
	return nil, nil
}

func TestRestartStageExecutorReevaluatesRuntimeStatusImmediatelyBeforeCompose(t *testing.T) {
	t.Parallel()

	aggregate := projectstore.ApplicationAggregate{
		Application: projectstore.Application{
			ApplicationRecordID:   17,
			ComposeProjectName:    "compose-demo",
			WorkspacePath:         "/srv/compose-demo",
			LifecycleReviewStatus: projectcontract.LifecycleReviewStatusConfirmed.String(),
			LifecycleStrategyKind: projectcontract.LifecycleStrategyKindStandard.String(),
		},
		Files: []projectstore.ApplicationFile{{
			Kind:         projectcontract.FileKindCompose.String(),
			AbsolutePath: "/srv/compose-demo/compose.yaml",
		}},
	}
	plan, err := lifecycleTaskPlan(aggregate, generated.ApplicationActionResponseActionApplicationActionRestart)
	if err != nil {
		t.Fatalf("build restart plan: %v", err)
	}
	stage := onlyLifecycleStage(t, plan)
	var input composeStageInput
	if err := json.Unmarshal(stage.Input, &input); err != nil {
		t.Fatalf("decode restart stage input: %v", err)
	}
	if args := input.Args; args[len(args)-1] != "restart" {
		t.Fatalf("submitted restart args = %#v", args)
	}

	t.Run("missing project switches to up", func(t *testing.T) {
		service, err := NewService(
			&stubProjectRepository{aggregate: aggregate},
			WithRuntimeReader(restartExecutorRuntimeReader{}),
		)
		if err != nil {
			t.Fatalf("new service: %v", err)
		}
		args, err := (&composeStageExecutor{typeName: moduleapi.StageExecutorType(composeStagePrefix + "restart"), service: service}).commandArgs(context.Background(), input)
		if err != nil {
			t.Fatalf("resolve restart args: %v", err)
		}
		if got, want := args[len(args)-2:], []string{"up", "-d"}; !equalStrings(got, want) {
			t.Fatalf("execution args suffix = %#v, want %#v", got, want)
		}
	})

	t.Run("runtime read failure keeps restart", func(t *testing.T) {
		service, err := NewService(
			&stubProjectRepository{aggregate: aggregate},
			WithRuntimeReader(restartExecutorRuntimeReader{err: errors.New("runtime unavailable")}),
		)
		if err != nil {
			t.Fatalf("new service: %v", err)
		}
		args, err := (&composeStageExecutor{typeName: moduleapi.StageExecutorType(composeStagePrefix + "restart"), service: service}).commandArgs(context.Background(), input)
		if err != nil {
			t.Fatalf("resolve restart args after runtime failure: %v", err)
		}
		if args[len(args)-1] != "restart" {
			t.Fatalf("runtime read fallback args = %#v, want restart", args)
		}
	})

	t.Run("legacy persisted input keeps its planned restart", func(t *testing.T) {
		args, err := (&composeStageExecutor{typeName: moduleapi.StageExecutorType(composeStagePrefix + "restart")}).commandArgs(context.Background(), composeStageInput{
			WorkspacePath: input.WorkspacePath,
			Args:          input.Args,
		})
		if err != nil {
			t.Fatalf("resolve legacy restart args: %v", err)
		}
		if args[len(args)-1] != "restart" {
			t.Fatalf("legacy restart args = %#v, want restart", args)
		}
	})
}
