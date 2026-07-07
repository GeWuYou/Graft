import { describe, expect, it } from 'vitest';

import {
  emptyWorkspaceTooltipRule,
  matchWorkspaceTooltipRule,
  moveWorkspaceTooltipRule,
  parseWorkspaceTooltipRules,
  serializeWorkspaceTooltipRules,
  summarizeWorkspaceTooltipPattern,
} from './workspace-tooltip-rules';

describe('workspace tooltip rules helpers', () => {
  it('parses the JSON string into normalized rules', () => {
    expect(
      parseWorkspaceTooltipRules(
        '[{"pattern":"^docker-compose(?:\\\\.[^.]+)?\\\\.ya?ml$","tooltip":"Compose 配置","enabled":true},{"pattern":"^\\\\.env(?:\\\\..+)?$","tooltip":"环境变量文件"}]',
      ),
    ).toEqual([
      {
        enabled: true,
        pattern: '^docker-compose(?:\\.[^.]+)?\\.ya?ml$',
        tooltip: 'Compose 配置',
      },
      {
        enabled: true,
        pattern: '^\\.env(?:\\..+)?$',
        tooltip: '环境变量文件',
      },
    ]);
  });

  it('serializes rules with trimmed pattern and tooltip fields', () => {
    expect(
      serializeWorkspaceTooltipRules([
        {
          enabled: true,
          pattern: '  ^docker-compose\\.ya?ml$  ',
          tooltip: '  Compose 配置  ',
        },
      ]),
    ).toBe('[{"enabled":true,"pattern":"^docker-compose\\\\.ya?ml$","tooltip":"Compose 配置"}]');
  });

  it('moves a rule to the target index without mutating unrelated items', () => {
    const rules = [
      { enabled: true, pattern: '^a$', tooltip: 'A' },
      { enabled: true, pattern: '^b$', tooltip: 'B' },
      { enabled: false, pattern: '^c$', tooltip: 'C' },
    ];

    expect(moveWorkspaceTooltipRule(rules, 2, 0)).toEqual([
      { enabled: false, pattern: '^c$', tooltip: 'C' },
      { enabled: true, pattern: '^a$', tooltip: 'A' },
      { enabled: true, pattern: '^b$', tooltip: 'B' },
    ]);
  });

  it('evaluates regex matches and reports invalid patterns', () => {
    expect(matchWorkspaceTooltipRule('^docker-compose(?:\\.[^.]+)?\\.ya?ml$', 'docker-compose.dev.yml')).toEqual({
      matched: true,
      valid: true,
    });
    expect(matchWorkspaceTooltipRule('^docker-compose', 'compose.yml')).toEqual({
      matched: false,
      valid: true,
    });
    expect(matchWorkspaceTooltipRule('[a-', 'docker-compose.yml')).toEqual({
      matched: false,
      valid: false,
    });
  });

  it('creates an enabled empty rule and summarizes long patterns', () => {
    expect(emptyWorkspaceTooltipRule()).toEqual({
      enabled: true,
      pattern: '',
      tooltip: '',
    });
    expect(
      summarizeWorkspaceTooltipPattern('^docker-compose(?:\\.[^.]+)?\\.ya?ml$-with-a-very-long-suffix-for-preview', 32),
    ).toBe('^docker-compose(?:\\.[^.]+)?\\.ya…');
  });
});
