import type {
  ApplicationImportDirectoryInspectFileEntry,
  ApplicationImportDirectoryRef,
  ApplicationImportDirectorySource,
  ApplicationImportInspectResponse,
  ApplicationImportRuntimeCandidate,
  ApplicationImportRuntimeInspectNetworkResource,
  ApplicationImportRuntimeInspectVolumeResource,
  ApplicationImportRuntimeMember,
  ApplicationImportServiceOption,
} from '../types/import';

export function buildDirectorySelection(
  source: ApplicationImportDirectorySource,
  path: string,
): ApplicationImportDirectoryRef {
  return {
    provider: source.provider,
    root_id: source.root_id,
    path: normalizeDirectoryPath(path),
  };
}

export function initialDirectoryPath(source: ApplicationImportDirectorySource) {
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

export function buildDirectoryBreadcrumbs(directory: ApplicationImportDirectoryRef) {
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

export function buildDirectorySourceLabel(source: ApplicationImportDirectorySource) {
  const suffix = source.managed ? ' (Managed)' : '';
  return `${source.label}${suffix}`;
}

export function buildSuggestedDisplayName(result: ApplicationImportInspectResponse) {
  return (result.display_name_suggested || result.compose_project_name || '').trim();
}

export function normalizeStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is string => typeof item === 'string');
}

function isApplicationImportDirectoryInspectFileEntry(
  value: unknown,
): value is ApplicationImportDirectoryInspectFileEntry {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const entry = value as Partial<ApplicationImportDirectoryInspectFileEntry>;
  return (
    isApplicationFileKind(entry.kind) &&
    isApplicationFileRole(entry.role) &&
    typeof entry.absolute_path === 'string' &&
    typeof entry.display_path === 'string' &&
    typeof entry.order_index === 'number' &&
    (typeof entry.last_observed_hash === 'string' ||
      entry.last_observed_hash === null ||
      typeof entry.last_observed_hash === 'undefined')
  );
}

function normalizeInspectFileEntries(value: unknown): ApplicationImportDirectoryInspectFileEntry[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter(isApplicationImportDirectoryInspectFileEntry);
}

function isApplicationImportRuntimeMember(value: unknown): value is ApplicationImportRuntimeMember {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const item = value as Partial<ApplicationImportRuntimeMember>;
  return (
    typeof item.container_id === 'string' &&
    typeof item.container_name === 'string' &&
    typeof item.service_name === 'string' &&
    typeof item.state === 'string'
  );
}

function normalizeRuntimeMembers(value: unknown): ApplicationImportRuntimeMember[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter(isApplicationImportRuntimeMember);
}

function normalizeServiceOptions(value: unknown): ApplicationImportServiceOption[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.flatMap((item) => {
    if (!item || typeof item !== 'object') {
      return [];
    }
    const option = item as Partial<ApplicationImportServiceOption>;
    if (typeof option.name !== 'string') {
      return [];
    }
    return [{ name: option.name, depends_on: normalizeStringArray(option.depends_on) }];
  });
}

function isApplicationFileKind(value: unknown): value is 'compose' | 'env' {
  return value === 'compose' || value === 'env';
}

function isApplicationFileRole(value: unknown): value is 'primary' | 'override' | 'detected' | 'manual' {
  return value === 'primary' || value === 'override' || value === 'detected' || value === 'manual';
}

function isApplicationImportRuntimeInspectNetworkResource(
  value: unknown,
): value is ApplicationImportRuntimeInspectNetworkResource {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const resource = value as Partial<ApplicationImportRuntimeInspectNetworkResource>;
  return typeof resource.name === 'string';
}

function isApplicationImportRuntimeInspectVolumeResource(
  value: unknown,
): value is ApplicationImportRuntimeInspectVolumeResource {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const resource = value as Partial<ApplicationImportRuntimeInspectVolumeResource>;
  return typeof resource.name === 'string';
}

function normalizeInspectNetworkResources(value: unknown) {
  if (!Array.isArray(value)) {
    return [] as Array<ApplicationImportRuntimeInspectNetworkResource | string>;
  }

  return value.filter(
    (item): item is ApplicationImportRuntimeInspectNetworkResource | string =>
      typeof item === 'string' || isApplicationImportRuntimeInspectNetworkResource(item),
  );
}

function normalizeInspectVolumeResources(value: unknown) {
  if (!Array.isArray(value)) {
    return [] as Array<ApplicationImportRuntimeInspectVolumeResource | string>;
  }

  return value.filter(
    (item): item is ApplicationImportRuntimeInspectVolumeResource | string =>
      typeof item === 'string' || isApplicationImportRuntimeInspectVolumeResource(item),
  );
}

/**
 * 对 inspect 响应中的可空数组字段做统一归一化，避免页面渲染直接消费 null。
 *
 * @param result - 原始导入检查结果
 * @returns 归一化后的检查结果；空输入保持为 null
 */
export function normalizeApplicationImportInspectResponse(
  result: ApplicationImportInspectResponse | null,
): ApplicationImportInspectResponse | null {
  if (!result) {
    return null;
  }

  return {
    ...result,
    compose_files: normalizeInspectFileEntries(result.compose_files),
    env_files: normalizeInspectFileEntries(result.env_files),
    services: normalizeStringArray(result.services),
    service_options: normalizeServiceOptions(result.service_options),
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
export function hasBlockingImportConflicts(result: ApplicationImportInspectResponse | null) {
  const normalized = normalizeApplicationImportInspectResponse(result);
  return Boolean(normalized?.conflicts?.length) || normalized?.validation_status === 'conflict';
}

/**
 * 判断候选是否处于可执行 inspect 的 ready 状态。
 *
 * @param candidate - 运行时候选
 * @returns `true` 表示可以进入 inspect
 */
export function isApplicationImportRuntimeCandidateReady(candidate: ApplicationImportRuntimeCandidate) {
  return candidate.importable && candidate.status === 'ready';
}

/**
 * 生成候选的稳定原因键。
 *
 * @param candidate - 运行时候选
 * @returns 优先返回首个稳定 reason code；否则回退到状态级原因键
 */
export function resolveApplicationImportRuntimeCandidateReasonKey(candidate: ApplicationImportRuntimeCandidate) {
  const [primaryReasonCode] = candidate.status_reason_codes;
  if (primaryReasonCode?.trim()) {
    return primaryReasonCode.trim();
  }

  return candidate.status || 'unavailable';
}
