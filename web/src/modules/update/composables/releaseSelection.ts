import type { DeploymentStrategy, UpdateChannel, UpdateStatus, VerifiedUpdateRelease } from '../types/update';

const fixedStrategyChannels: Partial<Record<DeploymentStrategy, UpdateChannel>> = {
  pinned_stable: 'stable',
  pinned_beta: 'beta',
};

/**
 * 跟踪策略只以服务端给出的 latest 为目标；固定策略从同通道、版本更高的已验证候选中选择最高版本。
 */
export function getAvailableUpdateRelease(status: UpdateStatus | null | undefined): VerifiedUpdateRelease | undefined {
  if (!status) return undefined;
  if (!isFixedStrategy(status.deployment_strategy)) return status.latest;
  return getFixedUpdateReleaseCandidates(status)[0];
}

export function getFixedUpdateReleaseCandidates(status: UpdateStatus | null | undefined): VerifiedUpdateRelease[] {
  if (!status) return [];
  const channel = fixedStrategyChannels[status.deployment_strategy];
  if (!channel) return [];

  return (status.available_releases ?? [])
    .filter(
      (release) => release.channel === channel && compareReleaseVersions(release.version, status.current_version) > 0,
    )
    .sort((left, right) => compareReleaseVersions(right.version, left.version));
}

export function hasAvailableUpdate(status: UpdateStatus | null | undefined): boolean {
  return Boolean(getAvailableUpdateRelease(status) && !status?.cache_stale && !status?.check_error);
}

export function isFixedStrategy(strategy: DeploymentStrategy): boolean {
  return Boolean(fixedStrategyChannels[strategy]);
}

function compareReleaseVersions(left: string, right: string) {
  const leftParts = parseReleaseVersion(left);
  const rightParts = parseReleaseVersion(right);
  if (!leftParts || !rightParts) return 0;

  for (let index = 0; index < 3; index += 1) {
    const difference = (leftParts[index] ?? 0) - (rightParts[index] ?? 0);
    if (difference !== 0) return difference;
  }

  const leftBeta = leftParts[3];
  const rightBeta = rightParts[3];
  if (leftBeta === null) return rightBeta === null ? 0 : 1;
  if (rightBeta === null) return -1;
  return leftBeta - rightBeta;
}

function parseReleaseVersion(version: string): [number, number, number, number | null] | null {
  const match = /^v?(\d+)\.(\d+)\.(\d+)(?:-beta\.(\d+))?$/.exec(version.trim());
  if (!match) return null;
  return [Number(match[1]), Number(match[2]), Number(match[3]), match[4] ? Number(match[4]) : null];
}
