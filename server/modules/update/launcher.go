package update

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/moby/moby/api/pkg/stdcopy"
	containertypes "github.com/moby/moby/api/types/container"
	mobyclient "github.com/moby/moby/client"
)

// ComposeRunnerLauncher 保留更新终态观察与受控恢复所需的 Docker 边界。
// 正常 Update Controller 启动不再经过该边界，而由 Runtime Agent 外部执行 lease 负责。
type ComposeRunnerLauncher interface {
	Close() error
}

// ComposeRunnerReceiptReader 读取保留 runner 容器日志中的无秘密结算回执。
type ComposeRunnerReceiptReader interface {
	ReadRunnerReceipts(context.Context) ([]RunnerReceipt, error)
}

// ComposeRunnerProgressReader 读取保留 runner 日志中无秘密的固定阶段标记。
type ComposeRunnerProgressReader interface {
	ReadRunnerProgress(context.Context) ([]RunnerOperationProgress, error)
}

// ComposeRunnerReceiptCleanup 按稳定 operation ID 清理已成功结算的 runner 容器。
type ComposeRunnerReceiptCleanup interface {
	RemoveRunner(context.Context, string) error
}

type dockerComposeRunnerLauncher struct{ client dockerRunnerClient }

type dockerRunnerClient interface {
	ImagePull(context.Context, string, mobyclient.ImagePullOptions) (mobyclient.ImagePullResponse, error)
	ContainerCreate(context.Context, mobyclient.ContainerCreateOptions) (mobyclient.ContainerCreateResult, error)
	ContainerStart(context.Context, string, mobyclient.ContainerStartOptions) (mobyclient.ContainerStartResult, error)
	ContainerInspect(context.Context, string, mobyclient.ContainerInspectOptions) (mobyclient.ContainerInspectResult, error)
	ContainerRemove(context.Context, string, mobyclient.ContainerRemoveOptions) (mobyclient.ContainerRemoveResult, error)
	ContainerList(context.Context, mobyclient.ContainerListOptions) (mobyclient.ContainerListResult, error)
	ContainerLogs(context.Context, string, mobyclient.ContainerLogsOptions) (mobyclient.ContainerLogsResult, error)
	Close() error
}

// RunnerReceiptLogMarker 是 runner 回执日志使用的稳定协议标记，写入与读取必须共用这一 authority。
const RunnerReceiptLogMarker = "GRAFT_UPDATE_RECEIPT:"

// RunnerFailureLogMarker 将早期 runner 失败绑定到操作，且不暴露 runner 原始输出。
const RunnerFailureLogMarker = "GRAFT_UPDATE_FAILURE:"

const (
	// RunnerFailureCodeStateWriteFailed identifies a runner that could not publish its terminal state.
	RunnerFailureCodeStateWriteFailed = "runner_state_write_failed"
	// RunnerFailureCodeExited identifies a retained runner that exited without a protocol failure marker.
	RunnerFailureCodeExited = "runner_exited"
	// RunnerFailureCodeStateCorrupt identifies a recovery that quarantined an unverifiable runner snapshot.
	RunnerFailureCodeStateCorrupt = "runner_state_corrupt"
	// RunnerFailureStagePermissionDenied identifies state-volume permission failures.
	RunnerFailureStagePermissionDenied = "permission_denied"
	// RunnerFailureStageIOFailed identifies non-permission state-volume write failures.
	RunnerFailureStageIOFailed = "io_failed"
)

const retainedRunnerLogTail = "200"

const (
	runnerProtocolLabel  = "io.graft.update.protocol"
	runnerOperationLabel = "io.graft.update.operation"
)

// RunnerProgressLogMarker 是 runner 阶段日志使用的稳定标记；标记后仅允许固定枚举值。
const RunnerProgressLogMarker = "GRAFT_UPDATE_STAGE:"

// RunnerOperationProgress 将 runner operation identity 与一个受限阶段关联。
type RunnerOperationProgress struct {
	OperationID string
	Progress    RunnerProgress
}

// RunnerFailureEvidence 是 runner 无法持久化终态快照时输出的受控诊断标记。
type RunnerFailureEvidence struct {
	ProtocolVersion int    `json:"protocol_version"`
	OperationID     string `json:"operation_id"`
	RunnerID        string `json:"runner_id"`
	FailureCode     string `json:"failure_code"`
	FailureStage    string `json:"failure_stage"`
}

func parseRunnerProgressLog(line string) (RunnerProgress, bool) {
	value := strings.TrimSpace(line)
	progress := RunnerProgress(strings.TrimSpace(strings.TrimPrefix(value, RunnerProgressLogMarker)))
	if !strings.HasPrefix(value, RunnerProgressLogMarker) {
		return "", false
	}
	_, ok := outcomeForRunnerProgress(progress)
	return progress, ok
}

func parseRunnerReceiptLog(line string) (RunnerReceipt, bool) {
	value := strings.TrimSpace(line)
	if !strings.HasPrefix(value, RunnerReceiptLogMarker) {
		return RunnerReceipt{}, false
	}
	encoded := strings.TrimSpace(strings.TrimPrefix(value, RunnerReceiptLogMarker))
	contents, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return RunnerReceipt{}, false
	}
	var receipt RunnerReceipt
	if err := json.Unmarshal(contents, &receipt); err != nil || !supportedRunnerProtocolVersion(receipt.ProtocolVersion) || !runnerOperationID.MatchString(receipt.OperationID) {
		return RunnerReceipt{}, false
	}
	return receipt, true
}

func parseRunnerFailureLog(line string) (RunnerFailureEvidence, bool) {
	value := strings.TrimSpace(line)
	if !strings.HasPrefix(value, RunnerFailureLogMarker) {
		return RunnerFailureEvidence{}, false
	}
	contents, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(value, RunnerFailureLogMarker)))
	if err != nil {
		return RunnerFailureEvidence{}, false
	}
	var evidence RunnerFailureEvidence
	if err := json.Unmarshal(contents, &evidence); err != nil || !supportedRunnerProtocolVersion(evidence.ProtocolVersion) || !runnerOperationID.MatchString(evidence.OperationID) || !runnerOperationID.MatchString(evidence.RunnerID) || !validRunnerFailureEvidence(evidence) {
		return RunnerFailureEvidence{}, false
	}
	return evidence, true
}

func validRunnerFailureEvidence(value RunnerFailureEvidence) bool {
	return value.FailureCode == RunnerFailureCodeStateWriteFailed && (value.FailureStage == RunnerFailureStagePermissionDenied || value.FailureStage == RunnerFailureStageIOFailed)
}

// Close 释放 Docker API client 持有的连接。
func (l *dockerComposeRunnerLauncher) Close() error {
	if l == nil || l.client == nil {
		return nil
	}
	return l.client.Close()
}

// NewDockerComposeRunnerLauncher 创建只用于终态观察与恢复的 Docker socket capability。
func NewDockerComposeRunnerLauncher() (ComposeRunnerLauncher, error) {
	client, err := mobyclient.New(mobyclient.WithHost("unix:///var/run/docker.sock"))
	if err != nil {
		return nil, fmt.Errorf("create compose runner docker client: %w", err)
	}
	return &dockerComposeRunnerLauncher{client: client}, nil
}

//nolint:cyclop // 读取与解码分别对应 Docker 日志和回执协议的失败边界。
func (l *dockerComposeRunnerLauncher) ReadRunnerReceipts(ctx context.Context) ([]RunnerReceipt, error) {
	if l == nil || l.client == nil {
		return nil, errors.New("compose runner receipt reader is unavailable")
	}
	filters := retainedRunnerFilters()
	result, err := l.client.ContainerList(ctx, mobyclient.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return nil, fmt.Errorf("list retained compose runners: %w", err)
	}
	var receipts []RunnerReceipt
	for _, item := range result.Items {
		if !supportedRunnerProtocol(item.Labels[runnerProtocolLabel]) {
			continue
		}
		logsResult, logErr := l.client.ContainerLogs(ctx, item.ID, mobyclient.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
		if logErr != nil {
			return nil, fmt.Errorf("read retained compose runner logs: %w", logErr)
		}
		logs := logsResult
		var stdout, stderr bytes.Buffer
		_, copyErr := stdcopy.StdCopy(&stdout, &stderr, logs)
		_ = logs.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("decode retained compose runner logs: %w", copyErr)
		}
		var containerReceipts []RunnerReceipt
		for _, line := range strings.Split(stdout.String()+"\n"+stderr.String(), "\n") {
			if receipt, ok := parseRunnerReceiptLog(line); ok && receipt.OperationID == item.Labels[runnerOperationLabel] && runnerProtocolMatchesVersion(item.Labels[runnerProtocolLabel], receipt.ProtocolVersion) {
				containerReceipts = append(containerReceipts, receipt)
			}
		}
		if len(containerReceipts) == 0 {
			continue
		}
		receipts = append(receipts, containerReceipts...)
	}
	return receipts, nil
}

// ReadRunnerProgress 从保留 runner 容器中恢复最后一次可识别的阶段标记。
func (l *dockerComposeRunnerLauncher) ReadRunnerProgress(ctx context.Context) ([]RunnerOperationProgress, error) {
	if l == nil || l.client == nil {
		return nil, errors.New("compose runner progress reader is unavailable")
	}
	filters := make(mobyclient.Filters).Add("label", "io.graft.update.protocol="+runnerProtocol)
	result, err := l.client.ContainerList(ctx, mobyclient.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return nil, fmt.Errorf("list retained compose runners: %w", err)
	}
	progresses := make([]RunnerOperationProgress, 0, len(result.Items))
	for _, item := range result.Items {
		progress, ok, err := l.readContainerProgress(ctx, item)
		if err != nil {
			return nil, err
		}
		if ok {
			progresses = append(progresses, progress)
		}
	}
	return progresses, nil
}

// ReadRunnerFailures 仅从非零退出的保留 runner 读取受控失败证据。
//
//nolint:cyclop,gocognit,gocyclo // 容器、退出状态、日志协议和 identity 的独立校验均为信任边界。
func (l *dockerComposeRunnerLauncher) ReadRunnerFailures(ctx context.Context) ([]RunnerFailureEvidence, error) {
	if l == nil || l.client == nil {
		return nil, errors.New("compose runner failure reader is unavailable")
	}
	filters := retainedRunnerFilters()
	result, err := l.client.ContainerList(ctx, mobyclient.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return nil, fmt.Errorf("list retained compose runners: %w", err)
	}
	failures := make([]RunnerFailureEvidence, 0, len(result.Items))
	for _, item := range result.Items {
		operationID := item.Labels[runnerOperationLabel]
		if !runnerOperationID.MatchString(operationID) || !supportedRunnerProtocol(item.Labels[runnerProtocolLabel]) {
			continue
		}
		inspected, inspectErr := l.client.ContainerInspect(ctx, item.ID, mobyclient.ContainerInspectOptions{})
		if inspectErr != nil {
			return nil, fmt.Errorf("inspect retained compose runner: %w", inspectErr)
		}
		if inspected.Container.State == nil || inspected.Container.State.Running || inspected.Container.State.ExitCode == 0 {
			continue
		}
		evidence := RunnerFailureEvidence{OperationID: operationID, FailureCode: RunnerFailureCodeExited, FailureStage: "unknown"}
		logs, logErr := l.readContainerLogs(ctx, item.ID)
		if logErr != nil {
			return nil, logErr
		}
		for _, line := range strings.Split(logs, "\n") {
			if parsed, ok := parseRunnerFailureLog(line); ok && parsed.OperationID == operationID && runnerProtocolMatchesVersion(item.Labels[runnerProtocolLabel], parsed.ProtocolVersion) {
				evidence = parsed
			}
		}
		failures = append(failures, evidence)
	}
	return failures, nil
}

func (l *dockerComposeRunnerLauncher) readContainerProgress(ctx context.Context, item containertypes.Summary) (RunnerOperationProgress, bool, error) {
	operationID := item.Labels[runnerOperationLabel]
	if !runnerOperationID.MatchString(operationID) {
		return RunnerOperationProgress{}, false, nil
	}
	logs, err := l.readContainerLogs(ctx, item.ID)
	if err != nil {
		return RunnerOperationProgress{}, false, err
	}
	progress, ok := latestRunnerProgress(logs)
	if !ok {
		return RunnerOperationProgress{}, false, nil
	}
	return RunnerOperationProgress{OperationID: operationID, Progress: progress}, true, nil
}

func (l *dockerComposeRunnerLauncher) readContainerLogs(ctx context.Context, id string) (string, error) {
	logs, err := l.client.ContainerLogs(ctx, id, mobyclient.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Tail: retainedRunnerLogTail})
	if err != nil {
		return "", fmt.Errorf("read retained compose runner logs: %w", err)
	}
	defer func() { _ = logs.Close() }()
	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, logs); err != nil {
		return "", fmt.Errorf("decode retained compose runner logs: %w", err)
	}
	return stdout.String() + "\n" + stderr.String(), nil
}

func latestRunnerProgress(logs string) (RunnerProgress, bool) {
	var latest RunnerProgress
	for _, line := range strings.Split(logs, "\n") {
		if progress, ok := parseRunnerProgressLog(line); ok {
			latest = progress
		}
	}
	return latest, latest != ""
}

// RemoveRunner 删除指定 operation ID 对应的 runner；调用方应仅在 receipt 成功结算后调用。
func (l *dockerComposeRunnerLauncher) RemoveRunner(ctx context.Context, operationID string) error {
	if l == nil || l.client == nil {
		return errors.New("compose runner receipt cleanup is unavailable")
	}
	if !runnerOperationID.MatchString(operationID) {
		return errors.New("compose runner receipt cleanup operation ID is invalid")
	}
	filters := retainedRunnerFilters().Add("label", runnerOperationLabel+"="+operationID)
	result, err := l.client.ContainerList(ctx, mobyclient.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return fmt.Errorf("list settled compose runners: %w", err)
	}
	for _, item := range result.Items {
		if item.Labels[runnerOperationLabel] != operationID || !supportedRunnerProtocol(item.Labels[runnerProtocolLabel]) {
			continue
		}
		if _, err := l.client.ContainerRemove(ctx, item.ID, mobyclient.ContainerRemoveOptions{}); err != nil {
			return fmt.Errorf("remove settled compose runner: %w", err)
		}
	}
	return nil
}

func supportedRunnerProtocolVersion(version int) bool {
	_, ok := runnerProtocolForVersion(version)
	return ok
}

func supportedRunnerProtocol(protocol string) bool {
	return protocol == runnerProtocol || protocol == legacyRunnerProtocol
}

func retainedRunnerFilters() mobyclient.Filters {
	return make(mobyclient.Filters).Add("label", runnerProtocolLabel)
}

func runnerProtocolMatchesVersion(protocol string, version int) bool {
	expected, ok := runnerProtocolForVersion(version)
	return ok && protocol == expected
}

func encodeRunnerInput(input RunnerInput) (string, error) {
	contents, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("encode compose runner input: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(contents), nil
}

func runnerStateVolumeName() (string, error) {
	name := strings.TrimSpace(os.Getenv("GRAFT_UPDATE_STATE_VOLUME"))
	if name == "" {
		return "graft-update-state", nil
	}
	if !runnerOperationID.MatchString(name) {
		return "", errors.New("configured update state volume name is invalid")
	}
	return name, nil
}
