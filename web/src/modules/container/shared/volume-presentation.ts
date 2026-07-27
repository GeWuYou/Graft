import type { components } from '@/contracts/openapi/generated/schema';

type Translate = (key: string) => string;
export type DockerVolumeRelationshipStatus = components['schemas']['docker-resource-relationship-status'];

/** 将服务端关系状态收敛为数据卷资产页的三态展示，未知状态不会泄漏为用户文案。 */
export function getDockerVolumeStatusPresentation(t: Translate, status: DockerVolumeRelationshipStatus) {
  if (status === 'used') {
    return { label: t('container.volume.status.inUse'), theme: 'success' as const };
  }
  if (status === 'unused') {
    return { label: t('container.volume.status.unused'), theme: 'warning' as const };
  }
  return { label: t('container.volume.status.abnormal'), theme: 'danger' as const };
}

export function isDockerVolumeSafeCleanupCandidate(status: DockerVolumeRelationshipStatus) {
  return status === 'unused';
}
