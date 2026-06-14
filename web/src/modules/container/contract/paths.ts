// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const CONTAINER_ROUTE_PATH = {
  LIST: '/ops/containers',
} as const;

export const CONTAINER_API_PATH = {
  LIST: '/api/ops/containers',
  DETAIL: '/api/ops/containers/{id}',
  LOGS: '/api/ops/containers/{id}/logs',
  START: '/api/ops/containers/{id}/start',
  STOP: '/api/ops/containers/{id}/stop',
  RESTART: '/api/ops/containers/{id}/restart',
} as const;

export function buildContainerDetailApiPath(containerId: string) {
  return CONTAINER_API_PATH.DETAIL.replace('{id}', encodeContainerPathParam(containerId));
}

export function buildContainerLogsApiPath(containerId: string) {
  return CONTAINER_API_PATH.LOGS.replace('{id}', encodeContainerPathParam(containerId));
}

export function buildContainerStartApiPath(containerId: string) {
  return CONTAINER_API_PATH.START.replace('{id}', encodeContainerPathParam(containerId));
}

export function buildContainerStopApiPath(containerId: string) {
  return CONTAINER_API_PATH.STOP.replace('{id}', encodeContainerPathParam(containerId));
}

export function buildContainerRestartApiPath(containerId: string) {
  return CONTAINER_API_PATH.RESTART.replace('{id}', encodeContainerPathParam(containerId));
}

function encodeContainerPathParam(containerId: string) {
  return encodeURIComponent(containerId);
}
