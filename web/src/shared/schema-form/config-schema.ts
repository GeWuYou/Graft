import { isJsonRecord, type JsonRecord, parseJsonRecord } from './json';

export type ConfigFieldType = 'string' | 'integer' | 'number' | 'boolean' | 'object' | 'array';

export type ConfigSchemaProperty = {
  type?: ConfigFieldType;
  enum?: Array<string | number | boolean>;
  enumLabels?: Record<string, ConfigSchemaOptionLabel>;
  minimum?: number;
  maximum?: number;
  minLength?: number;
  maxLength?: number;
  default?: unknown;
  title?: string;
  description?: string;
  placeholder?: string;
  xI18n?: ConfigSchemaI18n;
  'x-title-key'?: string;
  'x-description-key'?: string;
};

export type ConfigSchemaI18n = {
  titleKey?: string;
  descriptionKey?: string;
  placeholderKey?: string;
  unitKey?: string;
};

export type ConfigValidationReasonCode =
  | 'required'
  | 'additional_property'
  | 'type_mismatch'
  | 'enum'
  | 'below_minimum'
  | 'above_maximum'
  | 'too_short'
  | 'too_long';

export type ConfigValidationIssue = {
  field: string;
  key?: string;
  reasonCode: ConfigValidationReasonCode;
  constraint: string;
  minimum?: number;
  maximum?: number;
  expected?: unknown;
  actual?: unknown;
  schema?: ConfigSchemaProperty;
};

export type ConfigValidationResult = {
  valid: boolean;
  issues: ConfigValidationIssue[];
};

export type ConfigSchemaOptionLabel = {
  description?: string;
  descriptionKey?: string;
  label?: string;
  labelKey?: string;
};

export type ConfigSchema = ConfigSchemaProperty & {
  type?: ConfigFieldType;
  properties?: Record<string, ConfigSchemaProperty>;
  required?: string[];
  additionalProperties?: boolean;
};

export type ConfigSchemaField = {
  key: string;
  schema: ConfigSchemaProperty;
  required: boolean;
};

export function parseConfigSchema(value?: string | null | JsonRecord): ConfigSchema {
  const parsed = typeof value === 'string' || value === null || value === undefined ? parseJsonRecord(value) : value;
  return parseConfigSchemaRecord(parsed);
}

function parseConfigSchemaRecord(raw: JsonRecord): ConfigSchema {
  const type = parseFieldType(raw.type);

  return {
    ...parseProperty(raw),
    type,
    properties: parseProperties(raw.properties),
    required: Array.isArray(raw.required)
      ? raw.required.filter((item): item is string => typeof item === 'string')
      : [],
    additionalProperties: typeof raw.additionalProperties === 'boolean' ? raw.additionalProperties : undefined,
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

export function validateConfigRecord(schema: ConfigSchema, config: JsonRecord): ConfigValidationResult {
  const issues: ConfigValidationIssue[] = [];
  const properties = schema.properties ?? {};
  const required = new Set(schema.required ?? []);

  for (const key of required) {
    if (!(key in config)) {
      issues.push({
        field: `config_json.${key}`,
        key,
        reasonCode: 'required',
        constraint: 'required',
        schema: properties[key],
      });
    }
  }

  if (schema.additionalProperties === false) {
    for (const key of Object.keys(config)) {
      if (!(key in properties)) {
        issues.push({
          field: `config_json.${key}`,
          key,
          reasonCode: 'additional_property',
          constraint: 'additionalProperties',
          actual: key,
        });
      }
    }
  }

  for (const [key, property] of Object.entries(properties)) {
    if (!(key in config)) {
      continue;
    }
    const issue = validateConfigValue(`config_json.${key}`, key, property, config[key]);
    if (issue) {
      issues.push(issue);
    }
  }

  return {
    valid: issues.length === 0,
    issues,
  };
}

function validateConfigValue(
  field: string,
  key: string,
  property: ConfigSchemaProperty,
  value: unknown,
): ConfigValidationIssue | undefined {
  switch (property.type) {
    case 'string':
      if (typeof value !== 'string') {
        return typeIssue(field, key, property, 'string', value);
      }
      if (typeof property.minLength === 'number' && value.length < property.minLength) {
        return {
          field,
          key,
          reasonCode: 'too_short',
          constraint: 'minLength',
          minimum: property.minLength,
          actual: value.length,
          schema: property,
        };
      }
      if (typeof property.maxLength === 'number' && value.length > property.maxLength) {
        return {
          field,
          key,
          reasonCode: 'too_long',
          constraint: 'maxLength',
          maximum: property.maxLength,
          actual: value.length,
          schema: property,
        };
      }
      break;
    case 'integer':
      if (typeof value !== 'number' || !Number.isInteger(value)) {
        return typeIssue(field, key, property, 'integer', value);
      }
      {
        const rangeIssue = validateNumberRange(field, key, property, value);
        if (rangeIssue) {
          return rangeIssue;
        }
      }
      break;
    case 'number':
      if (typeof value !== 'number' || !Number.isFinite(value)) {
        return typeIssue(field, key, property, 'number', value);
      }
      {
        const rangeIssue = validateNumberRange(field, key, property, value);
        if (rangeIssue) {
          return rangeIssue;
        }
      }
      break;
    case 'boolean':
      if (typeof value !== 'boolean') {
        return typeIssue(field, key, property, 'boolean', value);
      }
      break;
    case 'object':
      if (!isJsonRecord(value)) {
        return typeIssue(field, key, property, 'object', value);
      }
      break;
    case 'array':
      if (!Array.isArray(value)) {
        return typeIssue(field, key, property, 'array', value);
      }
      break;
    default:
      break;
  }

  if (property.enum?.length && !property.enum.some((item) => item === value)) {
    return {
      field,
      key,
      reasonCode: 'enum',
      constraint: 'enum',
      expected: property.enum,
      actual: value,
      schema: property,
    };
  }

  return undefined;
}

function validateNumberRange(
  field: string,
  key: string,
  property: ConfigSchemaProperty,
  value: number,
): ConfigValidationIssue | undefined {
  if (typeof property.minimum === 'number' && value < property.minimum) {
    return {
      field,
      key,
      reasonCode: 'below_minimum',
      constraint: 'minimum',
      minimum: property.minimum,
      actual: value,
      schema: property,
    };
  }
  if (typeof property.maximum === 'number' && value > property.maximum) {
    return {
      field,
      key,
      reasonCode: 'above_maximum',
      constraint: 'maximum',
      maximum: property.maximum,
      actual: value,
      schema: property,
    };
  }
  return undefined;
}

function typeIssue(
  field: string,
  key: string,
  property: ConfigSchemaProperty,
  expected: ConfigFieldType,
  actual: unknown,
): ConfigValidationIssue {
  return {
    field,
    key,
    reasonCode: 'type_mismatch',
    constraint: 'type',
    expected,
    actual,
    schema: property,
  };
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
  property.enumLabels = parseEnumLabels(raw);
  assignNumber(raw, property, 'minimum');
  assignNumber(raw, property, 'maximum');
  assignNumber(raw, property, 'minLength');
  assignNumber(raw, property, 'maxLength');
  assignString(raw, property, 'title');
  assignString(raw, property, 'description');
  assignString(raw, property, 'placeholder');
  assignString(raw, property, 'x-title-key');
  assignString(raw, property, 'x-description-key');
  property.xI18n = parseI18n(raw);
  if ('default' in raw) {
    property.default = raw.default;
  }

  return property;
}

function parseI18n(raw: JsonRecord): ConfigSchemaI18n | undefined {
  const extension = isJsonRecord(raw['x-i18n']) ? raw['x-i18n'] : raw;
  const parsed: ConfigSchemaI18n = {};
  assignI18nString(extension, parsed, 'titleKey');
  assignI18nString(extension, parsed, 'descriptionKey');
  assignI18nString(extension, parsed, 'placeholderKey');
  assignI18nString(extension, parsed, 'unitKey');
  assignLegacyI18nString(raw, parsed, 'x-title-key', 'titleKey');
  assignLegacyI18nString(raw, parsed, 'x-description-key', 'descriptionKey');
  return Object.keys(parsed).length > 0 ? parsed : undefined;
}

function parseEnumLabels(raw: JsonRecord): Record<string, ConfigSchemaOptionLabel> | undefined {
  const i18nExtension = isJsonRecord(raw['x-i18n']) ? raw['x-i18n'] : {};
  const i18nEnumLabels = isJsonRecord(i18nExtension.enumLabels) ? i18nExtension.enumLabels : undefined;
  const rawOptions = isJsonRecord(raw.options)
    ? raw.options
    : isJsonRecord(raw.enumLabels)
      ? raw.enumLabels
      : i18nEnumLabels;
  if (!rawOptions) {
    return undefined;
  }

  const entries: Array<[string, ConfigSchemaOptionLabel]> = [];
  for (const [value, option] of Object.entries(rawOptions)) {
    if (typeof option === 'string') {
      const label =
        i18nEnumLabels === rawOptions
          ? ({ labelKey: option } satisfies ConfigSchemaOptionLabel)
          : ({ label: option } satisfies ConfigSchemaOptionLabel);
      entries.push([value, label]);
      continue;
    }
    if (!isJsonRecord(option)) {
      continue;
    }
    const label: ConfigSchemaOptionLabel = {};
    assignOptionLabelString(option, label, 'description');
    assignOptionLabelString(option, label, 'descriptionKey');
    assignOptionLabelString(option, label, 'label');
    assignOptionLabelString(option, label, 'labelKey');
    if (Object.keys(label).length > 0) {
      entries.push([value, label]);
    }
  }
  return entries.length > 0 ? Object.fromEntries(entries) : undefined;
}

function parseFieldType(value: unknown): ConfigFieldType | undefined {
  return value === 'string' ||
    value === 'integer' ||
    value === 'number' ||
    value === 'boolean' ||
    value === 'object' ||
    value === 'array'
    ? value
    : undefined;
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

function assignOptionLabelString(
  raw: JsonRecord,
  property: ConfigSchemaOptionLabel,
  key: keyof ConfigSchemaOptionLabel,
) {
  if (typeof raw[key] === 'string') {
    property[key] = raw[key];
  }
}

function assignI18nString(raw: JsonRecord, property: ConfigSchemaI18n, key: keyof ConfigSchemaI18n) {
  if (typeof raw[key] === 'string') {
    property[key] = raw[key];
  }
}

function assignLegacyI18nString(
  raw: JsonRecord,
  property: ConfigSchemaI18n,
  legacyKey: string,
  key: keyof ConfigSchemaI18n,
) {
  if (typeof raw[legacyKey] === 'string' && !property[key]) {
    property[key] = raw[legacyKey];
  }
}
