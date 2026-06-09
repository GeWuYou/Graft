// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { LocationQuery, LocationQueryValue } from 'vue-router';

type LogRouteQuery = Record<string, string | string[] | undefined> & {
  sort?: string | string[];
};

function readRouteString(source: LocationQuery | LogRouteQuery, key: string) {
  const rawValue = source[key] as LocationQueryValue | LocationQueryValue[] | undefined;
  const candidate = Array.isArray(rawValue) ? rawValue.find((item) => typeof item === 'string') : rawValue;

  return typeof candidate === 'string' ? candidate.trim() : '';
}

export function parseLogRouteQuery<TQuery extends LogRouteQuery>(
  query: LocationQuery | TQuery,
  keys: readonly string[],
): TQuery {
  const parsedQuery = Object.fromEntries(keys.map((key) => [key, readRouteString(query, key)])) as TQuery;
  const rawSort = query.sort as LocationQueryValue | LocationQueryValue[] | undefined;
  if (Array.isArray(rawSort)) {
    parsedQuery.sort = rawSort
      .filter((item): item is string => typeof item === 'string')
      .map((item) => item.trim())
      .filter(Boolean);
  } else if (typeof rawSort === 'string' && rawSort.trim()) {
    parsedQuery.sort = rawSort.trim();
  }

  return parsedQuery;
}

export function buildLogListLocation<TQuery extends LogRouteQuery>(
  path: string,
  keys: readonly string[],
  query: TQuery,
) {
  const normalizedQuery: Record<string, string | string[]> = {};
  const parsedQuery = parseLogRouteQuery(query, keys);

  keys.forEach((key) => {
    const value = parsedQuery[key];
    if (typeof value === 'string' && value) {
      normalizedQuery[key] = value;
    }
  });

  if (Array.isArray(parsedQuery.sort)) {
    const sortValues = parsedQuery.sort.filter((item): item is string => Boolean(item));
    if (sortValues.length) {
      normalizedQuery.sort = sortValues;
    }
  } else if (typeof parsedQuery.sort === 'string' && parsedQuery.sort) {
    normalizedQuery.sort = [parsedQuery.sort];
  }

  return {
    path,
    query: normalizedQuery,
  };
}
