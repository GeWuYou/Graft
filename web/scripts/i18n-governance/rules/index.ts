// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { I18nGovernanceRule } from '../types';
import { noDuplicateLocaleKeyRule } from './no-duplicate-locale-key';
import { noFallbackOnlyKeyFirstRule } from './no-fallback-only-key-first';
import { noHardcodedPluginMessageRule } from './no-hardcoded-plugin-message';
import { noHardcodedTemplateTextRule } from './no-hardcoded-template-text';
import { noHardcodedUiPropRule } from './no-hardcoded-ui-prop';
import { noLocaleCatalogDriftRule } from './no-locale-catalog-drift';
import { noMissingLocaleKeyRule } from './no-missing-locale-key';
import { noSystemConfigSchemaFallbackRule } from './no-system-config-schema-fallback';
import { noUnsafeDatetimeLocaleRule } from './no-unsafe-datetime-locale';
import { noUnsafeLocaleValueRule } from './no-unsafe-locale-value';
import { noUnusedLocaleKeyRule } from './no-unused-locale-key';

export const rules: I18nGovernanceRule[] = [
  noMissingLocaleKeyRule,
  noLocaleCatalogDriftRule,
  noUnusedLocaleKeyRule,
  noDuplicateLocaleKeyRule,
  noUnsafeDatetimeLocaleRule,
  noUnsafeLocaleValueRule,
  noHardcodedUiPropRule,
  noHardcodedPluginMessageRule,
  noHardcodedTemplateTextRule,
  noFallbackOnlyKeyFirstRule,
  noSystemConfigSchemaFallbackRule,
];
