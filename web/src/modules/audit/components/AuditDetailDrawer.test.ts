// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { createI18n } from 'vue-i18n';

import AuditDetailDrawer from './AuditDetailDrawer.vue';

const pushMock = vi.fn();

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: pushMock,
  }),
}));

vi.mock('@/shared/observability', async () => {
  const actual = await vi.importActual<typeof import('@/shared/observability')>('@/shared/observability');
  return {
    ...actual,
    copyText: vi.fn(async () => true),
  };
});

const i18n = createI18n({
  legacy: false,
  locale: 'en-US',
  messages: {
    'en-US': {
      audit: {
        common: {
          unknownActor: 'Anonymous',
          unknownResource: 'Unknown resource',
          source: {
            REQUEST: 'Audit Event',
            SECURITY_EVENT: 'Security Event',
            DOMAIN_EVENT: 'Domain Audit',
            UNKNOWN: 'Unknown',
          },
          result: { SUCCESS: 'Success', FAILED: 'Failed', DENIED: 'Denied', ERROR: 'Error' },
          risk: { LOW: 'Low', MEDIUM: 'Medium', HIGH: 'High', CRITICAL: 'Critical' },
        },
        logList: {
          detailTitle: 'Audit Detail',
          reasonFallback: 'No additional reason',
          drawer: {
            messageFallback: 'No additional message',
            sections: {
              basic: 'Basic',
              request: 'Request',
              security: 'Security',
              correlation: 'Correlation',
              risk: 'Risk',
              metadata: 'Metadata',
            },
            fields: {
              target: 'Target',
              source: 'Source',
              result: 'Result',
              reason: 'Reason',
              requestId: 'Request ID',
              sessionId: 'Session ID',
              ip: 'IP',
              userAgent: 'User-Agent',
              method: 'Method',
              path: 'Path',
              status: 'Status',
              eventType: 'Event Type',
              permission: 'Permission',
              securityTarget: 'Security Target',
              traceId: 'Trace ID',
            },
            actions: {
              copyRequestId: 'Copy',
              copyRequestIdSuccess: 'Copied',
              copyRequestIdFail: 'Copy failed',
              expandMetadata: 'Expand metadata',
              collapseMetadata: 'Collapse metadata',
              copyMetadata: 'Copy JSON',
              copyMetadataSuccess: 'Metadata copied',
              copyMetadataFail: 'Metadata copy failed',
              backToMonitor: 'Back to monitor',
              viewRelatedRequest: 'View Related Request',
              viewAccessLogRequest: 'View Access Log',
              openRelatedEvents: 'Open related events',
            },
            related: {
              sameRequest: 'Same Request',
              sameActor: 'Same Actor',
              sameResource: 'Same Resource',
              empty: 'Empty',
            },
            risk: {
              failedOperation: 'Failed operation',
              sensitiveOperation: 'Sensitive write',
              requestTrace: 'Request trace',
              securityEvent: 'Security Event',
            },
            metadataEmpty: 'No metadata',
          },
          columns: {
            actor: 'Actor',
            createdAt: 'Created At',
          },
        },
      },
    },
  },
});

describe('AuditDetailDrawer', () => {
  it('does not render the removed incident action and keeps request correlation actions', () => {
    const wrapper = mount(AuditDetailDrawer, {
      props: {
        visible: true,
        monitorOrigin: null,
        rows: [],
        record: {
          id: 1,
          action: 'auth.permission.denied',
          actor_display_name: 'Admin',
          actor_username: 'admin',
          actor_user_id: 1,
          resource_type: 'permission',
          resource_id: 'rbac.role.read',
          resource_name: 'rbac.role.read',
          target: { kind: 'incident', type: 'incident', id: '42', label: 'Incident #42' },
          request_id: 'req-1',
          trace_id: 'trace-1',
          session_id: 'sess-1',
          source: 'SECURITY_EVENT',
          result: 'DENIED',
          success: false,
          risk_level: 'HIGH',
          ip: '127.0.0.1',
          user_agent: 'vitest',
          request_method: 'POST',
          request_path: '/api/auth/login',
          status_code: 401,
          message: 'Denied',
          metadata: {
            eventType: 'auth.permission.denied',
            permission: 'rbac.role.read',
            targetName: 'rbac.role.read',
            traceId: 'trace-1',
          },
          created_at: '2026-05-31T04:00:00Z',
        },
      },
      global: {
        plugins: [i18n],
      },
    });

    expect(wrapper.text()).toContain('View Access Log');
    expect(wrapper.text()).toContain('Open related events');
    expect(wrapper.text()).toContain('Security');
    expect(wrapper.text()).toContain('auth.permission.denied');
    expect(wrapper.text()).toContain('rbac.role.read');
    expect(wrapper.text()).toContain('trace-1');
    expect(wrapper.text()).not.toContain('openIncident');
  });
});
