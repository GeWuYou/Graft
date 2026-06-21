import type { WebModuleRegistration } from '@/modules/types';

import { announcementBootstrapRouteRegistrations, announcementGlobalRouteRegistrations } from './bootstrap-routes';
import AnnouncementHeaderEntry from './components/AnnouncementHeaderEntry.vue';
import AnnouncementPopupHost from './components/AnnouncementPopupHost.vue';
import { ANNOUNCEMENT_PERMISSION_CODE } from './contract/permissions';

export const announcementModuleRegistration: WebModuleRegistration = {
  moduleId: 'announcement',
  bootstrapRoutes: announcementBootstrapRouteRegistrations,
  globalRoutes: announcementGlobalRouteRegistrations,
};

export const announcementModulePermissionCodes = ANNOUNCEMENT_PERMISSION_CODE;
export const announcementHeaderEntry = AnnouncementHeaderEntry;
export const announcementPopupHost = AnnouncementPopupHost;

export default announcementModuleRegistration;
