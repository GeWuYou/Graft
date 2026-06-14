// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"errors"
	"net/http"

	containercontract "graft/server/modules/container/contract"
)

func statusForError(err error) int {
	switch {
	case errors.Is(err, errInvalidRef), errors.Is(err, errLogsTooLarge):
		return http.StatusBadRequest
	case errors.Is(err, errRuntimeDisabled), errors.Is(err, errDangerousActionsDisabled):
		return http.StatusForbidden
	case errors.Is(err, errContainerNotFound):
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
	return containercontract.ContainerRuntimeDaemonUnavailable
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
	{err: errRuntimeDaemonUnavailable, key: containercontract.ContainerRuntimeDaemonUnavailable},
	{err: errContainerNotFound, key: containercontract.ContainerNotFound},
	{err: errInvalidRef, key: containercontract.ContainerInvalidRef},
	{err: errInvalidContainerState, key: containercontract.ContainerInvalidState},
	{err: errLogsTooLarge, key: containercontract.ContainerLogsTooLarge},
	{err: errContainerRuntimeTimeout, key: containercontract.ContainerTimeout},
	{err: errDangerousActionsDisabled, key: containercontract.ContainerDangerousActionsDisabled},
}
