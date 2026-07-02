import type {
  ProjectImportInspectResponse,
  ProjectImportRuntimeInspectNetworkResource,
  ProjectImportRuntimeInspectVolumeResource,
} from '../types/import';

export type ImportInspectResourceKey = 'containers' | 'networks' | 'volumes';

export type ImportInspectNetworkRow = {
  id: string;
  name: string;
  driver: string;
  scope: string;
  internal: boolean | null;
  containers: string[];
  containerCount: number;
  services: string[];
  serviceCount: number;
};

export type ImportInspectVolumeRow = {
  id: string;
  name: string;
  driver: string;
  anonymous: boolean;
  mountTarget: string;
  mountedBy: string[];
  containers: string[];
  containerCount: number;
};

const UNKNOWN_RESOURCE_VALUE = '-';

export function normalizeImportInspectNetworkRows(
  result: ProjectImportInspectResponse | null | undefined,
): ImportInspectNetworkRow[] {
  const resources = normalizeNetworkResources(result);
  if (resources.length) {
    return resources.map((item) => ({
      id: item.name,
      name: item.name,
      driver: readNonEmpty(item.driver, UNKNOWN_RESOURCE_VALUE),
      scope: readNonEmpty(item.scope, UNKNOWN_RESOURCE_VALUE),
      internal: typeof item.internal === 'boolean' ? item.internal : null,
      containers: normalizeStringList(item.containers),
      containerCount: normalizeCount(item.container_count, item.containers),
      services: normalizeStringList(item.services),
      serviceCount: normalizeCount(item.service_count, item.services),
    }));
  }

  return normalizeStringList(result?.networks).map((name) => ({
    id: name,
    name,
    driver: UNKNOWN_RESOURCE_VALUE,
    scope: UNKNOWN_RESOURCE_VALUE,
    internal: null,
    containers: [],
    containerCount: 0,
    services: [],
    serviceCount: 0,
  }));
}

export function normalizeImportInspectVolumeRows(
  result: ProjectImportInspectResponse | null | undefined,
): ImportInspectVolumeRow[] {
  const resources = normalizeVolumeResources(result);
  if (resources.length) {
    return resources.map((item) => ({
      id: item.name,
      name: item.name,
      driver: readNonEmpty(item.driver, UNKNOWN_RESOURCE_VALUE),
      anonymous: Boolean(item.anonymous),
      mountTarget: readNonEmpty(item.mount_target, UNKNOWN_RESOURCE_VALUE),
      mountedBy: normalizeStringList(item.mounted_by),
      containers: normalizeStringList(item.containers),
      containerCount: normalizeCount(item.container_count, item.containers),
    }));
  }

  return normalizeStringList(result?.volumes).map((name) => ({
    id: name,
    name,
    driver: UNKNOWN_RESOURCE_VALUE,
    anonymous: false,
    mountTarget: UNKNOWN_RESOURCE_VALUE,
    mountedBy: [],
    containers: [],
    containerCount: 0,
  }));
}

function normalizeNetworkResources(result: ProjectImportInspectResponse | null | undefined) {
  if (!Array.isArray(result?.networks)) {
    return [] as ProjectImportRuntimeInspectNetworkResource[];
  }

  return result.networks.filter(isNetworkResource);
}

function normalizeVolumeResources(result: ProjectImportInspectResponse | null | undefined) {
  if (!Array.isArray(result?.volumes)) {
    return [] as ProjectImportRuntimeInspectVolumeResource[];
  }

  return result.volumes.filter(isVolumeResource);
}

function isNetworkResource(value: unknown): value is ProjectImportRuntimeInspectNetworkResource {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const candidate = value as Partial<ProjectImportRuntimeInspectNetworkResource>;
  return typeof candidate.name === 'string';
}

function isVolumeResource(value: unknown): value is ProjectImportRuntimeInspectVolumeResource {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const candidate = value as Partial<ProjectImportRuntimeInspectVolumeResource>;
  return typeof candidate.name === 'string';
}

function normalizeStringList(value: unknown) {
  if (!Array.isArray(value)) {
    return [] as string[];
  }

  return value.filter((item): item is string => typeof item === 'string' && item.trim().length > 0);
}

function normalizeCount(value: unknown, listValue: unknown) {
  if (typeof value === 'number' && Number.isFinite(value) && value >= 0) {
    return value;
  }

  return normalizeStringList(listValue).length;
}

function readNonEmpty(value: unknown, fallback: string) {
  if (typeof value !== 'string') {
    return fallback;
  }

  const normalized = value.trim();
  return normalized || fallback;
}
