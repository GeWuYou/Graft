import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import ApplicationRuntimeTargetIndex from './runtime-target-index.vue';

const mocks = vi.hoisted(() => ({
  getApplicationComposeRuntimeTargets: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
  resolve: vi.fn((target) => target),
}));
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: { deployment: 'compose' } }),
  useRouter: () => ({ push: mocks.push, replace: mocks.replace, resolve: mocks.resolve }),
}));
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key }),
}));
vi.mock('../../api/project', () => ({
  getApplicationComposeRuntimeTargets: mocks.getApplicationComposeRuntimeTargets,
}));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('@/store/modules/tabs-router', () => ({
  useTabsRouterStore: () => ({ appendTabRouterList: vi.fn(), setActiveTabKey: vi.fn() }),
}));
vi.mock('@/utils/route/title', () => ({ localizeRouteTitleKey: (key: string) => key }));
vi.mock('../../shared/navigation', () => ({
  useApplicationCreateRouteNavigation: () => (target: unknown) => mocks.push(target),
}));

describe('ApplicationRuntimeTargetIndex', () => {
  it('renders API targets and preserves the selected Compose deployment in the source route', async () => {
    mocks.getApplicationComposeRuntimeTargets.mockResolvedValue({
      deployment_type: 'compose',
      items: [
        {
          runtime_target_id: 1,
          display_name: 'Local Docker',
          provider: 'docker',
          availability: true,
          readiness: 'ready',
          capabilities: ['compose_execution', 'workspace_access'],
        },
      ],
    });
    const wrapper = mount(ApplicationRuntimeTargetIndex, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot /><slot name="actions" /></header>' },
          't-card': { inheritAttrs: false, template: '<section v-bind="$attrs"><slot /></section>' },
          't-button': { inheritAttrs: false, template: '<button v-bind="$attrs"><slot /></button>' },
          't-tooltip': { template: '<div><slot /></div>' },
          't-alert': true,
        },
      },
    });
    await flushPromises();
    expect(wrapper.text()).toContain('Local Docker');
    await wrapper.get('[data-testid="project-runtime-target-1"]').trigger('click');
    expect(mocks.push).toHaveBeenCalledWith({
      name: 'ApplicationCreateSourceIndex',
      query: { deployment: 'compose', runtime_target_id: '1' },
    });

    await wrapper.get('[data-testid="project-runtime-target-back"]').trigger('click');
    expect(mocks.push).toHaveBeenLastCalledWith({
      name: 'ApplicationCreateMethodIndex',
      query: { deployment: 'compose' },
    });
  });

  it('does not transition when an unavailable target card is activated', async () => {
    mocks.push.mockClear();
    mocks.getApplicationComposeRuntimeTargets.mockResolvedValue({
      deployment_type: 'compose',
      items: [
        {
          runtime_target_id: 2,
          display_name: 'Unavailable Docker',
          provider: 'docker',
          availability: false,
          readiness: 'blocked',
          capabilities: [],
        },
      ],
    });
    const wrapper = mount(ApplicationRuntimeTargetIndex, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot /><slot name="actions" /></header>' },
          't-card': { inheritAttrs: false, template: '<section v-bind="$attrs"><slot /></section>' },
          't-button': { inheritAttrs: false, template: '<button v-bind="$attrs"><slot /></button>' },
          't-tooltip': { template: '<div><slot /></div>' },
          't-alert': true,
        },
      },
    });
    await flushPromises();

    const card = wrapper.get('[data-testid="project-runtime-target-2"]');
    await card.trigger('click');
    await card.trigger('keydown', { key: 'Enter' });

    expect(mocks.push).not.toHaveBeenCalled();
  });

  it('transitions from an actionable target card to source selection', async () => {
    mocks.push.mockClear();
    mocks.getApplicationComposeRuntimeTargets.mockResolvedValue({
      deployment_type: 'compose',
      items: [
        {
          runtime_target_id: 3,
          display_name: 'Ready Docker',
          provider: 'docker',
          availability: true,
          readiness: 'ready',
          capabilities: ['compose_execution'],
        },
      ],
    });
    const wrapper = mount(ApplicationRuntimeTargetIndex, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot /><slot name="actions" /></header>' },
          't-card': { inheritAttrs: false, template: '<section v-bind="$attrs"><slot /></section>' },
          't-button': { inheritAttrs: false, template: '<button v-bind="$attrs"><slot /></button>' },
          't-tooltip': { template: '<div><slot /></div>' },
          't-alert': true,
        },
      },
    });
    await flushPromises();

    await wrapper.get('[data-testid="project-runtime-target-3"]').trigger('keydown', { key: 'Enter' });

    expect(mocks.push).toHaveBeenCalledWith({
      name: 'ApplicationCreateSourceIndex',
      query: { deployment: 'compose', runtime_target_id: '3' },
    });
  });
});
