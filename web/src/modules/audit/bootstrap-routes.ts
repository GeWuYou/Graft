import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { AUDIT_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const overviewRouteTitle = localizeRouteTitleKey('audit.route.overview.title');
const overviewBreadcrumbTitle = localizeRouteTitleKey('audit.route.overview.breadcrumb');
const logListRouteTitle = localizeRouteTitleKey('audit.route.logList.title');
const logListBreadcrumbTitle = localizeRouteTitleKey('audit.route.logList.breadcrumb');
const incidentDetailRouteTitle = localizeRouteTitleKey('audit.route.incidentDetail.title');
const incidentDetailBreadcrumbTitle = localizeRouteTitleKey('audit.route.incidentDetail.breadcrumb');

export const auditBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...AUDIT_BOOTSTRAP_ROUTE.OVERVIEW,
    loadPage: () => import('./pages/overview/index.vue'),
    meta: {
      domain: 'audit',
      tabGroup: 'audit-overview',
      dashboard: true,
      pageKind: 'overview',
      semanticTitle: overviewRouteTitle,
      breadcrumbTitle: overviewBreadcrumbTitle,
      tabTitle: overviewRouteTitle,
    },
  },
  {
    ...AUDIT_BOOTSTRAP_ROUTE.LOG_LIST,
    loadPage: () => import('./pages/logs/index.vue'),
    meta: {
      domain: 'audit',
      tabGroup: 'audit-logs',
      pageKind: 'list',
      semanticTitle: logListRouteTitle,
      breadcrumbTitle: logListBreadcrumbTitle,
      tabTitle: logListRouteTitle,
    },
  },
  {
    ...AUDIT_BOOTSTRAP_ROUTE.INCIDENT_DETAIL,
    loadPage: () => import('./pages/incident/index.vue'),
    meta: {
      domain: 'audit',
      tabGroup: 'audit-incident',
      pageKind: 'detail',
      semanticTitle: incidentDetailRouteTitle,
      breadcrumbTitle: incidentDetailBreadcrumbTitle,
      tabTitle: incidentDetailRouteTitle,
    },
  },
];
