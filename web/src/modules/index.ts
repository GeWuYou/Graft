import type { BootstrapRouteRegistration, WebModuleRegistration } from './types';

type WebModuleRegistrationModule = {
  default: WebModuleRegistration;
};

type WebModuleBootstrapRoutesModule = {
  [key: string]: unknown;
};

const moduleRegistrationModules = import.meta.glob<WebModuleRegistrationModule>('./*/index.ts', {
  eager: true,
});
const moduleBootstrapRouteModules = import.meta.glob<WebModuleBootstrapRoutesModule>('./*/bootstrap-routes.ts', {
  eager: true,
});

export function resolveModuleRegistrationModulePaths(
  registrationModulePaths: string[],
  bootstrapRouteModulePaths: string[],
) {
  const moduleDirectories = new Set(
    bootstrapRouteModulePaths.map((modulePath) => modulePath.replace(/\/bootstrap-routes\.ts$/, '')),
  );

  return registrationModulePaths.filter((modulePath) => moduleDirectories.has(modulePath.replace(/\/index\.ts$/, '')));
}

function isBootstrapRouteRegistration(value: unknown): value is BootstrapRouteRegistration {
  return Boolean(
    value &&
    typeof value === 'object' &&
    typeof (value as Partial<BootstrapRouteRegistration>).menuPath === 'string' &&
    typeof (value as Partial<BootstrapRouteRegistration>).routeName === 'string' &&
    typeof (value as Partial<BootstrapRouteRegistration>).loadPage === 'function',
  );
}

function isWebModuleRegistration(value: unknown): value is WebModuleRegistration {
  return Boolean(
    value &&
    typeof value === 'object' &&
    typeof (value as Partial<WebModuleRegistration>).moduleId === 'string' &&
    Array.isArray((value as Partial<WebModuleRegistration>).bootstrapRoutes) &&
    (value as Partial<WebModuleRegistration>).bootstrapRoutes?.every(isBootstrapRouteRegistration),
  );
}

function loadModuleRegistrations() {
  const moduleIdRegistry = new Set<string>();
  const moduleRegistrationPaths = resolveModuleRegistrationModulePaths(
    Object.keys(moduleRegistrationModules),
    Object.keys(moduleBootstrapRouteModules),
  );

  return moduleRegistrationPaths.map((modulePath) => {
    const registrationModule = moduleRegistrationModules[modulePath];
    const registration = registrationModule.default;
    if (!isWebModuleRegistration(registration)) {
      throw new Error(`invalid module registration export: ${modulePath}`);
    }

    if (moduleIdRegistry.has(registration.moduleId)) {
      throw new Error(`duplicate module registration id: ${registration.moduleId}`);
    }

    moduleIdRegistry.add(registration.moduleId);
    return registration;
  });
}

function registerStableRouteName(
  routeNameRegistry: Map<string, string>,
  routeName: string,
  owner: string,
  source: 'parent' | 'child',
) {
  const existingOwner = routeNameRegistry.get(routeName);
  if (existingOwner) {
    throw new Error(`duplicate bootstrap route name (${source}): ${routeName} already owned by ${existingOwner}`);
  }

  routeNameRegistry.set(routeName, owner);
}

export function buildBootstrapRouteRegistrationMap(registrations: WebModuleRegistration[]) {
  const bootstrapRouteRegistrationMap = new Map<string, BootstrapRouteRegistration>();
  const stableRouteNameRegistry = new Map<string, string>();

  for (const moduleRegistration of registrations) {
    for (const routeRegistration of moduleRegistration.bootstrapRoutes) {
      if (bootstrapRouteRegistrationMap.has(routeRegistration.menuPath)) {
        throw new Error(`duplicate bootstrap route registration: ${routeRegistration.menuPath}`);
      }

      const owner = `${moduleRegistration.moduleId}:${routeRegistration.menuPath}`;
      registerStableRouteName(stableRouteNameRegistry, routeRegistration.routeName, owner, 'parent');
      registerStableRouteName(stableRouteNameRegistry, `${routeRegistration.routeName}Index`, owner, 'child');
      bootstrapRouteRegistrationMap.set(routeRegistration.menuPath, routeRegistration);
    }
  }

  return bootstrapRouteRegistrationMap;
}

const bootstrapRouteRegistrationMap = buildBootstrapRouteRegistrationMap(loadModuleRegistrations());

export function getBootstrapRouteRegistration(menuPath: string) {
  return bootstrapRouteRegistrationMap.get(menuPath);
}
