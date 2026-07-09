import { getContainers, type ProjectContainerSummary } from '@/modules/container/contract/project';

export const PROJECT_RUNTIME_CONTAINER_PAGE_SIZE = 100;

export function readProjectContainerSourceGroup(container: ProjectContainerSummary): string {
  return normalizeProjectContainerSourceValue(container.orchestrator?.group_value);
}

export function readProjectContainerSourceMember(container: ProjectContainerSummary): string {
  return normalizeProjectContainerSourceValue(container.orchestrator?.member_value);
}

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
      orchestrator: 'compose',
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

function normalizeProjectContainerSourceValue(value?: string | null): string {
  return value?.trim() || '';
}
