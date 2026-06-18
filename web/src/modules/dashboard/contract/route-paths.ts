// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0
/**
 * Normalizes a dashboard route path.
 *
 * Removes query and hash fragments, ensures the path starts with a single forward slash,
 * and removes trailing slashes (except for the root path `/`).
 *
 * @param path - The route path to normalize
 * @returns The normalized path, or an empty string if the input is empty after trimming.
 */

export function normalizeDashboardRoutePath(path: string) {
  const trimmed = path.trim();
  if (!trimmed) {
    return '';
  }

  const pathOnly = trimmed.split(/[?#]/, 1)[0] ?? '';
  const withLeadingSlash = pathOnly.startsWith('/') ? pathOnly : `/${pathOnly}`;
  return withLeadingSlash === '/' ? withLeadingSlash : withLeadingSlash.replace(/\/+$/, '');
}

/**
 * Normalizes a route path formed by joining a parent path with a child route path.
 *
 * If `routePath` is absolute (starts with `/`), it is normalized independently.
 * If `routePath` is empty, only `parentPath` is normalized. Otherwise, both paths
 * are joined with a `/` separator and the result is normalized.
 *
 * @param parentPath - The base route path
 * @param routePath - The route path to join with the parent
 * @returns The normalized joined dashboard route path
 */
export function normalizeJoinedDashboardRoutePath(parentPath: string, routePath: string) {
  if (routePath.startsWith('/')) {
    return normalizeDashboardRoutePath(routePath);
  }

  if (!routePath) {
    return normalizeDashboardRoutePath(parentPath);
  }

  return normalizeDashboardRoutePath(`${parentPath}/${routePath}`);
}
