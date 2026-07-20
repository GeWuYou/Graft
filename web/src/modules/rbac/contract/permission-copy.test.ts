import { describe, expect, it } from 'vitest';

import { localizedPermissionDescription, localizedPermissionDisplay } from '../shared/permission-copy';
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
  ['template.manage', 'TemplateManage'],
  ['template.publish', 'TemplatePublish'],
] as const;

describe('PERMISSION_COPY_BY_CODE', () => {
  it('maps Application permission codes to Application locale keys', () => {
    for (const [codeSuffix, permissionName] of applicationPermissionCopyCases) {
      expect(PERMISSION_COPY_BY_CODE[`application.${codeSuffix}`]).toEqual({
        descriptionKey: `rbac.permissionCatalog.application${permissionName}.description`,
        displayKey: `rbac.permissionCatalog.application${permissionName}.display`,
      });
      expect(PERMISSION_COPY_BY_CODE[`ops.application.${codeSuffix}`]).toBeUndefined();
    }
  });

  it('maps the security overview permission to human-readable locale keys', () => {
    expect(PERMISSION_COPY_BY_CODE['security.overview.read']).toEqual({
      displayKey: 'rbac.permissionCatalog.securityOverviewRead.display',
      descriptionKey: 'rbac.permissionCatalog.securityOverviewRead.description',
    });
  });

  it('does not leak backend Chinese fallback text into the English permission page', () => {
    const translate = ((key: string) =>
      key === 'rbac.permissionList.emptyDescription' ? 'No description' : key) as never;
    const permission = {
      code: 'ops.unknown.read',
      display: '读取未知权限',
      description: '允许读取未知权限。',
      display_key: 'rbac.permissionCatalog.unknownRead.display',
      description_key: 'rbac.permissionCatalog.unknownRead.description',
    };

    expect(localizedPermissionDisplay(translate, permission, 'en-US')).toBe('ops.unknown.read');
    expect(localizedPermissionDescription(translate, permission, 'rbac.permissionList.emptyDescription', 'en-US')).toBe(
      'No description',
    );
  });
});
