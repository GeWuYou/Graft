export type ServerStatusTone = 'healthy' | 'warning' | 'error' | 'unknown' | 'disabled';

export type ServerStatusTagTheme = 'success' | 'warning' | 'danger' | 'default';

export function resolveServerStatusTone(status?: string | null): ServerStatusTone {
  switch ((status ?? '').trim().toLowerCase()) {
    case 'healthy':
      return 'healthy';
    case 'degraded':
    case 'warning':
      return 'warning';
    case 'abnormal':
    case 'error':
      return 'error';
    case 'disabled':
    case 'notconfigured':
    case 'not_configured':
      return 'disabled';
    default:
      return 'unknown';
  }
}

export function serverStatusTagTheme(status: ServerStatusTone): ServerStatusTagTheme {
  switch (status) {
    case 'healthy':
      return 'success';
    case 'warning':
      return 'warning';
    case 'error':
      return 'danger';
    case 'disabled':
    case 'unknown':
    default:
      return 'default';
  }
}
