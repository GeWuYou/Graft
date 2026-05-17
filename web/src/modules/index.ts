import { rbacModuleRegistration } from './rbac';
import type { BootstrapRouteRegistration, WebModuleRegistration } from './types';
import { userModuleRegistration } from './user';

const allModuleRegistrations: WebModuleRegistration[] = [rbacModuleRegistration, userModuleRegistration];

const allBootstrapRouteRegistrations: BootstrapRouteRegistration[] = allModuleRegistrations.flatMap(
  (registration) => registration.bootstrapRoutes,
);

const bootstrapRouteRegistrationMap = new Map<string, BootstrapRouteRegistration>();

for (const registration of allBootstrapRouteRegistrations) {
  if (bootstrapRouteRegistrationMap.has(registration.menuPath)) {
    throw new Error(`duplicate bootstrap route registration: ${registration.menuPath}`);
  }

  bootstrapRouteRegistrationMap.set(registration.menuPath, registration);
}

export function getBootstrapRouteRegistration(menuPath: string) {
  return bootstrapRouteRegistrationMap.get(menuPath);
}
