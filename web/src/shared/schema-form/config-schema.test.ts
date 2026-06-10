// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import type { ConfigSchema } from './config-schema';
import { parseConfigSchema, validateConfigRecord } from './config-schema';
import { configEditorContainer, configFieldRenderer } from './field-renderer';
import { validateConfigEditorValue } from './renderer-validation';

describe('validateConfigRecord', () => {
  it('parses enum label and description metadata', () => {
    expect(
      parseConfigSchema({
        type: 'string',
        enum: ['hybrid'],
        'x-i18n': {
          enumLabels: {
            hybrid: {
              labelKey: 'systemConfig.options.dashboardQuickActionStrategy.hybrid',
              descriptionKey: 'systemConfig.options.dashboardQuickActionStrategyDescriptions.hybrid',
            },
          },
        },
      }).enumLabels?.hybrid,
    ).toEqual({
      labelKey: 'systemConfig.options.dashboardQuickActionStrategy.hybrid',
      descriptionKey: 'systemConfig.options.dashboardQuickActionStrategyDescriptions.hybrid',
    });
  });

  it('validates object and array property types', () => {
    const schema: ConfigSchema = {
      type: 'object',
      properties: {
        metadata: { type: 'object' },
        tags: { type: 'array' },
      },
    };

    expect(
      validateConfigRecord(schema, {
        metadata: { owner: 'scheduler' },
        tags: ['nightly'],
      }),
    ).toEqual({
      valid: true,
      issues: [],
    });

    expect(
      validateConfigRecord(schema, {
        metadata: ['not-an-object'],
        tags: { not: 'an-array' },
      }).issues,
    ).toEqual([
      expect.objectContaining({
        field: 'config_json.metadata',
        key: 'metadata',
        reasonCode: 'type_mismatch',
        constraint: 'type',
        expected: 'object',
        actual: ['not-an-object'],
        schema: schema.properties?.metadata,
      }),
      expect.objectContaining({
        field: 'config_json.tags',
        key: 'tags',
        reasonCode: 'type_mismatch',
        constraint: 'type',
        expected: 'array',
        actual: { not: 'an-array' },
        schema: schema.properties?.tags,
      }),
    ]);
  });
});

describe('config field renderer', () => {
  it('uses schema enum before item type fallback', () => {
    expect(configFieldRenderer({ type: 'boolean', enum: ['enabled', 'disabled'] }, 'boolean')).toBe('select');
  });

  it('falls back to item type only when schema cannot decide', () => {
    expect(configFieldRenderer({}, 'boolean')).toBe('switch');
    expect(configFieldRenderer({}, 'integer')).toBe('input-number');
    expect(configFieldRenderer({}, 'object')).toBe('json-textarea');
  });

  it('keeps small flat object editors in dialogs and complex editors in drawers', () => {
    const smallSchema: ConfigSchema = {
      type: 'object',
      properties: {
        retentionDays: { type: 'integer' },
        batchSize: { type: 'integer' },
      },
    };
    const nestedSchema: ConfigSchema = {
      type: 'object',
      properties: {
        metadata: { type: 'object' },
      },
    };

    expect(
      configEditorContainer(smallSchema, [{ key: 'retentionDays', schema: { type: 'integer' }, required: false }]),
    ).toBe('dialog');
    expect(
      configEditorContainer(nestedSchema, [{ key: 'metadata', schema: { type: 'object' }, required: false }]),
    ).toBe('drawer');
  });
});

describe('validateConfigEditorValue', () => {
  it('validates scalar enum values from schema', () => {
    const result = validateConfigEditorValue(
      {
        type: 'string',
        enum: ['most_used', 'recent', 'hybrid'],
      },
      'manual',
      'string',
    );

    expect(result).toEqual({
      valid: false,
      issues: [
        expect.objectContaining({
          field: 'value',
          reasonCode: 'enum',
          constraint: 'enum',
          expected: ['most_used', 'recent', 'hybrid'],
          actual: 'manual',
        }),
      ],
    });
  });

  it('validates scalar number ranges from schema', () => {
    expect(validateConfigEditorValue({ type: 'integer', minimum: 1, maximum: 24 }, 0, 'integer').issues).toEqual([
      expect.objectContaining({
        field: 'value',
        reasonCode: 'below_minimum',
        constraint: 'minimum',
        minimum: 1,
        actual: 0,
      }),
    ]);

    expect(validateConfigEditorValue({ type: 'integer', minimum: 1, maximum: 24 }, 30, 'integer').issues).toEqual([
      expect.objectContaining({
        field: 'value',
        reasonCode: 'above_maximum',
        constraint: 'maximum',
        maximum: 24,
        actual: 30,
      }),
    ]);
  });
});
