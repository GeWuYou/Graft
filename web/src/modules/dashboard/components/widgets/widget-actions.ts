import type { Router } from 'vue-router';

export function openDashboardRoute(router: Router, location: string) {
  void router.push(location);
}

export function formatDashboardDateTime(value: string) {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
}
