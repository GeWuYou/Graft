package build

import (
	"context"
	"database/sql"
	"errors"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

type recordingBuildRepository struct {
	created            buildstore.JobSnapshot
	snapshot           buildstore.JobSnapshot
	settledID          uint64
	getBuildIDErr      error
	createErr          error
	listResult         buildstore.ListResult
	listQuery          buildstore.ListQuery
	artifactResult     buildstore.V2ArtifactListResult
	v2Plan             moduleapi.BuildExecutionPlan
	workspaces         []moduleapi.BuildWorkspace
	workspaceQuery     buildstore.WorkspaceListQuery
	publicationSources []moduleapi.ArtifactPublicationSource
	promotionInput     moduleapi.OCIArtifactCopyInput
	promotionResult    moduleapi.OCIArtifactCopyResult
	promotionAuth      moduleapi.RegistryAuthExecution
	promotionSettled   bool
	platformArtifacts  []moduleapi.PlatformArtifact
	manifestInput      moduleapi.OCIManifestPublicationInput
	manifestSettled    bool
	inputSnapshots     buildstore.InputSnapshotListResult
}

func (r *recordingBuildRepository) CreateBuildInputSnapshot(_ context.Context, snapshotID, sourceReference, contentDigest, materializationRef string, _ uint64) (moduleapi.WorkspaceSnapshot, error) {
	return moduleapi.WorkspaceSnapshot{ID: snapshotID, SourceKind: moduleapi.WorkspaceSourceArchive, SourceReference: sourceReference, ContentDigest: contentDigest, MaterializationRef: materializationRef}, nil
}
func (r *recordingBuildRepository) GetBuildInputSnapshot(_ context.Context, snapshotID string, _ uint64) (moduleapi.WorkspaceSnapshot, error) {
	for _, snapshot := range r.inputSnapshots.Items {
		if snapshot.ID == snapshotID {
			return snapshot, nil
		}
	}
	return moduleapi.WorkspaceSnapshot{}, buildstore.ErrNotFound
}
func (r *recordingBuildRepository) ListBuildInputSnapshots(_ context.Context, _ uint64, _ int, _ int) (buildstore.InputSnapshotListResult, error) {
	return r.inputSnapshots, nil
}

func (r *recordingBuildRepository) RecordPlatformArtifact(_ context.Context, _ uint64, _ moduleapi.BuildExecutionPlan, artifact moduleapi.PlatformArtifact) error {
	r.platformArtifacts = append(r.platformArtifacts, artifact)
	return nil
}
func (r *recordingBuildRepository) ListPlatformArtifacts(context.Context, uint64, moduleapi.BuildExecutionPlan) ([]moduleapi.PlatformArtifact, error) {
	return append([]moduleapi.PlatformArtifact(nil), r.platformArtifacts...), nil
}
func (r *recordingBuildRepository) PrepareOCIManifestPublication(_ context.Context, _ uint64, plan moduleapi.BuildExecutionPlan) (moduleapi.OCIManifestPublicationInput, error) {
	r.manifestInput = moduleapi.OCIManifestPublicationInput{Destination: moduleapi.AuthorizedArtifactDestination(plan.Destination), PlatformArtifacts: append([]moduleapi.PlatformArtifact(nil), r.platformArtifacts...)}
	return r.manifestInput, nil
}
func (r *recordingBuildRepository) SettleOCIManifestPublication(context.Context, uint64, moduleapi.BuildExecutionPlan, moduleapi.OCIManifestPublicationResult, moduleapi.RegistryAuthExecution) error {
	r.manifestSettled = true
	return nil
}
func (*recordingBuildRepository) CreateWorkspace(context.Context, moduleapi.BuildWorkspace) error {
	return nil
}
func (*recordingBuildRepository) GetWorkspace(context.Context, string) (moduleapi.BuildWorkspace, error) {
	return moduleapi.BuildWorkspace{ID: "workspace_app", Name: "Application", SourceKind: moduleapi.WorkspaceSourceApplication, SourceReference: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X"}, nil
}
func (r *recordingBuildRepository) ListWorkspaces(_ context.Context, _ uint64, query buildstore.WorkspaceListQuery) (buildstore.WorkspaceListResult, error) {
	r.workspaceQuery = query
	return buildstore.WorkspaceListResult{Items: append([]moduleapi.BuildWorkspace(nil), r.workspaces...), Total: int64(len(r.workspaces))}, nil
}
func (r *recordingBuildRepository) MaterializeExecutionPlan(_ context.Context, _ *sql.Tx, submission moduleapi.TaskSubmission, plan moduleapi.BuildExecutionPlan, _ uint64) (string, error) {
	if r.createErr != nil || submission.TaskID == nil {
		return "", errors.New("execution plan materialization failed")
	}
	r.v2Plan = plan
	return plan.ID, nil
}
func (r *recordingBuildRepository) CreateJob(_ context.Context, value buildstore.JobSnapshot) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.created, r.snapshot = value, value
	return nil
}
func (r *recordingBuildRepository) MaterializeSubmissionSnapshot(_ context.Context, _ *sql.Tx, submission moduleapi.TaskSubmission, value buildstore.JobSnapshot) (string, error) {
	if r.createErr != nil {
		return "", r.createErr
	}
	value.SubmissionID = submission.ID
	if submission.TaskID != nil {
		value.TaskID = *submission.TaskID
	}
	r.created, r.snapshot = value, value
	return value.BuildID, nil
}
func (r *recordingBuildRepository) GetJobByTaskID(_ context.Context, taskID uint64) (buildstore.JobSnapshot, error) {
	if taskID != r.snapshot.TaskID {
		return buildstore.JobSnapshot{}, buildstore.ErrNotFound
	}
	return r.snapshot, nil
}
func (r *recordingBuildRepository) SettleBuildArtifact(_ context.Context, taskID uint64, _ moduleapi.BuildArtifactResult) error {
	r.settledID = taskID
	return nil
}
func (r *recordingBuildRepository) ListJobs(_ context.Context, query buildstore.ListQuery) (buildstore.ListResult, error) {
	r.listQuery = query
	return r.listResult, nil
}
func (r *recordingBuildRepository) ListV2Artifacts(context.Context, int, int) (buildstore.V2ArtifactListResult, error) {
	return r.artifactResult, nil
}
func (r *recordingBuildRepository) ListArtifactPublicationSources(context.Context, string) ([]moduleapi.ArtifactPublicationSource, error) {
	return append([]moduleapi.ArtifactPublicationSource(nil), r.publicationSources...), nil
}
func (r *recordingBuildRepository) SettleArtifactPromotion(_ context.Context, input moduleapi.OCIArtifactCopyInput, result moduleapi.OCIArtifactCopyResult, auth moduleapi.RegistryAuthExecution) error {
	r.promotionInput, r.promotionResult, r.promotionAuth, r.promotionSettled = input, result, auth, true
	return nil
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
	input                                                   moduleapi.SubmitTaskInput
	beginCalls, materializeCalls, discardCalls, submitCalls int
	err                                                     error
	views                                                   []moduleapi.TaskView
	batchErr                                                error
	batchIDs                                                []uint64
	beginSubmission                                         *moduleapi.TaskSubmission
}

func (r *recordingBuildTasks) Submit(_ context.Context, input moduleapi.SubmitTaskInput) (moduleapi.TaskReceipt, error) {
	r.submitCalls++
	r.input = input
	if r.err != nil {
		return moduleapi.TaskReceipt{}, r.err
	}
	return moduleapi.TaskReceipt{TaskID: 42, Status: moduleapi.TaskStatusReady}, nil
}
func (*recordingBuildTasks) SettleExternalReceipt(context.Context, moduleapi.ExternalTaskReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	return moduleapi.ExternalReceiptSettlement{}, errors.New("not implemented")
}
func (*recordingBuildTasks) Cancel(context.Context, uint64) error             { return nil }
func (*recordingBuildTasks) RetryStage(context.Context, uint64, uint64) error { return nil }
func (r *recordingBuildTasks) GetTasksByIDs(_ context.Context, ids []uint64) ([]moduleapi.TaskView, error) {
	r.batchIDs = append([]uint64(nil), ids...)
	return r.views, r.batchErr
}
func (r *recordingBuildTasks) BeginSubmission(_ context.Context, input moduleapi.BeginTaskSubmissionInput) (moduleapi.TaskSubmissionHandle, error) {
	r.beginCalls++
	r.input = input.Task
	if r.err != nil {
		return moduleapi.TaskSubmissionHandle{}, r.err
	}
	if r.beginSubmission != nil {
		return moduleapi.TaskSubmissionHandle{Submission: *r.beginSubmission, LeaseToken: "lease"}, nil
	}
	return moduleapi.TaskSubmissionHandle{Submission: moduleapi.TaskSubmission{ID: "submission_test", TaskType: input.Task.Type, Owner: input.Task.Owner, State: moduleapi.TaskSubmissionStateReserved, SubmissionVersion: 1}, LeaseToken: "lease"}, nil
}
func (*recordingBuildTasks) RenewSubmissionLease(context.Context, moduleapi.TaskSubmissionHandle) (moduleapi.TaskSubmissionHandle, error) {
	return moduleapi.TaskSubmissionHandle{}, errors.New("not implemented")
}
func (r *recordingBuildTasks) MaterializeSubmission(ctx context.Context, handle moduleapi.TaskSubmissionHandle, input moduleapi.SubmitTaskInput, writer moduleapi.TaskSubmissionWriter) (moduleapi.TaskReceipt, error) {
	r.materializeCalls++
	r.input = input
	if r.err != nil {
		return moduleapi.TaskReceipt{}, r.err
	}
	taskID := uint64(42)
	if _, err := writer.MaterializeTaskSubmission(ctx, nil, moduleapi.TaskSubmission{ID: handle.Submission.ID, TaskID: &taskID}); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	return moduleapi.TaskReceipt{TaskID: taskID, Status: moduleapi.TaskStatusReady}, nil
}
func (r *recordingBuildTasks) DiscardSubmission(context.Context, moduleapi.TaskSubmissionHandle, string) error {
	r.discardCalls++
	return nil
}
func (*recordingBuildTasks) ExpireSubmissions(context.Context, int) (int, error) { return 0, nil }
func (*recordingBuildTasks) GetSubmission(context.Context, string) (moduleapi.TaskSubmission, error) {
	return moduleapi.TaskSubmission{}, errors.New("not implemented")
}

type promotionRegistryStub struct {
	binding moduleapi.RegistryArtifactCopyBinding
	err     error
	copy    moduleapi.AuthorizedArtifactCopy
}

func (s *promotionRegistryStub) AuthorizeArtifactCopy(_ context.Context, _ uint64, source moduleapi.ArtifactPublicationSource, destination moduleapi.BuildDestination) (moduleapi.AuthorizedArtifactCopy, error) {
	return moduleapi.AuthorizedArtifactCopy{Source: source, Destination: moduleapi.AuthorizedArtifactDestination(destination)}, s.err
}
func (s *promotionRegistryStub) ResolveArtifactCopyBinding(_ context.Context, copy moduleapi.AuthorizedArtifactCopy) (moduleapi.RegistryArtifactCopyBinding, error) {
	s.copy = copy
	return s.binding, s.err
}
