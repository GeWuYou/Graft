import type { WebModuleRegistration } from '@/modules/types';

import { updateBootstrapRouteRegistrations } from './bootstrap-routes';
import UpdateNotification from './components/UpdateNotification.vue';
import UpdateProgressDialog from './components/UpdateProgressDialog.vue';
import UpdateProvider from './components/UpdateProvider.vue';
import UpdateVersionEntry from './components/UpdateVersionEntry.vue';
import { UPDATE_PERMISSION_CODE } from './contract/permissions';

export const updateModuleRegistration: WebModuleRegistration = {
  moduleId: 'platform-update',
  bootstrapRoutes: updateBootstrapRouteRegistrations,
};

export const updateModulePermissionCodes = UPDATE_PERMISSION_CODE;

/** 由后台壳层 Notice 挂载，复用 Provider 的发现状态；组件实现归平台更新模块所有。 */
export const updateNotification = UpdateNotification;

/** 由认证后的后台布局挂载一次，负责发现初始化与卸载清理；组件实现归平台更新模块所有。 */
export const updateProvider = UpdateProvider;
export const updateVersionEntry = UpdateVersionEntry;
export const updateProgressDialog = UpdateProgressDialog;

export default updateModuleRegistration;
