import type {
  ProjectConfigurationDiffRequest,
  ProjectConfigurationValidateRequest,
  ProjectDeployRequest,
} from '../types/project';

export type ProjectConfigurationDraft = {
  compose_file_content: string;
  env_file_content: string;
};

export function normalizeTextBlock(value: string) {
  const normalized = String(value ?? '')
    .replace(/\r\n/g, '\n')
    .split('\n')
    .map((line) => line.replace(/\s+$/g, ''))
    .join('\n')
    .trim();

  return normalized ? `${normalized}\n` : '';
}

export function buildConfigurationDraftRequest(
  draft: ProjectConfigurationDraft,
): ProjectConfigurationDiffRequest & ProjectConfigurationValidateRequest & ProjectDeployRequest {
  return {
    compose_file_content: normalizeTextBlock(draft.compose_file_content || ''),
    env_file_content: normalizeTextBlock(draft.env_file_content || ''),
  };
}
