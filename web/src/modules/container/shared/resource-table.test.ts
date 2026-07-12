import { describe, expect, it } from 'vitest';

import type { ContainerSummaryRecord } from '../types/container';
import { createContainerSourceQuickFilter } from './resource-table';

function buildRow(overrides: Partial<ContainerSummaryRecord> = {}): ContainerSummaryRecord {
  return {
    id: 'abc123',
    short_id: 'abc123',
    name: 'web',
    names: ['web'],
    image: 'nginx:latest',
    status: 'Up',
    state: 'running',
    runtime: 'docker',
    created_at: '2026-06-14T00:00:00Z',
    ports: [],
    deployment: {
      type: 'compose',
      managed: true,
      confidence: 'high',
      project: 'graft',
      service: 'web',
      warnings: [],
      action_level: 'allow',
      batch_action_allowed: true,
    },
    ...overrides,
  };
}

describe('createContainerSourceQuickFilter', () => {
  it('uses Compose deployment metadata for project and service filters', () => {
    const row = buildRow();

    expect(createContainerSourceQuickFilter(row, 'group')).toEqual({
      kind: 'compose_project',
      orchestrator: 'compose',
      value: 'graft',
    });
    expect(createContainerSourceQuickFilter(row, 'member')).toEqual({
      kind: 'compose_service',
      orchestrator: 'compose',
      value: 'web',
    });
  });

  it('does not synthesize quick filters when canonical scope fields are absent', () => {
    const row = buildRow({
      deployment: {
        type: 'compose',
        managed: true,
        confidence: 'high',
        warnings: [],
        action_level: 'allow',
        batch_action_allowed: true,
      },
    });

    expect(createContainerSourceQuickFilter(row, 'group')).toBeNull();
    expect(createContainerSourceQuickFilter(row, 'member')).toBeNull();
  });
});
