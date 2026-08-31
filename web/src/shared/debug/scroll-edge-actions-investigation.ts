import { createBehaviorInvestigation, type InvestigationPhase } from './behavior-investigation';

// FRONTEND-INVESTIGATION-TEMP:scroll-edge-actions-offset 仅用于确认悬浮轨道状态切换时的尺寸与位置变化。
const investigation = createBehaviorInvestigation({
  investigationId: 'scroll-edge-actions-offset',
  maxEvents: 240,
  allowedSummaryKeys: [
    'action',
    'atBottom',
    'atTop',
    'bottomLeft',
    'bottomRight',
    'bottomWidth',
    'buttonCount',
    'clientHeight',
    'compact',
    'controllerId',
    'isScrollable',
    'maxScrollTop',
    'registration',
    'rootLeft',
    'rootRight',
    'rootWidth',
    'scrollHeight',
    'scrollTop',
    'targetAttached',
    'topLeft',
    'topRight',
    'topWidth',
  ],
});

function formatPayloadSummary(payloadSummary: Record<string, unknown>): string {
  const entries = Object.entries(payloadSummary);
  if (entries.length === 0) {
    return '';
  }

  return entries
    .map(([key, value]) => `${key}=${typeof value === 'string' ? JSON.stringify(value) : String(value)}`)
    .join(' ');
}

export function emitScrollEdgeDebug(
  phase: InvestigationPhase,
  event: string,
  payloadSummary: Record<string, unknown> = {},
) {
  const payloadText = formatPayloadSummary(payloadSummary);

  investigation.emit({
    phase,
    // FRONTEND-INVESTIGATION-TEMP:payload-inline 取证日志传输器限制字段数量，关键布局数据内联到消息中。
    event: payloadText ? `${event} | ${payloadText}` : event,
    source: 'scroll-edge-actions',
    component: 'ScrollEdgeActions',
    payloadSummary,
  });
}
