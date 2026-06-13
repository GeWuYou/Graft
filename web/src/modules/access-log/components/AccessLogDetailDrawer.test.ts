// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { shallowMount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import type { AccessLogItem } from '../types/access-log';
import AccessLogDetailDrawer from './AccessLogDetailDrawer.vue';

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}));

const LogJsonPanelStub = defineComponent({
  name: 'LogJsonPanelStub',
  props: [
    'title',
    'expandLabel',
    'collapseLabel',
    'copyLabel',
    'copySuccessLabel',
    'copyFailLabel',
    'emptyText',
    'value',
  ],
  setup(props) {
    return () => h('pre', { 'data-testid': `json-panel-${props.title}` }, JSON.stringify(props.value));
  },
});

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: {
    'zh-CN': {
      accessLog: {
        actions: {
          copy: '复制',
          copyFail: '复制失败',
          copySuccess: '已复制',
          viewRelatedAuditRecords: '查看关联审计记录',
        },
        columns: {
          durationMs: '耗时',
          method: '方法',
          occurredAt: '完成时间',
          path: '路径',
          startedAt: '开始时间',
          statusCode: '状态码',
        },
        detail: {
          basic: '基础信息',
          clientIp: '客户端 IP',
          collapseContext: '收起原始 JSON',
          contextEmpty: '当前请求没有可展示的原始 JSON。',
          copyContext: '复制 JSON',
          copyContextFail: '复制原始 JSON 失败',
          copyContextSuccess: '原始 JSON 已复制',
          correlation: '关联信息',
          expandContext: '展开原始 JSON',
          network: '网络信息',
          occurredAtRaw: '完成时间原值',
          rawJson: '原始 JSON',
          relatedAudit: '关联审计排查',
          requestId: '请求 ID',
          requestSize: '请求大小',
          responseSize: '响应大小',
          route: '路由模板',
          user: '用户名',
          userAgent: '用户代理',
          userId: '用户 ID',
        },
        page: { detailTitle: '请求详情' },
        user: {
          anonymous: '匿名用户',
          noUserId: '未关联用户 ID',
          unauthenticated: '未登录请求',
          userIdValue: '用户 ID：{id}',
        },
      },
    },
  },
});

function accessLogRecord(): AccessLogItem {
  return {
    client_ip: '127.0.0.1',
    duration_ms: 3,
    id: 8,
    method: 'GET',
    occurred_at: '2026-06-13T08:00:01Z',
    path: '/api/access-log',
    request_id: 'req-8',
    request_size: 128,
    response_size: 256,
    route: '/api/access-log',
    started_at: '2026-06-13T08:00:00Z',
    status_code: 200,
    user_agent: 'Mozilla/5.0',
    user_id: 1,
    username: 'graft',
  } as AccessLogItem;
}

describe('AccessLogDetailDrawer', () => {
  it('renders structured descriptions and the raw JSON panel inside the drawer', () => {
    const record = accessLogRecord();
    const wrapper = shallowMount(AccessLogDetailDrawer, {
      props: {
        record,
        visible: true,
      },
      global: {
        plugins: [i18n],
        stubs: {
          LogJsonPanel: LogJsonPanelStub,
          TButton: true,
          TDescriptions: { template: '<section><slot /></section>' },
          TDescriptionsItem: { template: '<div><slot /></div>' },
          TDrawer: { template: '<aside><slot /></aside>' },
          TTabPanel: { template: '<div><slot /></div>' },
          TTabs: { template: '<div><slot /></div>' },
        },
      },
    });

    expect(wrapper.text()).toContain('/api/access-log');
    expect(wrapper.text()).not.toContain('trace-8');
    expect(wrapper.text()).toContain('Mozilla/5.0');
    expect(wrapper.get('[data-testid="json-panel-原始 JSON"]').text()).toContain('"id":8');
    expect(wrapper.get('[data-testid="json-panel-原始 JSON"]').text()).toContain('"request_id":"req-8"');
    expect(wrapper.get('[data-testid="json-panel-原始 JSON"]').text()).not.toContain('trace');
  });
});
