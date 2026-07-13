import type { ProjectImportInspectResponse } from '../types/import';
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

const defaultLifecycleWaitTimeoutSeconds = 120;

/**
 * 将工作目录前缀从生命周期文件路径中移除。
 *
 * @param path - 待规范化的文件路径
 * @param workingDirectory - 用于匹配并移除的工作目录
 * @returns 去除工作目录前缀后的路径
 */
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

/**
 * 将附加参数字符串解析为命令行参数数组。
 *
 * @param value - 以空白分隔、支持单引号、双引号和反斜杠转义的附加参数字符串
 * @returns 保留每个 argv 边界的参数数组
 */
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

/**
 * 将相对生命周期文件路径解析为相对于工作目录的路径。
 *
 * @param path - 要解析的文件路径
 * @param workingDirectory - 用于补全相对路径的工作目录
 * @returns 去除首尾空白后的绝对路径；绝对路径或缺少工作目录时返回原路径
 */
function resolveAbsoluteLifecycleFilePath(path: string, workingDirectory: string) {
  const value = path.trim();
  const normalizedWorkingDirectory = workingDirectory.trim().replace(/\/+$/g, '');

  if (!value || !normalizedWorkingDirectory || value.startsWith('/')) {
    return value;
  }

  return `${normalizedWorkingDirectory}/${value}`;
}

/**
 * 构建包含 compose 文件、配置文件、项目名称及配置文件路径的 Docker Compose 基础命令参数。
 *
 * @param config - 项目生命周期配置草稿
 * @param absolutePaths - 是否将 compose 文件路径解析为绝对路径
 * @returns Docker Compose 基础命令参数数组
 */
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

/**
 * 构建用于启动项目的 Docker Compose 命令。
 *
 * @param config - 项目生命周期配置草稿
 * @param absolutePaths - 是否使用 compose 文件的绝对路径
 * @returns 拼接后的 Docker Compose 启动命令
 */
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
  return command.map(formatAdditionalArg).join(' ');
}

/**
 * 构建指定生命周期操作的 Docker Compose 命令。
 *
 * @param config - 项目生命周期配置草稿
 * @param action - 要执行的操作
 * @returns 包含 Compose 配置参数和操作名称的命令字符串
 */
function buildSimpleCommand(config: ProjectLifecycleConfigurationDraft, action: 'stop' | 'restart' | 'down') {
  return [...buildComposeBaseCommand(config), action].join(' ');
}

/**
 * 构建指定生命周期操作的 Docker Compose 命令。
 *
 * @param config - 项目生命周期配置草稿
 * @param action - 要执行的操作
 * @param absolutePaths - 是否在命令中使用 compose 文件的绝对路径
 * @returns 拼接后的 Docker Compose 命令字符串
 */
function buildSimpleCommandWithPathMode(
  config: ProjectLifecycleConfigurationDraft,
  action: 'stop' | 'restart' | 'down',
  absolutePaths = false,
) {
  return [...buildComposeBaseCommand(config, absolutePaths), action].join(' ');
}

/**
 * 根据生命周期配置生成项目启动、停止、重启和重新部署命令预览。
 *
 * @param config - 用于生成命令的项目生命周期配置草稿
 * @returns 包含相对路径和绝对路径命令的生命周期命令预览
 */
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

/**
 * 将生命周期步骤类型映射为对应的本地化标题键。
 *
 * @param kind - 生命周期步骤类型
 * @returns 对应的本地化标题键
 */
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

/**
 * 将生成的 Compose 命令参数格式化为命令字符串，并按需归一化文件路径。
 *
 * @param argv - 命令参数数组
 * @param workingDirectory - Compose 文件的工作目录
 * @param absolutePaths - 是否保留绝对路径模式
 * @returns 格式化后的命令字符串
 */
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

/**
 * 将生成的生命周期命令映射为可展示的命令步骤。
 *
 * @param command - 生成的生命周期命令；未提供时返回空数组
 * @param workingDirectory - 用于归一化 Compose 文件路径的工作目录
 * @returns 包含标题键、相对路径命令和绝对路径命令的步骤列表
 */
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

/**
 * 将生成的生命周期命令映射为命令预览。
 *
 * @param config - 包含生成命令配置的项目生命周期配置；缺少该配置时返回 `null`
 * @param workingDirectory - 用于规范化命令中 Compose 文件路径的工作目录
 * @returns 映射后的生命周期命令预览；未提供生成命令时返回 `null`
 */
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

/**
 * 规范化生命周期等待超时时间。
 *
 * @param value - 待规范化的超时时间（秒）
 * @returns 有效的正整数超时时间；输入无效或不大于零时返回默认超时时间
 */
function normalizeWaitTimeoutSeconds(value: number | null | undefined) {
  if (typeof value !== 'number' || !Number.isFinite(value) || value <= 0) {
    return defaultLifecycleWaitTimeoutSeconds;
  }
  return Math.trunc(value);
}

/**
 * 生成用于比较生命周期配置草稿状态的规范化对象。
 *
 * @param draft - 要比较的生命周期配置草稿字段
 * @returns 包含规范化配置值的可比较状态对象
 */
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

/**
 * 根据项目详情构建生命周期配置草稿。
 *
 * @param detail - 包含生命周期配置、Compose 文件及项目来源信息的项目详情
 * @returns 应用默认值和规范化处理后的生命周期配置草稿
 */
export function buildLifecycleConfigurationDraft(
  detail: ProjectDetailResponseWithLifecycle,
): ProjectLifecycleConfigurationDraft {
  const source = detail.lifecycle_configuration;
  const config: ProjectLifecycleConfigurationDraft = {
    strategy_kind: source?.strategy_kind ?? 'standard',
    working_directory: detail.workspace_path,
    compose_files: normalizeComposeFiles(detail),
    canonical_project_name: detail.compose_project_name,
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
    review_status: normalizeLifecycleReviewStatus(detail.lifecycle_review_status, detail.source_kind),
    generated_commands: mapGeneratedCommands(source, detail.workspace_path),
  };

  return {
    ...config,
    generated_commands: config.generated_commands ?? buildClientGeneratedCommands(config),
  };
}

/**
 * 从导入检查快照构建可编辑的生命周期配置草稿。
 *
 * @param result - 导入检查结果，包含生命周期配置、工作目录和 Compose 文件信息
 * @returns 包含待审核状态和客户端生成命令预览的生命周期配置草稿
 */
export function buildImportLifecycleConfigurationDraft(
  result: ProjectImportInspectResponse,
): ProjectLifecycleConfigurationDraft {
  const source = result.lifecycle_configuration;
  const composeFiles = result.compose_files
    .map((file) => file.display_path || file.absolute_path || '')
    .filter(Boolean);
  const config: ProjectLifecycleConfigurationDraft = {
    strategy_kind: source.strategy_kind,
    working_directory: result.resolved_working_directory,
    compose_files: composeFiles.length ? composeFiles : ['compose.yaml'],
    canonical_project_name: result.canonical_project_name,
    profiles: Array.isArray(source.profiles) ? [...source.profiles] : [],
    down_before_redeploy: source.down_before_redeploy,
    pull_before_redeploy: source.pull_before_redeploy,
    build_before_up: source.build_before_up,
    force_recreate: source.force_recreate,
    remove_orphans: source.remove_orphans,
    wait_after_up: source.wait_after_up,
    wait_timeout_seconds: normalizeWaitTimeoutSeconds(source.wait_timeout_seconds),
    renew_anon_volumes: source.renew_anon_volumes,
    prune_images_after_redeploy: source.prune_images_after_redeploy,
    additional_args: formatAdditionalArgs(source.additional_args),
    review_status: 'review_required',
    generated_commands: null,
  };

  return { ...config, generated_commands: buildClientGeneratedCommands(config) };
}

/**
 * 将生命周期配置草稿转换为更新请求。
 *
 * @param draft - 要转换的生命周期配置草稿
 * @returns 包含结构化附加 argv 参数的生命周期配置更新请求
 */
export function buildLifecycleConfigurationRequest(
  draft: ProjectLifecycleConfigurationDraft,
): ProjectLifecycleConfigurationUpdateRequest {
  const { additional_args, ...request } = comparableLifecycleDraftState(draft);
  return {
    ...request,
    additional_args: normalizeAdditionalArgs(additional_args),
  };
}

/**
 * 判断生命周期配置草稿是否相对于基准草稿发生变化。
 *
 * @param current - 当前生命周期配置草稿
 * @param baseline - 用于比较的基准生命周期配置草稿
 * @returns 当前草稿与基准草稿的可比较状态不一致时为 `true`，否则为 `false`
 */
export function isLifecycleDraftDirty(
  current: ProjectLifecycleConfigurationDraft,
  baseline: ProjectLifecycleConfigurationDraft,
) {
  return (
    JSON.stringify(comparableLifecycleDraftState(current)) !== JSON.stringify(comparableLifecycleDraftState(baseline))
  );
}

/**
 * 获取项目生命周期审核状态的本地化标签。
 *
 * @param t - 用于解析本地化文本的翻译函数
 * @param value - 生命周期审核状态
 * @returns 对应审核状态的本地化标签
 */
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

/**
 * 根据输入文本更新生命周期配置草稿中的配置文件。
 *
 * @param draft - 要更新的生命周期配置草稿
 * @param value - 以逗号分隔的配置文件名称
 */
export function updateLifecycleDraftProfiles(draft: ProjectLifecycleConfigurationDraft, value: string) {
  draft.profiles = splitProfiles(value);
}

/**
 * 将生命周期命令步骤合并为可复制的命令文本。
 *
 * @param steps - 要格式化的生命周期命令步骤
 * @param options - 命令格式选项
 * @returns 使用 ` && ` 连接的命令文本
 */
export function formatLifecycleCommandCopyText(
  steps: ProjectLifecycleCommandStep[],
  options?: { absolutePaths?: boolean },
) {
  return steps
    .map((step) => (options?.absolutePaths ? step.absolute_command || step.command : step.command).trim())
    .filter(Boolean)
    .join(' && ');
}

/**
 * 解析指定操作对应的生命周期命令步骤。
 *
 * @param config - 生命周期配置草稿
 * @param action - 要解析的生命周期操作
 * @param options - 命令解析选项
 * @param options.preferClientGenerated - 是否优先使用客户端生成的命令
 * @returns 按优先级选定的命令步骤；不存在可用命令时返回空数组
 */
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
