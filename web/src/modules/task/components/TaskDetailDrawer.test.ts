import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import { resolveTaskFailureMessage } from '../shared/failure-copy';
import TaskDetailDrawer from './TaskDetailDrawer.vue';

const ResourceDetailLayoutStub = defineComponent({
  name: 'ResourceDetailLayout',
  props: {
    backLabel: { type: String, required: true },
    presentation: { type: String, required: true },
    size: { type: String, required: true },
    title: { type: String, required: true },
    visible: { type: Boolean, required: true },
  },
  emits: ['update:visible'],
  setup(_props, { slots }) {
    return () => h('section', { 'data-testid': 'task-detail-surface' }, slots.default?.());
  },
});

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: {
    'zh-CN': {
      task: {
        detail: {
          back: '返回任务列表',
          title: '任务详情',
        },
      },
    },
  },
});

describe('TaskDetailDrawer', () => {
  it('maps Runtime Agent failure codes to stable product copy without exposing provider text', () => {
    const translate = (key: string) => key;
    expect(resolveTaskFailureMessage('docker_runtime_unavailable', 'raw sdk error', translate)).toBe(
      'task.failureCodes.runtimeUnavailable',
    );
    expect(resolveTaskFailureMessage('unrecognized_provider_code', 'raw sdk error', translate)).toBe(
      'task.failureCodes.unknown',
    );
  });

  it('uses the shared responsive detail surface and forwards close requests', async () => {
    const wrapper = mount(TaskDetailDrawer, {
      props: { taskId: null, visible: true },
      global: {
        plugins: [i18n],
        stubs: {
          ResourceDetailLayout: ResourceDetailLayoutStub,
          't-loading': { template: '<div><slot /></div>' },
        },
      },
    });

    const surface = wrapper.getComponent(ResourceDetailLayoutStub);
    expect(surface.props()).toMatchObject({
      backLabel: '返回任务列表',
      presentation: 'overlay',
      size: 'large',
      title: '任务详情',
      visible: true,
    });

    surface.vm.$emit('update:visible', false);
    expect(wrapper.emitted('update:visible')).toEqual([[false]]);
    wrapper.unmount();
  });
});
