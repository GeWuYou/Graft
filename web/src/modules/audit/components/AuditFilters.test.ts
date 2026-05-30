import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import AuditFilters from './AuditFilters.vue';

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_, { attrs, emit, slots }) {
    return () => h('button', { ...attrs, onClick: () => emit('click') }, slots.default?.());
  },
});

const tagStub = defineComponent({
  name: 'TTagStub',
  emits: ['close'],
  setup(_, { slots }) {
    return () => h('div', [h('span', slots.default?.())]);
  },
});

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  setup(_, { slots }) {
    return () => h('div', [slots.default?.(), slots.content?.()]);
  },
});

const inputStub = defineComponent({
  name: 'TInputStub',
  props: ['modelValue', 'placeholder'],
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        value: props.modelValue,
        placeholder: props.placeholder,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
      });
  },
});

const dateRangeStub = defineComponent({
  name: 'TDateRangePickerStub',
  props: ['modelValue'],
  emits: ['update:modelValue'],
  setup() {
    return () => h('div', 'date-range');
  },
});

const selectStub = defineComponent({
  name: 'TSelectStub',
  props: ['modelValue'],
  emits: ['update:modelValue'],
  setup() {
    return () => h('div', 'select');
  },
});

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: {
    'zh-CN': {
      audit: {
        common: {
          source: {
            REQUEST: '审计事件',
            SECURITY_EVENT: '安全事件',
            DOMAIN_EVENT: '领域审计',
          },
        },
        logList: {
          presets: {
            label: '快捷筛选',
            all: '全部',
            highRisk: '高风险',
          },
          actions: {
            search: '查询',
            reset: '重置',
            addFilter: '添加筛选条件',
          },
          builder: {
            title: '筛选字段',
            hint: 'hint',
            fields: {
              action: '操作类型',
              result: '结果',
              riskLevel: '风险等级',
              source: '事件类型',
              actor: '操作人',
              resourceName: '审计目标',
              resourceType: '目标类型',
              requestId: 'Request ID',
              traceId: 'Trace ID',
              session: 'Session ID',
              resourceId: '资源 ID',
            },
          },
          filters: {
            keywordPlaceholder: '搜索操作、用户、目标对象、请求ID...',
            actionPlaceholder: '操作类型',
            resultPlaceholder: '结果',
            riskPlaceholder: '风险等级',
            sourcePlaceholder: '事件类型',
            actorPlaceholder: '操作者',
            resourceNamePlaceholder: '审计目标',
            resourceTypePlaceholder: '目标类型',
            requestIdPlaceholder: 'Request ID',
            traceIdPlaceholder: 'Trace ID',
            sessionPlaceholder: 'Session ID',
            resourceIdPlaceholder: '资源 ID',
            datePlaceholder: '时间范围',
          },
          sort: {
            title: '排序',
            tagPrefix: '排序',
            fieldPlaceholder: '排序字段',
            directionPlaceholder: '排序方向',
            createdAt: '创建时间',
            asc: '升序',
            desc: '降序',
          },
          filterOptions: {
            auth: '认证',
            role: '角色',
            permission: '权限',
            session: '会话',
            userResource: '用户',
            roleResource: '角色',
            permissionResource: '权限',
            authResource: '认证',
            SUCCESS: '成功',
            FAILED: '业务失败',
            DENIED: '权限拒绝',
            ERROR: '系统异常',
            LOW: '低风险',
            MEDIUM: '中风险',
            HIGH: '高风险',
            CRITICAL: '严重',
          },
        },
      },
    },
  },
});

describe('AuditFilters', () => {
  it('renders keyword search placeholder and active filter tags, and clears a tag through the same state model', async () => {
    const wrapper = mount(AuditFilters, {
      props: {
        activePreset: 'all',
        modelValue: {
          keyword: '',
          actor: 'admin',
          actorUserId: '',
          action: '',
          actionPrefix: '',
          source: '',
          createdRange: [],
          resourceType: '',
          resourceName: '',
          resourceId: '',
          result: 'FAILED',
          riskLevel: 'all',
          session: '',
          requestId: '',
          traceId: '',
          sorters: [{ field: 'created_at', direction: 'desc' }],
        },
        presets: [
          { key: 'all', title: '全部' },
          { key: 'high-risk', title: '高风险' },
        ],
      },
      global: {
        plugins: [i18n],
        stubs: {
          'management-toolbar': passthroughStub,
          't-button': buttonStub,
          't-tag': tagStub,
          't-input': inputStub,
          't-popup': passthroughStub,
          't-date-range-picker': dateRangeStub,
          't-select': selectStub,
        },
      },
    });

    expect(wrapper.find('input').attributes('placeholder')).toBe('搜索操作、用户、目标对象、请求ID...');
    expect(wrapper.text()).toContain('排序：创建时间 ↓');
    expect(wrapper.text()).toContain('操作人：admin');
    expect(wrapper.text()).toContain('结果：业务失败');

    const tags = wrapper.findAllComponents(tagStub);
    tags[0]?.vm.$emit('close');
    tags[1]?.vm.$emit('close');
    tags[2]?.vm.$emit('close');

    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toMatchObject({
      sorters: [],
    });
    expect(wrapper.emitted('update:modelValue')?.[1]?.[0]).toMatchObject({
      actor: 'admin',
      result: 'all',
    });
    expect(wrapper.emitted('update:modelValue')?.[2]?.[0]).toMatchObject({
      actor: '',
      result: 'FAILED',
    });
  });
});
