// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { ConfigFieldType, ConfigSchema, ConfigSchemaField, ConfigSchemaProperty } from './config-schema';

export type ConfigFieldRendererKind = 'select' | 'switch' | 'input-number' | 'json-textarea' | 'input';

export type ConfigEditorContainer = 'dialog' | 'drawer';

export function configFieldRenderer(
  schema: ConfigSchemaProperty,
  fallbackType?: ConfigFieldType | null,
): ConfigFieldRendererKind {
  if (schema.enum?.length) {
    return 'select';
  }

  const type = schema.type ?? fallbackType ?? undefined;
  switch (type) {
    case 'boolean':
      return 'switch';
    case 'integer':
    case 'number':
      return 'input-number';
    case 'object':
    case 'array':
      return 'json-textarea';
    case 'string':
    default:
      return 'input';
  }
}

function isObjectFieldRenderer(field: ConfigSchemaField) {
  return configFieldRenderer(field.schema) === 'json-textarea';
}

export function configEditorContainer(schema: ConfigSchema, fields: ConfigSchemaField[]): ConfigEditorContainer {
  if (schema.type === 'array') {
    return 'drawer';
  }
  if (schema.type === 'string' && typeof schema.maxLength === 'number' && schema.maxLength > 240) {
    return 'drawer';
  }
  if (schema.type !== 'object') {
    return 'dialog';
  }
  if (fields.length === 0 || fields.length >= 4) {
    return 'drawer';
  }
  return fields.some(isObjectFieldRenderer) ? 'drawer' : 'dialog';
}
