import { describe, expect, it } from 'vitest';

import {
  buildLifecycleConfigurationDraft,
  buildLifecycleConfigurationRequest,
  isLifecycleDraftDirty,
  resolveLifecycleCommandSteps,
} from './lifecycle';

function createProjectDetail() {
  return {
    canonical_project_name: 'compose-demo',
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
    },
    lifecycle_review_status: 'confirmed',
    source_kind: 'manual',
    working_directory: '/srv/compose-demo',
  } as never;
}

describe('project lifecycle helpers', () => {
  it('keeps additional args as session-only draft state across saves', () => {
    const firstDraft = buildLifecycleConfigurationDraft(createProjectDetail());
    firstDraft.additional_args = '--progress plain';

    expect(resolveLifecycleCommandSteps(firstDraft, 'up')).toEqual([
      {
        title_key: 'project.detail.lifecycle.step.up',
        command:
          'docker compose -f compose.yaml -f compose.override.yaml --profile web -p compose-demo up -d --remove-orphans --progress plain',
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
    });

    const refreshedDraft = buildLifecycleConfigurationDraft(createProjectDetail());
    expect(refreshedDraft.additional_args).toBe('--progress plain');
    expect(resolveLifecycleCommandSteps(refreshedDraft, 'up')).toEqual([
      {
        title_key: 'project.detail.lifecycle.step.up',
        command:
          'docker compose -f compose.yaml -f compose.override.yaml --profile web -p compose-demo up -d --remove-orphans --progress plain',
      },
    ]);
  });

  it('clears session additional args when the saved draft no longer carries them', () => {
    const draft = buildLifecycleConfigurationDraft(createProjectDetail());
    draft.additional_args = '--wait';
    buildLifecycleConfigurationRequest(draft);

    const clearedDraft = buildLifecycleConfigurationDraft(createProjectDetail());
    clearedDraft.additional_args = '   ';
    buildLifecycleConfigurationRequest(clearedDraft);

    expect(buildLifecycleConfigurationDraft(createProjectDetail()).additional_args).toBe('');
  });

  it('applies explicit defaults for newly added lifecycle fields', () => {
    const detail = createProjectDetail() as Record<string, unknown>;
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

  it('treats wait-timeout changes as dirty lifecycle state', () => {
    const baseline = buildLifecycleConfigurationDraft(createProjectDetail());
    const current = buildLifecycleConfigurationDraft(createProjectDetail());

    current.wait_after_up = true;
    current.wait_timeout_seconds = 180;

    expect(isLifecycleDraftDirty(current, baseline)).toBe(true);
  });
});
