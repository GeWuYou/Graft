import type { WebModuleRegistration } from '@/modules/types';

import { backupBootstrapRouteRegistrations } from './bootstrap-routes';

export const backupModuleRegistration: WebModuleRegistration = {
  moduleId: 'platform-backup',
  bootstrapRoutes: backupBootstrapRouteRegistrations,
};

export default backupModuleRegistration;
