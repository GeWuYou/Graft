import type { components } from '@/contracts/openapi/generated/schema';

type Translate = (key: string) => string;
type DockerResourceSource = components['schemas']['docker-resource-source'];
export type DockerResourceRelationshipStatus = components['schemas']['docker-resource-relationship-status'];

/** 将服务端资源来源枚举转换为模块 i18n 标签，避免页面重建同一显示映射。 */
export function getDockerResourceSourceLabel(t: Translate, source: DockerResourceSource) {
  return t(`container.resourceContext.sourceValues.${source}`);
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
