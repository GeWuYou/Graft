import type { BootstrapRouteRegistration, GlobalRouteRegistration, WebModuleRegistration } from './types';

type WebModuleRegistrationModule = {
  default: WebModuleRegistration;
};

type WebModuleBootstrapRoutesModule = {
  [key: string]: unknown;
};

type StableRouteNameRegistry = Map<string, string>;

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
  if (!value || typeof value !== 'object') {
    return false;
  }

  const registration = value as Partial<WebModuleRegistration>;
  const globalRoutes = registration.globalRoutes;

  return Boolean(
    typeof registration.moduleId === 'string' &&
    Array.isArray(registration.bootstrapRoutes) &&
    registration.bootstrapRoutes.every(isBootstrapRouteRegistration) &&
    (globalRoutes === undefined || (Array.isArray(globalRoutes) && globalRoutes.every(isGlobalRouteRegistration))),
  );
}

function isGlobalRouteRegistration(value: unknown): value is GlobalRouteRegistration {
  const meta = (value as Partial<GlobalRouteRegistration> | null)?.meta;

  return Boolean(
    value &&
    typeof value === 'object' &&
    typeof (value as Partial<GlobalRouteRegistration>).path === 'string' &&
    typeof (value as Partial<GlobalRouteRegistration>).routeName === 'string' &&
    typeof (value as Partial<GlobalRouteRegistration>).loadPage === 'function' &&
    meta &&
    typeof meta === 'object' &&
    !Array.isArray(meta),
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
  routeNameRegistry: StableRouteNameRegistry,
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

function registerStableRouteNamePair(routeNameRegistry: StableRouteNameRegistry, routeName: string, owner: string) {
  registerStableRouteName(routeNameRegistry, routeName, owner, 'parent');
  registerStableRouteName(routeNameRegistry, `${routeName}Index`, owner, 'child');
}

function registerModuleStableRouteNames(
  routeNameRegistry: StableRouteNameRegistry,
  moduleRegistration: WebModuleRegistration,
) {
  for (const routeRegistration of moduleRegistration.bootstrapRoutes) {
    registerStableRouteNamePair(
      routeNameRegistry,
      routeRegistration.routeName,
      `${moduleRegistration.moduleId}:${routeRegistration.menuPath}`,
    );
  }

  for (const routeRegistration of moduleRegistration.globalRoutes ?? []) {
    registerStableRouteNamePair(
      routeNameRegistry,
      routeRegistration.routeName,
      `${moduleRegistration.moduleId}:${routeRegistration.path}`,
    );
  }
}

function buildStableRouteNameRegistry(registrations: WebModuleRegistration[]) {
  const stableRouteNameRegistry = new Map<string, string>();

  for (const moduleRegistration of registrations) {
    registerModuleStableRouteNames(stableRouteNameRegistry, moduleRegistration);
  }

  return stableRouteNameRegistry;
}

function collectBootstrapRouteRegistrationMap(registrations: WebModuleRegistration[]) {
  const bootstrapRouteRegistrationMap = new Map<string, BootstrapRouteRegistration>();

  for (const moduleRegistration of registrations) {
    for (const routeRegistration of moduleRegistration.bootstrapRoutes) {
      if (bootstrapRouteRegistrationMap.has(routeRegistration.menuPath)) {
        throw new Error(`duplicate bootstrap route registration: ${routeRegistration.menuPath}`);
      }

      bootstrapRouteRegistrationMap.set(routeRegistration.menuPath, routeRegistration);
    }
  }

  return bootstrapRouteRegistrationMap;
}

export function buildBootstrapRouteRegistrationMap(registrations: WebModuleRegistration[]) {
  buildStableRouteNameRegistry(registrations);
  return collectBootstrapRouteRegistrationMap(registrations);
}

function collectGlobalRouteRegistrations(registrations: WebModuleRegistration[]) {
  const globalRouteRegistrations: GlobalRouteRegistration[] = [];
  const stablePathRegistry = new Map<string, string>();

  for (const moduleRegistration of registrations) {
    for (const routeRegistration of moduleRegistration.globalRoutes ?? []) {
      const owner = `${moduleRegistration.moduleId}:${routeRegistration.path}`;
      const existingOwner = stablePathRegistry.get(routeRegistration.path);
      if (existingOwner) {
        throw new Error(`duplicate global route path: ${routeRegistration.path} already owned by ${existingOwner}`);
      }

      stablePathRegistry.set(routeRegistration.path, owner);
      globalRouteRegistrations.push(routeRegistration);
    }
  }

  return globalRouteRegistrations;
}

export function buildGlobalRouteRegistrations(registrations: WebModuleRegistration[]) {
  buildStableRouteNameRegistry(registrations);
  return collectGlobalRouteRegistrations(registrations);
}

function buildModuleRouteRegistries(registrations: WebModuleRegistration[]) {
  buildStableRouteNameRegistry(registrations);

  return {
    bootstrapRouteRegistrationMap: collectBootstrapRouteRegistrationMap(registrations),
    globalRouteRegistrations: collectGlobalRouteRegistrations(registrations),
  };
}

const moduleRegistrations = loadModuleRegistrations();
const { bootstrapRouteRegistrationMap, globalRouteRegistrations } = buildModuleRouteRegistries(moduleRegistrations);

export function getBootstrapRouteRegistration(menuPath: string) {
  return bootstrapRouteRegistrationMap.get(menuPath);
}

export function getGlobalRouteRegistrations() {
  return [...globalRouteRegistrations];
}
