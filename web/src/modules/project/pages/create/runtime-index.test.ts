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
  it('makes only Docker Compose actionable', async () => {
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

    expect(wrapper.text()).toContain('project.runtime.items.dockerCompose.title');
    expect(wrapper.text()).toContain('project.runtime.items.dockerSwarm.title');
    expect(wrapper.findAll('.project-runtime-card__icon')).toHaveLength(5);
    expect(wrapper.findAll('[role="button"]')).toHaveLength(1);
    expect(wrapper.findAll('.project-runtime-card--disabled')).toHaveLength(4);
    expect(wrapper.get('[data-testid="project-runtime-docker-swarm"]').attributes('aria-disabled')).toBe('true');
    expect(wrapper.get('[data-testid="project-runtime-docker-swarm"]').attributes('tabindex')).toBe('-1');

    await wrapper.get('[data-testid="project-runtime-docker-compose"]').trigger('click');

    expect(push).toHaveBeenCalledWith({ name: 'ProjectCreateSourceIndex', query: { runtime: 'docker-compose' } });

    await wrapper.get('[data-testid="project-runtime-docker-compose"]').trigger('keydown.space');

    expect(push).toHaveBeenCalledTimes(2);
  });
});
