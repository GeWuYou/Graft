import type { ComputedRef } from 'vue';

import type { AuditClientFilterState } from './presentation';

export type AuditFilterKey = Exclude<keyof AuditClientFilterState, 'keyword' | 'createdRange' | 'sorters'>;

export type AuditFilterOption = {
  label: string;
  value: string;
};

type BaseAuditFilterDefinition<Key extends AuditFilterKey> = {
  key: Key;
  fieldLabelKey: string;
  placeholderKey: string;
};

export type AuditTextFilterKey = Extract<
  AuditFilterKey,
  'actor' | 'resourceName' | 'requestId' | 'session' | 'resourceId'
>;

export type AuditSingleSelectFilterKey = Extract<
  AuditFilterKey,
  'action' | 'source' | 'resourceType' | 'result' | 'riskLevel' | 'success'
>;

export type AuditMultiSelectFilterKey = Extract<
  AuditFilterKey,
  'actionPrefixes' | 'resourceTypes' | 'results' | 'riskLevels'
>;

export type AuditTagInputFilterKey = Extract<AuditFilterKey, 'actionKeywords' | 'requestPathPrefixes'>;

export type AuditTextFilterDefinition = BaseAuditFilterDefinition<AuditTextFilterKey> & {
  kind: 'text';
};

export type AuditSingleSelectFilterDefinition = BaseAuditFilterDefinition<AuditSingleSelectFilterKey> & {
  kind: 'select';
  options: ComputedRef<AuditFilterOption[]>;
};

export type AuditMultiSelectFilterDefinition = BaseAuditFilterDefinition<AuditMultiSelectFilterKey> & {
  kind: 'multi-select';
  options: ComputedRef<AuditFilterOption[]>;
};

export type AuditTagInputFilterDefinition = BaseAuditFilterDefinition<AuditTagInputFilterKey> & {
  kind: 'tag-input';
};

export type AuditFilterDefinition =
  | AuditTextFilterDefinition
  | AuditSingleSelectFilterDefinition
  | AuditMultiSelectFilterDefinition
  | AuditTagInputFilterDefinition;

export type AuditFilterDefinitionMap = Record<AuditFilterKey, AuditFilterDefinition>;
