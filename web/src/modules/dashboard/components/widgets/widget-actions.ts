import type { Router } from 'vue-router';

export function openDashboardRoute(router: Router, location: string) {
  void router.push(location);
}

export function formatDashboardDateTime(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}
