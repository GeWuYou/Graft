export const RUNTIME_TARGET_ROUTE_PATH = {
  LIST: '/infrastructure/runtime-targets',
  DETAIL: '/infrastructure/runtime-targets/:id',
} as const;

export function runtimeTargetDetailPath(id: number) {
  return '/infrastructure/runtime-targets/' + id;
}
