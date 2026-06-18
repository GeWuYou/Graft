// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"errors"
	"net/http"

	containercontract "graft/server/modules/container/contract"
)

// Statusforerror returns the HTTP status code for the given error.
func statusForError(err error) int {
	switch {
	case errors.Is(err, errInvalidRef), errors.Is(err, errInvalidListQuery), errors.Is(err, errInvalidBatchAction), errors.Is(err, errLogsTooLarge), errors.Is(err, errInvalidLogQuery), errors.Is(err, errMountUsageUnsupported):
		return http.StatusBadRequest
	case errors.Is(err, errRuntimeDisabled), errors.Is(err, errDangerousActionsDisabled), errors.Is(err, errRuntimePermissionDenied):
		return http.StatusForbidden
	case errors.Is(err, errContainerNotFound), errors.Is(err, errContainerMountNotFound):
		return http.StatusNotFound
	case errors.Is(err, errInvalidContainerState):
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
	{err: errContainerRuntimeTimeout, key: containercontract.ContainerTimeout},
	{err: errDangerousActionsDisabled, key: containercontract.ContainerDangerousActionsDisabled},
	{err: errMountUsageUnsupported, key: containercontract.ContainerMountUsageUnsupported},
	{err: errContainerMountNotFound, key: containercontract.ContainerMountNotFound},
}
