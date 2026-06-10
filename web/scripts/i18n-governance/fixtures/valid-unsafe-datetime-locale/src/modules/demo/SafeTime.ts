import { formatLocaleDateTime } from '@/shared/observability';

export const label = formatLocaleDateTime('2026-06-10T02:38:00Z', 'en-US');
export const fallback = new Date().toLocaleDateString('en-US');
