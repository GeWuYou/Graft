import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import ConfigEditorRenderer from './ConfigEditorRenderer.vue';

describe('ConfigEditorRenderer', () => {
  it('passes the dedicated workspace tooltip labels through without remapping them to other copy keys', () => {
    const wrapper = mount(ConfigEditorRenderer, {
      props: {
        modelValue: {
          rules: [],
        },
        rootSchema: {
          properties: {
            rules: {
              title: 'Rules',
              type: 'array',
              xGraft: {
                editor: 'workspace-tooltip-rule-list',
              },
            },
          },
          type: 'object',
        },
        labels: {
          advancedTitle: 'Advanced',
          basicInfoTitle: 'Basics',
          collectionEnabledCount: 'enabled',
          collectionRuleCount: 'rules',
          collectionTitle: 'Collection',
          detailDescription: 'Detail description',
          detailTitle: 'Detail',
          emptyTitle: 'Empty',
          emptyDescription: 'Empty description',
          expandPatternEditorAction: 'Expand pattern editor',
          invalidJson: 'Invalid JSON',
          invalidPatternLabel: 'Invalid pattern',
          jsonPlaceholder: 'JSON placeholder',
          noPatternSummary: 'No pattern',
          numberPlaceholder: 'Number placeholder',
          ruleAddAction: 'Add rule',
          ruleDisabledState: 'Disabled',
          ruleDownAction: 'Expand',
          ruleDragHint: 'Drag hint',
          ruleEnabledDescription: 'Enabled description',
          ruleEnabledLabel: 'Enabled',
          ruleEnabledState: 'Enabled state',
          ruleFallbackTitle: 'Fallback',
          ruleMatchedDescription: 'Matched description',
          ruleMatchedLabel: 'Matched',
          ruleMatchedRuleLabel: 'Matched rule',
          rulePatternDescription: 'Pattern description',
          rulePatternLabel: 'Pattern',
          rulePatternPlaceholder: 'Pattern placeholder',
          regexEditorLabel: 'Regex editor',
          rulePreviewJsonTitle: 'Preview',
          ruleRemoveAction: 'Remove',
          ruleTestEmptyDescription: 'Test empty description',
          ruleTestEmptyLabel: 'Test empty label',
          ruleTestLabel: 'Test label',
          ruleTestPlaceholder: 'Test placeholder',
          ruleTooltipLabel: 'Tooltip',
          ruleTooltipPlaceholder: 'Tooltip placeholder',
          ruleUnmatchedDescription: 'Unmatched description',
          ruleUnmatchedLabel: 'Unmatched',
          ruleUpAction: 'Move up',
          sectionDangerTitle: 'Danger zone',
          sectionTestTitle: 'Test section',
          sectionToggleTitle: 'Toggle section',
          selectPlaceholder: 'Select placeholder',
          stringPlaceholder: 'String placeholder',
          value: 'Value',
        },
      },
      global: {
        stubs: {
          WorkspaceTooltipRuleCollection: defineComponent({
            name: 'WorkspaceTooltipRuleCollectionStub',
            props: {
              labels: {
                type: Object,
                required: true,
              },
            },
            setup(props) {
              return () =>
                h('div', {
                  'data-basic-info-title': (props.labels as Record<string, string>).basicInfoTitle,
                  'data-empty-title': (props.labels as Record<string, string>).emptyTitle,
                  'data-expand-pattern-editor-action': (props.labels as Record<string, string>)
                    .expandPatternEditorAction,
                  'data-matched-rule-label': (props.labels as Record<string, string>).ruleMatchedRuleLabel,
                  'data-no-pattern-summary': (props.labels as Record<string, string>).noPatternSummary,
                  'data-regex-editor-label': (props.labels as Record<string, string>).regexEditorLabel,
                  'data-section-danger-title': (props.labels as Record<string, string>).sectionDangerTitle,
                  'data-section-test-title': (props.labels as Record<string, string>).sectionTestTitle,
                  'data-section-toggle-title': (props.labels as Record<string, string>).sectionToggleTitle,
                });
            },
          }),
          't-form-item': defineComponent({
            setup(_, { slots }) {
              return () => h('div', slots.default?.());
            },
          }),
        },
      },
    });

    const collection = wrapper.get('[data-basic-info-title]');
    expect(collection.attributes('data-basic-info-title')).toBe('Basics');
    expect(collection.attributes('data-empty-title')).toBe('Empty');
    expect(collection.attributes('data-expand-pattern-editor-action')).toBe('Expand pattern editor');
    expect(collection.attributes('data-matched-rule-label')).toBe('Matched rule');
    expect(collection.attributes('data-no-pattern-summary')).toBe('No pattern');
    expect(collection.attributes('data-regex-editor-label')).toBe('Regex editor');
    expect(collection.attributes('data-section-danger-title')).toBe('Danger zone');
    expect(collection.attributes('data-section-test-title')).toBe('Test section');
    expect(collection.attributes('data-section-toggle-title')).toBe('Toggle section');
  });
});
