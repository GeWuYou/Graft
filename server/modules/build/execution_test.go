package build

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

type recordingBuildRepository struct {
	created        buildstore.JobSnapshot
	snapshot       buildstore.JobSnapshot
	settledID      uint64
	settleCanceled bool
	settleDeadline bool
	getBuildIDErr  error
	createErr      error
}

func (r *recordingBuildRepository) CreateJob(_ context.Context, value buildstore.JobSnapshot) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.created = value
	r.snapshot = value
	return nil
}
func (r *recordingBuildRepository) GetJobByTaskID(_ context.Context, taskID uint64) (buildstore.JobSnapshot, error) {
	if taskID != r.snapshot.TaskID {
		return buildstore.JobSnapshot{}, buildstore.ErrNotFound
	}
	return r.snapshot, nil
}
func (r *recordingBuildRepository) SettleDockerArtifact(ctx context.Context, taskID uint64, _ moduleapi.DockerImageBuildResult) error {
	r.settledID = taskID
	r.settleCanceled = errors.Is(ctx.Err(), context.Canceled)
	_, r.settleDeadline = ctx.Deadline()
	return nil
}
func (*recordingBuildRepository) ListJobs(context.Context, buildstore.ListQuery) (buildstore.ListResult, error) {
	return buildstore.ListResult{}, nil
}

func (r *recordingBuildRepository) GetJobByBuildID(context.Context, string) (buildstore.JobProjection, error) {
	if r.getBuildIDErr != nil {
		return buildstore.JobProjection{}, r.getBuildIDErr
	}
	return buildstore.JobProjection{}, buildstore.ErrNotFound
}

type recordingBuildContexts struct{ calls int }

func (r *recordingBuildContexts) ResolveApplicationBuildContext(context.Context, string) (moduleapi.ApplicationBuildContext, error) {
	r.calls++
	return moduleapi.ApplicationBuildContext{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ApplicationRecordID: 9, DisplayName: "app", WorkspaceRoot: "/workspace/app", RuntimeTargetID: 4, RuntimeProvider: "docker", CanBuild: true}, nil
}

type recordingBuildTasks struct {
	input         moduleapi.SubmitTaskInput
	reserveCalls  int
	activateCalls int
	discardCalls  int
	err           error
}

func (r *recordingBuildTasks) ReserveTask(_ context.Context, input moduleapi.SubmitTaskInput) (moduleapi.TaskReservation, error) {
	r.reserveCalls++
	r.input = input
	if r.err != nil {
		return moduleapi.TaskReservation{}, r.err
	}
	return moduleapi.TaskReservation{TaskID: 42}, nil
}
func (r *recordingBuildTasks) ActivateTask(_ context.Context, reservation moduleapi.TaskReservation) (moduleapi.TaskReceipt, error) {
	r.activateCalls++
	return moduleapi.TaskReceipt{TaskID: reservation.TaskID, Status: moduleapi.TaskStatusPending}, nil
}
func (r *recordingBuildTasks) DiscardTaskReservation(context.Context, moduleapi.TaskReservation) error {
	r.discardCalls++
	return nil
}

type recordingBuildDocker struct {
	input moduleapi.DockerImageBuildInput
}

func (r *recordingBuildDocker) BuildImage(_ context.Context, input moduleapi.DockerImageBuildInput, _ moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	r.input = input
	return moduleapi.DockerImageBuildResult{ImageID: "sha256:image", Repository: input.ImageRepository, Tag: input.ImageTag}, nil
}

type buildStageRun struct{ input json.RawMessage }

func (buildStageRun) TaskID() uint64                                          { return 42 }
func (buildStageRun) StageID() uint64                                         { return 7 }
func (buildStageRun) Attempt() int                                            { return 1 }
func (r buildStageRun) Input() json.RawMessage                                { return r.input }
func (buildStageRun) CancellationRequested(context.Context) bool              { return false }
func (buildStageRun) AppendLog(context.Context, moduleapi.TaskLogEntry) error { return nil }

func TestSubmitFreezesAuthorizedBuildSnapshot(t *testing.T) {
	contexts := &recordingBuildContexts{}
	tasks := &recordingBuildTasks{}
	repository := &recordingBuildRepository{}
	service, err := NewService(contexts, tasks, &recordingBuildDocker{}, repository)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Submit(context.Background(), SubmitRequest{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", BuildArgs: []moduleapi.DockerImageBuildArg{{Name: "MODE", Value: "release"}}, IdempotencyKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	if contexts.calls != 1 || repository.created.TaskID != 42 || repository.created.WorkspaceRoot != "/workspace/app" || len(repository.created.BuildArgs) != 1 {
		t.Fatalf("unexpected frozen snapshot: %#v calls=%d", repository.created, contexts.calls)
	}
	if tasks.reserveCalls != 1 || tasks.activateCalls != 1 || tasks.discardCalls != 0 {
		t.Fatalf("reservation lifecycle = reserve:%d activate:%d discard:%d", tasks.reserveCalls, tasks.activateCalls, tasks.discardCalls)
	}
	var input moduleapi.BuildTaskInput
	if err := json.Unmarshal(tasks.input.Input, &input); err != nil || input.BuildID == "" {
		t.Fatalf("task input must contain build identity: %#v err=%v", input, err)
	}
}

func TestSubmitDiscardsReservationWhenSnapshotFreezeFails(t *testing.T) {
	tasks := &recordingBuildTasks{}
	repository := &recordingBuildRepository{createErr: errors.New("snapshot write failed")}
	service, err := NewService(&recordingBuildContexts{}, tasks, &recordingBuildDocker{}, repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Submit(context.Background(), SubmitRequest{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", IdempotencyKey: "key"}); err == nil {
		t.Fatal("Submit error = nil, want snapshot failure")
	}
	if tasks.reserveCalls != 1 || tasks.activateCalls != 0 || tasks.discardCalls != 1 {
		t.Fatalf("reservation lifecycle = reserve:%d activate:%d discard:%d", tasks.reserveCalls, tasks.activateCalls, tasks.discardCalls)
	}
}

func TestExecutorUsesBuildSnapshotAndSettlesArtifact(t *testing.T) {
	repository := &recordingBuildRepository{snapshot: buildstore.JobSnapshot{TaskID: 42, BuildID: "build_test", WorkspaceRoot: "/workspace/app", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", BuildArgs: []moduleapi.DockerImageBuildArg{{Name: "MODE", Value: "release"}}}}
	docker := &recordingBuildDocker{}
	raw, _ := json.Marshal(moduleapi.BuildTaskInput{BuildID: "build_test"})
	executor := &dockerfileBuildExecutor{repository: repository, docker: docker, cancels: make(map[uint64]context.CancelFunc)}
	if err := executor.Execute(context.Background(), buildStageRun{input: raw}); err != nil {
		t.Fatal(err)
	}
	if docker.input.WorkspaceRoot != "/workspace/app" || docker.input.ContextPath != "src" || repository.settledID != 42 {
		t.Fatalf("executor did not use and settle snapshot: docker=%#v settled=%d", docker.input, repository.settledID)
	}
}

func TestExecutorSettlesArtifactAfterCallerCancellation(t *testing.T) {
	repository := &recordingBuildRepository{snapshot: buildstore.JobSnapshot{TaskID: 42, BuildID: "build_test", WorkspaceRoot: "/workspace/app", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1"}}
	raw, _ := json.Marshal(moduleapi.BuildTaskInput{BuildID: "build_test"})
	executor := &dockerfileBuildExecutor{repository: repository, docker: &recordingBuildDocker{}, cancels: make(map[uint64]context.CancelFunc)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := executor.Execute(ctx, buildStageRun{input: raw}); err != nil {
		t.Fatal(err)
	}
	if repository.settleCanceled {
		t.Fatal("artifact settlement inherited caller cancellation")
	}
	if !repository.settleDeadline {
		t.Fatal("artifact settlement has no bounded timeout")
	}
}

func TestSubmitRejectsInvalidInputBeforeTaskSubmission(t *testing.T) {
	invalidRequests := []SubmitRequest{
		{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "../etc", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", IdempotencyKey: "key"},
		{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "/Dockerfile", ImageRepository: "example/app", ImageTag: "v1", IdempotencyKey: "key"},
		{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", BuildArgs: []moduleapi.DockerImageBuildArg{{Name: "MODE"}, {Name: "MODE"}}, IdempotencyKey: "key"},
		{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app:latest", ImageTag: "v1", IdempotencyKey: "key"},
		{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "invalid tag", IdempotencyKey: "key"},
	}

	for _, request := range invalidRequests {
		t.Run(request.ContextPath+request.DockerfilePath+request.ImageRepository+request.ImageTag, func(t *testing.T) {
			tasks := &recordingBuildTasks{}
			service, err := NewService(&recordingBuildContexts{}, tasks, &recordingBuildDocker{}, &recordingBuildRepository{})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Submit(context.Background(), request); !errors.Is(err, errInvalidBuildRequest) {
				t.Fatalf("Submit error = %v, want invalid request", err)
			}
			if tasks.reserveCalls != 0 {
				t.Fatalf("task reservation calls = %d, want 0", tasks.reserveCalls)
			}
		})
	}
}

func TestNormalizeSubmitRequestTrimsDockerReference(t *testing.T) {
	request, err := normalizeSubmitRequest(SubmitRequest{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: " example/app ", ImageTag: " v1 "})
	if err != nil {
		t.Fatal(err)
	}
	if request.ImageRepository != "example/app" || request.ImageTag != "v1" {
		t.Fatalf("normalized reference = %q:%q", request.ImageRepository, request.ImageTag)
	}
}
