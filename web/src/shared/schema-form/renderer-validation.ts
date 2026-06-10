// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type {
  ConfigFieldType,
  ConfigSchema,
  ConfigSchemaProperty,
  ConfigValidationIssue,
  ConfigValidationReasonCode,
} from './config-schema';
import { getConfigSchemaFields, validateConfigRecord } from './config-schema';
import { configFieldRenderer } from './field-renderer';
import { isJsonRecord } from './json';

export type ConfigEditorValidationResult = {
  valid: boolean;
  issues: ConfigValidationIssue[];
};

export function validateConfigEditorValue(
  schema: ConfigSchema,
  value: unknown,
  fallbackType?: ConfigFieldType | null,
): ConfigEditorValidationResult {
  if (schema.type === 'object' && isJsonRecord(value)) {
    return validateConfigRecord(schema, value);
  }

  const issue = validateScalarValue('value', schema, value, fallbackType);
  return {
    valid: !issue,
    issues: issue ? [issue] : [],
  };
}

function validateScalarValue(
  field: string,
  schema: ConfigSchemaProperty,
  value: unknown,
  fallbackType?: ConfigFieldType | null,
): ConfigValidationIssue | undefined {
  const renderer = configFieldRenderer(schema, fallbackType);
  const effectiveType = schema.type ?? fallbackType ?? undefined;

  if (renderer === 'select' && schema.enum?.length && !schema.enum.some((item) => item === value)) {
    return issue(field, 'enum', 'enum', value, schema.enum, schema);
  }

  switch (effectiveType) {
    case 'string':
      if (typeof value !== 'string') {
        return issue(field, 'type_mismatch', 'type', value, 'string', schema);
      }
      if (typeof schema.minLength === 'number' && value.length < schema.minLength) {
        return issue(field, 'too_short', 'minLength', value.length, schema.minLength, schema);
      }
      if (typeof schema.maxLength === 'number' && value.length > schema.maxLength) {
        return issue(field, 'too_long', 'maxLength', value.length, schema.maxLength, schema);
      }
      return undefined;
    case 'integer':
      if (typeof value !== 'number' || !Number.isInteger(value)) {
        return issue(field, 'type_mismatch', 'type', value, 'integer', schema);
      }
      return numberRangeIssue(field, value, schema);
    case 'number':
      if (typeof value !== 'number' || !Number.isFinite(value)) {
        return issue(field, 'type_mismatch', 'type', value, 'number', schema);
      }
      return numberRangeIssue(field, value, schema);
    case 'boolean':
      return typeof value === 'boolean' ? undefined : issue(field, 'type_mismatch', 'type', value, 'boolean', schema);
    case 'object':
      return isJsonRecord(value)
        ? validateObjectScalar(schema, value)
        : issue(field, 'type_mismatch', 'type', value, 'object', schema);
    case 'array':
      return Array.isArray(value) ? undefined : issue(field, 'type_mismatch', 'type', value, 'array', schema);
    default:
      return undefined;
  }
}

function validateObjectScalar(schema: ConfigSchemaProperty, value: Record<string, unknown>) {
  const objectSchema = schema as ConfigSchema;
  if (getConfigSchemaFields(objectSchema).length === 0) {
    return undefined;
  }
  const result = validateConfigRecord(objectSchema, value);
  return result.issues[0];
}

function numberRangeIssue(field: string, value: number, schema: ConfigSchemaProperty) {
  if (typeof schema.minimum === 'number' && value < schema.minimum) {
    return {
      ...issue(field, 'below_minimum', 'minimum', value, schema.minimum, schema),
      minimum: schema.minimum,
    };
  }
  if (typeof schema.maximum === 'number' && value > schema.maximum) {
    return {
      ...issue(field, 'above_maximum', 'maximum', value, schema.maximum, schema),
      maximum: schema.maximum,
    };
  }
  return undefined;
}

function issue(
  field: string,
  reasonCode: ConfigValidationReasonCode,
  constraint: string,
  actual: unknown,
  expected: unknown,
  schema: ConfigSchemaProperty,
): ConfigValidationIssue {
  return {
    field,
    reasonCode,
    constraint,
    expected,
    actual,
    schema,
  };
}
