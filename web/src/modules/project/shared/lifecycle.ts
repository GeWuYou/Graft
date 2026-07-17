import type { ApplicationImportInspectResponse } from '../types/import';
import type {
  ApplicationDetailResponseWithLifecycle,
  ApplicationLifecycleActionKey,
  ApplicationLifecycleCommandPreview,
  ApplicationLifecycleCommandStep,
  ApplicationLifecycleConfigurationDraft,
  ApplicationLifecycleConfigurationModel,
  ApplicationLifecycleConfigurationUpdateRequest,
  ApplicationLifecycleGeneratedCommand,
  ApplicationLifecycleReviewStatus,
  ApplicationLifecycleStrategyKind,
  ApplicationListItemWithLifecycle,
  ApplicationSourceType,
} from '../types/project';

const defaultLifecycleWaitTimeoutSeconds = 120;

function normalizeLifecycleFilePath(path: string, workspacePath?: string | null) {
  const value = path.trim();
  const normalizedWorkspacePath = (workspacePath ?? '').trim().replace(/\/+$/g, '');

  if (!value || !normalizedWorkspacePath) {
    return value;
  }

  const prefix = `${normalizedWorkspacePath}/`;
  if (value.startsWith(prefix)) {
    return value.slice(prefix.length);
  }

  return value;
}

function splitProfiles(value: string) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function normalizeAdditionalArgs(value: string) {
  const args: string[] = [];
  let current = '';
  let quote: '"' | "'" | null = null;
  let escaping = false;
  let tokenStarted = false;

  for (const character of value.trim()) {
    if (escaping) {
      current += character;
      escaping = false;
      tokenStarted = true;
      continue;
    }
    if (character === '\\') {
      escaping = true;
      tokenStarted = true;
      continue;
    }
    if (quote) {
      if (character === quote) {
        quote = null;
      } else {
        current += character;
      }
      tokenStarted = true;
      continue;
    }
    if (character === '"' || character === "'") {
      quote = character;
      tokenStarted = true;
      continue;
    }
    if (/\s/.test(character)) {
      if (tokenStarted) {
        args.push(current);
        current = '';
        tokenStarted = false;
      }
      continue;
    }
    current += character;
    tokenStarted = true;
  }

  if (escaping) {
    current += '\\';
  }
  if (tokenStarted) {
    args.push(current);
  }

  return args;
}

function formatAdditionalArg(value: string) {
  if (!value || /[\s'"\\]/.test(value)) {
    return `'${value.replace(/'/g, "'\\\\''")}'`;
  }
  return value;
}

function formatAdditionalArgs(values: readonly string[] | null | undefined) {
  return values?.map(formatAdditionalArg).join(' ') ?? '';
}

function resolveAbsoluteLifecycleFilePath(path: string, workspacePath?: string | null) {
  const value = path.trim();
  const normalizedWorkspacePath = (workspacePath ?? '').trim().replace(/\/+$/g, '');

  if (!value || !normalizedWorkspacePath || value.startsWith('/')) {
    return value;
  }

  return `${normalizedWorkspacePath}/${value}`;
}

function buildComposeBaseCommand(config: ApplicationLifecycleConfigurationDraft, absolutePaths = false) {
  const command = ['docker', 'compose'];
  const normalizePath = absolutePaths ? resolveAbsoluteLifecycleFilePath : normalizeLifecycleFilePath;

  for (const file of config.compose_files) {
    command.push('-f', normalizePath(file, config.workspace_path));
  }

  for (const profile of config.profiles) {
    command.push('--profile', profile);
  }

  if (config.compose_project_name.trim()) {
    command.push('-p', config.compose_project_name.trim());
  }

  return command;
}

function buildUpCommand(config: ApplicationLifecycleConfigurationDraft, absolutePaths = false) {
  const command = [...buildComposeBaseCommand(config, absolutePaths), 'up', '-d'];

  if (config.build_before_up) {
    command.push('--build');
  }
  if (config.force_recreate) {
    command.push('--force-recreate');
  }
  if (config.remove_orphans) {
    command.push('--remove-orphans');
  }
  if (config.renew_anon_volumes) {
    command.push('--renew-anon-volumes');
  }
  if (config.wait_after_up) {
    command.push('--wait');
    command.push('--wait-timeout', String(config.wait_timeout_seconds));
  }

  command.push(...normalizeAdditionalArgs(config.additional_args));
  return command.map(formatAdditionalArg).join(' ');
}

function buildSimpleCommand(config: ApplicationLifecycleConfigurationDraft, action: 'stop' | 'restart' | 'down') {
  return [...buildComposeBaseCommand(config), action].join(' ');
}

function buildSimpleCommandWithPathMode(
  config: ApplicationLifecycleConfigurationDraft,
  action: 'stop' | 'restart' | 'down',
  absolutePaths = false,
) {
  return [...buildComposeBaseCommand(config, absolutePaths), action].join(' ');
}

function buildClientGeneratedCommands(
  config: ApplicationLifecycleConfigurationDraft,
): ApplicationLifecycleCommandPreview {
  const commands: ApplicationLifecycleCommandPreview = {
    up: [
      {
        title_key: 'project.detail.lifecycle.step.up',
        command: buildUpCommand(config),
        absolute_command: buildUpCommand(config, true),
      },
    ],
    stop: [
      {
        title_key: 'project.detail.lifecycle.step.stop',
        command: buildSimpleCommand(config, 'stop'),
        absolute_command: buildSimpleCommandWithPathMode(config, 'stop', true),
      },
    ],
    restart: [
      {
        title_key: 'project.detail.lifecycle.step.restart',
        command: buildSimpleCommand(config, 'restart'),
        absolute_command: buildSimpleCommandWithPathMode(config, 'restart', true),
      },
    ],
    redeploy: [],
  };

  if (config.down_before_redeploy) {
    commands.redeploy?.push({
      title_key: 'project.detail.lifecycle.step.bringDown',
      command: buildSimpleCommand(config, 'down'),
      absolute_command: buildSimpleCommandWithPathMode(config, 'down', true),
    });
  }

  if (config.pull_before_redeploy) {
    commands.redeploy?.push({
      title_key: 'project.detail.lifecycle.step.pullImages',
      command: [...buildComposeBaseCommand(config), 'pull'].join(' '),
      absolute_command: [...buildComposeBaseCommand(config, true), 'pull'].join(' '),
    });
  }

  commands.redeploy?.push({
    title_key: 'project.detail.lifecycle.step.bringUp',
    command: buildUpCommand(config),
    absolute_command: buildUpCommand(config, true),
  });

  if (config.prune_images_after_redeploy) {
    commands.redeploy?.push({
      title_key: 'project.detail.lifecycle.step.pruneImages',
      command: 'docker image prune -f',
      absolute_command: 'docker image prune -f',
    });
  }

  return commands;
}

function normalizeLifecycleReviewStatus(
  value: string | null | undefined,
  sourceType: ApplicationSourceType,
): ApplicationLifecycleReviewStatus {
  if (value === 'review_required' || value === 'confirmed') {
    return value;
  }
  return sourceType === 'imported' ? 'review_required' : 'confirmed';
}

function normalizeComposeFiles(detail: Pick<ApplicationDetailResponseWithLifecycle, 'compose_files'>) {
  const files = detail.compose_files.map((file) => file.display_path || file.absolute_path || '').filter(Boolean);

  return files.length > 0 ? files : ['compose.yaml'];
}

function lifecycleStepTitleKey(kind: string) {
  switch (kind) {
    case 'down':
      return 'project.detail.lifecycle.step.bringDown';
    case 'pull':
      return 'project.detail.lifecycle.step.pullImages';
    case 'restart':
      return 'project.detail.lifecycle.step.restart';
    case 'stop':
      return 'project.detail.lifecycle.step.stop';
    case 'prune':
      return 'project.detail.lifecycle.step.pruneImages';
    case 'up':
    default:
      return 'project.detail.lifecycle.step.up';
  }
}

function normalizeGeneratedArgvCommand(argv: string[], workspacePath: string, absolutePaths: boolean) {
  return argv
    .map((arg, index) => {
      if (!absolutePaths && index > 0 && argv[index - 1] === '-f') {
        return normalizeLifecycleFilePath(arg, workspacePath);
      }

      return arg;
    })
    .join(' ');
}

function mapGeneratedCommand(
  command: ApplicationLifecycleGeneratedCommand | undefined,
  workspacePath: string,
): ApplicationLifecycleCommandStep[] {
  if (!command) {
    return [];
  }
  return command.steps.map((step) => ({
    title_key: lifecycleStepTitleKey(step.kind),
    command: step.argv.length ? normalizeGeneratedArgvCommand(step.argv, workspacePath, false) : step.display_command,
    absolute_command: step.argv.length
      ? normalizeGeneratedArgvCommand(step.argv, workspacePath, true)
      : step.display_command,
  }));
}

function mapGeneratedCommands(
  config: Pick<ApplicationLifecycleConfigurationModel, 'generated_commands'> | null | undefined,
  workspacePath: string,
): ApplicationLifecycleCommandPreview | null {
  if (!config?.generated_commands) {
    return null;
  }
  return {
    up: mapGeneratedCommand(config.generated_commands.up, workspacePath),
    stop: mapGeneratedCommand(config.generated_commands.stop, workspacePath),
    restart: mapGeneratedCommand(config.generated_commands.restart, workspacePath),
    redeploy: mapGeneratedCommand(config.generated_commands.redeploy, workspacePath),
  };
}

function normalizeWaitTimeoutSeconds(value: number | null | undefined) {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    return defaultLifecycleWaitTimeoutSeconds;
  }
  return Math.trunc(value);
}

export type LifecycleDraftSource = {
  strategy_kind?: ApplicationLifecycleStrategyKind;
  profiles?: string[] | null;
  down_before_redeploy?: boolean;
  pull_before_redeploy?: boolean;
  build_before_up?: boolean;
  force_recreate?: boolean;
  remove_orphans?: boolean;
  wait_after_up?: boolean;
  wait_timeout_seconds?: number | null;
  renew_anon_volumes?: boolean;
  prune_images_after_redeploy?: boolean;
  additional_args?: string[] | null;
};

function buildLifecycleDraftFromSource(
  source: LifecycleDraftSource,
  options: {
    workspacePath: string;
    composeFiles: string[];
    composeProjectName: string;
    reviewStatus: ApplicationLifecycleReviewStatus;
  },
): ApplicationLifecycleConfigurationDraft {
  const config: ApplicationLifecycleConfigurationDraft = {
    strategy_kind: source.strategy_kind ?? 'standard',
    workspace_path: options.workspacePath,
    compose_files: options.composeFiles,
    compose_project_name: options.composeProjectName,
    profiles: Array.isArray(source.profiles) ? [...source.profiles] : [],
    down_before_redeploy: source.down_before_redeploy ?? true,
    pull_before_redeploy: source.pull_before_redeploy ?? false,
    build_before_up: source.build_before_up ?? false,
    force_recreate: source.force_recreate ?? false,
    remove_orphans: source.remove_orphans ?? true,
    wait_after_up: source.wait_after_up ?? false,
    wait_timeout_seconds: normalizeWaitTimeoutSeconds(source.wait_timeout_seconds),
    renew_anon_volumes: source.renew_anon_volumes ?? false,
    prune_images_after_redeploy: source.prune_images_after_redeploy ?? false,
    additional_args: formatAdditionalArgs(source.additional_args ?? []),
    review_status: options.reviewStatus,
    generated_commands: null,
  };

  return { ...config, generated_commands: buildClientGeneratedCommands(config) };
}

function comparableLifecycleDraftState(
  draft: Pick<
    ApplicationLifecycleConfigurationDraft,
    | 'strategy_kind'
    | 'profiles'
    | 'down_before_redeploy'
    | 'pull_before_redeploy'
    | 'build_before_up'
    | 'force_recreate'
    | 'remove_orphans'
    | 'wait_after_up'
    | 'wait_timeout_seconds'
    | 'renew_anon_volumes'
    | 'prune_images_after_redeploy'
    | 'additional_args'
  >,
) {
  return {
    strategy_kind: draft.strategy_kind,
    profiles: draft.profiles.map((item) => item.trim()).filter(Boolean),
    down_before_redeploy: draft.down_before_redeploy,
    pull_before_redeploy: draft.pull_before_redeploy,
    build_before_up: draft.build_before_up,
    force_recreate: draft.force_recreate,
    remove_orphans: draft.remove_orphans,
    wait_after_up: draft.wait_after_up,
    wait_timeout_seconds: normalizeWaitTimeoutSeconds(draft.wait_timeout_seconds),
    renew_anon_volumes: draft.renew_anon_volumes,
    prune_images_after_redeploy: draft.prune_images_after_redeploy,
    additional_args: draft.additional_args.trim(),
  };
}

export function buildLifecycleConfigurationDraft(
  detail: ApplicationDetailResponseWithLifecycle,
): ApplicationLifecycleConfigurationDraft {
  const source = detail.lifecycle_configuration;
  const config: ApplicationLifecycleConfigurationDraft = {
    strategy_kind: source?.strategy_kind ?? 'standard',
    workspace_path: detail.workspace_path,
    compose_files: normalizeComposeFiles(detail),
    compose_project_name: detail.compose_project_name,
    profiles: source?.profiles ?? [],
    down_before_redeploy: source?.down_before_redeploy ?? true,
    pull_before_redeploy: source?.pull_before_redeploy ?? false,
    build_before_up: source?.build_before_up ?? false,
    force_recreate: source?.force_recreate ?? false,
    remove_orphans: source?.remove_orphans ?? true,
    wait_after_up: source?.wait_after_up ?? false,
    wait_timeout_seconds: normalizeWaitTimeoutSeconds(source?.wait_timeout_seconds),
    renew_anon_volumes: source?.renew_anon_volumes ?? false,
    prune_images_after_redeploy: source?.prune_images_after_redeploy ?? false,
    additional_args: formatAdditionalArgs(source?.additional_args),
    review_status: normalizeLifecycleReviewStatus(detail.lifecycle_review_status, detail.source_type),
    generated_commands: mapGeneratedCommands(source, detail.workspace_path),
  };

  return {
    ...config,
    generated_commands: config.generated_commands ?? buildClientGeneratedCommands(config),
  };
}

export function buildImportLifecycleConfigurationDraft(
  result: ApplicationImportInspectResponse,
): ApplicationLifecycleConfigurationDraft {
  const composeFiles = result.compose_files
    .map((file) => file.display_path || file.absolute_path || '')
    .filter(Boolean);
  return buildLifecycleDraftFromSource(result.lifecycle_configuration, {
    workspacePath: result.resolved_workspace_path,
    composeFiles: composeFiles.length ? composeFiles : ['compose.yaml'],
    composeProjectName: result.compose_project_name,
    reviewStatus: 'review_required',
  });
}

/**
 * 将创建预设的生命周期配置转换为向导草稿，确保模板与空白创建共用命令预览规则。
 */
export function buildBlankLifecycleConfigurationDraft(
  defaults: { lifecycle_configuration: LifecycleDraftSource },
  options: { composeFilePath: string; composeProjectName: string; workspacePath?: string },
): ApplicationLifecycleConfigurationDraft {
  return buildLifecycleDraftFromSource(defaults.lifecycle_configuration, {
    workspacePath: options.workspacePath ?? '',
    composeFiles: [options.composeFilePath],
    composeProjectName: options.composeProjectName,
    reviewStatus: 'confirmed',
  });
}

export function buildLifecycleConfigurationRequest(
  draft: ApplicationLifecycleConfigurationDraft,
): ApplicationLifecycleConfigurationUpdateRequest {
  const { additional_args, ...request } = comparableLifecycleDraftState(draft);
  return {
    ...request,
    additional_args: normalizeAdditionalArgs(additional_args),
  };
}

export function isLifecycleDraftDirty(
  current: ApplicationLifecycleConfigurationDraft,
  baseline: ApplicationLifecycleConfigurationDraft,
) {
  return (
    JSON.stringify(comparableLifecycleDraftState(current)) !== JSON.stringify(comparableLifecycleDraftState(baseline))
  );
}

export function projectLifecycleReviewStatusLabel(
  t: (key: string) => string,
  value: ApplicationLifecycleReviewStatus | null | undefined,
) {
  return value === 'review_required'
    ? t('project.lifecycle.reviewStatus.reviewRequired')
    : t('project.lifecycle.reviewStatus.confirmed');
}

export function projectLifecycleReviewStatusTheme(value: ApplicationLifecycleReviewStatus | null | undefined) {
  return value === 'review_required' ? 'warning' : 'success';
}

export function projectRequiresLifecycleReview(
  project: Pick<ApplicationListItemWithLifecycle, 'lifecycle_review_status' | 'source_type'>,
) {
  return normalizeLifecycleReviewStatus(project.lifecycle_review_status, project.source_type) === 'review_required';
}

export function lifecycleDraftProfilesText(config: Pick<ApplicationLifecycleConfigurationDraft, 'profiles'>) {
  return config.profiles.join(', ');
}

export function updateLifecycleDraftProfiles(draft: ApplicationLifecycleConfigurationDraft, value: string) {
  draft.profiles = splitProfiles(value);
}

export function formatLifecycleCommandCopyText(
  steps: ApplicationLifecycleCommandStep[],
  options?: { absolutePaths?: boolean },
) {
  return steps
    .map((step) => (options?.absolutePaths ? step.absolute_command || step.command : step.command).trim())
    .filter(Boolean)
    .join(' && ');
}

export function resolveLifecycleCommandSteps(
  config: ApplicationLifecycleConfigurationDraft,
  action: ApplicationLifecycleActionKey,
  options?: { preferClientGenerated?: boolean },
): ApplicationLifecycleCommandStep[] {
  if (options?.preferClientGenerated || config.additional_args.trim()) {
    return buildClientGeneratedCommands(config)[action] ?? [];
  }

  return config.generated_commands?.[action] ?? buildClientGeneratedCommands(config)[action] ?? [];
}
