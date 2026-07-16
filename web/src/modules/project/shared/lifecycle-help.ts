import type { ApplicationLifecycleConfigurationDraft } from '../types/project';

export type LifecycleHelpKey =
  | 'downBeforeRedeploy'
  | 'pullBeforeRedeploy'
  | 'buildBeforeUp'
  | 'forceRecreate'
  | 'waitAfterUp'
  | 'waitTimeoutSeconds'
  | 'removeOrphans'
  | 'renewAnonVolumes'
  | 'pruneImagesAfterRedeploy';

export type LifecycleHelpRecommendation = 'recommended' | 'optional' | 'defaultOff' | 'tunePerEnv' | 'dangerous';

export type LifecycleHelpDefinition = {
  key: LifecycleHelpKey;
  field: keyof Pick<
    ApplicationLifecycleConfigurationDraft,
    | 'down_before_redeploy'
    | 'pull_before_redeploy'
    | 'build_before_up'
    | 'force_recreate'
    | 'remove_orphans'
    | 'wait_after_up'
    | 'wait_timeout_seconds'
    | 'renew_anon_volumes'
    | 'prune_images_after_redeploy'
  >;
  control: 'switch' | 'number';
  titleKey: string;
  summaryKey: string;
  tooltipKey: string;
  detailKeyPrefix: string;
  recommendation: LifecycleHelpRecommendation;
  commandExample: string[] | ((draft: ApplicationLifecycleConfigurationDraft) => string[]);
  visible?: (draft: ApplicationLifecycleConfigurationDraft) => boolean;
  switchTestId?: string;
};

export type LifecycleSwitchHelpDefinition = LifecycleHelpDefinition & {
  control: 'switch';
  field: keyof Pick<
    ApplicationLifecycleConfigurationDraft,
    | 'down_before_redeploy'
    | 'pull_before_redeploy'
    | 'build_before_up'
    | 'force_recreate'
    | 'remove_orphans'
    | 'wait_after_up'
    | 'renew_anon_volumes'
    | 'prune_images_after_redeploy'
  >;
};

export type LifecycleNumberHelpDefinition = LifecycleHelpDefinition & {
  control: 'number';
  field: 'wait_timeout_seconds';
};

/**
 * 生成包含等待超时时间的 Docker Compose 启动命令。
 *
 * @param draft - 项目生命周期配置草稿
 * @returns 包含 Docker Compose 启动命令的单元素数组
 */
function waitTimeoutCommand(draft: ApplicationLifecycleConfigurationDraft) {
  return [`docker compose up -d --wait --wait-timeout ${draft.wait_timeout_seconds}`];
}

export const lifecycleHelpDefinitions: LifecycleHelpDefinition[] = [
  {
    key: 'downBeforeRedeploy',
    field: 'down_before_redeploy',
    control: 'switch',
    titleKey: 'project.detail.lifecycle.downBeforeRedeploy',
    summaryKey: 'project.detail.lifecycle.optionDescriptions.downBeforeRedeploy',
    tooltipKey: 'project.detail.lifecycle.help.items.downBeforeRedeploy.tooltip',
    detailKeyPrefix: 'project.detail.lifecycle.help.items.downBeforeRedeploy',
    recommendation: 'optional',
    commandExample: ['docker compose down'],
  },
  {
    key: 'pullBeforeRedeploy',
    field: 'pull_before_redeploy',
    control: 'switch',
    titleKey: 'project.detail.lifecycle.pullBeforeRedeploy',
    summaryKey: 'project.detail.lifecycle.optionDescriptions.pullBeforeRedeploy',
    tooltipKey: 'project.detail.lifecycle.help.items.pullBeforeRedeploy.tooltip',
    detailKeyPrefix: 'project.detail.lifecycle.help.items.pullBeforeRedeploy',
    recommendation: 'optional',
    commandExample: ['docker compose pull'],
    switchTestId: 'project-lifecycle-pull-before-redeploy-switch',
  },
  {
    key: 'buildBeforeUp',
    field: 'build_before_up',
    control: 'switch',
    titleKey: 'project.detail.lifecycle.buildBeforeUp',
    summaryKey: 'project.detail.lifecycle.optionDescriptions.buildBeforeUp',
    tooltipKey: 'project.detail.lifecycle.help.items.buildBeforeUp.tooltip',
    detailKeyPrefix: 'project.detail.lifecycle.help.items.buildBeforeUp',
    recommendation: 'defaultOff',
    commandExample: ['docker compose up -d --build'],
    switchTestId: 'project-lifecycle-build-before-up-switch',
  },
  {
    key: 'forceRecreate',
    field: 'force_recreate',
    control: 'switch',
    titleKey: 'project.detail.lifecycle.forceRecreate',
    summaryKey: 'project.detail.lifecycle.optionDescriptions.forceRecreate',
    tooltipKey: 'project.detail.lifecycle.help.items.forceRecreate.tooltip',
    detailKeyPrefix: 'project.detail.lifecycle.help.items.forceRecreate',
    recommendation: 'defaultOff',
    commandExample: ['docker compose up -d --force-recreate'],
  },
  {
    key: 'removeOrphans',
    field: 'remove_orphans',
    control: 'switch',
    titleKey: 'project.detail.lifecycle.removeOrphans',
    summaryKey: 'project.detail.lifecycle.optionDescriptions.removeOrphans',
    tooltipKey: 'project.detail.lifecycle.help.items.removeOrphans.tooltip',
    detailKeyPrefix: 'project.detail.lifecycle.help.items.removeOrphans',
    recommendation: 'recommended',
    commandExample: ['docker compose up -d --remove-orphans'],
    switchTestId: 'project-lifecycle-remove-orphans-switch',
  },
  {
    key: 'waitAfterUp',
    field: 'wait_after_up',
    control: 'switch',
    titleKey: 'project.detail.lifecycle.waitAfterUp',
    summaryKey: 'project.detail.lifecycle.optionDescriptions.waitAfterUp',
    tooltipKey: 'project.detail.lifecycle.help.items.waitAfterUp.tooltip',
    detailKeyPrefix: 'project.detail.lifecycle.help.items.waitAfterUp',
    recommendation: 'optional',
    commandExample: waitTimeoutCommand,
    switchTestId: 'project-lifecycle-wait-after-up-switch',
  },
  {
    key: 'waitTimeoutSeconds',
    field: 'wait_timeout_seconds',
    control: 'number',
    titleKey: 'project.detail.lifecycle.waitTimeoutSeconds',
    summaryKey: 'project.detail.lifecycle.optionDescriptions.waitTimeoutSeconds',
    tooltipKey: 'project.detail.lifecycle.help.items.waitTimeoutSeconds.tooltip',
    detailKeyPrefix: 'project.detail.lifecycle.help.items.waitTimeoutSeconds',
    recommendation: 'tunePerEnv',
    commandExample: waitTimeoutCommand,
    visible: (draft) => draft.wait_after_up,
  },
  {
    key: 'renewAnonVolumes',
    field: 'renew_anon_volumes',
    control: 'switch',
    titleKey: 'project.detail.lifecycle.renewAnonVolumes',
    summaryKey: 'project.detail.lifecycle.optionDescriptions.renewAnonVolumes',
    tooltipKey: 'project.detail.lifecycle.help.items.renewAnonVolumes.tooltip',
    detailKeyPrefix: 'project.detail.lifecycle.help.items.renewAnonVolumes',
    recommendation: 'dangerous',
    commandExample: ['docker compose up -d --renew-anon-volumes'],
    switchTestId: 'project-lifecycle-renew-anon-volumes-switch',
  },
  {
    key: 'pruneImagesAfterRedeploy',
    field: 'prune_images_after_redeploy',
    control: 'switch',
    titleKey: 'project.detail.lifecycle.pruneImagesAfterRedeploy',
    summaryKey: 'project.detail.lifecycle.optionDescriptions.pruneImagesAfterRedeploy',
    tooltipKey: 'project.detail.lifecycle.help.items.pruneImagesAfterRedeploy.tooltip',
    detailKeyPrefix: 'project.detail.lifecycle.help.items.pruneImagesAfterRedeploy',
    recommendation: 'defaultOff',
    commandExample: ['docker image prune -f'],
    switchTestId: 'project-lifecycle-prune-images-after-redeploy-switch',
  },
];

export const lifecycleSwitchHelpDefinitions = lifecycleHelpDefinitions.filter(
  (item): item is LifecycleSwitchHelpDefinition => item.control === 'switch',
);

export const lifecycleWaitTimeoutHelpDefinition: LifecycleNumberHelpDefinition = (() => {
  const definition = lifecycleHelpDefinitions.find(
    (item): item is LifecycleNumberHelpDefinition => item.key === 'waitTimeoutSeconds',
  );

  if (!definition) {
    throw new Error('Lifecycle wait timeout help definition is required.');
  }

  return definition;
})();
