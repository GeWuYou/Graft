import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { SYSTEM_CONFIG_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const listRouteTitle = localizeRouteTitleKey('systemConfig.route.list.title');
const listBreadcrumbTitle = localizeRouteTitleKey('systemConfig.route.list.breadcrumb');

/** 系统配置页作为平台设置路由注册到壳层，页面实现和配置数据仍由本模块拥有。 */
export const systemConfigBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...SYSTEM_CONFIG_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'platform',
      pageKind: 'list',
      pageSurface: 'form-detail',
      semanticTitle: listRouteTitle,
      breadcrumbTitle: listBreadcrumbTitle,
      tabTitle: listRouteTitle,
    },
  },
];
