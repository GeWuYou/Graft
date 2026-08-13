export const REGISTRY_ROUTE_PATH = {
  LIST: '/infrastructure/registries',
  DETAIL: '/infrastructure/registries/:connectionRef',
} as const;

export const REGISTRY_DETAIL_MODE = {
  EDIT: 'edit',
} as const;

export function registryDetailPath(
  connectionRef: string,
  options?: { mode?: (typeof REGISTRY_DETAIL_MODE)[keyof typeof REGISTRY_DETAIL_MODE] },
) {
  const path = `${REGISTRY_ROUTE_PATH.LIST}/${encodeURIComponent(connectionRef)}`;
  return options?.mode ? `${path}?mode=${options.mode}` : path;
}
