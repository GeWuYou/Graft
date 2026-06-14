// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"unicode"
)

const (
	runtimeNameDocker = "docker"

	containerActionStart   = "start"
	containerActionStop    = "stop"
	containerActionRestart = "restart"

	actionResultCompleted = "completed"
	actionResultUnchanged = "unchanged"
)

var (
	errRuntimeDisabled             = errors.New("container runtime disabled")
	errRuntimeSocketMissing        = errors.New("container runtime socket missing")
	errRuntimePermissionDenied     = errors.New("container runtime permission denied")
	errRuntimeDaemonUnavailable    = errors.New("container runtime daemon unavailable")
	errContainerNotFound           = errors.New("container not found")
	errInvalidRef                  = errors.New("invalid container reference")
	errInvalidContainerState       = errors.New("invalid container state")
	errLogsTooLarge                = errors.New("container logs tail exceeds limit")
	errContainerRuntimeTimeout     = errors.New("container runtime timeout")
	errDangerousActionsDisabled    = errors.New("dangerous container actions disabled")
	errUnsupportedContainerRuntime = errors.New("unsupported container runtime")
)

// Runtime is the module-owned boundary between API/service code and a concrete container runtime adapter.
type Runtime interface {
	Info(ctx context.Context) (RuntimeInfo, error)
	List(ctx context.Context, query ListQuery) ([]Summary, error)
	Detail(ctx context.Context, id Ref) (Detail, error)
	Logs(ctx context.Context, id Ref, query LogQuery) (Logs, error)
	Start(ctx context.Context, id Ref) (ActionResult, error)
	Stop(ctx context.Context, id Ref) (ActionResult, error)
	Restart(ctx context.Context, id Ref) (ActionResult, error)
	Close() error
}

// Ref is a validated Docker-compatible container id or name.
type Ref struct {
	Value string
}

// ListQuery carries future list filters while keeping the runtime signature stable.
type ListQuery struct{}

// LogQuery describes bounded container log retrieval options.
type LogQuery struct {
	Tail       int
	Since      string
	Timestamps bool
	Stdout     bool
	Stderr     bool
}

// RuntimeInfo is sanitized runtime metadata exposed by the container API.
type RuntimeInfo struct {
	Runtime           string
	Status            string
	Endpoint          string
	APIVersion        string
	ServerVersion     string
	OperatingSystem   string
	Architecture      string
	ContainersTotal   int
	ContainersRunning int
}

// Summary is a sanitized row for container list responses.
type Summary struct {
	ID            string
	Names         []string
	Image         string
	ImageID       string
	Labels        map[string]string
	Ports         []Port
	RestartPolicy string
	Runtime       string
	CreatedAt     string
	StartedAt     string
	State         string
	Status        string
}

// Detail is a sanitized container inspect view.
type Detail struct {
	Summary
	Command          []string
	Entrypoint       []string
	Mounts           []Mount
	Networks         []Network
	RuntimeInfo      RuntimeInfo
	InspectUpdatedAt string
	WorkingDir       string
}

// Port describes one exposed or published container port.
type Port struct {
	IP          string
	PrivatePort int
	PublicPort  *int
	Type        string
}

// Mount describes one mounted path without exposing raw inspect payloads.
type Mount struct {
	Type        string
	Name        string
	Source      string
	Destination string
	Mode        string
	ReadOnly    bool
}

// Network describes one network attachment.
type Network struct {
	Name       string
	NetworkID  string
	EndpointID string
	Gateway    string
	IPAddress  string
	MacAddress string
}

// Logs contains bounded log lines and the effective log options.
type Logs struct {
	ID         string
	Name       string
	Runtime    string
	Lines      []string
	Tail       int
	Since      string
	Timestamps bool
	Stdout     bool
	Stderr     bool
	Truncated  bool
}

// ActionResult describes a start, stop, or restart result for audit and API responses.
type ActionResult struct {
	ID           string
	Name         string
	Image        string
	Action       string
	Result       string
	Runtime      string
	StatusBefore string
	StatusAfter  string
}

func parseRef(raw string) (Ref, error) {
	unescaped, err := url.PathUnescape(raw)
	if err != nil {
		return Ref{}, errInvalidRef
	}
	value := strings.TrimSpace(unescaped)
	if value == "" || strings.Contains(value, "/") {
		return Ref{}, errInvalidRef
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return Ref{}, errInvalidRef
		}
	}
	return Ref{Value: value}, nil
}
