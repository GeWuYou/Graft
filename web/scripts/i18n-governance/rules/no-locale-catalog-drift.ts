// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { collectLocaleCatalogs, localePairKey, localeViolation, resolveKeyOwner } from '../locale-utils';
import type { I18nGovernanceRule, RuleViolation } from '../types';

const SUGGESTION = 'Keep paired zh-CN/en-US catalogs and root/module locale key ownership aligned.';

export const noLocaleCatalogDriftRule: I18nGovernanceRule = {
  id: 'no-locale-catalog-drift',
  description: 'Blocks zh-CN/en-US catalog pairing drift, key-set drift, and split root/module locale key ownership.',
  defaultSeverity: 'error',
  appliesTo: ['locale'],
  check(context) {
    const catalogs = collectLocaleCatalogs(context);
    const groupedFiles = new Map<string, Partial<Record<'zh-CN' | 'en-US', (typeof catalogs)[number]>>>();
    const ownerDefinitions = new Map<string, Map<string, Set<string>>>();
    const violations: RuleViolation[] = [];

    for (const catalog of catalogs) {
      const pairKey = localePairKey(catalog.file);
      const group = groupedFiles.get(pairKey) ?? {};
      group[catalog.locale] = catalog;
      groupedFiles.set(pairKey, group);

      for (const key of catalog.messages.keys()) {
        const keyOwner = resolveKeyOwner(catalog.file, key);
        const sourceOwners = ownerDefinitions.get(key) ?? new Map<string, Set<string>>();
        const files = sourceOwners.get(keyOwner) ?? new Set<string>();
        files.add(catalog.file);
        sourceOwners.set(keyOwner, files);
        ownerDefinitions.set(key, sourceOwners);
      }
    }

    for (const [key, owners] of ownerDefinitions) {
      const rootFiles = owners.get('root');
      if (!rootFiles) continue;

      const moduleOwners = [...owners.entries()].filter(([owner]) => owner.startsWith('module:'));
      if (moduleOwners.length === 0) continue;

      const moduleFiles = moduleOwners.flatMap(([, files]) => [...files]);
      violations.push(
        localeViolation(
          noLocaleCatalogDriftRule.id,
          'error',
          [...rootFiles, ...moduleFiles].sort().join(', '),
          `split locale ownership for ${key} between root and module catalogs`,
          SUGGESTION,
        ),
      );
    }

    for (const [pairKey, group] of groupedFiles) {
      if (!group['zh-CN'] || !group['en-US']) {
        violations.push(
          localeViolation(
            noLocaleCatalogDriftRule.id,
            'error',
            pairKey,
            'missing paired zh-CN/en-US locale file',
            SUGGESTION,
          ),
        );
        continue;
      }

      const zhFile = group['zh-CN'].file;
      const enFile = group['en-US'].file;
      const zhKeys = new Set(group['zh-CN'].messages.keys());
      const enKeys = new Set(group['en-US'].messages.keys());

      for (const key of [...zhKeys].sort()) {
        if (!enKeys.has(key)) {
          violations.push(
            localeViolation(noLocaleCatalogDriftRule.id, 'error', enFile, `missing locale key ${key}`, SUGGESTION),
          );
        }
      }

      for (const key of [...enKeys].sort()) {
        if (!zhKeys.has(key)) {
          violations.push(
            localeViolation(noLocaleCatalogDriftRule.id, 'error', zhFile, `missing locale key ${key}`, SUGGESTION),
          );
        }
      }
    }

    return violations.sort((left, right) => {
      if (left.filePath !== right.filePath) return left.filePath.localeCompare(right.filePath);
      return left.message.localeCompare(right.message);
    });
  },
};
