import { beforeEach, describe, expect, it, vi } from 'vitest';

const { loggerMocks } = vi.hoisted(() => ({ loggerMocks: { debug: vi.fn() } }));
vi.mock('@/utils/logger', () => ({ createLogger: () => loggerMocks }));
vi.mock('./runtime', () => ({ isDebugFlagEnabled: vi.fn(() => false) }));

import {
  createBehaviorInvestigation,
  FRONTEND_INVESTIGATION_MARKER,
  FRONTEND_INVESTIGATION_SCHEMA_VERSION,
} from './behavior-investigation';

describe('createBehaviorInvestigation', () => {
  beforeEach(() => loggerMocks.debug.mockReset());

  it('exposes the stable event marker and schema version', () => {
    expect(FRONTEND_INVESTIGATION_MARKER).toBe('FRONTEND-INVESTIGATION');
    expect(FRONTEND_INVESTIGATION_SCHEMA_VERSION).toBe(1);
  });

  it('is default-off and does not emit or log', () => {
    const session = createBehaviorInvestigation({ investigationId: 'case-1' });
    expect(session.emit({ phase: 'USER_ACTION', event: 'click', source: 'button' })).toBeNull();
    expect(session.events()).toHaveLength(0);
    expect(loggerMocks.debug).not.toHaveBeenCalled();
  });

  it('emits correlated, sequenced, bounded and sanitized events', () => {
    const session = createBehaviorInvestigation({
      investigationId: 'case-2',
      isEnabled: () => true,
      maxEvents: 2,
    });
    const root = session.emit({ phase: 'USER_ACTION', event: 'edit', source: 'row' });
    const child = session.emit({
      phase: 'API_REQUEST',
      event: 'request:start',
      source: 'queryFn',
      parentEventId: root?.eventId,
      requestSummary: { method: 'GET', token: 'hidden', path: '/registries' },
    });
    session.emit({
      phase: 'ERROR',
      event: 'request:error',
      source: 'axios',
      payloadSummary: { password: 'hidden', unsafeBusinessField: 'omitted' },
    });

    expect(child?.parentEventId).toBe(root?.eventId);
    expect(session.events()).toHaveLength(2);
    expect(session.events()[0].seq).toBe(2);
    expect(session.events()[0].requestSummary).toEqual({ method: 'GET', token: '[REDACTED]', path: '/registries' });
    expect(session.events()[1].payloadSummary).toEqual({ password: '[REDACTED]' });
    expect(loggerMocks.debug).toHaveBeenCalledTimes(3);
  });

  it('supports a case-scoped allowlist extension', () => {
    const session = createBehaviorInvestigation({ isEnabled: () => true, allowedSummaryKeys: ['drawerState'] });
    session.emit({ phase: 'STATE_CHANGE', event: 'drawer', source: 'model', stateSummary: { drawerState: 'closed' } });
    expect(session.events()[0].stateSummary).toEqual({ drawerState: 'closed' });
  });

  it('stops emitting after close', () => {
    const session = createBehaviorInvestigation({ isEnabled: () => true });
    session.close();
    expect(session.isClosed()).toBe(true);
    expect(session.emit({ phase: 'LIFECYCLE', event: 'mounted', source: 'component' })).toBeNull();
  });
});
