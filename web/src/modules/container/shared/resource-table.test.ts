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
    orchestrator: {
      type: 'compose',
      managed: true,
      confidence: 'high',
      group_scope_kind: 'compose_project',
      group_value: 'graft',
      group_display_name: 'graft',
      member_scope_kind: 'compose_service',
      member_value: 'web',
      member_display_name: 'web',
      warnings: [],
      action_level: 'allow',
      batch_action_allowed: true,
    },
    ...overrides,
  };
}

describe('createContainerSourceQuickFilter', () => {
  it('uses canonical orchestrator scope fields for group and member filters', () => {
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
      orchestrator: {
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
