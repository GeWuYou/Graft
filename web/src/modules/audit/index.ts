import type { WebModuleRegistration } from '@/modules/types';

import { auditBootstrapRouteRegistrations } from './bootstrap-routes';

export const auditModuleRegistration: WebModuleRegistration = {
  moduleId: 'audit',
  bootstrapRoutes: auditBootstrapRouteRegistrations,
};

export default auditModuleRegistration;
