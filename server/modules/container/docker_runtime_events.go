package container

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	dockerevents "github.com/moby/moby/api/types/events"
	mobyclient "github.com/moby/moby/client"

	containercontract "graft/server/modules/container/contract"
)

// dockerRuntimeEventFilters 返回用于筛选容器运行时事件的 Docker 过滤条件。
func dockerRuntimeEventFilters() mobyclient.Filters {
	filters := mobyclient.Filters{}
	return filters.Add("type", "container")
}

// consumeDockerRuntimeEvents 从 Docker 事件流中消费一条消息或终止信号，并在匹配时回调输出事件候选。
func consumeDockerRuntimeEvents(
	ctx context.Context,
	result *mobyclient.EventsResult,
	emit func(RuntimeEventCandidate) error,
) (bool, error) {
	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case message, ok := <-result.Messages:
		if !ok {
			result.Messages = nil
			return result.Err == nil, nil
		}
		candidate, matched := dockerRuntimeEventCandidate(message)
		if !matched {
			return false, nil
		}
		return false, emit(candidate)
	case err, ok := <-result.Err:
		if !ok {
			result.Err = nil
			return result.Messages == nil, nil
		}
		if err == io.EOF {
			return true, nil
		}
		return true, mapDockerError(err)
	}
}

// dockerRuntimeEventCandidate 将 Docker 事件转换为运行时事件候选项。
func dockerRuntimeEventCandidate(message dockerevents.Message) (RuntimeEventCandidate, bool) {
	resourceID := strings.TrimSpace(message.Actor.ID)
	if resourceID == "" {
		return RuntimeEventCandidate{}, false
	}

	eventType, attributes, ok := dockerCanonicalRuntimeEvent(message)
	if !ok {
		return RuntimeEventCandidate{}, false
	}

	occurredAt := time.Unix(message.Time, 0).UTC()
	if message.TimeNano > 0 {
		occurredAt = time.Unix(0, message.TimeNano).UTC()
	}

	return RuntimeEventCandidate{
		ResourceID: resourceID,
		EventType:  eventType,
		OccurredAt: occurredAt,
		Attributes: attributes,
	}, true
}

// dockerCanonicalRuntimeEvent 将 Docker 事件归一化为模块事件类型和属性。
func dockerCanonicalRuntimeEvent(
	message dockerevents.Message,
) (containercontract.RuntimeEventType, map[string]string, bool) {
	action, actionDetail := normalizeDockerRuntimeEventAction(message.Action)
	attrs := dockerRuntimeEventBaseAttributes(message)
	return dockerRuntimeEventFromAction(action, actionDetail, message, attrs)
}

// dockerRuntimeEventBaseAttributes 提取 Docker 运行时事件的基础属性。
func dockerRuntimeEventBaseAttributes(message dockerevents.Message) map[string]string {
	attrs := make(map[string]string)
	addRuntimeEventAttribute(attrs, "name", message.Actor.Attributes["name"])
	return attrs
}

// normalizeDockerRuntimeEventAction 规范化容器事件动作并拆分主动作与详情。
func normalizeDockerRuntimeEventAction(action dockerevents.Action) (string, string) {
	normalized := strings.ToLower(strings.TrimSpace(string(action)))
	if normalized == "" {
		return "", ""
	}
	prefix, detail, found := strings.Cut(normalized, ":")
	if !found {
		return normalized, ""
	}
	return strings.TrimSpace(prefix), strings.TrimSpace(detail)
}

// dockerRuntimeEventFromAction 将 Docker 事件动作转换为运行时事件类型，并补充相关属性。
func dockerRuntimeEventFromAction(
	action string,
	actionDetail string,
	message dockerevents.Message,
	attrs map[string]string,
) (containercontract.RuntimeEventType, map[string]string, bool) {
	switch action {
	case "create":
		return containercontract.RuntimeEventTypeContainerCreated, attrs, true
	case "start":
		return containercontract.RuntimeEventTypeContainerStarted, attrs, true
	case "restart":
		return containercontract.RuntimeEventTypeContainerRestarted, attrs, true
	case "destroy", "remove":
		return containercontract.RuntimeEventTypeContainerRemoved, attrs, true
	case "oom":
		return containercontract.RuntimeEventTypeContainerOOMKilled, attrs, true
	case "health_status":
		addRuntimeEventAttribute(
			attrs,
			"health_status",
			firstNonEmpty(message.Actor.Attributes["health_status"], actionDetail),
		)
		return containercontract.RuntimeEventTypeContainerHealthStatusChanged, attrs, true
	case "exec_create", "exec_start":
		addRuntimeEventAttribute(attrs, "exec_id", message.Actor.Attributes["execID"])
		addRuntimeEventAttribute(
			attrs,
			"exec_command",
			firstNonEmpty(message.Actor.Attributes["execCommand"], actionDetail),
		)
		return containercontract.RuntimeEventTypeContainerExecStarted, attrs, true
	case "exec_die":
		addRuntimeEventAttribute(attrs, "exec_id", message.Actor.Attributes["execID"])
		addRuntimeEventAttribute(attrs, "exec_exit_code", message.Actor.Attributes["exitCode"])
		return containercontract.RuntimeEventTypeContainerExecFinished, attrs, true
	default:
		return dockerStoppedRuntimeEvent(action, message, attrs)
	}
}

// dockerStoppedRuntimeEvent 将 stop、die 和 kill 事件归一化为容器停止事件，并补充退出码属性。
func dockerStoppedRuntimeEvent(
	action string,
	message dockerevents.Message,
	attrs map[string]string,
) (containercontract.RuntimeEventType, map[string]string, bool) {
	switch action {
	case "stop", "die", "kill":
		addRuntimeEventAttribute(attrs, "exit_code", message.Actor.Attributes["exitCode"])
		return containercontract.RuntimeEventTypeContainerStopped, attrs, true
	default:
		return "", nil, false
	}
}

// addRuntimeEventAttribute 向属性映射中写入非空且已修剪的键值对。
func addRuntimeEventAttribute(attributes map[string]string, key string, value string) {
	if attributes == nil {
		return
	}
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return
	}
	attributes[key] = value
}

// mapDockerShellError 将 Docker Shell 执行错误映射为特定领域的错误类型。
func mapDockerShellError(err error) error {
	if err == nil {
		return nil
	}
	mapped := mapDockerError(err)
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case errors.Is(mapped, errContainerNotFound):
		return errContainerNotFound
	case strings.Contains(message, "executable file not found"),
		strings.Contains(message, "not found in $path"):
		return errShellCommandNotFound
	default:
		if mapped != nil {
			return mapped
		}
		return errShellSessionFailed
	}
}
