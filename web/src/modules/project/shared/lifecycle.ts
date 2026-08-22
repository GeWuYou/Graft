import type { ApplicationImportInspectResponse } from '../types/import';
import type {
  ApplicationDetailResponseWithLifecycle,
  ApplicationLifecycleConfigurationDraft,
  ApplicationLifecycleConfigurationUpdateRequest,
  ApplicationLifecycleReviewStatus,
  ApplicationLifecycleStrategyKind,
  ApplicationListItemWithLifecycle,
  ApplicationSourceType,
} from '../types/project';

const defaultLifecycleWaitTimeoutSeconds = 120;

function splitProfiles(value: string) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
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
  managed_service_names?: string[] | null;
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
  return {
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
    managed_service_names: [...(source.managed_service_names ?? [])],
    declared_service_names: [],
    review_status: options.reviewStatus,
  };
}

function comparableLifecycleDraftState(draft: ApplicationLifecycleConfigurationDraft) {
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
    managed_service_names: [...draft.managed_service_names].sort(),
  };
}

export function buildLifecycleConfigurationDraft(
  detail: ApplicationDetailResponseWithLifecycle,
): ApplicationLifecycleConfigurationDraft {
  const source = detail.lifecycle_configuration;
  return {
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
    managed_service_names: [...(source?.managed_service_names ?? [])],
    declared_service_names: [],
    review_status: normalizeLifecycleReviewStatus(detail.lifecycle_review_status, detail.source_type),
  };
}

export function buildImportLifecycleConfigurationDraft(
  result: ApplicationImportInspectResponse,
): ApplicationLifecycleConfigurationDraft {
  const composeFiles = result.compose_files
    .map((file) => file.display_path || file.absolute_path || '')
    .filter(Boolean);
  const draft = buildLifecycleDraftFromSource(result.lifecycle_configuration, {
    workspacePath: result.resolved_workspace_path,
    composeFiles: composeFiles.length ? composeFiles : ['compose.yaml'],
    composeProjectName: result.compose_project_name,
    reviewStatus: 'review_required',
  });
  if (!draft.managed_service_names.length) {
    draft.managed_service_names = Array.isArray(result.services) ? [...result.services] : [];
  }
  draft.declared_service_names = Array.isArray(result.services) ? [...result.services] : [];
  return draft;
}

/**
 * 将创建预设的生命周期配置转换为向导草稿，保持模板与空白创建的领域策略一致。
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
  return comparableLifecycleDraftState(draft);
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
