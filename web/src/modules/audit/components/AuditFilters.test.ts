import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import AuditFilters from './AuditFilters.vue';

const logFilterBuilderStub = defineComponent({
  name: 'LogFilterBuilderStub',
  props: [
    'keywordPlaceholder',
    'tags',
    'fields',
    'sorters',
    'sortAddDisabled',
    'sortFieldOptionsByIndex',
    'sortMoveUpDisabled',
    'sortMoveDownDisabled',
  ],
  emits: ['close-tag'],
  setup(props, { emit }) {
    return () =>
      h('div', [
        h('span', { 'data-testid': 'keyword-placeholder' }, props.keywordPlaceholder),
        h('span', { 'data-testid': 'tags' }, JSON.stringify(props.tags)),
        h('span', { 'data-testid': 'fields' }, JSON.stringify(props.fields)),
        h('span', { 'data-testid': 'sorters' }, JSON.stringify(props.sorters)),
        h('span', { 'data-testid': 'sort-add-disabled' }, String(props.sortAddDisabled)),
        h('span', { 'data-testid': 'sort-field-options' }, JSON.stringify(props.sortFieldOptionsByIndex)),
        h('span', { 'data-testid': 'sort-move-up-disabled' }, JSON.stringify(props.sortMoveUpDisabled)),
        h('span', { 'data-testid': 'sort-move-down-disabled' }, JSON.stringify(props.sortMoveDownDisabled)),
        h('button', { 'data-testid': 'close-sorter', onClick: () => emit('close-tag', 'sorter:0') }),
      ]);
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
          presets: { label: '快捷筛选' },
          actions: {
            search: '查询',
            reset: '重置',
            addFilter: '添加筛选条件',
            addSorter: '添加排序项',
            removeSorter: '移除排序项',
            moveSorterUp: '上移',
            moveSorterDown: '下移',
          },
          builder: {
            title: '筛选字段',
            hint: 'hint',
            groups: {
              filters: '筛选条件',
            },
            fields: {
              timeRange: '时间范围',
              sorterBuilder: '排序方式',
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
          },
          sort: {
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
          businessCategory: {
            failedOperations: '失败操作',
            highRiskOperations: '高风险操作',
            sensitiveOperations: '敏感操作',
            authFailures: '认证失败',
            permissionDenials: '权限拒绝',
            rbacChanges: '权限配置变更',
            criticalSecurity: '关键安全事件',
          },
        },
      },
    },
  },
});

describe('AuditFilters', () => {
  it('passes keyword placeholder and active tags to the shared builder', () => {
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
        presets: [{ key: 'all', title: '全部' }],
      },
      global: {
        plugins: [i18n],
        stubs: {
          LogFilterBuilder: logFilterBuilderStub,
        },
      },
    });

    expect(wrapper.get('[data-testid="keyword-placeholder"]').text()).toBe('搜索操作、用户、目标对象、请求ID...');

    const tags = JSON.parse(wrapper.get('[data-testid="tags"]').text());
    expect(tags.map((tag: { label: string }) => tag.label)).toContain('排序 1: 创建时间 ↓');
    expect(tags.map((tag: { label: string }) => tag.label)).toContain('结果：业务失败');
    expect(tags.map((tag: { label: string }) => tag.label)).toContain('操作分类：权限配置动作、角色动作');
    expect(tags.map((tag: { label: string }) => tag.label)).toContain('结果集合：权限拒绝');
    expect(tags.map((tag: { label: string }) => tag.label)).toContain('风险等级集合：高风险、严重');
  });

  it('removes sorter through shared builder tag close event', async () => {
    const wrapper = mount(AuditFilters, {
      props: {
        activePreset: 'all',
        modelValue: {
          keyword: '',
          actor: '',
          success: 'all',
          action: '',
          actionPrefix: '',
          actionPrefixes: [],
          actionKeywords: [],
          requestPathPrefixes: [],
          source: '',
          businessCategory: '',
          createdRange: [],
          resourceType: '',
          resourceTypes: [],
          resourceName: '',
          resourceId: '',
          result: 'all',
          results: [],
          riskLevel: 'all',
          riskLevels: [],
          session: '',
          requestId: '',
          sorters: [{ field: 'created_at', direction: 'desc' }],
        },
        presets: [],
      },
      global: {
        plugins: [i18n],
        stubs: {
          LogFilterBuilder: logFilterBuilderStub,
        },
      },
    });

    await wrapper.get('[data-testid="close-sorter"]').trigger('click');

    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toMatchObject({
      sorters: [],
    });
  });

  it('normalizes duplicate created_at sorters and hard-disables add and move controls', () => {
    const wrapper = mount(AuditFilters, {
      props: {
        activePreset: 'all',
        modelValue: {
          keyword: '',
          actor: '',
          success: 'all',
          action: '',
          actionPrefix: '',
          actionPrefixes: [],
          actionKeywords: [],
          requestPathPrefixes: [],
          source: '',
          businessCategory: '',
          createdRange: [],
          resourceType: '',
          resourceTypes: [],
          resourceName: '',
          resourceId: '',
          result: 'all',
          results: [],
          riskLevel: 'all',
          riskLevels: [],
          session: '',
          requestId: '',
          sorters: [
            { field: 'created_at', direction: 'desc' },
            { field: 'created_at', direction: 'asc' },
          ],
        },
        presets: [],
      },
      global: {
        plugins: [i18n],
        stubs: {
          LogFilterBuilder: logFilterBuilderStub,
        },
      },
    });

    expect(JSON.parse(wrapper.get('[data-testid="sorters"]').text())).toEqual([
      { field: 'created_at', direction: 'desc' },
    ]);
    expect(wrapper.get('[data-testid="sort-add-disabled"]').text()).toBe('true');
    expect(JSON.parse(wrapper.get('[data-testid="sort-field-options"]').text())).toEqual([
      [{ label: '创建时间', value: 'created_at' }],
    ]);
    expect(JSON.parse(wrapper.get('[data-testid="sort-move-up-disabled"]').text())).toEqual([true]);
    expect(JSON.parse(wrapper.get('[data-testid="sort-move-down-disabled"]').text())).toEqual([true]);
  });
});
