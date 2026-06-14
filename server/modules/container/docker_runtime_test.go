// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

func TestReadDockerLogLinesDoesNotReportExactTailAsTruncated(t *testing.T) {
	t.Parallel()

	lines, truncated, err := readDockerLogLines(dockerLogStream(t, "one\n", "two\n"), 2)
	if err != nil {
		t.Fatalf("read log lines: %v", err)
	}
	if truncated {
		t.Fatalf("expected exactly tail lines to avoid truncation")
	}
	if !reflect.DeepEqual(lines, []string{"one", "two"}) {
		t.Fatalf("unexpected lines %#v", lines)
	}
}

func TestReadDockerLogLinesTruncatesOnlyAfterDiscardingLines(t *testing.T) {
	t.Parallel()

	lines, truncated, err := readDockerLogLines(dockerLogStream(t, "one\n", "two\n", "three\n"), 2)
	if err != nil {
		t.Fatalf("read log lines: %v", err)
	}
	if !truncated {
		t.Fatalf("expected more than tail lines to report truncation")
	}
	if !reflect.DeepEqual(lines, []string{"two", "three"}) {
		t.Fatalf("unexpected lines %#v", lines)
	}
}

func TestReadDockerLogLinesAvoidsUserSizedPreallocation(t *testing.T) {
	t.Parallel()

	const excessiveTail = int(^uint(0) >> 1)
	lines, truncated, err := readDockerLogLines(dockerLogStream(t, "one\n"), excessiveTail)
	if err != nil {
		t.Fatalf("read log lines: %v", err)
	}
	if truncated {
		t.Fatalf("expected one line to avoid truncation")
	}
	if !reflect.DeepEqual(lines, []string{"one"}) {
		t.Fatalf("unexpected lines %#v", lines)
	}
}

func TestDockerRuntimeLogsAvoidsRuntimeInfoCall(t *testing.T) {
	t.Parallel()

	client := &countingDockerClient{
		logReader: dockerLogReadCloser(t, "one\n", "two\n"),
		inspect: container.InspectResponse{
			ContainerJSONBase: &container.ContainerJSONBase{
				ID:   "full-id",
				Name: "/web",
			},
		},
	}
	runtime := &DockerRuntime{client: client, endpoint: "unix:///var/run/docker.sock"}

	logs, err := runtime.Logs(context.Background(), Ref{Value: "web"}, LogQuery{
		Tail:   2,
		Stdout: true,
	})
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if logs.ID != "full-id" || logs.Name != "web" {
		t.Fatalf("unexpected log metadata %#v", logs)
	}
	if calls := client.infoCalls.Load(); calls != 0 {
		t.Fatalf("expected logs to avoid Info calls, got %d", calls)
	}
	if calls := client.inspectCalls.Load(); calls != 1 {
		t.Fatalf("expected one inspect call for log metadata, got %d", calls)
	}
	if calls := client.logCalls.Load(); calls != 1 {
		t.Fatalf("expected one log call, got %d", calls)
	}
}

func TestDockerRuntimeLogsReturnsInvalidLogQueryError(t *testing.T) {
	t.Parallel()

	runtime := &DockerRuntime{client: &countingDockerClient{}, endpoint: "unix:///var/run/docker.sock"}

	_, err := runtime.Logs(context.Background(), Ref{Value: "web"}, LogQuery{
		Tail:   1,
		Since:  "not-a-time",
		Stdout: true,
	})
	if !errors.Is(err, errInvalidLogQuery) {
		t.Fatalf("expected invalid log query error, got %v", err)
	}
}

func dockerLogStream(t *testing.T, chunks ...string) io.Reader {
	t.Helper()

	var output bytes.Buffer
	writer := stdcopy.NewStdWriter(&output, stdcopy.Stdout)
	for _, chunk := range chunks {
		if _, err := writer.Write([]byte(chunk)); err != nil {
			t.Fatalf("write log chunk: %v", err)
		}
	}
	return bytes.NewReader(output.Bytes())
}

func dockerLogReadCloser(t *testing.T, chunks ...string) io.ReadCloser {
	t.Helper()

	return io.NopCloser(dockerLogStream(t, chunks...))
}

type countingDockerClient struct {
	infoCalls    atomic.Int64
	inspectCalls atomic.Int64
	logCalls     atomic.Int64
	logReader    io.ReadCloser
	inspect      container.InspectResponse
}

func (c *countingDockerClient) Info(context.Context) (systemInfo, error) {
	c.infoCalls.Add(1)
	return dockerClientSystemInfo{}, nil
}

func (c *countingDockerClient) ContainerList(context.Context, container.ListOptions) ([]container.Summary, error) {
	return nil, nil
}

func (c *countingDockerClient) ContainerInspect(context.Context, string) (container.InspectResponse, error) {
	c.inspectCalls.Add(1)
	return c.inspect, nil
}

func (c *countingDockerClient) ContainerLogs(context.Context, string, container.LogsOptions) (io.ReadCloser, error) {
	c.logCalls.Add(1)
	if c.logReader == nil {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}
	return c.logReader, nil
}

func (c *countingDockerClient) ContainerStart(context.Context, string, container.StartOptions) error {
	return nil
}

func (c *countingDockerClient) ContainerStop(context.Context, string, container.StopOptions) error {
	return nil
}

func (c *countingDockerClient) ContainerRestart(context.Context, string, container.StopOptions) error {
	return nil
}

func (c *countingDockerClient) Close() error {
	return nil
}
