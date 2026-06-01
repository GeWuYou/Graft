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
  props: ['modelValue', 'multiple', 'options', 'placeholder'],
  emits: ['update:modelValue'],
  setup(props) {
    return () =>
      h(
        'div',
        JSON.stringify({ modelValue: props.modelValue, multiple: props.multiple, placeholder: props.placeholder }),
      );
  },
});

const tagInputStub = defineComponent({
  name: 'TTagInputStub',
  props: ['modelValue', 'inputProps'],
  emits: ['update:modelValue'],
  setup(props) {
    return () => h('div', JSON.stringify({ modelValue: props.modelValue, inputProps: props.inputProps }));
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
          time: {
            last24h: '最近 24 小时',
            last7d: '最近 7 天',
            last30d: '最近 30 天',
            custom: '自定义时间范围',
          },
          presets: {
            label: '快捷筛选',
            all: '全部',
            failedOperations: '失败操作',
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
            groups: {
              time: '时间范围',
              filters: '筛选条件',
              sort: '排序方式',
            },
            summary: {
              customTime: '{range}',
              result: '结果 {value}',
              sortBy: '按{field}{direction}',
            },
            fields: {
              success: '成功状态',
              action: '操作类型',
              actionPrefixes: '操作分类',
              actionKeywords: '操作关键词',
              result: '结果',
              results: '结果集合',
              riskLevel: '风险等级',
              riskLevels: '风险等级集合',
              source: '事件类型',
              actor: '操作人',
              resourceName: '审计目标',
              resourceType: '目标类型',
              resourceTypes: '目标类型集合',
              requestPathPrefixes: '请求路径前缀',
              requestId: '请求 ID',
              session: 'Session ID',
              resourceId: '资源 ID',
              businessCategory: '业务分类',
            },
          },
          filters: {
            keywordPlaceholder: '搜索操作、用户、目标对象、请求ID...',
            actionPlaceholder: '操作类型',
            actionPrefixesPlaceholder: '选择操作分类',
            actionKeywordsPlaceholder: '输入操作关键词后回车',
            successPlaceholder: '成功状态',
            resultPlaceholder: '结果',
            resultsPlaceholder: '选择结果集合',
            riskPlaceholder: '风险等级',
            riskLevelsPlaceholder: '选择风险等级集合',
            sourcePlaceholder: '事件类型',
            businessCategoryPlaceholder: '业务分类',
            actorPlaceholder: '操作者',
            resourceNamePlaceholder: '审计目标',
            resourceTypePlaceholder: '目标类型',
            resourceTypesPlaceholder: '选择目标类型集合',
            requestIdPlaceholder: '请求 ID',
            sessionPlaceholder: 'Session ID',
            resourceIdPlaceholder: '资源 ID',
            requestPathPrefixesPlaceholder: '输入请求路径前缀后回车',
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
            authPrefix: '认证动作',
            rbacPrefix: '权限配置动作',
            role: '角色',
            rolePrefix: '角色动作',
            permission: '权限',
            permissionPrefix: '权限动作',
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
          success: 'all',
          action: '',
          actionPrefix: '',
          actionPrefixes: ['rbac.', 'role.'],
          actionKeywords: [],
          requestPathPrefixes: [],
          source: '',
          businessCategory: '',
          createdRange: [],
          resourceType: '',
          resourceTypes: [],
          resourceName: '',
          resourceId: '',
          result: 'FAILED',
          results: ['DENIED'],
          riskLevel: 'all',
          riskLevels: ['HIGH', 'CRITICAL'],
          session: '',
          requestId: '',
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
          't-tag-input': tagInputStub,
        },
      },
    });

    expect(wrapper.find('input').attributes('placeholder')).toBe('搜索操作、用户、目标对象、请求ID...');
    expect(wrapper.text()).toContain('排序：创建时间 ↓');
    expect(wrapper.text()).toContain('结果：业务失败');
    expect(wrapper.text()).toContain('筛选条件');
    expect(wrapper.text()).toContain('操作分类：权限配置动作、角色动作');
    expect(wrapper.text()).toContain('结果集合：权限拒绝');
    expect(wrapper.text()).toContain('风险等级集合：高风险、严重');

    const tags = wrapper.findAllComponents(tagStub);
    tags[0]?.vm.$emit('close');
    tags[1]?.vm.$emit('close');
    tags[2]?.vm.$emit('close');
    tags[3]?.vm.$emit('close');

    const updates = (wrapper.emitted('update:modelValue')?.map((entry) => entry[0]) ?? []) as Array<
      Record<string, unknown>
    >;

    expect(updates[0]).toMatchObject({
      sorters: [],
    });
    expect(updates).toContainEqual(
      expect.objectContaining({
        result: 'all',
      }),
    );
    expect(updates).toContainEqual(
      expect.objectContaining({
        actionPrefixes: [],
      }),
    );
    expect(updates).toContainEqual(
      expect.objectContaining({
        results: [],
      }),
    );
    expect(updates).not.toContainEqual(
      expect.objectContaining({
        actor: '',
      }),
    );
    expect(updates.find((item) => item.result === 'all')).toMatchObject({
      result: 'all',
    });
  });
});
