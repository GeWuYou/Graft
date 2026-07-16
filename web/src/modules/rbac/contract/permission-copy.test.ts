import { describe, expect, it } from 'vitest';

import { PERMISSION_COPY_BY_CODE } from './permission-copy';

const applicationPermissionCopyCases = [
  ['view', 'View'],
  ['import', 'Import'],
  ['refresh', 'Refresh'],
  ['lifecycle', 'Lifecycle'],
  ['destroy', 'Destroy'],
  ['create', 'Create'],
  ['creation-method.view', 'CreationMethodView'],
  ['discovery.view', 'DiscoveryView'],
  ['deploy', 'Deploy'],
] as const;

describe('PERMISSION_COPY_BY_CODE', () => {
  it('maps Application permission codes to Application locale keys', () => {
    for (const [codeSuffix, permissionName] of applicationPermissionCopyCases) {
      expect(PERMISSION_COPY_BY_CODE[`ops.application.${codeSuffix}`]).toEqual({
        descriptionKey: `rbac.permissionCatalog.application${permissionName}.description`,
        displayKey: `rbac.permissionCatalog.application${permissionName}.display`,
      });
      expect(PERMISSION_COPY_BY_CODE[`ops.project.${codeSuffix}`]).toBeUndefined();
    }
  });
});
