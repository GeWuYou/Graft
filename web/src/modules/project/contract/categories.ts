import type { ApplicationTemplateCategory } from '../types/project';

export const APPLICATION_TEMPLATE_CATEGORIES = [
  'database',
  'cache',
  'mq',
  'proxy',
  'storage',
  'monitoring',
  'logging',
  'cicd',
  'ai',
  'other',
] as const satisfies readonly ApplicationTemplateCategory[];
