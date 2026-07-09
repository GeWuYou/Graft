import type {
  ProjectDetailResponseWithLifecycle,
  ProjectLifecycleActionKey,
  ProjectLifecycleCommandPreview,
  ProjectLifecycleCommandStep,
  ProjectLifecycleConfigurationDraft,
  ProjectLifecycleConfigurationModel,
  ProjectLifecycleConfigurationUpdateRequest,
  ProjectLifecycleGeneratedCommand,
  ProjectLifecycleReviewStatus,
  ProjectListItemWithLifecycle,
  ProjectSourceKind,
} from '../types/project';

const lifecycleAdditionalArgsSession = new Map<string, string>();
const defaultLifecycleWaitTimeoutSeconds = 120;

function normalizeLifecycleFilePath(path: string, workingDirectory: string) {
  const value = path.trim();
  const normalizedWorkingDirectory = workingDirectory.trim().replace(/\/+$/g, '');

  if (!value || !normalizedWorkingDirectory) {
    return value;
  }

  const prefix = `${normalizedWorkingDirectory}/`;
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
  return value.trim().split(/\s+/).filter(Boolean);
}

function buildLifecycleSessionKey(
  config: Pick<ProjectLifecycleConfigurationDraft, 'canonical_project_name' | 'compose_files' | 'working_directory'>,
) {
  return [config.working_directory.trim(), config.canonical_project_name.trim(), ...config.compose_files].join('\n');
}

function readLifecycleSessionAdditionalArgs(
  config: Pick<ProjectLifecycleConfigurationDraft, 'canonical_project_name' | 'compose_files' | 'working_directory'>,
) {
  return lifecycleAdditionalArgsSession.get(buildLifecycleSessionKey(config)) ?? '';
}

function persistLifecycleSessionAdditionalArgs(
  config: Pick<
    ProjectLifecycleConfigurationDraft,
    'additional_args' | 'canonical_project_name' | 'compose_files' | 'working_directory'
  >,
) {
  const key = buildLifecycleSessionKey(config);
  const value = config.additional_args.trim();

  if (!value) {
    lifecycleAdditionalArgsSession.delete(key);
    return;
  }

  lifecycleAdditionalArgsSession.set(key, value);
}

function resolveAbsoluteLifecycleFilePath(path: string, workingDirectory: string) {
  const value = path.trim();
  const normalizedWorkingDirectory = workingDirectory.trim().replace(/\/+$/g, '');

  if (!value || !normalizedWorkingDirectory || value.startsWith('/')) {
    return value;
  }

  return `${normalizedWorkingDirectory}/${value}`;
}

function buildComposeBaseCommand(config: ProjectLifecycleConfigurationDraft, absolutePaths = false) {
  const command = ['docker', 'compose'];
  const normalizePath = absolutePaths ? resolveAbsoluteLifecycleFilePath : normalizeLifecycleFilePath;

  for (const file of config.compose_files) {
    command.push('-f', normalizePath(file, config.working_directory));
  }

  for (const profile of config.profiles) {
    command.push('--profile', profile);
  }

  if (config.canonical_project_name.trim()) {
    command.push('-p', config.canonical_project_name.trim());
  }

  return command;
}

function buildUpCommand(config: ProjectLifecycleConfigurationDraft, absolutePaths = false) {
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
  return command.join(' ');
}

function buildSimpleCommand(config: ProjectLifecycleConfigurationDraft, action: 'stop' | 'restart' | 'down') {
  return [...buildComposeBaseCommand(config), action].join(' ');
}

function buildSimpleCommandWithPathMode(
  config: ProjectLifecycleConfigurationDraft,
  action: 'stop' | 'restart' | 'down',
  absolutePaths = false,
) {
  return [...buildComposeBaseCommand(config, absolutePaths), action].join(' ');
}

function buildClientGeneratedCommands(config: ProjectLifecycleConfigurationDraft): ProjectLifecycleCommandPreview {
  const commands: ProjectLifecycleCommandPreview = {
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
  sourceKind: ProjectSourceKind,
): ProjectLifecycleReviewStatus {
  if (value === 'review_required' || value === 'confirmed') {
    return value;
  }
  return sourceKind === 'imported' ? 'review_required' : 'confirmed';
}

function normalizeComposeFiles(detail: Pick<ProjectDetailResponseWithLifecycle, 'compose_files'>) {
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

function normalizeGeneratedArgvCommand(argv: string[], workingDirectory: string, absolutePaths: boolean) {
  return argv
    .map((arg, index) => {
      if (!absolutePaths && index > 0 && argv[index - 1] === '-f') {
        return normalizeLifecycleFilePath(arg, workingDirectory);
      }

      return arg;
    })
    .join(' ');
}

function mapGeneratedCommand(
  command: ProjectLifecycleGeneratedCommand | undefined,
  workingDirectory: string,
): ProjectLifecycleCommandStep[] {
  if (!command) {
    return [];
  }
  return command.steps.map((step) => ({
    title_key: lifecycleStepTitleKey(step.kind),
    command: step.argv.length
      ? normalizeGeneratedArgvCommand(step.argv, workingDirectory, false)
      : step.display_command,
    absolute_command: step.argv.length
      ? normalizeGeneratedArgvCommand(step.argv, workingDirectory, true)
      : step.display_command,
  }));
}

function mapGeneratedCommands(
  config: Pick<ProjectLifecycleConfigurationModel, 'generated_commands'> | null | undefined,
  workingDirectory: string,
): ProjectLifecycleCommandPreview | null {
  if (!config?.generated_commands) {
    return null;
  }
  return {
    up: mapGeneratedCommand(config.generated_commands.up, workingDirectory),
    stop: mapGeneratedCommand(config.generated_commands.stop, workingDirectory),
    restart: mapGeneratedCommand(config.generated_commands.restart, workingDirectory),
    redeploy: mapGeneratedCommand(config.generated_commands.redeploy, workingDirectory),
  };
}

function normalizeWaitTimeoutSeconds(value: number | null | undefined) {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    return defaultLifecycleWaitTimeoutSeconds;
  }
  return Math.trunc(value);
}

function comparableLifecycleDraftState(
  draft: Pick<
    ProjectLifecycleConfigurationDraft,
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
  detail: ProjectDetailResponseWithLifecycle,
): ProjectLifecycleConfigurationDraft {
  const source = detail.lifecycle_configuration;
  const sessionAdditionalArgs = readLifecycleSessionAdditionalArgs({
    canonical_project_name: detail.canonical_project_name,
    compose_files: normalizeComposeFiles(detail),
    working_directory: detail.working_directory,
  });
  const config: ProjectLifecycleConfigurationDraft = {
    strategy_kind: source?.strategy_kind ?? 'standard',
    working_directory: detail.working_directory,
    compose_files: normalizeComposeFiles(detail),
    canonical_project_name: detail.canonical_project_name,
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
    additional_args: sessionAdditionalArgs,
    review_status: normalizeLifecycleReviewStatus(detail.lifecycle_review_status, detail.source_kind),
    generated_commands: mapGeneratedCommands(source, detail.working_directory),
  };

  return {
    ...config,
    generated_commands: config.generated_commands ?? buildClientGeneratedCommands(config),
  };
}

export function buildLifecycleConfigurationRequest(
  draft: ProjectLifecycleConfigurationDraft,
): ProjectLifecycleConfigurationUpdateRequest {
  persistLifecycleSessionAdditionalArgs(draft);
  const { additional_args: _additionalArgs, ...request } = comparableLifecycleDraftState(draft);
  return {
    ...request,
  };
}

export function isLifecycleDraftDirty(
  current: ProjectLifecycleConfigurationDraft,
  baseline: ProjectLifecycleConfigurationDraft,
) {
  return (
    JSON.stringify(comparableLifecycleDraftState(current)) !== JSON.stringify(comparableLifecycleDraftState(baseline))
  );
}

export function projectLifecycleReviewStatusLabel(
  t: (key: string) => string,
  value: ProjectLifecycleReviewStatus | null | undefined,
) {
  return value === 'review_required'
    ? t('project.lifecycle.reviewStatus.reviewRequired')
    : t('project.lifecycle.reviewStatus.confirmed');
}

export function projectLifecycleReviewStatusTheme(value: ProjectLifecycleReviewStatus | null | undefined) {
  return value === 'review_required' ? 'warning' : 'success';
}

export function projectRequiresLifecycleReview(
  project: Pick<ProjectListItemWithLifecycle, 'lifecycle_review_status' | 'source_kind'>,
) {
  return normalizeLifecycleReviewStatus(project.lifecycle_review_status, project.source_kind) === 'review_required';
}

export function lifecycleDraftProfilesText(config: Pick<ProjectLifecycleConfigurationDraft, 'profiles'>) {
  return config.profiles.join(', ');
}

export function updateLifecycleDraftProfiles(draft: ProjectLifecycleConfigurationDraft, value: string) {
  draft.profiles = splitProfiles(value);
}

export function formatLifecycleCommandCopyText(
  steps: ProjectLifecycleCommandStep[],
  options?: { absolutePaths?: boolean },
) {
  return steps
    .map((step) => (options?.absolutePaths ? step.absolute_command || step.command : step.command).trim())
    .filter(Boolean)
    .join(' && ');
}

export function resolveLifecycleCommandSteps(
  config: ProjectLifecycleConfigurationDraft,
  action: ProjectLifecycleActionKey,
  options?: { preferClientGenerated?: boolean },
): ProjectLifecycleCommandStep[] {
  if (options?.preferClientGenerated || config.additional_args.trim()) {
    return buildClientGeneratedCommands(config)[action] ?? [];
  }

  return config.generated_commands?.[action] ?? buildClientGeneratedCommands(config)[action] ?? [];
}
