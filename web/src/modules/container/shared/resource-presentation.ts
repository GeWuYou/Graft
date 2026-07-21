import type { components } from '@/contracts/openapi/generated/schema';

type Translate = (key: string) => string;
type DockerResourceSource = components['schemas']['docker-resource-source'];
export type DockerResourceRelationshipStatus = components['schemas']['docker-resource-relationship-status'];
type DockerResourceContext = components['schemas']['docker-resource-context'];

/** 将服务端资源来源枚举转换为模块 i18n 标签，避免页面重建同一显示映射。 */
export function getDockerResourceSourceLabel(t: Translate, source: DockerResourceSource) {
  return t(`container.resourceContext.sourceValues.${source}`);
}

/** 将资源上下文转换为列表中的紧凑来源说明，保持网络和数据卷使用同一展示规则。 */
export function getDockerResourceSourceDescription(t: Translate, context: DockerResourceContext) {
  const sourceLabel = getDockerResourceSourceLabel(t, context.source);
  if (context.source === 'compose') {
    const project = context.compose_project || context.compose_resource;
    if (!project) return sourceLabel;
    const resource =
      context.compose_resource && context.compose_resource !== project ? ` / ${context.compose_resource}` : '';
    return `${sourceLabel} · ${project}${resource}`;
  }
  if (context.source === 'managed' || context.source === 'imported' || context.source === 'unknown') return sourceLabel;
  const detail = context.managed_by || context.runtime_target;
  return detail ? `${sourceLabel} · ${detail}` : sourceLabel;
}

/** 将服务端关系状态转换为 TDesign 标签展示，未知和异常状态保留其告警语义。 */
export function getDockerResourceRelationshipPresentation(t: Translate, status: DockerResourceRelationshipStatus) {
  const theme =
    status === 'used'
      ? ('success' as const)
      : status === 'unused'
        ? ('default' as const)
        : status === 'unknown'
          ? ('warning' as const)
          : ('danger' as const);
  return { theme, label: t(`container.resourceContext.relationship.${status}`) };
}

/** 为无关联资源提供状态相关的空态说明。 */
export function getDockerResourceRelationEmptyLabel(t: Translate, status: DockerResourceRelationshipStatus) {
  return status === 'unknown' || status === 'exception'
    ? getDockerResourceRelationshipPresentation(t, status).label
    : t('container.resourceContext.noRelations');
}
