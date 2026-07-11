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

/**
 * 构建目录选择引用。
 *
 * @param source - 目录导入来源
 * @param path - 目录路径
 * @returns 包含 `provider`、`root_id` 和规范化后 `path` 的目录引用
 */
export function buildDirectorySelection(source: ProjectImportDirectorySource, path: string): ProjectImportDirectoryRef {
  return {
    provider: source.provider,
    root_id: source.root_id,
    path: normalizeDirectoryPath(path),
  };
}

/**
 * 获取来源的初始目录路径。
 *
 * @param source - 项目导入目录来源
 * @returns 规范化后的初始目录路径
 */
export function initialDirectoryPath(source: ProjectImportDirectorySource) {
  return normalizeDirectoryPath(source.initial_path || '');
}

/**
 * 规范化目录路径格式。
 *
 * @param path - 待规范化的路径
 * @returns 规范化后的路径；当输入为空、仅包含空白字符或为 `.` 时返回空字符串
 */
export function normalizeDirectoryPath(path: string | null | undefined) {
  const normalized = (path || '').trim().replace(/\\/g, '/');
  if (!normalized || normalized === '.') {
    return '';
  }

  return normalized.split('/').filter(Boolean).join('/');
}

/**
 * 将目录路径拆分为段列表。
 *
 * @param path - 要拆分的目录路径
 * @returns 路径段数组；当路径为空时返回空数组
 */
function splitDirectorySegments(path: string) {
  const normalized = normalizeDirectoryPath(path);
  return normalized ? normalized.split('/') : [];
}

/**
 * 生成目录的面包屑路径。
 *
 * @param directory - 要构建面包屑的目录引用
 * @returns 包含根节点和各级目录段的面包屑项数组
 */
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

/**
 * 生成导入来源的显示标签。
 *
 * @param source - 目录导入来源
 * @returns 来源标签；当 `source.managed` 为真时附加 ` (Managed)`
 */
export function buildDirectorySourceLabel(source: ProjectImportDirectorySource) {
  const suffix = source.managed ? ' (Managed)' : '';
  return `${source.label}${suffix}`;
}

/**
 * 生成建议显示名。
 *
 * @param result - 导入检查结果
 * @returns 优先取 `display_name_suggested`，否则取 `canonical_project_name`；两者都为空时返回空字符串。
 */
export function buildSuggestedDisplayName(result: ProjectImportInspectResponse) {
  return (result.display_name_suggested || result.canonical_project_name || '').trim();
}

/**
 * 将未知输入归一化为字符串数组。
 *
 * @param value - 可能来自接口的未知列表字段
 * @returns 过滤后的字符串数组
 */
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

/**
 * 将未知输入归一化为导入检查文件数组。
 *
 * @param value - 可能为 null 的文件列表
 * @returns 过滤后的检查文件数组
 */
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
