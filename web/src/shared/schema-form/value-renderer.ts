import type { ConfigSchemaProperty } from './config-schema';
import { valuePreview } from './json';

export type ConfigValuePresentation = {
  description: string;
  descriptionMode?: 'inline' | 'tooltip';
  value: string;
};

export type ConfigValueRendererInput = {
  booleanLabelResolver?: (value: boolean) => string;
  emptyValueLabel: string;
  optionDescriptionResolver?: (schema: ConfigSchemaProperty, value: unknown) => string;
  optionLabelResolver?: (schema: ConfigSchemaProperty, value: unknown) => string;
  schema?: ConfigSchemaProperty;
  schemaDescriptionResolver?: (schema: ConfigSchemaProperty) => string;
  unit?: string;
  value: unknown;
};

export function configValuePresentation(input: ConfigValueRendererInput): ConfigValuePresentation {
  const schema = input.schema;
  const optionMatchesEnum = schema?.enum?.some((option) => option === input.value) ?? false;
  const optionHasLabel = Object.hasOwn(schema?.enumLabels ?? {}, String(input.value));
  const optionText =
    schema && (optionMatchesEnum || (!schema.enum?.length && optionHasLabel))
      ? input.optionLabelResolver?.(schema, input.value)
      : '';
  if (optionText && schema) {
    return {
      description: input.optionDescriptionResolver?.(schema, input.value) ?? '',
      descriptionMode: 'tooltip',
      value: appendUnit(optionText, input.unit),
    };
  }

  return {
    description: schema ? (input.schemaDescriptionResolver?.(schema) ?? '') : '',
    descriptionMode: 'tooltip',
    value: appendUnit(
      valuePreview(input.value, input.emptyValueLabel, input.booleanLabelResolver ?? ((value) => String(value))),
      input.unit,
    ),
  };
}

function appendUnit(value: string, unit?: string) {
  return unit ? `${value} ${unit}` : value;
}
