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

function buildComposeBaseCommand(config: ProjectLifecycleConfigurationDraft) {
  const command = ['docker', 'compose'];

  for (const file of config.compose_files) {
    command.push('-f', normalizeLifecycleFilePath(file, config.working_directory));
  }

  for (const profile of config.profiles) {
    command.push('--profile', profile);
  }

  if (config.canonical_project_name.trim()) {
    command.push('-p', config.canonical_project_name.trim());
  }

  return command;
}

function buildUpCommand(config: ProjectLifecycleConfigurationDraft) {
  const command = [...buildComposeBaseCommand(config), 'up', '-d'];

  if (config.build_before_up) {
    command.push('--build');
  }
  if (config.force_recreate) {
    command.push('--force-recreate');
  }
  if (config.wait_after_up) {
    command.push('--wait');
  }

  command.push(...normalizeAdditionalArgs(config.additional_args));
  return command.join(' ');
}

function buildSimpleCommand(config: ProjectLifecycleConfigurationDraft, action: 'stop' | 'restart' | 'down') {
  return [...buildComposeBaseCommand(config), action].join(' ');
}

function buildClientGeneratedCommands(config: ProjectLifecycleConfigurationDraft): ProjectLifecycleCommandPreview {
  const commands: ProjectLifecycleCommandPreview = {
    up: [{ title_key: 'project.detail.lifecycle.step.up', command: buildUpCommand(config) }],
    stop: [{ title_key: 'project.detail.lifecycle.step.stop', command: buildSimpleCommand(config, 'stop') }],
    restart: [{ title_key: 'project.detail.lifecycle.step.restart', command: buildSimpleCommand(config, 'restart') }],
    redeploy: [],
  };

  if (config.pull_before_redeploy) {
    commands.redeploy?.push({
      title_key: 'project.detail.lifecycle.step.pullImages',
      command: [...buildComposeBaseCommand(config), 'pull'].join(' '),
    });
  }

  if (config.down_before_redeploy) {
    commands.redeploy?.push({
      title_key: 'project.detail.lifecycle.step.bringDown',
      command: buildSimpleCommand(config, 'down'),
    });
  }

  commands.redeploy?.push({
    title_key: 'project.detail.lifecycle.step.bringUp',
    command: buildUpCommand(config),
  });

  if (config.prune_images_after_redeploy) {
    commands.redeploy?.push({
      title_key: 'project.detail.lifecycle.step.pruneImages',
      command: 'docker image prune -f',
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

function mapGeneratedCommand(command: ProjectLifecycleGeneratedCommand | undefined): ProjectLifecycleCommandStep[] {
  if (!command) {
    return [];
  }
  return command.steps.map((step) => ({
    title_key: lifecycleStepTitleKey(step.kind),
    command: step.display_command,
  }));
}

function mapGeneratedCommands(
  config: Pick<ProjectLifecycleConfigurationModel, 'generated_commands'> | null | undefined,
): ProjectLifecycleCommandPreview | null {
  if (!config?.generated_commands) {
    return null;
  }
  return {
    up: mapGeneratedCommand(config.generated_commands.up),
    stop: mapGeneratedCommand(config.generated_commands.stop),
    restart: mapGeneratedCommand(config.generated_commands.restart),
    redeploy: mapGeneratedCommand(config.generated_commands.redeploy),
  };
}

export function buildLifecycleConfigurationDraft(
  detail: ProjectDetailResponseWithLifecycle,
): ProjectLifecycleConfigurationDraft {
  const source = detail.lifecycle_configuration;
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
    wait_after_up: source?.wait_after_up ?? false,
    prune_images_after_redeploy: source?.prune_images_after_redeploy ?? false,
    additional_args: '',
    review_status: normalizeLifecycleReviewStatus(detail.lifecycle_review_status, detail.source_kind),
    generated_commands: mapGeneratedCommands(source),
  };

  return {
    ...config,
    generated_commands: config.generated_commands ?? buildClientGeneratedCommands(config),
  };
}

export function buildLifecycleConfigurationRequest(
  draft: ProjectLifecycleConfigurationDraft,
): ProjectLifecycleConfigurationUpdateRequest {
  return {
    strategy_kind: draft.strategy_kind,
    profiles: draft.profiles.map((item) => item.trim()).filter(Boolean),
    down_before_redeploy: draft.down_before_redeploy,
    pull_before_redeploy: draft.pull_before_redeploy,
    build_before_up: draft.build_before_up,
    force_recreate: draft.force_recreate,
    wait_after_up: draft.wait_after_up,
    prune_images_after_redeploy: draft.prune_images_after_redeploy,
  };
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

export function resolveLifecycleCommandSteps(
  config: ProjectLifecycleConfigurationDraft,
  action: ProjectLifecycleActionKey,
): ProjectLifecycleCommandStep[] {
  return config.generated_commands?.[action] ?? buildClientGeneratedCommands(config)[action] ?? [];
}
