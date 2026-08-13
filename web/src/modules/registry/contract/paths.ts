export const REGISTRY_ROUTE_PATH = {
  LIST: '/infrastructure/registries',
  DETAIL: '/infrastructure/registries/:connectionRef',
} as const;

export function registryDetailPath(connectionRef: string) {
  return `/infrastructure/registries/${encodeURIComponent(connectionRef)}`;
}
