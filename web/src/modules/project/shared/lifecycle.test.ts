import { describe, expect, it } from 'vitest';

import {
  buildBlankLifecycleConfigurationDraft,
  buildImportLifecycleConfigurationDraft,
  buildLifecycleConfigurationDraft,
  buildLifecycleConfigurationRequest,
  formatLifecycleCommandCopyText,
  isLifecycleDraftDirty,
  resolveLifecycleCommandSteps,
} from './lifecycle';

function createApplicationDetail(additionalArgs: string[] = []) {
  return {
    compose_project_name: 'compose-demo',
    compose_files: [
      { absolute_path: '/srv/compose-demo/compose.yaml', display_path: 'compose.yaml' },
      { absolute_path: '/srv/compose-demo/compose.override.yaml', display_path: 'compose.override.yaml' },
    ],
    lifecycle_configuration: {
      strategy_kind: 'standard',
      profiles: ['web'],
      down_before_redeploy: true,
      pull_before_redeploy: false,
      build_before_up: false,
      force_recreate: false,
      remove_orphans: true,
      wait_after_up: false,
      wait_timeout_seconds: 120,
      renew_anon_volumes: false,
      prune_images_after_redeploy: false,
      additional_args: additionalArgs,
    },
    lifecycle_review_status: 'confirmed',
    source_type: 'manual',
    workspace_path: '/srv/compose-demo',
  } as never;
}

describe('project lifecycle helpers', () => {
  it('scopes lifecycle previews and per-action argv to selected services', () => {
    const draft = buildLifecycleConfigurationDraft(createApplicationDetail());
    draft.declared_service_names = ['api', 'db', 'worker'];
    draft.managed_service_names = ['api', 'worker'];
    draft.stop_args = '--timeout 30';
    draft.restart_args = '--no-deps';
    draft.pull_args = '--policy always';

    expect(resolveLifecycleCommandSteps(draft, 'stop', { preferClientGenerated: true })[0]?.command).toContain(
      'stop --timeout 30 api worker',
    );
    expect(resolveLifecycleCommandSteps(draft, 'restart', { preferClientGenerated: true })[0]?.command).toContain(
      'restart --no-deps api worker',
    );
    expect(
      resolveLifecycleCommandSteps(draft, 'restart', { preferClientGenerated: true })[0]?.absolute_command,
    ).toContain('restart --no-deps api worker');
    expect(resolveLifecycleCommandSteps(draft, 'redeploy', { preferClientGenerated: true })[0]?.command).toContain(
      'stop --timeout 30 api worker',
    );
    draft.pull_before_redeploy = true;
    expect(resolveLifecycleCommandSteps(draft, 'redeploy', { preferClientGenerated: true })[1]?.command).toContain(
      'pull --policy always api worker',
    );
  });

  it('hydrates blank-create lifecycle defaults into a command-preview draft', () => {
    const draft = buildBlankLifecycleConfigurationDraft(
      {
        lifecycle_configuration: {
          strategy_kind: 'standard',
          profiles: ['web'],
          down_before_redeploy: true,
          pull_before_redeploy: false,
          build_before_up: false,
          force_recreate: false,
          remove_orphans: true,
          wait_after_up: false,
          wait_timeout_seconds: 120,
          renew_anon_volumes: false,
          prune_images_after_redeploy: false,
          additional_args: ['--progress', 'plain'],
        },
      },
      { composeFilePath: 'compose.yaml', composeProjectName: 'orders-dev' },
    );

    expect(draft.additional_args).toBe('--progress plain');
    expect(resolveLifecycleCommandSteps(draft, 'up')).toEqual([
      expect.objectContaining({
        command: 'docker compose -f compose.yaml --profile web -p orders-dev up -d --remove-orphans --progress plain',
      }),
    ]);
  });

  it('preserves additional argv boundaries through lifecycle requests and inspection drafts', () => {
    const firstDraft = buildLifecycleConfigurationDraft(createApplicationDetail());
    firstDraft.additional_args = "--progress 'plain output'";

    expect(resolveLifecycleCommandSteps(firstDraft, 'up')).toEqual([
      {
        title_key: 'project.detail.lifecycle.step.up',
        command:
          "docker compose -f compose.yaml -f compose.override.yaml --profile web -p compose-demo up -d --remove-orphans --progress 'plain output'",
        absolute_command:
          "docker compose -f /srv/compose-demo/compose.yaml -f /srv/compose-demo/compose.override.yaml --profile web -p compose-demo up -d --remove-orphans --progress 'plain output'",
      },
    ]);

    expect(buildLifecycleConfigurationRequest(firstDraft)).toEqual({
      strategy_kind: 'standard',
      profiles: ['web'],
      down_before_redeploy: true,
      pull_before_redeploy: false,
      build_before_up: false,
      force_recreate: false,
      remove_orphans: true,
      wait_after_up: false,
      wait_timeout_seconds: 120,
      renew_anon_volumes: false,
      prune_images_after_redeploy: false,
      additional_args: ['--progress', 'plain output'],
      managed_service_names: [],
      stop_args: [],
      restart_args: [],
      pull_args: [],
    });

    const refreshedDraft = buildLifecycleConfigurationDraft(createApplicationDetail(['--progress', 'plain output']));
    expect(refreshedDraft.additional_args).toBe("--progress 'plain output'");
    expect(resolveLifecycleCommandSteps(refreshedDraft, 'up')).toEqual([
      {
        title_key: 'project.detail.lifecycle.step.up',
        command:
          "docker compose -f compose.yaml -f compose.override.yaml --profile web -p compose-demo up -d --remove-orphans --progress 'plain output'",
        absolute_command:
          "docker compose -f /srv/compose-demo/compose.yaml -f /srv/compose-demo/compose.override.yaml --profile web -p compose-demo up -d --remove-orphans --progress 'plain output'",
      },
    ]);
  });

  it('formats copied lifecycle commands with relative or absolute compose paths', () => {
    const detail = createApplicationDetail() as Record<string, unknown>;
    detail.compose_project_name = 'compose-demo-copy-mode';
    const draft = buildLifecycleConfigurationDraft(detail as never);
    draft.pull_before_redeploy = true;

    const steps = resolveLifecycleCommandSteps(draft, 'redeploy', { preferClientGenerated: true });

    expect(formatLifecycleCommandCopyText(steps)).toBe(
      'docker compose -f compose.yaml -f compose.override.yaml --profile web -p compose-demo-copy-mode down && docker compose -f compose.yaml -f compose.override.yaml --profile web -p compose-demo-copy-mode pull && docker compose -f compose.yaml -f compose.override.yaml --profile web -p compose-demo-copy-mode up -d --remove-orphans',
    );
    expect(formatLifecycleCommandCopyText(steps, { absolutePaths: true })).toBe(
      'docker compose -f /srv/compose-demo/compose.yaml -f /srv/compose-demo/compose.override.yaml --profile web -p compose-demo-copy-mode down && docker compose -f /srv/compose-demo/compose.yaml -f /srv/compose-demo/compose.override.yaml --profile web -p compose-demo-copy-mode pull && docker compose -f /srv/compose-demo/compose.yaml -f /srv/compose-demo/compose.override.yaml --profile web -p compose-demo-copy-mode up -d --remove-orphans',
    );
  });

  it('uses an empty draft value when persisted additional args are absent', () => {
    expect(buildLifecycleConfigurationDraft(createApplicationDetail()).additional_args).toBe('');
  });

  it('applies explicit defaults for newly added lifecycle fields', () => {
    const detail = createApplicationDetail() as Record<string, unknown>;
    const draft = buildLifecycleConfigurationDraft({
      ...detail,
      lifecycle_configuration: {
        strategy_kind: 'standard',
        profiles: ['web'],
        down_before_redeploy: true,
        pull_before_redeploy: false,
        build_before_up: false,
        force_recreate: false,
        wait_after_up: false,
        prune_images_after_redeploy: false,
      },
    } as never);

    expect(draft.remove_orphans).toBe(true);
    expect(draft.wait_timeout_seconds).toBe(120);
    expect(draft.renew_anon_volumes).toBe(false);
  });

  it('normalizes legacy null import profiles to an empty list', () => {
    const draft = buildImportLifecycleConfigurationDraft({
      compose_project_name: 'compose-demo',
      compose_files: [],
      lifecycle_configuration: {
        strategy_kind: 'standard',
        profiles: null,
        down_before_redeploy: true,
        pull_before_redeploy: false,
        build_before_up: false,
        force_recreate: false,
        remove_orphans: true,
        wait_after_up: false,
        wait_timeout_seconds: 120,
        renew_anon_volumes: false,
        prune_images_after_redeploy: false,
        additional_args: [],
      },
      resolved_workspace_path: '/srv/compose-demo',
    } as never);

    expect(draft.profiles).toEqual([]);
  });

  it('hydrates inspection additional args without flattening argv boundaries', () => {
    const draft = buildImportLifecycleConfigurationDraft({
      compose_project_name: 'compose-demo',
      compose_files: [],
      lifecycle_configuration: {
        strategy_kind: 'standard',
        profiles: [],
        down_before_redeploy: true,
        pull_before_redeploy: false,
        build_before_up: false,
        force_recreate: false,
        remove_orphans: true,
        wait_after_up: false,
        wait_timeout_seconds: 120,
        renew_anon_volumes: false,
        prune_images_after_redeploy: false,
        additional_args: ['--label', 'release channel'],
      },
      resolved_workspace_path: '/srv/compose-demo',
    } as never);

    expect(draft.additional_args).toBe("--label 'release channel'");
    expect(buildLifecycleConfigurationRequest(draft).additional_args).toEqual(['--label', 'release channel']);
  });

  it('treats wait-timeout changes as dirty lifecycle state', () => {
    const baseline = buildLifecycleConfigurationDraft(createApplicationDetail());
    const current = buildLifecycleConfigurationDraft(createApplicationDetail());

    current.wait_after_up = true;
    current.wait_timeout_seconds = 180;

    expect(isLifecycleDraftDirty(current, baseline)).toBe(true);
  });
});
