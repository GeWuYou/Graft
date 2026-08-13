import { describe, expect, it } from 'vitest';

import { REGISTRY_DETAIL_MODE, registryDetailPath } from './paths';

describe('registryDetailPath', () => {
  it('creates the canonical edit URL and encodes the connection reference', () => {
    expect(registryDetailPath('team/registry', { mode: REGISTRY_DETAIL_MODE.EDIT })).toBe(
      '/infrastructure/registries/team%2Fregistry?mode=edit',
    );
  });

  it('creates the ordinary detail URL when no mode is requested', () => {
    expect(registryDetailPath('registry-a')).toBe('/infrastructure/registries/registry-a');
  });
});
