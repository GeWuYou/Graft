function pad(value: number) {
  return String(value).padStart(2, '0');
}

function formatLocalDateTime(value: Date) {
  return `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())} ${pad(value.getHours())}:${pad(value.getMinutes())}:${pad(value.getSeconds())}`;
}

function formatRouteUtcDateTime(value: Date) {
  return value.toISOString();
}

function parseLocalDateTime(value: string) {
  const normalized = value.trim().replace(' ', 'T');
  const date = new Date(normalized);
  return Number.isNaN(date.getTime()) ? null : date;
}

export function localDateTimeToUtcIso(value: string) {
  const date = parseLocalDateTime(value);
  return date ? formatRouteUtcDateTime(date) : value;
}

function routeUtcToLocalDateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : formatLocalDateTime(date);
}

export function normalizeRouteRangeForPageState(range: string[]) {
  return range.filter(Boolean).map(routeUtcToLocalDateTime);
}

export function normalizePageStateRangeForRoute(range: string[]) {
  return range.filter(Boolean).map(localDateTimeToUtcIso);
}

export function buildRecentHoursLocalRange(now: Date, hours: number) {
  const end = new Date(now);
  const start = new Date(now.getTime() - hours * 60 * 60 * 1000);
  return [formatLocalDateTime(start), formatLocalDateTime(end)];
}

export function buildTodayLocalRange(now: Date) {
  const start = new Date(now);
  start.setHours(0, 0, 0, 0);
  return [formatLocalDateTime(start), formatLocalDateTime(now)];
}
