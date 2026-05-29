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
        'zh-CN': '安全审计 - 概览',
        'en-US': 'Security Audit - Overview',
      },
      breadcrumbTitle: {
        'zh-CN': '概览',
        'en-US': 'Overview',
      },
      tabTitle: {
        'zh-CN': '安全审计 - 概览',
        'en-US': 'Security Audit - Overview',
      },
    },
  },
  {
    ...AUDIT_BOOTSTRAP_ROUTE.LOG_LIST,
    loadPage: () => import('./pages/logs/index.vue'),
    meta: {
      domain: 'audit',
      tabGroup: 'audit',
      pageKind: 'list',
      semanticTitle: {
        'zh-CN': '安全审计 - 审计日志',
        'en-US': 'Security Audit - Audit Logs',
      },
      breadcrumbTitle: {
        'zh-CN': '审计日志',
        'en-US': 'Audit Logs',
      },
      tabTitle: {
        'zh-CN': '安全审计 - 审计日志',
        'en-US': 'Security Audit - Audit Logs',
      },
    },
  },
  {
    ...AUDIT_BOOTSTRAP_ROUTE.INCIDENT_DETAIL,
    loadPage: () => import('./pages/incident/index.vue'),
    meta: {
      domain: 'audit',
      tabGroup: 'audit',
      pageKind: 'detail',
      semanticTitle: {
        'zh-CN': '安全审计 - 事件钻取',
        'en-US': 'Security Audit - Incident Drilldown',
      },
      breadcrumbTitle: {
        'zh-CN': '事件钻取',
        'en-US': 'Incident Drilldown',
      },
      tabTitle: {
        'zh-CN': '安全审计 - 事件钻取',
        'en-US': 'Security Audit - Incident Drilldown',
      },
    },
  },
];
