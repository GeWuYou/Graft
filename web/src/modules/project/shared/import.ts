import type {
  ProjectImportDirectoryInspectFileEntry,
  ProjectImportDirectoryRef,
  ProjectImportDirectorySource,
  ProjectImportInspectResponse,
  ProjectImportRuntimeCandidate,
  ProjectImportRuntimeInspectNetworkResource,
  ProjectImportRuntimeInspectVolumeResource,
  ProjectImportRuntimeMember,
} from '../types/import';

export function buildDirectorySelection(source: ProjectImportDirectorySource, path: string): ProjectImportDirectoryRef {
  return {
    provider: source.provider,
    root_id: source.root_id,
    path: normalizeDirectoryPath(path),
  };
}

export function initialDirectoryPath(source: ProjectImportDirectorySource) {
  return normalizeDirectoryPath(source.initial_path || '');
}

export function normalizeDirectoryPath(path: string | null | undefined) {
  const normalized = (path || '').trim().replace(/\\/g, '/');
  if (!normalized || normalized === '.') {
    return '';
  }

  return normalized.split('/').filter(Boolean).join('/');
}

function splitDirectorySegments(path: string) {
  const normalized = normalizeDirectoryPath(path);
  return normalized ? normalized.split('/') : [];
}

export function buildDirectoryBreadcrumbs(directory: ProjectImportDirectoryRef) {
  const segments = splitDirectorySegments(directory.path);
  return [
    {
      key: '',
      label: directory.root_id,
      path: '',
    },
    ...segments.map((segment, index) => ({
      key: `${index}:${segment}`,
      label: segment,
      path: segments.slice(0, index + 1).join('/'),
    })),
  ];
}

export function buildDirectorySourceLabel(source: ProjectImportDirectorySource) {
  const suffix = source.managed ? ' (Managed)' : '';
  return `${source.label}${suffix}`;
}

export function buildSuggestedDisplayName(result: ProjectImportInspectResponse) {
  return (result.display_name_suggested || result.canonical_project_name || '').trim();
}

export function normalizeStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is string => typeof item === 'string');
}

function isProjectImportDirectoryInspectFileEntry(value: unknown): value is ProjectImportDirectoryInspectFileEntry {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const entry = value as Partial<ProjectImportDirectoryInspectFileEntry>;
  return (
    isProjectFileKind(entry.kind) &&
    isProjectFileRole(entry.role) &&
    typeof entry.absolute_path === 'string' &&
    typeof entry.display_path === 'string' &&
    typeof entry.order_index === 'number' &&
    (typeof entry.last_observed_hash === 'string' ||
      entry.last_observed_hash === null ||
      typeof entry.last_observed_hash === 'undefined')
  );
}

function normalizeInspectFileEntries(value: unknown): ProjectImportDirectoryInspectFileEntry[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter(isProjectImportDirectoryInspectFileEntry);
}

function isProjectImportRuntimeMember(value: unknown): value is ProjectImportRuntimeMember {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const item = value as Partial<ProjectImportRuntimeMember>;
  return (
    typeof item.container_id === 'string' &&
    typeof item.container_name === 'string' &&
    typeof item.service_name === 'string' &&
    typeof item.state === 'string'
  );
}

function normalizeRuntimeMembers(value: unknown): ProjectImportRuntimeMember[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter(isProjectImportRuntimeMember);
}

function isProjectFileKind(value: unknown): value is 'compose' | 'env' {
  return value === 'compose' || value === 'env';
}

function isProjectFileRole(value: unknown): value is 'primary' | 'override' | 'detected' | 'manual' {
  return value === 'primary' || value === 'override' || value === 'detected' || value === 'manual';
}

function isProjectImportRuntimeInspectNetworkResource(
  value: unknown,
): value is ProjectImportRuntimeInspectNetworkResource {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const resource = value as Partial<ProjectImportRuntimeInspectNetworkResource>;
  return typeof resource.name === 'string';
}

function isProjectImportRuntimeInspectVolumeResource(
  value: unknown,
): value is ProjectImportRuntimeInspectVolumeResource {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const resource = value as Partial<ProjectImportRuntimeInspectVolumeResource>;
  return typeof resource.name === 'string';
}

function normalizeInspectNetworkResources(value: unknown) {
  if (!Array.isArray(value)) {
    return [] as Array<ProjectImportRuntimeInspectNetworkResource | string>;
  }

  return value.filter(
    (item): item is ProjectImportRuntimeInspectNetworkResource | string =>
      typeof item === 'string' || isProjectImportRuntimeInspectNetworkResource(item),
  );
}

function normalizeInspectVolumeResources(value: unknown) {
  if (!Array.isArray(value)) {
    return [] as Array<ProjectImportRuntimeInspectVolumeResource | string>;
  }

  return value.filter(
    (item): item is ProjectImportRuntimeInspectVolumeResource | string =>
      typeof item === 'string' || isProjectImportRuntimeInspectVolumeResource(item),
  );
}

/**
 * 对 inspect 响应中的可空数组字段做统一归一化，避免页面渲染直接消费 null。
 *
 * @param result - 原始导入检查结果
 * @returns 归一化后的检查结果；空输入保持为 null
 */
export function normalizeProjectImportInspectResponse(
  result: ProjectImportInspectResponse | null,
): ProjectImportInspectResponse | null {
  if (!result) {
    return null;
  }

  return {
    ...result,
    compose_files: normalizeInspectFileEntries(result.compose_files),
    env_files: normalizeInspectFileEntries(result.env_files),
    services: normalizeStringArray(result.services),
    networks: normalizeInspectNetworkResources(result.networks),
    volumes: normalizeInspectVolumeResources(result.volumes),
    runtime_members: normalizeRuntimeMembers(result.runtime_members),
    warnings: normalizeStringArray(result.warnings),
    conflicts: normalizeStringArray(result.conflicts),
  };
}

/**
 * 判断导入结果是否存在阻塞性冲突。
 *
 * @param result - 导入检查结果
 * @returns `true` 如果存在冲突或校验状态为 `conflict`，`false` 否则。
 */
export function hasBlockingImportConflicts(result: ProjectImportInspectResponse | null) {
  const normalized = normalizeProjectImportInspectResponse(result);
  return Boolean(normalized?.conflicts?.length) || normalized?.validation_status === 'conflict';
}

/**
 * 判断候选是否处于可执行 inspect 的 ready 状态。
 *
 * @param candidate - 运行时候选
 * @returns `true` 表示可以进入 inspect
 */
export function isProjectImportRuntimeCandidateReady(candidate: ProjectImportRuntimeCandidate) {
  return candidate.importable && candidate.status === 'ready';
}

/**
 * 生成候选的稳定原因键。
 *
 * @param candidate - 运行时候选
 * @returns 优先返回首个稳定 reason code；否则回退到状态级原因键
 */
export function resolveProjectImportRuntimeCandidateReasonKey(candidate: ProjectImportRuntimeCandidate) {
  const [primaryReasonCode] = candidate.status_reason_codes;
  if (primaryReasonCode?.trim()) {
    return primaryReasonCode.trim();
  }

  return candidate.status || 'unavailable';
}
