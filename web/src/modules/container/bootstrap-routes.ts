import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { CONTAINER_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const listRouteTitle = localizeRouteTitleKey('container.route.list.title');
const listBreadcrumbTitle = localizeRouteTitleKey('container.route.list.breadcrumb');
const detailRouteTitle = localizeRouteTitleKey('container.route.detail.title');
const detailBreadcrumbTitle = localizeRouteTitleKey('container.route.detail.breadcrumb');
const imageRouteTitle = localizeRouteTitleKey('container.route.images.title');
const imageBreadcrumbTitle = localizeRouteTitleKey('container.route.images.breadcrumb');
const volumeListTitle = localizeRouteTitleKey('container.volume.route.list.title');

export const containerBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...CONTAINER_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'infrastructure',
      pageKind: 'list',
      semanticTitle: listRouteTitle,
      breadcrumbTitle: listBreadcrumbTitle,
      tabTitle: listRouteTitle,
    },
  },
  {
    ...CONTAINER_BOOTSTRAP_ROUTE.VOLUMES,
    loadPage: () => import('./pages/volumes/index.vue'),
    meta: {
      tabGroup: 'infrastructure',
      pageKind: 'list',
      semanticTitle: volumeListTitle,
      breadcrumbTitle: volumeListTitle,
      tabTitle: volumeListTitle,
    },
  },
  {
    ...CONTAINER_BOOTSTRAP_ROUTE.IMAGES,
    loadPage: () => import('./pages/images/index.vue'),
    meta: {
      tabGroup: 'infrastructure',
      pageKind: 'list',
      pageSurface: 'form-detail',
      semanticTitle: imageRouteTitle,
      breadcrumbTitle: imageBreadcrumbTitle,
      tabTitle: imageRouteTitle,
    },
  },
];

export const containerGlobalRouteRegistrations: GlobalRouteRegistration[] = [
  {
    ...CONTAINER_BOOTSTRAP_ROUTE.RESOURCES,
    navigationParentPath: CONTAINER_BOOTSTRAP_ROUTE.LIST.menuPath,
    loadPage: () => import('./pages/resources/index.vue'),
    meta: { hidden: true, hiddenMenu: true, pageKind: 'list', tabGroup: 'infrastructure', title: listRouteTitle },
  },
  {
    ...CONTAINER_BOOTSTRAP_ROUTE.DETAIL,
    navigationParentPath: CONTAINER_BOOTSTRAP_ROUTE.LIST.menuPath,
    loadPage: () => import('./pages/detail/index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: false,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: detailRouteTitle,
      breadcrumbTitle: detailBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'infrastructure',
      tabTitle: detailRouteTitle,
      title: detailRouteTitle,
      titleKey: 'container.route.detail.title',
    },
  },
];
