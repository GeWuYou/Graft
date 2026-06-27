package contract

// RuntimeEventSeverity is the canonical severity contract for container runtime events.
type RuntimeEventSeverity string

// String returns the wire-format severity value.
func (s RuntimeEventSeverity) String() string {
	return string(s)
}

const (
	// RuntimeEventSeverityInfo marks informational lifecycle/runtime facts.
	RuntimeEventSeverityInfo RuntimeEventSeverity = "info"
	// RuntimeEventSeverityWarning marks degraded or attention-required lifecycle facts.
	RuntimeEventSeverityWarning RuntimeEventSeverity = "warning"
	// RuntimeEventSeverityError marks severe runtime failure facts.
	RuntimeEventSeverityError RuntimeEventSeverity = "error"
)

// RuntimeEventType is the canonical container runtime event type contract.
type RuntimeEventType string

// String returns the wire-format event type value.
func (t RuntimeEventType) String() string {
	return string(t)
}

const (
	// RuntimeEventTypeContainerCreated identifies container creation.
	RuntimeEventTypeContainerCreated RuntimeEventType = "container.created"
	// RuntimeEventTypeContainerStarted identifies container start.
	RuntimeEventTypeContainerStarted RuntimeEventType = "container.started"
	// RuntimeEventTypeContainerRestarted identifies container restart.
	RuntimeEventTypeContainerRestarted RuntimeEventType = "container.restarted"
	// RuntimeEventTypeContainerStopped identifies container stop/exit.
	RuntimeEventTypeContainerStopped RuntimeEventType = "container.stopped"
	// RuntimeEventTypeContainerRemoved identifies container removal.
	RuntimeEventTypeContainerRemoved RuntimeEventType = "container.removed"
	// RuntimeEventTypeContainerOOMKilled identifies OOM kill.
	RuntimeEventTypeContainerOOMKilled RuntimeEventType = "container.oom_killed"
	// RuntimeEventTypeContainerHealthStatusChanged identifies health status transitions.
	RuntimeEventTypeContainerHealthStatusChanged RuntimeEventType = "container.health_status_changed"
	// RuntimeEventTypeContainerExecStarted identifies exec session start.
	RuntimeEventTypeContainerExecStarted RuntimeEventType = "container.exec_started"
	// RuntimeEventTypeContainerExecFinished identifies exec session finish.
	RuntimeEventTypeContainerExecFinished RuntimeEventType = "container.exec_finished"
)
