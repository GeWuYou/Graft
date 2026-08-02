package update

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"testing"

	containertypes "github.com/moby/moby/api/types/container"
	mobyclient "github.com/moby/moby/client"
)

func TestReadRunnerReceiptsRetainsValidAndInvalidContainersUntilExplicitCleanup(t *testing.T) {
	valid := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: "operation-1", Succeeded: true}
	legacy := RunnerReceipt{ProtocolVersion: legacyRunnerProtocolVersion, OperationID: "operation-legacy", Succeeded: true}
	client := &receiptDockerClient{items: []containertypes.Summary{{ID: "valid", Labels: runnerLabels("operation-1")}, {ID: "legacy", Labels: runnerLabelsForProtocol("operation-legacy", legacyRunnerProtocol)}, {ID: "pending", Labels: runnerLabels("operation-2")}}, logs: map[string][]byte{
		"valid":   multiplexRunnerLog(t, RunnerReceiptLogMarker, valid),
		"legacy":  multiplexRunnerLog(t, RunnerReceiptLogMarker, legacy),
		"pending": multiplexRunnerLog(t, RunnerReceiptLogMarker, RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: "operation-3"}),
	}}
	launcher := &dockerComposeRunnerLauncher{client: client}

	receipts, err := launcher.ReadRunnerReceipts(context.Background())
	if err != nil {
		t.Fatalf("read runner receipts: %v", err)
	}
	if len(receipts) != 2 || receipts[0] != valid || receipts[1] != legacy {
		t.Fatalf("receipts = %#v, want %#v", receipts, []RunnerReceipt{valid, legacy})
	}
	if len(client.removed) != 0 {
		t.Fatalf("removed containers after read = %#v, want none", client.removed)
	}
	if err := launcher.RemoveRunner(context.Background(), valid.OperationID); err != nil {
		t.Fatalf("remove settled runner: %v", err)
	}
	if len(client.removed) != 1 || client.removed[0] != "valid" {
		t.Fatalf("removed containers after cleanup = %#v, want [valid]", client.removed)
	}
}

func TestReadRunnerProgressKeepsOnlyBoundedLatestMarker(t *testing.T) {
	client := &receiptDockerClient{items: []containertypes.Summary{{ID: "runner", Labels: runnerLabels("operation-1")}}, logs: map[string][]byte{
		"runner": multiplexRunnerLines("ordinary\n" + RunnerProgressLogMarker + "BACKING_UP\n" + RunnerProgressLogMarker + "PULLING\n" + RunnerProgressLogMarker + "unsafe detail\n"),
	}}
	progress, err := (&dockerComposeRunnerLauncher{client: client}).ReadRunnerProgress(context.Background())
	if err != nil {
		t.Fatalf("read runner progress: %v", err)
	}
	if len(progress) != 1 || progress[0] != (RunnerOperationProgress{OperationID: "operation-1", Progress: RunnerProgressPulling}) {
		t.Fatalf("progress = %#v", progress)
	}
	if len(client.logOpts) != 1 || client.logOpts[0].Tail != retainedRunnerLogTail {
		t.Fatalf("runner log options = %#v", client.logOpts)
	}
}

func TestRunnerStateVolumeNameRejectsInvalidConfiguredValue(t *testing.T) {
	t.Setenv("GRAFT_UPDATE_STATE_VOLUME", "invalid/name")
	if _, err := runnerStateVolumeName(); err == nil {
		t.Fatal("expected invalid state volume name rejection")
	}
}

func runnerLabels(operationID string) map[string]string {
	return runnerLabelsForProtocol(operationID, runnerProtocol)
}

func runnerLabelsForProtocol(operationID string, protocol string) map[string]string {
	return map[string]string{"io.graft.update.operation": operationID, "io.graft.update.protocol": protocol}
}

func multiplexRunnerLog(t *testing.T, marker string, receipt RunnerReceipt) []byte {
	t.Helper()
	line := "ordinary runner log\n"
	if marker != "" {
		contents, err := json.Marshal(receipt)
		if err != nil {
			t.Fatalf("marshal receipt: %v", err)
		}
		line = marker + base64.RawStdEncoding.EncodeToString(contents) + "\n"
	}
	payload := []byte(line)
	header := make([]byte, 8)
	header[0] = 1
	// #nosec G115 -- test payloads are bounded by the in-memory fixture.
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	return append(header, payload...)
}

func multiplexRunnerLines(line string) []byte {
	payload := []byte(line)
	header := make([]byte, 8)
	header[0] = 1
	// #nosec G115 -- test payloads are bounded by the in-memory fixture.
	binary.BigEndian.PutUint32(header[4:], uint32(len(payload)))
	return append(header, payload...)
}

type receiptDockerClient struct {
	items   []containertypes.Summary
	logs    map[string][]byte
	removed []string
	logOpts []mobyclient.ContainerLogsOptions
}

func (c *receiptDockerClient) ImagePull(context.Context, string, mobyclient.ImagePullOptions) (mobyclient.ImagePullResponse, error) {
	return nil, errors.New("not implemented")
}
func (c *receiptDockerClient) ContainerCreate(context.Context, mobyclient.ContainerCreateOptions) (mobyclient.ContainerCreateResult, error) {
	return mobyclient.ContainerCreateResult{}, errors.New("not implemented")
}
func (c *receiptDockerClient) ContainerStart(context.Context, string, mobyclient.ContainerStartOptions) (mobyclient.ContainerStartResult, error) {
	return mobyclient.ContainerStartResult{}, errors.New("not implemented")
}
func (c *receiptDockerClient) ContainerInspect(context.Context, string, mobyclient.ContainerInspectOptions) (mobyclient.ContainerInspectResult, error) {
	return mobyclient.ContainerInspectResult{}, errors.New("not implemented")
}
func (c *receiptDockerClient) ContainerRemove(_ context.Context, id string, _ mobyclient.ContainerRemoveOptions) (mobyclient.ContainerRemoveResult, error) {
	c.removed = append(c.removed, id)
	return mobyclient.ContainerRemoveResult{}, nil
}
func (c *receiptDockerClient) ContainerList(context.Context, mobyclient.ContainerListOptions) (mobyclient.ContainerListResult, error) {
	return mobyclient.ContainerListResult{Items: c.items}, nil
}
func (c *receiptDockerClient) ContainerLogs(_ context.Context, id string, options mobyclient.ContainerLogsOptions) (mobyclient.ContainerLogsResult, error) {
	c.logOpts = append(c.logOpts, options)
	return io.NopCloser(bytes.NewReader(c.logs[id])), nil
}
func (c *receiptDockerClient) Close() error { return nil }

var _ dockerRunnerClient = (*receiptDockerClient)(nil)
