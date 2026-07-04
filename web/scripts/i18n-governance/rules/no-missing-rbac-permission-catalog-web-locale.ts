import { join } from 'node:path';

import { collectLocaleCatalogs, localeViolation } from '../locale-utils';
import type { I18nGovernanceRule, RuleViolation } from '../types';

const PERMISSION_KEY_PREFIX = 'rbac.permissionCatalog.';
const RBAC_WEB_LOCALE_PATHS = ['src/modules/rbac/locales/zh-CN.json', 'src/modules/rbac/locales/en-US.json'] as const;
const RBAC_PERMISSION_SOURCE_PATH_FRAGMENTS = [
  '/permission',
  '/module_registration',
  '/route_registration',
  '/bootstrap',
];
const PERMISSION_KEY_FIELD_PATTERN =
  /\b(?:DisplayKey|DescriptionKey|displayKey|descriptionKey|display_key|description_key)\s*:\s*"([^"$]+)"/g;

export const noMissingRbacPermissionCatalogWebLocaleRule: I18nGovernanceRule = {
  id: 'no-missing-rbac-permission-catalog-web-locale',
  description:
    'Blocks RBAC permission catalog locale keys that exist only in backend authority but are missing from the web RBAC locale catalogs.',
  defaultSeverity: 'error',
  appliesTo: ['go', 'locale'],
  check(context) {
    const violations: RuleViolation[] = [];
    const catalogs = collectLocaleCatalogs(context);
    const rbacWebCatalogs = catalogs.filter((catalog) => RBAC_WEB_LOCALE_PATHS.includes(catalog.file as never));
    const webDefinedKeys = new Set<string>();

    for (const catalog of rbacWebCatalogs) {
      for (const key of catalog.messages.keys()) webDefinedKeys.add(key);
    }

    const requiredKeys = new Set<string>();
    for (const file of context.serverFiles) {
      if (!RBAC_PERMISSION_SOURCE_PATH_FRAGMENTS.some((fragment) => file.relativePath.includes(fragment))) {
        continue;
      }
      for (const match of file.source.matchAll(PERMISSION_KEY_FIELD_PATTERN)) {
        const key = match[1]?.trim();
        if (key?.startsWith(PERMISSION_KEY_PREFIX)) {
          requiredKeys.add(key);
        }
      }
    }

    for (const key of [...requiredKeys].sort()) {
      if (!webDefinedKeys.has(key)) {
        violations.push(
          localeViolation(
            noMissingRbacPermissionCatalogWebLocaleRule.id,
            'error',
            join('src', 'modules', 'rbac', 'locales'),
            `RBAC permission locale key ${key} is missing from web RBAC locale catalogs`,
            'Add the key to web/src/modules/rbac/locales/zh-CN.json and en-US.json so permission pages can render localized labels.',
          ),
        );
      }
    }

    return violations;
  },
};
