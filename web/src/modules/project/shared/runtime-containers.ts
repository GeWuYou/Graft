import { getContainers, type ProjectContainerSummary } from '@/modules/container/contract/project';

export const PROJECT_RUNTIME_CONTAINER_PAGE_SIZE = 100;

/**
 * 读取并规范化容器的项目源分组值。
 *
 * @param container - 项目容器摘要
 * @returns 去除首尾空白的分组值；缺少有效值时返回空字符串
 */
export function readProjectContainerSourceGroup(container: ProjectContainerSummary): string {
  return normalizeProjectContainerSourceValue(container.deployment?.project ?? container.orchestrator?.group_value);
}

/**
 * 读取并规范化容器的来源成员值。
 *
 * @param container - 项目容器摘要
 * @returns 裁剪空白后的来源成员值；缺失或为空时返回空字符串
 */
export function readProjectContainerSourceMember(container: ProjectContainerSummary): string {
  return normalizeProjectContainerSourceValue(container.deployment?.service ?? container.orchestrator?.member_value);
}

/**
 * 获取指定 Compose 项目的运行时容器。
 *
 * @param canonicalProjectName - Compose 项目的规范名称；首尾空白会被忽略
 * @returns 与该项目关联的运行时容器列表；项目名称为空时返回空数组
 */
export async function fetchProjectRuntimeContainers(canonicalProjectName: string): Promise<ProjectContainerSummary[]> {
  const projectName = canonicalProjectName.trim();
  if (!projectName) {
    return [];
  }

  const rows: ProjectContainerSummary[] = [];
  let offset = 0;
  let total = 0;

  do {
    const payload = await getContainers({
      limit: PROJECT_RUNTIME_CONTAINER_PAGE_SIZE,
      offset,
      deployment_type: 'compose',
      source_scope: projectName,
      source_scope_kind: 'compose_project',
    });

    rows.push(...payload.items);
    total = payload.total;
    offset += payload.items.length;

    if (!payload.items.length) {
      break;
    }
  } while (rows.length < total);

  return rows;
}

/**
 * 规范化项目容器来源值，移除首尾空白并为空值提供空字符串。
 *
 * @param value - 待规范化的来源值
 * @returns 去除首尾空白后的值；输入缺失或仅包含空白时返回 `''`
 */
function normalizeProjectContainerSourceValue(value?: string | null): string {
  return value?.trim() || '';
}
