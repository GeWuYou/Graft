import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import ApplicationRuntimeIndex from './runtime-index.vue';

const push = vi.fn();
const resolve = vi.fn((target) => target);

vi.mock('vue-router', () => ({
  useRouter: () => ({ push, resolve }),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));
vi.mock('@/store/modules/tabs-router', () => ({
  useTabsRouterStore: () => ({ appendTabRouterList: vi.fn(), setActiveTabKey: vi.fn() }),
}));
vi.mock('@/utils/route/title', () => ({ localizeRouteTitleKey: (key: string) => key }));
vi.mock('../../shared/navigation', () => ({
  useApplicationCreateRouteNavigation: () => (target: unknown) => push(target),
}));

describe('ApplicationRuntimeIndex', () => {
  it('renders deployment adapters and keeps unsupported adapters non-focusable', () => {
    const wrapper = mount(ApplicationRuntimeIndex, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot /><slot name="actions" /></header>' },
          't-card': { inheritAttrs: false, template: '<section v-bind="$attrs"><slot /></section>' },
          't-button': { inheritAttrs: false, template: '<button v-bind="$attrs"><slot /></button>' },
          't-tag': { template: '<span><slot /></span>' },
        },
      },
    });

    expect(wrapper.text()).toContain('project.deployment.items.compose.title');
    expect(wrapper.text()).toContain('project.deployment.items.helm.title');
    expect(wrapper.text()).toContain('project.deployment.items.kustomize.title');
    expect(wrapper.text()).toContain('project.deployment.items.nomadJob.title');
    expect(wrapper.text()).toContain('project.deployment.recommended');
    expect(wrapper.text()).toContain('project.deployment.comingSoon');
    expect(wrapper.text()).toContain('project.deployment.comingSoonHelper');
    expect(wrapper.text()).toContain('project.deployment.items.compose.capabilities.0');
    expect(wrapper.text()).toContain('project.deployment.items.helm.capabilities.2');
    expect(wrapper.text()).toContain('project.deployment.items.kustomize.capabilities.1');
    expect(wrapper.text()).toContain('project.deployment.items.nomadJob.capabilities.1');

    expect(wrapper.findAll('.project-deployment-card--disabled')).toHaveLength(3);
    expect(wrapper.get('[data-testid="project-deployment-compose"]').attributes('aria-disabled')).toBeUndefined();
    expect(wrapper.find('[data-testid="project-deployment-swarm"]').exists()).toBe(false);
    for (const deployment of ['helm', 'kustomize', 'nomad-job']) {
      const card = wrapper.get(`[data-testid="project-deployment-${deployment}"]`);
      expect(card.attributes('aria-disabled')).toBe('true');
      expect(card.attributes('role')).toBeUndefined();
      expect(card.attributes('tabindex')).toBeUndefined();
    }
  });

  it('returns to application management', async () => {
    const wrapper = mount(ApplicationRuntimeIndex, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot /><slot name="actions" /></header>' },
          't-card': { inheritAttrs: false, template: '<section v-bind="$attrs"><slot /></section>' },
          't-button': { inheritAttrs: false, template: '<button v-bind="$attrs"><slot /></button>' },
          't-tag': { template: '<span><slot /></span>' },
        },
      },
    });

    await wrapper.get('[data-testid="project-deployment-back"]').trigger('click');

    expect(push).toHaveBeenCalledWith({ name: 'ApplicationList' });
  });

  it('makes only Compose actionable and routes to runtime target selection', async () => {
    const wrapper = mount(ApplicationRuntimeIndex, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot /><slot name="actions" /></header>' },
          't-card': { inheritAttrs: false, template: '<section v-bind="$attrs"><slot /></section>' },
          't-button': { inheritAttrs: false, template: '<button v-bind="$attrs"><slot /></button>' },
          't-tag': { template: '<span><slot /></span>' },
        },
      },
    });

    await wrapper.get('[data-testid="project-deployment-compose-select"]').trigger('click');

    expect(push).toHaveBeenCalledWith({
      name: 'ApplicationCreateRuntimeTargetIndex',
      query: { deployment: 'compose' },
    });
  });
});
