import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import ContainerRawJsonPanel from './ContainerRawJsonPanel.vue';

const testMocks = vi.hoisted(() => ({
  copyText: vi.fn(async () => true),
  messageError: vi.fn(),
  messageSuccess: vi.fn(),
}));

vi.mock('@/shared/observability', () => ({
  copyText: testMocks.copyText,
  formatLocaleDateTime: (value?: string | null) => value || '-',
}));

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    success: testMocks.messageSuccess,
    error: testMocks.messageError,
  },
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: 'zh-CN',
  }),
}));

describe('ContainerRawJsonPanel', () => {
  it('renders metadata chips and tree mode by default', () => {
    const wrapper = mountPanel();

    expect(wrapper.text()).toContain('原始 JSON');
    expect(wrapper.text()).toContain('当前策略：敏感值按脱敏脱敏展示，界面仍显示 *****，复制时可获得真实值 JSON。');
    expect(wrapper.text()).toContain('字段数 7');
    expect(wrapper.text()).toContain('已脱敏 1');
    expect(wrapper.text()).toContain('环境变量 2');
    expect(wrapper.text()).toContain('端口映射 1');
    expect(wrapper.text()).toContain('挂载 1');
    expect(wrapper.text()).toContain('网络 1');
    expect(wrapper.text()).toContain('更新时间 2026-06-14T01:08:00Z');
    expect(wrapper.find('[data-testid="container-raw-tree-viewer"]').exists()).toBe(true);
  });

  it('switches to source mode and copies formatted json', async () => {
    const wrapper = mountPanel(createRawValue(), createCopyValue());

    await wrapper.get('select').setValue('source');
    await wrapper.vm.$nextTick();

    expect(wrapper.find('[data-testid="container-raw-source-viewer"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('"environment_policy": "masked"');

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('复制'))
      ?.trigger('click');

    expect(testMocks.copyText).toHaveBeenCalledWith(expect.stringContaining('"copy_value": "real-token-value"'));
    expect(testMocks.copyText).not.toHaveBeenCalledWith(expect.stringContaining('"value": "[MASKED]"'));
    expect(testMocks.messageSuccess).toHaveBeenCalledWith('内容已复制。');
  });

  it('shows no-match feedback when search misses', async () => {
    const wrapper = mountPanel();

    await wrapper.get('input[placeholder="搜索字段或内容"]').setValue('missing-value');
    await wrapper.vm.$nextTick();

    expect(wrapper.text()).toContain('未找到匹配内容');
  });

  it('collapses and expands all tree nodes from the toolbar', async () => {
    const wrapper = mountPanel();

    expect(wrapper.text()).toContain('environment_policy');

    const collapseButton = wrapper.findAll('button').find((button) => button.text().includes('折叠全部'));
    expect(collapseButton).toBeTruthy();
    await collapseButton!.trigger('click');
    await wrapper.vm.$nextTick();

    expect(wrapper.text()).not.toContain('environment_policy');

    const expandButton = wrapper.findAll('button').find((button) => button.text().includes('展开全部'));
    expect(expandButton).toBeTruthy();
    await expandButton!.trigger('click');
    await wrapper.vm.$nextTick();

    expect(wrapper.text()).toContain('environment_policy');
  });

  it('keeps nested expand-all visibility in sync after value refresh', async () => {
    const initialValue = {
      details: {
        status: 'running',
      },
    };
    const nextValue = {
      details: {
        status: 'running',
        meta: {
          region: 'cn',
        },
      },
    };
    const wrapper = mountPanel(initialValue, initialValue);

    await wrapper.setProps({
      value: nextValue,
      copyValue: nextValue,
    });
    await wrapper.vm.$nextTick();

    expect(wrapper.text()).toContain('meta');
    expect(wrapper.text()).toContain('region');
    expect(wrapper.text()).toContain('"cn"');
  });

  it('updates toolbar view labels when props change', async () => {
    const wrapper = mountPanel();

    expect(readOptionLabels(wrapper)).toEqual(['树形视图', '源码视图']);

    await wrapper.setProps({
      treeLabel: 'Tree mode',
      sourceLabel: 'Source mode',
    });
    await wrapper.vm.$nextTick();

    expect(readOptionLabels(wrapper)).toEqual(['Tree mode', 'Source mode']);
  });

  it('applies shared scrollbar styling to tree and source viewports', async () => {
    const wrapper = mountPanel();

    expect(wrapper.find('.json-tree-viewer__viewport.graft-scrollbar').exists()).toBe(true);

    await wrapper.get('select').setValue('source');
    await wrapper.vm.$nextTick();

    expect(wrapper.find('.json-source-viewer__viewport.graft-scrollbar').exists()).toBe(true);
  });

  it('renders empty state for null raw json', () => {
    const wrapper = mountPanel(null);

    expect(wrapper.text()).toContain('暂无原始 JSON 数据');
  });

  it('blocks copy when raw json copy is disabled', async () => {
    const wrapper = mountPanel();
    testMocks.copyText.mockClear();
    testMocks.messageError.mockClear();
    await wrapper.setProps({
      value: createRawValue(),
      title: '原始 JSON',
      description: '容器原始 JSON 调试视图。',
      policyMessage: '当前策略：敏感值按脱敏脱敏展示，当前系统配置禁止复制包含敏感字段的 JSON。',
      rawCopyEnabled: false,
      searchPlaceholder: '搜索字段或内容',
      rootLabel: 'container',
      collapseTreeNodeLabel: '折叠节点',
      expandTreeNodeLabel: '展开节点',
      sourceLabel: '源码视图',
      treeLabel: '树形视图',
      copyLabel: '复制',
      copyMaskedTooltip: '复制包含敏感环境变量真实值的 JSON',
      copyDisabledMessage: '当前系统配置禁止复制包含敏感字段的 JSON',
      copySuccessLabel: '内容已复制。',
      copyErrorLabel: '内容复制失败。',
      expandAllLabel: '展开全部',
      collapseAllLabel: '折叠全部',
      formatLabel: '格式化',
      fieldCountLabel: '字段数',
      sensitiveFieldLabel: '敏感字段',
      maskedCountLabel: '已脱敏',
      environmentLabel: '环境变量',
      portLabel: '端口映射',
      mountedLabel: '挂载',
      networkLabel: '网络',
      updatedAtLabel: '更新时间',
      searchEmptyLabel: '未找到匹配内容',
      sensitiveLabel: '敏感',
      emptyLabel: '暂无原始 JSON 数据',
      errorLabel: '原始 JSON 无法格式化。',
    });

    const copyButton = wrapper.findAll('button').find((button) => button.text().includes('复制'));
    expect(copyButton).toBeTruthy();
    expect((copyButton!.element as HTMLButtonElement).disabled).toBe(true);

    expect(testMocks.copyText).not.toHaveBeenCalled();
    expect(testMocks.messageError).not.toHaveBeenCalled();
  });
});

function mountPanel(value: unknown = createRawValue(), copyValue: unknown = value) {
  return mount(ContainerRawJsonPanel, {
    props: {
      copyValue,
      value,
      title: '原始 JSON',
      description: '容器原始 JSON 调试视图。',
      policyMessage: '当前策略：敏感值按脱敏脱敏展示，界面仍显示 *****，复制时可获得真实值 JSON。',
      rawCopyEnabled: true,
      searchPlaceholder: '搜索字段或内容',
      rootLabel: 'container',
      collapseTreeNodeLabel: '折叠节点',
      expandTreeNodeLabel: '展开节点',
      sourceLabel: '源码视图',
      treeLabel: '树形视图',
      copyLabel: '复制',
      copyMaskedTooltip: '复制包含敏感环境变量真实值的 JSON',
      copyDisabledMessage: '当前系统配置禁止复制包含敏感字段的 JSON',
      copySuccessLabel: '内容已复制。',
      copyErrorLabel: '内容复制失败。',
      expandAllLabel: '展开全部',
      collapseAllLabel: '折叠全部',
      formatLabel: '格式化',
      fieldCountLabel: '字段数',
      sensitiveFieldLabel: '敏感字段',
      maskedCountLabel: '已脱敏',
      environmentLabel: '环境变量',
      portLabel: '端口映射',
      mountedLabel: '挂载',
      networkLabel: '网络',
      updatedAtLabel: '更新时间',
      searchEmptyLabel: '未找到匹配内容',
      sensitiveLabel: '敏感',
      emptyLabel: '暂无原始 JSON 数据',
      errorLabel: '原始 JSON 无法格式化。',
    },
    global: {
      stubs: {
        't-alert': defineComponent({
          props: ['message', 'title'],
          setup: (props) => () => h('div', String(props.message ?? props.title ?? '')),
        }),
        't-button': defineComponent({
          props: ['disabled'],
          emits: ['click'],
          setup:
            (props, { attrs, emit, slots }) =>
            () =>
              h(
                'button',
                {
                  ...attrs,
                  disabled: Boolean(props.disabled),
                  onClick: () => {
                    if (!props.disabled) {
                      emit('click');
                    }
                  },
                },
                slots.default?.(),
              ),
        }),
        't-empty': defineComponent({
          props: ['description'],
          setup: (props) => () => h('div', String(props.description ?? '')),
        }),
        't-input': defineComponent({
          props: ['modelValue', 'placeholder'],
          emits: ['update:modelValue'],
          setup:
            (props, { emit }) =>
            () =>
              h('input', {
                placeholder: props.placeholder,
                value: String(props.modelValue ?? ''),
                onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
              }),
        }),
        't-radio-group': defineComponent({
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          setup:
            (props, { emit }) =>
            () =>
              h(
                'select',
                {
                  value: String(props.modelValue ?? ''),
                  onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
                },
                (props.options as Array<{ label: string; value: string }>).map((option) =>
                  h('option', { value: option.value }, option.label),
                ),
              ),
        }),
        't-tag': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('span', slots.default?.()),
        }),
      },
    },
  });
}

function readOptionLabels(wrapper: ReturnType<typeof mountPanel>) {
  return wrapper.findAll('option').map((option) => option.text());
}

function createCopyValue() {
  return {
    ...createRawValue(),
    environment: [
      {
        key: 'APP_MODE',
        value: 'production',
      },
      {
        key: 'API_TOKEN',
        copy_value: 'real-token-value',
        display_value: '[MASKED]',
        value: 'real-token-value',
        value_masked: true,
        sensitive: true,
        masked: true,
      },
    ],
  };
}

function createRawValue() {
  return {
    id: 'container-1',
    inspect_updated_at: '2026-06-14T01:08:00Z',
    environment_policy: 'masked',
    environment: [
      { key: 'APP_MODE', value: 'production', masked: false, sensitive: false },
      { key: 'API_TOKEN', masked: true, sensitive: true },
    ],
    ports: [{ public_port: 8080 }],
    mounts: [{ destination: '/app' }],
    networks: [{ name: 'bridge' }],
  };
}
