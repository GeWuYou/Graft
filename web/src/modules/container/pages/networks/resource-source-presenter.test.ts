import { describe, expect, it } from 'vitest';

import { presentNetworkResourceSource } from './resource-source-presenter';

describe('presentNetworkResourceSource', () => {
  it('presents Compose identity and sorted metadata without parsing raw labels', () => {
    const presentation = presentNetworkResourceSource({
      source: {
        kind: 'compose',
        compose_project: 'arcane',
        compose_network: 'default',
        compose_version: '5.1.0',
      },
      label_groups: {
        system: { 'com.docker.compose.version': '5.1.0', 'com.docker.compose.project': 'arcane' },
        user: { team: 'platform', env: 'production' },
      },
    });

    expect(presentation.identity).toBe('arcane');
    expect(presentation.sourceFields).toEqual([
      { key: 'composeProject', value: 'arcane' },
      { key: 'composeNetwork', value: 'default' },
      { key: 'composeVersion', value: '5.1.0' },
    ]);
    expect(presentation.userLabels).toEqual([
      ['env', 'production'],
      ['team', 'platform'],
    ]);
    expect(presentation.userLabel).toBe('env=production');
    expect(presentation.remainingUserLabelCount).toBe(1);
  });

  it.each([
    ['swarm', { kind: 'swarm' as const, swarm_stack: 'edge' }, 'edge'],
    ['docker', { kind: 'docker' as const }, undefined],
    ['custom', { kind: 'custom' as const }, undefined],
    ['unknown', { kind: 'unknown' as const }, undefined],
  ])('presents %s only from the canonical source field', (_, source, identity) => {
    expect(presentNetworkResourceSource({ source }).identity).toBe(identity);
  });

  it('keeps the second list line empty of overflow when no user labels exist', () => {
    const presentation = presentNetworkResourceSource({
      source: { kind: 'custom' },
      label_groups: { system: { 'io.docker.network': 'managed' }, user: {} },
    });

    expect(presentation.userLabel).toBeUndefined();
    expect(presentation.remainingUserLabelCount).toBe(0);
    expect(presentation.systemLabels).toEqual([['io.docker.network', 'managed']]);
  });

  it('counts every user label after the first sorted summary label', () => {
    const presentation = presentNetworkResourceSource({
      source: { kind: 'custom' },
      label_groups: { user: { z: '3', a: '1', m: '2' } },
    });

    expect(presentation.userLabel).toBe('a=1');
    expect(presentation.remainingUserLabelCount).toBe(2);
  });
});
