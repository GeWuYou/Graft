import { isJsonRecord, type JsonRecord, parseJsonValue } from './json';

export type WorkspaceTooltipRule = {
  enabled: boolean;
  pattern: string;
  tooltip: string;
};

export type WorkspaceTooltipRuleCollectionLabels = {
  advancedTitle: string;
  basicInfoTitle: string;
  collectionEnabledCount: string;
  collectionRuleCount: string;
  collectionTitle: string;
  detailDescription: string;
  detailTitle: string;
  expandPatternEditorAction: string;
  emptyDescription: string;
  emptyTitle: string;
  invalidPatternLabel: string;
  noPatternSummary: string;
  regexEditorLabel: string;
  ruleAddAction: string;
  ruleDisabledState: string;
  ruleDownAction: string;
  ruleDragHint: string;
  ruleEnabledDescription: string;
  ruleEnabledLabel: string;
  ruleEnabledState: string;
  ruleFallbackTitle: string;
  ruleMatchedDescription: string;
  ruleMatchedLabel: string;
  ruleMatchedRuleLabel: string;
  rulePatternDescription: string;
  rulePatternLabel: string;
  rulePatternPlaceholder: string;
  rulePreviewJsonTitle: string;
  ruleRemoveAction: string;
  sectionDangerTitle: string;
  sectionTestTitle: string;
  sectionToggleTitle: string;
  ruleTestEmptyDescription: string;
  ruleTestEmptyLabel: string;
  ruleTestLabel: string;
  ruleTestPlaceholder: string;
  ruleTooltipLabel: string;
  ruleTooltipPlaceholder: string;
  ruleUnmatchedDescription: string;
  ruleUnmatchedLabel: string;
  ruleUpAction: string;
};

export function parseWorkspaceTooltipRules(value: unknown): WorkspaceTooltipRule[] {
  const parsed = typeof value === 'string' ? parseJsonValue(value) : value;
  if (!Array.isArray(parsed)) {
    return [];
  }

  return parsed
    .filter((item): item is JsonRecord => isJsonRecord(item))
    .map((item) => ({
      enabled: item.enabled !== false,
      pattern: typeof item.pattern === 'string' ? item.pattern : '',
      tooltip: typeof item.tooltip === 'string' ? item.tooltip : '',
    }));
}

export function serializeWorkspaceTooltipRules(value: WorkspaceTooltipRule[]) {
  return JSON.stringify(
    value.map((item) => ({
      enabled: item.enabled !== false,
      pattern: item.pattern.trim(),
      tooltip: item.tooltip.trim(),
    })),
  );
}

export function emptyWorkspaceTooltipRule(): WorkspaceTooltipRule {
  return {
    enabled: true,
    pattern: '',
    tooltip: '',
  };
}

export function moveWorkspaceTooltipRule(
  items: WorkspaceTooltipRule[],
  fromIndex: number,
  toIndex: number,
): WorkspaceTooltipRule[] {
  if (fromIndex < 0 || toIndex < 0 || fromIndex >= items.length || toIndex >= items.length || fromIndex === toIndex) {
    return items;
  }

  const nextItems = [...items];
  const [current] = nextItems.splice(fromIndex, 1);
  nextItems.splice(toIndex, 0, current);
  return nextItems;
}

export function summarizeWorkspaceTooltipPattern(pattern: string, maxLength = 56) {
  const normalized = pattern.trim().replace(/\s+/g, ' ');
  if (!normalized) {
    return '';
  }
  if (normalized.length <= maxLength) {
    return normalized;
  }
  return `${normalized.slice(0, maxLength - 1)}…`;
}

export function matchWorkspaceTooltipRule(pattern: string, sample: string) {
  try {
    const matcher = new RegExp(pattern);
    return {
      matched: matcher.test(sample),
      valid: true,
    };
  } catch {
    return {
      matched: false,
      valid: false,
    };
  }
}
