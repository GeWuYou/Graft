import type {
  ProjectImportDirectoryRef,
  ProjectImportDirectorySource,
  ProjectImportInspectResponse,
  ProjectImportRuntimeCandidate,
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
 * 判断导入结果是否存在阻塞性冲突。
 *
 * @param result - 导入检索结果
 * @returns `true` 如果存在冲突，`false` 否则。
 */
export function hasBlockingImportConflicts(result: ProjectImportInspectResponse | null) {
  return Boolean(result?.conflicts?.length);
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

/**
 * 汇总候选的配置文件 authority 列表。
 *
 * `working_directory` 只作为展示 hint；这里返回的 config files 才是更强的导入 authority。
 *
 * @param candidate - 运行时候选
 * @returns 运行时 authority 返回的 config files 列表
 */
export function collectProjectImportRuntimeCandidateConfigFiles(candidate: ProjectImportRuntimeCandidate) {
  return [...candidate.config_files];
}
