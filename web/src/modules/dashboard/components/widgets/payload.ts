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

function isString(value: unknown): value is string {
  return typeof value === 'string';
}

function hasRequiredStrings(value: unknown, keys: string[]) {
  return isRecord(value) && keys.every((key) => isString(value[key]));
}

function hasValidItems<T>(value: unknown, isValidItem: (item: unknown) => item is T): value is { items: T[] } {
  return hasArrayItems(value) && value.items.every(isValidItem);
}

function isStatGroupItem(value: unknown): value is DashboardStatGroupPayload['items'][number] {
  return isRecord(value) && hasRequiredStrings(value, ['key', 'label_key', 'label', 'value']);
}

function isAlertListItem(value: unknown): value is DashboardAlertListPayload['items'][number] {
  return (
    isRecord(value) && hasRequiredStrings(value, ['id', 'level', 'title_key', 'title']) && isAlertLevel(value.level)
  );
}

function isLinkListItem(value: unknown): value is DashboardLinkListPayload['items'][number] {
  return isRecord(value) && hasRequiredStrings(value, ['key', 'label_key', 'label', 'route_location']);
}

function isTimelineItem(value: unknown): value is DashboardTimelinePayload['items'][number] {
  return isRecord(value) && hasRequiredStrings(value, ['id', 'title_key', 'title', 'occurred_at']);
}

function isHealthStatus(value: unknown): value is DashboardHealthPayload['summary']['status'] {
  return isString(value) && ['healthy', 'degraded', 'disabled', 'unknown'].includes(value);
}

function isAlertLevel(value: unknown): value is DashboardAlertListPayload['items'][number]['level'] {
  return isString(value) && ['info', 'warning', 'error'].includes(value);
}

function isHealthItem(value: unknown): value is DashboardHealthPayload['items'][number] {
  return hasRequiredStrings(value, ['key', 'label_key', 'label']) && isRecord(value) && isHealthStatus(value.status);
}

export function asStatGroupPayload(value: unknown): DashboardStatGroupPayload | null {
  return hasValidItems(value, isStatGroupItem) ? (value as DashboardStatGroupPayload) : null;
}

export function asAlertListPayload(value: unknown): DashboardAlertListPayload | null {
  return hasValidItems(value, isAlertListItem) ? (value as DashboardAlertListPayload) : null;
}

export function asLinkListPayload(value: unknown): DashboardLinkListPayload | null {
  return hasValidItems(value, isLinkListItem) ? (value as DashboardLinkListPayload) : null;
}

export function asTimelinePayload(value: unknown): DashboardTimelinePayload | null {
  return hasValidItems(value, isTimelineItem) ? (value as DashboardTimelinePayload) : null;
}

export function asHealthPayload(value: unknown): DashboardHealthPayload | null {
  return isRecord(value) &&
    isRecord(value.summary) &&
    isHealthStatus(value.summary.status) &&
    hasValidItems(value, isHealthItem)
    ? (value as DashboardHealthPayload)
    : null;
}
