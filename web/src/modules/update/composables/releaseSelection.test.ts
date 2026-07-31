import { describe, expect, it } from 'vitest';

import type { UpdateStatus } from '../types/update';
import { getAvailableUpdateRelease, getFixedUpdateReleaseCandidates, hasAvailableUpdate } from './releaseSelection';

const status = (overrides: Partial<UpdateStatus> = {}) =>
  ({
    current_version: '0.11.0-beta.27',
    channel: 'beta',
    image_tag: 'v0.11.0-beta.27',
    deployment_strategy: 'pinned_beta',
    available_releases: [],
    installation_profile: {} as never,
    readiness: {} as never,
    cache_stale: false,
    check_error: '',
    ...overrides,
  }) as UpdateStatus;

const release = (version: string, channel: 'stable' | 'beta') => ({
  version,
  channel,
  notes: '',
  published_at: '2026-07-31T00:00:00Z',
  manifest_url: 'https://example.test/manifest.json',
  server_digest: 'server-digest',
  web_digest: 'web-digest',
  server_image: 'ghcr.io/example/graft-server',
  web_image: 'ghcr.io/example/graft-web',
  server_reference: 'ghcr.io/example/graft-server@sha256:server-digest',
  web_reference: 'ghcr.io/example/graft-web@sha256:web-digest',
  runner_image: 'ghcr.io/example/graft-compose-runner',
  runner_digest: 'runner-digest',
  runner_reference: 'ghcr.io/example/graft-compose-runner@sha256:runner-digest',
});

describe('update release selection', () => {
  it('selects the highest newer verified Beta candidate for a pinned Beta deployment', () => {
    const snapshot = status({
      available_releases: [
        release('0.11.0-beta.28', 'beta'),
        release('0.12.0-beta.1', 'beta'),
        release('0.12.0', 'stable'),
      ],
    });

    expect(getFixedUpdateReleaseCandidates(snapshot).map(({ version }) => version)).toEqual([
      '0.12.0-beta.1',
      '0.11.0-beta.28',
    ]);
    expect(getAvailableUpdateRelease(snapshot)?.version).toBe('0.12.0-beta.1');
    expect(hasAvailableUpdate(snapshot)).toBe(true);
  });

  it('keeps latest as the tracking strategy authority', () => {
    const snapshot = status({
      deployment_strategy: 'beta_tracking',
      latest: release('0.11.0-beta.28', 'beta'),
      available_releases: [release('0.12.0-beta.1', 'beta')],
    });

    expect(getAvailableUpdateRelease(snapshot)?.version).toBe('0.11.0-beta.28');
  });

  it('does not advertise stale or failed snapshots', () => {
    const snapshot = status({
      available_releases: [release('0.11.0-beta.28', 'beta')],
      cache_stale: true,
    });

    expect(getAvailableUpdateRelease(snapshot)?.version).toBe('0.11.0-beta.28');
    expect(hasAvailableUpdate(snapshot)).toBe(false);
  });
});
