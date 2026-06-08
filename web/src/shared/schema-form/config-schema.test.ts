import { describe, expect, it } from 'vitest';

import type { ConfigSchema } from './config-schema';
import { validateConfigRecord } from './config-schema';

describe('validateConfigRecord', () => {
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
