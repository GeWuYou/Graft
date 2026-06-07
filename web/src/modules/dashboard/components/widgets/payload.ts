import type {
  DashboardAlertListPayload,
  DashboardHealthPayload,
  DashboardLinkListPayload,
  DashboardStatGroupPayload,
  DashboardTimelinePayload,
} from '../../types/dashboard';

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value));
}

function hasArrayItems(value: unknown): value is { items: unknown[] } {
  return isRecord(value) && Array.isArray(value.items);
}

export function asStatGroupPayload(value: unknown): DashboardStatGroupPayload | null {
  return hasArrayItems(value) ? (value as DashboardStatGroupPayload) : null;
}

export function asAlertListPayload(value: unknown): DashboardAlertListPayload | null {
  return hasArrayItems(value) ? (value as DashboardAlertListPayload) : null;
}

export function asLinkListPayload(value: unknown): DashboardLinkListPayload | null {
  return hasArrayItems(value) ? (value as DashboardLinkListPayload) : null;
}

export function asTimelinePayload(value: unknown): DashboardTimelinePayload | null {
  return hasArrayItems(value) ? (value as DashboardTimelinePayload) : null;
}

export function asHealthPayload(value: unknown): DashboardHealthPayload | null {
  return isRecord(value) && isRecord(value.summary) && Array.isArray(value.items)
    ? (value as DashboardHealthPayload)
    : null;
}
