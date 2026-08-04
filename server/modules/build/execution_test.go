package build

import (
	"context"
	"encoding/json"
	"testing"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

type recordingBuildRepository struct {
	created   buildstore.JobSnapshot
	snapshot  buildstore.JobSnapshot
	settledID uint64
}

func (r *recordingBuildRepository) CreateJob(_ context.Context, value buildstore.JobSnapshot) error {
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
func (r *recordingBuildRepository) SettleDockerArtifact(_ context.Context, taskID uint64, _ moduleapi.DockerImageBuildResult) error {
	r.settledID = taskID
	return nil
}
func (*recordingBuildRepository) ListJobs(context.Context, int, int) (buildstore.ListResult, error) {
	return buildstore.ListResult{}, nil
}
func (*recordingBuildRepository) GetJobByBuildID(context.Context, string) (buildstore.JobProjection, error) {
	return buildstore.JobProjection{}, buildstore.ErrNotFound
}

type recordingBuildContexts struct{ calls int }

func (r *recordingBuildContexts) ResolveApplicationBuildContext(context.Context, uint64) (moduleapi.ApplicationBuildContext, error) {
	r.calls++
	return moduleapi.ApplicationBuildContext{ApplicationID: 9, DisplayName: "app", WorkspaceRoot: "/workspace/app", RuntimeTargetID: 4, RuntimeProvider: "docker", CanBuild: true}, nil
}

type recordingBuildTasks struct{ input moduleapi.SubmitTaskInput }

func (r *recordingBuildTasks) Submit(_ context.Context, input moduleapi.SubmitTaskInput) (moduleapi.TaskReceipt, error) {
	r.input = input
	return moduleapi.TaskReceipt{TaskID: 42, Status: moduleapi.TaskStatusPending}, nil
}
func (*recordingBuildTasks) SettleExternalReceipt(context.Context, moduleapi.ExternalTaskReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	return moduleapi.ExternalReceiptSettlement{}, nil
}
func (*recordingBuildTasks) Cancel(context.Context, uint64) error             { return nil }
func (*recordingBuildTasks) RetryStage(context.Context, uint64, uint64) error { return nil }

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
	_, err = service.Submit(context.Background(), SubmitRequest{ApplicationID: 9, ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", BuildArgs: []moduleapi.DockerImageBuildArg{{Name: "MODE", Value: "release"}}, IdempotencyKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	if contexts.calls != 1 || repository.created.TaskID != 42 || repository.created.WorkspaceRoot != "/workspace/app" || len(repository.created.BuildArgs) != 1 {
		t.Fatalf("unexpected frozen snapshot: %#v calls=%d", repository.created, contexts.calls)
	}
	var input moduleapi.BuildTaskInput
	if err := json.Unmarshal(tasks.input.Input, &input); err != nil || input.BuildID == "" {
		t.Fatalf("task input must contain build identity: %#v err=%v", input, err)
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
