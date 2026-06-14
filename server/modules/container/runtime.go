// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"errors"
	"net/url"
	"slices"
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

	defaultContainerListLimit     = 20
	maxContainerListLimit         = 100
	containerListKeywordMaxLength = 128
	containerShortIDLength        = 12

	containerHealthHealthy     = "healthy"
	containerHealthUnhealthy   = "unhealthy"
	containerHealthStarting    = "starting"
	containerHealthNone        = "none"
	containerHealthUnavailable = "unavailable"

	containerStatsUnavailableReason = "stats_not_collected"

	composeProjectLabel = "com.docker.compose.project"
	composeServiceLabel = "com.docker.compose.service"
)

var (
	errRuntimeDisabled             = errors.New("container runtime disabled")
	errRuntimeSocketMissing        = errors.New("container runtime socket missing")
	errRuntimePermissionDenied     = errors.New("container runtime permission denied")
	errRuntimeDaemonUnavailable    = errors.New("container runtime daemon unavailable")
	errContainerNotFound           = errors.New("container not found")
	errInvalidRef                  = errors.New("invalid container reference")
	errInvalidListQuery            = errors.New("invalid container list query")
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

// ListQuery describes bounded list pagination and low-cost runtime filters.
type ListQuery struct {
	Limit   int
	Offset  int
	Keyword string
	State   string
	Health  string
}

// ListResult is the service-owned list response model.
type ListResult struct {
	Runtime RuntimeInfo
	Items   []Summary
	Total   int
	Limit   int
	Offset  int
	Summary ListSummary
}

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

// ListSummary carries aggregate counts across the filtered list.
type ListSummary struct {
	Total             int
	Running           int
	Stopped           int
	Error             int
	Healthy           int
	Unhealthy         int
	HealthUnavailable int
}

// ResourceSummary is nullable-by-field runtime stats metadata for list rows.
type ResourceSummary struct {
	Available         bool
	UnavailableReason string
	CPUPercent        *float64
	MemoryUsageBytes  *int64
	MemoryLimitBytes  *int64
	MemoryPercent     *float64
}

// Summary is a sanitized row for container list responses.
type Summary struct {
	ID             string
	ShortID        string
	Name           string
	Names          []string
	Image          string
	ImageID        string
	Labels         map[string]string
	Ports          []Port
	PrimaryIP      string
	Networks       []Network
	NetworkSummary string
	Resource       ResourceSummary
	RestartCount   *int
	RestartPolicy  string
	Runtime        string
	CreatedAt      string
	StartedAt      string
	State          string
	Status         string
	Health         string
	ComposeProject string
	ComposeService string
	CanStart       bool
	CanStop        bool
	CanRestart     bool
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
	MessageKey   string
	Message      string
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

func normalizeListQuery(query ListQuery) (ListQuery, error) {
	if query.Limit == 0 {
		query.Limit = defaultContainerListLimit
	}
	if query.Limit < 1 || query.Limit > maxContainerListLimit {
		return ListQuery{}, errInvalidListQuery
	}
	if query.Offset < 0 {
		return ListQuery{}, errInvalidListQuery
	}
	query.Keyword = strings.TrimSpace(query.Keyword)
	if len(query.Keyword) > containerListKeywordMaxLength {
		return ListQuery{}, errInvalidListQuery
	}
	query.State = strings.TrimSpace(strings.ToLower(query.State))
	if query.State != "" && !isValidContainerState(query.State) {
		return ListQuery{}, errInvalidListQuery
	}
	query.Health = strings.TrimSpace(strings.ToLower(query.Health))
	if query.Health != "" && !isValidContainerHealth(query.Health) {
		return ListQuery{}, errInvalidListQuery
	}
	return query, nil
}

func isValidContainerState(state string) bool {
	return slices.Contains([]string{"created", "running", "paused", "restarting", "removing", "exited", "dead", "unknown"}, state)
}

func isValidContainerHealth(health string) bool {
	return slices.Contains([]string{
		containerHealthHealthy,
		containerHealthUnhealthy,
		containerHealthStarting,
		containerHealthNone,
		containerHealthUnavailable,
	}, health)
}
