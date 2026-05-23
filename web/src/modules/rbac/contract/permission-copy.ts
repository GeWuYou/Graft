import { MONITOR_PERMISSION_CODE } from '@/modules/monitor/contract/permissions';
import { USER_PERMISSION_CODE } from '@/modules/user/contract/permissions';

import { RBAC_PERMISSION_CODE } from './permissions';

export type PermissionCopyEntry = {
  descriptionKey: string;
  displayKey: string;
};

export const PERMISSION_COPY_BY_CODE: Record<string, PermissionCopyEntry> = {
  [USER_PERMISSION_CODE.CREATE]: {
    displayKey: 'rbac.permissionCatalog.userCreate.display',
    descriptionKey: 'rbac.permissionCatalog.userCreate.description',
  },
  [USER_PERMISSION_CODE.UPDATE]: {
    displayKey: 'rbac.permissionCatalog.userUpdate.display',
    descriptionKey: 'rbac.permissionCatalog.userUpdate.description',
  },
  [USER_PERMISSION_CODE.DISABLE]: {
    displayKey: 'rbac.permissionCatalog.userDisable.display',
    descriptionKey: 'rbac.permissionCatalog.userDisable.description',
  },
  'user.read': {
    displayKey: 'rbac.permissionCatalog.userRead.display',
    descriptionKey: 'rbac.permissionCatalog.userRead.description',
  },
  'user.session.read': {
    displayKey: 'rbac.permissionCatalog.userSessionRead.display',
    descriptionKey: 'rbac.permissionCatalog.userSessionRead.description',
  },
  'user.session.revoke': {
    displayKey: 'rbac.permissionCatalog.userSessionRevoke.display',
    descriptionKey: 'rbac.permissionCatalog.userSessionRevoke.description',
  },
  [RBAC_PERMISSION_CODE.ROLE_READ]: {
    displayKey: 'rbac.permissionCatalog.roleRead.display',
    descriptionKey: 'rbac.permissionCatalog.roleRead.description',
  },
  [RBAC_PERMISSION_CODE.ROLE_CREATE]: {
    displayKey: 'rbac.permissionCatalog.roleCreate.display',
    descriptionKey: 'rbac.permissionCatalog.roleCreate.description',
  },
  [RBAC_PERMISSION_CODE.ROLE_UPDATE]: {
    displayKey: 'rbac.permissionCatalog.roleUpdate.display',
    descriptionKey: 'rbac.permissionCatalog.roleUpdate.description',
  },
  [RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN]: {
    displayKey: 'rbac.permissionCatalog.rolePermissionAssign.display',
    descriptionKey: 'rbac.permissionCatalog.rolePermissionAssign.description',
  },
  [RBAC_PERMISSION_CODE.PERMISSION_READ]: {
    displayKey: 'rbac.permissionCatalog.permissionRead.display',
    descriptionKey: 'rbac.permissionCatalog.permissionRead.description',
  },
  [RBAC_PERMISSION_CODE.USER_ROLE_READ]: {
    displayKey: 'rbac.permissionCatalog.userRoleRead.display',
    descriptionKey: 'rbac.permissionCatalog.userRoleRead.description',
  },
  [RBAC_PERMISSION_CODE.USER_ROLE_ASSIGN]: {
    displayKey: 'rbac.permissionCatalog.userRoleAssign.display',
    descriptionKey: 'rbac.permissionCatalog.userRoleAssign.description',
  },
  [MONITOR_PERMISSION_CODE.SERVER_STATUS_READ]: {
    displayKey: 'rbac.permissionCatalog.monitorServerStatusRead.display',
    descriptionKey: 'rbac.permissionCatalog.monitorServerStatusRead.description',
  },
};
