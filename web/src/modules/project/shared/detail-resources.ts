import type { ApplicationServiceItem } from '../types/project';

export type ApplicationResourceType = 'containers' | 'networks' | 'volumes';

export type ApplicationNetworkResourceRow = {
  containerCount: number;
  containers: string[];
  driver: string;
  id: string;
  internal: boolean | null;
  name: string;
  scope: string;
  serviceCount: number;
  services: string[];
};

export type ApplicationVolumeResourceRow = {
  anonymous: boolean;
  containerCount: number;
  containers: string[];
  driver: string;
  id: string;
  mountTarget: string;
  mountedBy: string[];
  name: string;
};

type ParsedVolumeMount = {
  anonymous: boolean;
  mountTarget: string;
  name: string;
};

const UNKNOWN_RESOURCE_VALUE = '-';

function summarizeServiceMembers(service: ApplicationServiceItem) {
  return {
    containerNames: service.container_members.map((member) => member.container_name.trim()).filter(Boolean),
    serviceName: service.service_name.trim(),
  };
}

export function buildApplicationNetworkResourceRows(
  services: ApplicationServiceItem[],
): ApplicationNetworkResourceRow[] {
  const rows = new Map<string, ApplicationNetworkResourceRow>();

  for (const service of services) {
    const { containerNames, serviceName } = summarizeServiceMembers(service);

    for (const declaredNetwork of service.declared_networks ?? []) {
      const networkName = declaredNetwork.trim();
      if (!networkName) {
        continue;
      }

      const existing = rows.get(networkName) ?? {
        containerCount: 0,
        containers: [],
        driver: UNKNOWN_RESOURCE_VALUE,
        id: networkName,
        internal: null,
        name: networkName,
        scope: UNKNOWN_RESOURCE_VALUE,
        serviceCount: 0,
        services: [],
      };

      const serviceSet = new Set(existing.services);
      const containerSet = new Set(existing.containers);
      serviceSet.add(serviceName);
      for (const containerName of containerNames) {
        containerSet.add(containerName);
      }

      const nextRow: ApplicationNetworkResourceRow = {
        ...existing,
        containerCount: containerSet.size,
        containers: Array.from(containerSet).sort(),
        serviceCount: serviceSet.size,
        services: Array.from(serviceSet).sort(),
      };

      rows.set(networkName, nextRow);
    }
  }

  return Array.from(rows.values()).sort((left, right) => left.name.localeCompare(right.name));
}

export function buildApplicationVolumeResourceRows(services: ApplicationServiceItem[]): ApplicationVolumeResourceRow[] {
  const rows = new Map<string, ApplicationVolumeResourceRow>();

  for (const service of services) {
    const { containerNames, serviceName } = summarizeServiceMembers(service);

    for (const declaredVolume of service.declared_volumes ?? []) {
      const parsedMount = parseDeclaredVolumeMount(declaredVolume);
      if (!parsedMount) {
        continue;
      }

      const existing = rows.get(parsedMount.name) ?? {
        anonymous: parsedMount.anonymous,
        containerCount: 0,
        containers: [],
        driver: UNKNOWN_RESOURCE_VALUE,
        id: parsedMount.name,
        mountTarget: parsedMount.mountTarget,
        mountedBy: [],
        name: parsedMount.name,
      };

      const serviceSet = new Set(existing.mountedBy);
      const containerSet = new Set(existing.containers);
      serviceSet.add(serviceName);
      for (const containerName of containerNames) {
        containerSet.add(containerName);
      }

      const nextRow: ApplicationVolumeResourceRow = {
        ...existing,
        anonymous: existing.anonymous && parsedMount.anonymous,
        containerCount: containerSet.size,
        containers: Array.from(containerSet).sort(),
        mountedBy: Array.from(serviceSet).sort(),
      };

      rows.set(parsedMount.name, nextRow);
    }
  }

  return Array.from(rows.values()).sort((left, right) => left.name.localeCompare(right.name));
}

export function paginateApplicationResourceRows<T>(rows: T[], current: number, pageSize: number) {
  if (pageSize <= 0) {
    return rows;
  }

  const safeCurrent = Math.max(1, current);
  const start = (safeCurrent - 1) * pageSize;
  return rows.slice(start, start + pageSize);
}

export function parseDeclaredVolumeMount(value: string): ParsedVolumeMount | null {
  const mount = value.trim();
  if (!mount) {
    return null;
  }

  if (mount.includes('=')) {
    return parseLongSyntaxVolumeMount(mount);
  }

  return parseShortSyntaxVolumeMount(mount);
}

function parseLongSyntaxVolumeMount(value: string): ParsedVolumeMount | null {
  const pairs = value.split(',').map((item) => item.trim());
  const entries = new Map<string, string>();

  for (const pair of pairs) {
    const separatorIndex = pair.indexOf('=');
    if (separatorIndex <= 0) {
      continue;
    }

    const key = pair.slice(0, separatorIndex).trim().toLowerCase();
    const entryValue = pair.slice(separatorIndex + 1).trim();
    if (!key) {
      continue;
    }

    entries.set(key, entryValue);
  }

  const type = entries.get('type');
  if (type && type !== 'volume') {
    return null;
  }

  const source = entries.get('source') ?? entries.get('src') ?? '';
  const target = entries.get('target') ?? entries.get('dst') ?? entries.get('destination') ?? '';
  if (!target) {
    return null;
  }

  if (source) {
    if (looksLikeHostPath(source)) {
      return null;
    }

    return {
      anonymous: false,
      mountTarget: target,
      name: source,
    };
  }

  return {
    anonymous: true,
    mountTarget: target,
    name: target,
  };
}

function parseShortSyntaxVolumeMount(value: string): ParsedVolumeMount | null {
  const segments = value.split(':');
  if (segments.length === 1) {
    const target = segments[0]?.trim() ?? '';
    if (!target || !looksLikeContainerPath(target)) {
      return null;
    }

    return {
      anonymous: true,
      mountTarget: target,
      name: target,
    };
  }

  const source = segments[0]?.trim() ?? '';
  const target = segments[1]?.trim() ?? '';
  if (!source || !target) {
    return null;
  }

  if (looksLikeHostPath(source)) {
    return null;
  }

  return {
    anonymous: false,
    mountTarget: target,
    name: source,
  };
}

function looksLikeHostPath(value: string) {
  if (!value) {
    return false;
  }

  return (
    value.startsWith('/') ||
    value.startsWith('./') ||
    value.startsWith('../') ||
    value.startsWith('~/') ||
    /^[A-Za-z]:[\\/]/.test(value)
  );
}

function looksLikeContainerPath(value: string) {
  return value.startsWith('/');
}
