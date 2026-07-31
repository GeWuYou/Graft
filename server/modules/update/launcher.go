package update

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/moby/moby/api/pkg/stdcopy"
	containertypes "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	mobyclient "github.com/moby/moby/client"
)

// ComposeRunnerLauncher 是 server 启动一次性 runner 的最小 Docker 边界。
// 它不暴露通用容器创建能力，调用方无法提供命令、环境或挂载。
type ComposeRunnerLauncher interface {
	Launch(context.Context, RunnerInput) error
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

// RunnerProgressLogMarker 是 runner 阶段日志使用的稳定标记；标记后仅允许固定枚举值。
const RunnerProgressLogMarker = "GRAFT_UPDATE_STAGE:"

// RunnerOperationProgress 将 runner operation identity 与一个受限阶段关联。
type RunnerOperationProgress struct {
	OperationID string
	Progress    RunnerProgress
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

// Close 释放 Docker API client 持有的连接。
func (l *dockerComposeRunnerLauncher) Close() error {
	if l == nil || l.client == nil {
		return nil
	}
	return l.client.Close()
}

// NewDockerComposeRunnerLauncher 创建只可执行官方 Compose runner 的 Docker socket launcher。
func NewDockerComposeRunnerLauncher() (ComposeRunnerLauncher, error) {
	client, err := mobyclient.New(mobyclient.WithHost("unix:///var/run/docker.sock"))
	if err != nil {
		return nil, fmt.Errorf("create compose runner docker client: %w", err)
	}
	return &dockerComposeRunnerLauncher{client: client}, nil
}

//nolint:cyclop // 启动流程中的每条失败分支对应一个独立的资源与错误边界。
func (l *dockerComposeRunnerLauncher) Launch(ctx context.Context, input RunnerInput) error {
	if l == nil || l.client == nil {
		return errors.New("compose runner launcher is unavailable")
	}
	if err := ValidateRunnerInput(input); err != nil {
		return fmt.Errorf("validate compose runner launch: %w", err)
	}
	encodedInput, err := encodeRunnerInput(input)
	if err != nil {
		return err
	}
	pulled, err := l.client.ImagePull(ctx, input.Preflight.RunnerReference, mobyclient.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull digest-pinned compose runner: %w", err)
	}
	if _, err := io.Copy(io.Discard, pulled); err != nil {
		_ = pulled.Close()
		return fmt.Errorf("read compose runner pull result: %w", err)
	}
	if err := pulled.Close(); err != nil {
		return fmt.Errorf("close compose runner pull result: %w", err)
	}
	configuration, host := composeRunnerContainerConfig(input, encodedInput)
	options := mobyclient.ContainerCreateOptions{Config: &configuration, HostConfig: &host, NetworkingConfig: &network.NetworkingConfig{}}
	options.Name = composeRunnerContainerName(input.OperationID)
	created, err := l.client.ContainerCreate(ctx, options)
	if err != nil {
		return fmt.Errorf("create compose runner: %w", err)
	}
	if _, err := l.client.ContainerStart(ctx, created.ID, mobyclient.ContainerStartOptions{}); err != nil {
		if cleanupErr := l.removeUnstartedRunner(ctx, created.ID); cleanupErr != nil {
			return fmt.Errorf("start compose runner: %w; clean up unstarted runner: %v", err, cleanupErr)
		}
		return fmt.Errorf("start compose runner: %w", err)
	}
	return nil
}

func (l *dockerComposeRunnerLauncher) removeUnstartedRunner(ctx context.Context, id string) error {
	inspected, err := l.client.ContainerInspect(ctx, id, mobyclient.ContainerInspectOptions{})
	if err != nil {
		return err
	}
	if inspected.Container.State == nil {
		return errors.New("inspect compose runner returned no state")
	}
	if inspected.Container.State.Running {
		return nil
	}
	_, err = l.client.ContainerRemove(ctx, id, mobyclient.ContainerRemoveOptions{})
	return err
}

func composeRunnerContainerName(operationID string) string { return "graft-update-" + operationID }

//nolint:cyclop // 读取与解码分别对应 Docker 日志和回执协议的失败边界。
func (l *dockerComposeRunnerLauncher) ReadRunnerReceipts(ctx context.Context) ([]RunnerReceipt, error) {
	if l == nil || l.client == nil {
		return nil, errors.New("compose runner receipt reader is unavailable")
	}
	filters := make(mobyclient.Filters).Add("label", "io.graft.update.protocol="+runnerProtocol).Add("label", "io.graft.update.protocol="+legacyRunnerProtocol)
	result, err := l.client.ContainerList(ctx, mobyclient.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return nil, fmt.Errorf("list retained compose runners: %w", err)
	}
	var receipts []RunnerReceipt
	for _, item := range result.Items {
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
			if receipt, ok := parseRunnerReceiptLog(line); ok && receipt.OperationID == item.Labels["io.graft.update.operation"] && runnerProtocolMatchesVersion(item.Labels["io.graft.update.protocol"], receipt.ProtocolVersion) {
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

func (l *dockerComposeRunnerLauncher) readContainerProgress(ctx context.Context, item containertypes.Summary) (RunnerOperationProgress, bool, error) {
	operationID := item.Labels["io.graft.update.operation"]
	if !runnerOperationID.MatchString(operationID) {
		return RunnerOperationProgress{}, false, nil
	}
	logs, err := l.client.ContainerLogs(ctx, item.ID, mobyclient.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return RunnerOperationProgress{}, false, fmt.Errorf("read retained compose runner logs: %w", err)
	}
	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, logs); err != nil {
		_ = logs.Close()
		return RunnerOperationProgress{}, false, fmt.Errorf("decode retained compose runner logs: %w", err)
	}
	_ = logs.Close()
	progress, ok := latestRunnerProgress(stdout.String() + "\n" + stderr.String())
	if !ok {
		return RunnerOperationProgress{}, false, nil
	}
	return RunnerOperationProgress{OperationID: operationID, Progress: progress}, true, nil
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
	filters := make(mobyclient.Filters).Add("label", "io.graft.update.operation="+operationID).Add("label", "io.graft.update.protocol="+runnerProtocol).Add("label", "io.graft.update.protocol="+legacyRunnerProtocol)
	result, err := l.client.ContainerList(ctx, mobyclient.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return fmt.Errorf("list settled compose runners: %w", err)
	}
	for _, item := range result.Items {
		if item.Labels["io.graft.update.operation"] != operationID || !supportedRunnerProtocol(item.Labels["io.graft.update.protocol"]) {
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

func composeRunnerContainerConfig(input RunnerInput, inputPath string) (containertypes.Config, containertypes.HostConfig) {
	root := input.Preflight.ComposeRoot
	socket := input.Preflight.DockerSocket
	groups := []string{}
	if stat, err := os.Stat(socket); err == nil {
		if details, ok := stat.Sys().(*syscall.Stat_t); ok {
			groups = append(groups, strconv.FormatUint(uint64(details.Gid), 10))
		}
	}
	return containertypes.Config{Image: input.Preflight.RunnerReference, User: "0:0", Env: []string{"GRAFT_UPDATE_RUNNER_INPUT_B64=" + inputPath}, Labels: map[string]string{
		"io.graft.update.operation": input.OperationID,
		"io.graft.update.protocol":  runnerProtocol,
	}}, containertypes.HostConfig{AutoRemove: false, Binds: []string{root + ":" + root + ":rw", socket + ":" + socket + ":rw"}, GroupAdd: groups, NetworkMode: "none", ReadonlyRootfs: true, CapDrop: []string{"ALL"}, SecurityOpt: []string{"no-new-privileges:true"}}
}
