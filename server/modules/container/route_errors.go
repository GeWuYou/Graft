package container

import (
	"errors"
	"net/http"

	containercontract "graft/server/modules/container/contract"
)

// statusForError 将给定的错误映射到 HTTP 状态码。
// 无效的引用或查询映射到 400，权限或功能禁用问题映射到 403，
// 容器或资源不存在映射到 404，无效的状态或冲突映射到 409，
// 其他错误映射到 500。
func statusForError(err error) int {
	switch {
	case errors.Is(err, errInvalidRef), errors.Is(err, errInvalidListQuery), errors.Is(err, errInvalidBatchAction), errors.Is(err, errLogsTooLarge), errors.Is(err, errInvalidLogQuery), errors.Is(err, errMountUsageUnsupported), errors.Is(err, errShellTicketInvalid), errors.Is(err, errShellCommandNotFound), errors.Is(err, errShellInvalidSize):
		return http.StatusBadRequest
	case errors.Is(err, errRuntimeDisabled), errors.Is(err, errDangerousActionsDisabled), errors.Is(err, errRuntimePermissionDenied), errors.Is(err, errShellDisabled), errors.Is(err, errShellForbidden), errors.Is(err, errShellOriginDenied):
		return http.StatusForbidden
	case errors.Is(err, errContainerNotFound), errors.Is(err, errContainerMountNotFound):
		return http.StatusNotFound
	case errors.Is(err, errInvalidContainerState), errors.Is(err, errShellTicketExpired), errors.Is(err, errShellTicketUsed), errors.Is(err, errContainerNotRunning):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func messageKeyForError(err error) containercontract.MessageKey {
	for _, rule := range containerErrorMessageRules {
		if errors.Is(err, rule.err) {
			return rule.key
		}
	}
	return containercontract.ContainerRuntimeUnavailable
}

func fallbackMessageForError(err error) string {
	return messageKeyForError(err).String()
}

var containerErrorMessageRules = []struct {
	err error
	key containercontract.MessageKey
}{
	{err: errRuntimeDisabled, key: containercontract.ContainerRuntimeDisabled},
	{err: errRuntimeSocketMissing, key: containercontract.ContainerRuntimeSocketMissing},
	{err: errRuntimePermissionDenied, key: containercontract.ContainerRuntimePermissionDenied},
	{err: errRuntimeDaemonUnavailable, key: containercontract.ContainerRuntimeUnavailable},
	{err: errContainerNotFound, key: containercontract.ContainerNotFound},
	{err: errInvalidRef, key: containercontract.ContainerInvalidRef},
	{err: errInvalidListQuery, key: containercontract.ContainerInvalidListQuery},
	{err: errInvalidBatchAction, key: containercontract.ContainerInvalidBatchAction},
	{err: errInvalidContainerState, key: containercontract.ContainerInvalidState},
	{err: errLogsTooLarge, key: containercontract.ContainerLogsTooLarge},
	{err: errInvalidLogQuery, key: containercontract.ContainerInvalidLogQuery},
	{err: errShellDisabled, key: containercontract.ContainerShellDisabled},
	{err: errShellForbidden, key: containercontract.ContainerShellForbidden},
	{err: errShellTicketInvalid, key: containercontract.ContainerShellTicketInvalid},
	{err: errShellTicketExpired, key: containercontract.ContainerShellTicketExpired},
	{err: errShellTicketUsed, key: containercontract.ContainerShellTicketUsed},
	{err: errShellOriginDenied, key: containercontract.ContainerShellOriginDenied},
	{err: errContainerNotRunning, key: containercontract.ContainerShellContainerNotRunning},
	{err: errShellCommandNotFound, key: containercontract.ContainerShellCommandNotFound},
	{err: errShellInvalidSize, key: containercontract.ContainerShellInvalidSize},
	{err: errShellSessionFailed, key: containercontract.ContainerShellSessionFailed},
	{err: errContainerRuntimeTimeout, key: containercontract.ContainerTimeout},
	{err: errDangerousActionsDisabled, key: containercontract.ContainerDangerousActionsDisabled},
	{err: errMountUsageUnsupported, key: containercontract.ContainerMountUsageUnsupported},
	{err: errContainerMountNotFound, key: containercontract.ContainerMountNotFound},
}
