import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';

import ProjectRuntimeIndex from './runtime-index.vue';

const push = vi.fn();

vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

describe('ProjectRuntimeIndex', () => {
  it('makes only Compose actionable and routes to runtime target selection', async () => {
    const wrapper = mount(ProjectRuntimeIndex, {
      global: {
        stubs: {
          'management-page-content': { template: '<div><slot /></div>' },
          'management-page-header': { template: '<header><slot /></header>' },
          't-card': { inheritAttrs: false, template: '<section v-bind="$attrs"><slot /></section>' },
          't-tooltip': { template: '<div><slot /></div>' },
        },
      },
    });

    expect(wrapper.text()).toContain('project.deployment.items.compose.title');
    expect(wrapper.text()).toContain('project.deployment.items.swarm.title');
    expect(wrapper.findAll('[role="button"]')).toHaveLength(1);
    expect(wrapper.findAll('.project-deployment-card--disabled')).toHaveLength(3);
    expect(wrapper.get('[data-testid="project-deployment-swarm"]').attributes('aria-disabled')).toBe('true');

    await wrapper.get('[data-testid="project-deployment-compose"]').trigger('click');

    expect(push).toHaveBeenCalledWith({ name: 'ProjectCreateRuntimeTargetIndex', query: { deployment: 'compose' } });

    await wrapper.get('[data-testid="project-deployment-compose"]').trigger('keydown.space');

    expect(push).toHaveBeenCalledTimes(2);
  });
});
