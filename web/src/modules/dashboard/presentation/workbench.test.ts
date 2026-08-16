import { describe, expect, it } from 'vitest';

import {
  DASHBOARD_PREVIEW_PRESENTATION,
  DASHBOARD_PREVIEW_SCENARIO,
  EVIDENCE_STATE,
  PRESENTATION_STATUS,
  type PresentationItem,
  presentationStatusFromAlertLevel,
  presentationStatusFromHealthStatus,
  presentationStatusFromRequestAttentionKind,
  projectWorkbenchScenario,
  sortPresentationItems,
} from './workbench';

function item(id: string, status: PresentationItem['status'], occurredAt?: string): PresentationItem {
  return {
    id,
    region: status === PRESENTATION_STATUS.HEALTHY ? 'health' : 'attention',
    status,
    evidenceState: EVIDENCE_STATE.CONFIRMED,
    titleKey: id,
    descriptionKey: id,
    occurredAt,
  };
}

describe('dashboard workbench presentation model', () => {
  it('sorts by the five-state semantic order and keeps stable id ordering for ties', () => {
    const sorted = sortPresentationItems([
      item('healthy', PRESENTATION_STATUS.HEALTHY),
      item('info', PRESENTATION_STATUS.INFO),
      item('unknown', PRESENTATION_STATUS.UNKNOWN),
      item('warning-b', PRESENTATION_STATUS.WARNING, '2026-08-17T01:00:00Z'),
      item('error', PRESENTATION_STATUS.ERROR),
      item('warning-a', PRESENTATION_STATUS.WARNING, '2026-08-17T01:00:00Z'),
    ]);

    expect(sorted.map(({ id }) => id)).toEqual(['error', 'warning-a', 'warning-b', 'unknown', 'info', 'healthy']);
  });

  it('does not let legacy loader or display metadata override explicit presentation status', () => {
    const accessLog = DASHBOARD_PREVIEW_SCENARIO.items.find(({ id }) => id === 'access-log-source');
    const outbound = DASHBOARD_PREVIEW_SCENARIO.items.find(({ id }) => id === 'outbound-network');

    expect(accessLog).toMatchObject({
      status: PRESENTATION_STATUS.WARNING,
      evidenceState: EVIDENCE_STATE.SOURCE_FAILED,
      actionable: true,
      sourceMetadata: { loadStatus: 'error', displayState: 'critical', priority: 'critical' },
    });
    expect(outbound).toMatchObject({
      status: PRESENTATION_STATUS.UNKNOWN,
      evidenceState: EVIDENCE_STATE.MISSING,
      sourceMetadata: { displayState: 'critical', priority: 'critical' },
    });
  });

  it('keeps evidence state independent from severity and filters attention by explicit semantics', () => {
    const presentation = projectWorkbenchScenario(DASHBOARD_PREVIEW_SCENARIO);

    expect(presentation.attention.map(({ status }) => status)).toEqual([
      PRESENTATION_STATUS.WARNING,
      PRESENTATION_STATUS.UNKNOWN,
    ]);
    expect(presentation.attention.map(({ evidenceState }) => evidenceState)).toEqual([
      EVIDENCE_STATE.SOURCE_FAILED,
      EVIDENCE_STATE.MISSING,
    ]);
    expect(presentation.operational.needsReview).toBe(2);
    expect(presentation.operational.statusCounts.error).toBe(0);
  });

  it('rejects healthy or error claims without confirmed evidence', () => {
    const invalidScenario = {
      ...DASHBOARD_PREVIEW_SCENARIO,
      items: [
        {
          ...DASHBOARD_PREVIEW_SCENARIO.items[2],
          evidenceState: EVIDENCE_STATE.MISSING,
        },
      ],
    };

    expect(() => projectWorkbenchScenario(invalidScenario)).toThrow(/requires confirmed evidence/);
  });

  it('preserves an explicit error even when its loader completed normally', () => {
    const errorItem: PresentationItem = {
      id: 'confirmed-5xx',
      region: 'attention',
      status: PRESENTATION_STATUS.ERROR,
      evidenceState: EVIDENCE_STATE.CONFIRMED,
      titleKey: 'confirmed-5xx',
      descriptionKey: 'confirmed-5xx',
      sourceMetadata: { loadStatus: 'normal', displayState: 'normal', priority: 'normal' },
    };
    const presentation = projectWorkbenchScenario({
      generatedAt: '2026-08-17T03:20:00.000Z',
      operational: { enabledModules: 1, failedTasks: 0, highRiskEvents: 0 },
      items: [errorItem],
      quickActions: [],
    });

    expect(presentation.attention[0]?.status).toBe(PRESENTATION_STATUS.ERROR);
  });

  it('maps authoritative health and alert facts without collapsing distinct states', () => {
    expect(presentationStatusFromHealthStatus('healthy')).toBe(PRESENTATION_STATUS.HEALTHY);
    expect(presentationStatusFromHealthStatus('degraded')).toBe(PRESENTATION_STATUS.WARNING);
    expect(presentationStatusFromHealthStatus('unknown')).toBe(PRESENTATION_STATUS.UNKNOWN);
    expect(presentationStatusFromHealthStatus('disabled')).toBe(PRESENTATION_STATUS.INFO);
    expect(presentationStatusFromAlertLevel('error')).toBe(PRESENTATION_STATUS.ERROR);
    expect(presentationStatusFromAlertLevel('warning')).toBe(PRESENTATION_STATUS.WARNING);
    expect(presentationStatusFromAlertLevel('info')).toBe(PRESENTATION_STATUS.INFO);
    expect(presentationStatusFromRequestAttentionKind('server-error')).toBe(PRESENTATION_STATUS.ERROR);
    expect(presentationStatusFromRequestAttentionKind('client-error')).toBe(PRESENTATION_STATUS.WARNING);
    expect(presentationStatusFromRequestAttentionKind('slow-request')).toBe(PRESENTATION_STATUS.WARNING);
  });

  it('contains no invented error in the fixed design-validation scenario', () => {
    expect(DASHBOARD_PREVIEW_PRESENTATION.operational.statusCounts).toMatchObject({
      error: 0,
      warning: 1,
      unknown: 1,
      info: 3,
      healthy: 2,
    });
  });
});
