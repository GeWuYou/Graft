import { describe, expect, it } from 'vitest';

import type { UpdateStatus } from '../types/update';
import { isUpgradeEligible } from './updateEligibility';

const status = (overrides: Partial<UpdateStatus> = {}): UpdateStatus => ({
  current_version: '1.0.0',
  channel: 'stable',
  image_tag: 'latest',
  update_mode: 'stable_tracking',
  available_releases: [],
  latest: {
    version: '1.1.0',
    channel: 'stable',
    notes: '',
    published_at: '2026-07-23T00:00:00Z',
    manifest_url: 'https://example.com/manifest.json',
    server_digest: 'server-digest',
    web_digest: 'web-digest',
    server_image: 'ghcr.io/example/graft-server',
    web_image: 'ghcr.io/example/graft-web',
    server_reference: 'ghcr.io/example/graft-server@sha256:server-digest',
    web_reference: 'ghcr.io/example/graft-web@sha256:web-digest',
    runner_image: 'ghcr.io/example/graft-compose-runner',
    runner_digest: 'runner-digest',
    runner_reference: 'ghcr.io/example/graft-compose-runner@sha256:runner-digest',
  },
  installation_profile: {
    declared_mode: 'compose',
    detected_mode: 'compose',
    capability: 'compose_upgrade_available',
    guidance: '',
    compose_root_source: 'explicit_env',
    compose_candidates: [],
  },
  cache_stale: false,
  check_error: '',
  ...overrides,
});

describe('isUpgradeEligible', () => {
  it('requires a trusted release, executable capability, and manage permission', () => {
    expect(isUpgradeEligible(status(), true)).toBe(true);
    expect(isUpgradeEligible(status(), false)).toBe(false);
    expect(isUpgradeEligible(status({ cache_stale: true }), true)).toBe(false);
    expect(isUpgradeEligible(status({ check_error: 'catalog-unavailable' }), true)).toBe(false);
    expect(
      isUpgradeEligible(
        status({
          installation_profile: {
            declared_mode: 'binary',
            detected_mode: 'binary',
            capability: 'manual_guidance',
            guidance: 'manual guidance',
            compose_root_source: 'unavailable',
            compose_candidates: [],
          },
        }),
        true,
      ),
    ).toBe(false);
    expect(isUpgradeEligible(status({ latest: undefined }), true)).toBe(false);
    expect(isUpgradeEligible(status({ update_mode: 'pinned_stable', latest: undefined }), true)).toBe(false);
  });
});
