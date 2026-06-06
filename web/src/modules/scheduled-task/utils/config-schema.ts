import { isJsonRecord, type JsonRecord, parseJsonRecord } from './json';

export type ConfigFieldType = 'string' | 'integer' | 'number' | 'boolean';

export type ConfigSchemaProperty = {
  type?: ConfigFieldType;
  enum?: Array<string | number | boolean>;
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  default?: unknown;
  title?: string;
  description?: string;
  'x-title-key'?: string;
  'x-description-key'?: string;
};

export type ConfigSchema = {
  type?: 'object';
  properties?: Record<string, ConfigSchemaProperty>;
  required?: string[];
  additionalProperties?: boolean;
};

export type ConfigSchemaField = {
  key: string;
  schema: ConfigSchemaProperty;
  required: boolean;
};

export function parseConfigSchema(value?: string | null): ConfigSchema {
  const parsed = parseJsonRecord(value);
  if (parsed.type && parsed.type !== 'object') {
    return {};
  }

  return {
    type: parsed.type === 'object' ? 'object' : undefined,
    properties: parseProperties(parsed.properties),
    required: Array.isArray(parsed.required)
      ? parsed.required.filter((item): item is string => typeof item === 'string')
      : [],
    additionalProperties: typeof parsed.additionalProperties === 'boolean' ? parsed.additionalProperties : undefined,
  };
}

export function getConfigSchemaFields(schema: ConfigSchema): ConfigSchemaField[] {
  const required = new Set(schema.required ?? []);
  return Object.entries(schema.properties ?? {}).map(([key, property]) => ({
    key,
    schema: property,
    required: required.has(key),
  }));
}

export function mergeConfigRecords(defaultConfig: JsonRecord, config: JsonRecord): JsonRecord {
  return {
    ...defaultConfig,
    ...config,
  };
}

export function buildDefaultConfigFromSchema(schema: ConfigSchema): JsonRecord {
  const output: JsonRecord = {};
  for (const [key, property] of Object.entries(schema.properties ?? {})) {
    if ('default' in property) {
      output[key] = property.default;
    }
  }
  return output;
}

function parseProperties(value: unknown): Record<string, ConfigSchemaProperty> {
  if (!isJsonRecord(value)) {
    return {};
  }

  return Object.fromEntries(
    Object.entries(value)
      .filter((entry): entry is [string, JsonRecord] => isJsonRecord(entry[1]))
      .map(([key, raw]) => [key, parseProperty(raw)]),
  );
}

function parseProperty(raw: JsonRecord): ConfigSchemaProperty {
  const type = parseFieldType(raw.type);
  const property: ConfigSchemaProperty = {};

  if (type) {
    property.type = type;
  }
  if (Array.isArray(raw.enum)) {
    property.enum = raw.enum.filter(
      (value): value is string | number | boolean =>
        typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean',
    );
  }
  assignNumber(raw, property, 'minimum');
  assignNumber(raw, property, 'maximum');
  assignNumber(raw, property, 'minLength');
  assignNumber(raw, property, 'maxLength');
  assignString(raw, property, 'title');
  assignString(raw, property, 'description');
  assignString(raw, property, 'x-title-key');
  assignString(raw, property, 'x-description-key');
  if ('default' in raw) {
    property.default = raw.default;
  }

  return property;
}

function parseFieldType(value: unknown): ConfigFieldType | undefined {
  return value === 'string' || value === 'integer' || value === 'number' || value === 'boolean' ? value : undefined;
}

function assignNumber(raw: JsonRecord, property: ConfigSchemaProperty, key: keyof ConfigSchemaProperty) {
  if (typeof raw[key] === 'number') {
    property[key] = raw[key] as never;
  }
}

function assignString(raw: JsonRecord, property: ConfigSchemaProperty, key: keyof ConfigSchemaProperty) {
  if (typeof raw[key] === 'string') {
    property[key] = raw[key] as never;
  }
}
