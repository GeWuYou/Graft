import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import ProjectRuntimeTargetIndex from './runtime-target-index.vue';

const mocks = vi.hoisted(() => ({
  getProjectComposeRuntimeTargets: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
  resolve: vi.fn((target) => target),
}));
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: { deployment: 'compose' } }),
  useRouter: () => ({ push: mocks.push, replace: mocks.replace, resolve: mocks.resolve }),
}));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('../../api/project', () => ({ getProjectComposeRuntimeTargets: mocks.getProjectComposeRuntimeTargets }));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('@/store/modules/tabs-router', () => ({
  useTabsRouterStore: () => ({ appendTabRouterList: vi.fn(), setActiveTabKey: vi.fn() }),
}));
vi.mock('@/utils/route/title', () => ({ localizeRouteTitleKey: (key: string) => key }));
vi.mock('../../shared/navigation', () => ({
  useProjectCreateRouteNavigation: () => (target: unknown) => mocks.push(target),
}));

describe('ProjectRuntimeTargetIndex', () => {
  it('renders API targets and preserves the selected Compose deployment in the source route', async () => {
    mocks.getProjectComposeRuntimeTargets.mockResolvedValue({
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
    const wrapper = mount(ProjectRuntimeTargetIndex, {
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
      name: 'ProjectCreateSourceIndex',
      query: { deployment: 'compose', runtime_target_id: '1' },
    });

    await wrapper.get('[data-testid="project-runtime-target-back"]').trigger('click');
    expect(mocks.push).toHaveBeenLastCalledWith({ name: 'ProjectCreateMethodIndex' });
  });
});
