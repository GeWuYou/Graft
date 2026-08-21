import { describe, expect, it } from 'vitest';

import {
  buildBlankLifecycleConfigurationDraft,
  buildLifecycleConfigurationRequest,
  isLifecycleDraftDirty,
  updateLifecycleDraftProfiles,
} from './lifecycle';

const defaults = {
  lifecycle_configuration: {
    strategy_kind: 'standard' as const,
    profiles: ['metrics'],
    down_before_redeploy: true,
    pull_before_redeploy: false,
    build_before_up: false,
    force_recreate: false,
    remove_orphans: true,
    wait_after_up: true,
    wait_timeout_seconds: 90,
    renew_anon_volumes: false,
    prune_images_after_redeploy: false,
    managed_service_names: ['api'],
  },
};

describe('application lifecycle policy draft', () => {
  it('builds a provider-neutral request from typed policy fields', () => {
    const draft = buildBlankLifecycleConfigurationDraft(defaults, {
      composeFilePath: 'compose.yaml',
      composeProjectName: 'demo',
    });

    expect(buildLifecycleConfigurationRequest(draft)).toEqual({
      strategy_kind: 'standard',
      profiles: ['metrics'],
      down_before_redeploy: true,
      pull_before_redeploy: false,
      build_before_up: false,
      force_recreate: false,
      remove_orphans: true,
      wait_after_up: true,
      wait_timeout_seconds: 90,
      renew_anon_volumes: false,
      prune_images_after_redeploy: false,
      managed_service_names: ['api'],
    });
  });

  it('tracks policy changes without execution material', () => {
    const baseline = buildBlankLifecycleConfigurationDraft(defaults, {
      composeFilePath: 'compose.yaml',
      composeProjectName: 'demo',
    });
    const current = { ...baseline, profiles: [...baseline.profiles] };

    expect(isLifecycleDraftDirty(current, baseline)).toBe(false);
    updateLifecycleDraftProfiles(current, 'metrics, debug');
    expect(current.profiles).toEqual(['metrics', 'debug']);
    expect(isLifecycleDraftDirty(current, baseline)).toBe(true);
  });
});
