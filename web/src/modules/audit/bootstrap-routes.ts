import type { BootstrapRouteRegistration } from '@/modules/types';

import { AUDIT_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const auditBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...AUDIT_BOOTSTRAP_ROUTE.OVERVIEW,
    loadPage: () => import('./pages/overview/index.vue'),
    meta: {
      domain: 'audit',
      tabGroup: 'audit',
      dashboard: true,
      pageKind: 'overview',
      semanticTitle: {
        'zh-CN': '安全审计 · 概览',
        'en-US': 'Audit Overview',
      },
      tabTitle: {
        'zh-CN': '安全审计 · 概览',
        'en-US': 'Audit Overview',
      },
    },
  },
  {
    ...AUDIT_BOOTSTRAP_ROUTE.LOG_LIST,
    loadPage: () => import('./pages/logs/index.vue'),
    meta: {
      domain: 'audit',
      tabGroup: 'audit',
      pageKind: 'investigation',
      investigationSurface: true,
      semanticTitle: {
        'zh-CN': '安全审计 · 日志调查',
        'en-US': 'Audit Log Investigation',
      },
      tabTitle: {
        'zh-CN': '安全审计 · 日志',
        'en-US': 'Audit Logs',
      },
    },
  },
];
