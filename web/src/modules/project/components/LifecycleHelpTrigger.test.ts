import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { lifecycleHelpDefinitions } from '../shared/lifecycle-help';
import type { ApplicationLifecycleConfigurationDraft } from '../types/project';
import LifecycleHelpTrigger from './LifecycleHelpTrigger.vue';

const messages: Record<string, string> = {
  'project.detail.lifecycle.downBeforeRedeploy': 'Down Before Redeploy',
  'project.detail.lifecycle.help.common.ariaLabel': 'View help for {item}',
  'project.detail.lifecycle.help.common.sections.effect': 'Effect',
  'project.detail.lifecycle.help.common.sections.command': 'Actual Command',
  'project.detail.lifecycle.help.common.sections.scenarios': 'Recommended When',
  'project.detail.lifecycle.help.common.sections.risks': 'Costs and Risks',
  'project.detail.lifecycle.help.common.sections.recommendation': 'Default Guidance',
  'project.detail.lifecycle.help.common.tags.optional': 'Optional',
  'project.detail.lifecycle.help.items.downBeforeRedeploy.tooltip': 'Summary tooltip',
  'project.detail.lifecycle.help.items.downBeforeRedeploy.effect': 'Run down before up.',
  'project.detail.lifecycle.help.items.downBeforeRedeploy.scenarios': 'Use for a clean reset.',
  'project.detail.lifecycle.help.items.downBeforeRedeploy.risks': 'This creates downtime.',
  'project.detail.lifecycle.help.items.downBeforeRedeploy.recommendation': 'Enable only when downtime is acceptable.',
};

const TPopupStub = defineComponent({
  name: 'TPopup',
  props: {
    attach: { type: String, default: undefined },
    visible: { type: Boolean, default: false },
  },
  emits: ['update:visible'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'div',
        {
          'data-attach': props.attach,
          'data-popup-visible': props.visible ? 'true' : 'false',
          'data-stub': 'TPopup',
          onClickCapture: (event: MouseEvent) => {
            if ((event.target as HTMLElement).closest('button')) {
              emit('update:visible', !props.visible);
            }
          },
        },
        [slots.default?.(), props.visible ? h('div', { 'data-popup-content': 'true' }, slots.content?.()) : null],
      );
  },
});

const TTooltipStub = defineComponent({
  name: 'TTooltip',
  props: {
    attach: { type: String, default: undefined },
    content: { type: String, default: '' },
    disabled: { type: Boolean, default: false },
    visible: { type: Boolean, default: false },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-attach': props.attach,
          'data-disabled': props.disabled ? 'true' : 'false',
          'data-tooltip-content': props.content,
          'data-tooltip-visible': props.visible ? 'true' : 'false',
          'data-stub': 'TTooltip',
        },
        slots.default?.(),
      );
  },
});

const TTagStub = defineComponent({
  name: 'TTag',
  setup(_, { slots }) {
    return () => h('span', { 'data-stub': 'TTag' }, slots.default?.());
  },
});

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      const template = messages[key] ?? key;
      return template.replace(/\{(\w+)\}/g, (_, token) => String(params?.[token] ?? `{${token}}`));
    },
  }),
}));

const draft: ApplicationLifecycleConfigurationDraft = {
  additional_args: '',
  managed_service_names: [],
  declared_service_names: [],
  stop_args: '',
  restart_args: '',
  pull_args: '',
  build_before_up: false,
  compose_project_name: 'compose-demo',
  compose_files: ['compose.yaml'],
  down_before_redeploy: true,
  force_recreate: false,
  generated_commands: null,
  profiles: [],
  prune_images_after_redeploy: false,
  pull_before_redeploy: false,
  remove_orphans: true,
  renew_anon_volumes: false,
  review_status: 'confirmed',
  strategy_kind: 'standard',
  wait_after_up: false,
  wait_timeout_seconds: 120,
  workspace_path: '/srv/compose-demo',
};

function mountTrigger() {
  return mount(LifecycleHelpTrigger, {
    props: {
      definition: lifecycleHelpDefinitions[0],
      draft,
    },
    global: {
      stubs: {
        't-popup': TPopupStub,
        't-tag': TTagStub,
        't-tooltip': TTooltipStub,
      },
    },
  });
}

describe('LifecycleHelpTrigger', () => {
  it('shows tooltip on hover and focus, then hides it on leave and blur', async () => {
    const wrapper = mountTrigger();
    const tooltip = wrapper.get('[data-stub="TTooltip"]');
    const button = wrapper.get('button');

    expect(tooltip.attributes('data-attach')).toBe('body');
    expect(tooltip.attributes('data-tooltip-visible')).toBe('false');

    await button.trigger('mouseenter');
    expect(tooltip.attributes('data-tooltip-visible')).toBe('true');

    await button.trigger('mouseleave');
    expect(tooltip.attributes('data-tooltip-visible')).toBe('false');

    await button.trigger('focus');
    expect(tooltip.attributes('data-tooltip-visible')).toBe('true');

    await button.trigger('blur');
    expect(tooltip.attributes('data-tooltip-visible')).toBe('false');
  });

  it('opens popup on click, exposes aria state, and disables tooltip while expanded', async () => {
    const wrapper = mountTrigger();
    const button = wrapper.get('button');

    expect(button.attributes('aria-label')).toBe('View help for Down Before Redeploy');
    expect(button.attributes('aria-expanded')).toBe('false');

    await button.trigger('click');

    const popup = wrapper.get('[data-stub="TPopup"]');
    const tooltip = wrapper.get('[data-stub="TTooltip"]');

    expect(button.attributes('aria-expanded')).toBe('true');
    expect(popup.attributes('data-attach')).toBe('body');
    expect(popup.attributes('data-popup-visible')).toBe('true');
    expect(tooltip.attributes('data-disabled')).toBe('true');
    expect(wrapper.text()).toContain('Actual Command');
    expect(wrapper.text()).toContain('docker compose down');
    expect(wrapper.text()).toContain('This creates downtime.');
  });
});
